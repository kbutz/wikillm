"""
Enhanced Debug routes with persistence capabilities
"""
import logging
import time
import os
import sys
import asyncio
import subprocess
from typing import List, Dict, Any, Optional
from fastapi import HTTPException, Depends, BackgroundTasks, status, APIRouter, Response
from sqlalchemy.orm import Session
from sqlalchemy import func
from httpx import TimeoutException
from pydantic import BaseModel

from database import get_db
from models import User, Conversation, Message
from schemas import Message as MessageSchema
from enhanced_schemas import (
    ToolUsageTrace, ToolUsageAnalytics, SystemDebugInfo,
    ChatRequestWithDebug, ChatResponseWithDebug
)
from tool_usage_manager import ToolUsageManager
from enhanced_conversation_manager import EnhancedConversationManager
from debug_persistence_manager import DebugPersistenceManager
from memory_manager import MemoryManager, EnhancedMemoryManager
from lmstudio_client import lmstudio_client
from config import settings
from mcp_integration import get_mcp_tools_for_assistant

logger = logging.getLogger(__name__)

# Create router
debug_router = APIRouter(prefix="/debug", tags=["debug"])

# Global tool usage manager
tool_usage_manager = None

# Models for debug scripts
class DebugScript(BaseModel):
    """Debug script information"""
    name: str
    description: str
    type: str
    path: str

class DebugScriptResult(BaseModel):
    """Debug script execution result"""
    script_name: str
    success: bool
    output: str
    error: Optional[str] = None
    execution_time: float

class DebugPreferenceUpdate(BaseModel):
    """Update debug mode preference"""
    enabled: bool

class DebugDataResponse(BaseModel):
    """Debug data response"""
    conversation_id: int
    has_debug_data: bool
    debug_data: Dict[str, Any]

def get_tool_usage_manager(db: Session = Depends(get_db)) -> ToolUsageManager:
    """Get tool usage manager instance"""
    global tool_usage_manager
    if tool_usage_manager is None:
        tool_usage_manager = ToolUsageManager(db)
    return tool_usage_manager

def get_enhanced_conversation_manager(db: Session = Depends(get_db)) -> EnhancedConversationManager:
    return EnhancedConversationManager(db)

def get_enhanced_memory_manager(db: Session = Depends(get_db)) -> EnhancedMemoryManager:
    return EnhancedMemoryManager(db)

def get_debug_persistence_manager(db: Session = Depends(get_db)) -> DebugPersistenceManager:
    return DebugPersistenceManager(db)

