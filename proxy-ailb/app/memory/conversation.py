"""
Conversation Memory Manager
Provides conversation memory using ChromaDB
Simplified version ported from WaddleAI memory_integration.py
"""

import logging
from typing import Dict, Any, List, Optional
from datetime import datetime, timedelta
import chromadb
from chromadb.config import Settings
from sentence_transformers import SentenceTransformer

logger = logging.getLogger(__name__)


class ConversationMemoryManager:
    """Manages conversation memory for sessions"""

    def __init__(self, backend: str = "chromadb", persist_directory: str = "./ailb_memory"):
        self.backend = backend
        self.persist_directory = persist_directory
        self.client = None
        self.collection = None
        self.encoder = None

    async def initialize(self):
        """Initialize memory storage"""
        try:
            # Initialize ChromaDB client
            self.client = chromadb.PersistentClient(
                path=self.persist_directory,
                settings=Settings(
                    anonymized_telemetry=False,
                    allow_reset=True
                )
            )

            # Get or create collection
            try:
                self.collection = self.client.get_collection(name="conversations")
                logger.info("Loaded existing conversations collection")
            except:
                self.collection = self.client.create_collection(
                    name="conversations",
                    metadata={"description": "AILB conversation memory"}
                )
                logger.info("Created new conversations collection")

            # Initialize embedding model
            self.encoder = SentenceTransformer('all-MiniLM-L6-v2')
            logger.info("Memory manager initialized successfully")

        except Exception as e:
            logger.error(f"Failed to initialize memory manager: {e}")
            raise

    def _generate_embedding(self, text: str) -> List[float]:
        """Generate embedding for text"""
        try:
            embedding = self.encoder.encode(text, convert_to_tensor=False)
            return embedding.tolist()
        except Exception as e:
            logger.error(f"Failed to generate embedding: {e}")
            return None

    async def get_context(
        self,
        session_id: str,
        current_messages: List[Dict[str, str]],
        context_limit: int = 5
    ) -> Dict[str, Any]:
        """Get conversation context from memory"""
        try:
            # Extract query from current messages
            user_messages = [msg['content'] for msg in current_messages if msg.get('role') == 'user']
            if not user_messages:
                return {}

            query = " ".join(user_messages[-2:])  # Use last 2 user messages as query

            # Generate query embedding
            query_embedding = self._generate_embedding(query)
            if not query_embedding:
                return {}

            # Search in memory
            results = self.collection.query(
                query_embeddings=[query_embedding],
                where={"session_id": session_id},
                n_results=context_limit,
                include=["documents", "metadatas", "distances"]
            )

            # Build context
            memories = []
            if results and results['documents']:
                for i in range(len(results['documents'][0])):
                    distance = results['distances'][0][i]
                    score = 1.0 - distance

                    if score >= 0.7:  # Only include relevant memories
                        memories.append({
                            'content': results['documents'][0][i],
                            'metadata': results['metadatas'][0][i],
                            'score': score
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
                timestamp = memory['metadata'].get('timestamp', 'unknown')
                content = memory['content']
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
        """Store a conversation turn in memory"""
        try:
            # Combine user message and assistant response
            user_messages = [msg for msg in messages if msg.get('role') == 'user']
            last_user_message = user_messages[-1]['content'] if user_messages else ""

            conversation_text = f"User: {last_user_message}\nAssistant: {response}"

            # Generate embedding
            embedding = self._generate_embedding(conversation_text)
            if not embedding:
                return False

            # Generate memory ID
            memory_id = f"conv_{session_id}_{int(datetime.utcnow().timestamp() * 1000)}"

            # Prepare metadata
            memory_metadata = {
                "session_id": session_id,
                "timestamp": datetime.utcnow().isoformat(),
                "model": metadata.get('model', 'unknown'),
                "provider": metadata.get('provider', 'unknown'),
                "input_tokens": metadata.get('input_tokens', 0),
                "output_tokens": metadata.get('output_tokens', 0)
            }

            # Store in ChromaDB
            self.collection.add(
                ids=[memory_id],
                documents=[conversation_text],
                metadatas=[memory_metadata],
                embeddings=[embedding]
            )

            logger.debug(f"Stored conversation turn for session {session_id}")
            return True

        except Exception as e:
            logger.error(f"Failed to store conversation turn: {e}")
            return False
