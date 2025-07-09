"""
Enhanced Database Models with Debug Data Persistence
"""
from sqlalchemy import Column, Integer, String, DateTime, Text, Float, Boolean, ForeignKey, JSON
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy.orm import relationship
from sqlalchemy.sql import func
from datetime import datetime
from typing import Optional, Dict, Any

Base = declarative_base()


class User(Base):
    """User model for storing user information and preferences"""
    __tablename__ = "users"

    id = Column(Integer, primary_key=True, index=True)
    username = Column(String(50), unique=True, index=True, nullable=False)
    email = Column(String(100), unique=True, index=True, nullable=True)
    full_name = Column(String(100), nullable=True)
    created_at = Column(DateTime, default=func.now())
    updated_at = Column(DateTime, default=func.now(), onupdate=func.now())

    # Relationships
    conversations = relationship("Conversation", back_populates="user", cascade="all, delete-orphan")
    user_memory = relationship("UserMemory", back_populates="user", cascade="all, delete-orphan")
    preferences = relationship("UserPreference", back_populates="user", cascade="all, delete-orphan")


class Conversation(Base):
    """Conversation model for chat contexts"""
    __tablename__ = "conversations"

    id = Column(Integer, primary_key=True, index=True)
    user_id = Column(Integer, ForeignKey("users.id"), nullable=False)
    title = Column(String(200), nullable=False)
    created_at = Column(DateTime, default=func.now())
    updated_at = Column(DateTime, default=func.now(), onupdate=func.now())
    is_active = Column(Boolean, default=True)
    topic_tags = Column(JSON, nullable=True)  # For topic categorization

    # Relationships
    user = relationship("User", back_populates="conversations")
    messages = relationship("Message", back_populates="conversation", cascade="all, delete-orphan")
    summary = relationship("ConversationSummary", back_populates="conversation", uselist=False)
    debug_sessions = relationship("DebugSession", back_populates="conversation", cascade="all, delete-orphan")


class Message(Base):
    """Individual messages within conversations"""
    __tablename__ = "messages"

    id = Column(Integer, primary_key=True, index=True)
    conversation_id = Column(Integer, ForeignKey("conversations.id"), nullable=False)
    role = Column(String(20), nullable=False)  # 'user', 'assistant', 'system'
    content = Column(Text, nullable=False)
    timestamp = Column(DateTime, default=func.now())

    # Optional metadata
    token_count = Column(Integer, nullable=True)
    llm_model = Column(String(100), nullable=True)
    temperature = Column(Float, nullable=True)
    processing_time = Column(Float, nullable=True)  # seconds

    # Debug information
    debug_enabled = Column(Boolean, default=False)
    debug_data = Column(JSON, nullable=True)  # Store debug information

    # Relationships
    conversation = relationship("Conversation", back_populates="messages")
    debug_steps = relationship("DebugStep", back_populates="message", cascade="all, delete-orphan")
    llm_requests = relationship("LLMRequest", back_populates="message", cascade="all, delete-orphan")


class DebugSession(Base):
    """Debug session tracking for conversations"""
    __tablename__ = "debug_sessions"

    id = Column(Integer, primary_key=True, index=True)
    conversation_id = Column(Integer, ForeignKey("conversations.id"), nullable=False)
    user_id = Column(Integer, ForeignKey("users.id"), nullable=False)
    session_id = Column(String(100), nullable=False, index=True)  # Unique session identifier
    started_at = Column(DateTime, default=func.now())
    ended_at = Column(DateTime, nullable=True)
    is_active = Column(Boolean, default=True)

    # Session metadata
    total_messages = Column(Integer, default=0)
    total_steps = Column(Integer, default=0)
    total_tools_used = Column(Integer, default=0)
    total_processing_time = Column(Float, default=0.0)

    # Relationships
    conversation = relationship("Conversation", back_populates="debug_sessions")
    user = relationship("User")
    debug_steps = relationship("DebugStep", back_populates="debug_session", cascade="all, delete-orphan")


