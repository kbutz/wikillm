"""
Model Context Protocol (MCP) Client Manager
"""
import asyncio
import json
import logging
import uuid
import os
from typing import Dict, List, Optional, Any, Union
from dataclasses import dataclass
from enum import Enum
from pathlib import Path
import subprocess
import shutil

from pydantic import BaseModel, validator
import httpx

logger = logging.getLogger(__name__)


class MCPServerType(str, Enum):
    """MCP Server transport types"""
    STDIO = "stdio"
    HTTP = "http"
    WEBSOCKET = "websocket"


class MCPServerStatus(str, Enum):
    """MCP Server connection status"""
    DISCONNECTED = "disconnected"
    CONNECTING = "connecting"
    CONNECTED = "connected"
    ERROR = "error"


@dataclass
class MCPTool:
    """MCP Tool definition"""
    name: str
    description: str
    input_schema: Dict[str, Any]
    server_id: str


@dataclass
class MCPResource:
    """MCP Resource definition"""
    uri: str
    name: str
    description: Optional[str] = None
    mime_type: Optional[str] = None
    server_id: Optional[str] = None


@dataclass
class MCPPrompt:
    """MCP Prompt definition"""
    name: str
    description: str
    arguments: List[Dict[str, Any]]
    server_id: str


class MCPServerConfig(BaseModel):
    """MCP Server configuration"""
    server_id: str
    name: str
    description: Optional[str] = None
    type: MCPServerType
    command: Optional[str] = None  # For stdio servers
    args: Optional[List[str]] = None  # For stdio servers
    url: Optional[str] = None  # For HTTP/WebSocket servers
    env: Optional[Dict[str, str]] = None
    timeout: int = 30
    enabled: bool = True
    auto_reconnect: bool = True
    
    @validator('command')
    def validate_stdio_command(cls, v, values):
        if values.get('type') == MCPServerType.STDIO and not v:
            raise ValueError("command is required for stdio servers")
        return v
    
    @validator('url')
    def validate_http_url(cls, v, values):
        if values.get('type') in [MCPServerType.HTTP, MCPServerType.WEBSOCKET] and not v:
            raise ValueError("url is required for HTTP/WebSocket servers")
        return v


