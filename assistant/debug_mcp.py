#!/usr/bin/env python3
"""
MCP Integration Debug Script
"""
import asyncio
import json
import sys
import os
from typing import Dict, Any, Optional
import httpx
from pathlib import Path

# Add the current directory to Python path
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from mcp_client_manager import MCPClientManager, MCPServerConfig, MCPServerType
from mcp_integration import get_mcp_tools_for_assistant, handle_mcp_tool_call

class MCPDebugger:
    def __init__(self, base_url: str = "http://localhost:8000"):
        self.base_url = base_url
        self.client = httpx.AsyncClient()
        
    async def check_system_status(self) -> Dict[str, Any]:
        """Check system status"""
        try:
            response = await self.client.get(f"{self.base_url}/status")
            response.raise_for_status()
            return response.json()
        except Exception as e:
            return {"error": str(e)}
    
    async def check_mcp_status(self) -> Dict[str, Any]:
        """Check MCP status"""
        try:
            response = await self.client.get(f"{self.base_url}/mcp/status")
            response.raise_for_status()
            return response.json()
        except Exception as e:
            return {"error": str(e)}
    
    async def list_mcp_servers(self) -> Dict[str, Any]:
        """List MCP servers"""
        try:
            response = await self.client.get(f"{self.base_url}/mcp/servers")
            response.raise_for_status()
            return response.json()
        except Exception as e:
            return {"error": str(e)}
    
    async def list_mcp_tools(self) -> Dict[str, Any]:
        """List MCP tools"""
        try:
            response = await self.client.get(f"{self.base_url}/mcp/tools")
            response.raise_for_status()
            return response.json()
        except Exception as e:
            return {"error": str(e)}
    
    async def test_mcp_tool(self, tool_name: str, arguments: Dict[str, Any], server_id: Optional[str] = None) -> Dict[str, Any]:
        """Test MCP tool"""
        try:
            payload = {
                "tool_name": tool_name,
                "arguments": arguments
            }
            if server_id:
                payload["server_id"] = server_id
                
            response = await self.client.post(f"{self.base_url}/mcp/tools/call", json=payload)
            response.raise_for_status()
            return response.json()
        except Exception as e:
            return {"error": str(e)}
    
    async def add_test_server(self, server_config: Dict[str, Any]) -> Dict[str, Any]:
        """Add a test server"""
        try:
            response = await self.client.post(f"{self.base_url}/mcp/servers", json=server_config)
            response.raise_for_status()
            return response.json()
        except Exception as e:
            return {"error": str(e)}
    
    async def connect_server(self, server_id: str) -> Dict[str, Any]:
        """Connect to a server"""
        try:
            response = await self.client.post(f"{self.base_url}/mcp/servers/{server_id}/connect")
            response.raise_for_status()
            return response.json()
        except Exception as e:
            return {"error": str(e)}
    
    async def test_direct_mcp_manager(self) -> Dict[str, Any]:
        """Test MCP manager directly"""
        try:
            # Initialize MCP manager
            manager = MCPClientManager()
            await manager.initialize()
            
            # Get status
            status = manager.get_server_status()
            tools = manager.get_all_tools()
            resources = manager.get_all_resources()
            
            return {
                "status": status,
                "tools_count": len(tools),
                "resources_count": len(resources),
                "tools": [{"name": t.name, "server_id": t.server_id} for t in tools],
                "resources": [{"uri": r.uri, "server_id": r.server_id} for r in resources]
            }
        except Exception as e:
            return {"error": str(e)}
    
    async def run_full_debug(self) -> Dict[str, Any]:
        """Run full debug sequence"""
        results = {}
        
        print("🔍 Starting MCP Debug Sequence...")
        
        # 1. Check system status
        print("1. Checking system status...")
        results["system_status"] = await self.check_system_status()
        
        # 2. Check MCP status
        print("2. Checking MCP status...")
        results["mcp_status"] = await self.check_mcp_status()
        
        # 3. List servers
        print("3. Listing MCP servers...")
        results["servers"] = await self.list_mcp_servers()
        
        # 4. List tools
        print("4. Listing MCP tools...")
        results["tools"] = await self.list_mcp_tools()
        
        # 5. Test direct manager
        print("5. Testing direct MCP manager...")
        results["direct_manager"] = await self.test_direct_mcp_manager()
        
        # 6. Test example server (if we can add one)
        print("6. Testing example server setup...")
        example_server = {
            "server_id": "debug-test",
            "name": "Debug Test Server",
            "description": "Test server for debugging",
            "type": "stdio",
            "command": "echo",
            "args": ["hello from mcp"],
            "enabled": False,  # Don't enable by default
            "timeout": 5
        }
        results["add_server_test"] = await self.add_test_server(example_server)
        
        return results
    
    def print_results(self, results: Dict[str, Any]):
        """Print debug results in a formatted way"""
        print("\\n" + "="*60)
        print("🔍 MCP DEBUG RESULTS")
        print("="*60)
        
        # System Status
        print("\\n📊 SYSTEM STATUS:")
        if "error" in results.get("system_status", {}):
            print(f"  ❌ Error: {results['system_status']['error']}")
        else:
            sys_status = results.get("system_status", {})
            print(f"  • Status: {sys_status.get('status', 'unknown')}")
            print(f"  • LMStudio Connected: {sys_status.get('lmstudio_connected', False)}")
            print(f"  • MCP Servers: {sys_status.get('mcp_servers_connected', 0)}/{sys_status.get('mcp_servers_total', 0)}")
            print(f"  • MCP Tools: {sys_status.get('mcp_tools_available', 0)}")
        
        # MCP Status
        print("\\n🔌 MCP STATUS:")
        if "error" in results.get("mcp_status", {}):
            print(f"  ❌ Error: {results['mcp_status']['error']}")
        else:
            mcp_status = results.get("mcp_status", {})
            if mcp_status.get("success"):
                data = mcp_status.get("data", {})
                print(f"  • Total Servers: {data.get('total_servers', 0)}")
                print(f"  • Connected Servers: {data.get('connected_servers', 0)}")
                servers = data.get("servers", {})
                for server_id, server_info in servers.items():
                    print(f"    - {server_id}: {server_info.get('status', 'unknown')} ({server_info.get('tools_count', 0)} tools)")
        
        # Servers
        print("\\n🖥️  MCP SERVERS:")
        if "error" in results.get("servers", {}):
            print(f"  ❌ Error: {results['servers']['error']}")
        else:
            servers = results.get("servers", {})
            if servers.get("success"):
                server_list = servers.get("data", {}).get("servers", [])
                if server_list:
                    for server in server_list:
                        print(f"  • {server['name']} ({server['server_id']})")
                        print(f"    Status: {server['status']}")
                        print(f"    Tools: {server['capabilities']['tools']}")
                        if server.get('error'):
                            print(f"    Error: {server['error']}")
                else:
                    print("  📝 No servers configured")
        
        # Tools
        print("\\n⚡ MCP TOOLS:")
        if "error" in results.get("tools", {}):
            print(f"  ❌ Error: {results['tools']['error']}")
        else:
            tools = results.get("tools", {})
            if tools.get("success"):
                tool_list = tools.get("data", {}).get("tools", [])
                if tool_list:
                    for tool in tool_list:
                        print(f"  • {tool['name']} (from {tool['server_id']})")
                        print(f"    Description: {tool['description']}")
                else:
                    print("  📝 No tools available")
        
        # Direct Manager Test
        print("\\n🔧 DIRECT MANAGER TEST:")
        if "error" in results.get("direct_manager", {}):
            print(f"  ❌ Error: {results['direct_manager']['error']}")
        else:
            direct = results.get("direct_manager", {})
            print(f"  • Tools Count: {direct.get('tools_count', 0)}")
            print(f"  • Resources Count: {direct.get('resources_count', 0)}")
            print(f"  • Server Status: {direct.get('status', {})}")
        
        print("\\n" + "="*60)
        
        # Recommendations
        print("\\n💡 RECOMMENDATIONS:")
        
        # Check if MCP is working
        mcp_working = (
            "error" not in results.get("mcp_status", {}) and
            results.get("mcp_status", {}).get("success", False)
        )
        
        if not mcp_working:
            print("  ❌ MCP integration is not working properly")
            print("  📋 To fix:")
            print("     1. Ensure the assistant is running: python main.py")
            print("     2. Check that MCP files are in place")
            print("     3. Verify requirements are installed: pip install -r requirements.txt")
        else:
            servers_connected = results.get("mcp_status", {}).get("data", {}).get("connected_servers", 0)
            if servers_connected == 0:
                print("  ⚠️  MCP is working but no servers are connected")
                print("  📋 To add servers:")
                print("     1. Edit mcp_servers.json to configure servers")
                print("     2. Enable at least one server by setting 'enabled': true")
                print("     3. Ensure Node.js is installed for stdio servers")
                print("     4. Test with: npx -y @modelcontextprotocol/server-filesystem /tmp")
            else:
                print("  ✅ MCP integration is working correctly!")
                tools_count = results.get("tools", {}).get("data", {}).get("total_count", 0)
                print(f"     • {servers_connected} servers connected")
                print(f"     • {tools_count} tools available")
                print("     • Ready for use in conversations")
        
        print("\\n🔗 NEXT STEPS:")
        print("  1. Check the frontend debug panel for real-time status")
        print("  2. Try chatting with the assistant to test tool usage")
        print("  3. Monitor assistant.log for detailed error messages")
        print("\\n")

async def main():
    """Main debug function"""
    debugger = MCPDebugger()
    
    try:
        results = await debugger.run_full_debug()
        debugger.print_results(results)
        
        # Save results to file
        with open("mcp_debug_results.json", "w") as f:
            json.dump(results, f, indent=2, default=str)
        print("📄 Debug results saved to mcp_debug_results.json")
        
    except KeyboardInterrupt:
        print("\\n🛑 Debug interrupted by user")
    except Exception as e:
        print(f"\\n❌ Debug failed: {e}")
        import traceback
        traceback.print_exc()
    finally:
        await debugger.client.aclose()

if __name__ == "__main__":
    asyncio.run(main())
