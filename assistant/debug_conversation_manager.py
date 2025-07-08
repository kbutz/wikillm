"""
Debug-enabled Conversation Manager with step tracking and full LLM request logging
"""
import json
import logging
import time
from typing import List, Dict, Any, Optional, Tuple
from datetime import datetime
from sqlalchemy.orm import Session

from enhanced_conversation_manager import EnhancedConversationManager
from enhanced_message_schemas import (
    StepTracker, IntermediaryStepType, EnhancedMessage, LLMRequest, LLMResponse,
    ToolCall, ToolResult, ChatRequestWithDebug, ChatResponseWithDebug
)
from mcp_integration import get_mcp_tools_for_assistant, handle_mcp_tool_call

logger = logging.getLogger(__name__)


class DebugConversationManager(EnhancedConversationManager):
    """Enhanced conversation manager with comprehensive debug tracking"""
    
    def __init__(self, db: Session):
        super().__init__(db)
        self.debug_enabled = True
    
    async def process_message_with_debug(
        self,
        request: ChatRequestWithDebug,
        lmstudio_client
    ) -> ChatResponseWithDebug:
        """Process a message with full debug tracking"""
        
        # Initialize step tracker
        step_tracker = StepTracker()
        start_time = time.time()
        
        try:
            # Step 1: Validate request and get/create conversation
            step_id = step_tracker.start_step(
                IntermediaryStepType.CONTEXT_BUILDING,
                "Initialize Conversation",
                "Validating request and setting up conversation context"
            )
            
            # Verify user exists
            from models import User
            user = self.db.query(User).filter(User.id == request.user_id).first()
            if not user:
                step_tracker.complete_step(step_id, success=False, error_message="User not found")
                raise ValueError("User not found")
            
            # Get or create conversation
            conversation = None
            if request.conversation_id:
                conversation = self.get_conversation(request.conversation_id, request.user_id)
                if not conversation:
                    step_tracker.complete_step(step_id, success=False, error_message="Conversation not found")
                    raise ValueError("Conversation not found")
            else:
                conversation = self.create_conversation(request.user_id)
            
            step_tracker.complete_step(step_id, {
                "conversation_id": conversation.id,
                "user_id": request.user_id,
                "conversation_title": conversation.title
            })
            
            # Step 2: Add user message
            step_id = step_tracker.start_step(
                IntermediaryStepType.CONTEXT_BUILDING,
                "Add User Message",
                "Adding user message to conversation"
            )
            
            user_message = self.add_message(
                conversation.id,
                "user",
                request.message
            )
            
            step_tracker.complete_step(step_id, {
                "message_id": user_message.id,
                "message_length": len(request.message)
            })
            
            # Step 3: Build conversation context
            step_id = step_tracker.start_step(
                IntermediaryStepType.CONTEXT_BUILDING,
                "Build Conversation Context",
                "Building enhanced conversation context with memory and tools"
            )
            
            context = await self.build_tool_enhanced_context(
                conversation.id,
                request.user_id,
                max_messages=50,
                include_historical_context=True
            )
            
            step_tracker.complete_step(step_id, {
                "context_messages": len(context),
                "context_size_chars": sum(len(str(msg.get("content", ""))) for msg in context)
            })
            
            # Step 4: Get available tools
            step_id = step_tracker.start_step(
                IntermediaryStepType.TOOL_CALL,
                "Load Available Tools",
                "Loading MCP tools for assistant"
            )
            
            available_tools = get_mcp_tools_for_assistant()
            
            step_tracker.complete_step(step_id, {
                "tools_count": len(available_tools),
                "tool_names": [tool.get("function", {}).get("name") for tool in available_tools]
            })
            
            # Step 5: Prepare LLM request
            step_id = step_tracker.start_step(
                IntermediaryStepType.LLM_REQUEST,
                "Prepare LLM Request",
                "Building LLM request with context and tools"
            )
            
            llm_request_params = {
                "messages": context,
                "temperature": request.temperature,
                "max_tokens": request.max_tokens,
                "stream": False
            }
            
            # Add tools if available
            if available_tools:
                llm_request_params["tools"] = [
                    {
                        "type": "function",
                        "function": tool["function"]
                    } for tool in available_tools
                ]
                llm_request_params["tool_choice"] = "auto"
            
            # Create LLM request object for debugging
            llm_request = LLMRequest(
                model=getattr(lmstudio_client, 'model_name', 'unknown'),
                messages=context,
                temperature=request.temperature,
                max_tokens=request.max_tokens,
                tools=llm_request_params.get("tools"),
                tool_choice=llm_request_params.get("tool_choice"),
                stream=False,
                timestamp=datetime.now()
            )
            
            step_tracker.complete_step(step_id, {
                "model": llm_request.model,
                "context_tokens_estimate": self._estimate_token_count(context),
                "tools_enabled": len(available_tools) > 0
            })
            
            # Step 6: Make initial LLM call
            step_id = step_tracker.start_step(
                IntermediaryStepType.LLM_RESPONSE,
                "Initial LLM Call",
                "Making initial call to LLM"
            )
            
            llm_call_start = time.time()
            llm_response = await lmstudio_client.chat_completion(**llm_request_params)
            llm_call_time = int((time.time() - llm_call_start) * 1000)
            
            step_tracker.complete_step(step_id, {
                "processing_time_ms": llm_call_time,
                "response_tokens": llm_response.get("usage", {}).get("total_tokens", 0),
                "has_tool_calls": "tool_calls" in llm_response.get("choices", [{}])[0].get("message", {})
            })
            
            # Step 7: Process tool calls if present
            tool_calls = []
            tool_results = []
            final_response = llm_response
            
            initial_message = llm_response["choices"][0]["message"]
            if initial_message.get("tool_calls"):
                step_id = step_tracker.start_step(
                    IntermediaryStepType.TOOL_CALL,
                    "Process Tool Calls",
                    f"Processing {len(initial_message['tool_calls'])} tool calls"
                )
                
                tool_call_results = []
                
                for tool_call in initial_message["tool_calls"]:
                    tool_call_id = tool_call.get("id", "unknown")
                    function = tool_call.get("function", {})
                    tool_name = function.get("name", "")
                    
                    # Track tool call
                    tool_calls.append(ToolCall(
                        tool_name=tool_name,
                        arguments=json.loads(function.get("arguments", "{}")) if isinstance(function.get("arguments"), str) else function.get("arguments", {}),
                        server_id=tool_call.get("mcp_server_id")
                    ))
                    
                    if tool_name.startswith("mcp_"):
                        try:
                            tool_start = time.time()
                            arguments = json.loads(function.get("arguments", "{}")) if isinstance(function.get("arguments"), str) else function.get("arguments", {})
                            result = await handle_mcp_tool_call(tool_name, arguments)
                            tool_time = int((time.time() - tool_start) * 1000)
                            
                            # Track tool result
                            tool_results.append(ToolResult(
                                tool_name=tool_name,
                                success=result.get("success", False),
                                result=result,
                                execution_time_ms=tool_time
                            ))
                            
                            tool_call_results.append({
                                "tool_call_id": tool_call_id,
                                "role": "tool",
                                "name": tool_name,
                                "content": json.dumps(result)
                            })
                            
                        except Exception as e:
                            logger.error(f"Error processing tool call {tool_name}: {e}")
                            error_result = {
                                "success": False,
                                "error": f"Tool execution failed: {str(e)}"
                            }
                            
                            tool_results.append(ToolResult(
                                tool_name=tool_name,
                                success=False,
                                result=error_result,
                                error_message=str(e)
                            ))
                            
                            tool_call_results.append({
                                "tool_call_id": tool_call_id,
                                "role": "tool",
                                "name": tool_name,
                                "content": json.dumps(error_result)
                            })
                
                step_tracker.complete_step(step_id, {
                    "tools_called": len(tool_calls),
                    "successful_tools": sum(1 for result in tool_results if result.success),
                    "failed_tools": sum(1 for result in tool_results if not result.success)
                })
                
                # Step 8: Make follow-up LLM call with tool results
                if tool_call_results:
                    step_id = step_tracker.start_step(
                        IntermediaryStepType.LLM_RESPONSE,
                        "Follow-up LLM Call",
                        "Making follow-up call to LLM with tool results"
                    )
                    
                    # Build follow-up context
                    followup_message = {
                        "role": "assistant",
                        "tool_calls": initial_message.get("tool_calls", []),
                        "content": initial_message.get("content", "")
                    }
                    
                    followup_context = context + [followup_message] + tool_call_results
                    
                    followup_start = time.time()
                    final_response = await lmstudio_client.chat_completion(
                        messages=followup_context,
                        temperature=request.temperature,
                        max_tokens=request.max_tokens,
                        stream=False
                    )
                    followup_time = int((time.time() - followup_start) * 1000)
                    
                    step_tracker.complete_step(step_id, {
                        "processing_time_ms": followup_time,
                        "context_messages": len(followup_context),
                        "final_response_tokens": final_response.get("usage", {}).get("total_tokens", 0)
                    })
            
            # Step 9: Extract and store final response
            step_id = step_tracker.start_step(
                IntermediaryStepType.CONTEXT_BUILDING,
                "Store Assistant Response",
                "Extracting and storing final assistant response"
            )
            
            final_message = final_response["choices"][0]["message"]
            response_content = final_message.get("content") or ""
            
            # If no content but tool calls were made, create summary
            if not response_content and tool_calls:
                response_content = f"[Executed {len(tool_calls)} tool(s) - see results above]"
            
            processing_time = time.time() - start_time
            
            # Create LLM response object for debugging
            llm_response_obj = LLMResponse(
                response=final_response,
                timestamp=datetime.now(),
                processing_time_ms=int(processing_time * 1000),
                token_usage=final_response.get("usage")
            )
            
            # Add assistant message with debug metadata
            assistant_message = self.add_message(
                conversation.id,
                "assistant",
                response_content,
                metadata={
                    "model_used": llm_request.model,
                    "temperature": request.temperature,
                    "processing_time": processing_time,
                    "token_count": final_response.get("usage", {}).get("total_tokens"),
                    "tools_used": len(tool_calls),
                    "tool_calls_made": len(tool_results),
                    "debug_enabled": True,
                    "step_count": len(step_tracker.get_steps())
                }
            )
            
            step_tracker.complete_step(step_id, {
                "message_id": assistant_message.id,
                "response_length": len(response_content),
                "total_processing_time_ms": int(processing_time * 1000)
            })
            
            # Build enhanced message with debug information
            enhanced_message = EnhancedMessage(
                id=assistant_message.id,
                conversation_id=conversation.id,
                role=assistant_message.role,
                content=response_content,
                timestamp=assistant_message.timestamp,
                token_count=assistant_message.token_count,
                llm_model=assistant_message.llm_model,
                temperature=assistant_message.temperature,
                processing_time=assistant_message.processing_time,
                
                # Enhanced debug fields
                intermediary_steps=step_tracker.get_steps(),
                llm_request=llm_request if request.include_llm_request else None,
                llm_response=llm_response_obj,
                tool_calls=tool_calls if request.include_tool_details else [],
                tool_results=tool_results if request.include_tool_details else [],
                total_processing_time_ms=int(processing_time * 1000),
                step_count=len(step_tracker.get_steps()),
                error_count=sum(1 for step in step_tracker.get_steps() if not step.success)
            )
            
            # Build debug response
            debug_summary = step_tracker.get_summary()
            
            return ChatResponseWithDebug(
                message=enhanced_message,
                conversation_id=conversation.id,
                processing_time=processing_time,
                token_count=final_response.get("usage", {}).get("total_tokens"),
                total_steps=debug_summary["total_steps"],
                successful_steps=debug_summary["successful_steps"],
                failed_steps=debug_summary["failed_steps"],
                tools_used=[tool.tool_name for tool in tool_calls]
            )
            
        except Exception as e:
            logger.error(f"Error in debug conversation processing: {e}")
            
            # Add error step
            step_tracker.start_step(
                IntermediaryStepType.ERROR,
                "Processing Error",
                f"Error occurred during conversation processing: {str(e)}"
            )
            
            raise e
    
    def _estimate_token_count(self, messages: List[Dict[str, Any]]) -> int:
        """Rough estimate of token count for context messages"""
        total_chars = sum(len(str(msg.get("content", ""))) for msg in messages)
        # Rough estimate: ~4 characters per token
        return int(total_chars / 4)
    
    async def get_conversation_debug_summary(self, conversation_id: int) -> Dict[str, Any]:
        """Get debug summary for a conversation"""
        try:
            messages = self.get_conversation_messages(conversation_id)
            
            # Extract debug information from messages
            debug_messages = []
            total_steps = 0
            total_tools = 0
            total_processing_time = 0
            
            for message in messages:
                if message.role == "assistant" and message.metadata:
                    metadata = message.metadata
                    if metadata.get("debug_enabled"):
                        debug_messages.append(message)
                        total_steps += metadata.get("step_count", 0)
                        total_tools += metadata.get("tools_used", 0)
                        total_processing_time += metadata.get("processing_time", 0)
            
            return {
                "conversation_id": conversation_id,
                "total_messages": len(messages),
                "debug_messages": len(debug_messages),
                "total_steps": total_steps,
                "total_tools_used": total_tools,
                "total_processing_time": total_processing_time,
                "average_processing_time": total_processing_time / len(debug_messages) if debug_messages else 0,
                "debug_coverage": len(debug_messages) / len([m for m in messages if m.role == "assistant"]) if messages else 0
            }
            
        except Exception as e:
            logger.error(f"Error getting conversation debug summary: {e}")
            return {
                "conversation_id": conversation_id,
                "error": str(e)
            }
