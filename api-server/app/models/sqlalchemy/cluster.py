"""Cluster SQLAlchemy models - migrated from PYDAL"""

from datetime import datetime
from sqlalchemy import Boolean, Column, DateTime, ForeignKey, Integer, JSON, String, Text
from sqlalchemy.orm import relationship
from app.core.database import Base


class Cluster(Base):
    __tablename__ = "clusters"

    id = Column(Integer, primary_key=True, index=True)
    name = Column(String(100), unique=True, nullable=False, index=True)
    description = Column(Text)
    api_key_hash = Column(String(255), nullable=False)
    syslog_endpoint = Column(String(255))
    log_auth = Column(Boolean, default=True)
    log_netflow = Column(Boolean, default=True)
    log_debug = Column(Boolean, default=False)
    is_active = Column(Boolean, default=True)
    is_default = Column(Boolean, default=False)
    max_proxies = Column(Integer, default=3)
    created_by = Column(Integer, ForeignKey("auth_user.id"), nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow, nullable=False)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    metadata = Column(JSON)

    creator = relationship("User", back_populates="created_clusters", foreign_keys=[created_by])
    services = relationship("Service", back_populates="cluster")
    proxies = relationship("ProxyServer", back_populates="cluster")
    user_assignments = relationship("UserClusterAssignment", back_populates="cluster")


class UserClusterAssignment(Base):
    __tablename__ = "user_cluster_assignments"

    id = Column(Integer, primary_key=True, index=True)
    user_id = Column(Integer, ForeignKey("auth_user.id"), nullable=False)
    cluster_id = Column(Integer, ForeignKey("clusters.id"), nullable=False)
    role = Column(String(50), default="service_owner")
    assigned_by = Column(Integer, ForeignKey("auth_user.id"), nullable=False)
    assigned_at = Column(DateTime, default=datetime.utcnow)
    is_active = Column(Boolean, default=True)

    user = relationship("User", back_populates="cluster_assignments", foreign_keys=[user_id])
    cluster = relationship("Cluster", back_populates="user_assignments")
