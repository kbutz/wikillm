"""
Enhanced message schemas for chat UI with intermediary steps and full LLM requests
"""
from pydantic import BaseModel, Field
from typing import List, Dict, Any, Optional, Union
from datetime import datetime
from enum import Enum

from schemas import MessageRole, MessageBase, Message as BaseMessage


class IntermediaryStepType(str, Enum):
    """Types of intermediary steps in conversation processing"""
    TOOL_CALL = "tool_call"
    TOOL_RESULT = "tool_result"
    MEMORY_RETRIEVAL = "memory_retrieval"
    CONTEXT_BUILDING = "context_building"
    LLM_REQUEST = "llm_request"
    LLM_RESPONSE = "llm_response"
    ERROR = "error"


class IntermediaryStep(BaseModel):
    """Individual step in the conversation processing pipeline"""
    step_id: str
    step_type: IntermediaryStepType
    timestamp: datetime
    title: str
    description: Optional[str] = None
    data: Dict[str, Any] = Field(default_factory=dict)
    duration_ms: Optional[int] = None
    success: bool = True
    error_message: Optional[str] = None


class ToolCall(BaseModel):
    """Tool call information"""
    tool_name: str
    arguments: Dict[str, Any]
    server_id: Optional[str] = None


class ToolResult(BaseModel):
    """Tool execution result"""
    tool_name: str
    success: bool
    result: Any
    error_message: Optional[str] = None
    execution_time_ms: Optional[int] = None


class LLMRequest(BaseModel):
    """Full LLM request details"""
    model: str
    messages: List[Dict[str, Any]]
    temperature: Optional[float] = None
    max_tokens: Optional[int] = None
    tools: Optional[List[Dict[str, Any]]] = None
    tool_choice: Optional[str] = None
    stream: bool = False
    timestamp: datetime


class LLMResponse(BaseModel):
    """LLM response details"""
    response: Dict[str, Any]
    timestamp: datetime
    processing_time_ms: int
    token_usage: Optional[Dict[str, Any]] = None


class EnhancedMessage(BaseMessage):
    """Enhanced message with intermediary steps and full LLM context"""
    # Base message fields are inherited
    
    # Enhanced fields
    intermediary_steps: List[IntermediaryStep] = Field(default_factory=list)
    llm_request: Optional[LLMRequest] = None
    llm_response: Optional[LLMResponse] = None
    tool_calls: List[ToolCall] = Field(default_factory=list)
    tool_results: List[ToolResult] = Field(default_factory=list)
    
    # Processing metadata
    total_processing_time_ms: Optional[int] = None
    step_count: int = 0
    error_count: int = 0
    
    class Config:
        from_attributes = True


class ChatRequestWithDebug(BaseModel):
    """Chat request with debug information enabled"""
    message: str = Field(..., min_length=1)
    conversation_id: Optional[int] = None
    user_id: int
    temperature: Optional[float] = Field(None, ge=0.0, le=2.0)
    max_tokens: Optional[int] = Field(None, ge=1, le=4096)
    stream: bool = False
    
    # Debug options
    include_intermediary_steps: bool = True
    include_llm_request: bool = True
    include_tool_details: bool = True
    include_context_building: bool = True


class ChatResponseWithDebug(BaseModel):
    """Chat response with debug information"""
    message: EnhancedMessage
    conversation_id: int
    processing_time: float
    token_count: Optional[int] = None
    
    # Debug information
    total_steps: int = 0
    successful_steps: int = 0
    failed_steps: int = 0
    tools_used: List[str] = Field(default_factory=list)


class ConversationDebugSummary(BaseModel):
    """Summary of debug information for a conversation"""
    conversation_id: int
    total_messages: int
    messages_with_tools: int
    total_tool_calls: int
    total_intermediary_steps: int
    average_processing_time_ms: float
    most_used_tools: List[str]
    error_rate: float
    
    
class StepTracker:
    """Utility class for tracking intermediary steps during conversation processing"""
    
    def __init__(self):
        self.steps: List[IntermediaryStep] = []
        self.start_time = datetime.now()
        self.current_step_start: Optional[datetime] = None
    
    def start_step(self, step_type: IntermediaryStepType, title: str, description: Optional[str] = None) -> str:
        """Start tracking a new step"""
        step_id = f"step_{len(self.steps) + 1}_{step_type}_{int(datetime.now().timestamp() * 1000)}"
        self.current_step_start = datetime.now()
        
        step = IntermediaryStep(
            step_id=step_id,
            step_type=step_type,
            timestamp=self.current_step_start,
            title=title,
            description=description
        )
        
        self.steps.append(step)
        return step_id
    
    def update_step(self, step_id: str, data: Dict[str, Any] = None, success: bool = True, error_message: Optional[str] = None):
        """Update an existing step"""
        for step in self.steps:
            if step.step_id == step_id:
                if data:
                    step.data.update(data)
                step.success = success
                if error_message:
                    step.error_message = error_message
                
                # Calculate duration
                if self.current_step_start:
                    duration = datetime.now() - self.current_step_start
                    step.duration_ms = int(duration.total_seconds() * 1000)
                
                break
    
    def complete_step(self, step_id: str, data: Dict[str, Any] = None, success: bool = True, error_message: Optional[str] = None):
        """Complete a step and calculate final duration"""
        self.update_step(step_id, data, success, error_message)
        self.current_step_start = None
    
    def get_steps(self) -> List[IntermediaryStep]:
        """Get all tracked steps"""
        return self.steps
    
    def get_total_duration_ms(self) -> int:
        """Get total processing time in milliseconds"""
        if not self.steps:
            return 0
        
        total_duration = datetime.now() - self.start_time
        return int(total_duration.total_seconds() * 1000)
    
    def get_summary(self) -> Dict[str, Any]:
        """Get summary statistics"""
        successful_steps = sum(1 for step in self.steps if step.success)
        failed_steps = len(self.steps) - successful_steps
        
        return {
            "total_steps": len(self.steps),
            "successful_steps": successful_steps,
            "failed_steps": failed_steps,
            "total_duration_ms": self.get_total_duration_ms(),
            "step_types": [step.step_type for step in self.steps],
            "error_rate": failed_steps / len(self.steps) if self.steps else 0
        }
