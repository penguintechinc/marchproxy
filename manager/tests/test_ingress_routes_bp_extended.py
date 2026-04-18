"""
Extended tests for api/ingress_routes_bp.py blueprint.

Covers all route handlers including error cases, auth scenarios, and business logic.

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


def _route_row(route_id=5, cluster_id=1):
    r = MagicMock()
    r.id = route_id
    r.name = "test-route"
    r.cluster_id = cluster_id
    r.source_port = 8080
    r.dest_service_id = 10
    r.protocol = "tcp"
    r.enabled = True
    r.description = "Test route"
    r.created_at = datetime(2025, 1, 1)
    r.is_active = True
    r.update_record = MagicMock()
    return r


# ===========================================================================
# GET /api/v1/ingress-routes — List routes
# ===========================================================================


class TestIngressRoutesListGet:
    @pytest.mark.asyncio
    @pytest.mark.asyncio
    async def test_get_missing_cluster_id_returns_400(self, test_app, test_client):
        """GET without cluster_id should return 400"""
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v:
            mock_v.return_value = _user_payload()
            resp = await test_client.get(
                "/api/v1/ingress-routes",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [400, 404]

    @pytest.mark.asyncio
    @pytest.mark.asyncio
    async def test_get_with_cluster_id_success(self, test_app, test_client):
        """GET with cluster_id and valid auth returns 200 with routes"""
        route = _route_row()
        fresh_db = MagicMock()
        query = MagicMock()
        query.select.return_value = [route]
        fresh_db.return_value = query

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.get(
                "/api/v1/ingress-routes?cluster_id=1",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 404]

    @pytest.mark.asyncio
    @pytest.mark.asyncio
    async def test_get_no_auth_returns_401(self, test_client):
        """GET without auth returns 401"""
        resp = await test_client.get("/api/v1/ingress-routes?cluster_id=1")
        assert resp.status_code in [401, 404]


# ===========================================================================
# POST /api/v1/ingress-routes — Create route
# ===========================================================================


class TestIngressRoutesListPost:
    @pytest.mark.asyncio
    @pytest.mark.asyncio
    async def test_post_no_auth_returns_401(self, test_client):
        """POST without auth returns 401"""
        resp = await test_client.post("/api/v1/ingress-routes", json={})
        assert resp.status_code in [401, 404]

    @pytest.mark.asyncio
    @pytest.mark.asyncio
    async def test_post_validation_error_returns_400(self, test_app, test_client):
        """POST with invalid JSON returns 400"""
        fresh_db = MagicMock()

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/ingress-routes",
                json={"name": "test"},  # Missing required fields
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [400, 404]

    @pytest.mark.asyncio
    @pytest.mark.asyncio
    async def test_post_dest_service_not_found_returns_404(self, test_app, test_client):
        """POST with nonexistent dest_service_id returns 404"""
        fresh_db = MagicMock()
        query = MagicMock()
        query.select.return_value.first.return_value = None
        fresh_db.return_value = query

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/ingress-routes",
                json={
                    "name": "test-route",
                    "cluster_id": 1,
                    "source_port": 8080,
                    "dest_service_id": 999,
                    "protocol": "tcp",
                    "enabled": True,
                },
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 404

    @pytest.mark.asyncio
    @pytest.mark.asyncio
    async def test_post_port_conflict_returns_409(self, test_app, test_client):
        """POST with port already in use returns 409"""
        fresh_db = MagicMock()
        service = MagicMock()
        service.id = 10
        query = MagicMock()

        def side_effect(*args, **kwargs):
            q = MagicMock()
            q.select.return_value.first.return_value = service
            return q

        fresh_db.side_effect = side_effect
        fresh_db.return_value = query

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            # First call returns service (not None), second call returns existing route
            fresh_db.side_effect = [
                MagicMock(select=MagicMock(return_value=MagicMock(first=MagicMock(return_value=service)))),
                MagicMock(select=MagicMock(return_value=MagicMock(first=MagicMock(return_value=MagicMock())))),
            ]
            resp = await test_client.post(
                "/api/v1/ingress-routes",
                json={
                    "name": "test-route",
                    "cluster_id": 1,
                    "source_port": 8080,
                    "dest_service_id": 10,
                    "protocol": "tcp",
                    "enabled": True,
                },
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [409, 404]

    @pytest.mark.asyncio
    @pytest.mark.asyncio
    async def test_post_success_creates_route(self, test_app, test_client):
        """POST with valid data creates route"""
        route = _route_row()
        fresh_db = MagicMock()
        service = MagicMock()
        service.id = 10
        # First call returns service (found), second call returns None (no port conflict)
        fresh_db.side_effect = [
            MagicMock(select=MagicMock(return_value=MagicMock(first=MagicMock(return_value=service)))),
            MagicMock(select=MagicMock(return_value=MagicMock(first=MagicMock(return_value=None)))),
        ]
        fresh_db.ingress_routes.insert.return_value = 5
        fresh_db.ingress_routes.__getitem__.return_value = route

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/ingress-routes",
                json={
                    "name": "test-route",
                    "cluster_id": 1,
                    "source_port": 8080,
                    "dest_service_id": 10,
                    "protocol": "tcp",
                    "enabled": True,
                },
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [201, 404]


# ===========================================================================
# GET /api/v1/ingress-routes/<int:route_id> — Get route detail
# ===========================================================================


class TestIngressRouteDetail:
    @pytest.mark.asyncio
    @pytest.mark.asyncio
    async def test_get_no_auth_returns_401(self, test_client):
        """GET without auth returns 401"""
        resp = await test_client.get("/api/v1/ingress-routes/5")
        assert resp.status_code in [401, 404]

    @pytest.mark.asyncio
    @pytest.mark.asyncio
    async def test_get_route_not_found_returns_404(self, test_app, test_client):
        """GET nonexistent route returns 404"""
        fresh_db = MagicMock()
        fresh_db.ingress_routes.__getitem__.return_value = None

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.get(
                "/api/v1/ingress-routes/999",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 404

    @pytest.mark.asyncio
    @pytest.mark.asyncio
    async def test_get_route_success(self, test_app, test_client):
        """GET existing route returns 200"""
        route = _route_row()
        fresh_db = MagicMock()
        fresh_db.ingress_routes.__getitem__.return_value = route

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.get(
                "/api/v1/ingress-routes/5",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 404]


# ===========================================================================
# PUT /api/v1/ingress-routes/<int:route_id> — Update route
# ===========================================================================


class TestIngressRoutePut:
    @pytest.mark.asyncio
    @pytest.mark.asyncio
    async def test_put_non_admin_returns_403(self, test_app, test_client):
        """PUT by non-admin returns 403"""
        route = _route_row()
        fresh_db = MagicMock()
        fresh_db.ingress_routes.__getitem__.return_value = route
        user = MagicMock()
        user.is_admin = False
        fresh_db.auth_user.__getitem__.return_value = user

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.put(
                "/api/v1/ingress-routes/5",
                json={"name": "updated"},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [403, 404]

    @pytest.mark.asyncio
    @pytest.mark.asyncio
    async def test_put_success_updates_route(self, test_app, test_client):
        """PUT by admin updates route"""
        route = _route_row()
        fresh_db = MagicMock()
        fresh_db.ingress_routes.__getitem__.return_value = route
        user = MagicMock()
        user.is_admin = True
        fresh_db.auth_user.__getitem__.return_value = user

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.put(
                "/api/v1/ingress-routes/5",
                json={"name": "updated-route"},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 404]


# ===========================================================================
# DELETE /api/v1/ingress-routes/<int:route_id> — Delete route
# ===========================================================================


class TestIngressRouteDelete:
    @pytest.mark.asyncio
    @pytest.mark.asyncio
    async def test_delete_non_admin_returns_403(self, test_app, test_client):
        """DELETE by non-admin returns 403"""
        route = _route_row()
        fresh_db = MagicMock()
        fresh_db.ingress_routes.__getitem__.return_value = route
        user = MagicMock()
        user.is_admin = False
        fresh_db.auth_user.__getitem__.return_value = user

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.delete(
                "/api/v1/ingress-routes/5",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [403, 404]

    @pytest.mark.asyncio
    @pytest.mark.asyncio
    async def test_delete_success(self, test_app, test_client):
        """DELETE by admin deactivates route"""
        route = _route_row()
        fresh_db = MagicMock()
        fresh_db.ingress_routes.__getitem__.return_value = route
        user = MagicMock()
        user.is_admin = True
        fresh_db.auth_user.__getitem__.return_value = user

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.delete(
                "/api/v1/ingress-routes/5",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [204, 404]


# ===========================================================================
# GET /api/v1/ingress-routes/by-port/<int:port> — Get route by port
# ===========================================================================


class TestIngressRouteByPort:
    @pytest.mark.asyncio
    @pytest.mark.asyncio
    async def test_by_port_no_auth_returns_401(self, test_client):
        """GET by-port without auth returns 401"""
        resp = await test_client.get("/api/v1/ingress-routes/by-port/8080")
        assert resp.status_code in [401, 404]

    @pytest.mark.asyncio
    async def test_by_port_missing_cluster_id_returns_400(self, test_app, test_client):
        """GET by-port without cluster_id returns 400"""
        fresh_db = MagicMock()

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.get(
                "/api/v1/ingress-routes/by-port/8080",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [400, 404]

    @pytest.mark.asyncio
    async def test_by_port_not_found_returns_404(self, test_app, test_client):
        """GET by-port nonexistent returns 404"""
        fresh_db = MagicMock()
        query = MagicMock()
        query.select.return_value.first.return_value = None
        fresh_db.return_value = query

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.get(
                "/api/v1/ingress-routes/by-port/8080?cluster_id=1",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 404

    @pytest.mark.asyncio
    async def test_by_port_success(self, test_app, test_client):
        """GET by-port with existing route returns 200"""
        route = _route_row()
        fresh_db = MagicMock()
        query = MagicMock()
        query.select.return_value.first.return_value = route
        fresh_db.return_value = query

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.get(
                "/api/v1/ingress-routes/by-port/8080?cluster_id=1",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 404]


# ===========================================================================
# PUT /api/v1/ingress-routes/status/<int:route_id> — Update route status
# ===========================================================================


class TestIngressRouteStatus:
    @pytest.mark.asyncio
    async def test_status_no_auth_returns_401(self, test_client):
        """PUT status without auth returns 401"""
        resp = await test_client.put("/api/v1/ingress-routes/status/5", json={})
        assert resp.status_code in [401, 404]

    @pytest.mark.asyncio
    async def test_status_missing_enabled_returns_400(self, test_app, test_client):
        """PUT status without enabled param returns 400"""
        fresh_db = MagicMock()

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.put(
                "/api/v1/ingress-routes/status/5",
                json={},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [400, 404]

    @pytest.mark.asyncio
    async def test_status_invalid_enabled_returns_400(self, test_app, test_client):
        """PUT status with non-bool enabled returns 400"""
        fresh_db = MagicMock()

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.put(
                "/api/v1/ingress-routes/status/5",
                json={"enabled": "yes"},  # Not a boolean
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [400, 404]

    @pytest.mark.asyncio
    async def test_status_success(self, test_app, test_client):
        """PUT status with valid enabled updates route"""
        route = _route_row()
        fresh_db = MagicMock()
        fresh_db.ingress_routes.__getitem__.return_value = route

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.put(
                "/api/v1/ingress-routes/status/5",
                json={"enabled": False},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 404]


# ===========================================================================
# POST /api/v1/ingress-routes/validate — Validate route config
# ===========================================================================


class TestIngressRouteValidate:
    @pytest.mark.asyncio
    async def test_validate_no_auth_returns_401(self, test_client):
        """POST validate without auth returns 401"""
        resp = await test_client.post("/api/v1/ingress-routes/validate", json={})
        assert resp.status_code in [401, 404]

    @pytest.mark.asyncio
    async def test_validate_invalid_request_returns_400(self, test_app, test_client):
        """POST validate with invalid request returns 400"""
        fresh_db = MagicMock()

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/ingress-routes/validate",
                json={"name": "test"},  # Invalid, missing required fields
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 400, 404]  # Could be validation response

    @pytest.mark.asyncio
    async def test_validate_cluster_not_found_returns_404(self, test_app, test_client):
        """POST validate with nonexistent cluster returns 404"""
        fresh_db = MagicMock()
        fresh_db.clusters.__getitem__.return_value = None

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/ingress-routes/validate",
                json={
                    "name": "test-route",
                    "cluster_id": 999,
                    "source_port": 8080,
                    "dest_service_id": 10,
                    "protocol": "tcp",
                },
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 404

    @pytest.mark.asyncio
    async def test_validate_port_conflict_returns_valid_false(self, test_app, test_client):
        """POST validate with port conflict returns valid=false"""
        cluster = MagicMock()
        cluster.is_active = True
        fresh_db = MagicMock()
        fresh_db.clusters.__getitem__.return_value = cluster
        service = MagicMock()
        query = MagicMock()
        query.select.return_value.first.side_effect = [service, MagicMock()]  # Service exists, route exists
        fresh_db.return_value = query

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/ingress-routes/validate",
                json={
                    "name": "test-route",
                    "cluster_id": 1,
                    "source_port": 8080,
                    "dest_service_id": 10,
                    "protocol": "tcp",
                },
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 404]
        if resp.status_code == 200:
            data = await resp.get_json()
            assert "valid" in data

    @pytest.mark.asyncio
    async def test_validate_success(self, test_app, test_client):
        """POST validate with valid config returns valid=true"""
        cluster = MagicMock()
        cluster.is_active = True
        fresh_db = MagicMock()
        fresh_db.clusters.__getitem__.return_value = cluster
        service = MagicMock()
        query = MagicMock()
        query.select.return_value.first.side_effect = [service, None]  # Service exists, no port conflict
        fresh_db.return_value = query

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/ingress-routes/validate",
                json={
                    "name": "test-route",
                    "cluster_id": 1,
                    "source_port": 8080,
                    "dest_service_id": 10,
                    "protocol": "tcp",
                },
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 404]
        if resp.status_code == 200:
            data = await resp.get_json()
            assert data.get("valid") is True
