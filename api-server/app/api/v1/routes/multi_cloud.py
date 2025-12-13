"""
Multi-Cloud Intelligent Routing API

Enterprise-only feature for cloud-aware routing with health monitoring,
cost optimization, and automatic failover.
"""

import logging
from typing import List, Optional
from datetime import datetime, timedelta

from fastapi import APIRouter, HTTPException, Depends, status
from sqlalchemy import select, and_
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.database import get_db
from app.core.license import license_validator
from app.models.sqlalchemy.enterprise import RouteTable, RouteHealthStatus as RouteHealthModel
from app.schemas.multi_cloud import (
    RouteTableCreate,
    RouteTableUpdate,
    RouteTableResponse,
    RouteHealthStatus
)

router = APIRouter()
logger = logging.getLogger(__name__)

FEATURE_NAME = "multi_cloud_routing"


async def check_enterprise_license():
    """Check if multi-cloud routing feature is available."""
    has_feature = await license_validator.check_feature(FEATURE_NAME)
    if not has_feature:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail={
                "error": "Enterprise feature not available",
                "feature": FEATURE_NAME,
                "message": "Multi-cloud routing requires an Enterprise license",
                "upgrade_url": "https://www.penguintech.io/marchproxy/pricing"
            }
        )


def route_to_response(route: RouteTable, health_data: List = None) -> RouteTableResponse:
    """Convert database model to response schema."""
    return RouteTableResponse(
        id=route.id,
        name=route.name,
        description=route.description,
        service_id=route.service_id,
        cluster_id=route.cluster_id,
        algorithm=route.algorithm,
        routes=route.routes,
        health_probe_config=route.health_probe_config,
        enable_auto_failover=route.enable_auto_failover,
        enabled=route.enabled,
        created_at=route.created_at,
        updated_at=route.updated_at,
        health_status=health_data
    )


def health_to_response(health: RouteHealthModel) -> RouteHealthStatus:
    """Convert health model to response schema."""
    return RouteHealthStatus(
        endpoint=health.endpoint,
        is_healthy=health.is_healthy,
        last_check=health.last_check,
        rtt_ms=health.rtt_ms,
        consecutive_failures=health.consecutive_failures,
        consecutive_successes=health.consecutive_successes,
        last_error=health.last_error
    )


@router.get("/routes", response_model=List[RouteTableResponse])
async def list_route_tables(
    cluster_id: Optional[int] = None,
    service_id: Optional[int] = None,
    enabled_only: bool = False,
    db: AsyncSession = Depends(get_db),
    _: None = Depends(check_enterprise_license)
):
    """List all route tables with optional filters."""
    query = select(RouteTable)

    conditions = []
    if cluster_id is not None:
        conditions.append(RouteTable.cluster_id == cluster_id)
    if service_id is not None:
        conditions.append(RouteTable.service_id == service_id)
    if enabled_only:
        conditions.append(RouteTable.enabled == True)

    if conditions:
        query = query.where(and_(*conditions))

    query = query.order_by(RouteTable.created_at.desc())

    result = await db.execute(query)
    routes = result.scalars().all()

    return [route_to_response(r) for r in routes]


@router.post("/routes", response_model=RouteTableResponse, status_code=status.HTTP_201_CREATED)
async def create_route_table(
    route_table: RouteTableCreate,
    db: AsyncSession = Depends(get_db),
    _: None = Depends(check_enterprise_license)
):
    """Create a new route table."""
    db_route = RouteTable(
        name=route_table.name,
        description=route_table.description,
        service_id=route_table.service_id,
        cluster_id=route_table.cluster_id,
        algorithm=route_table.algorithm or 'latency',
        routes=route_table.routes or [],
        health_probe_config=route_table.health_probe_config.model_dump() if route_table.health_probe_config else {
            "protocol": "tcp",
            "port": None,
            "path": None,
            "interval_seconds": 30,
            "timeout_seconds": 5,
            "unhealthy_threshold": 3,
            "healthy_threshold": 2
        },
        enable_auto_failover=route_table.enable_auto_failover if route_table.enable_auto_failover is not None else True,
        enabled=route_table.enabled if route_table.enabled is not None else True,
        created_at=datetime.utcnow(),
        updated_at=datetime.utcnow()
    )

    db.add(db_route)
    await db.commit()
    await db.refresh(db_route)

    logger.info(f"Created route table: {db_route.name} (ID: {db_route.id})")
    return route_to_response(db_route)


