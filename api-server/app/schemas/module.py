"""Module management Pydantic schemas"""

from typing import Optional, Dict, Any, List
from datetime import datetime
from pydantic import BaseModel, Field
from app.models.sqlalchemy.module import ModuleType, ModuleStatus, DeploymentStatus


# ==================== Module Schemas ====================

class ModuleBase(BaseModel):
    """Base module schema"""
    name: str = Field(..., min_length=1, max_length=100, description="Module name")
    type: ModuleType = Field(..., description="Module type")
    description: Optional[str] = Field(None, description="Module description")
    config: Dict[str, Any] = Field(default_factory=dict, description="Module configuration")
    grpc_host: Optional[str] = Field(None, max_length=255, description="gRPC host")
    grpc_port: Optional[int] = Field(None, ge=1, le=65535, description="gRPC port")
    version: Optional[str] = Field(None, max_length=50, description="Module version")
    image: Optional[str] = Field(None, max_length=255, description="Docker image")
    replicas: int = Field(default=1, ge=1, le=100, description="Number of replicas")


class ModuleCreate(ModuleBase):
    """Schema for creating a new module"""
    enabled: bool = Field(default=False, description="Enable module on creation")


class ModuleUpdate(BaseModel):
    """Schema for updating a module"""
    name: Optional[str] = Field(None, min_length=1, max_length=100)
    description: Optional[str] = None
    config: Optional[Dict[str, Any]] = None
    grpc_host: Optional[str] = None
    grpc_port: Optional[int] = Field(None, ge=1, le=65535)
    version: Optional[str] = None
    image: Optional[str] = None
    replicas: Optional[int] = Field(None, ge=1, le=100)
    enabled: Optional[bool] = None
    status: Optional[ModuleStatus] = None


class ModuleResponse(ModuleBase):
    """Schema for module response"""
    id: int
    status: ModuleStatus
    enabled: bool
    health_status: str
    last_health_check: Optional[datetime]
    created_by: int
    created_at: datetime
    updated_at: datetime

    class Config:
        from_attributes = True


class ModuleListResponse(BaseModel):
    """Schema for list of modules"""
    total: int
    modules: List[ModuleResponse]


class ModuleHealthResponse(BaseModel):
    """Schema for module health check response"""
    module_id: int
    module_name: str
    health_status: str
    uptime_seconds: Optional[int]
    version: Optional[str]
    active_connections: Optional[int]
    last_check: datetime


# ==================== Module Route Schemas ====================

class ModuleRouteBase(BaseModel):
    """Base module route schema"""
    name: str = Field(..., min_length=1, max_length=100, description="Route name")
    match_rules: Dict[str, Any] = Field(
        ...,
        description="Match rules (e.g., {'host': 'example.com', 'path': '/api/*'})"
    )
    backend_config: Dict[str, Any] = Field(
        ...,
        description="Backend config (e.g., {'target': 'http://backend:8080', 'timeout': 30})"
    )
    rate_limit: Optional[float] = Field(None, ge=0, description="Rate limit (requests/second)")
    priority: int = Field(default=100, ge=0, le=1000, description="Priority (higher first)")
    enabled: bool = Field(default=True, description="Enable route")


class ModuleRouteCreate(ModuleRouteBase):
    """Schema for creating a new module route"""
    pass


class ModuleRouteUpdate(BaseModel):
    """Schema for updating a module route"""
    name: Optional[str] = Field(None, min_length=1, max_length=100)
    match_rules: Optional[Dict[str, Any]] = None
    backend_config: Optional[Dict[str, Any]] = None
    rate_limit: Optional[float] = Field(None, ge=0)
    priority: Optional[int] = Field(None, ge=0, le=1000)
    enabled: Optional[bool] = None


class ModuleRouteResponse(ModuleRouteBase):
    """Schema for module route response"""
    id: int
    module_id: int
    created_at: datetime
    updated_at: datetime

    class Config:
        from_attributes = True


