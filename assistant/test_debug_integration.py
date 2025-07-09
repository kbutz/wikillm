#!/usr/bin/env python3
"""
Test script to verify debug integration is working
"""
import asyncio
import sys
import os
import logging

# Add the current directory to the Python path
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from config import settings
from database import init_database, get_db
from lmstudio_client import lmstudio_client
from debug_routes import debug_router
from enhanced_schemas import ChatRequestWithDebug

# Set up logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

async def test_debug_integration():
    """Test the debug integration end-to-end"""
    
    try:
        # Initialize database
        init_database()
        
        # Test LMStudio client with debug context
        logger.info("Testing LMStudio client with debug context...")
        
        debug_context = {}
        
        # Simple test messages
        test_messages = [
            {"role": "user", "content": "Hello, can you help me test the debug system?"}
        ]
        
        # Test the LMStudio client
        response = await lmstudio_client.chat_completion(
            messages=test_messages,
            temperature=0.7,
            max_tokens=100,
            debug_context=debug_context
        )
        
        logger.info("LMStudio Response:")
        logger.info(f"Response: {response}")
        
        logger.info("Debug Context:")
        logger.info(f"Debug context: {debug_context}")
        
        # Check if debug information was captured
        if 'llm_request_payload' in debug_context:
            logger.info("✓ LLM request payload captured")
        else:
            logger.warning("✗ LLM request payload NOT captured")
            
        if 'llm_response_raw' in debug_context:
            logger.info("✓ LLM response raw captured")
        else:
            logger.warning("✗ LLM response raw NOT captured")
            
        if 'llm_processing_time_ms' in debug_context:
            logger.info(f"✓ Processing time: {debug_context['llm_processing_time_ms']}ms")
        else:
            logger.warning("✗ Processing time NOT captured")
            
        return True
        
    except Exception as e:
        logger.error(f"Test failed: {e}")
        import traceback
        traceback.print_exc()
        return False

async def test_lmstudio_connection():
    """Test LMStudio connection"""
    try:
        health = await lmstudio_client.health_check()
        logger.info(f"LMStudio health check: {health}")
        
        if health:
            logger.info("✓ LMStudio connection successful")
        else:
            logger.warning("✗ LMStudio connection failed")
            logger.info("Make sure LMStudio is running and accessible")
            
        return health
        
    except Exception as e:
        logger.error(f"LMStudio connection test failed: {e}")
        return False

async def main():
    """Main test function"""
    logger.info("Starting debug integration test...")
    
    # Test LMStudio connection first
    logger.info("=" * 50)
    logger.info("Testing LMStudio Connection")
    logger.info("=" * 50)
    
    connection_ok = await test_lmstudio_connection()
    
    if not connection_ok:
        logger.error("LMStudio connection failed - cannot continue with debug test")
        return False
        
    # Test debug integration
    logger.info("=" * 50)
    logger.info("Testing Debug Integration")
    logger.info("=" * 50)
    
    debug_ok = await test_debug_integration()
    
    if debug_ok:
        logger.info("✓ Debug integration test passed")
    else:
        logger.error("✗ Debug integration test failed")
        
    return debug_ok

if __name__ == "__main__":
    success = asyncio.run(main())
    sys.exit(0 if success else 1)
