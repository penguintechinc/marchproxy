"""
Coverage tests for manager/models/auth_native.py

Targets uncovered lines:
- APITokenManager.validate_token (217-238): token found, token not found
- create_admin_user (255-273): admin exists, admin created
- require_permission decorator (348-361): denied/granted
- require_admin decorator (364-378): not user, not admin, is admin

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import pytest
from datetime import datetime, timedelta
from unittest.mock import MagicMock, patch, AsyncMock
import bcrypt


class TestAPITokenManager:
    """Tests for APITokenManager"""

    def setup_method(self):
        """Setup fixtures for each test"""
        self.mock_auth = MagicMock()
        self.mock_db = MagicMock()
        self.mock_auth.db = self.mock_db

    @patch("manager.models.auth_native.APITokenManager._setup_token_table")
    def test_create_token_with_ttl(self, mock_setup):
        """Test create_token with TTL"""
        from manager.models.auth_native import APITokenManager

        manager = APITokenManager(self.mock_auth)
        token, token_id = manager.create_token(1, "test-token", {"admin": True}, ttl_days=30)

        assert token is not None
        assert token_id is not None
        assert isinstance(token, str)
        assert isinstance(token_id, str)
        self.mock_db.api_tokens.insert.assert_called_once()

    @patch("manager.models.auth_native.APITokenManager._setup_token_table")
    def test_create_token_without_ttl(self, mock_setup):
        """Test create_token without TTL (infinite)"""
        from manager.models.auth_native import APITokenManager

        manager = APITokenManager(self.mock_auth)
        token, token_id = manager.create_token(1, "infinite-token", {"read": True})

        assert token is not None
        assert token_id is not None
        self.mock_db.api_tokens.insert.assert_called_once()
        # Verify expires_at is None
        call_kwargs = self.mock_db.api_tokens.insert.call_args[1]
        assert call_kwargs["expires_at"] is None


class TestCreateAdminUser:
    """Tests for create_admin_user function"""

    def test_create_admin_user_already_exists(self):
        """Test when admin user already exists"""
        from manager.models.auth_native import create_admin_user

        mock_auth = MagicMock()
        mock_db = MagicMock()
        mock_auth.db = mock_db

        # Mock existing user
        mock_existing_user = MagicMock()
        mock_existing_user.id = 5
        query_mock = MagicMock()
        query_mock.select.return_value.first.return_value = mock_existing_user
        mock_db.return_value = query_mock
        mock_db.auth_user = MagicMock()

        result = create_admin_user(mock_auth, email="admin@test.com")

        assert result == 5
        # Verify register not called
        mock_auth.register.assert_not_called()

    def test_create_admin_user_new(self):
        """Test creating a new admin user"""
        from manager.models.auth_native import create_admin_user

        mock_auth = MagicMock()
        mock_db = MagicMock()
        mock_auth.db = mock_db

        # Mock no existing user
        query_mock = MagicMock()
        query_mock.select.return_value.first.return_value = None
        mock_db.return_value = query_mock
        mock_db.auth_user = MagicMock()

        # Mock register call
        mock_auth.register.return_value = {"id": 10}

        # Mock user record
        mock_user = MagicMock()
        mock_db.auth_user.__getitem__.return_value = mock_user

        with patch("manager.models.auth_native.APITokenManager") as mock_token_mgr_cls:
            mock_token_mgr = MagicMock()
            mock_token_mgr.create_token.return_value = ("token-secret", "tok-001")
            mock_token_mgr_cls.return_value = mock_token_mgr

            result = create_admin_user(mock_auth, email="newadmin@test.com")

        assert result == 10
        # Verify register was called
        mock_auth.register.assert_called_once()
        # Verify user was updated with admin privileges
        mock_user.update_record.assert_called_once()
        update_call_kwargs = mock_user.update_record.call_args[1]
        assert update_call_kwargs["is_admin"] is True

    def test_create_admin_user_register_returns_no_id(self):
        """Test when register doesn't return user id"""
        from manager.models.auth_native import create_admin_user

        mock_auth = MagicMock()
        mock_db = MagicMock()
        mock_auth.db = mock_db

        # Mock no existing user
        query_mock = MagicMock()
        query_mock.select.return_value.first.return_value = None
        mock_db.return_value = query_mock
        mock_db.auth_user = MagicMock()

        # Mock register returning None for id
        mock_auth.register.return_value = {"id": None}

        result = create_admin_user(mock_auth)

        assert result is None


class TestRequirePermissionDecorator:
    """Tests for require_permission decorator"""

    def test_require_permission_denied(self):
        """Test require_permission denies access when check fails"""
        from manager.models.auth_native import require_permission

        mock_auth = MagicMock()

        with patch("manager.models.auth_native.check_permission", return_value=False):
            decorator = require_permission(mock_auth, "read_clusters")

            def test_func(*args, **kwargs):
                return "success"

            wrapped = decorator(test_func)

            with patch("quart.abort") as mock_abort:
                wrapped()
                mock_abort.assert_called_once_with(403)

    def test_require_permission_granted(self):
        """Test require_permission allows access when check succeeds"""
        from manager.models.auth_native import require_permission

        mock_auth = MagicMock()

        with patch("manager.models.auth_native.check_permission", return_value=True):
            decorator = require_permission(mock_auth, "read_clusters")

            def test_func(*args, **kwargs):
                return "success"

            wrapped = decorator(test_func)

            result = wrapped()
            assert result == "success"


class TestRequireAdminDecorator:
    """Tests for require_admin decorator"""

    def test_require_admin_no_user(self):
        """Test require_admin denies when user is None"""
        from manager.models.auth_native import require_admin

        mock_auth = MagicMock()
        mock_auth.get_user.return_value = None

        decorator = require_admin(mock_auth)

        def test_func(*args, **kwargs):
            return "success"

        wrapped = decorator(test_func)

        with patch("quart.abort") as mock_abort:
            wrapped()
            mock_abort.assert_called_once_with(403)

    def test_require_admin_not_admin(self):
        """Test require_admin denies when user is not admin"""
        from manager.models.auth_native import require_admin

        mock_auth = MagicMock()
        mock_auth.get_user.return_value = {"is_admin": False, "user_id": 1}

        decorator = require_admin(mock_auth)

        def test_func(*args, **kwargs):
            return "success"

        wrapped = decorator(test_func)

        with patch("quart.abort") as mock_abort:
            wrapped()
            mock_abort.assert_called_once_with(403)

    def test_require_admin_is_admin(self):
        """Test require_admin allows when user is admin"""
        from manager.models.auth_native import require_admin

        mock_auth = MagicMock()
        mock_auth.get_user.return_value = {"is_admin": True, "user_id": 1}

        decorator = require_admin(mock_auth)

        def test_func(*args, **kwargs):
            return "success"

        wrapped = decorator(test_func)

        result = wrapped()
        assert result == "success"
