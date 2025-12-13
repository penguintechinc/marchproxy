"""
API v1 routes

Phase 2: Core CRUD operations for clusters, services, proxies, and users.
Phase 3+: Enterprise features (traffic shaping, multi-cloud, observability, xDS).
"""

from fastapi import APIRouter

# Phase 2: Core routes
from app.api.v1.routes import auth, clusters, services, proxies, users, config, certificates

# Create API router
api_router = APIRouter()

# Include Phase 2 core routes
api_router.include_router(auth.router, tags=["Authentication"])
api_router.include_router(clusters.router, tags=["Clusters"])
api_router.include_router(services.router, tags=["Services"])
api_router.include_router(proxies.router, tags=["Proxies"])
api_router.include_router(users.router, tags=["Users"])
api_router.include_router(config.router, tags=["Configuration"])
api_router.include_router(certificates.router, tags=["Certificates"])

# Phase 3+: Enterprise feature routes (optional, will fail gracefully if not available)
try:
    from app.api.v1.routes import traffic_shaping, multi_cloud, observability

    api_router.include_router(
        traffic_shaping.router,
        prefix="/traffic-shaping",
        tags=["Enterprise - Traffic Shaping"]
    )
    api_router.include_router(
        multi_cloud.router,
        prefix="/multi-cloud",
        tags=["Enterprise - Multi-Cloud Routing"]
    )
    api_router.include_router(
        observability.router,
        prefix="/observability",
        tags=["Enterprise - Observability"]
    )
except ImportError:
    # Phase 3 routes not yet implemented
    pass

__all__ = ["api_router"]
