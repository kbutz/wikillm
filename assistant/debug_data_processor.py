# debug_data_processor.py
"""
Enhanced Debug Data Processing Module
Ensures debug data is properly flagged and attached to messages
"""
import logging
from typing import Dict, List, Optional, Any
from datetime import datetime
from sqlalchemy.orm import Session
from models import Message, DebugStep, LLMRequest, DebugSession
from schemas import Message as MessageSchema

logger = logging.getLogger(__name__)


class DebugDataProcessor:
    """Processes and attaches debug data to messages for inline display"""
    
    def __init__(self, db: Session):
        self.db = db
    
    def process_debug_data_for_message(self, message: Message) -> MessageSchema:
        """Process and attach debug data to a message"""
        try:
            # Convert to Pydantic model
            message_schema = MessageSchema.model_validate(message)
            
            # Initialize debug data flag
            has_debug_data = False
            debug_fields = {
                "intermediary_steps": False,
                "llm_request": False,
                "llm_response": False,
                "tool_calls": False,
                "tool_results": False
            }
            
            # Get debug steps for this message
            debug_steps = self.db.query(DebugStep).filter(
                DebugStep.message_id == message.id
            ).order_by(DebugStep.step_order).all()
            
            if debug_steps:
                # Convert debug steps to intermediary steps format
                intermediary_steps = []
                for step in debug_steps:
                    step_data = {
                        "step_id": step.step_id,
                        "step_type": step.step_type,
                        "timestamp": step.timestamp.isoformat(),
                        "title": step.title,
                        "description": step.description or "",
                        "data": step.input_data or {},
                        "duration_ms": step.duration_ms or 0,
                        "success": step.success,
                        "error_message": step.error_message
                    }
                    intermediary_steps.append(step_data)
                
                # Attach to message
                message_schema.intermediary_steps = intermediary_steps
                has_debug_data = True
                debug_fields["intermediary_steps"] = True
                
                logger.info(f"Attached {len(intermediary_steps)} debug steps to message {message.id}")
            
            # Get LLM requests for this message
            llm_requests = self.db.query(LLMRequest).filter(
                LLMRequest.message_id == message.id
            ).order_by(LLMRequest.timestamp).all()
            
            if llm_requests:
                # Get the most recent LLM request
                llm_request = llm_requests[-1]
                
                # Convert to LLM request format
                llm_request_data = {
                    "model": llm_request.model,
                    "messages": llm_request.request_messages,
                    "temperature": llm_request.temperature,
                    "max_tokens": llm_request.max_tokens,
                    "tools": llm_request.tools_available,
                    "tool_choice": "auto" if llm_request.tools_available else None,
                    "stream": llm_request.stream,
                    "timestamp": llm_request.timestamp.isoformat(),
                    "request_id": llm_request.request_id,
                    "processing_time_ms": llm_request.processing_time_ms
                }
                
                # Attach to message
                message_schema.llm_request = llm_request_data
                has_debug_data = True
                debug_fields["llm_request"] = True
                
                # Convert to LLM response format
                llm_response_data = {
                    "response": llm_request.response_data,
                    "timestamp": llm_request.timestamp.isoformat(),
                    "processing_time_ms": llm_request.processing_time_ms or 0,
                    "token_usage": llm_request.token_usage,
                    "request_id": llm_request.request_id
                }
                
                # Attach to message
                message_schema.llm_response = llm_response_data
                debug_fields["llm_response"] = True
                
                # Add tool calls and results if available
                if llm_request.tool_calls:
                    message_schema.tool_calls = llm_request.tool_calls
                    debug_fields["tool_calls"] = True
                
                if llm_request.tool_results:
                    message_schema.tool_results = llm_request.tool_results
                    debug_fields["tool_results"] = True
                
                logger.info(f"Attached LLM request/response data to message {message.id}")
            
            # Set debug enabled flag
            message_schema.debug_enabled = has_debug_data
            
            # Create comprehensive debug data
            debug_data = {
                "debug_enabled": True,
                "has_debug_data": has_debug_data,
                "debug_fields": debug_fields,
                "timestamp": datetime.now().isoformat(),
                "debug_steps_count": len(debug_steps),
                "llm_requests_count": len(llm_requests),
                "processing_complete": True
            }
            
            # Attach debug data to message
            message_schema.debug_data = debug_data
            
            logger.info(f"Debug data processing complete for message {message.id}: has_debug_data={has_debug_data}")
            
            return message_schema
            
        except Exception as e:
            logger.error(f"Error processing debug data for message {message.id}: {e}")
            # Return message with error debug data
            message_schema = MessageSchema.model_validate(message)
            message_schema.debug_enabled = True
            message_schema.debug_data = {
                "debug_enabled": True,
                "has_debug_data": False,
                "debug_fields": {
                    "intermediary_steps": False,
                    "llm_request": False,
                    "llm_response": False,
                    "tool_calls": False,
                    "tool_results": False
                },
                "error": f"Debug data processing failed: {str(e)}",
                "timestamp": datetime.now().isoformat()
            }
            return message_schema
    
    def ensure_debug_data_completeness(self, message_id: int) -> bool:
        """Ensure debug data is complete for a message"""
        try:
            # Get the message
            message = self.db.query(Message).filter(Message.id == message_id).first()
            if not message:
                logger.warning(f"Message {message_id} not found for debug data check")
                return False
            
            # Check if debug was enabled for this message
            if not message.debug_enabled:
                logger.info(f"Debug not enabled for message {message_id}")
                return False
            
            # Check for debug steps
            debug_steps = self.db.query(DebugStep).filter(
                DebugStep.message_id == message_id
            ).count()
            
            # Check for LLM requests
            llm_requests = self.db.query(LLMRequest).filter(
                LLMRequest.message_id == message_id
            ).count()
            
            has_debug_data = debug_steps > 0 or llm_requests > 0
            
            # Update message debug data if needed
            if has_debug_data:
                debug_data = {
                    "debug_enabled": True,
                    "has_debug_data": True,
                    "debug_fields": {
                        "intermediary_steps": debug_steps > 0,
                        "llm_request": llm_requests > 0,
                        "llm_response": llm_requests > 0,
                        "tool_calls": False,  # Will be set during processing
                        "tool_results": False  # Will be set during processing
                    },
                    "timestamp": datetime.now().isoformat(),
                    "debug_steps_count": debug_steps,
                    "llm_requests_count": llm_requests,
                    "completeness_check": True
                }
                
                # Update message debug data
                message.debug_data = debug_data
                self.db.commit()
                
                logger.info(f"Updated debug data completeness for message {message_id}")
                return True
            
            return False
            
        except Exception as e:
            logger.error(f"Error checking debug data completeness for message {message_id}: {e}")
            return False
    
    def batch_process_debug_data(self, conversation_id: int) -> int:
        """Batch process debug data for all messages in a conversation"""
        try:
            # Get all debug-enabled messages in conversation
            messages = self.db.query(Message).filter(
                Message.conversation_id == conversation_id,
                Message.debug_enabled == True
            ).all()
            
            processed_count = 0
            
            for message in messages:
                if self.ensure_debug_data_completeness(message.id):
                    processed_count += 1
            
            logger.info(f"Batch processed debug data for {processed_count} messages in conversation {conversation_id}")
            return processed_count
            
        except Exception as e:
            logger.error(f"Error batch processing debug data for conversation {conversation_id}: {e}")
            return 0


# Migration function to fix existing messages
def migrate_existing_debug_data(db: Session):
    """Migrate existing debug data to ensure proper flagging"""
    
    debug_processor = DebugDataProcessor(db)
    
    # Get all messages with debug enabled but no debug data
    messages_needing_fix = db.query(Message).filter(
        Message.debug_enabled == True,
        Message.debug_data.is_(None)
    ).all()
    
    fixed_count = 0
    
    for message in messages_needing_fix:
        if debug_processor.ensure_debug_data_completeness(message.id):
            fixed_count += 1
    
    logger.info(f"Migration complete: Fixed debug data for {fixed_count} messages")
    return fixed_count
