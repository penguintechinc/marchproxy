"""
Conversation Memory Manager
Provides conversation memory using mem0 HTTP API (WaddleAI pgvector backend)
"""

import logging
import httpx
import json
from typing import Dict, Any, List, Optional
from datetime import datetime
import os

logger = logging.getLogger(__name__)


class ConversationMemoryManager:
    """Manages conversation memory using mem0 API (WaddleAI pgvector backend)"""

    def __init__(self, backend: str = "mem0", persist_directory: str = "./ailb_memory"):
        self.backend = backend
        self.persist_directory = persist_directory
        self.mem0_endpoint = os.getenv(
            'MEM0_ENDPOINT',
            'http://waddleai-proxy:8080/mem0'
        )
        self.http_client = None
        self.timeout = 5.0

    async def initialize(self):
        """Initialize memory client (HTTP client for mem0)"""
        try:
            self.http_client = httpx.AsyncClient(
                timeout=self.timeout,
                base_url=self.mem0_endpoint
            )

            # Test connectivity
            try:
                response = await self.http_client.get('/memories?user_id=test&limit=1')
                if response.status_code in (200, 400):  # 400 is OK if user not found
                    logger.info(f"Memory manager connected to {self.mem0_endpoint}")
            except Exception as e:
                logger.warning(f"Could not verify mem0 endpoint {self.mem0_endpoint}: {e}")
                # Continue anyway - endpoint might come online later

            logger.info("Memory manager initialized successfully")

        except Exception as e:
            logger.error(f"Failed to initialize memory manager: {e}")
            raise

    async def get_context(
        self,
        session_id: str,
        current_messages: List[Dict[str, str]],
        context_limit: int = 5
    ) -> Dict[str, Any]:
        """Get conversation context from mem0 memory"""
        try:
            # Extract query from current messages
            user_messages = [msg['content'] for msg in current_messages if msg.get('role') == 'user']
            if not user_messages:
                return {}

            query = " ".join(user_messages[-2:])  # Use last 2 user messages as query

            # Call mem0 search API
            response = await self.http_client.post(
                '/memories/search',
                json={
                    'query': query,
                    'user_id': session_id,
                    'agent_id': session_id,
                    'limit': context_limit,
                    'threshold': 0.7
                }
            )

            if response.status_code != 200:
                logger.warning(f"mem0 search failed: {response.status_code}")
                return {}

            data = response.json()
            results = data.get('results', [])

            # Build context
            memories = []
            for result in results:
                memories.append({
                    'content': result.get('memory', ''),
                    'metadata': result.get('metadata', {}),
                    'score': result.get('score', 0.0)
                })

            return {
                'session_id': session_id,
                'relevant_memories': memories,
                'memory_count': len(memories)
            }

        except Exception as e:
            logger.error(f"Failed to get conversation context: {e}")
            return {}

    async def enhance_messages(
        self,
        messages: List[Dict[str, str]],
        context: Dict[str, Any]
    ) -> List[Dict[str, str]]:
        """Enhance messages with memory context"""
        try:
            memories = context.get('relevant_memories', [])
            if not memories:
                return messages

            # Build context text
            context_parts = []
            for memory in memories[:3]:  # Use top 3 memories
                metadata = memory.get('metadata', {})
                timestamp = metadata.get('timestamp', metadata.get('created_at', 'unknown'))
                content = memory.get('content', '')

                if len(content) > 300:
                    content = content[:300] + "..."
                context_parts.append(f"[{timestamp}] {content}")

            context_text = "Previous conversation context:\n" + "\n".join(context_parts)

            # Add context to system message or create new system message
            enhanced_messages = []
            has_system_message = False

            for msg in messages:
                if msg.get('role') == 'system':
                    # Enhance existing system message
                    enhanced_content = msg['content'] + f"\n\n{context_text}"
                    enhanced_messages.append({
                        'role': 'system',
                        'content': enhanced_content
                    })
                    has_system_message = True
                else:
                    enhanced_messages.append(msg)

            # If no system message, add context as new system message
            if not has_system_message:
                enhanced_messages.insert(0, {
                    'role': 'system',
                    'content': context_text
                })

            return enhanced_messages

        except Exception as e:
            logger.error(f"Failed to enhance messages with context: {e}")
            return messages

    async def store_turn(
        self,
        session_id: str,
        messages: List[Dict[str, str]],
        response: str,
        metadata: Dict[str, Any]
    ) -> bool:
        """Store a conversation turn in mem0 memory"""
        try:
            # Combine user message and assistant response
            user_messages = [msg for msg in messages if msg.get('role') == 'user']
            last_user_message = user_messages[-1]['content'] if user_messages else ""

            conversation_text = f"User: {last_user_message}\nAssistant: {response}"

            # Call mem0 add memory API
            response_obj = await self.http_client.post(
                '/memories',
                json={
                    'messages': [
                        {'role': 'user', 'content': last_user_message},
                        {'role': 'assistant', 'content': response}
                    ],
                    'user_id': session_id,
                    'agent_id': session_id,
                    'metadata': {
                        'model': metadata.get('model', 'unknown'),
                        'provider': metadata.get('provider', 'unknown'),
                        'input_tokens': metadata.get('input_tokens', 0),
                        'output_tokens': metadata.get('output_tokens', 0)
                    }
                }
            )

            if response_obj.status_code not in (200, 201):
                logger.warning(f"mem0 store failed: {response_obj.status_code}")
                return False

            return True

        except Exception as e:
            logger.error(f"Failed to store conversation turn: {e}")
            return False

    async def close(self):
        """Close HTTP client connection"""
        if self.http_client:
            await self.http_client.aclose()
