#!/usr/bin/env python3
"""
Enhanced debug tools for MCP integration troubleshooting
"""
import json
import logging
from typing import Dict, Any, Optional

def debug_llm_response(response: Dict[str, Any], context: str = "") -> None:
    """Debug LLM response structure for tool call issues"""
    logger = logging.getLogger(__name__)
    
    try:
        logger.info(f"=== DEBUG LLM RESPONSE {context} ===")
        
        # Check basic structure
        if "choices" not in response:
            logger.error("❌ Missing 'choices' in LLM response")
            return
            
        if not response["choices"]:
            logger.error("❌ Empty 'choices' array in LLM response")
            return
            
        choice = response["choices"][0]
        if "message" not in choice:
            logger.error("❌ Missing 'message' in first choice")
            return
            
        message = choice["message"]
        
        # Log message structure
        logger.info(f"Message keys: {list(message.keys())}")
        
        # Check content
        content = message.get("content")
        if content is None:
            logger.warning("⚠️  Message content is None")
        elif content == "":
            logger.warning("⚠️  Message content is empty string")
        else:
            logger.info(f"✅ Message content present: {len(content)} chars")
            
        # Check tool calls
        tool_calls = message.get("tool_calls")
        if tool_calls:
            logger.info(f"✅ Tool calls present: {len(tool_calls)} calls")
            for i, tool_call in enumerate(tool_calls):
                logger.info(f"   Tool {i+1}: {tool_call.get('function', {}).get('name', 'unknown')}")
        else:
            logger.info("ℹ️  No tool calls in message")
            
        # Check role
        role = message.get("role", "unknown")
        logger.info(f"Message role: {role}")
        
    except Exception as e:
        logger.error(f"❌ Error debugging LLM response: {e}")
        logger.error(f"Response structure: {json.dumps(response, indent=2, default=str)}")

def safe_extract_content(message: Dict[str, Any], fallback: str = "") -> str:
    """Safely extract content from LLM message with proper fallbacks"""
    try:
        content = message.get("content")
        
        if content is None:
            return fallback
        elif isinstance(content, str):
            return content
        else:
            # Handle unexpected content types
            return str(content)
            
    except Exception as e:
        logging.getLogger(__name__).error(f"Error extracting content: {e}")
        return fallback

def build_safe_followup_message(message: Dict[str, Any]) -> Dict[str, Any]:
    """Build a safe followup message that handles null content"""
    try:
        followup = {
            "role": "assistant"
        }
        
        # Add content only if it exists and is not null
        content = message.get("content")
        if content is not None and content != "":
            followup["content"] = content
            
        # Add tool calls if present
        tool_calls = message.get("tool_calls")
        if tool_calls:
            followup["tool_calls"] = tool_calls
            
        return followup
        
    except Exception as e:
        logging.getLogger(__name__).error(f"Error building safe followup message: {e}")
        return {
            "role": "assistant",
            "content": "[Error processing message]"
        }

def validate_chat_response_structure(response: Dict[str, Any]) -> tuple[bool, str]:
    """Validate chat response structure and return validation result"""
    try:
        # Check basic structure
        if not isinstance(response, dict):
            return False, "Response is not a dictionary"
            
        if "choices" not in response:
            return False, "Missing 'choices' key in response"
            
        choices = response["choices"]
        if not isinstance(choices, list) or len(choices) == 0:
            return False, "Invalid or empty 'choices' array"
            
        # Check first choice
        choice = choices[0]
        if not isinstance(choice, dict):
            return False, "First choice is not a dictionary"
            
        if "message" not in choice:
            return False, "Missing 'message' in first choice"
            
        message = choice["message"]
        if not isinstance(message, dict):
            return False, "Message is not a dictionary"
            
        # Check message has either content or tool_calls
        has_content = message.get("content") is not None
        has_tool_calls = message.get("tool_calls") is not None
        
        if not has_content and not has_tool_calls:
            return False, "Message has neither content nor tool_calls"
            
        return True, "Response structure is valid"
        
    except Exception as e:
        return False, f"Validation error: {str(e)}"

# Enhanced logging configuration for debugging
def setup_enhanced_debug_logging():
    """Setup enhanced logging for MCP debugging"""
    debug_logger = logging.getLogger("mcp_debug")
    debug_logger.setLevel(logging.DEBUG)
    
    # Create debug file handler
    debug_handler = logging.FileHandler("mcp_debug.log")
    debug_handler.setLevel(logging.DEBUG)
    
    # Create formatter
    formatter = logging.Formatter(
        '%(asctime)s - %(name)s - %(levelname)s - %(funcName)s:%(lineno)d - %(message)s'
    )
    debug_handler.setFormatter(formatter)
    
    debug_logger.addHandler(debug_handler)
    return debug_logger
