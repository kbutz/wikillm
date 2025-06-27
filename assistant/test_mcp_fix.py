#!/usr/bin/env python3
"""
Test script to verify MCP integration fix
"""
import asyncio
import json
import sys
import logging
from typing import Dict, Any
import httpx

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

BASE_URL = "http://localhost:8000"  # Adjust if your server runs on different port

async def test_mcp_status():
    """Test MCP status endpoint"""
    async with httpx.AsyncClient() as client:
        try:
            response = await client.get(f"{BASE_URL}/mcp/status")
            if response.status_code == 200:
                data = response.json()
                logger.info("✅ MCP Status endpoint working")
                logger.info(f"   Connected servers: {data.get('data', {}).get('connected_servers', 0)}")
                logger.info(f"   Total servers: {data.get('data', {}).get('total_servers', 0)}")
                return True
            else:
                logger.error(f"❌ MCP Status failed: {response.status_code}")
                return False
        except Exception as e:
            logger.error(f"❌ MCP Status error: {e}")
            return False

async def test_system_status():
    """Test enhanced system status"""
    async with httpx.AsyncClient() as client:
        try:
            response = await client.get(f"{BASE_URL}/status")
            if response.status_code == 200:
                data = response.json()
                logger.info("✅ System status endpoint working")
                logger.info(f"   LMStudio connected: {data.get('lmstudio_connected', False)}")
                logger.info(f"   MCP servers connected: {data.get('mcp_servers_connected', 0)}")
                logger.info(f"   MCP tools available: {data.get('mcp_tools_available', 0)}")
                return True
            else:
                logger.error(f"❌ System Status failed: {response.status_code}")
                return False
        except Exception as e:
            logger.error(f"❌ System Status error: {e}")
            return False

async def test_mcp_tools():
    """Test MCP tools listing"""
    async with httpx.AsyncClient() as client:
        try:
            response = await client.get(f"{BASE_URL}/mcp/tools")
            if response.status_code == 200:
                data = response.json()
                tools_count = data.get('data', {}).get('total_count', 0)
                logger.info(f"✅ MCP Tools endpoint working - {tools_count} tools available")
                
                # List first few tools
                tools = data.get('data', {}).get('tools', [])
                for i, tool in enumerate(tools[:3]):
                    logger.info(f"   Tool {i+1}: {tool.get('name', 'unknown')}")
                
                return True
            else:
                logger.error(f"❌ MCP Tools failed: {response.status_code}")
                return False
        except Exception as e:
            logger.error(f"❌ MCP Tools error: {e}")
            return False

async def test_chat_with_tools():
    """Test chat endpoint with MCP tools"""
    async with httpx.AsyncClient(timeout=30.0) as client:
        try:
            # First create a user
            user_response = await client.post(f"{BASE_URL}/users/", json={
                "username": "test_user_mcp",
                "email": "test.mcp@example.com",
                "full_name": "MCP Test User"
            })
            
            if user_response.status_code not in [200, 201, 400]:  # 400 if user exists
                logger.error(f"❌ User creation failed: {user_response.status_code}")
                return False
            
            user_data = user_response.json()
            user_id = user_data.get('id', 1)  # Default to 1 if user already exists
            
            # Test chat request that might trigger tool usage
            chat_request = {
                "user_id": user_id,
                "message": "Can you help me list the files in the current directory?",
                "temperature": 0.7,
                "max_tokens": 500
            }
            
            logger.info("🔄 Testing chat with MCP tools...")
            response = await client.post(f"{BASE_URL}/chat", json=chat_request)
            
            if response.status_code == 200:
                data = response.json()
                logger.info("✅ Chat endpoint working with MCP tools")
                logger.info(f"   Response length: {len(data.get('message', {}).get('content', ''))}")
                logger.info(f"   Processing time: {data.get('processing_time', 0):.2f}s")
                logger.info(f"   Tools available: {data.get('message', {}).get('metadata', {}).get('mcp_tools_available', 0)}")
                logger.info(f"   Tool calls made: {data.get('message', {}).get('metadata', {}).get('tool_calls_made', 0)}")
                return True
            else:
                logger.error(f"❌ Chat with tools failed: {response.status_code}")
                logger.error(f"   Response: {response.text}")
                return False
                
        except Exception as e:
            logger.error(f"❌ Chat test error: {e}")
            return False

async def test_conversation_tools_analytics():
    """Test conversation tools analytics"""
    async with httpx.AsyncClient() as client:
        try:
            # Use existing conversation or create new one
            response = await client.get(f"{BASE_URL}/conversations/1/tools/analytics?user_id=1")
            
            if response.status_code == 200:
                data = response.json()
                logger.info("✅ Tools analytics endpoint working")
                analytics = data.get('data', {})
                logger.info(f"   Total tool calls: {analytics.get('total_tool_calls', 0)}")
                logger.info(f"   Success rate: {analytics.get('success_rate', 0):.2%}")
                return True
            elif response.status_code == 404:
                logger.info("ℹ️  Tools analytics - no conversation found (expected for new setup)")
                return True
            else:
                logger.error(f"❌ Tools analytics failed: {response.status_code}")
                return False
        except Exception as e:
            logger.error(f"❌ Tools analytics error: {e}")
            return False

async def run_all_tests():
    """Run all MCP integration tests"""
    logger.info("🚀 Starting MCP Integration Tests...")
    logger.info("=" * 50)
    
    tests = [
        ("System Status", test_system_status),
        ("MCP Status", test_mcp_status),
        ("MCP Tools", test_mcp_tools),
        ("Chat with Tools", test_chat_with_tools),
        ("Tools Analytics", test_conversation_tools_analytics),
    ]
    
    results = []
    
    for test_name, test_func in tests:
        logger.info(f"\n🧪 Running {test_name}...")
        try:
            result = await test_func()
            results.append((test_name, result))
        except Exception as e:
            logger.error(f"❌ {test_name} failed with exception: {e}")
            results.append((test_name, False))
    
    logger.info("\n" + "=" * 50)
    logger.info("📊 TEST RESULTS:")
    logger.info("=" * 50)
    
    passed = 0
    for test_name, result in results:
        status = "✅ PASS" if result else "❌ FAIL"
        logger.info(f"{status} {test_name}")
        if result:
            passed += 1
    
    logger.info(f"\n🎯 Summary: {passed}/{len(results)} tests passed")
    
    if passed == len(results):
        logger.info("🎉 All tests passed! MCP integration is working correctly.")
        return 0
    else:
        logger.error("⚠️  Some tests failed. Check the logs above for details.")
        return 1

if __name__ == "__main__":
    exit_code = asyncio.run(run_all_tests())
    sys.exit(exit_code)
