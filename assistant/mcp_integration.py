"""
FastAPI integration for MCP (Model Context Protocol) functionality
"""
import json
import logging
import os
from typing import Dict, List, Optional, Any
from fastapi import FastAPI, HTTPException, Depends, BackgroundTasks, status
from fastapi.responses import JSONResponse
from pydantic import BaseModel
from sqlalchemy.orm import Session

from mcp_client_manager import (
    MCPClientManager, MCPServerConfig, MCPServerType, MCPServerStatus,
    MCPTool, MCPResource, MCPPrompt
)
from database import get_db

logger = logging.getLogger(__name__)

# Global MCP client manager instance
mcp_manager: Optional[MCPClientManager] = None


class MCPServerConfigRequest(BaseModel):
    """Request model for MCP server configuration"""
    server_id: str
    name: str
    description: Optional[str] = None
    type: MCPServerType
    command: Optional[str] = None
    args: Optional[List[str]] = None
    url: Optional[str] = None
    env: Optional[Dict[str, str]] = None
    timeout: int = 30
    enabled: bool = True
    auto_reconnect: bool = True


class MCPToolCallRequest(BaseModel):
    """Request model for calling MCP tools"""
    tool_name: str
    arguments: Dict[str, Any]
    server_id: Optional[str] = None


class MCPResourceReadRequest(BaseModel):
    """Request model for reading MCP resources"""
    uri: str
    server_id: Optional[str] = None


class MCPPromptGetRequest(BaseModel):
    """Request model for getting MCP prompts"""
    name: str
    arguments: Optional[Dict[str, Any]] = None
    server_id: Optional[str] = None


class MCPResponse(BaseModel):
    """Generic MCP response model"""
    success: bool
    data: Optional[Dict[str, Any]] = None
    error: Optional[str] = None


async def get_mcp_manager() -> MCPClientManager:
    """Dependency to get MCP client manager"""
    global mcp_manager
    if mcp_manager is None:
        mcp_manager = MCPClientManager()
        await mcp_manager.initialize()
    return mcp_manager


