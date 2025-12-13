"""
Anthropic Provider Connector
Handles connections to Anthropic Claude API
"""

import logging
from typing import Dict, List, Optional, Any, Tuple
import anthropic
import tiktoken

logger = logging.getLogger(__name__)


class AnthropicConnector:
    """Anthropic Claude API connector"""

    def __init__(self, api_key: str, models: Optional[List[str]] = None):
        self.api_key = api_key
        self.model_list = models or ['claude-3-opus-20240229', 'claude-3-sonnet-20240229', 'claude-3-haiku-20240307']

        # Initialize client
        self.client = anthropic.AsyncAnthropic(api_key=self.api_key)

        # Token estimator (Anthropic doesn't provide tokenizers)
        try:
            self.token_estimator = tiktoken.encoding_for_model("gpt-3.5-turbo")
        except Exception:
            self.token_estimator = None

    async def chat_completion(
        self,
        messages: List[Dict[str, str]],
        model: str,
        **kwargs
    ) -> Tuple[str, Dict[str, Any]]:
        """Generate Anthropic chat completion"""
        try:
            # Convert messages to Anthropic format
            system_message = ""
            user_messages = []

            for msg in messages:
                if msg['role'] == 'system':
                    system_message = msg['content']
                else:
                    user_messages.append(msg)

            # Anthropic API call
            response = await self.client.messages.create(
                model=model,
                max_tokens=kwargs.get('max_tokens', 1024),
                system=system_message if system_message else None,
                messages=user_messages
            )

            content = response.content[0].text

            # Estimate token usage
            input_text = system_message + " ".join([msg['content'] for msg in user_messages])
            input_tokens = self._estimate_tokens(input_text)
            output_tokens = self._estimate_tokens(content)

            usage_info = {
                'input_tokens': input_tokens,
                'output_tokens': output_tokens,
                'total_tokens': input_tokens + output_tokens,
                'model': model,
                'finish_reason': response.stop_reason,
                'provider': 'anthropic'
            }

            return content, usage_info

        except Exception as e:
            logger.error(f"Anthropic completion failed: {e}")
            raise

    def _estimate_tokens(self, text: str) -> int:
        """Estimate tokens for Anthropic models"""
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
        """List Anthropic models"""
        # Anthropic doesn't have a models endpoint, return configured models
        return [
            {
                'id': model_id,
                'object': 'model',
                'owned_by': 'anthropic',
                'provider': 'anthropic'
            }
            for model_id in self.model_list
        ]

    async def health_check(self) -> Dict[str, Any]:
        """Check Anthropic API health"""
        try:
            # Simple test message
            test_response = await self.client.messages.create(
                model=self.model_list[0] if self.model_list else 'claude-3-haiku-20240307',
                max_tokens=1,
                messages=[{"role": "user", "content": "hi"}]
            )
            return {
                'status': 'healthy',
                'provider': 'anthropic'
            }
        except Exception as e:
            return {
                'status': 'unhealthy',
                'provider': 'anthropic',
                'error': str(e)
            }

    async def close(self):
        """Close the client"""
        await self.client.close()
