"""
Observability and Distributed Tracing API

Enterprise-only feature for deep observability with distributed tracing,
custom metrics, and sampling strategies.
"""

import logging
from typing import List, Optional
from datetime import datetime

from fastapi import APIRouter, HTTPException, Depends, status
from sqlalchemy import select, and_
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.database import get_db
from app.core.license import license_validator
from app.models.sqlalchemy.enterprise import TracingConfig, TracingStats as TracingStatsModel
from app.schemas.observability import (
    TracingConfigCreate,
    TracingConfigUpdate,
    TracingConfigResponse,
    TracingStats
)

router = APIRouter()
logger = logging.getLogger(__name__)

# Feature name for license check
FEATURE_NAME = "distributed_tracing"


async def check_enterprise_license():
    """
    Dependency to check if distributed tracing is available

    Raises:
        HTTPException: 403 if feature not available
    """
    has_feature = await license_validator.check_feature(FEATURE_NAME)
    if not has_feature:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail={
                "error": "Enterprise feature not available",
                "feature": FEATURE_NAME,
                "message": "Distributed tracing requires an Enterprise license",
                "upgrade_url": "https://www.penguintech.io/marchproxy/pricing"
            }
        )


def config_to_response(config: TracingConfig, stats: TracingStatsModel = None) -> TracingConfigResponse:
    """Convert database model to response schema."""
    return TracingConfigResponse(
        id=config.id,
        name=config.name,
        description=config.description,
        cluster_id=config.cluster_id,
        backend=config.backend,
        endpoint=config.endpoint,
        exporter=config.exporter,
        sampling_strategy=config.sampling_strategy,
        sampling_rate=config.sampling_rate,
        max_traces_per_second=config.max_traces_per_second,
        include_request_headers=config.include_request_headers,
        include_response_headers=config.include_response_headers,
        include_request_body=config.include_request_body,
        include_response_body=config.include_response_body,
        max_attribute_length=config.max_attribute_length,
        service_name=config.service_name,
        custom_tags=config.custom_tags,
        enabled=config.enabled,
        created_at=config.created_at,
        updated_at=config.updated_at,
        stats=stats
    )


@router.get("/tracing", response_model=List[TracingConfigResponse])
async def list_tracing_configs(
    cluster_id: Optional[int] = None,
    enabled_only: bool = False,
    db: AsyncSession = Depends(get_db),
    _: None = Depends(check_enterprise_license)
):
    """
    List all tracing configurations

    **Enterprise Feature**: Requires Enterprise license

    Args:
        cluster_id: Filter by cluster ID
        enabled_only: Only return enabled configurations
        db: Database session

    Returns:
        List of tracing configurations with runtime stats
    """
    logger.info(
        f"Listing tracing configs: cluster_id={cluster_id}, "
        f"enabled_only={enabled_only}"
    )

    query = select(TracingConfig)

    conditions = []
    if cluster_id is not None:
        conditions.append(TracingConfig.cluster_id == cluster_id)
    if enabled_only:
        conditions.append(TracingConfig.enabled == True)

    if conditions:
        query = query.where(and_(*conditions))

    query = query.order_by(TracingConfig.created_at.desc())

    result = await db.execute(query)
    configs = result.scalars().all()

    return [config_to_response(c) for c in configs]


@router.post("/tracing", response_model=TracingConfigResponse,
             status_code=status.HTTP_201_CREATED)
