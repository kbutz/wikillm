"""
Main FastAPI application for AI Assistant
"""
import logging
import time
import json
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
from lmstudio_client import lmstudio_client
from memory_manager import MemoryManager, EnhancedMemoryManager
from conversation_manager import ConversationManager

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
    """Application lifespan management"""
    # Startup
    logger.info("Starting AI Assistant API...")
    init_database()

    # Check LMStudio connection
    lmstudio_connected = await lmstudio_client.health_check()
    if lmstudio_connected:
        logger.info("LMStudio connection successful")
    else:
        logger.warning("LMStudio connection failed - some features may not work")

    yield

    # Shutdown
    logger.info("Shutting down AI Assistant API...")


# Create FastAPI app
app = FastAPI(
    title=settings.api_title,
    version=settings.api_version,
    description="Production-ready AI Assistant with conversation memory and user personalization",
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


# Dependency for getting conversation manager
def get_conversation_manager(db: Session = Depends(get_db)) -> ConversationManager:
    return ConversationManager(db)


def get_memory_manager(db: Session = Depends(get_db)) -> MemoryManager:
    return MemoryManager(db)


def get_enhanced_memory_manager(db: Session = Depends(get_db)) -> EnhancedMemoryManager:
    return EnhancedMemoryManager(db)


# User management endpoints
@app.post("/users/", response_model=UserSchema, status_code=status.HTTP_201_CREATED)
def create_user(user: UserCreate, db: Session = Depends(get_db)):
    """Create a new user"""
    # Check if user already exists
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


# Conversation management endpoints
@app.post("/conversations/", response_model=ConversationSchema, status_code=status.HTTP_201_CREATED)
def create_conversation(
    conversation: ConversationCreate,
    db: Session = Depends(get_db),
    conv_manager: ConversationManager = Depends(get_conversation_manager)
):
    """Create a new conversation"""
    # Verify user exists
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
    conv_manager: ConversationManager = Depends(get_conversation_manager)
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
    conv_manager: ConversationManager = Depends(get_conversation_manager)
):
    """Get all conversations for a user"""
    return conv_manager.get_user_conversations(user_id, limit)


@app.delete("/conversations/{conversation_id}")
def delete_conversation(
    conversation_id: int,
    user_id: int,
    db: Session = Depends(get_db),
    conv_manager: ConversationManager = Depends(get_conversation_manager)
):
    """Delete a conversation"""
    success = conv_manager.delete_conversation(conversation_id, user_id)
    if not success:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Conversation not found"
        )
    return {"message": "Conversation deleted successfully"}


# Search endpoints
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
        
        # Verify user exists
        user = db.query(User).filter(User.id == user_id).first()
        if not user:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail="User not found"
            )
        
        results = await search_manager.search_conversations(user_id, q, limit)
        
        # Format results for API response
        formatted_results = []
        for summary in results:
            try:
                formatted_results.append({
                    "conversation_id": summary.conversation_id,
                    "title": summary.conversation.title,
                    "summary": summary.summary,
                    "keywords": summary.keywords,
                    "priority_score": getattr(summary, 'priority_score', 0.0),
                    "created_at": summary.conversation.created_at,
                    "updated_at": summary.conversation.updated_at
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


@app.get("/users/{user_id}/priorities")
async def get_user_priorities(
    user_id: int,
    db: Session = Depends(get_db)
):
    """Extract priorities from conversation history"""
    try:
        from search_manager import SearchManager
        search_manager = SearchManager(db)
        
        # Verify user exists
        user = db.query(User).filter(User.id == user_id).first()
        if not user:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail="User not found"
            )
        
        priorities = await search_manager.get_user_priorities(user_id)
        
        return {
            "user_id": user_id,
            "extracted_at": datetime.now().isoformat(),
            "priorities": priorities
        }
    except Exception as e:
        logger.error(f"Get priorities error: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Priority extraction failed: {str(e)}"
        )


@app.post("/chat/with-history", response_model=ChatResponse)
async def chat_with_history(
    request: ChatRequest,
    background_tasks: BackgroundTasks,
    db: Session = Depends(get_db),
    conv_manager: ConversationManager = Depends(get_conversation_manager),
    memory_manager: MemoryManager = Depends(get_memory_manager),
    enhanced_memory: EnhancedMemoryManager = Depends(get_enhanced_memory_manager)
):
    """Chat with full historical context from past conversations"""
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

        # Build conversation context WITH historical context enabled
        context = await conv_manager.build_conversation_context(
            conversation.id,
            request.user_id,
            max_messages=settings.max_conversation_history,
            include_historical_context=True
        )

        # Get LLM response
        llm_response = await lmstudio_client.chat_completion(
            messages=context,
            temperature=request.temperature,
            max_tokens=request.max_tokens,
            stream=False
        )

        # Extract response content
        response_content = llm_response["choices"][0]["message"]["content"]

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
                "token_count": llm_response.get("usage", {}).get("total_tokens"),
                "historical_context": True
            }
        )

        # Extract and store enhanced memories (background task)
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

        # Create conversation summary more frequently (every 3 messages after 5)
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
            token_count=llm_response.get("usage", {}).get("total_tokens")
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