# Enhanced chat endpoint with comprehensive tool usage tracing and persistence
@debug_router.post("/chat", response_model=ChatResponseWithDebug)
async def chat_with_debug_tracing(
    request: ChatRequestWithDebug,
    background_tasks: BackgroundTasks,
    db: Session = Depends(get_db),
    conv_manager: EnhancedConversationManager = Depends(get_enhanced_conversation_manager),
    enhanced_memory: EnhancedMemoryManager = Depends(get_enhanced_memory_manager),
    tool_manager: ToolUsageManager = Depends(get_tool_usage_manager),
    debug_persistence: DebugPersistenceManager = Depends(get_debug_persistence_manager)
):
    """Enhanced chat endpoint with comprehensive tool usage tracing and persistence"""
    start_time = time.time()
    trace_id = None
    debug_session = None

    try:
        # Verify user exists
        user = db.query(User).filter(User.id == request.user_id).first()
        if not user:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail="User not found"
            )

        # Get or create conversation
        if request.conversation_id:
            conversation = conv_manager.get_conversation(request.conversation_id, request.user_id)
            if not conversation:
                raise HTTPException(
                    status_code=status.HTTP_404_NOT_FOUND,
                    detail="Conversation not found"
                )
        else:
            conversation = conv_manager.create_conversation(request.user_id)

        # Initialize debug session if debug tracing is enabled
        if request.enable_tool_trace:
            debug_session = debug_persistence.get_or_create_debug_session(
                conversation.id, request.user_id
            )

            # Initialize tool usage tracing
            trace_id = tool_manager.create_trace(
                conversation_id=conversation.id,
                user_id=request.user_id
            )

        # Add user message
        user_message = conv_manager.add_message(
            conversation.id,
            "user",
            request.message
        )

        # Mark message as debug-enabled if tracing is on
        if request.enable_tool_trace:
            user_message.debug_enabled = True
            db.commit()

        # Update trace with message ID
        if trace_id:
            tool_manager.active_traces[trace_id].message_id = user_message.id

        # Step 1: Context Building with tracing
        if trace_id:
            async with tool_manager.trace_step(
                trace_id,
                tool_name="context_builder",
                tool_type="internal",
                step_type="processing",
                description="Building conversation context with enhanced features",
                input_data={"max_messages": settings.max_conversation_history}
            ):
                context = await conv_manager.build_tool_enhanced_context(
                    conversation.id,
                    request.user_id,
                    max_messages=settings.max_conversation_history,
                    include_historical_context=True
                )

                # Store debug step
                if debug_session:
                    debug_persistence.store_debug_step(
                        message_id=user_message.id,
                        debug_session_id=debug_session.id,
                        step_type="context_building",
                        step_order=1,
                        title="Context Building",
                        description="Building conversation context with enhanced features",
                        success=True,
                        input_data={"max_messages": settings.max_conversation_history},
                        output_data={"context_size": len(context)}
                    )
        else:
            context = await conv_manager.build_tool_enhanced_context(
                conversation.id,
                request.user_id,
                max_messages=settings.max_conversation_history,
                include_historical_context=True
            )

        # Step 2: Tool Discovery with tracing
        if trace_id:
            async with tool_manager.trace_step(
                trace_id,
                tool_name="mcp_tool_discovery",
                tool_type="mcp",
                step_type="discovery",
                description="Discovering available MCP tools"
            ):
                available_tools = get_mcp_tools_for_assistant()

                # Store debug step
                if debug_session:
                    debug_persistence.store_debug_step(
                        message_id=user_message.id,
                        debug_session_id=debug_session.id,
                        step_type="tool_discovery",
                        step_order=2,
                        title="Tool Discovery",
                        description="Discovering available MCP tools",
                        success=True,
                        output_data={"tools_discovered": len(available_tools)}
                    )
        else:
            available_tools = get_mcp_tools_for_assistant()

        # Step 3: LLM Request with tracing
        debug_context = {}  # Always create debug context
        
        if trace_id:
            async with tool_manager.trace_step(
                trace_id,
                tool_name="lmstudio_llm",
                tool_type="llm",
                step_type="inference",
                description="Making LLM inference call",
                input_data={
                    "model": settings.lmstudio_model,
                    "context_size": len(context),
                    "tools_available": len(available_tools)
                }
            ):
                # Prepare LLM request
                llm_request_params = {
                    "messages": context,
                    "temperature": request.temperature,
                    "max_tokens": request.max_tokens,
                    "stream": False,
                    "debug_context": debug_context
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

                # Get LLM response
                llm_response = await lmstudio_client.chat_completion(**llm_request_params)

                # Store LLM request/response with debug data
                if debug_session:
                    debug_persistence.store_llm_request(
                        message_id=user_message.id,
                        model=settings.lmstudio_model,
                        request_messages=context,
                        response_data=llm_response,
                        temperature=request.temperature,
                        max_tokens=request.max_tokens,
                        stream=False,
                        processing_time_ms=debug_context.get('llm_processing_time_ms', int((time.time() - start_time) * 1000)),
                        token_usage=llm_response.get("usage"),
                        tools_available=available_tools
                    )

                    # Store debug step with full request payload
                    debug_persistence.store_debug_step(
                        message_id=user_message.id,
                        debug_session_id=debug_session.id,
                        step_type="llm_request",
                        step_order=3,
                        title="LLM Request",
                        description="Making LLM inference call",
                        success=True,
                        input_data={
                            "model": settings.lmstudio_model,
                            "context_size": len(context),
                            "tools_available": len(available_tools),
                            "full_request_payload": debug_context.get('llm_request_payload', {}),
                            "temperature": request.temperature,
                            "max_tokens": request.max_tokens,
                            "messages": context
                        },
                        output_data={
                            "token_usage": llm_response.get("usage"),
                            "has_tool_calls": bool(llm_response.get("choices", [{}])[0].get("message", {}).get("tool_calls")),
                            "processing_time_ms": debug_context.get('llm_processing_time_ms'),
                            "full_response": debug_context.get('llm_response_raw', {}),
                            "response_tokens": debug_context.get('llm_response_tokens', 0)
                        }
                    )
        else:
            # Standard processing without tracing - still capture debug data
            llm_request_params = {
                "messages": context,
                "temperature": request.temperature,
                "max_tokens": request.max_tokens,
                "stream": False,
                "debug_context": debug_context
            }

            if available_tools:
                llm_request_params["tools"] = [
                    {
                        "type": "function",
                        "function": tool["function"]
                    } for tool in available_tools
                ]
                llm_request_params["tool_choice"] = "auto"

            llm_response = await lmstudio_client.chat_completion(**llm_request_params)
            
            # Store LLM request/response even without tracing if debug is enabled
            if debug_session:
                debug_persistence.store_llm_request(
                    message_id=user_message.id,
                    model=settings.lmstudio_model,
                    request_messages=context,
                    response_data=llm_response,
                    temperature=request.temperature,
                    max_tokens=request.max_tokens,
                    stream=False,
                    processing_time_ms=debug_context.get('llm_processing_time_ms', 0),
                    token_usage=llm_response.get("usage"),
                    tools_available=available_tools
                )

        # Step 4: Tool Processing with tracing
        if trace_id:
            async with tool_manager.trace_step(
                trace_id,
                tool_name="tool_processor",
                tool_type="internal",
                step_type="processing",
                description="Processing tool calls from LLM response"
            ):
                processed_response = await conv_manager.process_llm_response_with_tools(
                    llm_response, conversation.id
                )

                # Store debug step
                if debug_session:
                    debug_persistence.store_debug_step(
                        message_id=user_message.id,
                        debug_session_id=debug_session.id,
                        step_type="tool_processing",
                        step_order=4,
                        title="Tool Processing",
                        description="Processing tool calls from LLM response",
                        success=True,
                        input_data={"has_tool_calls": bool(llm_response.get("choices", [{}])[0].get("message", {}).get("tool_calls"))},
                        output_data={
                            "requires_followup": processed_response.get("requires_followup", False),
                            "tool_results_count": len(processed_response.get("tool_results", []))
                        }
                    )
        else:
            processed_response = await conv_manager.process_llm_response_with_tools(
                llm_response, conversation.id
            )

        # Handle follow-up if needed
        final_response = llm_response
        if processed_response.get("requires_followup"):
            if trace_id:
                async with tool_manager.trace_step(
                    trace_id,
                    tool_name="followup_processor",
                    tool_type="internal",
                    step_type="processing",
                    description="Processing follow-up LLM call with tool results"
                ):
                    tool_results = processed_response.get("tool_results", [])
                    initial_message = llm_response["choices"][0]["message"]
                    followup_message = {
                        "role": "assistant",
                        "tool_calls": initial_message.get("tool_calls", []),
                        "content": initial_message.get("content", "")
                    }
                    followup_context = context + [followup_message] + tool_results
                    final_response = await lmstudio_client.chat_completion(
                        messages=followup_context,
                        temperature=request.temperature,
                        max_tokens=request.max_tokens,
                        stream=False,
                        debug_context=debug_context
                    )

                    # Store followup debug step
                    if debug_session:
                        debug_persistence.store_debug_step(
                            message_id=user_message.id,
                            debug_session_id=debug_session.id,
                            step_type="followup_processing",
                            step_order=5,
                            title="Follow-up Processing",
                            description="Processing follow-up LLM call with tool results",
                            success=True,
                            input_data={"tool_results_count": len(tool_results)},
                            output_data={"final_response_token_usage": final_response.get("usage")}
                        )

        # Extract response content
        final_message = final_response["choices"][0]["message"]
        response_content = final_message.get("content") or ""

        if not response_content and final_message.get("tool_calls"):
            response_content = "[Tool calls executed - see tool results above]"

        # Store assistant message
        processing_time = time.time() - start_time
        assistant_message = conv_manager.add_message(
            conversation.id,
            "assistant",
            response_content,
            metadata={
                "model_used": settings.lmstudio_model,
                "temperature": request.temperature or settings.default_temperature,
                "processing_time": processing_time,
                "token_count": final_response.get("usage", {}).get("total_tokens"),
                "mcp_tools_available": len(available_tools),
                "tool_calls_made": len(processed_response.get("tool_results", [])),
                "trace_id": trace_id if trace_id else None
            }
        )

        # Mark assistant message as debug-enabled if tracing is on
        if request.enable_tool_trace:
            assistant_message.debug_enabled = True
            db.commit()

        # Finalize trace
        trace = None
        if trace_id:
            trace = tool_manager.finalize_trace(trace_id)

        # Background memory extraction
        if request.user_id:
            background_tasks.add_task(
                extract_and_store_enhanced_memories,
                request.user_id,
                request.message,
                response_content,
                conversation.id,
                enhanced_memory
            )

        # Convert SQLAlchemy Message model to Pydantic Message model for serialization
        message_schema = MessageSchema.model_validate(assistant_message)

        # Always set debug_enabled flag when debug is requested
        if request.enable_tool_trace:
            message_schema.debug_enabled = True

        # If debug is enabled, fetch debug data and add it to the message
        if request.enable_tool_trace and debug_session:
            # Wait a moment for the database to be updated
            import asyncio
            await asyncio.sleep(0.1)
            
            # Get debug steps for this message
            debug_steps = debug_persistence.get_message_debug_steps(assistant_message.id)
            logger.info(f"Retrieved {len(debug_steps)} debug steps for message {assistant_message.id}")

            # Get LLM requests for this message
            llm_requests = debug_persistence.get_message_llm_requests(assistant_message.id)
            logger.info(f"Retrieved {len(llm_requests)} LLM requests for message {assistant_message.id}")

            if debug_steps:
                # Convert debug steps to intermediary steps format
                intermediary_steps = []
                for step in debug_steps:
                    intermediary_step = {
                        "step_id": step.step_id,
                        "step_type": step.step_type,
                        "timestamp": step.timestamp.isoformat(),
                        "title": step.title,
                        "description": step.description,
                        "data": step.input_data or {},
                        "duration_ms": step.duration_ms,
                        "success": step.success,
                        "error_message": step.error_message
                    }
                    intermediary_steps.append(intermediary_step)

                # Add intermediary steps to message
                message_schema.intermediary_steps = intermediary_steps
                logger.info(f"Added {len(intermediary_steps)} debug steps to message {assistant_message.id}")

            if llm_requests and len(llm_requests) > 0:
                # Get the first LLM request (usually there's only one)
                llm_request = llm_requests[0]
                logger.info(f"Processing LLM request {llm_request.request_id} for message {assistant_message.id}")

                # Convert to LLM request format with full payload
                llm_request_data = {
                    "model": llm_request.model,
                    "messages": llm_request.request_messages,
                    "temperature": llm_request.temperature,
                    "max_tokens": llm_request.max_tokens,
                    "tools": llm_request.tools_available,
                    "tool_choice": "auto" if llm_request.tools_available else None,
                    "stream": llm_request.stream,
                    "timestamp": llm_request.timestamp.isoformat(),
                    "request_id": llm_request.request_id,
                    "processing_time_ms": llm_request.processing_time_ms
                }

                # Add LLM request to message
                message_schema.llm_request = llm_request_data
                logger.info(f"Added LLM request data to message {assistant_message.id}")

                # Convert to LLM response format
                llm_response_data = {
                    "response": llm_request.response_data,
                    "timestamp": llm_request.timestamp.isoformat(),
                    "processing_time_ms": llm_request.processing_time_ms,
                    "token_usage": llm_request.token_usage,
                    "request_id": llm_request.request_id
                }

                # Add LLM response to message
                message_schema.llm_response = llm_response_data
                logger.info(f"Added LLM response data to message {assistant_message.id}")

                # Add tool calls and results if available
                if llm_request.tool_calls:
                    message_schema.tool_calls = llm_request.tool_calls
                    logger.info(f"Added {len(llm_request.tool_calls)} tool calls to message {assistant_message.id}")

                if llm_request.tool_results:
                    message_schema.tool_results = llm_request.tool_results
                    logger.info(f"Added {len(llm_request.tool_results)} tool results to message {assistant_message.id}")
            else:
                logger.warning(f"No LLM request data found for message {assistant_message.id}")
                
            # If no debug data was found, add a debug flag and log
            if not debug_steps and not llm_requests:
                logger.warning(f"No debug data found for message {assistant_message.id} despite debug being enabled")
                message_schema.debug_data = {
                    "debug_enabled": True,
                    "debug_session_id": debug_session.id,
                    "message_id": assistant_message.id,
                    "error": "No debug data found in database",
                    "debug_context_captured": bool(debug_context),
                    "debug_context_keys": list(debug_context.keys()) if debug_context else []
                }
            else:
                # Add debug context info for troubleshooting
                message_schema.debug_data = {
                    "debug_enabled": True,
                    "debug_session_id": debug_session.id,
                    "message_id": assistant_message.id,
                    "debug_steps_count": len(debug_steps),
                    "llm_requests_count": len(llm_requests),
                    "debug_context_captured": bool(debug_context),
                    "debug_context_keys": list(debug_context.keys()) if debug_context else []
                }

        return ChatResponseWithDebug(
            message=message_schema,
            conversation_id=conversation.id,
            processing_time=processing_time,
            token_count=final_response.get("usage", {}).get("total_tokens"),
            tool_trace=trace,
            debug_enabled=request.enable_tool_trace,
            # Debug response fields
            total_steps=len(debug_steps) if debug_steps else 0,
            successful_steps=len([s for s in debug_steps if s.success]) if debug_steps else 0,
            failed_steps=len([s for s in debug_steps if not s.success]) if debug_steps else 0,
            tools_used=processed_response.get("tool_results", [])
        )

    except TimeoutException as e:
        logger.error(f"LMStudio timeout error: {e}")
        if trace_id:
            tool_manager.finalize_trace(trace_id)
        raise HTTPException(
            status_code=status.HTTP_504_GATEWAY_TIMEOUT,
            detail="LMStudio timeout - please try again"
        )
    except Exception as e:
        logger.error(f"Debug chat error: {e}")
        if trace_id:
            tool_manager.finalize_trace(trace_id)
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Chat processing failed: {str(e)}"
        )