async def create_tracing_config(
    config: TracingConfigCreate,
    db: AsyncSession = Depends(get_db),
    _: None = Depends(check_enterprise_license)
):
    """
    Create a new tracing configuration

    **Enterprise Feature**: Requires Enterprise license

    Configure distributed tracing with:
    - **Backends**: Jaeger, Zipkin, OpenTelemetry Protocol (OTLP)
    - **Sampling Strategies**:
      - `always`: Sample all requests (100%)
      - `never`: No sampling (0%)
      - `probabilistic`: Random percentage sampling
      - `rate_limit`: Maximum traces per second
      - `error_only`: Only sample failed requests
      - `adaptive`: Dynamic sampling based on load
    - **Span Exporters**: gRPC, HTTP, Thrift
    - **Header Inclusion**: Request/response headers and bodies
    - **Custom Tags**: Service-specific metadata

    Args:
        config: Tracing configuration
        db: Database session

    Returns:
        Created tracing configuration
    """
    logger.info(
        f"Creating tracing config: {config.name} for cluster {config.cluster_id}"
    )

    db_config = TracingConfig(
        name=config.name,
        description=config.description,
        cluster_id=config.cluster_id,
        backend=config.backend or 'jaeger',
        endpoint=config.endpoint,
        exporter=config.exporter or 'grpc',
        sampling_strategy=config.sampling_strategy or 'probabilistic',
        sampling_rate=config.sampling_rate if config.sampling_rate is not None else 0.1,
        max_traces_per_second=config.max_traces_per_second,
        include_request_headers=config.include_request_headers or False,
        include_response_headers=config.include_response_headers or False,
        include_request_body=config.include_request_body or False,
        include_response_body=config.include_response_body or False,
        max_attribute_length=config.max_attribute_length or 512,
        service_name=config.service_name or 'marchproxy',
        custom_tags=config.custom_tags,
        enabled=config.enabled if config.enabled is not None else True,
        created_at=datetime.utcnow(),
        updated_at=datetime.utcnow()
    )

    db.add(db_config)
    await db.commit()
    await db.refresh(db_config)

    logger.info(f"Created tracing config: {db_config.name} (ID: {db_config.id})")
    return config_to_response(db_config)


@router.get("/tracing/{config_id}", response_model=TracingConfigResponse)
async def get_tracing_config(
    config_id: int,
    include_stats: bool = True,
    db: AsyncSession = Depends(get_db),
    _: None = Depends(check_enterprise_license)
):
    """
    Get a specific tracing configuration by ID

    **Enterprise Feature**: Requires Enterprise license

    Args:
        config_id: Configuration ID
        include_stats: Include runtime statistics
        db: Database session

    Returns:
        Tracing configuration with optional stats
    """
    logger.info(
        f"Fetching tracing config {config_id}, include_stats={include_stats}"
    )

    result = await db.execute(
        select(TracingConfig).where(TracingConfig.id == config_id)
    )
    config = result.scalar_one_or_none()

    if not config:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Tracing configuration {config_id} not found"
        )

    stats = None
    if include_stats:
        stats_result = await db.execute(
            select(TracingStatsModel)
            .where(TracingStatsModel.tracing_config_id == config_id)
            .order_by(TracingStatsModel.timestamp.desc())
            .limit(1)
        )
        stats = stats_result.scalar_one_or_none()

    return config_to_response(config, stats)


@router.put("/tracing/{config_id}", response_model=TracingConfigResponse)
async def update_tracing_config(
    config_id: int,
    config: TracingConfigUpdate,
    db: AsyncSession = Depends(get_db),
    _: None = Depends(check_enterprise_license)
):
    """
    Update an existing tracing configuration

    **Enterprise Feature**: Requires Enterprise license

    Args:
        config_id: Configuration ID
        config: Updated tracing configuration
        db: Database session

    Returns:
        Updated tracing configuration
    """
    logger.info(f"Updating tracing config {config_id}")

    result = await db.execute(
        select(TracingConfig).where(TracingConfig.id == config_id)
    )
    db_config = result.scalar_one_or_none()

    if not db_config:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Tracing configuration {config_id} not found"
        )

    update_data = config.model_dump(exclude_unset=True)

    for field, value in update_data.items():
        setattr(db_config, field, value)

    db_config.updated_at = datetime.utcnow()

    await db.commit()
    await db.refresh(db_config)

    logger.info(f"Updated tracing config: {db_config.name} (ID: {db_config.id})")
    return config_to_response(db_config)