@router.get("/routes/{route_id}", response_model=RouteTableResponse)
async def get_route_table(
    route_id: int,
    include_health: bool = True,
    db: AsyncSession = Depends(get_db),
    _: None = Depends(check_enterprise_license)
):
    """Get a specific route table by ID."""
    result = await db.execute(
        select(RouteTable).where(RouteTable.id == route_id)
    )
    route = result.scalar_one_or_none()

    if not route:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Route table {route_id} not found"
        )

    health_data = None
    if include_health:
        health_result = await db.execute(
            select(RouteHealthModel).where(RouteHealthModel.route_table_id == route_id)
        )
        health_records = health_result.scalars().all()
        health_data = [health_to_response(h) for h in health_records]

    return route_to_response(route, health_data)


@router.put("/routes/{route_id}", response_model=RouteTableResponse)
async def update_route_table(
    route_id: int,
    route_update: RouteTableUpdate,
    db: AsyncSession = Depends(get_db),
    _: None = Depends(check_enterprise_license)
):
    """Update an existing route table."""
    result = await db.execute(
        select(RouteTable).where(RouteTable.id == route_id)
    )
    route = result.scalar_one_or_none()

    if not route:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Route table {route_id} not found"
        )

    update_data = route_update.model_dump(exclude_unset=True)

    for field, value in update_data.items():
        if field == "health_probe_config" and value is not None:
            setattr(route, field, value.model_dump() if hasattr(value, 'model_dump') else value)
        else:
            setattr(route, field, value)

    route.updated_at = datetime.utcnow()

    await db.commit()
    await db.refresh(route)

    logger.info(f"Updated route table: {route.name} (ID: {route.id})")
    return route_to_response(route)


@router.delete("/routes/{route_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_route_table(
    route_id: int,
    db: AsyncSession = Depends(get_db),
    _: None = Depends(check_enterprise_license)
):
    """Delete a route table and associated health records."""
    result = await db.execute(
        select(RouteTable).where(RouteTable.id == route_id)
    )
    route = result.scalar_one_or_none()

    if not route:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Route table {route_id} not found"
        )

    # Delete associated health records
    health_result = await db.execute(
        select(RouteHealthModel).where(RouteHealthModel.route_table_id == route_id)
    )
    for health in health_result.scalars().all():
        await db.delete(health)

    await db.delete(route)
    await db.commit()

    logger.info(f"Deleted route table: {route_id}")


@router.get("/routes/{route_id}/health", response_model=List[RouteHealthStatus])
async def get_route_health(
    route_id: int,
    db: AsyncSession = Depends(get_db),
    _: None = Depends(check_enterprise_license)
):
    """Get real-time health status for all routes in a table."""
    # Verify route exists
    route_result = await db.execute(
        select(RouteTable).where(RouteTable.id == route_id)
    )
    if not route_result.scalar_one_or_none():
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Route table {route_id} not found"
        )

    result = await db.execute(
        select(RouteHealthModel)
        .where(RouteHealthModel.route_table_id == route_id)
        .order_by(RouteHealthModel.last_check.desc())
    )
    health_records = result.scalars().all()

    return [health_to_response(h) for h in health_records]


@router.post("/routes/{route_id}/enable", response_model=RouteTableResponse)
async def enable_route_table(
    route_id: int,
    db: AsyncSession = Depends(get_db),
    _: None = Depends(check_enterprise_license)
):
    """Enable a route table."""
    result = await db.execute(
        select(RouteTable).where(RouteTable.id == route_id)
    )
    route = result.scalar_one_or_none()

    if not route:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Route table {route_id} not found"
        )

    route.enabled = True
    route.updated_at = datetime.utcnow()

    await db.commit()
    await db.refresh(route)

    logger.info(f"Enabled route table: {route.name} (ID: {route.id})")
    return route_to_response(route)


