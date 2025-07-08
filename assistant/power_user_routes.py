"""
Power User API Routes for Enhanced User Data Management
"""
import logging
from datetime import datetime, timedelta
from typing import List, Optional, Dict, Any
from fastapi import APIRouter, Depends, HTTPException, status, Query
from fastapi.responses import StreamingResponse
from sqlalchemy.orm import Session
from sqlalchemy import func, desc, and_, or_, text
import json
import csv
import io

from database import get_db
from models import User, Conversation, Message, UserMemory, UserPreference, ConversationSummary
from schemas import User as UserSchema, UserCreate, UserMemory as UserMemorySchema, UserPreference as UserPreferenceSchema

logger = logging.getLogger(__name__)

# Create router
power_user_router = APIRouter(prefix="/api/power-user", tags=["power-user"])


@power_user_router.get("/users", response_model=List[UserSchema])
def get_all_users(skip: int = 0, limit: int = 100, db: Session = Depends(get_db)):
    """Get all users with pagination"""
    users = db.query(User).offset(skip).limit(limit).all()
    return users


@power_user_router.delete("/users/{user_id}")
def delete_user(user_id: int, db: Session = Depends(get_db)):
    """Delete a user and all associated data"""
    user = db.query(User).filter(User.id == user_id).first()
    if not user:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="User not found"
        )
    
    # Delete user (cascades to conversations, messages, memories, preferences)
    db.delete(user)
    db.commit()
    
    logger.info(f"Deleted user {user_id} and all associated data")
    return {"success": True, "message": "User deleted successfully"}


@power_user_router.put("/users/{user_id}", response_model=UserSchema)
def update_user(user_id: int, user_data: dict, db: Session = Depends(get_db)):
    """Update user information"""
    user = db.query(User).filter(User.id == user_id).first()
    if not user:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="User not found"
        )
    
    # Update allowed fields
    allowed_fields = ['username', 'email', 'full_name']
    for field in allowed_fields:
        if field in user_data:
            setattr(user, field, user_data[field])
    
    user.updated_at = datetime.utcnow()
    db.commit()
    db.refresh(user)
    
    return user


@power_user_router.get("/users/{user_id}/data")
def get_comprehensive_user_data(user_id: int, db: Session = Depends(get_db)):
    """Get comprehensive user data including analytics"""
    user = db.query(User).filter(User.id == user_id).first()
    if not user:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="User not found"
        )
    
    # Get user's data
    memories = db.query(UserMemory).filter(UserMemory.user_id == user_id).all()
    conversations = db.query(Conversation).filter(Conversation.user_id == user_id).all()
    preferences = db.query(UserPreference).filter(UserPreference.user_id == user_id).all()
    
    # Calculate analytics
    total_messages = db.query(func.count(Message.id)).join(Conversation).filter(
        Conversation.user_id == user_id
    ).scalar() or 0
    
    # Average response time from messages
    avg_response_time = db.query(func.avg(Message.processing_time)).join(Conversation).filter(
        Conversation.user_id == user_id,
        Message.role == 'assistant',
        Message.processing_time.isnot(None)
    ).scalar() or 0
    
    # Most active hour (simplified)
    most_active_hour = 10  # Default placeholder
    
    # Topics discussed (from conversation summaries)
    topics_query = db.query(ConversationSummary.keywords).join(Conversation).filter(
        Conversation.user_id == user_id,
        ConversationSummary.keywords.isnot(None)
    ).all()
    
    topics_discussed = []
    for topic_row in topics_query:
        if topic_row.keywords:
            topics_discussed.extend(topic_row.keywords.split(','))
    
    # Get unique topics and limit to top 10
    unique_topics = list(set([t.strip() for t in topics_discussed if t.strip()]))[:10]
    
    # Memory utilization (simple metric)
    memory_utilization = min(len(memories) / 100.0, 1.0)  # Assume 100 memories is full utilization
    
    # Conversation engagement (active conversations / total conversations)
    active_conversations = db.query(func.count(Conversation.id)).filter(
        Conversation.user_id == user_id,
        Conversation.is_active == True
    ).scalar() or 0
    
    total_conversations = len(conversations)
    conversation_engagement = active_conversations / max(total_conversations, 1)
    
    analytics = {
        "totalMessages": total_messages,
        "averageResponseTime": round(avg_response_time, 2),
        "mostActiveHour": most_active_hour,
        "topicsDiscussed": unique_topics,
        "memoryUtilization": round(memory_utilization, 2),
        "conversationEngagement": round(conversation_engagement, 2),
        "toolUsage": {
            "totalToolCalls": 0,  # Placeholder
            "mostUsedTools": {},
            "successRate": 0.0
        },
        "temporalPatterns": {
            "hourlyActivity": [0] * 24,
            "dailyActivity": [0] * 7,
            "weeklyActivity": [0] * 52
        }
    }
    
    return {
        "user": user,
        "memories": memories,
        "conversations": conversations,
        "preferences": preferences,
        "analytics": analytics
    }