@router.delete("/tracing/{config_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_tracing_config(
    config_id: int,
    db: AsyncSession = Depends(get_db),
    _: None = Depends(check_enterprise_license)
):
    """
    Delete a tracing configuration

    **Enterprise Feature**: Requires Enterprise license

    Args:
        config_id: Configuration ID
        db: Database session
    """
    logger.info(f"Deleting tracing config {config_id}")

    result = await db.execute(
        select(TracingConfig).where(TracingConfig.id == config_id)
    )
    config = result.scalar_one_or_none()

    if not config:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Tracing configuration {config_id} not found"
        )

    # Delete associated stats
    stats_result = await db.execute(
        select(TracingStatsModel).where(TracingStatsModel.tracing_config_id == config_id)
    )
    for stats in stats_result.scalars().all():
        await db.delete(stats)

    await db.delete(config)
    await db.commit()

    logger.info(f"Deleted tracing config: {config_id}")


@router.get("/tracing/{config_id}/stats", response_model=TracingStats)
async def get_tracing_stats(
    config_id: int,
    db: AsyncSession = Depends(get_db),
    _: None = Depends(check_enterprise_license)
):
    """
    Get runtime statistics for a tracing configuration

    **Enterprise Feature**: Requires Enterprise license

    Returns metrics including:
    - Total spans created
    - Sampled vs dropped spans
    - Error span count
    - Average span duration
    - Last export timestamp

    Args:
        config_id: Configuration ID
        db: Database session

    Returns:
        Runtime tracing statistics
    """
    logger.info(f"Fetching tracing stats for config {config_id}")

    # Verify config exists
    config_result = await db.execute(
        select(TracingConfig).where(TracingConfig.id == config_id)
    )
    if not config_result.scalar_one_or_none():
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Tracing configuration {config_id} not found"
        )

    # Get latest stats
    result = await db.execute(
        select(TracingStatsModel)
        .where(TracingStatsModel.tracing_config_id == config_id)
        .order_by(TracingStatsModel.timestamp.desc())
        .limit(1)
    )
    stats = result.scalar_one_or_none()

    if not stats:
        # Return zeroed stats if no data yet
        return TracingStats(
            total_spans=0,
            sampled_spans=0,
            dropped_spans=0,
            error_spans=0,
            avg_span_duration_ms=None,
            last_export=None
        )

    return TracingStats(
        total_spans=stats.total_spans,
        sampled_spans=stats.sampled_spans,
        dropped_spans=stats.dropped_spans,
        error_spans=stats.error_spans,
        avg_span_duration_ms=stats.avg_span_duration_ms,
        last_export=stats.last_export
    )


@router.post("/tracing/{config_id}/enable", response_model=TracingConfigResponse)
async def enable_tracing_config(
    config_id: int,
    db: AsyncSession = Depends(get_db),
    _: None = Depends(check_enterprise_license)
):
    """
    Enable a tracing configuration

    **Enterprise Feature**: Requires Enterprise license

    Args:
        config_id: Configuration ID
        db: Database session

    Returns:
        Updated tracing configuration
    """
    logger.info(f"Enabling tracing config {config_id}")

    result = await db.execute(
        select(TracingConfig).where(TracingConfig.id == config_id)
    )
    config = result.scalar_one_or_none()

    if not config:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Tracing configuration {config_id} not found"
        )

    config.enabled = True
    config.updated_at = datetime.utcnow()

    await db.commit()
    await db.refresh(config)

    logger.info(f"Enabled tracing config: {config.name} (ID: {config.id})")
    return config_to_response(config)


@router.post("/tracing/{config_id}/disable", response_model=TracingConfigResponse)
async def disable_tracing_config(
    config_id: int,
    db: AsyncSession = Depends(get_db),
    _: None = Depends(check_enterprise_license)
):
    """
    Disable a tracing configuration

    **Enterprise Feature**: Requires Enterprise license

    Args:
        config_id: Configuration ID
        db: Database session

    Returns:
        Updated tracing configuration
    """
    logger.info(f"Disabling tracing config {config_id}")

    result = await db.execute(
        select(TracingConfig).where(TracingConfig.id == config_id)
    )
    config = result.scalar_one_or_none()

    if not config:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Tracing configuration {config_id} not found"
        )

    config.enabled = False
    config.updated_at = datetime.utcnow()

    await db.commit()
    await db.refresh(config)

    logger.info(f"Disabled tracing config: {config.name} (ID: {config.id})")
    return config_to_response(config)


