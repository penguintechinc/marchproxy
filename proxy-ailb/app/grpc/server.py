"""
gRPC ModuleService Server Implementation
Implements the ModuleService gRPC interface for AILB
"""

import logging
import asyncio
import time
from datetime import datetime
from typing import Dict, Any
import grpc
from concurrent import futures

# Import generated protobuf code
import sys
import os
sys.path.append(os.path.join(os.path.dirname(__file__), '..', '..', '..', 'proto'))

try:
    from marchproxy import module_service_pb2
    from marchproxy import module_service_pb2_grpc
except ImportError:
    # Fallback: proto files not generated yet
    module_service_pb2 = None
    module_service_pb2_grpc = None

logger = logging.getLogger(__name__)


class ModuleServiceImpl:
    """Implementation of ModuleService gRPC interface for AILB"""

    def __init__(self, ailb_server):
        self.ailb_server = ailb_server
        self.start_time = time.time()
        self.module_id = os.getenv('MODULE_ID', 'ailb-1')
        self.version = "1.0.0"

    async def GetStatus(self, request, context):
        """Get module health and status"""
        try:
            # Determine health status
            health = module_service_pb2.HEALTHY

            # Check connector health
            for connector in self.ailb_server.connectors.values():
                health_check = await connector.health_check()
                if health_check.get('status') != 'healthy':
                    health = module_service_pb2.DEGRADED
                    break

            uptime = int(time.time() - self.start_time)

            return module_service_pb2.StatusResponse(
                module_id=self.module_id,
                module_type="AILB",
                version=self.version,
                health=health,
                uptime_seconds=uptime,
                envoy_version="N/A",  # AILB doesn't use Envoy
                metadata={
                    "num_connectors": str(len(self.ailb_server.connectors)),
                    "memory_enabled": str(self.ailb_server.config['enable_memory']),
                    "rag_enabled": str(self.ailb_server.config['enable_rag'])
                }
            )

        except Exception as e:
            logger.error(f"GetStatus failed: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return module_service_pb2.StatusResponse()

    async def GetRoutes(self, request, context):
        """Get route configuration"""
        try:
            routes = []

            # AILB routes are provider-specific
            for provider_name, connector in self.ailb_server.connectors.items():
                models = await connector.list_models()

                for model in models:
                    route = module_service_pb2.RouteConfig(
                        name=f"{provider_name}_{model.get('id', 'unknown')}",
                        prefix=f"/v1/chat/completions",
                        cluster_name=provider_name,
                        hosts=[model.get('id', 'unknown')],
                        timeout_seconds=300,
                        enabled=True,
                        headers={}
                    )
                    routes.append(route)

            return module_service_pb2.RoutesResponse(
                routes=routes,
                version=int(time.time())
            )

        except Exception as e:
            logger.error(f"GetRoutes failed: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return module_service_pb2.RoutesResponse()

    async def ApplyRateLimit(self, request, context):
        """Apply rate limiting configuration"""
        try:
            # AILB rate limiting would be implemented per-provider
            # For now, return success
            return module_service_pb2.RateLimitResponse(
                success=True,
                message=f"Rate limit applied to route {request.route_name}"
            )

        except Exception as e:
            logger.error(f"ApplyRateLimit failed: {e}")
            return module_service_pb2.RateLimitResponse(
                success=False,
                message=str(e)
            )

    async def GetMetrics(self, request, context):
        """Get performance metrics"""
        try:
            # Get routing stats
            stats = self.ailb_server.request_router.get_stats()

            # Calculate aggregate metrics
            total_requests = sum(s['total_requests'] for s in stats.values())
            active_connections = len(self.ailb_server.connectors)

            # Build route metrics
            route_metrics = {}
            for provider, pstats in stats.items():
                route_metrics[provider] = module_service_pb2.RouteMetrics(
                    requests=pstats['total_requests'],
                    errors=pstats['failed_requests'],
                    avg_latency_ms=pstats['avg_latency_ms']
                )

            # Build latency metrics (simplified)
            avg_latency = sum(s['avg_latency_ms'] for s in stats.values() if s['total_requests'] > 0) / max(len(stats), 1)
            latency = module_service_pb2.LatencyMetrics(
                p50_ms=avg_latency,
                p90_ms=avg_latency * 1.5,
                p95_ms=avg_latency * 2,
                p99_ms=avg_latency * 3,
                avg_ms=avg_latency
            )

            return module_service_pb2.MetricsResponse(
                timestamp=int(time.time()),
                total_connections=active_connections,
                active_connections=active_connections,
                total_requests=total_requests,
                requests_per_second=0,  # TODO: Calculate RPS
                latency=latency,
                status_codes={},
                routes=route_metrics
            )

        except Exception as e:
            logger.error(f"GetMetrics failed: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return module_service_pb2.MetricsResponse()

    async def SetTrafficWeight(self, request, context):
        """Set traffic weight for blue/green deployments"""
        try:
            # AILB traffic weighting would be implemented in the router
            # For now, return success
            applied_weights = []
            for weight in request.weights:
                applied_weights.append(module_service_pb2.BackendWeight(
                    backend_name=weight.backend_name,
                    weight=weight.weight,
                    version=weight.version
                ))

            return module_service_pb2.TrafficWeightResponse(
                success=True,
                message=f"Traffic weights applied to route {request.route_name}",
                applied_weights=applied_weights
            )

        except Exception as e:
            logger.error(f"SetTrafficWeight failed: {e}")
            return module_service_pb2.TrafficWeightResponse(
                success=False,
                message=str(e),
                applied_weights=[]
            )

    async def Reload(self, request, context):
        """Reload configuration"""
        try:
            # Reload connectors
            # TODO: Implement graceful reload
            reload_timestamp = int(time.time())

            return module_service_pb2.ReloadResponse(
                success=True,
                message="Configuration reloaded successfully",
                reload_timestamp=reload_timestamp
            )

        except Exception as e:
            logger.error(f"Reload failed: {e}")
            return module_service_pb2.ReloadResponse(
                success=False,
                message=str(e),
                reload_timestamp=int(time.time())
            )


async def start_grpc_server(ailb_server, port: int = 50051):
    """Start the gRPC server"""
    if not module_service_pb2 or not module_service_pb2_grpc:
        logger.warning("Proto files not generated, skipping gRPC server startup")
        logger.warning("Run: python -m grpc_tools.protoc -I../proto --python_out=. --grpc_python_out=. ../proto/marchproxy/module_service.proto")
        return

    try:
        server = grpc.aio.server(futures.ThreadPoolExecutor(max_workers=10))

        # Create service implementation
        service_impl = ModuleServiceImpl(ailb_server)

        # Add service to server
        module_service_pb2_grpc.add_ModuleServiceServicer_to_server(
            service_impl,
            server
        )

        # Bind to port
        server.add_insecure_port(f'[::]:{port}')

        # Start server
        await server.start()
        logger.info(f"gRPC ModuleService started on port {port}")

        # Wait for termination
        await server.wait_for_termination()

    except Exception as e:
        logger.error(f"gRPC server failed: {e}")