class ModuleRouteListResponse(BaseModel):
    """Schema for list of module routes"""
    total: int
    routes: List[ModuleRouteResponse]


# ==================== Scaling Policy Schemas ====================

class ScalingPolicyBase(BaseModel):
    """Base scaling policy schema"""
    min_instances: int = Field(default=1, ge=1, le=100, description="Minimum instances")
    max_instances: int = Field(default=10, ge=1, le=100, description="Maximum instances")
    scale_up_threshold: float = Field(
        default=80.0,
        ge=0,
        le=100,
        description="Scale up threshold (%)"
    )
    scale_down_threshold: float = Field(
        default=20.0,
        ge=0,
        le=100,
        description="Scale down threshold (%)"
    )
    cooldown_seconds: int = Field(
        default=300,
        ge=0,
        le=3600,
        description="Cooldown period (seconds)"
    )
    metric: str = Field(
        default="cpu",
        description="Metric to monitor (cpu, memory, requests_per_second)"
    )
    enabled: bool = Field(default=True, description="Enable auto-scaling")


class ScalingPolicyCreate(ScalingPolicyBase):
    """Schema for creating a scaling policy"""
    pass


class ScalingPolicyUpdate(BaseModel):
    """Schema for updating a scaling policy"""
    min_instances: Optional[int] = Field(None, ge=1, le=100)
    max_instances: Optional[int] = Field(None, ge=1, le=100)
    scale_up_threshold: Optional[float] = Field(None, ge=0, le=100)
    scale_down_threshold: Optional[float] = Field(None, ge=0, le=100)
    cooldown_seconds: Optional[int] = Field(None, ge=0, le=3600)
    metric: Optional[str] = None
    enabled: Optional[bool] = None


class ScalingPolicyResponse(ScalingPolicyBase):
    """Schema for scaling policy response"""
    id: int
    module_id: int
    created_at: datetime
    updated_at: datetime

    class Config:
        from_attributes = True


# ==================== Deployment Schemas ====================

class DeploymentBase(BaseModel):
    """Base deployment schema"""
    version: str = Field(..., min_length=1, max_length=50, description="Version")
    image: str = Field(..., min_length=1, max_length=255, description="Docker image")
    config: Dict[str, Any] = Field(default_factory=dict, description="Deployment config")
    environment: Dict[str, Any] = Field(default_factory=dict, description="Environment vars")


class DeploymentCreate(DeploymentBase):
    """Schema for creating a new deployment"""
    traffic_weight: float = Field(
        default=0.0,
        ge=0,
        le=100,
        description="Initial traffic weight (%)"
    )


class DeploymentUpdate(BaseModel):
    """Schema for updating a deployment"""
    status: Optional[DeploymentStatus] = None
    traffic_weight: Optional[float] = Field(None, ge=0, le=100)
    config: Optional[Dict[str, Any]] = None
    environment: Optional[Dict[str, Any]] = None


class DeploymentResponse(DeploymentBase):
    """Schema for deployment response"""
    id: int
    module_id: int
    status: DeploymentStatus
    traffic_weight: float
    health_check_passed: bool
    health_check_message: Optional[str]
    previous_deployment_id: Optional[int]
    deployed_by: int
    deployed_at: datetime
    completed_at: Optional[datetime]

    class Config:
        from_attributes = True


class DeploymentListResponse(BaseModel):
    """Schema for list of deployments"""
    total: int
    deployments: List[DeploymentResponse]


class DeploymentPromoteRequest(BaseModel):
    """Schema for promoting a deployment"""
    traffic_weight: float = Field(
        default=100.0,
        ge=0,
        le=100,
        description="Target traffic weight (%)"
    )
    incremental: bool = Field(
        default=False,
        description="Incremental rollout (gradual traffic shift)"
    )


class DeploymentRollbackRequest(BaseModel):
    """Schema for rolling back a deployment"""
    reason: str = Field(..., min_length=1, description="Rollback reason")
