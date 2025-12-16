"""
Traffic Shaping and QoS Configuration API

Enterprise-only feature for advanced traffic management with priority queues,
bandwidth limits, and DSCP marking.
"""

import logging
from typing import List, Optional
from datetime import datetime

from fastapi import APIRouter, HTTPException, Depends, status
from sqlalchemy import select, and_
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.database import get_db
from app.core.license import license_validator
from app.models.sqlalchemy.enterprise import QoSPolicy
from app.schemas.traffic_shaping import (
    QoSPolicyCreate,
    QoSPolicyUpdate,
    QoSPolicyResponse
)

router = APIRouter()
logger = logging.getLogger(__name__)

FEATURE_NAME = "traffic_shaping"


async def check_enterprise_license():
    """Check if traffic shaping feature is available."""
    has_feature = await license_validator.check_feature(FEATURE_NAME)
    if not has_feature:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail={
                "error": "Enterprise feature not available",
                "feature": FEATURE_NAME,
                "message": "Traffic shaping requires an Enterprise license",
                "upgrade_url": "https://www.penguintech.io/marchproxy/pricing"
            }
        )


def policy_to_response(policy: QoSPolicy) -> QoSPolicyResponse:
    """Convert database model to response schema."""
    return QoSPolicyResponse(
        id=policy.id,
        name=policy.name,
        description=policy.description,
        service_id=policy.service_id,
        cluster_id=policy.cluster_id,
        bandwidth_config=policy.bandwidth_config,
        priority_config=policy.priority_config,
        enabled=policy.enabled,
        created_at=policy.created_at,
        updated_at=policy.updated_at
    )


@router.get("/policies", response_model=List[QoSPolicyResponse])
async def list_qos_policies(
    cluster_id: Optional[int] = None,
    service_id: Optional[int] = None,
    enabled_only: bool = False,
    db: AsyncSession = Depends(get_db),
    _: None = Depends(check_enterprise_license)
):
    """List all QoS policies with optional filters."""
    query = select(QoSPolicy)

    conditions = []
    if cluster_id is not None:
        conditions.append(QoSPolicy.cluster_id == cluster_id)
    if service_id is not None:
        conditions.append(QoSPolicy.service_id == service_id)
    if enabled_only:
        conditions.append(QoSPolicy.enabled == True)

    if conditions:
        query = query.where(and_(*conditions))

    query = query.order_by(QoSPolicy.created_at.desc())

    result = await db.execute(query)
    policies = result.scalars().all()

    return [policy_to_response(p) for p in policies]


@router.post("/policies", response_model=QoSPolicyResponse, status_code=status.HTTP_201_CREATED)
async def create_qos_policy(
    policy: QoSPolicyCreate,
    db: AsyncSession = Depends(get_db),
    _: None = Depends(check_enterprise_license)
):
    """Create a new QoS policy."""
    db_policy = QoSPolicy(
        name=policy.name,
        description=policy.description,
        service_id=policy.service_id,
        cluster_id=policy.cluster_id,
        bandwidth_config=policy.bandwidth_config.model_dump() if policy.bandwidth_config else {
            "ingress_mbps": None,
            "egress_mbps": None,
            "burst_size_kb": 1024
        },
        priority_config=policy.priority_config.model_dump() if policy.priority_config else {
            "priority": "P2",
            "weight": 1,
            "max_latency_ms": 100,
            "dscp_marking": "BE"
        },
        enabled=policy.enabled if policy.enabled is not None else True,
        created_at=datetime.utcnow(),
        updated_at=datetime.utcnow()
    )

    db.add(db_policy)
    await db.commit()
    await db.refresh(db_policy)

    logger.info(f"Created QoS policy: {db_policy.name} (ID: {db_policy.id})")
    return policy_to_response(db_policy)


@router.get("/policies/{policy_id}", response_model=QoSPolicyResponse)
async def get_qos_policy(
    policy_id: int,
    db: AsyncSession = Depends(get_db),
    _: None = Depends(check_enterprise_license)
):
    """Get a specific QoS policy by ID."""
    result = await db.execute(
        select(QoSPolicy).where(QoSPolicy.id == policy_id)
    )
    policy = result.scalar_one_or_none()

    if not policy:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"QoS policy {policy_id} not found"
        )

    return policy_to_response(policy)


@router.put("/policies/{policy_id}", response_model=QoSPolicyResponse)
async def update_qos_policy(
    policy_id: int,
    policy_update: QoSPolicyUpdate,
    db: AsyncSession = Depends(get_db),
    _: None = Depends(check_enterprise_license)
):
    """Update an existing QoS policy."""
    result = await db.execute(
        select(QoSPolicy).where(QoSPolicy.id == policy_id)
    )
    policy = result.scalar_one_or_none()

    if not policy:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"QoS policy {policy_id} not found"
        )

    update_data = policy_update.model_dump(exclude_unset=True)

    for field, value in update_data.items():
        if field == "bandwidth_config" and value is not None:
            setattr(policy, field, value.model_dump() if hasattr(value, 'model_dump') else value)
        elif field == "priority_config" and value is not None:
            setattr(policy, field, value.model_dump() if hasattr(value, 'model_dump') else value)
        else:
            setattr(policy, field, value)

    policy.updated_at = datetime.utcnow()

    await db.commit()
    await db.refresh(policy)

    logger.info(f"Updated QoS policy: {policy.name} (ID: {policy.id})")
    return policy_to_response(policy)


@router.delete("/policies/{policy_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_qos_policy(
    policy_id: int,
    db: AsyncSession = Depends(get_db),
    _: None = Depends(check_enterprise_license)
):
    """Delete a QoS policy."""
    result = await db.execute(
        select(QoSPolicy).where(QoSPolicy.id == policy_id)
    )
    policy = result.scalar_one_or_none()

    if not policy:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"QoS policy {policy_id} not found"
        )

    await db.delete(policy)
    await db.commit()

    logger.info(f"Deleted QoS policy: {policy_id}")


@router.post("/policies/{policy_id}/enable", response_model=QoSPolicyResponse)
async def enable_qos_policy(
    policy_id: int,
    db: AsyncSession = Depends(get_db),
    _: None = Depends(check_enterprise_license)
):
    """Enable a QoS policy."""
    result = await db.execute(
        select(QoSPolicy).where(QoSPolicy.id == policy_id)
    )
    policy = result.scalar_one_or_none()

    if not policy:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"QoS policy {policy_id} not found"
        )

    policy.enabled = True
    policy.updated_at = datetime.utcnow()

    await db.commit()
    await db.refresh(policy)

    logger.info(f"Enabled QoS policy: {policy.name} (ID: {policy.id})")
    return policy_to_response(policy)


@router.post("/policies/{policy_id}/disable", response_model=QoSPolicyResponse)
async def disable_qos_policy(
    policy_id: int,
    db: AsyncSession = Depends(get_db),
    _: None = Depends(check_enterprise_license)
):
    """Disable a QoS policy."""
    result = await db.execute(
        select(QoSPolicy).where(QoSPolicy.id == policy_id)
    )
    policy = result.scalar_one_or_none()

    if not policy:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"QoS policy {policy_id} not found"
        )

    policy.enabled = False
    policy.updated_at = datetime.utcnow()

    await db.commit()
    await db.refresh(policy)

    logger.info(f"Disabled QoS policy: {policy.name} (ID: {policy.id})")
    return policy_to_response(policy)
