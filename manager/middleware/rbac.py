"""
RBAC Middleware and Decorators for MarchProxy

Provides decorators and middleware for enforcing role-based access control
with OAuth2-style scoped permissions.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import functools
import logging
from typing import Callable, List, Optional, Union

from quart import abort, request, g
from werkzeug.exceptions import Forbidden, Unauthorized

from models.rbac import RBACModel, Permissions, PermissionScope

logger = logging.getLogger(__name__)


class RBACMiddleware:
    """RBAC middleware for Quart applications"""

    def __init__(self, app=None):
        self.app = app
        if app is not None:
            self.init_app(app)

    def init_app(self, app):
        """Initialize middleware with Quart app"""
        app.before_request(self.load_user_permissions)

    async def load_user_permissions(self):
        """Load user permissions into request context"""
        # Get current user from session/JWT
        user_id = g.get('user_id')
        if user_id:
            db = g.get('db')
            if db:
                g.permissions = RBACModel.get_user_permissions(db, user_id)
            else:
                g.permissions = {'global': [], 'cluster': {}, 'service': {}}
        else:
            g.permissions = {'global': [], 'cluster': {}, 'service': {}}


def requires_permission(
    permission: str,
    resource_type: Optional[str] = None,
    resource_id_param: Optional[str] = None
):
    """
    Decorator to require specific permission for route access.

    Args:
        permission: Permission scope required (e.g., Permissions.GLOBAL_ADMIN)
        resource_type: Type of resource ('cluster' or 'service')
        resource_id_param: Request parameter name containing resource ID

    Example:
        @requires_permission(Permissions.GLOBAL_CLUSTER_WRITE)
        async def create_cluster():
            ...

        @requires_permission(Permissions.CLUSTER_WRITE, 'cluster', 'cluster_id')
        async def update_cluster(cluster_id):
            ...
    """
    def decorator(f: Callable) -> Callable:
        @functools.wraps(f)
        async def decorated_function(*args, **kwargs):
            # Check if user is authenticated
            user_id = g.get('user_id')
            if not user_id:
                logger.warning(f"Unauthenticated access attempt to {f.__name__}")
                abort(401, "Authentication required")

            db = g.get('db')
            if not db:
                logger.error("Database not available in request context")
                abort(500, "Internal server error")

            # Get resource ID if specified
            resource_id = None
            if resource_id_param:
                # Try to get from kwargs first (URL params)
                resource_id = kwargs.get(resource_id_param)
                # Try request args if not in kwargs
                if resource_id is None:
                    resource_id = request.args.get(resource_id_param)
                # Try JSON body
                if resource_id is None and request.is_json:
                    json_data = await request.get_json()
                    resource_id = json_data.get(resource_id_param)

                if resource_id:
                    try:
                        resource_id = int(resource_id)
                    except (ValueError, TypeError):
                        abort(400, f"Invalid {resource_id_param}")

            # Check permission
            has_perm = RBACModel.has_permission(
                db, user_id, permission, resource_type, resource_id
            )

            if not has_perm:
                logger.warning(
                    f"Permission denied: user={user_id}, permission={permission}, "
                    f"resource_type={resource_type}, resource_id={resource_id}"
                )
                abort(403, "Insufficient permissions")

            return await f(*args, **kwargs)

        return decorated_function
    return decorator


def requires_role(
    role_name: str,
    scope: PermissionScope = PermissionScope.GLOBAL,
    resource_id_param: Optional[str] = None
):
    """
    Decorator to require specific role for route access.

    Args:
        role_name: Role name required (e.g., 'admin', 'maintainer')
        scope: Permission scope (global, cluster, service)
        resource_id_param: Request parameter name containing resource ID for scoped roles

    Example:
        @requires_role('admin')
        async def admin_dashboard():
            ...

        @requires_role('cluster_admin', PermissionScope.CLUSTER, 'cluster_id')
        async def manage_cluster(cluster_id):
            ...
    """
    def decorator(f: Callable) -> Callable:
        @functools.wraps(f)
        async def decorated_function(*args, **kwargs):
            # Check if user is authenticated
            user_id = g.get('user_id')
            if not user_id:
                abort(401, "Authentication required")

            db = g.get('db')
            if not db:
                abort(500, "Internal server error")

            # Get resource ID if specified
            resource_id = None
            if resource_id_param and scope != PermissionScope.GLOBAL:
                resource_id = kwargs.get(resource_id_param)
                if resource_id is None:
                    resource_id = request.args.get(resource_id_param)
                if resource_id is None and request.is_json:
                    json_data = await request.get_json()
                    resource_id = json_data.get(resource_id_param)

                if resource_id:
                    try:
                        resource_id = int(resource_id)
                    except (ValueError, TypeError):
                        abort(400, f"Invalid {resource_id_param}")

            # Check if user has role
            user_roles = RBACModel.get_user_roles(db, user_id)

            has_role = False
            for role in user_roles:
                if role['role_name'] == role_name:
                    if scope == PermissionScope.GLOBAL and role['scope'] == scope.value:
                        has_role = True
                        break
                    elif resource_id and role['resource_id'] == resource_id:
                        has_role = True
                        break

            if not has_role:
                logger.warning(
                    f"Role requirement not met: user={user_id}, role={role_name}, "
                    f"scope={scope.value}, resource_id={resource_id}"
                )
                abort(403, "Insufficient permissions")

            return await f(*args, **kwargs)

        return decorated_function
    return decorator


def requires_any_permission(*permissions: str):
    """
    Decorator to require ANY of the specified permissions.

    Example:
        @requires_any_permission(Permissions.GLOBAL_ADMIN, Permissions.CLUSTER_WRITE)
        async def manage_resource():
            ...
    """
    def decorator(f: Callable) -> Callable:
        @functools.wraps(f)
        async def decorated_function(*args, **kwargs):
            user_id = g.get('user_id')
            if not user_id:
                abort(401, "Authentication required")

            db = g.get('db')
            if not db:
                abort(500, "Internal server error")

            # Check if user has any of the required permissions
            user_perms = RBACModel.get_user_permissions(db, user_id)

            has_permission = False
            for perm in permissions:
                if perm in user_perms['global']:
                    has_permission = True
                    break
                # Could also check scoped permissions here

            if not has_permission:
                logger.warning(
                    f"No required permissions: user={user_id}, required={permissions}"
                )
                abort(403, "Insufficient permissions")

            return await f(*args, **kwargs)

        return decorated_function
    return decorator


def requires_all_permissions(*permissions: str):
    """
    Decorator to require ALL of the specified permissions.

    Example:
        @requires_all_permissions(
            Permissions.CLUSTER_READ,
            Permissions.CLUSTER_WRITE
        )
        async def manage_cluster():
            ...
    """
    def decorator(f: Callable) -> Callable:
        @functools.wraps(f)
        async def decorated_function(*args, **kwargs):
            user_id = g.get('user_id')
            if not user_id:
                abort(401, "Authentication required")

            db = g.get('db')
            if not db:
                abort(500, "Internal server error")

            # Check if user has all required permissions
            user_perms = RBACModel.get_user_permissions(db, user_id)

            missing_permissions = []
            for perm in permissions:
                if perm not in user_perms['global']:
                    missing_permissions.append(perm)

            if missing_permissions:
                logger.warning(
                    f"Missing permissions: user={user_id}, missing={missing_permissions}"
                )
                abort(403, "Insufficient permissions")

            return await f(*args, **kwargs)

        return decorated_function
    return decorator


def is_admin(user_id: int, db) -> bool:
    """Helper function to check if user is admin"""
    perms = RBACModel.get_user_permissions(db, user_id)
    return Permissions.GLOBAL_ADMIN in perms['global']


def can_manage_users(user_id: int, db) -> bool:
    """Helper function to check if user can manage other users"""
    perms = RBACModel.get_user_permissions(db, user_id)
    return (
        Permissions.GLOBAL_ADMIN in perms['global'] or
        Permissions.GLOBAL_USER_WRITE in perms['global']
    )


def can_access_cluster(user_id: int, cluster_id: int, db) -> bool:
    """Helper function to check if user can access cluster"""
    return RBACModel.has_permission(
        db, user_id, Permissions.CLUSTER_READ, 'cluster', cluster_id
    )


def can_access_service(user_id: int, service_id: int, db) -> bool:
    """Helper function to check if user can access service"""
    return RBACModel.has_permission(
        db, user_id, Permissions.SERVICE_READ, 'service', service_id
    )