# Debug data persistence endpoints
@debug_router.get("/conversations/{conversation_id}/data", response_model=DebugDataResponse)
async def get_conversation_debug_data(
    conversation_id: int,
    user_id: int,
    db: Session = Depends(get_db),
    debug_persistence: DebugPersistenceManager = Depends(get_debug_persistence_manager)
):
    """Get persistent debug data for a conversation"""
    try:
        debug_data = debug_persistence.get_conversation_debug_data(conversation_id, user_id)

        return DebugDataResponse(
            conversation_id=conversation_id,
            has_debug_data=len(debug_data["messages"]) > 0,
            debug_data=debug_data
        )
    except ValueError as e:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=str(e)
        )
    except Exception as e:
        logger.error(f"Failed to get debug data: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to get debug data: {str(e)}"
        )

@debug_router.get("/conversations/{conversation_id}/summary")
async def get_conversation_debug_summary(
    conversation_id: int,
    user_id: int,
    db: Session = Depends(get_db),
    debug_persistence: DebugPersistenceManager = Depends(get_debug_persistence_manager)
):
    """Get debug summary for a conversation"""
    try:
        summary = debug_persistence.get_debug_session_summary(conversation_id, user_id)
        return {"success": True, "data": summary}
    except Exception as e:
        logger.error(f"Failed to get debug summary: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to get debug summary: {str(e)}"
        )

