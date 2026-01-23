"""
SQLAlchemy Schema Models for MarchProxy Manager

IMPORTANT: This file is for DATABASE SCHEMA CREATION ONLY.
Use SQLAlchemy to define tables and generate SQL for initial schema setup.
PyDAL handles all runtime database operations and migrations.

For runtime operations, use the PyDAL models in:
- auth.py (users, sessions, api_tokens)
- cluster.py (clusters, user_cluster_assignments)
- proxy.py (proxy_servers, proxy_metrics)
- service.py (services, user_service_assignments)
- mapping.py (mappings)
- certificate.py (certificates, tls_proxy_cas, tls_proxy_configs)
- license.py (license_cache)
- block_rules.py (block_rules, block_rule_sync)
- rate_limiting.py (rate_limits, xdp_rate_limits, xdp_rate_limit_stats, xdp_rate_limit_whitelist)

SQLAlchemy 2.0 declarative syntax with proper relationships and indexes.
Run: python -c "from manager.models.sqlalchemy_schema import Base, engine; Base.metadata.create_all(engine)"

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import os
from datetime import datetime
from typing import Optional
from sqlalchemy import (
    create_engine,
    Column,
    Integer,
    String,
    Boolean,
    DateTime,
    Text,
    Float,
    BigInteger,
    ForeignKey,
    JSON,
    Index,
    UniqueConstraint,
    event,
)
from sqlalchemy.orm import declarative_base, relationship, sessionmaker
from sqlalchemy.pool import StaticPool

# Get database configuration from environment
DB_TYPE = os.getenv("DB_TYPE", "postgresql").lower()
DB_HOST = os.getenv("DB_HOST", "localhost")
DB_PORT = os.getenv("DB_PORT", "5432")
DB_USER = os.getenv("DB_USER", "marchproxy")
DB_PASSWORD = os.getenv("DB_PASSWORD", "password")
DB_NAME = os.getenv("DB_NAME", "marchproxy")

# Build connection string based on DB_TYPE
if DB_TYPE == "postgresql":
    DATABASE_URL = f"postgresql://{DB_USER}:{DB_PASSWORD}@{DB_HOST}:{DB_PORT}/{DB_NAME}"
elif DB_TYPE == "mysql":
    DATABASE_URL = f"mysql+pymysql://{DB_USER}:{DB_PASSWORD}@{DB_HOST}:{DB_PORT}/{DB_NAME}"
elif DB_TYPE == "mariadb":
    DATABASE_URL = f"mysql+pymysql://{DB_USER}:{DB_PASSWORD}@{DB_HOST}:{DB_PORT}/{DB_NAME}"
elif DB_TYPE == "sqlite":
    db_path = os.getenv("DB_PATH", "./marchproxy.db")
    DATABASE_URL = f"sqlite:///{db_path}"
else:
    DATABASE_URL = f"postgresql://{DB_USER}:{DB_PASSWORD}@{DB_HOST}:{DB_PORT}/{DB_NAME}"

# Create engine based on type
if DB_TYPE == "sqlite":
    engine = create_engine(
        DATABASE_URL, connect_args={"check_same_thread": False}, poolclass=StaticPool
    )
else:
    engine = create_engine(DATABASE_URL, pool_pre_ping=True, pool_size=10, max_overflow=20)

Base = declarative_base()


# ============================================================================
# AUTH MODELS (authentication, sessions, API tokens)
# ============================================================================


class User(Base):
    """User model for authentication"""

    __tablename__ = "users"

    id = Column(Integer, primary_key=True)
    username = Column(String(50), unique=True, nullable=False, index=True)
    email = Column(String(255), unique=True, nullable=False, index=True)
    password_hash = Column(String(255), nullable=False)
    is_admin = Column(Boolean, default=False)
    is_active = Column(Boolean, default=True, index=True)
    totp_secret = Column(String(32))
    totp_enabled = Column(Boolean, default=False)
    auth_provider = Column(String(50), default="local")
    external_id = Column(String(255))
    last_login = Column(DateTime)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    meta = Column("metadata", JSON, default={})

    # Relationships
    sessions = relationship("Session", back_populates="user", cascade="all, delete-orphan")
    api_tokens = relationship("APIToken", back_populates="user", cascade="all, delete-orphan")
    clusters_created = relationship(
        "Cluster", back_populates="created_by_user", foreign_keys="Cluster.created_by"
    )
    user_cluster_assignments = relationship(
        "UserClusterAssignment", back_populates="user", cascade="all, delete-orphan"
    )
    services_created = relationship(
        "Service", back_populates="created_by_user", foreign_keys="Service.created_by"
    )
    user_service_assignments = relationship(
        "UserServiceAssignment", back_populates="user", cascade="all, delete-orphan"
    )
    mappings_created = relationship(
        "Mapping", back_populates="created_by_user", foreign_keys="Mapping.created_by"
    )
    certificates_created = relationship(
        "Certificate",
        back_populates="created_by_user",
        foreign_keys="Certificate.created_by",
    )
    block_rules_created = relationship(
        "BlockRule",
        back_populates="created_by_user",
        foreign_keys="BlockRule.created_by",
    )
    xdp_rate_limits = relationship(
        "XDPRateLimit",
        back_populates="created_by_user",
        foreign_keys="XDPRateLimit.created_by",
    )
    xdp_whitelist = relationship(
        "XDPRateLimitWhitelist",
        back_populates="created_by_user",
        foreign_keys="XDPRateLimitWhitelist.created_by",
    )
    tls_proxy_cas = relationship(
        "TLSProxyCA",
        back_populates="created_by_user",
        foreign_keys="TLSProxyCA.created_by",
    )
    user_roles = relationship(
        "UserRole",
        back_populates="user",
        foreign_keys="UserRole.user_id",
        cascade="all, delete-orphan",
    )
    permission_cache = relationship(
        "UserPermissionCache",
        back_populates="user",
        uselist=False,
        cascade="all, delete-orphan",
    )


class Session(Base):
    """Session model for managing user sessions"""

    __tablename__ = "sessions"

    id = Column(Integer, primary_key=True)
    session_id = Column(String(64), unique=True, nullable=False, index=True)
    user_id = Column(Integer, ForeignKey("users.id"), nullable=False)
    ip_address = Column(String(45))
    user_agent = Column(String(255))
    data = Column(JSON)
    expires_at = Column(DateTime, nullable=False, index=True)
    created_at = Column(DateTime, default=datetime.utcnow)
    last_activity = Column(DateTime, default=datetime.utcnow)

    # Relationships
    user = relationship("User", back_populates="sessions")

    __table_args__ = (
        Index("idx_sessions_user_id", "user_id"),
        Index("idx_sessions_expires_at", "expires_at"),
    )


class APIToken(Base):
    """API Token model for service authentication"""

    __tablename__ = "api_tokens"

    id = Column(Integer, primary_key=True)
    token_id = Column(String(64), unique=True, nullable=False, index=True)
    name = Column(String(100), nullable=False)
    token_hash = Column(String(255), nullable=False)
    user_id = Column(Integer, ForeignKey("users.id"))
    service_id = Column(Integer, ForeignKey("services.id"))
    cluster_id = Column(Integer, ForeignKey("clusters.id"))
    permissions = Column(JSON, default={})
    expires_at = Column(DateTime)
    last_used = Column(DateTime)
    is_active = Column(Boolean, default=True, index=True)
    created_at = Column(DateTime, default=datetime.utcnow)
    meta = Column("metadata", JSON, default={})

    # Relationships
    user = relationship("User", back_populates="api_tokens")
    service = relationship("Service", foreign_keys=[service_id])
    cluster = relationship("Cluster", foreign_keys=[cluster_id])

    __table_args__ = (
        Index("idx_api_tokens_user_id", "user_id"),
        Index("idx_api_tokens_service_id", "service_id"),
        Index("idx_api_tokens_cluster_id", "cluster_id"),
        Index("idx_api_tokens_is_active", "is_active"),
    )


# ============================================================================
# CLUSTER MODELS (clusters, user assignments)
# ============================================================================


class Cluster(Base):
    """Cluster model for multi-tenant proxy management"""

    __tablename__ = "clusters"

    id = Column(Integer, primary_key=True)
    name = Column(String(100), unique=True, nullable=False, index=True)
    description = Column(Text)
    api_key_hash = Column(String(255), nullable=False)
    syslog_endpoint = Column(String(255))
    log_auth = Column(Boolean, default=True)
    log_netflow = Column(Boolean, default=True)
    log_debug = Column(Boolean, default=False)
    is_active = Column(Boolean, default=True, index=True)
    is_default = Column(Boolean, default=False, index=True)
    max_proxies = Column(Integer, default=3)
    created_by = Column(Integer, ForeignKey("users.id"), nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    meta = Column("metadata", JSON, default={})

    # Relationships
    created_by_user = relationship(
        "User", back_populates="clusters_created", foreign_keys=[created_by]
    )
    proxy_servers = relationship(
        "ProxyServer", back_populates="cluster", cascade="all, delete-orphan"
    )
    services = relationship("Service", back_populates="cluster", cascade="all, delete-orphan")
    mappings = relationship("Mapping", back_populates="cluster", cascade="all, delete-orphan")
    user_cluster_assignments = relationship(
        "UserClusterAssignment", back_populates="cluster", cascade="all, delete-orphan"
    )
    block_rules = relationship("BlockRule", back_populates="cluster", cascade="all, delete-orphan")
    xdp_rate_limits = relationship(
        "XDPRateLimit", back_populates="cluster", cascade="all, delete-orphan"
    )
    tls_proxy_cas = relationship(
        "TLSProxyCA", back_populates="cluster", cascade="all, delete-orphan"
    )

    __table_args__ = (
        Index("idx_clusters_created_by", "created_by"),
        Index("idx_clusters_is_active", "is_active"),
    )


class UserClusterAssignment(Base):
    """User-cluster assignment for Enterprise multi-cluster access"""

    __tablename__ = "user_cluster_assignments"

    id = Column(Integer, primary_key=True)
    user_id = Column(Integer, ForeignKey("users.id"), nullable=False, index=True)
    cluster_id = Column(Integer, ForeignKey("clusters.id"), nullable=False, index=True)
    role = Column(String(50), default="service_owner")
    assigned_by = Column(Integer, ForeignKey("users.id"), nullable=False)
    assigned_at = Column(DateTime, default=datetime.utcnow)
    is_active = Column(Boolean, default=True, index=True)

    # Relationships
    user = relationship("User", back_populates="user_cluster_assignments", foreign_keys=[user_id])
    cluster = relationship("Cluster", back_populates="user_cluster_assignments")
    assigned_by_user = relationship("User", foreign_keys=[assigned_by])

    __table_args__ = (
        UniqueConstraint("user_id", "cluster_id", name="uq_user_cluster_assignments"),
        Index("idx_user_cluster_assignments_user_id", "user_id"),
        Index("idx_user_cluster_assignments_cluster_id", "cluster_id"),
    )


# ============================================================================
# PROXY MODELS (proxy servers, metrics)
# ============================================================================


class ProxyServer(Base):
    """Proxy server registration and status management"""

    __tablename__ = "proxy_servers"

    id = Column(Integer, primary_key=True)
    name = Column(String(100), unique=True, nullable=False, index=True)
    hostname = Column(String(255), nullable=False)
    ip_address = Column(String(45), nullable=False)
    port = Column(Integer, default=8080)
    cluster_id = Column(Integer, ForeignKey("clusters.id"), nullable=False, index=True)
    status = Column(String(20), default="pending", index=True)
    version = Column(String(50))
    capabilities = Column(JSON, default={})
    license_validated = Column(Boolean, default=False)
    license_validation_at = Column(DateTime)
    last_seen = Column(DateTime, index=True)
    last_config_fetch = Column(DateTime)
    config_version = Column(String(64))
    registered_at = Column(DateTime, default=datetime.utcnow)
    meta = Column("metadata", JSON, default={})

    # Relationships
    cluster = relationship("Cluster", back_populates="proxy_servers")
    proxy_metrics = relationship(
        "ProxyMetric", back_populates="proxy_server", cascade="all, delete-orphan"
    )
    xdp_rate_limit_stats = relationship(
        "XDPRateLimitStats", back_populates="proxy", cascade="all, delete-orphan"
    )

    __table_args__ = (
        Index("idx_proxy_servers_cluster_id", "cluster_id"),
        Index("idx_proxy_servers_status", "status"),
        Index("idx_proxy_servers_last_seen", "last_seen"),
    )


class ProxyMetric(Base):
    """Proxy metrics and performance tracking"""

    __tablename__ = "proxy_metrics"

    id = Column(Integer, primary_key=True)
    proxy_id = Column(Integer, ForeignKey("proxy_servers.id"), nullable=False, index=True)
    timestamp = Column(DateTime, default=datetime.utcnow, index=True)
    cpu_usage = Column(Float)
    memory_usage = Column(Float)
    connections_active = Column(Integer)
    connections_total = Column(Integer)
    bytes_sent = Column(BigInteger)
    bytes_received = Column(BigInteger)
    requests_per_second = Column(Float)
    latency_avg = Column(Float)
    latency_p95 = Column(Float)
    errors_per_second = Column(Float)
    meta = Column("metadata", JSON, default={})

    # Relationships
    proxy_server = relationship("ProxyServer", back_populates="proxy_metrics")

    __table_args__ = (
        Index("idx_proxy_metrics_proxy_id", "proxy_id"),
        Index("idx_proxy_metrics_timestamp", "timestamp"),
    )


# ============================================================================
# SERVICE MODELS (services, user assignments)
# ============================================================================


class Service(Base):
    """Service model for proxy target configuration"""

    __tablename__ = "services"

    id = Column(Integer, primary_key=True)
    name = Column(String(100), unique=True, nullable=False, index=True)
    ip_fqdn = Column(String(255), nullable=False)
    port = Column(Integer, nullable=False)
    protocol = Column(String(10), default="tcp")
    collection = Column(String(100), index=True)
    cluster_id = Column(Integer, ForeignKey("clusters.id"), nullable=False, index=True)
    auth_type = Column(String(20), default="none")
    token_base64 = Column(String(255))
    jwt_secret = Column(String(255))
    jwt_expiry = Column(Integer, default=3600)
    jwt_algorithm = Column(String(10), default="HS256")
    tls_enabled = Column(Boolean, default=False)
    tls_verify = Column(Boolean, default=True)
    health_check_enabled = Column(Boolean, default=False)
    health_check_path = Column(String(255))
    health_check_interval = Column(Integer, default=30)
    is_active = Column(Boolean, default=True, index=True)
    created_by = Column(Integer, ForeignKey("users.id"), nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    meta = Column("metadata", JSON, default={})

    # Relationships
    cluster = relationship("Cluster", back_populates="services")
    created_by_user = relationship(
        "User", back_populates="services_created", foreign_keys=[created_by]
    )
    user_service_assignments = relationship(
        "UserServiceAssignment", back_populates="service", cascade="all, delete-orphan"
    )

    __table_args__ = (
        Index("idx_services_cluster_id", "cluster_id"),
        Index("idx_services_created_by", "created_by"),
        Index("idx_services_is_active", "is_active"),
        Index("idx_services_collection", "collection"),
    )


class UserServiceAssignment(Base):
    """User-service assignment for access control"""

    __tablename__ = "user_service_assignments"

    id = Column(Integer, primary_key=True)
    user_id = Column(Integer, ForeignKey("users.id"), nullable=False, index=True)
    service_id = Column(Integer, ForeignKey("services.id"), nullable=False, index=True)
    assigned_by = Column(Integer, ForeignKey("users.id"), nullable=False)
    assigned_at = Column(DateTime, default=datetime.utcnow)
    is_active = Column(Boolean, default=True, index=True)

    # Relationships
    user = relationship("User", back_populates="user_service_assignments", foreign_keys=[user_id])
    service = relationship("Service", back_populates="user_service_assignments")
    assigned_by_user = relationship("User", foreign_keys=[assigned_by])

    __table_args__ = (
        UniqueConstraint("user_id", "service_id", name="uq_user_service_assignments"),
        Index("idx_user_service_assignments_user_id", "user_id"),
        Index("idx_user_service_assignments_service_id", "service_id"),
    )


# ============================================================================
# MAPPING MODELS (source-destination routing)
# ============================================================================


class Mapping(Base):
    """Mapping model for source-destination service routing"""

    __tablename__ = "mappings"

    id = Column(Integer, primary_key=True)
    name = Column(String(100), unique=True, nullable=False, index=True)
    description = Column(Text)
    source_services = Column(JSON, nullable=False)  # List of service references
    dest_services = Column(JSON, nullable=False)  # List of service references
    cluster_id = Column(Integer, ForeignKey("clusters.id"), nullable=False, index=True)
    protocols = Column(JSON, default=["tcp"])
    ports = Column(JSON, nullable=False)  # List of port definitions
    auth_required = Column(Boolean, default=True)
    priority = Column(Integer, default=100)
    is_active = Column(Boolean, default=True, index=True)
    created_by = Column(Integer, ForeignKey("users.id"), nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    comments = Column(Text)
    meta = Column("metadata", JSON, default={})

    # Relationships
    cluster = relationship("Cluster", back_populates="mappings")
    created_by_user = relationship(
        "User", back_populates="mappings_created", foreign_keys=[created_by]
    )

    __table_args__ = (
        Index("idx_mappings_cluster_id", "cluster_id"),
        Index("idx_mappings_created_by", "created_by"),
        Index("idx_mappings_is_active", "is_active"),
        Index("idx_mappings_priority", "priority"),
    )


# ============================================================================
# CERTIFICATE MODELS (TLS certificates, CA, proxy configs)
# ============================================================================


class Certificate(Base):
    """Certificate model for TLS certificate management"""

    __tablename__ = "certificates"

    id = Column(Integer, primary_key=True)
    name = Column(String(100), unique=True, nullable=False, index=True)
    description = Column(Text)
    cert_data = Column(Text, nullable=False)
    key_data = Column(Text, nullable=False)
    ca_bundle = Column(Text)
    source_type = Column(String(20), nullable=False)  # upload, infisical, vault
    source_config = Column(JSON, default={})
    domain_names = Column(JSON, default=[])
    issuer = Column(String(255))
    serial_number = Column(String(100))
    fingerprint_sha256 = Column(String(64))
    auto_renew = Column(Boolean, default=False)
    renewal_threshold_days = Column(Integer, default=30)
    issued_at = Column(DateTime)
    expires_at = Column(DateTime, index=True)
    next_renewal_check = Column(DateTime)
    renewal_attempts = Column(Integer, default=0)
    last_renewal_attempt = Column(DateTime)
    renewal_error = Column(Text)
    is_active = Column(Boolean, default=True, index=True)
    created_by = Column(Integer, ForeignKey("users.id"), nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    meta = Column("metadata", JSON, default={})

    # Relationships
    created_by_user = relationship(
        "User", back_populates="certificates_created", foreign_keys=[created_by]
    )

    __table_args__ = (
        Index("idx_certificates_created_by", "created_by"),
        Index("idx_certificates_expires_at", "expires_at"),
        Index("idx_certificates_is_active", "is_active"),
    )


class TLSProxyCA(Base):
    """TLS Proxy CA management for TLS proxying (Enterprise feature)"""

    __tablename__ = "tls_proxy_cas"

    id = Column(Integer, primary_key=True)
    name = Column(String(100), unique=True, nullable=False)
    description = Column(Text)
    cluster_id = Column(Integer, ForeignKey("clusters.id"), nullable=False, index=True)

    # CA Certificate and Key
    ca_cert_data = Column(Text, nullable=False)
    ca_key_data = Column(Text, nullable=False)
    ca_cert_chain = Column(Text)

    # CA Metadata
    ca_subject = Column(String(255))
    ca_serial_number = Column(String(100))
    ca_fingerprint_sha256 = Column(String(64))
    ca_issued_at = Column(DateTime)
    ca_expires_at = Column(DateTime)

    # Wildcard Certificate for proxying
    wildcard_cert_data = Column(Text, nullable=False)
    wildcard_key_data = Column(Text, nullable=False)
    wildcard_domain = Column(String(255), nullable=False)
    wildcard_issued_at = Column(DateTime)
    wildcard_expires_at = Column(DateTime)
    wildcard_serial_number = Column(String(100))

    # Generation Configuration
    key_type = Column(String(20), default="ecc")  # 'ecc' or 'rsa'
    key_size = Column(Integer, default=384)
    hash_algorithm = Column(String(20), default="sha512")
    lifetime_years = Column(Integer, default=10)

    # Usage Configuration
    enabled = Column(Boolean, default=False)
    auto_generated = Column(Boolean, default=False)
    requires_enterprise = Column(Boolean, default=True)
    license_validated = Column(Boolean, default=False)

    # Management
    created_by = Column(Integer, ForeignKey("users.id"), nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    is_active = Column(Boolean, default=True, index=True)
    meta = Column("metadata", JSON, default={})

    # Relationships
    cluster = relationship("Cluster", back_populates="tls_proxy_cas")
    created_by_user = relationship(
        "User", back_populates="tls_proxy_cas", foreign_keys=[created_by]
    )
    tls_proxy_configs = relationship(
        "TLSProxyConfig", back_populates="ca", cascade="all, delete-orphan"
    )

    __table_args__ = (
        Index("idx_tls_proxy_cas_cluster_id", "cluster_id"),
        Index("idx_tls_proxy_cas_created_by", "created_by"),
        Index("idx_tls_proxy_cas_is_active", "is_active"),
    )


class TLSProxyConfig(Base):
    """TLS proxy configuration (Enterprise feature)"""

    __tablename__ = "tls_proxy_configs"

    id = Column(Integer, primary_key=True)
    name = Column(String(100), nullable=False)
    cluster_id = Column(Integer, ForeignKey("clusters.id"), nullable=False, index=True)
    ca_id = Column(Integer, ForeignKey("tls_proxy_cas.id"), nullable=False)

    # Protocol Detection Settings
    enabled = Column(Boolean, default=False)
    protocol_detection = Column(Boolean, default=True)
    port_based_detection = Column(Boolean, default=False)
    target_ports = Column(JSON, default=[])

    # TLS Proxy Behavior
    intercept_mode = Column(String(20), default="transparent")
    certificate_validation = Column(String(20), default="none")
    preserve_sni = Column(Boolean, default=True)
    log_connections = Column(Boolean, default=True)
    log_decrypted_content = Column(Boolean, default=False)

    # Performance Settings
    max_concurrent_connections = Column(Integer, default=10000)
    connection_timeout_seconds = Column(Integer, default=300)
    buffer_size_kb = Column(Integer, default=64)

    # Enterprise Features
    requires_enterprise = Column(Boolean, default=True)
    license_validated = Column(Boolean, default=False)

    # Management
    created_by = Column(Integer, ForeignKey("users.id"), nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    is_active = Column(Boolean, default=True, index=True)
    priority = Column(Integer, default=100)

    # Relationships
    ca = relationship("TLSProxyCA", back_populates="tls_proxy_configs")
    created_by_user = relationship("User", foreign_keys=[created_by])

    __table_args__ = (
        Index("idx_tls_proxy_configs_cluster_id", "cluster_id"),
        Index("idx_tls_proxy_configs_ca_id", "ca_id"),
        Index("idx_tls_proxy_configs_is_active", "is_active"),
    )


# ============================================================================
# LICENSE MODELS (license validation caching)
# ============================================================================


class LicenseCache(Base):
    """License cache model for storing validation results"""

    __tablename__ = "license_cache"

    id = Column(Integer, primary_key=True)
    license_key = Column(String(255), unique=True, nullable=False, index=True)
    validation_data = Column(JSON, default={})
    is_valid = Column(Boolean, default=False)
    is_enterprise = Column(Boolean, default=False)
    max_proxies = Column(Integer, default=3)
    features = Column(JSON, default={})
    expires_at = Column(DateTime, index=True)
    last_validated = Column(DateTime, default=datetime.utcnow, index=True)
    last_keepalive = Column(DateTime)
    keepalive_count = Column(Integer, default=0)
    validation_count = Column(Integer, default=0)
    error_message = Column(Text)

    __table_args__ = (
        Index("idx_license_cache_license_key", "license_key"),
        Index("idx_license_cache_is_valid", "is_valid"),
        Index("idx_license_cache_last_validated", "last_validated"),
    )


# ============================================================================
# BLOCK RULES MODELS (threat intelligence, traffic control)
# ============================================================================


class BlockRule(Base):
    """Block rule model for threat intelligence and traffic control"""

    __tablename__ = "block_rules"

    id = Column(Integer, primary_key=True)
    name = Column(String(255), nullable=False, index=True)
    description = Column(Text)
    cluster_id = Column(Integer, ForeignKey("clusters.id"), nullable=False, index=True)
    rule_type = Column(String(20), nullable=False)  # ip, cidr, domain, url_pattern, port
    layer = Column(String(5), nullable=False)  # L4, L7
    value = Column(String(500), nullable=False)
    ports = Column(JSON)  # List of ports
    protocols = Column(JSON, default=["tcp", "udp"])  # List of protocols
    wildcard = Column(Boolean, default=False)
    match_type = Column(String(20), default="exact")  # exact, prefix, suffix, regex, contains
    action = Column(String(20), default="deny")  # deny, drop, allow, log
    priority = Column(Integer, default=1000, index=True)
    apply_to_alb = Column(Boolean, default=True)
    apply_to_nlb = Column(Boolean, default=True)
    apply_to_egress = Column(Boolean, default=True)
    source = Column(String(20), default="manual")  # manual, threat_feed, api
    source_feed_name = Column(String(255))
    is_active = Column(Boolean, default=True, index=True)
    expires_at = Column(DateTime)
    hit_count = Column(Integer, default=0)
    last_hit = Column(DateTime)
    created_by = Column(Integer, ForeignKey("users.id"))
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)

    # Relationships
    cluster = relationship("Cluster", back_populates="block_rules")
    created_by_user = relationship(
        "User", back_populates="block_rules_created", foreign_keys=[created_by]
    )

    __table_args__ = (
        Index("idx_block_rules_cluster_id", "cluster_id"),
        Index("idx_block_rules_created_by", "created_by"),
        Index("idx_block_rules_is_active", "is_active"),
        Index("idx_block_rules_priority", "priority"),
        Index("idx_block_rules_rule_type", "rule_type"),
        Index("idx_block_rules_layer", "layer"),
    )


class BlockRuleSync(Base):
    """Block rule sync tracking model"""

    __tablename__ = "block_rule_sync"

    id = Column(Integer, primary_key=True)
    proxy_id = Column(Integer, ForeignKey("proxy_servers.id"), nullable=False, unique=True)
    last_sync_version = Column(String(64))
    last_sync_at = Column(DateTime)
    rules_count = Column(Integer, default=0)
    sync_status = Column(String(20), default="pending")  # pending, synced, error
    sync_error = Column(Text)

    __table_args__ = (Index("idx_block_rule_sync_proxy_id", "proxy_id"),)


# ============================================================================
# RATE LIMITING MODELS (API and XDP network-level)
# ============================================================================


class RateLimit(Base):
    """Rate limiting model for API endpoints"""

    __tablename__ = "rate_limits"

    id = Column(Integer, primary_key=True)
    client_id = Column(String(255), nullable=False, index=True)  # IP or user ID
    endpoint = Column(String(255), nullable=False, index=True)
    request_count = Column(Integer, default=0)
    window_start = Column(DateTime, nullable=False)
    last_request = Column(DateTime, default=datetime.utcnow)
    is_blocked = Column(Boolean, default=False, index=True)
    block_until = Column(DateTime)
    meta = Column("metadata", JSON, default={})

    __table_args__ = (
        Index("idx_rate_limits_client_id", "client_id"),
        Index("idx_rate_limits_endpoint", "endpoint"),
        Index("idx_rate_limits_is_blocked", "is_blocked"),
        UniqueConstraint("client_id", "endpoint", name="uq_rate_limits_client_endpoint"),
    )


class XDPRateLimit(Base):
    """XDP network-level rate limiting model (Enterprise)"""

    __tablename__ = "xdp_rate_limits"

    id = Column(Integer, primary_key=True)
    cluster_id = Column(Integer, ForeignKey("clusters.id"), nullable=False, index=True)
    name = Column(String(255), nullable=False)
    description = Column(Text)
    enabled = Column(Boolean, default=False)

    # Global rate limits
    global_pps_limit = Column(Integer, default=0)  # 0 = unlimited
    global_enabled = Column(Boolean, default=False)

    # Per-IP rate limits
    per_ip_pps_limit = Column(Integer, default=0)  # 0 = unlimited
    per_ip_enabled = Column(Boolean, default=True)

    # Timing configuration
    window_size_ns = Column(BigInteger, default=1000000000)  # 1 second in nanoseconds
    burst_allowance = Column(Integer, default=100)

    # Action configuration
    action = Column(Integer, default=1)  # 0=PASS, 1=DROP, 2=RATE_LIMIT

    # Network interface configuration
    interfaces = Column(JSON, default=[])

    # License and feature validation
    requires_enterprise = Column(Boolean, default=True)
    license_validated = Column(Boolean, default=False)
    license_last_check = Column(DateTime)

    # Priority and ordering
    priority = Column(Integer, default=100, index=True)

    # Management
    created_by = Column(Integer, ForeignKey("users.id"), nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    is_active = Column(Boolean, default=True, index=True)
    meta = Column("metadata", JSON, default={})

    # Relationships
    cluster = relationship("Cluster", back_populates="xdp_rate_limits")
    created_by_user = relationship(
        "User", back_populates="xdp_rate_limits", foreign_keys=[created_by]
    )
    stats = relationship(
        "XDPRateLimitStats", back_populates="rate_limit", cascade="all, delete-orphan"
    )
    whitelist = relationship(
        "XDPRateLimitWhitelist",
        back_populates="rate_limit",
        cascade="all, delete-orphan",
    )

    __table_args__ = (
        Index("idx_xdp_rate_limits_cluster_id", "cluster_id"),
        Index("idx_xdp_rate_limits_created_by", "created_by"),
        Index("idx_xdp_rate_limits_is_active", "is_active"),
        Index("idx_xdp_rate_limits_priority", "priority"),
    )


class XDPRateLimitStats(Base):
    """XDP rate limit statistics model"""

    __tablename__ = "xdp_rate_limit_stats"

    id = Column(Integer, primary_key=True)
    rate_limit_id = Column(Integer, ForeignKey("xdp_rate_limits.id"), nullable=False, index=True)
    proxy_id = Column(Integer, ForeignKey("proxy_servers.id"), nullable=False, index=True)
    interface_name = Column(String(32))

    # Statistics data
    total_packets = Column(BigInteger, default=0)
    passed_packets = Column(BigInteger, default=0)
    dropped_packets = Column(BigInteger, default=0)
    rate_limited_ips = Column(BigInteger, default=0)
    global_drops = Column(BigInteger, default=0)
    per_ip_drops = Column(BigInteger, default=0)

    # Timing
    stats_timestamp = Column(DateTime, default=datetime.utcnow, index=True)
    collection_interval = Column(Integer, default=60)  # seconds

    # Performance metrics
    cpu_usage_percent = Column(Float)
    memory_usage_bytes = Column(BigInteger)
    xdp_processing_time_ns = Column(BigInteger)

    meta = Column("metadata", JSON, default={})

    # Relationships
    rate_limit = relationship("XDPRateLimit", back_populates="stats")
    proxy = relationship("ProxyServer", back_populates="xdp_rate_limit_stats")

    __table_args__ = (
        Index("idx_xdp_rate_limit_stats_rate_limit_id", "rate_limit_id"),
        Index("idx_xdp_rate_limit_stats_proxy_id", "proxy_id"),
        Index("idx_xdp_rate_limit_stats_stats_timestamp", "stats_timestamp"),
    )


class XDPRateLimitWhitelist(Base):
    """IP whitelist for XDP rate limiting exceptions"""

    __tablename__ = "xdp_rate_limit_whitelist"

    id = Column(Integer, primary_key=True)
    rate_limit_id = Column(Integer, ForeignKey("xdp_rate_limits.id"), nullable=False, index=True)
    ip_address = Column(String(45), nullable=False)  # Support IPv4 and IPv6
    ip_mask = Column(Integer, default=32)  # CIDR mask
    description = Column(Text)
    whitelist_type = Column(String(32), default="manual")  # manual, automatic, temporary
    expires_at = Column(DateTime)
    created_by = Column(Integer, ForeignKey("users.id"), nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow)
    is_active = Column(Boolean, default=True, index=True)

    # Relationships
    rate_limit = relationship("XDPRateLimit", back_populates="whitelist")
    created_by_user = relationship(
        "User", back_populates="xdp_whitelist", foreign_keys=[created_by]
    )

    __table_args__ = (
        Index("idx_xdp_rate_limit_whitelist_rate_limit_id", "rate_limit_id"),
        Index("idx_xdp_rate_limit_whitelist_created_by", "created_by"),
        Index("idx_xdp_rate_limit_whitelist_is_active", "is_active"),
    )


# ============================================================================
# RBAC (Role-Based Access Control) Tables
# ============================================================================


class Role(Base):
    """Roles with OAuth2-style scoped permissions"""

    __tablename__ = "roles"

    id = Column(Integer, primary_key=True)
    name = Column(String(50), nullable=False, unique=True)
    display_name = Column(String(100), nullable=False)
    description = Column(Text)
    scope = Column(String(20), nullable=False)  # global, cluster, service
    permissions = Column(JSON, default=[])  # List of permission scopes
    is_system = Column(Boolean, default=False)  # System role (cannot be deleted)
    is_active = Column(Boolean, default=True, index=True)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)

    # Relationships
    user_roles = relationship("UserRole", back_populates="role")

    __table_args__ = (
        Index("idx_roles_name", "name"),
        Index("idx_roles_scope", "scope"),
        Index("idx_roles_is_active", "is_active"),
    )


class UserRole(Base):
    """User role assignments with scope"""

    __tablename__ = "user_roles"

    id = Column(Integer, primary_key=True)
    user_id = Column(Integer, ForeignKey("users.id"), nullable=False, index=True)
    role_id = Column(Integer, ForeignKey("roles.id"), nullable=False, index=True)
    scope = Column(String(20), nullable=False)  # global, cluster, service
    resource_id = Column(Integer)  # Cluster or Service ID (null for global)
    granted_by = Column(Integer, ForeignKey("users.id"))
    granted_at = Column(DateTime, default=datetime.utcnow)
    expires_at = Column(DateTime)  # Optional expiration
    is_active = Column(Boolean, default=True, index=True)

    # Relationships
    user = relationship("User", foreign_keys=[user_id], back_populates="user_roles")
    role = relationship("Role", back_populates="user_roles")
    granted_by_user = relationship("User", foreign_keys=[granted_by])

    __table_args__ = (
        Index("idx_user_roles_user_id", "user_id"),
        Index("idx_user_roles_role_id", "role_id"),
        Index("idx_user_roles_scope", "scope"),
        Index("idx_user_roles_resource_id", "resource_id"),
        Index("idx_user_roles_is_active", "is_active"),
    )


class UserPermissionCache(Base):
    """Denormalized permission cache for performance"""

    __tablename__ = "user_permissions_cache"

    id = Column(Integer, primary_key=True)
    user_id = Column(Integer, ForeignKey("users.id"), nullable=False, unique=True, index=True)
    global_permissions = Column(JSON, default=[])  # List of global permissions
    cluster_permissions = Column(JSON, default={})  # {cluster_id: [perms]}
    service_permissions = Column(JSON, default={})  # {service_id: [perms]}
    last_updated = Column(DateTime, default=datetime.utcnow)

    # Relationships
    user = relationship("User", back_populates="permission_cache")

    __table_args__ = (Index("idx_user_permissions_cache_user_id", "user_id"),)


# ============================================================================
# Database initialization and session management
# ============================================================================


def create_all_tables():
    """Create all tables in the database"""
    Base.metadata.create_all(engine)


def get_session():
    """Get a new database session"""
    Session = sessionmaker(bind=engine)
    return Session()


def init_db():
    """Initialize the database with all schema"""
    create_all_tables()
    print(f"Database initialized successfully using {DB_TYPE}")


if __name__ == "__main__":
    init_db()