@power_user_router.get("/users/{user_id}/preferences", response_model=List[UserPreferenceSchema])
def get_user_preferences(user_id: int, db: Session = Depends(get_db)):
    """Get user preferences"""
    user = db.query(User).filter(User.id == user_id).first()
    if not user:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="User not found"
        )
    
    preferences = db.query(UserPreference).filter(UserPreference.user_id == user_id).all()
    return preferences


@power_user_router.post("/users/{user_id}/preferences", response_model=UserPreferenceSchema)
def add_user_preference(user_id: int, preference_data: dict, db: Session = Depends(get_db)):
    """Add a new user preference"""
    user = db.query(User).filter(User.id == user_id).first()
    if not user:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="User not found"
        )
    
    preference = UserPreference(
        user_id=user_id,
        category=preference_data.get('category', 'general'),
        key=preference_data.get('key'),
        value=preference_data.get('value')
    )
    
    db.add(preference)
    db.commit()
    db.refresh(preference)
    
    return preference


@power_user_router.put("/users/{user_id}/preferences/{preference_id}", response_model=UserPreferenceSchema)
def update_user_preference(user_id: int, preference_id: int, preference_data: dict, db: Session = Depends(get_db)):
    """Update a user preference"""
    preference = db.query(UserPreference).filter(
        UserPreference.id == preference_id,
        UserPreference.user_id == user_id
    ).first()
    
    if not preference:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Preference not found"
        )
    
    # Update allowed fields
    allowed_fields = ['category', 'key', 'value']
    for field in allowed_fields:
        if field in preference_data:
            setattr(preference, field, preference_data[field])
    
    preference.updated_at = datetime.utcnow()
    db.commit()
    db.refresh(preference)
    
    return preference


@power_user_router.delete("/users/{user_id}/preferences/{preference_id}")
def delete_user_preference(user_id: int, preference_id: int, db: Session = Depends(get_db)):
    """Delete a user preference"""
    preference = db.query(UserPreference).filter(
        UserPreference.id == preference_id,
        UserPreference.user_id == user_id
    ).first()
    
    if not preference:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Preference not found"
        )
    
    db.delete(preference)
    db.commit()
    
    return {"success": True, "message": "Preference deleted successfully"}


@power_user_router.get("/users/{user_id}/analytics")
def get_user_analytics(user_id: int, db: Session = Depends(get_db)):
    """Get detailed user analytics"""
    user = db.query(User).filter(User.id == user_id).first()
    if not user:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="User not found"
        )
    
    # Get conversation metrics
    conversations = db.query(Conversation).filter(Conversation.user_id == user_id).all()
    total_conversations = len(conversations)
    active_conversations = sum(1 for c in conversations if c.is_active)
    
    # Get message metrics
    total_messages = db.query(func.count(Message.id)).join(Conversation).filter(
        Conversation.user_id == user_id
    ).scalar() or 0
    
    # Average messages per conversation
    avg_messages_per_conv = total_messages / max(total_conversations, 1)
    
    # Response time analytics
    avg_response_time = db.query(func.avg(Message.processing_time)).join(Conversation).filter(
        Conversation.user_id == user_id,
        Message.role == 'assistant',
        Message.processing_time.isnot(None)
    ).scalar() or 0
    
    # Memory analytics
    memories = db.query(UserMemory).filter(UserMemory.user_id == user_id).all()
    memory_types = {}
    for memory in memories:
        memory_types[memory.memory_type] = memory_types.get(memory.memory_type, 0) + 1
    
    # Temporal patterns (simplified)
    hourly_activity = [0] * 24
    daily_activity = [0] * 7
    weekly_activity = [0] * 52
    
    # Get message timestamps and calculate patterns
    messages_with_timestamps = db.query(Message.timestamp).join(Conversation).filter(
        Conversation.user_id == user_id
    ).all()
    
    for msg_timestamp in messages_with_timestamps:
        if msg_timestamp.timestamp:
            dt = msg_timestamp.timestamp
            hourly_activity[dt.hour] += 1
            daily_activity[dt.weekday()] += 1
            week_of_year = dt.isocalendar()[1] - 1
            if 0 <= week_of_year < 52:
                weekly_activity[week_of_year] += 1
    
    return {
        "totalMessages": total_messages,
        "averageResponseTime": round(avg_response_time, 2),
        "mostActiveHour": hourly_activity.index(max(hourly_activity)) if hourly_activity else 0,
        "topicsDiscussed": [],  # Placeholder
        "memoryUtilization": len(memories) / 100.0,  # Assume 100 is full utilization
        "conversationEngagement": active_conversations / max(total_conversations, 1),
        "toolUsage": {
            "totalToolCalls": 0,
            "mostUsedTools": {},
            "successRate": 0.0
        },
        "temporalPatterns": {
            "hourlyActivity": hourly_activity,
            "dailyActivity": daily_activity,
            "weeklyActivity": weekly_activity
        },
        "conversationMetrics": {
            "totalConversations": total_conversations,
            "activeConversations": active_conversations,
            "averageMessagesPerConversation": round(avg_messages_per_conv, 2)
        },
        "memoryMetrics": {
            "totalMemories": len(memories),
            "memoryTypes": memory_types,
            "averageConfidence": sum(m.confidence for m in memories) / max(len(memories), 1)
        }
    }


