"""
Blue/Green Deployment API

Manages blue/green deployments for zero-downtime updates and rollbacks.
"""

import logging
from typing import Annotated

from fastapi import APIRouter, Depends, HTTPException, status, Query
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.database import get_db
from app.dependencies import get_current_user, require_admin
from app.models.sqlalchemy.user import User
from app.schemas.module import (
    DeploymentCreate,
    DeploymentUpdate,
    DeploymentResponse,
    DeploymentListResponse,
    DeploymentPromoteRequest,
    DeploymentRollbackRequest,
)
from app.services.module_service_scaling import DeploymentService

router = APIRouter(prefix="/modules/{module_id}/deployments", tags=["deployments"])
logger = logging.getLogger(__name__)


@router.get("", response_model=DeploymentListResponse)
async def list_deployments(
    module_id: int,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(get_current_user)],
    skip: int = Query(0, ge=0),
    limit: int = Query(100, ge=1, le=1000)
):
    """
    List all deployments for a module.

    Returns deployment history including active, inactive, and rolled-back
    deployments, ordered by deployment time (newest first).
    """
    service = DeploymentService(db)
    deployments, total = await service.list_deployments(module_id, skip, limit)

    return DeploymentListResponse(
        total=total,
        deployments=[DeploymentResponse.model_validate(d) for d in deployments]
    )


@router.post("", response_model=DeploymentResponse, status_code=status.HTTP_201_CREATED)
async def create_deployment(
    module_id: int,
    deployment_data: DeploymentCreate,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(require_admin)]
):
    """
    Create a new deployment for a module (Admin only).

    Initiates a new deployment with specified version and configuration.
    The deployment starts with the specified traffic weight (default 0%).

    For blue/green deployments:
    1. Create deployment with traffic_weight=0
    2. Verify deployment health
    3. Gradually promote using /promote endpoint
    4. Rollback if issues occur using /rollback endpoint

    Example:
    ```json
    {
        "version": "v2.0.0",
        "image": "myregistry/mymodule:v2.0.0",
        "config": {"feature_x": true},
        "environment": {"ENV": "production"},
        "traffic_weight": 0.0
    }
    ```
    """
    service = DeploymentService(db)
    deployment = await service.create_deployment(
        module_id,
        deployment_data,
        current_user.id
    )

    logger.info(
        f"Deployment created: {deployment.version} for module {module_id} "
        f"by user {current_user.username}"
    )

    return DeploymentResponse.model_validate(deployment)


@router.get("/{deployment_id}", response_model=DeploymentResponse)
async def get_deployment(
    module_id: int,
    deployment_id: int,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(get_current_user)]
):
    """
    Get deployment details by ID.

    Returns detailed information about a specific deployment including
    status, traffic weight, health checks, and configuration.
    """
    service = DeploymentService(db)
    deployment = await service.get_deployment(deployment_id)

    # Verify deployment belongs to module
    if deployment.module_id != module_id:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Deployment not found for this module"
        )

    return DeploymentResponse.model_validate(deployment)


@router.patch("/{deployment_id}", response_model=DeploymentResponse)
async def update_deployment(
    module_id: int,
    deployment_id: int,
    deployment_data: DeploymentUpdate,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(require_admin)]
):
    """
    Update deployment configuration (Admin only).

    Updates deployment settings such as status, traffic weight,
    or configuration. Use the /promote and /rollback endpoints
    for controlled traffic management.
    """
    service = DeploymentService(db)
    deployment = await service.get_deployment(deployment_id)

    # Verify deployment belongs to module
    if deployment.module_id != module_id:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Deployment not found for this module"
        )

    deployment = await service.update_deployment(deployment_id, deployment_data)
    return DeploymentResponse.model_validate(deployment)


@router.post("/{deployment_id}/promote", response_model=DeploymentResponse)
async def promote_deployment(
    module_id: int,
    deployment_id: int,
    promote_data: DeploymentPromoteRequest,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(require_admin)]
):
    """
    Promote deployment to receive more traffic (Admin only).

    Increases the traffic weight for this deployment. Use incremental=true
    for gradual traffic shift (canary deployments).

    Example workflows:

    1. Instant switch (blue/green):
       POST /promote {"traffic_weight": 100.0, "incremental": false}

    2. Gradual rollout (canary):
       POST /promote {"traffic_weight": 100.0, "incremental": true}
       Repeat until traffic_weight reaches 100%

    When traffic_weight reaches 100%, the deployment is marked as ACTIVE
    and the previous deployment is marked as INACTIVE.
    """
    service = DeploymentService(db)
    deployment = await service.get_deployment(deployment_id)

    # Verify deployment belongs to module
    if deployment.module_id != module_id:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Deployment not found for this module"
        )

    deployment = await service.promote_deployment(
        deployment_id,
        promote_data.traffic_weight,
        promote_data.incremental
    )

    logger.info(
        f"Deployment promoted: {deployment.version} (ID: {deployment.id}) "
        f"to {deployment.traffic_weight}% traffic by user {current_user.username}"
    )

    return DeploymentResponse.model_validate(deployment)


@router.post("/{deployment_id}/rollback", response_model=DeploymentResponse)
async def rollback_deployment(
    module_id: int,
    deployment_id: int,
    rollback_data: DeploymentRollbackRequest,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(require_admin)]
):
    """
    Rollback deployment to previous version (Admin only).

    Immediately sets traffic_weight to 0% for this deployment and
    restores the previous deployment to 100% traffic.

    This is used when issues are detected with the new deployment
    and immediate rollback is needed.

    Returns the previous deployment that is now active.
    """
    service = DeploymentService(db)
    deployment = await service.get_deployment(deployment_id)

    # Verify deployment belongs to module
    if deployment.module_id != module_id:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Deployment not found for this module"
        )

    previous_deployment = await service.rollback_deployment(
        deployment_id,
        rollback_data.reason
    )

    logger.warning(
        f"Deployment rolled back: {deployment.version} -> {previous_deployment.version} "
        f"(module {module_id}) by user {current_user.username}. "
        f"Reason: {rollback_data.reason}"
    )

    return DeploymentResponse.model_validate(previous_deployment)


@router.get("/{deployment_id}/health", response_model=dict)
async def check_deployment_health(
    module_id: int,
    deployment_id: int,
    db: Annotated[AsyncSession, Depends(get_db)],
    current_user: Annotated[User, Depends(get_current_user)]
):
    """
    Check health of a deployment.

    Performs health checks on the deployment to verify it's ready
    to receive traffic. This should be used before promoting a
    deployment.

    Returns:
        Health status with details about the deployment state.
    """
    service = DeploymentService(db)
    deployment = await service.get_deployment(deployment_id)

    # Verify deployment belongs to module
    if deployment.module_id != module_id:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Deployment not found for this module"
        )

    # Placeholder health check - in production, would verify:
    # - All pods/containers are running
    # - Health check endpoints returning 200
    # - Metrics within acceptable ranges
    return {
        "deployment_id": deployment.id,
        "version": deployment.version,
        "status": deployment.status.value,
        "health_check_passed": deployment.health_check_passed,
        "health_check_message": deployment.health_check_message,
        "traffic_weight": deployment.traffic_weight,
        "ready_for_promotion": deployment.health_check_passed and deployment.traffic_weight < 100
    }
