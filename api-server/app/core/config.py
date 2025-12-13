"""
Application Configuration using Pydantic Settings

Centralized configuration for MarchProxy API Server.
Environment variables override defaults defined here.
"""

from typing import List, Optional
from pydantic import Field, PostgresDsn, field_validator
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """
    Application settings loaded from environment variables.

    All settings can be overridden via environment variables.
    """

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        case_sensitive=True,
        extra="ignore"
    )

    # Application
    APP_NAME: str = "MarchProxy API Server"
    APP_VERSION: str = "1.0.0"
    VERSION: str = "1.0.0"  # Alias for compatibility
    DEBUG: bool = False
    API_V1_PREFIX: str = "/api/v1"

    # Server
    HOST: str = "0.0.0.0"
    PORT: int = 8000
    WORKERS: int = 4

    # Database
    DATABASE_URL: PostgresDsn = Field(
        default="postgresql+asyncpg://marchproxy:marchproxy@postgres:5432/marchproxy",
        description="PostgreSQL connection string"
    )
    DATABASE_POOL_SIZE: int = 20
    DATABASE_MAX_OVERFLOW: int = 10

    # Redis Cache
    REDIS_URL: str = "redis://redis:6379/0"
    CACHE_TTL: int = 300  # 5 minutes

    # JWT Authentication
    SECRET_KEY: str = Field(
        default="CHANGE_ME_IN_PRODUCTION_MINIMUM_32_CHARS",
        min_length=32,
        description="Secret key for JWT signing"
    )
    ALGORITHM: str = "HS256"
    ACCESS_TOKEN_EXPIRE_MINUTES: int = 30
    REFRESH_TOKEN_EXPIRE_DAYS: int = 7

    # License Server Integration
    LICENSE_SERVER_URL: str = "https://license.penguintech.io"
    LICENSE_KEY: Optional[str] = Field(default="", description="Enterprise license key")
    PRODUCT_NAME: str = "marchproxy"
    RELEASE_MODE: bool = False  # False = development, True = production

    # xDS Control Plane
    XDS_GRPC_PORT: int = 18000
    XDS_NODE_ID: str = "marchproxy-api-server"
    XDS_SERVER_URL: str = "http://localhost:19000"  # HTTP API for xDS server

    # Monitoring
    METRICS_PORT: int = 9090
    HEALTH_CHECK_PATH: str = "/healthz"

    # CORS
    CORS_ORIGINS: List[str] = [
        "http://localhost:3000",
        "http://webui:3000"
    ]
    CORS_ALLOW_CREDENTIALS: bool = True
    CORS_ALLOW_METHODS: List[str] = ["*"]
    CORS_ALLOW_HEADERS: List[str] = ["*"]

    # Pagination
    DEFAULT_PAGE_SIZE: int = 20
    MAX_PAGE_SIZE: int = 100

    # Community vs Enterprise
    COMMUNITY_MAX_PROXIES: int = 3

    # Logging
    LOG_LEVEL: str = "INFO"
    LOG_FORMAT: str = "json"

    # Infisical Integration
    INFISICAL_URL: str = "https://app.infisical.com"
    INFISICAL_TOKEN: Optional[str] = Field(
        default=None,
        description="Infisical API token for certificate management"
    )

    # HashiCorp Vault Integration
    VAULT_URL: str = "http://vault:8200"
    VAULT_TOKEN: Optional[str] = Field(
        default=None,
        description="Vault authentication token"
    )
    VAULT_PKI_MOUNT: str = "pki"
    VAULT_NAMESPACE: Optional[str] = Field(
        default=None,
        description="Vault namespace (for Vault Enterprise)"
    )

    @field_validator("CORS_ORIGINS", mode="before")
    @classmethod
    def parse_cors_origins(cls, v: str | List[str]) -> List[str]:
        """Parse CORS origins from comma-separated string or list."""
        if isinstance(v, str):
            return [origin.strip() for origin in v.split(",")]
        return v


# Global settings instance
settings = Settings()
