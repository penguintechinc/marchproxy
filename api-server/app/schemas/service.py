"""
Service management Pydantic schemas
"""

from typing import Optional
from datetime import datetime
from pydantic import BaseModel, Field, field_validator


class ServiceBase(BaseModel):
    """Base service schema"""
    name: str = Field(..., min_length=1, max_length=100, description="Service name")
    ip_fqdn: str = Field(..., min_length=1, max_length=255, description="Service IP or FQDN")
    port: int = Field(..., ge=1, le=65535, description="Service port")
    protocol: str = Field(default="tcp", description="Protocol (tcp/udp/http/https)")
    collection: Optional[str] = Field(None, max_length=100, description="Service collection/group")
    cluster_id: int = Field(..., description="Cluster ID this service belongs to")
    auth_type: str = Field(default="none", description="Authentication type (none/base64/jwt)")
    tls_enabled: bool = Field(default=False, description="Enable TLS for this service")
    tls_verify: bool = Field(default=True, description="Verify TLS certificates")
    health_check_enabled: bool = Field(default=False, description="Enable health checks")
    health_check_path: Optional[str] = Field(None, description="Health check endpoint path")
    health_check_interval: int = Field(default=30, ge=5, description="Health check interval in seconds")

    @field_validator("auth_type")
    @classmethod
    def validate_auth_type(cls, v: str) -> str:
        if v not in ["none", "base64", "jwt"]:
            raise ValueError("auth_type must be one of: none, base64, jwt")
        return v

    @field_validator("protocol")
    @classmethod
    def validate_protocol(cls, v: str) -> str:
        if v not in ["tcp", "udp", "http", "https"]:
            raise ValueError("protocol must be one of: tcp, udp, http, https")
        return v


class ServiceCreate(ServiceBase):
    """Schema for creating a new service"""
    # Optional fields for JWT configuration
    jwt_expiry: Optional[int] = Field(None, ge=300, description="JWT expiry in seconds (min 5 min)")
    jwt_algorithm: Optional[str] = Field("HS256", description="JWT signing algorithm")


class ServiceUpdate(BaseModel):
    """Schema for updating a service"""
    name: Optional[str] = Field(None, min_length=1, max_length=100)
    ip_fqdn: Optional[str] = Field(None, min_length=1, max_length=255)
    port: Optional[int] = Field(None, ge=1, le=65535)
    protocol: Optional[str] = None
    collection: Optional[str] = None
    auth_type: Optional[str] = None
    tls_enabled: Optional[bool] = None
    tls_verify: Optional[bool] = None
    health_check_enabled: Optional[bool] = None
    health_check_path: Optional[str] = None
    health_check_interval: Optional[int] = Field(None, ge=5)
    is_active: Optional[bool] = None


class ServiceResponse(ServiceBase):
    """Schema for service response"""
    id: int
    token_base64: Optional[str] = Field(None, description="Base64 auth token (if auth_type=base64)")
    jwt_secret: Optional[str] = Field(None, description="JWT secret (if auth_type=jwt)")
    jwt_expiry: int = Field(default=3600, description="JWT expiry in seconds")
    jwt_algorithm: str = Field(default="HS256", description="JWT algorithm")
    is_active: bool
    created_by: int
    created_at: datetime
    updated_at: datetime

    class Config:
        from_attributes = True


class ServiceListResponse(BaseModel):
    """Schema for list of services"""
    total: int
    services: list[ServiceResponse]


class ServiceTokenRotateRequest(BaseModel):
    """Request to rotate service authentication token"""
    auth_type: str = Field(..., description="Type of token to rotate (base64/jwt)")

    @field_validator("auth_type")
    @classmethod
    def validate_auth_type(cls, v: str) -> str:
        if v not in ["base64", "jwt"]:
            raise ValueError("auth_type must be either base64 or jwt")
        return v


class ServiceTokenRotateResponse(BaseModel):
    """Response when rotating service token"""
    service_id: int
    auth_type: str
    new_token: Optional[str] = Field(None, description="New Base64 token (show only once)")
    new_jwt_secret: Optional[str] = Field(None, description="New JWT secret (show only once)")
    message: str = Field(default="Authentication token rotated successfully")
