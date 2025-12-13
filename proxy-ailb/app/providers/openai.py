"""
OpenAI Provider Connector
Handles connections to OpenAI API
"""

import logging
from typing import Dict, List, Optional, Any, Tuple
import openai
import tiktoken

logger = logging.getLogger(__name__)


class OpenAIConnector:
    """OpenAI API connector"""

    def __init__(self, api_key: str, base_url: Optional[str] = None, models: Optional[List[str]] = None):
        self.api_key = api_key
        self.base_url = base_url
        self.model_list = models or ['gpt-4', 'gpt-3.5-turbo']

        # Initialize client
        self.client = openai.AsyncOpenAI(
            api_key=self.api_key,
            base_url=self.base_url
        )

        # Initialize tokenizers
        self.encoders = {}
        try:
            for model in ['gpt-4', 'gpt-3.5-turbo']:
                self.encoders[model] = tiktoken.encoding_for_model(model)
            self.default_encoder = tiktoken.encoding_for_model("gpt-3.5-turbo")
        except Exception as e:
            logger.warning(f"Failed to initialize tokenizers: {e}")
            self.default_encoder = None

    async def chat_completion(
        self,
        messages: List[Dict[str, str]],
        model: str,
        **kwargs
    ) -> Tuple[str, Dict[str, Any]]:
        """Generate OpenAI chat completion"""
        try:
            response = await self.client.chat.completions.create(
                model=model,
                messages=messages,
                **kwargs
            )

            content = response.choices[0].message.content
            usage_info = {
                'input_tokens': response.usage.prompt_tokens,
                'output_tokens': response.usage.completion_tokens,
                'total_tokens': response.usage.total_tokens,
                'model': response.model,
                'finish_reason': response.choices[0].finish_reason,
                'provider': 'openai'
            }

            return content, usage_info

        except Exception as e:
            logger.error(f"OpenAI completion failed: {e}")
            raise

    async def count_tokens(self, text: str, model: str) -> int:
        """Count tokens using OpenAI tokenizer"""
        try:
            encoder = self.encoders.get(model, self.default_encoder)
            if encoder:
                return len(encoder.encode(text))
            else:
                # Fallback: rough estimation
                return len(text) // 4
        except Exception as e:
            logger.warning(f"Token counting failed: {e}")
            return len(text) // 4

    async def list_models(self) -> List[Dict[str, Any]]:
        """List OpenAI models"""
        try:
            models_response = await self.client.models.list()
            models = []

            for model in models_response.data:
                if not self.model_list or model.id in self.model_list:
                    models.append({
                        'id': model.id,
                        'object': 'model',
                        'created': model.created,
                        'owned_by': model.owned_by,
                        'provider': 'openai'
                    })

            return models

        except Exception as e:
            logger.error(f"Failed to list OpenAI models: {e}")
            # Return configured models as fallback
            return [{'id': m, 'object': 'model', 'provider': 'openai'} for m in self.model_list]

    async def health_check(self) -> Dict[str, Any]:
        """Check OpenAI API health"""
        try:
            await self.client.models.list()
            return {
                'status': 'healthy',
                'provider': 'openai'
            }
        except Exception as e:
            return {
                'status': 'unhealthy',
                'provider': 'openai',
                'error': str(e)
            }

    async def close(self):
        """Close the client"""
        await self.client.close()
