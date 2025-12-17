"""
Pydantic models for rate limiting configuration and status
"""

from pydantic import BaseModel, Field, validator
from typing import Optional
from datetime import datetime


class RateLimitConfig(BaseModel):
    """Rate limit configuration for an API key"""

    tpm_limit: int = Field(
        default=10000,
        ge=0,
        description="Tokens per minute limit (0 for unlimited)"
    )
    rpm_limit: int = Field(
        default=60,
        ge=0,
        description="Requests per minute limit (0 for unlimited)"
    )
    window_seconds: int = Field(
        default=60,
        ge=1,
        le=300,
        description="Time window for sliding window algorithm (1-300 seconds)"
    )
    enabled: bool = Field(
        default=True,
        description="Whether rate limiting is enabled"
    )

    @validator('tpm_limit', 'rpm_limit')
    def validate_limits(cls, v):
        """Validate that limits are reasonable"""
        if v < 0:
            raise ValueError("Limit cannot be negative")
        if v > 1_000_000:
            raise ValueError("Limit exceeds maximum allowed value")
        return v

    class Config:
        json_schema_extra = {
            "example": {
                "tpm_limit": 10000,
                "rpm_limit": 60,
                "window_seconds": 60,
                "enabled": True
            }
        }


class RateLimitStatus(BaseModel):
    """Current rate limit status for an API key"""

    current_tpm: int = Field(
        default=0,
        ge=0,
        description="Current tokens used in this window"
    )
    current_rpm: int = Field(
        default=0,
        ge=0,
        description="Current requests made in this window"
    )
    reset_at: datetime = Field(
        description="When the rate limit window resets"
    )
    is_limited: bool = Field(
        default=False,
        description="Whether the key is currently rate limited"
    )
    limit_reason: Optional[str] = Field(
        default=None,
        description="Reason for rate limiting (tpm_exceeded or rpm_exceeded)"
    )
    remaining_tpm: Optional[int] = Field(
        default=None,
        ge=0,
        description="Remaining tokens in this window"
    )
    remaining_rpm: Optional[int] = Field(
        default=None,
        ge=0,
        description="Remaining requests in this window"
    )

    class Config:
        json_schema_extra = {
            "example": {
                "current_tpm": 5000,
                "current_rpm": 30,
                "reset_at": "2025-12-16T12:00:00Z",
                "is_limited": False,
                "limit_reason": None,
                "remaining_tpm": 5000,
                "remaining_rpm": 30
            }
        }
