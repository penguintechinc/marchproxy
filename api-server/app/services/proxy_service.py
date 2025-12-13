"""
Proxy Service - Business logic for proxy registration and management

Handles proxy registration, heartbeat tracking, license validation,
and proxy count enforcement.
"""

import hashlib
import logging
from datetime import datetime, timedelta
from typing import Optional

from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.config import settings
from app.core.license import license_validator
from app.models.sqlalchemy.cluster import Cluster
from app.models.sqlalchemy.proxy import ProxyServer, ProxyMetrics

logger = logging.getLogger(__name__)


def hash_api_key(api_key: str) -> str:
    """Hash API key for secure storage and verification"""
    return hashlib.sha256(api_key.encode()).hexdigest()


class ProxyServiceError(Exception):
    """Base exception for proxy service errors"""
    pass


class InvalidAPIKeyError(ProxyServiceError):
    """Raised when cluster API key is invalid"""
    pass


class ProxyLimitExceededError(ProxyServiceError):
    """Raised when proxy count limit is exceeded"""
    pass


class ProxyService:
    """Service for managing proxy registration and lifecycle"""

    def __init__(self, db: AsyncSession):
        self.db = db

    async def verify_cluster_api_key(self, api_key: str) -> Cluster:
        """
        Verify cluster API key and return cluster

        Args:
            api_key: Cluster API key to verify

        Returns:
            Cluster object if valid

        Raises:
            InvalidAPIKeyError: If API key is invalid or cluster is inactive
        """
        api_key_hash = hash_api_key(api_key)
        stmt = select(Cluster).where(
            Cluster.api_key_hash == api_key_hash,
            Cluster.is_active == True  # noqa: E712
        )
        cluster = (await self.db.execute(stmt)).scalar_one_or_none()

        if not cluster:
            logger.warning(f"Invalid API key attempt: {api_key[:8]}...")
            raise InvalidAPIKeyError("Invalid cluster API key")

        return cluster

    async def check_proxy_limit(self, cluster: Cluster) -> int:
        """
        Check current proxy count against cluster and license limits

        Args:
            cluster: Cluster to check

        Returns:
            Current active proxy count

        Raises:
            ProxyLimitExceededError: If limit is exceeded
        """
        stmt = select(func.count()).select_from(ProxyServer).where(
            ProxyServer.cluster_id == cluster.id,
            ProxyServer.status != "offline"
        )
        proxy_count = (await self.db.execute(stmt)).scalar() or 0

        # Get license limits
        license_info = await license_validator.validate_license()

        # Enforce the lower of cluster.max_proxies or license limit
        effective_limit = min(cluster.max_proxies, license_info.max_proxies)

        if proxy_count >= effective_limit:
            raise ProxyLimitExceededError(
                f"Proxy limit reached ({effective_limit}). "
                f"License tier: {license_info.tier.value}, "
                f"max proxies: {license_info.max_proxies}"
            )

        return proxy_count

    async def register_proxy(
        self,
        cluster: Cluster,
        name: str,
        hostname: str,
        ip_address: str,
        port: int,
        version: Optional[str] = None,
        capabilities: Optional[dict] = None
    ) -> ProxyServer:
        """
        Register or update a proxy server

        Args:
            cluster: Cluster to register in
            name: Proxy name
            hostname: Proxy hostname
            ip_address: Proxy IP address
            port: Proxy port
            version: Proxy version
            capabilities: Proxy capabilities dict

        Returns:
            ProxyServer object

        Raises:
            ProxyLimitExceededError: If proxy limit exceeded
        """
        # Check if proxy already exists (re-registration)
        stmt = select(ProxyServer).where(
            ProxyServer.name == name,
            ProxyServer.cluster_id == cluster.id
        )
        existing = (await self.db.execute(stmt)).scalar_one_or_none()

        if existing:
            # Update existing proxy
            existing.hostname = hostname
            existing.ip_address = ip_address
            existing.port = port
            existing.version = version
            existing.capabilities = capabilities
            existing.status = "active"
            existing.last_seen = datetime.utcnow()
            await self.db.commit()
            await self.db.refresh(existing)

            logger.info(
                f"Proxy re-registered: {name} in cluster {cluster.name}"
            )
            return existing

        # Check proxy count limit for new registrations
        # This will raise ProxyLimitExceededError if limit exceeded
        current_count = await self.check_proxy_limit(cluster)

        # Create new proxy
        new_proxy = ProxyServer(
            name=name,
            hostname=hostname,
            ip_address=ip_address,
            port=port,
            cluster_id=cluster.id,
            version=version,
            capabilities=capabilities,
            status="active",
            registered_at=datetime.utcnow(),
            last_seen=datetime.utcnow()
        )
        self.db.add(new_proxy)
        await self.db.commit()
        await self.db.refresh(new_proxy)

        logger.info(f"Proxy registered: {name} in cluster {cluster.name}")
        return new_proxy

    async def update_heartbeat(
        self,
        proxy_id: int,
        status: str,
        config_version: Optional[str] = None,
        metrics: Optional[dict] = None
    ) -> ProxyServer:
        """
        Update proxy heartbeat and optionally record metrics

        Args:
            proxy_id: Proxy ID
            status: Current proxy status
            config_version: Current config version
            metrics: Optional metrics dict

        Returns:
            Updated ProxyServer object
        """
        stmt = select(ProxyServer).where(ProxyServer.id == proxy_id)
        proxy = (await self.db.execute(stmt)).scalar_one_or_none()

        if not proxy:
            raise ValueError(f"Proxy {proxy_id} not found")

        # Update heartbeat
        proxy.status = status
        proxy.last_seen = datetime.utcnow()
        if config_version:
            proxy.config_version = config_version

        # Store metrics if provided
        if metrics:
            proxy_metrics = ProxyMetrics(
                proxy_id=proxy.id,
                timestamp=datetime.utcnow(),
                **metrics
            )
            self.db.add(proxy_metrics)

        await self.db.commit()
        await self.db.refresh(proxy)

        return proxy

    async def deregister_proxy(self, proxy_id: int) -> bool:
        """
        Deregister a proxy (set status to offline)

        Args:
            proxy_id: Proxy ID to deregister

        Returns:
            True if successful
        """
        stmt = select(ProxyServer).where(ProxyServer.id == proxy_id)
        proxy = (await self.db.execute(stmt)).scalar_one_or_none()

        if not proxy:
            return False

        proxy.status = "offline"
        proxy.last_seen = datetime.utcnow()
        await self.db.commit()

        logger.info(f"Proxy deregistered: {proxy.name}")
        return True

    async def get_stale_proxies(
        self,
        cluster_id: Optional[int] = None,
        threshold_minutes: int = 5
    ) -> list[ProxyServer]:
        """
        Get proxies that haven't sent heartbeat recently

        Args:
            cluster_id: Optional cluster filter
            threshold_minutes: Minutes since last heartbeat

        Returns:
            List of stale proxies
        """
        threshold_time = datetime.utcnow() - timedelta(minutes=threshold_minutes)
        stmt = select(ProxyServer).where(
            ProxyServer.last_seen < threshold_time,
            ProxyServer.status == "active"
        )

        if cluster_id:
            stmt = stmt.where(ProxyServer.cluster_id == cluster_id)

        proxies = (await self.db.execute(stmt)).scalars().all()
        return list(proxies)

    async def mark_stale_proxies_offline(
        self,
        threshold_minutes: int = 5
    ) -> int:
        """
        Mark stale proxies as offline

        Args:
            threshold_minutes: Minutes since last heartbeat

        Returns:
            Number of proxies marked offline
        """
        stale_proxies = await self.get_stale_proxies(
            threshold_minutes=threshold_minutes
        )

        count = 0
        for proxy in stale_proxies:
            proxy.status = "degraded"
            count += 1

        if count > 0:
            await self.db.commit()
            logger.info(f"Marked {count} proxies as degraded")

        return count