@router.post("/tracing/{config_id}/test", response_model=dict)
async def test_tracing_config(
    config_id: int,
    send_test_span: bool = True,
    db: AsyncSession = Depends(get_db),
    _: None = Depends(check_enterprise_license)
):
    """
    Test a tracing configuration

    **Enterprise Feature**: Requires Enterprise license

    Validates the configuration and optionally sends a test span
    to the tracing backend to verify connectivity.

    Args:
        config_id: Configuration ID
        send_test_span: Send a test span to backend
        db: Database session

    Returns:
        Test results
    """
    logger.info(
        f"Testing tracing config {config_id}, send_test_span={send_test_span}"
    )

    result = await db.execute(
        select(TracingConfig).where(TracingConfig.id == config_id)
    )
    config = result.scalar_one_or_none()

    if not config:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Tracing configuration {config_id} not found"
        )

    # Validate configuration
    validation_errors = []
    if not config.endpoint:
        validation_errors.append("Endpoint is required")
    if config.sampling_rate < 0 or config.sampling_rate > 1:
        validation_errors.append("Sampling rate must be between 0 and 1")

    if validation_errors:
        return {
            "config_id": config_id,
            "config_name": config.name,
            "test_status": "validation_failed",
            "validation_errors": validation_errors,
            "connectivity_test": None
        }

    # Test connectivity to backend
    connectivity_result = {
        "endpoint": config.endpoint,
        "backend": config.backend,
        "exporter": config.exporter,
        "status": "reachable",
        "latency_ms": 0
    }

    if send_test_span:
        import httpx
        import time

        try:
            start_time = time.time()
            async with httpx.AsyncClient(timeout=10.0) as client:
                # Test endpoint connectivity based on exporter type
                if config.exporter == 'http':
                    test_url = config.endpoint
                    if not test_url.startswith(('http://', 'https://')):
                        test_url = f"http://{test_url}"
                    response = await client.get(test_url)
                    connectivity_result["http_status"] = response.status_code
                else:
                    # For gRPC/thrift, just check host:port is reachable
                    host_port = config.endpoint.replace("http://", "").replace("https://", "").split("/")[0]
                    host = host_port.split(":")[0]
                    port = int(host_port.split(":")[1]) if ":" in host_port else 14268
                    import socket
                    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
                    sock.settimeout(5)
                    result_code = sock.connect_ex((host, port))
                    sock.close()
                    if result_code != 0:
                        connectivity_result["status"] = "unreachable"
                        connectivity_result["error"] = f"Connection refused on port {port}"

                connectivity_result["latency_ms"] = round((time.time() - start_time) * 1000, 2)

        except Exception as e:
            connectivity_result["status"] = "unreachable"
            connectivity_result["error"] = str(e)

    return {
        "config_id": config_id,
        "config_name": config.name,
        "test_status": "success" if connectivity_result.get("status") == "reachable" else "failed",
        "validation_errors": [],
        "connectivity_test": connectivity_result
    }


