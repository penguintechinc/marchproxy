"""
Pydantic models for billing and cost tracking
"""

from datetime import datetime
from typing import Optional, Dict, List
from pydantic import BaseModel, Field, validator
from enum import Enum


class BudgetPeriod(str, Enum):
    """Budget tracking period"""
    HOURLY = "hourly"
    DAILY = "daily"
    WEEKLY = "weekly"
    MONTHLY = "monthly"
    TOTAL = "total"


class BudgetStatus(str, Enum):
    """Budget status"""
    OK = "ok"
    WARNING = "warning"
    EXCEEDED = "exceeded"


class CostConfig(BaseModel):
    """
    Model pricing configuration
    Stores pricing per 1K tokens for input and output separately
    """
    model_name: str = Field(
        ...,
        description="Model identifier (e.g., gpt-4, claude-3-opus)"
    )
    provider: str = Field(
        ...,
        description="Provider name (openai, anthropic, etc.)"
    )
    input_price_per_1k: float = Field(
        ...,
        ge=0,
        description="Price per 1K input tokens in USD"
    )
    output_price_per_1k: float = Field(
        ...,
        ge=0,
        description="Price per 1K output tokens in USD"
    )
    currency: str = Field(
        default="USD",
        description="Currency code"
    )
    effective_date: datetime = Field(
        default_factory=datetime.utcnow,
        description="When this pricing becomes effective"
    )
    notes: Optional[str] = Field(
        None,
        description="Additional notes about pricing"
    )

    @validator('model_name', 'provider')
    def validate_not_empty(cls, v):
        """Validate that strings are not empty"""
        if not v or not v.strip():
            raise ValueError("Value cannot be empty")
        return v.strip().lower()

    @validator('currency')
    def validate_currency(cls, v):
        """Validate currency code"""
        if not v or len(v) != 3:
            raise ValueError("Currency must be 3-letter code (e.g., USD)")
        return v.upper()

    class Config:
        json_schema_extra = {
            "example": {
                "model_name": "gpt-4",
                "provider": "openai",
                "input_price_per_1k": 0.03,
                "output_price_per_1k": 0.06,
                "currency": "USD",
                "notes": "GPT-4 standard pricing"
            }
        }


class UsageRecord(BaseModel):
    """
    Individual usage record for tracking API calls
    """
    id: Optional[str] = Field(
        None,
        description="Unique record identifier"
    )
    key_id: str = Field(
        ...,
        description="Virtual key ID that made the request"
    )
    model: str = Field(
        ...,
        description="Model used for the request"
    )
    provider: str = Field(
        ...,
        description="Provider that served the request"
    )
    input_tokens: int = Field(
        ...,
        ge=0,
        description="Number of input tokens used"
    )
    output_tokens: int = Field(
        ...,
        ge=0,
        description="Number of output tokens generated"
    )
    total_tokens: int = Field(
        ...,
        ge=0,
        description="Total tokens (input + output)"
    )
    cost: float = Field(
        ...,
        ge=0,
        description="Total cost in USD"
    )
    timestamp: datetime = Field(
        default_factory=datetime.utcnow,
        description="When the request was made"
    )
    request_id: Optional[str] = Field(
        None,
        description="Original request identifier"
    )
    session_id: Optional[str] = Field(
        None,
        description="Session identifier if applicable"
    )
    user_id: Optional[str] = Field(
        None,
        description="User who owns the key"
    )
    metadata: Dict = Field(
        default_factory=dict,
        description="Additional metadata"
    )

    @validator('total_tokens')
    def validate_total_tokens(cls, v, values):
        """Validate total tokens matches input + output"""
        if 'input_tokens' in values and 'output_tokens' in values:
            expected = values['input_tokens'] + values['output_tokens']
            if v != expected:
                raise ValueError(
                    f"Total tokens ({v}) must equal "
                    f"input ({values['input_tokens']}) + "
                    f"output ({values['output_tokens']})"
                )
        return v

    class Config:
        json_encoders = {
            datetime: lambda v: v.isoformat()
        }
        json_schema_extra = {
            "example": {
                "key_id": "key_abc123",
                "model": "gpt-4",
                "provider": "openai",
                "input_tokens": 1000,
                "output_tokens": 500,
                "total_tokens": 1500,
                "cost": 0.045,
                "request_id": "req_xyz789"
            }
        }


