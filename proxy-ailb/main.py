"""
AILB (AI Load Balancer) Container - Main Entry Point
FastAPI proxy server for AI/LLM requests with intelligent routing
Ported from WaddleAI proxy server
"""

import sys
import os
import asyncio
import logging
from contextlib import asynccontextmanager
from typing import Optional, Dict, Any, List
import time
from datetime import datetime

from fastapi import FastAPI, HTTPException, Request, Depends, Header
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse, Response
import uvicorn
import structlog
from prometheus_client import generate_latest, CONTENT_TYPE_LATEST

from app.providers.openai import OpenAIConnector
from app.providers.anthropic import AnthropicConnector
from app.providers.ollama import OllamaConnector
from app.router.intelligent import LLMRequestRouter, RoutingStrategy
from app.memory.conversation import ConversationMemoryManager
from app.rag.retrieval import RAGManager
from app.grpc.server import start_grpc_server
from app.keys.manager import KeyManager
from app.keys import routes as keys_routes
from app.ratelimit.limiter import RateLimiter
from app.ratelimit.middleware import RateLimitMiddleware
from app.billing.tracker import CostTracker
from app.billing import routes as billing_routes

# Configure structured logging
structlog.configure(
    processors=[
        structlog.processors.TimeStamper(fmt="ISO"),
        structlog.processors.JSONRenderer()
    ],
    wrapper_class=structlog.stdlib.BoundLogger,
    logger_factory=structlog.stdlib.LoggerFactory(),
    cache_logger_on_first_use=True,
)

logger = structlog.get_logger(__name__)


class AILBServer:
    """AI Load Balancer Server"""

    def __init__(self):
        self.connectors: Dict[str, Any] = {}
        self.request_router = None
        self.memory_manager = None
        self.rag_manager = None
        self.grpc_server_task = None

        # Key management, rate limiting, and billing
        self.key_manager = None
        self.rate_limiter = None
        self.cost_tracker = None

        # Configuration from environment
        self.config = {
            'grpc_port': int(os.getenv('GRPC_PORT', '50051')),
            'http_port': int(os.getenv('HTTP_PORT', '8080')),
            'default_routing_strategy': os.getenv('ROUTING_STRATEGY', 'load_balanced'),
            'enable_memory': os.getenv('ENABLE_MEMORY', 'true').lower() == 'true',
            'enable_rag': os.getenv('ENABLE_RAG', 'false').lower() == 'true',
            'memory_backend': os.getenv('MEMORY_BACKEND', 'chromadb'),
            'rag_backend': os.getenv('RAG_BACKEND', 'chromadb'),
        }

    async def startup(self):
        """Initialize server components"""
        logger.info("Starting AILB Server")

        # Initialize key manager
        self.key_manager = KeyManager()
        logger.info("Key manager initialized")

        # Initialize rate limiter
        self.rate_limiter = RateLimiter()
        logger.info("Rate limiter initialized")

        # Initialize cost tracker
        self.cost_tracker = CostTracker()
        logger.info("Cost tracker initialized")

        # Initialize LLM provider connectors
        await self._init_connectors()

        # Initialize request router
        self.request_router = LLMRequestRouter(
            connectors=self.connectors,
            default_strategy=RoutingStrategy(self.config['default_routing_strategy'])
        )

        # Initialize memory manager if enabled
        if self.config['enable_memory']:
            self.memory_manager = ConversationMemoryManager(
                backend=self.config['memory_backend']
            )
            await self.memory_manager.initialize()
            logger.info("Memory manager initialized")

        # Initialize RAG manager if enabled
        if self.config['enable_rag']:
            self.rag_manager = RAGManager(
                backend=self.config['rag_backend']
            )
            await self.rag_manager.initialize()
            logger.info("RAG manager initialized")

        # Start gRPC server in background
        self.grpc_server_task = asyncio.create_task(
            start_grpc_server(self, self.config['grpc_port'])
        )

        logger.info("AILB server initialized successfully")

    async def _init_connectors(self):
        """Initialize LLM provider connectors from environment"""
        # OpenAI connector
        if os.getenv('OPENAI_API_KEY'):
            self.connectors['openai'] = OpenAIConnector(
                api_key=os.getenv('OPENAI_API_KEY'),
                base_url=os.getenv('OPENAI_BASE_URL'),
                models=os.getenv('OPENAI_MODELS', 'gpt-4,gpt-3.5-turbo').split(',')
            )
            logger.info("OpenAI connector initialized")

        # Anthropic connector
        if os.getenv('ANTHROPIC_API_KEY'):
            self.connectors['anthropic'] = AnthropicConnector(
                api_key=os.getenv('ANTHROPIC_API_KEY'),
                models=os.getenv('ANTHROPIC_MODELS', 'claude-3-opus-20240229,claude-3-sonnet-20240229').split(',')
            )
            logger.info("Anthropic connector initialized")

        # Ollama connector
        if os.getenv('OLLAMA_BASE_URL'):
            self.connectors['ollama'] = OllamaConnector(
                base_url=os.getenv('OLLAMA_BASE_URL', 'http://localhost:11434')
            )
            # Auto-discover available models
            await self.connectors['ollama'].discover_models()
            logger.info("Ollama connector initialized")

    async def shutdown(self):
        """Cleanup server components"""
        if self.grpc_server_task:
            self.grpc_server_task.cancel()
            try:
                await self.grpc_server_task
            except asyncio.CancelledError:
                pass

        # Close all connectors
        for connector in self.connectors.values():
            if hasattr(connector, 'close'):
                await connector.close()

        logger.info("AILB server shutdown complete")


