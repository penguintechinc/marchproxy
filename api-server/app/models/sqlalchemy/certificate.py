"""
Certificate SQLAlchemy model

Manages TLS certificates from multiple sources (Infisical, Vault, direct upload).
"""

from datetime import datetime
from typing import Optional

from sqlalchemy import Boolean, Column, DateTime, Integer, String, Text, Enum as SQLEnum
from sqlalchemy.orm import relationship
import enum

from app.core.database import Base


class CertificateSource(str, enum.Enum):
    """Source of the certificate"""
    INFISICAL = "infisical"
    VAULT = "vault"
    UPLOAD = "upload"


class Certificate(Base):
    """Certificate model for TLS management"""

    __tablename__ = "certificates"

    id = Column(Integer, primary_key=True, index=True)
    name = Column(String(100), unique=True, nullable=False, index=True)
    description = Column(Text)

    # Certificate data (PEM encoded)
    cert_data = Column(Text, nullable=False)
    key_data = Column(Text, nullable=False)
    ca_chain = Column(Text)  # Optional CA chain

    # Certificate metadata
    source_type = Column(SQLEnum(CertificateSource), nullable=False)
    common_name = Column(String(255))
    subject_alt_names = Column(Text)  # JSON array of SANs
    issuer = Column(String(255))

    # Validity dates
    valid_from = Column(DateTime)
    valid_until = Column(DateTime, nullable=False, index=True)

    # Auto-renewal configuration
    auto_renew = Column(Boolean, default=False)
    renew_before_days = Column(Integer, default=30)

    # Infisical configuration
    infisical_secret_path = Column(String(255))
    infisical_project_id = Column(String(100))
    infisical_environment = Column(String(50))

    # Vault configuration
    vault_path = Column(String(255))
    vault_role = Column(String(100))
    vault_common_name = Column(String(255))

    # Status
    is_active = Column(Boolean, default=True)
    last_renewal = Column(DateTime)
    renewal_error = Column(Text)

    # Audit fields
    created_by = Column(Integer, nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow, nullable=False)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)

    # Metadata
    metadata = Column(Text)  # JSON field for additional data

    def __repr__(self):
        return f"<Certificate(id={self.id}, name={self.name}, source={self.source_type})>"

    @property
    def is_expired(self) -> bool:
        """Check if certificate has expired"""
        if not self.valid_until:
            return False
        return datetime.utcnow() > self.valid_until

    @property
    def days_until_expiry(self) -> int:
        """Calculate days until certificate expires"""
        if not self.valid_until:
            return 0
        delta = self.valid_until - datetime.utcnow()
        return max(0, delta.days)

    @property
    def needs_renewal(self) -> bool:
        """Check if certificate needs renewal based on renew_before_days"""
        if not self.auto_renew:
            return False
        return self.days_until_expiry <= self.renew_before_days
