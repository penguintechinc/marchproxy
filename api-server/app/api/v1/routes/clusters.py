"""
Cluster management API routes

Handles CRUD operations for clusters, API key management, and cluster assignments.
"""

import hashlib
import logging
import secrets
from datetime import datetime
from typing import Annotated, List

from fastapi import APIRouter, Depends, HTTPException, status, Query
from sqlalchemy import select, update, delete, func
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.database import get_db
from app.dependencies import get_current_user, require_admin
from app.models.sqlalchemy.user import User
from app.models.sqlalchemy.cluster import Cluster, UserClusterAssignment
from app.schemas.cluster import (
    ClusterCreate,
    ClusterUpdate,
    ClusterResponse,
    ClusterListResponse,
    ClusterAPIKeyRotateResponse,
)

router = APIRouter(prefix="/clusters", tags=["clusters"])
logger = logging.getLogger(__name__)


def generate_api_key() -> str:
    """Generate a secure API key"""
    return secrets.token_urlsafe(48)


def hash_api_key(api_key: str) -> str:
    """Hash API key for storage"""
    return hashlib.sha256(api_key.encode()).hexdigest()


@router.get("", response_model=ClusterListResponse)
async def list_clusters(
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(get_current_user)],
    skip: int = Query(0, ge=0),
    limit: int = Query(100, ge=1, le=1000),
    active_only: bool = Query(True)
):
    """
    List all clusters accessible to the current user.

    Admins see all clusters. Regular users see only assigned clusters.
    """
    query = select(Cluster)

    if active_only:
        query = query.where(Cluster.is_active == True)

    # Non-admin users only see assigned clusters
    if not current_user.is_admin:
        query = query.join(UserClusterAssignment).where(
            UserClusterAssignment.user_id == current_user.id,
            UserClusterAssignment.is_active == True
        )

    # Get total count
    count_query = select(func.count()).select_from(query.subquery())
    total_result = await db.execute(count_query)
    total = total_result.scalar() or 0

    # Get paginated results
    query = query.offset(skip).limit(limit).order_by(Cluster.created_at.desc())
    result = await db.execute(query)
    clusters = result.scalars().all()

    return ClusterListResponse(
        total=total,
        clusters=[ClusterResponse.model_validate(c) for c in clusters]
    )


@router.post("", response_model=ClusterResponse, status_code=status.HTTP_201_CREATED)
async def create_cluster(
    cluster_data: ClusterCreate,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(require_admin)]
):
    """
    Create a new cluster (Admin only).

    Generates a unique API key for cluster authentication.
    Returns the API key only once - it cannot be retrieved later.
    """
    # Check if cluster name already exists
    stmt = select(Cluster).where(Cluster.name == cluster_data.name)
    existing = await db.execute(stmt)
    if existing.scalar_one_or_none():
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Cluster with name '{cluster_data.name}' already exists"
        )

    # Generate API key
    api_key = generate_api_key()
    api_key_hash = hash_api_key(api_key)

    # Create cluster
    new_cluster = Cluster(
        name=cluster_data.name,
        description=cluster_data.description,
        api_key_hash=api_key_hash,
        syslog_endpoint=cluster_data.syslog_endpoint,
        log_auth=cluster_data.log_auth,
        log_netflow=cluster_data.log_netflow,
        log_debug=cluster_data.log_debug,
        max_proxies=cluster_data.max_proxies,
        is_active=True,
        is_default=False,
        created_by=current_user.id,
        created_at=datetime.utcnow()
    )

    db.add(new_cluster)
    await db.commit()
    await db.refresh(new_cluster)

    logger.info(f"Cluster created: {new_cluster.name} (ID: {new_cluster.id}) by user {current_user.username}")

    # Return response with API key (only shown once)
    response = ClusterResponse.model_validate(new_cluster)
    # Temporarily attach the plain API key to response
    response.api_key = api_key  # type: ignore

    return response


@router.get("/{cluster_id}", response_model=ClusterResponse)
async def get_cluster(
    cluster_id: int,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(get_current_user)]
):
    """
    Get cluster details by ID.

    Admins can access any cluster. Regular users only see assigned clusters.
    """
    stmt = select(Cluster).where(Cluster.id == cluster_id)

    # Non-admin users need assignment check
    if not current_user.is_admin:
        stmt = stmt.join(UserClusterAssignment).where(
            UserClusterAssignment.user_id == current_user.id,
            UserClusterAssignment.is_active == True
        )

    result = await db.execute(stmt)
    cluster = result.scalar_one_or_none()

    if not cluster:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Cluster not found or access denied"
        )

    return ClusterResponse.model_validate(cluster)


@router.patch("/{cluster_id}", response_model=ClusterResponse)
async def update_cluster(
    cluster_id: int,
    cluster_data: ClusterUpdate,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(require_admin)]
):
    """
    Update cluster details (Admin only).

    Does not update API key - use rotate_api_key endpoint for that.
    """
    stmt = select(Cluster).where(Cluster.id == cluster_id)
    result = await db.execute(stmt)
    cluster = result.scalar_one_or_none()

    if not cluster:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Cluster not found"
        )

    # Update fields
    update_data = cluster_data.model_dump(exclude_unset=True)
    for field, value in update_data.items():
        setattr(cluster, field, value)

    cluster.updated_at = datetime.utcnow()
    await db.commit()
    await db.refresh(cluster)

    logger.info(f"Cluster updated: {cluster.name} (ID: {cluster.id}) by user {current_user.username}")

    return ClusterResponse.model_validate(cluster)


@router.delete("/{cluster_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_cluster(
    cluster_id: int,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(require_admin)],
    permanent: bool = Query(False, description="Permanently delete instead of deactivate")
):
    """
    Delete or deactivate a cluster (Admin only).

    By default, clusters are soft-deleted (deactivated).
    Use permanent=true for hard delete.
    """
    stmt = select(Cluster).where(Cluster.id == cluster_id)
    result = await db.execute(stmt)
    cluster = result.scalar_one_or_none()

    if not cluster:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Cluster not found"
        )

    # Cannot delete default cluster
    if cluster.is_default:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Cannot delete the default cluster"
        )

    if permanent:
        await db.delete(cluster)
        logger.warning(f"Cluster permanently deleted: {cluster.name} (ID: {cluster.id}) by user {current_user.username}")
    else:
        cluster.is_active = False
        cluster.updated_at = datetime.utcnow()
        logger.info(f"Cluster deactivated: {cluster.name} (ID: {cluster.id}) by user {current_user.username}")

    await db.commit()


@router.post("/{cluster_id}/rotate-api-key", response_model=ClusterAPIKeyRotateResponse)
async def rotate_cluster_api_key(
    cluster_id: int,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(require_admin)]
):
    """
    Rotate cluster API key (Admin only).

    Generates a new API key and returns it (only shown once).
    Old API key becomes invalid immediately.
    """
    stmt = select(Cluster).where(Cluster.id == cluster_id)
    result = await db.execute(stmt)
    cluster = result.scalar_one_or_none()

    if not cluster:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Cluster not found"
        )

    # Generate new API key
    new_api_key = generate_api_key()
    new_api_key_hash = hash_api_key(new_api_key)

    # Update cluster
    cluster.api_key_hash = new_api_key_hash
    cluster.updated_at = datetime.utcnow()

    await db.commit()

    logger.warning(f"API key rotated for cluster: {cluster.name} (ID: {cluster.id}) by user {current_user.username}")

    return ClusterAPIKeyRotateResponse(
        cluster_id=cluster.id,
        new_api_key=new_api_key,
        message="API key rotated successfully. Update all proxy configurations with the new key."
    )
