"""Service SQLAlchemy models - migrated from PYDAL"""

from datetime import datetime
from sqlalchemy import Boolean, Column, DateTime, ForeignKey, Integer, JSON, String, Text
from sqlalchemy.orm import relationship
from app.core.database import Base


class Service(Base):
    __tablename__ = "services"

    id = Column(Integer, primary_key=True, index=True)
    name = Column(String(100), unique=True, nullable=False, index=True)
    ip_fqdn = Column(String(255), nullable=False)
    port = Column(Integer, nullable=False)
    protocol = Column(String(10), default="tcp")
    collection = Column(String(100))
    cluster_id = Column(Integer, ForeignKey("clusters.id"), nullable=False)
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
    is_active = Column(Boolean, default=True)
    created_by = Column(Integer, ForeignKey("auth_user.id"), nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow, nullable=False)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    metadata = Column(JSON)

    cluster = relationship("Cluster", back_populates="services")
    user_assignments = relationship("UserServiceAssignment", back_populates="service")


class UserServiceAssignment(Base):
    __tablename__ = "user_service_assignments"

    id = Column(Integer, primary_key=True, index=True)
    user_id = Column(Integer, ForeignKey("auth_user.id"), nullable=False)
    service_id = Column(Integer, ForeignKey("services.id"), nullable=False)
    assigned_by = Column(Integer, ForeignKey("auth_user.id"), nullable=False)
    assigned_at = Column(DateTime, default=datetime.utcnow)
    is_active = Column(Boolean, default=True)

    user = relationship("User", back_populates="service_assignments", foreign_keys=[user_id])
    service = relationship("Service", back_populates="user_assignments")
