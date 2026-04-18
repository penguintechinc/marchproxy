"""
Targeted tests for services_bp.py and mappings_bp.py uncovered paths.

Covers:
- services_bp: POST create (line 109), GET not found (line 134),
  PUT updates (lines 146-190), POST auth invalid/exception (lines 245-250),
  POST rotate JWT (lines 269-281), POST token (lines 302-307),
  POST/DELETE assign/unassign (lines 334-336, 351-353)
- mappings_bp: GET list (line 46), POST create (line 106),
  GET detail (line 144), PUT update (lines 155-186)

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from datetime import datetime
from unittest.mock import AsyncMock, MagicMock, patch

import pytest


# ---------------------------------------------------------------------------
# Helper payloads
# ---------------------------------------------------------------------------

def _admin_payload():
    """Admin user payload for mocking _validate_token."""
    return {
        "user_id": 1,
        "sub": "1",
        "username": "admin",
        "is_admin": True,
        "scope": ["*:admin", "*:read", "*:write"],
        "roles": ["admin"],
        "tenant": "test",
        "session_id": "sess-1",
    }


def _user_payload():
    """Regular user payload (non-admin)."""
    return {
        "user_id": 2,
        "sub": "2",
        "username": "testuser",
        "is_admin": False,
        "scope": ["*:read"],
        "roles": [],
        "tenant": "test",
        "session_id": "sess-2",
    }


# ---------------------------------------------------------------------------
# Mock factories
# ---------------------------------------------------------------------------

def _make_service_mock(service_id=1, auth_type="none"):
    """Create a mock service object."""
    s = MagicMock()
    s.id = service_id
    s.name = f"service-{service_id}"
    s.ip_fqdn = f"10.0.0.{service_id}"
    s.port = 8080 + service_id
    s.protocol = "tcp"
    s.collection = None
    s.cluster_id = 1
    s.auth_type = auth_type
    s.tls_enabled = False
    s.tls_verify = False
    s.health_check_enabled = False
    s.health_check_path = "/health"
    s.health_check_interval = 30
    s.jwt_expiry = 3600
    s.created_at = datetime(2025, 1, 1, 10, 0, 0)
    s.updated_at = datetime(2025, 1, 1, 10, 0, 0)
    s.update_record = MagicMock()
    return s


def _make_mapping_mock(mapping_id=1):
    """Create a mock mapping object."""
    m = MagicMock()
    m.id = mapping_id
    m.name = f"mapping-{mapping_id}"
    m.description = f"Test mapping {mapping_id}"
    m.source_services = [{"id": 1, "name": "service-1"}]
    m.dest_services = [{"id": 2, "name": "service-2"}]
    m.cluster_id = 1
    m.protocols = ["tcp"]
    m.ports = [{"port": 8080}]
    m.auth_required = False
    m.priority = 100
    m.created_at = datetime(2025, 1, 1, 10, 0, 0)
    m.updated_at = datetime(2025, 1, 1, 10, 0, 0)
    m.comments = None
    m.update_record = MagicMock()
    return m


def _make_auth_user_mock(user_id=1, is_admin=True):
    """Create a mock auth_user object."""
    u = MagicMock()
    u.id = user_id
    u.username = f"user-{user_id}"
    u.is_admin = is_admin
    return u


# ===========================================================================
# services_bp tests
# ===========================================================================

class TestCreateService:
    """POST /api/v1/services - create service"""

    @pytest.mark.asyncio
    async def test_create_service_success(self, test_app, test_client):
        """Covers line 109: return jsonify(response.dict()), 201"""
        service_mock = _make_service_mock(service_id=1)
        test_app.db.services.__getitem__ = MagicMock(return_value=service_mock)

        with patch("models.service.ServiceModel.create_service", return_value=1):
            response = await test_client.post(
                "/api/v1/services",
                json={
                    "name": "test-service",
                    "ip_fqdn": "10.0.0.1",
                    "port": 8080,
                    "cluster_id": 1,
                    "protocol": "tcp",
                    "collection": None,
                    "auth_type": "none",
                    "tls_enabled": False,
                    "tls_verify": False,
                },
                headers={"Authorization": "Bearer admin-token"},
            )
        assert response.status_code == 201
        data = await response.get_json()
        assert data["id"] == 1
        assert "service" in data["name"].lower()

    @pytest.mark.asyncio
    async def test_create_service_validation_error(self, test_app, test_client):
        """Missing required fields -> validation error"""
        with patch("models.service.ServiceModel.create_service"):
            response = await test_client.post(
                "/api/v1/services",
                json={"name": "test"},
                headers={"Authorization": "Bearer admin-token"},
            )
        assert response.status_code == 400

    @pytest.mark.asyncio
    async def test_create_service_exception(self, test_app, test_client):
        """Exception during create -> 500"""
        with patch(
            "models.service.ServiceModel.create_service",
            side_effect=Exception("DB error"),
        ):
            response = await test_client.post(
                "/api/v1/services",
                json={
                    "name": "test-service",
                    "ip_fqdn": "10.0.0.1",
                    "port": 8080,
                    "cluster_id": 1,
                    "protocol": "tcp",
                    "collection": None,
                    "auth_type": "none",
                    "tls_enabled": False,
                    "tls_verify": False,
                },
                headers={"Authorization": "Bearer admin-token"},
            )
        assert response.status_code == 500
        data = await response.get_json()
        assert "error" in data


class TestGetServiceDetail:
    """GET /api/v1/services/<service_id>"""

    @pytest.mark.asyncio
    async def test_get_service_config_not_found(self, test_app, test_client):
        """Covers line 134: service config not found -> 404"""
        test_app.db.services.__getitem__ = MagicMock(
            return_value=_make_service_mock()
        )

        with patch(
            "models.service.ServiceModel.get_service_config", return_value=None
        ):
            response = await test_client.get(
                "/api/v1/services/1",
                headers={"Authorization": "Bearer admin-token"},
            )
        assert response.status_code == 404
        data = await response.get_json()
        assert data["error"] == "Service not found"

    @pytest.mark.asyncio
    async def test_get_service_success(self, test_app, test_client):
        """GET service returns config successfully"""
        test_app.db.services.__getitem__ = MagicMock(
            return_value=_make_service_mock()
        )

        with patch(
            "models.service.ServiceModel.get_service_config",
            return_value={"id": 1, "name": "test-service"},
        ):
            response = await test_client.get(
                "/api/v1/services/1",
                headers={"Authorization": "Bearer admin-token"},
            )
        assert response.status_code == 200


class TestUpdateService:
    """PUT /api/v1/services/<service_id> - update service"""

    @pytest.mark.asyncio
    async def test_update_service_all_fields(self, test_app, test_client):
        """Covers lines 146-190: update all fields -> 200"""
        service = _make_service_mock(service_id=1)
        test_app.db.services.__getitem__ = MagicMock(return_value=service)

        response = await test_client.put(
            "/api/v1/services/1",
            json={
                "name": "updated-service",
                "ip_fqdn": "10.0.0.2",
                "port": 9000,
                "protocol": "http",
                "collection": "test-collection",
                "auth_type": "jwt",
                "tls_enabled": True,
                "tls_verify": True,
                "health_check_enabled": True,
                "health_check_path": "/healthz",
                "health_check_interval": 60,
            },
            headers={"Authorization": "Bearer admin-token"},
        )
        assert response.status_code == 200
        data = await response.get_json()
        assert data["id"] == 1

    @pytest.mark.asyncio
    async def test_update_service_partial_fields(self, test_app, test_client):
        """Update only some fields -> 200"""
        service = _make_service_mock(service_id=1)
        test_app.db.services.__getitem__ = MagicMock(return_value=service)

        response = await test_client.put(
            "/api/v1/services/1",
            json={
                "port": 9999,
                "protocol": "http",
            },
            headers={"Authorization": "Bearer admin-token"},
        )
        assert response.status_code == 200

    @pytest.mark.asyncio
    async def test_update_service_non_admin_forbidden(self, test_app, test_client):
        """Non-admin PUT -> 403"""
        service = _make_service_mock(service_id=1)
        test_app.db.services.__getitem__ = MagicMock(return_value=service)

        with patch(
            "middleware.auth._validate_token", new_callable=AsyncMock
        ) as mv:
            mv.return_value = _user_payload()
            response = await test_client.put(
                "/api/v1/services/1",
                json={"port": 9999},
                headers={"Authorization": "Bearer user-token"},
            )
        assert response.status_code == 403

    @pytest.mark.asyncio
    async def test_update_service_validation_error(self, test_app, test_client):
        """Invalid JSON -> validation error"""
        service = _make_service_mock(service_id=1)
        test_app.db.services.__getitem__ = MagicMock(return_value=service)

        response = await test_client.put(
            "/api/v1/services/1",
            json={"port": "invalid-port"},
            headers={"Authorization": "Bearer admin-token"},
        )
        assert response.status_code == 400


class TestSetServiceAuth:
    """POST /api/v1/services/<service_id>/auth"""

    @pytest.mark.asyncio
    async def test_set_auth_invalid_type(self, test_app, test_client):
        """Covers line 245: invalid auth type -> 400"""
        test_app.db.services.__getitem__ = MagicMock(
            return_value=_make_service_mock()
        )

        response = await test_client.post(
            "/api/v1/services/1/auth",
            json={"auth_type": "invalid_type"},
            headers={"Authorization": "Bearer admin-token"},
        )
        assert response.status_code == 400
        data = await response.get_json()
        # Could be validation error or "Invalid auth type" depending on order
        assert "error" in data

    @pytest.mark.asyncio
    async def test_set_auth_base64_success(self, test_app, test_client):
        """Set base64 auth -> 200"""
        test_app.db.services.__getitem__ = MagicMock(
            return_value=_make_service_mock()
        )

        with patch(
            "models.service.ServiceModel.set_base64_auth", return_value="token123"
        ):
            response = await test_client.post(
                "/api/v1/services/1/auth",
                json={"auth_type": "base64"},
                headers={"Authorization": "Bearer admin-token"},
            )
        assert response.status_code == 200
        data = await response.get_json()
        assert data["auth_type"] == "base64"

    @pytest.mark.asyncio
    async def test_set_auth_jwt_success(self, test_app, test_client):
        """Set JWT auth -> 200"""
        test_app.db.services.__getitem__ = MagicMock(
            return_value=_make_service_mock()
        )

        with patch(
            "models.service.ServiceModel.set_jwt_auth", return_value="jwt-secret"
        ):
            response = await test_client.post(
                "/api/v1/services/1/auth",
                json={
                    "auth_type": "jwt",
                    "jwt_expiry": 3600,
                    "jwt_algorithm": "HS256",
                },
                headers={"Authorization": "Bearer admin-token"},
            )
        assert response.status_code == 200
        data = await response.get_json()
        assert data["auth_type"] == "jwt"

    @pytest.mark.asyncio
    async def test_set_auth_none_success(self, test_app, test_client):
        """Set auth to 'none' -> 200"""
        service = _make_service_mock()
        test_app.db.services.__getitem__ = MagicMock(return_value=service)

        response = await test_client.post(
            "/api/v1/services/1/auth",
            json={"auth_type": "none"},
            headers={"Authorization": "Bearer admin-token"},
        )
        assert response.status_code == 200
        data = await response.get_json()
        assert data["auth_type"] == "none"
        service.update_record.assert_called_once()

    @pytest.mark.asyncio
    async def test_set_auth_exception(self, test_app, test_client):
        """Covers lines 248-250: exception -> 500"""
        test_app.db.services.__getitem__ = MagicMock(
            return_value=_make_service_mock()
        )

        with patch(
            "models.service.ServiceModel.set_base64_auth",
            side_effect=Exception("DB error"),
        ):
            response = await test_client.post(
                "/api/v1/services/1/auth",
                json={"auth_type": "base64"},
                headers={"Authorization": "Bearer admin-token"},
            )
        assert response.status_code == 500
        data = await response.get_json()
        assert "error" in data


class TestRotateServiceJwt:
    """POST /api/v1/services/<service_id>/auth/rotate"""

    @pytest.mark.asyncio
    async def test_rotate_jwt_success(self, test_app, test_client):
        """Covers lines 269, 277: success path -> 200"""
        service = _make_service_mock(auth_type="jwt")
        test_app.db.services.__getitem__ = MagicMock(return_value=service)

        with patch(
            "models.service.ServiceModel.rotate_jwt_secret", return_value="new-secret"
        ):
            response = await test_client.post(
                "/api/v1/services/1/auth/rotate",
                headers={"Authorization": "Bearer admin-token"},
            )
        assert response.status_code == 200
        data = await response.get_json()
        assert data["auth_type"] == "jwt"
        assert data["jwt_secret"] == "new-secret"

    @pytest.mark.asyncio
    async def test_rotate_jwt_not_jwt_auth(self, test_app, test_client):
        """Service not using JWT -> 404"""
        service = _make_service_mock(auth_type="base64")
        test_app.db.services.__getitem__ = MagicMock(return_value=service)

        response = await test_client.post(
            "/api/v1/services/1/auth/rotate",
            headers={"Authorization": "Bearer admin-token"},
        )
        assert response.status_code == 404

    @pytest.mark.asyncio
    async def test_rotate_jwt_failed(self, test_app, test_client):
        """rotate_jwt_secret returns None -> 500"""
        service = _make_service_mock(auth_type="jwt")
        test_app.db.services.__getitem__ = MagicMock(return_value=service)

        with patch(
            "models.service.ServiceModel.rotate_jwt_secret", return_value=None
        ):
            response = await test_client.post(
                "/api/v1/services/1/auth/rotate",
                headers={"Authorization": "Bearer admin-token"},
            )
        assert response.status_code == 500

    @pytest.mark.asyncio
    async def test_rotate_jwt_exception(self, test_app, test_client):
        """Covers lines 279-281: exception -> 500"""
        service = _make_service_mock(auth_type="jwt")
        test_app.db.services.__getitem__ = MagicMock(return_value=service)

        with patch(
            "models.service.ServiceModel.rotate_jwt_secret",
            side_effect=Exception("DB error"),
        ):
            response = await test_client.post(
                "/api/v1/services/1/auth/rotate",
                headers={"Authorization": "Bearer admin-token"},
            )
        assert response.status_code == 500
        data = await response.get_json()
        assert "error" in data


class TestCreateServiceToken:
    """POST /api/v1/services/<service_id>/token"""

    @pytest.mark.asyncio
    async def test_create_token_success(self, test_app, test_client):
        """Covers lines 302, 305: success path -> 200"""
        service = _make_service_mock(auth_type="jwt")
        test_app.db.services.__getitem__ = MagicMock(return_value=service)

        with patch(
            "models.service.ServiceModel.create_jwt_token", return_value="token123"
        ):
            response = await test_client.post(
                "/api/v1/services/1/token",
                json={"additional_claims": {}},
                headers={"Authorization": "Bearer admin-token"},
            )
        assert response.status_code == 200
        data = await response.get_json()
        assert data["token"] == "token123"
        assert data["service_id"] == 1

    @pytest.mark.asyncio
    async def test_create_token_not_jwt_auth(self, test_app, test_client):
        """Service not using JWT -> 404"""
        service = _make_service_mock(auth_type="base64")
        test_app.db.services.__getitem__ = MagicMock(return_value=service)

        response = await test_client.post(
            "/api/v1/services/1/token",
            json={},
            headers={"Authorization": "Bearer admin-token"},
        )
        assert response.status_code == 404

    @pytest.mark.asyncio
    async def test_create_token_failed(self, test_app, test_client):
        """create_jwt_token returns None -> 500"""
        service = _make_service_mock(auth_type="jwt")
        test_app.db.services.__getitem__ = MagicMock(return_value=service)

        with patch(
            "models.service.ServiceModel.create_jwt_token", return_value=None
        ):
            response = await test_client.post(
                "/api/v1/services/1/token",
                json={},
                headers={"Authorization": "Bearer admin-token"},
            )
        assert response.status_code == 500

    @pytest.mark.asyncio
    async def test_create_token_exception(self, test_app, test_client):
        """Covers lines 306-308: exception -> 500"""
        service = _make_service_mock(auth_type="jwt")
        test_app.db.services.__getitem__ = MagicMock(return_value=service)

        with patch(
            "models.service.ServiceModel.create_jwt_token",
            side_effect=Exception("Token error"),
        ):
            response = await test_client.post(
                "/api/v1/services/1/token",
                json={},
                headers={"Authorization": "Bearer admin-token"},
            )
        assert response.status_code == 500
        data = await response.get_json()
        assert "error" in data


class TestAssignUserToService:
    """POST /api/v1/services/<service_id>/assign"""

    @pytest.mark.asyncio
    async def test_assign_user_success(self, test_app, test_client):
        """Assign user to service -> 200"""
        with patch(
            "models.service.UserServiceAssignmentModel.assign_user_to_service",
            return_value=True,
        ):
            response = await test_client.post(
                "/api/v1/services/1/assign",
                json={"user_id": 2},
                headers={"Authorization": "Bearer admin-token"},
            )
        assert response.status_code == 200
        data = await response.get_json()
        assert "message" in data

    @pytest.mark.asyncio
    async def test_assign_user_failed(self, test_app, test_client):
        """Assign returns False -> 500"""
        with patch(
            "models.service.UserServiceAssignmentModel.assign_user_to_service",
            return_value=False,
        ):
            response = await test_client.post(
                "/api/v1/services/1/assign",
                json={"user_id": 2},
                headers={"Authorization": "Bearer admin-token"},
            )
        assert response.status_code == 500

    @pytest.mark.asyncio
    async def test_assign_user_exception(self, test_app, test_client):
        """Covers lines 334-336: exception -> 500"""
        with patch(
            "models.service.UserServiceAssignmentModel.assign_user_to_service",
            side_effect=Exception("Assign error"),
        ):
            response = await test_client.post(
                "/api/v1/services/1/assign",
                json={"user_id": 2},
                headers={"Authorization": "Bearer admin-token"},
            )
        assert response.status_code == 500
        data = await response.get_json()
        assert "error" in data


class TestRemoveUserFromService:
    """DELETE /api/v1/services/<service_id>/unassign/<user_id>"""

    @pytest.mark.asyncio
    async def test_remove_user_success(self, test_app, test_client):
        """Remove user from service -> 200"""
        with patch(
            "models.service.UserServiceAssignmentModel.remove_user_from_service",
            return_value=True,
        ):
            response = await test_client.delete(
                "/api/v1/services/1/unassign/2",
                headers={"Authorization": "Bearer admin-token"},
            )
        assert response.status_code == 200
        data = await response.get_json()
        assert "message" in data

    @pytest.mark.asyncio
    async def test_remove_user_failed(self, test_app, test_client):
        """Remove returns False -> 500"""
        with patch(
            "models.service.UserServiceAssignmentModel.remove_user_from_service",
            return_value=False,
        ):
            response = await test_client.delete(
                "/api/v1/services/1/unassign/2",
                headers={"Authorization": "Bearer admin-token"},
            )
        assert response.status_code == 500

    @pytest.mark.asyncio
    async def test_remove_user_exception(self, test_app, test_client):
        """Covers lines 351-353: exception -> 500"""
        with patch(
            "models.service.UserServiceAssignmentModel.remove_user_from_service",
            side_effect=Exception("Remove error"),
        ):
            response = await test_client.delete(
                "/api/v1/services/1/unassign/2",
                headers={"Authorization": "Bearer admin-token"},
            )
        assert response.status_code == 500
        data = await response.get_json()
        assert "error" in data


# ===========================================================================
# mappings_bp tests
# ===========================================================================

class TestListMappings:
    """GET /api/v1/mappings"""

    @pytest.mark.asyncio
    async def test_list_mappings_with_results(self, test_app, test_client):
        """Covers line 46: result.append in loop"""
        with patch(
            "models.mapping.MappingModel.get_cluster_mappings",
            return_value=[
                {
                    "id": 1,
                    "name": "mapping-1",
                    "description": "Test mapping",
                    "source_services": [{"id": 1, "name": "svc-1"}],
                    "dest_services": [{"id": 2, "name": "svc-2"}],
                    "protocols": ["tcp"],
                    "ports": [{"port": 8080}],
                    "auth_required": False,
                    "priority": 100,
                    "created_at": datetime(2025, 1, 1),
                },
                {
                    "id": 2,
                    "name": "mapping-2",
                    "description": "Another mapping",
                    "source_services": [{"id": 3, "name": "svc-3"}],
                    "dest_services": [{"id": 4, "name": "svc-4"}],
                    "protocols": ["http"],
                    "ports": [{"port": 80}],
                    "auth_required": True,
                    "priority": 200,
                    "created_at": datetime(2025, 1, 1),
                },
            ],
        ):
            response = await test_client.get(
                "/api/v1/mappings?cluster_id=1",
                headers={"Authorization": "Bearer admin-token"},
            )
        assert response.status_code == 200
        data = await response.get_json()
        assert len(data["mappings"]) == 2
        assert data["mappings"][0]["name"] == "mapping-1"
        assert data["mappings"][1]["name"] == "mapping-2"

    @pytest.mark.asyncio
    async def test_list_mappings_no_cluster_id(self, test_app, test_client):
        """Missing cluster_id parameter -> 400"""
        response = await test_client.get(
            "/api/v1/mappings",
            headers={"Authorization": "Bearer admin-token"},
        )
        assert response.status_code == 400


class TestCreateMapping:
    """POST /api/v1/mappings"""

    @pytest.mark.asyncio
    async def test_create_mapping_success(self, test_app, test_client):
        """Covers line 106: return jsonify(response.dict()), 201"""
        mapping_mock = _make_mapping_mock(mapping_id=1)
        test_app.db.mappings.__getitem__ = MagicMock(return_value=mapping_mock)

        with patch("models.mapping.MappingModel.create_mapping", return_value=1):
            response = await test_client.post(
                "/api/v1/mappings",
                json={
                    "name": "test-mapping",
                    "source_services": ["service-1"],
                    "dest_services": ["service-2"],
                    "ports": ["8080"],
                    "cluster_id": 1,
                    "protocols": ["tcp"],
                    "auth_required": False,
                    "priority": 100,
                    "description": "Test mapping",
                    "comments": None,
                },
                headers={"Authorization": "Bearer admin-token"},
            )
        assert response.status_code == 201
        data = await response.get_json()
        assert data["id"] == 1
        assert "mapping" in data["name"].lower()

    @pytest.mark.asyncio
    async def test_create_mapping_validation_error(self, test_app, test_client):
        """Invalid input -> validation error"""
        response = await test_client.post(
            "/api/v1/mappings",
            json={"name": "test"},
            headers={"Authorization": "Bearer admin-token"},
        )
        assert response.status_code == 400

    @pytest.mark.asyncio
    async def test_create_mapping_value_error(self, test_app, test_client):
        """ValueError during create -> 400"""
        with patch(
            "models.mapping.MappingModel.create_mapping",
            side_effect=ValueError("Invalid port"),
        ):
            response = await test_client.post(
                "/api/v1/mappings",
                json={
                    "name": "test-mapping",
                    "source_services": ["service-1"],
                    "dest_services": ["service-2"],
                    "ports": ["8080"],
                    "cluster_id": 1,
                    "protocols": ["tcp"],
                    "auth_required": False,
                    "priority": 100,
                    "description": "Test",
                    "comments": None,
                },
                headers={"Authorization": "Bearer admin-token"},
            )
        assert response.status_code == 400

    @pytest.mark.asyncio
    async def test_create_mapping_exception(self, test_app, test_client):
        """Exception during create -> 500"""
        with patch(
            "models.mapping.MappingModel.create_mapping",
            side_effect=Exception("DB error"),
        ):
            response = await test_client.post(
                "/api/v1/mappings",
                json={
                    "name": "test-mapping",
                    "source_services": ["service-1"],
                    "dest_services": ["service-2"],
                    "ports": ["8080"],
                    "cluster_id": 1,
                    "protocols": ["tcp"],
                    "auth_required": False,
                    "priority": 100,
                    "description": "Test",
                    "comments": None,
                },
                headers={"Authorization": "Bearer admin-token"},
            )
        assert response.status_code == 500


class TestGetMappingDetail:
    """GET /api/v1/mappings/<mapping_id>"""

    @pytest.mark.asyncio
    async def test_get_mapping_success(self, test_app, test_client):
        """Covers line 144: return jsonify(response.dict()), 200"""
        mapping = _make_mapping_mock(mapping_id=1)
        test_app.db.mappings.__getitem__ = MagicMock(return_value=mapping)

        response = await test_client.get(
            "/api/v1/mappings/1",
            headers={"Authorization": "Bearer admin-token"},
        )
        assert response.status_code == 200
        data = await response.get_json()
        assert data["id"] == 1
        assert data["name"] == "mapping-1"

    @pytest.mark.asyncio
    async def test_get_mapping_not_found(self, test_app, test_client):
        """Mapping doesn't exist -> 404"""
        test_app.db.mappings.__getitem__ = MagicMock(return_value=None)

        response = await test_client.get(
            "/api/v1/mappings/999",
            headers={"Authorization": "Bearer admin-token"},
        )
        assert response.status_code == 404


