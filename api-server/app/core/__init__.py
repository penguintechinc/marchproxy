"""
Core application modules

Exports commonly used core functionality for easy importing.
"""

from app.core.config import settings
from app.core.database import get_db, Base, engine, init_db, close_db
from app.core.security import (
    verify_password,
    get_password_hash,
    create_access_token,
    create_refresh_token,
    decode_token,
    generate_totp_secret,
    verify_totp_code,
    get_totp_uri,
)

__all__ = [
    # Config
    "settings",
    # Database
    "get_db",
    "Base",
    "engine",
    "init_db",
    "close_db",
    # Security
    "verify_password",
    "get_password_hash",
    "create_access_token",
    "create_refresh_token",
    "decode_token",
    "generate_totp_secret",
    "verify_totp_code",
    "get_totp_uri",
]
