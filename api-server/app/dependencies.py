"""
FastAPI dependencies for auth, database, and license validation
"""

from typing import Annotated
from fastapi import Depends, HTTPException, Header, status
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.database import get_db
from app.core.security import decode_token
from app.models.sqlalchemy.user import User
from app.core.license import LicenseManager

# Security scheme
security = HTTPBearer()

# License manager instance
license_manager = LicenseManager()


async def get_current_user(
    credentials: Annotated[HTTPAuthorizationCredentials, Depends(security)],
    db: Annotated[AsyncSession, Depends(get_db)]
) -> User:
    """
    Dependency to get current authenticated user from JWT token
    """
    token = credentials.credentials
    try:
        payload = decode_token(token)
        user_id: str = payload.get("sub")
        if user_id is None:
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="Could not validate credentials",
                headers={"WWW-Authenticate": "Bearer"},
            )
    except Exception:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Could not validate credentials",
            headers={"WWW-Authenticate": "Bearer"},
        )

    stmt = select(User).where(User.id == int(user_id))
    result = await db.execute(stmt)
    user = result.scalar_one_or_none()

    if user is None or not user.is_active:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="User not found or inactive"
        )

    return user


async def require_admin(
    current_user: Annotated[User, Depends(get_current_user)]
) -> User:
    """
    Dependency to require admin privileges
    """
    if not current_user.is_admin:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Admin privileges required"
        )
    return current_user


async def validate_license_feature(
    feature: str,
    x_license_key: Annotated[str | None, Header()] = None
) -> bool:
    """
    Dependency to validate enterprise license features

    Args:
        feature: Feature name to check (e.g., "unlimited_proxies", "saml_authentication")
        x_license_key: License key from header

    Returns:
        True if feature is available, raises HTTPException otherwise
    """
    if not x_license_key:
        raise HTTPException(
            status_code=status.HTTP_402_PAYMENT_REQUIRED,
            detail=f"Enterprise feature '{feature}' requires a valid license"
        )

    validation = await license_manager.validate_license(x_license_key)
    if not validation.get("valid"):
        raise HTTPException(
            status_code=status.HTTP_402_PAYMENT_REQUIRED,
            detail="Invalid license key"
        )

    features = validation.get("features", [])
    if feature not in features:
        raise HTTPException(
            status_code=status.HTTP_402_PAYMENT_REQUIRED,
            detail=f"Your license does not include the '{feature}' feature"
        )

    return True
