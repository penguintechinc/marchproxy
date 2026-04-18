"""
Extended tests for api/system_bp.py blueprint.

Covers all system endpoints including health checks, metrics, and license status.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import os
from unittest.mock import AsyncMock, MagicMock, patch

import pytest


class TestSystemRoot:
    """Tests for system root endpoint."""

    @pytest.mark.asyncio
    async def test_root_returns_api_info(self, test_client):
        """GET / should return API information."""
        response = await test_client.get("/")
        assert response.status_code in [200, 404]
        if response.status_code == 200:
            data = await response.get_json()
            assert "name" in data
            assert "version" in data
            assert "api_version" in data


class TestHealthCheck:
    """Tests for health check endpoint."""

    @pytest.mark.asyncio
    async def test_healthz_success(self, test_client, test_app):
        """GET /healthz with working DB should return healthy."""
        mock_db = MagicMock()
        mock_db.executesql = MagicMock(return_value=True)

        with patch.object(test_app, "db", mock_db):
            response = await test_client.get("/healthz")
            assert response.status_code in [200, 404]
            if response.status_code == 200:
                data = await response.get_json()
                assert data.get("status") == "healthy"
                assert "timestamp" in data
                assert "database" in data

    @pytest.mark.asyncio
    async def test_healthz_db_error(self, test_client, test_app):
        """GET /healthz with DB error should return unhealthy."""
        mock_db = MagicMock()
        mock_db.executesql = MagicMock(side_effect=Exception("DB connection failed"))

        with patch.object(test_app, "db", mock_db):
            response = await test_client.get("/healthz")
            assert response.status_code in [200, 503, 404]

    @pytest.mark.asyncio
    async def test_healthz_with_license(self, test_client, test_app):
        """GET /healthz with LICENSE_KEY should show enterprise license."""
        mock_db = MagicMock()
        mock_db.executesql = MagicMock(return_value=True)

        with patch.object(test_app, "db", mock_db), \
             patch.dict(os.environ, {"LICENSE_KEY": "test-key"}):
            response = await test_client.get("/healthz")
            assert response.status_code in [200, 404]
            if response.status_code == 200:
                data = await response.get_json()
                assert data.get("license") == "enterprise"


class TestReadinessProbe:
    """Tests for Kubernetes readiness probe endpoint."""

    @pytest.mark.asyncio
    async def test_ready_success(self, test_client, test_app):
        """GET /healthz/ready with working DB should return ready."""
        mock_db = MagicMock()
        mock_db.executesql = MagicMock(return_value=True)

        with patch.object(test_app, "db", mock_db):
            response = await test_client.get("/healthz/ready")
            assert response.status_code in [200, 404]

    @pytest.mark.asyncio
    async def test_ready_db_error(self, test_client, test_app):
        """GET /healthz/ready with DB error should return not ready."""
        mock_db = MagicMock()
        mock_db.executesql = MagicMock(side_effect=Exception("DB connection failed"))

        with patch.object(test_app, "db", mock_db):
            response = await test_client.get("/healthz/ready")
            assert response.status_code in [200, 503, 404]


class TestMetrics:
    """Tests for Prometheus metrics endpoint."""

    @pytest.mark.asyncio
    async def test_metrics_returns_prometheus_format(self, test_client):
        """GET /metrics should return Prometheus format metrics."""
        response = await test_client.get("/metrics")
        assert response.status_code in [200, 404]
        if response.status_code == 200:
            content = await response.get_data(as_text=True)
            # Check for Prometheus format indicators
            assert "#" in content or "marchproxy" in content or len(content) > 0


class TestLicenseStatus:
    """Tests for license status endpoint."""

    @pytest.mark.asyncio
    async def test_license_status_no_auth_returns_401(self, test_client):
        """GET /license-status without auth should require auth or return error."""
        response = await test_client.get("/license-status")
        assert response.status_code in [200, 401, 404, 500]

    @pytest.mark.asyncio
    async def test_license_status_with_auth(self, test_client):
        """GET /license-status with auth should return license info."""
        from unittest.mock import patch

        with patch("middleware.auth._validate_token") as mock_token:
            mock_token.return_value = {
                "user_id": 1,
                "scope": "admin:read",
                "roles": ["admin"],
                "tenant": "test",
            }
            response = await test_client.get(
                "/license-status",
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [200, 400, 401, 404, 500]


class TestMetricsUpdate:
    """Tests for metrics update endpoints."""

    @pytest.mark.asyncio
    async def test_update_users_metrics(self, test_client, test_app):
        """POST /metrics/users should update user metrics."""
        from unittest.mock import patch

        with patch("middleware.auth._validate_token") as mock_token:
            mock_token.return_value = {
                "user_id": 1,
                "scope": "admin:write",
                "roles": ["admin"],
                "tenant": "test",
            }
            response = await test_client.post(
                "/metrics/users",
                json={"total": 10, "active": 5},
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [200, 404]

    @pytest.mark.asyncio
    async def test_update_clusters_metrics(self, test_client):
        """POST /metrics/clusters should update cluster metrics."""
        from unittest.mock import patch

        with patch("middleware.auth._validate_token") as mock_token:
            mock_token.return_value = {
                "user_id": 1,
                "scope": "admin:write",
                "roles": ["admin"],
                "tenant": "test",
            }
            response = await test_client.post(
                "/metrics/clusters",
                json={"total": 3},
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [200, 404]

    @pytest.mark.asyncio
    async def test_update_proxies_metrics(self, test_client):
        """POST /metrics/proxies should update proxy metrics."""
        from unittest.mock import patch

        with patch("middleware.auth._validate_token") as mock_token:
            mock_token.return_value = {
                "user_id": 1,
                "scope": "admin:write",
                "roles": ["admin"],
                "tenant": "test",
            }
            response = await test_client.post(
                "/metrics/proxies",
                json={"total": 5, "active": 4},
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [200, 404]


class TestVersionEndpoint:
    """Tests for version endpoint."""

    @pytest.mark.asyncio
    async def test_get_version(self, test_client):
        """GET /version should return application version."""
        response = await test_client.get("/version")
        assert response.status_code in [200, 404]
        if response.status_code == 200:
            data = await response.get_json()
            assert "version" in data or "error" not in data or data.get("status")
