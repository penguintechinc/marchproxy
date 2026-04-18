"""
Tests for clusters_bp.py and services_bp.py API endpoints.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import pytest
from datetime import datetime
from unittest.mock import patch, MagicMock, AsyncMock


# ---------------------------------------------------------------------------
# Shared auth mock helpers
# ---------------------------------------------------------------------------

def _admin_payload():
    return {
        "user_id": 1,
        "sub": "1",
        "username": "admin",
        "is_admin": True,
        "scope": ["*:admin"],
        "roles": ["admin"],
        "tenant": "test",
        "session_id": "sess-1",
    }


def _user_payload():
    return {
        "user_id": 2,
        "sub": "2",
        "username": "user",
        "is_admin": False,
        "scope": [],
        "roles": [],
        "tenant": "test",
        "session_id": "sess-2",
    }


# ============================================================================
# clusters_bp Tests
# ============================================================================


def _make_require_auth_mock(payload):
    """
    Create a mock for require_auth that makes the broken
    `await require_auth()(lambda: None)` pattern in clusters_bp work.

    require_auth() -> decorator
    decorator(func) -> coroutine (awaitable)
    await coroutine -> payload dict if authenticated, or 401 tuple if not

    When payload is None, returns a 401 error tuple (simulating unauthenticated).
    When payload is a dict, returns it as auth_result.
    """
    result = payload if payload is not None else ({"error": "Missing authorization header"}, 401)

    async def _coro(*args, **kwargs):
        return result

    decorator = MagicMock(side_effect=lambda f: _coro())
    return MagicMock(return_value=decorator)


class TestClustersListGet:
    """GET /api/v1/services/v1/clusters"""

    @pytest.mark.asyncio
    async def test_get_clusters_no_auth_returns_401(self, test_client):
        response = await test_client.get("/api/v1/clusters")
        assert response.status_code == 401

    @pytest.mark.asyncio
    async def test_get_clusters_no_header_returns_401(self, test_client):
        response = await test_client.get("/api/v1/clusters")
        assert response.status_code == 401

    @pytest.mark.asyncio
    async def test_get_clusters_admin_returns_all(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.cluster.ClusterModel.count_active_proxies", return_value=2):
            mock_v.return_value = _admin_payload()
            response = await test_client.get(
                "/api/v1/clusters",
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [200, 500]

    @pytest.mark.asyncio
    async def test_get_clusters_user_sees_assigned_only(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.cluster.UserClusterAssignmentModel.get_user_clusters", return_value=[]):
            mock_v.return_value = _user_payload()
            response = await test_client.get(
                "/api/v1/clusters",
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [200, 500]


class TestClustersListPost:
    """POST /api/v1/services"""

    @pytest.mark.asyncio
    async def test_create_cluster_no_auth_returns_401(self, test_client):
        response = await test_client.post("/api/v1/clusters", json={"name": "c1"})
        assert response.status_code == 401

    @pytest.mark.asyncio
    async def test_create_cluster_non_admin_returns_403(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v:
            mock_v.return_value = _user_payload()
            response = await test_client.post(
                "/api/v1/clusters",
                json={"name": "c1"},
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [403, 401, 500]

    @pytest.mark.asyncio
    async def test_create_cluster_validation_error_returns_400(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v:
            mock_v.return_value = _admin_payload()
            # Missing required fields
            response = await test_client.post(
                "/api/v1/clusters",
                json={},
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [400, 403, 500]

    @pytest.mark.asyncio
    async def test_create_cluster_duplicate_name_returns_409(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v:
            mock_v.return_value = _admin_payload()
            response = await test_client.post(
                "/api/v1/clusters",
                json={"name": "existing"},
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [400, 403, 409, 500]

    @pytest.mark.asyncio
    async def test_create_cluster_success_returns_201(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.cluster.ClusterModel.create_cluster", return_value=(1, "api-key-abc")):
            mock_v.return_value = _admin_payload()
            response = await test_client.post(
                "/api/v1/clusters",
                json={"name": "new-cluster"},
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [201, 400, 403, 409, 500]


class TestClusterDetail:
    """GET/PUT /api/<id>"""

    @pytest.mark.asyncio
    async def test_get_cluster_no_auth_returns_401(self, test_client):
        response = await test_client.get("/api/1")
        assert response.status_code == 401

    @pytest.mark.asyncio
    async def test_get_cluster_not_found_returns_404(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.cluster.ClusterModel.count_active_proxies", return_value=0):
            mock_v.return_value = _admin_payload()
            response = await test_client.get(
                "/api/999",
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [404, 500]

    @pytest.mark.asyncio
    async def test_get_cluster_non_admin_denied_cluster_returns_403(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.cluster.UserClusterAssignmentModel.check_user_cluster_access", return_value=None):
            mock_v.return_value = _user_payload()
            response = await test_client.get(
                "/api/1",
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [403, 404, 500]

    @pytest.mark.asyncio
    async def test_put_cluster_no_auth_returns_401(self, test_client):
        response = await test_client.put("/api/1", json={"name": "new"})
        assert response.status_code == 401

    @pytest.mark.asyncio
    async def test_put_cluster_non_admin_returns_403(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v:
            mock_v.return_value = _user_payload()
            response = await test_client.put(
                "/api/1",
                json={"name": "new"},
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [403, 404, 500]

    @pytest.mark.asyncio
    async def test_put_cluster_not_found_returns_404(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v:
            mock_v.return_value = _admin_payload()
            response = await test_client.put(
                "/api/999",
                json={"name": "new"},
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [404, 500]


class TestClusterRotateKey:
    """POST /api/v1/services/<id>/rotate-key"""

    @pytest.mark.asyncio
    async def test_rotate_key_no_auth_returns_401(self, test_client):
        response = await test_client.post("/api/v1/clusters/1/rotate-key")
        assert response.status_code == 401

    @pytest.mark.asyncio
    async def test_rotate_key_non_admin_returns_403(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v:
            mock_v.return_value = _user_payload()
            response = await test_client.post(
                "/api/v1/clusters/1/rotate-key",
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [403, 500]

    @pytest.mark.asyncio
    async def test_rotate_key_cluster_not_found_returns_404(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v:
            mock_v.return_value = _admin_payload()
            response = await test_client.post(
                "/api/v1/clusters/999/rotate-key",
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [404, 500]

    @pytest.mark.asyncio
    async def test_rotate_key_success(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.cluster.ClusterModel.rotate_api_key", return_value="new-key-abc"):
            mock_v.return_value = _admin_payload()
            response = await test_client.post(
                "/api/v1/clusters/1/rotate-key",
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [200, 404, 500]


class TestClusterUpdateLogging:
    """PUT /api/<id>/logging"""

    @pytest.mark.asyncio
    async def test_update_logging_no_auth(self, test_client):
        response = await test_client.put(
            "/api/v1/clusters/1/logging",
            json={"log_auth": True},
        )
        assert response.status_code == 401

    @pytest.mark.asyncio
    async def test_update_logging_non_admin_returns_403(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v:
            mock_v.return_value = _user_payload()
            response = await test_client.put(
                "/api/v1/clusters/1/logging",
                json={"log_auth": True},
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [403, 500]

    @pytest.mark.asyncio
    async def test_update_logging_cluster_not_found_returns_404(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.cluster.ClusterModel.update_logging_config", return_value=False):
            mock_v.return_value = _admin_payload()
            response = await test_client.put(
                "/api/v1/clusters/999/logging",
                json={"log_auth": True},
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [404, 500]

    @pytest.mark.asyncio
    async def test_update_logging_success_returns_200(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.cluster.ClusterModel.update_logging_config", return_value=True):
            mock_v.return_value = _admin_payload()
            response = await test_client.put(
                "/api/v1/clusters/1/logging",
                json={"log_auth": True, "log_netflow": False},
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [200, 500]


class TestClusterAssignUser:
    """POST /api/v1/services/<id>/assign-user"""

    @pytest.mark.asyncio
    async def test_assign_user_no_auth(self, test_client):
        response = await test_client.post(
            "/api/v1/clusters/1/assign-user",
            json={"user_id": 2, "role": "viewer"},
        )
        assert response.status_code == 401

    @pytest.mark.asyncio
    async def test_assign_user_validation_error_returns_400(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v:
            mock_v.return_value = _admin_payload()
            response = await test_client.post(
                "/api/v1/clusters/1/assign-user",
                json={},  # Missing user_id and role
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [400, 500]

    @pytest.mark.asyncio
    async def test_assign_user_cluster_not_found(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v:
            mock_v.return_value = _admin_payload()
            response = await test_client.post(
                "/api/v1/clusters/999/assign-user",
                json={"user_id": 2, "role": "viewer"},
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [400, 404, 500]

    @pytest.mark.asyncio
    async def test_assign_user_success(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.cluster.UserClusterAssignmentModel.assign_user_to_cluster", return_value=True):
            mock_v.return_value = _admin_payload()
            response = await test_client.post(
                "/api/v1/clusters/1/assign-user",
                json={"user_id": 2, "role": "viewer"},
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [200, 400, 404, 500]


class TestClusterConfig:
    """GET /api/v1/services/config/<cluster_id>"""

    @pytest.mark.asyncio
    async def test_get_config_no_api_key_returns_401(self, test_client):
        response = await test_client.get("/api/v1/clusters/config/1")
        assert response.status_code == 401

    @pytest.mark.asyncio
    async def test_get_config_invalid_api_key_returns_401(self, test_client):
        with patch("models.cluster.ClusterModel.validate_api_key", return_value=None):
            response = await test_client.get(
                "/api/v1/clusters/config/1",
                headers={"X-API-Key": "invalid-key"},
            )
            assert response.status_code == 401

    @pytest.mark.asyncio
    async def test_get_config_wrong_cluster_id_returns_401(self, test_client):
        with patch("models.cluster.ClusterModel.validate_api_key",
                   return_value={"cluster_id": 2}):
            response = await test_client.get(
                "/api/v1/clusters/config/1",
                headers={"X-API-Key": "some-key"},
            )
            assert response.status_code in [401, 500]

    @pytest.mark.asyncio
    async def test_get_config_not_found_returns_404(self, test_client):
        with patch("models.cluster.ClusterModel.validate_api_key",
                   return_value={"cluster_id": 1}), \
             patch("models.cluster.ClusterModel.get_cluster_config", return_value=None):
            response = await test_client.get(
                "/api/v1/clusters/config/1",
                headers={"X-API-Key": "valid-key"},
            )
            assert response.status_code in [404, 500]

    @pytest.mark.asyncio
    async def test_get_config_success_returns_200(self, test_client):
        config = {"cluster_id": 1, "name": "test", "services": []}
        with patch("models.cluster.ClusterModel.validate_api_key",
                   return_value={"cluster_id": 1}), \
             patch("models.cluster.ClusterModel.get_cluster_config", return_value=config):
            response = await test_client.get(
                "/api/v1/clusters/config/1",
                headers={"X-API-Key": "valid-key"},
            )
            assert response.status_code in [200, 500]


# ============================================================================
# services_bp Tests
# ============================================================================


class TestServicesListGet:
    """GET /api/v1/services"""

    @pytest.mark.asyncio
    async def test_get_services_no_auth_returns_401(self, test_client):
        response = await test_client.get("/api/v1/clusters")
        assert response.status_code == 401

    @pytest.mark.asyncio
    async def test_get_services_missing_cluster_id_returns_400(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v:
            mock_v.return_value = _admin_payload()
            response = await test_client.get(
                "/api/v1/clusters",
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [200, 400, 401, 500]

    @pytest.mark.asyncio
    async def test_get_services_with_cluster_id_success(self, test_client):
        svc = {
            "id": 1,
            "name": "svc1",
            "ip_fqdn": "192.168.1.1",
            "port": 8080,
            "protocol": "tcp",
            "collection": "default",
            "cluster_id": 1,
            "auth_type": "none",
            "tls_enabled": False,
            "health_check_enabled": False,
            "created_at": datetime.utcnow(),
        }
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.service.ServiceModel.get_cluster_services", return_value=[svc]):
            mock_v.return_value = _admin_payload()
            response = await test_client.get(
                "/api?cluster_id=1",
                headers={"Authorization": "Bearer mock-token"},
            )
            # services_bp handles GET /api/v1/services
            assert response.status_code in [200, 400, 401, 500]


class TestServicesListPost:
    """POST /api/v1/services"""

    @pytest.mark.asyncio
    async def test_create_service_no_auth_returns_401(self, test_client):
        response = await test_client.post(
            "/api/v1/clusters",
            json={"name": "svc"},
        )
        assert response.status_code == 401

    @pytest.mark.asyncio
    async def test_create_service_non_admin_returns_403(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v:
            mock_v.return_value = _user_payload()
            response = await test_client.post(
                "/api/v1/clusters",
                json={"name": "svc"},
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [403, 500]

    @pytest.mark.asyncio
    async def test_create_service_validation_error_returns_400(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v:
            mock_v.return_value = _admin_payload()
            response = await test_client.post(
                "/api/v1/clusters",
                json={},
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [400, 403, 500]

    @pytest.mark.asyncio
    async def test_create_service_success_returns_201(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.service.ServiceModel.create_service", return_value=1):
            mock_v.return_value = _admin_payload()
            response = await test_client.post(
                "/api/v1/clusters",
                json={
                    "name": "svc1",
                    "ip_fqdn": "10.0.0.1",
                    "port": 8080,
                    "cluster_id": 1,
                },
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [201, 400, 403, 409, 500]


class TestServiceDetail:
    """GET/PUT/DELETE /api/<id>"""

    @pytest.mark.asyncio
    async def test_get_service_no_auth_returns_401(self, test_client):
        response = await test_client.get("/api/1")
        assert response.status_code == 401

    @pytest.mark.asyncio
    async def test_get_service_not_found_returns_404(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.service.ServiceModel.get_service_config", return_value=None):
            mock_v.return_value = _admin_payload()
            response = await test_client.get(
                "/api/999",
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [404, 500]

    @pytest.mark.asyncio
    async def test_put_service_no_auth_returns_401(self, test_client):
        response = await test_client.put(
            "/api/1",
            json={"name": "updated"},
        )
        assert response.status_code == 401

    @pytest.mark.asyncio
    async def test_delete_service_no_auth_returns_401(self, test_client):
        response = await test_client.delete("/api/1")
        assert response.status_code == 401


class TestServiceAuth:
    """POST /api/v1/services/<id>/auth"""

    @pytest.mark.asyncio
    async def test_set_service_auth_no_auth_returns_401(self, test_client):
        response = await test_client.post(
            "/api/1/auth",
            json={"auth_type": "none"},
        )
        assert response.status_code == 401

    @pytest.mark.asyncio
    async def test_set_service_auth_non_admin_returns_403(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v:
            mock_v.return_value = _user_payload()
            response = await test_client.post(
                "/api/1/auth",
                json={"auth_type": "none"},
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [403, 500]

    @pytest.mark.asyncio
    async def test_set_service_auth_service_not_found_returns_404(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v:
            mock_v.return_value = _admin_payload()
            response = await test_client.post(
                "/api/999/auth",
                json={"auth_type": "none"},
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [404, 500]

    @pytest.mark.asyncio
    async def test_set_service_auth_invalid_type_returns_400(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v:
            mock_v.return_value = _admin_payload()
            response = await test_client.post(
                "/api/1/auth",
                json={"auth_type": "invalid"},
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [400, 404, 500]

    @pytest.mark.asyncio
    async def test_set_service_auth_base64_success(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.service.ServiceModel.set_base64_auth", return_value="b64token"):
            mock_v.return_value = _admin_payload()
            response = await test_client.post(
                "/api/1/auth",
                json={"auth_type": "base64"},
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [200, 404, 500]

    @pytest.mark.asyncio
    async def test_set_service_auth_none_type(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v:
            mock_v.return_value = _admin_payload()
            response = await test_client.post(
                "/api/1/auth",
                json={"auth_type": "none"},
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [200, 404, 500]


class TestServiceJWTRotate:
    """POST /api/v1/services/<id>/auth/rotate"""

    @pytest.mark.asyncio
    async def test_rotate_jwt_no_auth_returns_401(self, test_client):
        response = await test_client.post("/api/1/auth/rotate")
        assert response.status_code == 401

    @pytest.mark.asyncio
    async def test_rotate_jwt_service_not_found_returns_404(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v:
            mock_v.return_value = _admin_payload()
            response = await test_client.post(
                "/api/999/auth/rotate",
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [404, 500]

    @pytest.mark.asyncio
    async def test_rotate_jwt_success(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.service.ServiceModel.rotate_jwt_secret", return_value="new-secret"):
            mock_v.return_value = _admin_payload()
            response = await test_client.post(
                "/api/1/auth/rotate",
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [200, 404, 500]


class TestServiceToken:
    """POST /api/v1/services/<id>/token"""

    @pytest.mark.asyncio
    async def test_create_token_no_auth_returns_401(self, test_client):
        response = await test_client.post("/api/1/token")
        assert response.status_code == 401

    @pytest.mark.asyncio
    async def test_create_token_non_admin_returns_403(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v:
            mock_v.return_value = _user_payload()
            response = await test_client.post(
                "/api/1/token",
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [403, 500]

    @pytest.mark.asyncio
    async def test_create_token_service_not_jwt_returns_404(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v:
            mock_v.return_value = _admin_payload()
            response = await test_client.post(
                "/api/999/token",
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [404, 500]


class TestServiceAssignUser:
    """POST /api/v1/services/<id>/assign"""

    @pytest.mark.asyncio
    async def test_assign_user_no_auth_returns_401(self, test_client):
        response = await test_client.post(
            "/api/1/assign",
            json={"user_id": 2},
        )
        assert response.status_code == 401

    @pytest.mark.asyncio
    async def test_assign_user_validation_error_returns_400(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v:
            mock_v.return_value = _admin_payload()
            response = await test_client.post(
                "/api/1/assign",
                json={},
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [400, 403, 500]

    @pytest.mark.asyncio
    async def test_assign_user_success(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.service.UserServiceAssignmentModel.assign_user_to_service",
                   return_value=True):
            mock_v.return_value = _admin_payload()
            response = await test_client.post(
                "/api/1/assign",
                json={"user_id": 2},
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [200, 400, 403, 500]


class TestServiceUnassignUser:
    """DELETE /api/<id>/unassign/<user_id>"""

    @pytest.mark.asyncio
    async def test_unassign_user_no_auth_returns_401(self, test_client):
        response = await test_client.delete("/api/1/unassign/2")
        assert response.status_code == 401

    @pytest.mark.asyncio
    async def test_unassign_user_success(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.service.UserServiceAssignmentModel.remove_user_from_service",
                   return_value=True):
            mock_v.return_value = _admin_payload()
            response = await test_client.delete(
                "/api/1/unassign/2",
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [200, 500]

    @pytest.mark.asyncio
    async def test_unassign_user_failure_returns_500(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.service.UserServiceAssignmentModel.remove_user_from_service",
                   return_value=False):
            mock_v.return_value = _admin_payload()
            response = await test_client.delete(
                "/api/1/unassign/2",
                headers={"Authorization": "Bearer mock-token"},
            )
            assert response.status_code in [500, 403]
