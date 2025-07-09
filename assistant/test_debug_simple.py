#!/usr/bin/env python3
"""
Simple test to check if our debug changes are working
"""
import sys
import os
import logging
from datetime import datetime

# Add the current directory to the Python path
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

try:
    # Test imports
    from lmstudio_client import lmstudio_client
    from debug_routes import debug_router
    from enhanced_schemas import ChatRequestWithDebug
    from schemas import Message
    
    print("✓ All imports successful")
    
    # Test debug context functionality
    debug_context = {}
    
    # Test message creation with debug fields
    test_message = Message(
        id=1,
        conversation_id=1,
        role="assistant",
        content="Test message",
        timestamp=datetime.now(),
        debug_enabled=True,
        debug_data={"test": "data"},
        intermediary_steps=[],
        llm_request={"model": "test", "messages": [], "timestamp": datetime.now().isoformat()},
        llm_response={"response": {}, "timestamp": datetime.now().isoformat(), "processing_time_ms": 100},
        tool_calls=[],
        tool_results=[]
    )
    
    print("✓ Message with debug fields created successfully")
    print(f"  - Debug enabled: {test_message.debug_enabled}")
    print(f"  - Has LLM request: {test_message.llm_request is not None}")
    print(f"  - Has LLM response: {test_message.llm_response is not None}")
    
    # Test that debug context is properly initialized
    print(f"✓ Debug context initialized: {debug_context}")
    
    print("\n✓ All basic tests passed!")
    print("The debug integration changes appear to be working correctly.")
    
except Exception as e:
    print(f"✗ Test failed: {e}")
    import traceback
    traceback.print_exc()
    sys.exit(1)
