# Enhanced schemas.py - Add tool usage tracking schemas

from pydantic import BaseModel, Field
from typing import List, Optional, Dict, Any, Union, ForwardRef
from datetime import datetime
from enum import Enum

# Import Message model from schemas for serialization
from schemas import Message as MessageSchema
from models import Message as MessageModel


class ToolUsageStep(BaseModel):
    """Individual tool usage step"""
    step_id: str = Field(..., description="Unique identifier for this step")
    tool_name: str = Field(..., description="Name of the tool used")
    tool_type: str = Field(..., description="Type of tool (mcp, internal, etc.)")
    server_id: Optional[str] = Field(None, description="MCP server ID if applicable")
    step_type: str = Field(..., description="Type of step (query, retrieval, processing, etc.)")
    description: str = Field(..., description="Human-readable description of the step")
    timestamp: datetime = Field(default_factory=datetime.now)
    duration_ms: Optional[int] = Field(None, description="Duration in milliseconds")

    # Input/Output data
    input_data: Optional[Dict[str, Any]] = Field(None, description="Input parameters")
    output_data: Optional[Dict[str, Any]] = Field(None, description="Output results")

    # Status and metadata
    status: str = Field(..., description="Status: pending, success, error, timeout")
    error_message: Optional[str] = Field(None, description="Error message if failed")
    metadata: Optional[Dict[str, Any]] = Field(None, description="Additional metadata")


class ToolUsageTrace(BaseModel):
    """Complete tool usage trace for a chat request"""
    trace_id: str = Field(..., description="Unique identifier for this trace")
    conversation_id: int = Field(..., description="Associated conversation ID")
    message_id: Optional[int] = Field(None, description="Associated message ID")
    user_id: int = Field(..., description="User ID")

    # Timing information
    start_time: datetime = Field(default_factory=datetime.now)
    end_time: Optional[datetime] = Field(None)
    total_duration_ms: Optional[int] = Field(None)

    # Tool usage steps
    steps: List[ToolUsageStep] = Field(default_factory=list)

    # Summary information
    total_steps: int = Field(0, description="Total number of steps")
    successful_steps: int = Field(0, description="Number of successful steps")
    failed_steps: int = Field(0, description="Number of failed steps")
    tools_used: List[str] = Field(default_factory=list, description="List of unique tools used")

    # RAG-specific information
    rag_queries: List[Dict[str, Any]] = Field(default_factory=list, description="RAG queries executed")
    memories_retrieved: List[Dict[str, Any]] = Field(default_factory=list, description="Memories retrieved")
    context_size: Optional[int] = Field(None, description="Final context size in tokens")


class ChatRequestWithDebug(BaseModel):
    """Extended chat request with debug options"""
    message: str = Field(..., min_length=1)
    conversation_id: Optional[int] = None
    user_id: int
    temperature: Optional[float] = Field(None, ge=0.0, le=2.0)
    max_tokens: Optional[int] = Field(None, ge=1, le=4096)
    stream: bool = False

    # Debug options
    enable_tool_trace: bool = Field(True, description="Enable tool usage tracing")
    show_debug_steps: bool = Field(True, description="Show debug steps in response")
    trace_level: str = Field("detailed", description="Trace level: basic, detailed, verbose")


class ChatResponseWithDebug(BaseModel):
    """Extended chat response with debug information"""
    message: MessageSchema  # Using the Pydantic Message model for proper serialization
    conversation_id: int
    processing_time: float
    token_count: Optional[int] = None

    # Debug information
    tool_trace: Optional[ToolUsageTrace] = Field(None, description="Tool usage trace")
    debug_enabled: bool = Field(False, description="Whether debug mode was enabled")

    class Config:
        from_attributes = True


class ToolUsageAnalytics(BaseModel):
    """Analytics for tool usage in a conversation"""
    conversation_id: int
    total_tool_calls: int
    unique_tools_used: int
    average_response_time: float
    success_rate: float
    most_used_tool: Optional[str] = None

    # Detailed breakdown
    tool_breakdown: Dict[str, Dict[str, Any]] = Field(default_factory=dict)
    temporal_analysis: List[Dict[str, Any]] = Field(default_factory=list)
    error_patterns: List[Dict[str, Any]] = Field(default_factory=list)

    # RAG-specific analytics
    rag_performance: Dict[str, Any] = Field(default_factory=dict)
    memory_utilization: Dict[str, Any] = Field(default_factory=dict)


class SystemDebugInfo(BaseModel):
    """System-wide debug information"""
    system_status: Dict[str, Any]
    mcp_servers: List[Dict[str, Any]]
    available_tools: List[Dict[str, Any]]
    memory_stats: Dict[str, Any]
    performance_metrics: Dict[str, Any]
    recent_errors: List[Dict[str, Any]]


class DebugCommand(BaseModel):
    """Debug command for power users"""
    command: str = Field(..., description="Debug command to execute")
    parameters: Optional[Dict[str, Any]] = Field(None, description="Command parameters")
    user_id: int = Field(..., description="User ID for authorization")


class DebugCommandResponse(BaseModel):
    """Response to debug command"""
    command: str
    success: bool
    result: Optional[Dict[str, Any]] = None
    error: Optional[str] = None
    execution_time: float
    timestamp: datetime = Field(default_factory=datetime.now)