# Chat endpoints
@app.post("/chat", response_model=ChatResponse)
async def chat(
    request: ChatRequest,
    background_tasks: BackgroundTasks,
    db: Session = Depends(get_db),
    conv_manager: ConversationManager = Depends(get_conversation_manager),
    memory_manager: MemoryManager = Depends(get_memory_manager),
    enhanced_memory: EnhancedMemoryManager = Depends(get_enhanced_memory_manager)
):
    """Send a chat message and get response"""
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

        # Build conversation context WITH historical context enabled by default
        context = await conv_manager.build_conversation_context(
            conversation.id,
            request.user_id,
            max_messages=settings.max_conversation_history,
            include_historical_context=True  # Always enabled now
        )

        # Get LLM response
        llm_response = await lmstudio_client.chat_completion(
            messages=context,
            temperature=request.temperature,
            max_tokens=request.max_tokens,
            stream=False
        )

        # Extract response content
        response_content = llm_response["choices"][0]["message"]["content"]

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
                "token_count": llm_response.get("usage", {}).get("total_tokens")
            }
        )

        # Extract and store enhanced memories (background task)
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

        # Create conversation summary more frequently (every 3 messages after 5)
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
            token_count=llm_response.get("usage", {}).get("total_tokens")
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
async def chat_stream(
    request: ChatRequest,
    background_tasks: BackgroundTasks,
    db: Session = Depends(get_db),
    conv_manager: ConversationManager = Depends(get_conversation_manager),
    memory_manager: MemoryManager = Depends(get_memory_manager),
    enhanced_memory: EnhancedMemoryManager = Depends(get_enhanced_memory_manager)
):
    """Stream chat response"""
    if not request.stream:
        request.stream = True

    async def generate():
        start_time = time.time()
        full_response = ""

        try:
            # Verify user and get/create conversation (same as above)
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

            # Build context WITH historical context enabled
            context = await conv_manager.build_conversation_context(
                conversation.id,
                request.user_id,
                max_messages=settings.max_conversation_history,
                include_historical_context=True  # Always enabled now
            )

            # Stream LLM response
            async for chunk in lmstudio_client.chat_completion(
                messages=context,
                temperature=request.temperature,
                max_tokens=request.max_tokens,
                stream=True
            ):
                if "choices" in chunk and len(chunk["choices"]) > 0:
                    delta = chunk["choices"][0].get("delta", {})
                    if "content" in delta:
                        content = delta["content"]
                        full_response += content

                        response_data = {
                            "chunk": content,
                            "conversation_id": conversation.id,
                            "finished": False
                        }
                        yield f"data: {json.dumps(response_data)}\\n\\n"

            # Send completion signal
            processing_time = time.time() - start_time
            completion_data = {
                "chunk": "",
                "conversation_id": conversation.id,
                "finished": True,
                "processing_time": processing_time
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
                    "processing_time": processing_time
                }
            )

            # Background tasks for enhanced memory extraction
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


# Memory management endpoints
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


