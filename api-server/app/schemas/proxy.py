"""
Proxy server Pydantic schemas
"""

from typing import Optional, Dict, Any
from datetime import datetime
from pydantic import BaseModel, Field


class ProxyRegisterRequest(BaseModel):
    """Request schema for proxy registration"""
    name: str = Field(..., min_length=1, max_length=100, description="Proxy server name")
    hostname: str = Field(..., min_length=1, max_length=255, description="Proxy hostname")
    ip_address: str = Field(..., description="Proxy IP address")
    port: int = Field(default=8080, ge=1, le=65535, description="Proxy port")
    version: Optional[str] = Field(None, max_length=50, description="Proxy version")
    capabilities: Optional[Dict[str, Any]] = Field(None, description="Proxy capabilities (XDP, AF_XDP, etc)")
    cluster_api_key: str = Field(..., description="Cluster API key for authentication")


class ProxyHeartbeatRequest(BaseModel):
    """Request schema for proxy heartbeat"""
    proxy_id: int = Field(..., description="Proxy server ID")
    cluster_api_key: str = Field(..., description="Cluster API key for authentication")
    status: str = Field(default="active", description="Current proxy status")
    config_version: Optional[str] = Field(None, description="Current configuration version")
    metrics: Optional[Dict[str, Any]] = Field(None, description="Current metrics snapshot")


class ProxyResponse(BaseModel):
    """Schema for proxy response"""
    id: int
    name: str
    hostname: str
    ip_address: str
    port: int
    cluster_id: int
    status: str
    version: Optional[str]
    capabilities: Optional[Dict[str, Any]]
    license_validated: bool
    license_validation_at: Optional[datetime]
    last_seen: Optional[datetime]
    last_config_fetch: Optional[datetime]
    config_version: Optional[str]
    registered_at: datetime

    class Config:
        from_attributes = True


class ProxyListResponse(BaseModel):
    """Schema for list of proxies"""
    total: int
    proxies: list[ProxyResponse]


class ProxyConfigResponse(BaseModel):
    """Configuration response for proxy"""
    config_version: str = Field(..., description="Configuration version hash")
    cluster: Dict[str, Any] = Field(..., description="Cluster configuration")
    services: list[Dict[str, Any]] = Field(..., description="Services configuration")
    mappings: list[Dict[str, Any]] = Field(..., description="Service mappings")
    certificates: Optional[Dict[str, Any]] = Field(None, description="TLS certificates if applicable")
    logging: Dict[str, Any] = Field(..., description="Logging configuration")


class ProxyMetricsRequest(BaseModel):
    """Request schema for reporting proxy metrics"""
    proxy_id: int
    cpu_usage: Optional[float] = Field(None, ge=0, le=100, description="CPU usage percentage")
    memory_usage: Optional[float] = Field(None, ge=0, le=100, description="Memory usage percentage")
    connections_active: Optional[int] = Field(None, ge=0, description="Active connections")
    connections_total: Optional[int] = Field(None, ge=0, description="Total connections")
    bytes_sent: Optional[int] = Field(None, ge=0, description="Total bytes sent")
    bytes_received: Optional[int] = Field(None, ge=0, description="Total bytes received")
    requests_per_second: Optional[float] = Field(None, ge=0, description="Current RPS")
    latency_avg: Optional[float] = Field(None, ge=0, description="Average latency in ms")
    latency_p95: Optional[float] = Field(None, ge=0, description="P95 latency in ms")
    errors_per_second: Optional[float] = Field(None, ge=0, description="Error rate")
