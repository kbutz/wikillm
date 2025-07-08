"""
Enhanced Main FastAPI application with MCP Integration
"""
import logging
import time
import json
import os
from contextlib import asynccontextmanager
from typing import List, Optional, Dict, Any, Tuple
from fastapi import FastAPI, HTTPException, Depends, BackgroundTasks, status
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import StreamingResponse
from sqlalchemy.orm import Session
from sqlalchemy import func
from httpx import TimeoutException
from datetime import datetime

# Local imports
from config import settings
from database import init_database, get_db
from models import User, Conversation, Message, UserMemory, SystemLog
from schemas import (
    UserCreate, User as UserSchema,
    ConversationCreate, Conversation as ConversationSchema,
    MessageCreate, Message as MessageSchema,
    ChatRequest, ChatResponse, StreamChatResponse,
    UserMemoryCreate, UserMemory as UserMemorySchema,
    SystemStatus, ErrorResponse, MemoryType
)
from enhanced_schemas import (
    ToolUsageTrace, ToolUsageAnalytics, SystemDebugInfo,
    ChatRequestWithDebug, ChatResponseWithDebug
)
from enhanced_message_schemas import (
    ChatRequestWithDebug as MessageDebugRequest,
    ChatResponseWithDebug as MessageDebugResponse,
    EnhancedMessage
)
from debug_conversation_manager import DebugConversationManager
from tool_usage_manager import ToolUsageManager
from lmstudio_client import lmstudio_client
from memory_manager import MemoryManager, EnhancedMemoryManager
from enhanced_conversation_manager import EnhancedConversationManager

# MCP Integration imports
from mcp_integration import (
    register_mcp_routes, initialize_mcp_system, shutdown_mcp_system,
    get_mcp_tools_for_assistant, handle_mcp_tool_call
)

# Admin routes import
from admin_routes import admin_router

# Debug routes import
from debug_routes import debug_router

