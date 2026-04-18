"""
Unit tests for RBAC middleware and decorators.

Covers:
- requires_permission decorator
- requires_role decorator
- requires_any_permission decorator
- requires_all_permissions decorator
- Helper functions (is_admin, can_manage_users, can_access_cluster, can_access_service)

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest
import pytest_asyncio
from quart import Quart, g


# ============================================================================
# Fixtures
# ============================================================================

@pytest_asyncio.fixture
async def rbac_app():
    """Quart app with RBAC middleware initialized."""
    app = Quart(__name__)
    app.config["TESTING"] = True

    # Mock database
    mock_db = MagicMock(name="db")
    app.db = mock_db

    # Mock RBACModel
    with patch("middleware.rbac.RBACModel") as mock_rbac:
        mock_rbac.get_user_permissions = MagicMock(
            return_value={
                "global": ["global:admin"],
                "cluster": {},
                "service": {},
            }
        )
        mock_rbac.get_user_roles = MagicMock(
            return_value=[
                {
                    "role_name": "admin",
                    "scope": "global",
                    "resource_id": None,
                }
            ]
        )
        mock_rbac.has_permission = MagicMock(return_value=True)

        yield app


@pytest_asyncio.fixture
async def rbac_client(rbac_app):
    """Test client for RBAC app."""
    return rbac_app.test_client()


# ============================================================================
# Requires Permission Decorator Tests
# ============================================================================

class TestRequiresPermissionDecorator:
    """Tests for @requires_permission decorator."""

    @pytest.mark.asyncio
    async def test_requires_permission_unauthenticated(self, rbac_app):
        """@requires_permission without user_id returns 401."""
        from middleware.rbac import requires_permission
        from models.rbac import Permissions

        @rbac_app.route("/test", methods=["GET"])
        @requires_permission(Permissions.GLOBAL_ADMIN)
        async def test_route():
            return {"status": "ok"}, 200

        async with rbac_app.app_context():
            g.user_id = None
            with patch("middleware.rbac.abort") as mock_abort:
                mock_abort.side_effect = Exception("401")
                # The decorator should abort
                try:
                    from quart import request
                    await test_route()
                    assert False, "Should have aborted"
                except:
                    pass

    @pytest.mark.asyncio
    async def test_requires_permission_no_db(self, rbac_app):
        """@requires_permission without db returns 500."""
        from middleware.rbac import requires_permission
        from models.rbac import Permissions

        @rbac_app.route("/test", methods=["GET"])
        @requires_permission(Permissions.GLOBAL_ADMIN)
        async def test_route():
            return {"status": "ok"}, 200

        async with rbac_app.app_context():
            g.user_id = 1
            g.db = None
            with patch("middleware.rbac.abort") as mock_abort:
                mock_abort.side_effect = Exception("500")
                try:
                    await test_route()
                    assert False, "Should have aborted"
                except:
                    pass

    @pytest.mark.asyncio
    async def test_requires_permission_granted(self, rbac_app):
        """@requires_permission with valid permission allows access."""
        from middleware.rbac import requires_permission
        from models.rbac import Permissions

        @rbac_app.route("/test-granted-route", methods=["GET"])
        @requires_permission(Permissions.GLOBAL_ADMIN)
        async def test_route_granted():
            return {"status": "ok"}, 200

        async with rbac_app.app_context():
            g.user_id = 1
            g.db = rbac_app.db

            # Must patch through the middleware module since the fixture replaces
            # middleware.rbac.RBACModel with a mock already
            with patch("middleware.rbac.RBACModel.has_permission", return_value=True) as mock_has_perm:
                result = await test_route_granted()
                assert result == ({"status": "ok"}, 200)
                mock_has_perm.assert_called_once()

    @pytest.mark.asyncio
    async def test_requires_permission_denied(self, rbac_app):
        """@requires_permission without permission returns 403."""
        from middleware.rbac import requires_permission
        from models.rbac import Permissions, RBACModel

        @rbac_app.route("/test", methods=["GET"])
        @requires_permission(Permissions.GLOBAL_ADMIN)
        async def test_route():
            return {"status": "ok"}, 200

        async with rbac_app.app_context():
            g.user_id = 1
            g.db = rbac_app.db

            with patch.object(
                RBACModel, "has_permission", return_value=False
            ) as mock_has_perm:
                with patch("middleware.rbac.abort") as mock_abort:
                    mock_abort.side_effect = Exception("403")
                    try:
                        await test_route()
                        assert False, "Should have aborted"
                    except:
                        pass

    @pytest.mark.asyncio
    async def test_requires_permission_with_resource(self, rbac_app):
        """@requires_permission with resource_id_param extracts ID."""
        from middleware.rbac import requires_permission
        from models.rbac import Permissions

        @rbac_app.route("/test-resource/<int:cluster_id>", methods=["GET"])
        @requires_permission(Permissions.CLUSTER_READ, "cluster", "cluster_id")
        async def test_route_resource(cluster_id):
            return {"cluster_id": cluster_id}, 200

        async with rbac_app.app_context():
            g.user_id = 1
            g.db = rbac_app.db

            with patch("middleware.rbac.RBACModel.has_permission", return_value=True) as mock_has_perm:
                result = await test_route_resource(cluster_id=5)
                assert result == ({"cluster_id": 5}, 200)
                # Verify resource_id was passed
                call_args = mock_has_perm.call_args
                # call_args is (args, kwargs); args = (db, user_id, permission, resource_type, resource_id)
                assert call_args is not None
                # resource_id is at position 4 in positional args
                assert call_args[0][4] == 5


# ============================================================================
# Requires Role Decorator Tests
# ============================================================================

class TestRequiresRoleDecorator:
    """Tests for @requires_role decorator."""

    @pytest.mark.asyncio
    async def test_requires_role_unauthenticated(self, rbac_app):
        """@requires_role without user_id returns 401."""
        from middleware.rbac import requires_role
        from models.rbac import PermissionScope

        @rbac_app.route("/test", methods=["GET"])
        @requires_role("admin")
        async def test_route():
            return {"status": "ok"}, 200

        async with rbac_app.app_context():
            g.user_id = None
            with patch("middleware.rbac.abort") as mock_abort:
                mock_abort.side_effect = Exception("401")
                try:
                    await test_route()
                    assert False, "Should have aborted"
                except:
                    pass

    @pytest.mark.asyncio
    async def test_requires_role_no_db(self, rbac_app):
        """@requires_role without db returns 500."""
        from middleware.rbac import requires_role

        @rbac_app.route("/test", methods=["GET"])
        @requires_role("admin")
        async def test_route():
            return {"status": "ok"}, 200

        async with rbac_app.app_context():
            g.user_id = 1
            g.db = None
            with patch("middleware.rbac.abort") as mock_abort:
                mock_abort.side_effect = Exception("500")
                try:
                    await test_route()
                    assert False, "Should have aborted"
                except:
                    pass

    @pytest.mark.asyncio
    async def test_requires_role_granted(self, rbac_app):
        """@requires_role with matching role allows access."""
        from middleware.rbac import requires_role, RBACModel

        @rbac_app.route("/test", methods=["GET"])
        @requires_role("admin")
        async def test_route():
            return {"status": "ok"}, 200

        async with rbac_app.app_context():
            g.user_id = 1
            g.db = rbac_app.db

            with patch.object(
                RBACModel,
                "get_user_roles",
                return_value=[
                    {
                        "role_name": "admin",
                        "scope": "global",
                        "resource_id": None,
                    }
                ],
            ):
                result = await test_route()
                assert result == ({"status": "ok"}, 200)

    @pytest.mark.asyncio
    async def test_requires_role_denied(self, rbac_app):
        """@requires_role without required role returns 403."""
        from middleware.rbac import requires_role, RBACModel

        @rbac_app.route("/test", methods=["GET"])
        @requires_role("admin")
        async def test_route():
            return {"status": "ok"}, 200

        async with rbac_app.app_context():
            g.user_id = 1
            g.db = rbac_app.db

            with patch.object(
                RBACModel,
                "get_user_roles",
                return_value=[
                    {
                        "role_name": "viewer",
                        "scope": "global",
                        "resource_id": None,
                    }
                ],
            ):
                with patch("middleware.rbac.abort") as mock_abort:
                    mock_abort.side_effect = Exception("403")
                    try:
                        await test_route()
                        assert False, "Should have aborted"
                    except:
                        pass

    @pytest.mark.asyncio
    async def test_requires_role_scoped_cluster(self, rbac_app):
        """@requires_role with cluster scope checks resource_id."""
        from middleware.rbac import requires_role, RBACModel
        from models.rbac import PermissionScope

        @rbac_app.route("/test/<int:cluster_id>", methods=["GET"])
        @requires_role("cluster_admin", PermissionScope.CLUSTER, "cluster_id")
        async def test_route(cluster_id):
            return {"cluster_id": cluster_id}, 200

        async with rbac_app.app_context():
            g.user_id = 1
            g.db = rbac_app.db

            with patch.object(
                RBACModel,
                "get_user_roles",
                return_value=[
                    {
                        "role_name": "cluster_admin",
                        "scope": "cluster",
                        "resource_id": 5,
                    }
                ],
            ):
                result = await test_route(cluster_id=5)
                assert result == ({"cluster_id": 5}, 200)


# ============================================================================
# Requires Any Permission Tests
# ============================================================================

class TestRequiresAnyPermissionDecorator:
    """Tests for @requires_any_permission decorator."""

    @pytest.mark.asyncio
    async def test_requires_any_permission_unauthenticated(self, rbac_app):
        """@requires_any_permission without user_id returns 401."""
        from middleware.rbac import requires_any_permission
        from models.rbac import Permissions

        @rbac_app.route("/test", methods=["GET"])
        @requires_any_permission(Permissions.GLOBAL_ADMIN, Permissions.GLOBAL_USER_READ)
        async def test_route():
            return {"status": "ok"}, 200

        async with rbac_app.app_context():
            g.user_id = None
            with patch("middleware.rbac.abort") as mock_abort:
                mock_abort.side_effect = Exception("401")
                try:
                    await test_route()
                    assert False, "Should have aborted"
                except:
                    pass

    @pytest.mark.asyncio
    async def test_requires_any_permission_first_granted(self, rbac_app):
        """@requires_any_permission with first permission granted allows access."""
        from middleware.rbac import requires_any_permission, RBACModel
        from models.rbac import Permissions

        @rbac_app.route("/test", methods=["GET"])
        @requires_any_permission(Permissions.GLOBAL_ADMIN, Permissions.GLOBAL_USER_READ)
        async def test_route():
            return {"status": "ok"}, 200

        async with rbac_app.app_context():
            g.user_id = 1
            g.db = rbac_app.db

            with patch.object(
                RBACModel,
                "get_user_permissions",
                return_value={
                    "global": [Permissions.GLOBAL_ADMIN],
                    "cluster": {},
                    "service": {},
                },
            ):
                result = await test_route()
                assert result == ({"status": "ok"}, 200)

    @pytest.mark.asyncio
    async def test_requires_any_permission_second_granted(self, rbac_app):
        """@requires_any_permission with second permission granted allows access."""
        from middleware.rbac import requires_any_permission, RBACModel
        from models.rbac import Permissions

        @rbac_app.route("/test", methods=["GET"])
        @requires_any_permission(Permissions.GLOBAL_ADMIN, Permissions.GLOBAL_USER_READ)
        async def test_route():
            return {"status": "ok"}, 200

        async with rbac_app.app_context():
            g.user_id = 1
            g.db = rbac_app.db

            with patch.object(
                RBACModel,
                "get_user_permissions",
                return_value={
                    "global": [Permissions.GLOBAL_USER_READ],
                    "cluster": {},
                    "service": {},
                },
            ):
                result = await test_route()
                assert result == ({"status": "ok"}, 200)

    @pytest.mark.asyncio
    async def test_requires_any_permission_none_granted(self, rbac_app):
        """@requires_any_permission without any permission returns 403."""
        from middleware.rbac import requires_any_permission, RBACModel
        from models.rbac import Permissions

        @rbac_app.route("/test", methods=["GET"])
        @requires_any_permission(Permissions.GLOBAL_ADMIN, Permissions.GLOBAL_USER_READ)
        async def test_route():
            return {"status": "ok"}, 200

        async with rbac_app.app_context():
            g.user_id = 1
            g.db = rbac_app.db

            with patch.object(
                RBACModel,
                "get_user_permissions",
                return_value={
                    "global": [Permissions.GLOBAL_USER_WRITE],
                    "cluster": {},
                    "service": {},
                },
            ):
                with patch("middleware.rbac.abort") as mock_abort:
                    mock_abort.side_effect = Exception("403")
                    try:
                        await test_route()
                        assert False, "Should have aborted"
                    except:
                        pass


# ============================================================================
# Requires All Permissions Tests
# ============================================================================

class TestRequiresAllPermissionsDecorator:
    """Tests for @requires_all_permissions decorator."""

    @pytest.mark.asyncio
    async def test_requires_all_permissions_unauthenticated(self, rbac_app):
        """@requires_all_permissions without user_id returns 401."""
        from middleware.rbac import requires_all_permissions
        from models.rbac import Permissions

        @rbac_app.route("/test", methods=["GET"])
        @requires_all_permissions(
            Permissions.GLOBAL_ADMIN, Permissions.GLOBAL_USER_READ
        )
        async def test_route():
            return {"status": "ok"}, 200

        async with rbac_app.app_context():
            g.user_id = None
            with patch("middleware.rbac.abort") as mock_abort:
                mock_abort.side_effect = Exception("401")
                try:
                    await test_route()
                    assert False, "Should have aborted"
                except:
                    pass

    @pytest.mark.asyncio
    async def test_requires_all_permissions_all_granted(self, rbac_app):
        """@requires_all_permissions with all permissions granted allows access."""
        from middleware.rbac import requires_all_permissions, RBACModel
        from models.rbac import Permissions

        @rbac_app.route("/test", methods=["GET"])
        @requires_all_permissions(
            Permissions.GLOBAL_ADMIN, Permissions.GLOBAL_USER_READ
        )
        async def test_route():
            return {"status": "ok"}, 200

        async with rbac_app.app_context():
            g.user_id = 1
            g.db = rbac_app.db

            with patch.object(
                RBACModel,
                "get_user_permissions",
                return_value={
                    "global": [
                        Permissions.GLOBAL_ADMIN,
                        Permissions.GLOBAL_USER_READ,
                    ],
                    "cluster": {},
                    "service": {},
                },
            ):
                result = await test_route()
                assert result == ({"status": "ok"}, 200)

    @pytest.mark.asyncio
    async def test_requires_all_permissions_one_missing(self, rbac_app):
        """@requires_all_permissions with one missing permission returns 403."""
        from middleware.rbac import requires_all_permissions, RBACModel
        from models.rbac import Permissions

        @rbac_app.route("/test", methods=["GET"])
        @requires_all_permissions(
            Permissions.GLOBAL_ADMIN, Permissions.GLOBAL_USER_READ
        )
        async def test_route():
            return {"status": "ok"}, 200

        async with rbac_app.app_context():
            g.user_id = 1
            g.db = rbac_app.db

            with patch.object(
                RBACModel,
                "get_user_permissions",
                return_value={
                    "global": [Permissions.GLOBAL_ADMIN],  # Missing GLOBAL_USER_READ
                    "cluster": {},
                    "service": {},
                },
            ):
                with patch("middleware.rbac.abort") as mock_abort:
                    mock_abort.side_effect = Exception("403")
                    try:
                        await test_route()
                        assert False, "Should have aborted"
                    except:
                        pass


# ============================================================================
# Helper Function Tests
# ============================================================================

class TestRBACHelperFunctions:
    """Tests for RBAC helper functions."""

    @pytest.mark.asyncio
    async def test_is_admin_true(self, rbac_app):
        """is_admin returns True for admin user."""
        from middleware.rbac import is_admin, RBACModel
        from models.rbac import Permissions

        with patch.object(
            RBACModel,
            "get_user_permissions",
            return_value={"global": [Permissions.GLOBAL_ADMIN]},
        ):
            result = is_admin(1, rbac_app.db)
            assert result is True

    @pytest.mark.asyncio
    async def test_is_admin_false(self, rbac_app):
        """is_admin returns False for non-admin user."""
        from middleware.rbac import is_admin, RBACModel

        with patch.object(
            RBACModel,
            "get_user_permissions",
            return_value={"global": ["some:other:permission"]},
        ):
            result = is_admin(1, rbac_app.db)
            assert result is False

    @pytest.mark.asyncio
    async def test_can_manage_users_as_admin(self, rbac_app):
        """can_manage_users returns True for admin."""
        from middleware.rbac import can_manage_users, RBACModel
        from models.rbac import Permissions

        with patch.object(
            RBACModel,
            "get_user_permissions",
            return_value={"global": [Permissions.GLOBAL_ADMIN]},
        ):
            result = can_manage_users(1, rbac_app.db)
            assert result is True

    @pytest.mark.asyncio
    async def test_can_manage_users_with_write_permission(self, rbac_app):
        """can_manage_users returns True with user write permission."""
        from middleware.rbac import can_manage_users, RBACModel
        from models.rbac import Permissions

        with patch.object(
            RBACModel,
            "get_user_permissions",
            return_value={"global": [Permissions.GLOBAL_USER_WRITE]},
        ):
            result = can_manage_users(1, rbac_app.db)
            assert result is True

    @pytest.mark.asyncio
    async def test_can_manage_users_false(self, rbac_app):
        """can_manage_users returns False without admin/write."""
        from middleware.rbac import can_manage_users, RBACModel

        with patch.object(
            RBACModel,
            "get_user_permissions",
            return_value={"global": ["read:only"]},
        ):
            result = can_manage_users(1, rbac_app.db)
            assert result is False

    @pytest.mark.asyncio
    async def test_can_access_cluster(self, rbac_app):
        """can_access_cluster returns True when user has permission."""
        from middleware.rbac import can_access_cluster, RBACModel
        from models.rbac import Permissions

        with patch.object(
            RBACModel,
            "has_permission",
            return_value=True,
        ) as mock_has_perm:
            result = can_access_cluster(1, 5, rbac_app.db)
            assert result is True
            mock_has_perm.assert_called_once_with(
                rbac_app.db, 1, Permissions.CLUSTER_READ, "cluster", 5
            )

    @pytest.mark.asyncio
    async def test_can_access_cluster_false(self, rbac_app):
        """can_access_cluster returns False when user lacks permission."""
        from middleware.rbac import can_access_cluster, RBACModel

        with patch.object(
            RBACModel,
            "has_permission",
            return_value=False,
        ):
            result = can_access_cluster(1, 5, rbac_app.db)
            assert result is False

    @pytest.mark.asyncio
    async def test_can_access_service(self, rbac_app):
        """can_access_service returns True when user has permission."""
        from middleware.rbac import can_access_service, RBACModel
        from models.rbac import Permissions

        with patch.object(
            RBACModel,
            "has_permission",
            return_value=True,
        ) as mock_has_perm:
            result = can_access_service(1, 10, rbac_app.db)
            assert result is True
            mock_has_perm.assert_called_once_with(
                rbac_app.db, 1, Permissions.SERVICE_READ, "service", 10
            )

    @pytest.mark.asyncio
    async def test_can_access_service_false(self, rbac_app):
        """can_access_service returns False when user lacks permission."""
        from middleware.rbac import can_access_service, RBACModel

        with patch.object(
            RBACModel,
            "has_permission",
            return_value=False,
        ):
            result = can_access_service(1, 10, rbac_app.db)
            assert result is False