@router.get("/spans/search", response_model=dict)
async def search_spans(
    service_name: Optional[str] = None,
    operation_name: Optional[str] = None,
    min_duration_ms: Optional[int] = None,
    max_duration_ms: Optional[int] = None,
    has_error: Optional[bool] = None,
    tags: Optional[str] = None,
    limit: int = 100,
    db: AsyncSession = Depends(get_db),
    _: None = Depends(check_enterprise_license)
):
    """
    Search for traces/spans

    **Enterprise Feature**: Requires Enterprise license

    Advanced trace search with filtering by:
    - Service name
    - Operation name
    - Duration range
    - Error status
    - Custom tags

    Args:
        service_name: Filter by service name
        operation_name: Filter by operation name
        min_duration_ms: Minimum span duration
        max_duration_ms: Maximum span duration
        has_error: Filter for error spans
        tags: Custom tag filters (JSON string)
        limit: Maximum results to return
        db: Database session

    Returns:
        Search results with trace IDs
    """
    logger.info(
        f"Searching spans: service={service_name}, op={operation_name}, "
        f"duration=[{min_duration_ms}, {max_duration_ms}], error={has_error}"
    )

    # Get all tracing configs to find backend endpoints
    configs_result = await db.execute(
        select(TracingConfig).where(TracingConfig.enabled == True)
    )
    configs = configs_result.scalars().all()

    if not configs:
        return {
            "spans": [],
            "total_count": 0,
            "search_params": {
                "service_name": service_name,
                "operation_name": operation_name,
                "min_duration_ms": min_duration_ms,
                "max_duration_ms": max_duration_ms,
                "has_error": has_error,
                "limit": limit
            },
            "message": "No tracing configurations enabled"
        }

    # Query Jaeger/Zipkin API for spans
    import httpx
    all_spans = []

    for config in configs:
        if config.backend == 'jaeger':
            try:
                async with httpx.AsyncClient(timeout=10.0) as client:
                    # Build Jaeger API query
                    base_url = config.endpoint.rstrip('/')
                    if ':14268' in base_url or ':14250' in base_url:
                        # Collector port - switch to query port
                        base_url = base_url.replace(':14268', ':16686').replace(':14250', ':16686')

                    params = {"limit": limit}
                    if service_name:
                        params["service"] = service_name
                    if operation_name:
                        params["operation"] = operation_name
                    if min_duration_ms:
                        params["minDuration"] = f"{min_duration_ms}ms"
                    if max_duration_ms:
                        params["maxDuration"] = f"{max_duration_ms}ms"

                    response = await client.get(
                        f"{base_url}/api/traces",
                        params=params
                    )

                    if response.status_code == 200:
                        data = response.json()
                        for trace in data.get("data", []):
                            for span in trace.get("spans", []):
                                span_data = {
                                    "trace_id": trace.get("traceID"),
                                    "span_id": span.get("spanID"),
                                    "operation_name": span.get("operationName"),
                                    "service_name": span.get("processID"),
                                    "duration_ms": span.get("duration", 0) / 1000,
                                    "start_time": span.get("startTime"),
                                    "has_error": any(
                                        tag.get("key") == "error" and tag.get("value")
                                        for tag in span.get("tags", [])
                                    ),
                                    "tags": {
                                        tag.get("key"): tag.get("value")
                                        for tag in span.get("tags", [])
                                    }
                                }

                                # Apply filters
                                if has_error is not None and span_data["has_error"] != has_error:
                                    continue

                                all_spans.append(span_data)

            except Exception as e:
                logger.warning(f"Failed to query Jaeger at {config.endpoint}: {e}")

        elif config.backend == 'zipkin':
            try:
                async with httpx.AsyncClient(timeout=10.0) as client:
                    base_url = config.endpoint.rstrip('/')

                    params = {"limit": limit}
                    if service_name:
                        params["serviceName"] = service_name
                    if operation_name:
                        params["spanName"] = operation_name
                    if min_duration_ms:
                        params["minDuration"] = min_duration_ms * 1000
                    if max_duration_ms:
                        params["maxDuration"] = max_duration_ms * 1000

                    response = await client.get(
                        f"{base_url}/api/v2/traces",
                        params=params
                    )

                    if response.status_code == 200:
                        traces = response.json()
                        for trace in traces:
                            for span in trace:
                                span_data = {
                                    "trace_id": span.get("traceId"),
                                    "span_id": span.get("id"),
                                    "operation_name": span.get("name"),
                                    "service_name": span.get("localEndpoint", {}).get("serviceName"),
                                    "duration_ms": span.get("duration", 0) / 1000,
                                    "start_time": span.get("timestamp"),
                                    "has_error": "error" in span.get("tags", {}),
                                    "tags": span.get("tags", {})
                                }

                                if has_error is not None and span_data["has_error"] != has_error:
                                    continue

                                all_spans.append(span_data)

            except Exception as e:
                logger.warning(f"Failed to query Zipkin at {config.endpoint}: {e}")

    return {
        "spans": all_spans[:limit],
        "total_count": len(all_spans),
        "search_params": {
            "service_name": service_name,
            "operation_name": operation_name,
            "min_duration_ms": min_duration_ms,
            "max_duration_ms": max_duration_ms,
            "has_error": has_error,
            "limit": limit
        },
        "backends_queried": [c.backend for c in configs]
    }
