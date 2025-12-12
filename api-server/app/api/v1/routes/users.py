"""
User management API routes

Handles user CRUD operations, cluster/service assignments (Admin only).
"""

import logging
from datetime import datetime
from typing import Annotated

from fastapi import APIRouter, Depends, HTTPException, status, Query
from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.database import get_db
from app.core.security import get_password_hash
from app.dependencies import get_current_user, require_admin
from app.models.sqlalchemy.user import User
from app.models.sqlalchemy.cluster import UserClusterAssignment
from app.models.sqlalchemy.service import UserServiceAssignment
from app.schemas.user import (
    UserCreate,
    UserUpdate,
    UserResponse,
    UserListResponse,
    UserClusterAssignmentCreate,
    UserServiceAssignmentCreate,
)

router = APIRouter(prefix="/users", tags=["users"])
logger = logging.getLogger(__name__)


@router.get("", response_model=UserListResponse)
async def list_users(
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(require_admin)],
    skip: int = Query(0, ge=0),
    limit: int = Query(100, ge=1, le=1000),
    active_only: bool = Query(True)
):
    """List all users (Admin only)"""
    query = select(User)
    if active_only:
        query = query.where(User.is_active == True)

    count_query = select(func.count()).select_from(query.subquery())
    total = (await db.execute(count_query)).scalar() or 0

    query = query.offset(skip).limit(limit).order_by(User.created_at.desc())
    users = (await db.execute(query)).scalars().all()

    return UserListResponse(
        total=total,
        users=[UserResponse.model_validate(u) for u in users]
    )


@router.post("", response_model=UserResponse, status_code=status.HTTP_201_CREATED)
async def create_user(
    user_data: UserCreate,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(require_admin)]
):
    """Create a new user (Admin only)"""
    # Check uniqueness
    stmt = select(User).where(
        (User.username == user_data.username) | (User.email == user_data.email)
    )
    if (await db.execute(stmt)).scalar_one_or_none():
        raise HTTPException(status.HTTP_400_BAD_REQUEST, "Username or email already exists")

    new_user = User(
        email=user_data.email,
        username=user_data.username,
        first_name=user_data.first_name,
        last_name=user_data.last_name,
        password_hash=get_password_hash(user_data.password),
        is_admin=user_data.is_admin,
        is_active=user_data.is_active,
        is_verified=True,  # Admin-created users are pre-verified
        created_at=datetime.utcnow()
    )

    db.add(new_user)
    await db.commit()
    await db.refresh(new_user)

    logger.info(f"User created: {new_user.username} by {current_user.username}")
    return UserResponse.model_validate(new_user)


@router.get("/{user_id}", response_model=UserResponse)
async def get_user(
    user_id: int,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(require_admin)]
):
    """Get user details (Admin only)"""
    user = (await db.execute(select(User).where(User.id == user_id))).scalar_one_or_none()
    if not user:
        raise HTTPException(status.HTTP_404_NOT_FOUND, "User not found")
    return UserResponse.model_validate(user)


@router.patch("/{user_id}", response_model=UserResponse)
async def update_user(
    user_id: int,
    user_data: UserUpdate,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(require_admin)]
):
    """Update user (Admin only, excludes password)"""
    user = (await db.execute(select(User).where(User.id == user_id))).scalar_one_or_none()
    if not user:
        raise HTTPException(status.HTTP_404_NOT_FOUND, "User not found")

    for field, value in user_data.model_dump(exclude_unset=True).items():
        setattr(user, field, value)

    user.updated_at = datetime.utcnow()
    await db.commit()
    await db.refresh(user)

    logger.info(f"User updated: {user.username} by {current_user.username}")
    return UserResponse.model_validate(user)


@router.delete("/{user_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_user(
    user_id: int,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(require_admin)],
    permanent: bool = Query(False)
):
    """Delete or deactivate user (Admin only)"""
    user = (await db.execute(select(User).where(User.id == user_id))).scalar_one_or_none()
    if not user:
        raise HTTPException(status.HTTP_404_NOT_FOUND, "User not found")

    # Cannot delete self
    if user.id == current_user.id:
        raise HTTPException(status.HTTP_400_BAD_REQUEST, "Cannot delete yourself")

    if permanent:
        await db.delete(user)
        logger.warning(f"User deleted: {user.username} by {current_user.username}")
    else:
        user.is_active = False
        user.updated_at = datetime.utcnow()
        logger.info(f"User deactivated: {user.username} by {current_user.username}")

    await db.commit()


@router.post("/{user_id}/cluster-assignments", status_code=status.HTTP_201_CREATED)
async def assign_user_to_cluster(
    user_id: int,
    assignment: UserClusterAssignmentCreate,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(require_admin)]
):
    """Assign user to cluster (Admin only)"""
    # Verify user and cluster exist
    user = (await db.execute(select(User).where(User.id == assignment.user_id))).scalar_one_or_none()
    if not user:
        raise HTTPException(status.HTTP_404_NOT_FOUND, "User not found")

    # Check if already assigned
    stmt = select(UserClusterAssignment).where(
        UserClusterAssignment.user_id == assignment.user_id,
        UserClusterAssignment.cluster_id == assignment.cluster_id
    )
    existing = (await db.execute(stmt)).scalar_one_or_none()
    if existing:
        existing.is_active = True
        existing.role = assignment.role
    else:
        new_assignment = UserClusterAssignment(
            user_id=assignment.user_id,
            cluster_id=assignment.cluster_id,
            role=assignment.role,
            assigned_by=current_user.id,
            assigned_at=datetime.utcnow()
        )
        db.add(new_assignment)

    await db.commit()
    logger.info(f"User {user.username} assigned to cluster {assignment.cluster_id} by {current_user.username}")
    return {"status": "ok", "message": "User assigned to cluster"}


@router.post("/{user_id}/service-assignments", status_code=status.HTTP_201_CREATED)
async def assign_user_to_service(
    user_id: int,
    assignment: UserServiceAssignmentCreate,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(require_admin)]
):
    """Assign user to service (Admin only)"""
    user = (await db.execute(select(User).where(User.id == assignment.user_id))).scalar_one_or_none()
    if not user:
        raise HTTPException(status.HTTP_404_NOT_FOUND, "User not found")

    stmt = select(UserServiceAssignment).where(
        UserServiceAssignment.user_id == assignment.user_id,
        UserServiceAssignment.service_id == assignment.service_id
    )
    existing = (await db.execute(stmt)).scalar_one_or_none()
    if existing:
        existing.is_active = True
    else:
        new_assignment = UserServiceAssignment(
            user_id=assignment.user_id,
            service_id=assignment.service_id,
            assigned_by=current_user.id,
            assigned_at=datetime.utcnow()
        )
        db.add(new_assignment)

    await db.commit()
    logger.info(f"User {user.username} assigned to service {assignment.service_id} by {current_user.username}")
    return {"status": "ok", "message": "User assigned to service"}
