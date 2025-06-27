#!/usr/bin/env python3
"""
Simple MCP Test Script
"""
import asyncio
import json
import sys
import os
from pathlib import Path

# Add the current directory to Python path
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

async def test_basic_mcp():
    """Test basic MCP functionality"""
    print("🧪 Testing Basic MCP Functionality...")
    
    try:
        # Test 1: Import MCP modules
        print("1. Testing imports...")
        from mcp_client_manager import MCPClientManager, MCPServerConfig, MCPServerType
        from mcp_integration import get_mcp_tools_for_assistant
        print("   ✅ MCP modules imported successfully")
        
        # Test 2: Initialize MCP manager
        print("2. Testing MCP manager initialization...")
        manager = MCPClientManager()
        await manager.initialize()
        print("   ✅ MCP manager initialized")
        
        # Test 3: Check configuration file
        print("3. Checking configuration file...")
        config_path = Path("mcp_servers.json")
        if config_path.exists():
            with open(config_path, 'r') as f:
                config = json.load(f)
            print(f"   ✅ Configuration loaded: {len(config.get('servers', []))} servers configured")
            
            # Show server configurations
            for server in config.get('servers', []):
                status = "enabled" if server.get('enabled') else "disabled"
                print(f"      - {server['name']} ({server['server_id']}): {status}")
        else:
            print("   ⚠️  No configuration file found")
        
        # Test 4: Check server status
        print("4. Checking server status...")
        status = manager.get_server_status()
        if status:
            connected = sum(1 for s in status.values() if s.get('status') == 'connected')
            total = len(status)
            print(f"   📊 Server status: {connected}/{total} connected")
            
            for server_id, server_info in status.items():
                print(f"      - {server_id}: {server_info.get('status', 'unknown')}")
        else:
            print("   📝 No servers configured")
        
        # Test 5: Check available tools
        print("5. Checking available tools...")
        tools = manager.get_all_tools()
        print(f"   🔧 Available tools: {len(tools)}")
        
        for tool in tools:
            print(f"      - {tool.name} (from {tool.server_id})")
        
        # Test 6: Check assistant tool integration
        print("6. Testing assistant tool integration...")
        assistant_tools = get_mcp_tools_for_assistant()
        print(f"   🤖 Assistant-ready tools: {len(assistant_tools)}")
        
        # Test 7: Check resources
        print("7. Checking available resources...")
        resources = manager.get_all_resources()
        print(f"   📁 Available resources: {len(resources)}")
        
        for resource in resources:
            print(f"      - {resource.name} ({resource.uri})")
        
        return True
        
    except ImportError as e:
        print(f"   ❌ Import error: {e}")
        print("   💡 Make sure all MCP files are in place")
        return False
    except Exception as e:
        print(f"   ❌ Error: {e}")
        import traceback
        traceback.print_exc()
        return False

async def test_sample_server():
    """Test adding a sample server"""
    print("\n🔧 Testing Sample Server Setup...")
    
    try:
        from mcp_client_manager import MCPClientManager, MCPServerConfig, MCPServerType
        
        # Test simple echo server
        print("1. Testing simple echo server...")
        
        # Check if we can run a simple command
        import subprocess
        import shutil
        
        # Check if basic commands are available
        if shutil.which('echo'):
            print("   ✅ echo command available")
        else:
            print("   ❌ echo command not found")
            return False
        
        # Check if node/npm is available for real MCP servers
        if shutil.which('node'):
            print("   ✅ Node.js available")
            
            # Test if we can run npx
            if shutil.which('npx'):
                print("   ✅ npx available")
                
                # Test if we can access MCP server
                try:
                    result = subprocess.run([
                        'npx', '--help'
                    ], capture_output=True, text=True, timeout=10)
                    
                    if result.returncode == 0:
                        print("   ✅ npx working correctly")
                    else:
                        print("   ⚠️  npx available but may have issues")
                        
                except subprocess.TimeoutExpired:
                    print("   ⚠️  npx command timed out")
                except Exception as e:
                    print(f"   ⚠️  npx test failed: {e}")
            else:
                print("   ❌ npx not found - needed for most MCP servers")
        else:
            print("   ❌ Node.js not found - needed for most MCP servers")
            print("   💡 Install Node.js: https://nodejs.org/")
        
        return True
        
    except Exception as e:
        print(f"   ❌ Error: {e}")
        return False

