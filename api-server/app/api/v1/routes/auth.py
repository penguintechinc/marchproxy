"""
Authentication API routes

Handles user authentication, registration, 2FA, and token management.
"""

import logging
from datetime import datetime
from typing import Annotated

from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.database import get_db
from app.core.security import (
    create_access_token,
    create_refresh_token,
    get_password_hash,
    verify_password,
    generate_totp_secret,
    verify_totp_code,
    get_totp_uri,
)
from app.dependencies import get_current_user
from app.models.sqlalchemy.user import User
from app.schemas.auth import (
    LoginRequest,
    LoginResponse,
    TokenResponse,
    RefreshTokenRequest,
    Enable2FAResponse,
    Verify2FARequest,
    ChangePasswordRequest,
)

router = APIRouter(prefix="/auth", tags=["authentication"])
logger = logging.getLogger(__name__)


@router.post("/login", response_model=LoginResponse)
async def login(
    credentials: LoginRequest,
    db: Annotated[AsyncSession, Depends(get_db)]
):
    """
    Authenticate user and return JWT tokens.

    Supports 2FA if enabled for the user.
    """
    # Find user by username or email
    stmt = select(User).where(
        (User.username == credentials.username) | (User.email == credentials.username)
    )
    result = await db.execute(stmt)
    user = result.scalar_one_or_none()

    if not user or not verify_password(credentials.password, user.password_hash):
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Incorrect username or password",
            headers={"WWW-Authenticate": "Bearer"},
        )

    if not user.is_active:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="User account is deactivated"
        )

    # Check 2FA if enabled
    if user.totp_enabled:
        if not credentials.totp_code:
            return LoginResponse(
                user_id=user.id,
                username=user.username,
                email=user.email,
                is_admin=user.is_admin,
                requires_2fa=True
            )

        if not verify_totp_code(user.totp_secret, credentials.totp_code):
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="Invalid 2FA code"
            )

    # Update last login
    user.last_login = datetime.utcnow()
    await db.commit()

    # Generate tokens
    access_token = create_access_token(subject=str(user.id))
    refresh_token = create_refresh_token(subject=str(user.id))

    logger.info(f"User {user.username} logged in successfully")

    return LoginResponse(
        user_id=user.id,
        username=user.username,
        email=user.email,
        is_admin=user.is_admin,
        requires_2fa=False,
        access_token=access_token,
        refresh_token=refresh_token,
        expires_in=3600  # 1 hour
    )


@router.post("/register", response_model=LoginResponse, status_code=status.HTTP_201_CREATED)
async def register(
    credentials: LoginRequest,
    db: Annotated[AsyncSession, Depends(get_db)]
):
    """
    Register a new user account.

    First user becomes admin automatically.
    """
    # Check if username or email already exists
    stmt = select(User).where(
        (User.username == credentials.username) | (User.email == credentials.username)
    )
    result = await db.execute(stmt)
    existing_user = result.scalar_one_or_none()

    if existing_user:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Username or email already registered"
        )

    # Check if this is the first user
    count_stmt = select(User)
    count_result = await db.execute(count_stmt)
    is_first_user = len(count_result.scalars().all()) == 0

    # Create new user
    new_user = User(
        email=credentials.username,  # For now, use username as email
        username=credentials.username,
        password_hash=get_password_hash(credentials.password),
        is_active=True,
        is_admin=is_first_user,  # First user is admin
        is_verified=is_first_user,  # First user is auto-verified
        totp_enabled=False,
        created_at=datetime.utcnow()
    )

    db.add(new_user)
    await db.commit()
    await db.refresh(new_user)

    # Generate tokens
    access_token = create_access_token(subject=str(new_user.id))
    refresh_token = create_refresh_token(subject=str(new_user.id))

    logger.info(f"New user registered: {new_user.username}")

    return LoginResponse(
        user_id=new_user.id,
        username=new_user.username,
        email=new_user.email,
        is_admin=new_user.is_admin,
        requires_2fa=False,
        access_token=access_token,
        refresh_token=refresh_token,
        expires_in=3600
    )


