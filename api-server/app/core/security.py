"""
Security utilities for authentication and authorization

Handles JWT token generation/validation, password hashing, and 2FA.
"""

from datetime import datetime, timedelta
from typing import Optional

import pyotp
from jose import JWTError, jwt
from passlib.context import CryptContext

from app.core.config import settings

# Password hashing context
pwd_context = CryptContext(schemes=["bcrypt"], deprecated="auto")


def verify_password(plain_password: str, hashed_password: str) -> bool:
    """Verify a plain password against its hash"""
    return pwd_context.verify(plain_password, hashed_password)


def get_password_hash(password: str) -> str:
    """Generate password hash"""
    return pwd_context.hash(password)


def create_access_token(
    subject: str, expires_delta: Optional[timedelta] = None
) -> str:
    """
    Create JWT access token

    Args:
        subject: User ID to encode in the token
        expires_delta: Optional custom expiration time

    Returns:
        Encoded JWT token string
    """
    to_encode = {"sub": subject}
    if expires_delta:
        expire = datetime.utcnow() + expires_delta
    else:
        expire = datetime.utcnow() + timedelta(
            minutes=settings.ACCESS_TOKEN_EXPIRE_MINUTES
        )
    to_encode.update({"exp": expire, "type": "access"})
    encoded_jwt = jwt.encode(
        to_encode, settings.SECRET_KEY, algorithm=settings.ALGORITHM
    )
    return encoded_jwt


def create_refresh_token(subject: str) -> str:
    """
    Create JWT refresh token

    Args:
        subject: User ID to encode in the token

    Returns:
        Encoded JWT refresh token string
    """
    to_encode = {"sub": subject}
    expire = datetime.utcnow() + timedelta(days=settings.REFRESH_TOKEN_EXPIRE_DAYS)
    to_encode.update({"exp": expire, "type": "refresh"})
    encoded_jwt = jwt.encode(
        to_encode, settings.SECRET_KEY, algorithm=settings.ALGORITHM
    )
    return encoded_jwt


def decode_token(token: str) -> dict:
    """
    Decode and verify JWT token

    Args:
        token: JWT token string

    Returns:
        Decoded token payload

    Raises:
        JWTError: If token is invalid or expired
    """
    payload = jwt.decode(
        token, settings.SECRET_KEY, algorithms=[settings.ALGORITHM]
    )
    return payload


def generate_totp_secret() -> str:
    """Generate a new TOTP secret for 2FA"""
    return pyotp.random_base32()


def get_totp_uri(secret: str, name: str, issuer: str) -> str:
    """
    Generate TOTP provisioning URI for QR code

    Args:
        secret: TOTP secret
        name: User's username or email
        issuer: Issuer name (e.g., "MarchProxy")

    Returns:
        Provisioning URI string
    """
    totp = pyotp.TOTP(secret)
    return totp.provisioning_uri(
        name=name,
        issuer_name=issuer
    )


def verify_totp_code(secret: str, code: str) -> bool:
    """
    Verify TOTP code

    Args:
        secret: User's TOTP secret
        code: 6-digit code from authenticator app

    Returns:
        True if code is valid, False otherwise
    """
    totp = pyotp.TOTP(secret)
    return totp.verify(code, valid_window=1)  # Allow 1 step tolerance


async def get_current_user(
    token: str,
    db: "AsyncSession"
) -> "User":
    """
    Get current user from JWT token

    Args:
        token: JWT access token
        db: Database session

    Returns:
        User object

    Raises:
        HTTPException: If token is invalid or user not found
    """
    from fastapi import HTTPException, status
    from sqlalchemy import select
    from app.models.sqlalchemy.user import User

    try:
        payload = decode_token(token)
        user_id: str = payload.get("sub")
        if user_id is None:
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="Could not validate credentials",
                headers={"WWW-Authenticate": "Bearer"},
            )
    except JWTError:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Could not validate credentials",
            headers={"WWW-Authenticate": "Bearer"},
        )

    stmt = select(User).where(User.id == int(user_id))
    result = await db.execute(stmt)
    user = result.scalar_one_or_none()

    if user is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="User not found"
        )

    return user