@debug_router.get("/users/{user_id}/preference")
async def get_user_debug_preference(
    user_id: int,
    db: Session = Depends(get_db),
    debug_persistence: DebugPersistenceManager = Depends(get_debug_persistence_manager)
):
    """Get user's debug mode preference"""
    try:
        enabled = debug_persistence.get_user_debug_preference(user_id)
        return {"success": True, "data": {"enabled": enabled}}
    except Exception as e:
        logger.error(f"Failed to get debug preference: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to get debug preference: {str(e)}"
        )

@debug_router.post("/users/{user_id}/preference")
async def set_user_debug_preference(
    user_id: int,
    preference: DebugPreferenceUpdate,
    db: Session = Depends(get_db),
    debug_persistence: DebugPersistenceManager = Depends(get_debug_persistence_manager)
):
    """Set user's debug mode preference"""
    try:
        # Verify user exists
        user = db.query(User).filter(User.id == user_id).first()
        if not user:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail="User not found"
            )

        debug_persistence.set_user_debug_preference(user_id, preference.enabled)
        return {"success": True, "message": "Debug preference updated"}
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Failed to set debug preference: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to set debug preference: {str(e)}"
        )

@debug_router.post("/sessions/{session_id}/end")
async def end_debug_session(
    session_id: str,
    db: Session = Depends(get_db),
    debug_persistence: DebugPersistenceManager = Depends(get_debug_persistence_manager)
):
    """End a debug session"""
    try:
        success = debug_persistence.end_debug_session(session_id)
        if success:
            return {"success": True, "message": "Debug session ended"}
        else:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail="Debug session not found"
            )
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Failed to end debug session: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to end debug session: {str(e)}"
        )

