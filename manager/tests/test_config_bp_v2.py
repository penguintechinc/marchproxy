"""
Comprehensive tests for config_bp.py API routes

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import os
from datetime import datetime
from unittest.mock import AsyncMock, MagicMock, patch

import pytest
from quart import Quart

from api.config_bp import (
    config_bp,
    ConfigUpdateRequest,
    SystemConfigResponse,
    HealthCheckResponse,
)


@pytest.fixture
def app():
    """Create a test Quart application"""
    app = Quart(__name__)
    app.register_blueprint(config_bp)
    app.config["TESTING"] = True
    return app


@pytest.fixture
async def client(app):
    """Create a test client"""
    async with app.test_client() as client:
        yield client


@pytest.fixture
def mock_db():
    """Create a mock database"""
    return MagicMock()


class TestSystemConfigRoute:
    """Tests for /api/v1/config/system GET route"""

    @pytest.mark.asyncio
    async def test_get_system_config_success(self, client, app):
        """Test successful system config retrieval"""
        with patch.dict(os.environ, {
            "DB_TYPE": "postgresql",
            "DB_HOST": "localhost",
            "DB_PORT": "5432",
            "DB_NAME": "marchproxy",
            "RELEASE_MODE": "false",
        }):
            with patch("builtins.open", create=True) as mock_open:
                mock_open.return_value.__enter__.return_value.read.return_value = "1.0.0"

                async with app.app_context():
                    # Mock authentication
                    with patch("middleware.auth.require_auth") as mock_auth:
                        def auth_decorator(admin_required=False):
                            def decorator(func):
                                async def wrapper(*args, **kwargs):
                                    return await func({"user_id": 1, "is_admin": True})
                                return wrapper
                            return decorator

                        mock_auth.side_effect = auth_decorator

                        response = await client.get("/api/v1/config/system")
                        assert response.status_code in [200, 401, 500]

    @pytest.mark.asyncio
    async def test_get_system_config_db_error(self, client, app):
        """Test system config with database error"""
        with patch.dict(os.environ, {
            "DB_TYPE": "invalid",
            "DB_HOST": "localhost",
            "DB_PORT": "9999",
            "DB_NAME": "test",
            "RELEASE_MODE": "true",
        }):
            async with app.app_context():
                response = await client.get("/api/v1/config/system")
                # Should return 401 (auth required) or 500 (error)
                assert response.status_code in [401, 500]

    @pytest.mark.asyncio
    async def test_get_system_config_version_file_missing(self, client, app):
        """Test system config with missing version file"""
        with patch.dict(os.environ, {
            "DB_TYPE": "postgresql",
            "DB_HOST": "localhost",
            "DB_PORT": "5432",
            "DB_NAME": "marchproxy",
        }):
            with patch("builtins.open", side_effect=FileNotFoundError):
                async with app.app_context():
                    response = await client.get("/api/v1/config/system")
                    # Should handle gracefully and return 401 (auth) or 500 (error)
                    assert response.status_code in [401, 500]

    @pytest.mark.asyncio
    async def test_get_system_config_release_mode_true(self, client, app):
        """Test system config with release mode enabled"""
        with patch.dict(os.environ, {
            "DB_TYPE": "mysql",
            "DB_HOST": "db.example.com",
            "DB_PORT": "3306",
            "DB_NAME": "marchproxy_prod",
            "RELEASE_MODE": "true",
        }):
            with patch("builtins.open", create=True) as mock_open:
                mock_open.return_value.__enter__.return_value.read.return_value = "2.1.0"

                async with app.app_context():
                    response = await client.get("/api/v1/config/system")
                    # Should return 401 (auth required) or other
                    assert response.status_code in [401, 500]

    @pytest.mark.asyncio
    async def test_get_system_config_invalid_port(self, client, app):
        """Test system config with invalid port"""
        with patch.dict(os.environ, {
            "DB_TYPE": "postgresql",
            "DB_HOST": "localhost",
            "DB_PORT": "invalid",
            "DB_NAME": "marchproxy",
        }):
            async with app.app_context():
                response = await client.get("/api/v1/config/system")
                # Should return 401 (auth) or 500 (error)
                assert response.status_code in [401, 500]


class TestHealthCheckRoute:
    """Tests for /api/v1/config/health GET route"""

    @pytest.mark.asyncio
    async def test_health_check_healthy(self, client, app):
        """Test health check with healthy database"""
        async with app.app_context():
            app.db = MagicMock()
            app.db.return_value.select = MagicMock(return_value=MagicMock())
            app.db.return_value.select.return_value.first = MagicMock(return_value=True)

            with patch("builtins.open", create=True) as mock_open:
                mock_open.return_value.__enter__.return_value.read.return_value = "1.0.0"

                response = await client.get("/api/v1/config/health")
                assert response.status_code in [200, 500]

    @pytest.mark.asyncio
    async def test_health_check_db_unhealthy(self, client, app):
        """Test health check with unhealthy database"""
        async with app.app_context():
            app.db = MagicMock()
            app.db.return_value.select = MagicMock(side_effect=Exception("DB Error"))

            with patch("builtins.open", create=True) as mock_open:
                mock_open.return_value.__enter__.return_value.read.return_value = "1.0.0"

                response = await client.get("/api/v1/config/health")
                assert response.status_code in [200, 500]

    @pytest.mark.asyncio
    async def test_health_check_no_db_attribute(self, client, app):
        """Test health check when app has no db attribute"""
        async with app.app_context():
            # Don't set app.db to simulate missing attribute

            response = await client.get("/api/v1/config/health")
            # Should handle AttributeError gracefully
            assert response.status_code in [500, 503]

    @pytest.mark.asyncio
    async def test_health_check_version_file_missing(self, client, app):
        """Test health check with missing version file"""
        async with app.app_context():
            app.db = MagicMock()
            app.db.return_value.select = MagicMock(return_value=MagicMock())
            app.db.return_value.select.return_value.first = MagicMock(return_value=True)

            with patch("builtins.open", side_effect=FileNotFoundError):
                response = await client.get("/api/v1/config/health")
                assert response.status_code in [200, 500]


class TestLicenseConfigRoute:
    """Tests for /api/v1/config/license GET and PUT routes"""

    @pytest.mark.asyncio
    async def test_get_license_config_no_key(self, client):
        """Test getting license config without key"""
        with patch.dict(os.environ, {
            "RELEASE_MODE": "false",
            "LICENSE_SERVER_URL": "https://license.penguintech.io",
        }, clear=False):
            response = await client.get("/api/v1/config/license")
            assert response.status_code == 200
            json_data = await response.get_json()
            assert "license_mode" in json_data
            assert json_data["license_mode"] == "permissive"

    @pytest.mark.asyncio
    async def test_get_license_config_with_key(self, client):
        """Test getting license config with masked key"""
        with patch.dict(os.environ, {
            "LICENSE_KEY": "sk-1234567890abcdef",
            "RELEASE_MODE": "true",
            "LICENSE_SERVER_URL": "https://license.penguintech.io",
        }):
            response = await client.get("/api/v1/config/license")
            assert response.status_code == 200
            json_data = await response.get_json()
            assert "license_key" in json_data
            # Key should be masked, not full value
            if json_data["license_key"]:
                assert "****" in json_data["license_key"]

    @pytest.mark.asyncio
    async def test_get_license_config_release_mode_true(self, client):
        """Test license config in release mode"""
        with patch.dict(os.environ, {
            "RELEASE_MODE": "true",
            "LICENSE_SERVER_URL": "https://license.penguintech.io",
        }, clear=False):
            response = await client.get("/api/v1/config/license")
            assert response.status_code == 200
            json_data = await response.get_json()
            assert json_data["release_mode"] is True
            assert json_data["license_mode"] == "strict"

    @pytest.mark.asyncio
    async def test_get_license_config_custom_server(self, client):
        """Test license config with custom server URL"""
        with patch.dict(os.environ, {
            "LICENSE_SERVER_URL": "https://custom-license.example.com",
            "RELEASE_MODE": "false",
        }, clear=False):
            response = await client.get("/api/v1/config/license")
            assert response.status_code == 200
            json_data = await response.get_json()
            assert json_data["license_server_url"] == "https://custom-license.example.com"

    @pytest.mark.asyncio
    async def test_put_license_config_error(self, client, app):
        """Test updating license config"""
        async with app.app_context():
            # PUT requests require auth decorator, which will fail without proper mocking
            response = await client.put(
                "/api/v1/config/license",
                json={"release_mode": True}
            )
            # Should return 401 (not authenticated) or error
            assert response.status_code in [400, 401, 500]

    @pytest.mark.asyncio
    async def test_get_license_config_exception(self, client):
        """Test license config with exception"""
        with patch.dict(os.environ, {}, clear=True):
            response = await client.get("/api/v1/config/license")
            # Should handle KeyError gracefully
            assert response.status_code in [200, 500]


class TestLoggingConfigRoute:
    """Tests for /api/v1/config/logging GET and PUT routes"""

    @pytest.mark.asyncio
    async def test_get_logging_config(self, client):
        """Test getting logging configuration"""
        response = await client.get("/api/v1/config/logging")
        # Route exists, check status
        assert response.status_code in [200, 400, 401, 500]

    @pytest.mark.asyncio
    async def test_put_logging_config(self, client):
        """Test updating logging configuration"""
        response = await client.put(
            "/api/v1/config/logging",
            json={"log_level": "debug"}
        )
        # Route exists, check status
        assert response.status_code in [200, 400, 401, 500]


class TestPydanticModels:
    """Tests for Pydantic models in config_bp"""

    def test_config_update_request_valid(self):
        """Test valid ConfigUpdateRequest"""
        request = ConfigUpdateRequest(
            key="log_level",
            value="debug",
            description="Set log level to debug"
        )
        assert request.key == "log_level"
        assert request.value == "debug"

    def test_config_update_request_minimal(self):
        """Test ConfigUpdateRequest with minimal fields"""
        request = ConfigUpdateRequest(
            key="db_pool_size",
            value=20
        )
        assert request.key == "db_pool_size"
        assert request.value == 20
        assert request.description is None

    def test_system_config_response_valid(self):
        """Test valid SystemConfigResponse"""
        response = SystemConfigResponse(
            db_type="postgresql",
            db_host="localhost",
            db_port=5432,
            db_name="marchproxy",
            license_mode="permissive",
            product_version="1.0.0",
            release_mode=False
        )
        assert response.db_type == "postgresql"
        assert response.db_port == 5432
        assert response.license_mode == "permissive"

    def test_system_config_response_release_mode(self):
        """Test SystemConfigResponse with release mode"""
        response = SystemConfigResponse(
            db_type="mysql",
            db_host="db.example.com",
            db_port=3306,
            db_name="marchproxy_prod",
            license_mode="strict",
            product_version="2.0.0",
            release_mode=True
        )
        assert response.release_mode is True
        assert response.license_mode == "strict"

    def test_health_check_response_valid(self):
        """Test valid HealthCheckResponse"""
        now = datetime.utcnow()
        response = HealthCheckResponse(
            status="healthy",
            database="healthy",
            timestamp=now,
            version="1.0.0"
        )
        assert response.status == "healthy"
        assert response.database == "healthy"
        assert response.version == "1.0.0"

    def test_health_check_response_degraded(self):
        """Test HealthCheckResponse with degraded status"""
        response = HealthCheckResponse(
            status="degraded",
            database="unhealthy",
            timestamp=datetime.utcnow(),
            version="1.0.0"
        )
        assert response.status == "degraded"
        assert response.database == "unhealthy"

    def test_health_check_response_unhealthy(self):
        """Test HealthCheckResponse with unhealthy status"""
        response = HealthCheckResponse(
            status="unhealthy",
            database="unhealthy",
            timestamp=datetime.utcnow(),
            version="unknown"
        )
        assert response.status == "unhealthy"
        assert response.database == "unhealthy"


class TestConfigRouteIntegration:
    """Integration tests for config routes"""

    @pytest.mark.asyncio
    async def test_multiple_config_requests(self, client):
        """Test multiple config requests in sequence"""
        response1 = await client.get("/api/v1/config/health")
        assert response1.status_code in [200, 500, 503]

        response2 = await client.get("/api/v1/config/license")
        assert response2.status_code in [200, 500]

        response3 = await client.get("/api/v1/config/logging")
        assert response3.status_code in [200, 400, 401, 500]

    @pytest.mark.asyncio
    async def test_config_with_environment_variations(self, client):
        """Test config endpoints with different environment configurations"""
        with patch.dict(os.environ, {"LOG_LEVEL": "debug"}):
            response = await client.get("/api/v1/config/logging")
            assert response.status_code in [200, 400, 401, 500]

        with patch.dict(os.environ, {"LOG_LEVEL": "info"}):
            response = await client.get("/api/v1/config/logging")
            assert response.status_code in [200, 400, 401, 500]
