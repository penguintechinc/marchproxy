"""
Comprehensive unit tests for enterprise_auth.py models - Coverage improvement v3

Tests focus on uncovered branches and edge cases (excluding SCIM):
- EnterpriseAuthProviderModel creation and retrieval
- SAMLAuthenticator request/response handling
- OAuth2Authenticator auth flows
- Attribute mapping
- Error handling paths
- Configuration validation

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
)


@pytest.fixture
def mock_db():
    """Create a mock database instance"""
    db = MagicMock()
    db.enterprise_auth_providers = MagicMock()
    db.users = MagicMock()
    db.scim_users = MagicMock()
    return db


class TestEnterpriseAuthProviderModelCreate:
    """Tests for EnterpriseAuthProviderModel creation"""

    def test_create_saml_provider_success(self, mock_db):
        """Successfully create SAML provider"""
        mock_db.enterprise_auth_providers.insert.return_value = 1

        saml_config = {
            "idp_sso_url": "https://idp.example.com/sso",
            "idp_x509_cert": "cert-data",
            "sp_entity_id": "sp-entity",
            "idp_entity_id": "idp-entity",
        }

        result = EnterpriseAuthProviderModel.create_saml_provider(
            mock_db, "test-saml", saml_config, 1
        )

        assert result == 1
        mock_db.enterprise_auth_providers.insert.assert_called_once()
        call_kwargs = mock_db.enterprise_auth_providers.insert.call_args[1]
        assert call_kwargs["name"] == "test-saml"
        assert call_kwargs["provider_type"] == "saml"
        assert call_kwargs["auto_provision"] is True
        assert call_kwargs["default_role"] == "service_owner"

    def test_create_saml_provider_missing_required_field(self, mock_db):
        """SAML provider creation fails with missing required fields"""
        saml_config = {
            "idp_sso_url": "https://idp.example.com/sso",
            # Missing idp_x509_cert and sp_entity_id
        }

        with pytest.raises(ValueError, match="Missing required SAML configuration fields"):
            EnterpriseAuthProviderModel.create_saml_provider(
                mock_db, "test-saml", saml_config, 1
            )

    def test_create_saml_provider_custom_auto_provision(self, mock_db):
        """SAML provider with auto_provision disabled"""
        mock_db.enterprise_auth_providers.insert.return_value = 2

        saml_config = {
            "idp_sso_url": "https://idp.example.com/sso",
            "idp_x509_cert": "cert-data",
            "sp_entity_id": "sp-entity",
            "idp_entity_id": "idp-entity",
        }

        result = EnterpriseAuthProviderModel.create_saml_provider(
            mock_db,
            "test-saml-no-provision",
            saml_config,
            1,
            auto_provision=False,
        )

        assert result == 2
        call_kwargs = mock_db.enterprise_auth_providers.insert.call_args[1]
        assert call_kwargs["auto_provision"] is False

    def test_create_saml_provider_custom_role(self, mock_db):
        """SAML provider with custom default role"""
        mock_db.enterprise_auth_providers.insert.return_value = 3

        saml_config = {
            "idp_sso_url": "https://idp.example.com/sso",
            "idp_x509_cert": "cert-data",
            "sp_entity_id": "sp-entity",
            "idp_entity_id": "idp-entity",
        }

        result = EnterpriseAuthProviderModel.create_saml_provider(
            mock_db,
            "test-saml-admin",
            saml_config,
            1,
            default_role="admin",
        )

        assert result == 3
        call_kwargs = mock_db.enterprise_auth_providers.insert.call_args[1]
        assert call_kwargs["default_role"] == "admin"

    def test_create_oauth2_provider_success(self, mock_db):
        """Successfully create OAuth2 provider"""
        mock_db.enterprise_auth_providers.insert.return_value = 10

        oauth2_config = {
            "client_id": "client-123",
            "client_secret": "secret-456",
            "auth_url": "https://oauth.example.com/authorize",
            "token_url": "https://oauth.example.com/token",
            "user_info_url": "https://oauth.example.com/userinfo",
        }

        result = EnterpriseAuthProviderModel.create_oauth2_provider(
            mock_db, "test-oauth2", oauth2_config, 1
        )

        assert result == 10
        mock_db.enterprise_auth_providers.insert.assert_called_once()
        call_kwargs = mock_db.enterprise_auth_providers.insert.call_args[1]
        assert call_kwargs["name"] == "test-oauth2"
        assert call_kwargs["provider_type"] == "oauth2"
        assert call_kwargs["config"] == oauth2_config

    def test_create_oauth2_provider_missing_required_field(self, mock_db):
        """OAuth2 provider creation fails with missing required fields"""
        oauth2_config = {
            "client_id": "client-123",
            # Missing client_secret, auth_url, token_url, user_info_url
        }

        with pytest.raises(ValueError, match="Missing required OAuth2 configuration fields"):
            EnterpriseAuthProviderModel.create_oauth2_provider(
                mock_db, "test-oauth2", oauth2_config, 1
            )

    def test_create_oauth2_provider_custom_settings(self, mock_db):
        """OAuth2 provider with custom auto_provision and role"""
        mock_db.enterprise_auth_providers.insert.return_value = 11

        oauth2_config = {
            "client_id": "client-123",
            "client_secret": "secret-456",
            "auth_url": "https://oauth.example.com/authorize",
            "token_url": "https://oauth.example.com/token",
            "user_info_url": "https://oauth.example.com/userinfo",
        }

        result = EnterpriseAuthProviderModel.create_oauth2_provider(
            mock_db,
            "test-oauth2-viewer",
            oauth2_config,
            1,
            auto_provision=False,
            default_role="viewer",
        )

        assert result == 11
        call_kwargs = mock_db.enterprise_auth_providers.insert.call_args[1]
        assert call_kwargs["auto_provision"] is False
        assert call_kwargs["default_role"] == "viewer"


class TestSAMLAuthenticatorInit:
    """Tests for SAMLAuthenticator initialization"""

    @patch("models.enterprise_auth.SAML2_AVAILABLE", True)
    @patch("models.enterprise_auth.Saml2Config")
    @patch("models.enterprise_auth.Saml2Client")
    def test_saml_authenticator_init(self, mock_client_class, mock_config_class):
        """SAMLAuthenticator initializes with config"""
        config = {
            "sp_entity_id": "sp-entity",
            "idp_sso_url": "https://idp.example.com/sso",
            "idp_x509_cert": "cert-data",
            "idp_entity_id": "idp-entity",
        }

        mock_config_instance = MagicMock()
        mock_config_class.return_value = mock_config_instance
        mock_client_class.return_value = MagicMock()

        authenticator = SAMLAuthenticator(config, "https://app.example.com")

        assert authenticator.base_url == "https://app.example.com"
        assert authenticator.config == config

    @patch("models.enterprise_auth.SAML2_AVAILABLE", True)
    @patch("models.enterprise_auth.Saml2Config")
    @patch("models.enterprise_auth.Saml2Client")
    def test_saml_authenticator_base_url_trailing_slash(self, mock_client_class, mock_config_class):
        """SAMLAuthenticator removes trailing slashes from base_url"""
        config = {
            "sp_entity_id": "sp-entity",
            "idp_sso_url": "https://idp.example.com/sso",
            "idp_x509_cert": "cert-data",
            "idp_entity_id": "idp-entity",
        }

        mock_config_instance = MagicMock()
        mock_config_class.return_value = mock_config_instance
        mock_client_class.return_value = MagicMock()

        authenticator = SAMLAuthenticator(config, "https://app.example.com/")

        assert authenticator.base_url == "https://app.example.com"


class TestSAMLAuthenticatorCreateAuthRequest:
    """Tests for SAMLAuthenticator.create_auth_request()"""

    @patch("models.enterprise_auth.SAML2_AVAILABLE", True)
    @patch("models.enterprise_auth.Saml2Config")
    @patch("models.enterprise_auth.Saml2Client")
    def test_create_auth_request_success(self, mock_client_class, mock_config_class):
        """Create SAML auth request successfully"""
        config = {
            "sp_entity_id": "sp-entity",
            "idp_sso_url": "https://idp.example.com/sso",
            "idp_x509_cert": "cert-data",
            "idp_entity_id": "idp-entity",
        }

        mock_config_instance = MagicMock()
        mock_config_class.return_value = mock_config_instance

        mock_client_instance = MagicMock()
        mock_client_class.return_value = mock_client_instance

        req_id = "req-123"
        redirect_url = "https://idp.example.com/sso?SAMLRequest=..."
        mock_client_instance.prepare_for_authenticate.return_value = (
            req_id,
            {"headers": [("Location", redirect_url)]},
        )

        authenticator = SAMLAuthenticator(config, "https://app.example.com")
        returned_id, returned_url = authenticator.create_auth_request()

        assert returned_id == req_id
        assert returned_url == redirect_url

    @patch("models.enterprise_auth.SAML2_AVAILABLE", True)
    @patch("models.enterprise_auth.Saml2Config")
    @patch("models.enterprise_auth.Saml2Client")
    def test_create_auth_request_with_relay_state(self, mock_client_class, mock_config_class):
        """Create SAML auth request with relay state"""
        config = {
            "sp_entity_id": "sp-entity",
            "idp_sso_url": "https://idp.example.com/sso",
            "idp_x509_cert": "cert-data",
            "idp_entity_id": "idp-entity",
        }

        mock_config_instance = MagicMock()
        mock_config_class.return_value = mock_config_instance

        mock_client_instance = MagicMock()
        mock_client_class.return_value = mock_client_instance

        req_id = "req-456"
        redirect_url = "https://idp.example.com/sso?SAMLRequest=...&RelayState=home"
        mock_client_instance.prepare_for_authenticate.return_value = (
            req_id,
            {"headers": [("Location", redirect_url)]},
        )

        authenticator = SAMLAuthenticator(config, "https://app.example.com")
        returned_id, returned_url = authenticator.create_auth_request(relay_state="home")

        assert returned_id == req_id
        assert returned_url == redirect_url


class TestSAMLAuthenticatorProcessResponse:
    """Tests for SAMLAuthenticator.process_response()"""

    @patch("models.enterprise_auth.SAML2_AVAILABLE", True)
    @patch("models.enterprise_auth.Saml2Config")
    @patch("models.enterprise_auth.Saml2Client")
    def test_process_response_success(self, mock_client_class, mock_config_class):
        """Process SAML response successfully"""
        config = {
            "sp_entity_id": "sp-entity",
            "idp_sso_url": "https://idp.example.com/sso",
            "idp_x509_cert": "cert-data",
            "idp_entity_id": "idp-entity",
            "attribute_mapping": {
                "email": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
                "name": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name",
            },
        }

        mock_config_instance = MagicMock()
        mock_config_class.return_value = mock_config_instance

        mock_client_instance = MagicMock()
        mock_client_class.return_value = mock_client_instance

        mock_response = MagicMock()
        mock_response.name_id = "user@example.com"
        mock_response.session_index.return_value = "session-123"
        mock_response.get_identity.return_value = {
            "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress": [
                "user@example.com"
            ],
            "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name": ["John Doe"],
        }

        mock_client_instance.parse_authn_request_response.return_value = mock_response

        authenticator = SAMLAuthenticator(config, "https://app.example.com")
        result = authenticator.process_response("SAMLResponse=...")

        assert result is not None
        assert result["provider"] == "saml"
        assert result["external_id"] == "user@example.com"
        assert result["session_index"] == "session-123"
        assert result["attributes"]["email"] == "user@example.com"

    @patch("models.enterprise_auth.SAML2_AVAILABLE", True)
    @patch("models.enterprise_auth.Saml2Config")
    @patch("models.enterprise_auth.Saml2Client")
    def test_process_response_invalid_response(self, mock_client_class, mock_config_class):
        """Process invalid SAML response returns None"""
        config = {
            "sp_entity_id": "sp-entity",
            "idp_sso_url": "https://idp.example.com/sso",
            "idp_x509_cert": "cert-data",
            "idp_entity_id": "idp-entity",
        }

        mock_config_instance = MagicMock()
        mock_config_class.return_value = mock_config_instance

        mock_client_instance = MagicMock()
        mock_client_class.return_value = mock_client_instance

        mock_client_instance.parse_authn_request_response.return_value = None

        authenticator = SAMLAuthenticator(config, "https://app.example.com")
        result = authenticator.process_response("invalid-saml")

        assert result is None

    @patch("models.enterprise_auth.SAML2_AVAILABLE", True)
    @patch("models.enterprise_auth.Saml2Config")
    @patch("models.enterprise_auth.Saml2Client")
    def test_process_response_no_identity(self, mock_client_class, mock_config_class):
        """Process SAML response with no identity returns None"""
        config = {
            "sp_entity_id": "sp-entity",
            "idp_sso_url": "https://idp.example.com/sso",
            "idp_x509_cert": "cert-data",
            "idp_entity_id": "idp-entity",
        }

        mock_config_instance = MagicMock()
        mock_config_class.return_value = mock_config_instance

        mock_client_instance = MagicMock()
        mock_client_class.return_value = mock_client_instance

        mock_response = MagicMock()
        mock_response.get_identity.return_value = None

        mock_client_instance.parse_authn_request_response.return_value = mock_response

        authenticator = SAMLAuthenticator(config, "https://app.example.com")
        result = authenticator.process_response("SAMLResponse=...")

        assert result is None

    @patch("models.enterprise_auth.SAML2_AVAILABLE", True)
    @patch("models.enterprise_auth.Saml2Config")
    @patch("models.enterprise_auth.Saml2Client")
    def test_process_response_exception(self, mock_client_class, mock_config_class):
        """Process SAML response with exception returns None"""
        config = {
            "sp_entity_id": "sp-entity",
            "idp_sso_url": "https://idp.example.com/sso",
            "idp_x509_cert": "cert-data",
            "idp_entity_id": "idp-entity",
        }

        mock_config_instance = MagicMock()
        mock_config_class.return_value = mock_config_instance

        mock_client_instance = MagicMock()
        mock_client_class.return_value = mock_client_instance

        mock_client_instance.parse_authn_request_response.side_effect = Exception(
            "Invalid SAML"
        )

        authenticator = SAMLAuthenticator(config, "https://app.example.com")
        result = authenticator.process_response("invalid-saml")

        assert result is None


class TestSAMLAuthenticatorCreateLogoutRequest:
    """Tests for SAMLAuthenticator.create_logout_request()"""

    @patch("models.enterprise_auth.SAML2_AVAILABLE", True)
    @patch("models.enterprise_auth.Saml2Config")
    @patch("models.enterprise_auth.Saml2Client")
    def test_create_logout_request_success(self, mock_client_class, mock_config_class):
        """Create SAML logout request successfully"""
        config = {
            "sp_entity_id": "sp-entity",
            "idp_sso_url": "https://idp.example.com/sso",
            "idp_x509_cert": "cert-data",
            "idp_entity_id": "idp-entity",
        }

        mock_config_instance = MagicMock()
        mock_config_class.return_value = mock_config_instance

        mock_client_instance = MagicMock()
        mock_client_class.return_value = mock_client_instance

        logout_url = "https://idp.example.com/slo?SAMLRequest=..."
        mock_client_instance.global_logout.return_value = (
            "logout-req-123",
            {"headers": [("Location", logout_url)]},
        )

        authenticator = SAMLAuthenticator(config, "https://app.example.com")
        result = authenticator.create_logout_request("user@example.com", "session-123")

        assert result == logout_url

    @patch("models.enterprise_auth.SAML2_AVAILABLE", True)
    @patch("models.enterprise_auth.Saml2Config")
    @patch("models.enterprise_auth.Saml2Client")
    def test_create_logout_request_no_location(self, mock_client_class, mock_config_class):
        """Create logout request with no Location header returns None"""
        config = {
            "sp_entity_id": "sp-entity",
            "idp_sso_url": "https://idp.example.com/sso",
            "idp_x509_cert": "cert-data",
            "idp_entity_id": "idp-entity",
        }

        mock_config_instance = MagicMock()
        mock_config_class.return_value = mock_config_instance

        mock_client_instance = MagicMock()
        mock_client_class.return_value = mock_client_instance

        mock_client_instance.global_logout.return_value = (
            "logout-req-456",
            {"headers": [("Other-Header", "value")]},
        )

        authenticator = SAMLAuthenticator(config, "https://app.example.com")
        result = authenticator.create_logout_request("user@example.com", "session-123")

        assert result is None


class TestOAuth2AuthenticatorInit:
    """Tests for OAuth2Authenticator initialization"""

    def test_oauth2_authenticator_init(self):
        """OAuth2Authenticator initializes with config"""
        config = {
            "client_id": "client-123",
            "client_secret": "secret-456",
            "auth_url": "https://oauth.example.com/authorize",
            "token_url": "https://oauth.example.com/token",
            "user_info_url": "https://oauth.example.com/userinfo",
        }

        authenticator = OAuth2Authenticator(config, "https://app.example.com")

        assert authenticator.base_url == "https://app.example.com"
        assert (
            authenticator.redirect_uri == "https://app.example.com/api/auth/oauth2/callback"
        )

    def test_oauth2_authenticator_base_url_trailing_slash(self):
        """OAuth2Authenticator removes trailing slashes from base_url"""
        config = {
            "client_id": "client-123",
            "client_secret": "secret-456",
            "auth_url": "https://oauth.example.com/authorize",
            "token_url": "https://oauth.example.com/token",
            "user_info_url": "https://oauth.example.com/userinfo",
        }

        authenticator = OAuth2Authenticator(config, "https://app.example.com/")

        assert authenticator.base_url == "https://app.example.com"


class TestOAuth2AuthenticatorCreateAuthUrl:
    """Tests for OAuth2Authenticator.create_auth_url()"""

    def test_create_auth_url_basic(self):
        """Create basic OAuth2 auth URL"""
        config = {
            "client_id": "client-123",
            "client_secret": "secret-456",
            "auth_url": "https://oauth.example.com/authorize",
            "token_url": "https://oauth.example.com/token",
            "user_info_url": "https://oauth.example.com/userinfo",
        }

        authenticator = OAuth2Authenticator(config, "https://app.example.com")
        auth_url = authenticator.create_auth_url("state-123")

        assert "https://oauth.example.com/authorize" in auth_url
        assert "client_id=client-123" in auth_url
        assert "state=state-123" in auth_url
        assert "response_type=code" in auth_url
        assert "redirect_uri=" in auth_url

    def test_create_auth_url_custom_scope(self):
        """Create OAuth2 auth URL with custom scope"""
        config = {
            "client_id": "client-123",
            "client_secret": "secret-456",
            "auth_url": "https://oauth.example.com/authorize",
            "token_url": "https://oauth.example.com/token",
            "user_info_url": "https://oauth.example.com/userinfo",
            "scope": "openid email profile groups",
        }

        authenticator = OAuth2Authenticator(config, "https://app.example.com")
        auth_url = authenticator.create_auth_url("state-456")

        assert "scope=" in auth_url


class TestOAuth2AuthenticatorExchangeCode:
    """Tests for OAuth2Authenticator.exchange_code()"""

    @pytest.mark.asyncio
    async def test_exchange_code_success(self):
        """Exchange authorization code for token successfully"""
        config = {
            "client_id": "client-123",
            "client_secret": "secret-456",
            "auth_url": "https://oauth.example.com/authorize",
            "token_url": "https://oauth.example.com/token",
            "user_info_url": "https://oauth.example.com/userinfo",
        }

        authenticator = OAuth2Authenticator(config, "https://app.example.com")

        mock_response_token = MagicMock()
        mock_response_token.status_code = 200
        mock_response_token.json.return_value = {
            "access_token": "access-token-xyz",
            "refresh_token": "refresh-token-abc",
            "token_type": "Bearer",
        }

        mock_response_user = MagicMock()
        mock_response_user.status_code = 200
        mock_response_user.json.return_value = {
            "sub": "user-123",
            "email": "user@example.com",
            "name": "John Doe",
        }

        with patch("httpx.AsyncClient") as mock_client_class:
            mock_client = MagicMock()
            mock_client.__aenter__.return_value = mock_client
            mock_client.__aexit__.return_value = None
            mock_client.post = AsyncMock(return_value=mock_response_token)
            mock_client.get = AsyncMock(return_value=mock_response_user)

            mock_client_class.return_value = mock_client

            result = await authenticator.exchange_code("code-xyz", "state-123")

            assert result is not None
            assert result["provider"] == "oauth2"
            assert result["external_id"] == "user-123"
            assert result["access_token"] == "access-token-xyz"
            assert result["refresh_token"] == "refresh-token-abc"

    @pytest.mark.asyncio
    async def test_exchange_code_token_error(self):
        """Exchange code with token endpoint error"""
        config = {
            "client_id": "client-123",
            "client_secret": "secret-456",
            "auth_url": "https://oauth.example.com/authorize",
            "token_url": "https://oauth.example.com/token",
            "user_info_url": "https://oauth.example.com/userinfo",
        }

        authenticator = OAuth2Authenticator(config, "https://app.example.com")

        mock_response = MagicMock()
        mock_response.status_code = 400
        mock_response.json.return_value = {"error": "invalid_code"}

        with patch("httpx.AsyncClient") as mock_client_class:
            mock_client = MagicMock()
            mock_client.__aenter__.return_value = mock_client
            mock_client.__aexit__.return_value = None
            mock_client.post = AsyncMock(return_value=mock_response)

            mock_client_class.return_value = mock_client

            result = await authenticator.exchange_code("invalid-code", "state-123")

            assert result is None

    @pytest.mark.asyncio
    async def test_exchange_code_no_access_token(self):
        """Exchange code with missing access token in response"""
        config = {
            "client_id": "client-123",
            "client_secret": "secret-456",
            "auth_url": "https://oauth.example.com/authorize",
            "token_url": "https://oauth.example.com/token",
            "user_info_url": "https://oauth.example.com/userinfo",
        }

        authenticator = OAuth2Authenticator(config, "https://app.example.com")

        mock_response = MagicMock()
        mock_response.status_code = 200
        mock_response.json.return_value = {"token_type": "Bearer"}  # Missing access_token

        with patch("httpx.AsyncClient") as mock_client_class:
            mock_client = MagicMock()
            mock_client.__aenter__.return_value = mock_client
            mock_client.__aexit__.return_value = None
            mock_client.post = AsyncMock(return_value=mock_response)

            mock_client_class.return_value = mock_client

            result = await authenticator.exchange_code("code-xyz", "state-123")

            assert result is None


class TestOAuth2AuthenticatorGetUserInfo:
    """Tests for OAuth2Authenticator._get_user_info()"""

    @pytest.mark.asyncio
    async def test_get_user_info_success(self):
        """Get user info successfully"""
        config = {
            "client_id": "client-123",
            "client_secret": "secret-456",
            "auth_url": "https://oauth.example.com/authorize",
            "token_url": "https://oauth.example.com/token",
            "user_info_url": "https://oauth.example.com/userinfo",
        }

        authenticator = OAuth2Authenticator(config, "https://app.example.com")

        mock_response = MagicMock()
        mock_response.status_code = 200
        mock_response.json.return_value = {
            "sub": "user-123",
            "email": "user@example.com",
            "name": "John Doe",
        }

        mock_client = AsyncMock()
        mock_client.get = AsyncMock(return_value=mock_response)

        result = await authenticator._get_user_info(mock_client, "access-token-xyz")

        assert result is not None
        assert result["sub"] == "user-123"
        assert result["email"] == "user@example.com"

    @pytest.mark.asyncio
    async def test_get_user_info_request_error(self):
        """Get user info request fails"""
        config = {
            "client_id": "client-123",
            "client_secret": "secret-456",
            "auth_url": "https://oauth.example.com/authorize",
            "token_url": "https://oauth.example.com/token",
            "user_info_url": "https://oauth.example.com/userinfo",
        }

        authenticator = OAuth2Authenticator(config, "https://app.example.com")

        mock_response = MagicMock()
        mock_response.status_code = 401

        mock_client = AsyncMock()
        mock_client.get = AsyncMock(return_value=mock_response)

        result = await authenticator._get_user_info(mock_client, "invalid-token")

        assert result is None


class TestOAuth2AuthenticatorMapAttributes:
    """Tests for OAuth2Authenticator._map_attributes()"""

    def test_map_attributes_success(self):
        """Map OAuth2 user attributes successfully"""
        config = {
            "client_id": "client-123",
            "client_secret": "secret-456",
            "auth_url": "https://oauth.example.com/authorize",
            "token_url": "https://oauth.example.com/token",
            "user_info_url": "https://oauth.example.com/userinfo",
            "attribute_mapping": {
                "email": "email",
                "username": "preferred_username",
                "first_name": "given_name",
                "last_name": "family_name",
            },
        }

        authenticator = OAuth2Authenticator(config, "https://app.example.com")

        user_info = {
            "email": "user@example.com",
            "preferred_username": "john_doe",
            "given_name": "John",
            "family_name": "Doe",
            "sub": "user-123",
        }

        result = authenticator._map_attributes(user_info)

        assert result["email"] == "user@example.com"
        assert result["username"] == "john_doe"
        assert result["first_name"] == "John"
        assert result["last_name"] == "Doe"

    def test_map_attributes_missing_field(self):
        """Map attributes with missing fields"""
        config = {
            "client_id": "client-123",
            "client_secret": "secret-456",
            "auth_url": "https://oauth.example.com/authorize",
            "token_url": "https://oauth.example.com/token",
            "user_info_url": "https://oauth.example.com/userinfo",
            "attribute_mapping": {
                "email": "email",
                "username": "preferred_username",
                "first_name": "given_name",
            },
        }

        authenticator = OAuth2Authenticator(config, "https://app.example.com")

        user_info = {
            "email": "user@example.com",
            # Missing preferred_username and given_name
        }

        result = authenticator._map_attributes(user_info)

        assert result["email"] == "user@example.com"
        assert "username" not in result
        assert "first_name" not in result

    def test_map_attributes_empty_mapping(self):
        """Map attributes with empty mapping"""
        config = {
            "client_id": "client-123",
            "client_secret": "secret-456",
            "auth_url": "https://oauth.example.com/authorize",
            "token_url": "https://oauth.example.com/token",
            "user_info_url": "https://oauth.example.com/userinfo",
            "attribute_mapping": {},
        }

        authenticator = OAuth2Authenticator(config, "https://app.example.com")

        user_info = {
            "email": "user@example.com",
            "sub": "user-123",
        }

        result = authenticator._map_attributes(user_info)

        assert result == {}
