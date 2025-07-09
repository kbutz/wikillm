"""
Pydantic schemas for API request/response models
"""
from pydantic import BaseModel, Field
from typing import List, Optional, Dict, Any, Union
from datetime import datetime
from enum import Enum


class MessageRole(str, Enum):
    USER = "user"
    ASSISTANT = "assistant"
    SYSTEM = "system"


class MemoryType(str, Enum):
    EXPLICIT = "explicit"
    IMPLICIT = "implicit"
    PREFERENCE = "preference"


# Base schemas
class UserBase(BaseModel):
    username: str = Field(..., min_length=1, max_length=50)
    email: Optional[str] = Field(None, max_length=100)
    full_name: Optional[str] = Field(None, max_length=100)


class UserCreate(UserBase):
    pass


class User(UserBase):
    id: int
    created_at: datetime
    updated_at: datetime

    class Config:
        from_attributes = True


# Message schemas
class MessageBase(BaseModel):
    role: MessageRole
    content: str = Field(..., min_length=1)


class MessageCreate(MessageBase):
    conversation_id: int


class Message(MessageBase):
    id: int
    conversation_id: int
    timestamp: datetime
    token_count: Optional[int] = None
    llm_model: Optional[str] = None
    temperature: Optional[float] = None
    processing_time: Optional[float] = None
    
    # Debug fields
    debug_enabled: Optional[bool] = None
    debug_data: Optional[Dict[str, Any]] = None
    intermediary_steps: Optional[List[Dict[str, Any]]] = None
    llm_request: Optional[Dict[str, Any]] = None
    llm_response: Optional[Dict[str, Any]] = None
    tool_calls: Optional[List[Dict[str, Any]]] = None
    tool_results: Optional[List[Dict[str, Any]]] = None

    class Config:
        from_attributes = True


# Conversation schemas
class ConversationBase(BaseModel):
    title: str = Field(..., min_length=1, max_length=200)


class ConversationCreate(ConversationBase):
    user_id: int


class Conversation(ConversationBase):
    id: int
    user_id: int
    created_at: datetime
    updated_at: datetime
    is_active: bool = True
    messages: List[Message] = []

    class Config:
        from_attributes = True


# Memory schemas
class UserMemoryBase(BaseModel):
    memory_type: MemoryType
    key: str = Field(..., min_length=1, max_length=200)
    value: str = Field(..., min_length=1)
    confidence: float = Field(1.0, ge=0.0, le=1.0)
    source: Optional[str] = Field(None, max_length=100)


class UserMemoryCreate(UserMemoryBase):
    user_id: int


class UserMemory(UserMemoryBase):
    id: int
    user_id: int
    created_at: datetime
    updated_at: datetime
    last_accessed: datetime
    access_count: int

    class Config:
        from_attributes = True


# Preference schemas
class UserPreferenceBase(BaseModel):
    category: str = Field(..., min_length=1, max_length=50)
    key: str = Field(..., min_length=1, max_length=100)
    value: Union[str, int, float, bool, Dict[str, Any], List[Any]]


class UserPreferenceCreate(UserPreferenceBase):
    user_id: int


class UserPreference(UserPreferenceBase):
    id: int
    user_id: int
    created_at: datetime
    updated_at: datetime

    class Config:
        from_attributes = True


# Chat request/response schemas
class ChatRequest(BaseModel):
    message: str = Field(..., min_length=1)
    conversation_id: Optional[int] = None
    user_id: int
    temperature: Optional[float] = Field(None, ge=0.0, le=2.0)
    max_tokens: Optional[int] = Field(None, ge=1, le=4096)
    stream: bool = False


class ChatResponse(BaseModel):
    message: Message
    conversation_id: int
    processing_time: float
    token_count: Optional[int] = None


class StreamChatResponse(BaseModel):
    chunk: str
    conversation_id: int
    finished: bool = False


# System schemas
class SystemStatus(BaseModel):
    status: str
    version: str
    lmstudio_connected: bool
    database_connected: bool
    active_conversations: int
    total_users: int


class ErrorResponse(BaseModel):
    error: str
    detail: Optional[str] = None
    timestamp: datetime = Field(default_factory=datetime.now)


# Bulk operations
class BulkMemoryUpdate(BaseModel):
    memories: List[UserMemoryCreate]


class ConversationSummaryRequest(BaseModel):
    conversation_id: int
    force_update: bool = False


class ConversationSummary(BaseModel):
    id: int
    conversation_id: int
    summary: str
    message_count: int
    created_at: datetime

    class Config:
        from_attributes = True
