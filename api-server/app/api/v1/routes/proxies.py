"""
Proxy registration and heartbeat API routes

Handles proxy server registration, heartbeat, configuration fetch, and metrics reporting.
"""

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
from app.schemas.proxy import (
    ProxyRegisterRequest,
    ProxyHeartbeatRequest,
    ProxyResponse,
    ProxyListResponse,
    ProxyMetricsRequest,
)
from app.services.proxy_service import (
    ProxyService,
    InvalidAPIKeyError,
    ProxyLimitExceededError,
)

router = APIRouter(prefix="/proxies", tags=["proxies"])
logger = logging.getLogger(__name__)


@router.post("/register", response_model=ProxyResponse, status_code=status.HTTP_201_CREATED)
async def register_proxy(
    proxy_data: ProxyRegisterRequest,
    db: Annotated[AsyncSession, Depends(get_db)]
):
    """
    Register a new proxy server with cluster API key authentication.
    Used by proxy containers on startup.
    """
    service = ProxyService(db)

    try:
        # Verify cluster API key
        cluster = await service.verify_cluster_api_key(proxy_data.cluster_api_key)

        # Register proxy (handles limit checks and re-registration)
        proxy = await service.register_proxy(
            cluster=cluster,
            name=proxy_data.name,
            hostname=proxy_data.hostname,
            ip_address=proxy_data.ip_address,
            port=proxy_data.port,
            version=proxy_data.version,
            capabilities=proxy_data.capabilities
        )

        return ProxyResponse.model_validate(proxy)

    except InvalidAPIKeyError:
        raise HTTPException(
            status.HTTP_401_UNAUTHORIZED,
            "Invalid cluster API key"
        )
    except ProxyLimitExceededError as e:
        raise HTTPException(status.HTTP_429_TOO_MANY_REQUESTS, str(e))


@router.post("/heartbeat")
async def proxy_heartbeat(
    heartbeat: ProxyHeartbeatRequest,
    db: Annotated[AsyncSession, Depends(get_db)]
):
    """
    Proxy heartbeat to report status and optionally metrics.
    Should be called every 30-60 seconds by proxies.
    """
    service = ProxyService(db)

    try:
        # Verify cluster API key
        await service.verify_cluster_api_key(heartbeat.cluster_api_key)

        # Update heartbeat
        await service.update_heartbeat(
            proxy_id=heartbeat.proxy_id,
            status=heartbeat.status,
            config_version=heartbeat.config_version,
            metrics=heartbeat.metrics
        )

        return {"status": "ok", "message": "Heartbeat received"}

    except InvalidAPIKeyError:
        raise HTTPException(
            status.HTTP_401_UNAUTHORIZED,
            "Invalid cluster API key"
        )
    except ValueError as e:
        raise HTTPException(status.HTTP_404_NOT_FOUND, str(e))




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

    # Apply cluster access control for non-admin users
    if not current_user.is_admin:
        # Get clusters the user has access to via their cluster assignments
        from app.models.sqlalchemy.cluster import UserClusterAccess
        user_clusters_query = select(UserClusterAccess.cluster_id).where(
            UserClusterAccess.user_id == current_user.id
        )
        user_clusters_result = await db.execute(user_clusters_query)
        allowed_cluster_ids = [row[0] for row in user_clusters_result.fetchall()]

        if allowed_cluster_ids:
            query = query.where(ProxyServer.cluster_id.in_(allowed_cluster_ids))
        else:
            # User has no cluster access - return empty list
            return ProxyListResponse(total=0, proxies=[])

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
    stmt = select(ProxyServer).where(ProxyServer.id == proxy_id)
    proxy = (await db.execute(stmt)).scalar_one_or_none()
    if not proxy:
        raise HTTPException(status.HTTP_404_NOT_FOUND, "Proxy not found")
    return ProxyResponse.model_validate(proxy)


@router.delete("/{proxy_id}", status_code=status.HTTP_204_NO_CONTENT)
async def deregister_proxy(
    proxy_id: int,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(get_current_user)]
):
    """
    Deregister a proxy server

    Sets proxy status to offline. Does not delete the record
    to preserve history and metrics.
    """
    service = ProxyService(db)

    success = await service.deregister_proxy(proxy_id)
    if not success:
        raise HTTPException(status.HTTP_404_NOT_FOUND, "Proxy not found")

    return None


@router.get("/{proxy_id}/metrics")
async def get_proxy_metrics(
    proxy_id: int,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(get_current_user)],
    limit: int = Query(100, ge=1, le=1000, description="Number of metrics to return")
):
    """
    Get recent metrics for a proxy

    Returns the most recent metrics entries.
    """
    stmt = select(ProxyServer).where(ProxyServer.id == proxy_id)
    proxy = (await db.execute(stmt)).scalar_one_or_none()
    if not proxy:
        raise HTTPException(status.HTTP_404_NOT_FOUND, "Proxy not found")

    # Get recent metrics
    metrics_stmt = (
        select(ProxyMetrics)
        .where(ProxyMetrics.proxy_id == proxy_id)
        .order_by(ProxyMetrics.timestamp.desc())
        .limit(limit)
    )
    metrics = (await db.execute(metrics_stmt)).scalars().all()

    return {
        "proxy_id": proxy_id,
        "proxy_name": proxy.name,
        "metrics_count": len(metrics),
        "metrics": [
            {
                "timestamp": m.timestamp.isoformat(),
                "cpu_usage": m.cpu_usage,
                "memory_usage": m.memory_usage,
                "connections_active": m.connections_active,
                "connections_total": m.connections_total,
                "bytes_sent": m.bytes_sent,
                "bytes_received": m.bytes_received,
                "requests_per_second": m.requests_per_second,
                "latency_avg": m.latency_avg,
                "latency_p95": m.latency_p95,
                "errors_per_second": m.errors_per_second,
            }
            for m in metrics
        ]
    }


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
