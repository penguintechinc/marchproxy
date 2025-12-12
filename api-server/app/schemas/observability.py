"""
Pydantic schemas for observability and distributed tracing
"""

from datetime import datetime
from enum import Enum
from typing import Optional

from pydantic import BaseModel, Field, validator


class TracingBackend(str, Enum):
    """Tracing backend options"""
    JAEGER = "jaeger"
    ZIPKIN = "zipkin"
    OTLP = "otlp"  # OpenTelemetry Protocol


class SamplingStrategy(str, Enum):
    """Sampling strategy for traces"""
    ALWAYS = "always"           # Sample all requests (100%)
    NEVER = "never"             # No sampling (0%)
    PROBABILISTIC = "probabilistic"  # Random percentage
    RATE_LIMIT = "rate_limit"   # Max traces per second
    ERROR_ONLY = "error_only"   # Only sample errors
    ADAPTIVE = "adaptive"       # Dynamic based on load


class SpanExporter(str, Enum):
    """Span exporter protocol"""
    GRPC = "grpc"
    HTTP = "http"
    THRIFT = "thrift"


class TracingConfigCreate(BaseModel):
    """Schema for creating tracing configuration"""
    name: str = Field(..., min_length=1, max_length=100)
    description: Optional[str] = Field(None, max_length=500)
    cluster_id: int = Field(..., description="Cluster ID")

    # Backend configuration
    backend: TracingBackend = Field(
        TracingBackend.JAEGER,
        description="Tracing backend type"
    )
    endpoint: str = Field(
        ..., min_length=1, max_length=255,
        description="Backend endpoint (host:port)"
    )
    exporter: SpanExporter = Field(
        SpanExporter.GRPC,
        description="Span exporter protocol"
    )

    # Sampling configuration
    sampling_strategy: SamplingStrategy = Field(
        SamplingStrategy.PROBABILISTIC,
        description="Sampling strategy"
    )
    sampling_rate: float = Field(
        0.1, ge=0.0, le=1.0,
        description="Sampling rate for probabilistic (0.0-1.0)"
    )
    max_traces_per_second: Optional[int] = Field(
        None, ge=1, le=100000,
        description="Max traces/sec for rate_limit strategy"
    )

    # Advanced options
    include_request_headers: bool = Field(
        False, description="Include request headers in spans"
    )
    include_response_headers: bool = Field(
        False, description="Include response headers in spans"
    )
    include_request_body: bool = Field(
        False, description="Include request body in spans (privacy concern)"
    )
    include_response_body: bool = Field(
        False, description="Include response body in spans (privacy concern)"
    )
    max_attribute_length: int = Field(
        512, ge=64, le=4096,
        description="Max length for span attributes"
    )

    # Service tagging
    service_name: str = Field(
        "marchproxy", min_length=1, max_length=100,
        description="Service name for traces"
    )
    custom_tags: Optional[dict[str, str]] = Field(
        None, description="Custom tags for all spans"
    )

    enabled: bool = Field(True, description="Enable tracing")

    @validator('name')
    def validate_name(cls, v):
        """Validate config name"""
        if not v.strip():
            raise ValueError("Configuration name cannot be empty")
        return v.strip()

    @validator('endpoint')
    def validate_endpoint(cls, v):
        """Validate endpoint format"""
        if not v.strip():
            raise ValueError("Endpoint cannot be empty")
        # Basic validation for host:port format
        if ':' not in v:
            raise ValueError("Endpoint must be in format 'host:port'")
        return v.strip()

    @validator('sampling_rate')
    def validate_sampling_rate(cls, v, values):
        """Validate sampling rate for probabilistic strategy"""
        if 'sampling_strategy' in values:
            if values['sampling_strategy'] == SamplingStrategy.PROBABILISTIC:
                if v < 0.0 or v > 1.0:
                    raise ValueError("Sampling rate must be between 0.0 and 1.0")
        return v


class TracingConfigUpdate(BaseModel):
    """Schema for updating tracing configuration"""
    name: Optional[str] = Field(None, min_length=1, max_length=100)
    description: Optional[str] = Field(None, max_length=500)
    backend: Optional[TracingBackend] = None
    endpoint: Optional[str] = Field(None, min_length=1, max_length=255)
    exporter: Optional[SpanExporter] = None
    sampling_strategy: Optional[SamplingStrategy] = None
    sampling_rate: Optional[float] = Field(None, ge=0.0, le=1.0)
    max_traces_per_second: Optional[int] = Field(None, ge=1, le=100000)
    include_request_headers: Optional[bool] = None
    include_response_headers: Optional[bool] = None
    include_request_body: Optional[bool] = None
    include_response_body: Optional[bool] = None
    max_attribute_length: Optional[int] = Field(None, ge=64, le=4096)
    service_name: Optional[str] = Field(None, min_length=1, max_length=100)
    custom_tags: Optional[dict[str, str]] = None
    enabled: Optional[bool] = None


class TracingStats(BaseModel):
    """Runtime tracing statistics"""
    total_spans: int = 0
    sampled_spans: int = 0
    dropped_spans: int = 0
    error_spans: int = 0
    avg_span_duration_ms: Optional[float] = None
    last_export: Optional[datetime] = None


class TracingConfigResponse(BaseModel):
    """Schema for tracing configuration response"""
    id: int
    name: str
    description: Optional[str]
    cluster_id: int
    backend: TracingBackend
    endpoint: str
    exporter: SpanExporter
    sampling_strategy: SamplingStrategy
    sampling_rate: float
    max_traces_per_second: Optional[int]
    include_request_headers: bool
    include_response_headers: bool
    include_request_body: bool
    include_response_body: bool
    max_attribute_length: int
    service_name: str
    custom_tags: Optional[dict[str, str]]
    enabled: bool
    created_at: datetime
    updated_at: datetime
    # Runtime statistics
    stats: Optional[TracingStats] = None

    class Config:
        from_attributes = True