# Global server instance
ailb_server = AILBServer()


@asynccontextmanager
async def lifespan(app: FastAPI):
    """FastAPI lifespan context manager"""
    await ailb_server.startup()

    # Add rate limiting middleware after server startup
    app.add_middleware(
        RateLimitMiddleware,
        limiter=ailb_server.rate_limiter,
        exempt_paths=["/healthz", "/metrics", "/docs", "/openapi.json", "/redoc"]
    )

    # Include routers
    app.include_router(keys_routes.router)
    app.include_router(billing_routes.router)

    # Set global instances for route dependencies
    keys_routes._key_manager = ailb_server.key_manager
    billing_routes._cost_tracker = ailb_server.cost_tracker

    yield
    await ailb_server.shutdown()


# FastAPI app
app = FastAPI(
    title="AILB - AI Load Balancer",
    description="Intelligent AI/LLM proxy with routing, memory, and RAG support",
    version="1.0.0",
    lifespan=lifespan
)

# CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Rate limiting middleware
# Note: Added after startup when rate_limiter is initialized
# This is handled by adding it in the lifespan startup hook


# Health check endpoints
@app.get("/healthz")
async def health_check():
    """Kubernetes-style health check"""
    return {"status": "healthy"}


@app.get("/metrics")
async def prometheus_metrics():
    """Prometheus metrics endpoint"""
    # TODO: Implement metrics collection
    return Response(
        content="# AILB metrics\n",
        media_type=CONTENT_TYPE_LATEST
    )


