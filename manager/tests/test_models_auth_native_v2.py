"""
Unit tests for auth_native model

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import io
import base64
from datetime import datetime, timedelta
from unittest.mock import MagicMock, patch, Mock
import pytest


class TestTOTPManager:
    """Test TOTPManager class"""

    def setup_method(self):
        """Setup test fixtures"""
        self.mock_auth = MagicMock()
        self.mock_db = MagicMock()
        self.mock_auth.db = self.mock_db

    def test_enable_2fa_user_not_found(self):
        """Test enable_2fa with non-existent user"""
        from manager.models.auth_native import TOTPManager

        self.mock_db.auth_user.__getitem__.return_value = None
        manager = TOTPManager(self.mock_auth)

        result = manager.enable_2fa(999, "password123")
        assert result is None

    def test_enable_2fa_password_verification_fails(self):
        """Test enable_2fa with incorrect password"""
        from manager.models.auth_native import TOTPManager

        mock_user = MagicMock()
        mock_user.password = "hashed_password"
        self.mock_db.auth_user.__getitem__.return_value = mock_user
        self.mock_auth.verify_password.return_value = False

        manager = TOTPManager(self.mock_auth)
        result = manager.enable_2fa(1, "wrong_password")

        assert result is None
        self.mock_auth.verify_password.assert_called_once()

    @patch('manager.models.auth_native.pyotp.random_base32')
    @patch('manager.models.auth_native.pyotp.TOTP')
    def test_enable_2fa_success(self, mock_totp_class, mock_random_base32):
        """Test enable_2fa success flow"""
        from manager.models.auth_native import TOTPManager

        mock_user = MagicMock()
        mock_user.password = "hashed_password"
        mock_user.email = "test@example.com"
        self.mock_db.auth_user.__getitem__.return_value = mock_user
        self.mock_auth.verify_password.return_value = True
        mock_random_base32.return_value = "TESTSECRET123"

        mock_totp = MagicMock()
        mock_totp.provisioning_uri.return_value = "otpauth://totp/test@example.com?secret=TESTSECRET123"
        mock_totp_class.return_value = mock_totp

        manager = TOTPManager(self.mock_auth)

        with patch.object(manager, '_generate_qr_code', return_value='base64qrcode'):
            result = manager.enable_2fa(1, "correct_password")

        assert result is not None
        assert 'secret' in result
        assert result['secret'] == 'TESTSECRET123'
        assert 'qr_uri' in result
        mock_user.update_record.assert_called_once()

    def test_verify_and_complete_2fa_user_not_found(self):
        """Test verify_and_complete_2fa with non-existent user"""
        from manager.models.auth_native import TOTPManager

        self.mock_db.auth_user.__getitem__.return_value = None
        manager = TOTPManager(self.mock_auth)

        result = manager.verify_and_complete_2fa(999, "TESTSECRET", "123456")
        assert result is False

    def test_verify_and_complete_2fa_secret_mismatch(self):
        """Test verify_and_complete_2fa with mismatched secret"""
        from manager.models.auth_native import TOTPManager

        mock_user = MagicMock()
        mock_user.totp_secret = "DIFFERENTSECRET"
        self.mock_db.auth_user.__getitem__.return_value = mock_user

        manager = TOTPManager(self.mock_auth)
        result = manager.verify_and_complete_2fa(1, "TESTSECRET", "123456")

        assert result is False

    @patch('manager.models.auth_native.pyotp.TOTP')
    def test_verify_and_complete_2fa_invalid_code(self, mock_totp_class):
        """Test verify_and_complete_2fa with invalid TOTP code"""
        from manager.models.auth_native import TOTPManager

        mock_user = MagicMock()
        mock_user.totp_secret = "TESTSECRET"
        self.mock_db.auth_user.__getitem__.return_value = mock_user

        mock_totp = MagicMock()
        mock_totp.verify.return_value = False
        mock_totp_class.return_value = mock_totp

        manager = TOTPManager(self.mock_auth)
        result = manager.verify_and_complete_2fa(1, "TESTSECRET", "000000")

        assert result is False
        mock_totp.verify.assert_called_once()

    @patch('manager.models.auth_native.pyotp.TOTP')
    def test_verify_and_complete_2fa_success(self, mock_totp_class):
        """Test verify_and_complete_2fa success"""
        from manager.models.auth_native import TOTPManager

        mock_user = MagicMock()
        mock_user.totp_secret = "TESTSECRET"
        self.mock_db.auth_user.__getitem__.return_value = mock_user

        mock_totp = MagicMock()
        mock_totp.verify.return_value = True
        mock_totp_class.return_value = mock_totp

        manager = TOTPManager(self.mock_auth)
        result = manager.verify_and_complete_2fa(1, "TESTSECRET", "123456")

        assert result is True
        mock_user.update_record.assert_called_once_with(totp_enabled=True)

    def test_disable_2fa_user_not_found(self):
        """Test disable_2fa with non-existent user"""
        from manager.models.auth_native import TOTPManager

        self.mock_db.auth_user.__getitem__.return_value = None
        manager = TOTPManager(self.mock_auth)

        result = manager.disable_2fa(999, "password123")
        assert result is False

    def test_disable_2fa_password_verification_fails(self):
        """Test disable_2fa with incorrect password"""
        from manager.models.auth_native import TOTPManager

        mock_user = MagicMock()
        self.mock_db.auth_user.__getitem__.return_value = mock_user
        self.mock_auth.verify_password.return_value = False

        manager = TOTPManager(self.mock_auth)
        result = manager.disable_2fa(1, "wrong_password")

        assert result is False

    @patch('manager.models.auth_native.pyotp.TOTP')
    def test_disable_2fa_enabled_no_code(self, mock_totp_class):
        """Test disable_2fa when 2FA enabled but no code provided"""
        from manager.models.auth_native import TOTPManager

        mock_user = MagicMock()
        mock_user.totp_enabled = True
        mock_user.totp_secret = "TESTSECRET"
        self.mock_db.auth_user.__getitem__.return_value = mock_user
        self.mock_auth.verify_password.return_value = True

        manager = TOTPManager(self.mock_auth)
        result = manager.disable_2fa(1, "correct_password", None)

        assert result is False

    @patch('manager.models.auth_native.pyotp.TOTP')
    def test_disable_2fa_enabled_invalid_code(self, mock_totp_class):
        """Test disable_2fa when 2FA enabled with invalid code"""
        from manager.models.auth_native import TOTPManager

        mock_user = MagicMock()
        mock_user.totp_enabled = True
        mock_user.totp_secret = "TESTSECRET"
        self.mock_db.auth_user.__getitem__.return_value = mock_user
        self.mock_auth.verify_password.return_value = True

        mock_totp = MagicMock()
        mock_totp.verify.return_value = False
        mock_totp_class.return_value = mock_totp

        manager = TOTPManager(self.mock_auth)
        result = manager.disable_2fa(1, "correct_password", "000000")

        assert result is False

    @patch('manager.models.auth_native.pyotp.TOTP')
    def test_disable_2fa_success_with_code(self, mock_totp_class):
        """Test disable_2fa success with valid TOTP code"""
        from manager.models.auth_native import TOTPManager

        mock_user = MagicMock()
        mock_user.totp_enabled = True
        mock_user.totp_secret = "TESTSECRET"
        self.mock_db.auth_user.__getitem__.return_value = mock_user
        self.mock_auth.verify_password.return_value = True

        mock_totp = MagicMock()
        mock_totp.verify.return_value = True
        mock_totp_class.return_value = mock_totp

        manager = TOTPManager(self.mock_auth)
        result = manager.disable_2fa(1, "correct_password", "123456")

        assert result is True
        mock_user.update_record.assert_called_once_with(totp_enabled=False, totp_secret=None)

    def test_disable_2fa_success_disabled(self):
        """Test disable_2fa when 2FA already disabled"""
        from manager.models.auth_native import TOTPManager

        mock_user = MagicMock()
        mock_user.totp_enabled = False
        self.mock_db.auth_user.__getitem__.return_value = mock_user
        self.mock_auth.verify_password.return_value = True

        manager = TOTPManager(self.mock_auth)
        result = manager.disable_2fa(1, "correct_password")

        assert result is True
        mock_user.update_record.assert_called_once_with(totp_enabled=False, totp_secret=None)

    @patch('manager.models.auth_native.pyotp.TOTP')
    def test_verify_totp_user_not_found(self, mock_totp_class):
        """Test verify_totp with non-existent user"""
        from manager.models.auth_native import TOTPManager

        self.mock_db.auth_user.__getitem__.return_value = None
        manager = TOTPManager(self.mock_auth)

        result = manager.verify_totp(999, "123456")
        assert result is False

    @patch('manager.models.auth_native.pyotp.TOTP')
    def test_verify_totp_disabled(self, mock_totp_class):
        """Test verify_totp when 2FA disabled"""
        from manager.models.auth_native import TOTPManager

        mock_user = MagicMock()
        mock_user.totp_enabled = False
        self.mock_db.auth_user.__getitem__.return_value = mock_user

        manager = TOTPManager(self.mock_auth)
        result = manager.verify_totp(1, "123456")

        assert result is False

    @patch('manager.models.auth_native.pyotp.TOTP')
    def test_verify_totp_success(self, mock_totp_class):
        """Test verify_totp success"""
        from manager.models.auth_native import TOTPManager

        mock_user = MagicMock()
        mock_user.totp_enabled = True
        mock_user.totp_secret = "TESTSECRET"
        self.mock_db.auth_user.__getitem__.return_value = mock_user

        mock_totp = MagicMock()
        mock_totp.verify.return_value = True
        mock_totp_class.return_value = mock_totp

        manager = TOTPManager(self.mock_auth)
        result = manager.verify_totp(1, "123456")

        assert result is True

    @patch('manager.models.auth_native.qrcode.QRCode')
    def test_generate_qr_code(self, mock_qrcode_class):
        """Test _generate_qr_code"""
        from manager.models.auth_native import TOTPManager

        manager = TOTPManager(self.mock_auth)

        mock_qr = MagicMock()
        mock_img = MagicMock()
        mock_qr.make_image.return_value = mock_img
        mock_qrcode_class.return_value = mock_qr

        result = manager._generate_qr_code("otpauth://totp/test")

        assert isinstance(result, str)
        mock_qr.add_data.assert_called_once_with("otpauth://totp/test")
        mock_qr.make.assert_called_once()
        mock_img.save.assert_called_once()




class TestPermissionFunctions:
    """Test permission-related functions"""

    def test_check_permission_no_user(self):
        """Test check_permission with no user"""
        from manager.models.auth_native import check_permission

        mock_auth = MagicMock()
        mock_auth.user_id = None

        result = check_permission(mock_auth, "read_clusters")
        assert result is False

    def test_check_permission_admin_always_allowed(self):
        """Test check_permission for admin user"""
        from manager.models.auth_native import check_permission

        mock_auth = MagicMock()
        mock_auth.user_id = 1
        mock_auth.get_user.return_value = {"is_admin": True}

        result = check_permission(mock_auth, "any_permission")
        assert result is True

    def test_check_permission_regular_user_has_permission(self):
        """Test check_permission for regular user with permission"""
        from manager.models.auth_native import check_permission

        mock_auth = MagicMock()
        mock_auth.user_id = 1
        mock_auth.get_user.return_value = {"is_admin": False}
        mock_auth.has_permission.return_value = True

        result = check_permission(mock_auth, "read_clusters")
        assert result is True

    def test_check_permission_regular_user_no_permission(self):
        """Test check_permission for regular user without permission"""
        from manager.models.auth_native import check_permission

        mock_auth = MagicMock()
        mock_auth.user_id = 1
        mock_auth.get_user.return_value = {"is_admin": False}
        mock_auth.has_permission.return_value = False

        result = check_permission(mock_auth, "delete_clusters")
        assert result is False


class TestCreateAdminUser:
    """Test create_admin_user function"""

    def test_create_admin_user_already_exists(self):
        """Test create_admin_user when admin already exists"""
        from manager.models.auth_native import create_admin_user

        mock_auth = MagicMock()
        mock_db = MagicMock()
        mock_auth.db = mock_db

        mock_user = MagicMock()
        mock_user.id = 5
        mock_db.return_value.select.return_value.first.return_value = mock_user

        result = create_admin_user(mock_auth, email="admin@test.com")
        assert result == 5



class TestSetupAuthGroups:
    """Test setup_auth_groups function"""

    def test_setup_auth_groups(self):
        """Test setup_auth_groups creates groups and permissions"""
        from manager.models.auth_native import setup_auth_groups

        mock_auth = MagicMock()
        mock_auth.add_group.side_effect = [1, 2]  # admin_group_id, service_owner_group_id

        result = setup_auth_groups(mock_auth)

        assert result["admin"] == 1
        assert result["service_owner"] == 2
        # Check admin group gets all permissions
        admin_calls = [call for call in mock_auth.add_permission.call_args_list if call[0][0] == 1]
        assert len(admin_calls) > 0


class TestSetupAuth:
    """Test setup_auth function"""

    def test_setup_auth_not_implemented(self):
        """Test setup_auth raises NotImplementedError"""
        from manager.models.auth_native import setup_auth

        with pytest.raises(NotImplementedError):
            setup_auth(MagicMock())
