"""
LLM Response Processing Utility
Handles removal of <think></think> tags from Qwen-based thinking model responses

This module provides utilities to:
1. Remove <think></think> tags from LLM responses
2. Process chat completions and streaming responses
3. Handle summary generation and memory extraction
4. Provide debugging utilities for thinking model responses
"""
import re
import logging
from typing import Dict, Any, Optional, List, Union

logger = logging.getLogger(__name__)


class LLMResponseProcessor:
    """
    Utility class for processing LLM responses, specifically handling
    thinking model responses that contain <think></think> tags
    """
    
    @staticmethod
    def remove_thinking_tags(text: str) -> str:
        """
        Remove <think></think> tags and their content from LLM responses.
        
        Args:
            text: The raw LLM response text
            
        Returns:
            Cleaned text with thinking tags removed
        """
        if not text:
            return text
            
        # Pattern to match <think>...</think> tags, including multi-line content
        # Uses non-greedy matching to handle multiple think blocks
        pattern = r'<think>.*?</think>'
        
        # Remove all thinking blocks
        cleaned_text = re.sub(pattern, '', text, flags=re.DOTALL | re.IGNORECASE)
        
        # Clean up extra whitespace that might be left behind
        cleaned_text = re.sub(r'\n\s*\n\s*\n', '\n\n', cleaned_text)  # Multiple newlines
        cleaned_text = cleaned_text.strip()
        
        return cleaned_text
    
    @staticmethod
    def process_chat_response(response: Dict[str, Any]) -> Dict[str, Any]:
        """
        Process a chat completion response, removing thinking tags from content.
        
        Args:
            response: The raw chat completion response from LMStudio
            
        Returns:
            Processed response with thinking tags removed
        """
        if not response or "choices" not in response:
            return response
            
        processed_response = response.copy()
        
        for choice in processed_response.get("choices", []):
            message = choice.get("message", {})
            content = message.get("content", "")
            
            if content:
                # Remove thinking tags from content
                cleaned_content = LLMResponseProcessor.remove_thinking_tags(content)
                message["content"] = cleaned_content
                
        return processed_response
    
    @staticmethod
    def process_streaming_chunk(chunk: Dict[str, Any]) -> Dict[str, Any]:
        """
        Process a streaming chunk, filtering out thinking tag content.
        
        Args:
            chunk: A streaming response chunk
            
        Returns:
            Processed chunk with thinking content filtered
        """
        if not chunk or "choices" not in chunk:
            return chunk
            
        processed_chunk = chunk.copy()
        
        for choice in processed_chunk.get("choices", []):
            delta = choice.get("delta", {})
            content = delta.get("content", "")
            
            if content:
                # Check if this content is part of a thinking block
                if LLMResponseProcessor._is_thinking_content(content):
                    # Skip this chunk entirely
                    return None
                    
                # If content contains start or end of thinking tags, clean it
                if "<think>" in content or "</think>" in content:
                    cleaned_content = LLMResponseProcessor.remove_thinking_tags(content)
                    delta["content"] = cleaned_content
                    
        return processed_chunk
    
    @staticmethod
    def _is_thinking_content(content: str) -> bool:
        """
        Check if content appears to be inside a thinking block.
        This is a heuristic for streaming responses.
        
        Args:
            content: The content to check
            
        Returns:
            True if content appears to be thinking content
        """
        # Simple heuristic: if content contains thinking patterns
        thinking_patterns = [
            "let me think",
            "i need to consider",
            "thinking about",
            "let me analyze",
            "i should think about"
        ]
        
        content_lower = content.lower()
        return any(pattern in content_lower for pattern in thinking_patterns)
    
    @staticmethod
    def extract_thinking_content(text: str) -> List[str]:
        """
        Extract the content inside <think></think> tags for debugging purposes.
        
        Args:
            text: The raw LLM response text
            
        Returns:
            List of thinking content blocks
        """
        if not text:
            return []
            
        pattern = r'<think>(.*?)</think>'
        matches = re.findall(pattern, text, flags=re.DOTALL | re.IGNORECASE)
        
        return [match.strip() for match in matches]
    
    @staticmethod
    def has_thinking_tags(text: str) -> bool:
        """
        Check if text contains thinking tags.
        
        Args:
            text: The text to check
            
        Returns:
            True if text contains thinking tags
        """
        if not text:
            return False
            
        return bool(re.search(r'<think>.*?</think>', text, flags=re.DOTALL | re.IGNORECASE))
    
    @staticmethod
    def process_summary_text(text: str) -> str:
        """
        Process text for summary generation, ensuring thinking tags are removed.
        
        Args:
            text: The text to process for summary
            
        Returns:
            Cleaned text ready for summary generation
        """
        cleaned_text = LLMResponseProcessor.remove_thinking_tags(text)
        
        # Additional cleaning for summary generation
        # Remove excessive whitespace and normalize
        cleaned_text = re.sub(r'\s+', ' ', cleaned_text)
        cleaned_text = cleaned_text.strip()
        
        return cleaned_text
    
    @staticmethod
    def process_memory_extraction_text(text: str) -> str:
        """
        Process text for memory extraction, ensuring thinking tags are removed.
        
        Args:
            text: The text to process for memory extraction
            
        Returns:
            Cleaned text ready for memory extraction
        """
        cleaned_text = LLMResponseProcessor.remove_thinking_tags(text)
        
        # Keep more structure for memory extraction
        cleaned_text = cleaned_text.strip()
        
        return cleaned_text