# OpenAI Compatible API Endpoints
@app.post("/v1/chat/completions")
async def chat_completions(
    request: Request,
    authorization: Optional[str] = Header(None),
    x_preferred_model: Optional[str] = Header(None, alias="X-Preferred-Model")
):
    """OpenAI-compatible chat completions endpoint"""
    start_time = time.time()

    try:
        # Validate API key
        api_key = None
        key_id = None
        if authorization:
            api_key = authorization[7:] if authorization.startswith("Bearer ") else authorization
            validation_result = ailb_server.key_manager.validate_key(api_key)

            if not validation_result.valid:
                raise HTTPException(
                    status_code=401,
                    detail=f"Invalid API key: {validation_result.error}"
                )

            key_id = validation_result.key_id
            logger.info(f"Request authenticated with key: {key_id}")

        # Parse request
        body = await request.json()
        messages = body.get("messages", [])
        model = body.get("model") or x_preferred_model or "gpt-3.5-turbo"

        # Check budget before request (if key is provided)
        if key_id:
            # Estimate cost based on typical request size (conservative estimate)
            estimated_tokens = sum(len(str(m.get('content', ''))) // 4 for m in messages) + 500
            estimated_cost = ailb_server.cost_tracker.calculate_cost(
                model=model,
                input_tokens=estimated_tokens // 2,
                output_tokens=estimated_tokens // 2
            )

            can_proceed = ailb_server.cost_tracker.check_budget(key_id, estimated_cost)
            if not can_proceed:
                raise HTTPException(
                    status_code=402,
                    detail="Budget exceeded. Request would exceed spending limit."
                )

        # Get session ID for memory
        session_id = body.get('session_id') or request.headers.get('X-Session-ID')

        # Enhance messages with memory if enabled
        if ailb_server.memory_manager and session_id:
            memory_context = await ailb_server.memory_manager.get_context(
                session_id=session_id,
                current_messages=messages
            )
            messages = await ailb_server.memory_manager.enhance_messages(
                messages=messages,
                context=memory_context
            )

        # Enhance with RAG if enabled
        if ailb_server.rag_manager:
            rag_context = await ailb_server.rag_manager.get_context(
                messages=messages,
                collection=body.get('rag_collection', 'default'),
                top_k=body.get('rag_top_k', 3)
            )
            messages = await ailb_server.rag_manager.enhance_messages(
                messages=messages,
                context=rag_context
            )

        # Route request to appropriate LLM provider
        response_text, usage_info = await ailb_server.request_router.route_request(
            model=model,
            messages=messages,
            **{k: v for k, v in body.items() if k not in ['messages', 'model', 'session_id']}
        )

        # Get actual token usage
        input_tokens = usage_info.get('input_tokens', 0)
        output_tokens = usage_info.get('output_tokens', 0)
        total_tokens = usage_info.get('total_tokens', input_tokens + output_tokens)

        # Record usage and cost (if key is provided)
        if key_id:
            try:
                # Determine provider from model
                provider = "unknown"
                if "gpt" in model.lower():
                    provider = "openai"
                elif "claude" in model.lower():
                    provider = "anthropic"
                elif "ollama" in model.lower():
                    provider = "ollama"

                # Record usage in cost tracker
                ailb_server.cost_tracker.record_usage(
                    key_id=key_id,
                    model=model,
                    provider=provider,
                    input_tokens=input_tokens,
                    output_tokens=output_tokens,
                    request_id=f"chatcmpl-{int(time.time())}",
                    session_id=session_id
                )

                # Record usage in key manager
                cost = ailb_server.cost_tracker.calculate_cost(
                    model=model,
                    input_tokens=input_tokens,
                    output_tokens=output_tokens
                )
                ailb_server.key_manager.record_usage(
                    key_id=key_id,
                    tokens=total_tokens,
                    cost=cost,
                    model=model,
                    provider=provider
                )

                logger.info(
                    f"Recorded usage: key={key_id}, model={model}, "
                    f"tokens={total_tokens}, cost=${cost:.4f}"
                )

            except Exception as e:
                logger.error(f"Failed to record usage: {str(e)}")
                # Don't fail the request if usage recording fails

        # Store conversation in memory if enabled
        if ailb_server.memory_manager and session_id:
            asyncio.create_task(ailb_server.memory_manager.store_turn(
                session_id=session_id,
                messages=messages,
                response=response_text,
                metadata=usage_info
            ))

        # Return OpenAI-compatible response
        return {
            "id": f"chatcmpl-{int(time.time())}",
            "object": "chat.completion",
            "created": int(time.time()),
            "model": model,
            "choices": [{
                "index": 0,
                "message": {
                    "role": "assistant",
                    "content": response_text
                },
                "finish_reason": "stop"
            }],
            "usage": {
                "prompt_tokens": input_tokens,
                "completion_tokens": output_tokens,
                "total_tokens": total_tokens
            }
        }

    except Exception as e:
        logger.error("Chat completion failed", error=str(e))
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/v1/models")
async def list_models():
    """List available models"""
    try:
        all_models = []
        for provider_name, connector in ailb_server.connectors.items():
            models = await connector.list_models()
            all_models.extend(models)

        return {
            "object": "list",
            "data": all_models
        }
    except Exception as e:
        logger.error("Failed to list models", error=str(e))
        raise HTTPException(status_code=500, detail="Failed to list models")


@app.get("/api/routing/stats")
async def get_routing_stats():
    """Get LLM provider routing statistics"""
    try:
        stats = ailb_server.request_router.get_stats()
        return {
            "routing_strategy": ailb_server.request_router.default_strategy.value,
            "provider_stats": stats
        }
    except Exception as e:
        logger.error(f"Failed to get routing stats: {e}")
        raise HTTPException(status_code=500, detail="Failed to get routing stats")


if __name__ == "__main__":
    # Development server
    uvicorn.run(
        "main:app",
        host=os.getenv("HOST", "0.0.0.0"),
        port=int(os.getenv("HTTP_PORT", "8080")),
        reload=True,
        log_level="info"
    )