class BudgetConfig(BaseModel):
    """
    Budget configuration for a virtual key
    """
    key_id: str = Field(
        ...,
        description="Virtual key ID this budget applies to"
    )
    max_budget: float = Field(
        ...,
        gt=0,
        description="Maximum budget allowed in USD"
    )
    alert_threshold: float = Field(
        default=0.8,
        ge=0.0,
        le=1.0,
        description="Alert when spend reaches this percentage (0.0-1.0)"
    )
    period: BudgetPeriod = Field(
        default=BudgetPeriod.MONTHLY,
        description="Budget tracking period"
    )
    enabled: bool = Field(
        default=True,
        description="Whether budget enforcement is enabled"
    )
    alert_email: Optional[str] = Field(
        None,
        description="Email to send alerts to"
    )
    created_at: datetime = Field(
        default_factory=datetime.utcnow,
        description="When budget was created"
    )
    updated_at: Optional[datetime] = Field(
        None,
        description="When budget was last updated"
    )

    @validator('alert_threshold')
    def validate_threshold(cls, v):
        """Validate alert threshold is reasonable"""
        if v <= 0 or v > 1.0:
            raise ValueError("Alert threshold must be between 0.0 and 1.0")
        return v

    @validator('alert_email')
    def validate_email(cls, v):
        """Basic email validation"""
        if v and '@' not in v:
            raise ValueError("Invalid email address")
        return v

    class Config:
        json_schema_extra = {
            "example": {
                "key_id": "key_abc123",
                "max_budget": 100.0,
                "alert_threshold": 0.8,
                "period": "monthly",
                "enabled": True,
                "alert_email": "admin@example.com"
            }
        }


class SpendSummary(BaseModel):
    """
    Aggregated spending summary
    """
    total_cost: float = Field(
        default=0.0,
        ge=0,
        description="Total cost across all filters"
    )
    total_requests: int = Field(
        default=0,
        ge=0,
        description="Total number of requests"
    )
    total_input_tokens: int = Field(
        default=0,
        ge=0,
        description="Total input tokens"
    )
    total_output_tokens: int = Field(
        default=0,
        ge=0,
        description="Total output tokens"
    )
    total_tokens: int = Field(
        default=0,
        ge=0,
        description="Total tokens (input + output)"
    )
    by_model: Dict[str, float] = Field(
        default_factory=dict,
        description="Cost breakdown by model"
    )
    by_provider: Dict[str, float] = Field(
        default_factory=dict,
        description="Cost breakdown by provider"
    )
    by_key: Dict[str, float] = Field(
        default_factory=dict,
        description="Cost breakdown by virtual key"
    )
    period_start: Optional[datetime] = Field(
        None,
        description="Start of reporting period"
    )
    period_end: Optional[datetime] = Field(
        None,
        description="End of reporting period"
    )
    period_type: Optional[BudgetPeriod] = Field(
        None,
        description="Type of period being reported"
    )

    class Config:
        json_encoders = {
            datetime: lambda v: v.isoformat()
        }
        json_schema_extra = {
            "example": {
                "total_cost": 45.50,
                "total_requests": 1000,
                "total_input_tokens": 50000,
                "total_output_tokens": 25000,
                "total_tokens": 75000,
                "by_model": {
                    "gpt-4": 30.00,
                    "gpt-3.5-turbo": 15.50
                },
                "by_provider": {
                    "openai": 45.50
                },
                "by_key": {
                    "key_abc123": 45.50
                },
                "period_type": "monthly"
            }
        }


class BudgetStatusResponse(BaseModel):
    """
    Current budget status for a key
    """
    key_id: str = Field(
        ...,
        description="Virtual key ID"
    )
    budget_config: Optional[BudgetConfig] = Field(
        None,
        description="Budget configuration"
    )
    current_spend: float = Field(
        default=0.0,
        ge=0,
        description="Current spending in this period"
    )
    budget_remaining: Optional[float] = Field(
        None,
        ge=0,
        description="Remaining budget (None if unlimited)"
    )
    budget_used_percent: Optional[float] = Field(
        None,
        ge=0,
        le=100,
        description="Percentage of budget used"
    )
    status: BudgetStatus = Field(
        default=BudgetStatus.OK,
        description="Current budget status"
    )
    alert_triggered: bool = Field(
        default=False,
        description="Whether alert threshold has been reached"
    )
    can_make_request: bool = Field(
        default=True,
        description="Whether budget allows more requests"
    )
    period_start: datetime = Field(
        ...,
        description="Start of current period"
    )
    period_end: datetime = Field(
        ...,
        description="End of current period"
    )
    estimated_requests_remaining: Optional[int] = Field(
        None,
        ge=0,
        description="Estimated requests remaining based on average cost"
    )

    class Config:
        json_encoders = {
            datetime: lambda v: v.isoformat()
        }
        json_schema_extra = {
            "example": {
                "key_id": "key_abc123",
                "current_spend": 75.00,
                "budget_remaining": 25.00,
                "budget_used_percent": 75.0,
                "status": "warning",
                "alert_triggered": False,
                "can_make_request": True,
                "period_start": "2025-12-01T00:00:00Z",
                "period_end": "2025-12-31T23:59:59Z"
            }
        }