def check_prerequisites():
    """Check system prerequisites"""
    print("🔍 Checking Prerequisites...")
    
    issues = []
    
    # Check Python version
    python_version = sys.version_info
    if python_version >= (3, 8):
        print(f"   ✅ Python {python_version.major}.{python_version.minor}.{python_version.micro}")
    else:
        print(f"   ❌ Python {python_version.major}.{python_version.minor}.{python_version.micro} (need 3.8+)")
        issues.append("Python version too old")
    
    # Check required packages
    required_packages = ['httpx', 'pydantic', 'asyncio']
    for package in required_packages:
        try:
            __import__(package)
            print(f"   ✅ {package} available")
        except ImportError:
            print(f"   ❌ {package} not found")
            issues.append(f"Missing package: {package}")
    
    # Check files exist
    required_files = [
        'mcp_client_manager.py',
        'mcp_integration.py', 
        'enhanced_conversation_manager.py',
        'main.py'
    ]
    
    for file in required_files:
        if Path(file).exists():
            print(f"   ✅ {file}")
        else:
            print(f"   ❌ {file} not found")
            issues.append(f"Missing file: {file}")
    
    return len(issues) == 0, issues

def provide_setup_guidance():
    """Provide setup guidance"""
    print("\n📋 SETUP GUIDANCE:")
    print("="*50)
    
    print("\n1. 📁 Ensure all MCP files are in place:")
    print("   - mcp_client_manager.py")
    print("   - mcp_integration.py")
    print("   - enhanced_conversation_manager.py")
    print("   - mcp_servers.json")
    
    print("\n2. 📦 Install required packages:")
    print("   pip install -r requirements.txt")
    
    print("\n3. 🟢 Install Node.js (for MCP servers):")
    print("   - macOS: brew install node")
    print("   - Ubuntu: sudo apt install nodejs npm")
    print("   - Windows: Download from https://nodejs.org/")
    
    print("\n4. ⚙️  Configure MCP servers:")
    print("   - Edit mcp_servers.json")
    print("   - Set 'enabled': true for servers you want to use")
    print("   - Update paths and API keys as needed")
    
    print("\n5. 🧪 Test a simple server:")
    print("   npx -y @modelcontextprotocol/server-filesystem /tmp")
    
    print("\n6. 🚀 Start the assistant:")
    print("   python main.py")
    
    print("\n7. 🔍 Use the debug panel in the frontend")
    print("   - Click the settings icon in the sidebar")
    print("   - Check server status and test connections")

async def main():
    """Main test function"""
    print("🔍 MCP Integration Test")
    print("=" * 50)
    
    # Check prerequisites first
    prereqs_ok, issues = check_prerequisites()
    
    if not prereqs_ok:
        print(f"\n❌ Prerequisites check failed with {len(issues)} issues:")
        for issue in issues:
            print(f"   - {issue}")
        provide_setup_guidance()
        return
    
    print("\n✅ Prerequisites check passed!")
    
    # Test basic MCP functionality
    mcp_ok = await test_basic_mcp()
    
    if not mcp_ok:
        print("\n❌ Basic MCP test failed")
        provide_setup_guidance()
        return
    
    # Test sample server setup
    server_ok = await test_sample_server()
    
    # Summary
    print("\n" + "="*50)
    print("📊 TEST SUMMARY")
    print("="*50)
    
    if mcp_ok and server_ok:
        print("✅ All tests passed!")
        print("\n🎉 MCP integration is ready to use!")
        print("\n🔗 Next steps:")
        print("   1. Start the assistant: python main.py")
        print("   2. Open the frontend debug panel")
        print("   3. Configure and enable MCP servers")
        print("   4. Test tools in conversations")
    else:
        print("⚠️  Some tests had issues")
        provide_setup_guidance()
    
    print(f"\n📄 For detailed debugging, run: python debug_mcp.py")

if __name__ == "__main__":
    asyncio.run(main())
