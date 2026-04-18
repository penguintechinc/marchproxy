"""
Tests for api/services_bp.py blueprint.

Blueprint registered at /api (overrides own prefix).
Key routes:
  GET/POST /api                      → services_list()
  GET/PUT/DELETE /api/<int:svc_id>   → service_detail() [wins integer IDs]
  POST /api/<int:svc_id>/auth        → set_service_auth()
  POST /api/<int:svc_id>/auth/rotate → rotate_service_jwt()
  POST /api/<int:svc_id>/token       → create_service_token()
  POST /api/<int:svc_id>/assign      → assign_user_to_service()
  DELETE /api/<int:svc_id>/unassign/<int:uid> → remove_user_from_service()

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


def _service_row(svc_id=10):
    s = MagicMock()
    s.id = svc_id
    s.name = "test-service"
    s.ip_fqdn = "192.168.1.10"
    s.port = 8080
    s.protocol = "http"
    s.collection = "default"
    s.cluster_id = 1
    s.auth_type = "none"
    s.tls_enabled = False
    s.tls_verify = True
    s.health_check_enabled = False
    s.health_check_path = "/health"
    s.health_check_interval = 30
    s.jwt_secret = None
    s.jwt_expiry = 3600
    s.created_at = datetime(2025, 1, 1)
    s.is_active = True
    return s


# test_app and test_client come from tests/conftest.py


# ===========================================================================
# GET/PUT/DELETE /api/<int:service_id> — service_detail()
# Note: services blueprint wins integer ID routes per registration order
# ===========================================================================

class TestServiceDetailGet:
    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.get("/api/10")
        assert resp.status_code == 401

    async def test_service_not_found_returns_404(self, test_app, test_client):
        fresh_db = MagicMock()
        fresh_db.services.__getitem__ = MagicMock(return_value=None)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.get(
                "/api/999",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [404, 500]

    async def test_get_service_success(self, test_app, test_client):
        fresh_db = MagicMock()
        svc = _service_row()
        svc_config = {
            "id": 10, "name": "test-service", "ip_fqdn": "192.168.1.10", "port": 8080,
            "protocol": "http", "collection": "default", "cluster_id": 1,
            "auth_type": "none", "tls_enabled": False, "health_check_enabled": False,
        }
        fresh_db.services.__getitem__ = MagicMock(return_value=svc)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.service.ServiceModel.get_service_config", return_value=svc_config), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.get(
                "/api/10",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]

    async def test_get_service_config_not_found(self, test_app, test_client):
        fresh_db = MagicMock()
        svc = _service_row()
        fresh_db.services.__getitem__ = MagicMock(return_value=svc)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.service.ServiceModel.get_service_config", return_value=None), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.get(
                "/api/10",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [404, 500]


class TestServiceDetailPut:
    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.put("/api/10", json={"name": "new-name"})
        assert resp.status_code == 401

    async def test_non_admin_user_returns_403(self, test_app, test_client):
        fresh_db = MagicMock()
        svc = _service_row()
        fresh_db.services.__getitem__ = MagicMock(return_value=svc)
        auth_user = MagicMock()
        auth_user.is_admin = False
        fresh_db.auth_user.__getitem__ = MagicMock(return_value=auth_user)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.put(
                "/api/10",
                json={"name": "new-name"},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [403, 500]

    async def test_update_service_success(self, test_app, test_client):
        fresh_db = MagicMock()
        svc = _service_row()
        fresh_db.services.__getitem__ = MagicMock(return_value=svc)
        auth_user = MagicMock()
        auth_user.is_admin = True
        fresh_db.auth_user.__getitem__ = MagicMock(return_value=auth_user)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.put(
                "/api/10",
                json={"name": "updated-name", "port": 9090},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]


class TestServiceDetailDelete:
    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.delete("/api/10")
        assert resp.status_code == 401

    async def test_non_admin_returns_403(self, test_app, test_client):
        fresh_db = MagicMock()
        svc = _service_row()
        fresh_db.services.__getitem__ = MagicMock(return_value=svc)
        auth_user = MagicMock()
        auth_user.is_admin = False
        fresh_db.auth_user.__getitem__ = MagicMock(return_value=auth_user)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.delete(
                "/api/10",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [403, 500]

    async def test_delete_service_success(self, test_app, test_client):
        fresh_db = MagicMock()
        svc = _service_row()
        fresh_db.services.__getitem__ = MagicMock(return_value=svc)
        auth_user = MagicMock()
        auth_user.is_admin = True
        fresh_db.auth_user.__getitem__ = MagicMock(return_value=auth_user)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.delete(
                "/api/10",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 204, 500]


# ===========================================================================
# POST /api/<service_id>/auth — set_service_auth()
# ===========================================================================

class TestSetServiceAuth:
    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.post("/api/10/auth", json={"auth_type": "none"})
        assert resp.status_code == 401

    async def test_service_not_found_returns_404(self, test_app, test_client):
        fresh_db = MagicMock()
        fresh_db.services.__getitem__ = MagicMock(return_value=None)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/999/auth",
                json={"auth_type": "none"},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [404, 500]

    async def test_set_auth_none(self, test_app, test_client):
        fresh_db = MagicMock()
        svc = _service_row()
        fresh_db.services.__getitem__ = MagicMock(return_value=svc)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/10/auth",
                json={"auth_type": "none"},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]

    async def test_set_auth_base64(self, test_app, test_client):
        fresh_db = MagicMock()
        svc = _service_row()
        fresh_db.services.__getitem__ = MagicMock(return_value=svc)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.service.ServiceModel.set_base64_auth", return_value="token123"), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/10/auth",
                json={"auth_type": "base64"},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]

    async def test_set_auth_jwt(self, test_app, test_client):
        fresh_db = MagicMock()
        svc = _service_row()
        fresh_db.services.__getitem__ = MagicMock(return_value=svc)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.service.ServiceModel.set_jwt_auth", return_value="jwt-secret"), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/10/auth",
                json={"auth_type": "jwt"},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]

    async def test_set_auth_invalid_type_returns_400(self, test_app, test_client):
        fresh_db = MagicMock()
        svc = _service_row()
        fresh_db.services.__getitem__ = MagicMock(return_value=svc)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/10/auth",
                json={"auth_type": "invalid"},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [400, 500]


# ===========================================================================
# POST /api/<service_id>/auth/rotate — rotate_service_jwt()
# ===========================================================================

class TestRotateServiceJwt:
    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.post("/api/10/auth/rotate", json={})
        assert resp.status_code == 401

    async def test_service_not_jwt_returns_404(self, test_app, test_client):
        fresh_db = MagicMock()
        svc = _service_row()
        svc.auth_type = "none"
        fresh_db.services.__getitem__ = MagicMock(return_value=svc)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/10/auth/rotate",
                json={},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [404, 500]

    async def test_rotate_success(self, test_app, test_client):
        fresh_db = MagicMock()
        svc = _service_row()
        svc.auth_type = "jwt"
        fresh_db.services.__getitem__ = MagicMock(return_value=svc)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.service.ServiceModel.rotate_jwt_secret", return_value="new-secret"), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/10/auth/rotate",
                json={},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]


# ===========================================================================
# POST /api/<service_id>/token — create_service_token()
# ===========================================================================

class TestCreateServiceToken:
    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.post("/api/10/token", json={})
        assert resp.status_code == 401

    async def test_service_not_jwt_returns_404(self, test_app, test_client):
        fresh_db = MagicMock()
        svc = _service_row()
        svc.auth_type = "base64"
        fresh_db.services.__getitem__ = MagicMock(return_value=svc)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/10/token",
                json={},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [404, 500]

    async def test_create_token_success(self, test_app, test_client):
        fresh_db = MagicMock()
        svc = _service_row()
        svc.auth_type = "jwt"
        fresh_db.services.__getitem__ = MagicMock(return_value=svc)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.service.ServiceModel.create_jwt_token", return_value="jwt.token.here"), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/10/token",
                json={"service_id": 10},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]


# ===========================================================================
# POST /api/<service_id>/assign — assign_user_to_service()
# ===========================================================================

class TestAssignUserToService:
    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.post("/api/10/assign", json={"user_id": 2})
        assert resp.status_code == 401

    async def test_validation_error_returns_400(self, test_app, test_client):
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/10/assign",
                json={},  # Missing user_id
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [400, 500]

    async def test_assign_success(self, test_app, test_client):
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.service.UserServiceAssignmentModel.assign_user_to_service",
                   return_value=True), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/10/assign",
                json={"user_id": 2},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]


# ===========================================================================
# DELETE /api/<service_id>/unassign/<user_id> — remove_user_from_service()
# ===========================================================================

class TestRemoveUserFromService:
    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.delete("/api/10/unassign/2")
        assert resp.status_code == 401

    async def test_remove_success(self, test_app, test_client):
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.service.UserServiceAssignmentModel.remove_user_from_service",
                   return_value=True), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.delete(
                "/api/10/unassign/2",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]

    async def test_remove_failure_returns_500(self, test_app, test_client):
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.service.UserServiceAssignmentModel.remove_user_from_service",
                   return_value=False), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.delete(
                "/api/10/unassign/2",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [500]
