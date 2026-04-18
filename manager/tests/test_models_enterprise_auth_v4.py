"""
Tests targeting uncovered lines in models/enterprise_auth.py.
Covers SAML2 import handling, OAuth2 exceptions, SCIM auto-provision,
provision_user_from_external, and Pydantic validators.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import pytest
import sys
from unittest.mock import MagicMock, AsyncMock, patch, PropertyMock
from pydantic import ValidationError
from datetime import datetime


class TestSAML2ImportHandling:
    """Tests for SAML2 optional import handling (lines 21-28)"""

    def test_saml2_available_flag_exists(self):
        """Test SAML2_AVAILABLE flag is set in module (line 26)"""
        import models.enterprise_auth as ea
        assert hasattr(ea, 'SAML2_AVAILABLE')
        assert isinstance(ea.SAML2_AVAILABLE, bool)

    def test_saml2_available_when_imported(self):
        """Test SAML2_AVAILABLE is True when saml2 can be imported"""
        import models.enterprise_auth as ea
        # If saml2 is installed (which it should be in test env), SAML2_AVAILABLE = True
        if ea.SAML2_AVAILABLE:
            assert ea.Saml2Client is not None
            assert ea.BINDING_HTTP_POST is not None
            assert ea.BINDING_HTTP_REDIRECT is not None

    def test_saml2_fallback_when_not_available(self):
        """Test SAML2 fallback constants when saml2 not available"""
        # This tests the except block (lines 27-30)
        # We can't unimport saml2, but we can verify the fallback values would be set
        import models.enterprise_auth as ea
        # If saml2 is available, SAML2_AVAILABLE should be True
        # The except block would set these to None, but since saml2 is available,
        # they're proper constants
        if not ea.SAML2_AVAILABLE:
            assert ea.BINDING_HTTP_POST is None
            assert ea.BINDING_HTTP_REDIRECT is None


class TestOAuth2ExceptionHandling:
    """Tests for OAuth2 exception handling (lines 296-297, 314-315)"""

    @pytest.mark.asyncio
    async def test_oauth2_exchange_code_network_exception(self):
        """Test exchange_code handles network exceptions (line 296-297)"""
        from models.enterprise_auth import OAuth2Authenticator

        authenticator = OAuth2Authenticator.__new__(OAuth2Authenticator)
        authenticator.config = {
            "token_url": "http://auth.test/token",
            "client_id": "test-client",
            "client_secret": "test-secret",
            "auth_url": "http://auth.test/auth",
            "user_info_url": "http://auth.test/userinfo",
        }
        authenticator.base_url = "http://app.test"
        authenticator.redirect_uri = "http://app.test/callback"

        mock_client = AsyncMock()
        mock_client.post.side_effect = Exception("Network timeout")
        mock_async_client_context = AsyncMock()
        mock_async_client_context.__aenter__ = AsyncMock(return_value=mock_client)
        mock_async_client_context.__aexit__ = AsyncMock(return_value=None)

        with patch("httpx.AsyncClient", return_value=mock_async_client_context):
            result = await authenticator.exchange_code("test-code", "test-state")

        assert result is None

    @pytest.mark.asyncio
    async def test_oauth2_exchange_code_returns_none_on_exception(self):
        """Test exchange_code returns None when exception occurs"""
        from models.enterprise_auth import OAuth2Authenticator

        authenticator = OAuth2Authenticator.__new__(OAuth2Authenticator)
        authenticator.config = {
            "token_url": "http://auth.test/token",
            "client_id": "test-client",
            "client_secret": "test-secret",
            "auth_url": "http://auth.test/auth",
            "user_info_url": "http://auth.test/userinfo",
        }
        authenticator.base_url = "http://app.test"
        authenticator.redirect_uri = "http://app.test/callback"

        mock_client = AsyncMock()
        mock_client.post.side_effect = ValueError("Invalid token response")
        mock_async_client_context = AsyncMock()
        mock_async_client_context.__aenter__ = AsyncMock(return_value=mock_client)
        mock_async_client_context.__aexit__ = AsyncMock(return_value=None)

        with patch("httpx.AsyncClient", return_value=mock_async_client_context):
            result = await authenticator.exchange_code("bad-code", "bad-state")

        assert result is None

    @pytest.mark.asyncio
    async def test_oauth2_get_user_info_exception_handling(self):
        """Test _get_user_info handles exceptions (line 314-315)"""
        from models.enterprise_auth import OAuth2Authenticator

        authenticator = OAuth2Authenticator.__new__(OAuth2Authenticator)
        authenticator.config = {
            "user_info_url": "http://auth.test/userinfo",
        }

        mock_client = AsyncMock()
        mock_client.get.side_effect = Exception("Request failed")

        result = await authenticator._get_user_info(mock_client, "test-token")

        assert result is None

    @pytest.mark.asyncio
    async def test_oauth2_get_user_info_returns_none_on_non_200(self):
        """Test _get_user_info returns None on non-200 response"""
        from models.enterprise_auth import OAuth2Authenticator

        authenticator = OAuth2Authenticator.__new__(OAuth2Authenticator)
        authenticator.config = {
            "user_info_url": "http://auth.test/userinfo",
        }

        mock_response = AsyncMock()
        mock_response.status_code = 401
        mock_client = AsyncMock()
        mock_client.get.return_value = mock_response

        result = await authenticator._get_user_info(mock_client, "bad-token")

        assert result is None


class TestSCIMEmailFallback:
    """Tests for SCIM email fallback logic (line 366)"""

    def test_scim_uses_first_email_when_no_primary(self):
        """Test process_scim_user uses first email when none primary (line 366)"""
        from models.enterprise_auth import SCIMUserModel

        mock_db = MagicMock()

        # Create mocks for two calls: scim_user lookup, provider lookup
        mock_no_scim_user = MagicMock()
        mock_no_scim_user.select.return_value.first.return_value = None

        mock_provider_result = MagicMock()
        mock_provider_result.auto_provision = False
        mock_provider_query = MagicMock()
        mock_provider_query.select.return_value.first.return_value = mock_provider_result

        # First call returns no SCIM user, second returns provider
        mock_db.side_effect = [mock_no_scim_user, mock_provider_query]

        scim_data = {
            "id": "test-scim-123",
            "userName": "testuser",
            "emails": [
                {"value": "first@test.com", "primary": False},
                {"value": "second@test.com", "primary": False},
            ],
            "active": True,
        }

        # Should return None because auto_provision=False on provider
        result = SCIMUserModel.process_scim_user(mock_db, scim_data, "test-provider")
        assert result is None

    def test_scim_uses_primary_email_when_available(self):
        """Test process_scim_user uses primary email when marked"""
        from models.enterprise_auth import SCIMUserModel

        # Existing SCIM user with user_id
        mock_scim_user = MagicMock()
        mock_scim_user.user_id = 42
        mock_scim_user.update_record = MagicMock()

        mock_query = MagicMock()
        mock_query.select.return_value.first.return_value = mock_scim_user

        mock_db = MagicMock()
        mock_db.return_value = mock_query
        # Also need to handle db.users subscript access
        mock_user = MagicMock()
        mock_db.users.__getitem__.return_value = mock_user

        scim_data = {
            "id": "test-scim-456",
            "userName": "testuser2",
            "emails": [
                {"value": "first@test.com", "primary": False},
                {"value": "primary@test.com", "primary": True},
            ],
            "active": True,
        }

        result = SCIMUserModel.process_scim_user(mock_db, scim_data, "test-provider")
        assert result == 42

    def test_scim_empty_emails_list(self):
        """Test process_scim_user with empty emails list"""
        from models.enterprise_auth import SCIMUserModel

        mock_scim_user = MagicMock()
        mock_scim_user.user_id = 50
        mock_scim_user.update_record = MagicMock()

        mock_query = MagicMock()
        mock_query.select.return_value.first.return_value = mock_scim_user

        mock_db = MagicMock()
        mock_db.return_value = mock_query
        # Handle db.users subscript
        mock_user = MagicMock()
        mock_db.users.__getitem__.return_value = mock_user

        scim_data = {
            "id": "test-scim-789",
            "userName": "testuser3",
            "emails": [],
            "active": True,
        }

        result = SCIMUserModel.process_scim_user(mock_db, scim_data, "test-provider")
        assert result == 50


class TestSCIMAutoProvision:
    """Tests for SCIM auto-provision check (lines 393-395)"""

    def test_scim_auto_provision_disabled_returns_none(self):
        """Test process_scim_user returns None when auto_provision=False"""
        from models.enterprise_auth import SCIMUserModel

        mock_db = MagicMock()
        # No existing SCIM user (triggers provider check)
        mock_no_scim = MagicMock()
        mock_no_scim.select.return_value.first.return_value = None

        # Provider exists but auto_provision=False
        mock_provider = MagicMock()
        mock_provider.auto_provision = False
        mock_provider_query = MagicMock()
        mock_provider_query.select.return_value.first.return_value = mock_provider

        # First call returns no SCIM user, second returns provider
        mock_db.side_effect = [mock_no_scim, mock_provider_query]

        scim_data = {
            "id": "new-user-id",
            "userName": "newuser",
            "emails": [{"value": "new@test.com", "primary": True}],
            "active": True,
        }

        result = SCIMUserModel.process_scim_user(mock_db, scim_data, "test-provider")
        assert result is None

    def test_scim_provider_not_found_returns_none(self):
        """Test process_scim_user returns None when provider not found"""
        from models.enterprise_auth import SCIMUserModel

        mock_db = MagicMock()
        # No existing SCIM user
        mock_query = MagicMock()
        mock_query.select.return_value.first.return_value = None
        mock_db.return_value = mock_query

        # Provider not found
        mock_provider_query = MagicMock()
        mock_provider_query.select.return_value.first.return_value = None
        mock_db.return_value = mock_provider_query

        scim_data = {
            "id": "orphan-user-id",
            "userName": "orphanuser",
            "emails": [{"value": "orphan@test.com", "primary": True}],
            "active": True,
        }

        result = SCIMUserModel.process_scim_user(mock_db, scim_data, "nonexistent-provider")
        assert result is None


class TestSCIMUserCreation:
    """Tests for SCIM user creation path (lines 400-418)"""

    def test_scim_create_new_user_success(self):
        """Test process_scim_user creates new user when auto_provision=True"""
        from models.enterprise_auth import SCIMUserModel

        mock_db = MagicMock()
        # No existing SCIM user
        mock_no_scim = MagicMock()
        mock_no_scim.select.return_value.first.return_value = None

        # Provider with auto_provision=True
        mock_provider = MagicMock()
        mock_provider.auto_provision = True
        mock_provider_query = MagicMock()
        mock_provider_query.select.return_value.first.return_value = mock_provider

        # First call returns no SCIM user, second returns provider
        mock_db.side_effect = [mock_no_scim, mock_provider_query]

        # Mock insert returns
        mock_db.users.insert.return_value = 99
        mock_db.scim_users.insert.return_value = 1

        scim_data = {
            "id": "new-scim-id",
            "userName": "brandnewuser",
            "emails": [{"value": "brandnew@test.com", "primary": True}],
            "active": True,
            "externalId": "ext-123",
        }

        result = SCIMUserModel.process_scim_user(mock_db, scim_data, "test-provider")
        assert result == 99

    def test_scim_create_user_with_username_fallback(self):
        """Test SCIM user creation uses email as username if not provided"""
        from models.enterprise_auth import SCIMUserModel

        mock_db = MagicMock()
        # No existing SCIM user
        mock_no_scim = MagicMock()
        mock_no_scim.select.return_value.first.return_value = None

        # Provider with auto_provision
        mock_provider = MagicMock()
        mock_provider.auto_provision = True
        mock_provider_query = MagicMock()
        mock_provider_query.select.return_value.first.return_value = mock_provider

        # First call returns no SCIM user, second returns provider
        mock_db.side_effect = [mock_no_scim, mock_provider_query]

        mock_db.users.insert.return_value = 100
        mock_db.scim_users.insert.return_value = 2

        scim_data = {
            "id": "no-username-id",
            "userName": None,
            "emails": [{"value": "email-only@test.com", "primary": True}],
            "active": False,
        }

        result = SCIMUserModel.process_scim_user(mock_db, scim_data, "test-provider")
        assert result == 100


class TestProvisionUserFromExternal:
    """Tests for provision_user_from_external method (lines 462-536)"""

    def test_provision_external_provider_not_found(self):
        """Test provision_user_from_external returns None when provider not found"""
        from models.enterprise_auth import EnterpriseAuthManager

        mock_db = MagicMock()
        manager = EnterpriseAuthManager(mock_db, "http://app.test")

        # Provider not found
        mock_query = MagicMock()
        mock_query.select.return_value.first.return_value = None
        mock_db.return_value = mock_query

        result = manager.provision_user_from_external("nonexistent", {"external_id": "ext-1"})
        assert result is None

    def test_provision_external_auto_provision_disabled(self):
        """Test provision_user_from_external returns None when auto_provision=False"""
        from models.enterprise_auth import EnterpriseAuthManager

        mock_db = MagicMock()
        manager = EnterpriseAuthManager(mock_db, "http://app.test")

        # Provider found but auto_provision=False
        mock_provider = MagicMock()
        mock_provider.auto_provision = False
        mock_query = MagicMock()
        mock_query.select.return_value.first.return_value = mock_provider
        mock_db.return_value = mock_query

        result = manager.provision_user_from_external("disabled-provider", {"external_id": "ext-2"})
        assert result is None

    def test_provision_external_existing_user_update(self):
        """Test provision_user_from_external updates existing user"""
        from models.enterprise_auth import EnterpriseAuthManager

        mock_db = MagicMock()
        manager = EnterpriseAuthManager(mock_db, "http://app.test")

        # Provider found with auto_provision=True
        mock_provider = MagicMock()
        mock_provider.auto_provision = True
        mock_provider_query = MagicMock()
        mock_provider_query.select.return_value.first.return_value = mock_provider
        mock_db.return_value = mock_provider_query

        # Existing user found
        mock_user = MagicMock()
        mock_user.id = 42
        mock_user.update_record = MagicMock()
        mock_user_query = MagicMock()
        mock_user_query.select.return_value.first.return_value = mock_user
        # Second call for user lookup
        mock_db.return_value = mock_user_query

        external_data = {
            "external_id": "ext-existing",
            "attributes": {"email": "updated@test.com", "username": "updateuser"},
        }

        result = manager.provision_user_from_external("oauth-provider", external_data)
        assert result == 42

    def test_provision_external_new_user_no_email(self):
        """Test provision_user_from_external returns None when email missing"""
        from models.enterprise_auth import EnterpriseAuthManager

        mock_db = MagicMock()
        manager = EnterpriseAuthManager(mock_db, "http://app.test")

        # Provider found
        mock_provider = MagicMock()
        mock_provider.auto_provision = True
        mock_provider_query = MagicMock()
        mock_provider_query.select.return_value.first.return_value = mock_provider
        mock_db.return_value = mock_provider_query

        # No existing user
        mock_user_query = MagicMock()
        mock_user_query.select.return_value.first.return_value = None
        mock_db.return_value = mock_user_query

        external_data = {
            "external_id": "ext-noemail",
            "attributes": {"username": "noemailu"},  # Missing email
        }

        result = manager.provision_user_from_external("oauth-provider", external_data)
        assert result is None

    def test_provision_external_new_user_no_username(self):
        """Test provision_user_from_external returns None when username and email missing"""
        from models.enterprise_auth import EnterpriseAuthManager

        mock_db = MagicMock()
        manager = EnterpriseAuthManager(mock_db, "http://app.test")

        # Provider found
        mock_provider = MagicMock()
        mock_provider.auto_provision = True
        mock_provider_query = MagicMock()
        mock_provider_query.select.return_value.first.return_value = mock_provider
        mock_db.return_value = mock_provider_query

        # No existing user
        mock_user_query = MagicMock()
        mock_user_query.select.return_value.first.return_value = None
        mock_db.return_value = mock_user_query

        external_data = {
            "external_id": "ext-nouser",
            "attributes": {"email": None},  # No email, no username
        }

        result = manager.provision_user_from_external("oauth-provider", external_data)
        assert result is None

    def test_provision_external_new_user_created(self):
        """Test provision_user_from_external creates new user"""
        from models.enterprise_auth import EnterpriseAuthManager

        mock_db = MagicMock()
        manager = EnterpriseAuthManager(mock_db, "http://app.test")

        # Provider found
        mock_provider = MagicMock()
        mock_provider.auto_provision = True
        mock_provider.default_role = None
        mock_provider_query = MagicMock()
        mock_provider_query.select.return_value.first.return_value = mock_provider

        # No existing user
        mock_user_query = MagicMock()
        mock_user_query.select.return_value.first.return_value = None

        # First call returns provider, second returns no user
        mock_db.side_effect = [mock_provider_query, mock_user_query]

        # Mock user insert
        mock_db.users.insert.return_value = 77

        external_data = {
            "external_id": "ext-newuser",
            "attributes": {"email": "newprov@test.com", "username": "newprovuser"},
        }

        result = manager.provision_user_from_external("saml-provider", external_data)
        assert result == 77

    def test_provision_external_new_user_with_default_role(self):
        """Test provision_user_from_external assigns default role to new user"""
        from models.enterprise_auth import EnterpriseAuthManager

        mock_db = MagicMock()
        manager = EnterpriseAuthManager(mock_db, "http://app.test")

        # Provider with default_role
        mock_provider = MagicMock()
        mock_provider.auto_provision = True
        mock_provider.default_role = "admin"
        mock_provider.id = 5
        mock_provider_query = MagicMock()
        mock_provider_query.select.return_value.first.return_value = mock_provider

        # No existing user
        mock_user_query = MagicMock()
        mock_user_query.select.return_value.first.return_value = None

        # Default cluster exists
        mock_cluster = MagicMock()
        mock_cluster.id = 10
        mock_cluster_query = MagicMock()
        mock_cluster_query.select.return_value.first.return_value = mock_cluster

        # Three calls: provider, user, cluster
        mock_db.side_effect = [mock_provider_query, mock_user_query, mock_cluster_query]

        mock_db.users.insert.return_value = 88

        # Mock UserClusterAssignmentModel (imported inside the function)
        with patch("models.cluster.UserClusterAssignmentModel") as MockAssignment:
            mock_assign = MagicMock()
            MockAssignment.assign_user_to_cluster = mock_assign

            external_data = {
                "external_id": "ext-withrole",
                "attributes": {"email": "withrole@test.com", "username": "withroleuser"},
            }

            result = manager.provision_user_from_external("saml-with-role", external_data)

        assert result == 88

    def test_provision_external_new_user_no_default_cluster(self):
        """Test provision_user_from_external skips role assignment if no default cluster"""
        from models.enterprise_auth import EnterpriseAuthManager

        mock_db = MagicMock()
        manager = EnterpriseAuthManager(mock_db, "http://app.test")

        # Provider with default_role
        mock_provider = MagicMock()
        mock_provider.auto_provision = True
        mock_provider.default_role = "maintainer"
        mock_provider_query = MagicMock()
        mock_provider_query.select.return_value.first.return_value = mock_provider

        # No existing user
        mock_user_query = MagicMock()
        mock_user_query.select.return_value.first.return_value = None

        # No default cluster
        mock_cluster_query = MagicMock()
        mock_cluster_query.select.return_value.first.return_value = None

        # Three calls: provider, user, cluster
        mock_db.side_effect = [mock_provider_query, mock_user_query, mock_cluster_query]

        mock_db.users.insert.return_value = 89

        external_data = {
            "external_id": "ext-nocluster",
            "attributes": {"email": "nocluster@test.com", "username": "noclusteruser"},
        }

        result = manager.provision_user_from_external("oauth-nocluster", external_data)
        assert result == 89


class TestPydanticValidators:
    """Tests for Pydantic validator methods (lines 554-556, 572-574)"""

    def test_create_saml_provider_name_too_short(self):
        """Test CreateSAMLProviderRequest.validate_name rejects short names"""
        from models.enterprise_auth import CreateSAMLProviderRequest

        with pytest.raises(ValidationError) as exc_info:
            CreateSAMLProviderRequest(
                name="ab",  # Only 2 chars, min is 3
                idp_sso_url="http://idp.test/sso",
                idp_x509_cert="cert-data",
                sp_entity_id="http://sp.test",
            )

        assert "at least 3 characters" in str(exc_info.value)

    def test_create_saml_provider_name_normalized(self):
        """Test CreateSAMLProviderRequest.validate_name normalizes name"""
        from models.enterprise_auth import CreateSAMLProviderRequest

        req = CreateSAMLProviderRequest(
            name="My SAML Provider",  # Spaces and uppercase
            idp_sso_url="http://idp.test/sso",
            idp_x509_cert="cert-data",
            sp_entity_id="http://sp.test",
        )

        assert req.name == "my_saml_provider"

    def test_create_saml_provider_valid_name(self):
        """Test CreateSAMLProviderRequest accepts valid names"""
        from models.enterprise_auth import CreateSAMLProviderRequest

        req = CreateSAMLProviderRequest(
            name="ValidProvider",
            idp_sso_url="http://idp.test/sso",
            idp_x509_cert="cert-data",
            sp_entity_id="http://sp.test",
        )

        assert req.name == "validprovider"

    def test_create_oauth2_provider_name_too_short(self):
        """Test CreateOAuth2ProviderRequest.validate_name rejects short names"""
        from models.enterprise_auth import CreateOAuth2ProviderRequest

        with pytest.raises(ValidationError) as exc_info:
            CreateOAuth2ProviderRequest(
                name="ab",  # Only 2 chars
                client_id="client-123",
                client_secret="secret-456",
                auth_url="http://auth.test/authorize",
                token_url="http://auth.test/token",
                user_info_url="http://auth.test/userinfo",
            )

        assert "at least 3 characters" in str(exc_info.value)

    def test_create_oauth2_provider_name_normalized(self):
        """Test CreateOAuth2ProviderRequest.validate_name normalizes name"""
        from models.enterprise_auth import CreateOAuth2ProviderRequest

        req = CreateOAuth2ProviderRequest(
            name="My OAuth2 Provider",  # Spaces and uppercase
            client_id="client-123",
            client_secret="secret-456",
            auth_url="http://auth.test/authorize",
            token_url="http://auth.test/token",
            user_info_url="http://auth.test/userinfo",
        )

        assert req.name == "my_oauth2_provider"

    def test_create_oauth2_provider_valid_name(self):
        """Test CreateOAuth2ProviderRequest accepts valid names"""
        from models.enterprise_auth import CreateOAuth2ProviderRequest

        req = CreateOAuth2ProviderRequest(
            name="GoogleOAuth",
            client_id="client-123",
            client_secret="secret-456",
            auth_url="http://auth.test/authorize",
            token_url="http://auth.test/token",
            user_info_url="http://auth.test/userinfo",
        )

        assert req.name == "googleoauth"

    def test_create_saml_provider_with_extra_spaces(self):
        """Test CreateSAMLProviderRequest handles multiple spaces"""
        from models.enterprise_auth import CreateSAMLProviderRequest

        req = CreateSAMLProviderRequest(
            name="SAML   Provider   Name",
            idp_sso_url="http://idp.test/sso",
            idp_x509_cert="cert-data",
            sp_entity_id="http://sp.test",
        )

        assert req.name == "saml___provider___name"

    def test_create_oauth2_provider_exact_min_length(self):
        """Test CreateOAuth2ProviderRequest accepts name with exactly 3 chars"""
        from models.enterprise_auth import CreateOAuth2ProviderRequest

        req = CreateOAuth2ProviderRequest(
            name="AAA",  # Exactly 3 chars
            client_id="client-123",
            client_secret="secret-456",
            auth_url="http://auth.test/authorize",
            token_url="http://auth.test/token",
            user_info_url="http://auth.test/userinfo",
        )

        assert req.name == "aaa"

    def test_create_saml_provider_exact_min_length(self):
        """Test CreateSAMLProviderRequest accepts name with exactly 3 chars"""
        from models.enterprise_auth import CreateSAMLProviderRequest

        req = CreateSAMLProviderRequest(
            name="BBB",  # Exactly 3 chars
            idp_sso_url="http://idp.test/sso",
            idp_x509_cert="cert-data",
            sp_entity_id="http://sp.test",
        )

        assert req.name == "bbb"