@router.post("/routes/{route_id}/disable", response_model=RouteTableResponse)
async def disable_route_table(
    route_id: int,
    db: AsyncSession = Depends(get_db),
    _: None = Depends(check_enterprise_license)
):
    """Disable a route table."""
    result = await db.execute(
        select(RouteTable).where(RouteTable.id == route_id)
    )
    route = result.scalar_one_or_none()

    if not route:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Route table {route_id} not found"
        )

    route.enabled = False
    route.updated_at = datetime.utcnow()

    await db.commit()
    await db.refresh(route)

    logger.info(f"Disabled route table: {route.name} (ID: {route.id})")
    return route_to_response(route)


@router.post("/routes/{route_id}/test-failover", response_model=dict)
async def test_failover(
    route_id: int,
    simulate_failure_endpoint: Optional[str] = None,
    db: AsyncSession = Depends(get_db),
    _: None = Depends(check_enterprise_license)
):
    """Test failover behavior for a route table."""
    result = await db.execute(
        select(RouteTable).where(RouteTable.id == route_id)
    )
    route = result.scalar_one_or_none()

    if not route:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Route table {route_id} not found"
        )

    # Get current health status
    health_result = await db.execute(
        select(RouteHealthModel).where(RouteHealthModel.route_table_id == route_id)
    )
    health_records = health_result.scalars().all()

    healthy_endpoints = [h.endpoint for h in health_records if h.is_healthy]
    unhealthy_endpoints = [h.endpoint for h in health_records if not h.is_healthy]

    # Simulate failover
    if simulate_failure_endpoint:
        if simulate_failure_endpoint in healthy_endpoints:
            healthy_endpoints.remove(simulate_failure_endpoint)
            unhealthy_endpoints.append(simulate_failure_endpoint)

    failover_target = healthy_endpoints[0] if healthy_endpoints else None

    return {
        "route_id": route_id,
        "route_name": route.name,
        "algorithm": route.algorithm,
        "auto_failover_enabled": route.enable_auto_failover,
        "simulated_failure": simulate_failure_endpoint,
        "healthy_endpoints": healthy_endpoints,
        "unhealthy_endpoints": unhealthy_endpoints,
        "failover_target": failover_target,
        "test_status": "success" if failover_target else "no_healthy_endpoints",
        "message": f"Failover would route to: {failover_target}" if failover_target else "No healthy endpoints available"
    }


@router.get("/analytics/cost", response_model=dict)
async def get_cost_analytics(
    cluster_id: Optional[int] = None,
    service_id: Optional[int] = None,
    route_id: Optional[int] = None,
    days: int = 7,
    db: AsyncSession = Depends(get_db),
    _: None = Depends(check_enterprise_license)
):
    """Get cost analytics for multi-cloud routing."""
    # Query route tables based on filters
    query = select(RouteTable)
    conditions = []

    if cluster_id is not None:
        conditions.append(RouteTable.cluster_id == cluster_id)
    if service_id is not None:
        conditions.append(RouteTable.service_id == service_id)
    if route_id is not None:
        conditions.append(RouteTable.id == route_id)

    if conditions:
        query = query.where(and_(*conditions))

    result = await db.execute(query)
    routes = result.scalars().all()

    # Aggregate cost data (in real implementation, this would query metrics/telemetry)
    by_provider = {"aws": 0.0, "gcp": 0.0, "azure": 0.0, "other": 0.0}
    by_region = {}
    total_cost = 0.0

    for route in routes:
        for endpoint in route.routes:
            provider = endpoint.get("provider", "other").lower()
            region = endpoint.get("region", "unknown")

            # Simulated cost calculation based on route configuration
            estimated_cost = 0.01 * days  # $0.01 per endpoint per day baseline

            if provider in by_provider:
                by_provider[provider] += estimated_cost
            else:
                by_provider["other"] += estimated_cost

            by_region[region] = by_region.get(region, 0.0) + estimated_cost
            total_cost += estimated_cost

    return {
        "period_days": days,
        "period_start": (datetime.utcnow() - timedelta(days=days)).isoformat(),
        "period_end": datetime.utcnow().isoformat(),
        "total_cost_usd": round(total_cost, 2),
        "by_provider": {k: round(v, 2) for k, v in by_provider.items() if v > 0},
        "by_region": {k: round(v, 2) for k, v in by_region.items()},
        "route_count": len(routes),
        "currency": "USD"
    }