@power_user_router.get("/users/{user_id}/conversations/summaries")
def get_conversation_summaries(user_id: int, db: Session = Depends(get_db)):
    """Get conversation summaries for a user"""
    user = db.query(User).filter(User.id == user_id).first()
    if not user:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="User not found"
        )
    
    summaries = db.query(ConversationSummary).join(Conversation).filter(
        Conversation.user_id == user_id
    ).all()
    
    result = []
    for summary in summaries:
        result.append({
            "conversation_id": summary.conversation_id,
            "title": summary.conversation.title,
            "summary": summary.summary,
            "keywords": summary.keywords.split(',') if summary.keywords else [],
            "priority_score": summary.priority_score,
            "created_at": summary.created_at.isoformat(),
            "updated_at": summary.updated_at.isoformat()
        })
    
    return result


@power_user_router.get("/users/{user_id}/export")
def export_user_data(user_id: int, format: str = Query("json", regex="^(json|csv)$"), db: Session = Depends(get_db)):
    """Export comprehensive user data"""
    user = db.query(User).filter(User.id == user_id).first()
    if not user:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="User not found"
        )
    
    # Get all user data
    memories = db.query(UserMemory).filter(UserMemory.user_id == user_id).all()
    conversations = db.query(Conversation).filter(Conversation.user_id == user_id).all()
    preferences = db.query(UserPreference).filter(UserPreference.user_id == user_id).all()
    
    # Get all messages for user's conversations
    messages = db.query(Message).join(Conversation).filter(
        Conversation.user_id == user_id
    ).all()
    
    export_data = {
        "user": {
            "id": user.id,
            "username": user.username,
            "email": user.email,
            "full_name": user.full_name,
            "created_at": user.created_at.isoformat(),
            "updated_at": user.updated_at.isoformat()
        },
        "memories": [
            {
                "id": m.id,
                "memory_type": m.memory_type,
                "key": m.key,
                "value": m.value,
                "confidence": m.confidence,
                "source": m.source,
                "created_at": m.created_at.isoformat(),
                "updated_at": m.updated_at.isoformat(),
                "last_accessed": m.last_accessed.isoformat(),
                "access_count": m.access_count
            } for m in memories
        ],
        "conversations": [
            {
                "id": c.id,
                "title": c.title,
                "created_at": c.created_at.isoformat(),
                "updated_at": c.updated_at.isoformat(),
                "is_active": c.is_active,
                "message_count": len([msg for msg in messages if msg.conversation_id == c.id])
            } for c in conversations
        ],
        "preferences": [
            {
                "id": p.id,
                "category": p.category,
                "key": p.key,
                "value": p.value,
                "created_at": p.created_at.isoformat(),
                "updated_at": p.updated_at.isoformat()
            } for p in preferences
        ],
        "messages": [
            {
                "id": m.id,
                "conversation_id": m.conversation_id,
                "role": m.role,
                "content": m.content,
                "timestamp": m.timestamp.isoformat(),
                "processing_time": m.processing_time,
                "token_count": m.token_count,
                "llm_model": m.llm_model,
                "temperature": m.temperature
            } for m in messages
        ],
        "export_timestamp": datetime.utcnow().isoformat()
    }
    
    if format == "json":
        # Create JSON export
        json_data = json.dumps(export_data, indent=2)
        
        def generate():
            yield json_data
        
        return StreamingResponse(
            generate(),
            media_type="application/json",
            headers={"Content-Disposition": f"attachment; filename=user_{user_id}_data.json"}
        )
    
    elif format == "csv":
        # Create CSV export (simplified)
        output = io.StringIO()
        
        # Write conversations to CSV
        writer = csv.writer(output)
        writer.writerow(["Type", "ID", "Title/Key", "Content/Value", "Created", "Updated"])
        
        for conv in conversations:
            writer.writerow([
                "Conversation",
                conv.id,
                conv.title,
                f"Active: {conv.is_active}",
                conv.created_at.isoformat(),
                conv.updated_at.isoformat()
            ])
        
        for memory in memories:
            writer.writerow([
                f"Memory ({memory.memory_type})",
                memory.id,
                memory.key,
                memory.value,
                memory.created_at.isoformat(),
                memory.updated_at.isoformat()
            ])
        
        for pref in preferences:
            writer.writerow([
                f"Preference ({pref.category})",
                pref.id,
                pref.key,
                str(pref.value),
                pref.created_at.isoformat(),
                pref.updated_at.isoformat()
            ])
        
        csv_data = output.getvalue()
        output.close()
        
        def generate():
            yield csv_data
        
        return StreamingResponse(
            generate(),
            media_type="text/csv",
            headers={"Content-Disposition": f"attachment; filename=user_{user_id}_data.csv"}
        )