# Power user routes import
from power_user_routes import power_user_router

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler('assistant.log'),
        logging.StreamHandler()
    ]
)
logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan management with MCP integration"""
    # Startup
    logger.info("Starting AI Assistant API with MCP integration...")
    init_database()

    # Initialize MCP system
    await initialize_mcp_system()

    # Check LMStudio connection
    lmstudio_connected = await lmstudio_client.health_check()
    if lmstudio_connected:
        logger.info("LMStudio connection successful")
    else:
        logger.warning("LMStudio connection failed - some features may not work")

    yield

    # Shutdown
    logger.info("Shutting down AI Assistant API...")
    await shutdown_mcp_system()


# Create FastAPI app
app = FastAPI(
    title=settings.api_title,
    version=settings.api_version,
    description="Production-ready AI Assistant with MCP integration, conversation memory and user personalization",
    lifespan=lifespan
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=settings.cors_origins,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Register MCP routes
register_mcp_routes(app)

# Register admin routes
app.include_router(admin_router)

# Register debug routes
app.include_router(debug_router)

# Register power user routes
app.include_router(power_user_router)


# Enhanced dependency for getting conversation manager with MCP integration
def get_enhanced_conversation_manager(db: Session = Depends(get_db)) -> EnhancedConversationManager:
    return EnhancedConversationManager(db)


def get_debug_conversation_manager(db: Session = Depends(get_db)) -> DebugConversationManager:
    return DebugConversationManager(db)


def get_memory_manager(db: Session = Depends(get_db)) -> MemoryManager:
    return MemoryManager(db)


def get_enhanced_memory_manager(db: Session = Depends(get_db)) -> EnhancedMemoryManager:
    return EnhancedMemoryManager(db)


# Global tool usage manager
tool_usage_manager = None


def get_tool_usage_manager(db: Session = Depends(get_db)) -> ToolUsageManager:
    """Get tool usage manager instance"""
    global tool_usage_manager
    if tool_usage_manager is None:
        tool_usage_manager = ToolUsageManager(db)
    return tool_usage_manager


# User management endpoints (unchanged)
@app.post("/users/", response_model=UserSchema, status_code=status.HTTP_201_CREATED)
def create_user(user: UserCreate, db: Session = Depends(get_db)):
    """Create a new user"""
    existing_user = db.query(User).filter(User.username == user.username).first()
    if existing_user:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Username already registered"
        )

    db_user = User(**user.dict())
    db.add(db_user)
    db.commit()
    db.refresh(db_user)

    logger.info(f"Created new user: {user.username}")
    return db_user


@app.get("/users/{user_id}", response_model=UserSchema)
def get_user(user_id: int, db: Session = Depends(get_db)):
    """Get user by ID"""
    user = db.query(User).filter(User.id == user_id).first()
    if not user:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="User not found"
        )
    return user


@app.get("/users/", response_model=List[UserSchema])
def list_users(skip: int = 0, limit: int = 100, db: Session = Depends(get_db)):
    """List all users"""
    users = db.query(User).offset(skip).limit(limit).all()
    return users


# Conversation management endpoints with MCP integration
@app.post("/conversations/", response_model=ConversationSchema, status_code=status.HTTP_201_CREATED)
def create_conversation(
    conversation: ConversationCreate,
    db: Session = Depends(get_db),
    conv_manager: EnhancedConversationManager = Depends(get_enhanced_conversation_manager)
):
    """Create a new conversation"""
    user = db.query(User).filter(User.id == conversation.user_id).first()
    if not user:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="User not found"
        )

    return conv_manager.create_conversation(conversation.user_id, conversation.title)


@app.get("/conversations/{conversation_id}", response_model=ConversationSchema)
def get_conversation(
    conversation_id: int,
    user_id: int,
    db: Session = Depends(get_db),
    conv_manager: EnhancedConversationManager = Depends(get_enhanced_conversation_manager)
):
    """Get a specific conversation"""
    conversation = conv_manager.get_conversation(conversation_id, user_id)
    if not conversation:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Conversation not found"
        )
    return conversation


@app.get("/users/{user_id}/conversations", response_model=List[ConversationSchema])
def get_user_conversations(
    user_id: int,
    limit: int = 50,
    db: Session = Depends(get_db),
    conv_manager: EnhancedConversationManager = Depends(get_enhanced_conversation_manager)
):
    """Get all conversations for a user"""
    return conv_manager.get_user_conversations(user_id, limit)


@app.delete("/conversations/{conversation_id}")
def delete_conversation(
    conversation_id: int,
    user_id: int,
    db: Session = Depends(get_db),
    conv_manager: EnhancedConversationManager = Depends(get_enhanced_conversation_manager)
):
    """Delete a conversation"""
    success = conv_manager.delete_conversation(conversation_id, user_id)
    if not success:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Conversation not found"
        )
    return {"message": "Conversation deleted successfully"}


# Tool usage analytics endpoint
@app.get("/conversations/{conversation_id}/tools/analytics")
async def get_conversation_tool_analytics(
    conversation_id: int,
    user_id: int,
    db: Session = Depends(get_db),
    conv_manager: EnhancedConversationManager = Depends(get_enhanced_conversation_manager)
):
    """Get tool usage analytics for a conversation"""
    try:
        # Verify conversation belongs to user
        conversation = conv_manager.get_conversation(conversation_id, user_id)
        if not conversation:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail="Conversation not found"
            )

        analytics = await conv_manager.get_tool_usage_analytics(conversation_id)
        return {"success": True, "data": analytics}

    except Exception as e:
        logger.error(f"Failed to get tool analytics: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to get tool analytics: {str(e)}"
        )


# Enhanced chat endpoints with MCP tool integration
@app.post("/chat", response_model=ChatResponse)
async def chat_with_mcp_tools(
    request: ChatRequest,
    background_tasks: BackgroundTasks,
    db: Session = Depends(get_db),
    conv_manager: EnhancedConversationManager = Depends(get_enhanced_conversation_manager),
    memory_manager: MemoryManager = Depends(get_memory_manager),
    enhanced_memory: EnhancedMemoryManager = Depends(get_enhanced_memory_manager)
):
    """Enhanced chat endpoint with MCP tool integration"""
    start_time = time.time()

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

        # Add user message
        user_message = conv_manager.add_message(
            conversation.id,
            "user",
            request.message
        )

        # Build enhanced conversation context with MCP tools
        context = await conv_manager.build_tool_enhanced_context(
            conversation.id,
            request.user_id,
            max_messages=settings.max_conversation_history,
            include_historical_context=True
        )

        # Get available MCP tools for this request
        available_tools = get_mcp_tools_for_assistant()

        # Prepare LLM request with tools
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

        # Get initial LLM response
        llm_response = await lmstudio_client.chat_completion(**llm_request_params)

        # Process tool calls if present
        processed_response = await conv_manager.process_llm_response_with_tools(
            llm_response, conversation.id
        )

        # Handle follow-up if tool calls were made
        final_response = llm_response
        if processed_response.get("requires_followup"):
            # Add tool results to context and get final response
            tool_results = processed_response.get("tool_results", [])

            # Build follow-up context - handle null content for tool calls
            initial_message = llm_response["choices"][0]["message"]
            followup_message = {
                "role": "assistant",
                "tool_calls": initial_message.get("tool_calls", []),
                "content": initial_message.get("content", "")  # Always include content, default to empty string
            }

            followup_context = context + [followup_message] + tool_results

            # Get final response incorporating tool results
            final_response = await lmstudio_client.chat_completion(
                messages=followup_context,
                temperature=request.temperature,
                max_tokens=request.max_tokens,
                stream=False
            )

        # Extract response content safely
        final_message = final_response["choices"][0]["message"]
        response_content = final_message.get("content") or ""

        # If final response still has no content but has tool calls, create a summary
        if not response_content and final_message.get("tool_calls"):
            response_content = "[Tool calls executed - see tool results above]"

        # Add assistant message
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
                "tool_calls_made": len(processed_response.get("tool_results", []))
            }
        )

        # Background tasks
        background_tasks.add_task(
            extract_and_store_enhanced_memories,
            request.user_id,
            request.message,
            response_content,
            conversation.id,
            enhanced_memory
        )

        # Auto-generate conversation title if it's the first exchange
        if len(conv_manager.get_conversation_messages(conversation.id)) == 2:
            background_tasks.add_task(
                auto_generate_title,
                conversation.id,
                conv_manager
            )

        # Create conversation summary
        message_count = len(conv_manager.get_conversation_messages(conversation.id))
        if message_count >= 5 and message_count % 3 == 0:
            background_tasks.add_task(
                create_conversation_summary_task,
                conversation.id,
                conv_manager
            )

        return ChatResponse(
            message=assistant_message,
            conversation_id=conversation.id,
            processing_time=processing_time,
            token_count=final_response.get("usage", {}).get("total_tokens")
        )

    except TimeoutException as e:
        logger.error(f"LMStudio timeout error: {e}")
        raise HTTPException(
            status_code=status.HTTP_504_GATEWAY_TIMEOUT,
            detail="LMStudio is taking longer than expected to respond. This may be due to high processing load. Please wait a moment and try again."
        )
    except Exception as e:
        logger.error(f"Chat error: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Chat processing failed: {str(e)}"
        )


@app.post("/chat/stream")
async def chat_stream_with_mcp_tools(
    request: ChatRequest,
    background_tasks: BackgroundTasks,
    db: Session = Depends(get_db),
    conv_manager: EnhancedConversationManager = Depends(get_enhanced_conversation_manager),
    memory_manager: MemoryManager = Depends(get_memory_manager),
    enhanced_memory: EnhancedMemoryManager = Depends(get_enhanced_memory_manager)
):
    """Enhanced streaming chat endpoint with MCP tool integration"""
    if not request.stream:
        request.stream = True

    async def generate():
        start_time = time.time()
        full_response = ""
        tool_calls_made = 0

        try:
            # Verify user and get/create conversation
            user = db.query(User).filter(User.id == request.user_id).first()
            if not user:
                yield f"data: {json.dumps({'error': 'User not found'})}\\n\\n"
                return

            if request.conversation_id:
                conversation = conv_manager.get_conversation(request.conversation_id, request.user_id)
                if not conversation:
                    yield f"data: {json.dumps({'error': 'Conversation not found'})}\\n\\n"
                    return
            else:
                conversation = conv_manager.create_conversation(request.user_id)

            # Add user message
            user_message = conv_manager.add_message(
                conversation.id,
                "user",
                request.message
            )

            # Build context with MCP tools
            context = await conv_manager.build_tool_enhanced_context(
                conversation.id,
                request.user_id,
                max_messages=settings.max_conversation_history,
                include_historical_context=True
            )

            # Get available MCP tools
            available_tools = get_mcp_tools_for_assistant()

            # Prepare streaming request
            stream_params = {
                "messages": context,
                "temperature": request.temperature,
                "max_tokens": request.max_tokens,
                "stream": True
            }

            # Add tools if available
            if available_tools:
                stream_params["tools"] = [
                    {
                        "type": "function", 
                        "function": tool["function"]
                    } for tool in available_tools
                ]
                stream_params["tool_choice"] = "auto"

            # Stream LLM response
            tool_calls = []

            async for chunk in lmstudio_client.chat_completion(**stream_params):
                if "choices" in chunk and len(chunk["choices"]) > 0:
                    delta = chunk["choices"][0].get("delta", {})

                    # Handle regular content
                    if "content" in delta and delta["content"]:
                        content = delta["content"]
                        full_response += content

                        response_data = {
                            "chunk": content,
                            "conversation_id": conversation.id,
                            "finished": False,
                            "type": "content"
                        }
                        yield f"data: {json.dumps(response_data)}\\n\\n"

                    # Handle tool calls
                    if "tool_calls" in delta:
                        for tool_call in delta["tool_calls"]:
                            # Process tool call (this is a simplified version)
                            # In a full implementation, you'd need to handle partial tool calls
                            if tool_call.get("function", {}).get("name"):
                                tool_name = tool_call["function"]["name"]
                                if tool_name.startswith("mcp_"):
                                    # Signal tool call to user
                                    tool_data = {
                                        "chunk": f"[Using tool: {tool_name}]",
                                        "conversation_id": conversation.id,
                                        "finished": False,
                                        "type": "tool_call"
                                    }
                                    yield f"data: {json.dumps(tool_data)}\\n\\n"
                                    tool_calls_made += 1

            # Send completion signal
            processing_time = time.time() - start_time
            completion_data = {
                "chunk": "",
                "conversation_id": conversation.id,
                "finished": True,
                "processing_time": processing_time,
                "tool_calls_made": tool_calls_made
            }
            yield f"data: {json.dumps(completion_data)}\\n\\n"

            # Store assistant message
            conv_manager.add_message(
                conversation.id,
                "assistant",
                full_response,
                metadata={
                    "model_used": settings.lmstudio_model,
                    "temperature": request.temperature or settings.default_temperature,
                    "processing_time": processing_time,
                    "mcp_tools_available": len(available_tools),
                    "tool_calls_made": tool_calls_made,
                    "streaming": True
                }
            )

            # Background tasks
            background_tasks.add_task(
                extract_and_store_enhanced_memories,
                request.user_id,
                request.message,
                full_response,
                conversation.id,
                enhanced_memory
            )

        except TimeoutException as e:
            logger.error(f"LMStudio stream timeout error: {e}")
            error_message = "LMStudio is taking longer than expected to respond. This may be due to high processing load. Please wait a moment and try again."
            error_data = {"error": error_message, "finished": True}
            yield f"data: {json.dumps(error_data)}\\n\\n"
        except Exception as e:
            logger.error(f"Stream chat error: {e}")
            error_data = {"error": str(e), "finished": True}
            yield f"data: {json.dumps(error_data)}\\n\\n"

    return StreamingResponse(generate(), media_type="text/plain")


# Enhanced debug chat endpoint
@app.post("/chat/debug", response_model=MessageDebugResponse)
async def chat_with_debug(
    request: MessageDebugRequest,
    background_tasks: BackgroundTasks,
    db: Session = Depends(get_db),
    debug_conv_manager: DebugConversationManager = Depends(get_debug_conversation_manager),
    enhanced_memory: EnhancedMemoryManager = Depends(get_enhanced_memory_manager)
):
    """Enhanced chat endpoint with comprehensive debug information"""
    try:
        # Process message with full debug tracking
        debug_response = await debug_conv_manager.process_message_with_debug(
            request,
            lmstudio_client
        )
        
        # Background tasks for memory extraction
        background_tasks.add_task(
            extract_and_store_enhanced_memories,
            request.user_id,
            request.message,
            debug_response.message.content,
            debug_response.conversation_id,
            enhanced_memory
        )
        
        # Auto-generate conversation title if needed
        if len(debug_conv_manager.get_conversation_messages(debug_response.conversation_id)) == 2:
            background_tasks.add_task(
                auto_generate_title,
                debug_response.conversation_id,
                debug_conv_manager
            )
        
        return debug_response
        
    except Exception as e:
        logger.error(f"Debug chat error: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Debug chat processing failed: {str(e)}"
        )


# Conversation debug summary endpoint
@app.get("/conversations/{conversation_id}/debug")
async def get_conversation_debug_summary(
    conversation_id: int,
    user_id: int,
    db: Session = Depends(get_db),
    debug_conv_manager: DebugConversationManager = Depends(get_debug_conversation_manager)
):
    """Get debug summary for a conversation"""
    try:
        # Verify conversation belongs to user
        conversation = debug_conv_manager.get_conversation(conversation_id, user_id)
        if not conversation:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail="Conversation not found"
            )
        
        debug_summary = await debug_conv_manager.get_conversation_debug_summary(conversation_id)
        return {"success": True, "data": debug_summary}
        
    except Exception as e:
        logger.error(f"Failed to get conversation debug summary: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to get debug summary: {str(e)}"
        )


# Search endpoints (unchanged from original)
@app.get("/users/{user_id}/conversations/search")
async def search_conversations(
    user_id: int,
    q: str,
    limit: int = 5,
    db: Session = Depends(get_db)
):
    """Search past conversations"""
    try:
        from search_manager import SearchManager
        search_manager = SearchManager(db)

        user = db.query(User).filter(User.id == user_id).first()
        if not user:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail="User not found"
            )

        results = await search_manager.search_conversations(user_id, q, limit)

        formatted_results = []
        for summary in results:
            try:
                # Check if conversation is not None before accessing its attributes
                if summary.conversation is not None:
                    formatted_results.append({
                        "conversation_id": summary.conversation_id,
                        "title": summary.conversation.title,
                        "summary": summary.summary,
                        "keywords": summary.keywords,
                        "priority_score": getattr(summary, 'priority_score', 0.0),
                        "created_at": summary.conversation.created_at,
                        "updated_at": summary.conversation.updated_at
                    })
                else:
                    # Use default values if conversation is None
                    formatted_results.append({
                        "conversation_id": summary.conversation_id,
                        "title": "Previous conversation",
                        "summary": summary.summary,
                        "keywords": summary.keywords,
                        "priority_score": getattr(summary, 'priority_score', 0.0),
                        "created_at": None,
                        "updated_at": None
                    })
            except Exception as e:
                logger.warning(f"Could not format search result: {e}")
                continue

        return {
            "query": q,
            "results": formatted_results,
            "total_found": len(formatted_results)
        }
    except Exception as e:
        logger.error(f"Search conversations error: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Search failed: {str(e)}"
        )


# Memory management endpoints (unchanged)
@app.post("/users/{user_id}/memory", response_model=UserMemorySchema)
def add_user_memory(
    user_id: int,
    memory: UserMemoryCreate,
    db: Session = Depends(get_db),
    memory_manager: MemoryManager = Depends(get_memory_manager)
):
    """Add explicit user memory"""
    memory.user_id = user_id
    return memory_manager.store_memory(memory)


@app.get("/users/{user_id}/memory", response_model=List[UserMemorySchema])
def get_user_memory(
    user_id: int,
    memory_type: Optional[MemoryType] = None,
    limit: int = 100,
    db: Session = Depends(get_db),
    memory_manager: MemoryManager = Depends(get_memory_manager)
):
    """Get user memory entries"""
    return memory_manager.get_user_memories(user_id, memory_type, limit=limit)


# Enhanced system status with MCP information
@app.get("/status", response_model=SystemStatus)
async def get_system_status(db: Session = Depends(get_db)):
    """Get enhanced system status including MCP information"""
    lmstudio_connected = await lmstudio_client.health_check()

    # Get database stats
    total_users = db.query(func.count(User.id)).scalar()
    active_conversations = db.query(func.count(Conversation.id)).filter(
        Conversation.is_active == True
    ).scalar()

    # Get MCP status
    try:
        from mcp_integration import mcp_manager
        # Get MCP status
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
            else:
                mcp_servers = []
        except Exception as e:
            logger.error(f"Failed to get MCP status: {e}")
            mcp_servers = []

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
        try:
            tool_debug_info = {}
            if tool_usage_manager:
                tool_debug_info = tool_usage_manager.get_system_debug_info()
        except Exception as e:
            logger.error(f"Failed to get tool debug info: {e}")
            tool_debug_info = {}

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
                "traces_per_minute": 0,  # TODO: Calculate actual metrics
                "average_response_time": 0,
                "success_rate": 0
            },
            recent_errors=[]  # TODO: Add error tracking
        )

    except Exception as e:
        logger.error(f"Failed to get system debug info: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Failed to get debug info: {str(e)}"
        )


# Health check endpoint
@app.get("/health")
def health_check():
    """Simple health check endpoint"""
    try:
        # Get a database session
        db = next(get_db())
        
        lmstudio_connected = lmstudio_client.is_connected()
        total_users = db.query(func.count(User.id)).scalar()
        active_conversations = db.query(func.count(Conversation.id)).filter(
            Conversation.is_active == True
        ).scalar()

        # Get MCP status if available
        try:
            from mcp_integration import mcp_manager
            if mcp_manager:
                mcp_status = mcp_manager.get_server_status()
                mcp_connected_count = sum(1 for s in mcp_status.values() if s["status"] == "connected")
                mcp_total_count = len(mcp_status)
                mcp_tools_count = sum(s["tools_count"] for s in mcp_status.values())
            else:
                mcp_connected_count = 0
                mcp_total_count = 0
                mcp_tools_count = 0
        except Exception as e:
            logger.error(f"Failed to get MCP status: {e}")
            mcp_connected_count = 0
            mcp_total_count = 0
            mcp_tools_count = 0
        
        db.close()
        
    except Exception as e:
        logger.error(f"Health check error: {e}")
        return {
            "status": "unhealthy",
            "error": str(e),
            "version": settings.api_version
        }

    return {
        "status": "healthy",
        "version": settings.api_version,
        "lmstudio_connected": lmstudio_connected,
        "database_connected": True,
        "active_conversations": active_conversations,
        "total_users": total_users,
        "mcp_servers_connected": mcp_connected_count,
        "mcp_servers_total": mcp_total_count,
        "mcp_tools_available": mcp_tools_count
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


async def auto_generate_title(conversation_id: int, conv_manager: EnhancedConversationManager):
    """Background task to auto-generate conversation title"""
    try:
        title = await conv_manager.generate_conversation_title(conversation_id)
        if title:
            conversation = conv_manager.db.query(Conversation).filter(
                Conversation.id == conversation_id
            ).first()
            if conversation:
                conv_manager.update_conversation_title(conversation_id, conversation.user_id, title)
                logger.info(f"Auto-generated title for conversation {conversation_id}: {title}")
    except Exception as e:
        logger.error(f"Failed to generate conversation title: {e}")


async def create_conversation_summary_task(conversation_id: int, conv_manager: EnhancedConversationManager):
    """Background task to create conversation summary"""
    try:
        summary = await conv_manager.create_conversation_summary(conversation_id)
        if summary:
            logger.info(f"Created summary for conversation {conversation_id}")
    except Exception as e:
        logger.error(f"Failed to create conversation summary: {e}")



if __name__ == "__main__":
    import uvicorn
    uvicorn.run(
        "main:app",
        host=settings.api_host,
        port=settings.api_port,
        reload=True
    )
