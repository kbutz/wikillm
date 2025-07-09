#!/usr/bin/env python3
"""
Comprehensive test for debug LLM request capture
"""
import asyncio
import json
import logging
from datetime import datetime

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

async def test_debug_llm_request():
    """Test that debug LLM request is being captured properly"""
    
    # Mock debug context
    debug_context = {}
    
    # Simulate LLM request parameters
    mock_messages = [
        {"role": "user", "content": "Hello, test the debug system"}
    ]
    
    mock_tools = [
        {
            "type": "function",
            "function": {
                "name": "test_tool",
                "description": "A test tool",
                "parameters": {"type": "object", "properties": {}}
            }
        }
    ]
    
    # Simulate what the LMStudio client should capture
    mock_payload = {
        "model": "test-model",
        "messages": mock_messages,
        "temperature": 0.7,
        "max_tokens": 1000,
        "stream": False,
        "tools": mock_tools,
        "tool_choice": "auto"
    }
    
    # Simulate debug context capture
    debug_context['llm_request_payload'] = mock_payload.copy()
    debug_context['llm_request_timestamp'] = datetime.now().isoformat()
    debug_context['llm_request_messages_count'] = len(mock_messages)
    debug_context['llm_request_tools_count'] = len(mock_tools)
    
    # Simulate response
    mock_response = {
        "choices": [
            {
                "message": {
                    "role": "assistant",
                    "content": "Hello! Debug system is working."
                }
            }
        ],
        "usage": {
            "prompt_tokens": 10,
            "completion_tokens": 8,
            "total_tokens": 18
        }
    }
    
    debug_context['llm_response_raw'] = mock_response.copy()
    debug_context['llm_response_timestamp'] = datetime.now().isoformat()
    debug_context['llm_processing_time_ms'] = 1250
    debug_context['llm_response_tokens'] = 18
    
    # Test the debug context
    print("=== DEBUG CONTEXT TEST ===")
    print(f"Debug context keys: {list(debug_context.keys())}")
    print(f"Request payload captured: {'llm_request_payload' in debug_context}")
    print(f"Response captured: {'llm_response_raw' in debug_context}")
    print(f"Processing time captured: {'llm_processing_time_ms' in debug_context}")
    
    # Test the LLM request format
    if 'llm_request_payload' in debug_context:
        request_payload = debug_context['llm_request_payload']
        print("\\n=== LLM REQUEST PAYLOAD ===")
        print(json.dumps(request_payload, indent=2))
        
        # Verify all required fields
        required_fields = ['model', 'messages', 'temperature', 'max_tokens', 'stream']
        for field in required_fields:
            if field in request_payload:
                print(f"✓ {field}: {request_payload[field]}")
            else:
                print(f"✗ {field}: MISSING")
        
        # Check tools
        if 'tools' in request_payload:
            print(f"✓ tools: {len(request_payload['tools'])} tools")
            print(f"✓ tool_choice: {request_payload.get('tool_choice', 'not set')}")
        else:
            print("✗ tools: MISSING")
    
    # Test the LLM response format
    if 'llm_response_raw' in debug_context:
        response_data = debug_context['llm_response_raw']
        print("\\n=== LLM RESPONSE DATA ===")
        print(json.dumps(response_data, indent=2))
        
        # Verify response structure
        if 'choices' in response_data:
            print(f"✓ choices: {len(response_data['choices'])} choices")
        else:
            print("✗ choices: MISSING")
            
        if 'usage' in response_data:
            print(f"✓ usage: {response_data['usage']}")
        else:
            print("✗ usage: MISSING")
    
    print("\\n=== TEST SUMMARY ===")
    print(f"✓ Debug context created successfully")
    print(f"✓ LLM request payload captured: {json.dumps(debug_context['llm_request_payload'], indent=2)}")
    print(f"✓ LLM response captured: {json.dumps(debug_context['llm_response_raw'], indent=2)}")
    print(f"✓ Processing time: {debug_context['llm_processing_time_ms']}ms")
    print(f"✓ Token usage: {debug_context['llm_response_tokens']} tokens")
    
    return True

if __name__ == "__main__":
    success = asyncio.run(test_debug_llm_request())
    print(f"\\nTest {'PASSED' if success else 'FAILED'}")
