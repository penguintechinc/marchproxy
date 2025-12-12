"""
Service management API routes (Phase 2 - Core CRUD, no xDS yet)

Handles CRUD operations for services and authentication token management.
"""

import base64
import logging
import secrets
from datetime import datetime
from typing import Annotated

from fastapi import APIRouter, Depends, HTTPException, status, Query
from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.database import get_db
from app.dependencies import get_current_user, require_admin
from app.models.sqlalchemy.user import User
from app.models.sqlalchemy.service import Service, UserServiceAssignment
from app.models.sqlalchemy.cluster import Cluster, UserClusterAssignment
from app.schemas.service import (
    ServiceCreate,
    ServiceUpdate,
    ServiceResponse,
    ServiceListResponse,
    ServiceTokenRotateRequest,
    ServiceTokenRotateResponse,
)
from app.services.xds_service import trigger_xds_update

router = APIRouter(prefix="/services", tags=["services"])
logger = logging.getLogger(__name__)


def generate_base64_token() -> str:
    """Generate a secure Base64 token"""
    return base64.b64encode(secrets.token_bytes(32)).decode()


def generate_jwt_secret() -> str:
    """Generate a secure JWT secret"""
    return secrets.token_urlsafe(64)


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
    """Create a new service with auto-generated auth tokens"""
    # Verify cluster access
    if not current_user.is_admin:
        stmt = select(UserClusterAssignment).where(
            UserClusterAssignment.user_id == current_user.id,
            UserClusterAssignment.cluster_id == service_data.cluster_id,
            UserClusterAssignment.is_active == True
        )
        if not (await db.execute(stmt)).scalar_one_or_none():
            raise HTTPException(status.HTTP_403_FORBIDDEN, "No access to this cluster")

    # Verify cluster exists
    if not (await db.execute(select(Cluster).where(Cluster.id == service_data.cluster_id))).scalar_one_or_none():
        raise HTTPException(status.HTTP_404_NOT_FOUND, "Cluster not found")

    # Check name uniqueness
    if (await db.execute(select(Service).where(Service.name == service_data.name))).scalar_one_or_none():
        raise HTTPException(status.HTTP_400_BAD_REQUEST, f"Service '{service_data.name}' already exists")

    # Generate auth tokens
    token_base64, jwt_secret = None, None
    if service_data.auth_type == "base64":
        token_base64 = generate_base64_token()
    elif service_data.auth_type == "jwt":
        jwt_secret = generate_jwt_secret()

    new_service = Service(
        **service_data.model_dump(exclude={'jwt_expiry', 'jwt_algorithm'}),
        token_base64=token_base64,
        jwt_secret=jwt_secret,
        jwt_expiry=service_data.jwt_expiry or 3600,
        jwt_algorithm=service_data.jwt_algorithm or "HS256",
        is_active=True,
        created_by=current_user.id,
        created_at=datetime.utcnow()
    )

    db.add(new_service)
    await db.commit()
    await db.refresh(new_service)

    # Auto-assign to creator if not admin
    if not current_user.is_admin:
        db.add(UserServiceAssignment(
            user_id=current_user.id,
            service_id=new_service.id,
            assigned_by=current_user.id
        ))
        await db.commit()

    logger.info(f"Service created: {new_service.name} by {current_user.username}")

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
    """Get service details"""
    stmt = select(Service).where(Service.id == service_id)
    if not current_user.is_admin:
        stmt = stmt.join(UserServiceAssignment).where(
            UserServiceAssignment.user_id == current_user.id,
            UserServiceAssignment.is_active == True
        )

    service = (await db.execute(stmt)).scalar_one_or_none()
    if not service:
        raise HTTPException(status.HTTP_404_NOT_FOUND, "Service not found")
    return ServiceResponse.model_validate(service)


@router.patch("/{service_id}", response_model=ServiceResponse)
async def update_service(
    service_id: int,
    service_data: ServiceUpdate,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(get_current_user)]
):
    """Update service (excludes auth tokens - use rotate endpoint)"""
    service = await get_service(service_id, db, current_user)  # Reuse access check

    for field, value in service_data.model_dump(exclude_unset=True).items():
        setattr(service, field, value)

    service.updated_at = datetime.utcnow()
    await db.commit()
    await db.refresh(service)

    logger.info(f"Service updated: {service.name} by {current_user.username}")

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
    permanent: bool = Query(False)
):
    """Delete or deactivate service"""
    service = await get_service(service_id, db, current_user)  # Reuse access check

    cluster_id = service.cluster_id

    if permanent:
        await db.delete(service)
        logger.warning(f"Service deleted: {service.name} by {current_user.username}")
    else:
        service.is_active = False
        service.updated_at = datetime.utcnow()
        logger.info(f"Service deactivated: {service.name} by {current_user.username}")

    await db.commit()

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
    """Rotate service authentication token"""
    service = await get_service(service_id, db, current_user)  # Reuse access check

    new_token, new_jwt_secret = None, None
    if rotation.auth_type == "base64":
        new_token = generate_base64_token()
        service.token_base64, service.auth_type = new_token, "base64"
    elif rotation.auth_type == "jwt":
        new_jwt_secret = generate_jwt_secret()
        service.jwt_secret, service.auth_type = new_jwt_secret, "jwt"

    service.updated_at = datetime.utcnow()
    await db.commit()

    logger.warning(f"Auth token rotated for {service.name} by {current_user.username}")
    return ServiceTokenRotateResponse(
        service_id=service.id,
        auth_type=rotation.auth_type,
        new_token=new_token,
        new_jwt_secret=new_jwt_secret
    )
