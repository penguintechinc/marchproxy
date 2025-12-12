"""
Pydantic schemas for traffic shaping and QoS
"""

from datetime import datetime
from enum import Enum
from typing import Optional

from pydantic import BaseModel, Field, validator


class PriorityLevel(str, Enum):
    """Priority queue levels"""
    P0 = "P0"  # <1ms - Interactive
    P1 = "P1"  # <10ms - Real-time
    P2 = "P2"  # <100ms - Bulk
    P3 = "P3"  # Best effort


class DSCPMarking(str, Enum):
    """DSCP marking values"""
    EF = "EF"      # Expedited Forwarding
    AF41 = "AF41"  # Assured Forwarding
    AF31 = "AF31"
    AF21 = "AF21"
    AF11 = "AF11"
    BE = "BE"      # Best Effort


class BandwidthLimit(BaseModel):
    """Bandwidth limit configuration"""
    ingress_mbps: Optional[int] = Field(
        None, ge=1, le=100000,
        description="Ingress bandwidth limit in Mbps"
    )
    egress_mbps: Optional[int] = Field(
        None, ge=1, le=100000,
        description="Egress bandwidth limit in Mbps"
    )
    burst_size_kb: Optional[int] = Field(
        1024, ge=1, le=10240,
        description="Burst size in KB for token bucket"
    )


class PriorityQueueConfig(BaseModel):
    """Priority queue configuration"""
    priority: PriorityLevel = Field(
        ..., description="Priority level (P0-P3)"
    )
    weight: int = Field(
        1, ge=1, le=100,
        description="Queue weight for weighted fair queuing"
    )
    max_latency_ms: Optional[int] = Field(
        None, ge=1, le=1000,
        description="Maximum latency SLA in milliseconds"
    )
    dscp_marking: DSCPMarking = Field(
        DSCPMarking.BE, description="DSCP marking for packets"
    )


class QoSPolicyCreate(BaseModel):
    """Schema for creating a QoS policy"""
    name: str = Field(..., min_length=1, max_length=100)
    description: Optional[str] = Field(None, max_length=500)
    service_id: int = Field(..., description="Service to apply policy to")
    cluster_id: int = Field(..., description="Cluster ID")

    # Bandwidth limits
    bandwidth: BandwidthLimit = Field(
        default_factory=BandwidthLimit,
        description="Bandwidth limits"
    )

    # Priority configuration
    priority_config: PriorityQueueConfig = Field(
        ..., description="Priority queue configuration"
    )

    # Enable/disable
    enabled: bool = Field(True, description="Enable this policy")

    @validator('name')
    def validate_name(cls, v):
        """Validate policy name"""
        if not v.strip():
            raise ValueError("Policy name cannot be empty")
        return v.strip()


class QoSPolicyUpdate(BaseModel):
    """Schema for updating a QoS policy"""
    name: Optional[str] = Field(None, min_length=1, max_length=100)
    description: Optional[str] = Field(None, max_length=500)
    bandwidth: Optional[BandwidthLimit] = None
    priority_config: Optional[PriorityQueueConfig] = None
    enabled: Optional[bool] = None


class QoSPolicyResponse(BaseModel):
    """Schema for QoS policy response"""
    id: int
    name: str
    description: Optional[str]
    service_id: int
    cluster_id: int
    bandwidth: BandwidthLimit
    priority_config: PriorityQueueConfig
    enabled: bool
    created_at: datetime
    updated_at: datetime

    class Config:
        from_attributes = True
