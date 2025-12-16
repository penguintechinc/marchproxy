"""
gRPC Client Service for Module Communication

Provides async gRPC client for communicating with module containers
to retrieve health status, metrics, and control operations.
"""

import logging
from typing import Optional, Dict, Any
from datetime import datetime

import grpc
from grpc import aio

logger = logging.getLogger(__name__)


class ModuleGRPCClient:
    """
    gRPC client for module communication

    Each module exposes a gRPC service for:
    - Health checks
    - Metrics collection
    - Configuration updates
    - Control operations (start/stop/reload)
    """

    def __init__(self, host: str, port: int, timeout: int = 5):
        """
        Initialize gRPC client

        Args:
            host: Module gRPC host
            port: Module gRPC port
            timeout: Request timeout in seconds
        """
        self.host = host
        self.port = port
        self.timeout = timeout
        self.address = f"{host}:{port}"
        self._channel: Optional[aio.Channel] = None

    async def _get_channel(self) -> aio.Channel:
        """Get or create gRPC channel"""
        if self._channel is None:
            self._channel = aio.insecure_channel(self.address)
        return self._channel

    async def close(self):
        """Close gRPC channel"""
        if self._channel:
            await self._channel.close()
            self._channel = None

    async def health_check(self) -> Dict[str, Any]:
        """
        Perform health check on module

        Returns:
            Dictionary with health status:
            {
                "status": "healthy" | "unhealthy" | "unknown",
                "uptime_seconds": int,
                "version": str,
                "active_connections": int,
                "last_check": datetime
            }

        Note:
            This is a simplified implementation. In production, you would
            use the actual gRPC service definition from .proto files.
        """
        try:
            channel = await self._get_channel()

            # Simple connectivity check
            # In production, replace with actual health check RPC
            await channel.channel_ready()

            # Placeholder response - replace with actual gRPC call
            return {
                "status": "healthy",
                "uptime_seconds": 0,
                "version": "unknown",
                "active_connections": 0,
                "last_check": datetime.utcnow()
            }

        except grpc.RpcError as e:
            logger.error(f"gRPC health check failed for {self.address}: {e}")
            return {
                "status": "unhealthy",
                "uptime_seconds": 0,
                "version": "unknown",
                "active_connections": 0,
                "last_check": datetime.utcnow(),
                "error": str(e)
            }
        except Exception as e:
            logger.error(f"Health check error for {self.address}: {e}")
            return {
                "status": "unknown",
                "uptime_seconds": 0,
                "version": "unknown",
                "active_connections": 0,
                "last_check": datetime.utcnow(),
                "error": str(e)
            }

    async def get_metrics(self) -> Dict[str, Any]:
        """
        Get metrics from module

        Returns:
            Dictionary with module metrics:
            {
                "cpu_percent": float,
                "memory_percent": float,
                "requests_per_second": float,
                "error_rate": float,
                "latency_p50": float,
                "latency_p95": float,
                "latency_p99": float
            }
        """
        try:
            channel = await self._get_channel()

            # Placeholder - replace with actual gRPC metrics call
            return {
                "cpu_percent": 0.0,
                "memory_percent": 0.0,
                "requests_per_second": 0.0,
                "error_rate": 0.0,
                "latency_p50": 0.0,
                "latency_p95": 0.0,
                "latency_p99": 0.0
            }

        except Exception as e:
            logger.error(f"Failed to get metrics from {self.address}: {e}")
            return {}

    async def update_config(self, config: Dict[str, Any]) -> bool:
        """
        Update module configuration via gRPC

        Args:
            config: Configuration dictionary

        Returns:
            True if successful, False otherwise
        """
        try:
            channel = await self._get_channel()

            # Placeholder - replace with actual gRPC config update call
            logger.info(f"Updating config for {self.address}: {config}")
            return True

        except Exception as e:
            logger.error(f"Failed to update config for {self.address}: {e}")
            return False

    async def reload_routes(self) -> bool:
        """
        Trigger route reload on module

        Returns:
            True if successful, False otherwise
        """
        try:
            channel = await self._get_channel()

            # Placeholder - replace with actual gRPC reload call
            logger.info(f"Reloading routes for {self.address}")
            return True

        except Exception as e:
            logger.error(f"Failed to reload routes for {self.address}: {e}")
            return False

    async def start_module(self) -> bool:
        """
        Start module operation

        Returns:
            True if successful, False otherwise
        """
        try:
            channel = await self._get_channel()

            # Placeholder - replace with actual gRPC start call
            logger.info(f"Starting module at {self.address}")
            return True

        except Exception as e:
            logger.error(f"Failed to start module at {self.address}: {e}")
            return False

    async def stop_module(self) -> bool:
        """
        Stop module operation

        Returns:
            True if successful, False otherwise
        """
        try:
            channel = await self._get_channel()

            # Placeholder - replace with actual gRPC stop call
            logger.info(f"Stopping module at {self.address}")
            return True

        except Exception as e:
            logger.error(f"Failed to stop module at {self.address}: {e}")
            return False


class ModuleGRPCClientManager:
    """
    Manages gRPC clients for all modules

    Provides a central point for module gRPC communication
    with connection pooling and caching.
    """

    def __init__(self):
        self._clients: Dict[int, ModuleGRPCClient] = {}

    def get_client(self, module_id: int, host: str, port: int) -> ModuleGRPCClient:
        """
        Get or create gRPC client for module

        Args:
            module_id: Module ID
            host: gRPC host
            port: gRPC port

        Returns:
            ModuleGRPCClient instance
        """
        if module_id not in self._clients:
            self._clients[module_id] = ModuleGRPCClient(host, port)
        return self._clients[module_id]

    async def close_all(self):
        """Close all gRPC clients"""
        for client in self._clients.values():
            await client.close()
        self._clients.clear()

    async def health_check_all(self) -> Dict[int, Dict[str, Any]]:
        """
        Perform health check on all modules

        Returns:
            Dictionary mapping module_id to health status
        """
        results = {}
        for module_id, client in self._clients.items():
            results[module_id] = await client.health_check()
        return results


# Global gRPC client manager
grpc_client_manager = ModuleGRPCClientManager()
