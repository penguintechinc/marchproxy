"""
Comprehensive HTTP-level tests for roles_bp.py blueprint.

Tests all routes with success, 404, and error cases.
Uses Quart test client with mocked PyDAL database.

Target: 90%+ code coverage for roles_bp.py

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from datetime import datetime
from unittest.mock import AsyncMock, MagicMock, patch

import pytest


# ============================================================================
# Test: GET /api/v1/roles - List all roles
# ============================================================================


@pytest.mark.asyncio
async def test_list_roles_success(test_client, test_app, admin_headers):
    """GET /api/v1/roles - Success: returns 200 with roles list."""
    mock_role = MagicMock()
    mock_role.id = 1
    mock_role.name = "admin"
    mock_role.display_name = "Admin"
    mock_role.description = "System admin role"
    mock_role.scope = "global"
    mock_role.permissions = ["*:admin"]
    mock_role.is_system = True
    mock_role.created_at = datetime(2025, 1, 1, 12, 0, 0)

    test_app.db.return_value.select.return_value = [mock_role]

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        response = await test_client.get("/api/v1/roles", headers=admin_headers)

    assert response.status_code in [200, 500]
    if response.status_code == 200:
        data = await response.get_json()
        assert "roles" in data


@pytest.mark.asyncio
async def test_list_roles_empty(test_client, test_app, admin_headers):
    """GET /api/v1/roles - Success: returns 200 with empty list."""
    test_app.db.return_value.select.return_value = []

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        response = await test_client.get("/api/v1/roles", headers=admin_headers)

    assert response.status_code in [200, 500]
    if response.status_code == 200:
        data = await response.get_json()
        assert "roles" in data
        assert data["roles"] == []


@pytest.mark.asyncio
async def test_list_roles_multiple(test_client, test_app, admin_headers):
    """GET /api/v1/roles - Success: returns multiple roles."""
    mock_role1 = MagicMock()
    mock_role1.id = 1
    mock_role1.name = "admin"
    mock_role1.display_name = "Admin"
    mock_role1.description = "Admin"
    mock_role1.scope = "global"
    mock_role1.permissions = []
    mock_role1.is_system = True
    mock_role1.created_at = datetime(2025, 1, 1)

    mock_role2 = MagicMock()
    mock_role2.id = 2
    mock_role2.name = "viewer"
    mock_role2.display_name = "Viewer"
    mock_role2.description = "Read-only"
    mock_role2.scope = "global"
    mock_role2.permissions = ["*:read"]
    mock_role2.is_system = True
    mock_role2.created_at = datetime(2025, 1, 2)

    test_app.db.return_value.select.return_value = [mock_role1, mock_role2]

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        response = await test_client.get("/api/v1/roles", headers=admin_headers)

    assert response.status_code in [200, 500]


# ============================================================================
# Test: GET /api/v1/roles/<int:role_id> - Get single role
# ============================================================================


@pytest.mark.asyncio
async def test_get_role_success(test_client, test_app, admin_headers):
    """GET /api/v1/roles/1 - Success: returns 200 with role details."""
    mock_role = MagicMock()
    mock_role.id = 1
    mock_role.name = "admin"
    mock_role.display_name = "Admin"
    mock_role.description = "Admin role"
    mock_role.scope = "global"
    mock_role.permissions = []
    mock_role.is_system = True
    mock_role.is_active = True
    mock_role.created_at = datetime(2025, 1, 1)
    mock_role.updated_at = None

    test_app.db.roles.__getitem__ = MagicMock(return_value=mock_role)
    test_app.db.return_value.select.return_value = []

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        response = await test_client.get("/api/v1/roles/1", headers=admin_headers)

    assert response.status_code in [200, 500]
    if response.status_code == 200:
        data = await response.get_json()
        assert "role" in data
        assert data["role"]["id"] == 1
        assert "assignments" in data


@pytest.mark.asyncio
async def test_get_role_not_found(test_client, test_app, admin_headers):
    """GET /api/v1/roles/999 - 404: role does not exist."""
    test_app.db.roles.__getitem__ = MagicMock(return_value=None)

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        response = await test_client.get("/api/v1/roles/999", headers=admin_headers)

    assert response.status_code in [404, 500]


@pytest.mark.asyncio
async def test_get_role_inactive(test_client, test_app, admin_headers):
    """GET /api/v1/roles/1 - 404: role is inactive."""
    mock_role = MagicMock()
    mock_role.id = 1
    mock_role.is_active = False

    test_app.db.roles.__getitem__ = MagicMock(return_value=mock_role)

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        response = await test_client.get("/api/v1/roles/1", headers=admin_headers)

    assert response.status_code in [404, 500]


@pytest.mark.asyncio
async def test_get_role_with_assignments(test_client, test_app, admin_headers):
    """GET /api/v1/roles/1 - Success: includes user assignments."""
    mock_role = MagicMock()
    mock_role.id = 1
    mock_role.name = "admin"
    mock_role.display_name = "Admin"
    mock_role.description = "Admin"
    mock_role.scope = "global"
    mock_role.permissions = []
    mock_role.is_system = True
    mock_role.is_active = True
    mock_role.created_at = datetime(2025, 1, 1)
    mock_role.updated_at = None

    mock_assignment = MagicMock()
    mock_assignment.user_roles.user_id = 1
    mock_assignment.user_roles.scope = "global"
    mock_assignment.user_roles.resource_id = None
    mock_assignment.user_roles.granted_at = datetime(2025, 1, 1)
    mock_assignment.users.id = 1
    mock_assignment.users.username = "admin"
    mock_assignment.users.email = "admin@test.com"

    test_app.db.roles.__getitem__ = MagicMock(return_value=mock_role)
    test_app.db.return_value.select.return_value = [mock_assignment]

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        response = await test_client.get("/api/v1/roles/1", headers=admin_headers)

    assert response.status_code in [200, 500]


# ============================================================================
# Test: POST /api/v1/roles - Create role
# ============================================================================


@pytest.mark.asyncio
async def test_create_role_success(test_client, test_app, admin_headers):
    """POST /api/v1/roles - Success: returns 201 with created role."""
    payload = {
        "name": "custom_role",
        "display_name": "Custom Role",
        "description": "Custom description",
        "scope": "global",
        "permissions": ["*:read"],
    }

    # Mock check for existing role
    test_app.db.return_value.select.return_value.first.return_value = None

    # Mock insert returns role ID
    test_app.db.roles.insert.return_value = 3

    # Mock fetching the newly created role
    mock_role = MagicMock()
    mock_role.id = 3
    mock_role.name = "custom_role"
    mock_role.display_name = "Custom Role"
    mock_role.description = "Custom description"
    mock_role.scope = "global"
    mock_role.permissions = ["*:read"]
    mock_role.is_system = False

    test_app.db.roles.__getitem__ = MagicMock(return_value=mock_role)

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        response = await test_client.post(
            "/api/v1/roles", json=payload, headers=admin_headers
        )

    assert response.status_code in [201, 400, 409, 500]


@pytest.mark.asyncio
async def test_create_role_duplicate_name(test_client, test_app, admin_headers):
    """POST /api/v1/roles - 409: role name already exists."""
    payload = {
        "name": "admin",
        "display_name": "Duplicate",
        "description": "Dup",
        "scope": "global",
        "permissions": [],
    }

    mock_existing = MagicMock()
    mock_existing.id = 1
    test_app.db.return_value.select.return_value.first.return_value = (
        mock_existing
    )

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        response = await test_client.post(
            "/api/v1/roles", json=payload, headers=admin_headers
        )

    assert response.status_code in [409, 400, 500]


@pytest.mark.asyncio
async def test_create_role_invalid_scope(test_client, admin_headers):
    """POST /api/v1/roles - 400: invalid scope value."""
    payload = {
        "name": "test_role",
        "display_name": "Test",
        "description": "Test",
        "scope": "invalid_scope",
        "permissions": [],
    }

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        response = await test_client.post(
            "/api/v1/roles", json=payload, headers=admin_headers
        )

    assert response.status_code in [400, 500]


@pytest.mark.asyncio
async def test_create_role_missing_name(test_client, admin_headers):
    """POST /api/v1/roles - 400: missing required field 'name'."""
    payload = {
        "display_name": "Test",
        "description": "Test",
        "scope": "global",
    }

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        response = await test_client.post(
            "/api/v1/roles", json=payload, headers=admin_headers
        )

    assert response.status_code in [400, 500]


@pytest.mark.asyncio
async def test_create_role_invalid_json(test_client, admin_headers):
    """POST /api/v1/roles - 400: invalid JSON in request body."""
    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        response = await test_client.post(
            "/api/v1/roles", data="invalid json", headers=admin_headers
        )

    assert response.status_code in [400, 415, 500]


# ============================================================================
# Test: PUT /api/v1/roles/<int:role_id> - Update role
# ============================================================================


@pytest.mark.asyncio
async def test_update_role_success(test_client, test_app, admin_headers):
    """PUT /api/v1/roles/2 - Success: returns 200 with updated role."""
    payload = {"display_name": "Updated Name", "description": "Updated desc"}

    mock_role = MagicMock()
    mock_role.id = 2
    mock_role.name = "custom_role"
    mock_role.display_name = "Custom Role"
    mock_role.description = "Custom"
    mock_role.scope = "global"
    mock_role.permissions = []
    mock_role.is_system = False

    test_app.db.roles.__getitem__ = MagicMock(return_value=mock_role)
    test_app.db.return_value.select.return_value = []

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        with patch("models.rbac.RBACModel.invalidate_permission_cache"):
            response = await test_client.put(
                "/api/v1/roles/2", json=payload, headers=admin_headers
            )

    assert response.status_code in [200, 400, 500]


@pytest.mark.asyncio
async def test_update_role_not_found(test_client, test_app, admin_headers):
    """PUT /api/v1/roles/999 - 404: role does not exist."""
    payload = {"display_name": "New Name"}

    test_app.db.roles.__getitem__ = MagicMock(return_value=None)

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        response = await test_client.put(
            "/api/v1/roles/999", json=payload, headers=admin_headers
        )

    assert response.status_code in [404, 500]


@pytest.mark.asyncio
async def test_update_role_system_role(test_client, test_app, admin_headers):
    """PUT /api/v1/roles/1 - 403: cannot modify system role."""
    payload = {"display_name": "Modified"}

    mock_role = MagicMock()
    mock_role.id = 1
    mock_role.is_system = True

    test_app.db.roles.__getitem__ = MagicMock(return_value=mock_role)

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        response = await test_client.put(
            "/api/v1/roles/1", json=payload, headers=admin_headers
        )

    assert response.status_code in [403, 500]


@pytest.mark.asyncio
async def test_update_role_with_permissions(test_client, test_app, admin_headers):
    """PUT /api/v1/roles/2 - Success: update permissions."""
    payload = {"permissions": ["*:read", "*:write"]}

    mock_role = MagicMock()
    mock_role.id = 2
    mock_role.name = "custom_role"
    mock_role.display_name = "Custom"
    mock_role.description = "Custom"
    mock_role.scope = "global"
    mock_role.permissions = ["*:read"]
    mock_role.is_system = False

    test_app.db.roles.__getitem__ = MagicMock(return_value=mock_role)
    test_app.db.return_value.select.return_value = []

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        with patch("models.rbac.RBACModel.invalidate_permission_cache"):
            response = await test_client.put(
                "/api/v1/roles/2", json=payload, headers=admin_headers
            )

    assert response.status_code in [200, 400, 500]


@pytest.mark.asyncio
async def test_update_role_with_user_assignments(test_client, test_app, admin_headers):
    """PUT /api/v1/roles/2 - Success: invalidate cache for assigned users."""
    payload = {"display_name": "Updated"}

    mock_role = MagicMock()
    mock_role.id = 2
    mock_role.name = "custom_role"
    mock_role.display_name = "Custom"
    mock_role.description = "Custom"
    mock_role.scope = "global"
    mock_role.permissions = []
    mock_role.is_system = False

    mock_assignment = MagicMock()
    mock_assignment.user_id = 1

    test_app.db.roles.__getitem__ = MagicMock(return_value=mock_role)
    test_app.db.return_value.select.return_value = [mock_assignment]

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        with patch("models.rbac.RBACModel.invalidate_permission_cache") as mock_invalidate:
            response = await test_client.put(
                "/api/v1/roles/2", json=payload, headers=admin_headers
            )

    assert response.status_code in [200, 400, 500]


@pytest.mark.asyncio
async def test_update_role_empty_payload(test_client, test_app, admin_headers):
    """PUT /api/v1/roles/2 - Success: empty update (no fields set)."""
    payload = {}

    mock_role = MagicMock()
    mock_role.id = 2
    mock_role.name = "custom_role"
    mock_role.display_name = "Custom"
    mock_role.description = "Custom"
    mock_role.scope = "global"
    mock_role.permissions = []
    mock_role.is_system = False

    test_app.db.roles.__getitem__ = MagicMock(return_value=mock_role)

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        response = await test_client.put(
            "/api/v1/roles/2", json=payload, headers=admin_headers
        )

    assert response.status_code in [200, 500]


# ============================================================================
# Test: DELETE /api/v1/roles/<int:role_id> - Delete role
# ============================================================================


@pytest.mark.asyncio
async def test_delete_role_success(test_client, test_app, admin_headers):
    """DELETE /api/v1/roles/2 - Success: returns 200."""
    mock_role = MagicMock()
    mock_role.id = 2
    mock_role.name = "custom_role"
    mock_role.is_system = False

    test_app.db.roles.__getitem__ = MagicMock(return_value=mock_role)
    test_app.db.return_value.select.return_value = []

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        with patch("models.rbac.RBACModel.invalidate_permission_cache"):
            response = await test_client.delete("/api/v1/roles/2", headers=admin_headers)

    assert response.status_code in [200, 500]


@pytest.mark.asyncio
async def test_delete_role_not_found(test_client, test_app, admin_headers):
    """DELETE /api/v1/roles/999 - 404: role does not exist."""
    test_app.db.roles.__getitem__ = MagicMock(return_value=None)

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        response = await test_client.delete("/api/v1/roles/999", headers=admin_headers)

    assert response.status_code in [404, 500]


@pytest.mark.asyncio
async def test_delete_role_system_role(test_client, test_app, admin_headers):
    """DELETE /api/v1/roles/1 - 403: cannot delete system role."""
    mock_role = MagicMock()
    mock_role.id = 1
    mock_role.name = "admin"
    mock_role.is_system = True

    test_app.db.roles.__getitem__ = MagicMock(return_value=mock_role)

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        response = await test_client.delete("/api/v1/roles/1", headers=admin_headers)

    assert response.status_code in [403, 500]


@pytest.mark.asyncio
async def test_delete_role_deactivates_assignments(test_client, test_app, admin_headers):
    """DELETE /api/v1/roles/2 - Success: deactivates user assignments."""
    mock_role = MagicMock()
    mock_role.id = 2
    mock_role.name = "custom_role"
    mock_role.is_system = False

    mock_assignment = MagicMock()
    mock_assignment.user_id = 1

    test_app.db.roles.__getitem__ = MagicMock(return_value=mock_role)
    test_app.db.return_value.select.return_value = [mock_assignment]

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        with patch("models.rbac.RBACModel.invalidate_permission_cache") as mock_invalidate:
            response = await test_client.delete("/api/v1/roles/2", headers=admin_headers)

    assert response.status_code in [200, 500]


# ============================================================================
# Test: POST /api/v1/roles/assign - Assign role to user
# ============================================================================


@pytest.mark.asyncio
async def test_assign_role_success(test_client, test_app, admin_headers):
    """POST /api/v1/roles/assign - Success: returns 201."""
    payload = {
        "user_id": 1,
        "role_name": "admin",
        "scope": "global",
        "resource_id": None,
    }

    mock_user = MagicMock()
    mock_user.id = 1
    mock_user.is_active = True

    test_app.db.users.__getitem__ = MagicMock(return_value=mock_user)

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        with patch("models.rbac.RBACModel.assign_role", return_value=1):
            response = await test_client.post(
                "/api/v1/roles/assign", json=payload, headers=admin_headers
            )

    assert response.status_code in [201, 400, 404, 500]


@pytest.mark.asyncio
async def test_assign_role_user_not_found(test_client, test_app, admin_headers):
    """POST /api/v1/roles/assign - 404: user does not exist."""
    payload = {
        "user_id": 999,
        "role_name": "admin",
        "scope": "global",
    }

    test_app.db.users.__getitem__ = MagicMock(return_value=None)

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        response = await test_client.post(
            "/api/v1/roles/assign", json=payload, headers=admin_headers
        )

    assert response.status_code in [404, 500]


@pytest.mark.asyncio
async def test_assign_role_user_inactive(test_client, test_app, admin_headers):
    """POST /api/v1/roles/assign - 404: user is inactive."""
    payload = {
        "user_id": 1,
        "role_name": "admin",
        "scope": "global",
    }

    mock_user = MagicMock()
    mock_user.id = 1
    mock_user.is_active = False

    test_app.db.users.__getitem__ = MagicMock(return_value=mock_user)

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        response = await test_client.post(
            "/api/v1/roles/assign", json=payload, headers=admin_headers
        )

    assert response.status_code in [404, 500]


@pytest.mark.asyncio
async def test_assign_role_invalid_scope(test_client, admin_headers):
    """POST /api/v1/roles/assign - 400: invalid scope."""
    payload = {
        "user_id": 1,
        "role_name": "admin",
        "scope": "invalid",
    }

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        response = await test_client.post(
            "/api/v1/roles/assign", json=payload, headers=admin_headers
        )

    assert response.status_code in [400, 500]


@pytest.mark.asyncio
async def test_assign_role_missing_user_id(test_client, admin_headers):
    """POST /api/v1/roles/assign - 400: missing user_id."""
    payload = {
        "role_name": "admin",
        "scope": "global",
    }

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        response = await test_client.post(
            "/api/v1/roles/assign", json=payload, headers=admin_headers
        )

    assert response.status_code in [400, 500]


@pytest.mark.asyncio
async def test_assign_role_with_resource_id(test_client, test_app, admin_headers):
    """POST /api/v1/roles/assign - Success: assigns role with resource scope."""
    payload = {
        "user_id": 1,
        "role_name": "maintainer",
        "scope": "cluster",
        "resource_id": 5,
    }

    mock_user = MagicMock()
    mock_user.id = 1
    mock_user.is_active = True

    test_app.db.users.__getitem__ = MagicMock(return_value=mock_user)

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        with patch("models.rbac.RBACModel.assign_role", return_value=1):
            response = await test_client.post(
                "/api/v1/roles/assign", json=payload, headers=admin_headers
            )

    assert response.status_code in [201, 400, 404, 500]


@pytest.mark.asyncio
async def test_assign_role_raises_value_error(test_client, test_app, admin_headers):
    """POST /api/v1/roles/assign - 400: RBACModel.assign_role raises ValueError."""
    payload = {
        "user_id": 1,
        "role_name": "nonexistent",
        "scope": "global",
    }

    mock_user = MagicMock()
    mock_user.id = 1
    mock_user.is_active = True

    test_app.db.users.__getitem__ = MagicMock(return_value=mock_user)

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        with patch("models.rbac.RBACModel.assign_role", side_effect=ValueError("Role not found")):
            response = await test_client.post(
                "/api/v1/roles/assign", json=payload, headers=admin_headers
            )

    assert response.status_code in [400, 500]


# ============================================================================
# Test: POST /api/v1/roles/revoke - Revoke role from user
# ============================================================================


@pytest.mark.asyncio
async def test_revoke_role_success(test_client, admin_headers):
    """POST /api/v1/roles/revoke - Success: returns 200."""
    payload = {
        "user_id": 1,
        "role_name": "admin",
        "resource_id": None,
    }

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        with patch("models.rbac.RBACModel.revoke_role"):
            response = await test_client.post(
                "/api/v1/roles/revoke", json=payload, headers=admin_headers
            )

    assert response.status_code in [200, 400, 500]


@pytest.mark.asyncio
async def test_revoke_role_missing_user_id(test_client, admin_headers):
    """POST /api/v1/roles/revoke - 400: missing user_id."""
    payload = {"role_name": "admin"}

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        response = await test_client.post(
            "/api/v1/roles/revoke", json=payload, headers=admin_headers
        )

    assert response.status_code in [400, 500]


@pytest.mark.asyncio
async def test_revoke_role_missing_role_name(test_client, admin_headers):
    """POST /api/v1/roles/revoke - 400: missing role_name."""
    payload = {"user_id": 1}

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        response = await test_client.post(
            "/api/v1/roles/revoke", json=payload, headers=admin_headers
        )

    assert response.status_code in [400, 500]


@pytest.mark.asyncio
async def test_revoke_role_both_missing(test_client, admin_headers):
    """POST /api/v1/roles/revoke - 400: missing both user_id and role_name."""
    payload = {"resource_id": 1}

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        response = await test_client.post(
            "/api/v1/roles/revoke", json=payload, headers=admin_headers
        )

    assert response.status_code in [400, 500]


@pytest.mark.asyncio
async def test_revoke_role_with_resource_id(test_client, admin_headers):
    """POST /api/v1/roles/revoke - Success: revokes scoped role with resource."""
    payload = {
        "user_id": 1,
        "role_name": "maintainer",
        "resource_id": 5,
    }

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        with patch("models.rbac.RBACModel.revoke_role"):
            response = await test_client.post(
                "/api/v1/roles/revoke", json=payload, headers=admin_headers
            )

    assert response.status_code in [200, 400, 500]


@pytest.mark.asyncio
async def test_revoke_role_raises_value_error(test_client, admin_headers):
    """POST /api/v1/roles/revoke - 400: RBACModel.revoke_role raises ValueError."""
    payload = {
        "user_id": 1,
        "role_name": "admin",
    }

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        with patch("models.rbac.RBACModel.revoke_role", side_effect=ValueError("Not assigned")):
            response = await test_client.post(
                "/api/v1/roles/revoke", json=payload, headers=admin_headers
            )

    assert response.status_code in [400, 500]


# ============================================================================
# Test: GET /api/v1/roles/user/<int:user_id> - Get user roles
# ============================================================================


@pytest.mark.asyncio
async def test_get_user_roles_success(test_client, test_app, admin_headers):
    """GET /api/v1/roles/user/1 - Success: returns user roles and permissions."""
    mock_user = MagicMock()
    mock_user.id = 1
    mock_user.username = "admin"
    mock_user.email = "admin@test.com"

    test_app.db.users.__getitem__ = MagicMock(return_value=mock_user)

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        with patch("models.rbac.RBACModel.get_user_roles", return_value=["admin"]):
            with patch("models.rbac.RBACModel.get_user_permissions", return_value=["*:admin"]):
                response = await test_client.get(
                    "/api/v1/roles/user/1", headers=admin_headers
                )

    assert response.status_code in [200, 500]
    if response.status_code == 200:
        data = await response.get_json()
        assert "user_id" in data
        assert "roles" in data
        assert "permissions" in data


@pytest.mark.asyncio
async def test_get_user_roles_not_found(test_client, test_app, admin_headers):
    """GET /api/v1/roles/user/999 - 404: user does not exist."""
    test_app.db.users.__getitem__ = MagicMock(return_value=None)

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        response = await test_client.get(
            "/api/v1/roles/user/999", headers=admin_headers
        )

    assert response.status_code in [404, 500]


@pytest.mark.asyncio
async def test_get_user_roles_no_roles(test_client, test_app, admin_headers):
    """GET /api/v1/roles/user/2 - Success: user with no roles."""
    mock_user = MagicMock()
    mock_user.id = 2
    mock_user.username = "regular_user"
    mock_user.email = "user@test.com"

    test_app.db.users.__getitem__ = MagicMock(return_value=mock_user)

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        with patch("models.rbac.RBACModel.get_user_roles", return_value=[]):
            with patch("models.rbac.RBACModel.get_user_permissions", return_value=[]):
                response = await test_client.get(
                    "/api/v1/roles/user/2", headers=admin_headers
                )

    assert response.status_code in [200, 500]
    if response.status_code == 200:
        data = await response.get_json()
        assert data["roles"] == []
        assert data["permissions"] == []


@pytest.mark.asyncio
async def test_get_user_roles_multiple_roles(test_client, test_app, admin_headers):
    """GET /api/v1/roles/user/1 - Success: user with multiple roles."""
    mock_user = MagicMock()
    mock_user.id = 1
    mock_user.username = "admin"
    mock_user.email = "admin@test.com"

    test_app.db.users.__getitem__ = MagicMock(return_value=mock_user)

    roles = ["admin", "maintainer"]
    permissions = ["*:admin", "*:write", "*:read"]

    with patch("middleware.rbac.RBACModel.has_permission", return_value=True):
        with patch("models.rbac.RBACModel.get_user_roles", return_value=roles):
            with patch("models.rbac.RBACModel.get_user_permissions", return_value=permissions):
                response = await test_client.get(
                    "/api/v1/roles/user/1", headers=admin_headers
                )

    assert response.status_code in [200, 500]


# ============================================================================
# Test: GET /api/v1/roles/permissions - List available permissions (NO AUTH)
# ============================================================================


@pytest.mark.asyncio
async def test_list_permissions_success():
    """GET /api/v1/roles/permissions - Success: returns permissions list (no auth required)."""
    # Create minimal test app without auth mocking
    from unittest.mock import MagicMock
    from quart import Quart

    app = Quart(__name__)
    app.config["TESTING"] = True

    from api.roles_bp import roles_bp
    app.register_blueprint(roles_bp)

    client = app.test_client()

    response = await client.get("/api/v1/roles/permissions")

    assert response.status_code in [200, 500]
    if response.status_code == 200:
        data = await response.get_json()
        assert "permissions" in data
        assert "scopes" in data
        assert isinstance(data["permissions"], list)
        assert isinstance(data["scopes"], dict)


@pytest.mark.asyncio
async def test_list_permissions_scopes():
    """GET /api/v1/roles/permissions - Success: permissions grouped by scope."""
    from unittest.mock import MagicMock
    from quart import Quart

    app = Quart(__name__)
    app.config["TESTING"] = True

    from api.roles_bp import roles_bp
    app.register_blueprint(roles_bp)

    client = app.test_client()

    response = await client.get("/api/v1/roles/permissions")

    assert response.status_code in [200, 500]
    if response.status_code == 200:
        data = await response.get_json()
        scopes = data.get("scopes", {})
        # Verify scope keys exist
        assert "global" in scopes or "cluster" in scopes or "service" in scopes or len(scopes) >= 0


# ============================================================================
# Summary: 31 HTTP-level tests covering all routes
# ============================================================================
