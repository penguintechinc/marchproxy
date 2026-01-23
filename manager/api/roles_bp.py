"""
Role Management API Blueprint for MarchProxy

Provides RESTful API endpoints for managing roles and permissions.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import logging
from datetime import datetime
from typing import Dict, List

from quart import Blueprint, g, jsonify, request
from pydantic import BaseModel, Field, validator

from middleware.rbac import (
    requires_permission,
    requires_role,
    is_admin,
    can_manage_users,
)
from models.rbac import RBACModel, Permissions, PermissionScope, RoleType, DEFAULT_ROLES

logger = logging.getLogger(__name__)

# Create blueprint
roles_bp = Blueprint("roles", __name__, url_prefix="/api/v1/roles")


# Pydantic models for request validation
class RoleCreateRequest(BaseModel):
    """Request model for creating a role"""

    name: str = Field(..., min_length=1, max_length=50)
    display_name: str = Field(..., min_length=1, max_length=100)
    description: str = Field(default="")
    scope: str = Field(
        ..., pattern=f"^({'|'.join([s.value for s in PermissionScope])})$"
    )
    permissions: List[str] = Field(default=[])


class RoleUpdateRequest(BaseModel):
    """Request model for updating a role"""

    display_name: str = Field(None, min_length=1, max_length=100)
    description: str = None
    permissions: List[str] = None
    is_active: bool = None


class RoleAssignmentRequest(BaseModel):
    """Request model for assigning a role to a user"""

    user_id: int = Field(..., gt=0)
    role_name: str = Field(..., min_length=1)
    scope: str = Field(default="global")
    resource_id: int = Field(None, gt=0)

    @validator("scope")
    def validate_scope(cls, v):
        if v not in [s.value for s in PermissionScope]:
            raise ValueError(f"Invalid scope: {v}")
        return v


# Routes
@roles_bp.route("", methods=["GET"])
@requires_permission(Permissions.GLOBAL_ADMIN)
async def list_roles():
    """
    List all roles

    Requires: global:admin permission
    Returns: List of all roles
    """
    db = g.db

    roles = db(db.roles.is_active == True).select(orderby=db.roles.name)

    return (
        jsonify(
            {
                "roles": [
                    {
                        "id": role.id,
                        "name": role.name,
                        "display_name": role.display_name,
                        "description": role.description,
                        "scope": role.scope,
                        "permissions": role.permissions or [],
                        "is_system": role.is_system,
                        "created_at": (
                            role.created_at.isoformat() if role.created_at else None
                        ),
                    }
                    for role in roles
                ]
            }
        ),
        200,
    )


@roles_bp.route("/<int:role_id>", methods=["GET"])
@requires_permission(Permissions.GLOBAL_ADMIN)
async def get_role(role_id: int):
    """
    Get role details

    Requires: global:admin permission
    Returns: Role details
    """
    db = g.db

    role = db.roles[role_id]
    if not role or not role.is_active:
        return jsonify({"error": "Role not found"}), 404

    # Get users with this role
    assignments = db(
        (db.user_roles.role_id == role_id) & (db.user_roles.is_active == True)
    ).select(
        db.user_roles.ALL,
        db.users.id,
        db.users.username,
        db.users.email,
        left=db.users.on(db.user_roles.user_id == db.users.id),
    )

    return (
        jsonify(
            {
                "role": {
                    "id": role.id,
                    "name": role.name,
                    "display_name": role.display_name,
                    "description": role.description,
                    "scope": role.scope,
                    "permissions": role.permissions or [],
                    "is_system": role.is_system,
                    "created_at": (
                        role.created_at.isoformat() if role.created_at else None
                    ),
                    "updated_at": (
                        role.updated_at.isoformat() if role.updated_at else None
                    ),
                },
                "assignments": [
                    {
                        "user_id": a.users.id,
                        "username": a.users.username,
                        "email": a.users.email,
                        "scope": a.user_roles.scope,
                        "resource_id": a.user_roles.resource_id,
                        "granted_at": (
                            a.user_roles.granted_at.isoformat()
                            if a.user_roles.granted_at
                            else None
                        ),
                    }
                    for a in assignments
                ],
            }
        ),
        200,
    )


@roles_bp.route("", methods=["POST"])
@requires_permission(Permissions.GLOBAL_ADMIN)
async def create_role():
    """
    Create a new custom role

    Requires: global:admin permission
    Body: RoleCreateRequest
    Returns: Created role
    """
    data = await request.get_json()

    try:
        role_data = RoleCreateRequest(**data)
    except Exception as e:
        return jsonify({"error": f"Invalid request: {str(e)}"}), 400

    db = g.db

    # Check if role name already exists
    existing = db(db.roles.name == role_data.name).select().first()
    if existing:
        return jsonify({"error": "Role name already exists"}), 409

    # Create role
    role_id = db.roles.insert(
        name=role_data.name,
        display_name=role_data.display_name,
        description=role_data.description,
        scope=role_data.scope,
        permissions=role_data.permissions,
        is_system=False,
        is_active=True,
        created_at=datetime.utcnow(),
    )

    db.commit()

    role = db.roles[role_id]
    logger.info(f"Created custom role: {role_data.name} (ID: {role_id})")

    return (
        jsonify(
            {
                "role": {
                    "id": role.id,
                    "name": role.name,
                    "display_name": role.display_name,
                    "description": role.description,
                    "scope": role.scope,
                    "permissions": role.permissions or [],
                    "is_system": role.is_system,
                }
            }
        ),
        201,
    )


@roles_bp.route("/<int:role_id>", methods=["PUT"])
@requires_permission(Permissions.GLOBAL_ADMIN)
async def update_role(role_id: int):
    """
    Update role details

    Requires: global:admin permission
    Body: RoleUpdateRequest
    Returns: Updated role
    """
    data = await request.get_json()

    try:
        update_data = RoleUpdateRequest(**data)
    except Exception as e:
        return jsonify({"error": f"Invalid request: {str(e)}"}), 400

    db = g.db

    role = db.roles[role_id]
    if not role:
        return jsonify({"error": "Role not found"}), 404

    # Cannot modify system roles
    if role.is_system:
        return jsonify({"error": "Cannot modify system role"}), 403

    # Update role
    update_dict = update_data.dict(exclude_unset=True)
    if update_dict:
        update_dict["updated_at"] = datetime.utcnow()
        db(db.roles.id == role_id).update(**update_dict)

        # Invalidate permission cache for all users with this role
        assignments = db(
            (db.user_roles.role_id == role_id) & (db.user_roles.is_active == True)
        ).select(db.user_roles.user_id, distinct=True)

        for assignment in assignments:
            RBACModel.invalidate_permission_cache(db, assignment.user_id)

        db.commit()

        logger.info(f"Updated role: {role.name} (ID: {role_id})")

    role = db.roles[role_id]
    return (
        jsonify(
            {
                "role": {
                    "id": role.id,
                    "name": role.name,
                    "display_name": role.display_name,
                    "description": role.description,
                    "scope": role.scope,
                    "permissions": role.permissions or [],
                    "is_system": role.is_system,
                }
            }
        ),
        200,
    )


@roles_bp.route("/<int:role_id>", methods=["DELETE"])
@requires_permission(Permissions.GLOBAL_ADMIN)
async def delete_role(role_id: int):
    """
    Delete a custom role

    Requires: global:admin permission
    Returns: Success message
    """
    db = g.db

    role = db.roles[role_id]
    if not role:
        return jsonify({"error": "Role not found"}), 404

    # Cannot delete system roles
    if role.is_system:
        return jsonify({"error": "Cannot delete system role"}), 403

    # Deactivate role
    db(db.roles.id == role_id).update(is_active=False, updated_at=datetime.utcnow())

    # Deactivate all assignments
    db(db.user_roles.role_id == role_id).update(is_active=False)

    # Invalidate permission cache for affected users
    assignments = db(db.user_roles.role_id == role_id).select(
        db.user_roles.user_id, distinct=True
    )
    for assignment in assignments:
        RBACModel.invalidate_permission_cache(db, assignment.user_id)

    db.commit()

    logger.info(f"Deleted role: {role.name} (ID: {role_id})")

    return jsonify({"message": "Role deleted successfully"}), 200


@roles_bp.route("/assign", methods=["POST"])
@requires_permission(Permissions.GLOBAL_USER_WRITE)
async def assign_role():
    """
    Assign role to user

    Requires: global:users:write permission
    Body: RoleAssignmentRequest
    Returns: Assignment details
    """
    data = await request.get_json()

    try:
        assignment_data = RoleAssignmentRequest(**data)
    except Exception as e:
        return jsonify({"error": f"Invalid request: {str(e)}"}), 400

    db = g.db
    user_id = g.user_id

    # Verify user exists
    user = db.users[assignment_data.user_id]
    if not user or not user.is_active:
        return jsonify({"error": "User not found"}), 404

    # Assign role
    try:
        scope = PermissionScope(assignment_data.scope)
        assignment_id = RBACModel.assign_role(
            db,
            assignment_data.user_id,
            assignment_data.role_name,
            scope,
            assignment_data.resource_id,
            granted_by=user_id,
        )

        logger.info(
            f"Assigned role {assignment_data.role_name} to user {assignment_data.user_id} "
            f"(scope: {assignment_data.scope}, resource: {assignment_data.resource_id})"
        )

        return (
            jsonify(
                {
                    "message": "Role assigned successfully",
                    "assignment_id": assignment_id,
                }
            ),
            201,
        )

    except ValueError as e:
        return jsonify({"error": str(e)}), 400


@roles_bp.route("/revoke", methods=["POST"])
@requires_permission(Permissions.GLOBAL_USER_WRITE)
async def revoke_role():
    """
    Revoke role from user

    Requires: global:users:write permission
    Body: {user_id, role_name, resource_id}
    Returns: Success message
    """
    data = await request.get_json()

    user_id = data.get("user_id")
    role_name = data.get("role_name")
    resource_id = data.get("resource_id")

    if not user_id or not role_name:
        return jsonify({"error": "user_id and role_name required"}), 400

    db = g.db

    try:
        RBACModel.revoke_role(db, user_id, role_name, resource_id)

        logger.info(
            f"Revoked role {role_name} from user {user_id} "
            f"(resource: {resource_id})"
        )

        return jsonify({"message": "Role revoked successfully"}), 200

    except ValueError as e:
        return jsonify({"error": str(e)}), 400


@roles_bp.route("/user/<int:user_id>", methods=["GET"])
@requires_permission(Permissions.GLOBAL_USER_READ)
async def get_user_roles(user_id: int):
    """
    Get all roles assigned to a user

    Requires: global:users:read permission
    Returns: User roles and permissions
    """
    db = g.db

    # Verify user exists
    user = db.users[user_id]
    if not user:
        return jsonify({"error": "User not found"}), 404

    # Get roles
    roles = RBACModel.get_user_roles(db, user_id)

    # Get permissions
    permissions = RBACModel.get_user_permissions(db, user_id)

    return (
        jsonify(
            {
                "user_id": user_id,
                "username": user.username,
                "email": user.email,
                "roles": roles,
                "permissions": permissions,
            }
        ),
        200,
    )


@roles_bp.route("/permissions", methods=["GET"])
async def list_available_permissions():
    """
    List all available permissions

    Returns: List of all permission scopes
    """
    # Get all permission attributes from Permissions class
    all_permissions = [
        getattr(Permissions, attr)
        for attr in dir(Permissions)
        if not attr.startswith("_") and isinstance(getattr(Permissions, attr), str)
    ]

    return (
        jsonify(
            {
                "permissions": all_permissions,
                "scopes": {
                    "global": [p for p in all_permissions if p.startswith("global:")],
                    "cluster": [p for p in all_permissions if p.startswith("cluster:")],
                    "service": [p for p in all_permissions if p.startswith("service:")],
                },
            }
        ),
        200,
    )
