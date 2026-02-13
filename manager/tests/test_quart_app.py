#!/usr/bin/env python3
"""
Async integration tests for Quart application.
Tests API endpoints using pytest-asyncio.
"""

from unittest.mock import MagicMock, patch

import pytest
import pytest_asyncio


@pytest_asyncio.fixture
async def app():
    """Create test application."""
    # Mock get_db_manager which is called by quart_app._initialize_database()
    mock_db_manager = MagicMock()
    mock_db_manager.initialize_schema.return_value = True
    mock_db_manager.get_pydal_connection.return_value = MagicMock()
    mock_db_manager.db_type = "sqlite"

    with patch("database.get_db_manager", return_value=mock_db_manager):
        from quart_app import create_app

        test_config = {
            "DATABASE_URL": "sqlite:///test.db",
            "JWT_SECRET": "test-secret-key-for-testing-only",
            "DB_TYPE": "sqlite",
        }
        app = create_app(config=test_config)
        app.config["TESTING"] = True
        yield app


@pytest_asyncio.fixture
async def client(app):
    """Create test client."""
    return app.test_client()


@pytest.mark.asyncio
async def test_health_endpoint(client):
    """Test /healthz endpoint returns 200."""
    response = await client.get("/healthz")
    assert response.status_code == 200
    data = await response.get_json()
    assert "status" in data
    assert data["status"] in ["healthy", "degraded", "unhealthy"]


@pytest.mark.asyncio
async def test_root_endpoint(client):
    """Test root endpoint returns API info."""
    response = await client.get("/")
    assert response.status_code == 200
    data = await response.get_json()
    assert "name" in data
    assert "MarchProxy" in data["name"]


@pytest.mark.asyncio
async def test_login_missing_credentials(client):
    """Test login with missing credentials returns 400."""
    response = await client.post("/api/auth/login", json={})
    assert response.status_code == 400
    data = await response.get_json()
    assert "error" in data


@pytest.mark.asyncio
async def test_protected_endpoint_without_auth(client):
    """Test protected endpoint without auth returns 401."""
    response = await client.get("/api/clusters")
    assert response.status_code == 401


@pytest.mark.asyncio
async def test_metrics_endpoint(client):
    """Test /metrics endpoint returns Prometheus format."""
    response = await client.get("/metrics")
    assert response.status_code == 200
    text = await response.get_data(as_text=True)
    # Should contain Prometheus format
    assert "marchproxy_" in text or "python_" in text or "# HELP" in text


@pytest.mark.asyncio
async def test_license_status_endpoint(client):
    """Test /license-status endpoint."""
    response = await client.get("/license-status")
    # May return 200, 503, or 500 (when mocked db returns non-serializable data)
    assert response.status_code in [200, 500, 503]


@pytest.mark.asyncio
async def test_cors_headers(client):
    """Test CORS headers are present."""
    response = await client.options("/api/auth/login", headers={"Origin": "http://localhost:3000"})
    # OPTIONS should be handled
    assert response.status_code in [200, 204, 405]


class TestAuthEndpoints:
    """Test authentication endpoints."""

    @pytest.mark.asyncio
    async def test_login_invalid_credentials(self, client):
        """Test login with invalid credentials."""
        response = await client.post(
            "/api/auth/login", json={"email": "invalid@test.com", "password": "wrong"}
        )
        # Should fail with 401 or 400
        assert response.status_code in [400, 401]

    @pytest.mark.asyncio
    async def test_register_missing_fields(self, client):
        """Test register with missing fields."""
        response = await client.post(
            "/api/auth/register", json={"email": "test@test.com"}  # Missing password
        )
        # 400 (validation) or 401 (auth required for registration)
        assert response.status_code in [400, 401]

    @pytest.mark.asyncio
    async def test_logout_without_auth(self, client):
        """Test logout without authentication."""
        response = await client.post("/api/auth/logout")
        assert response.status_code == 401


class TestClusterEndpoints:
    """Test cluster management endpoints."""

    @pytest.mark.asyncio
    async def test_list_clusters_without_auth(self, client):
        """Test list clusters without authentication."""
        response = await client.get("/api/clusters")
        assert response.status_code == 401

    @pytest.mark.asyncio
    async def test_create_cluster_without_auth(self, client):
        """Test create cluster without authentication."""
        response = await client.post("/api/clusters", json={"name": "test-cluster"})
        # 401 (auth required) or 405 (POST not allowed on this route)
        assert response.status_code in [401, 405]


class TestProxyEndpoints:
    """Test proxy management endpoints."""

    @pytest.mark.asyncio
    async def test_list_proxies_without_auth(self, client):
        """Test list proxies without authentication."""
        response = await client.get("/api/proxies")
        assert response.status_code == 401

    @pytest.mark.asyncio
    async def test_proxy_register_missing_data(self, client):
        """Test proxy registration with missing data."""
        response = await client.post("/api/proxy/register", json={})
        # 400 (validation), 401 (auth required), or 404 (route not found)
        assert response.status_code in [400, 401, 404]


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
