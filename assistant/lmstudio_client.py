"""
LMStudio Integration Client - Fixed with MCP Tool Support
"""
import httpx
import json
import logging
from typing import Dict, Any, Optional, AsyncGenerator, List
from config import settings
from llm_response_processor import LLMResponseProcessor, ThinkingModelHandler

logger = logging.getLogger(__name__)


class LMStudioClient:
    """Client for interacting with LMStudio's local API"""
    
    def __init__(self):
        self.base_url = settings.lmstudio_base_url
        self.model = settings.lmstudio_model
        self.timeout = settings.lmstudio_timeout
        self.client = httpx.AsyncClient(timeout=self.timeout)
        
        # Initialize thinking model handler
        self.thinking_handler = ThinkingModelHandler(enable_thinking_logs=False)
        self.response_processor = LLMResponseProcessor()
    
    async def health_check(self) -> bool:
        """Check if LMStudio is running and accessible"""
        try:
            response = await self.client.get(f"{self.base_url}/v1/models")
            return response.status_code == 200
        except Exception as e:
            logger.error(f"LMStudio health check failed: {e}")
            return False
    
    async def get_models(self) -> List[Dict[str, Any]]:
        """Get available models from LMStudio"""
        try:
            response = await self.client.get(f"{self.base_url}/v1/models")
            response.raise_for_status()
            return response.json().get("data", [])
        except Exception as e:
            logger.error(f"Failed to get models: {e}")
            return []
    
    async def chat_completion(
        self,
        messages: List[Dict[str, str]],
        temperature: float = None,
        max_tokens: int = None,
        stream: bool = False,
        tools: Optional[List[Dict[str, Any]]] = None,
        tool_choice: Optional[str] = None,
        debug_context: Optional[Dict[str, Any]] = None
    ) -> Dict[str, Any]:
        """Send chat completion request to LMStudio with MCP tool support and debug capture"""
        
        payload = {
            "model": self.model,
            "messages": messages,
            "temperature": temperature or settings.default_temperature,
            "max_tokens": max_tokens or settings.default_max_tokens,
            "stream": stream
        }
        
        # Add tools if provided (for MCP integration)
        if tools:
            payload["tools"] = tools
            if tool_choice:
                payload["tool_choice"] = tool_choice
        
        # Store debug information if provided
        if debug_context is not None:
            from datetime import datetime
            debug_context['llm_request_payload'] = payload.copy()
            debug_context['llm_request_timestamp'] = datetime.now().isoformat()
            debug_context['llm_request_messages_count'] = len(messages)
            debug_context['llm_request_tools_count'] = len(tools) if tools else 0
            logger.info(f"LLM Request captured: {payload['model']} - {len(messages)} messages, {len(tools) if tools else 0} tools")
        
        try:
            if stream:
                return await self._stream_completion(payload, debug_context)
            else:
                return await self._single_completion(payload, debug_context)
        except Exception as e:
            logger.error(f"Chat completion failed: {e}")
            if debug_context:
                debug_context['llm_request_error'] = str(e)
            raise
    
    async def _single_completion(self, payload: Dict[str, Any], debug_context: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """Handle single (non-streaming) completion"""
        import time
        
        start_time = time.time()
        
        response = await self.client.post(
            f"{self.base_url}/v1/chat/completions",
            json=payload
        )
        response.raise_for_status()
        raw_response = response.json()
        
        processing_time_ms = int((time.time() - start_time) * 1000)
        
        # Store debug information if provided
        if debug_context is not None:
            from datetime import datetime
            debug_context['llm_response_raw'] = raw_response.copy()
            debug_context['llm_response_timestamp'] = datetime.now().isoformat()
            debug_context['llm_processing_time_ms'] = processing_time_ms
            debug_context['llm_response_status'] = response.status_code
            debug_context['llm_response_tokens'] = raw_response.get('usage', {}).get('total_tokens', 0)
            
            # Log the full request/response for debugging
            logger.info(f"LLM Request completed in {processing_time_ms}ms, {debug_context['llm_response_tokens']} tokens")
            logger.debug(f"Full LLM Request: {debug_context.get('llm_request_payload', {})}")
            logger.debug(f"Full LLM Response: {raw_response}")
        
        # Process response to remove thinking tags
        processed_response = self.response_processor.process_chat_response(raw_response)
        
        return processed_response
    
    async def _stream_completion(self, payload: Dict[str, Any], debug_context: Optional[Dict[str, Any]] = None) -> AsyncGenerator[Dict[str, Any], None]:
        """Handle streaming completion"""
        import time
        
        start_time = time.time()
        
        async with self.client.stream(
            "POST",
            f"{self.base_url}/v1/chat/completions",
            json=payload
        ) as response:
            response.raise_for_status()
            
            # Store debug information if provided
            if debug_context:
                debug_context['llm_stream_start_time'] = start_time
                debug_context['llm_response_status'] = response.status_code
                debug_context['llm_stream_chunks'] = []
            
            in_thinking_block = False
            accumulated_content = ""
            
            async for line in response.aiter_lines():
                if line.startswith("data: "):
                    data = line[6:]  # Remove "data: " prefix
                    if data.strip() == "[DONE]":
                        # Calculate final processing time
                        if debug_context:
                            from datetime import datetime
                            debug_context['llm_processing_time_ms'] = int((time.time() - start_time) * 1000)
                            debug_context['llm_response_timestamp'] = datetime.now().isoformat()
                        break
                    
                    try:
                        chunk = json.loads(data)
                        
                        # Store chunk for debugging
                        if debug_context:
                            debug_context['llm_stream_chunks'].append(chunk)
                        
                        # Process chunk to handle thinking tags
                        processed_chunk = self._process_streaming_chunk(chunk)
                        
                        if processed_chunk is not None:
                            yield processed_chunk
                            
                    except json.JSONDecodeError:
                        continue
    
    def _process_streaming_chunk(self, chunk: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """Process a streaming chunk to handle thinking tags"""
        if not chunk or "choices" not in chunk:
            return chunk
            
        processed_chunk = chunk.copy()
        
        for choice in processed_chunk.get("choices", []):
            delta = choice.get("delta", {})
            content = delta.get("content", "")
            
            if content:
                # Check if this chunk contains thinking tag markers
                if "<think>" in content or "</think>" in content:
                    # Remove thinking tags from this chunk
                    cleaned_content = self.response_processor.remove_thinking_tags(content)
                    delta["content"] = cleaned_content
                    
                    # If the cleaned content is empty, skip this chunk
                    if not cleaned_content.strip():
                        return None
                        
                # Simple heuristic: if content looks like thinking content, skip it
                elif self._looks_like_thinking_content(content):
                    return None
                    
        return processed_chunk
    
    def _looks_like_thinking_content(self, content: str) -> bool:
        """Simple heuristic to detect thinking content in streaming"""
        # This is a simple heuristic - in practice, you might want more sophisticated detection
        thinking_phrases = [
            "let me think", "i need to", "let me analyze", "thinking about",
            "i should", "let me consider", "i'm thinking", "hmm", "well"
        ]
        
        content_lower = content.lower().strip()
        
        # Skip very short content that might be thinking
        if len(content_lower) < 10 and any(phrase in content_lower for phrase in thinking_phrases):
            return True
            
        return False
    
    def is_connected(self) -> bool:
        """Check if client is connected (for backward compatibility)"""
        # This is a simple check - in a real implementation you might want to cache this
        try:
            import asyncio
            return asyncio.create_task(self.health_check()).result()
        except:
            return False
    
    async def create_embedding(self, text: str) -> List[float]:
        """Create text embedding (if supported by the model)"""
        try:
            payload = {
                "model": self.model,
                "input": text
            }
            
            response = await self.client.post(
                f"{self.base_url}/v1/embeddings",
                json=payload
            )
            response.raise_for_status()
            
            result = response.json()
            return result["data"][0]["embedding"]
        except Exception as e:
            logger.warning(f"Embedding creation failed: {e}")
            return []
    
    async def __aenter__(self):
        return self
    
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        await self.client.aclose()


# Global client instance
lmstudio_client = LMStudioClient()
