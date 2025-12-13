"""
MarchProxy API Server - FastAPI Application

Main entry point for the FastAPI application.
"""

import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from prometheus_client import make_asgi_app

from app.core.config import settings
from app.core.database import engine, Base, close_db

# Configure logging
logging.basicConfig(
    level=logging.DEBUG if settings.DEBUG else logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Lifespan context manager for startup/shutdown events"""
    # Startup
    logger.info("Starting MarchProxy API Server v%s", settings.APP_VERSION)
    logger.info("Environment: %s", "development" if settings.DEBUG else "production")
    logger.info("Release mode: %s", settings.RELEASE_MODE)

    # Create database tables
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
        logger.info("Database tables created/verified")

    # Initialize xDS bridge (Phase 3 - optional)
    try:
        from app.services.xds_bridge import get_xds_bridge
        xds_bridge = get_xds_bridge(settings.XDS_SERVER_URL)
        is_healthy = await xds_bridge.health_check()
        if is_healthy:
            logger.info("✓ xDS server is healthy")
        else:
            logger.warning("⚠ xDS server is not responding (may start later)")
    except ImportError:
        logger.info("xDS bridge not available (Phase 3 feature)")
        xds_bridge = None

    yield

    # Shutdown
    logger.info("Shutting down MarchProxy API Server")
    if xds_bridge:
        await xds_bridge.close()
    await close_db()


# Create FastAPI application
app = FastAPI(
    title=settings.APP_NAME,
    version=settings.APP_VERSION,
    description="Enterprise-grade egress proxy management",
    lifespan=lifespan,
    docs_url="/api/docs",
    redoc_url="/api/redoc",
    openapi_url="/api/openapi.json"
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=settings.CORS_ORIGINS,
    allow_credentials=settings.CORS_ALLOW_CREDENTIALS,
    allow_methods=settings.CORS_ALLOW_METHODS,
    allow_headers=settings.CORS_ALLOW_HEADERS,
)

# Mount Prometheus metrics
metrics_app = make_asgi_app()
app.mount("/metrics", metrics_app)


@app.get("/healthz")
async def health_check():
    """Health check endpoint"""
    return {
        "status": "healthy",
        "version": settings.APP_VERSION,
        "service": "marchproxy-api-server"
    }


@app.get("/")
async def root():
    """Root endpoint"""
    return {
        "service": settings.APP_NAME,
        "version": settings.APP_VERSION,
        "docs": "/api/docs",
        "health": "/healthz",
        "metrics": "/metrics"
    }


# Mount API v1 router (includes all Phase 2 routes)
from app.api.v1 import api_router
app.include_router(api_router, prefix="/api/v1")

logger.info("API routes mounted successfully")


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(
        "app.main:app",
        host=settings.HOST,
        port=settings.PORT,
        reload=settings.DEBUG,
        workers=1 if settings.DEBUG else settings.WORKERS
    )
