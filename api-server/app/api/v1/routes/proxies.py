"""
Proxy registration and heartbeat API routes

Handles proxy server registration, heartbeat, configuration fetch, and metrics reporting.
"""

import hashlib
import logging
from datetime import datetime
from typing import Annotated

from fastapi import APIRouter, Depends, HTTPException, status, Query, Header
from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.database import get_db
from app.dependencies import get_current_user
from app.models.sqlalchemy.user import User
from app.models.sqlalchemy.proxy import ProxyServer, ProxyMetrics
from app.models.sqlalchemy.cluster import Cluster
from app.schemas.proxy import (
    ProxyRegisterRequest,
    ProxyHeartbeatRequest,
    ProxyResponse,
    ProxyListResponse,
    ProxyConfigResponse,
    ProxyMetricsRequest,
)

router = APIRouter(prefix="/proxies", tags=["proxies"])
logger = logging.getLogger(__name__)


def hash_api_key(api_key: str) -> str:
    """Hash API key for verification"""
    return hashlib.sha256(api_key.encode()).hexdigest()


async def verify_cluster_api_key(api_key: str, db: AsyncSession) -> Cluster:
    """Verify cluster API key and return cluster"""
    api_key_hash = hash_api_key(api_key)
    stmt = select(Cluster).where(
        Cluster.api_key_hash == api_key_hash,
        Cluster.is_active == True
    )
    cluster = (await db.execute(stmt)).scalar_one_or_none()
    if not cluster:
        raise HTTPException(status.HTTP_401_UNAUTHORIZED, "Invalid cluster API key")
    return cluster


@router.post("/register", response_model=ProxyResponse, status_code=status.HTTP_201_CREATED)
async def register_proxy(
    proxy_data: ProxyRegisterRequest,
    db: Annotated[AsyncSession, Depends(get_db)]
):
    """
    Register a new proxy server with cluster API key authentication.
    Used by proxy containers on startup.
    """
    # Verify cluster API key
    cluster = await verify_cluster_api_key(proxy_data.cluster_api_key, db)

    # Check proxy count limit
    proxy_count_stmt = select(func.count()).select_from(ProxyServer).where(
        ProxyServer.cluster_id == cluster.id,
        ProxyServer.status != "offline"
    )
    proxy_count = (await db.execute(proxy_count_stmt)).scalar() or 0

    if proxy_count >= cluster.max_proxies:
        raise HTTPException(
            status.HTTP_429_TOO_MANY_REQUESTS,
            f"Cluster proxy limit reached ({cluster.max_proxies}). Upgrade license for more proxies."
        )

    # Check if proxy name already exists in this cluster
    existing_stmt = select(ProxyServer).where(
        ProxyServer.name == proxy_data.name,
        ProxyServer.cluster_id == cluster.id
    )
    existing = (await db.execute(existing_stmt)).scalar_one_or_none()

    if existing:
        # Update existing proxy registration
        existing.hostname = proxy_data.hostname
        existing.ip_address = proxy_data.ip_address
        existing.port = proxy_data.port
        existing.version = proxy_data.version
        existing.capabilities = proxy_data.capabilities
        existing.status = "active"
        existing.last_seen = datetime.utcnow()
        await db.commit()
        await db.refresh(existing)
        proxy = existing
        logger.info(f"Proxy re-registered: {proxy.name} in cluster {cluster.name}")
    else:
        # Create new proxy
        new_proxy = ProxyServer(
            name=proxy_data.name,
            hostname=proxy_data.hostname,
            ip_address=proxy_data.ip_address,
            port=proxy_data.port,
            cluster_id=cluster.id,
            version=proxy_data.version,
            capabilities=proxy_data.capabilities,
            status="active",
            registered_at=datetime.utcnow(),
            last_seen=datetime.utcnow()
        )
        db.add(new_proxy)
        await db.commit()
        await db.refresh(new_proxy)
        proxy = new_proxy
        logger.info(f"Proxy registered: {proxy.name} in cluster {cluster.name}")

    return ProxyResponse.model_validate(proxy)


