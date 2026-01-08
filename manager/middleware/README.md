# Authentication Middleware for MarchProxy Manager

Comprehensive JWT-based authentication and authorization middleware for protecting API endpoints in MarchProxy Manager.

## Overview

The authentication middleware provides:

- **JWT Token Validation**: Extract and validate JWT tokens from `Authorization: Bearer <token>` headers
- **User Context Management**: Store authenticated user information in `g.user` for access in route handlers
- **Admin-Only Routes**: Enforce admin-only access with the `admin_required=True` parameter
- **License-Gated Features**: Optional feature-based access control via `license_feature` parameter
- **Async Support**: Full support for both synchronous and asynchronous route handlers
- **Error Handling**: Standardized JSON error responses with appropriate HTTP status codes
- **Manual Context**: `AuthContext` context manager for fine-grained authentication control

## Files

### Core Implementation
- **`auth.py`** (318 lines) - Main authentication middleware implementation
  - `require_auth()` - Main decorator for protecting routes
  - `get_current_user()` - Helper to access authenticated user
  - `is_admin()` - Helper to check admin status
  - `AuthContext` - Context manager for manual auth checking
  - Private functions for token extraction and validation

### Package Structure
- **`__init__.py`** (23 lines) - Package initialization, exports public API
- **`README.md`** - This file
- **`USAGE.md`** - Comprehensive usage guide with examples
- **`INTEGRATION_EXAMPLES.py`** - Real-world integration patterns

## Installation

The middleware is already integrated into the manager application. Import from the middleware package:

```python
from middleware.auth import require_auth, get_current_user, is_admin, AuthContext
```

## Quick Start

### Protect a Route

```python
from py4web import application
from middleware.auth import require_auth, get_current_user

@application.route('/api/profile', methods=['GET'])
@require_auth()
def get_profile():
    user = get_current_user()
    return {"username": user['username']}
```

### Admin-Only Route

```python
@application.route('/api/admin/users', methods=['GET'])
@require_auth(admin_required=True)
def list_users():
    # Only admins can access
    return {"users": [...]}
```

### License-Gated Feature

```python
@application.route('/api/advanced-rules', methods=['GET'])
@require_auth(license_feature="advanced_blocking")
def get_advanced_rules():
    # Only if license includes this feature
    return {"rules": [...]}
```

## API Reference

### Decorators

#### `@require_auth(admin_required=False, license_feature=None)`

Decorator to protect routes requiring authentication.

**Parameters:**
- `admin_required` (bool): If True, only admins can access. Default: False
- `license_feature` (str): If set, check if user has access to this feature. Default: None

**Returns:** JSON error response with appropriate HTTP status on failure

**Supported Sync/Async:** Yes (handles both automatically)

**Examples:**
```python
@require_auth()                                      # Any authenticated user
@require_auth(admin_required=True)                   # Admins only
@require_auth(license_feature="threat_intel")       # With license check
@require_auth(admin_required=True, license_feature="advanced_blocking")  # Both
```

### Helper Functions

#### `get_current_user() -> Optional[dict]`

Get the current authenticated user's payload.

**Returns:**
- `dict`: User payload with `user_id`, `username`, `email`, `is_admin`, `type`, `exp`, `iat`, `iss`
- `None`: If no user authenticated

**Example:**
```python
user = get_current_user()
if user:
    print(f"User ID: {user['user_id']}")
```

#### `is_admin() -> bool`

Check if current user is an administrator.

**Returns:** `bool` - True if admin, False otherwise

**Example:**
```python
if is_admin():
    # Admin-only logic
```

### Context Manager

#### `AuthContext()`

Manual authentication context for complex authorization flows.

**Methods:**
- `is_authenticated()` -> bool: Check if authenticated
- `get_user()` -> Optional[dict]: Get user payload
- `is_admin()` -> bool: Check if admin
- `has_feature(feature: str)` -> bool: Check license feature

**Example:**
```python
with AuthContext() as auth:
    if not auth.is_authenticated():
        return {"error": "Not authenticated"}, 401
    if not auth.is_admin():
        return {"error": "Admin required"}, 403
    user = auth.get_user()
    # Process request
```

## Error Responses

