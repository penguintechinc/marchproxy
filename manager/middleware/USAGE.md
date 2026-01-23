# Authentication Middleware Usage Guide

## Overview

The authentication middleware provides JWT token validation, user authentication, and authorization decorators for protecting API endpoints in MarchProxy Manager.

## Features

- **JWT Token Validation**: Extract and validate tokens from `Authorization: Bearer <token>` headers
- **User Context**: Store authenticated user info in `g.user` for handler access
- **Admin-Only Routes**: Enforce admin-only access with `admin_required=True`
- **License Gating**: Optional feature-based access control via `license_feature` parameter
- **Async Support**: Full support for both sync and async route handlers
- **Error Handling**: Standardized JSON error responses with appropriate HTTP status codes

## Installation

The middleware is already integrated into the manager application. Simply import from the middleware package:

```python
from middleware.auth import require_auth, get_current_user, is_admin
```

## Basic Usage

### Protecting a Route with Authentication

```python
from py4web import request, response
from middleware.auth import require_auth, get_current_user

@require_auth()
def get_user_profile():
    """Get current user's profile (requires authentication)"""
    user = get_current_user()
    return {
        "user_id": user['user_id'],
        "username": user['username'],
        "email": user['email'],
        "is_admin": user['is_admin']
    }
```

### Admin-Only Routes

```python
@require_auth(admin_required=True)
def create_user():
    """Create new user (admin-only)"""
    data = request.json
    # Admin-only logic here
    return {"status": "user_created"}

@require_auth(admin_required=True)
def delete_user(user_id):
    """Delete a user (admin-only)"""
    # Admin-only deletion logic
    return {"status": "user_deleted"}
```

### License-Gated Features

```python
@require_auth(license_feature="advanced_blocking")
def get_advanced_blocking_config():
    """Get advanced blocking rules (requires license)"""
    return {"blocking_rules": [...]}

@require_auth(license_feature="threat_intelligence")
def get_threat_intelligence():
    """Access threat intelligence feed (licensed feature)"""
    return {"threats": [...]}
```

### Async Handlers

```python
@require_auth(admin_required=True)
async def async_create_cluster():
    """Create cluster with async handler"""
    data = request.json
    # Async operations here
    return {"cluster_id": 123}
```

## Helper Functions

### get_current_user()

Returns the authenticated user payload.

```python
from middleware.auth import get_current_user

user = get_current_user()
if user:
    print(f"User: {user['username']}")
    print(f"Admin: {user['is_admin']}")
else:
    print("No user authenticated")
```

**Returns:**
- `dict`: User payload with keys: `user_id`, `username`, `email`, `is_admin`, `type`, `exp`, `iat`, `iss`
- `None`: If no user is authenticated

### is_admin()

Check if current user is an administrator.

```python
from middleware.auth import is_admin

if is_admin():
    print("User is admin")
else:
    print("User is not admin")
```

**Returns:**
- `bool`: True if current user is admin, False otherwise

## Manual Authentication Context

For more control over authentication flow, use the `AuthContext` context manager:

```python
from middleware.auth import AuthContext

def some_handler():
    with AuthContext() as auth:
        if not auth.is_authenticated():
            return {"error": "Not authenticated"}, 401

        user = auth.get_user()
        if not auth.is_admin():
            return {"error": "Admin required"}, 403

        if not auth.has_feature("advanced_blocking"):
            return {"error": "Feature not licensed"}, 403

        # Process request with authenticated user
        return {"data": "..."}
```

**AuthContext Methods:**
- `is_authenticated()`: Check if user is authenticated
- `get_user()`: Get current user payload
- `is_admin()`: Check if user is admin
- `has_feature(feature)`: Check if user has access to feature

## Error Responses

The middleware returns standardized JSON error responses:

### Missing Authorization Header
```json
{
  "error": "Missing authorization header"
}
```
**Status Code:** 401 Unauthorized

### Invalid or Expired Token
```json
{
  "error": "Invalid or expired token"
}
```
**Status Code:** 401 Unauthorized

### Admin Access Required
```json
{
  "error": "Admin access required"
}
```
**Status Code:** 403 Forbidden

