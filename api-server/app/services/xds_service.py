"""
xDS Service - Complete Integration with FastAPI and Database

This service manages the complete lifecycle of xDS configuration updates,
translating MarchProxy database models into Envoy xDS configurations.
"""

import asyncio
import logging
from typing import List, Dict, Any, Optional
from datetime import datetime
import httpx
from sqlalchemy.orm import Session

logger = logging.getLogger(__name__)


class XDSService:
    """
    Complete xDS Service for MarchProxy

    Manages translation of database models to Envoy xDS configurations
    and communicates with the Go xDS control plane.
    """

    def __init__(self, xds_server_url: str = "http://localhost:19000"):
        """
        Initialize the xDS service

        Args:
            xds_server_url: URL of the Go xDS control plane HTTP API
        """
        self.xds_server_url = xds_server_url
        self.client = httpx.AsyncClient(timeout=30.0)
        self._update_lock = asyncio.Lock()
        self._last_update = None
        self._update_count = 0
        self._last_error = None

    async def close(self):
        """Close the HTTP client"""
        await self.client.aclose()

    async def update_envoy_config(
        self,
        cluster_id: int,
        db: Session
    ) -> bool:
        """
        Update Envoy configuration for a specific cluster

        Args:
            cluster_id: The cluster ID to update
            db: Database session

        Returns:
            True if successful, False otherwise
        """
        async with self._update_lock:
            try:
                # Build configuration from database
                config = await self._build_config_from_db(cluster_id, db)

                if not config:
                    logger.warning(f"No configuration to update for cluster {cluster_id}")
                    return True  # Not an error, just no config

                # Validate configuration before sending
                if not self._validate_config(config):
                    logger.error("Configuration validation failed")
                    self._last_error = "Configuration validation failed"
                    return False

                # Send to xDS control plane
                response = await self.client.post(
                    f"{self.xds_server_url}/v1/config",
                    json=config,
                    timeout=30.0
                )

                if response.status_code == 200:
                    self._last_update = datetime.utcnow()
                    self._update_count += 1
                    self._last_error = None

                    result = response.json()
                    logger.info(
                        f"Successfully updated xDS config for cluster {cluster_id}, "
                        f"version: {result.get('version')}"
                    )
                    return True
                else:
                    error_msg = f"HTTP {response.status_code}: {response.text}"
                    logger.error(f"Failed to update xDS config: {error_msg}")
                    self._last_error = error_msg
                    return False

            except httpx.TimeoutException as e:
                error_msg = f"Timeout updating xDS config: {str(e)}"
                logger.error(error_msg)
                self._last_error = error_msg
                return False
            except Exception as e:
                error_msg = f"Error updating xDS config: {str(e)}"
                logger.error(error_msg, exc_info=True)
                self._last_error = error_msg
                return False

    async def _build_config_from_db(
        self,
        cluster_id: int,
        db: Session
    ) -> Optional[Dict[str, Any]]:
        """
        Build xDS configuration from database models

        Args:
            cluster_id: Cluster ID
            db: Database session (can be sync or async)

        Returns:
            Configuration dictionary or None
        """
        try:
            # Import models here to avoid circular dependencies
            from app.models.sqlalchemy.service import Service
            from app.models.sqlalchemy.mapping import Mapping
            from app.models.sqlalchemy.certificate import Certificate
            from sqlalchemy import select

            # Handle both sync and async sessions
            if hasattr(db, 'execute'):
                # Async session
                stmt = select(Service).filter(
                    Service.cluster_id == cluster_id,
                    Service.is_active == True
                )
                result = await db.execute(stmt)
                services = result.scalars().all()
            else:
                # Sync session
                services = db.query(Service).filter(
                    Service.cluster_id == cluster_id,
                    Service.is_active == True
                ).all()

            if not services:
                return None

            # Query mappings for this cluster
            mappings = db.query(Mapping).filter(
                Mapping.cluster_id == cluster_id,
                Mapping.is_active == True
            ).all()

            # Query certificates (active and not expired)
            certificates = db.query(Certificate).filter(
                Certificate.is_active == True,
                Certificate.valid_until > datetime.utcnow()
            ).all()

            # Build service configurations with enhanced TLS/protocol support
            service_configs = []
            cert_mapping = {cert.name: cert for cert in certificates}

            for svc in services:
                # Determine TLS configuration
                tls_enabled = svc.tls_enabled if hasattr(svc, 'tls_enabled') else False
                tls_cert_name = None

                # Check if service has an associated certificate
                if tls_enabled and hasattr(svc, 'extra_metadata') and svc.extra_metadata:
                    import json
                    try:
                        metadata = json.loads(svc.extra_metadata) if isinstance(svc.extra_metadata, str) else svc.extra_metadata
                        tls_cert_name = metadata.get('tls_cert_name')
                    except:
                        pass

                service_configs.append({
                    "name": f"cluster_{cluster_id}_service_{svc.id}",
                    "hosts": [svc.ip_fqdn] if svc.ip_fqdn else [],
                    "port": self._extract_port(svc),
                    "protocol": self._determine_protocol(svc, mappings),
                    "tls_enabled": tls_enabled,
                    "tls_cert_name": tls_cert_name,
                    "tls_verify": svc.tls_verify if hasattr(svc, 'tls_verify') else True,
                    "health_check_path": svc.health_check_path if hasattr(svc, 'health_check_path') else "/healthz",
                    "timeout_seconds": 30,  # Default timeout
                    "http2_enabled": self._is_http2_enabled(svc),
                    "websocket_upgrade": self._is_websocket_enabled(svc),
                })

            # Build route configurations
            route_configs = []
            for mapping in mappings:
                # Get source and destination services
                src_services = self._parse_service_list(mapping.source_services, services)
                dst_services = self._parse_service_list(mapping.dest_services, services)

                for src in src_services:
                    for dst in dst_services:
                        route_configs.append({
                            "name": f"route_{mapping.id}_{src.id}_{dst.id}",
                            "prefix": "/",  # Default prefix, can be enhanced
                            "cluster_name": f"cluster_{cluster_id}_service_{dst.id}",
                            "hosts": [src.ip_fqdn] if src.ip_fqdn else ["*"],
                            "timeout": 30,  # Default timeout in seconds
                        })

            # Build certificate configurations
            cert_configs = []
            for cert in certificates:
                cert_configs.append({
                    "name": cert.name,
                    "cert_chain": cert.cert_data,
                    "private_key": cert.key_data,
                    "ca_cert": cert.ca_chain if hasattr(cert, 'ca_chain') else "",
                    "require_client": False,  # Can be enhanced based on metadata
                })

            # Build complete configuration
            config = {
                "version": str(int(datetime.utcnow().timestamp())),
                "services": service_configs,
                "routes": route_configs,
                "certificates": cert_configs,
            }

            logger.debug(
                f"Built config for cluster {cluster_id}: "
                f"{len(service_configs)} services, {len(route_configs)} routes, "
                f"{len(cert_configs)} certificates"
            )

            return config

        except Exception as e:
            logger.error(f"Error building config from database: {str(e)}", exc_info=True)
            return None

    def _extract_port(self, service: Any) -> int:
        """Extract port from service configuration"""
        # This is a placeholder - actual implementation depends on your schema
        # You might have a port field or need to parse it from mappings
        return getattr(service, 'port', 8080)

    def _determine_protocol(self, service: Any, mappings: List[Any]) -> str:
        """Determine protocol for a service"""
        # Check service protocol field first
        if hasattr(service, 'protocol') and service.protocol:
            return service.protocol.lower()

        # Check if any mapping for this service uses HTTP/HTTPS
        for mapping in mappings:
            protocols = mapping.protocols.lower() if hasattr(mapping, 'protocols') and mapping.protocols else ""
            if "grpc" in protocols:
                return "grpc"
            if "http2" in protocols:
                return "http2"
            if "https" in protocols:
                return "https"
            if "http" in protocols:
                return "http"

        return "http"  # Default

    def _is_http2_enabled(self, service: Any) -> bool:
        """Check if HTTP/2 is enabled for a service"""
        # Check protocol
        if hasattr(service, 'protocol') and service.protocol:
            proto = service.protocol.lower()
            if proto in ["http2", "grpc"]:
                return True

        # Check metadata for explicit HTTP/2 setting
        if hasattr(service, 'extra_metadata') and service.extra_metadata:
            import json
            try:
                metadata = json.loads(service.extra_metadata) if isinstance(service.extra_metadata, str) else service.extra_metadata
                return metadata.get('http2_enabled', False)
            except:
                pass

        return False

    def _is_websocket_enabled(self, service: Any) -> bool:
        """Check if WebSocket upgrade is enabled for a service"""
        # Check metadata for WebSocket setting
        if hasattr(service, 'extra_metadata') and service.extra_metadata:
            import json
            try:
                metadata = json.loads(service.extra_metadata) if isinstance(service.extra_metadata, str) else service.extra_metadata
                return metadata.get('websocket_upgrade', False)
            except:
                pass

        return False

    def _parse_service_list(
        self,
        service_list: str,
        all_services: List[Any]
    ) -> List[Any]:
        """
        Parse service list string and return matching services

        Args:
            service_list: Comma-separated service IDs or names
            all_services: List of all available services

        Returns:
            List of matching service objects
        """
        if not service_list:
            return []

        # Handle special cases
        if service_list.lower() == "all":
            return all_services

        # Parse comma-separated list
        service_identifiers = [s.strip() for s in service_list.split(",")]

        matching_services = []
        for identifier in service_identifiers:
            # Try to match by ID first
            try:
                service_id = int(identifier)
                for svc in all_services:
                    if svc.id == service_id:
                        matching_services.append(svc)
                        break
            except ValueError:
                # Not a number, try matching by name
                for svc in all_services:
                    if svc.name == identifier:
                        matching_services.append(svc)
                        break

        return matching_services

    def _validate_config(self, config: Dict[str, Any]) -> bool:
        """
        Validate configuration before sending to xDS server

        Args:
            config: Configuration dictionary

        Returns:
            True if valid, False otherwise
        """
        try:
            # Check required fields
            if "version" not in config:
                logger.error("Config missing 'version' field")
                return False

            if "services" not in config or not isinstance(config["services"], list):
                logger.error("Config missing or invalid 'services' field")
                return False

            if "routes" not in config or not isinstance(config["routes"], list):
                logger.error("Config missing or invalid 'routes' field")
                return False

            # Validate each service
            for svc in config["services"]:
                if not isinstance(svc, dict):
                    logger.error("Invalid service configuration")
                    return False

                required_fields = ["name", "hosts", "port", "protocol"]
                for field in required_fields:
                    if field not in svc:
                        logger.error(f"Service missing required field: {field}")
                        return False

                # Validate port
                if not isinstance(svc["port"], int) or svc["port"] < 1 or svc["port"] > 65535:
                    logger.error(f"Invalid port: {svc['port']}")
                    return False

                # Validate protocol
                if svc["protocol"] not in ["http", "https", "grpc"]:
                    logger.error(f"Invalid protocol: {svc['protocol']}")
                    return False

            # Validate each route
            for route in config["routes"]:
                if not isinstance(route, dict):
                    logger.error("Invalid route configuration")
                    return False

                required_fields = ["name", "prefix", "cluster_name", "hosts", "timeout"]
                for field in required_fields:
                    if field not in route:
                        logger.error(f"Route missing required field: {field}")
                        return False

                # Validate timeout
                if not isinstance(route["timeout"], int) or route["timeout"] < 1:
                    logger.error(f"Invalid timeout: {route['timeout']}")
                    return False

            return True

        except Exception as e:
            logger.error(f"Error validating config: {str(e)}", exc_info=True)
            return False

    async def rollback_to_version(self, version: int) -> bool:
        """
        Rollback to a previous configuration version

        Args:
            version: Version number to rollback to

        Returns:
            True if successful, False otherwise
        """
        try:
            response = await self.client.post(
                f"{self.xds_server_url}/v1/rollback/{version}",
                timeout=30.0
            )

            if response.status_code == 200:
                result = response.json()
                logger.info(
                    f"Successfully rolled back to version {version}, "
                    f"new version: {result.get('new_version')}"
                )
                return True
            else:
                logger.error(
                    f"Failed to rollback: HTTP {response.status_code} - {response.text}"
                )
                return False

        except Exception as e:
            logger.error(f"Error rolling back: {str(e)}", exc_info=True)
            return False

    async def get_current_version(self) -> Optional[int]:
        """
        Get current xDS configuration version

        Returns:
            Current version number or None if error
        """
        try:
            response = await self.client.get(
                f"{self.xds_server_url}/v1/version",
                timeout=10.0
            )

            if response.status_code == 200:
                data = response.json()
                return data.get("version")
            else:
                logger.warning(f"Could not get version: HTTP {response.status_code}")
                return None

        except Exception as e:
            logger.error(f"Error getting version: {str(e)}", exc_info=True)
            return None

    async def health_check(self) -> bool:
        """
        Check if xDS control plane is healthy

        Returns:
            True if healthy, False otherwise
        """
        try:
            response = await self.client.get(
                f"{self.xds_server_url}/healthz",
                timeout=5.0
            )
            return response.status_code == 200
        except Exception as e:
            logger.warning(f"xDS health check failed: {str(e)}")
            return False

    def get_stats(self) -> Dict[str, Any]:
        """
        Get service statistics

        Returns:
            Dictionary with service stats
        """
        return {
            "xds_server_url": self.xds_server_url,
            "last_update": self._last_update.isoformat() if self._last_update else None,
            "update_count": self._update_count,
            "last_error": self._last_error,
        }


# Global xDS service instance
_xds_service: Optional[XDSService] = None


def get_xds_service(xds_server_url: str = "http://localhost:19000") -> XDSService:
    """
    Get the global xDS service instance

    Args:
        xds_server_url: URL of the xDS control plane

    Returns:
        XDSService instance
    """
    global _xds_service
    if _xds_service is None:
        _xds_service = XDSService(xds_server_url)
    return _xds_service


async def trigger_xds_update(cluster_id: int, db: Session) -> bool:
    """
    Convenience function to trigger xDS update for a cluster

    Args:
        cluster_id: Cluster ID
        db: Database session

    Returns:
        True if successful, False otherwise
    """
    xds_service = get_xds_service()
    return await xds_service.update_envoy_config(cluster_id, db)
