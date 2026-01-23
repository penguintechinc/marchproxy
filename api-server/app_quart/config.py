"""Application configuration from environment variables."""
import os
from dataclasses import dataclass


@dataclass(slots=True)
class Config:
    """Application configuration loaded from environment variables."""

    # Database
    DATABASE_URL: str = os.getenv(
        'DATABASE_URL',
        'postgresql+asyncpg://marchproxy:marchproxy123@localhost:5432/marchproxy'
    )
    REDIS_URL: str = os.getenv('REDIS_URL', 'redis://localhost:6379/0')

    # Security
    SECRET_KEY: str = os.getenv('SECRET_KEY', 'change-this-in-production')
    SECURITY_PASSWORD_SALT: str = os.getenv(
        'SECURITY_PASSWORD_SALT',
        'change-this-salt'
    )

    # JWT
    JWT_ACCESS_TOKEN_EXPIRES: int = int(
        os.getenv('JWT_ACCESS_TOKEN_EXPIRES', '3600')
    )

    # Kong Admin API (internal network)
    KONG_ADMIN_URL: str = os.getenv('KONG_ADMIN_URL', 'http://kong:8001')

    # Application
    DEBUG: bool = os.getenv('DEBUG', 'false').lower() == 'true'
    LOG_LEVEL: str = os.getenv('LOG_LEVEL', 'INFO')

    # CORS
    CORS_ORIGINS: list = os.getenv(
        'CORS_ORIGINS',
        'http://localhost:3000'
    ).split(',')


config = Config()