### License Feature Not Available
```json
{
  "error": "Feature 'advanced_blocking' not licensed"
}
```
**Status Code:** 403 Forbidden

## JWT Token Format

Tokens are created by the JWT manager with the following payload structure:

```python
{
    "user_id": 1,
    "username": "john_doe",
    "email": "john@example.com",
    "is_admin": False,
    "type": "access",  # or "refresh"
    "iat": 1704888000,  # Issued at timestamp
    "exp": 1704974400,  # Expiration timestamp
    "iss": "marchproxy"  # Issuer
}
```

## Integration with py4web Routes

The middleware works seamlessly with py4web route registration. Update your route handlers to use the decorator:

```python
from py4web import application, request, response
from middleware.auth import require_auth, get_current_user, is_admin

@application.route('/api/users/profile', methods=['GET'])
@require_auth()
def get_profile():
    """Get current user's profile"""
    user = get_current_user()
    return {"user": user}

@application.route('/api/users', methods=['POST'])
@require_auth(admin_required=True)
def create_user():
    """Create new user (admin only)"""
    data = request.json
    # Create user logic
    return {"status": "created"}

@application.route('/api/advanced-features', methods=['GET'])
@require_auth(license_feature="advanced_blocking")
def get_advanced_features():
    """Get advanced features (licensed)"""
    return {"features": [...]}
```

## Best Practices

1. **Always use `@require_auth()` on protected routes**: Don't rely on manual checks
2. **Specify `admin_required=True` for admin endpoints**: Be explicit about requirements
3. **Use `license_feature` for premium features**: Enable feature-based access control
4. **Access user context with `get_current_user()`**: Never access raw request data for auth
5. **Handle errors gracefully**: Check for auth failures and return appropriate responses
6. **Log authentication events**: Monitor failed auth attempts for security
7. **Keep JWT secret secure**: Use strong, random secrets in production
8. **Rotate tokens regularly**: Implement token refresh mechanisms
9. **Use HTTPS in production**: Prevent token interception
10. **Validate token signature**: Always validate using JWTManager, never skip validation

## Configuration

### JWT Secret

Set the JWT secret via environment variable:

```bash
export JWT_SECRET="your-super-secret-jwt-key-change-in-production"
```

### JWT TTL

Configure token time-to-live (default: 24 hours):

```python
from models.auth import JWTManager

jwt_manager = JWTManager(
    secret_key="your-secret",
    algorithm="HS256",
    ttl_hours=24  # Token expires in 24 hours
)
```

### License Manager

The middleware can check license features if a license manager is configured:

```python
from models.license import LicenseManager

license_manager = LicenseManager(db, license_key)
current_app.license_manager = license_manager
```

## Troubleshooting

### Token Not Being Extracted
- Ensure Authorization header format: `Bearer <token>`
- Check for extra spaces or case sensitivity issues
- Verify header is being sent by client

### Token Validation Failing
- Verify JWT secret matches between token creation and validation
- Check token expiration time
- Confirm token wasn't modified or corrupted

### Admin Check Always Failing
- Verify token contains `is_admin` field set to `True`
- Check JWT manager is properly initialized
- Ensure user was created with admin flag

### License Feature Check Not Working
- Verify license manager is configured in `current_app`
- Implement actual license checking in `_check_license_feature()`
- Check license database/cache for feature availability

## Examples

See `/home/penguin/code/MarchProxy/manager/api/` for complete examples of protected endpoints.

Example protected endpoint in `api/auth.py`:
```python
@require_auth()
def profile():
    """Get/update user profile"""
    user = get_current_user()
    # Profile logic
```

Example admin-only endpoint:
```python
@require_auth(admin_required=True)
def manage_users():
    """Manage users (admin only)"""
    # Admin logic
```

## File Locations

- **Main Implementation**: `/home/penguin/code/MarchProxy/manager/middleware/auth.py`
- **Package Init**: `/home/penguin/code/MarchProxy/manager/middleware/__init__.py`
- **Usage Guide**: `/home/penguin/code/MarchProxy/manager/middleware/USAGE.md`
