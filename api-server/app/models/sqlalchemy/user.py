"""
User SQLAlchemy model

Migrated from py4web auth_user table.
"""

from datetime import datetime
from typing import Optional

from sqlalchemy import Boolean, Column, DateTime, Integer, String, Text
from sqlalchemy.orm import relationship

from app.core.database import Base


class User(Base):
    """User model for authentication and authorization"""

    __tablename__ = "auth_user"

    id = Column(Integer, primary_key=True, index=True)
    email = Column(String(255), unique=True, nullable=False, index=True)
    username = Column(String(128), unique=True, nullable=False, index=True)
    password_hash = Column(String(255), nullable=False)
    first_name = Column(String(128))
    last_name = Column(String(128))

    # 2FA/TOTP
    totp_secret = Column(String(32))
    totp_enabled = Column(Boolean, default=False)

    # Account status
    is_active = Column(Boolean, default=True)
    is_admin = Column(Boolean, default=False)
    is_verified = Column(Boolean, default=False)

    # Timestamps
    created_at = Column(DateTime, default=datetime.utcnow, nullable=False)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    last_login = Column(DateTime)

    # Relationships
    created_clusters = relationship("Cluster", back_populates="creator", foreign_keys="Cluster.created_by")
    cluster_assignments = relationship("UserClusterAssignment", back_populates="user")
    service_assignments = relationship("UserServiceAssignment", back_populates="user")

    def __repr__(self):
        return f"<User(id={self.id}, username={self.username}, email={self.email})>"
