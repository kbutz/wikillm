"""
Enhanced Conversation Manager with MCP Tool Integration
"""
import json
import logging
from typing import List, Dict, Any, Optional
from sqlalchemy.orm import Session

from conversation_manager import ConversationManager as BaseConversationManager
from mcp_integration import get_mcp_tools_for_assistant, handle_mcp_tool_call

logger = logging.getLogger(__name__)


class EnhancedConversationManager(BaseConversationManager):
    """Enhanced conversation manager with MCP tool integration"""
    
    def __init__(self, db: Session):
        super().__init__(db)
        self.mcp_tools_cache = []
        self.mcp_tools_last_updated = None
    
    async def build_conversation_context(
        self,
        conversation_id: int,
        user_id: int,
        max_messages: int = 50,
        include_historical_context: bool = True,
        include_mcp_tools: bool = True
    ) -> List[Dict[str, Any]]:
        """Build conversation context with MCP tools integration"""
        
        # Get base conversation context
        context = await super().build_conversation_context(
            conversation_id, user_id, max_messages, include_historical_context
        )
        
        # Add MCP tools to system message if enabled
        if include_mcp_tools:
            mcp_tools = await self._get_available_mcp_tools()
            if mcp_tools:
                # Find or create system message
                system_message = None
                for i, msg in enumerate(context):
                    if msg.get("role") == "system":
                        system_message = msg
                        break
                
                if not system_message:
                    # Create new system message
                    system_message = {
                        "role": "system",
                        "content": "You are an AI assistant with access to various tools and services."
                    }
                    context.insert(0, system_message)
                
                # Add MCP tools information to system message
                mcp_info = self._format_mcp_tools_for_system_message(mcp_tools)
                if mcp_info:
                    system_message["content"] += f"\n\n{mcp_info}"
        
        return context
    
    async def _get_available_mcp_tools(self) -> List[Dict[str, Any]]:
        """Get available MCP tools with caching"""
        try:
            # Simple cache mechanism - in production, you'd want more sophisticated caching
            import time
            current_time = time.time()
            
            if (self.mcp_tools_last_updated is None or 
                current_time - self.mcp_tools_last_updated > 60):  # Cache for 1 minute
                
                self.mcp_tools_cache = get_mcp_tools_for_assistant()
                self.mcp_tools_last_updated = current_time
            
            return self.mcp_tools_cache
            
        except Exception as e:
            logger.error(f"Failed to get MCP tools: {e}")
            return []
    
    def _format_mcp_tools_for_system_message(self, mcp_tools: List[Dict[str, Any]]) -> str:
        """Format MCP tools information for the system message"""
        if not mcp_tools:
            return ""
        
        tools_info = ["AVAILABLE TOOLS:", "You have access to the following Model Context Protocol (MCP) tools:"]
        
        # Group tools by server
        tools_by_server = {}
        for tool in mcp_tools:
            server_id = tool.get("mcp_server_id", "unknown")
            if server_id not in tools_by_server:
                tools_by_server[server_id] = []
            tools_by_server[server_id].append(tool)
        
        for server_id, server_tools in tools_by_server.items():
            tools_info.append(f"\n=== MCP Server: {server_id} ===")
            
            for tool in server_tools:
                func_info = tool.get("function", {})
                tool_name = func_info.get("name", "unknown")
                description = func_info.get("description", "No description available")
                
                tools_info.append(f"- {tool_name}: {description}")
                
                # Add parameter information if available
                parameters = func_info.get("parameters", {})
                if parameters and "properties" in parameters:
                    props = parameters["properties"]
                    required = parameters.get("required", [])
                    
                    param_info = []
                    for param_name, param_def in props.items():
                        param_type = param_def.get("type", "unknown")
                        param_desc = param_def.get("description", "")
                        required_marker = " (required)" if param_name in required else ""
                        param_info.append(f"    • {param_name} ({param_type}){required_marker}: {param_desc}")
                    
                    if param_info:
                        tools_info.extend(param_info)
        
        tools_info.append("\nTo use these tools, call them by their exact function name with the appropriate parameters.")
        
        return "\n".join(tools_info)
    
    async def process_llm_response_with_tools(
        self,
        llm_response: Dict[str, Any],
        conversation_id: int
    ) -> Dict[str, Any]:
        """Process LLM response and handle tool calls"""
        
        # Check if the response contains tool calls
        if "tool_calls" not in llm_response.get("choices", [{}])[0].get("message", {}):
            return llm_response
        
        message = llm_response["choices"][0]["message"]
        tool_calls = message.get("tool_calls", [])
        
        if not tool_calls:
            return llm_response
        
        # Process each tool call
        tool_results = []
        
        for tool_call in tool_calls:
            tool_call_id = tool_call.get("id", "unknown")
            function = tool_call.get("function", {})
            tool_name = function.get("name", "")
            
            # Check if this is an MCP tool call
            if tool_name.startswith("mcp_"):
                try:
                    # Parse arguments
                    arguments_str = function.get("arguments", "{}")
                    if isinstance(arguments_str, str):
                        arguments = json.loads(arguments_str)
                    else:
                        arguments = arguments_str
                    
                    # Call MCP tool
                    result = await handle_mcp_tool_call(tool_name, arguments)
                    
                    tool_results.append({
                        "tool_call_id": tool_call_id,
                        "role": "tool",
                        "name": tool_name,
                        "content": json.dumps(result)
                    })
                    
                    # Log tool usage
                    self._log_tool_usage(conversation_id, tool_name, arguments, result)
                    
                except Exception as e:
                    logger.error(f"Error processing MCP tool call {tool_name}: {e}")
                    error_result = {
                        "success": False,
                        "error": f"Tool execution failed: {str(e)}"
                    }
                    tool_results.append({
                        "tool_call_id": tool_call_id,
                        "role": "tool",
                        "name": tool_name,
                        "content": json.dumps(error_result)
                    })
        
        # If we processed any tool calls, we need to make another LLM call
        # to get the final response incorporating the tool results
        if tool_results:
            return {
                **llm_response,
                "tool_results": tool_results,
                "requires_followup": True
            }
        
        return llm_response
    
    def _log_tool_usage(
        self,
        conversation_id: int,
        tool_name: str,
        arguments: Dict[str, Any],
        result: Dict[str, Any]
    ):
        """Log tool usage for analytics and debugging"""
        try:
            # Add a message to track tool usage
            tool_usage_data = {
                "tool_name": tool_name,
                "arguments": arguments,
                "result": result,
                "timestamp": "utcnow"
            }
            
            self.add_message(
                conversation_id,
                "system",
                f"Tool used: {tool_name}",
                metadata={
                    "message_type": "tool_usage",
                    "tool_data": tool_usage_data
                }
            )
            
        except Exception as e:
            logger.error(f"Failed to log tool usage: {e}")
    
    async def build_tool_enhanced_context(
        self,
        conversation_id: int,
        user_id: int,
        max_messages: int = 50,
        include_historical_context: bool = True
    ) -> List[Dict[str, Any]]:
        """Build conversation context specifically optimized for tool usage"""
        
        context = await self.build_conversation_context(
            conversation_id,
            user_id,
            max_messages,
            include_historical_context,
            include_mcp_tools=True
        )
        
        # Add tool usage instructions to system message
        for message in context:
            if message.get("role") == "system":
                tool_instructions = self._get_tool_usage_instructions()
                message["content"] += f"\n\n{tool_instructions}"
                break
        
        return context
    
    def _get_tool_usage_instructions(self) -> str:
        """Get instructions for tool usage"""
        return """
TOOL USAGE GUIDELINES:
1. Use tools when they can provide accurate, up-to-date information or perform specific actions
2. Always explain what tool you're using and why before calling it
3. After receiving tool results, interpret and summarize them for the user
4. If a tool call fails, explain what went wrong and suggest alternatives
5. Be efficient - don't call multiple tools for the same information
6. Consider the user's context and preferences when choosing which tools to use

When calling tools:
- Ensure all required parameters are provided
- Use clear, descriptive parameter values
- Handle tool responses gracefully, even if they contain errors
"""
    
    async def get_tool_usage_analytics(self, conversation_id: int) -> Dict[str, Any]:
        """Get analytics about tool usage in a conversation"""
        try:
            # Get all tool usage messages
            messages = self.get_conversation_messages(conversation_id)
            
            tool_usage_stats = {
                "total_tool_calls": 0,
                "tools_used": {},
                "success_rate": 0,
                "most_used_tool": None,
                "tool_timeline": []
            }
            
            successful_calls = 0
            
            for message in messages:
                if (message.metadata and 
                    message.metadata.get("message_type") == "tool_usage"):
                    
                    tool_data = message.metadata.get("tool_data", {})
                    tool_name = tool_data.get("tool_name", "unknown")
                    result = tool_data.get("result", {})
                    
                    tool_usage_stats["total_tool_calls"] += 1
                    
                    # Track tool usage count
                    if tool_name not in tool_usage_stats["tools_used"]:
                        tool_usage_stats["tools_used"][tool_name] = 0
                    tool_usage_stats["tools_used"][tool_name] += 1
                    
                    # Track success rate
                    if result.get("success", False):
                        successful_calls += 1
                    
                    # Add to timeline
                    tool_usage_stats["tool_timeline"].append({
                        "tool_name": tool_name,
                        "timestamp": message.created_at,
                        "success": result.get("success", False)
                    })
            
            # Calculate success rate
            if tool_usage_stats["total_tool_calls"] > 0:
                tool_usage_stats["success_rate"] = successful_calls / tool_usage_stats["total_tool_calls"]
            
            # Find most used tool
            if tool_usage_stats["tools_used"]:
                tool_usage_stats["most_used_tool"] = max(
                    tool_usage_stats["tools_used"].items(),
                    key=lambda x: x[1]
                )[0]
            
            return tool_usage_stats
            
        except Exception as e:
            logger.error(f"Failed to get tool usage analytics: {e}")
            return {
                "total_tool_calls": 0,
                "tools_used": {},
                "success_rate": 0,
                "most_used_tool": None,
                "tool_timeline": [],
                "error": str(e)
            }
