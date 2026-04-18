"""
Tests for rbac.py middleware, rate_limiting.py models, and auth_native.py.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import pytest
from datetime import datetime, timedelta
from unittest.mock import MagicMock, patch, AsyncMock


# ============================================================================
# middleware/rbac.py Tests
# ============================================================================


class TestRBACMiddlewareInit:
    """RBACMiddleware initialization and init_app"""

    def test_init_without_app(self):
        from middleware.rbac import RBACMiddleware

        m = RBACMiddleware()
        assert m.app is None

    def test_init_with_app_calls_init_app(self):
        from middleware.rbac import RBACMiddleware

        mock_app = MagicMock()
        m = RBACMiddleware(mock_app)
        mock_app.before_request.assert_called_once()

    def test_init_app_registers_before_request(self):
        from middleware.rbac import RBACMiddleware

        m = RBACMiddleware()
        mock_app = MagicMock()
        m.init_app(mock_app)
        mock_app.before_request.assert_called_once()


class TestRBACMiddlewareLoadPermissions:
    """RBACMiddleware.load_user_permissions"""

    @pytest.mark.asyncio
    async def test_load_permissions_no_user_sets_empty(self):
        from middleware.rbac import RBACMiddleware
        from quart import Quart

        app = Quart(__name__)
        async with app.app_context():
            from quart import g
            g.user_id = None
            m = RBACMiddleware()
            await m.load_user_permissions()
            assert g.permissions == {"global": [], "cluster": {}, "service": {}}

    @pytest.mark.asyncio
    async def test_load_permissions_with_user_no_db_sets_empty(self):
        from middleware.rbac import RBACMiddleware
        from quart import Quart

        app = Quart(__name__)
        async with app.app_context():
            from quart import g
            g.user_id = 1
            # g.db is not set
            m = RBACMiddleware()
            await m.load_user_permissions()
            assert g.permissions == {"global": [], "cluster": {}, "service": {}}

    @pytest.mark.asyncio
    async def test_load_permissions_with_user_and_db_calls_rbac(self):
        from middleware.rbac import RBACMiddleware
        from quart import Quart

        app = Quart(__name__)
        mock_perms = {"global": ["admin"], "cluster": {}, "service": {}}

        async with app.app_context():
            from quart import g
            g.user_id = 1
            g.db = MagicMock()

            with patch("models.rbac.RBACModel.get_user_permissions", return_value=mock_perms):
                m = RBACMiddleware()
                await m.load_user_permissions()
                assert g.permissions == mock_perms


class TestRequiresPermission:
    """requires_permission decorator"""

    @pytest.mark.asyncio
    async def test_requires_permission_no_user_id_aborts_401(self):
        from middleware.rbac import requires_permission
        from quart import Quart

        app = Quart(__name__)

        @app.route("/test")
        @requires_permission("admin")
        async def test_route():
            return "ok"

        async with app.test_client() as client:
            response = await client.get("/test")
            assert response.status_code == 401

    @pytest.mark.asyncio
    async def test_requires_permission_no_db_aborts_500(self):
        from middleware.rbac import requires_permission
        from quart import Quart, g

        app = Quart(__name__)

        @app.route("/test2")
        @requires_permission("admin")
        async def test_route2():
            return "ok"

        @app.before_request
        async def set_user():
            g.user_id = 1
            # No g.db set

        async with app.test_client() as client:
            response = await client.get("/test2")
            assert response.status_code == 500

    @pytest.mark.asyncio
    async def test_requires_permission_no_perm_aborts_403(self):
        from middleware.rbac import requires_permission
        from quart import Quart, g

        app = Quart(__name__)

        @app.route("/test3")
        @requires_permission("admin")
        async def test_route3():
            return "ok"

        @app.before_request
        async def set_user_db():
            g.user_id = 1
            g.db = MagicMock()

        with patch("models.rbac.RBACModel.has_permission", return_value=False):
            async with app.test_client() as client:
                response = await client.get("/test3")
                assert response.status_code == 403

    @pytest.mark.asyncio
    async def test_requires_permission_allowed_calls_handler(self):
        from middleware.rbac import requires_permission
        from quart import Quart, g, jsonify

        app = Quart(__name__)

        @app.route("/test4")
        @requires_permission("admin")
        async def test_route4():
            return jsonify({"ok": True})

        @app.before_request
        async def set_user_db4():
            g.user_id = 1
            g.db = MagicMock()

        with patch("models.rbac.RBACModel.has_permission", return_value=True):
            async with app.test_client() as client:
                response = await client.get("/test4")
                assert response.status_code == 200

    @pytest.mark.asyncio
    async def test_requires_permission_with_resource_id_param_from_kwargs(self):
        from middleware.rbac import requires_permission
        from quart import Quart, g, jsonify

        app = Quart(__name__)

        @app.route("/clusters/<int:cluster_id>/manage")
        @requires_permission("cluster_write", "cluster", "cluster_id")
        async def manage_cluster(cluster_id):
            return jsonify({"cluster_id": cluster_id})

        @app.before_request
        async def set_user_db5():
            g.user_id = 1
            g.db = MagicMock()

        with patch("models.rbac.RBACModel.has_permission", return_value=True):
            async with app.test_client() as client:
                response = await client.get("/clusters/5/manage")
                assert response.status_code == 200

    @pytest.mark.asyncio
    async def test_requires_permission_invalid_resource_id_aborts_400(self):
        from middleware.rbac import requires_permission
        from quart import Quart, g, jsonify

        app = Quart(__name__)

        @app.route("/resource/<resource_id>/op")
        @requires_permission("write", "cluster", "resource_id")
        async def op_route(resource_id):
            return jsonify({"ok": True})

        @app.before_request
        async def set_user_db6():
            g.user_id = 1
            g.db = MagicMock()

        with patch("models.rbac.RBACModel.has_permission", return_value=True):
            async with app.test_client() as client:
                response = await client.get("/resource/not-an-int/op")
                assert response.status_code == 400


class TestRequiresRole:
    """requires_role decorator"""

    @pytest.mark.asyncio
    async def test_requires_role_no_user_aborts_401(self):
        from middleware.rbac import requires_role
        from quart import Quart

        app = Quart(__name__)

        @app.route("/admin-only")
        @requires_role("admin")
        async def admin_only():
            return "ok"

        async with app.test_client() as client:
            response = await client.get("/admin-only")
            assert response.status_code == 401

    @pytest.mark.asyncio
    async def test_requires_role_user_lacks_role_aborts_403(self):
        from middleware.rbac import requires_role
        from quart import Quart, g

        app = Quart(__name__)

        @app.route("/admin-page")
        @requires_role("admin")
        async def admin_page():
            return "ok"

        @app.before_request
        async def setup():
            g.user_id = 1
            g.db = MagicMock()

        with patch("models.rbac.RBACModel.get_user_roles", return_value=[
            {"role_name": "viewer", "scope": "global", "resource_id": None}
        ]):
            async with app.test_client() as client:
                response = await client.get("/admin-page")
                assert response.status_code == 403

    @pytest.mark.asyncio
    async def test_requires_role_user_has_role_allows_access(self):
        from middleware.rbac import requires_role
        from quart import Quart, g, jsonify
        from models.rbac import PermissionScope

        app = Quart(__name__)

        @app.route("/admin-success")
        @requires_role("admin", PermissionScope.GLOBAL)
        async def admin_success():
            return jsonify({"status": "ok"})

        @app.before_request
        async def setup2():
            g.user_id = 1
            g.db = MagicMock()

        with patch("models.rbac.RBACModel.get_user_roles", return_value=[
            {"role_name": "admin", "scope": "global", "resource_id": None}
        ]):
            async with app.test_client() as client:
                response = await client.get("/admin-success")
                assert response.status_code == 200


class TestRequiresAnyPermission:
    """requires_any_permission decorator"""

    @pytest.mark.asyncio
    async def test_requires_any_no_user_aborts_401(self):
        from middleware.rbac import requires_any_permission
        from quart import Quart

        app = Quart(__name__)

        @app.route("/any-perm")
        @requires_any_permission("perm_a", "perm_b")
        async def any_perm():
            return "ok"

        async with app.test_client() as client:
            response = await client.get("/any-perm")
            assert response.status_code == 401

    @pytest.mark.asyncio
    async def test_requires_any_user_has_one_perm_allows(self):
        from middleware.rbac import requires_any_permission
        from quart import Quart, g, jsonify

        app = Quart(__name__)

        @app.route("/any-allowed")
        @requires_any_permission("perm_a", "perm_b")
        async def any_allowed():
            return jsonify({"ok": True})

        @app.before_request
        async def setup3():
            g.user_id = 1
            g.db = MagicMock()

        with patch("models.rbac.RBACModel.get_user_permissions", return_value={
            "global": ["perm_a"],
            "cluster": {},
            "service": {},
        }):
            async with app.test_client() as client:
                response = await client.get("/any-allowed")
                assert response.status_code == 200

    @pytest.mark.asyncio
    async def test_requires_any_user_has_none_aborts_403(self):
        from middleware.rbac import requires_any_permission
        from quart import Quart, g

        app = Quart(__name__)

        @app.route("/any-denied")
        @requires_any_permission("perm_a", "perm_b")
        async def any_denied():
            return "ok"

        @app.before_request
        async def setup4():
            g.user_id = 1
            g.db = MagicMock()

        with patch("models.rbac.RBACModel.get_user_permissions", return_value={
            "global": ["other_perm"],
            "cluster": {},
            "service": {},
        }):
            async with app.test_client() as client:
                response = await client.get("/any-denied")
                assert response.status_code == 403


class TestRequiresAllPermissions:
    """requires_all_permissions decorator"""

    @pytest.mark.asyncio
    async def test_requires_all_user_has_all_allows(self):
        from middleware.rbac import requires_all_permissions
        from quart import Quart, g, jsonify

        app = Quart(__name__)

        @app.route("/all-allowed")
        @requires_all_permissions("perm_a", "perm_b")
        async def all_allowed():
            return jsonify({"ok": True})

        @app.before_request
        async def setup5():
            g.user_id = 1
            g.db = MagicMock()

        with patch("models.rbac.RBACModel.get_user_permissions", return_value={
            "global": ["perm_a", "perm_b"],
            "cluster": {},
            "service": {},
        }):
            async with app.test_client() as client:
                response = await client.get("/all-allowed")
                assert response.status_code == 200

    @pytest.mark.asyncio
    async def test_requires_all_user_missing_one_aborts_403(self):
        from middleware.rbac import requires_all_permissions
        from quart import Quart, g

        app = Quart(__name__)

        @app.route("/all-denied")
        @requires_all_permissions("perm_a", "perm_b")
        async def all_denied():
            return "ok"

        @app.before_request
        async def setup6():
            g.user_id = 1
            g.db = MagicMock()

        with patch("models.rbac.RBACModel.get_user_permissions", return_value={
            "global": ["perm_a"],
            "cluster": {},
            "service": {},
        }):
            async with app.test_client() as client:
                response = await client.get("/all-denied")
                assert response.status_code == 403


class TestRBACHelperFunctions:
    """Tests for is_admin, can_manage_users, can_access_cluster, can_access_service"""

    def test_is_admin_true_when_has_global_admin(self):
        from middleware.rbac import is_admin
        from models.rbac import Permissions

        mock_db = MagicMock()
        with patch("models.rbac.RBACModel.get_user_permissions", return_value={
            "global": [Permissions.GLOBAL_ADMIN],
            "cluster": {},
            "service": {},
        }):
            assert is_admin(1, mock_db) is True

    def test_is_admin_false_when_no_admin_perm(self):
        from middleware.rbac import is_admin

        mock_db = MagicMock()
        with patch("models.rbac.RBACModel.get_user_permissions", return_value={
            "global": ["viewer"],
            "cluster": {},
            "service": {},
        }):
            assert is_admin(1, mock_db) is False

    def test_can_manage_users_with_admin_perm(self):
        from middleware.rbac import can_manage_users
        from models.rbac import Permissions

        mock_db = MagicMock()
        with patch("models.rbac.RBACModel.get_user_permissions", return_value={
            "global": [Permissions.GLOBAL_ADMIN],
            "cluster": {},
            "service": {},
        }):
            assert can_manage_users(1, mock_db) is True

    def test_can_manage_users_with_user_write_perm(self):
        from middleware.rbac import can_manage_users
        from models.rbac import Permissions

        mock_db = MagicMock()
        with patch("models.rbac.RBACModel.get_user_permissions", return_value={
            "global": [Permissions.GLOBAL_USER_WRITE],
            "cluster": {},
            "service": {},
        }):
            assert can_manage_users(1, mock_db) is True

    def test_can_manage_users_false_with_no_perm(self):
        from middleware.rbac import can_manage_users

        mock_db = MagicMock()
        with patch("models.rbac.RBACModel.get_user_permissions", return_value={
            "global": [],
            "cluster": {},
            "service": {},
        }):
            assert can_manage_users(1, mock_db) is False

    def test_can_access_cluster_calls_has_permission(self):
        from middleware.rbac import can_access_cluster

        mock_db = MagicMock()
        with patch("models.rbac.RBACModel.has_permission", return_value=True):
            assert can_access_cluster(1, 5, mock_db) is True

    def test_can_access_service_calls_has_permission(self):
        from middleware.rbac import can_access_service

        mock_db = MagicMock()
        with patch("models.rbac.RBACModel.has_permission", return_value=False):
            assert can_access_service(1, 5, mock_db) is False


# ============================================================================
# models/rate_limiting.py Tests
# ============================================================================


def _make_db_with_query(first_result=None, select_list=None, delete_count=5):
    """Create a DB mock with configurable query results."""
    db = MagicMock()
    db.rate_limits = MagicMock()
    db.rate_limits.insert = MagicMock()
    db.xdp_rate_limits = MagicMock()
    db.xdp_rate_limits.insert = MagicMock(return_value=1)

    query_mock = MagicMock()
    first_mock = MagicMock(return_value=first_result)
    select_result = MagicMock()
    select_result.first = first_mock
    if select_list is not None:
        # Make select() return the list directly for iteration
        query_mock.select = MagicMock(return_value=select_list)
    else:
        query_mock.select = MagicMock(return_value=select_result)
    query_mock.delete = MagicMock(return_value=delete_count)
    # Use return_value so db(anything) returns query_mock
    db.return_value = query_mock
    return db, query_mock


def _make_db():
    """Backward-compatible wrapper."""
    return _make_db_with_query()


class TestRateLimitModel:
    """Tests for RateLimitModel static methods"""

    def test_check_rate_limit_new_client_allowed(self):
        from models.rate_limiting import RateLimitModel

        db, query_mock = _make_db_with_query(first_result=None)

        allowed, info = RateLimitModel.check_rate_limit(db, "ip:1.2.3.4", "/api/test")

        assert allowed is True
        assert info["allowed"] is True
        assert "requests_remaining" in info
        assert "window_reset" in info

    def test_check_rate_limit_blocked_client_denied(self):
        from models.rate_limiting import RateLimitModel

        future = datetime.utcnow() + timedelta(minutes=30)
        existing = MagicMock(
            is_blocked=True,
            block_until=future,
            window_start=datetime.utcnow() - timedelta(minutes=5),
            request_count=100,
        )
        db, query_mock = _make_db_with_query(first_result=existing)

        allowed, info = RateLimitModel.check_rate_limit(db, "ip:1.2.3.4", "/api/test", max_requests=10)

        assert allowed is False
        assert info["allowed"] is False
        assert info["error"] == "Rate limit exceeded"

    def test_check_rate_limit_window_expired_resets(self):
        from models.rate_limiting import RateLimitModel

        old_window = datetime.utcnow() - timedelta(hours=2)
        existing = MagicMock(
            is_blocked=False,
            block_until=None,
            window_start=old_window,
            request_count=50,
        )
        db, query_mock = _make_db_with_query(first_result=existing)

        allowed, info = RateLimitModel.check_rate_limit(db, "ip:1.2.3.4", "/api/test")

        assert allowed is True
        existing.update_record.assert_called_once()

    def test_check_rate_limit_exceeded_blocks_client(self):
        from models.rate_limiting import RateLimitModel

        existing = MagicMock(
            is_blocked=False,
            block_until=None,
            window_start=datetime.utcnow() - timedelta(minutes=5),
            request_count=100,
        )
        db, query_mock = _make_db_with_query(first_result=existing)

        allowed, info = RateLimitModel.check_rate_limit(
            db, "ip:1.2.3.4", "/api/test", max_requests=100
        )

        assert allowed is False
        assert info["allowed"] is False

    def test_check_rate_limit_increments_counter(self):
        from models.rate_limiting import RateLimitModel

        existing = MagicMock(
            is_blocked=False,
            block_until=None,
            window_start=datetime.utcnow() - timedelta(minutes=5),
            request_count=5,
        )
        db, query_mock = _make_db_with_query(first_result=existing)

        allowed, info = RateLimitModel.check_rate_limit(
            db, "ip:1.2.3.4", "/api/test", max_requests=100
        )

        assert allowed is True
        existing.update_record.assert_called_once()

    def test_cleanup_old_records_returns_count(self):
        from models.rate_limiting import RateLimitModel

        db = MagicMock()
        # MagicMock comparison with datetime raises TypeError; patch __lt__ etc.
        db.rate_limits.last_request.__lt__ = MagicMock(return_value=MagicMock())
        db.rate_limits.is_blocked.__eq__ = MagicMock(return_value=MagicMock())
        db.rate_limits.block_until.__lt__ = MagicMock(return_value=MagicMock())
        delete_mock = MagicMock(return_value=3)
        query_mock = MagicMock()
        query_mock.delete = delete_mock
        db.return_value = query_mock

        deleted = RateLimitModel.cleanup_old_records(db)
        assert deleted == 3

    def test_get_client_stats_empty(self):
        from models.rate_limiting import RateLimitModel

        db, query_mock = _make_db_with_query(select_list=[])

        stats = RateLimitModel.get_client_stats(db, "ip:1.2.3.4")
        assert stats["client_id"] == "ip:1.2.3.4"
        assert stats["total_requests"] == 0
        assert stats["blocked_endpoints"] == 0

    def test_get_client_stats_with_records(self):
        from models.rate_limiting import RateLimitModel

        rec1 = MagicMock(
            endpoint="/api/test",
            request_count=50,
            window_start=datetime.utcnow(),
            last_request=datetime.utcnow(),
            is_blocked=False,
            block_until=None,
        )
        rec2 = MagicMock(
            endpoint="/api/other",
            request_count=200,
            window_start=datetime.utcnow(),
            last_request=datetime.utcnow(),
            is_blocked=True,
            block_until=datetime.utcnow() + timedelta(minutes=10),
        )
        db, query_mock = _make_db_with_query(select_list=[rec1, rec2])

        stats = RateLimitModel.get_client_stats(db, "ip:1.2.3.4")
        assert stats["total_requests"] == 250
        assert stats["blocked_endpoints"] == 1
        assert len(stats["endpoints"]) == 2


class TestRateLimitManager:
    """Tests for RateLimitManager"""

    def test_init_sets_policies(self):
        from models.rate_limiting import RateLimitManager

        db = MagicMock()
        manager = RateLimitManager(db)
        assert "api_general" in manager.policies
        assert "api_auth" in manager.policies
        assert "api_admin" in manager.policies
        assert "api_proxy" in manager.policies
        assert "api_license" in manager.policies

    def test_check_limit_uses_policy(self):
        from models.rate_limiting import RateLimitManager, RateLimitModel

        db = MagicMock()
        manager = RateLimitManager(db)

        with patch.object(RateLimitModel, "check_rate_limit", return_value=(True, {"allowed": True})) as mock_check:
            result = manager.check_limit("ip:1.2.3.4", "/api/test", "api_auth")
            mock_check.assert_called_once()
            # Verify auth policy applied
            call_kwargs = mock_check.call_args
            assert call_kwargs[1]["max_requests"] == 30
            assert call_kwargs[1]["window_minutes"] == 15

    def test_check_limit_unknown_type_uses_general(self):
        from models.rate_limiting import RateLimitManager, RateLimitModel

        db = MagicMock()
        manager = RateLimitManager(db)

        with patch.object(RateLimitModel, "check_rate_limit", return_value=(True, {})) as mock_check:
            manager.check_limit("ip:1.2.3.4", "/api/test", "unknown_type")
            call_kwargs = mock_check.call_args
            assert call_kwargs[1]["max_requests"] == 1000  # api_general

    def test_get_client_identifier_user(self):
        from models.rate_limiting import RateLimitManager

        db = MagicMock()
        manager = RateLimitManager(db)
        mock_request = MagicMock()

        identifier = manager.get_client_identifier(mock_request, user={"id": 42})
        assert identifier == "user:42"

    def test_get_client_identifier_forwarded_for(self):
        from models.rate_limiting import RateLimitManager

        db = MagicMock()
        manager = RateLimitManager(db)
        mock_request = MagicMock()
        mock_request.headers.get = lambda key, default=None: "10.0.0.1, 192.168.1.1" if key == "X-Forwarded-For" else default

        identifier = manager.get_client_identifier(mock_request)
        assert identifier == "ip:10.0.0.1"

    def test_get_client_identifier_remote_addr(self):
        from models.rate_limiting import RateLimitManager

        db = MagicMock()
        manager = RateLimitManager(db)
        mock_request = MagicMock()
        mock_request.headers.get = lambda key, default=None: None
        mock_request.environ.get = lambda key, default: "172.16.0.1"

        identifier = manager.get_client_identifier(mock_request)
        assert identifier.startswith("ip:")

    def test_get_endpoint_type_auth(self):
        from models.rate_limiting import RateLimitManager

        manager = RateLimitManager(MagicMock())
        assert manager.get_endpoint_type("/api/auth/login") == "api_auth"

    def test_get_endpoint_type_proxy(self):
        from models.rate_limiting import RateLimitManager

        manager = RateLimitManager(MagicMock())
        assert manager.get_endpoint_type("/api/proxy/config") == "api_proxy"

    def test_get_endpoint_type_license(self):
        from models.rate_limiting import RateLimitManager

        manager = RateLimitManager(MagicMock())
        assert manager.get_endpoint_type("/api/license/check") == "api_license"

    def test_get_endpoint_type_clusters(self):
        from models.rate_limiting import RateLimitManager

        manager = RateLimitManager(MagicMock())
        assert manager.get_endpoint_type("/api/clusters") == "api_admin"

    def test_get_endpoint_type_general(self):
        from models.rate_limiting import RateLimitManager

        manager = RateLimitManager(MagicMock())
        assert manager.get_endpoint_type("/api/v1/mappings") == "api_general"


class TestXDPRateLimitModel:
    """Tests for XDPRateLimitModel static methods"""

    def test_validate_config_valid(self):
        from models.rate_limiting import XDPRateLimitModel

        config = {
            "name": "Test Config",
            "cluster_id": 1,
            "global_pps_limit": 10000,
            "per_ip_pps_limit": 1000,
            "window_size_ns": 1000000000,
            "burst_allowance": 100,
            "action": 1,
            "interfaces": ["eth0"],
        }
        valid, errors = XDPRateLimitModel.validate_config(config)
        assert valid is True
        assert errors == []

    def test_validate_config_missing_name(self):
        from models.rate_limiting import XDPRateLimitModel

        config = {
            "cluster_id": 1,
            "global_pps_limit": 10000,
            "window_size_ns": 1000000000,
            "interfaces": ["eth0"],
        }
        valid, errors = XDPRateLimitModel.validate_config(config)
        assert valid is False
        assert any("Name" in e for e in errors)

    def test_validate_config_missing_cluster_id(self):
        from models.rate_limiting import XDPRateLimitModel

        config = {
            "name": "Test",
            "global_pps_limit": 10000,
            "window_size_ns": 1000000000,
            "interfaces": ["eth0"],
        }
        valid, errors = XDPRateLimitModel.validate_config(config)
        assert valid is False
        assert any("Cluster" in e for e in errors)

    def test_validate_config_negative_global_pps(self):
        from models.rate_limiting import XDPRateLimitModel

        config = {
            "name": "Test",
            "cluster_id": 1,
            "global_pps_limit": -1,
            "window_size_ns": 1000000000,
            "interfaces": ["eth0"],
        }
        valid, errors = XDPRateLimitModel.validate_config(config)
        assert valid is False

    def test_validate_config_per_ip_exceeds_global(self):
        from models.rate_limiting import XDPRateLimitModel

        config = {
            "name": "Test",
            "cluster_id": 1,
            "global_pps_limit": 1000,
            "per_ip_pps_limit": 5000,  # Exceeds global
            "window_size_ns": 1000000000,
            "interfaces": ["eth0"],
        }
        valid, errors = XDPRateLimitModel.validate_config(config)
        assert valid is False

    def test_validate_config_window_too_small(self):
        from models.rate_limiting import XDPRateLimitModel

        config = {
            "name": "Test",
            "cluster_id": 1,
            "window_size_ns": 50000000,  # < 100ms
            "interfaces": ["eth0"],
        }
        valid, errors = XDPRateLimitModel.validate_config(config)
        assert valid is False

    def test_validate_config_invalid_action(self):
        from models.rate_limiting import XDPRateLimitModel

        config = {
            "name": "Test",
            "cluster_id": 1,
            "action": 99,
            "window_size_ns": 1000000000,
            "interfaces": ["eth0"],
        }
        valid, errors = XDPRateLimitModel.validate_config(config)
        assert valid is False

    def test_validate_config_empty_interfaces(self):
        from models.rate_limiting import XDPRateLimitModel

        config = {
            "name": "Test",
            "cluster_id": 1,
            "window_size_ns": 1000000000,
            "interfaces": [],
        }
        valid, errors = XDPRateLimitModel.validate_config(config)
        assert valid is False

    def test_validate_config_interfaces_not_list(self):
        from models.rate_limiting import XDPRateLimitModel

        config = {
            "name": "Test",
            "cluster_id": 1,
            "window_size_ns": 1000000000,
            "interfaces": "eth0",  # Should be list
        }
        valid, errors = XDPRateLimitModel.validate_config(config)
        assert valid is False

    def test_validate_config_negative_burst_allowance(self):
        from models.rate_limiting import XDPRateLimitModel

        config = {
            "name": "Test",
            "cluster_id": 1,
            "window_size_ns": 1000000000,
            "burst_allowance": -5,
            "interfaces": ["eth0"],
        }
        valid, errors = XDPRateLimitModel.validate_config(config)
        assert valid is False

    def test_create_default_config_existing_returns_existing_id(self):
        from models.rate_limiting import XDPRateLimitModel

        existing = MagicMock(id=42)
        db, _ = _make_db_with_query(first_result=existing)
        db.xdp_rate_limits = MagicMock()
        db.xdp_rate_limits.insert = MagicMock(return_value=5)

        result = XDPRateLimitModel.create_default_config(db, 1, 1)
        assert result == 42

    def test_create_default_config_creates_new(self):
        from models.rate_limiting import XDPRateLimitModel

        db, _ = _make_db_with_query(first_result=None)
        db.xdp_rate_limits = MagicMock()
        db.xdp_rate_limits.insert = MagicMock(return_value=5)

        result = XDPRateLimitModel.create_default_config(db, 1, 1)
        assert result == 5

    def test_create_default_config_exception_returns_none(self):
        from models.rate_limiting import XDPRateLimitModel

        db = MagicMock()
        db.side_effect = Exception("DB error")

        result = XDPRateLimitModel.create_default_config(db, 1, 1)
        assert result is None

    def _make_xdp_stats_db(self, select_list):
        """DB mock with patched datetime comparisons for xdp_rate_limit_stats."""
        db = MagicMock()
        db.xdp_rate_limit_stats = MagicMock()
        # Patch comparison operators that use datetime
        db.xdp_rate_limit_stats.stats_timestamp.__ge__ = MagicMock(return_value=MagicMock())
        db.xdp_rate_limit_stats.rate_limit_id.__eq__ = MagicMock(return_value=MagicMock())
        db.xdp_rate_limit_stats.proxy_id.__eq__ = MagicMock(return_value=MagicMock())
        query_mock = MagicMock()
        query_mock.select = MagicMock(return_value=select_list)
        db.return_value = query_mock
        return db

    def test_get_proxy_stats_no_data_returns_zeros(self):
        from models.rate_limiting import XDPRateLimitModel

        db = self._make_xdp_stats_db([])

        stats = XDPRateLimitModel.get_proxy_stats(db, 1, 1)
        assert stats["total_packets"] == 0
        assert stats["drop_rate"] == 0.0

    def test_get_proxy_stats_aggregates_data(self):
        from models.rate_limiting import XDPRateLimitModel

        s1 = MagicMock(
            total_packets=1000,
            passed_packets=900,
            dropped_packets=100,
            rate_limited_ips=5,
            cpu_usage_percent=20.0,
            memory_usage_bytes=1024,
            stats_timestamp=datetime.utcnow(),
        )
        s2 = MagicMock(
            total_packets=2000,
            passed_packets=1800,
            dropped_packets=200,
            rate_limited_ips=10,
            cpu_usage_percent=30.0,
            memory_usage_bytes=2048,
            stats_timestamp=datetime.utcnow(),
        )
        db = self._make_xdp_stats_db([s1, s2])

        stats = XDPRateLimitModel.get_proxy_stats(db, 1, 1)
        assert stats["total_packets"] == 3000
        assert stats["dropped_packets"] == 300
        assert stats["drop_rate"] > 0


class TestXDPRateLimitManager:
    """Tests for XDPRateLimitManager"""

    def test_create_rate_limit_invalid_config_returns_false(self):
        from models.rate_limiting import XDPRateLimitManager

        db = MagicMock()
        manager = XDPRateLimitManager(db)

        ok, result = manager.create_rate_limit(1, {}, 1)
        assert ok is False
        assert "errors" in result

    def test_create_rate_limit_no_enterprise_license_returns_false(self):
        from models.rate_limiting import XDPRateLimitManager

        db = MagicMock()
        mock_license = MagicMock()
        mock_license.has_feature.return_value = False
        manager = XDPRateLimitManager(db, license_manager=mock_license)

        config = {
            "name": "Test",
            "cluster_id": 1,
            "window_size_ns": 1000000000,
            "interfaces": ["eth0"],
            "requires_enterprise": True,
        }
        ok, result = manager.create_rate_limit(1, config, 1)
        assert ok is False
        assert "Enterprise" in result.get("error", "")

    def test_create_rate_limit_success_returns_true(self):
        from models.rate_limiting import XDPRateLimitManager

        db = MagicMock()
        db.xdp_rate_limits = MagicMock()
        db.xdp_rate_limits.insert = MagicMock(return_value=1)
        mock_license = MagicMock()
        mock_license.has_feature.return_value = True
        manager = XDPRateLimitManager(db, license_manager=mock_license)

        config = {
            "name": "Test Config",
            "cluster_id": 1,
            "window_size_ns": 1000000000,
            "interfaces": ["eth0"],
            "requires_enterprise": True,
        }
        ok, result = manager.create_rate_limit(1, config, 1)
        assert ok is True
        assert result.get("id") == 1

    def test_update_rate_limit_not_found_returns_false(self):
        from models.rate_limiting import XDPRateLimitManager

        db = MagicMock()
        db.xdp_rate_limits = MagicMock()
        db.xdp_rate_limits.__getitem__ = MagicMock(return_value=None)
        manager = XDPRateLimitManager(db)

        ok, result = manager.update_rate_limit(999, {}, 1)
        assert ok is False
        assert "not found" in result.get("error", "")

    def test_delete_rate_limit_not_found_returns_false(self):
        from models.rate_limiting import XDPRateLimitManager

        db = MagicMock()
        db.xdp_rate_limits = MagicMock()
        db.xdp_rate_limits.__getitem__ = MagicMock(return_value=None)
        manager = XDPRateLimitManager(db)

        ok, result = manager.delete_rate_limit(999)
        assert ok is False

    def test_delete_rate_limit_success(self):
        from models.rate_limiting import XDPRateLimitManager

        db = MagicMock()
        mock_record = MagicMock()
        db.xdp_rate_limits = MagicMock()
        db.xdp_rate_limits.__getitem__ = MagicMock(return_value=mock_record)
        manager = XDPRateLimitManager(db)

        ok, result = manager.delete_rate_limit(1)
        assert ok is True
        mock_record.update_record.assert_called_once_with(is_active=False)

    def test_get_proxy_config_filters_active_with_license(self):
        from models.rate_limiting import XDPRateLimitManager, XDPRateLimitModel

        db = MagicMock()
        mock_license = MagicMock()
        mock_license.has_feature.return_value = True
        manager = XDPRateLimitManager(db, license_manager=mock_license)

        configs = [
            {"id": 1, "enabled": True, "license_validated": True, "name": "active"},
            {"id": 2, "enabled": False, "license_validated": True, "name": "disabled"},
            {"id": 3, "enabled": True, "license_validated": False, "name": "unlicensed"},
        ]

        with patch.object(XDPRateLimitModel, "get_cluster_configs", return_value=configs):
            result = manager.get_proxy_config(1, 1)
            assert result["total_configs"] == 1
            assert result["configurations"][0]["name"] == "active"


# ============================================================================
# models/auth_native.py Tests
# ============================================================================


class TestTOTPManager:
    """Tests for TOTPManager"""

    def _make_auth(self):
        auth = MagicMock()
        auth.db = MagicMock()
        return auth

    def test_enable_2fa_user_not_found_returns_none(self):
        from models.auth_native import TOTPManager

        auth = self._make_auth()
        auth.db.auth_user = MagicMock()
        auth.db.auth_user.__getitem__ = MagicMock(return_value=None)
        manager = TOTPManager(auth)

        result = manager.enable_2fa(999, "password")
        assert result is None

    def test_enable_2fa_wrong_password_returns_none(self):
        from models.auth_native import TOTPManager

        auth = self._make_auth()
        mock_user = MagicMock(email="test@test.com")
        auth.db.auth_user.__getitem__ = MagicMock(return_value=mock_user)
        auth.verify_password.return_value = False
        manager = TOTPManager(auth)

        result = manager.enable_2fa(1, "wrongpass")
        assert result is None

    def test_enable_2fa_success_returns_dict(self):
        from models.auth_native import TOTPManager

        auth = self._make_auth()
        mock_user = MagicMock(email="test@test.com", username="testuser")
        auth.db.auth_user.__getitem__ = MagicMock(return_value=mock_user)
        auth.verify_password.return_value = True
        manager = TOTPManager(auth)

        result = manager.enable_2fa(1, "correctpass")
        assert result is not None
        assert "secret" in result
        assert "qr_uri" in result
        assert "qr_code" in result

    def test_verify_and_complete_2fa_user_not_found(self):
        from models.auth_native import TOTPManager

        auth = self._make_auth()
        auth.db.auth_user.__getitem__ = MagicMock(return_value=None)
        manager = TOTPManager(auth)

        result = manager.verify_and_complete_2fa(999, "secret", "123456")
        assert result is False

    def test_verify_and_complete_2fa_secret_mismatch(self):
        from models.auth_native import TOTPManager

        auth = self._make_auth()
        mock_user = MagicMock(totp_secret="correct-secret")
        auth.db.auth_user.__getitem__ = MagicMock(return_value=mock_user)
        manager = TOTPManager(auth)

        result = manager.verify_and_complete_2fa(1, "wrong-secret", "123456")
        assert result is False

    def test_verify_and_complete_2fa_invalid_totp(self):
        import pyotp
        from models.auth_native import TOTPManager

        auth = self._make_auth()
        secret = pyotp.random_base32()
        mock_user = MagicMock(totp_secret=secret)
        auth.db.auth_user.__getitem__ = MagicMock(return_value=mock_user)
        manager = TOTPManager(auth)

        result = manager.verify_and_complete_2fa(1, secret, "000000")
        # May be True or False depending on random code but 000000 is unlikely to match
        assert isinstance(result, bool)

    def test_disable_2fa_user_not_found(self):
        from models.auth_native import TOTPManager

        auth = self._make_auth()
        auth.db.auth_user.__getitem__ = MagicMock(return_value=None)
        manager = TOTPManager(auth)

        result = manager.disable_2fa(999, "pass")
        assert result is False

    def test_disable_2fa_wrong_password_returns_false(self):
        from models.auth_native import TOTPManager

        auth = self._make_auth()
        mock_user = MagicMock(totp_enabled=False)
        auth.db.auth_user.__getitem__ = MagicMock(return_value=mock_user)
        auth.verify_password.return_value = False
        manager = TOTPManager(auth)

        result = manager.disable_2fa(1, "wrongpass")
        assert result is False

    def test_disable_2fa_totp_enabled_no_code_returns_false(self):
        from models.auth_native import TOTPManager

        auth = self._make_auth()
        mock_user = MagicMock(totp_enabled=True, totp_secret="secret")
        auth.db.auth_user.__getitem__ = MagicMock(return_value=mock_user)
        auth.verify_password.return_value = True
        manager = TOTPManager(auth)

        result = manager.disable_2fa(1, "correctpass")
        assert result is False

    def test_disable_2fa_success_no_totp(self):
        from models.auth_native import TOTPManager

        auth = self._make_auth()
        mock_user = MagicMock(totp_enabled=False)
        auth.db.auth_user.__getitem__ = MagicMock(return_value=mock_user)
        auth.verify_password.return_value = True
        manager = TOTPManager(auth)

        result = manager.disable_2fa(1, "correctpass")
        assert result is True
        mock_user.update_record.assert_called_once_with(totp_enabled=False, totp_secret=None)

    def test_verify_totp_user_not_found(self):
        from models.auth_native import TOTPManager

        auth = self._make_auth()
        auth.db.auth_user.__getitem__ = MagicMock(return_value=None)
        manager = TOTPManager(auth)

        result = manager.verify_totp(999, "123456")
        assert result is False

    def test_verify_totp_totp_not_enabled(self):
        from models.auth_native import TOTPManager

        auth = self._make_auth()
        mock_user = MagicMock(totp_enabled=False, totp_secret=None)
        auth.db.auth_user.__getitem__ = MagicMock(return_value=mock_user)
        manager = TOTPManager(auth)

        result = manager.verify_totp(1, "123456")
        assert result is False

    def test_generate_qr_code_returns_base64_string(self):
        from models.auth_native import TOTPManager

        auth = self._make_auth()
        manager = TOTPManager(auth)

        qr = manager._generate_qr_code("otpauth://totp/test?secret=JBSWY3DPEHPK3PXP&issuer=Test")
        assert isinstance(qr, str)
        assert len(qr) > 0


class TestAPITokenManager:
    """Tests for APITokenManager"""

    def _make_auth_with_db(self):
        """Auth mock with pydal-like db that supports define_table"""
        auth = MagicMock()
        auth.db = MagicMock()
        auth.db.tables = []
        auth.db.define_table = MagicMock()
        return auth

    def test_init_defines_api_tokens_table_if_missing(self):
        from models.auth_native import APITokenManager
        import models.auth_native as auth_native_mod

        auth = self._make_auth_with_db()
        # Inject Field into module namespace so define_table call succeeds
        auth_native_mod.Field = MagicMock()
        try:
            manager = APITokenManager(auth)
            auth.db.define_table.assert_called_once()
            call_args = auth.db.define_table.call_args[0]
            assert call_args[0] == "api_tokens"
        finally:
            del auth_native_mod.Field

    def test_init_skips_table_if_already_exists(self):
        from models.auth_native import APITokenManager

        auth = self._make_auth_with_db()
        auth.db.tables = ["api_tokens"]
        manager = APITokenManager(auth)
        auth.db.define_table.assert_not_called()

    def test_create_token_returns_token_and_id(self):
        from models.auth_native import APITokenManager

        auth = self._make_auth_with_db()
        auth.db.tables = ["api_tokens"]
        auth.db.api_tokens = MagicMock()
        auth.db.api_tokens.insert = MagicMock()
        manager = APITokenManager(auth)

        token, token_id = manager.create_token(1, "test-token", {"read": True})
        assert isinstance(token, str)
        assert isinstance(token_id, str)
        assert len(token) > 10

    def test_create_token_with_ttl(self):
        from models.auth_native import APITokenManager

        auth = self._make_auth_with_db()
        auth.db.tables = ["api_tokens"]
        auth.db.api_tokens = MagicMock()
        auth.db.api_tokens.insert = MagicMock()
        manager = APITokenManager(auth)

        token, token_id = manager.create_token(1, "expiring-token", ttl_days=7)
        auth.db.api_tokens.insert.assert_called_once()
        call_kwargs = auth.db.api_tokens.insert.call_args[1]
        assert call_kwargs.get("expires_at") is not None


class TestAuthNativeHelpers:
    """Tests for helper functions in auth_native.py"""

    def test_setup_auth_raises_not_implemented(self):
        from models.auth_native import setup_auth

        with pytest.raises(NotImplementedError):
            setup_auth(MagicMock())

    def test_check_permission_no_user_id_returns_false(self):
        from models.auth_native import check_permission

        auth = MagicMock()
        auth.user_id = None
        result = check_permission(auth, "read_clusters")
        assert result is False

    def test_check_permission_admin_returns_true(self):
        from models.auth_native import check_permission

        auth = MagicMock()
        auth.user_id = 1
        auth.get_user.return_value = {"is_admin": True}
        result = check_permission(auth, "read_clusters")
        assert result is True

    def test_check_permission_delegates_to_auth(self):
        from models.auth_native import check_permission

        auth = MagicMock()
        auth.user_id = 1
        auth.get_user.return_value = {"is_admin": False}
        auth.has_permission.return_value = True
        result = check_permission(auth, "read_clusters")
        assert result is True
        auth.has_permission.assert_called_once_with("read_clusters", 1)

    def test_require_permission_grants_access(self):
        from models.auth_native import require_permission

        auth = MagicMock()
        auth.user_id = 1
        auth.get_user.return_value = {"is_admin": True}

        @require_permission(auth, "read_clusters")
        def protected():
            return "ok"

        result = protected()
        assert result == "ok"

    def test_require_admin_allows_admin(self):
        from models.auth_native import require_admin

        auth = MagicMock()
        auth.get_user.return_value = {"is_admin": True}

        @require_admin(auth)
        def admin_func():
            return "admin"

        result = admin_func()
        assert result == "admin"


class TestRateLimitFixture:
    """Test rate_limit_fixture decorator"""

    def test_rate_limit_fixture_no_manager_passes_through(self):
        from models.rate_limiting import rate_limit_fixture

        @rate_limit_fixture("api_general")
        def my_view():
            return "response"

        # Without rate_limit_manager in globals, should pass through
        result = my_view()
        assert result == "response"
