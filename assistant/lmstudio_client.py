"""
LMStudio Integration Client
"""
import httpx
import json
import logging
from typing import Dict, Any, Optional, AsyncGenerator, List
from config import settings

logger = logging.getLogger(__name__)


class LMStudioClient:
    """Client for interacting with LMStudio's local API"""
    
    def __init__(self):
        self.base_url = settings.lmstudio_base_url
        self.model = settings.lmstudio_model
        self.timeout = settings.lmstudio_timeout
        self.client = httpx.AsyncClient(timeout=self.timeout)
    
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
        stream: bool = False
    ) -> Dict[str, Any]:
        """Send chat completion request to LMStudio"""
        
        payload = {
            "model": self.model,
            "messages": messages,
            "temperature": temperature or settings.default_temperature,
            "max_tokens": max_tokens or settings.default_max_tokens,
            "stream": stream
        }
        
        try:
            if stream:
                return await self._stream_completion(payload)
            else:
                return await self._single_completion(payload)
        except Exception as e:
            logger.error(f"Chat completion failed: {e}")
            raise
    
    async def _single_completion(self, payload: Dict[str, Any]) -> Dict[str, Any]:
        """Handle single (non-streaming) completion"""
        response = await self.client.post(
            f"{self.base_url}/v1/chat/completions",
            json=payload
        )
        response.raise_for_status()
        return response.json()
    
    async def _stream_completion(self, payload: Dict[str, Any]) -> AsyncGenerator[Dict[str, Any], None]:
        """Handle streaming completion"""
        async with self.client.stream(
            "POST",
            f"{self.base_url}/v1/chat/completions",
            json=payload
        ) as response:
            response.raise_for_status()
            
            async for line in response.aiter_lines():
                if line.startswith("data: "):
                    data = line[6:]  # Remove "data: " prefix
                    if data.strip() == "[DONE]":
                        break
                    
                    try:
                        chunk = json.loads(data)
                        yield chunk
                    except json.JSONDecodeError:
                        continue
    
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
