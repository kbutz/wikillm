"""
Database Models for AI Assistant
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

    # Relationships
    user = relationship("User", back_populates="conversations")
    messages = relationship("Message", back_populates="conversation", cascade="all, delete-orphan")


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
    model_used = Column(String(100), nullable=True)
    temperature = Column(Float, nullable=True)
    processing_time = Column(Float, nullable=True)  # seconds

    # Relationships
    conversation = relationship("Conversation", back_populates="messages")


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
    category = Column(String(50), nullable=False)  # 'response_style', 'model_settings', etc.
    key = Column(String(100), nullable=False)
    value = Column(JSON, nullable=False)
    created_at = Column(DateTime, default=func.now())
    updated_at = Column(DateTime, default=func.now(), onupdate=func.now())

    # Relationships
    user = relationship("User", back_populates="preferences")


class ConversationSummary(Base):
    """Conversation summaries for memory efficiency"""
    __tablename__ = "conversation_summaries"

    id = Column(Integer, primary_key=True, index=True)
    conversation_id = Column(Integer, ForeignKey("conversations.id"), nullable=False)
    summary = Column(Text, nullable=False)
    message_count = Column(Integer, nullable=False)
    created_at = Column(DateTime, default=func.now())

    # Relationships
    conversation = relationship("Conversation")


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