class MCPClient:
    """Individual MCP Client for a single server"""
    
    def __init__(self, config: MCPServerConfig):
        self.config = config
        self.session_id = str(uuid.uuid4())
        self.status = MCPServerStatus.DISCONNECTED
        self.process: Optional[subprocess.Popen] = None
        self.http_client: Optional[httpx.AsyncClient] = None
        self.tools: Dict[str, MCPTool] = {}
        self.resources: Dict[str, MCPResource] = {}
        self.prompts: Dict[str, MCPPrompt] = {}
        self.error_message: Optional[str] = None
        
    async def connect(self) -> bool:
        """Connect to the MCP server"""
        try:
            self.status = MCPServerStatus.CONNECTING
            self.error_message = None
            
            if self.config.type == MCPServerType.STDIO:
                return await self._connect_stdio()
            elif self.config.type == MCPServerType.HTTP:
                return await self._connect_http()
            elif self.config.type == MCPServerType.WEBSOCKET:
                return await self._connect_websocket()
            else:
                raise ValueError(f"Unsupported server type: {self.config.type}")
                
        except Exception as e:
            logger.error(f"Failed to connect to MCP server {self.config.server_id}: {e}")
            self.status = MCPServerStatus.ERROR
            self.error_message = str(e)
            return False
    
    async def _connect_stdio(self) -> bool:
        """Connect to stdio-based MCP server"""
        try:
            # Validate command exists
            if not shutil.which(self.config.command):
                raise FileNotFoundError(f"Command not found: {self.config.command}")
            
            # Build command with arguments
            cmd = [self.config.command] + (self.config.args or [])
            
            # Start process
            self.process = subprocess.Popen(
                cmd,
                stdin=subprocess.PIPE,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True,
                env={**os.environ, **(self.config.env or {})}
            )
            
            # Initialize MCP protocol
            init_request = {
                "jsonrpc": "2.0",
                "id": 1,
                "method": "initialize",
                "params": {
                    "protocolVersion": "2024-11-05",
                    "capabilities": {
                        "roots": {"listChanged": True},
                        "sampling": {},
                        "elicitation": {}
                    },
                    "clientInfo": {
                        "name": "wikillm-assistant",
                        "version": "1.0.0"
                    }
                }
            }
            
            # Send initialize request
            self.process.stdin.write(json.dumps(init_request) + "\n")
            self.process.stdin.flush()
            
            # Read initialize response
            response_line = self.process.stdout.readline()
            if not response_line:
                raise Exception("No response from MCP server")
            
            response = json.loads(response_line.strip())
            if "error" in response:
                raise Exception(f"MCP server error: {response['error']}")
            
            # Send initialized notification
            initialized_notification = {
                "jsonrpc": "2.0",
                "method": "notifications/initialized"
            }
            self.process.stdin.write(json.dumps(initialized_notification) + "\n")
            self.process.stdin.flush()
            
            # List available capabilities
            await self._list_server_capabilities()
            
            self.status = MCPServerStatus.CONNECTED
            logger.info(f"Connected to stdio MCP server {self.config.server_id}")
            return True
            
        except Exception as e:
            if self.process:
                self.process.terminate()
                self.process = None
            raise e
    
    async def _connect_http(self) -> bool:
        """Connect to HTTP-based MCP server"""
        try:
            self.http_client = httpx.AsyncClient(
                base_url=self.config.url,
                timeout=self.config.timeout
            )
            
            # Initialize MCP protocol over HTTP
            init_request = {
                "jsonrpc": "2.0",
                "id": 1,
                "method": "initialize",
                "params": {
                    "protocolVersion": "2024-11-05",
                    "capabilities": {
                        "roots": {"listChanged": True},
                        "sampling": {},
                        "elicitation": {}
                    },
                    "clientInfo": {
                        "name": "wikillm-assistant",
                        "version": "1.0.0"
                    }
                }
            }
            
            response = await self.http_client.post("/mcp", json=init_request)
            response.raise_for_status()
            
            result = response.json()
            if "error" in result:
                raise Exception(f"MCP server error: {result['error']}")
            
            # Send initialized notification
            initialized_notification = {
                "jsonrpc": "2.0",
                "method": "notifications/initialized"
            }
            await self.http_client.post("/mcp", json=initialized_notification)
            
            # List available capabilities
            await self._list_server_capabilities()
            
            self.status = MCPServerStatus.CONNECTED
            logger.info(f"Connected to HTTP MCP server {self.config.server_id}")
            return True
            
        except Exception as e:
            if self.http_client:
                await self.http_client.aclose()
                self.http_client = None
            raise e
    
    async def _connect_websocket(self) -> bool:
        """Connect to WebSocket-based MCP server"""
        # WebSocket implementation would go here
        # For now, raise not implemented
        raise NotImplementedError("WebSocket MCP servers not yet implemented")
    
    async def _list_server_capabilities(self):
        """List and cache server capabilities"""
        try:
            # List tools
            tools_response = await self._send_request("tools/list", {})
            if tools_response and "tools" in tools_response:
                for tool_info in tools_response["tools"]:
                    tool = MCPTool(
                        name=tool_info["name"],
                        description=tool_info.get("description", ""),
                        input_schema=tool_info.get("inputSchema", {}),
                        server_id=self.config.server_id
                    )
                    self.tools[tool.name] = tool
                    
            # List resources
            resources_response = await self._send_request("resources/list", {})
            if resources_response and "resources" in resources_response:
                for resource_info in resources_response["resources"]:
                    resource = MCPResource(
                        uri=resource_info["uri"],
                        name=resource_info.get("name", ""),
                        description=resource_info.get("description"),
                        mime_type=resource_info.get("mimeType"),
                        server_id=self.config.server_id
                    )
                    self.resources[resource.uri] = resource
                    
            # List prompts
            prompts_response = await self._send_request("prompts/list", {})
            if prompts_response and "prompts" in prompts_response:
                for prompt_info in prompts_response["prompts"]:
                    prompt = MCPPrompt(
                        name=prompt_info["name"],
                        description=prompt_info.get("description", ""),
                        arguments=prompt_info.get("arguments", []),
                        server_id=self.config.server_id
                    )
                    self.prompts[prompt.name] = prompt
                    
            logger.info(f"Listed capabilities for {self.config.server_id}: "
                       f"{len(self.tools)} tools, {len(self.resources)} resources, "
                       f"{len(self.prompts)} prompts")
                       
        except Exception as e:
            logger.warning(f"Failed to list capabilities for {self.config.server_id}: {e}")
    
    async def _send_request(self, method: str, params: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """Send JSON-RPC request to MCP server"""
        request = {
            "jsonrpc": "2.0",
            "id": str(uuid.uuid4()),
            "method": method,
            "params": params
        }
        
        try:
            if self.config.type == MCPServerType.STDIO:
                return await self._send_stdio_request(request)
            elif self.config.type == MCPServerType.HTTP:
                return await self._send_http_request(request)
            else:
                raise NotImplementedError(f"Request sending not implemented for {self.config.type}")
                
        except Exception as e:
            logger.error(f"Failed to send request to {self.config.server_id}: {e}")
            return None
    
    async def _send_stdio_request(self, request: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """Send request via stdio"""
        if not self.process or self.process.poll() is not None:
            raise Exception("Process not running")
            
        self.process.stdin.write(json.dumps(request) + "\n")
        self.process.stdin.flush()
        
        response_line = self.process.stdout.readline()
        if not response_line:
            raise Exception("No response from server")
            
        response = json.loads(response_line.strip())
        if "error" in response:
            raise Exception(f"Server error: {response['error']}")
            
        return response.get("result")
    
    async def _send_http_request(self, request: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """Send request via HTTP"""
        if not self.http_client:
            raise Exception("HTTP client not initialized")
            
        response = await self.http_client.post("/mcp", json=request)
        response.raise_for_status()
        
        result = response.json()
        if "error" in result:
            raise Exception(f"Server error: {result['error']}")
            
        return result.get("result")
    
    async def call_tool(self, tool_name: str, arguments: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """Call a tool on the MCP server"""
        if tool_name not in self.tools:
            raise ValueError(f"Tool {tool_name} not found on server {self.config.server_id}")
            
        return await self._send_request("tools/call", {
            "name": tool_name,
            "arguments": arguments
        })
    
    async def read_resource(self, uri: str) -> Optional[Dict[str, Any]]:
        """Read a resource from the MCP server"""
        return await self._send_request("resources/read", {"uri": uri})
    
    async def get_prompt(self, name: str, arguments: Optional[Dict[str, Any]] = None) -> Optional[Dict[str, Any]]:
        """Get a prompt from the MCP server"""
        params = {"name": name}
        if arguments:
            params["arguments"] = arguments
            
        return await self._send_request("prompts/get", params)
    
    async def disconnect(self):
        """Disconnect from the MCP server"""
        try:
            if self.process:
                self.process.terminate()
                try:
                    self.process.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    self.process.kill()
                self.process = None
                
            if self.http_client:
                await self.http_client.aclose()
                self.http_client = None
                
            self.status = MCPServerStatus.DISCONNECTED
            self.tools.clear()
            self.resources.clear()
            self.prompts.clear()
            
        except Exception as e:
            logger.error(f"Error disconnecting from {self.config.server_id}: {e}")


class MCPClientManager:
    """Manager for multiple MCP clients"""
    
    def __init__(self, config_path: Optional[str] = None):
        self.config_path = config_path or "mcp_servers.json"
        self.clients: Dict[str, MCPClient] = {}
        self.configurations: Dict[str, MCPServerConfig] = {}
        
    async def initialize(self):
        """Initialize the MCP client manager"""
        await self.load_configurations()
        await self.connect_all_servers()
    
    async def load_configurations(self):
        """Load MCP server configurations from file"""
        try:
            config_file = Path(self.config_path)
            if config_file.exists():
                with open(config_file, 'r') as f:
                    configs_data = json.load(f)
                    
                # Clear existing configurations
                self.configurations.clear()
                    
                for config_data in configs_data.get("servers", []):
                    config = MCPServerConfig(**config_data)
                    self.configurations[config.server_id] = config
                    
                logger.info(f"Loaded {len(self.configurations)} MCP server configurations")
                
                # Log each server's status
                for server_id, config in self.configurations.items():
                    status = "enabled" if config.enabled else "disabled"
                    logger.info(f"  - {config.name} ({server_id}): {status}")
                    
            else:
                logger.info("No MCP configuration file found, starting with empty configuration")
                await self._create_default_config()
                
        except Exception as e:
            logger.error(f"Failed to load MCP configurations: {e}")
            await self._create_default_config()
    
    async def _create_default_config(self):
        """Create a default configuration file with examples"""
        default_config = {
            "version": "1.0.0",
            "servers": [
                {
                    "server_id": "filesystem-example",
                    "name": "Filesystem Server",
                    "description": "Example filesystem MCP server",
                    "type": "stdio",
                    "command": "npx",
                    "args": ["-y", "@modelcontextprotocol/server-filesystem", "/path/to/allowed/directory"],
                    "enabled": False,
                    "timeout": 30
                },
                {
                    "server_id": "web-search-example", 
                    "name": "Web Search Server",
                    "description": "Example web search MCP server",
                    "type": "http",
                    "url": "http://localhost:3001",
                    "enabled": False,
                    "timeout": 30
                }
            ]
        }
        
        try:
            with open(self.config_path, 'w') as f:
                json.dump(default_config, f, indent=2)
            logger.info(f"Created default MCP configuration at {self.config_path}")
        except Exception as e:
            logger.error(f"Failed to create default configuration: {e}")
    
    async def save_configurations(self):
        """Save current configurations to file"""
        try:
            config_data = {
                "version": "1.0.0",
                "servers": [config.dict() for config in self.configurations.values()]
            }
            
            with open(self.config_path, 'w') as f:
                json.dump(config_data, f, indent=2)
                
            logger.info(f"Saved {len(self.configurations)} MCP server configurations")
            
        except Exception as e:
            logger.error(f"Failed to save MCP configurations: {e}")
    
    async def add_server(self, config: MCPServerConfig) -> bool:
        """Add a new MCP server configuration"""
        try:
            if config.server_id in self.configurations:
                raise ValueError(f"Server {config.server_id} already exists")
                
            self.configurations[config.server_id] = config
            await self.save_configurations()
            
            if config.enabled:
                await self.connect_server(config.server_id)
                
            return True
            
        except Exception as e:
            logger.error(f"Failed to add MCP server {config.server_id}: {e}")
            return False
    
    async def remove_server(self, server_id: str) -> bool:
        """Remove an MCP server configuration"""
        try:
            if server_id in self.clients:
                await self.disconnect_server(server_id)
                
            if server_id in self.configurations:
                del self.configurations[server_id]
                await self.save_configurations()
                return True
            else:
                return False
                
        except Exception as e:
            logger.error(f"Failed to remove MCP server {server_id}: {e}")
            return False
    
    async def update_server(self, server_id: str, config: MCPServerConfig) -> bool:
        """Update an MCP server configuration"""
        try:
            if server_id not in self.configurations:
                raise ValueError(f"Server {server_id} not found")
                
            # Disconnect if currently connected
            if server_id in self.clients:
                await self.disconnect_server(server_id)
                
            # Update configuration
            self.configurations[server_id] = config
            await self.save_configurations()
            
            # Reconnect if enabled
            if config.enabled:
                await self.connect_server(server_id)
                
            return True
            
        except Exception as e:
            logger.error(f"Failed to update MCP server {server_id}: {e}")
            return False
    
    async def connect_server(self, server_id: str) -> bool:
        """Connect to a specific MCP server"""
        try:
            if server_id not in self.configurations:
                raise ValueError(f"Server {server_id} not configured")
                
            config = self.configurations[server_id]
            if not config.enabled:
                logger.warning(f"Server {server_id} is disabled")
                # Allow connection attempt even if disabled, but warn user
                logger.info(f"Attempting to connect to disabled server {server_id}")
                
            if server_id in self.clients:
                await self.disconnect_server(server_id)
                
            client = MCPClient(config)
            success = await client.connect()
            
            if success:
                self.clients[server_id] = client
                logger.info(f"Successfully connected to MCP server {server_id}")
            else:
                logger.error(f"Failed to connect to MCP server {server_id}: Connection attempt failed")
            
            return success
            
        except Exception as e:
            logger.error(f"Failed to connect to MCP server {server_id}: {e}")
            return False
    
    async def disconnect_server(self, server_id: str):
        """Disconnect from a specific MCP server"""
        if server_id in self.clients:
            await self.clients[server_id].disconnect()
            del self.clients[server_id]
            logger.info(f"Disconnected from MCP server {server_id}")
    
    async def connect_all_servers(self):
        """Connect to all enabled MCP servers"""
        for server_id, config in self.configurations.items():
            if config.enabled:
                await self.connect_server(server_id)
    
    async def disconnect_all_servers(self):
        """Disconnect from all MCP servers"""
        for server_id in list(self.clients.keys()):
            await self.disconnect_server(server_id)
    
    def get_all_tools(self) -> List[MCPTool]:
        """Get all available tools from all connected servers"""
        tools = []
        for client in self.clients.values():
            if client.status == MCPServerStatus.CONNECTED:
                tools.extend(client.tools.values())
        return tools
    
    def get_all_resources(self) -> List[MCPResource]:
        """Get all available resources from all connected servers"""
        resources = []
        for client in self.clients.values():
            if client.status == MCPServerStatus.CONNECTED:
                resources.extend(client.resources.values())
        return resources
    
    def get_all_prompts(self) -> List[MCPPrompt]:
        """Get all available prompts from all connected servers"""
        prompts = []
        for client in self.clients.values():
            if client.status == MCPServerStatus.CONNECTED:
                prompts.extend(client.prompts.values())
        return prompts
    
    async def call_tool(self, tool_name: str, arguments: Dict[str, Any], server_id: Optional[str] = None) -> Optional[Dict[str, Any]]:
        """Call a tool, optionally specifying the server"""
        try:
            if server_id:
                if server_id not in self.clients:
                    raise ValueError(f"Server {server_id} not connected")
                return await self.clients[server_id].call_tool(tool_name, arguments)
            else:
                # Find the first server that has this tool
                for client in self.clients.values():
                    if client.status == MCPServerStatus.CONNECTED and tool_name in client.tools:
                        return await client.call_tool(tool_name, arguments)
                raise ValueError(f"Tool {tool_name} not found on any connected server")
                
        except Exception as e:
            logger.error(f"Failed to call tool {tool_name}: {e}")
            return None
    
    async def read_resource(self, uri: str, server_id: Optional[str] = None) -> Optional[Dict[str, Any]]:
        """Read a resource, optionally specifying the server"""
        try:
            if server_id:
                if server_id not in self.clients:
                    raise ValueError(f"Server {server_id} not connected")
                return await self.clients[server_id].read_resource(uri)
            else:
                # Find the first server that has this resource
                for client in self.clients.values():
                    if client.status == MCPServerStatus.CONNECTED and uri in client.resources:
                        return await client.read_resource(uri)
                raise ValueError(f"Resource {uri} not found on any connected server")
                
        except Exception as e:
            logger.error(f"Failed to read resource {uri}: {e}")
            return None
    
    async def get_prompt(self, name: str, arguments: Optional[Dict[str, Any]] = None, server_id: Optional[str] = None) -> Optional[Dict[str, Any]]:
        """Get a prompt, optionally specifying the server"""
        try:
            if server_id:
                if server_id not in self.clients:
                    raise ValueError(f"Server {server_id} not connected")
                return await self.clients[server_id].get_prompt(name, arguments)
            else:
                # Find the first server that has this prompt
                for client in self.clients.values():
                    if client.status == MCPServerStatus.CONNECTED and name in client.prompts:
                        return await client.get_prompt(name, arguments)
                raise ValueError(f"Prompt {name} not found on any connected server")
                
        except Exception as e:
            logger.error(f"Failed to get prompt {name}: {e}")
            return None
    
    def get_server_status(self) -> Dict[str, Dict[str, Any]]:
        """Get status of all configured servers"""
        status = {}
        
        for server_id, config in self.configurations.items():
            client = self.clients.get(server_id)
            
            status[server_id] = {
                "name": config.name,
                "type": config.type,
                "enabled": config.enabled,
                "status": client.status if client else MCPServerStatus.DISCONNECTED,
                "error": client.error_message if client else None,
                "tools_count": len(client.tools) if client else 0,
                "resources_count": len(client.resources) if client else 0,
                "prompts_count": len(client.prompts) if client else 0
            }
            
        return status
