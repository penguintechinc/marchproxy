"""
Integration examples showing how to use the authentication middleware
with MarchProxy Manager's existing API endpoints.

These examples demonstrate various patterns for protecting routes with
JWT authentication, admin-only access, and license-gated features.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from py4web import application, request, response
from middleware.auth import require_auth, get_current_user, is_admin, AuthContext
import logging

logger = logging.getLogger(__name__)


# Example 1: Basic authenticated endpoint
@application.route("/api/user/profile", methods=["GET"])
@require_auth()
def get_user_profile():
    """
    Get current user's profile.

    Requires: Valid JWT token in Authorization header

    Request:
        GET /api/user/profile
        Authorization: Bearer <token>

    Response (200):
        {
            "user_id": 1,
            "username": "john_doe",
            "email": "john@example.com",
            "is_admin": false
        }

    Response (401):
        {"error": "Missing authorization header"}
        {"error": "Invalid or expired token"}
    """
    user = get_current_user()
    return {
        "user_id": user["user_id"],
        "username": user["username"],
        "email": user["email"],
        "is_admin": user["is_admin"],
    }


# Example 2: Admin-only endpoint
@application.route("/api/admin/users", methods=["GET"])
@require_auth(admin_required=True)
def list_all_users():
    """
    List all users in the system.

    Requires: Valid JWT token + Admin role

    Request:
        GET /api/admin/users
        Authorization: Bearer <admin_token>

    Response (200):
        {
            "users": [
                {"user_id": 1, "username": "admin", "is_admin": true},
                {"user_id": 2, "username": "user1", "is_admin": false}
            ]
        }

    Response (401):
        {"error": "Missing authorization header"}

    Response (403):
        {"error": "Admin access required"}
    """
    # In real implementation, fetch from database
    return {
        "users": [
            {"user_id": 1, "username": "admin", "is_admin": True},
            {"user_id": 2, "username": "user1", "is_admin": False},
        ]
    }


# Example 3: Admin-only create endpoint
@application.route("/api/admin/users", methods=["POST"])
@require_auth(admin_required=True)
def create_user():
    """
    Create a new user.

    Requires: Valid JWT token + Admin role

    Request:
        POST /api/admin/users
        Authorization: Bearer <admin_token>
        {
            "username": "newuser",
            "email": "newuser@example.com",
            "password": "SecurePass123"
        }

    Response (201):
        {"user_id": 3, "username": "newuser"}

    Response (400):
        {"error": "Validation error", "details": "..."}

    Response (403):
        {"error": "Admin access required"}
    """
    try:
        data = request.json
        # Validate input
        if not data.get("username") or not data.get("password"):
            response.status = 400
            return {"error": "Missing required fields"}

        # In real implementation, create user in database
        user_id = 3
        return {"user_id": user_id, "username": data["username"]}
    except Exception as e:
        logger.error(f"Error creating user: {e}")
        response.status = 500
        return {"error": "Failed to create user"}


# Example 4: License-gated feature
@application.route("/api/advanced/threat-intelligence", methods=["GET"])
@require_auth(license_feature="threat_intelligence")
def get_threat_intelligence():
    """
    Get threat intelligence feed.

    Requires: Valid JWT token + threat_intelligence feature licensed

    Request:
        GET /api/advanced/threat-intelligence
        Authorization: Bearer <token>

    Response (200):
        {
            "threats": [
                {"id": 1, "type": "malware", "severity": "high"},
                {"id": 2, "type": "phishing", "severity": "medium"}
            ]
        }

    Response (403):
        {"error": "Feature 'threat_intelligence' not licensed"}
    """
    return {
        "threats": [
            {"id": 1, "type": "malware", "severity": "high"},
            {"id": 2, "type": "phishing", "severity": "medium"},
        ]
    }


# Example 5: License-gated admin endpoint
@application.route("/api/admin/advanced-blocking", methods=["GET"])
@require_auth(admin_required=True, license_feature="advanced_blocking")
def get_advanced_blocking_config():
    """
    Get advanced blocking configuration.

    Requires: Valid JWT token + Admin role + advanced_blocking licensed

    Request:
        GET /api/admin/advanced-blocking
        Authorization: Bearer <admin_token>

    Response (200):
        {
            "blocking_rules": [...],
            "threat_feeds": [...]
        }

    Response (403):
        {"error": "Admin access required"}
        OR
        {"error": "Feature 'advanced_blocking' not licensed"}
    """
    return {"blocking_rules": [], "threat_feeds": []}


# Example 6: Manual authentication context (for complex flows)
@application.route("/api/cluster/config/<int:cluster_id>", methods=["GET", "PUT"])
def manage_cluster_config(cluster_id):
    """
    Manage cluster configuration with flexible auth checking.

    Requires:
        - GET: Authenticated user with access to cluster
        - PUT: Admin role

    Request:
        GET /api/cluster/config/1
        Authorization: Bearer <token>

        PUT /api/cluster/config/1
        Authorization: Bearer <admin_token>

    Response (200):
        {"config": {...}}

    Response (401):
        {"error": "Not authenticated"}

    Response (403):
        {"error": "No access to cluster"}
        OR
        {"error": "Admin required for updates"}
    """
    with AuthContext() as auth:
        # Check basic authentication
        if not auth.is_authenticated():
            response.status = 401
            return {"error": "Not authenticated"}

        # Handle GET
        if request.method == "GET":
            user = auth.get_user()
            # Check user has access to this cluster
            # (In real implementation, check database)
            return {"config": {"cluster_id": cluster_id}}

        # Handle PUT - admin only
        elif request.method == "PUT":
            if not auth.is_admin():
                response.status = 403
                return {"error": "Admin required for updates"}

            data = request.json
            # Update cluster config in database
            return {"status": "updated"}

    response.status = 405
    return {"error": "Method not allowed"}


# Example 7: Using helper functions in handlers
@application.route("/api/user/activity", methods=["GET"])
@require_auth()
def get_user_activity():
    """
    Get user's activity log.

    Uses helper functions to access user info.
    """
    user = get_current_user()
    user_id = user["user_id"]
    is_admin_user = is_admin()

    # If admin, show all activity, otherwise show only own activity
    if is_admin_user:
        # Show system-wide activity
        activity = [
            {"user_id": 1, "action": "login"},
            {"user_id": 2, "action": "create_rule"},
        ]
    else:
        # Show only this user's activity
        activity = [
            {"user_id": user_id, "action": "login"},
            {"user_id": user_id, "action": "update_profile"},
        ]

    return {"activity": activity}


# Example 8: Async handler with authentication
@application.route("/api/async/proxy-health", methods=["GET"])
@require_auth()
async def get_proxy_health():
    """
    Get proxy health status with async operations.

    The middleware supports async handlers seamlessly.
    """
    user = get_current_user()

    # In real implementation, could do async database queries,
    # HTTP requests to proxies, etc.
    return {
        "proxies": [
            {"id": 1, "status": "healthy"},
            {"id": 2, "status": "healthy"},
            {"id": 3, "status": "degraded"},
        ]
    }


# Example 9: Conditional authorization based on resource ownership
@application.route("/api/cluster/<int:cluster_id>/rules", methods=["GET", "DELETE"])
@require_auth()
def manage_cluster_rules(cluster_id):
    """
    Manage cluster block rules with conditional authorization.

    Authorization:
    - GET: User must be assigned to cluster
    - DELETE: User must be assigned AND have admin role for cluster
    """
    user = get_current_user()
    user_id = user["user_id"]

    # In real implementation, check database for cluster membership
    # and roles

    if request.method == "GET":
        # Check if user has access to this cluster
        has_access = True  # Would check database
        if not has_access:
            response.status = 403
            return {"error": "No access to this cluster"}

        return {"rules": []}

    elif request.method == "DELETE":
        # Delete requires admin access to cluster
        has_admin_access = False  # Would check database
        if not has_admin_access:
            response.status = 403
            return {"error": "Admin access required for this cluster"}

        return {"status": "rules_deleted"}


# Example 10: Token refresh endpoint (no auth required)
@application.route("/api/auth/refresh", methods=["POST"])
def refresh_token():
    """
    Refresh access token using refresh token.

    Note: This endpoint does NOT use @require_auth() because the request
    includes a refresh token rather than access token.

    Request:
        POST /api/auth/refresh
        {
            "refresh_token": "<refresh_token>"
        }

    Response (200):
        {
            "access_token": "<new_access_token>",
            "token_type": "Bearer",
            "expires_in": 86400
        }

    Response (401):
        {"error": "Invalid refresh token"}
    """
    try:
        data = request.json
        refresh_token = data.get("refresh_token")

        if not refresh_token:
            response.status = 400
            return {"error": "Missing refresh_token"}

        # In real implementation, use jwt_manager to refresh token
        # new_token = jwt_manager.refresh_access_token(refresh_token)

        return {
            "access_token": "<new_token>",
            "token_type": "Bearer",
            "expires_in": 86400,
        }
    except Exception as e:
        logger.error(f"Token refresh failed: {e}")
        response.status = 401
        return {"error": "Invalid refresh token"}


# Integration notes:
#
# 1. All @require_auth() decorated endpoints will automatically:
#    - Extract JWT token from Authorization header
#    - Validate token using JWTManager
#    - Store user payload in g.user
#    - Return 401 if auth fails
#    - Return 403 if admin_required or license_feature check fails
#
# 2. Use get_current_user() to access user data in handlers
#
# 3. Use is_admin() to check if current user is admin
#
# 4. Use AuthContext for manual auth handling in complex flows
#
# 5. All error responses are JSON with appropriate HTTP status codes
#
# 6. Middleware supports both sync and async handlers
#
# 7. Always put @require_auth() BEFORE other decorators
#
# 8. Token must be in format: "Bearer <token>"
#    NOT "Bearer: <token>" or "Bearer  <token>" (extra spaces)
