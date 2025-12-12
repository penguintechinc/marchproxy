"""
Cluster management Pydantic schemas
"""

from typing import Optional
from datetime import datetime
from pydantic import BaseModel, Field


class ClusterBase(BaseModel):
    """Base cluster schema"""
    name: str = Field(..., min_length=1, max_length=100, description="Cluster name")
    description: Optional[str] = Field(None, description="Cluster description")
    syslog_endpoint: Optional[str] = Field(None, max_length=255, description="Syslog endpoint (host:port)")
    log_auth: bool = Field(default=True, description="Log authentication events")
    log_netflow: bool = Field(default=True, description="Log netflow data")
    log_debug: bool = Field(default=False, description="Enable debug logging")
    max_proxies: int = Field(default=3, ge=1, description="Maximum number of proxies (Community: 3)")


class ClusterCreate(ClusterBase):
    """Schema for creating a new cluster"""
    pass


class ClusterUpdate(BaseModel):
    """Schema for updating a cluster"""
    name: Optional[str] = Field(None, min_length=1, max_length=100)
    description: Optional[str] = None
    syslog_endpoint: Optional[str] = None
    log_auth: Optional[bool] = None
    log_netflow: Optional[bool] = None
    log_debug: Optional[bool] = None
    max_proxies: Optional[int] = Field(None, ge=1)
    is_active: Optional[bool] = None


class ClusterResponse(ClusterBase):
    """Schema for cluster response"""
    id: int
    api_key_hash: str = Field(..., description="Hashed API key (never show plain key except on creation)")
    is_active: bool
    is_default: bool
    proxy_count: int = Field(default=0, description="Current number of registered proxies")
    created_by: int
    created_at: datetime
    updated_at: datetime

    class Config:
        from_attributes = True


class ClusterListResponse(BaseModel):
    """Schema for list of clusters"""
    total: int
    clusters: list[ClusterResponse]


class ClusterAPIKeyRotateResponse(BaseModel):
    """Response when rotating cluster API key"""
    cluster_id: int
    new_api_key: str = Field(..., description="New API key (show only once)")
    message: str = Field(default="API key rotated successfully. Update your proxy configurations.")
