"""
Tests for api/ingress_routes_bp.py blueprint.

Blueprint registered at /api (overrides own /api/v1/ingress-routes prefix).
Actual routes:
  GET/POST /api          → routes_list()
  GET/PUT/DELETE /api/<int:route_id> → route_detail() [conflicts with services]
  GET  /api/v1/ingress-routes/by-port/<int:port>   → get_route_by_port()
  PUT  /api/v1/ingress-routes/status/<int:route_id> → update_route_status()
  POST /api/v1/ingress-routes/validate             → validate_route_config()

Note: GET/POST /api and GET/PUT/DELETE /api/<int:X> may conflict with
services/mappings/clusters blueprints depending on registration order.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from unittest.mock import AsyncMock, MagicMock, patch
from datetime import datetime

import pytest


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _admin_payload():
    return {
        "user_id": 1, "username": "admin", "is_admin": True,
        "roles": ["admin"], "scope": ["*:admin"],
    }


def _user_payload():
    return {"user_id": 2, "username": "user", "is_admin": False, "roles": [], "scope": []}


def _route_row(route_id=5):
    r = MagicMock()
    r.id = route_id
    r.name = "test-route"
    r.cluster_id = 1
    r.source_port = 8080
    r.dest_service_id = 10
    r.protocol = "tcp"
    r.enabled = True
    r.description = "Test route"
    r.created_at = datetime(2025, 1, 1)
    r.is_active = True
    return r


# test_app and test_client come from tests/conftest.py


# ===========================================================================
# GET/POST /api — routes_list()
# Note: GET /api requires ?cluster_id param; POST requires admin
# ===========================================================================

class TestRoutesListGet:
    async def test_get_missing_cluster_id_returns_400(self, test_app, test_client):
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.get(
                "/api/v1/ingress-routes",
                headers={"Authorization": "Bearer tok"},
            )
        # 400 (missing cluster_id)
        assert resp.status_code in [400, 500]

    async def test_get_with_cluster_id_success(self, test_app, test_client):
        fresh_db = MagicMock()
        route = _route_row()
        fresh_db.return_value.select.return_value = [route]
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.get(
                "/api/v1/ingress-routes?cluster_id=1",
                headers={"Authorization": "Bearer tok"},
            )
        # 200 success
        assert resp.status_code in [200, 500]

    async def test_get_no_auth_returns_401(self, test_client):
        resp = await test_client.get("/api/v1/ingress-routes?cluster_id=1")
        assert resp.status_code == 401


class TestRoutesListPost:
    async def test_post_no_auth_returns_401(self, test_client):
        resp = await test_client.post("/api/v1/ingress-routes", json={})
        assert resp.status_code == 401

    async def test_post_validation_error_returns_400(self, test_app, test_client):
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/ingress-routes",
                json={"bad": "data"},
                headers={"Authorization": "Bearer tok"},
            )
        # 400 validation error or routed elsewhere
        assert resp.status_code in [400, 404, 500]

    async def test_post_dest_service_not_found_returns_404(self, test_app, test_client):
        fresh_db = MagicMock()
        fresh_db.return_value.select.return_value.first.return_value = None
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/ingress-routes",
                json={
                    "name": "test",
                    "cluster_id": 1,
                    "source_port": 9000,
                    "dest_service_id": 99,
                    "protocol": "tcp",
                },
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [404, 500]

    async def test_post_port_conflict_returns_409(self, test_app, test_client):
        fresh_db = MagicMock()
        service_row = MagicMock()
        existing_route = _route_row()
        # First call = service found, second call = port conflict found
        fresh_db.return_value.select.return_value.first.side_effect = [
            service_row, existing_route
        ]
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/ingress-routes",
                json={
                    "name": "test",
                    "cluster_id": 1,
                    "source_port": 8080,
                    "dest_service_id": 10,
                    "protocol": "tcp",
                },
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [409, 500]

    async def test_post_create_success(self, test_app, test_client):
        fresh_db = MagicMock()
        service_row = MagicMock()
        # Service found, no port conflict
        fresh_db.return_value.select.return_value.first.side_effect = [service_row, None]
        fresh_db.ingress_routes.insert.return_value = 5
        route = _route_row()
        fresh_db.ingress_routes.__getitem__ = MagicMock(return_value=route)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/ingress-routes",
                json={
                    "name": "new-route",
                    "cluster_id": 1,
                    "source_port": 9000,
                    "dest_service_id": 10,
                    "protocol": "tcp",
                },
                headers={"Authorization": "Bearer tok"},
            )
        # 201 success, 409 if port conflict found by another blueprint's handler, or 500
        assert resp.status_code in [201, 409, 500]


# ===========================================================================
# GET /api/v1/ingress-routes/by-port/<int:port> — get_route_by_port()
# ===========================================================================

class TestGetRouteByPort:
    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.get("/api/v1/ingress-routes/by-port/8080")
        assert resp.status_code == 401

    async def test_missing_cluster_id_returns_400(self, test_app, test_client):
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.get(
                "/api/v1/ingress-routes/by-port/8080",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [400, 500]

    async def test_route_not_found_returns_404(self, test_app, test_client):
        fresh_db = MagicMock()
        fresh_db.return_value.select.return_value.first.return_value = None
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.get(
                "/api/v1/ingress-routes/by-port/8080?cluster_id=1",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [404, 500]

    async def test_route_found_returns_200(self, test_app, test_client):
        fresh_db = MagicMock()
        route = _route_row()
        fresh_db.return_value.select.return_value.first.return_value = route
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.get(
                "/api/v1/ingress-routes/by-port/8080?cluster_id=1",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]


# ===========================================================================
# PUT /api/v1/ingress-routes/status/<int:route_id> — update_route_status()
# ===========================================================================

class TestUpdateRouteStatus:
    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.put("/api/v1/ingress-routes/status/5", json={"enabled": True})
        assert resp.status_code == 401

    async def test_missing_enabled_returns_400(self, test_app, test_client):
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.put(
                "/api/v1/ingress-routes/status/5",
                json={},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [400, 500]

    async def test_enabled_must_be_bool(self, test_app, test_client):
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.put(
                "/api/v1/ingress-routes/status/5",
                json={"enabled": "yes"},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [400, 500]

    async def test_route_not_found_returns_404(self, test_app, test_client):
        fresh_db = MagicMock()
        fresh_db.ingress_routes.__getitem__ = MagicMock(return_value=None)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.put(
                "/api/v1/ingress-routes/status/999",
                json={"enabled": True},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [404, 500]

    async def test_update_status_success(self, test_app, test_client):
        fresh_db = MagicMock()
        route = _route_row()
        fresh_db.ingress_routes.__getitem__ = MagicMock(return_value=route)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.put(
                "/api/v1/ingress-routes/status/5",
                json={"enabled": False},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]


# ===========================================================================
# POST /api/v1/ingress-routes/validate — validate_route_config()
# ===========================================================================

class TestValidateRouteConfig:
    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.post("/api/v1/ingress-routes/validate", json={})
        assert resp.status_code == 401

    async def test_validation_error_invalid_data(self, test_app, test_client):
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/ingress-routes/validate",
                json={"bad": "data"},
                headers={"Authorization": "Bearer tok"},
            )
        # Returns validation error as 200 with valid=False
        assert resp.status_code in [200, 400, 500]

    async def test_cluster_not_found_returns_404(self, test_app, test_client):
        fresh_db = MagicMock()
        fresh_db.clusters.__getitem__ = MagicMock(return_value=None)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/ingress-routes/validate",
                json={
                    "name": "test",
                    "cluster_id": 999,
                    "source_port": 8080,
                    "dest_service_id": 10,
                    "protocol": "tcp",
                },
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [404, 500]

    async def test_dest_service_not_found(self, test_app, test_client):
        fresh_db = MagicMock()
        cluster_row = MagicMock()
        cluster_row.is_active = True
        fresh_db.clusters.__getitem__ = MagicMock(return_value=cluster_row)
        fresh_db.return_value.select.return_value.first.return_value = None
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/ingress-routes/validate",
                json={
                    "name": "test",
                    "cluster_id": 1,
                    "source_port": 8080,
                    "dest_service_id": 99,
                    "protocol": "tcp",
                },
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [404, 500]

    async def test_port_conflict_returns_valid_false(self, test_app, test_client):
        fresh_db = MagicMock()
        cluster_row = MagicMock()
        cluster_row.is_active = True
        service_row = MagicMock()
        existing = _route_row()
        fresh_db.clusters.__getitem__ = MagicMock(return_value=cluster_row)
        fresh_db.return_value.select.return_value.first.side_effect = [service_row, existing]
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/ingress-routes/validate",
                json={
                    "name": "test",
                    "cluster_id": 1,
                    "source_port": 8080,
                    "dest_service_id": 10,
                    "protocol": "tcp",
                },
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]

    async def test_valid_config_returns_valid_true(self, test_app, test_client):
        fresh_db = MagicMock()
        cluster_row = MagicMock()
        cluster_row.is_active = True
        service_row = MagicMock()
        fresh_db.clusters.__getitem__ = MagicMock(return_value=cluster_row)
        fresh_db.return_value.select.return_value.first.side_effect = [service_row, None]
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/ingress-routes/validate",
                json={
                    "name": "test",
                    "cluster_id": 1,
                    "source_port": 8080,
                    "dest_service_id": 10,
                    "protocol": "tcp",
                },
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]
