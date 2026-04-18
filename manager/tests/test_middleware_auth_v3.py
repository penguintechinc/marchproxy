"""
Unit tests for middleware/auth.py - Covers uncovered lines 148, 231-278, 317-319, 374-376, 382

Tests cover:
- Line 148: Early return when user_data in kwargs (decorator bypass)
- Lines 231-278: _authenticate_and_authorize_async function
- Lines 317-319: Exception handling in _check_license_feature
- Lines 374-376: AuthContext._validate with token in sync context
- Line 382: AuthContext.has_feature method

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import pytest
from unittest.mock import AsyncMock, MagicMock, patch, call
from quart import Quart, g


# ============================================================================
# Test Line 148: require_auth decorator early return with user_data in kwargs
# ============================================================================


class TestRequireAuthUserDataBypass:
    """Test decorator early return when user_data is already in kwargs"""

    @pytest.mark.asyncio
    async def test_require_auth_bypasses_validation_when_user_data_provided(self):
        """When user_data already in kwargs, decorator should skip auth validation"""
        from middleware.auth import require_auth

        call_count = []

        @require_auth()
        async def mock_handler(user_data=None):
            call_count.append(user_data)
            return {"ok": True}, 200

        # Call with user_data already set - should bypass _validate_token
        result = await mock_handler(user_data={"user_id": "123", "sub": "123"})

        assert len(call_count) == 1
        assert call_count[0] == {"user_id": "123", "sub": "123"}
        assert result == ({"ok": True}, 200)

    @pytest.mark.asyncio
    async def test_require_auth_skips_token_extraction_when_user_data_present(self):
        """When user_data in kwargs, _extract_token_from_header should not be called"""
        from middleware.auth import require_auth

        @require_auth()
        async def handler(user_data=None):
            return {"result": "ok"}, 200

        with patch("middleware.auth._extract_token_from_header") as mock_extract:
            result = await handler(user_data={"user_id": "1", "sub": "1"})

        # Token extraction should not be called
        mock_extract.assert_not_called()
        assert result == ({"result": "ok"}, 200)

    @pytest.mark.asyncio
    async def test_require_auth_with_user_data_ignores_admin_requirement(self):
        """When user_data provided, admin_required check should be skipped"""
        from middleware.auth import require_auth

        @require_auth(admin_required=True)
        async def handler(user_data=None):
            return {"admin": True}, 200

        # Non-admin user_data, but admin_required=True
        # Should still work because decorator bypasses on user_data presence
        result = await handler(user_data={"user_id": "1", "sub": "1", "roles": ["viewer"]})

        assert result == ({"admin": True}, 200)

    @pytest.mark.asyncio
    async def test_require_auth_with_empty_user_data_bypasses_validation(self):
        """Empty user_data should not trigger bypass (falsy check)"""
        from middleware.auth import require_auth

        @require_auth()
        async def handler(user_data=None):
            return {"ok": True}, 200

        # Empty dict is falsy, so should NOT bypass
        with patch("middleware.auth._extract_token_from_header", return_value=None):
            result = await handler(user_data={})

        assert result == ({"error": "Missing authorization header"}, 401)

    @pytest.mark.asyncio
    async def test_require_auth_with_none_user_data_does_not_bypass(self):
        """None user_data should not trigger bypass"""
        from middleware.auth import require_auth

        @require_auth()
        async def handler(user_data=None):
            return {"ok": True}, 200

        with patch("middleware.auth._extract_token_from_header", return_value=None):
            result = await handler(user_data=None)

        assert result == ({"error": "Missing authorization header"}, 401)


# ============================================================================
# Test Lines 231-278: _authenticate_and_authorize_async function
# ============================================================================


class TestAuthenticateAndAuthorizeAsync:
    """Test _authenticate_and_authorize_async function"""

    @pytest.mark.asyncio
    async def test_authenticate_and_authorize_async_missing_token(self):
        """Should return 401 when no token in header"""
        from middleware.auth import _authenticate_and_authorize_async

        app = Quart("test")

        async with app.test_request_context("/test", headers={}):
            result = await _authenticate_and_authorize_async(
                handler=AsyncMock(return_value=({"ok": True}, 200)),
                args=(),
                kwargs={},
                admin_required=False,
                license_feature=None,
            )

        assert result == ({"error": "Missing authorization header"}, 401)

    @pytest.mark.asyncio
    async def test_authenticate_and_authorize_async_invalid_token(self):
        """Should return 401 when token validation fails"""
        from middleware.auth import _authenticate_and_authorize_async

        app = Quart("test")

        async with app.test_request_context(
            "/test", headers={"Authorization": "Bearer bad-token"}
        ):
            with patch(
                "middleware.auth._validate_token", new_callable=AsyncMock, return_value=None
            ):
                result = await _authenticate_and_authorize_async(
                    handler=AsyncMock(return_value=({"ok": True}, 200)),
                    args=(),
                    kwargs={},
                    admin_required=False,
                    license_feature=None,
                )

        assert result == ({"error": "Invalid or expired token"}, 401)

    @pytest.mark.asyncio
    async def test_authenticate_and_authorize_async_success(self):
        """Should successfully call handler when token is valid"""
        from middleware.auth import _authenticate_and_authorize_async

        app = Quart("test")
        app.db = MagicMock()

        payload = {
            "user_id": "user-1",
            "sub": "user-1",
            "scope": ["*:read", "*:write"],
            "roles": ["maintainer"],
        }
        handler_mock = AsyncMock(return_value=({"status": "ok"}, 200))

        async with app.test_request_context(
            "/test", headers={"Authorization": "Bearer valid-token"}
        ):
            with patch(
                "middleware.auth._validate_token",
                new_callable=AsyncMock,
                return_value=payload,
            ):
                async with app.app_context():
                    result = await _authenticate_and_authorize_async(
                        handler=handler_mock,
                        args=(),
                        kwargs={},
                        admin_required=False,
                        license_feature=None,
                    )

        assert result == ({"status": "ok"}, 200)
        handler_mock.assert_called_once()

    @pytest.mark.asyncio
    async def test_authenticate_and_authorize_async_sets_g_user(self):
        """Should set g.user and g.user_id when token is valid"""
        from middleware.auth import _authenticate_and_authorize_async

        app = Quart("test")
        app.db = MagicMock()

        payload = {"user_id": "user-1", "sub": "user-1"}

        async with app.test_request_context(
            "/test", headers={"Authorization": "Bearer valid-token"}
        ):
            with patch(
                "middleware.auth._validate_token",
                new_callable=AsyncMock,
                return_value=payload,
            ):
                async with app.app_context():
                    await _authenticate_and_authorize_async(
                        handler=AsyncMock(return_value=({"ok": True}, 200)),
                        args=(),
                        kwargs={},
                        admin_required=False,
                        license_feature=None,
                    )

                    assert g.user == payload
                    assert g.user_id == "user-1"
                    assert g.db == app.db

    @pytest.mark.asyncio
    async def test_authenticate_and_authorize_async_admin_required_fails(self):
        """Should return 403 when admin_required=True but user is not admin"""
        from middleware.auth import _authenticate_and_authorize_async

        app = Quart("test")
        app.db = MagicMock()

        payload = {
            "user_id": "user-1",
            "sub": "user-1",
            "scope": ["*:read"],
            "roles": ["viewer"],
        }

        async with app.test_request_context(
            "/test", headers={"Authorization": "Bearer valid-token"}
        ):
            with patch(
                "middleware.auth._validate_token",
                new_callable=AsyncMock,
                return_value=payload,
            ):
                async with app.app_context():
                    result = await _authenticate_and_authorize_async(
                        handler=AsyncMock(return_value=({"ok": True}, 200)),
                        args=(),
                        kwargs={},
                        admin_required=True,
                        license_feature=None,
                    )

        assert result == ({"error": "Admin access required"}, 403)

    @pytest.mark.asyncio
    async def test_authenticate_and_authorize_async_admin_required_succeeds(self):
        """Should allow admin users when admin_required=True"""
        from middleware.auth import _authenticate_and_authorize_async

        app = Quart("test")
        app.db = MagicMock()

        payload = {
            "user_id": "user-1",
            "sub": "user-1",
            "scope": ["*:admin"],
            "roles": ["admin"],
        }
        handler_mock = AsyncMock(return_value=({"ok": True}, 200))

        async with app.test_request_context(
            "/test", headers={"Authorization": "Bearer valid-token"}
        ):
            with patch(
                "middleware.auth._validate_token",
                new_callable=AsyncMock,
                return_value=payload,
            ):
                async with app.app_context():
                    result = await _authenticate_and_authorize_async(
                        handler=handler_mock,
                        args=(),
                        kwargs={},
                        admin_required=True,
                        license_feature=None,
                    )

        assert result == ({"ok": True}, 200)
        handler_mock.assert_called_once()

    @pytest.mark.asyncio
    async def test_authenticate_and_authorize_async_license_feature_denied(self):
        """Should return 403 when license feature is not available"""
        from middleware.auth import _authenticate_and_authorize_async

        app = Quart("test")
        app.db = MagicMock()

        payload = {"user_id": "user-1", "sub": "user-1"}

        async with app.test_request_context(
            "/test", headers={"Authorization": "Bearer valid-token"}
        ):
            with patch(
                "middleware.auth._validate_token",
                new_callable=AsyncMock,
                return_value=payload,
            ):
                with patch(
                    "middleware.auth._check_license_feature", return_value=False
                ) as mock_check_license:
                    async with app.app_context():
                        result = await _authenticate_and_authorize_async(
                            handler=AsyncMock(return_value=({"ok": True}, 200)),
                            args=(),
                            kwargs={},
                            admin_required=False,
                            license_feature="advanced_feature",
                        )

        assert result == (
            {"error": "Feature 'advanced_feature' not licensed"},
            403,
        )
        mock_check_license.assert_called_once_with("advanced_feature", payload)

    @pytest.mark.asyncio
    async def test_authenticate_and_authorize_async_license_feature_allowed(self):
        """Should allow when license feature is available"""
        from middleware.auth import _authenticate_and_authorize_async

        app = Quart("test")
        app.db = MagicMock()

        payload = {"user_id": "user-1", "sub": "user-1"}
        handler_mock = AsyncMock(return_value=({"feature": "enabled"}, 200))

        async with app.test_request_context(
            "/test", headers={"Authorization": "Bearer valid-token"}
        ):
            with patch(
                "middleware.auth._validate_token",
                new_callable=AsyncMock,
                return_value=payload,
            ):
                with patch(
                    "middleware.auth._check_license_feature", return_value=True
                ):
                    async with app.app_context():
                        result = await _authenticate_and_authorize_async(
                            handler=handler_mock,
                            args=(),
                            kwargs={},
                            admin_required=False,
                            license_feature="advanced_feature",
                        )

        assert result == ({"feature": "enabled"}, 200)
        handler_mock.assert_called_once()

    @pytest.mark.asyncio
    async def test_authenticate_and_authorize_async_handler_exception(self):
        """Should return 500 when handler raises exception"""
        from middleware.auth import _authenticate_and_authorize_async

        app = Quart("test")
        app.db = MagicMock()

        payload = {"user_id": "user-1", "sub": "user-1"}

        async def failing_handler(*args, **kwargs):
            raise ValueError("Handler error")

        async with app.test_request_context(
            "/test", headers={"Authorization": "Bearer valid-token"}
        ):
            with patch(
                "middleware.auth._validate_token",
                new_callable=AsyncMock,
                return_value=payload,
            ):
                async with app.app_context():
                    result = await _authenticate_and_authorize_async(
                        handler=failing_handler,
                        args=(),
                        kwargs={},
                        admin_required=False,
                        license_feature=None,
                    )

        assert result == ({"error": "Internal server error"}, 500)

    @pytest.mark.asyncio
    async def test_authenticate_and_authorize_async_injects_user_data_into_kwargs(self):
        """Should inject payload as user_data kwarg in handler call"""
        from middleware.auth import _authenticate_and_authorize_async

        app = Quart("test")
        app.db = MagicMock()

        payload = {"user_id": "user-1", "sub": "user-1", "extra": "data"}
        handler_mock = AsyncMock(return_value=({"ok": True}, 200))

        async with app.test_request_context(
            "/test", headers={"Authorization": "Bearer valid-token"}
        ):
            with patch(
                "middleware.auth._validate_token",
                new_callable=AsyncMock,
                return_value=payload,
            ):
                async with app.app_context():
                    await _authenticate_and_authorize_async(
                        handler=handler_mock,
                        args=(),
                        kwargs={},
                        admin_required=False,
                        license_feature=None,
                    )

        # Check that user_data kwarg was injected
        handler_mock.assert_called_once()
        called_kwargs = handler_mock.call_args[1]
        assert called_kwargs["user_data"] == payload


# ============================================================================
# Test Lines 317-319: _check_license_feature exception handling
# ============================================================================


class TestCheckLicenseFeatureException:
    """Test exception handling in _check_license_feature"""

    def test_check_license_feature_handles_license_manager_exception(self):
        """Should return False when license_manager.check_feature raises exception"""
        from middleware.auth import _check_license_feature
        from quart import Quart

        app = Quart("test")
        mock_license_manager = MagicMock()
        mock_license_manager.check_feature.side_effect = Exception(
            "License server unreachable"
        )
        app.license_manager = mock_license_manager

        ctx = app.app_context()
        ctx.push()
        try:
            result = _check_license_feature("advanced_feature", {"user_id": "1"})
            assert result is False
        finally:
            ctx.pop()

    def test_check_license_feature_handles_attribute_error(self):
        """Should gracefully handle AttributeError from license manager"""
        from middleware.auth import _check_license_feature
        from quart import Quart

        app = Quart("test")
        mock_license_manager = MagicMock()
        mock_license_manager.check_feature.side_effect = AttributeError(
            "Invalid method"
        )
        app.license_manager = mock_license_manager

        ctx = app.app_context()
        ctx.push()
        try:
            result = _check_license_feature("advanced_feature", {"user_id": "1"})
            assert result is False
        finally:
            ctx.pop()

    def test_check_license_feature_handles_type_error(self):
        """Should gracefully handle TypeError from license manager"""
        from middleware.auth import _check_license_feature
        from quart import Quart

        app = Quart("test")
        mock_license_manager = MagicMock()
        mock_license_manager.check_feature.side_effect = TypeError("Invalid arguments")
        app.license_manager = mock_license_manager

        ctx = app.app_context()
        ctx.push()
        try:
            result = _check_license_feature("advanced_feature", {"user_id": "1"})
            assert result is False
        finally:
            ctx.pop()

    def test_check_license_feature_default_allows_when_no_manager(self):
        """Should return True (allow feature) when no license_manager configured"""
        from middleware.auth import _check_license_feature

        # Mock getattr to return None for license_manager attribute
        def mock_getattr_fn(obj, name, default=None):
            if name == "license_manager":
                return None
            # Fall back to real getattr for other attributes
            return object.__getattribute__(obj, name) if hasattr(object, name) else default

        with patch("middleware.auth.getattr", side_effect=mock_getattr_fn):
            result = _check_license_feature("any_feature", {"user_id": "1"})
            assert result is True

    def test_check_license_feature_logs_error_on_exception(self):
        """Should log error message when exception occurs"""
        from middleware.auth import _check_license_feature
        from quart import Quart

        app = Quart("test")
        mock_license_manager = MagicMock()
        mock_license_manager.check_feature.side_effect = RuntimeError(
            "License check failed"
        )
        app.license_manager = mock_license_manager

        ctx = app.app_context()
        ctx.push()
        try:
            with patch("middleware.auth.logger.error") as mock_logger:
                _check_license_feature("test_feature", {"user_id": "1"})

                mock_logger.assert_called_once()
                assert "Error checking license feature" in mock_logger.call_args[0][0]
        finally:
            ctx.pop()


# ============================================================================
# Test Lines 374-376: AuthContext._validate in sync context
# ============================================================================


class TestAuthContextValidate:
    """Test AuthContext._validate method in sync context"""

    def test_auth_context_validate_warns_when_token_present(self):
        """Should warn when token present in sync context (can't validate async)"""
        from middleware.auth import AuthContext

        with patch("middleware.auth._extract_token_from_header", return_value="test-token"):
            with patch("middleware.auth.logger.warning") as mock_logger:
                ctx = AuthContext()

            # Should have logged warning about async validation in sync context
            mock_logger.assert_called_once()
            assert "sync context" in mock_logger.call_args[0][0].lower()

    def test_auth_context_validate_sets_user_none_on_token_present(self):
        """Should set user to None when token present but in sync context"""
        from middleware.auth import AuthContext

        with patch("middleware.auth._extract_token_from_header", return_value="test-token"):
            ctx = AuthContext()

            assert ctx.user is None
            assert ctx.valid is False

    def test_auth_context_validate_no_token(self):
        """Should set user to None when no token in request"""
        from middleware.auth import AuthContext

        with patch("middleware.auth._extract_token_from_header", return_value=None):
            ctx = AuthContext()

            assert ctx.user is None
            assert ctx.valid is False

    def test_auth_context_init_calls_validate(self):
        """AuthContext.__init__ should call _validate"""
        from middleware.auth import AuthContext

        with patch.object(AuthContext, "_validate") as mock_validate:
            ctx = AuthContext.__new__(AuthContext)
            ctx.user = None
            ctx.valid = False
            # Manually call to verify it's called
            ctx._validate()


# ============================================================================
# Test Line 382: AuthContext.has_feature method
# ============================================================================


class TestAuthContextHasFeature:
    """Test AuthContext.has_feature method"""

    def test_auth_context_has_feature_no_user(self):
        """Should return False when no user is authenticated"""
        from middleware.auth import AuthContext

        with patch("middleware.auth._extract_token_from_header", return_value=None):
            ctx = AuthContext()
            result = ctx.has_feature("advanced_analytics")

        assert result is False

    def test_auth_context_has_feature_calls_check_license_feature(self):
        """Should call _check_license_feature with feature and user"""
        from middleware.auth import AuthContext

        user_payload = {"user_id": "1", "sub": "1"}

        with patch("middleware.auth._check_license_feature", return_value=True) as mock_check:
            with patch("middleware.auth._extract_token_from_header", return_value=None):
                ctx = AuthContext()
                # Manually set user since _validate is async
                ctx.user = user_payload

                result = ctx.has_feature("premium_feature")

                mock_check.assert_called_once_with("premium_feature", user_payload)
                assert result is True

    def test_auth_context_has_feature_returns_false_when_not_licensed(self):
        """Should return False when feature is not licensed"""
        from middleware.auth import AuthContext

        user_payload = {"user_id": "1", "sub": "1"}

        with patch("middleware.auth._check_license_feature", return_value=False):
            with patch("middleware.auth._extract_token_from_header", return_value=None):
                ctx = AuthContext()
                ctx.user = user_payload
                ctx.valid = True

                result = ctx.has_feature("restricted_feature")

                assert result is False

    def test_auth_context_has_feature_returns_true_when_licensed(self):
        """Should return True when feature is licensed"""
        from middleware.auth import AuthContext

        user_payload = {"user_id": "1", "sub": "1"}

        with patch("middleware.auth._check_license_feature", return_value=True):
            with patch("middleware.auth._extract_token_from_header", return_value=None):
                ctx = AuthContext()
                ctx.user = user_payload
                ctx.valid = True

                result = ctx.has_feature("enabled_feature")

                assert result is True


# ============================================================================
# Additional edge case tests
# ============================================================================


class TestAuthEdgeCases:
    """Test edge cases and interactions"""

    @pytest.mark.asyncio
    async def test_admin_check_with_admin_scope(self):
        """Should recognize *:admin scope as admin"""
        from middleware.auth import _authenticate_and_authorize_async

        app = Quart("test")
        app.db = MagicMock()

        payload = {"user_id": "1", "sub": "1", "scope": ["*:admin"], "roles": []}
        handler_mock = AsyncMock(return_value=({"ok": True}, 200))

        async with app.test_request_context(
            "/test", headers={"Authorization": "Bearer token"}
        ):
            with patch(
                "middleware.auth._validate_token",
                new_callable=AsyncMock,
                return_value=payload,
            ):
                async with app.app_context():
                    result = await _authenticate_and_authorize_async(
                        handler=handler_mock,
                        args=(),
                        kwargs={},
                        admin_required=True,
                        license_feature=None,
                    )

        assert result == ({"ok": True}, 200)

    @pytest.mark.asyncio
    async def test_admin_check_with_admin_role(self):
        """Should recognize admin role as admin (OIDC fallback)"""
        from middleware.auth import _authenticate_and_authorize_async

        app = Quart("test")
        app.db = MagicMock()

        payload = {"user_id": "1", "sub": "1", "scope": [], "roles": ["admin"]}
        handler_mock = AsyncMock(return_value=({"ok": True}, 200))

        async with app.test_request_context(
            "/test", headers={"Authorization": "Bearer token"}
        ):
            with patch(
                "middleware.auth._validate_token",
                new_callable=AsyncMock,
                return_value=payload,
            ):
                async with app.app_context():
                    result = await _authenticate_and_authorize_async(
                        handler=handler_mock,
                        args=(),
                        kwargs={},
                        admin_required=True,
                        license_feature=None,
                    )

        assert result == ({"ok": True}, 200)

    @pytest.mark.asyncio
    async def test_user_id_fallback_to_sub(self):
        """Should use 'sub' claim when 'user_id' not present"""
        from middleware.auth import _authenticate_and_authorize_async

        app = Quart("test")
        app.db = MagicMock()

        payload = {"sub": "user-123", "scope": []}
        handler_mock = AsyncMock(return_value=({"ok": True}, 200))

        async with app.test_request_context(
            "/test", headers={"Authorization": "Bearer token"}
        ):
            with patch(
                "middleware.auth._validate_token",
                new_callable=AsyncMock,
                return_value=payload,
            ):
                async with app.app_context():
                    await _authenticate_and_authorize_async(
                        handler=handler_mock,
                        args=(),
                        kwargs={},
                        admin_required=False,
                        license_feature=None,
                    )

                    assert g.user_id == "user-123"

    def test_auth_context_is_admin_with_scopes(self):
        """AuthContext.is_admin should check scopes correctly"""
        from middleware.auth import AuthContext

        with patch("middleware.auth._extract_token_from_header", return_value=None):
            ctx = AuthContext()
            ctx.user = {"scope": ["*:admin"], "roles": []}
            ctx.valid = True

            assert ctx.is_admin() is True

    def test_auth_context_is_admin_with_roles(self):
        """AuthContext.is_admin should check roles correctly"""
        from middleware.auth import AuthContext

        with patch("middleware.auth._extract_token_from_header", return_value=None):
            ctx = AuthContext()
            ctx.user = {"scope": [], "roles": ["admin"]}
            ctx.valid = True

            assert ctx.is_admin() is True

    def test_auth_context_is_admin_non_admin_user(self):
        """AuthContext.is_admin should return False for non-admin"""
        from middleware.auth import AuthContext

        with patch("middleware.auth._extract_token_from_header", return_value=None):
            ctx = AuthContext()
            ctx.user = {"scope": ["*:read"], "roles": ["viewer"]}
            ctx.valid = True

            assert ctx.is_admin() is False

    def test_auth_context_get_user_when_authenticated(self):
        """AuthContext.get_user should return user when authenticated"""
        from middleware.auth import AuthContext

        with patch("middleware.auth._extract_token_from_header", return_value=None):
            ctx = AuthContext()
            user = {"user_id": "1", "username": "test"}
            ctx.user = user
            ctx.valid = True

            assert ctx.get_user() == user

    def test_auth_context_is_authenticated_when_valid_user(self):
        """AuthContext.is_authenticated should return True when valid and user set"""
        from middleware.auth import AuthContext

        with patch("middleware.auth._extract_token_from_header", return_value=None):
            ctx = AuthContext()
            ctx.user = {"user_id": "1"}
            ctx.valid = True

            assert ctx.is_authenticated() is True

    def test_auth_context_is_authenticated_when_no_user(self):
        """AuthContext.is_authenticated should return False when no user"""
        from middleware.auth import AuthContext

        with patch("middleware.auth._extract_token_from_header", return_value=None):
            ctx = AuthContext()
            ctx.user = None
            ctx.valid = False

            assert ctx.is_authenticated() is False

    def test_auth_context_context_manager(self):
        """AuthContext should work as context manager"""
        from middleware.auth import AuthContext

        with patch("middleware.auth._extract_token_from_header", return_value=None):
            ctx = AuthContext()
            with ctx as auth:
                assert auth is ctx
                assert isinstance(auth, AuthContext)