@app.delete("/users/{user_id}/memory/{memory_id}")
def delete_user_memory(
    user_id: int,
    memory_id: int,
    db: Session = Depends(get_db)
):
    """Delete a user memory entry"""
    memory = db.query(UserMemory).filter(
        UserMemory.id == memory_id,
        UserMemory.user_id == user_id
    ).first()

    if not memory:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Memory entry not found"
        )

    db.delete(memory)
    db.commit()
    return {"message": "Memory deleted successfully"}


@app.put("/users/{user_id}/memory/{memory_id}", response_model=UserMemorySchema)
def update_user_memory(
    user_id: int,
    memory_id: int,
    memory_update: UserMemoryCreate,
    db: Session = Depends(get_db)
):
    """Update a user memory entry"""
    memory = db.query(UserMemory).filter(
        UserMemory.id == memory_id,
        UserMemory.user_id == user_id
    ).first()

    if not memory:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Memory entry not found"
        )

    # Update memory fields
    memory.memory_type = memory_update.memory_type
    memory.key = memory_update.key
    memory.value = memory_update.value
    memory.confidence = memory_update.confidence
    memory.source = memory_update.source
    memory.updated_at = func.now()

    db.commit()
    db.refresh(memory)
    return memory


# Enhanced memory search endpoint
@app.get("/users/{user_id}/memory/search")
async def search_user_memories(
    user_id: int,
    q: str,
    limit: int = 10,
    db: Session = Depends(get_db),
    enhanced_memory: EnhancedMemoryManager = Depends(get_enhanced_memory_manager)
):
    """Search user memories semantically"""
    memories = await enhanced_memory.search_memories_semantic(user_id, q, limit)
    
    return {
        "query": q,
        "results": [
            {
                "id": m.id,
                "key": m.key,
                "value": m.value,
                "confidence": m.confidence,
                "memory_type": m.memory_type,
                "source": m.source,
                "created_at": m.created_at,
                "access_count": m.access_count
            }
            for m in memories
        ],
        "total_found": len(memories)
    }


# System status endpoint
@app.get("/status", response_model=SystemStatus)
async def get_system_status(db: Session = Depends(get_db)):
    """Get system status"""
    lmstudio_connected = await lmstudio_client.health_check()

    # Get database stats
    total_users = db.query(func.count(User.id)).scalar()
    active_conversations = db.query(func.count(Conversation.id)).filter(
        Conversation.is_active == True
    ).scalar()

    return SystemStatus(
        status="healthy",
        version=settings.api_version,
        lmstudio_connected=lmstudio_connected,
        database_connected=True,
        active_conversations=active_conversations,
        total_users=total_users
    )


# Background task functions
async def extract_and_store_memories(
    user_id: int,
    user_message: str,
    assistant_response: str,
    memory_manager: MemoryManager
):
    """Background task to extract and store implicit memories"""
    try:
        memories = memory_manager.extract_implicit_memory(user_id, user_message, assistant_response)
        if memories:
            memory_manager.store_memories(memories)
            logger.info(f"Stored {len(memories)} implicit memories for user {user_id}")
    except Exception as e:
        logger.error(f"Failed to extract memories: {e}")


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


async def auto_generate_title(conversation_id: int, conv_manager: ConversationManager):
    """Background task to auto-generate conversation title"""
    try:
        title = await conv_manager.generate_conversation_title(conversation_id)
        if title:
            # Get conversation to find owner
            conversation = conv_manager.db.query(Conversation).filter(
                Conversation.id == conversation_id
            ).first()
            if conversation:
                conv_manager.update_conversation_title(conversation_id, conversation.user_id, title)
                logger.info(f"Auto-generated title for conversation {conversation_id}: {title}")
    except Exception as e:
        logger.error(f"Failed to generate conversation title: {e}")


async def create_conversation_summary_task(conversation_id: int, conv_manager: ConversationManager):
    """Background task to create conversation summary"""
    try:
        summary = await conv_manager.create_conversation_summary(conversation_id)
        if summary:
            logger.info(f"Created summary for conversation {conversation_id}")
    except Exception as e:
        logger.error(f"Failed to create conversation summary: {e}")


# Health check endpoint
@app.get("/health")
def health_check():
    """Simple health check endpoint"""
    return {"status": "healthy", "timestamp": time.time()}


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(
        "main:app",
        host=settings.api_host,
        port=settings.api_port,
        reload=True
    )
