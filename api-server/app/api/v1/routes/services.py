"""
Service management API routes

Handles CRUD operations for services and authentication token management.
Integrates with xDS service for configuration updates.
"""

import logging
from typing import Annotated

from fastapi import APIRouter, Depends, HTTPException, status, Query
from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.database import get_db
from app.dependencies import get_current_user, require_admin
from app.models.sqlalchemy.user import User
from app.models.sqlalchemy.service import Service, UserServiceAssignment
from app.schemas.service import (
    ServiceCreate,
    ServiceUpdate,
    ServiceResponse,
    ServiceListResponse,
    ServiceTokenRotateRequest,
    ServiceTokenRotateResponse,
)
from app.services.service_service import ServiceService
from app.services.xds_service import trigger_xds_update

router = APIRouter(prefix="/services", tags=["services"])
logger = logging.getLogger(__name__)


@router.get("", response_model=ServiceListResponse)
async def list_services(
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(get_current_user)],
    cluster_id: int | None = Query(None),
    skip: int = Query(0, ge=0),
    limit: int = Query(100, ge=1, le=1000),
    active_only: bool = Query(True)
):
    """List services accessible to current user"""
    query = select(Service)
    if cluster_id:
        query = query.where(Service.cluster_id == cluster_id)
    if active_only:
        query = query.where(Service.is_active == True)

    if not current_user.is_admin:
        query = query.join(UserServiceAssignment).where(
            UserServiceAssignment.user_id == current_user.id,
            UserServiceAssignment.is_active == True
        )

    count_query = select(func.count()).select_from(query.subquery())
    total_result = await db.execute(count_query)
    total = total_result.scalar() or 0

    query = query.offset(skip).limit(limit).order_by(Service.created_at.desc())
    result = await db.execute(query)
    services = result.scalars().all()

    return ServiceListResponse(
        total=total,
        services=[ServiceResponse.model_validate(s) for s in services]
    )


@router.post("", response_model=ServiceResponse, status_code=status.HTTP_201_CREATED)
async def create_service(
    service_data: ServiceCreate,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(get_current_user)]
):
    """
    Create a new service with auto-generated auth tokens.

    Validates cluster access, generates auth tokens based on auth_type,
    and triggers xDS configuration update.
    """
    service = ServiceService(db)
    new_service = await service.create_service(service_data, current_user)

    # Trigger xDS update for the cluster
    try:
        await trigger_xds_update(new_service.cluster_id, db)
        logger.info(f"xDS update triggered for cluster {new_service.cluster_id}")
    except Exception as e:
        logger.error(f"Failed to trigger xDS update: {str(e)}", exc_info=True)
        # Don't fail the request if xDS update fails

    return ServiceResponse.model_validate(new_service)


@router.get("/{service_id}", response_model=ServiceResponse)
async def get_service(
    service_id: int,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(get_current_user)]
):
    """
    Get service details by ID.

    Admins can access any service. Regular users only see assigned services.
    """
    svc_service = ServiceService(db)
    service = await svc_service.get_service(service_id, current_user)
    return ServiceResponse.model_validate(service)


@router.patch("/{service_id}", response_model=ServiceResponse)
async def update_service(
    service_id: int,
    service_data: ServiceUpdate,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(get_current_user)]
):
    """
    Update service details (excludes auth tokens - use rotate endpoint).

    Triggers xDS configuration update for the cluster.
    """
    svc_service = ServiceService(db)
    service = await svc_service.update_service(service_id, service_data, current_user)

    # Trigger xDS update for the cluster
    try:
        await trigger_xds_update(service.cluster_id, db)
        logger.info(f"xDS update triggered for cluster {service.cluster_id}")
    except Exception as e:
        logger.error(f"Failed to trigger xDS update: {str(e)}", exc_info=True)

    return ServiceResponse.model_validate(service)


@router.delete("/{service_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_service(
    service_id: int,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(get_current_user)],
    permanent: bool = Query(False, description="Permanently delete instead of deactivate")
):
    """
    Delete or deactivate a service.

    By default, services are soft-deleted (deactivated).
    Use permanent=true for hard delete.
    Triggers xDS configuration update for the cluster.
    """
    svc_service = ServiceService(db)
    cluster_id = await svc_service.delete_service(service_id, current_user, permanent)

    # Trigger xDS update for the cluster
    try:
        await trigger_xds_update(cluster_id, db)
        logger.info(f"xDS update triggered for cluster {cluster_id}")
    except Exception as e:
        logger.error(f"Failed to trigger xDS update: {str(e)}", exc_info=True)


@router.post("/{service_id}/rotate-token", response_model=ServiceTokenRotateResponse)
async def rotate_token(
    service_id: int,
    rotation: ServiceTokenRotateRequest,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(get_current_user)]
):
    """
    Rotate service authentication token.

    Generates a new Base64 token or JWT secret based on auth_type.
    Old token becomes invalid immediately.
    """
    svc_service = ServiceService(db)
    service, new_token, new_jwt_secret = await svc_service.rotate_token(
        service_id,
        rotation.auth_type,
        current_user
    )

    return ServiceTokenRotateResponse(
        service_id=service.id,
        auth_type=rotation.auth_type,
        new_token=new_token,
        new_jwt_secret=new_jwt_secret,
        message="Authentication token rotated successfully"
    )
