"""
Debug Persistence Manager for storing and retrieving debug data
"""
import uuid
import json
import logging
from typing import Dict, List, Optional, Any, Tuple
from datetime import datetime
from sqlalchemy.orm import Session
from sqlalchemy import desc, func

from models import (
    DebugSession, DebugStep, LLMRequest, Message, Conversation, 
    User, UserPreference
)

logger = logging.getLogger(__name__)


class DebugPersistenceManager:
    """Manages persistent storage and retrieval of debug data"""

    def __init__(self, db: Session):
        self.db = db

    def create_debug_session(self, conversation_id: int, user_id: int) -> DebugSession:
        """Create a new debug session"""
        session_id = str(uuid.uuid4())

        debug_session = DebugSession(
            conversation_id=conversation_id,
            user_id=user_id,
            session_id=session_id,
            started_at=datetime.now(),
            is_active=True
        )

        self.db.add(debug_session)
        self.db.commit()
        self.db.refresh(debug_session)

        logger.info(f"Created debug session {session_id} for conversation {conversation_id}")
        return debug_session

    def get_active_debug_session(self, conversation_id: int, user_id: int) -> Optional[DebugSession]:
        """Get the current active debug session for a conversation"""
        return self.db.query(DebugSession).filter(
            DebugSession.conversation_id == conversation_id,
            DebugSession.user_id == user_id,
            DebugSession.is_active == True
        ).first()

    def get_or_create_debug_session(self, conversation_id: int, user_id: int) -> DebugSession:
        """Get existing active debug session or create a new one"""
        debug_session = self.get_active_debug_session(conversation_id, user_id)

        if not debug_session:
            debug_session = self.create_debug_session(conversation_id, user_id)

        return debug_session

    def end_debug_session(self, session_id: str) -> bool:
        """End a debug session"""
        debug_session = self.db.query(DebugSession).filter(
            DebugSession.session_id == session_id
        ).first()

        if debug_session:
            debug_session.ended_at = datetime.now()
            debug_session.is_active = False
            self.db.commit()
            logger.info(f"Ended debug session {session_id}")
            return True

        return False

    def store_debug_step(
        self,
        message_id: int,
        debug_session_id: int,
        step_type: str,
        step_order: int,
        title: str,
        description: Optional[str] = None,
        duration_ms: Optional[int] = None,
        success: bool = True,
        error_message: Optional[str] = None,
        input_data: Optional[Dict[str, Any]] = None,
        output_data: Optional[Dict[str, Any]] = None,
        step_metadata: Optional[Dict[str, Any]] = None
    ) -> DebugStep:
        """Store a debug step"""
        step_id = str(uuid.uuid4())

        debug_step = DebugStep(
            message_id=message_id,
            debug_session_id=debug_session_id,
            step_id=step_id,
            step_type=step_type,
            step_order=step_order,
            title=title,
            description=description,
            timestamp=datetime.now(),
            duration_ms=duration_ms,
            success=success,
            error_message=error_message,
            input_data=input_data,
            output_data=output_data,
            step_metadata=step_metadata
        )

        self.db.add(debug_step)
        self.db.commit()
        self.db.refresh(debug_step)

        # Update debug session stats
        self.update_debug_session_stats(debug_session_id)

        return debug_step

    def store_llm_request(
        self,
        message_id: int,
        model: str,
        request_messages: List[Dict[str, Any]],
        response_data: Dict[str, Any],
        temperature: Optional[float] = None,
        max_tokens: Optional[int] = None,
        stream: bool = False,
        processing_time_ms: Optional[int] = None,
        token_usage: Optional[Dict[str, Any]] = None,
        tools_available: Optional[List[Dict[str, Any]]] = None,
        tools_used: Optional[List[str]] = None,
        tool_calls: Optional[List[Dict[str, Any]]] = None,
        tool_results: Optional[List[Dict[str, Any]]] = None
    ) -> LLMRequest:
        """Store an LLM request/response"""
        request_id = str(uuid.uuid4())

        llm_request = LLMRequest(
            message_id=message_id,
            request_id=request_id,
            model=model,
            temperature=temperature,
            max_tokens=max_tokens,
            stream=stream,
            request_messages=request_messages,
            response_data=response_data,
            timestamp=datetime.now(),
            processing_time_ms=processing_time_ms,
            token_usage=token_usage,
            tools_available=tools_available,
            tools_used=tools_used,
            tool_calls=tool_calls,
            tool_results=tool_results
        )

        self.db.add(llm_request)
        self.db.commit()
        self.db.refresh(llm_request)

        return llm_request

    def update_debug_session_stats(self, debug_session_id: int):
        """Update debug session statistics"""
        debug_session = self.db.query(DebugSession).filter(
            DebugSession.id == debug_session_id
        ).first()

        if debug_session:
            # Count messages in this session
            message_count = self.db.query(Message).filter(
                Message.conversation_id == debug_session.conversation_id,
                Message.debug_enabled == True
            ).count()

            # Count steps in this session
            step_count = self.db.query(DebugStep).filter(
                DebugStep.debug_session_id == debug_session_id
            ).count()

            # Count tools used
            tool_count = self.db.query(DebugStep).filter(
                DebugStep.debug_session_id == debug_session_id,
                DebugStep.step_type.in_(['tool_call', 'tool_result'])
            ).count()

            # Calculate total processing time
            total_processing_time = self.db.query(
                func.sum(DebugStep.duration_ms)
            ).filter(
                DebugStep.debug_session_id == debug_session_id
            ).scalar() or 0

            debug_session.total_messages = message_count
            debug_session.total_steps = step_count
            debug_session.total_tools_used = tool_count
            debug_session.total_processing_time = total_processing_time / 1000.0  # Convert to seconds

            self.db.commit()

    def get_conversation_debug_data(self, conversation_id: int, user_id: int) -> Dict[str, Any]:
        """Get all debug data for a conversation"""
        # Verify conversation access
        conversation = self.db.query(Conversation).filter(
            Conversation.id == conversation_id,
            Conversation.user_id == user_id
        ).first()

        if not conversation:
            raise ValueError("Conversation not found or access denied")

        # Get debug sessions
        debug_sessions = self.db.query(DebugSession).filter(
            DebugSession.conversation_id == conversation_id,
            DebugSession.user_id == user_id
        ).order_by(desc(DebugSession.started_at)).all()

        # Get messages with debug data
        messages = self.db.query(Message).filter(
            Message.conversation_id == conversation_id,
            Message.debug_enabled == True
        ).order_by(Message.timestamp).all()

        debug_data = {
            "conversation_id": conversation_id,
            "conversation_title": conversation.title,
            "debug_sessions": [],
            "messages": []
        }

        # Process debug sessions
        for session in debug_sessions:
            session_data = {
                "session_id": session.session_id,
                "started_at": session.started_at.isoformat(),
                "ended_at": session.ended_at.isoformat() if session.ended_at else None,
                "is_active": session.is_active,
                "total_messages": session.total_messages,
                "total_steps": session.total_steps,
                "total_tools_used": session.total_tools_used,
                "total_processing_time": session.total_processing_time
            }
            debug_data["debug_sessions"].append(session_data)

        # Process messages with debug data
        for message in messages:
            message_data = {
                "message_id": message.id,
                "role": message.role,
                "content": message.content,
                "timestamp": message.timestamp.isoformat(),
                "processing_time": message.processing_time,
                "token_count": message.token_count,
                "debug_steps": [],
                "llm_requests": []
            }

            # Get debug steps for this message
            debug_steps = self.db.query(DebugStep).filter(
                DebugStep.message_id == message.id
            ).order_by(DebugStep.step_order).all()

            for step in debug_steps:
                step_data = {
                    "step_id": step.step_id,
                    "step_type": step.step_type,
                    "step_order": step.step_order,
                    "title": step.title,
                    "description": step.description,
                    "timestamp": step.timestamp.isoformat(),
                    "duration_ms": step.duration_ms,
                    "success": step.success,
                    "error_message": step.error_message,
                    "input_data": step.input_data,
                    "output_data": step.output_data,
                    "metadata": step.step_metadata
                }
                message_data["debug_steps"].append(step_data)

            # Get LLM requests for this message
            llm_requests = self.db.query(LLMRequest).filter(
                LLMRequest.message_id == message.id
            ).order_by(LLMRequest.timestamp).all()

            for request in llm_requests:
                request_data = {
                    "request_id": request.request_id,
                    "model": request.model,
                    "temperature": request.temperature,
                    "max_tokens": request.max_tokens,
                    "stream": request.stream,
                    "timestamp": request.timestamp.isoformat(),
                    "processing_time_ms": request.processing_time_ms,
                    "token_usage": request.token_usage,
                    "tools_available": request.tools_available,
                    "tools_used": request.tools_used,
                    "tool_calls": request.tool_calls,
                    "tool_results": request.tool_results,
                    "request_messages": request.request_messages,
                    "response_data": request.response_data
                }
                message_data["llm_requests"].append(request_data)

            debug_data["messages"].append(message_data)

        return debug_data

    def get_user_debug_preference(self, user_id: int) -> bool:
        """Get user's debug mode preference"""
        preference = self.db.query(UserPreference).filter(
            UserPreference.user_id == user_id,
            UserPreference.category == "debug_mode",
            UserPreference.key == "enabled"
        ).first()

        if preference:
            return preference.value.get("enabled", False)

        return False

    def set_user_debug_preference(self, user_id: int, enabled: bool):
        """Set user's debug mode preference"""
        preference = self.db.query(UserPreference).filter(
            UserPreference.user_id == user_id,
            UserPreference.category == "debug_mode",
            UserPreference.key == "enabled"
        ).first()

        if preference:
            preference.value = {"enabled": enabled}
            preference.updated_at = datetime.now()
        else:
            preference = UserPreference(
                user_id=user_id,
                category="debug_mode",
                key="enabled",
                value={"enabled": enabled}
            )
            self.db.add(preference)

        self.db.commit()
        logger.info(f"Set debug mode preference for user {user_id}: {enabled}")

    def get_debug_session_summary(self, conversation_id: int, user_id: int) -> Dict[str, Any]:
        """Get debug session summary for a conversation"""
        debug_sessions = self.db.query(DebugSession).filter(
            DebugSession.conversation_id == conversation_id,
            DebugSession.user_id == user_id
        ).all()

        if not debug_sessions:
            return {
                "conversation_id": conversation_id,
                "has_debug_data": False,
                "total_sessions": 0,
                "active_sessions": 0,
                "total_messages": 0,
                "total_steps": 0,
                "total_tools_used": 0,
                "total_processing_time": 0.0
            }

        active_sessions = [s for s in debug_sessions if s.is_active]
        total_messages = sum(s.total_messages for s in debug_sessions)
        total_steps = sum(s.total_steps for s in debug_sessions)
        total_tools_used = sum(s.total_tools_used for s in debug_sessions)
        total_processing_time = sum(s.total_processing_time for s in debug_sessions)

        return {
            "conversation_id": conversation_id,
            "has_debug_data": True,
            "total_sessions": len(debug_sessions),
            "active_sessions": len(active_sessions),
            "total_messages": total_messages,
            "total_steps": total_steps,
            "total_tools_used": total_tools_used,
            "total_processing_time": total_processing_time,
            "sessions": [
                {
                    "session_id": s.session_id,
                    "started_at": s.started_at.isoformat(),
                    "ended_at": s.ended_at.isoformat() if s.ended_at else None,
                    "is_active": s.is_active,
                    "total_messages": s.total_messages,
                    "total_steps": s.total_steps,
                    "total_tools_used": s.total_tools_used,
                    "total_processing_time": s.total_processing_time
                }
                for s in debug_sessions
            ]
        }

    def get_message_debug_steps(self, message_id: int) -> List[DebugStep]:
        """Get debug steps for a specific message"""
        debug_steps = self.db.query(DebugStep).filter(
            DebugStep.message_id == message_id
        ).order_by(DebugStep.step_order).all()

        return debug_steps

    def get_message_llm_requests(self, message_id: int) -> List[LLMRequest]:
        """Get LLM requests for a specific message"""
        llm_requests = self.db.query(LLMRequest).filter(
            LLMRequest.message_id == message_id
        ).all()

        return llm_requests

    def cleanup_old_debug_data(self, days_old: int = 30) -> int:
        """Clean up old debug data"""
        from datetime import timedelta

        cutoff_date = datetime.now() - timedelta(days=days_old)

        # Get old debug sessions
        old_sessions = self.db.query(DebugSession).filter(
            DebugSession.started_at < cutoff_date,
            DebugSession.is_active == False
        ).all()

        count = 0
        for session in old_sessions:
            # Delete related debug steps
            self.db.query(DebugStep).filter(
                DebugStep.debug_session_id == session.id
            ).delete()

            # Delete the session
            self.db.delete(session)
            count += 1

        # Clean up old LLM requests
        old_llm_requests = self.db.query(LLMRequest).filter(
            LLMRequest.timestamp < cutoff_date
        ).all()

        for request in old_llm_requests:
            self.db.delete(request)

        self.db.commit()
        logger.info(f"Cleaned up {count} old debug sessions and {len(old_llm_requests)} old LLM requests")

        return count + len(old_llm_requests)
