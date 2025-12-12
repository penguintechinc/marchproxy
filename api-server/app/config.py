"""
Application Configuration using Pydantic Settings

Environment variables override defaults defined here.
"""

from functools import lru_cache
from typing import Optional

from pydantic import Field, PostgresDsn, validator
from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    """Application settings loaded from environment variables"""

    # Application
    APP_NAME: str = "MarchProxy API Server"
    APP_VERSION: str = "1.0.0"
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
        default="CHANGE_ME_IN_PRODUCTION",
        description="Secret key for JWT signing"
    )
    ALGORITHM: str = "HS256"
    ACCESS_TOKEN_EXPIRE_MINUTES: int = 30
    REFRESH_TOKEN_EXPIRE_DAYS: int = 7

    # License Server Integration
    LICENSE_SERVER_URL: str = "https://license.penguintech.io"
    LICENSE_KEY: Optional[str] = None
    PRODUCT_NAME: str = "marchproxy"
    RELEASE_MODE: bool = False  # False = development, True = production

    # xDS Control Plane
    XDS_GRPC_PORT: int = 18000
    XDS_NODE_ID: str = "marchproxy-xds"
    XDS_SERVER_URL: str = "http://localhost:19000"  # HTTP API for xDS server

    # Monitoring
    METRICS_PORT: int = 9090
    HEALTH_CHECK_PATH: str = "/healthz"

    # CORS
    CORS_ORIGINS: list[str] = ["http://localhost:3000", "http://webui:3000"]
    CORS_ALLOW_CREDENTIALS: bool = True
    CORS_ALLOW_METHODS: list[str] = ["*"]
    CORS_ALLOW_HEADERS: list[str] = ["*"]

    # Pagination
    DEFAULT_PAGE_SIZE: int = 20
    MAX_PAGE_SIZE: int = 100

    # Community vs Enterprise
    COMMUNITY_MAX_PROXIES: int = 3

    class Config:
        env_file = ".env"
        case_sensitive = True


@lru_cache()
def get_settings() -> Settings:
    """
    Get cached settings instance.

    Using lru_cache ensures settings are loaded once and reused.
    """
    return Settings()