def register_mcp_routes(app: FastAPI):
    """Register MCP-related routes with the FastAPI app"""
    
    @app.get("/mcp/status")
    async def get_mcp_status(manager: MCPClientManager = Depends(get_mcp_manager)):
        """Get status of all MCP servers"""
        try:
            status = manager.get_server_status()
            return JSONResponse(
                status_code=200,
                content={
                    "success": True,
                    "data": {
                        "servers": status,
                        "total_servers": len(status),
                        "connected_servers": sum(1 for s in status.values() if s["status"] == MCPServerStatus.CONNECTED)
                    }
                }
            )
        except Exception as e:
            logger.error(f"Failed to get MCP status: {e}")
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail=f"Failed to get MCP status: {str(e)}"
            )
    
    @app.get("/mcp/servers")
    async def list_mcp_servers(manager: MCPClientManager = Depends(get_mcp_manager)):
        """List all configured MCP servers"""
        try:
            servers = []
            for server_id, config in manager.configurations.items():
                client = manager.clients.get(server_id)
                servers.append({
                    "server_id": server_id,
                    "name": config.name,
                    "description": config.description,
                    "type": config.type,
                    "enabled": config.enabled,
                    "status": client.status if client else MCPServerStatus.DISCONNECTED,
                    "error": client.error_message if client else None,
                    "capabilities": {
                        "tools": len(client.tools) if client else 0,
                        "resources": len(client.resources) if client else 0,
                        "prompts": len(client.prompts) if client else 0
                    }
                })
            
            return JSONResponse(
                status_code=200,
                content={"success": True, "data": {"servers": servers}}
            )
        except Exception as e:
            logger.error(f"Failed to list MCP servers: {e}")
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail=f"Failed to list MCP servers: {str(e)}"
            )
    
    @app.post("/mcp/servers")
    async def add_mcp_server(
        config_request: MCPServerConfigRequest,
        manager: MCPClientManager = Depends(get_mcp_manager)
    ):
        """Add a new MCP server configuration"""
        try:
            config = MCPServerConfig(**config_request.dict())
            success = await manager.add_server(config)
            
            if success:
                return JSONResponse(
                    status_code=201,
                    content={
                        "success": True,
                        "data": {"message": f"MCP server {config.server_id} added successfully"}
                    }
                )
            else:
                raise HTTPException(
                    status_code=status.HTTP_400_BAD_REQUEST,
                    detail=f"Failed to add MCP server {config.server_id}"
                )
        except Exception as e:
            logger.error(f"Failed to add MCP server: {e}")
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail=f"Failed to add MCP server: {str(e)}"
            )
    
    @app.put("/mcp/servers/{server_id}")
    async def update_mcp_server(
        server_id: str,
        config_request: MCPServerConfigRequest,
        manager: MCPClientManager = Depends(get_mcp_manager)
    ):
        """Update an MCP server configuration"""
        try:
            config = MCPServerConfig(**config_request.dict())
            success = await manager.update_server(server_id, config)
            
            if success:
                return JSONResponse(
                    status_code=200,
                    content={
                        "success": True,
                        "data": {"message": f"MCP server {server_id} updated successfully"}
                    }
                )
            else:
                raise HTTPException(
                    status_code=status.HTTP_404_NOT_FOUND,
                    detail=f"MCP server {server_id} not found"
                )
        except Exception as e:
            logger.error(f"Failed to update MCP server {server_id}: {e}")
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail=f"Failed to update MCP server: {str(e)}"
            )
    
    @app.delete("/mcp/servers/{server_id}")
    async def remove_mcp_server(
        server_id: str,
        manager: MCPClientManager = Depends(get_mcp_manager)
    ):
        """Remove an MCP server configuration"""
        try:
            success = await manager.remove_server(server_id)
            
            if success:
                return JSONResponse(
                    status_code=200,
                    content={
                        "success": True,
                        "data": {"message": f"MCP server {server_id} removed successfully"}
                    }
                )
            else:
                raise HTTPException(
                    status_code=status.HTTP_404_NOT_FOUND,
                    detail=f"MCP server {server_id} not found"
                )
        except Exception as e:
            logger.error(f"Failed to remove MCP server {server_id}: {e}")
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail=f"Failed to remove MCP server: {str(e)}"
            )
    
    @app.post("/mcp/servers/{server_id}/connect")
    async def connect_mcp_server(
        server_id: str,
        manager: MCPClientManager = Depends(get_mcp_manager)
    ):
        """Connect to a specific MCP server"""
        try:
            # Check if server exists
            if server_id not in manager.configurations:
                raise HTTPException(
                    status_code=status.HTTP_404_NOT_FOUND,
                    detail=f"MCP server '{server_id}' not found in configuration"
                )
            
            config = manager.configurations[server_id]
            logger.info(f"Attempting to connect to MCP server {server_id} (enabled: {config.enabled})")
            
            success = await manager.connect_server(server_id)
            
            if success:
                return JSONResponse(
                    status_code=200,
                    content={
                        "success": True,
                        "data": {"message": f"Successfully connected to MCP server {server_id}"}
                    }
                )
            else:
                # Get more detailed error information
                client = manager.clients.get(server_id)
                error_detail = "Connection failed"
                if client and client.error_message:
                    error_detail = client.error_message
                elif not config.enabled:
                    error_detail = f"Server {server_id} is disabled in configuration"
                
                raise HTTPException(
                    status_code=status.HTTP_400_BAD_REQUEST,
                    detail=f"Failed to connect to MCP server {server_id}: {error_detail}"
                )
        except HTTPException:
            raise
        except Exception as e:
            logger.error(f"Failed to connect to MCP server {server_id}: {e}")
            import traceback
            traceback.print_exc()
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail=f"Failed to connect to MCP server: {str(e)}"
            )
    
    @app.post("/mcp/reload")
    async def reload_mcp_configuration(
        manager: MCPClientManager = Depends(get_mcp_manager)
    ):
        """Reload MCP server configurations from file"""
        try:
            logger.info("Reloading MCP server configurations...")
            
            # Disconnect all current servers
            await manager.disconnect_all_servers()
            
            # Reload configurations
            await manager.load_configurations()
            
            # Reconnect enabled servers
            await manager.connect_all_servers()
            
            return JSONResponse(
                status_code=200,
                content={
                    "success": True,
                    "data": {
                        "message": "MCP configurations reloaded successfully",
                        "total_servers": len(manager.configurations),
                        "enabled_servers": len([c for c in manager.configurations.values() if c.enabled])
                    }
                }
            )
        except Exception as e:
            logger.error(f"Failed to reload MCP configurations: {e}")
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail=f"Failed to reload MCP configurations: {str(e)}"
            )
    
    @app.post("/mcp/servers/{server_id}/disconnect")
    async def disconnect_mcp_server(
        server_id: str,
        manager: MCPClientManager = Depends(get_mcp_manager)
    ):
        """Disconnect from a specific MCP server"""
        try:
            await manager.disconnect_server(server_id)
            return JSONResponse(
                status_code=200,
                content={
                    "success": True,
                    "data": {"message": f"Disconnected from MCP server {server_id}"}
                }
            )
        except Exception as e:
            logger.error(f"Failed to disconnect from MCP server {server_id}: {e}")
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail=f"Failed to disconnect from MCP server: {str(e)}"
            )
    
    @app.get("/mcp/tools")
    async def list_mcp_tools(manager: MCPClientManager = Depends(get_mcp_manager)):
        """List all available MCP tools"""
        try:
            tools = manager.get_all_tools()
            tools_data = []
            
            for tool in tools:
                tools_data.append({
                    "name": tool.name,
                    "description": tool.description,
                    "input_schema": tool.input_schema,
                    "server_id": tool.server_id
                })
            
            return JSONResponse(
                status_code=200,
                content={
                    "success": True,
                    "data": {
                        "tools": tools_data,
                        "total_count": len(tools_data)
                    }
                }
            )
        except Exception as e:
            logger.error(f"Failed to list MCP tools: {e}")
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail=f"Failed to list MCP tools: {str(e)}"
            )
    
    @app.post("/mcp/tools/call")
    async def call_mcp_tool(
        request: MCPToolCallRequest,
        manager: MCPClientManager = Depends(get_mcp_manager)
    ):
        """Call an MCP tool"""
        try:
            result = await manager.call_tool(
                request.tool_name,
                request.arguments,
                request.server_id
            )
            
            if result is not None:
                return JSONResponse(
                    status_code=200,
                    content={
                        "success": True,
                        "data": result
                    }
                )
            else:
                raise HTTPException(
                    status_code=status.HTTP_400_BAD_REQUEST,
                    detail=f"Failed to call tool {request.tool_name}"
                )
        except Exception as e:
            logger.error(f"Failed to call MCP tool {request.tool_name}: {e}")
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail=f"Failed to call MCP tool: {str(e)}"
            )
    
    @app.get("/mcp/resources")
    async def list_mcp_resources(manager: MCPClientManager = Depends(get_mcp_manager)):
        """List all available MCP resources"""
        try:
            resources = manager.get_all_resources()
            resources_data = []
            
            for resource in resources:
                resources_data.append({
                    "uri": resource.uri,
                    "name": resource.name,
                    "description": resource.description,
                    "mime_type": resource.mime_type,
                    "server_id": resource.server_id
                })
            
            return JSONResponse(
                status_code=200,
                content={
                    "success": True,
                    "data": {
                        "resources": resources_data,
                        "total_count": len(resources_data)
                    }
                }
            )
        except Exception as e:
            logger.error(f"Failed to list MCP resources: {e}")
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail=f"Failed to list MCP resources: {str(e)}"
            )
    
    @app.post("/mcp/resources/read")
    async def read_mcp_resource(
        request: MCPResourceReadRequest,
        manager: MCPClientManager = Depends(get_mcp_manager)
    ):
        """Read an MCP resource"""
        try:
            result = await manager.read_resource(request.uri, request.server_id)
            
            if result is not None:
                return JSONResponse(
                    status_code=200,
                    content={
                        "success": True,
                        "data": result
                    }
                )
            else:
                raise HTTPException(
                    status_code=status.HTTP_404_NOT_FOUND,
                    detail=f"Resource {request.uri} not found"
                )
        except Exception as e:
            logger.error(f"Failed to read MCP resource {request.uri}: {e}")
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail=f"Failed to read MCP resource: {str(e)}"
            )
    
    @app.get("/mcp/prompts")
    async def list_mcp_prompts(manager: MCPClientManager = Depends(get_mcp_manager)):
        """List all available MCP prompts"""
        try:
            prompts = manager.get_all_prompts()
            prompts_data = []
            
            for prompt in prompts:
                prompts_data.append({
                    "name": prompt.name,
                    "description": prompt.description,
                    "arguments": prompt.arguments,
                    "server_id": prompt.server_id
                })
            
            return JSONResponse(
                status_code=200,
                content={
                    "success": True,
                    "data": {
                        "prompts": prompts_data,
                        "total_count": len(prompts_data)
                    }
                }
            )
        except Exception as e:
            logger.error(f"Failed to list MCP prompts: {e}")
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail=f"Failed to list MCP prompts: {str(e)}"
            )
    
    @app.post("/mcp/prompts/get")
    async def get_mcp_prompt(
        request: MCPPromptGetRequest,
        manager: MCPClientManager = Depends(get_mcp_manager)
    ):
        """Get an MCP prompt"""
        try:
            result = await manager.get_prompt(
                request.name,
                request.arguments,
                request.server_id
            )
            
            if result is not None:
                return JSONResponse(
                    status_code=200,
                    content={
                        "success": True,
                        "data": result
                    }
                )
            else:
                raise HTTPException(
                    status_code=status.HTTP_404_NOT_FOUND,
                    detail=f"Prompt {request.name} not found"
                )
        except Exception as e:
            logger.error(f"Failed to get MCP prompt {request.name}: {e}")
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail=f"Failed to get MCP prompt: {str(e)}"
            )


