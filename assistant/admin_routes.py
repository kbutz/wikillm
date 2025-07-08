"""
Admin API routes for WikiLLM Assistant
Provides admin tools for user management, memory inspection, and system monitoring
"""
import logging
import json
from datetime import datetime, timedelta
from typing import List, Optional, Dict, Any
from fastapi import APIRouter, HTTPException, Depends, status, Response
from fastapi.responses import StreamingResponse
from sqlalchemy.orm import Session
from sqlalchemy import func, desc, and_
from pydantic import BaseModel
from io import StringIO

from database import get_db
from models import User, Conversation, Message, UserMemory, UserPreference, SystemLog
from schemas import User as UserSchema

logger = logging.getLogger(__name__)

# Create admin router
admin_router = APIRouter(prefix="/admin", tags=["admin"])

# Pydantic models for admin responses
class AdminUserResponse(BaseModel):
    id: int
    username: str
    email: Optional[str]
    full_name: Optional[str]
    created_at: datetime
    updated_at: datetime
    last_active: Optional[datetime]
    conversation_count: int
    memory_size: int
    memory_entries: int

class AdminConversationResponse(BaseModel):
    id: int
    title: str
    created_at: datetime
    updated_at: datetime
    message_count: int
    last_message: Optional[str]
    user_id: int
    username: str

class AdminMemoryResponse(BaseModel):
    personal_info: Dict[str, Any]
    conversation_history: List[Dict[str, Any]]
    context_memory: Dict[str, Any]
    preferences: Dict[str, Any]
    size: int
    last_updated: datetime

class AdminSystemStats(BaseModel):
    total_users: int
    active_users: int
    total_conversations: int
    total_messages: int
    total_memory_entries: int
    system_health: Dict[str, Any]