@power_user_router.post("/users/{user_id}/switch")
def switch_active_user(user_id: int, db: Session = Depends(get_db)):
    """Switch to active user (for session management)"""
    user = db.query(User).filter(User.id == user_id).first()
    if not user:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="User not found"
        )
    
    # In a real implementation, this would update session state
    # For now, we'll just return success
    return {"success": True, "message": f"Switched to user {user.username}"}


@power_user_router.get("/users/{user_id}/search")
def search_user_data(
    user_id: int, 
    q: str = Query(..., description="Search query"),
    type: Optional[str] = Query(None, regex="^(memories|conversations|preferences|all)$"),
    db: Session = Depends(get_db)
):
    """Search through user's data"""
    user = db.query(User).filter(User.id == user_id).first()
    if not user:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="User not found"
        )
    
    results = {"memories": [], "conversations": [], "preferences": [], "total_results": 0}
    
    if type in [None, "all", "memories"]:
        # Search memories
        memory_results = db.query(UserMemory).filter(
            UserMemory.user_id == user_id,
            or_(
                UserMemory.key.ilike(f"%{q}%"),
                UserMemory.value.ilike(f"%{q}%")
            )
        ).all()
        results["memories"] = memory_results
    
    if type in [None, "all", "conversations"]:
        # Search conversations
        conversation_results = db.query(Conversation).filter(
            Conversation.user_id == user_id,
            Conversation.title.ilike(f"%{q}%")
        ).all()
        results["conversations"] = conversation_results
    
    if type in [None, "all", "preferences"]:
        # Search preferences
        preference_results = db.query(UserPreference).filter(
            UserPreference.user_id == user_id,
            or_(
                UserPreference.key.ilike(f"%{q}%"),
                UserPreference.category.ilike(f"%{q}%")
            )
        ).all()
        results["preferences"] = preference_results
    
    results["total_results"] = len(results["memories"]) + len(results["conversations"]) + len(results["preferences"])
    
    return results


@power_user_router.get("/system/overview")
def get_system_overview(db: Session = Depends(get_db)):
    """Get system-wide overview for power users"""
    total_users = db.query(func.count(User.id)).scalar()
    total_conversations = db.query(func.count(Conversation.id)).scalar()
    total_memories = db.query(func.count(UserMemory.id)).scalar()
    
    # Active users in last 7 days (simplified)
    week_ago = datetime.utcnow() - timedelta(days=7)
    active_users_week = db.query(func.count(func.distinct(Conversation.user_id))).filter(
        Conversation.updated_at >= week_ago
    ).scalar()
    
    # Active users in last 30 days
    month_ago = datetime.utcnow() - timedelta(days=30)
    active_users_month = db.query(func.count(func.distinct(Conversation.user_id))).filter(
        Conversation.updated_at >= month_ago
    ).scalar()
    
    return {
        "total_users": total_users,
        "total_conversations": total_conversations,
        "total_memories": total_memories,
        "system_health": {
            "database_performance": 0.95,  # Placeholder
            "memory_usage": 0.65,  # Placeholder
            "response_time_avg": 2.3  # Placeholder
        },
        "user_engagement_metrics": {
            "daily_active_users": active_users_week // 7,  # Rough estimate
            "weekly_active_users": active_users_week,
            "monthly_active_users": active_users_month
        },
        "trending_topics": []  # Placeholder
    }