@debug_router.post("/cleanup")
async def cleanup_old_debug_data(
    days_old: int = 30,
    db: Session = Depends(get_db),
    debug_persistence: DebugPersistenceManager = Depends(get_debug_persistence_manager)
):
    """Clean up old debug data"""
    try:
        count = debug_persistence.cleanup_old_debug_data(days_old)
        return {"success": True, "message": f"Cleaned up {count} old debug records"}
    except Exception as e:
        logger.error(f"Failed to cleanup debug data: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to cleanup debug data: {str(e)}"
        )

# Existing endpoints (keep all existing functionality)
@debug_router.get("/conversations/{conversation_id}/tool-usage", response_model=ToolUsageAnalytics)
async def get_tool_usage_analytics(
    conversation_id: int,
    user_id: int,
    db: Session = Depends(get_db),
    tool_manager: ToolUsageManager = Depends(get_tool_usage_manager)
):
    """Get comprehensive tool usage analytics for a conversation"""
    try:
        # Verify conversation belongs to user
        conversation = db.query(Conversation).filter(
            Conversation.id == conversation_id,
            Conversation.user_id == user_id
        ).first()

        if not conversation:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail="Conversation not found"
            )

        analytics = tool_manager.get_analytics(conversation_id)
        return analytics

    except Exception as e:
        logger.error(f"Failed to get tool usage analytics: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to get analytics: {str(e)}"
        )

