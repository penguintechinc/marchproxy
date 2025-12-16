"""
Configuration Builder Service

Builds comprehensive proxy configuration from database state.
Used by proxies to fetch their runtime configuration.
"""

import hashlib
import json
import logging
from datetime import datetime
from typing import Optional

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload

from app.models.sqlalchemy.cluster import Cluster
from app.models.sqlalchemy.service import Service
from app.models.sqlalchemy.mapping import Mapping
from app.models.sqlalchemy.certificate import Certificate

logger = logging.getLogger(__name__)


class ConfigBuilder:
    """Builds proxy configuration from database state"""

    def __init__(self, db: AsyncSession):
        self.db = db

    async def build_cluster_config(
        self,
        cluster: Cluster,
        include_certificates: bool = True
    ) -> dict:
        """
        Build complete configuration for a cluster

        Args:
            cluster: Cluster to build config for
            include_certificates: Include certificate data

        Returns:
            Complete configuration dict
        """
        # Fetch all related data
        services = await self._get_cluster_services(cluster.id)
        mappings = await self._get_cluster_mappings(cluster.id)
        certificates = None

        if include_certificates:
            certificates = await self._get_active_certificates()

        # Build configuration structure
        config_data = {
            "cluster": {
                "id": cluster.id,
                "name": cluster.name,
                "description": cluster.description,
                "max_proxies": cluster.max_proxies,
                "syslog_endpoint": cluster.syslog_endpoint,
                "log_auth": cluster.log_auth,
                "log_netflow": cluster.log_netflow,
                "log_debug": cluster.log_debug,
            },
            "services": self._serialize_services(services),
            "mappings": self._serialize_mappings(mappings),
            "certificates": self._serialize_certificates(certificates)
            if certificates else None,
            "logging": {
                "endpoint": cluster.syslog_endpoint,
                "auth": cluster.log_auth,
                "netflow": cluster.log_netflow,
                "debug": cluster.log_debug,
            },
            "generated_at": datetime.utcnow().isoformat(),
        }

        # Generate configuration version hash
        config_version = self._generate_config_hash(config_data)
        config_data["config_version"] = config_version

        return config_data

    async def _get_cluster_services(self, cluster_id: int) -> list[Service]:
        """Fetch all active services for a cluster"""
        stmt = select(Service).where(
            Service.cluster_id == cluster_id,
            Service.is_active == True  # noqa: E712
        ).order_by(Service.name)

        result = await self.db.execute(stmt)
        return list(result.scalars().all())

    async def _get_cluster_mappings(self, cluster_id: int) -> list[Mapping]:
        """Fetch all active mappings for a cluster"""
        stmt = select(Mapping).where(
            Mapping.cluster_id == cluster_id,
            Mapping.is_active == True  # noqa: E712
        ).order_by(Mapping.name)

        result = await self.db.execute(stmt)
        return list(result.scalars().all())

    async def _get_active_certificates(self) -> list[Certificate]:
        """Fetch all active, non-expired certificates"""
        stmt = select(Certificate).where(
            Certificate.is_active == True,  # noqa: E712
            Certificate.valid_until > datetime.utcnow()
        ).order_by(Certificate.name)

        result = await self.db.execute(stmt)
        return list(result.scalars().all())

    def _serialize_services(self, services: list[Service]) -> list[dict]:
        """Convert services to configuration format"""
        return [
            {
                "id": svc.id,
                "name": svc.name,
                "ip_fqdn": svc.ip_fqdn,
                "port": svc.port,
                "protocol": svc.protocol,
                "collection": svc.collection,
                "auth": {
                    "type": svc.auth_type,
                    "token_base64": svc.token_base64
                    if svc.auth_type == "base64" else None,
                    "jwt": {
                        "secret": svc.jwt_secret,
                        "expiry": svc.jwt_expiry,
                        "algorithm": svc.jwt_algorithm,
                    } if svc.auth_type == "jwt" else None,
                },
                "tls": {
                    "enabled": svc.tls_enabled,
                    "verify": svc.tls_verify,
                },
                "health_check": {
                    "enabled": svc.health_check_enabled,
                    "path": svc.health_check_path,
                    "interval": svc.health_check_interval,
                } if svc.health_check_enabled else None,
                "metadata": svc.extra_metadata,
            }
            for svc in services
        ]

    def _serialize_mappings(self, mappings: list[Mapping]) -> list[dict]:
        """Convert mappings to configuration format"""
        return [
            {
                "id": mapping.id,
                "name": mapping.name,
                "description": mapping.description,
                "source_services": self._parse_service_list(
                    mapping.source_services
                ),
                "dest_services": self._parse_service_list(
                    mapping.dest_services
                ),
                "protocols": mapping.protocols.split(","),
                "ports": self._parse_port_config(mapping.ports),
                "auth_required": mapping.auth_required,
            }
            for mapping in mappings
        ]

    def _serialize_certificates(
        self,
        certificates: list[Certificate]
    ) -> dict:
        """Convert certificates to configuration format"""
        return {
            cert.name: {
                "id": cert.id,
                "cert": cert.cert_data,
                "ca_chain": cert.ca_chain,
                "common_name": cert.common_name,
                "valid_until": cert.valid_until.isoformat(),
                "subject_alt_names": json.loads(cert.subject_alt_names)
                if cert.subject_alt_names else [],
            }
            for cert in certificates
        }

    def _parse_service_list(self, service_str: str) -> list:
        """
        Parse service list string to list of IDs or special values

        Supports:
        - "all" -> ["all"]
        - "1,2,3" -> [1, 2, 3]
        """
        if service_str.lower() == "all":
            return ["all"]

        try:
            return [int(s.strip()) for s in service_str.split(",")]
        except ValueError:
            logger.warning(f"Invalid service list: {service_str}")
            return []

    def _parse_port_config(self, port_str: str) -> list:
        """
        Parse port configuration string

        Supports:
        - Single port: "80" -> [80]
        - Range: "80-443" -> {"range": [80, 443]}
        - List: "80,443,8080" -> [80, 443, 8080]
        - Mixed: "80,443-8443,9000" -> [80, {"range": [443, 8443]}, 9000]
        """
        results = []

        for part in port_str.split(","):
            part = part.strip()

            if "-" in part:
                # Port range
                try:
                    start, end = part.split("-")
                    results.append({
                        "range": [int(start.strip()), int(end.strip())]
                    })
                except ValueError:
                    logger.warning(f"Invalid port range: {part}")
            else:
                # Single port
                try:
                    results.append(int(part))
                except ValueError:
                    logger.warning(f"Invalid port: {part}")

        return results

    def _generate_config_hash(self, config_data: dict) -> str:
        """
        Generate deterministic hash of configuration

        Used for version tracking and change detection.
        """
        # Create stable JSON representation
        config_json = json.dumps(
            config_data,
            sort_keys=True,
            default=str
        )

        # Generate MD5 hash (sufficient for version tracking)
        return hashlib.md5(config_json.encode()).hexdigest()

    async def get_service_by_name(
        self,
        cluster_id: int,
        service_name: str
    ) -> Optional[Service]:
        """Get service by name within a cluster"""
        stmt = select(Service).where(
            Service.cluster_id == cluster_id,
            Service.name == service_name,
            Service.is_active == True  # noqa: E712
        )

        result = await self.db.execute(stmt)
        return result.scalar_one_or_none()

    async def validate_mapping(
        self,
        cluster_id: int,
        source_services: str,
        dest_services: str
    ) -> tuple[bool, Optional[str]]:
        """
        Validate that mapping service IDs exist

        Returns:
            Tuple of (is_valid, error_message)
        """
        # Parse service lists
        source_list = self._parse_service_list(source_services)
        dest_list = self._parse_service_list(dest_services)

        # "all" is always valid
        if "all" in source_list:
            source_list = ["all"]
        if "all" in dest_list:
            dest_list = ["all"]

        # Get numeric IDs
        source_ids = [s for s in source_list if isinstance(s, int)]
        dest_ids = [d for d in dest_list if isinstance(d, int)]

        # Validate source services exist
        if source_ids and "all" not in source_list:
            stmt = select(Service.id).where(
                Service.cluster_id == cluster_id,
                Service.id.in_(source_ids),
                Service.is_active == True  # noqa: E712
            )
            result = await self.db.execute(stmt)
            found_ids = set(result.scalars().all())

            missing = set(source_ids) - found_ids
            if missing:
                return False, f"Source services not found: {missing}"

        # Validate destination services exist
        if dest_ids and "all" not in dest_list:
            stmt = select(Service.id).where(
                Service.cluster_id == cluster_id,
                Service.id.in_(dest_ids),
                Service.is_active == True  # noqa: E712
            )
            result = await self.db.execute(stmt)
            found_ids = set(result.scalars().all())

            missing = set(dest_ids) - found_ids
            if missing:
                return False, f"Destination services not found: {missing}"

        return True, None