# User Management Endpoints
@admin_router.get("/users", response_model=List[AdminUserResponse])
def get_all_users(skip: int = 0, limit: int = 100, db: Session = Depends(get_db)):
    """Get all users with admin details"""
    try:
        users = db.query(User).offset(skip).limit(limit).all()
        
        admin_users = []
        for user in users:
            # Get conversation count
            conversation_count = db.query(func.count(Conversation.id)).filter(
                Conversation.user_id == user.id
            ).scalar()
            
            # Get memory stats
            memory_entries = db.query(func.count(UserMemory.id)).filter(
                UserMemory.user_id == user.id
            ).scalar()
            
            # Calculate memory size (rough estimate)
            memory_size = db.query(func.sum(func.length(UserMemory.value))).filter(
                UserMemory.user_id == user.id
            ).scalar() or 0
            
            # Get last activity (last message timestamp)
            last_message = db.query(Message).join(Conversation).filter(
                Conversation.user_id == user.id
            ).order_by(desc(Message.timestamp)).first()
            
            last_active = last_message.timestamp if last_message else user.created_at
            
            admin_users.append(AdminUserResponse(
                id=user.id,
                username=user.username,
                email=user.email,
                full_name=user.full_name,
                created_at=user.created_at,
                updated_at=user.updated_at,
                last_active=last_active,
                conversation_count=conversation_count,
                memory_size=memory_size,
                memory_entries=memory_entries
            ))
        
        return admin_users
    except Exception as e:
        logger.error(f"Failed to get users: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@admin_router.get("/users/{user_id}")
def get_user_details(user_id: int, db: Session = Depends(get_db)):
    """Get detailed user information"""
    try:
        user = db.query(User).filter(User.id == user_id).first()
        if not user:
            raise HTTPException(status_code=404, detail="User not found")
        
        # Get detailed stats
        conversations = db.query(Conversation).filter(Conversation.user_id == user_id).all()
        memories = db.query(UserMemory).filter(UserMemory.user_id == user_id).all()
        preferences = db.query(UserPreference).filter(UserPreference.user_id == user_id).all()
        
        return {
            "user": user,
            "conversations": conversations,
            "memories": memories,
            "preferences": preferences,
            "stats": {
                "conversation_count": len(conversations),
                "memory_count": len(memories),
                "preference_count": len(preferences)
            }
        }
    except Exception as e:
        logger.error(f"Failed to get user details: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@admin_router.post("/users", response_model=UserSchema)
def create_admin_user(user_data: Dict[str, Any], db: Session = Depends(get_db)):
    """Create a new user (admin function)"""
    try:
        # Check if username already exists
        existing_user = db.query(User).filter(User.username == user_data["username"]).first()
        if existing_user:
            raise HTTPException(status_code=400, detail="Username already exists")
        
        # Create new user
        new_user = User(
            username=user_data["username"],
            email=user_data.get("email"),
            full_name=user_data.get("full_name")
        )
        
        db.add(new_user)
        db.commit()
        db.refresh(new_user)
        
        logger.info(f"Admin created user: {new_user.username}")
        return new_user
    except Exception as e:
        db.rollback()
        logger.error(f"Failed to create user: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@admin_router.delete("/users/{user_id}")
def delete_user(user_id: int, db: Session = Depends(get_db)):
    """Delete a user and all associated data"""
    try:
        user = db.query(User).filter(User.id == user_id).first()
        if not user:
            raise HTTPException(status_code=404, detail="User not found")
        
        # Delete all associated data (handled by cascade)
        db.delete(user)
        db.commit()
        
        logger.info(f"Admin deleted user: {user.username}")
        return {"message": "User deleted successfully"}
    except Exception as e:
        db.rollback()
        logger.error(f"Failed to delete user: {e}")
        raise HTTPException(status_code=500, detail=str(e))

# Memory Management Endpoints
@admin_router.get("/users/{user_id}/memory", response_model=AdminMemoryResponse)
def get_user_memory(user_id: int, db: Session = Depends(get_db)):
    """Get comprehensive user memory data"""
    try:
        user = db.query(User).filter(User.id == user_id).first()
        if not user:
            raise HTTPException(status_code=404, detail="User not found")
        
        # Get all memory entries
        memories = db.query(UserMemory).filter(UserMemory.user_id == user_id).all()
        preferences = db.query(UserPreference).filter(UserPreference.user_id == user_id).all()
        conversations = db.query(Conversation).filter(Conversation.user_id == user_id).all()
        
        # Organize memory data
        personal_info = {}
        context_memory = {}
        conversation_history = []
        
        # Parse memory entries
        for memory in memories:
            if memory.memory_type == "explicit":
                personal_info[memory.key] = {
                    "value": memory.value,
                    "confidence": memory.confidence,
                    "source": memory.source,
                    "last_accessed": memory.last_accessed.isoformat(),
                    "access_count": memory.access_count
                }
            elif memory.memory_type == "implicit":
                context_memory[memory.key] = {
                    "value": memory.value,
                    "confidence": memory.confidence,
                    "created_at": memory.created_at.isoformat()
                }
        
        # Parse conversation history
        for conv in conversations:
            message_count = db.query(func.count(Message.id)).filter(
                Message.conversation_id == conv.id
            ).scalar()
            
            last_message = db.query(Message).filter(
                Message.conversation_id == conv.id
            ).order_by(desc(Message.timestamp)).first()
            
            conversation_history.append({
                "id": conv.id,
                "title": conv.title,
                "created_at": conv.created_at.isoformat(),
                "message_count": message_count,
                "last_message": last_message.content[:100] + "..." if last_message else None,
                "is_active": conv.is_active
            })
        
        # Parse preferences
        prefs = {}
        for pref in preferences:
            if pref.category not in prefs:
                prefs[pref.category] = {}
            prefs[pref.category][pref.key] = pref.value
        
        # Calculate total memory size
        total_size = sum(len(str(m.value)) for m in memories)
        
        return AdminMemoryResponse(
            personal_info=personal_info,
            conversation_history=conversation_history,
            context_memory=context_memory,
            preferences=prefs,
            size=total_size,
            last_updated=max([m.updated_at for m in memories] + [datetime.now()])
        )
    except Exception as e:
        logger.error(f"Failed to get user memory: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@admin_router.put("/users/{user_id}/memory")
def update_user_memory(user_id: int, memory_data: Dict[str, Any], db: Session = Depends(get_db)):
    """Update user memory data"""
    try:
        user = db.query(User).filter(User.id == user_id).first()
        if not user:
            raise HTTPException(status_code=404, detail="User not found")
        
        # Update personal info
        if "personal_info" in memory_data:
            for key, value in memory_data["personal_info"].items():
                existing_memory = db.query(UserMemory).filter(
                    and_(UserMemory.user_id == user_id, UserMemory.key == key)
                ).first()
                
                if existing_memory:
                    existing_memory.value = str(value)
                    existing_memory.updated_at = datetime.now()
                else:
                    new_memory = UserMemory(
                        user_id=user_id,
                        memory_type="explicit",
                        key=key,
                        value=str(value),
                        confidence=1.0,
                        source="admin_update"
                    )
                    db.add(new_memory)
        
        # Update context memory
        if "context_memory" in memory_data:
            for key, value in memory_data["context_memory"].items():
                existing_memory = db.query(UserMemory).filter(
                    and_(UserMemory.user_id == user_id, UserMemory.key == key)
                ).first()
                
                if existing_memory:
                    existing_memory.value = str(value)
                    existing_memory.updated_at = datetime.now()
                else:
                    new_memory = UserMemory(
                        user_id=user_id,
                        memory_type="implicit",
                        key=key,
                        value=str(value),
                        confidence=1.0,
                        source="admin_update"
                    )
                    db.add(new_memory)
        
        db.commit()
        logger.info(f"Admin updated memory for user {user_id}")
        return {"message": "Memory updated successfully"}
    except Exception as e:
        db.rollback()
        logger.error(f"Failed to update user memory: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@admin_router.delete("/users/{user_id}/memory")
def clear_user_memory(user_id: int, memory_type: Optional[str] = None, db: Session = Depends(get_db)):
    """Clear user memory (all or specific type)"""
    try:
        user = db.query(User).filter(User.id == user_id).first()
        if not user:
            raise HTTPException(status_code=404, detail="User not found")
        
        query = db.query(UserMemory).filter(UserMemory.user_id == user_id)
        
        if memory_type:
            query = query.filter(UserMemory.memory_type == memory_type)
        
        deleted_count = query.delete()
        db.commit()
        
        logger.info(f"Admin cleared {deleted_count} memory entries for user {user_id}")
        return {"message": f"Cleared {deleted_count} memory entries"}
    except Exception as e:
        db.rollback()
        logger.error(f"Failed to clear user memory: {e}")
        raise HTTPException(status_code=500, detail=str(e))

# Conversation Management Endpoints
@admin_router.get("/users/{user_id}/conversations", response_model=List[AdminConversationResponse])
def get_user_conversations(user_id: int, limit: int = 100, db: Session = Depends(get_db)):
    """Get user conversations with admin details"""
    try:
        user = db.query(User).filter(User.id == user_id).first()
        if not user:
            raise HTTPException(status_code=404, detail="User not found")
        
        conversations = db.query(Conversation).filter(
            Conversation.user_id == user_id
        ).order_by(desc(Conversation.updated_at)).limit(limit).all()
        
        admin_conversations = []
        for conv in conversations:
            message_count = db.query(func.count(Message.id)).filter(
                Message.conversation_id == conv.id
            ).scalar()
            
            last_message = db.query(Message).filter(
                Message.conversation_id == conv.id
            ).order_by(desc(Message.timestamp)).first()
            
            admin_conversations.append(AdminConversationResponse(
                id=conv.id,
                title=conv.title,
                created_at=conv.created_at,
                updated_at=conv.updated_at,
                message_count=message_count,
                last_message=last_message.content[:100] + "..." if last_message else None,
                user_id=conv.user_id,
                username=user.username
            ))
        
        return admin_conversations
    except Exception as e:
        logger.error(f"Failed to get user conversations: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@admin_router.get("/conversations/{conversation_id}/messages")
def get_conversation_messages(conversation_id: int, db: Session = Depends(get_db)):
    """Get all messages in a conversation"""
    try:
        conversation = db.query(Conversation).filter(Conversation.id == conversation_id).first()
        if not conversation:
            raise HTTPException(status_code=404, detail="Conversation not found")
        
        messages = db.query(Message).filter(
            Message.conversation_id == conversation_id
        ).order_by(Message.timestamp).all()
        
        return {
            "conversation": conversation,
            "messages": messages
        }
    except Exception as e:
        logger.error(f"Failed to get conversation messages: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@admin_router.delete("/conversations/{conversation_id}")
def delete_conversation(conversation_id: int, db: Session = Depends(get_db)):
    """Delete a conversation and all messages"""
    try:
        conversation = db.query(Conversation).filter(Conversation.id == conversation_id).first()
        if not conversation:
            raise HTTPException(status_code=404, detail="Conversation not found")
        
        # Delete all messages (handled by cascade)
        db.delete(conversation)
        db.commit()
        
        logger.info(f"Admin deleted conversation {conversation_id}")
        return {"message": "Conversation deleted successfully"}
    except Exception as e:
        db.rollback()
        logger.error(f"Failed to delete conversation: {e}")
        raise HTTPException(status_code=500, detail=str(e))

# Data Export Endpoints
@admin_router.get("/users/{user_id}/export")
def export_user_data(user_id: int, db: Session = Depends(get_db)):
    """Export all user data as JSON"""
    try:
        user = db.query(User).filter(User.id == user_id).first()
        if not user:
            raise HTTPException(status_code=404, detail="User not found")
        
        # Get all user data
        conversations = db.query(Conversation).filter(Conversation.user_id == user_id).all()
        memories = db.query(UserMemory).filter(UserMemory.user_id == user_id).all()
        preferences = db.query(UserPreference).filter(UserPreference.user_id == user_id).all()
        
        # Get all messages for user's conversations
        conv_messages = {}
        for conv in conversations:
            messages = db.query(Message).filter(Message.conversation_id == conv.id).all()
            conv_messages[conv.id] = [
                {
                    "id": msg.id,
                    "role": msg.role,
                    "content": msg.content,
                    "timestamp": msg.timestamp.isoformat(),
                    "token_count": msg.token_count,
                    "processing_time": msg.processing_time
                } for msg in messages
            ]
        
        export_data = {
            "user": {
                "id": user.id,
                "username": user.username,
                "email": user.email,
                "full_name": user.full_name,
                "created_at": user.created_at.isoformat(),
                "updated_at": user.updated_at.isoformat()
            },
            "conversations": [
                {
                    "id": conv.id,
                    "title": conv.title,
                    "created_at": conv.created_at.isoformat(),
                    "updated_at": conv.updated_at.isoformat(),
                    "is_active": conv.is_active,
                    "messages": conv_messages.get(conv.id, [])
                } for conv in conversations
            ],
            "memories": [
                {
                    "id": mem.id,
                    "memory_type": mem.memory_type,
                    "key": mem.key,
                    "value": mem.value,
                    "confidence": mem.confidence,
                    "source": mem.source,
                    "created_at": mem.created_at.isoformat(),
                    "updated_at": mem.updated_at.isoformat(),
                    "last_accessed": mem.last_accessed.isoformat(),
                    "access_count": mem.access_count
                } for mem in memories
            ],
            "preferences": [
                {
                    "id": pref.id,
                    "category": pref.category,
                    "key": pref.key,
                    "value": pref.value,
                    "created_at": pref.created_at.isoformat(),
                    "updated_at": pref.updated_at.isoformat()
                } for pref in preferences
            ],
            "exported_at": datetime.now().isoformat()
        }
        
        # Create JSON response
        json_str = json.dumps(export_data, indent=2)
        
        return StreamingResponse(
            StringIO(json_str),
            media_type="application/json",
            headers={"Content-Disposition": f"attachment; filename=user_{user_id}_export.json"}
        )
    except Exception as e:
        logger.error(f"Failed to export user data: {e}")
        raise HTTPException(status_code=500, detail=str(e))

# System Statistics and Monitoring
@admin_router.get("/system/stats", response_model=AdminSystemStats)
def get_system_stats(db: Session = Depends(get_db)):
    """Get comprehensive system statistics"""
    try:
        # Basic counts
        total_users = db.query(func.count(User.id)).scalar()
        total_conversations = db.query(func.count(Conversation.id)).scalar()
        total_messages = db.query(func.count(Message.id)).scalar()
        total_memory_entries = db.query(func.count(UserMemory.id)).scalar()
        
        # Active users (users with activity in last 7 days)
        active_cutoff = datetime.now() - timedelta(days=7)
        active_users = db.query(func.count(func.distinct(Conversation.user_id))).join(
            Message, Conversation.id == Message.conversation_id
        ).filter(Message.timestamp >= active_cutoff).scalar()
        
        # System health metrics
        recent_errors = db.query(func.count(SystemLog.id)).filter(
            and_(
                SystemLog.level == "ERROR",
                SystemLog.timestamp >= datetime.now() - timedelta(hours=24)
            )
        ).scalar()
        
        system_health = {
            "status": "healthy" if recent_errors < 10 else "degraded",
            "recent_errors": recent_errors,
            "database_size": "N/A",  # Could be calculated if needed
            "uptime": "N/A"  # Could be tracked if needed
        }
        
        return AdminSystemStats(
            total_users=total_users,
            active_users=active_users,
            total_conversations=total_conversations,
            total_messages=total_messages,
            total_memory_entries=total_memory_entries,
            system_health=system_health
        )
    except Exception as e:
        logger.error(f"Failed to get system stats: {e}")
        raise HTTPException(status_code=500, detail=str(e))

# User Impersonation (for testing/debugging)
@admin_router.post("/users/{user_id}/impersonate")
def impersonate_user(user_id: int, db: Session = Depends(get_db)):
    """Create impersonation session for user (for debugging)"""
    try:
        user = db.query(User).filter(User.id == user_id).first()
        if not user:
            raise HTTPException(status_code=404, detail="User not found")
        
        # Update last active
        user.updated_at = datetime.now()
        db.commit()
        
        logger.info(f"Admin impersonating user: {user.username}")
        return {
            "message": "Impersonation session created",
            "user": {
                "id": user.id,
                "username": user.username,
                "email": user.email
            }
        }
    except Exception as e:
        logger.error(f"Failed to create impersonation session: {e}")
        raise HTTPException(status_code=500, detail=str(e))

# Health check for admin system
@admin_router.get("/health")
def admin_health_check():
    """Admin system health check"""
    return {
        "status": "healthy",
        "timestamp": datetime.now().isoformat(),
        "admin_system": "operational"
    }
