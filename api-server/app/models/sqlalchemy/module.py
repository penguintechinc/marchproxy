"""Module management SQLAlchemy models for Phase 7 Unified NLB Architecture"""

from datetime import datetime
from sqlalchemy import (
    Boolean, Column, DateTime, Enum as SQLEnum, Float,
    ForeignKey, Integer, JSON, String, Text
)
from sqlalchemy.orm import relationship
from app.core.database import Base
import enum


class ModuleType(str, enum.Enum):
    """Module type enumeration"""
    L7_HTTP = "l7_http"
    L4_TCP = "l4_tcp"
    L4_UDP = "l4_udp"
    L3_NETWORK = "l3_network"
    OBSERVABILITY = "observability"
    ZERO_TRUST = "zero_trust"
    MULTI_CLOUD = "multi_cloud"


class ModuleStatus(str, enum.Enum):
    """Module status enumeration"""
    DISABLED = "disabled"
    ENABLED = "enabled"
    ERROR = "error"
    STARTING = "starting"
    STOPPING = "stopping"


class DeploymentStatus(str, enum.Enum):
    """Deployment status enumeration"""
    PENDING = "pending"
    ACTIVE = "active"
    INACTIVE = "inactive"
    ROLLING_OUT = "rolling_out"
    ROLLED_BACK = "rolled_back"
    FAILED = "failed"


class Module(Base):
    """Module configuration and state"""
    __tablename__ = "modules"

    id = Column(Integer, primary_key=True, index=True)
    name = Column(String(100), unique=True, nullable=False, index=True)
    type = Column(SQLEnum(ModuleType), nullable=False)
    description = Column(Text)
    status = Column(SQLEnum(ModuleStatus), default=ModuleStatus.DISABLED)
    enabled = Column(Boolean, default=False)

    # Module configuration (JSON)
    config = Column(JSON, default=dict)

    # gRPC connection info
    grpc_host = Column(String(255))
    grpc_port = Column(Integer)

    # Health status from gRPC
    health_status = Column(String(50), default="unknown")
    last_health_check = Column(DateTime)

    # Metadata
    version = Column(String(50))
    image = Column(String(255))  # Docker image
    replicas = Column(Integer, default=1)

    created_by = Column(Integer, ForeignKey("auth_user.id"), nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow, nullable=False)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)

    # Relationships
    creator = relationship("User", foreign_keys=[created_by])
    routes = relationship("ModuleRoute", back_populates="module", cascade="all, delete-orphan")
    scaling_policy = relationship(
        "ScalingPolicy",
        back_populates="module",
        uselist=False,
        cascade="all, delete-orphan"
    )
    deployments = relationship("Deployment", back_populates="module", cascade="all, delete-orphan")


class ModuleRoute(Base):
    """Route configuration per module"""
    __tablename__ = "module_routes"

    id = Column(Integer, primary_key=True, index=True)
    module_id = Column(Integer, ForeignKey("modules.id"), nullable=False)
    name = Column(String(100), nullable=False)

    # Matching rules (JSON)
    match_rules = Column(JSON, nullable=False)
    # Example: {"host": "example.com", "path": "/api/*", "method": ["GET", "POST"]}

    # Backend configuration (JSON)
    backend_config = Column(JSON, nullable=False)
    # Example: {"target": "http://backend:8080", "timeout": 30, "retries": 3}

    # Rate limiting (requests per second)
    rate_limit = Column(Float)

    # Priority (higher = processed first)
    priority = Column(Integer, default=100)

    enabled = Column(Boolean, default=True)

    created_at = Column(DateTime, default=datetime.utcnow, nullable=False)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)

    # Relationships
    module = relationship("Module", back_populates="routes")


class ScalingPolicy(Base):
    """Auto-scaling policy per module"""
    __tablename__ = "scaling_policies"

    id = Column(Integer, primary_key=True, index=True)
    module_id = Column(Integer, ForeignKey("modules.id"), nullable=False, unique=True)

    # Instance limits
    min_instances = Column(Integer, default=1, nullable=False)
    max_instances = Column(Integer, default=10, nullable=False)

    # Scaling thresholds (percentage CPU/Memory)
    scale_up_threshold = Column(Float, default=80.0)  # Scale up at 80% CPU
    scale_down_threshold = Column(Float, default=20.0)  # Scale down at 20% CPU

    # Cooldown period (seconds)
    cooldown_seconds = Column(Integer, default=300)  # 5 minutes

    # Metric to monitor
    metric = Column(String(50), default="cpu")  # cpu, memory, requests_per_second

    enabled = Column(Boolean, default=True)

    created_at = Column(DateTime, default=datetime.utcnow, nullable=False)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)

    # Relationships
    module = relationship("Module", back_populates="scaling_policy")


class Deployment(Base):
    """Blue/green deployment tracking"""
    __tablename__ = "deployments"

    id = Column(Integer, primary_key=True, index=True)
    module_id = Column(Integer, ForeignKey("modules.id"), nullable=False)

    version = Column(String(50), nullable=False)
    status = Column(SQLEnum(DeploymentStatus), default=DeploymentStatus.PENDING)

    # Traffic weight (0-100%)
    traffic_weight = Column(Float, default=0.0)

    # Deployment configuration (JSON)
    config = Column(JSON, default=dict)

    # Image and environment
    image = Column(String(255), nullable=False)
    environment = Column(JSON, default=dict)

    # Rollback information
    previous_deployment_id = Column(Integer, ForeignKey("deployments.id"), nullable=True)

    # Health check results
    health_check_passed = Column(Boolean, default=False)
    health_check_message = Column(Text)

    deployed_by = Column(Integer, ForeignKey("auth_user.id"), nullable=False)
    deployed_at = Column(DateTime, default=datetime.utcnow, nullable=False)
    completed_at = Column(DateTime)

    # Relationships
    module = relationship("Module", back_populates="deployments")
    deployer = relationship("User", foreign_keys=[deployed_by])
    previous_deployment = relationship(
        "Deployment",
        remote_side=[id],
        foreign_keys=[previous_deployment_id]
    )
