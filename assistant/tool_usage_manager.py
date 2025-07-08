"""
Tool Usage Manager for tracing and analytics
"""
import json
import time
import uuid
import logging
from typing import Dict, List, Any, Optional, AsyncGenerator
from datetime import datetime, timedelta
from sqlalchemy.orm import Session
from contextlib import asynccontextmanager

from models import Message, Conversation, User, UserMemory
from enhanced_schemas import ToolUsageStep, ToolUsageTrace, ToolUsageAnalytics

logger = logging.getLogger(__name__)


class ToolUsageManager:
    """Manages tool usage tracing and analytics for debugging RAG pipeline"""
    
    def __init__(self, db: Session):
        self.db = db
        self.active_traces: Dict[str, ToolUsageTrace] = {}
        self.trace_storage: Dict[str, ToolUsageTrace] = {}  # In-memory storage
        self.max_traces = 1000  # Maximum traces to keep in memory
    
    def create_trace(self, conversation_id: int, user_id: int, message_id: Optional[int] = None) -> str:
        """Create a new tool usage trace"""
        trace_id = str(uuid.uuid4())
        trace = ToolUsageTrace(
            trace_id=trace_id,
            conversation_id=conversation_id,
            user_id=user_id,
            message_id=message_id,
            start_time=datetime.now()
        )
        self.active_traces[trace_id] = trace
        return trace_id
    
    def add_step(
        self,
        trace_id: str,
        tool_name: str,
        tool_type: str,
        step_type: str,
        description: str,
        input_data: Optional[Dict[str, Any]] = None,
        server_id: Optional[str] = None,
        metadata: Optional[Dict[str, Any]] = None
    ) -> str:
        """Add a new step to an active trace"""
        if trace_id not in self.active_traces:
            logger.warning(f"Trace {trace_id} not found in active traces")
            return ""
        
        step_id = str(uuid.uuid4())
        step = ToolUsageStep(
            step_id=step_id,
            tool_name=tool_name,
            tool_type=tool_type,
            server_id=server_id,
            step_type=step_type,
            description=description,
            input_data=input_data,
            status="pending",
            metadata=metadata or {}
        )
        
        self.active_traces[trace_id].steps.append(step)
        return step_id
    
    def update_step(
        self,
        trace_id: str,
        step_id: str,
        status: str,
        output_data: Optional[Dict[str, Any]] = None,
        error_message: Optional[str] = None,
        duration_ms: Optional[int] = None
    ):
        """Update an existing step with results"""
        if trace_id not in self.active_traces:
            return
        
        trace = self.active_traces[trace_id]
        for step in trace.steps:
            if step.step_id == step_id:
                step.status = status
                step.output_data = output_data
                step.error_message = error_message
                step.duration_ms = duration_ms
                break
    
    def finalize_trace(self, trace_id: str) -> Optional[ToolUsageTrace]:
        """Finalize a trace and move it to storage"""
        if trace_id not in self.active_traces:
            return None
        
        trace = self.active_traces[trace_id]
        trace.end_time = datetime.now()
        
        # Calculate summary statistics
        trace.total_steps = len(trace.steps)
        trace.successful_steps = sum(1 for step in trace.steps if step.status == "success")
        trace.failed_steps = sum(1 for step in trace.steps if step.status == "error")
        trace.tools_used = list(set(step.tool_name for step in trace.steps))
        
        if trace.start_time and trace.end_time:
            trace.total_duration_ms = int((trace.end_time - trace.start_time).total_seconds() * 1000)
        
        # Extract RAG-specific information
        trace.rag_queries = [
            {
                "step_id": step.step_id,
                "query": step.input_data.get("query", "") if step.input_data else "",
                "results_count": len(step.output_data.get("results", [])) if step.output_data else 0
            }
            for step in trace.steps
            if step.step_type == "query" and step.tool_type == "rag"
        ]
        
        trace.memories_retrieved = [
            {
                "step_id": step.step_id,
                "memory_type": step.metadata.get("memory_type", "") if step.metadata else "",
                "count": len(step.output_data.get("memories", [])) if step.output_data else 0
            }
            for step in trace.steps
            if step.step_type == "retrieval" and "memory" in step.tool_name.lower()
        ]
        
        # Store trace
        self.trace_storage[trace_id] = trace
        
        # Clean up old traces if necessary
        if len(self.trace_storage) > self.max_traces:
            oldest_trace_id = min(self.trace_storage.keys(), 
                                key=lambda x: self.trace_storage[x].start_time)
            del self.trace_storage[oldest_trace_id]
        
        # Remove from active traces
        del self.active_traces[trace_id]
        
        return trace
    
    def get_trace(self, trace_id: str) -> Optional[ToolUsageTrace]:
        """Get a trace by ID"""
        if trace_id in self.active_traces:
            return self.active_traces[trace_id]
        return self.trace_storage.get(trace_id)
    
    def get_conversation_traces(self, conversation_id: int, limit: int = 10) -> List[ToolUsageTrace]:
        """Get all traces for a conversation"""
        traces = []
        for trace in self.trace_storage.values():
            if trace.conversation_id == conversation_id:
                traces.append(trace)
        
        # Sort by start time, most recent first
        traces.sort(key=lambda x: x.start_time, reverse=True)
        return traces[:limit]
    
    def get_user_traces(self, user_id: int, limit: int = 20) -> List[ToolUsageTrace]:
        """Get all traces for a user"""
        traces = []
        for trace in self.trace_storage.values():
            if trace.user_id == user_id:
                traces.append(trace)
        
        # Sort by start time, most recent first
        traces.sort(key=lambda x: x.start_time, reverse=True)
        return traces[:limit]
    
    def get_analytics(self, conversation_id: int) -> ToolUsageAnalytics:
        """Generate analytics for a conversation"""
        traces = self.get_conversation_traces(conversation_id, limit=100)
        
        if not traces:
            return ToolUsageAnalytics(
                conversation_id=conversation_id,
                total_tool_calls=0,
                unique_tools_used=0,
                average_response_time=0.0,
                success_rate=0.0
            )
        
        # Calculate metrics
        total_calls = sum(trace.total_steps for trace in traces)
        successful_calls = sum(trace.successful_steps for trace in traces)
        all_tools = set()
        response_times = []
        
        tool_breakdown = {}
        temporal_data = []
        error_patterns = []
        
        for trace in traces:
            if trace.total_duration_ms:
                response_times.append(trace.total_duration_ms)
            
            for step in trace.steps:
                all_tools.add(step.tool_name)
                
                # Tool breakdown
                if step.tool_name not in tool_breakdown:
                    tool_breakdown[step.tool_name] = {
                        "total_calls": 0,
                        "successful_calls": 0,
                        "average_duration": 0.0,
                        "error_count": 0
                    }
                
                tool_breakdown[step.tool_name]["total_calls"] += 1
                if step.status == "success":
                    tool_breakdown[step.tool_name]["successful_calls"] += 1
                elif step.status == "error":
                    tool_breakdown[step.tool_name]["error_count"] += 1
                    if step.error_message:
                        error_patterns.append({
                            "tool": step.tool_name,
                            "error": step.error_message,
                            "timestamp": step.timestamp
                        })
                
                if step.duration_ms:
                    current_avg = tool_breakdown[step.tool_name]["average_duration"]
                    current_count = tool_breakdown[step.tool_name]["total_calls"]
                    tool_breakdown[step.tool_name]["average_duration"] = (
                        (current_avg * (current_count - 1) + step.duration_ms) / current_count
                    )
            
            # Temporal analysis
            temporal_data.append({
                "timestamp": trace.start_time,
                "duration": trace.total_duration_ms,
                "tools_used": len(trace.tools_used),
                "success_rate": trace.successful_steps / trace.total_steps if trace.total_steps > 0 else 0
            })
        
        # RAG performance analysis
        rag_performance = self._analyze_rag_performance(traces)
        memory_utilization = self._analyze_memory_utilization(traces)
        
        return ToolUsageAnalytics(
            conversation_id=conversation_id,
            total_tool_calls=total_calls,
            unique_tools_used=len(all_tools),
            average_response_time=sum(response_times) / len(response_times) if response_times else 0.0,
            success_rate=successful_calls / total_calls if total_calls > 0 else 0.0,
            most_used_tool=max(tool_breakdown.keys(), key=lambda x: tool_breakdown[x]["total_calls"]) if tool_breakdown else None,
            tool_breakdown=tool_breakdown,
            temporal_analysis=temporal_data,
            error_patterns=error_patterns,
            rag_performance=rag_performance,
            memory_utilization=memory_utilization
        )
    
    def _analyze_rag_performance(self, traces: List[ToolUsageTrace]) -> Dict[str, Any]:
        """Analyze RAG-specific performance metrics"""
        rag_queries = []
        for trace in traces:
            rag_queries.extend(trace.rag_queries)
        
        if not rag_queries:
            return {}
        
        total_queries = len(rag_queries)
        avg_results = sum(q.get("results_count", 0) for q in rag_queries) / total_queries
        
        return {
            "total_queries": total_queries,
            "average_results_per_query": avg_results,
            "query_distribution": self._get_query_distribution(rag_queries),
            "retrieval_effectiveness": self._calculate_retrieval_effectiveness(rag_queries)
        }
    
    def _analyze_memory_utilization(self, traces: List[ToolUsageTrace]) -> Dict[str, Any]:
        """Analyze memory utilization patterns"""
        memories = []
        for trace in traces:
            memories.extend(trace.memories_retrieved)
        
        if not memories:
            return {}
        
        memory_types = {}
        for memory in memories:
            mem_type = memory.get("memory_type", "unknown")
            if mem_type not in memory_types:
                memory_types[mem_type] = {"count": 0, "total_retrieved": 0}
            memory_types[mem_type]["count"] += 1
            memory_types[mem_type]["total_retrieved"] += memory.get("count", 0)
        
        return {
            "memory_types_used": list(memory_types.keys()),
            "memory_type_distribution": memory_types,
            "total_memories_retrieved": sum(m.get("count", 0) for m in memories)
        }
    
    def _get_query_distribution(self, queries: List[Dict[str, Any]]) -> Dict[str, int]:
        """Get distribution of query types"""
        # Simple classification based on query content
        distribution = {"fact_queries": 0, "context_queries": 0, "memory_queries": 0, "other": 0}
        
        for query in queries:
            query_text = query.get("query", "").lower()
            if any(word in query_text for word in ["fact", "information", "data"]):
                distribution["fact_queries"] += 1
            elif any(word in query_text for word in ["context", "background", "history"]):
                distribution["context_queries"] += 1
            elif any(word in query_text for word in ["memory", "remember", "recall"]):
                distribution["memory_queries"] += 1
            else:
                distribution["other"] += 1
        
        return distribution
    
    def _calculate_retrieval_effectiveness(self, queries: List[Dict[str, Any]]) -> float:
        """Calculate retrieval effectiveness score"""
        if not queries:
            return 0.0
        
        # Simple heuristic: queries with more results are considered more effective
        total_score = 0.0
        for query in queries:
            results_count = query.get("results_count", 0)
            # Score based on having results (1.0) vs no results (0.0)
            score = 1.0 if results_count > 0 else 0.0
            total_score += score
        
        return total_score / len(queries)
    
    @asynccontextmanager
    async def trace_step(
        self,
        trace_id: str,
        tool_name: str,
        tool_type: str,
        step_type: str,
        description: str,
        input_data: Optional[Dict[str, Any]] = None,
        server_id: Optional[str] = None,
        metadata: Optional[Dict[str, Any]] = None
    ):
        """Context manager for tracing a tool usage step"""
        start_time = time.time()
        step_id = self.add_step(
            trace_id, tool_name, tool_type, step_type, description, 
            input_data, server_id, metadata
        )
        
        try:
            yield step_id
            duration_ms = int((time.time() - start_time) * 1000)
            self.update_step(trace_id, step_id, "success", duration_ms=duration_ms)
        except Exception as e:
            duration_ms = int((time.time() - start_time) * 1000)
            self.update_step(
                trace_id, step_id, "error", 
                error_message=str(e), duration_ms=duration_ms
            )
            raise
    
    async def trace_rag_query(
        self, 
        trace_id: str, 
        query: str, 
        results: List[Any]
    ) -> str:
        """Trace a RAG query execution"""
        async with self.trace_step(
            trace_id,
            tool_name="rag_retriever",
            tool_type="rag",
            step_type="query",
            description=f"Executing RAG query: {query[:100]}...",
            input_data={"query": query},
            metadata={"query_length": len(query)}
        ) as step_id:
            self.update_step(
                trace_id, step_id, "success",
                output_data={"results": results, "results_count": len(results)}
            )
            return step_id
    
    async def trace_memory_retrieval(
        self,
        trace_id: str,
        memory_type: str,
        query: str,
        memories: List[UserMemory]
    ) -> str:
        """Trace memory retrieval"""
        async with self.trace_step(
            trace_id,
            tool_name=f"memory_retriever_{memory_type}",
            tool_type="memory",
            step_type="retrieval",
            description=f"Retrieving {memory_type} memories for query",
            input_data={"query": query, "memory_type": memory_type},
            metadata={"memory_type": memory_type}
        ) as step_id:
            memory_data = [
                {
                    "id": m.id,
                    "key": m.key,
                    "confidence": m.confidence,
                    "last_accessed": m.last_accessed.isoformat() if m.last_accessed else None
                }
                for m in memories
            ]
            self.update_step(
                trace_id, step_id, "success",
                output_data={"memories": memory_data, "count": len(memories)}
            )
            return step_id
    
    def get_system_debug_info(self) -> Dict[str, Any]:
        """Get comprehensive system debug information"""
        return {
            "active_traces": len(self.active_traces),
            "stored_traces": len(self.trace_storage),
            "total_memory_usage": sum(
                len(trace.steps) for trace in self.trace_storage.values()
            ),
            "recent_activity": [
                {
                    "trace_id": trace.trace_id,
                    "conversation_id": trace.conversation_id,
                    "start_time": trace.start_time.isoformat(),
                    "steps": len(trace.steps),
                    "status": "active" if trace.trace_id in self.active_traces else "completed"
                }
                for trace in sorted(
                    list(self.trace_storage.values()) + list(self.active_traces.values()),
                    key=lambda x: x.start_time,
                    reverse=True
                )[:10]
            ]
        }