@debug_router.get("/conversations/{conversation_id}/tool-traces", response_model=List[ToolUsageTrace])
async def get_conversation_tool_traces(
    conversation_id: int,
    user_id: int,
    limit: int = 10,
    db: Session = Depends(get_db),
    tool_manager: ToolUsageManager = Depends(get_tool_usage_manager)
):
    """Get tool usage traces for a conversation"""
    try:
        # Verify conversation belongs to user
        conversation = db.query(Conversation).filter(
            Conversation.id == conversation_id,
            Conversation.user_id == user_id
        ).first()

        if not conversation:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail="Conversation not found"
            )

        traces = tool_manager.get_conversation_traces(conversation_id, limit)
        return traces

    except Exception as e:
        logger.error(f"Failed to get tool traces: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to get traces: {str(e)}"
        )

@debug_router.get("/users/{user_id}/tool-traces", response_model=List[ToolUsageTrace])
async def get_user_tool_traces(
    user_id: int,
    limit: int = 20,
    db: Session = Depends(get_db),
    tool_manager: ToolUsageManager = Depends(get_tool_usage_manager)
):
    """Get tool usage traces for a user"""
    try:
        # Verify user exists
        user = db.query(User).filter(User.id == user_id).first()
        if not user:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail="User not found"
            )

        traces = tool_manager.get_user_traces(user_id, limit)
        return traces

    except Exception as e:
        logger.error(f"Failed to get user traces: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to get traces: {str(e)}"
        )

