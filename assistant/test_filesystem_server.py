#!/usr/bin/env python3
"""
Test MCP Filesystem Server Connection
"""
import asyncio
import subprocess
import shutil
import os
import sys
from pathlib import Path

async def test_filesystem_server():
    """Test the MCP filesystem server directly"""
    print("🧪 Testing MCP Filesystem Server...")
    
    # Check if npx is available
    if not shutil.which('npx'):
        print("❌ npx not found - install Node.js first")
        return False
    
    # Check if the directory exists
    test_dir = "/Users/kyle.butz/go/src/github.com/kbutz/wikillm/assistant/tmp"
    if not os.path.exists(test_dir):
        print(f"❌ Test directory not found: {test_dir}")
        return False
    
    print(f"✅ Test directory exists: {test_dir}")
    
    # Test the MCP server command
    cmd = [
        'npx', '-y', '@modelcontextprotocol/server-filesystem', test_dir
    ]
    
    print(f"🔧 Testing command: {' '.join(cmd)}")
    
    try:
        # Start the process
        process = subprocess.Popen(
            cmd,
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True
        )
        
        # Send initialization request
        init_request = {
            "jsonrpc": "2.0",
            "id": 1,
            "method": "initialize",
            "params": {
                "protocolVersion": "2024-11-05",
                "capabilities": {},
                "clientInfo": {
                    "name": "test-client",
                    "version": "1.0.0"
                }
            }
        }
        
        import json
        request_str = json.dumps(init_request) + "\\n"
        
        print("📤 Sending initialization request...")
        process.stdin.write(request_str)
        process.stdin.flush()
        
        # Read response with timeout
        import select
        
        if sys.platform != 'win32':
            # Unix-like systems
            ready, _, _ = select.select([process.stdout], [], [], 10)
            if ready:
                response = process.stdout.readline()
                if response:
                    print("📥 Received response:")
                    print(f"   {response.strip()}")
                    
                    # Try to parse the response
                    try:
                        parsed = json.loads(response.strip())
                        if "result" in parsed:
                            print("✅ MCP server initialized successfully!")
                            
                            # Send tools/list request
                            tools_request = {
                                "jsonrpc": "2.0",
                                "id": 2,
                                "method": "tools/list",
                                "params": {}
                            }
                            
                            tools_str = json.dumps(tools_request) + "\\n"
                            process.stdin.write(tools_str)
                            process.stdin.flush()
                            
                            # Read tools response
                            ready, _, _ = select.select([process.stdout], [], [], 5)
                            if ready:
                                tools_response = process.stdout.readline()
                                if tools_response:
                                    print("🔧 Tools available:")
                                    tools_parsed = json.loads(tools_response.strip())
                                    if "result" in tools_parsed and "tools" in tools_parsed["result"]:
                                        for tool in tools_parsed["result"]["tools"]:
                                            print(f"   - {tool['name']}: {tool.get('description', 'No description')}")
                                        
                                        result = True
                                    else:
                                        print("⚠️  No tools found in response")
                                        result = False
                                else:
                                    print("⚠️  No tools response received")
                                    result = False
                            else:
                                print("⚠️  Tools request timed out")
                                result = False
                        else:
                            print(f"❌ Error in response: {parsed}")
                            result = False
                    except json.JSONDecodeError as e:
                        print(f"❌ Invalid JSON response: {e}")
                        result = False
                else:
                    print("❌ No response received")
                    result = False
            else:
                print("❌ Request timed out")
                result = False
        else:
            # Windows - simpler approach
            try:
                stdout, stderr = process.communicate(input=request_str, timeout=10)
                if stdout:
                    print("📥 Received response:")
                    print(f"   {stdout.strip()}")
                    result = '"result"' in stdout
                else:
                    print("❌ No response received")
                    result = False
            except subprocess.TimeoutExpired:
                print("❌ Request timed out")
                result = False
        
        # Clean up
        process.terminate()
        return result
        
    except FileNotFoundError:
        print("❌ MCP server package not found")
        print("💡 Try installing: npx -y @modelcontextprotocol/server-filesystem")
        return False
    except Exception as e:
        print(f"❌ Error testing server: {e}")
        return False

async def test_server_auto_install():
    """Test auto-installing the MCP server"""
    print("\\n📦 Testing MCP server auto-install...")
    
    if not shutil.which('npx'):
        print("❌ npx not found - cannot auto-install")
        return False
    
    try:
        # Try to install/verify the server
        cmd = ['npx', '-y', '@modelcontextprotocol/server-filesystem', '--help']
        
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=30
        )
        
        if result.returncode == 0:
            print("✅ MCP filesystem server is available")
            return True
        else:
            print(f"❌ Server installation failed: {result.stderr}")
            return False
            
    except subprocess.TimeoutExpired:
        print("⚠️  Installation timed out (may still be working)")
        return False
    except Exception as e:
        print(f"❌ Installation error: {e}")
        return False

def check_configuration():
    """Check the MCP server configuration"""
    print("\\n⚙️  Checking MCP configuration...")
    
    config_file = Path("mcp_servers.json")
    if not config_file.exists():
        print("❌ mcp_servers.json not found")
        return False
    
    try:
        import json
        with open(config_file, 'r') as f:
            config = json.load(f)
        
        servers = config.get('servers', [])
        filesystem_servers = [s for s in servers if 'filesystem' in s['server_id']]
        
        if not filesystem_servers:
            print("❌ No filesystem servers configured")
            return False
        
        for server in filesystem_servers:
            print(f"📋 Server: {server['name']} ({server['server_id']})")
            print(f"   Enabled: {server.get('enabled', False)}")
            print(f"   Command: {server.get('command', 'N/A')}")
            print(f"   Args: {server.get('args', [])}")
            
            # Check if the directory exists
            if server.get('args'):
                dir_path = server['args'][-1]  # Last arg is usually the directory
                if os.path.exists(dir_path):
                    print(f"   ✅ Directory exists: {dir_path}")
                else:
                    print(f"   ❌ Directory missing: {dir_path}")
                    return False
        
        return True
        
    except Exception as e:
        print(f"❌ Configuration error: {e}")
        return False

async def main():
    """Main test function"""
    print("🔍 MCP Filesystem Server Test")
    print("=" * 40)
    
    # Check configuration
    config_ok = check_configuration()
    if not config_ok:
        print("\\n❌ Configuration check failed")
        return
    
    # Test auto-install
    install_ok = await test_server_auto_install()
    if not install_ok:
        print("\\n❌ Server installation test failed")
        return
    
    # Test server
    server_ok = await test_filesystem_server()
    
    print("\\n" + "=" * 40)
    if server_ok:
        print("✅ MCP filesystem server test passed!")
        print("\\n🎯 Next steps:")
        print("   1. Restart the assistant: python main.py")
        print("   2. Try connecting via the debug panel")
        print("   3. Test in a conversation: 'What files are in my tmp directory?'")
    else:
        print("❌ MCP filesystem server test failed")
        print("\\n🔧 Troubleshooting:")
        print("   1. Install Node.js: https://nodejs.org/")
        print("   2. Test manually: npx -y @modelcontextprotocol/server-filesystem /tmp")
        print("   3. Check directory permissions")
        print("   4. Review assistant.log for errors")

if __name__ == "__main__":
    asyncio.run(main())