All authentication failures return JSON error responses with appropriate HTTP status codes:

### 401 Unauthorized

**Missing Authorization Header:**
```json
{"error": "Missing authorization header"}
```

**Invalid or Expired Token:**
```json
{"error": "Invalid or expired token"}
```

### 403 Forbidden

**Admin Access Required:**
```json
{"error": "Admin access required"}
```

**License Feature Not Available:**
```json
{"error": "Feature 'advanced_blocking' not licensed"}
```

## JWT Token Format

Tokens contain the following payload:

```json
{
    "user_id": 1,
    "username": "john_doe",
    "email": "john@example.com",
    "is_admin": false,
    "type": "access",
    "iat": 1704888000,
    "exp": 1704974400,
    "iss": "marchproxy"
}
```

**Header Format:**
```
Authorization: Bearer <token>
```

Note: Space between "Bearer" and token is required. Invalid formats:
- ❌ `Bearer: <token>` (colon instead of space)
- ❌ `Bearer  <token>` (extra spaces)
- ❌ `<token>` (missing Bearer prefix)

## Configuration

### JWT Secret

Set via environment variable (required):

```bash
export JWT_SECRET="your-super-secret-jwt-key-change-in-production"
```

### JWT Manager

Already initialized in `app.py`:

```python
from models.auth import JWTManager

JWT_SECRET = os.environ.get('JWT_SECRET', 'default-key')
jwt_manager = JWTManager(JWT_SECRET, algorithm='HS256', ttl_hours=24)
current_app.jwt_manager = jwt_manager
```

### License Manager

For license feature checking:

```python
from models.license import LicenseManager

license_manager = LicenseManager(db, license_key)
current_app.license_manager = license_manager
```

## Best Practices

1. ✅ Always use `@require_auth()` on protected routes
2. ✅ Be explicit with `admin_required=True` for admin endpoints
3. ✅ Use `license_feature` for premium functionality
4. ✅ Access user context with `get_current_user()`, never raw request data
5. ✅ Log authentication failures for security monitoring
6. ✅ Use HTTPS in production to prevent token interception
7. ✅ Keep JWT secret secure and rotate regularly
8. ✅ Validate token signature (always use JWTManager)
9. ✅ Implement token refresh mechanisms
10. ✅ Handle auth failures gracefully with proper error responses

## Troubleshooting

### Token Not Extracted
- Verify header format: `Authorization: Bearer <token>`
- Check for case sensitivity issues
- Ensure header is being sent by client

### Token Validation Failing
- Verify JWT secret matches
- Check token expiration
- Confirm token wasn't modified

### Admin Check Failing
- Verify `is_admin` field in token is `True`
- Check JWTManager initialization
- Ensure user created with admin flag

### License Feature Not Working
- Implement actual checking in `_check_license_feature()`
- Verify license manager configured in `current_app`
- Check license database/cache

## Documentation

- **`README.md`** - Overview (this file)
- **`USAGE.md`** - Comprehensive usage guide with examples
- **`INTEGRATION_EXAMPLES.py`** - Real-world integration patterns
- **`auth.py`** - Source code with detailed docstrings

## Related Files

- **Auth Models:** `/home/penguin/code/MarchProxy/manager/models/auth.py`
- **License Models:** `/home/penguin/code/MarchProxy/manager/models/license.py`
- **Main App:** `/home/penguin/code/MarchProxy/manager/app.py`
- **Auth API:** `/home/penguin/code/MarchProxy/manager/api/auth.py`

## Integration Status

✅ Middleware created and ready for integration with existing API endpoints
✅ Supports py4web framework
✅ Compatible with JWTManager from auth models
✅ Async/await support for modern handlers
✅ Comprehensive error handling
✅ Full documentation with examples

## Next Steps

1. Import middleware in route handlers
2. Add `@require_auth()` decorator to protected endpoints
3. Update API endpoints to use the decorator
4. Test token extraction and validation
5. Verify admin-only routes
6. Test license feature checking (when implemented)

## Support

For issues, questions, or contributions:
- Check `USAGE.md` for comprehensive examples
- Review `INTEGRATION_EXAMPLES.py` for integration patterns
- See source code docstrings for implementation details
