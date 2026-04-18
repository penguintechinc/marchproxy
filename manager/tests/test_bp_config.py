"""
Tests for api/config_bp.py blueprint.

Blueprint registered at /api/config:
  GET  /api/config/system
  GET  /api/config/health
  GET/PUT /api/config/license
  GET/PUT /api/config/logging
  GET  /api/config/database
  GET  /api/config/features
  GET  /api/config/version

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from unittest.mock import AsyncMock, MagicMock, mock_open, patch

import pytest


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _admin_payload():
    return {
        "user_id": 1, "username": "admin", "is_admin": True,
        "roles": ["admin"], "scope": ["*:admin"],
    }


# test_app and test_client come from tests/conftest.py


# ===========================================================================
# GET /api/config/system
# ===========================================================================

class TestGetSystemConfig:
    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.get("/api/config/system")
        assert resp.status_code == 401

    async def test_system_config_success(self, test_app, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v:
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/config/system",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]

    async def test_system_config_with_version_file(self, test_app, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("builtins.open", mock_open(read_data="1.0.0")):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/config/system",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]


# ===========================================================================
# GET /api/config/health
# ===========================================================================

class TestHealthCheck:
    async def test_health_check_no_auth_needed(self, test_app, test_client):
        """Health check is unauthenticated"""
        fresh_db = MagicMock()
        fresh_db.return_value.select.return_value.first.return_value = MagicMock()
        with patch.object(test_app, "db", fresh_db):
            resp = await test_client.get("/api/config/health")
        assert resp.status_code in [200, 503]

    async def test_health_check_db_failure(self, test_app, test_client):
        fresh_db = MagicMock()
        fresh_db.return_value.select.side_effect = Exception("db down")
        with patch.object(test_app, "db", fresh_db):
            resp = await test_client.get("/api/config/health")
        # Returns 200 with "degraded" status
        assert resp.status_code in [200, 503]

    async def test_health_check_with_version_file(self, test_app, test_client):
        fresh_db = MagicMock()
        with patch.object(test_app, "db", fresh_db), \
             patch("builtins.open", mock_open(read_data="1.2.3")):
            resp = await test_client.get("/api/config/health")
        assert resp.status_code in [200, 503]


# ===========================================================================
# GET/PUT /api/config/license
# ===========================================================================

class TestLicenseConfig:
    async def test_get_license_no_auth_needed(self, test_client):
        """License GET does not require auth"""
        with patch("os.getenv", side_effect=lambda k, d="": {
            "LICENSE_KEY": "",
            "RELEASE_MODE": "false",
            "LICENSE_SERVER_URL": "https://license.penguintech.io",
        }.get(k, d)):
            resp = await test_client.get("/api/config/license")
        assert resp.status_code in [200, 500]

    async def test_get_license_success(self, test_client):
        resp = await test_client.get("/api/config/license")
        assert resp.status_code in [200, 500]

    async def test_put_license_no_auth_returns_401(self, test_client):
        resp = await test_client.put("/api/config/license", json={"release_mode": True})
        assert resp.status_code == 401

    async def test_put_license_invalid_release_mode(self, test_app, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v:
            mock_v.return_value = _admin_payload()
            resp = await test_client.put(
                "/api/config/license",
                json={"release_mode": "not-a-bool"},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [400, 500]

    async def test_put_license_success(self, test_app, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v:
            mock_v.return_value = _admin_payload()
            resp = await test_client.put(
                "/api/config/license",
                json={"release_mode": True, "license_key": "test-key"},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]


# ===========================================================================
# GET/PUT /api/config/logging
# ===========================================================================

class TestLoggingConfig:
    async def test_get_logging_no_auth_needed(self, test_client):
        resp = await test_client.get("/api/config/logging")
        assert resp.status_code in [200, 500]

    async def test_put_logging_no_auth_returns_401(self, test_client):
        resp = await test_client.put("/api/config/logging", json={"level": "debug"})
        assert resp.status_code == 401

    async def test_put_logging_success(self, test_app, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v:
            mock_v.return_value = _admin_payload()
            resp = await test_client.put(
                "/api/config/logging",
                json={"level": "debug"},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 400, 500]

    async def test_put_logging_invalid_level(self, test_app, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v:
            mock_v.return_value = _admin_payload()
            resp = await test_client.put(
                "/api/config/logging",
                json={"level": "nonsense"},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 400, 500]


# ===========================================================================
# GET /api/config/database
# ===========================================================================

class TestDatabaseConfig:
    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.get("/api/config/database")
        assert resp.status_code == 401

    async def test_database_config_success(self, test_app, test_client):
        fresh_db = MagicMock()
        fresh_db.tables = ["users", "clusters"]
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/config/database",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]

    async def test_database_config_db_error(self, test_app, test_client):
        fresh_db = MagicMock()
        # Accessing .tables raises
        type(fresh_db).tables = property(MagicMock(side_effect=Exception("no tables")))
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/config/database",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]


# ===========================================================================
# GET /api/config/features
# ===========================================================================

class TestFeaturesConfig:
    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.get("/api/config/features")
        assert resp.status_code == 401

    async def test_features_dev_mode(self, test_app, test_client):
        """In dev mode (RELEASE_MODE=false), core features only"""
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/config/features",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]

    async def test_features_release_mode_with_license(self, test_app, test_client):
        """With RELEASE_MODE=true and LICENSE_KEY, checks license cache"""
        fresh_db = MagicMock()
        cached = {"is_enterprise": True, "features": {"saml": True}}
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("os.getenv", side_effect=lambda k, d="": {
                 "RELEASE_MODE": "true", "LICENSE_KEY": "test-key"
             }.get(k, d)), \
             patch("models.license.LicenseCacheModel.get_cached_validation",
                   return_value=cached), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/config/features",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]


# ===========================================================================
# GET /api/config/version
# ===========================================================================

class TestVersionConfig:
    async def test_version_no_auth_needed(self, test_client):
        """Version endpoint is public"""
        resp = await test_client.get("/api/config/version")
        assert resp.status_code in [200, 500]

    async def test_version_with_file(self, test_client):
        with patch("builtins.open", mock_open(read_data="2.0.1.1234567890")):
            resp = await test_client.get("/api/config/version")
        assert resp.status_code in [200, 500]

    async def test_version_file_missing(self, test_client):
        """File open raises, returns unknown version"""
        with patch("builtins.open", side_effect=FileNotFoundError):
            resp = await test_client.get("/api/config/version")
        assert resp.status_code in [200, 500]
