"""
Comprehensive tests for License Blueprint API endpoints

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import pytest
from datetime import datetime, timedelta
from unittest.mock import AsyncMock, MagicMock, patch
import json

from api.license_bp import (
    license_bp,
    ValidateLicenseRequest,
    LicenseStatusResponse,
    LicenseKeepaliveRequest,
)


# ============================================================================
# Fixtures
# ============================================================================

@pytest.fixture
def app():
    """Create a test Quart application"""
    from quart import Quart

    app = Quart(__name__)
    app.config["TESTING"] = True
    app.register_blueprint(license_bp)

    # Mock database
    app.db = MagicMock()
    app.db.license_cache = MagicMock()
    app.config["LICENSE_SERVER_URL"] = "https://license.penguintech.io"
    app.config["PRODUCT_NAME"] = "marchproxy"

    return app


@pytest.fixture
def mock_auth():
    """Mock authentication context"""
    return {
        "user_id": "user1",
        "sub": "user1",
        "scope": "clusters:read clusters:write license:read license:write",
        "roles": ["admin"],
        "tenant": "test",
    }


@pytest.fixture
def valid_license_key():
    """Valid license key format"""
    return "PENG-1234-5678-9012-3456-ABCD"


@pytest.fixture
def sample_license_response():
    """Sample license server response"""
    return {
        "valid": True,
        "tier": "enterprise",
        "max_proxies": 10,
        "features": [
            {"name": "multi_cluster", "entitled": True},
            {"name": "saml_authentication", "entitled": True},
        ],
        "expires_at": (datetime.utcnow() + timedelta(days=365)).isoformat() + "Z",
    }


# ============================================================================
# POST /api/v1/license/validate Tests
# ============================================================================

class TestValidateLicenseEndpoint:
    """Tests for /validate endpoint"""

    def test_validate_license_request_model(self):
        """Test ValidateLicenseRequest model validation"""
        # Valid request
        req = ValidateLicenseRequest(license_key="PENG-1234-5678-9012-3456-ABCD")
        assert req.license_key == "PENG-1234-5678-9012-3456-ABCD"

    @pytest.mark.asyncio
    async def test_validate_license_cached(self, app, mock_auth, valid_license_key):
        """Test license validation with cache hit"""
        cached_data = {
            "is_valid": True,
            "is_enterprise": True,
            "max_proxies": 10,
            "features": {"multi_cluster": True},
            "expires_at": datetime.utcnow() + timedelta(days=30),
        }

        with patch("middleware.auth._validate_token", return_value=mock_auth):
            with patch("api.license_bp.LicenseCacheModel") as mock_cache:
                mock_cache.get_cached_validation.return_value = cached_data

                async with app.test_client() as client:
                    response = await client.post(
                        "/api/v1/license/validate",
                        json={"license_key": valid_license_key},
                        headers={"Authorization": "Bearer mock-token"},
                    )

                assert response.status_code == 200
                data = await response.get_json()
                assert data["is_valid"] is True

    @pytest.mark.asyncio
    async def test_validate_license_invalid_json(self, app, mock_auth):
        """Test validation with invalid JSON"""
        with patch("middleware.auth._validate_token", return_value=mock_auth):
            async with app.test_client() as client:
                response = await client.post(
                    "/api/v1/license/validate",
                    json={"invalid_field": "value"},
                    headers={"Authorization": "Bearer mock-token"},
                )

            assert response.status_code == 400
            data = await response.get_json()
            assert "error" in data

    def test_license_status_response_model(self):
        """Test LicenseStatusResponse model"""
        resp = LicenseStatusResponse(
            is_valid=True,
            is_enterprise=True,
            max_proxies=10,
            tier="enterprise",
            features={"multi_cluster": True},
        )
        assert resp.is_valid is True
        assert resp.tier == "enterprise"


# ============================================================================
# GET /api/v1/license/status Tests
# ============================================================================

class TestGetLicenseStatusEndpoint:
    """Tests for /status endpoint"""

    @pytest.mark.asyncio
    async def test_get_license_status_success(self, app, mock_auth, valid_license_key):
        """Test successful license status retrieval"""
        cached_data = {
            "is_valid": True,
            "is_enterprise": True,
            "max_proxies": 10,
            "features": {"multi_cluster": True},
            "expires_at": datetime.utcnow() + timedelta(days=30),
            "last_keepalive": datetime.utcnow(),
        }

        with patch("middleware.auth._validate_token", return_value=mock_auth):
            with patch("api.license_bp.LicenseCacheModel") as mock_cache:
                mock_cache.get_cached_validation.return_value = cached_data

                async with app.test_client() as client:
                    response = await client.get(
                        f"/api/v1/license/status?license_key={valid_license_key}",
                        headers={"Authorization": "Bearer mock-token"},
                    )

                assert response.status_code == 200
                data = await response.get_json()
                assert data["is_valid"] is True
                assert data["is_enterprise"] is True

    @pytest.mark.asyncio
    async def test_get_license_status_missing_key(self, app, mock_auth):
        """Test status endpoint without license key"""
        with patch("middleware.auth._validate_token", return_value=mock_auth):
            async with app.test_client() as client:
                response = await client.get(
                    "/api/v1/license/status",
                    headers={"Authorization": "Bearer mock-token"},
                )

            assert response.status_code == 400
            data = await response.get_json()
            assert "error" in data

    @pytest.mark.asyncio
    async def test_get_license_status_not_validated(self, app, mock_auth, valid_license_key):
        """Test status when license not validated yet"""
        with patch("middleware.auth._validate_token", return_value=mock_auth):
            with patch("api.license_bp.LicenseCacheModel") as mock_cache:
                mock_cache.get_cached_validation.return_value = None

                async with app.test_client() as client:
                    response = await client.get(
                        f"/api/v1/license/status?license_key={valid_license_key}",
                        headers={"Authorization": "Bearer mock-token"},
                    )

                assert response.status_code == 404

    @pytest.mark.asyncio
    async def test_get_license_status_missed_keepalive(self, app, mock_auth, valid_license_key):
        """Test status when enterprise license missed keepalives"""
        cached_data = {
            "is_valid": True,
            "is_enterprise": True,
            "max_proxies": 10,
            "features": {},
            "last_keepalive": datetime.utcnow() - timedelta(hours=25),
        }

        with patch("middleware.auth._validate_token", return_value=mock_auth):
            with patch("api.license_bp.LicenseCacheModel") as mock_cache:
                mock_cache.get_cached_validation.return_value = cached_data

                async with app.test_client() as client:
                    response = await client.get(
                        f"/api/v1/license/status?license_key={valid_license_key}",
                        headers={"Authorization": "Bearer mock-token"},
                    )

                assert response.status_code == 200
                data = await response.get_json()
                assert data["is_valid"] is False

    @pytest.mark.asyncio
    async def test_get_license_status_server_error(self, app, mock_auth, valid_license_key):
        """Test status endpoint with server error"""
        with patch("middleware.auth._validate_token", return_value=mock_auth):
            with patch("api.license_bp.LicenseCacheModel") as mock_cache:
                mock_cache.get_cached_validation.side_effect = Exception("DB error")

                async with app.test_client() as client:
                    response = await client.get(
                        f"/api/v1/license/status?license_key={valid_license_key}",
                        headers={"Authorization": "Bearer mock-token"},
                    )

                assert response.status_code == 500


# ============================================================================
# POST /api/v1/license/keepalive Tests
# ============================================================================

class TestSendKeepaliveEndpoint:
    """Tests for /keepalive endpoint"""

    @pytest.mark.asyncio
    async def test_send_keepalive_success(self, app, mock_auth, valid_license_key):
        """Test successful keepalive send"""
        cached_data = {
            "is_valid": True,
            "is_enterprise": True,
            "id": 1,
            "keepalive_count": 5,
        }

        with patch("middleware.auth._validate_token", return_value=mock_auth):
            with patch("httpx.AsyncClient") as mock_client_class:
                mock_response = AsyncMock()
                mock_response.status_code = 200
                mock_response.json.return_value = {
                    "message": "Keepalive received",
                    "next_keepalive_due": (datetime.utcnow() + timedelta(hours=24)).isoformat(),
                }

                mock_client = AsyncMock()
                mock_client.__aenter__.return_value = mock_client
                mock_client.post.return_value = mock_response
                mock_client_class.return_value = mock_client

                with patch("api.license_bp.LicenseCacheModel") as mock_cache:
                    mock_cache.get_cached_validation.return_value = cached_data

                    async with app.test_client() as client:
                        response = await client.post(
                            "/api/v1/license/keepalive",
                            json={"license_key": valid_license_key},
                            headers={"Authorization": "Bearer mock-token"},
                        )

                    assert response.status_code == 200
                    data = await response.get_json()
                    assert "Keepalive sent successfully" in data["message"]

    @pytest.mark.asyncio
    async def test_send_keepalive_not_enterprise(self, app, mock_auth, valid_license_key):
        """Test keepalive on non-enterprise license"""
        cached_data = {
            "is_valid": True,
            "is_enterprise": False,
        }

        with patch("middleware.auth._validate_token", return_value=mock_auth):
            with patch("api.license_bp.LicenseCacheModel") as mock_cache:
                mock_cache.get_cached_validation.return_value = cached_data

                async with app.test_client() as client:
                    response = await client.post(
                        "/api/v1/license/keepalive",
                        json={"license_key": valid_license_key},
                        headers={"Authorization": "Bearer mock-token"},
                    )

                assert response.status_code == 400
                data = await response.get_json()
                assert "not enterprise" in data["error"]

    @pytest.mark.asyncio
    async def test_send_keepalive_invalid_json(self, app, mock_auth):
        """Test keepalive with invalid JSON"""
        with patch("middleware.auth._validate_token", return_value=mock_auth):
            async with app.test_client() as client:
                response = await client.post(
                    "/api/v1/license/keepalive",
                    json={"invalid_field": "value"},
                    headers={"Authorization": "Bearer mock-token"},
                )

            assert response.status_code == 400

    def test_license_keepalive_request_model(self):
        """Test LicenseKeepaliveRequest model"""
        req = LicenseKeepaliveRequest(
            license_key="PENG-1234-5678-9012-3456-ABCD",
            usage_stats={"proxies": 5},
        )
        assert req.license_key == "PENG-1234-5678-9012-3456-ABCD"
        assert req.usage_stats == {"proxies": 5}

    @pytest.mark.asyncio
    async def test_send_keepalive_with_usage_stats(self, app, mock_auth, valid_license_key):
        """Test keepalive with usage statistics"""
        cached_data = {
            "is_valid": True,
            "is_enterprise": True,
            "id": 1,
            "keepalive_count": 0,
        }

        usage_stats = {
            "active_proxies": 5,
            "active_users": 3,
            "active_clusters": 2,
        }

        with patch("middleware.auth._validate_token", return_value=mock_auth):
            with patch("httpx.AsyncClient") as mock_client_class:
                mock_response = AsyncMock()
                mock_response.status_code = 200
                mock_response.json.return_value = {"message": "Keepalive received"}

                mock_client = AsyncMock()
                mock_client.__aenter__.return_value = mock_client
                mock_client.post.return_value = mock_response
                mock_client_class.return_value = mock_client

                with patch("api.license_bp.LicenseCacheModel") as mock_cache:
                    mock_cache.get_cached_validation.return_value = cached_data

                    async with app.test_client() as client:
                        response = await client.post(
                            "/api/v1/license/keepalive",
                            json={
                                "license_key": valid_license_key,
                                "usage_stats": usage_stats,
                            },
                            headers={"Authorization": "Bearer mock-token"},
                        )

                    assert response.status_code == 200


# ============================================================================
# GET /api/v1/license/features Tests
# ============================================================================

class TestCheckFeaturesEndpoint:
    """Tests for /features endpoint"""

    @pytest.mark.asyncio
    async def test_check_features_success(self, app, mock_auth, valid_license_key):
        """Test successful features check"""
        cached_data = {
            "is_valid": True,
            "is_enterprise": True,
            "features": {
                "multi_cluster": True,
                "saml_authentication": True,
                "oauth2_authentication": False,
            },
        }

        with patch("middleware.auth._validate_token", return_value=mock_auth):
            with patch("api.license_bp.LicenseCacheModel") as mock_cache:
                mock_cache.get_cached_validation.return_value = cached_data

                async with app.test_client() as client:
                    response = await client.get(
                        f"/api/v1/license/features?license_key={valid_license_key}",
                        headers={"Authorization": "Bearer mock-token"},
                    )

                assert response.status_code == 200
                data = await response.get_json()
                assert data["tier"] == "enterprise"
                assert data["features"]["multi_cluster"] is True

    @pytest.mark.asyncio
    async def test_check_features_missing_key(self, app, mock_auth):
        """Test features endpoint without license key"""
        with patch("middleware.auth._validate_token", return_value=mock_auth):
            async with app.test_client() as client:
                response = await client.get(
                    "/api/v1/license/features",
                    headers={"Authorization": "Bearer mock-token"},
                )

            assert response.status_code == 400

    @pytest.mark.asyncio
    async def test_check_features_not_validated(self, app, mock_auth, valid_license_key):
        """Test features when license not validated"""
        with patch("middleware.auth._validate_token", return_value=mock_auth):
            with patch("api.license_bp.LicenseCacheModel") as mock_cache:
                mock_cache.get_cached_validation.return_value = None

                async with app.test_client() as client:
                    response = await client.get(
                        f"/api/v1/license/features?license_key={valid_license_key}",
                        headers={"Authorization": "Bearer mock-token"},
                    )

                assert response.status_code == 404

    @pytest.mark.asyncio
    async def test_check_features_community(self, app, mock_auth, valid_license_key):
        """Test features check for community license"""
        cached_data = {
            "is_valid": True,
            "is_enterprise": False,
            "features": {},
        }

        with patch("middleware.auth._validate_token", return_value=mock_auth):
            with patch("api.license_bp.LicenseCacheModel") as mock_cache:
                mock_cache.get_cached_validation.return_value = cached_data

                async with app.test_client() as client:
                    response = await client.get(
                        f"/api/v1/license/features?license_key={valid_license_key}",
                        headers={"Authorization": "Bearer mock-token"},
                    )

                assert response.status_code == 200
                data = await response.get_json()
                assert data["tier"] == "community"

    @pytest.mark.asyncio
    async def test_check_features_server_error(self, app, mock_auth, valid_license_key):
        """Test features endpoint with server error"""
        with patch("middleware.auth._validate_token", return_value=mock_auth):
            with patch("api.license_bp.LicenseCacheModel") as mock_cache:
                mock_cache.get_cached_validation.side_effect = Exception("DB error")

                async with app.test_client() as client:
                    response = await client.get(
                        f"/api/v1/license/features?license_key={valid_license_key}",
                        headers={"Authorization": "Bearer mock-token"},
                    )

                assert response.status_code == 500


# ============================================================================
# Pydantic Model Tests
# ============================================================================

class TestPydanticModels:
    """Tests for API request/response models"""

    def test_validate_license_request_valid(self):
        """Test valid license validation request"""
        req = ValidateLicenseRequest(license_key="PENG-1234-5678-9012-3456-ABCD")
        assert req.license_key == "PENG-1234-5678-9012-3456-ABCD"

    def test_license_keepalive_request_minimal(self):
        """Test minimal keepalive request"""
        req = LicenseKeepaliveRequest(license_key="PENG-1234-5678-9012-3456-ABCD")
        assert req.license_key == "PENG-1234-5678-9012-3456-ABCD"
        assert req.usage_stats is None

    def test_license_keepalive_request_with_stats(self):
        """Test keepalive request with usage stats"""
        stats = {
            "active_proxies": 5,
            "active_users": 3,
        }
        req = LicenseKeepaliveRequest(
            license_key="PENG-1234-5678-9012-3456-ABCD",
            usage_stats=stats,
        )
        assert req.usage_stats == stats

    def test_license_status_response(self):
        """Test license status response model"""
        resp = LicenseStatusResponse(
            is_valid=True,
            is_enterprise=True,
            max_proxies=10,
            tier="enterprise",
            features={"multi_cluster": True},
        )
        assert resp.is_valid is True
        assert resp.tier == "enterprise"
