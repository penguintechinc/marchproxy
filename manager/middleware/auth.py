"""
Authentication middleware for MarchProxy Manager

Provides JWT token validation, user authentication, and authorization
decorators for protecting API endpoints.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import logging
from functools import wraps
from typing import Any, Callable, Optional

from quart import current_app, g, request

logger = logging.getLogger(__name__)


def get_current_user() -> Optional[dict]:
    """
    Get the current authenticated user from request context.

    Returns:
        dict: User payload with user_id, username, email, is_admin, etc.
        None: If no user is authenticated
    """
    return getattr(g, "user", None)


def is_admin() -> bool:
    """
    Check if the current user is an administrator.

    Returns:
        bool: True if current user is admin, False otherwise
    """
    user = get_current_user()
    return user.get("is_admin", False) if user else False


def _extract_token_from_header() -> Optional[str]:
    """
    Extract JWT token from Authorization header.

    Expected format: Authorization: Bearer <token>

    Returns:
        str: JWT token if present and valid format
        None: If token is missing or invalid format
    """
    auth_header = request.headers.get("Authorization", "")

    if not auth_header:
        return None

    parts = auth_header.split()

    if len(parts) != 2 or parts[0].lower() != "bearer":
        return None

    return parts[1]


def _validate_token(token: str) -> Optional[dict]:
    """
    Validate JWT token using current_app's JWT manager.

    Args:
        token: JWT token string to validate

    Returns:
        dict: Decoded token payload if valid
        None: If token is invalid or expired
    """
    try:
        jwt_manager = getattr(current_app, "jwt_manager", None)
        if not jwt_manager:
            logger.error("JWT manager not configured in current_app")
            return None

        payload = jwt_manager.decode_token(token)
        return payload

    except Exception as e:
        logger.debug(f"Token validation failed: {e}")
        return None


def require_auth(admin_required: bool = False, license_feature: Optional[str] = None) -> Callable:
    """
    Decorator to protect routes requiring authentication.

    This decorator validates JWT tokens from the Authorization header,
    stores the user payload in g.user for handler access, and optionally
    enforces admin-only or license-gated access.

    Args:
        admin_required: If True, only users with is_admin=True can access
        license_feature: If set, check if this feature is licensed
                        (optional, can be placeholder for future use)

    Returns:
        Callable: Decorated function that validates auth before execution

    Raises:
        Returns JSON error responses:
        - 401: Missing or invalid authorization header
        - 401: Invalid or expired token
        - 403: Admin access required but user is not admin
        - 403: Required license feature not available

    Examples:
        @require_auth()
        def get_user_profile():
            user = get_current_user()
            return {"username": user['username']}

        @require_auth(admin_required=True)
        def delete_user(user_id):
            # Only admins can access this endpoint
            return {"status": "deleted"}

        @require_auth(license_feature="advanced_blocking")
        def get_advanced_features():
            # Only accessible if license includes this feature
            return {"features": [...]}
    """

    def decorator(handler: Callable) -> Callable:
        # Quart routes are all async
        @wraps(handler)
        async def async_decorated(*args: Any, **kwargs: Any) -> Any:
            return await _authenticate_and_authorize_async(
                handler, args, kwargs, admin_required, license_feature
            )

        return async_decorated

    return decorator


async def _authenticate_and_authorize_async(
    handler: Callable,
    args: tuple,
    kwargs: dict,
    admin_required: bool,
    license_feature: Optional[str],
) -> Any:
    """
    Perform authentication and authorization checks (async).

    Validates JWT token, checks admin requirements, and verifies
    license features if configured.

    Args:
        handler: The async route handler function to call
        args: Positional arguments for handler
        kwargs: Keyword arguments for handler
        admin_required: Whether admin access is required
        license_feature: License feature to check (if any)

    Returns:
        dict: Handler response on success, or error response on failure
    """
    # Extract and validate token
    token = _extract_token_from_header()

    if not token:
        logger.debug("Missing authorization header in request")
        return ({"error": "Missing authorization header"}, 401)

    # Decode and validate token
    payload = _validate_token(token)

    if payload is None:
        logger.debug("Invalid or expired token")
        return ({"error": "Invalid or expired token"}, 401)

    # Store user in request context
    g.user = payload

    # Check admin requirement
    if admin_required and not payload.get("is_admin", False):
        logger.warning(f"Non-admin user {payload.get('user_id')} attempted admin access")
        return ({"error": "Admin access required"}, 403)

    # Check license feature (placeholder for future implementation)
    if license_feature:
        license_valid = _check_license_feature(license_feature, payload)
        if not license_valid:
            logger.warning(
                f"License feature '{license_feature}' not available "
                f"for user {payload.get('user_id')}"
            )
            return ({"error": f"Feature '{license_feature}' not licensed"}, 403)

    # Call the actual handler (await since it's async in Quart)
    try:
        return await handler(*args, **kwargs)
    except Exception as e:
        logger.error(f"Error in authenticated handler: {e}", exc_info=True)
        return ({"error": "Internal server error"}, 500)


def _check_license_feature(feature: str, user_payload: dict) -> bool:
    """
    Check if a license feature is available for the user.

    This is a placeholder implementation that can be expanded to check
    actual license server or cache for feature availability.

    Args:
        feature: Feature identifier to check
        user_payload: JWT payload containing user and license info

    Returns:
        bool: True if feature is available, False otherwise
    """
    try:
        # Placeholder: Allow all features for now
        # Future implementation can check:
        # - License server
        # - License cache in database
        # - Feature flags
        # - User tier/subscription level

        license_manager = getattr(current_app, "license_manager", None)

        if license_manager:
            # TODO: Implement actual license checking
            # is_available = license_manager.check_feature(
            #     feature,
            #     user_payload.get('user_id')
            # )
            # return is_available
            pass

        # Default: allow all features (development mode)
        return True

    except Exception as e:
        logger.error(f"Error checking license feature '{feature}': {e}")
        return False


# Optional: Helper context manager for manual auth checking
class AuthContext:
    """
    Manual authentication context for validating auth without decorator.

    Useful when you need more control over auth flow or want to handle
    authentication errors differently.

    Examples:
        with AuthContext() as auth:
            if not auth.is_authenticated():
                return {"error": "Not authenticated"}, 401
            user = auth.get_user()
            if not auth.is_admin():
                return {"error": "Admin required"}, 403
    """

    def __init__(self):
        self.user = None
        self.valid = False
        self._validate()

    def _validate(self):
        """Validate token and extract user payload"""
        token = _extract_token_from_header()
        if token:
            payload = _validate_token(token)
            if payload:
                self.user = payload
                self.valid = True
                g.user = self.user

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        pass

    def is_authenticated(self) -> bool:
        """Check if user is authenticated"""
        return self.valid and self.user is not None

    def get_user(self) -> Optional[dict]:
        """Get current user payload"""
        return self.user

    def is_admin(self) -> bool:
        """Check if current user is admin"""
        return self.user.get("is_admin", False) if self.user else False

    def has_feature(self, feature: str) -> bool:
        """Check if user has access to a feature"""
        if not self.user:
            return False
        return _check_license_feature(feature, self.user)
