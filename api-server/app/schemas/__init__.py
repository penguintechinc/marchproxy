"""
Pydantic schemas for API request/response validation
"""

# Core Phase 2 schemas
from app.schemas.auth import (
    LoginRequest,
    LoginResponse,
    TokenResponse,
    RefreshTokenRequest,
    Enable2FAResponse,
    Verify2FARequest,
    ChangePasswordRequest,
)
from app.schemas.cluster import (
    ClusterBase,
    ClusterCreate,
    ClusterUpdate,
    ClusterResponse,
    ClusterListResponse,
    ClusterAPIKeyRotateResponse,
)
from app.schemas.service import (
    ServiceBase,
    ServiceCreate,
    ServiceUpdate,
    ServiceResponse,
    ServiceListResponse,
    ServiceTokenRotateRequest,
    ServiceTokenRotateResponse,
)
from app.schemas.proxy import (
    ProxyRegisterRequest,
    ProxyHeartbeatRequest,
    ProxyResponse,
    ProxyListResponse,
    ProxyConfigResponse,
    ProxyMetricsRequest,
)
from app.schemas.user import (
    UserBase,
    UserCreate,
    UserUpdate,
    UserResponse,
    UserListResponse,
    UserClusterAssignmentCreate,
    UserServiceAssignmentCreate,
)

# Enterprise schemas (Phase 3+) - optional imports
try:
    from app.schemas.traffic_shaping import (
        QoSPolicyCreate,
        QoSPolicyUpdate,
        QoSPolicyResponse,
        PriorityQueueConfig,
        BandwidthLimit
    )
    from app.schemas.multi_cloud import (
        RouteTableCreate,
        RouteTableUpdate,
        RouteTableResponse,
        HealthProbeConfig,
        RoutingAlgorithm,
        CloudProvider
    )
    from app.schemas.observability import (
        TracingConfigCreate,
        TracingConfigUpdate,
        TracingConfigResponse,
        SamplingStrategy,
        TracingBackend
    )
    _HAS_ENTERPRISE = True
except ImportError:
    _HAS_ENTERPRISE = False

__all__ = [
    # Auth schemas
    "LoginRequest",
    "LoginResponse",
    "TokenResponse",
    "RefreshTokenRequest",
    "Enable2FAResponse",
    "Verify2FARequest",
    "ChangePasswordRequest",
    # Cluster schemas
    "ClusterBase",
    "ClusterCreate",
    "ClusterUpdate",
    "ClusterResponse",
    "ClusterListResponse",
    "ClusterAPIKeyRotateResponse",
    # Service schemas
    "ServiceBase",
    "ServiceCreate",
    "ServiceUpdate",
    "ServiceResponse",
    "ServiceListResponse",
    "ServiceTokenRotateRequest",
    "ServiceTokenRotateResponse",
    # Proxy schemas
    "ProxyRegisterRequest",
    "ProxyHeartbeatRequest",
    "ProxyResponse",
    "ProxyListResponse",
    "ProxyConfigResponse",
    "ProxyMetricsRequest",
    # User schemas
    "UserBase",
    "UserCreate",
    "UserUpdate",
    "UserResponse",
    "UserListResponse",
    "UserClusterAssignmentCreate",
    "UserServiceAssignmentCreate",
]

if _HAS_ENTERPRISE:
    __all__.extend([
        "QoSPolicyCreate", "QoSPolicyUpdate", "QoSPolicyResponse",
        "PriorityQueueConfig", "BandwidthLimit",
        "RouteTableCreate", "RouteTableUpdate", "RouteTableResponse",
        "HealthProbeConfig", "RoutingAlgorithm", "CloudProvider",
        "TracingConfigCreate", "TracingConfigUpdate", "TracingConfigResponse",
        "SamplingStrategy", "TracingBackend",
    ])
