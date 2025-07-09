"""
Enhanced Conversation Manager with Optimized Context Building and Tool Selection
"""
import json
import logging
import re
from typing import List, Dict, Any, Optional
from sqlalchemy.orm import Session

from conversation_manager import ConversationManager as BaseConversationManager
from mcp_integration import get_mcp_tools_for_assistant, handle_mcp_tool_call
from lmstudio_client import lmstudio_client

logger = logging.getLogger(__name__)


class EnhancedConversationManager(BaseConversationManager):
    """Enhanced conversation manager with optimized context building and tool selection"""
    
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
                    system_message["content"] += f"\\n\\n{mcp_info}"
        
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
    
    def _extract_topics_from_context(self, conversation_context: List[Dict]) -> List[str]:
        """Extract key topics from conversation context"""
        topics = []
        
        # Extract keywords from recent messages
        for message in conversation_context[-5:]:  # Last 5 messages
            if message.get("role") == "user":
                content = message.get("content", "")
                # Simple keyword extraction
                words = re.findall(r'\\b[a-zA-Z]{4,}\\b', content.lower())
                topics.extend(words)
        
        # Remove duplicates and common words
        stop_words = {'this', 'that', 'with', 'from', 'they', 'have', 'been', 'were', 'said', 'each', 'which', 'their', 'time', 'will', 'about', 'would', 'there', 'could', 'other', 'more', 'very', 'what', 'know', 'just', 'first', 'into', 'over', 'think', 'also', 'your', 'work', 'life', 'only', 'new', 'years', 'way', 'may', 'say', 'come', 'its', 'now', 'find', 'long', 'down', 'day', 'did', 'get', 'has', 'him', 'his', 'how', 'man', 'new', 'now', 'old', 'see', 'two', 'who', 'boy', 'did', 'its', 'let', 'put', 'say', 'she', 'too', 'use'}
        
        unique_topics = []
        seen = set()
        for topic in topics:
            if topic not in stop_words and topic not in seen and len(topic) >= 4:
                unique_topics.append(topic)
                seen.add(topic)
        
        return unique_topics[:10]  # Return top 10 topics
    
    async def get_relevant_tools(
        self, 
        user_message: str, 
        conversation_context: List[Dict],
        max_tools: int = 5
    ) -> List[Dict]:
        """Select only relevant tools based on conversation context"""
        
        all_tools = await self._get_available_mcp_tools()
        
        if not all_tools:
            return []
        
        # If we have 5 or fewer tools, return all
        if len(all_tools) <= max_tools:
            return all_tools
        
        try:
            # Extract topics from conversation context
            topics = self._extract_topics_from_context(conversation_context)
            
            # Create tool relevance prompt
            tool_descriptions = []
            for i, tool in enumerate(all_tools):
                func_info = tool.get("function", {})
                tool_name = func_info.get("name", "unknown")
                description = func_info.get("description", "")
                tool_descriptions.append(f"{i+1}. {tool_name}: {description}")
            
            tool_relevance_prompt = f"""Given this user message: "{user_message}"
            
And conversation context about: {', '.join(topics)}
            
Which of these tools are most relevant? Return the numbers of the top {max_tools} most useful tools as a comma-separated list.
            
Available tools:
{chr(10).join(tool_descriptions)}
            
Example response: 3,7,1,9,2
            
Return only the numbers, nothing else."""
            
            response = await lmstudio_client.chat_completion(
                messages=[
                    {"role": "system", "content": "You are a tool relevance assessment system. Return only comma-separated numbers."},
                    {"role": "user", "content": tool_relevance_prompt}
                ],
                temperature=0.1,
                max_tokens=50
            )
            
            content = response["choices"][0]["message"]["content"].strip()
            
            # Parse the response
            try:
                tool_indices = [int(n.strip()) - 1 for n in content.split(",") if n.strip().isdigit()]
                
                # Return selected tools
                relevant_tools = []
                for idx in tool_indices[:max_tools]:
                    if 0 <= idx < len(all_tools):
                        relevant_tools.append(all_tools[idx])
                
                if relevant_tools:
                    logger.info(f"Selected {len(relevant_tools)} relevant tools from {len(all_tools)} available")
                    return relevant_tools
                
            except (ValueError, IndexError) as e:
                logger.warning(f"Failed to parse tool selection: {e}")
            
            # Fallback: return first max_tools tools
            logger.info(f"Using fallback tool selection: first {max_tools} tools")
            return all_tools[:max_tools]
            
        except Exception as e:
            logger.error(f"Tool relevance assessment failed: {e}")
            # Fallback to first max_tools tools
            return all_tools[:max_tools]
    
    def _format_mcp_tools_for_system_message(self, mcp_tools: List[Dict[str, Any]]) -> str:
        """Format MCP tools information for the system message with concise descriptions"""
        if not mcp_tools:
            return ""
        
        tools_info = ["AVAILABLE TOOLS:"]
        
        # Group tools by server for better organization
        tools_by_server = {}
        for tool in mcp_tools:
            server_id = tool.get("mcp_server_id", "unknown")
            if server_id not in tools_by_server:
                tools_by_server[server_id] = []
            tools_by_server[server_id].append(tool)
        
        for server_id, server_tools in tools_by_server.items():
            if len(tools_by_server) > 1:  # Only show server name if multiple servers
                tools_info.append(f"\\n{server_id.upper()}:")
            
            for tool in server_tools:
                func_info = tool.get("function", {})
                tool_name = func_info.get("name", "unknown")
                description = func_info.get("description", "No description")
                
                # Truncate long descriptions
                if len(description) > 100:
                    description = description[:97] + "..."
                
                tools_info.append(f"- {tool_name}: {description}")
        
        tools_info.append("\\nCall tools using their exact function name with appropriate parameters.")
        
        return "\\n".join(tools_info)
    
    async def build_optimized_context(
        self,
        conversation_id: int,
        user_id: int,
        max_messages: int = 20,
        include_historical_context: bool = True
    ) -> List[Dict[str, Any]]:
        """Build optimized context with structured information and relevant tools"""
        
        # Get base conversation messages
        messages = self.get_recent_messages(conversation_id, max_messages)
        
        # Convert to context format
        context = []
        for message in messages:
            context.append({
                "role": message.role,
                "content": message.content
            })
        
        # Get current user message for tool selection
        current_message = ""
        if context:
            for msg in reversed(context):
                if msg.get("role") == "user":
                    current_message = msg.get("content", "")
                    break
        
        # Get consolidated user profile
        try:
            from memory_manager import EnhancedMemoryManager
            enhanced_memory = EnhancedMemoryManager(self.db)
            user_profile = await enhanced_memory.get_consolidated_user_profile(user_id)
        except Exception as e:
            logger.error(f"Failed to get user profile: {e}")
            user_profile = {}
        
        # Get relevant tools only
        relevant_tools = await self.get_relevant_tools(current_message, context, max_tools=5)
        
        # Get structured historical context
        historical_context = {}
        if include_historical_context:
            try:
                from search_manager import SearchManager
                search_manager = SearchManager(self.db)
                historical_context = await search_manager.get_structured_historical_context(
                    user_id, current_message, limit=2
                )
            except Exception as e:
                logger.error(f"Failed to get historical context: {e}")
        
        # Build optimized system message
        system_content = self._build_structured_system_message(
            user_profile, relevant_tools, historical_context
        )
        
        # Insert system message at the beginning
        return [{"role": "system", "content": system_content}] + context
    
    def _build_structured_system_message(
        self,
        user_profile: Dict,
        relevant_tools: List,
        historical_context: Dict
    ) -> str:
        """Build structured, token-efficient system message"""
        
        parts = ["You are a helpful AI assistant."]
        
        # Consolidated user profile (high-confidence facts only)
        if user_profile:
            parts.append("\\nUSER PROFILE:")
            
            if user_profile.get("personal"):
                personal_items = [f"{k}: {v}" for k, v in user_profile["personal"].items()]
                if personal_items:
                    parts.append(f"Personal: {'; '.join(personal_items[:3])}")
            
            if user_profile.get("preferences"):
                pref_items = [f"{k}: {v}" for k, v in user_profile["preferences"].items()]
                if pref_items:
                    parts.append(f"Preferences: {'; '.join(pref_items[:3])}")
            
            if user_profile.get("skills"):
                skill_items = [f"{k}: {v}" for k, v in user_profile["skills"].items()]
                if skill_items:
                    parts.append(f"Skills: {'; '.join(skill_items[:3])}")
            
            if user_profile.get("projects"):
                project_items = [f"{k}: {v}" for k, v in user_profile["projects"].items()]
                if project_items:
                    parts.append(f"Current Projects: {'; '.join(project_items[:2])}")
            
            if user_profile.get("context"):
                context_items = [f"{k}: {v}" for k, v in user_profile["context"].items()]
                if context_items:
                    parts.append(f"Context: {'; '.join(context_items[:2])}")
        
        # Relevant tools only (concise format)
        if relevant_tools:
            tools_info = self._format_mcp_tools_for_system_message(relevant_tools)
            parts.append(f"\\n{tools_info}")
        
        # Structured historical insights (actionable only)
        if historical_context:
            if historical_context.get("relevant_solutions"):
                parts.append("\\nPREVIOUS SOLUTIONS:")
                for solution in historical_context["relevant_solutions"][:2]:
                    parts.append(f"- {solution}")
            
            if historical_context.get("project_continuations"):
                parts.append("\\nPROJECT CONTINUATIONS:")
                for continuation in historical_context["project_continuations"][:2]:
                    parts.append(f"- {continuation}")
            
            if historical_context.get("similar_topics"):
                parts.append("\\nRELATED TOPICS:")
                topics = ", ".join(historical_context["similar_topics"][:4])
                parts.append(f"Previously discussed: {topics}")
        
        # Usage guidelines
        parts.append("\\nProvide helpful, personalized responses using available tools when beneficial.")
        
        return "\\n".join(parts)
    
    async def build_tool_enhanced_context(
        self,
        conversation_id: int,
        user_id: int,
        max_messages: int = 50,
        include_historical_context: bool = True
    ) -> List[Dict[str, Any]]:
        """Build conversation context specifically optimized for tool usage"""
        
        # Use the new optimized context building
        return await self.build_optimized_context(
            conversation_id,
            user_id,
            max_messages,
            include_historical_context
        )
    
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
    
    def _get_tool_usage_instructions(self) -> str:
        """Get concise instructions for tool usage"""
        return """TOOL USAGE:
1. Use tools for accurate, up-to-date information or specific actions
2. Explain tool usage briefly before calling
3. Interpret and summarize tool results for the user
4. Handle tool failures gracefully with alternatives
5. Be efficient - avoid redundant tool calls

Ensure all required parameters are provided with clear, descriptive values."""
    
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
