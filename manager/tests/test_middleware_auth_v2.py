"""
Tests for authentication middleware

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import pytest
from unittest.mock import MagicMock, patch, AsyncMock
from middleware.auth import (
    get_current_user,
    is_admin,
    _extract_token_from_header,
    _validate_token,
    require_auth,
    _check_license_feature,
    AuthContext,
)


class TestGetCurrentUser:
    """Test get_current_user function"""

    def test_get_current_user_present(self):
        """Test getting current user when present in context"""
        # Create a mock g object with user attribute
        mock_g_user = {"user_id": 123, "username": "testuser"}

        with patch("middleware.auth.g", MagicMock(user=mock_g_user)):
            result = get_current_user()
            assert result == {"user_id": 123, "username": "testuser"}

    def test_get_current_user_absent(self):
        """Test getting current user when absent from context"""
        with patch("middleware.auth.getattr", return_value=None):
            result = get_current_user()
            assert result is None


class TestIsAdmin:
    """Test is_admin function"""

    def test_is_admin_with_admin_scope(self):
        """Test is_admin returns True when user has admin scope"""
        with patch("middleware.auth.get_current_user") as mock_get_user:
            mock_get_user.return_value = {
                "user_id": 123,
                "scope": ["*:admin", "*:read", "*:write"],
                "roles": ["service_owner"],
            }

            result = is_admin()

            assert result is True

    def test_is_admin_with_admin_role(self):
        """Test is_admin returns True when user has admin role"""
        with patch("middleware.auth.get_current_user") as mock_get_user:
            mock_get_user.return_value = {
                "user_id": 123,
                "scope": ["*:read", "*:write"],
                "roles": ["admin"],
            }

            result = is_admin()

            assert result is True

    def test_is_admin_without_admin(self):
        """Test is_admin returns False when user is not admin"""
        with patch("middleware.auth.get_current_user") as mock_get_user:
            mock_get_user.return_value = {
                "user_id": 123,
                "scope": ["*:read"],
                "roles": ["viewer"],
            }

            result = is_admin()

            assert result is False

    def test_is_admin_no_user(self):
        """Test is_admin returns False when no user is present"""
        with patch("middleware.auth.get_current_user") as mock_get_user:
            mock_get_user.return_value = None

            result = is_admin()

            assert result is False


class TestExtractTokenFromHeader:
    """Test _extract_token_from_header function"""

    def test_extract_token_valid(self):
        """Test extracting valid Bearer token"""
        mock_request = MagicMock()
        mock_request.headers.get.return_value = "Bearer test-token-123"

        with patch("middleware.auth.request", mock_request):
            result = _extract_token_from_header()
            assert result == "test-token-123"

    def test_extract_token_missing_header(self):
        """Test extracting token when header missing"""
        mock_request = MagicMock()
        mock_request.headers.get.return_value = ""

        with patch("middleware.auth.request", mock_request):
            result = _extract_token_from_header()
            assert result is None

    def test_extract_token_invalid_format(self):
        """Test extracting token with invalid format"""
        mock_request = MagicMock()
        mock_request.headers.get.return_value = "InvalidFormat token"

        with patch("middleware.auth.request", mock_request):
            result = _extract_token_from_header()
            assert result is None

    def test_extract_token_missing_token_part(self):
        """Test extracting token when Bearer keyword without token"""
        mock_request = MagicMock()
        mock_request.headers.get.return_value = "Bearer"

        with patch("middleware.auth.request", mock_request):
            result = _extract_token_from_header()
            assert result is None

    def test_extract_token_case_insensitive(self):
        """Test Bearer keyword is case-insensitive"""
        mock_request = MagicMock()
        mock_request.headers.get.return_value = "bearer test-token-123"

        with patch("middleware.auth.request", mock_request):
            result = _extract_token_from_header()
            assert result == "test-token-123"


class TestValidateToken:
    """Test _validate_token function"""

    @pytest.mark.asyncio
    async def test_validate_token_success(self):
        """Test successful token validation"""
        mock_claims = MagicMock()
        mock_claims.model_dump.return_value = {
            "user_id": 123,
            "username": "testuser",
            "scope": ["*:admin"],
        }

        mock_oidc_rp = AsyncMock()
        mock_oidc_rp.validate_token = AsyncMock(return_value=mock_claims)

        mock_app = MagicMock()
        mock_app.oidc_rp = mock_oidc_rp

        with patch("middleware.auth.current_app", mock_app):
            result = await _validate_token("test-token")
            assert result == {"user_id": 123, "username": "testuser", "scope": ["*:admin"]}

    @pytest.mark.asyncio
    async def test_validate_token_no_oidc_rp(self):
        """Test token validation when OIDC RP not configured"""
        mock_app = MagicMock()
        mock_app.oidc_rp = None

        with patch("middleware.auth.current_app", mock_app):
            result = await _validate_token("test-token")
            assert result is None

    @pytest.mark.asyncio
    async def test_validate_token_invalid_token(self):
        """Test token validation with invalid token"""
        mock_oidc_rp = AsyncMock()
        mock_oidc_rp.validate_token = AsyncMock(side_effect=Exception("Invalid token"))

        mock_app = MagicMock()
        mock_app.oidc_rp = mock_oidc_rp

        with patch("middleware.auth.current_app", mock_app):
            result = await _validate_token("invalid-token")
            assert result is None


class TestRequireAuthDecorator:
    """Test require_auth decorator"""

    @pytest.mark.asyncio
    async def test_require_auth_success(self):
        """Test require_auth with valid token"""
        async def mock_handler(user_data=None):
            return {"success": True, "user_id": user_data["user_id"]}

        decorated = require_auth()(mock_handler)

        mock_claims = MagicMock()
        mock_claims.model_dump.return_value = {"user_id": 123, "scope": ["*:read"]}

        mock_oidc_rp = MagicMock()
        mock_oidc_rp.validate_token = AsyncMock(return_value=mock_claims)

        with patch("middleware.auth._extract_token_from_header", return_value="valid-token"):
            with patch("middleware.auth._validate_token", return_value={"user_id": 123, "scope": ["*:read"]}):
                result = await decorated()

                assert result == {"success": True, "user_id": 123}

    @pytest.mark.asyncio
    async def test_require_auth_missing_token(self):
        """Test require_auth with missing token"""
        async def mock_handler(user_data=None):
            return {"success": True}

        decorated = require_auth()(mock_handler)

        with patch("middleware.auth._extract_token_from_header", return_value=None):
            result = await decorated()

            assert result == ({"error": "Missing authorization header"}, 401)

    @pytest.mark.asyncio
    async def test_require_auth_invalid_token(self):
        """Test require_auth with invalid token"""
        async def mock_handler(user_data=None):
            return {"success": True}

        decorated = require_auth()(mock_handler)

        with patch("middleware.auth._extract_token_from_header", return_value="invalid-token"):
            with patch("middleware.auth._validate_token", return_value=None):
                result = await decorated()

                assert result == ({"error": "Invalid or expired token"}, 401)

    @pytest.mark.asyncio
    async def test_require_auth_admin_required_success(self):
        """Test require_auth with admin_required and valid admin user"""
        async def mock_handler(user_data=None):
            return {"success": True}

        decorated = require_auth(admin_required=True)(mock_handler)

        with patch("middleware.auth._extract_token_from_header", return_value="admin-token"):
            with patch("middleware.auth._validate_token", return_value={"user_id": 123, "scope": ["*:admin"], "roles": ["admin"]}):
                result = await decorated()

                assert result == {"success": True}

    @pytest.mark.asyncio
    async def test_require_auth_admin_required_non_admin(self):
        """Test require_auth with admin_required but user is not admin"""
        async def mock_handler(user_data=None):
            return {"success": True}

        decorated = require_auth(admin_required=True)(mock_handler)

        with patch("middleware.auth._extract_token_from_header", return_value="user-token"):
            with patch("middleware.auth._validate_token", return_value={"user_id": 123, "scope": ["*:read"], "roles": ["viewer"]}):
                result = await decorated()

                assert result == ({"error": "Admin access required"}, 403)

    @pytest.mark.asyncio
    async def test_require_auth_license_feature(self):
        """Test require_auth with license_feature check"""
        async def mock_handler(user_data=None):
            return {"success": True}

        decorated = require_auth(license_feature="advanced_feature")(mock_handler)

        with patch("middleware.auth._extract_token_from_header", return_value="valid-token"):
            with patch("middleware.auth._validate_token", return_value={"user_id": 123, "scope": ["*:read"]}):
                with patch("middleware.auth._check_license_feature", return_value=True):
                    result = await decorated()

                    assert result == {"success": True}

    @pytest.mark.asyncio
    async def test_require_auth_license_feature_not_available(self):
        """Test require_auth when license feature not available"""
        async def mock_handler(user_data=None):
            return {"success": True}

        decorated = require_auth(license_feature="premium_feature")(mock_handler)

        with patch("middleware.auth._extract_token_from_header", return_value="valid-token"):
            with patch("middleware.auth._validate_token", return_value={"user_id": 123, "scope": ["*:read"]}):
                with patch("middleware.auth._check_license_feature", return_value=False):
                    result = await decorated()

                    assert result == ({"error": "Feature 'premium_feature' not licensed"}, 403)

    @pytest.mark.asyncio
    async def test_require_auth_handler_exception(self):
        """Test require_auth handles exceptions in handler"""
        async def mock_handler(user_data=None):
            raise Exception("Handler error")

        decorated = require_auth()(mock_handler)

        with patch("middleware.auth._extract_token_from_header", return_value="valid-token"):
            with patch("middleware.auth._validate_token", return_value={"user_id": 123, "scope": ["*:read"]}):
                result = await decorated()

                assert result == ({"error": "Internal server error"}, 500)


class TestCheckLicenseFeature:
    """Test _check_license_feature function"""

    def test_check_license_feature_no_manager(self):
        """Test license feature check when no license manager configured"""
        mock_app = MagicMock()
        mock_app.license_manager = None

        with patch("middleware.auth.current_app", mock_app):
            result = _check_license_feature("test_feature", {"user_id": 123})
            assert result is True  # Default allows all features

    def test_check_license_feature_with_manager(self):
        """Test license feature check with license manager"""
        mock_license_manager = MagicMock()
        mock_app = MagicMock()
        mock_app.license_manager = mock_license_manager

        with patch("middleware.auth.current_app", mock_app):
            result = _check_license_feature("test_feature", {"user_id": 123})
            assert result is True  # Currently allows all features

    def test_check_license_feature_exception(self):
        """Test license feature check returns True by default (development mode)"""
        mock_license_manager = MagicMock()
        mock_license_manager.check_feature = MagicMock(side_effect=Exception("License check failed"))
        mock_app = MagicMock()
        mock_app.license_manager = mock_license_manager

        with patch("middleware.auth.current_app", mock_app):
            result = _check_license_feature("test_feature", {"user_id": 123})
            # Implementation returns True by default (development mode)
            assert result is True


class TestAuthContext:
    """Test AuthContext class"""

    def test_auth_context_init(self):
        """Test AuthContext initialization"""
        with patch("middleware.auth._extract_token_from_header", return_value=None):
            context = AuthContext()

            assert context.user is None
            assert context.valid is False

    def test_auth_context_is_authenticated_false(self):
        """Test AuthContext.is_authenticated when not authenticated"""
        with patch("middleware.auth._extract_token_from_header", return_value=None):
            context = AuthContext()

            assert context.is_authenticated() is False

    def test_auth_context_context_manager(self):
        """Test AuthContext as context manager"""
        with patch("middleware.auth._extract_token_from_header", return_value=None):
            with AuthContext() as auth:
                assert auth is not None
                assert auth.is_authenticated() is False
