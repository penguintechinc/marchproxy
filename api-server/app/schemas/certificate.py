"""
Certificate Pydantic schemas for request/response validation
"""

from datetime import datetime
from typing import Optional
from pydantic import BaseModel, Field, validator

from app.models.sqlalchemy.certificate import CertificateSource


class CertificateBase(BaseModel):
    """Base certificate schema with common fields"""
    name: str = Field(..., min_length=1, max_length=100, description="Certificate name")
    description: Optional[str] = Field(None, description="Certificate description")
    source_type: CertificateSource = Field(..., description="Certificate source (infisical, vault, upload)")
    auto_renew: bool = Field(default=False, description="Enable automatic renewal")
    renew_before_days: int = Field(default=30, ge=1, le=90, description="Renew before expiry (days)")


class CertificateUpload(CertificateBase):
    """Schema for direct certificate upload"""
    cert_data: str = Field(..., min_length=1, description="PEM-encoded certificate")
    key_data: str = Field(..., min_length=1, description="PEM-encoded private key")
    ca_chain: Optional[str] = Field(None, description="PEM-encoded CA chain")

    @validator("cert_data", "key_data", "ca_chain")
    def validate_pem_format(cls, v):
        """Validate PEM format"""
        if v and not (v.startswith("-----BEGIN") and "-----END" in v):
            raise ValueError("Must be in PEM format")
        return v


class CertificateInfisical(CertificateBase):
    """Schema for Infisical certificate"""
    infisical_secret_path: str = Field(..., description="Infisical secret path")
    infisical_project_id: str = Field(..., description="Infisical project ID")
    infisical_environment: str = Field(default="production", description="Infisical environment")


class CertificateVault(CertificateBase):
    """Schema for HashiCorp Vault certificate"""
    vault_path: str = Field(..., description="Vault PKI path")
    vault_role: str = Field(..., description="Vault PKI role")
    vault_common_name: str = Field(..., description="Common name for certificate")


class CertificateCreate(BaseModel):
    """Union schema for creating certificates from any source"""
    # Will be validated based on source_type
    name: str = Field(..., min_length=1, max_length=100)
    description: Optional[str] = None
    source_type: CertificateSource
    auto_renew: bool = False
    renew_before_days: int = Field(default=30, ge=1, le=90)

    # Upload fields (required if source_type=upload)
    cert_data: Optional[str] = None
    key_data: Optional[str] = None
    ca_chain: Optional[str] = None

    # Infisical fields (required if source_type=infisical)
    infisical_secret_path: Optional[str] = None
    infisical_project_id: Optional[str] = None
    infisical_environment: Optional[str] = "production"

    # Vault fields (required if source_type=vault)
    vault_path: Optional[str] = None
    vault_role: Optional[str] = None
    vault_common_name: Optional[str] = None

    @validator("cert_data", "key_data")
    def validate_upload_fields(cls, v, values):
        """Validate required fields for upload source"""
        if values.get("source_type") == CertificateSource.UPLOAD:
            if not v:
                raise ValueError("cert_data and key_data required for upload source")
        return v

    @validator("infisical_secret_path", "infisical_project_id")
    def validate_infisical_fields(cls, v, values):
        """Validate required fields for Infisical source"""
        if values.get("source_type") == CertificateSource.INFISICAL:
            if not v:
                raise ValueError("Infisical configuration required for infisical source")
        return v

    @validator("vault_path", "vault_role", "vault_common_name")
    def validate_vault_fields(cls, v, values):
        """Validate required fields for Vault source"""
        if values.get("source_type") == CertificateSource.VAULT:
            if not v:
                raise ValueError("Vault configuration required for vault source")
        return v


class CertificateUpdate(BaseModel):
    """Schema for updating certificate"""
    description: Optional[str] = None
    auto_renew: Optional[bool] = None
    renew_before_days: Optional[int] = Field(None, ge=1, le=90)
    is_active: Optional[bool] = None


class CertificateResponse(BaseModel):
    """Schema for certificate response (excludes sensitive data)"""
    id: int
    name: str
    description: Optional[str]
    source_type: CertificateSource
    common_name: Optional[str]
    issuer: Optional[str]
    valid_from: Optional[datetime]
    valid_until: datetime
    auto_renew: bool
    renew_before_days: int
    is_active: bool
    is_expired: bool
    days_until_expiry: int
    needs_renewal: bool
    last_renewal: Optional[datetime]
    renewal_error: Optional[str]
    created_at: datetime
    updated_at: Optional[datetime]

    class Config:
        from_attributes = True


class CertificateDetailResponse(CertificateResponse):
    """Detailed certificate response (includes cert data, excludes private key)"""
    cert_data: str
    ca_chain: Optional[str]
    subject_alt_names: Optional[str]

    # Source-specific fields
    infisical_secret_path: Optional[str]
    infisical_project_id: Optional[str]
    infisical_environment: Optional[str]
    vault_path: Optional[str]
    vault_role: Optional[str]
    vault_common_name: Optional[str]

    class Config:
        from_attributes = True


class CertificateRenewResponse(BaseModel):
    """Response for certificate renewal operation"""
    certificate_id: int
    renewed: bool
    message: str
    valid_until: Optional[datetime]
    error: Optional[str]
