"""Proxy Server SQLAlchemy models - migrated from PYDAL"""

from datetime import datetime
from sqlalchemy import BigInteger, Boolean, Column, DateTime, Float, ForeignKey, Integer, JSON, String
from sqlalchemy.orm import relationship
from app.core.database import Base


class ProxyServer(Base):
    __tablename__ = "proxy_servers"

    id = Column(Integer, primary_key=True, index=True)
    name = Column(String(100), unique=True, nullable=False, index=True)
    hostname = Column(String(255), nullable=False)
    ip_address = Column(String(45), nullable=False)
    port = Column(Integer, default=8080)
    cluster_id = Column(Integer, ForeignKey("clusters.id"), nullable=False)
    status = Column(String(20), default="pending")
    version = Column(String(50))
    capabilities = Column(JSON)
    license_validated = Column(Boolean, default=False)
    license_validation_at = Column(DateTime)
    last_seen = Column(DateTime)
    last_config_fetch = Column(DateTime)
    config_version = Column(String(64))
    registered_at = Column(DateTime, default=datetime.utcnow)
    metadata = Column(JSON)

    cluster = relationship("Cluster", back_populates="proxies")
    metrics = relationship("ProxyMetrics", back_populates="proxy")


class ProxyMetrics(Base):
    __tablename__ = "proxy_metrics"

    id = Column(Integer, primary_key=True, index=True)
    proxy_id = Column(Integer, ForeignKey("proxy_servers.id"), nullable=False)
    timestamp = Column(DateTime, default=datetime.utcnow, nullable=False)
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
    metadata = Column(JSON)

    proxy = relationship("ProxyServer", back_populates="metrics")