class DebugStep(Base):
    """Individual debug steps within a message processing"""
    __tablename__ = "debug_steps"

    id = Column(Integer, primary_key=True, index=True)
    message_id = Column(Integer, ForeignKey("messages.id"), nullable=False)
    debug_session_id = Column(Integer, ForeignKey("debug_sessions.id"), nullable=False)
    step_id = Column(String(100), nullable=False, index=True)  # Unique step identifier

    # Step information
    step_type = Column(String(50), nullable=False)  # 'tool_call', 'tool_result', 'memory_retrieval', etc.
    step_order = Column(Integer, nullable=False)  # Order within the message processing
    title = Column(String(200), nullable=False)
    description = Column(Text, nullable=True)

    # Timing
    timestamp = Column(DateTime, default=func.now())
    duration_ms = Column(Integer, nullable=True)

    # Status
    success = Column(Boolean, default=True)
    error_message = Column(Text, nullable=True)

    # Data
    input_data = Column(JSON, nullable=True)
    output_data = Column(JSON, nullable=True)
    step_metadata = Column(JSON, nullable=True)

    # Relationships
    message = relationship("Message", back_populates="debug_steps")
    debug_session = relationship("DebugSession", back_populates="debug_steps")


class LLMRequest(Base):
    """LLM request/response tracking for debug purposes"""
    __tablename__ = "llm_requests"

    id = Column(Integer, primary_key=True, index=True)
    message_id = Column(Integer, ForeignKey("messages.id"), nullable=False)
    request_id = Column(String(100), nullable=False, index=True)  # Unique request identifier

    # Request information
    model = Column(String(100), nullable=False)
    temperature = Column(Float, nullable=True)
    max_tokens = Column(Integer, nullable=True)
    stream = Column(Boolean, default=False)

    # Request/Response data
    request_messages = Column(JSON, nullable=False)  # Full request context
    response_data = Column(JSON, nullable=False)  # Full response

    # Timing and usage
    timestamp = Column(DateTime, default=func.now())
    processing_time_ms = Column(Integer, nullable=True)
    token_usage = Column(JSON, nullable=True)

    # Tools information
    tools_available = Column(JSON, nullable=True)
    tools_used = Column(JSON, nullable=True)
    tool_calls = Column(JSON, nullable=True)
    tool_results = Column(JSON, nullable=True)

    # Relationships
    message = relationship("Message", back_populates="llm_requests")


class UserMemory(Base):
    """User memory entries for personalization"""
    __tablename__ = "user_memory"

    id = Column(Integer, primary_key=True, index=True)
    user_id = Column(Integer, ForeignKey("users.id"), nullable=False)
    memory_type = Column(String(50), nullable=False)  # 'explicit', 'implicit', 'preference'
    key = Column(String(200), nullable=False, index=True)
    value = Column(Text, nullable=False)
    confidence = Column(Float, default=1.0)  # 0.0 to 1.0
    source = Column(String(100), nullable=True)  # where this memory came from
    created_at = Column(DateTime, default=func.now())
    updated_at = Column(DateTime, default=func.now(), onupdate=func.now())
    last_accessed = Column(DateTime, default=func.now())
    access_count = Column(Integer, default=0)

    # Relationships
    user = relationship("User", back_populates="user_memory")


class UserPreference(Base):
    """User preferences and settings"""
    __tablename__ = "user_preferences"

    id = Column(Integer, primary_key=True, index=True)
    user_id = Column(Integer, ForeignKey("users.id"), nullable=False)
    category = Column(String(50), nullable=False)  # 'response_style', 'model_settings', 'debug_mode', etc.
    key = Column(String(100), nullable=False)
    value = Column(JSON, nullable=False)
    created_at = Column(DateTime, default=func.now())
    updated_at = Column(DateTime, default=func.now(), onupdate=func.now())

    # Relationships
    user = relationship("User", back_populates="preferences")


class ConversationSummary(Base):
    """Conversation summaries for memory efficiency and search"""
    __tablename__ = "conversation_summaries"

    id = Column(Integer, primary_key=True, index=True)
    conversation_id = Column(Integer, ForeignKey("conversations.id"), nullable=False, unique=True)
    summary = Column(Text, nullable=False)
    keywords = Column(Text, nullable=True)  # Comma-separated keywords for search
    message_count = Column(Integer, nullable=False)
    priority_score = Column(Float, default=0.0)  # For ranking important conversations
    created_at = Column(DateTime, default=func.now())
    updated_at = Column(DateTime, default=func.now(), onupdate=func.now())

    # Relationships
    conversation = relationship("Conversation", back_populates="summary")


class SystemLog(Base):
    """System logs for debugging and monitoring"""
    __tablename__ = "system_logs"

    id = Column(Integer, primary_key=True, index=True)
    level = Column(String(20), nullable=False)  # 'DEBUG', 'INFO', 'WARNING', 'ERROR'
    message = Column(Text, nullable=False)
    component = Column(String(100), nullable=True)
    user_id = Column(Integer, ForeignKey("users.id"), nullable=True)
    log_metadata = Column(JSON, nullable=True)
    timestamp = Column(DateTime, default=func.now())