async def initialize_mcp_system():
    """Initialize the MCP system on application startup"""
    global mcp_manager
    try:
        logger.info("Initializing MCP system...")
        mcp_manager = MCPClientManager()
        await mcp_manager.initialize()
        logger.info("MCP system initialized successfully")
        
        # Log connected servers
        status = mcp_manager.get_server_status()
        connected_count = sum(1 for s in status.values() if s["status"] == MCPServerStatus.CONNECTED)
        logger.info(f"MCP: {connected_count}/{len(status)} servers connected")
        
    except Exception as e:
        logger.error(f"Failed to initialize MCP system: {e}")


async def shutdown_mcp_system():
    """Shutdown the MCP system on application shutdown"""
    global mcp_manager
    if mcp_manager:
        try:
            logger.info("Shutting down MCP system...")
            await mcp_manager.disconnect_all_servers()
            logger.info("MCP system shutdown complete")
        except Exception as e:
            logger.error(f"Error during MCP shutdown: {e}")


def get_mcp_tools_for_assistant() -> List[Dict[str, Any]]:
    """Get MCP tools formatted for the assistant's tool system"""
    global mcp_manager
    if not mcp_manager:
        return []
    
    assistant_tools = []
    mcp_tools = mcp_manager.get_all_tools()
    
    for tool in mcp_tools:
        # Convert MCP tool to assistant tool format
        assistant_tool = {
            "type": "function",
            "function": {
                "name": f"mcp_{tool.server_id}_{tool.name}",
                "description": f"[MCP {tool.server_id}] {tool.description}",
                "parameters": tool.input_schema
            },
            "mcp_server_id": tool.server_id,
            "mcp_tool_name": tool.name
        }
        assistant_tools.append(assistant_tool)
    
    return assistant_tools