class TestUpdateMapping:
    """PUT /api/v1/mappings/<mapping_id>"""

    @pytest.mark.asyncio
    async def test_update_mapping_all_fields(self, test_app, test_client):
        """Covers lines 155-186: update all fields -> 200"""
        mapping = _make_mapping_mock(mapping_id=1)
        auth_user = _make_auth_user_mock(user_id=1, is_admin=True)
        test_app.db.mappings.__getitem__ = MagicMock(return_value=mapping)
        test_app.db.auth_user.__getitem__ = MagicMock(return_value=auth_user)

        with patch(
            "models.mapping.MappingModel._normalize_service_list",
            return_value=["service-1", "service-2"],
        ), patch(
            "models.mapping.MappingModel._normalize_port_list",
            return_value=["8080", "9000"],
        ):
            response = await test_client.put(
                "/api/v1/mappings/1",
                json={
                    "name": "updated-mapping",
                    "description": "Updated description",
                    "source_services": ["service-1"],
                    "dest_services": ["service-2"],
                    "protocols": ["tcp", "http"],
                    "ports": ["8080", "80"],
                    "auth_required": True,
                    "priority": 50,
                    "comments": "Test comment",
                },
                headers={"Authorization": "Bearer admin-token"},
            )
        assert response.status_code == 200
        data = await response.get_json()
        assert data["id"] == 1

    @pytest.mark.asyncio
    async def test_update_mapping_partial_fields(self, test_app, test_client):
        """Update only some fields -> 200"""
        mapping = _make_mapping_mock(mapping_id=1)
        auth_user = _make_auth_user_mock(user_id=1, is_admin=True)
        test_app.db.mappings.__getitem__ = MagicMock(return_value=mapping)
        test_app.db.auth_user.__getitem__ = MagicMock(return_value=auth_user)

        response = await test_client.put(
            "/api/v1/mappings/1",
            json={
                "priority": 75,
                "auth_required": True,
            },
            headers={"Authorization": "Bearer admin-token"},
        )
        assert response.status_code == 200

    @pytest.mark.asyncio
    async def test_update_mapping_non_admin_forbidden(self, test_app, test_client):
        """Non-admin PUT -> 403"""
        mapping = _make_mapping_mock(mapping_id=1)
        auth_user = _make_auth_user_mock(user_id=2, is_admin=False)
        test_app.db.mappings.__getitem__ = MagicMock(return_value=mapping)
        test_app.db.auth_user.__getitem__ = MagicMock(return_value=auth_user)

        with patch(
            "middleware.auth._validate_token", new_callable=AsyncMock
        ) as mv:
            mv.return_value = _user_payload()
            response = await test_client.put(
                "/api/v1/mappings/1",
                json={"priority": 50},
                headers={"Authorization": "Bearer user-token"},
            )
        assert response.status_code == 403

    @pytest.mark.asyncio
    async def test_update_mapping_validation_error(self, test_app, test_client):
        """Invalid JSON -> validation error"""
        mapping = _make_mapping_mock(mapping_id=1)
        auth_user = _make_auth_user_mock(user_id=1, is_admin=True)
        test_app.db.mappings.__getitem__ = MagicMock(return_value=mapping)
        test_app.db.auth_user.__getitem__ = MagicMock(return_value=auth_user)

        response = await test_client.put(
            "/api/v1/mappings/1",
            json={"priority": "invalid"},
            headers={"Authorization": "Bearer admin-token"},
        )
        assert response.status_code == 400
