"""
Role-Based Access Control (RBAC) Models for MarchProxy

Implements OAuth2-style scoped permissions with three levels:
- Global: System-wide permissions
- Cluster: Cluster-specific permissions
- Service: Service-specific permissions

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from dataclasses import dataclass
from datetime import datetime
from enum import Enum
from typing import Dict, List, Optional, Set
from pydal import DAL, Field


class PermissionScope(Enum):
    """Permission scope levels"""

    GLOBAL = "global"  # System-wide permissions
    CLUSTER = "cluster"  # Cluster-level permissions
    SERVICE = "service"  # Service-level permissions


class RoleType(Enum):
    """Standard role types as defined in STANDARDS.md"""

    ADMIN = "admin"  # Full system access
    MAINTAINER = "maintainer"  # Read/write, no user management
    VIEWER = "viewer"  # Read-only access
    SERVICE_OWNER = "service_owner"  # Service creator/owner
    CLUSTER_ADMIN = "cluster_admin"  # Cluster administrator


# OAuth2-style permission scopes
class Permissions:
    """OAuth2-style permission scopes"""

    # Global permissions (system-wide)
    GLOBAL_ADMIN = "global:admin"  # Full system admin
    GLOBAL_USER_READ = "global:users:read"  # Read all users
    GLOBAL_USER_WRITE = "global:users:write"  # Manage all users
    GLOBAL_CLUSTER_READ = "global:clusters:read"  # Read all clusters
    GLOBAL_CLUSTER_WRITE = "global:clusters:write"  # Manage all clusters
    GLOBAL_SERVICE_READ = "global:services:read"  # Read all services
    GLOBAL_SERVICE_WRITE = "global:services:write"  # Manage all services
    GLOBAL_SETTINGS = "global:settings"  # System settings
    GLOBAL_LICENSE = "global:license"  # License management

    # Cluster permissions (cluster-scoped)
    CLUSTER_READ = "cluster:read"  # Read cluster details
    CLUSTER_WRITE = "cluster:write"  # Update cluster
    CLUSTER_DELETE = "cluster:delete"  # Delete cluster
    CLUSTER_SERVICES_READ = "cluster:services:read"  # Read cluster services
    CLUSTER_SERVICES_WRITE = "cluster:services:write"  # Manage cluster services
    CLUSTER_USERS_READ = "cluster:users:read"  # Read cluster users
    CLUSTER_USERS_WRITE = "cluster:users:write"  # Manage cluster users

    # Service permissions (service-scoped)
    SERVICE_READ = "service:read"  # Read service details
    SERVICE_WRITE = "service:write"  # Update service
    SERVICE_DELETE = "service:delete"  # Delete service
    SERVICE_PROXY_READ = "service:proxies:read"  # Read service proxies
    SERVICE_PROXY_WRITE = "service:proxies:write"  # Manage service proxies
    SERVICE_CERT_READ = "service:certs:read"  # Read certificates
    SERVICE_CERT_WRITE = "service:certs:write"  # Manage certificates


# Default role definitions with their permissions
DEFAULT_ROLES = {
    RoleType.ADMIN.value: {
        "name": "Admin",
        "description": "Full system access - can manage everything",
        "scope": PermissionScope.GLOBAL.value,
        "permissions": [
            Permissions.GLOBAL_ADMIN,
            Permissions.GLOBAL_USER_READ,
            Permissions.GLOBAL_USER_WRITE,
            Permissions.GLOBAL_CLUSTER_READ,
            Permissions.GLOBAL_CLUSTER_WRITE,
            Permissions.GLOBAL_SERVICE_READ,
            Permissions.GLOBAL_SERVICE_WRITE,
            Permissions.GLOBAL_SETTINGS,
            Permissions.GLOBAL_LICENSE,
        ],
    },
    RoleType.MAINTAINER.value: {
        "name": "Maintainer",
        "description": "Read/write access - no user management",
        "scope": PermissionScope.GLOBAL.value,
        "permissions": [
            Permissions.GLOBAL_CLUSTER_READ,
            Permissions.GLOBAL_CLUSTER_WRITE,
            Permissions.GLOBAL_SERVICE_READ,
            Permissions.GLOBAL_SERVICE_WRITE,
            Permissions.GLOBAL_USER_READ,  # Can read but not write users
        ],
    },
    RoleType.VIEWER.value: {
        "name": "Viewer",
        "description": "Read-only access to all resources",
        "scope": PermissionScope.GLOBAL.value,
        "permissions": [
            Permissions.GLOBAL_CLUSTER_READ,
            Permissions.GLOBAL_SERVICE_READ,
            Permissions.GLOBAL_USER_READ,
        ],
    },
    RoleType.CLUSTER_ADMIN.value: {
        "name": "Cluster Admin",
        "description": "Full access to specific cluster",
        "scope": PermissionScope.CLUSTER.value,
        "permissions": [
            Permissions.CLUSTER_READ,
            Permissions.CLUSTER_WRITE,
            Permissions.CLUSTER_SERVICES_READ,
            Permissions.CLUSTER_SERVICES_WRITE,
            Permissions.CLUSTER_USERS_READ,
            Permissions.CLUSTER_USERS_WRITE,
        ],
    },
    RoleType.SERVICE_OWNER.value: {
        "name": "Service Owner",
        "description": "Full access to specific service",
        "scope": PermissionScope.SERVICE.value,
        "permissions": [
            Permissions.SERVICE_READ,
            Permissions.SERVICE_WRITE,
            Permissions.SERVICE_DELETE,
            Permissions.SERVICE_PROXY_READ,
            Permissions.SERVICE_PROXY_WRITE,
            Permissions.SERVICE_CERT_READ,
            Permissions.SERVICE_CERT_WRITE,
        ],
    },
}


@dataclass
class RoleAssignment:
    """Role assignment with scope"""

    role_id: int
    scope: PermissionScope
    resource_id: Optional[int] = None  # Cluster ID or Service ID for scoped roles


class RBACModel:
    """RBAC model with database operations"""

    @staticmethod
    def define_tables(db: DAL):
        """Define RBAC tables in database"""

        # Roles table - defines available roles
        db.define_table(
            "roles",
            Field("name", type="string", unique=True, required=True, length=50),
            Field("display_name", type="string", required=True, length=100),
            Field("description", type="text"),
            Field(
                "scope",
                type="string",
                required=True,
                length=20,
                requires=lambda v: v in [s.value for s in PermissionScope],
            ),
            Field("permissions", type="json", default=[]),  # List of permission scopes
            Field(
                "is_system", type="boolean", default=False
            ),  # System role (cannot be deleted)
            Field("is_active", type="boolean", default=True),
            Field("created_at", type="datetime", default=datetime.utcnow),
            Field("updated_at", type="datetime", update=datetime.utcnow),
        )

        # User role assignments - links users to roles with scope
        db.define_table(
            "user_roles",
            Field("user_id", type="reference users", required=True),
            Field("role_id", type="reference roles", required=True),
            Field(
                "scope",
                type="string",
                required=True,
                length=20,
                requires=lambda v: v in [s.value for s in PermissionScope],
            ),
            Field(
                "resource_id", type="integer"
            ),  # Cluster or Service ID (null for global)
            Field("granted_by", type="reference users"),
            Field("granted_at", type="datetime", default=datetime.utcnow),
            Field("expires_at", type="datetime"),  # Optional expiration
            Field("is_active", type="boolean", default=True),
        )

        # Permission cache - denormalized for performance
        db.define_table(
            "user_permissions_cache",
            Field("user_id", type="reference users", required=True, unique=True),
            Field("global_permissions", type="json", default=[]),
            Field(
                "cluster_permissions", type="json", default={}
            ),  # {cluster_id: [perms]}
            Field(
                "service_permissions", type="json", default={}
            ),  # {service_id: [perms]}
            Field("last_updated", type="datetime", default=datetime.utcnow),
        )

        return db

    @staticmethod
    def initialize_default_roles(db: DAL):
        """Create default system roles"""
        for role_key, role_data in DEFAULT_ROLES.items():
            existing = db(db.roles.name == role_key).select().first()
            if not existing:
                db.roles.insert(
                    name=role_key,
                    display_name=role_data["name"],
                    description=role_data["description"],
                    scope=role_data["scope"],
                    permissions=role_data["permissions"],
                    is_system=True,
                    is_active=True,
                )
        db.commit()

    @staticmethod
    def assign_role(
        db: DAL,
        user_id: int,
        role_name: str,
        scope: PermissionScope = PermissionScope.GLOBAL,
        resource_id: Optional[int] = None,
        granted_by: Optional[int] = None,
    ) -> int:
        """Assign role to user with scope"""

        # Get role
        role = db(db.roles.name == role_name).select().first()
        if not role:
            raise ValueError(f"Role {role_name} not found")

        # Validate scope matches role scope
        if role.scope != scope.value:
            raise ValueError(f"Role {role_name} requires scope {role.scope}")

        # Check if assignment already exists
        existing = (
            db(
                (db.user_roles.user_id == user_id)
                & (db.user_roles.role_id == role.id)
                & (db.user_roles.scope == scope.value)
                & (db.user_roles.resource_id == resource_id)
                & (db.user_roles.is_active == True)
            )
            .select()
            .first()
        )

        if existing:
            return existing.id

        # Create assignment
        assignment_id = db.user_roles.insert(
            user_id=user_id,
            role_id=role.id,
            scope=scope.value,
            resource_id=resource_id,
            granted_by=granted_by,
            granted_at=datetime.utcnow(),
            is_active=True,
        )

        # Invalidate permission cache
        RBACModel.invalidate_permission_cache(db, user_id)

        db.commit()
        return assignment_id

    @staticmethod
    def revoke_role(
        db: DAL, user_id: int, role_name: str, resource_id: Optional[int] = None
    ):
        """Revoke role from user"""

        role = db(db.roles.name == role_name).select().first()
        if not role:
            raise ValueError(f"Role {role_name} not found")

        query = (
            (db.user_roles.user_id == user_id)
            & (db.user_roles.role_id == role.id)
            & (db.user_roles.is_active == True)
        )

        if resource_id is not None:
            query &= db.user_roles.resource_id == resource_id

        db(query).update(is_active=False)

        # Invalidate permission cache
        RBACModel.invalidate_permission_cache(db, user_id)

        db.commit()

    @staticmethod
    def get_user_permissions(db: DAL, user_id: int) -> Dict[str, any]:
        """Get all permissions for user (cached)"""

        # Check cache first
        cache = db(db.user_permissions_cache.user_id == user_id).select().first()
        if cache:
            return {
                "global": cache.global_permissions or [],
                "cluster": cache.cluster_permissions or {},
                "service": cache.service_permissions or {},
            }

        # Build permissions from role assignments
        permissions = {
            "global": set(),
            "cluster": {},
            "service": {},
        }

        # Get all active role assignments
        assignments = db(
            (db.user_roles.user_id == user_id) & (db.user_roles.is_active == True)
        ).select(
            db.user_roles.ALL,
            db.roles.ALL,
            left=db.roles.on(db.user_roles.role_id == db.roles.id),
        )

        for assignment in assignments:
            role_perms = assignment.roles.permissions or []

            if assignment.user_roles.scope == PermissionScope.GLOBAL.value:
                permissions["global"].update(role_perms)

            elif assignment.user_roles.scope == PermissionScope.CLUSTER.value:
                cluster_id = str(assignment.user_roles.resource_id)
                if cluster_id not in permissions["cluster"]:
                    permissions["cluster"][cluster_id] = set()
                permissions["cluster"][cluster_id].update(role_perms)

            elif assignment.user_roles.scope == PermissionScope.SERVICE.value:
                service_id = str(assignment.user_roles.resource_id)
                if service_id not in permissions["service"]:
                    permissions["service"][service_id] = set()
                permissions["service"][service_id].update(role_perms)

        # Convert sets to lists for JSON storage
        result = {
            "global": list(permissions["global"]),
            "cluster": {k: list(v) for k, v in permissions["cluster"].items()},
            "service": {k: list(v) for k, v in permissions["service"].items()},
        }

        # Cache the result
        if cache:
            db(db.user_permissions_cache.user_id == user_id).update(
                global_permissions=result["global"],
                cluster_permissions=result["cluster"],
                service_permissions=result["service"],
                last_updated=datetime.utcnow(),
            )
        else:
            db.user_permissions_cache.insert(
                user_id=user_id,
                global_permissions=result["global"],
                cluster_permissions=result["cluster"],
                service_permissions=result["service"],
                last_updated=datetime.utcnow(),
            )

        db.commit()
        return result

    @staticmethod
    def has_permission(
        db: DAL,
        user_id: int,
        permission: str,
        resource_type: Optional[str] = None,
        resource_id: Optional[int] = None,
    ) -> bool:
        """Check if user has specific permission"""

        perms = RBACModel.get_user_permissions(db, user_id)

        # Check global permissions first
        if permission in perms["global"]:
            return True

        # Check global admin
        if Permissions.GLOBAL_ADMIN in perms["global"]:
            return True

        # Check resource-specific permissions
        if resource_type == "cluster" and resource_id:
            cluster_perms = perms["cluster"].get(str(resource_id), [])
            if permission in cluster_perms:
                return True

        if resource_type == "service" and resource_id:
            service_perms = perms["service"].get(str(resource_id), [])
            if permission in service_perms:
                return True

        return False

    @staticmethod
    def invalidate_permission_cache(db: DAL, user_id: int):
        """Invalidate permission cache for user"""
        db(db.user_permissions_cache.user_id == user_id).delete()

    @staticmethod
    def get_user_roles(db: DAL, user_id: int) -> List[Dict]:
        """Get all roles assigned to user"""

        assignments = db(
            (db.user_roles.user_id == user_id) & (db.user_roles.is_active == True)
        ).select(
            db.user_roles.ALL,
            db.roles.ALL,
            left=db.roles.on(db.user_roles.role_id == db.roles.id),
        )

        roles = []
        for assignment in assignments:
            roles.append(
                {
                    "assignment_id": assignment.user_roles.id,
                    "role_name": assignment.roles.name,
                    "role_display_name": assignment.roles.display_name,
                    "scope": assignment.user_roles.scope,
                    "resource_id": assignment.user_roles.resource_id,
                    "granted_at": assignment.user_roles.granted_at,
                    "granted_by": assignment.user_roles.granted_by,
                }
            )

        return roles