async def handle_mcp_tool_call(tool_name: str, arguments: Dict[str, Any]) -> Dict[str, Any]:
    """Handle MCP tool call from the assistant"""
    global mcp_manager
    if not mcp_manager:
        raise ValueError("MCP system not initialized")
    
    # Extract server_id and tool_name from function name
    # Format: mcp_{server_id}_{tool_name}
    if not tool_name.startswith("mcp_"):
        raise ValueError(f"Invalid MCP tool name format: {tool_name}")
    
    parts = tool_name[4:].split("_", 1)  # Remove "mcp_" prefix
    if len(parts) != 2:
        raise ValueError(f"Invalid MCP tool name format: {tool_name}")
    
    server_id, actual_tool_name = parts
    
    try:
        result = await mcp_manager.call_tool(actual_tool_name, arguments, server_id)
        
        if result is None:
            return {
                "success": False,
                "error": f"Tool {actual_tool_name} on server {server_id} returned no result"
            }
        
        return {
            "success": True,
            "result": result,
            "server_id": server_id,
            "tool_name": actual_tool_name
        }
        
    except Exception as e:
        logger.error(f"MCP tool call failed: {e}")
        return {
            "success": False,
            "error": str(e),
            "server_id": server_id,
            "tool_name": actual_tool_name
        }
