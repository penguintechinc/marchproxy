"""
Pydantic schemas for multi-cloud routing
"""

from datetime import datetime
from enum import Enum
from typing import Optional

from pydantic import BaseModel, Field, validator


class CloudProvider(str, Enum):
    """Cloud provider enumeration"""
    AWS = "aws"
    GCP = "gcp"
    AZURE = "azure"
    ON_PREM = "on_prem"


class RoutingAlgorithm(str, Enum):
    """Routing algorithm options"""
    LATENCY = "latency"           # Lowest RTT
    COST = "cost"                 # Cheapest egress
    GEO_PROXIMITY = "geo"         # Nearest region
    WEIGHTED_RR = "weighted_rr"   # Weighted round-robin
    FAILOVER = "failover"         # Active-passive


class HealthCheckProtocol(str, Enum):
    """Health check protocol"""
    TCP = "tcp"
    HTTP = "http"
    HTTPS = "https"
    ICMP = "icmp"


class HealthProbeConfig(BaseModel):
    """Health probe configuration"""
    protocol: HealthCheckProtocol = Field(
        HealthCheckProtocol.TCP, description="Health check protocol"
    )
    port: Optional[int] = Field(
        None, ge=1, le=65535,
        description="Port for TCP/HTTP probes"
    )
    path: Optional[str] = Field(
        None, max_length=200,
        description="HTTP path for health checks"
    )
    interval_seconds: int = Field(
        30, ge=5, le=300,
        description="Interval between health checks"
    )
    timeout_seconds: int = Field(
        5, ge=1, le=30,
        description="Timeout for each health check"
    )
    unhealthy_threshold: int = Field(
        3, ge=1, le=10,
        description="Consecutive failures before marking unhealthy"
    )
    healthy_threshold: int = Field(
        2, ge=1, le=10,
        description="Consecutive successes before marking healthy"
    )

    @validator('timeout_seconds')
    def validate_timeout(cls, v, values):
        """Ensure timeout is less than interval"""
        if 'interval_seconds' in values and v >= values['interval_seconds']:
            raise ValueError("Timeout must be less than interval")
        return v


class CloudRoute(BaseModel):
    """Individual cloud route configuration"""
    provider: CloudProvider = Field(..., description="Cloud provider")
    region: str = Field(..., min_length=1, max_length=50)
    endpoint: str = Field(..., min_length=1, max_length=255)
    weight: int = Field(
        1, ge=1, le=100,
        description="Weight for weighted routing (1-100)"
    )
    cost_per_gb: Optional[float] = Field(
        None, ge=0, le=1000,
        description="Cost per GB for cost-based routing"
    )
    is_active: bool = Field(
        True, description="Whether this route is active"
    )


class RouteTableCreate(BaseModel):
    """Schema for creating a route table"""
    name: str = Field(..., min_length=1, max_length=100)
    description: Optional[str] = Field(None, max_length=500)
    service_id: int = Field(..., description="Service to route")
    cluster_id: int = Field(..., description="Cluster ID")

    # Routing configuration
    algorithm: RoutingAlgorithm = Field(
        RoutingAlgorithm.LATENCY,
        description="Routing algorithm"
    )
    routes: list[CloudRoute] = Field(
        ..., min_items=1, max_items=20,
        description="List of cloud routes"
    )

    # Health monitoring
    health_probe: HealthProbeConfig = Field(
        default_factory=HealthProbeConfig,
        description="Health probe configuration"
    )

    # Failover
    enable_auto_failover: bool = Field(
        True, description="Enable automatic failover"
    )

    enabled: bool = Field(True, description="Enable this route table")

    @validator('name')
    def validate_name(cls, v):
        """Validate route table name"""
        if not v.strip():
            raise ValueError("Route table name cannot be empty")
        return v.strip()

    @validator('routes')
    def validate_routes(cls, v):
        """Validate routes configuration"""
        if not v:
            raise ValueError("At least one route is required")
        # Ensure at least one route is active
        if not any(route.is_active for route in v):
            raise ValueError("At least one route must be active")
        return v


class RouteTableUpdate(BaseModel):
    """Schema for updating a route table"""
    name: Optional[str] = Field(None, min_length=1, max_length=100)
    description: Optional[str] = Field(None, max_length=500)
    algorithm: Optional[RoutingAlgorithm] = None
    routes: Optional[list[CloudRoute]] = Field(
        None, min_items=1, max_items=20
    )
    health_probe: Optional[HealthProbeConfig] = None
    enable_auto_failover: Optional[bool] = None
    enabled: Optional[bool] = None


class RouteHealthStatus(BaseModel):
    """Health status for a route"""
    endpoint: str
    is_healthy: bool
    last_check: datetime
    rtt_ms: Optional[float] = None
    consecutive_failures: int = 0
    consecutive_successes: int = 0


class RouteTableResponse(BaseModel):
    """Schema for route table response"""
    id: int
    name: str
    description: Optional[str]
    service_id: int
    cluster_id: int
    algorithm: RoutingAlgorithm
    routes: list[CloudRoute]
    health_probe: HealthProbeConfig
    enable_auto_failover: bool
    enabled: bool
    created_at: datetime
    updated_at: datetime
    # Runtime health status
    health_status: Optional[list[RouteHealthStatus]] = None

    class Config:
        from_attributes = True
