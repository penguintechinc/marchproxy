"""
Ollama Provider Connector
Handles connections to Ollama local LLM server
"""

import logging
from typing import Dict, List, Optional, Any, Tuple
import aiohttp
import tiktoken

logger = logging.getLogger(__name__)


class OllamaConnector:
    """Ollama local LLM connector"""

    def __init__(self, base_url: str = "http://localhost:11434"):
        self.base_url = base_url
        self.model_list = []
        self.session = None

        # Token estimator
        try:
            self.token_estimator = tiktoken.encoding_for_model("gpt-3.5-turbo")
        except Exception:
            self.token_estimator = None

    async def _ensure_session(self):
        """Ensure HTTP session is initialized"""
        if self.session is None:
            self.session = aiohttp.ClientSession()

    async def discover_models(self):
        """Discover available models from Ollama"""
        try:
            await self._ensure_session()
            async with self.session.get(f"{self.base_url}/api/tags") as response:
                if response.status == 200:
                    result = await response.json()
                    self.model_list = [model['name'] for model in result.get('models', [])]
                    logger.info(f"Discovered {len(self.model_list)} Ollama models: {self.model_list}")
        except Exception as e:
            logger.error(f"Failed to discover Ollama models: {e}")

    async def chat_completion(
        self,
        messages: List[Dict[str, str]],
        model: str,
        **kwargs
    ) -> Tuple[str, Dict[str, Any]]:
        """Generate Ollama chat completion"""
        try:
            await self._ensure_session()

            payload = {
                'model': model,
                'messages': messages,
                'stream': False,
                'options': {
                    'temperature': kwargs.get('temperature', 0.7),
                    'num_predict': kwargs.get('max_tokens', -1)
                }
            }

            async with self.session.post(
                f"{self.base_url}/api/chat",
                json=payload,
                timeout=aiohttp.ClientTimeout(total=300)
            ) as response:
                if response.status != 200:
                    raise Exception(f"Ollama API error: {response.status}")

                result = await response.json()
                content = result['message']['content']

                # Estimate token usage
                input_text = " ".join([msg['content'] for msg in messages])
                input_tokens = self._estimate_tokens(input_text)
                output_tokens = self._estimate_tokens(content)

                usage_info = {
                    'input_tokens': input_tokens,
                    'output_tokens': output_tokens,
                    'total_tokens': input_tokens + output_tokens,
                    'model': model,
                    'finish_reason': result.get('done_reason', 'stop'),
                    'provider': 'ollama'
                }

                return content, usage_info

        except Exception as e:
            logger.error(f"Ollama completion failed: {e}")
            raise

    def _estimate_tokens(self, text: str) -> int:
        """Estimate tokens for text"""
        try:
            if self.token_estimator:
                return len(self.token_estimator.encode(text))
            else:
                return len(text) // 4
        except Exception:
            return len(text) // 4

    async def count_tokens(self, text: str, model: str) -> int:
        """Estimate tokens for text"""
        return self._estimate_tokens(text)

    async def list_models(self) -> List[Dict[str, Any]]:
        """List Ollama models"""
        try:
            await self._ensure_session()
            async with self.session.get(f"{self.base_url}/api/tags") as response:
                if response.status != 200:
                    return []

                result = await response.json()
                models = []

                for model_data in result.get('models', []):
                    models.append({
                        'id': model_data['name'],
                        'object': 'model',
                        'owned_by': 'ollama',
                        'provider': 'ollama',
                        'size': model_data.get('size', 0)
                    })

                return models

        except Exception as e:
            logger.error(f"Failed to list Ollama models: {e}")
            return []

    async def health_check(self) -> Dict[str, Any]:
        """Check Ollama health"""
        try:
            await self._ensure_session()
            async with self.session.get(f"{self.base_url}/api/tags") as response:
                if response.status == 200:
                    return {
                        'status': 'healthy',
                        'provider': 'ollama',
                        'endpoint': self.base_url
                    }
                else:
                    raise Exception(f"HTTP {response.status}")

        except Exception as e:
            return {
                'status': 'unhealthy',
                'provider': 'ollama',
                'error': str(e)
            }

    async def close(self):
        """Close the HTTP session"""
        if self.session:
            await self.session.close()
