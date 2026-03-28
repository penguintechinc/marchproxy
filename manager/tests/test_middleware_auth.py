"""
Unit tests for manager/middleware/auth.py

Tests JWT token extraction, validation, and the require_auth decorator.
No real JWT signing or database connections are used.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from middleware.auth import _extract_token_from_header, require_auth


pytestmark = pytest.mark.unit


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _make_request_ctx(auth_header: str = ""):
    """Return a mock Quart request with the given Authorization header."""
    req = MagicMock()
    req.headers = {"Authorization": auth_header} if auth_header else {}
    return req


# ---------------------------------------------------------------------------
# _extract_token_from_header
# ---------------------------------------------------------------------------


class TestExtractTokenFromHeader:
    def test_valid_bearer_token_returned(self):
        with patch("middleware.auth.request") as mock_req:
            mock_req.headers.get.return_value = "Bearer mytoken123"
            result = _extract_token_from_header()
        assert result == "mytoken123"

    def test_missing_header_returns_none(self):
        with patch("middleware.auth.request") as mock_req:
            mock_req.headers.get.return_value = ""
            result = _extract_token_from_header()
        assert result is None

    def test_no_authorization_header_returns_none(self):
        with patch("middleware.auth.request") as mock_req:
            mock_req.headers.get.return_value = None
            # Empty string from fallback ""
            mock_req.headers.get.side_effect = None
            mock_req.headers.get.return_value = ""
            result = _extract_token_from_header()
        assert result is None

    def test_malformed_header_single_word_returns_none(self):
        with patch("middleware.auth.request") as mock_req:
            mock_req.headers.get.return_value = "Bearer"
            result = _extract_token_from_header()
        assert result is None

    def test_malformed_header_wrong_scheme_returns_none(self):
        with patch("middleware.auth.request") as mock_req:
            mock_req.headers.get.return_value = "Basic dXNlcjpwYXNz"
            result = _extract_token_from_header()
        assert result is None

    def test_malformed_header_too_many_parts_returns_none(self):
        with patch("middleware.auth.request") as mock_req:
            mock_req.headers.get.return_value = "Bearer token extra"
            result = _extract_token_from_header()
        assert result is None

    def test_bearer_case_insensitive(self):
        with patch("middleware.auth.request") as mock_req:
            mock_req.headers.get.return_value = "BEARER mytoken456"
            result = _extract_token_from_header()
        assert result == "mytoken456"


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

        @test_app.route("/test-auth-missing")
        @require_auth()
        async def _protected():
            return {"ok": True}

        response = await test_client.get("/test-auth-missing")
        assert response.status_code == 401

    async def test_invalid_token_returns_401(self, test_client, test_app):
        """Authorization header present but JWT validation fails → 401."""

        @test_app.route("/test-auth-invalid")
        @require_auth()
        async def _protected_invalid():
            return {"ok": True}

        with patch.object(
            test_app, "jwt_manager", create=True
        ) as mock_jwt:
            mock_jwt.decode_token.side_effect = Exception("bad token")
            response = await test_client.get(
                "/test-auth-invalid",
                headers={"Authorization": "Bearer bad.token.here"},
            )
        assert response.status_code == 401

    async def test_valid_token_returns_200(self, test_client, test_app, user_payload):
        """Valid token → handler runs, returns 200."""

        @test_app.route("/test-auth-valid")
        @require_auth()
        async def _protected_valid():
            return {"ok": True}

        mock_jwt = MagicMock()
        mock_jwt.decode_token.return_value = user_payload

        with patch.object(test_app, "jwt_manager", mock_jwt, create=True):
            response = await test_client.get(
                "/test-auth-valid",
                headers={"Authorization": "Bearer valid.token.here"},
            )
        assert response.status_code == 200

    async def test_admin_required_non_admin_returns_403(
        self, test_client, test_app, user_payload
    ):
        """admin_required=True but user.is_admin is False → 403."""

        @test_app.route("/test-auth-admin-only")
        @require_auth(admin_required=True)
        async def _admin_only():
            return {"ok": True}

        # user_payload has is_admin=False
        mock_jwt = MagicMock()
        mock_jwt.decode_token.return_value = user_payload

        with patch.object(test_app, "jwt_manager", mock_jwt, create=True):
            response = await test_client.get(
                "/test-auth-admin-only",
                headers={"Authorization": "Bearer user.token.here"},
            )
        assert response.status_code == 403

    async def test_admin_required_with_admin_user_returns_200(
        self, test_client, test_app, admin_payload
    ):
        """admin_required=True and user.is_admin is True → 200."""

        @test_app.route("/test-auth-admin-pass")
        @require_auth(admin_required=True)
        async def _admin_pass():
            return {"ok": True}

        # admin_payload has is_admin=True
        mock_jwt = MagicMock()
        mock_jwt.decode_token.return_value = admin_payload

        with patch.object(test_app, "jwt_manager", mock_jwt, create=True):
            response = await test_client.get(
                "/test-auth-admin-pass",
                headers={"Authorization": "Bearer admin.token.here"},
            )
        assert response.status_code == 200