@debug_router.get("/system-info", response_model=SystemDebugInfo)
async def get_system_debug_info(
    db: Session = Depends(get_db),
    tool_manager: ToolUsageManager = Depends(get_tool_usage_manager)
):
    """Get comprehensive system debug information"""
    try:
        # Get system status
        lmstudio_connected = await lmstudio_client.health_check()
        total_users = db.query(func.count(User.id)).scalar()
        active_conversations = db.query(func.count(Conversation.id)).filter(
            Conversation.is_active == True
        ).scalar()

        # Get MCP status
        mcp_servers = []
        try:
            from mcp_integration import mcp_manager
            if mcp_manager:
                mcp_status = mcp_manager.get_server_status()
                mcp_servers = [
                    {
                        "server_id": server_id,
                        "status": status_info["status"],
                        "tools_count": status_info["tools_count"],
                        "last_seen": status_info.get("last_seen", "unknown")
                    }
                    for server_id, status_info in mcp_status.items()
                ]
        except Exception as e:
            logger.error(f"Failed to get MCP status: {e}")

        # Get available tools
        available_tools = get_mcp_tools_for_assistant()
        tool_list = [
            {
                "name": tool.get("function", {}).get("name", "unknown"),
                "server_id": tool.get("mcp_server_id", "unknown"),
                "description": tool.get("function", {}).get("description", "")
            }
            for tool in available_tools
        ]

        # Get tool usage debug info
        tool_debug_info = tool_manager.get_system_debug_info()

        return SystemDebugInfo(
            system_status={
                "lmstudio_connected": lmstudio_connected,
                "database_connected": True,
                "total_users": total_users,
                "active_conversations": active_conversations,
                "api_version": settings.api_version
            },
            mcp_servers=mcp_servers,
            available_tools=tool_list,
            memory_stats=tool_debug_info,
            performance_metrics={
                "traces_per_minute": 0,
                "average_response_time": 0,
                "success_rate": 0
            },
            recent_errors=[]
        )

    except Exception as e:
        logger.error(f"Failed to get system debug info: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to get debug info: {str(e)}"
        )

