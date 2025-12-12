"""
SQLAlchemy models for Enterprise features

Database schema for traffic shaping, multi-cloud routing, and observability.
"""

from datetime import datetime
from typing import Optional

from sqlalchemy import (
    Column, Integer, String, Text, Boolean, Float, DateTime, JSON,
    ForeignKey, CheckConstraint, Index
)
from sqlalchemy.orm import relationship

from app.core.database import Base


class QoSPolicy(Base):
    """QoS Policy model for traffic shaping"""
    __tablename__ = "qos_policies"

    id = Column(Integer, primary_key=True, index=True)
    name = Column(String(100), nullable=False, index=True)
    description = Column(Text, nullable=True)
    service_id = Column(Integer, nullable=False, index=True)
    cluster_id = Column(Integer, nullable=False, index=True)

    # Bandwidth limits (stored as JSON for flexibility)
    bandwidth_config = Column(
        JSON,
        nullable=False,
        default={"ingress_mbps": None, "egress_mbps": None, "burst_size_kb": 1024}
    )

    # Priority queue configuration (stored as JSON)
    priority_config = Column(
        JSON,
        nullable=False,
        default={
            "priority": "P2",
            "weight": 1,
            "max_latency_ms": 100,
            "dscp_marking": "BE"
        }
    )

    enabled = Column(Boolean, nullable=False, default=True)
    created_at = Column(DateTime, nullable=False, default=datetime.utcnow)
    updated_at = Column(
        DateTime,
        nullable=False,
        default=datetime.utcnow,
        onupdate=datetime.utcnow
    )

    __table_args__ = (
        Index('idx_qos_service_cluster', 'service_id', 'cluster_id'),
        CheckConstraint(
            "name IS NOT NULL AND length(name) > 0",
            name='ck_qos_name_not_empty'
        ),
    )


class RouteTable(Base):
    """Route table model for multi-cloud routing"""
    __tablename__ = "route_tables"

    id = Column(Integer, primary_key=True, index=True)
    name = Column(String(100), nullable=False, index=True)
    description = Column(Text, nullable=True)
    service_id = Column(Integer, nullable=False, index=True)
    cluster_id = Column(Integer, nullable=False, index=True)

    # Routing algorithm: latency, cost, geo, weighted_rr, failover
    algorithm = Column(String(20), nullable=False, default='latency')

    # Routes configuration (array of route objects)
    routes = Column(
        JSON,
        nullable=False,
        default=[]
    )

    # Health probe configuration
    health_probe_config = Column(
        JSON,
        nullable=False,
        default={
            "protocol": "tcp",
            "port": None,
            "path": None,
            "interval_seconds": 30,
            "timeout_seconds": 5,
            "unhealthy_threshold": 3,
            "healthy_threshold": 2
        }
    )

    enable_auto_failover = Column(Boolean, nullable=False, default=True)
    enabled = Column(Boolean, nullable=False, default=True)
    created_at = Column(DateTime, nullable=False, default=datetime.utcnow)
    updated_at = Column(
        DateTime,
        nullable=False,
        default=datetime.utcnow,
        onupdate=datetime.utcnow
    )

    __table_args__ = (
        Index('idx_route_service_cluster', 'service_id', 'cluster_id'),
        CheckConstraint(
            "algorithm IN ('latency', 'cost', 'geo', 'weighted_rr', 'failover')",
            name='ck_route_algorithm'
        ),
    )


class RouteHealthStatus(Base):
    """Health status tracking for routes"""
    __tablename__ = "route_health_status"

    id = Column(Integer, primary_key=True, index=True)
    route_table_id = Column(
        Integer,
        nullable=False,
        index=True
    )
    endpoint = Column(String(255), nullable=False, index=True)
    is_healthy = Column(Boolean, nullable=False, default=True)
    last_check = Column(DateTime, nullable=False, default=datetime.utcnow)
    rtt_ms = Column(Float, nullable=True)
    consecutive_failures = Column(Integer, nullable=False, default=0)
    consecutive_successes = Column(Integer, nullable=False, default=0)
    last_error = Column(Text, nullable=True)

    __table_args__ = (
        Index('idx_health_route_endpoint', 'route_table_id', 'endpoint'),
        Index('idx_health_last_check', 'last_check'),
    )


class TracingConfig(Base):
    """Tracing configuration model for observability"""
    __tablename__ = "tracing_configs"

    id = Column(Integer, primary_key=True, index=True)
    name = Column(String(100), nullable=False, index=True)
    description = Column(Text, nullable=True)
    cluster_id = Column(Integer, nullable=False, index=True)

    # Backend: jaeger, zipkin, otlp
    backend = Column(String(20), nullable=False, default='jaeger')
    endpoint = Column(String(255), nullable=False)
    exporter = Column(String(20), nullable=False, default='grpc')

    # Sampling strategy: always, never, probabilistic, rate_limit, error_only, adaptive
    sampling_strategy = Column(String(20), nullable=False, default='probabilistic')
    sampling_rate = Column(Float, nullable=False, default=0.1)
    max_traces_per_second = Column(Integer, nullable=True)

    # Header/body inclusion
    include_request_headers = Column(Boolean, nullable=False, default=False)
    include_response_headers = Column(Boolean, nullable=False, default=False)
    include_request_body = Column(Boolean, nullable=False, default=False)
    include_response_body = Column(Boolean, nullable=False, default=False)
    max_attribute_length = Column(Integer, nullable=False, default=512)

    # Service identification
    service_name = Column(String(100), nullable=False, default='marchproxy')
    custom_tags = Column(JSON, nullable=True)

    enabled = Column(Boolean, nullable=False, default=True)
    created_at = Column(DateTime, nullable=False, default=datetime.utcnow)
    updated_at = Column(
        DateTime,
        nullable=False,
        default=datetime.utcnow,
        onupdate=datetime.utcnow
    )

    __table_args__ = (
        CheckConstraint(
            "backend IN ('jaeger', 'zipkin', 'otlp')",
            name='ck_tracing_backend'
        ),
        CheckConstraint(
            "exporter IN ('grpc', 'http', 'thrift')",
            name='ck_tracing_exporter'
        ),
        CheckConstraint(
            "sampling_strategy IN ('always', 'never', 'probabilistic', 'rate_limit', 'error_only', 'adaptive')",
            name='ck_sampling_strategy'
        ),
        CheckConstraint(
            "sampling_rate >= 0.0 AND sampling_rate <= 1.0",
            name='ck_sampling_rate'
        ),
    )


class TracingStats(Base):
    """Runtime tracing statistics"""
    __tablename__ = "tracing_stats"

    id = Column(Integer, primary_key=True, index=True)
    tracing_config_id = Column(
        Integer,
        nullable=False,
        index=True
    )
    timestamp = Column(DateTime, nullable=False, default=datetime.utcnow, index=True)

    total_spans = Column(Integer, nullable=False, default=0)
    sampled_spans = Column(Integer, nullable=False, default=0)
    dropped_spans = Column(Integer, nullable=False, default=0)
    error_spans = Column(Integer, nullable=False, default=0)
    avg_span_duration_ms = Column(Float, nullable=True)
    last_export = Column(DateTime, nullable=True)

    __table_args__ = (
        Index('idx_stats_config_timestamp', 'tracing_config_id', 'timestamp'),
    )
