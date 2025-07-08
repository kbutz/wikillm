"""
Debug routes for tool usage tracing and system monitoring
"""
import logging
import time
from typing import List
from fastapi import HTTPException, Depends, BackgroundTasks, status, APIRouter
from sqlalchemy.orm import Session
from sqlalchemy import func
from httpx import TimeoutException

from database import get_db
from models import User, Conversation, Message
from enhanced_schemas import (
    ToolUsageTrace, ToolUsageAnalytics, SystemDebugInfo,
    ChatRequestWithDebug, ChatResponseWithDebug
)
from tool_usage_manager import ToolUsageManager
from enhanced_conversation_manager import EnhancedConversationManager
from memory_manager import MemoryManager, EnhancedMemoryManager
from lmstudio_client import lmstudio_client
from config import settings
from mcp_integration import get_mcp_tools_for_assistant

logger = logging.getLogger(__name__)

# Create router
debug_router = APIRouter(prefix="/debug", tags=["debug"])

# Global tool usage manager
tool_usage_manager = None

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

# Enhanced chat endpoint with comprehensive tool usage tracing
@debug_router.post("/chat", response_model=ChatResponseWithDebug)
async def chat_with_debug_tracing(
    request: ChatRequestWithDebug,
    background_tasks: BackgroundTasks,
    db: Session = Depends(get_db),
    conv_manager: EnhancedConversationManager = Depends(get_enhanced_conversation_manager),
    enhanced_memory: EnhancedMemoryManager = Depends(get_enhanced_memory_manager),
    tool_manager: ToolUsageManager = Depends(get_tool_usage_manager)
):
    """Enhanced chat endpoint with comprehensive tool usage tracing"""
    start_time = time.time()
    trace_id = None

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

        # Initialize tool usage tracing
        if request.enable_tool_trace:
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
        else:
            available_tools = get_mcp_tools_for_assistant()

        # Step 3: LLM Request with tracing
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

                # Get LLM response
                llm_response = await lmstudio_client.chat_completion(**llm_request_params)
        else:
            # Standard processing without tracing
            llm_request_params = {
                "messages": context,
                "temperature": request.temperature,
                "max_tokens": request.max_tokens,
                "stream": False
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
                        stream=False
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

        return ChatResponseWithDebug(
            message=assistant_message,
            conversation_id=conversation.id,
            processing_time=processing_time,
            token_count=final_response.get("usage", {}).get("total_tokens"),
            tool_trace=trace,
            debug_enabled=request.enable_tool_trace
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


# Tool usage analytics endpoints
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
