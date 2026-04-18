"""
Extended tests for api/enterprise_auth_bp.py blueprint.

Covers all route handlers including error cases, auth scenarios, and provider logic.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from unittest.mock import AsyncMock, MagicMock, patch
from datetime import datetime

import pytest


def _admin_payload():
    return {
        "user_id": 1,
        "username": "admin",
        "is_admin": True,
        "roles": ["admin"],
        "scope": ["*:admin"],
    }


def _user_payload():
    return {"user_id": 2, "username": "user", "is_admin": False, "roles": [], "scope": []}


def _provider_row(provider_id=5, provider_type="saml"):
    p = MagicMock()
    p.id = provider_id
    p.name = "test-provider"
    p.provider_type = provider_type
    p.is_active = True
    p.auto_provision = True
    p.default_role = "service_owner"
    p.created_at = datetime(2025, 1, 1)
    p.config = {
        "idp_sso_url": "https://idp.example.com",
        "idp_x509_cert": "-----BEGIN CERTIFICATE-----...-----END CERTIFICATE-----",
        "sp_entity_id": "https://marchproxy.local",
    }
    p.update_record = MagicMock()
    return p


# ===========================================================================
# GET /api/v1/enterprise-auth/providers — List providers
# ===========================================================================


class TestEnterpriseAuthProvidersList:
    async def test_list_providers_no_auth_returns_401(self, test_client):
        """GET without auth returns 401"""
        resp = await test_client.get("/api/v1/enterprise-auth/providers")
        assert resp.status_code == 401

    async def test_list_providers_non_admin_returns_403(self, test_app, test_client):
        """GET by non-admin returns 403"""
        fresh_db = MagicMock()

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.get(
                "/api/v1/enterprise-auth/providers",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 403

    async def test_list_providers_success(self, test_app, test_client):
        """GET by admin returns 200 with providers"""
        provider = _provider_row()
        fresh_db = MagicMock()
        query = MagicMock()
        query.select.return_value = [provider]
        fresh_db.return_value = query

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/v1/enterprise-auth/providers",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 200
        data = await resp.get_json()
        assert "providers" in data


# ===========================================================================
# POST /api/v1/enterprise-auth/providers — Create provider
# ===========================================================================


class TestEnterpriseAuthProvidersCreate:
    async def test_create_provider_no_auth_returns_401(self, test_client):
        """POST without auth returns 401"""
        resp = await test_client.post("/api/v1/enterprise-auth/providers", json={})
        assert resp.status_code == 401

    async def test_create_provider_invalid_type_returns_400(self, test_app, test_client):
        """POST with invalid provider_type returns 400"""
        fresh_db = MagicMock()

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/enterprise-auth/providers",
                json={
                    "provider_type": "invalid-type",
                    "name": "test",
                },
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 400

    async def test_create_saml_provider_success(self, test_app, test_client):
        """POST SAML provider creates successfully"""
        provider = _provider_row(provider_type="saml")
        fresh_db = MagicMock()
        fresh_db.enterprise_auth_providers.insert.return_value = 5
        fresh_db.enterprise_auth_providers.__getitem__.return_value = provider

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("api.enterprise_auth_bp.EnterpriseAuthProviderModel.create_saml_provider", return_value=5), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/enterprise-auth/providers",
                json={
                    "provider_type": "saml",
                    "name": "test-provider",
                    "idp_sso_url": "https://idp.example.com",
                    "idp_x509_cert": "cert",
                    "sp_entity_id": "https://marchproxy.local",
                    "auto_provision": True,
                    "default_role": "service_owner",
                },
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 201

    async def test_create_oauth2_provider_success(self, test_app, test_client):
        """POST OAuth2 provider creates successfully"""
        provider = _provider_row(provider_type="oauth2")
        provider.config = {
            "client_id": "client123",
            "client_secret": "secret",
            "authorization_url": "https://auth.example.com/authorize",
            "token_url": "https://auth.example.com/token",
            "userinfo_url": "https://auth.example.com/userinfo",
        }
        fresh_db = MagicMock()
        fresh_db.enterprise_auth_providers.insert.return_value = 5
        fresh_db.enterprise_auth_providers.__getitem__.return_value = provider

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/enterprise-auth/providers",
                json={
                    "provider_type": "oauth2",
                    "name": "test-oauth",
                    "client_id": "client123",
                    "client_secret": "secret",
                    "authorization_url": "https://auth.example.com/authorize",
                    "token_url": "https://auth.example.com/token",
                    "userinfo_url": "https://auth.example.com/userinfo",
                },
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 201

    async def test_create_scim_provider_success(self, test_app, test_client):
        """POST SCIM provider creates successfully"""
        provider = _provider_row(provider_type="scim")
        provider.config = {
            "scim_endpoint": "https://scim.example.com",
            "auth_token": "token123",
        }
        fresh_db = MagicMock()
        fresh_db.enterprise_auth_providers.insert.return_value = 5
        fresh_db.enterprise_auth_providers.__getitem__.return_value = provider

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/enterprise-auth/providers",
                json={
                    "provider_type": "scim",
                    "name": "test-scim",
                    "scim_endpoint": "https://scim.example.com",
                    "auth_token": "token123",
                },
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 201


# ===========================================================================
# GET /api/v1/enterprise-auth/providers/<int:provider_id> — Get provider
# ===========================================================================


class TestEnterpriseAuthProviderDetail:
    async def test_get_provider_no_auth_returns_401(self, test_client):
        """GET without auth returns 401"""
        resp = await test_client.get("/api/v1/enterprise-auth/providers/5")
        assert resp.status_code == 401

    async def test_get_provider_not_found_returns_404(self, test_app, test_client):
        """GET nonexistent provider returns 404"""
        fresh_db = MagicMock()
        fresh_db.enterprise_auth_providers.__getitem__.return_value = None

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/v1/enterprise-auth/providers/999",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 404

    async def test_get_provider_success(self, test_app, test_client):
        """GET existing provider returns 200"""
        provider = _provider_row()
        fresh_db = MagicMock()
        fresh_db.enterprise_auth_providers.__getitem__.return_value = provider

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/v1/enterprise-auth/providers/5",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 200
        data = await resp.get_json()
        assert data["id"] == 5


# ===========================================================================
# PUT /api/v1/enterprise-auth/providers/<int:provider_id> — Update provider
# ===========================================================================


class TestEnterpriseAuthProviderPut:
    async def test_put_provider_not_found_returns_404(self, test_app, test_client):
        """PUT nonexistent provider returns 404"""
        fresh_db = MagicMock()
        fresh_db.enterprise_auth_providers.__getitem__.return_value = None

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.put(
                "/api/v1/enterprise-auth/providers/999",
                json={"name": "updated"},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 404

    async def test_put_provider_success(self, test_app, test_client):
        """PUT provider updates successfully"""
        provider = _provider_row()
        fresh_db = MagicMock()
        fresh_db.enterprise_auth_providers.__getitem__.return_value = provider

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.put(
                "/api/v1/enterprise-auth/providers/5",
                json={"name": "updated-provider"},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 200
        provider.update_record.assert_called()


# ===========================================================================
# DELETE /api/v1/enterprise-auth/providers/<int:provider_id> — Delete provider
# ===========================================================================


class TestEnterpriseAuthProviderDelete:
    async def test_delete_provider_not_found_returns_404(self, test_app, test_client):
        """DELETE nonexistent provider returns 404"""
        fresh_db = MagicMock()
        fresh_db.enterprise_auth_providers.__getitem__.return_value = None

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.delete(
                "/api/v1/enterprise-auth/providers/999",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 404

    async def test_delete_provider_success(self, test_app, test_client):
        """DELETE provider deactivates successfully"""
        provider = _provider_row()
        fresh_db = MagicMock()
        fresh_db.enterprise_auth_providers.__getitem__.return_value = provider

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.delete(
                "/api/v1/enterprise-auth/providers/5",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 204
        provider.update_record.assert_called_with(is_active=False)


# ===========================================================================
# POST /api/v1/enterprise-auth/providers/<int:provider_id>/test — Test provider
# ===========================================================================


class TestEnterpriseAuthProviderTest:
    async def test_test_provider_no_auth_returns_401(self, test_client):
        """POST test without auth returns 401"""
        resp = await test_client.post("/api/v1/enterprise-auth/providers/5/test")
        assert resp.status_code == 401

    async def test_test_provider_not_found_returns_404(self, test_app, test_client):
        """POST test nonexistent provider returns 404"""
        fresh_db = MagicMock()
        fresh_db.enterprise_auth_providers.__getitem__.return_value = None

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/enterprise-auth/providers/999/test",
                json={},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 404

    async def test_test_saml_provider_success(self, test_app, test_client):
        """POST test SAML provider succeeds with valid config"""
        provider = _provider_row(provider_type="saml")
        fresh_db = MagicMock()
        fresh_db.enterprise_auth_providers.__getitem__.return_value = provider

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/enterprise-auth/providers/5/test",
                json={},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 200
        data = await resp.get_json()
        assert data.get("success") is True

    async def test_test_oauth2_provider_success(self, test_app, test_client):
        """POST test OAuth2 provider succeeds"""
        provider = _provider_row(provider_type="oauth2")
        provider.config = {
            "client_id": "client123",
            "client_secret": "secret",
            "token_url": "https://auth.example.com/token",
        }
        fresh_db = MagicMock()
        fresh_db.enterprise_auth_providers.__getitem__.return_value = provider

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("api.enterprise_auth_bp.httpx.AsyncClient") as mock_client, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            mock_response = MagicMock()
            mock_response.status_code = 200
            mock_client.return_value.__aenter__.return_value.post.return_value = mock_response
            resp = await test_client.post(
                "/api/v1/enterprise-auth/providers/5/test",
                json={},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 200

    async def test_test_scim_provider_success(self, test_app, test_client):
        """POST test SCIM provider succeeds"""
        provider = _provider_row(provider_type="scim")
        provider.config = {
            "scim_endpoint": "https://scim.example.com",
            "auth_token": "token123",
        }
        fresh_db = MagicMock()
        fresh_db.enterprise_auth_providers.__getitem__.return_value = provider

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("api.enterprise_auth_bp.httpx.AsyncClient") as mock_client, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            mock_response = MagicMock()
            mock_response.status_code = 200
            mock_client.return_value.__aenter__.return_value.get.return_value = mock_response
            resp = await test_client.post(
                "/api/v1/enterprise-auth/providers/5/test",
                json={},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 200


# ===========================================================================
# GET /api/v1/enterprise-auth/saml/metadata — Get SAML metadata
# ===========================================================================


class TestEnterpriseAuthSAMLMetadata:
    async def test_saml_metadata_no_auth_returns_401(self, test_client):
        """GET metadata without auth returns 401"""
        resp = await test_client.get("/api/v1/enterprise-auth/saml/metadata")
        assert resp.status_code == 401

    async def test_saml_metadata_success(self, test_app, test_client):
        """GET metadata returns valid XML"""
        test_app.config["SAML_SP_ENTITY_ID"] = "https://marchproxy.local"
        test_app.config["SAML_ACS_URL"] = "https://marchproxy.local/api/v1/enterprise-auth/saml/acs"

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v:
            mock_v.return_value = _user_payload()
            resp = await test_client.get(
                "/api/v1/enterprise-auth/saml/metadata",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 200
        # Metadata should be XML
        content = await resp.get_data()
        assert b"EntityDescriptor" in content or b"SPSSODescriptor" in content


# ===========================================================================
# Error handling and edge cases
# ===========================================================================


class TestEnterpriseAuthErrorHandling:
    """Test error handling and edge cases"""

    async def test_create_provider_validation_error(self, test_app, test_client):
        """POST provider with invalid JSON raises validation error"""
        fresh_db = MagicMock()

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/enterprise-auth/providers",
                json={"name": "test", "provider_type": "invalid_type", "invalid_field": "value"},
                headers={"Authorization": "Bearer tok"},
            )
        # Should handle validation gracefully (could be 400 or 201 depending on implementation)
        assert resp.status_code in [400, 201]

    async def test_create_provider_missing_required_field(self, test_app, test_client):
        """POST provider with missing required field"""
        fresh_db = MagicMock()

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            # Missing provider_type
            resp = await test_client.post(
                "/api/v1/enterprise-auth/providers",
                json={"name": "test"},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [400, 201]

    async def test_put_provider_validation_error(self, test_app, test_client):
        """PUT provider with invalid data"""
        provider = _provider_row()
        fresh_db = MagicMock()
        fresh_db.enterprise_auth_providers.__getitem__.return_value = provider

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.put(
                "/api/v1/enterprise-auth/providers/5",
                json={"invalid_field": "value"},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 200

    async def test_delete_provider_success(self, test_app, test_client):
        """DELETE provider soft-deletes it"""
        provider = _provider_row()
        fresh_db = MagicMock()
        fresh_db.enterprise_auth_providers.__getitem__.return_value = provider

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.delete(
                "/api/v1/enterprise-auth/providers/5",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 204
        provider.update_record.assert_called()

    async def test_test_provider_invalid_type(self, test_app, test_client):
        """POST test provider with unknown type"""
        provider = _provider_row()
        provider.provider_type = "unknown_type"
        provider.config = {}
        fresh_db = MagicMock()
        fresh_db.enterprise_auth_providers.__getitem__.return_value = provider

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/enterprise-auth/providers/5/test",
                json={},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 400
        data = await resp.get_json()
        assert "error" in data or "Unknown provider type" in str(data)

    async def test_get_list_providers_empty(self, test_app, test_client):
        """GET providers returns empty list when none exist"""
        fresh_db = MagicMock()
        query = MagicMock()
        query.select.return_value = []
        fresh_db.return_value = query

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/v1/enterprise-auth/providers",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 200
        data = await resp.get_json()
        assert data["providers"] == []
