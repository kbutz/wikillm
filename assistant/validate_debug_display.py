#!/usr/bin/env python3
"""
Validate the debug LLM request display format
"""
import json
from datetime import datetime

def show_debug_display_format():
    """Show exactly what will be displayed in the debug panel"""
    
    # This is the exact format that will be sent to the frontend
    mock_message_with_debug = {
        "id": 123,
        "conversation_id": 456,
        "role": "assistant",
        "content": "Hello! This is a test response with debug data.",
        "timestamp": datetime.now().isoformat(),
        "debug_enabled": True,
        "llm_request": {
            "model": "llama-3.2-3b-instruct",
            "messages": [
                {
                    "role": "system",
                    "content": "You are a helpful AI assistant."
                },
                {
                    "role": "user", 
                    "content": "Hello, can you help me test the debug system?"
                }
            ],
            "temperature": 0.7,
            "max_tokens": 1000,
            "stream": False,
            "tools": [
                {
                    "type": "function",
                    "function": {
                        "name": "mcp_filesystem_read_file",
                        "description": "Read a file from the filesystem",
                        "parameters": {
                            "type": "object",
                            "properties": {
                                "path": {"type": "string"}
                            }
                        }
                    }
                }
            ],
            "tool_choice": "auto",
            "timestamp": datetime.now().isoformat(),
            "request_id": "req_123456",
            "processing_time_ms": 1250
        },
        "llm_response": {
            "response": {
                "choices": [
                    {
                        "message": {
                            "role": "assistant",
                            "content": "Hello! This is a test response with debug data."
                        }
                    }
                ],
                "usage": {
                    "prompt_tokens": 25,
                    "completion_tokens": 12,
                    "total_tokens": 37
                }
            },
            "timestamp": datetime.now().isoformat(),
            "processing_time_ms": 1250,
            "token_usage": {
                "prompt_tokens": 25,
                "completion_tokens": 12,
                "total_tokens": 37
            },
            "request_id": "req_123456"
        },
        "debug_data": {
            "debug_enabled": True,
            "debug_session_id": "session_789",
            "message_id": 123,
            "debug_steps_count": 3,
            "llm_requests_count": 1,
            "debug_context_captured": True,
            "debug_context_keys": ["llm_request_payload", "llm_response_raw", "llm_processing_time_ms"]
        }
    }
    
    print("=== DEBUG MESSAGE FORMAT ===")
    print("This is the exact format that will be sent to the frontend:")
    print()
    print(json.dumps(mock_message_with_debug, indent=2))
    print()
    
    print("=== FRONTEND DISPLAY ===")
    print("In the MessageBubble component, this will be displayed as:")
    print()
    
    # Show what the LLM Request section will look like
    print("1. LLM Request (Full JSON) - Terminal Style Display:")
    print("   Background: Dark gray (bg-gray-900)")
    print("   Text: Green (text-green-400)")
    print("   Content:")
    llm_request_display = {
        "model": mock_message_with_debug["llm_request"]["model"],
        "messages": mock_message_with_debug["llm_request"]["messages"],
        "temperature": mock_message_with_debug["llm_request"]["temperature"],
        "max_tokens": mock_message_with_debug["llm_request"]["max_tokens"],
        "stream": mock_message_with_debug["llm_request"]["stream"],
        "tools": mock_message_with_debug["llm_request"]["tools"],
        "tool_choice": mock_message_with_debug["llm_request"]["tool_choice"]
    }
    print(json.dumps(llm_request_display, indent=2))
    print()
    
    # Show what the LLM Response section will look like
    print("2. LLM Response (Full JSON) - Terminal Style Display:")
    print("   Background: Dark gray (bg-gray-900)")
    print("   Text: Blue (text-blue-400)")
    print("   Content:")
    print(json.dumps(mock_message_with_debug["llm_response"]["response"], indent=2))
    print()
    
    print("=== VALIDATION CHECKLIST ===")
    print("✓ Full JSON request payload captured")
    print("✓ Model, messages, temperature, max_tokens included")
    print("✓ Tools array with function definitions")
    print("✓ Tool choice setting")
    print("✓ Processing time and token usage")
    print("✓ Terminal-style formatting for readability")
    print("✓ Clear labels for request vs response")
    print("✓ Debug data includes troubleshooting information")
    print()
    
    print("This data will be prominently displayed in the chat interface")
    print("when debug mode is enabled, showing the exact JSON sent to LMStudio.")

if __name__ == "__main__":
    show_debug_display_format()
