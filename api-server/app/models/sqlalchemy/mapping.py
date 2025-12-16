"""
Mapping database model for service-to-service routing configuration
"""

from sqlalchemy import Column, Integer, String, Boolean, Text, ForeignKey, DateTime
from sqlalchemy.orm import relationship
from datetime import datetime
from app.core.database import Base


class Mapping(Base):
    """
    Service-to-service mapping model for defining routing rules

    Defines which services can communicate with which other services,
    with protocol and port specifications.
    """

    __tablename__ = "mappings"

    id = Column(Integer, primary_key=True, index=True)
    name = Column(String(255), nullable=False, unique=True, index=True)
    description = Column(Text, nullable=True)

    # Cluster association
    cluster_id = Column(Integer, ForeignKey("clusters.id"), nullable=False, index=True)

    # Source and destination services (comma-separated IDs or "all")
    source_services = Column(String(500), nullable=False)
    dest_services = Column(String(500), nullable=False)

    # Protocol configuration (TCP, UDP, HTTP, HTTPS, etc.)
    protocols = Column(String(255), nullable=False, default="tcp,udp")

    # Port configuration (single, range, or comma-separated list)
    ports = Column(String(255), nullable=False)

    # Authentication requirements
    auth_required = Column(Boolean, default=True, nullable=False)

    # Additional configuration
    comments = Column(Text, nullable=True)
    is_active = Column(Boolean, default=True, nullable=False)

    # Audit fields
    created_by = Column(Integer, ForeignKey("users.id"), nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow, nullable=False)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)

    # Relationships
    cluster = relationship("Cluster", back_populates="mappings")
    creator = relationship("User", foreign_keys=[created_by])

    def __repr__(self):
        return f"<Mapping {self.name} (cluster {self.cluster_id})>"