@router.post("/refresh", response_model=TokenResponse)
async def refresh_token(
    request: RefreshTokenRequest,
    db: Annotated[AsyncSession, Depends(get_db)]
):
    """
    Refresh access token using refresh token.
    """
    from jose import jwt, JWTError
    from app.core.config import settings

    try:
        # Decode and validate refresh token
        payload = jwt.decode(
            request.refresh_token,
            settings.SECRET_KEY,
            algorithms=[settings.ALGORITHM]
        )

        # Check token type
        token_type = payload.get("type")
        if token_type != "refresh":
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="Invalid token type"
            )

        # Extract user ID
        user_id = payload.get("sub")
        if not user_id:
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="Invalid token payload"
            )

        # Verify user still exists and is active
        stmt = select(User).where(User.id == int(user_id))
        result = await db.execute(stmt)
        user = result.scalar_one_or_none()

        if not user:
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="User not found"
            )

        if not user.is_active:
            raise HTTPException(
                status_code=status.HTTP_403_FORBIDDEN,
                detail="User account is deactivated"
            )

        # Generate new tokens
        new_access_token = create_access_token(subject=str(user.id))
        new_refresh_token = create_refresh_token(subject=str(user.id))

        logger.info(f"Tokens refreshed for user {user.username}")

        return TokenResponse(
            access_token=new_access_token,
            refresh_token=new_refresh_token,
            token_type="bearer",
            expires_in=3600
        )

    except JWTError as e:
        logger.warning(f"Invalid refresh token: {e}")
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid or expired refresh token"
        )


@router.post("/2fa/enable", response_model=Enable2FAResponse)
async def enable_2fa(
    current_user: Annotated[User, Depends(get_current_user)],
    db: Annotated[AsyncSession, Depends(get_db)]
):
    """
    Enable 2FA for the current user.

    Returns TOTP secret and QR code URI.
    """
    if current_user.totp_enabled:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="2FA is already enabled"
        )

    # Generate new TOTP secret
    secret = generate_totp_secret()
    totp_uri = get_totp_uri(
        secret=secret,
        name=current_user.email,
        issuer="MarchProxy"
    )

    # Generate backup codes
    backup_codes = [generate_totp_secret()[:8] for _ in range(10)]

    # Update user (but don't enable yet - wait for verification)
    current_user.totp_secret = secret
    await db.commit()

    logger.info(f"2FA setup initiated for user {current_user.username}")

    return Enable2FAResponse(
        secret=secret,
        qr_code_uri=totp_uri,
        backup_codes=backup_codes
    )


@router.post("/2fa/verify")
async def verify_2fa(
    request: Verify2FARequest,
    current_user: Annotated[User, Depends(get_current_user)],
    db: Annotated[AsyncSession, Depends(get_db)]
):
    """
    Verify 2FA code and enable 2FA for the user.
    """
    if not current_user.totp_secret:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="2FA setup not initiated"
        )

    if not verify_totp_code(current_user.totp_secret, request.totp_code):
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid 2FA code"
        )

    # Enable 2FA
    current_user.totp_enabled = True
    await db.commit()

    logger.info(f"2FA enabled for user {current_user.username}")

    return {"message": "2FA enabled successfully"}


@router.post("/2fa/disable")
async def disable_2fa(
    request: Verify2FARequest,
    current_user: Annotated[User, Depends(get_current_user)],
    db: Annotated[AsyncSession, Depends(get_db)]
):
    """
    Disable 2FA for the current user.

    Requires valid 2FA code to confirm.
    """
    if not current_user.totp_enabled:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="2FA is not enabled"
        )

    if not verify_totp_code(current_user.totp_secret, request.totp_code):
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid 2FA code"
        )

    # Disable 2FA
    current_user.totp_enabled = False
    current_user.totp_secret = None
    await db.commit()

    logger.info(f"2FA disabled for user {current_user.username}")

    return {"message": "2FA disabled successfully"}


@router.post("/change-password")
async def change_password(
    request: ChangePasswordRequest,
    current_user: Annotated[User, Depends(get_current_user)],
    db: Annotated[AsyncSession, Depends(get_db)]
):
    """
    Change user password.

    Requires current password for verification.
    """
    # Verify current password
    if not verify_password(request.current_password, current_user.password_hash):
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Current password is incorrect"
        )

    # Validate new password
    if request.new_password != request.confirm_password:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="New passwords do not match"
        )

    # Update password
    current_user.password_hash = get_password_hash(request.new_password)
    await db.commit()

    logger.info(f"Password changed for user {current_user.username}")

    return {"message": "Password changed successfully"}


@router.get("/me")
async def get_current_user_info(
    current_user: Annotated[User, Depends(get_current_user)]
):
    """
    Get current user information.
    """
    return {
        "id": current_user.id,
        "username": current_user.username,
        "email": current_user.email,
        "first_name": current_user.first_name,
        "last_name": current_user.last_name,
        "is_active": current_user.is_active,
        "is_admin": current_user.is_admin,
        "is_verified": current_user.is_verified,
        "totp_enabled": current_user.totp_enabled,
        "created_at": current_user.created_at.isoformat() if current_user.created_at else None,
        "last_login": current_user.last_login.isoformat() if current_user.last_login else None,
    }


@router.post("/logout")
async def logout(
    current_user: Annotated[User, Depends(get_current_user)]
):
    """
    Logout current user.

    In a stateless JWT system, this is primarily client-side.
    Server-side token revocation would require a blacklist.
    """
    logger.info(f"User {current_user.username} logged out")
    return {"message": "Logged out successfully"}
