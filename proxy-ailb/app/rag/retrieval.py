"""
RAG Manager
Provides knowledge base retrieval for context enrichment
Simplified version ported from WaddleAI rag_integration.py
"""

import logging
from typing import Dict, Any, List, Optional
from datetime import datetime
import chromadb
from chromadb.config import Settings
from sentence_transformers import SentenceTransformer

logger = logging.getLogger(__name__)


class RAGManager:
    """Manages knowledge base retrieval for RAG"""

    def __init__(self, backend: str = "chromadb", persist_directory: str = "./ailb_rag"):
        self.backend = backend
        self.persist_directory = persist_directory
        self.client = None
        self.collections = {}
        self.encoder = None

    async def initialize(self):
        """Initialize RAG storage"""
        try:
            # Initialize ChromaDB client
            self.client = chromadb.PersistentClient(
                path=self.persist_directory,
                settings=Settings(
                    anonymized_telemetry=False,
                    allow_reset=True
                )
            )

            # Initialize embedding model
            self.encoder = SentenceTransformer('all-MiniLM-L6-v2')
            logger.info("RAG manager initialized successfully")

        except Exception as e:
            logger.error(f"Failed to initialize RAG manager: {e}")
            raise

    def _get_or_create_collection(self, collection_name: str):
        """Get or create a collection"""
        if collection_name in self.collections:
            return self.collections[collection_name]

        try:
            collection = self.client.get_collection(name=collection_name)
            logger.info(f"Loaded existing collection: {collection_name}")
        except:
            collection = self.client.create_collection(
                name=collection_name,
                metadata={"description": "AILB RAG knowledge base"}
            )
            logger.info(f"Created new collection: {collection_name}")

        self.collections[collection_name] = collection
        return collection

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
        messages: List[Dict[str, str]],
        collection: str = "default",
        top_k: int = 3
    ) -> Dict[str, Any]:
        """Get RAG context from knowledge base"""
        try:
            # Extract query from user messages
            user_messages = [msg['content'] for msg in messages if msg.get('role') == 'user']
            if not user_messages:
                return {}

            query = user_messages[-1]  # Use last user message as query

            # Get collection
            rag_collection = self._get_or_create_collection(collection)

            # Generate query embedding
            query_embedding = self._generate_embedding(query)
            if not query_embedding:
                return {}

            # Search knowledge base
            results = rag_collection.query(
                query_embeddings=[query_embedding],
                n_results=top_k,
                include=["documents", "metadatas", "distances"]
            )

            # Build context
            documents = []
            if results and results['documents']:
                for i in range(len(results['documents'][0])):
                    distance = results['distances'][0][i]
                    score = 1.0 - distance

                    if score >= 0.7:  # Only include relevant documents
                        documents.append({
                            'content': results['documents'][0][i],
                            'metadata': results['metadatas'][0][i],
                            'score': score
                        })

            return {
                'collection': collection,
                'documents': documents,
                'document_count': len(documents)
            }

        except Exception as e:
            logger.error(f"Failed to get RAG context: {e}")
            return {}

    async def enhance_messages(
        self,
        messages: List[Dict[str, str]],
        context: Dict[str, Any]
    ) -> List[Dict[str, str]]:
        """Enhance messages with RAG context"""
        try:
            documents = context.get('documents', [])
            if not documents:
                return messages

            # Build context text from documents
            context_parts = []
            for idx, doc in enumerate(documents[:3]):  # Use top 3 documents
                content = doc['content']
                score = doc['score']
                context_parts.append(f"[Document {idx+1}] (Relevance: {score:.2f})\n{content}")

            rag_context = "Relevant Knowledge Base Context:\n" + "\n\n".join(context_parts)

            # Add context to system message or create new system message
            enhanced_messages = []
            has_system_message = False

            for msg in messages:
                if msg.get('role') == 'system':
                    # Enhance existing system message
                    enhanced_content = msg['content'] + f"\n\n{rag_context}"
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
                    'content': rag_context
                })

            logger.info(f"Enriched request with {len(documents)} RAG documents")
            return enhanced_messages

        except Exception as e:
            logger.error(f"Failed to enhance messages with RAG context: {e}")
            return messages

    async def add_documents(
        self,
        documents: List[Dict[str, str]],
        collection: str = "default"
    ) -> int:
        """Add documents to knowledge base"""
        try:
            rag_collection = self._get_or_create_collection(collection)

            ids = []
            contents = []
            metadatas = []
            embeddings = []

            for idx, doc in enumerate(documents):
                content = doc.get('content', '')
                metadata = doc.get('metadata', {})

                # Generate embedding
                embedding = self._generate_embedding(content)
                if not embedding:
                    continue

                doc_id = doc.get('id', f"doc_{collection}_{idx}_{int(datetime.now().timestamp())}")
                ids.append(doc_id)
                contents.append(content)
                metadatas.append(metadata)
                embeddings.append(embedding)

            # Add to collection
            if ids:
                rag_collection.add(
                    ids=ids,
                    documents=contents,
                    metadatas=metadatas,
                    embeddings=embeddings
                )
                logger.info(f"Added {len(ids)} documents to collection '{collection}'")

            return len(ids)

        except Exception as e:
            logger.error(f"Failed to add documents: {e}")
            return 0
