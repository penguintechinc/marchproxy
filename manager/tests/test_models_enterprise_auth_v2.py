"""
Tests for enterprise authentication models (SAML, OAuth2, SCIM)

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import pytest
from unittest.mock import MagicMock, patch, AsyncMock
from datetime import datetime
from models.enterprise_auth import (
    EnterpriseAuthProviderModel,
    SAMLAuthenticator,
    OAuth2Authenticator,
    SCIMUserModel,
)


class TestEnterpriseAuthProviderModel:
    """Test EnterpriseAuthProviderModel"""

    def test_define_table(self):
        """Test table definition"""
        mock_db = MagicMock()
        EnterpriseAuthProviderModel.define_table(mock_db)
        mock_db.define_table.assert_called_once()
        call_args = mock_db.define_table.call_args[0]
        assert call_args[0] == "enterprise_auth_providers"

    def test_create_saml_provider_success(self):
        """Test successful SAML provider creation"""
        mock_db = MagicMock()
        mock_table = MagicMock()
        mock_db.enterprise_auth_providers = mock_table
        mock_table.insert.return_value = 1

        saml_config = {
            "idp_sso_url": "https://idp.example.com/sso",
            "idp_x509_cert": "cert-data",
            "sp_entity_id": "sp-entity-id",
            "idp_entity_id": "idp-entity-id",
        }

        result = EnterpriseAuthProviderModel.create_saml_provider(
            mock_db, "test-saml", saml_config, 1
        )

        assert result == 1
        mock_table.insert.assert_called_once()
        call_kwargs = mock_table.insert.call_args[1]
        assert call_kwargs["name"] == "test-saml"
        assert call_kwargs["provider_type"] == "saml"
        assert call_kwargs["auto_provision"] is True

    def test_create_saml_provider_missing_required_fields(self):
        """Test SAML provider creation with missing required fields"""
        mock_db = MagicMock()

        saml_config = {
            "idp_sso_url": "https://idp.example.com/sso",
            # Missing idp_x509_cert and sp_entity_id
        }

        with pytest.raises(ValueError, match="Missing required SAML configuration"):
            EnterpriseAuthProviderModel.create_saml_provider(
                mock_db, "test-saml", saml_config, 1
            )

    def test_create_oauth2_provider_success(self):
        """Test successful OAuth2 provider creation"""
        mock_db = MagicMock()
        mock_table = MagicMock()
        mock_db.enterprise_auth_providers = mock_table
        mock_table.insert.return_value = 2

        oauth2_config = {
            "client_id": "client-123",
            "client_secret": "secret-xyz",
            "auth_url": "https://auth.example.com/authorize",
            "token_url": "https://auth.example.com/token",
            "user_info_url": "https://auth.example.com/userinfo",
        }

        result = EnterpriseAuthProviderModel.create_oauth2_provider(
            mock_db, "test-oauth2", oauth2_config, 1
        )

        assert result == 2
        mock_table.insert.assert_called_once()
        call_kwargs = mock_table.insert.call_args[1]
        assert call_kwargs["provider_type"] == "oauth2"
        assert call_kwargs["default_role"] == "service_owner"

    def test_create_oauth2_provider_missing_required_fields(self):
        """Test OAuth2 provider creation with missing required fields"""
        mock_db = MagicMock()

        oauth2_config = {
            "client_id": "client-123",
            # Missing client_secret, auth_url, token_url, user_info_url
        }

        with pytest.raises(ValueError, match="Missing required OAuth2 configuration"):
            EnterpriseAuthProviderModel.create_oauth2_provider(
                mock_db, "test-oauth2", oauth2_config, 1
            )

    def test_create_oauth2_provider_custom_role(self):
        """Test OAuth2 provider creation with custom default role"""
        mock_db = MagicMock()
        mock_table = MagicMock()
        mock_db.enterprise_auth_providers = mock_table
        mock_table.insert.return_value = 3

        oauth2_config = {
            "client_id": "client-123",
            "client_secret": "secret-xyz",
            "auth_url": "https://auth.example.com/authorize",
            "token_url": "https://auth.example.com/token",
            "user_info_url": "https://auth.example.com/userinfo",
        }

        result = EnterpriseAuthProviderModel.create_oauth2_provider(
            mock_db, "test-oauth2", oauth2_config, 1, default_role="admin"
        )

        assert result == 3
        call_kwargs = mock_table.insert.call_args[1]
        assert call_kwargs["default_role"] == "admin"


class TestSAMLAuthenticator:
    """Test SAMLAuthenticator"""

    @patch("models.enterprise_auth.SAML2_AVAILABLE", False)
    def test_saml_unavailable_creates_client(self):
        """Test SAML client creation when SAML2 not available"""
        config = {
            "idp_sso_url": "https://idp.example.com/sso",
            "idp_x509_cert": "cert",
            "sp_entity_id": "sp-id",
            "idp_entity_id": "idp-id",
        }
        # When SAML2 not available, this should raise
        with patch("models.enterprise_auth.Saml2Client", None):
            with pytest.raises((TypeError, AttributeError)):
                SAMLAuthenticator(config, "https://example.com")

    @patch("models.enterprise_auth.Saml2Client")
    @patch("models.enterprise_auth.SAML2_AVAILABLE", True)
    def test_create_auth_request(self, mock_saml_client_class):
        """Test SAML auth request creation"""
        mock_client_instance = MagicMock()
        mock_saml_client_class.return_value = mock_client_instance
        mock_client_instance.prepare_for_authenticate.return_value = (
            "req-id-123",
            {"headers": [("Location", "https://idp.example.com/sso?SAMLRequest=...")]}
        )

        config = {
            "idp_sso_url": "https://idp.example.com/sso",
            "idp_x509_cert": "cert",
            "sp_entity_id": "sp-id",
            "idp_entity_id": "idp-id",
        }

        with patch("models.enterprise_auth.Saml2Config"):
            authenticator = SAMLAuthenticator(config, "https://example.com")
            req_id, auth_url = authenticator.create_auth_request()

            assert req_id == "req-id-123"
            assert "https://idp.example.com/sso" in auth_url

    @patch("models.enterprise_auth.Saml2Client")
    @patch("models.enterprise_auth.SAML2_AVAILABLE", True)
    def test_process_response_success(self, mock_saml_client_class):
        """Test successful SAML response processing"""
        mock_client_instance = MagicMock()
        mock_saml_client_class.return_value = mock_client_instance

        mock_authn_response = MagicMock()
        mock_authn_response.get_identity.return_value = {
            "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress": ["user@example.com"],
            "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name": ["John"],
        }
        mock_authn_response.name_id = "user@example.com"
        mock_authn_response.session_index.return_value = "session-123"

        mock_client_instance.parse_authn_request_response.return_value = mock_authn_response

        config = {
            "idp_sso_url": "https://idp.example.com/sso",
            "idp_x509_cert": "cert",
            "sp_entity_id": "sp-id",
            "idp_entity_id": "idp-id",
            "attribute_mapping": {
                "email": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
                "username": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name",
            }
        }

        with patch("models.enterprise_auth.Saml2Config"):
            authenticator = SAMLAuthenticator(config, "https://example.com")
            result = authenticator.process_response("saml-response", "req-id-123")

            assert result is not None
            assert result["provider"] == "saml"
            assert result["external_id"] == "user@example.com"
            assert result["attributes"]["email"] == "user@example.com"
            assert result["session_index"] == "session-123"

    @patch("models.enterprise_auth.Saml2Client")
    @patch("models.enterprise_auth.SAML2_AVAILABLE", True)
    def test_process_response_invalid(self, mock_saml_client_class):
        """Test SAML response processing with invalid response"""
        mock_client_instance = MagicMock()
        mock_saml_client_class.return_value = mock_client_instance
        mock_client_instance.parse_authn_request_response.return_value = None

        config = {
            "idp_sso_url": "https://idp.example.com/sso",
            "idp_x509_cert": "cert",
            "sp_entity_id": "sp-id",
            "idp_entity_id": "idp-id",
        }

        with patch("models.enterprise_auth.Saml2Config"):
            authenticator = SAMLAuthenticator(config, "https://example.com")
            result = authenticator.process_response("invalid-response")

            assert result is None

    @patch("models.enterprise_auth.Saml2Client")
    @patch("models.enterprise_auth.SAML2_AVAILABLE", True)
    def test_create_logout_request(self, mock_saml_client_class):
        """Test SAML logout request creation"""
        mock_client_instance = MagicMock()
        mock_saml_client_class.return_value = mock_client_instance
        mock_client_instance.global_logout.return_value = (
            "logout-req-id",
            {"headers": [("Location", "https://idp.example.com/slo?SAMLRequest=...")]}
        )

        config = {
            "idp_sso_url": "https://idp.example.com/sso",
            "idp_x509_cert": "cert",
            "sp_entity_id": "sp-id",
            "idp_entity_id": "idp-id",
        }

        with patch("models.enterprise_auth.Saml2Config"):
            authenticator = SAMLAuthenticator(config, "https://example.com")
            logout_url = authenticator.create_logout_request("user@example.com", "session-123")

            assert logout_url is not None
            assert "idp.example.com" in logout_url


class TestOAuth2Authenticator:
    """Test OAuth2Authenticator"""

    def test_init(self):
        """Test OAuth2Authenticator initialization"""
        config = {
            "client_id": "client-123",
            "client_secret": "secret",
            "auth_url": "https://auth.example.com/authorize",
            "token_url": "https://auth.example.com/token",
            "user_info_url": "https://auth.example.com/userinfo",
        }

        authenticator = OAuth2Authenticator(config, "https://example.com")

        assert authenticator.config == config
        assert authenticator.base_url == "https://example.com"
        assert authenticator.redirect_uri == "https://example.com/api/auth/oauth2/callback"

    def test_create_auth_url(self):
        """Test OAuth2 authorization URL creation"""
        config = {
            "client_id": "client-123",
            "client_secret": "secret",
            "auth_url": "https://auth.example.com/authorize",
            "token_url": "https://auth.example.com/token",
            "user_info_url": "https://auth.example.com/userinfo",
            "scope": "openid email profile",
        }

        authenticator = OAuth2Authenticator(config, "https://example.com")
        auth_url = authenticator.create_auth_url("state-123")

        assert "https://auth.example.com/authorize" in auth_url
        assert "client_id=client-123" in auth_url
        assert "state=state-123" in auth_url
        assert "response_type=code" in auth_url

    @pytest.mark.asyncio
    async def test_exchange_code_success(self):
        """Test successful OAuth2 code exchange"""
        config = {
            "client_id": "client-123",
            "client_secret": "secret",
            "auth_url": "https://auth.example.com/authorize",
            "token_url": "https://auth.example.com/token",
            "user_info_url": "https://auth.example.com/userinfo",
        }

        authenticator = OAuth2Authenticator(config, "https://example.com")

        token_response = {
            "access_token": "access-token-123",
            "refresh_token": "refresh-token-456",
        }

        user_info = {
            "sub": "user-123",
            "email": "user@example.com",
            "name": "John Doe",
        }

        with patch("models.enterprise_auth.httpx.AsyncClient") as mock_client_class:
            mock_client = AsyncMock()
            mock_client_class.return_value.__aenter__.return_value = mock_client

            token_response_obj = MagicMock()
            token_response_obj.status_code = 200
            token_response_obj.json.return_value = token_response

            user_response_obj = MagicMock()
            user_response_obj.status_code = 200
            user_response_obj.json.return_value = user_info

            mock_client.post.return_value = token_response_obj
            mock_client.get.return_value = user_response_obj

            result = await authenticator.exchange_code("code-xyz", "state-123")

            assert result is not None
            assert result["provider"] == "oauth2"
            assert result["external_id"] == "user-123"
            assert result["access_token"] == "access-token-123"

    @pytest.mark.asyncio
    async def test_exchange_code_failure(self):
        """Test OAuth2 code exchange failure"""
        config = {
            "client_id": "client-123",
            "client_secret": "secret",
            "auth_url": "https://auth.example.com/authorize",
            "token_url": "https://auth.example.com/token",
            "user_info_url": "https://auth.example.com/userinfo",
        }

        authenticator = OAuth2Authenticator(config, "https://example.com")

        with patch("models.enterprise_auth.httpx.AsyncClient") as mock_client_class:
            mock_client = AsyncMock()
            mock_client_class.return_value.__aenter__.return_value = mock_client

            token_response_obj = MagicMock()
            token_response_obj.status_code = 400
            mock_client.post.return_value = token_response_obj

            result = await authenticator.exchange_code("invalid-code", "state-123")

            assert result is None

    def test_map_attributes(self):
        """Test OAuth2 attribute mapping"""
        config = {
            "client_id": "client-123",
            "client_secret": "secret",
            "auth_url": "https://auth.example.com/authorize",
            "token_url": "https://auth.example.com/token",
            "user_info_url": "https://auth.example.com/userinfo",
            "attribute_mapping": {
                "email": "email",
                "username": "preferred_username",
            }
        }

        authenticator = OAuth2Authenticator(config, "https://example.com")

        user_info = {
            "email": "user@example.com",
            "preferred_username": "johndoe",
            "name": "John Doe",
        }

        mapped = authenticator._map_attributes(user_info)

        assert mapped["email"] == "user@example.com"
        assert mapped["username"] == "johndoe"
        assert "name" not in mapped


class TestSCIMUserModel:
    """Test SCIMUserModel"""

    def test_define_table(self):
        """Test SCIM user table definition"""
        mock_db = MagicMock()
        SCIMUserModel.define_table(mock_db)
        mock_db.define_table.assert_called_once()
        call_args = mock_db.define_table.call_args[0]
        assert call_args[0] == "scim_users"

    def test_process_scim_user_new_user_auto_provision_enabled(self):
        """Test SCIM user processing for new user with auto-provisioning enabled"""
        mock_db = MagicMock()
        mock_scim_table = MagicMock()
        mock_provider_table = MagicMock()
        mock_users_table = MagicMock()

        mock_db.scim_users = mock_scim_table
        mock_db.enterprise_auth_providers = mock_provider_table
        mock_db.users = mock_users_table

        # Simulate no existing SCIM user
        mock_scim_table.return_value.select.return_value.first.return_value = None

        # Simulate provider with auto-provisioning enabled
        mock_provider = MagicMock()
        mock_provider.auto_provision = True
        mock_provider_table.return_value.select.return_value.first.return_value = (
            mock_provider
        )

        # Mock user creation
        mock_users_table.insert.return_value = 1
        mock_scim_table.insert.return_value = 1

        scim_data = {
            "id": "scim-123",
            "userName": "johndoe",
            "emails": [{"value": "john@example.com", "primary": True}],
            "name": {"givenName": "John", "familyName": "Doe"},
            "active": True,
        }

        result = SCIMUserModel.process_scim_user(mock_db, scim_data, "test-provider")

        assert result == 1
        mock_users_table.insert.assert_called_once()
        mock_scim_table.insert.assert_called_once()

    def test_process_scim_user_missing_id(self):
        """Test SCIM user processing with missing SCIM ID"""
        mock_db = MagicMock()

        scim_data = {
            "userName": "johndoe",
            "emails": [{"value": "john@example.com", "primary": True}],
        }

        result = SCIMUserModel.process_scim_user(mock_db, scim_data, "test-provider")

        assert result is None

    def test_process_scim_user_existing_user_update(self):
        """Test SCIM user processing for existing user update"""
        mock_db = MagicMock()
        mock_scim_table = MagicMock()
        mock_users_table = MagicMock()

        mock_db.scim_users = mock_scim_table
        mock_db.users = mock_users_table

        # Simulate existing SCIM user
        mock_scim_user = MagicMock()
        mock_scim_user.user_id = 1
        mock_scim_user.update_record = MagicMock()
        mock_scim_table.return_value.select.return_value.first.return_value = (
            mock_scim_user
        )

        # Mock existing user
        mock_user = MagicMock()
        mock_user.update_record = MagicMock()
        mock_users_table.__getitem__.return_value = mock_user

        scim_data = {
            "id": "scim-123",
            "userName": "johndoe",
            "emails": [{"value": "john@example.com", "primary": True}],
            "active": True,
        }

        result = SCIMUserModel.process_scim_user(mock_db, scim_data, "test-provider")

        assert result == 1
        mock_scim_user.update_record.assert_called_once()
        mock_user.update_record.assert_called_once()

    def test_process_scim_user_auto_provision_disabled(self):
        """Test SCIM user processing with auto-provisioning disabled"""
        mock_db = MagicMock()
        mock_scim_table = MagicMock()
        mock_provider_table = MagicMock()

        mock_db.scim_users = mock_scim_table
        mock_db.enterprise_auth_providers = mock_provider_table

        # Simulate no existing SCIM user
        mock_scim_table.return_value.select.return_value.first.return_value = None

        # Simulate provider with auto-provisioning disabled
        mock_provider = MagicMock()
        mock_provider.auto_provision = False
        mock_provider_table.return_value.select.return_value.first.return_value = (
            mock_provider
        )

        scim_data = {
            "id": "scim-123",
            "userName": "johndoe",
            "emails": [{"value": "john@example.com", "primary": True}],
        }

        result = SCIMUserModel.process_scim_user(mock_db, scim_data, "test-provider")

        assert result is None
