"""
Pydantic models for virtual key management
"""

from datetime import datetime
from typing import Optional, List, Dict, Any
from pydantic import BaseModel, Field, validator
from enum import Enum


class KeyStatus(str, Enum):
    """Virtual key status"""
    ACTIVE = "active"
    INACTIVE = "inactive"
    EXPIRED = "expired"
    REVOKED = "revoked"


class VirtualKey(BaseModel):
    """Virtual key data model"""
    id: str = Field(..., description="Unique key identifier")
    key_hash: str = Field(..., description="SHA-256 hash of the full key")
    name: str = Field(..., description="Human-readable key name")
    user_id: str = Field(..., description="User who owns this key")
    team_id: Optional[str] = Field(None, description="Team ID if applicable")
    created_at: datetime = Field(
        default_factory=datetime.utcnow,
        description="Key creation timestamp"
    )
    expires_at: Optional[datetime] = Field(
        None,
        description="Key expiration timestamp"
    )
    is_active: bool = Field(
        default=True,
        description="Whether key is active"
    )
    allowed_models: List[str] = Field(
        default_factory=lambda: ["*"],
        description="List of allowed model names, '*' for all"
    )
    max_budget: Optional[float] = Field(
        None,
        description="Maximum budget in USD, None for unlimited"
    )
    spent: float = Field(
        default=0.0,
        description="Total amount spent in USD"
    )
    tpm_limit: Optional[int] = Field(
        None,
        description="Tokens per minute limit, None for unlimited"
    )
    rpm_limit: Optional[int] = Field(
        None,
        description="Requests per minute limit, None for unlimited"
    )
    metadata: Dict[str, Any] = Field(
        default_factory=dict,
        description="Additional metadata"
    )
    last_used: Optional[datetime] = Field(
        None,
        description="Last usage timestamp"
    )
    total_requests: int = Field(
        default=0,
        description="Total number of requests made with this key"
    )

    @validator('expires_at')
    def validate_expiry(cls, v, values):
        """Validate expiry date is in the future"""
        if v and v <= datetime.utcnow():
            raise ValueError("Expiry date must be in the future")
        return v

    @validator('max_budget')
    def validate_budget(cls, v):
        """Validate budget is positive"""
        if v is not None and v <= 0:
            raise ValueError("Budget must be positive")
        return v

    @validator('tpm_limit', 'rpm_limit')
    def validate_limits(cls, v):
        """Validate rate limits are positive"""
        if v is not None and v <= 0:
            raise ValueError("Rate limits must be positive")
        return v

    class Config:
        """Pydantic config"""
        json_encoders = {
            datetime: lambda v: v.isoformat()
        }

    def is_expired(self) -> bool:
        """Check if key is expired"""
        if not self.expires_at:
            return False
        return datetime.utcnow() > self.expires_at

    def is_budget_exceeded(self) -> bool:
        """Check if budget is exceeded"""
        if not self.max_budget:
            return False
        return self.spent >= self.max_budget

    def get_status(self) -> KeyStatus:
        """Get current key status"""
        if not self.is_active:
            return KeyStatus.INACTIVE
        if self.is_expired():
            return KeyStatus.EXPIRED
        if self.is_budget_exceeded():
            return KeyStatus.REVOKED
        return KeyStatus.ACTIVE


class KeyCreate(BaseModel):
    """Request model for creating a new virtual key"""
    name: str = Field(..., min_length=1, max_length=255)
    user_id: str = Field(..., min_length=1)
    team_id: Optional[str] = None
    expires_days: Optional[int] = Field(
        None,
        gt=0,
        description="Number of days until expiration"
    )
    allowed_models: Optional[List[str]] = Field(
        default=["*"],
        description="List of allowed models"
    )
    max_budget: Optional[float] = Field(
        None,
        gt=0,
        description="Maximum budget in USD"
    )
    tpm_limit: Optional[int] = Field(
        None,
        gt=0,
        description="Tokens per minute limit"
    )
    rpm_limit: Optional[int] = Field(
        None,
        gt=0,
        description="Requests per minute limit"
    )
    metadata: Dict[str, Any] = Field(
        default_factory=dict,
        description="Additional metadata"
    )

    @validator('name')
    def validate_name(cls, v):
        """Validate key name"""
        if not v or not v.strip():
            raise ValueError("Key name cannot be empty")
        return v.strip()


class KeyUpdate(BaseModel):
    """Request model for updating a virtual key"""
    name: Optional[str] = Field(None, min_length=1, max_length=255)
    is_active: Optional[bool] = None
    allowed_models: Optional[List[str]] = None
    max_budget: Optional[float] = Field(None, gt=0)
    tpm_limit: Optional[int] = Field(None, gt=0)
    rpm_limit: Optional[int] = Field(None, gt=0)
    metadata: Optional[Dict[str, Any]] = None

    class Config:
        """Pydantic config"""
        # Only include fields that were explicitly set
        exclude_unset = True


class KeyResponse(BaseModel):
    """Response model for key operations"""
    id: str
    name: str
    user_id: str
    team_id: Optional[str]
    created_at: datetime
    expires_at: Optional[datetime]
    status: KeyStatus
    allowed_models: List[str]
    max_budget: Optional[float]
    spent: float
    budget_remaining: Optional[float]
    tpm_limit: Optional[int]
    rpm_limit: Optional[int]
    last_used: Optional[datetime]
    total_requests: int
    metadata: Dict[str, Any]

    class Config:
        """Pydantic config"""
        json_encoders = {
            datetime: lambda v: v.isoformat()
        }

    @classmethod
    def from_virtual_key(cls, key: VirtualKey) -> "KeyResponse":
        """Create response from VirtualKey model"""
        budget_remaining = None
        if key.max_budget:
            budget_remaining = max(0.0, key.max_budget - key.spent)

        return cls(
            id=key.id,
            name=key.name,
            user_id=key.user_id,
            team_id=key.team_id,
            created_at=key.created_at,
            expires_at=key.expires_at,
            status=key.get_status(),
            allowed_models=key.allowed_models,
            max_budget=key.max_budget,
            spent=key.spent,
            budget_remaining=budget_remaining,
            tpm_limit=key.tpm_limit,
            rpm_limit=key.rpm_limit,
            last_used=key.last_used,
            total_requests=key.total_requests,
            metadata=key.metadata
        )


class KeyCreateResponse(BaseModel):
    """Response model for key creation (includes the actual key)"""
    key: str = Field(..., description="The actual API key (only shown once)")
    key_data: KeyResponse


class KeyUsage(BaseModel):
    """Model for tracking key usage"""
    key_id: str
    timestamp: datetime = Field(default_factory=datetime.utcnow)
    tokens: int = Field(..., gt=0)
    cost: float = Field(..., ge=0)
    model: str
    provider: str
    request_id: Optional[str] = None

    class Config:
        """Pydantic config"""
        json_encoders = {
            datetime: lambda v: v.isoformat()
        }


class KeyValidationResult(BaseModel):
    """Result of key validation"""
    valid: bool
    key_id: Optional[str] = None
    error: Optional[str] = None
    key_data: Optional[KeyResponse] = None
    rate_limit_info: Optional[Dict[str, Any]] = None