# Debug script endpoints (keep existing functionality)
def get_debug_scripts() -> List[DebugScript]:
    """Get list of available debug scripts"""
    scripts = []
    script_dir = os.path.dirname(os.path.abspath(__file__))

    # Define known debug scripts with descriptions
    known_scripts = {
        "verify_memory_system.py": {
            "description": "Verify that all memory aspects are working correctly",
            "type": "verify"
        },
        "enhanced_verify_memory_system.py": {
            "description": "Verify that all memory aspects are working correctly",
            "type": "verify"
        },
        "verify_admin_implementation.py": {
            "description": "Verify admin functionality implementation",
            "type": "verify"
        },
        "check_memory_status.py": {
            "description": "Check the status of the memory system",
            "type": "check"
        },
        "debug_mcp.py": {
            "description": "Debug MCP functionality",
            "type": "debug"
        },
        "test_mcp.py": {
            "description": "Test MCP functionality",
            "type": "test"
        },
        "test_filesystem_server.py": {
            "description": "Test filesystem server functionality",
            "type": "test"
        },
        "test_mcp_fix.py": {
            "description": "Test MCP fixes",
            "type": "test"
        },
        "enhanced_migration.py": {
            "description": "Run enhanced database migration",
            "type": "migration"
        },
        "memory_system_test.py": {
            "description": "Test memory system functionality",
            "type": "test"
        }
    }

    # Find all Python files that match known debug scripts
    for script_name, script_info in known_scripts.items():
        script_path = os.path.join(script_dir, script_name)
        if os.path.exists(script_path):
            scripts.append(DebugScript(
                name=script_name,
                description=script_info["description"],
                type=script_info["type"],
                path=script_path
            ))

    return scripts

@debug_router.get("/scripts", response_model=List[DebugScript])
async def list_debug_scripts():
    """List all available debug scripts"""
    try:
        scripts = get_debug_scripts()
        return scripts
    except Exception as e:
        logger.error(f"Failed to list debug scripts: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to list debug scripts: {str(e)}"
        )

@debug_router.post("/scripts/{script_name}/run", response_model=DebugScriptResult)
async def run_debug_script(script_name: str):
    """Run a debug script and return the result"""
    start_time = time.time()

    try:
        # Get all available scripts
        scripts = get_debug_scripts()

        # Find the requested script
        script = next((s for s in scripts if s.name == script_name), None)
        if not script:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"Script {script_name} not found"
            )

        # Run the script
        result = await execute_script(script.path)

        execution_time = time.time() - start_time
        return DebugScriptResult(
            script_name=script_name,
            success=result["success"],
            output=result["output"],
            error=result["error"],
            execution_time=execution_time
        )
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Failed to run debug script {script_name}: {e}")
        execution_time = time.time() - start_time
        return DebugScriptResult(
            script_name=script_name,
            success=False,
            output="",
            error=str(e),
            execution_time=execution_time
        )

async def execute_script(script_path: str) -> Dict[str, Any]:
    """Execute a Python script and return the result"""
    try:
        # Check if script exists
        if not os.path.exists(script_path):
            return {
                "success": False,
                "output": "",
                "error": f"Script not found: {script_path}"
            }

        # Run the script as a subprocess and capture output
        process = await asyncio.create_subprocess_exec(
            sys.executable, script_path,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE
        )

        stdout, stderr = await process.communicate()

        if process.returncode == 0:
            return {
                "success": True,
                "output": stdout.decode(),
                "error": None
            }
        else:
            return {
                "success": False,
                "output": stdout.decode(),
                "error": stderr.decode()
            }
    except Exception as e:
        logger.error(f"Script execution error: {e}")
        return {
            "success": False,
            "output": "",
            "error": str(e)
        }

# Background task functions
async def extract_and_store_enhanced_memories(
    user_id: int,
    user_message: str,
    assistant_response: str,
    conversation_id: int,
    enhanced_memory: EnhancedMemoryManager
):
    """Enhanced background task to extract and store facts and memories"""
    try:
        memories = await enhanced_memory.extract_and_store_facts(
            user_id, user_message, assistant_response, conversation_id
        )
        if memories:
            logger.info(f"Stored {len(memories)} enhanced memories for user {user_id}")
    except Exception as e:
        logger.error(f"Failed to extract enhanced memories: {e}")
