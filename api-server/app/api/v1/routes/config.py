"""
Configuration API Routes

Provides proxy configuration endpoints for fetching runtime configuration.
Used by proxy containers to get their service mappings, certificates, etc.
"""

import logging
from typing import Annotated

from fastapi import APIRouter, Depends, HTTPException, Header, status
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.database import get_db
from app.services.proxy_service import ProxyService, InvalidAPIKeyError
from app.services.config_builder import ConfigBuilder

router = APIRouter(prefix="/config", tags=["configuration"])
logger = logging.getLogger(__name__)


@router.get("/{cluster_id}")
async def get_cluster_config(
    cluster_id: int,
    cluster_api_key: Annotated[str, Header()],
    db: Annotated[AsyncSession, Depends(get_db)],
    include_certificates: bool = True
):
    """
    Get complete configuration for a cluster

    Authentication: Requires valid cluster API key in header

    Returns:
    - Cluster settings
    - Active services
    - Service mappings
    - TLS certificates (if include_certificates=True)
    - Logging configuration

    This endpoint is called by proxy containers on startup and periodically
    to fetch configuration updates.
    """
    proxy_service = ProxyService(db)

    # Verify cluster API key
    try:
        cluster = await proxy_service.verify_cluster_api_key(cluster_api_key)
    except InvalidAPIKeyError:
        raise HTTPException(
            status.HTTP_401_UNAUTHORIZED,
            "Invalid cluster API key"
        )

    # Verify cluster ID matches
    if cluster.id != cluster_id:
        raise HTTPException(
            status.HTTP_403_FORBIDDEN,
            "Cluster ID does not match API key"
        )

    # Build configuration
    config_builder = ConfigBuilder(db)
    config = await config_builder.build_cluster_config(
        cluster,
        include_certificates=include_certificates
    )

    logger.info(
        f"Configuration fetched for cluster {cluster.name} "
        f"(version: {config['config_version']})"
    )

    return config


@router.get("/validate/{cluster_id}")
async def validate_cluster_config(
    cluster_id: int,
    cluster_api_key: Annotated[str, Header()],
    db: Annotated[AsyncSession, Depends(get_db)]
):
    """
    Validate cluster configuration

    Checks for:
    - Valid services
    - Valid mappings
    - Certificate expiry
    - Configuration consistency

    Returns validation results and warnings.
    """
    proxy_service = ProxyService(db)

    # Verify cluster API key
    try:
        cluster = await proxy_service.verify_cluster_api_key(cluster_api_key)
    except InvalidAPIKeyError:
        raise HTTPException(
            status.HTTP_401_UNAUTHORIZED,
            "Invalid cluster API key"
        )

    # Verify cluster ID matches
    if cluster.id != cluster_id:
        raise HTTPException(
            status.HTTP_403_FORBIDDEN,
            "Cluster ID does not match API key"
        )

    # Build configuration
    config_builder = ConfigBuilder(db)
    config = await config_builder.build_cluster_config(cluster)

    # Perform validation
    validation_results = {
        "cluster_id": cluster_id,
        "cluster_name": cluster.name,
        "config_version": config["config_version"],
        "valid": True,
        "warnings": [],
        "errors": [],
        "stats": {
            "services": len(config["services"]),
            "mappings": len(config["mappings"]),
            "certificates": len(config["certificates"]) if config["certificates"] else 0,
        }
    }

    # Check for services with no mappings
    service_ids = {svc["id"] for svc in config["services"]}
    mapped_services = set()

    for mapping in config["mappings"]:
        for svc_id in mapping["source_services"]:
            if isinstance(svc_id, int):
                mapped_services.add(svc_id)
        for svc_id in mapping["dest_services"]:
            if isinstance(svc_id, int):
                mapped_services.add(svc_id)

    unmapped = service_ids - mapped_services
    if unmapped:
        validation_results["warnings"].append({
            "type": "unmapped_services",
            "message": f"Services with no mappings: {unmapped}",
            "service_ids": list(unmapped)
        })

    # Check for expired or expiring certificates
    if config["certificates"]:
        from datetime import datetime
        now = datetime.utcnow()

        for cert_name, cert_data in config["certificates"].items():
            valid_until = datetime.fromisoformat(cert_data["valid_until"])
            days_remaining = (valid_until - now).days

            if days_remaining < 0:
                validation_results["errors"].append({
                    "type": "expired_certificate",
                    "message": f"Certificate '{cert_name}' has expired",
                    "certificate": cert_name,
                    "expired_days": abs(days_remaining)
                })
                validation_results["valid"] = False
            elif days_remaining < 30:
                validation_results["warnings"].append({
                    "type": "expiring_certificate",
                    "message": f"Certificate '{cert_name}' expires in {days_remaining} days",
                    "certificate": cert_name,
                    "days_remaining": days_remaining
                })

    # Check for empty mappings
    if not config["mappings"]:
        validation_results["warnings"].append({
            "type": "no_mappings",
            "message": "No service mappings configured"
        })

    # Check for services without authentication
    for svc in config["services"]:
        if svc["auth"]["type"] == "none":
            validation_results["warnings"].append({
                "type": "no_auth",
                "message": f"Service '{svc['name']}' has no authentication",
                "service_id": svc["id"],
                "service_name": svc["name"]
            })

    logger.info(
        f"Configuration validation for cluster {cluster.name}: "
        f"valid={validation_results['valid']}, "
        f"warnings={len(validation_results['warnings'])}, "
        f"errors={len(validation_results['errors'])}"
    )

    return validation_results


@router.get("/version/{cluster_id}")
async def get_config_version(
    cluster_id: int,
    cluster_api_key: Annotated[str, Header()],
    db: Annotated[AsyncSession, Depends(get_db)]
):
    """
    Get current configuration version for a cluster

    Lightweight endpoint for proxies to check if configuration has changed
    without fetching the entire config.

    Returns only the config version hash.
    """
    proxy_service = ProxyService(db)

    # Verify cluster API key
    try:
        cluster = await proxy_service.verify_cluster_api_key(cluster_api_key)
    except InvalidAPIKeyError:
        raise HTTPException(
            status.HTTP_401_UNAUTHORIZED,
            "Invalid cluster API key"
        )

    # Verify cluster ID matches
    if cluster.id != cluster_id:
        raise HTTPException(
            status.HTTP_403_FORBIDDEN,
            "Cluster ID does not match API key"
        )

    # Build configuration to get version
    config_builder = ConfigBuilder(db)
    config = await config_builder.build_cluster_config(
        cluster,
        include_certificates=False  # Don't need cert data for version check
    )

    return {
        "cluster_id": cluster_id,
        "cluster_name": cluster.name,
        "config_version": config["config_version"],
        "generated_at": config["generated_at"]
    }
