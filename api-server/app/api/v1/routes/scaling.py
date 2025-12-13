"""
Auto-scaling policy API

Manages auto-scaling policies for modules based on CPU, memory, or request metrics.
"""

import logging
from typing import Annotated, Optional

from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.database import get_db
from app.dependencies import get_current_user, require_admin
from app.models.sqlalchemy.user import User
from app.schemas.module import (
    ScalingPolicyCreate,
    ScalingPolicyUpdate,
    ScalingPolicyResponse,
)
from app.services.module_service_scaling import ScalingService

router = APIRouter(prefix="/modules/{module_id}/scaling", tags=["auto-scaling"])
logger = logging.getLogger(__name__)


@router.get("", response_model=Optional[ScalingPolicyResponse])
async def get_scaling_policy(
    module_id: int,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(get_current_user)]
):
    """
    Get auto-scaling policy for a module.

    Returns the current auto-scaling configuration for the module,
    or None if no policy is configured.
    """
    service = ScalingService(db)
    policy = await service.get_policy(module_id)

    if not policy:
        return None

    return ScalingPolicyResponse.model_validate(policy)


@router.post("", response_model=ScalingPolicyResponse, status_code=status.HTTP_201_CREATED)
async def create_scaling_policy(
    module_id: int,
    policy_data: ScalingPolicyCreate,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(require_admin)]
):
    """
    Create auto-scaling policy for a module (Admin only).

    Configures auto-scaling behavior based on specified metrics.
    If a policy already exists, it will be updated instead.

    Supported metrics:
    - cpu: Scale based on CPU usage percentage
    - memory: Scale based on memory usage percentage
    - requests_per_second: Scale based on request rate

    Example:
    ```json
    {
        "min_instances": 2,
        "max_instances": 10,
        "scale_up_threshold": 80.0,
        "scale_down_threshold": 20.0,
        "cooldown_seconds": 300,
        "metric": "cpu",
        "enabled": true
    }
    ```
    """
    service = ScalingService(db)

    # Validate min/max instances
    if policy_data.min_instances > policy_data.max_instances:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="min_instances cannot be greater than max_instances"
        )

    # Validate thresholds
    if policy_data.scale_down_threshold >= policy_data.scale_up_threshold:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="scale_down_threshold must be less than scale_up_threshold"
        )

    policy = await service.create_or_update_policy(module_id, policy_data)
    return ScalingPolicyResponse.model_validate(policy)


@router.put("", response_model=ScalingPolicyResponse)
async def update_scaling_policy(
    module_id: int,
    policy_data: ScalingPolicyUpdate,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(require_admin)]
):
    """
    Update auto-scaling policy for a module (Admin only).

    Updates the existing auto-scaling configuration.
    If no policy exists, a 404 error is returned.
    """
    service = ScalingService(db)

    # Check if policy exists
    existing_policy = await service.get_policy(module_id)
    if not existing_policy:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Scaling policy not found. Use POST to create a new policy."
        )

    # Validate constraints if provided
    update_dict = policy_data.model_dump(exclude_unset=True)

    # Get current values for validation
    min_instances = update_dict.get("min_instances", existing_policy.min_instances)
    max_instances = update_dict.get("max_instances", existing_policy.max_instances)
    scale_down = update_dict.get("scale_down_threshold", existing_policy.scale_down_threshold)
    scale_up = update_dict.get("scale_up_threshold", existing_policy.scale_up_threshold)

    if min_instances > max_instances:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="min_instances cannot be greater than max_instances"
        )

    if scale_down >= scale_up:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="scale_down_threshold must be less than scale_up_threshold"
        )

    policy = await service.create_or_update_policy(module_id, policy_data)
    return ScalingPolicyResponse.model_validate(policy)


@router.delete("", status_code=status.HTTP_204_NO_CONTENT)
async def delete_scaling_policy(
    module_id: int,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(require_admin)]
):
    """
    Delete auto-scaling policy for a module (Admin only).

    Removes auto-scaling configuration. The module will maintain
    its current replica count without automatic scaling.
    """
    service = ScalingService(db)
    await service.delete_policy(module_id)


@router.post("/enable", response_model=ScalingPolicyResponse)
async def enable_scaling_policy(
    module_id: int,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(require_admin)]
):
    """
    Enable auto-scaling for a module (Admin only).

    Activates the auto-scaling policy. The system will begin
    monitoring metrics and scaling instances accordingly.
    """
    service = ScalingService(db)
    existing_policy = await service.get_policy(module_id)

    if not existing_policy:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="No scaling policy configured for this module"
        )

    update_data = ScalingPolicyUpdate(enabled=True)
    policy = await service.create_or_update_policy(module_id, update_data)

    logger.info(f"Auto-scaling enabled for module {module_id}")
    return ScalingPolicyResponse.model_validate(policy)


@router.post("/disable", response_model=ScalingPolicyResponse)
async def disable_scaling_policy(
    module_id: int,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(require_admin)]
):
    """
    Disable auto-scaling for a module (Admin only).

    Deactivates the auto-scaling policy. The module will maintain
    its current replica count without automatic adjustments.
    """
    service = ScalingService(db)
    existing_policy = await service.get_policy(module_id)

    if not existing_policy:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="No scaling policy configured for this module"
        )

    update_data = ScalingPolicyUpdate(enabled=False)
    policy = await service.create_or_update_policy(module_id, update_data)

    logger.info(f"Auto-scaling disabled for module {module_id}")
    return ScalingPolicyResponse.model_validate(policy)
