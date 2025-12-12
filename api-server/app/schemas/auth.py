"""
Authentication and authorization Pydantic schemas
"""

from typing import Optional
from pydantic import BaseModel, EmailStr, Field


class LoginRequest(BaseModel):
    """Login request schema"""
    username: str = Field(..., min_length=3, max_length=128, description="Username or email")
    password: str = Field(..., min_length=8, description="User password")
    totp_code: Optional[str] = Field(None, min_length=6, max_length=6, description="2FA TOTP code if enabled")


class TokenResponse(BaseModel):
    """JWT token response"""
    access_token: str = Field(..., description="JWT access token")
    refresh_token: str = Field(..., description="JWT refresh token")
    token_type: str = Field(default="bearer", description="Token type")
    expires_in: int = Field(..., description="Token expiry in seconds")


class LoginResponse(BaseModel):
    """Login response schema"""
    user_id: int
    username: str
    email: str
    is_admin: bool
    requires_2fa: bool
    access_token: Optional[str] = None
    refresh_token: Optional[str] = None
    token_type: str = "bearer"
    expires_in: Optional[int] = None


class RefreshTokenRequest(BaseModel):
    """Refresh token request"""
    refresh_token: str = Field(..., description="Valid refresh token")


class Enable2FAResponse(BaseModel):
    """2FA enrollment response"""
    secret: str = Field(..., description="Base32 encoded TOTP secret")
    qr_code_uri: str = Field(..., description="TOTP provisioning URI for QR code generation")
    backup_codes: list[str] = Field(..., description="One-time backup codes")


class Verify2FARequest(BaseModel):
    """2FA verification request"""
    totp_code: str = Field(..., min_length=6, max_length=6, description="6-digit TOTP code")


class ChangePasswordRequest(BaseModel):
    """Password change request"""
    current_password: str = Field(..., min_length=8, description="Current password")
    new_password: str = Field(..., min_length=8, description="New password")
    confirm_password: str = Field(..., min_length=8, description="Confirm new password")
