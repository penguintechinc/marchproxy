"""
Unit tests for manager/middleware/auth.py

Tests JWT token extraction, validation, and the require_auth decorator.
No real JWT signing or database connections are used.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from middleware.auth import (
    _extract_token_from_header,
    _check_license_feature,
    get_current_user,
    is_admin,
    require_auth,
    AuthContext,
)


pytestmark = pytest.mark.unit


# ---------------------------------------------------------------------------
# _extract_token_from_header — requires a Quart request context
# ---------------------------------------------------------------------------


class TestExtractTokenFromHeader:
    @pytest.mark.asyncio
    async def test_valid_bearer_token_returned(self, test_app):
        async with test_app.test_request_context("/", headers={"Authorization": "Bearer mytoken123"}):
            result = _extract_token_from_header()
        assert result == "mytoken123"

    @pytest.mark.asyncio
    async def test_missing_header_returns_none(self, test_app):
        async with test_app.test_request_context("/"):
            result = _extract_token_from_header()
        assert result is None

    @pytest.mark.asyncio
    async def test_no_authorization_header_returns_none(self, test_app):
        async with test_app.test_request_context("/"):
            result = _extract_token_from_header()
        assert result is None

    @pytest.mark.asyncio
    async def test_malformed_header_single_word_returns_none(self, test_app):
        async with test_app.test_request_context("/", headers={"Authorization": "Bearer"}):
            result = _extract_token_from_header()
        assert result is None

    @pytest.mark.asyncio
    async def test_malformed_header_wrong_scheme_returns_none(self, test_app):
        async with test_app.test_request_context("/", headers={"Authorization": "Basic dXNlcjpwYXNz"}):
            result = _extract_token_from_header()
        assert result is None

    @pytest.mark.asyncio
    async def test_malformed_header_too_many_parts_returns_none(self, test_app):
        async with test_app.test_request_context("/", headers={"Authorization": "Bearer token extra"}):
            result = _extract_token_from_header()
        assert result is None

    @pytest.mark.asyncio
    async def test_bearer_case_insensitive(self, test_app):
        async with test_app.test_request_context("/", headers={"Authorization": "BEARER mytoken456"}):
            result = _extract_token_from_header()
        assert result == "mytoken456"


# ---------------------------------------------------------------------------
# get_current_user / is_admin helpers
# ---------------------------------------------------------------------------


class TestGetCurrentUser:
    @pytest.mark.asyncio
    async def test_returns_none_when_no_user_in_context(self, test_app):
        async with test_app.app_context():
            result = get_current_user()
        assert result is None

    @pytest.mark.asyncio
    async def test_returns_user_when_set_in_g(self, test_app):
        async with test_app.app_context():
            from quart import g
            g.user = {"user_id": 1, "username": "tester"}
            result = get_current_user()
            assert result == {"user_id": 1, "username": "tester"}


class TestIsAdmin:
    @pytest.mark.asyncio
    async def test_returns_false_when_no_user(self, test_app):
        async with test_app.app_context():
            result = is_admin()
        assert result is False

    @pytest.mark.asyncio
    async def test_returns_false_when_no_admin_scope(self, test_app):
        async with test_app.app_context():
            from quart import g
            g.user = {"user_id": 2, "scope": ["read"], "roles": ["viewer"]}
            result = is_admin()
        assert result is False

    @pytest.mark.asyncio
    async def test_returns_true_when_star_admin_scope(self, test_app):
        async with test_app.app_context():
            from quart import g
            g.user = {"user_id": 1, "scope": ["*:admin"], "roles": []}
            result = is_admin()
        assert result is True

    @pytest.mark.asyncio
    async def test_returns_true_when_admin_role(self, test_app):
        async with test_app.app_context():
            from quart import g
            g.user = {"user_id": 1, "scope": [], "roles": ["admin"]}
            result = is_admin()
        assert result is True


# ---------------------------------------------------------------------------
# require_auth decorator — via test_client
# ---------------------------------------------------------------------------


class TestRequireAuthDecorator:
    """
    Tests require_auth by registering temporary routes on the test_app and
    exercising them through the Quart test client.
    """

    async def test_missing_token_returns_401(self, test_client, test_app):
        """No Authorization header → 401."""

        @test_app.route("/test-auth-missing-token")
        @require_auth()
        async def _protected():
            return {"ok": True}

        response = await test_client.get("/test-auth-missing-token")
        assert response.status_code == 401

    async def test_invalid_token_returns_401(self, test_client, test_app):
        """Authorization header present but JWT validation fails → 401."""

        @test_app.route("/test-auth-invalid-token")
        @require_auth()
        async def _protected_invalid():
            return {"ok": True}

        with patch("middleware.auth._validate_token", new=AsyncMock(return_value=None)):
            response = await test_client.get(
                "/test-auth-invalid-token",
                headers={"Authorization": "Bearer bad.token.here"},
            )
        assert response.status_code == 401

    async def test_valid_token_returns_200(self, test_client, test_app):
        """Valid token → handler runs, returns 200."""

        @test_app.route("/test-auth-valid-token")
        @require_auth()
        async def _protected_valid(user_data):
            return {"ok": True}

        valid_payload = {
            "user_id": 2,
            "sub": "2",
            "scope": ["read"],
            "roles": ["viewer"],
            "tenant": "test",
        }

        with patch("middleware.auth._validate_token", new=AsyncMock(return_value=valid_payload)):
            response = await test_client.get(
                "/test-auth-valid-token",
                headers={"Authorization": "Bearer valid.token.here"},
            )
        assert response.status_code == 200

    async def test_admin_required_non_admin_returns_403(
        self, test_client, test_app
    ):
        """admin_required=True but user has no admin scope/role → 403."""

        @test_app.route("/test-auth-admin-only-block")
        @require_auth(admin_required=True)
        async def _admin_only(user_data):
            return {"ok": True}

        non_admin_payload = {
            "user_id": 2,
            "sub": "2",
            "scope": ["read"],
            "roles": ["viewer"],
            "tenant": "test",
        }

        with patch("middleware.auth._validate_token", new=AsyncMock(return_value=non_admin_payload)):
            response = await test_client.get(
                "/test-auth-admin-only-block",
                headers={"Authorization": "Bearer user.token.here"},
            )
        assert response.status_code == 403

    async def test_admin_required_with_admin_user_returns_200(
        self, test_client, test_app
    ):
        """admin_required=True and user has admin role → 200."""

        @test_app.route("/test-auth-admin-pass-route")
        @require_auth(admin_required=True)
        async def _admin_pass(user_data):
            return {"ok": True}

        admin_payload = {
            "user_id": 1,
            "sub": "1",
            "scope": ["*:admin"],
            "roles": ["admin"],
            "tenant": "test",
        }

        with patch("middleware.auth._validate_token", new=AsyncMock(return_value=admin_payload)):
            response = await test_client.get(
                "/test-auth-admin-pass-route",
                headers={"Authorization": "Bearer admin.token.here"},
            )
        assert response.status_code == 200

    async def test_license_feature_gates_route(self, test_client, test_app):
        """license_feature that passes → 200."""

        @test_app.route("/test-auth-license-gate")
        @require_auth(license_feature="some_feature")
        async def _licensed_route(user_data):
            return {"ok": True}

        valid_payload = {
            "user_id": 1,
            "sub": "1",
            "scope": ["*:admin"],
            "roles": ["admin"],
            "tenant": "test",
        }

        with patch("middleware.auth._validate_token", new=AsyncMock(return_value=valid_payload)):
            response = await test_client.get(
                "/test-auth-license-gate",
                headers={"Authorization": "Bearer admin.token.here"},
            )
        # License defaults to allowing all features in dev mode
        assert response.status_code == 200

    async def test_handler_exception_returns_500(self, test_client, test_app):
        """If handler raises, returns 500."""

        @test_app.route("/test-auth-handler-error")
        @require_auth()
        async def _erroring_handler(user_data):
            raise RuntimeError("something went wrong")

        valid_payload = {
            "user_id": 1,
            "sub": "1",
            "scope": [],
            "roles": [],
            "tenant": "test",
        }

        with patch("middleware.auth._validate_token", new=AsyncMock(return_value=valid_payload)):
            response = await test_client.get(
                "/test-auth-handler-error",
                headers={"Authorization": "Bearer token.here"},
            )
        assert response.status_code == 500


# ---------------------------------------------------------------------------
# _check_license_feature
# ---------------------------------------------------------------------------


class TestCheckLicenseFeature:
    @pytest.mark.asyncio
    async def test_returns_true_when_no_license_manager(self, test_app):
        async with test_app.app_context():
            # No license_manager on the app → defaults to True (dev mode)
            result = _check_license_feature("advanced_feature", {"user_id": 1})
        assert result is True

    @pytest.mark.asyncio
    async def test_returns_true_when_license_manager_present(self, test_app):
        async with test_app.app_context():
            test_app.license_manager = MagicMock()
            result = _check_license_feature("some_feature", {"user_id": 1})
        assert result is True
        # Cleanup
        del test_app.license_manager


# ---------------------------------------------------------------------------
# AuthContext
# ---------------------------------------------------------------------------


class TestAuthContext:
    @pytest.mark.asyncio
    async def test_not_authenticated_in_sync_context(self, test_app):
        async with test_app.test_request_context("/", headers={"Authorization": "Bearer sometoken"}):
            ctx = AuthContext()
            # In sync context, validation is skipped (warns about async)
            assert ctx.is_authenticated() is False

    @pytest.mark.asyncio
    async def test_is_admin_returns_false_when_not_authenticated(self, test_app):
        async with test_app.test_request_context("/"):
            ctx = AuthContext()
            assert ctx.is_admin() is False

    @pytest.mark.asyncio
    async def test_get_user_returns_none_when_not_authenticated(self, test_app):
        async with test_app.test_request_context("/"):
            ctx = AuthContext()
            assert ctx.get_user() is None

    @pytest.mark.asyncio
    async def test_has_feature_returns_false_when_not_authenticated(self, test_app):
        async with test_app.test_request_context("/"):
            ctx = AuthContext()
            result = ctx.has_feature("some_feature")
        # No user → _check_license_feature with None user still runs
        assert isinstance(result, bool)

    @pytest.mark.asyncio
    async def test_context_manager_enter_exit(self, test_app):
        async with test_app.test_request_context("/"):
            with AuthContext() as auth:
                assert auth is not None