class ThinkingModelHandler:
    """
    Handler specifically for Qwen-based thinking models that use <think></think> tags
    """
    
    def __init__(self, enable_thinking_logs: bool = False):
        """
        Initialize the thinking model handler.
        
        Args:
            enable_thinking_logs: Whether to log thinking content for debugging
        """
        self.enable_thinking_logs = enable_thinking_logs
        self.processor = LLMResponseProcessor()
    
    def process_response(self, response: Dict[str, Any]) -> Dict[str, Any]:
        """
        Process a complete response from a thinking model.
        
        Args:
            response: The raw response from LMStudio
            
        Returns:
            Processed response with thinking content handled
        """
        if not response:
            return response
            
        # Extract thinking content for logging if enabled
        if self.enable_thinking_logs and "choices" in response:
            for choice in response.get("choices", []):
                content = choice.get("message", {}).get("content", "")
                if content:
                    thinking_blocks = self.processor.extract_thinking_content(content)
                    if thinking_blocks:
                        logger.debug(f"Thinking blocks found: {len(thinking_blocks)}")
                        for i, block in enumerate(thinking_blocks):
                            logger.debug(f"Thinking block {i+1}: {block[:200]}...")
        
        # Remove thinking tags from the response
        return self.processor.process_chat_response(response)
    
    def process_streaming_response(self, chunks: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
        """
        Process a list of streaming chunks from a thinking model.
        
        Args:
            chunks: List of streaming response chunks
            
        Returns:
            Processed chunks with thinking content filtered
        """
        processed_chunks = []
        
        for chunk in chunks:
            processed_chunk = self.processor.process_streaming_chunk(chunk)
            if processed_chunk is not None:  # Skip filtered chunks
                processed_chunks.append(processed_chunk)
        
        return processed_chunks
    
    def should_use_thinking_model(self, user_message: str) -> bool:
        """
        Determine if a thinking model should be used for this request.
        
        Args:
            user_message: The user's message
            
        Returns:
            True if thinking model would be beneficial
        """
        # Heuristics for when thinking models are most useful
        thinking_indicators = [
            "analyze", "compare", "explain", "reason", "solve", "calculate",
            "think about", "consider", "evaluate", "pros and cons",
            "step by step", "break down", "complex", "difficult"
        ]
        
        message_lower = user_message.lower()
        return any(indicator in message_lower for indicator in thinking_indicators)
