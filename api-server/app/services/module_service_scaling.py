"""
Module Service - Scaling and Deployment Logic

Additional business logic for auto-scaling policies and blue/green deployments.
Extends ModuleService functionality.
"""

import logging
from datetime import datetime
from typing import Optional, List

from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession
from fastapi import HTTPException, status

from app.models.sqlalchemy.module import (
    Module, ScalingPolicy, Deployment,
    DeploymentStatus
)
from app.schemas.module import (
    ScalingPolicyCreate, ScalingPolicyUpdate,
    DeploymentCreate, DeploymentUpdate
)
from app.services.grpc_client import grpc_client_manager

logger = logging.getLogger(__name__)


class ScalingService:
    """Business logic for auto-scaling policies"""

    def __init__(self, db: AsyncSession):
        self.db = db

    async def create_or_update_policy(
        self,
        module_id: int,
        policy_data: ScalingPolicyCreate | ScalingPolicyUpdate
    ) -> ScalingPolicy:
        """
        Create or update scaling policy for module

        Args:
            module_id: Module ID
            policy_data: Scaling policy data

        Returns:
            ScalingPolicy object
        """
        # Check if policy already exists
        stmt = select(ScalingPolicy).where(ScalingPolicy.module_id == module_id)
        result = await self.db.execute(stmt)
        policy = result.scalar_one_or_none()

        if policy:
            # Update existing policy
            update_data = policy_data.model_dump(exclude_unset=True)
            for field, value in update_data.items():
                setattr(policy, field, value)
            policy.updated_at = datetime.utcnow()
            logger.info(f"Scaling policy updated for module {module_id}")
        else:
            # Create new policy
            policy = ScalingPolicy(
                module_id=module_id,
                **policy_data.model_dump(),
                created_at=datetime.utcnow()
            )
            self.db.add(policy)
            logger.info(f"Scaling policy created for module {module_id}")

        await self.db.commit()
        await self.db.refresh(policy)
        return policy

    async def get_policy(self, module_id: int) -> Optional[ScalingPolicy]:
        """Get scaling policy for module"""
        stmt = select(ScalingPolicy).where(ScalingPolicy.module_id == module_id)
        result = await self.db.execute(stmt)
        return result.scalar_one_or_none()

    async def delete_policy(self, module_id: int) -> None:
        """Delete scaling policy for module"""
        stmt = select(ScalingPolicy).where(ScalingPolicy.module_id == module_id)
        result = await self.db.execute(stmt)
        policy = result.scalar_one_or_none()

        if not policy:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail="Scaling policy not found"
            )

        await self.db.delete(policy)
        await self.db.commit()
        logger.info(f"Scaling policy deleted for module {module_id}")


