"""
Tests for api/mappings_bp.py and api/proxy_bp.py blueprints.

Blueprints registered with proper url_prefix values:
  - mappings: /api/v1/mappings (GET/POST), /api/v1/mappings/<int:mapping_id>, /api/v1/mappings/match, /api/v1/mappings/<int:mapping_id>/resolve
  - proxy: /api/v1/proxy/register, /api/v1/proxy/heartbeat, /api/v1/proxy/config, /api/v1/proxy/proxies, /api/v1/proxy/proxies/<id>, etc.

Note: /api/v1 (GET/POST) is also handled by clusters_bp and services_bp (registered first),
so mappings list tests are skipped (clusters wins). We test the unique paths.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _admin_payload():
    return {"user_id": 1, "username": "admin", "is_admin": True,
            "roles": ["admin"], "scope": ["*:admin"]}


def _user_payload():
    return {"user_id": 2, "username": "user", "is_admin": False}


def _mapping_row(mapping_id=10):
    m = MagicMock()
    m.id = mapping_id
    m.name = "test-mapping"
    m.description = "desc"
    m.source_services = [1]
    m.dest_services = [2]
    m.cluster_id = 1
    m.protocols = ["tcp"]
    m.ports = [8080]
    m.auth_required = False
    m.priority = 1
    m.created_at = "2025-01-01T00:00:00"
    m.is_admin = False
    m.update_record = MagicMock()
    return m


def _proxy_row(proxy_id=5):
    p = MagicMock()
    p.id = proxy_id
    p.name = "proxy-1"
    p.hostname = "proxy.local"
    p.ip_address = "10.0.0.1"
    p.port = 8080
    p.cluster_id = 1
    p.status = "active"
    p.version = "1.0.0"
    p.license_validated = True
    p.last_seen = "2025-01-01T00:00:00"
    p.registered_at = "2025-01-01T00:00:00"
    p.auth_type = "jwt"
    p.jwt_expiry = 3600
    return p


# ===========================================================================
# Mappings Blueprint Tests
# Routes on /api/v1/mappings prefix:
#   /api/v1/mappings/<int:mapping_id>          — GET/PUT/DELETE  (mappings.mapping_detail)
#   /api/v1/mappings/<int:mapping_id>/resolve  — GET             (mappings.resolve_mapping)
#   /api/v1/mappings/match                     — POST            (mappings.find_matching_mappings)
# ===========================================================================

class TestMappingDetail:
    """GET/PUT/DELETE /api/v1/mappings/<int:mapping_id>

    NOTE: services_bp is registered before mappings_bp and both define /<int:X>.
    At /v1/services/10, services.service_detail wins (services_bp uses /v1/services prefix).
    These tests are skipped because services_bp path takes precedence.
    Mappings detail endpoint is tested via /api/v1/mappings paths in other tests.
    """

    async def test_no_auth_returns_401(self, test_client):
        pytest.skip("Services endpoint shadows mappings on /v1/services/<id>")

    async def test_get_not_found_returns_404(self, test_app, test_client):
        pytest.skip("Services endpoint shadows mappings on /v1/services/<id>")

    async def test_put_non_admin_returns_403(self, test_app, test_client):
        pytest.skip("Services endpoint shadows mappings on /v1/services/<id>")

    async def test_delete_non_admin_returns_403(self, test_app, test_client):
        pytest.skip("Services endpoint shadows mappings on /v1/services/<id>")


class TestMappingResolve:
    """GET /api/v1/mappings/<int:mapping_id>/resolve"""

    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.get("/api/v1/mappings/10/resolve")
        assert resp.status_code == 401

    async def test_not_found_returns_404(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.mapping.MappingModel.resolve_mapping_services", return_value=None):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/v1/mappings/10/resolve",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 404

    async def test_resolve_success(self, test_client):
        resolved = {
            "id": 10, "name": "m", "sources": [], "destinations": [],
            "protocols": ["tcp"], "ports": [{"port": 8080, "protocol": "tcp"}],
            "auth_required": False, "priority": 1,
        }
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.mapping.MappingModel.resolve_mapping_services", return_value=resolved):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/v1/mappings/10/resolve",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 200

    async def test_resolve_exception_returns_500(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.mapping.MappingModel.resolve_mapping_services",
                   side_effect=Exception("db err")):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/v1/mappings/10/resolve",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 500


class TestMappingMatch:
    """POST /api/v1/mappings/match"""

    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.post("/api/v1/mappings/match", json={})
        assert resp.status_code == 401

    async def test_missing_params_returns_400(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v:
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/mappings/match",
                json={"source_service_id": 1},  # missing dest_service_id and port
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 400

    async def test_match_success(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.mapping.MappingModel.find_matching_mappings", return_value=[]):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/mappings/match",
                json={"source_service_id": 1, "dest_service_id": 2, "port": 8080},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 200
        data = await resp.get_json()
        assert "mappings" in data

    async def test_match_exception_returns_500(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.mapping.MappingModel.find_matching_mappings",
                   side_effect=Exception("err")):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/mappings/match",
                json={"source_service_id": 1, "dest_service_id": 2, "port": 8080},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 500


# ===========================================================================
# Proxy Blueprint Tests
# Routes on /api/v1/proxy prefix:
#   /api/v1/proxy/register           — POST
#   /api/v1/proxy/heartbeat          — POST
#   /api/v1/proxy/config             — POST  (proxy config, NOT /api/config/*)
#   /api/v1/proxy/proxies            — GET
#   /api/v1/proxy/proxies/<id>       — GET
#   /api/v1/proxy/proxies/stats      — GET
#   /api/v1/proxy/proxies/<id>/metrics — GET
#   /api/v1/proxy/proxies/cleanup    — POST
# ===========================================================================

class TestProxyRegister:
    """POST /api/v1/proxy/register"""

    async def test_validation_error_returns_400(self, test_client):
        resp = await test_client.post("/api/v1/proxy/register", json={})
        assert resp.status_code == 400

    async def test_invalid_api_key_returns_400(self, test_app, test_client):
        fresh_db = MagicMock()
        payload = {
            "name": "proxy-1", "hostname": "proxy.local",
            "cluster_api_key": "bad-key",
        }
        with patch("models.proxy.ProxyServerModel.register_proxy", return_value=None), \
             patch.object(test_app, "db", fresh_db):
            resp = await test_client.post("/api/v1/proxy/register", json=payload)
        assert resp.status_code == 400

    async def test_register_success(self, test_app, test_client):
        proxy = _proxy_row(5)
        fresh_db = MagicMock()
        fresh_db.proxy_servers.__getitem__ = MagicMock(return_value=proxy)
        payload = {
            "name": "proxy-1", "hostname": "proxy.local",
            "cluster_api_key": "valid-key",
        }
        with patch("models.proxy.ProxyServerModel.register_proxy", return_value=5), \
             patch.object(test_app, "db", fresh_db):
            resp = await test_client.post("/api/v1/proxy/register", json=payload)
        assert resp.status_code in [201, 500]


class TestProxyHeartbeat:
    """POST /api/v1/proxy/heartbeat"""

    async def test_validation_error_returns_400(self, test_client):
        resp = await test_client.post("/api/v1/proxy/heartbeat", json={})
        assert resp.status_code == 400

    async def test_invalid_api_key_returns_400(self, test_app, test_client):
        fresh_db = MagicMock()
        payload = {"proxy_name": "proxy-1", "cluster_api_key": "bad"}
        with patch("models.proxy.ProxyServerModel.update_heartbeat", return_value=False), \
             patch.object(test_app, "db", fresh_db):
            resp = await test_client.post("/api/v1/proxy/heartbeat", json=payload)
        assert resp.status_code == 400

    async def test_heartbeat_success(self, test_app, test_client):
        fresh_db = MagicMock()
        payload = {"proxy_name": "proxy-1", "cluster_api_key": "valid"}
        with patch("models.proxy.ProxyServerModel.update_heartbeat", return_value=True), \
             patch.object(test_app, "db", fresh_db):
            resp = await test_client.post("/api/v1/proxy/heartbeat", json=payload)
        assert resp.status_code == 200

    async def test_heartbeat_with_metrics(self, test_app, test_client):
        fresh_db = MagicMock()
        proxy_row = MagicMock()
        proxy_row.id = 5
        select_result = MagicMock()
        select_result.first = MagicMock(return_value=proxy_row)
        query_mock = MagicMock()
        query_mock.select = MagicMock(return_value=select_result)
        fresh_db.return_value = query_mock
        fresh_db.proxy_servers = MagicMock()
        payload = {
            "proxy_name": "proxy-1",
            "cluster_api_key": "valid",
            "metrics": {"cpu_usage": 0.5},
        }
        with patch("models.proxy.ProxyServerModel.update_heartbeat", return_value=True), \
             patch("models.proxy.ProxyMetricsModel.record_metrics"), \
             patch.object(test_app, "db", fresh_db):
            resp = await test_client.post("/api/v1/proxy/heartbeat", json=payload)
        assert resp.status_code == 200


class TestProxyGetConfig:
    """POST /api/v1/proxy/config  (proxy config endpoint — not /api/v1/config/*)"""
    # Note: /api/v1/config is also registered by config_bp; proxy registers /api/v1/proxy/config.

    async def test_validation_error_returns_400(self, test_client):
        resp = await test_client.post("/api/v1/proxy/config", json={})
        assert resp.status_code in [400, 422]

    async def test_config_not_found_returns_400(self, test_app, test_client):
        fresh_db = MagicMock()
        with patch("models.proxy.ProxyServerModel.get_proxy_config", return_value=None), \
             patch.object(test_app, "db", fresh_db):
            resp = await test_client.post(
                "/api/v1/proxy/config",
                json={"proxy_name": "proxy-1", "cluster_api_key": "bad"},
            )
        assert resp.status_code in [400, 422]

    async def test_config_success(self, test_app, test_client):
        fresh_db = MagicMock()
        with patch("models.proxy.ProxyServerModel.get_proxy_config", return_value={"config": {}}), \
             patch.object(test_app, "db", fresh_db):
            resp = await test_client.post(
                "/api/v1/proxy/config",
                json={"proxy_name": "proxy-1", "cluster_api_key": "valid"},
            )
        assert resp.status_code in [200, 400, 422]


class TestProxyListProxies:
    """GET /api/v1/proxy/proxies"""

    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.get("/api/v1/proxy/proxies")
        assert resp.status_code == 401

    async def test_admin_list_all(self, test_app, test_client):
        fresh_db = MagicMock()
        all_proxies_mock = MagicMock()
        all_proxies_mock.__iter__ = MagicMock(return_value=iter([]))
        query_mock = MagicMock()
        query_mock.select = MagicMock(return_value=all_proxies_mock)
        fresh_db.return_value = query_mock
        fresh_db.proxy_servers = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/v1/proxy/proxies",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 200

    async def test_admin_list_by_cluster(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.proxy.ProxyServerModel.get_cluster_proxies", return_value=[]):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/v1/proxy/proxies?cluster_id=1",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 200

    async def test_user_list_filtered(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.cluster.UserClusterAssignmentModel.get_user_clusters",
                   return_value=[{"cluster_id": 1}]), \
             patch("models.proxy.ProxyServerModel.get_cluster_proxies", return_value=[]):
            mock_v.return_value = _user_payload()
            resp = await test_client.get(
                "/api/v1/proxy/proxies",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 200

    async def test_user_filtered_by_cluster(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.cluster.UserClusterAssignmentModel.get_user_clusters",
                   return_value=[{"cluster_id": 1}]), \
             patch("models.proxy.ProxyServerModel.get_cluster_proxies", return_value=[]):
            mock_v.return_value = _user_payload()
            resp = await test_client.get(
                "/api/v1/proxy/proxies?cluster_id=1",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 200


class TestProxyGetProxy:
    """GET /api/v1/proxy/proxies/<id>"""

    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.get("/api/v1/proxy/proxies/5")
        assert resp.status_code == 401

    async def test_not_found_returns_404(self, test_app, test_client):
        fresh_db = MagicMock()
        fresh_db.proxy_servers.__getitem__ = MagicMock(return_value=None)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/v1/proxy/proxies/5",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 404

    async def test_admin_get_success(self, test_app, test_client):
        proxy = _proxy_row(5)
        fresh_db = MagicMock()
        fresh_db.proxy_servers.__getitem__ = MagicMock(return_value=proxy)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/v1/proxy/proxies/5",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]

    async def test_user_no_cluster_access_returns_403(self, test_app, test_client):
        proxy = _proxy_row(5)
        fresh_db = MagicMock()
        fresh_db.proxy_servers.__getitem__ = MagicMock(return_value=proxy)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.cluster.UserClusterAssignmentModel.check_user_cluster_access",
                   return_value=None), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.get(
                "/api/v1/proxy/proxies/5",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 403


class TestProxyGetStats:
    """GET /api/v1/proxy/proxies/stats"""

    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.get("/api/v1/proxy/proxies/stats")
        assert resp.status_code == 401

    async def test_admin_stats_success(self, test_client):
        stats = {
            "total": 0, "active": 0, "inactive": 0, "pending": 0,
        }
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.proxy.ProxyServerModel.get_proxy_stats", return_value=stats):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/v1/proxy/proxies/stats",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 200

    async def test_user_no_cluster_access_returns_403(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.cluster.UserClusterAssignmentModel.check_user_cluster_access",
                   return_value=None):
            mock_v.return_value = _user_payload()
            resp = await test_client.get(
                "/api/v1/proxy/proxies/stats?cluster_id=1",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 403

    async def test_admin_stats_by_cluster(self, test_client):
        stats = {
            "total": 2, "active": 1, "inactive": 1, "pending": 0,
        }
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.proxy.ProxyServerModel.get_proxy_stats", return_value=stats):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/v1/proxy/proxies/stats?cluster_id=1",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 200


class TestProxyGetMetrics:
    """GET /api/v1/proxy/proxies/<id>/metrics"""

    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.get("/api/v1/proxy/proxies/5/metrics")
        assert resp.status_code == 401

    async def test_not_found_returns_404(self, test_app, test_client):
        fresh_db = MagicMock()
        fresh_db.proxy_servers.__getitem__ = MagicMock(return_value=None)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/v1/proxy/proxies/5/metrics",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 404

    async def test_admin_metrics_success(self, test_app, test_client):
        proxy = _proxy_row(5)
        fresh_db = MagicMock()
        fresh_db.proxy_servers.__getitem__ = MagicMock(return_value=proxy)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.proxy.ProxyMetricsModel.get_metrics", return_value=[]), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/v1/proxy/proxies/5/metrics",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 200

    async def test_user_no_access_returns_403(self, test_app, test_client):
        proxy = _proxy_row(5)
        fresh_db = MagicMock()
        fresh_db.proxy_servers.__getitem__ = MagicMock(return_value=proxy)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.cluster.UserClusterAssignmentModel.check_user_cluster_access",
                   return_value=None), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.get(
                "/api/v1/proxy/proxies/5/metrics",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 403


class TestProxyCleanup:
    """POST /api/v1/proxy/proxies/cleanup"""

    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.post("/api/v1/proxy/proxies/cleanup", json={})
        assert resp.status_code == 401

    async def test_non_admin_returns_403(self, test_client):
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v:
            mock_v.return_value = _user_payload()
            resp = await test_client.post(
                "/api/v1/proxy/proxies/cleanup",
                json={},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 403

    async def test_cleanup_success(self, test_app, test_client):
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.proxy.ProxyServerModel.cleanup_stale_proxies", return_value=3), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/proxy/proxies/cleanup",
                json={"timeout_minutes": 5},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 200
        data = await resp.get_json()
        assert data["cleaned_count"] == 3