@router.post("/heartbeat")
async def proxy_heartbeat(
    heartbeat: ProxyHeartbeatRequest,
    db: Annotated[AsyncSession, Depends(get_db)]
):
    """
    Proxy heartbeat to report status and optionally metrics.
    Should be called every 30-60 seconds by proxies.
    """
    # Verify cluster API key
    await verify_cluster_api_key(heartbeat.cluster_api_key, db)

    # Get proxy
    stmt = select(ProxyServer).where(ProxyServer.id == heartbeat.proxy_id)
    proxy = (await db.execute(stmt)).scalar_one_or_none()
    if not proxy:
        raise HTTPException(status.HTTP_404_NOT_FOUND, "Proxy not found")

    # Update heartbeat
    proxy.status = heartbeat.status
    proxy.last_seen = datetime.utcnow()
    if heartbeat.config_version:
        proxy.config_version = heartbeat.config_version

    # Store metrics if provided
    if heartbeat.metrics:
        metrics = ProxyMetrics(
            proxy_id=proxy.id,
            timestamp=datetime.utcnow(),
            **heartbeat.metrics
        )
        db.add(metrics)

    await db.commit()
    return {"status": "ok", "message": "Heartbeat received"}


@router.get("/config", response_model=ProxyConfigResponse)
async def get_proxy_config(
    cluster_api_key: Annotated[str, Header()],
    db: Annotated[AsyncSession, Depends(get_db)]
):
    """
    Get configuration for all proxies in a cluster.
    Returns services, mappings, logging config, etc.
    """
    cluster = await verify_cluster_api_key(cluster_api_key, db)

    # TODO: Implement full config generation (Phase 2.1)
    # For now, return basic structure
    config = {
        "config_version": hashlib.md5(str(datetime.utcnow()).encode()).hexdigest(),
        "cluster": {
            "id": cluster.id,
            "name": cluster.name,
            "syslog_endpoint": cluster.syslog_endpoint,
            "log_auth": cluster.log_auth,
            "log_netflow": cluster.log_netflow,
            "log_debug": cluster.log_debug,
        },
        "services": [],  # TODO: Fetch from database
        "mappings": [],  # TODO: Fetch from database
        "certificates": None,  # TODO: Fetch if needed
        "logging": {
            "endpoint": cluster.syslog_endpoint,
            "auth": cluster.log_auth,
            "netflow": cluster.log_netflow,
            "debug": cluster.log_debug,
        }
    }

    return config


@router.get("", response_model=ProxyListResponse)
async def list_proxies(
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(get_current_user)],
    cluster_id: int | None = Query(None),
    skip: int = Query(0, ge=0),
    limit: int = Query(100, ge=1, le=1000)
):
    """List registered proxies (admin and service owners)"""
    query = select(ProxyServer)
    if cluster_id:
        query = query.where(ProxyServer.cluster_id == cluster_id)

    # TODO: Add cluster access control for non-admin users

    count_query = select(func.count()).select_from(query.subquery())
    total = (await db.execute(count_query)).scalar() or 0

    query = query.offset(skip).limit(limit).order_by(ProxyServer.registered_at.desc())
    proxies = (await db.execute(query)).scalars().all()

    return ProxyListResponse(
        total=total,
        proxies=[ProxyResponse.model_validate(p) for p in proxies]
    )


@router.get("/{proxy_id}", response_model=ProxyResponse)
async def get_proxy(
    proxy_id: int,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(get_current_user)]
):
    """Get proxy details"""
    proxy = (await db.execute(select(ProxyServer).where(ProxyServer.id == proxy_id))).scalar_one_or_none()
    if not proxy:
        raise HTTPException(status.HTTP_404_NOT_FOUND, "Proxy not found")
    return ProxyResponse.model_validate(proxy)


@router.post("/{proxy_id}/metrics")
async def report_metrics(
    proxy_id: int,
    metrics: ProxyMetricsRequest,
    db: Annotated[AsyncSession, Depends(get_db)]
):
    """Report detailed proxy metrics"""
    proxy_metrics = ProxyMetrics(
        proxy_id=proxy_id,
        timestamp=datetime.utcnow(),
        **metrics.model_dump(exclude={'proxy_id'})
    )
    db.add(proxy_metrics)
    await db.commit()
    return {"status": "ok", "message": "Metrics recorded"}
