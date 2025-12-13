"""
Cluster Service - Business logic for cluster management

Handles cluster creation, updates, API key rotation, and license validation.
"""

import hashlib
import logging
import secrets
from datetime import datetime
from typing import Optional

from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession
from fastapi import HTTPException, status

from app.models.sqlalchemy.cluster import Cluster, UserClusterAssignment
from app.models.sqlalchemy.proxy import ProxyServer
from app.models.sqlalchemy.user import User
from app.schemas.cluster import ClusterCreate, ClusterUpdate
from app.core.license import license_validator

logger = logging.getLogger(__name__)


def generate_api_key() -> str:
    """Generate a secure API key"""
    return secrets.token_urlsafe(48)


def hash_api_key(api_key: str) -> str:
    """Hash API key for storage"""
    return hashlib.sha256(api_key.encode()).hexdigest()


class ClusterService:
    """Business logic for cluster management"""

    def __init__(self, db: AsyncSession):
        self.db = db

    async def validate_cluster_limit(self) -> None:
        """
        Validate that creating a new cluster doesn't exceed license limits

        Raises:
            HTTPException: If cluster limit would be exceeded
        """
        # Get current cluster count
        stmt = select(func.count()).select_from(Cluster).where(Cluster.is_active == True)
        result = await self.db.execute(stmt)
        current_count = result.scalar() or 0

        # Get license info
        license_info = await license_validator.validate_license()

        # Community tier: Only 1 cluster (the default cluster)
        # Enterprise tier: Check if multi_cluster feature is enabled
        if license_info.tier.value == "community":
            if current_count >= 1:
                raise HTTPException(
                    status_code=status.HTTP_402_PAYMENT_REQUIRED,
                    detail="Community tier limited to 1 cluster. Upgrade to Enterprise for multi-cluster support."
                )
        else:
            # Enterprise: Check for multi_cluster feature
            has_multi_cluster = await license_validator.check_feature("multi_cluster")
            if not has_multi_cluster and current_count >= 1:
                raise HTTPException(
                    status_code=status.HTTP_402_PAYMENT_REQUIRED,
                    detail="Your Enterprise license does not include multi-cluster support."
                )

    async def get_cluster_proxy_count(self, cluster_id: int) -> int:
        """
        Get the current number of active proxies for a cluster

        Args:
            cluster_id: The cluster ID

        Returns:
            Number of active proxies
        """
        stmt = select(func.count()).select_from(ProxyServer).where(
            ProxyServer.cluster_id == cluster_id,
            ProxyServer.is_active == True
        )
        result = await self.db.execute(stmt)
        return result.scalar() or 0

    async def validate_proxy_limit(self, cluster_id: int) -> None:
        """
        Validate that cluster hasn't exceeded proxy limits

        Args:
            cluster_id: The cluster ID

        Raises:
            HTTPException: If proxy limit exceeded
        """
        # Get current proxy count for this cluster
        current_count = await self.get_cluster_proxy_count(cluster_id)

        # Get license info
        license_info = await license_validator.validate_license()

        # Check against license limits
        if current_count >= license_info.max_proxies:
            raise HTTPException(
                status_code=status.HTTP_402_PAYMENT_REQUIRED,
                detail=f"Proxy limit reached ({license_info.max_proxies}). "
                       f"Upgrade license for more proxies."
            )

    async def create_cluster(
        self,
        cluster_data: ClusterCreate,
        current_user: User
    ) -> tuple[Cluster, str]:
        """
        Create a new cluster with license validation

        Args:
            cluster_data: Cluster creation data
            current_user: User creating the cluster

        Returns:
            Tuple of (Cluster object, API key string)

        Raises:
            HTTPException: On validation errors
        """
        # Validate cluster limits
        await self.validate_cluster_limit()

        # Check if cluster name already exists
        stmt = select(Cluster).where(Cluster.name == cluster_data.name)
        existing = await self.db.execute(stmt)
        if existing.scalar_one_or_none():
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail=f"Cluster with name '{cluster_data.name}' already exists"
            )

        # Generate API key
        api_key = generate_api_key()
        api_key_hash = hash_api_key(api_key)

        # Get license info to set appropriate max_proxies
        license_info = await license_validator.validate_license()
        max_proxies = min(cluster_data.max_proxies, license_info.max_proxies)

        # Create cluster
        new_cluster = Cluster(
            name=cluster_data.name,
            description=cluster_data.description,
            api_key_hash=api_key_hash,
            syslog_endpoint=cluster_data.syslog_endpoint,
            log_auth=cluster_data.log_auth,
            log_netflow=cluster_data.log_netflow,
            log_debug=cluster_data.log_debug,
            max_proxies=max_proxies,
            is_active=True,
            is_default=False,
            created_by=current_user.id,
            created_at=datetime.utcnow()
        )

        self.db.add(new_cluster)
        await self.db.commit()
        await self.db.refresh(new_cluster)

        logger.info(
            f"Cluster created: {new_cluster.name} (ID: {new_cluster.id}) "
            f"by user {current_user.username}"
        )

        return new_cluster, api_key

    async def update_cluster(
        self,
        cluster_id: int,
        cluster_data: ClusterUpdate,
        current_user: User
    ) -> Cluster:
        """
        Update cluster details

        Args:
            cluster_id: Cluster ID to update
            cluster_data: Update data
            current_user: User performing the update

        Returns:
            Updated Cluster object

        Raises:
            HTTPException: If cluster not found
        """
        stmt = select(Cluster).where(Cluster.id == cluster_id)
        result = await self.db.execute(stmt)
        cluster = result.scalar_one_or_none()

        if not cluster:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail="Cluster not found"
            )

        # Update fields
        update_data = cluster_data.model_dump(exclude_unset=True)

        # Validate max_proxies against license if being updated
        if "max_proxies" in update_data:
            license_info = await license_validator.validate_license()
            if update_data["max_proxies"] > license_info.max_proxies:
                update_data["max_proxies"] = license_info.max_proxies
                logger.warning(
                    f"max_proxies capped at {license_info.max_proxies} "
                    f"per license limit"
                )

        for field, value in update_data.items():
            setattr(cluster, field, value)

        cluster.updated_at = datetime.utcnow()
        await self.db.commit()
        await self.db.refresh(cluster)

        logger.info(
            f"Cluster updated: {cluster.name} (ID: {cluster.id}) "
            f"by user {current_user.username}"
        )

        return cluster

    async def rotate_api_key(self, cluster_id: int, current_user: User) -> tuple[Cluster, str]:
        """
        Rotate cluster API key

        Args:
            cluster_id: Cluster ID
            current_user: User performing the rotation

        Returns:
            Tuple of (Cluster object, new API key)

        Raises:
            HTTPException: If cluster not found
        """
        stmt = select(Cluster).where(Cluster.id == cluster_id)
        result = await self.db.execute(stmt)
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

        await self.db.commit()

        logger.warning(
            f"API key rotated for cluster: {cluster.name} (ID: {cluster.id}) "
            f"by user {current_user.username}"
        )

        return cluster, new_api_key

    async def delete_cluster(
        self,
        cluster_id: int,
        current_user: User,
        permanent: bool = False
    ) -> None:
        """
        Delete or deactivate a cluster

        Args:
            cluster_id: Cluster ID
            current_user: User performing the deletion
            permanent: If True, permanently delete; otherwise soft delete

        Raises:
            HTTPException: If cluster not found or is default cluster
        """
        stmt = select(Cluster).where(Cluster.id == cluster_id)
        result = await self.db.execute(stmt)
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
            await self.db.delete(cluster)
            logger.warning(
                f"Cluster permanently deleted: {cluster.name} (ID: {cluster.id}) "
                f"by user {current_user.username}"
            )
        else:
            cluster.is_active = False
            cluster.updated_at = datetime.utcnow()
            logger.info(
                f"Cluster deactivated: {cluster.name} (ID: {cluster.id}) "
                f"by user {current_user.username}"
            )

        await self.db.commit()

    async def check_user_cluster_access(
        self,
        user: User,
        cluster_id: int
    ) -> bool:
        """
        Check if a user has access to a cluster

        Args:
            user: User to check
            cluster_id: Cluster ID

        Returns:
            True if user has access, False otherwise
        """
        # Admins have access to all clusters
        if user.is_admin:
            return True

        # Check for active assignment
        stmt = select(UserClusterAssignment).where(
            UserClusterAssignment.user_id == user.id,
            UserClusterAssignment.cluster_id == cluster_id,
            UserClusterAssignment.is_active == True
        )
        result = await self.db.execute(stmt)
        assignment = result.scalar_one_or_none()

        return assignment is not None