class DeploymentService:
    """Business logic for blue/green deployments"""

    def __init__(self, db: AsyncSession):
        self.db = db

    async def create_deployment(
        self,
        module_id: int,
        deployment_data: DeploymentCreate,
        current_user_id: int
    ) -> Deployment:
        """
        Create a new deployment for module

        Args:
            module_id: Module ID
            deployment_data: Deployment data
            current_user_id: User ID creating deployment

        Returns:
            Deployment object
        """
        # Verify module exists
        stmt = select(Module).where(Module.id == module_id)
        result = await self.db.execute(stmt)
        module = result.scalar_one_or_none()

        if not module:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail="Module not found"
            )

        # Get current active deployment (if any)
        current_deployment = await self._get_active_deployment(module_id)

        # Create new deployment
        new_deployment = Deployment(
            module_id=module_id,
            version=deployment_data.version,
            image=deployment_data.image,
            config=deployment_data.config,
            environment=deployment_data.environment,
            traffic_weight=deployment_data.traffic_weight,
            status=DeploymentStatus.PENDING,
            previous_deployment_id=current_deployment.id if current_deployment else None,
            deployed_by=current_user_id,
            deployed_at=datetime.utcnow()
        )

        self.db.add(new_deployment)
        await self.db.commit()
        await self.db.refresh(new_deployment)

        logger.info(
            f"Deployment created: {new_deployment.version} "
            f"for module {module_id} (ID: {new_deployment.id})"
        )

        return new_deployment

    async def list_deployments(
        self,
        module_id: int,
        skip: int = 0,
        limit: int = 100
    ) -> tuple[List[Deployment], int]:
        """List all deployments for a module"""
        query = select(Deployment).where(Deployment.module_id == module_id)

        # Get total count
        count_query = select(func.count()).select_from(query.subquery())
        total_result = await self.db.execute(count_query)
        total = total_result.scalar() or 0

        # Get paginated results
        query = query.offset(skip).limit(limit).order_by(Deployment.deployed_at.desc())
        result = await self.db.execute(query)
        deployments = result.scalars().all()

        return list(deployments), total

    async def get_deployment(self, deployment_id: int) -> Deployment:
        """Get deployment by ID"""
        stmt = select(Deployment).where(Deployment.id == deployment_id)
        result = await self.db.execute(stmt)
        deployment = result.scalar_one_or_none()

        if not deployment:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail="Deployment not found"
            )

        return deployment

    async def _get_active_deployment(self, module_id: int) -> Optional[Deployment]:
        """Get currently active deployment for module"""
        stmt = select(Deployment).where(
            Deployment.module_id == module_id,
            Deployment.status == DeploymentStatus.ACTIVE
        ).order_by(Deployment.deployed_at.desc())

        result = await self.db.execute(stmt)
        return result.first()

    async def promote_deployment(
        self,
        deployment_id: int,
        traffic_weight: float = 100.0,
        incremental: bool = False
    ) -> Deployment:
        """
        Promote deployment (increase traffic weight)

        Args:
            deployment_id: Deployment ID to promote
            traffic_weight: Target traffic weight (0-100%)
            incremental: If True, gradually shift traffic

        Returns:
            Updated Deployment object
        """
        deployment = await self.get_deployment(deployment_id)

        # Validate traffic weight
        if not 0 <= traffic_weight <= 100:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="Traffic weight must be between 0 and 100"
            )

        if incremental:
            # Gradual traffic shift (implementation would be more complex)
            deployment.traffic_weight = min(deployment.traffic_weight + 10, traffic_weight)
        else:
            deployment.traffic_weight = traffic_weight

        # Update status
        if deployment.traffic_weight == 100:
            deployment.status = DeploymentStatus.ACTIVE
            deployment.completed_at = datetime.utcnow()

            # Deactivate previous deployment
            if deployment.previous_deployment_id:
                stmt = select(Deployment).where(
                    Deployment.id == deployment.previous_deployment_id
                )
                result = await self.db.execute(stmt)
                prev = result.scalar_one_or_none()
                if prev:
                    prev.status = DeploymentStatus.INACTIVE
                    prev.traffic_weight = 0.0

        elif deployment.traffic_weight > 0:
            deployment.status = DeploymentStatus.ROLLING_OUT

        await self.db.commit()
        await self.db.refresh(deployment)

        logger.info(
            f"Deployment promoted: {deployment.version} "
            f"(ID: {deployment.id}) to {deployment.traffic_weight}% traffic"
        )

        return deployment

    async def rollback_deployment(
        self,
        deployment_id: int,
        reason: str
    ) -> Deployment:
        """
        Rollback deployment to previous version

        Args:
            deployment_id: Deployment ID to rollback
            reason: Rollback reason

        Returns:
            Previous deployment (now active)
        """
        deployment = await self.get_deployment(deployment_id)

        if not deployment.previous_deployment_id:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="No previous deployment to rollback to"
            )

        # Mark current deployment as rolled back
        deployment.status = DeploymentStatus.ROLLED_BACK
        deployment.traffic_weight = 0.0
        deployment.health_check_message = f"Rolled back: {reason}"

        # Restore previous deployment
        stmt = select(Deployment).where(
            Deployment.id == deployment.previous_deployment_id
        )
        result = await self.db.execute(stmt)
        previous = result.scalar_one_or_none()

        if not previous:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail="Previous deployment not found"
            )

        previous.status = DeploymentStatus.ACTIVE
        previous.traffic_weight = 100.0

        await self.db.commit()
        await self.db.refresh(previous)

        logger.warning(
            f"Deployment rolled back: {deployment.version} -> {previous.version}. "
            f"Reason: {reason}"
        )

        return previous

    async def update_deployment(
        self,
        deployment_id: int,
        deployment_data: DeploymentUpdate
    ) -> Deployment:
        """Update deployment configuration"""
        deployment = await self.get_deployment(deployment_id)

        # Update fields
        update_data = deployment_data.model_dump(exclude_unset=True)
        for field, value in update_data.items():
            setattr(deployment, field, value)

        await self.db.commit()
        await self.db.refresh(deployment)

        logger.info(f"Deployment updated: {deployment.version} (ID: {deployment.id})")
        return deployment
