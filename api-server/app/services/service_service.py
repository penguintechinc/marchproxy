"""
Service Service - Business logic for service management

Handles service creation, updates, token management, and service-to-service mappings.
"""

import base64
import logging
import secrets
from datetime import datetime
from typing import Optional

from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession
from fastapi import HTTPException, status

from app.models.sqlalchemy.service import Service, UserServiceAssignment
from app.models.sqlalchemy.cluster import Cluster, UserClusterAssignment
from app.models.sqlalchemy.user import User
from app.schemas.service import ServiceCreate, ServiceUpdate
from app.core.license import license_validator

logger = logging.getLogger(__name__)


def generate_base64_token() -> str:
    """Generate a secure Base64 token"""
    return base64.b64encode(secrets.token_bytes(32)).decode()


def generate_jwt_secret() -> str:
    """Generate a secure JWT secret"""
    return secrets.token_urlsafe(64)


class ServiceService:
    """Business logic for service management"""

    def __init__(self, db: AsyncSession):
        self.db = db

    async def validate_cluster_access(
        self,
        user: User,
        cluster_id: int
    ) -> Cluster:
        """
        Validate user has access to a cluster and return the cluster

        Args:
            user: User to validate
            cluster_id: Cluster ID

        Returns:
            Cluster object

        Raises:
            HTTPException: If access denied or cluster not found
        """
        # Check if cluster exists
        stmt = select(Cluster).where(Cluster.id == cluster_id)
        result = await self.db.execute(stmt)
        cluster = result.scalar_one_or_none()

        if not cluster:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail="Cluster not found"
            )

        # Admins have access to all clusters
        if user.is_admin:
            return cluster

        # Check user has access to this cluster
        stmt = select(UserClusterAssignment).where(
            UserClusterAssignment.user_id == user.id,
            UserClusterAssignment.cluster_id == cluster_id,
            UserClusterAssignment.is_active == True
        )
        result = await self.db.execute(stmt)
        assignment = result.scalar_one_or_none()

        if not assignment:
            raise HTTPException(
                status_code=status.HTTP_403_FORBIDDEN,
                detail="No access to this cluster"
            )

        return cluster

    async def validate_service_name_unique(self, name: str) -> None:
        """
        Validate that service name is unique

        Args:
            name: Service name to check

        Raises:
            HTTPException: If name already exists
        """
        stmt = select(Service).where(Service.name == name)
        result = await self.db.execute(stmt)
        existing = result.scalar_one_or_none()

        if existing:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail=f"Service with name '{name}' already exists"
            )

    async def create_service(
        self,
        service_data: ServiceCreate,
        current_user: User
    ) -> Service:
        """
        Create a new service with auto-generated auth tokens

        Args:
            service_data: Service creation data
            current_user: User creating the service

        Returns:
            Created Service object

        Raises:
            HTTPException: On validation errors
        """
        # Validate cluster access
        cluster = await self.validate_cluster_access(
            current_user,
            service_data.cluster_id
        )

        # Validate service name uniqueness
        await self.validate_service_name_unique(service_data.name)

        # Generate auth tokens based on auth_type
        token_base64 = None
        jwt_secret = None

        if service_data.auth_type == "base64":
            token_base64 = generate_base64_token()
        elif service_data.auth_type == "jwt":
            jwt_secret = generate_jwt_secret()

        # Create service
        new_service = Service(
            name=service_data.name,
            ip_fqdn=service_data.ip_fqdn,
            port=service_data.port,
            protocol=service_data.protocol,
            collection=service_data.collection,
            cluster_id=service_data.cluster_id,
            auth_type=service_data.auth_type,
            token_base64=token_base64,
            jwt_secret=jwt_secret,
            jwt_expiry=service_data.jwt_expiry or 3600,
            jwt_algorithm=service_data.jwt_algorithm or "HS256",
            tls_enabled=service_data.tls_enabled,
            tls_verify=service_data.tls_verify,
            health_check_enabled=service_data.health_check_enabled,
            health_check_path=service_data.health_check_path,
            health_check_interval=service_data.health_check_interval,
            is_active=True,
            created_by=current_user.id,
            created_at=datetime.utcnow()
        )

        self.db.add(new_service)
        await self.db.commit()
        await self.db.refresh(new_service)

        # Auto-assign to creator if not admin
        if not current_user.is_admin:
            assignment = UserServiceAssignment(
                user_id=current_user.id,
                service_id=new_service.id,
                assigned_by=current_user.id,
                assigned_at=datetime.utcnow(),
                is_active=True
            )
            self.db.add(assignment)
            await self.db.commit()

        logger.info(
            f"Service created: {new_service.name} (ID: {new_service.id}) "
            f"by user {current_user.username}"
        )

        return new_service

    async def get_service(
        self,
        service_id: int,
        current_user: User
    ) -> Service:
        """
        Get service by ID with access validation

        Args:
            service_id: Service ID
            current_user: User requesting the service

        Returns:
            Service object

        Raises:
            HTTPException: If not found or access denied
        """
        stmt = select(Service).where(Service.id == service_id)

        # Non-admin users need assignment check
        if not current_user.is_admin:
            stmt = stmt.join(UserServiceAssignment).where(
                UserServiceAssignment.user_id == current_user.id,
                UserServiceAssignment.is_active == True
            )

        result = await self.db.execute(stmt)
        service = result.scalar_one_or_none()

        if not service:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail="Service not found or access denied"
            )

        return service

    async def update_service(
        self,
        service_id: int,
        service_data: ServiceUpdate,
        current_user: User
    ) -> Service:
        """
        Update service details (excludes auth tokens)

        Args:
            service_id: Service ID to update
            service_data: Update data
            current_user: User performing the update

        Returns:
            Updated Service object

        Raises:
            HTTPException: If not found or access denied
        """
        # Get service with access validation
        service = await self.get_service(service_id, current_user)

        # Update fields
        update_data = service_data.model_dump(exclude_unset=True)
        for field, value in update_data.items():
            setattr(service, field, value)

        service.updated_at = datetime.utcnow()
        await self.db.commit()
        await self.db.refresh(service)

        logger.info(
            f"Service updated: {service.name} (ID: {service.id}) "
            f"by user {current_user.username}"
        )

        return service

    async def rotate_token(
        self,
        service_id: int,
        auth_type: str,
        current_user: User
    ) -> tuple[Service, Optional[str], Optional[str]]:
        """
        Rotate service authentication token

        Args:
            service_id: Service ID
            auth_type: Type of token to rotate (base64 or jwt)
            current_user: User performing the rotation

        Returns:
            Tuple of (Service, new_token, new_jwt_secret)

        Raises:
            HTTPException: If not found or access denied
        """
        # Get service with access validation
        service = await self.get_service(service_id, current_user)

        new_token = None
        new_jwt_secret = None

        if auth_type == "base64":
            new_token = generate_base64_token()
            service.token_base64 = new_token
            service.auth_type = "base64"
        elif auth_type == "jwt":
            new_jwt_secret = generate_jwt_secret()
            service.jwt_secret = new_jwt_secret
            service.auth_type = "jwt"

        service.updated_at = datetime.utcnow()
        await self.db.commit()

        logger.warning(
            f"Auth token rotated for service: {service.name} (ID: {service.id}) "
            f"by user {current_user.username}"
        )

        return service, new_token, new_jwt_secret

    async def delete_service(
        self,
        service_id: int,
        current_user: User,
        permanent: bool = False
    ) -> int:
        """
        Delete or deactivate a service

        Args:
            service_id: Service ID
            current_user: User performing the deletion
            permanent: If True, permanently delete; otherwise soft delete

        Returns:
            Cluster ID of the deleted service (for xDS updates)

        Raises:
            HTTPException: If not found or access denied
        """
        # Get service with access validation
        service = await self.get_service(service_id, current_user)
        cluster_id = service.cluster_id

        if permanent:
            await self.db.delete(service)
            logger.warning(
                f"Service permanently deleted: {service.name} (ID: {service.id}) "
                f"by user {current_user.username}"
            )
        else:
            service.is_active = False
            service.updated_at = datetime.utcnow()
            logger.info(
                f"Service deactivated: {service.name} (ID: {service.id}) "
                f"by user {current_user.username}"
            )

        await self.db.commit()
        return cluster_id

    async def validate_auth_token_exclusivity(
        self,
        cluster_id: int
    ) -> None:
        """
        Validate that all services in a cluster use the same auth type
        (Base64 and JWT are mutually exclusive per cluster)

        Args:
            cluster_id: Cluster ID to check

        Raises:
            HTTPException: If mixed auth types detected
        """
        stmt = select(Service.auth_type, func.count()).where(
            Service.cluster_id == cluster_id,
            Service.is_active == True,
            Service.auth_type.in_(["base64", "jwt"])
        ).group_by(Service.auth_type)

        result = await self.db.execute(stmt)
        auth_types = result.all()

        if len(auth_types) > 1:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="Cannot mix Base64 and JWT authentication in the same cluster"
            )

    async def get_service_config_for_proxy(
        self,
        cluster_id: int
    ) -> list[dict]:
        """
        Generate service configuration for proxy consumption

        Args:
            cluster_id: Cluster ID

        Returns:
            List of service configurations
        """
        stmt = select(Service).where(
            Service.cluster_id == cluster_id,
            Service.is_active == True
        )
        result = await self.db.execute(stmt)
        services = result.scalars().all()

        config = []
        for svc in services:
            service_config = {
                "id": svc.id,
                "name": svc.name,
                "ip_fqdn": svc.ip_fqdn,
                "port": svc.port,
                "protocol": svc.protocol,
                "collection": svc.collection,
                "auth_type": svc.auth_type,
                "tls_enabled": svc.tls_enabled,
                "tls_verify": svc.tls_verify,
                "health_check_enabled": svc.health_check_enabled,
                "health_check_path": svc.health_check_path,
                "health_check_interval": svc.health_check_interval,
            }

            # Include auth tokens for proxy use
            if svc.auth_type == "base64":
                service_config["token_base64"] = svc.token_base64
            elif svc.auth_type == "jwt":
                service_config["jwt_secret"] = svc.jwt_secret
                service_config["jwt_expiry"] = svc.jwt_expiry
                service_config["jwt_algorithm"] = svc.jwt_algorithm

            config.append(service_config)

        return config
