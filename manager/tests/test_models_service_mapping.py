"""
Comprehensive unit tests for models/service.py and models/mapping.py

Tests ServiceModel, UserServiceAssignmentModel, MappingModel, and their
Pydantic validation models.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from datetime import datetime
from typing import Any, Dict, List
from unittest.mock import MagicMock, patch

import jwt
import pytest

from models.service import (
    CreateServiceRequest,
    ServiceModel,
    SetServiceAuthRequest,
    UpdateServiceRequest,
    UserServiceAssignmentModel,
)
from models.mapping import (
    CreateMappingRequest,
    MappingModel,
)


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _make_service_record(
    service_id: int = 1,
    name: str = "test-service",
    ip_fqdn: str = "10.0.0.1",
    port: int = 8080,
    auth_type: str = "none",
    is_active: bool = True,
    cluster_id: int = 1,
    collection: str = None,
    jwt_secret: str = "secret",
    jwt_expiry: int = 3600,
    jwt_algorithm: str = "HS256",
    token_base64: str = None,
    tls_enabled: bool = False,
    tls_verify: bool = True,
    health_check_enabled: bool = False,
    health_check_path: str = None,
    health_check_interval: int = 30,
) -> MagicMock:
    svc = MagicMock()
    svc.id = service_id
    svc.name = name
    svc.ip_fqdn = ip_fqdn
    svc.port = port
    svc.auth_type = auth_type
    svc.is_active = is_active
    svc.cluster_id = cluster_id
    svc.collection = collection
    svc.jwt_secret = jwt_secret
    svc.jwt_expiry = jwt_expiry
    svc.jwt_algorithm = jwt_algorithm
    svc.token_base64 = token_base64
    svc.tls_enabled = tls_enabled
    svc.tls_verify = tls_verify
    svc.health_check_enabled = health_check_enabled
    svc.health_check_path = health_check_path
    svc.health_check_interval = health_check_interval
    svc.protocol = "tcp"
    svc.created_at = datetime(2025, 1, 1)
    svc.update_record = MagicMock()
    return svc


class _FakeSelectResult:
    """Reusable fake that supports len(), iter(), and first() like PyDAL rows."""
    def __init__(self, items, first_item=None):
        self._items = list(items) if items else []
        self._first = first_item if first_item is not None else (self._items[0] if self._items else None)
    def __iter__(self):
        return iter(self._items)
    def __len__(self):
        return len(self._items)
    def first(self):
        return self._first


def _make_db(service_record=None, services_list=None):
    """Create a mock db configured for service tests."""
    db = MagicMock(name="db")

    # db.services[id] -> service record
    db.services.__getitem__ = MagicMock(return_value=service_record)

    # db.services.insert() -> 1
    db.services.insert = MagicMock(return_value=1)

    # db(query).select() -> list — use db.return_value, not db.__call__
    query_mock = MagicMock(name="query")
    select_result_instance = _FakeSelectResult(services_list or [], first_item=service_record)
    query_mock.select = MagicMock(return_value=select_result_instance)
    query_mock.count = MagicMock(return_value=len(services_list or []))
    query_mock.update = MagicMock(return_value=1)
    db.return_value = query_mock

    # users table
    db.users.__getitem__ = MagicMock(return_value=None)
    db.user_service_assignments = MagicMock()

    return db, query_mock, select_result_instance


# ===========================================================================
# ServiceModel Tests
# ===========================================================================

class TestServiceModelCreateService:
    """Tests for ServiceModel.create_service"""

    def test_create_service_basic(self):
        db, _, _ = _make_db()
        service_id = ServiceModel.create_service(
            db=db,
            name="my-service",
            ip_fqdn="192.168.1.1",
            port=8080,
            cluster_id=1,
            created_by=1,
        )
        assert service_id == 1
        db.services.insert.assert_called_once()
        call_kwargs = db.services.insert.call_args[1]
        assert call_kwargs["name"] == "my-service"
        assert call_kwargs["ip_fqdn"] == "192.168.1.1"
        assert call_kwargs["port"] == 8080

    def test_create_service_with_all_params(self):
        db, _, _ = _make_db()
        ServiceModel.create_service(
            db=db,
            name="full-service",
            ip_fqdn="10.0.0.5",
            port=443,
            cluster_id=2,
            created_by=3,
            protocol="https",
            collection="web",
            auth_type="jwt",
            tls_enabled=True,
            tls_verify=False,
        )
        call_kwargs = db.services.insert.call_args[1]
        assert call_kwargs["protocol"] == "https"
        assert call_kwargs["collection"] == "web"
        assert call_kwargs["auth_type"] == "jwt"
        assert call_kwargs["tls_enabled"] is True
        assert call_kwargs["tls_verify"] is False


class TestServiceModelTokenGeneration:
    """Tests for generate_base64_token and generate_jwt_secret."""

    def test_generate_base64_token_is_string(self):
        token = ServiceModel.generate_base64_token()
        assert isinstance(token, str)
        assert len(token) > 0

    def test_generate_base64_token_is_unique(self):
        t1 = ServiceModel.generate_base64_token()
        t2 = ServiceModel.generate_base64_token()
        assert t1 != t2

    def test_generate_jwt_secret_is_string(self):
        secret = ServiceModel.generate_jwt_secret()
        assert isinstance(secret, str)
        assert len(secret) > 0

    def test_generate_jwt_secret_is_unique(self):
        s1 = ServiceModel.generate_jwt_secret()
        s2 = ServiceModel.generate_jwt_secret()
        assert s1 != s2


class TestServiceModelSetBase64Auth:
    """Tests for set_base64_auth."""

    def test_returns_token_on_success(self):
        svc = _make_service_record(service_id=5)
        db, _, _ = _make_db(service_record=svc)
        result = ServiceModel.set_base64_auth(db, 5)
        assert result is not None
        assert isinstance(result, str)
        svc.update_record.assert_called_once()
        call_kwargs = svc.update_record.call_args[1]
        assert call_kwargs["auth_type"] == "base64"
        assert call_kwargs["jwt_secret"] is None

    def test_returns_none_when_service_not_found(self):
        db, _, _ = _make_db(service_record=None)
        result = ServiceModel.set_base64_auth(db, 999)
        assert result is None


class TestServiceModelSetJwtAuth:
    """Tests for set_jwt_auth."""

    def test_returns_secret_on_success(self):
        svc = _make_service_record(service_id=3)
        db, _, _ = _make_db(service_record=svc)
        result = ServiceModel.set_jwt_auth(db, 3, expiry_seconds=7200, algorithm="HS512")
        assert result is not None
        assert isinstance(result, str)
        call_kwargs = svc.update_record.call_args[1]
        assert call_kwargs["auth_type"] == "jwt"
        assert call_kwargs["jwt_expiry"] == 7200
        assert call_kwargs["jwt_algorithm"] == "HS512"
        assert call_kwargs["token_base64"] is None

    def test_returns_none_when_service_not_found(self):
        db, _, _ = _make_db(service_record=None)
        result = ServiceModel.set_jwt_auth(db, 999)
        assert result is None


class TestServiceModelRotateJwtSecret:
    """Tests for rotate_jwt_secret."""

    def test_rotates_secret_successfully(self):
        svc = _make_service_record(auth_type="jwt")
        db, _, _ = _make_db(service_record=svc)
        new_secret = ServiceModel.rotate_jwt_secret(db, 1)
        assert new_secret is not None
        assert isinstance(new_secret, str)
        svc.update_record.assert_called_once()

    def test_returns_none_when_service_not_found(self):
        db, _, _ = _make_db(service_record=None)
        result = ServiceModel.rotate_jwt_secret(db, 999)
        assert result is None

    def test_returns_none_when_auth_type_not_jwt(self):
        svc = _make_service_record(auth_type="base64")
        db, _, _ = _make_db(service_record=svc)
        result = ServiceModel.rotate_jwt_secret(db, 1)
        assert result is None


class TestServiceModelValidateServiceToken:
    """Tests for validate_service_token."""

    def test_returns_false_when_service_not_found(self):
        db, _, _ = _make_db(service_record=None)
        assert ServiceModel.validate_service_token(db, 999, "tok") is False

    def test_returns_false_when_service_inactive(self):
        svc = _make_service_record(is_active=False)
        db, _, _ = _make_db(service_record=svc)
        assert ServiceModel.validate_service_token(db, 1, "tok") is False

    def test_validates_base64_token_correctly(self):
        svc = _make_service_record(auth_type="base64", token_base64="correct-token")
        db, _, _ = _make_db(service_record=svc)
        assert ServiceModel.validate_service_token(db, 1, "correct-token") is True
        assert ServiceModel.validate_service_token(db, 1, "wrong-token") is False

    def test_validates_jwt_token_correctly(self):
        secret = "test-jwt-secret"
        svc = _make_service_record(auth_type="jwt", jwt_secret=secret, jwt_algorithm="HS256")
        db, _, _ = _make_db(service_record=svc)
        # Create a valid JWT for service_id=1
        payload = {
            "service_id": 1,
            "iat": datetime.utcnow(),
            "exp": datetime.utcnow().timestamp() + 3600,
        }
        import time
        payload_with_exp = {
            "service_id": 1,
            "exp": int(time.time()) + 3600,
        }
        token = jwt.encode(payload_with_exp, secret, algorithm="HS256")
        assert ServiceModel.validate_service_token(db, 1, token) is True

    def test_rejects_invalid_jwt_token(self):
        svc = _make_service_record(auth_type="jwt", jwt_secret="secret")
        db, _, _ = _make_db(service_record=svc)
        assert ServiceModel.validate_service_token(db, 1, "not-a-jwt") is False

    def test_auth_type_none_returns_true(self):
        svc = _make_service_record(auth_type="none")
        db, _, _ = _make_db(service_record=svc)
        assert ServiceModel.validate_service_token(db, 1, "anything") is True


class TestServiceModelCreateJwtToken:
    """Tests for create_jwt_token."""

    def test_creates_valid_jwt(self):
        secret = "my-jwt-secret"
        svc = _make_service_record(
            service_id=2, name="svc", auth_type="jwt",
            jwt_secret=secret, jwt_expiry=3600, jwt_algorithm="HS256"
        )
        db, _, _ = _make_db(service_record=svc)
        token = ServiceModel.create_jwt_token(db, 2)
        assert token is not None
        decoded = jwt.decode(token, secret, algorithms=["HS256"])
        assert decoded["service_id"] == 2
        assert decoded["service_name"] == "svc"
        assert decoded["iss"] == "marchproxy"

    def test_includes_additional_claims(self):
        secret = "my-secret"
        svc = _make_service_record(
            service_id=3, auth_type="jwt",
            jwt_secret=secret, jwt_expiry=3600, jwt_algorithm="HS256"
        )
        db, _, _ = _make_db(service_record=svc)
        token = ServiceModel.create_jwt_token(db, 3, additional_claims={"custom": "value"})
        decoded = jwt.decode(token, secret, algorithms=["HS256"])
        assert decoded["custom"] == "value"

    def test_returns_none_when_service_not_found(self):
        db, _, _ = _make_db(service_record=None)
        assert ServiceModel.create_jwt_token(db, 999) is None

    def test_returns_none_when_auth_type_not_jwt(self):
        svc = _make_service_record(auth_type="base64")
        db, _, _ = _make_db(service_record=svc)
        assert ServiceModel.create_jwt_token(db, 1) is None


class TestServiceModelGetClusterServices:
    """Tests for get_cluster_services."""

    def test_returns_all_services_for_admin_user(self):
        svc = _make_service_record()
        db, query_mock, select_result = _make_db(service_record=svc, services_list=[svc])
        admin_user = MagicMock()
        admin_user.get = MagicMock(side_effect=lambda k, d=None: True if k == "is_admin" else d)
        db.users.__getitem__ = MagicMock(return_value=admin_user)
        result = ServiceModel.get_cluster_services(db, cluster_id=1, user_id=1)
        assert isinstance(result, list)

    def test_returns_services_without_user_filter(self):
        svc = _make_service_record()
        db, _, select_result = _make_db(service_record=svc, services_list=[svc])
        result = ServiceModel.get_cluster_services(db, cluster_id=1)
        assert isinstance(result, list)

    def test_filters_by_assignments_for_non_admin(self):
        svc = _make_service_record()
        regular_user = MagicMock()
        regular_user.get = MagicMock(return_value=False)
        db, query_mock, select_result = _make_db(service_record=svc, services_list=[svc])
        db.users.__getitem__ = MagicMock(return_value=regular_user)
        result = ServiceModel.get_cluster_services(db, cluster_id=1, user_id=2)
        assert isinstance(result, list)


class TestServiceModelGetServiceConfig:
    """Tests for get_service_config."""

    def _make_config_db(self, service_record):
        """Helper that produces a db where db(query).select().first() returns service_record."""
        db = MagicMock(name="db")
        query_mock = MagicMock()
        items = [service_record] if service_record else []
        query_mock.select = MagicMock(return_value=_FakeSelectResult(items, first_item=service_record))
        db.return_value = query_mock
        db.services = MagicMock()
        return db

    def test_returns_none_when_not_found(self):
        db = self._make_config_db(None)
        result = ServiceModel.get_service_config(db, 999)
        assert result is None

    def test_returns_basic_config(self):
        svc = _make_service_record(auth_type="none")
        db = self._make_config_db(svc)
        result = ServiceModel.get_service_config(db, 1)
        assert result is not None
        assert result["id"] == 1
        assert result["auth_type"] == "none"
        assert "token_base64" not in result
        assert "jwt_secret" not in result

    def test_includes_base64_token_in_config(self):
        svc = _make_service_record(auth_type="base64", token_base64="my-b64-token")
        db = self._make_config_db(svc)
        result = ServiceModel.get_service_config(db, 1)
        assert result["token_base64"] == "my-b64-token"

    def test_includes_jwt_config(self):
        svc = _make_service_record(
            auth_type="jwt", jwt_secret="my-secret",
            jwt_expiry=7200, jwt_algorithm="HS512"
        )
        db = self._make_config_db(svc)
        result = ServiceModel.get_service_config(db, 1)
        assert result["jwt_secret"] == "my-secret"
        assert result["jwt_expiry"] == 7200
        assert result["jwt_algorithm"] == "HS512"

    def test_includes_health_check_config_when_enabled(self):
        svc = _make_service_record(
            health_check_enabled=True,
            health_check_path="/health",
            health_check_interval=60,
        )
        db = self._make_config_db(svc)
        result = ServiceModel.get_service_config(db, 1)
        assert result["health_check_path"] == "/health"
        assert result["health_check_interval"] == 60


# ===========================================================================
# UserServiceAssignmentModel Tests
# ===========================================================================

class TestUserServiceAssignmentModel:
    """Tests for UserServiceAssignmentModel."""

    def _make_assignment_db(self, existing=None, user_is_admin=False):
        db = MagicMock(name="db")
        query_mock = MagicMock()
        items = [existing] if existing else []
        select_result = _FakeSelectResult(items, first_item=existing)
        query_mock.select = MagicMock(return_value=select_result)
        query_mock.update = MagicMock(return_value=1)
        db.return_value = query_mock

        user_mock = MagicMock()
        user_mock.get = MagicMock(return_value=user_is_admin)
        db.users = MagicMock()
        db.users.__getitem__ = MagicMock(return_value=user_mock if user_is_admin else None)
        db.user_service_assignments = MagicMock()
        db.user_service_assignments.insert = MagicMock(return_value=1)
        db.services = MagicMock()
        return db, query_mock, select_result

    def test_assign_user_returns_true_when_existing(self):
        existing = MagicMock()
        db, _, _ = self._make_assignment_db(existing=existing)
        result = UserServiceAssignmentModel.assign_user_to_service(db, 1, 2, 3)
        assert result is True
        db.user_service_assignments.insert.assert_not_called()

    def test_assign_user_inserts_when_no_existing(self):
        db, _, _ = self._make_assignment_db(existing=None)
        result = UserServiceAssignmentModel.assign_user_to_service(db, 1, 2, 3)
        assert result is True
        db.user_service_assignments.insert.assert_called_once()

    def test_remove_user_from_service(self):
        db, query_mock, _ = self._make_assignment_db()
        query_mock.update = MagicMock(return_value=1)
        result = UserServiceAssignmentModel.remove_user_from_service(db, 1, 2)
        assert result is True  # update returns 1 > 0

    def test_check_access_admin_user(self):
        db, _, _ = self._make_assignment_db(user_is_admin=True)
        user_mock = MagicMock()
        user_mock.get = MagicMock(return_value=True)
        db.users.__getitem__ = MagicMock(return_value=user_mock)
        result = UserServiceAssignmentModel.check_user_service_access(db, 1, 2)
        assert result is True

    def test_check_access_with_assignment(self):
        existing = MagicMock()
        db, _, _ = self._make_assignment_db(existing=existing)
        db.users.__getitem__ = MagicMock(return_value=None)
        result = UserServiceAssignmentModel.check_user_service_access(db, 2, 3)
        assert result is True

    def test_check_access_no_assignment(self):
        db, _, _ = self._make_assignment_db(existing=None)
        db.users.__getitem__ = MagicMock(return_value=None)
        result = UserServiceAssignmentModel.check_user_service_access(db, 2, 3)
        assert result is False


# ===========================================================================
# MappingModel._normalize_port_list Tests
# ===========================================================================

class TestNormalizePortList:
    """Tests for MappingModel._normalize_port_list — pure logic, no DB needed."""

    def test_single_int_port(self):
        result = MappingModel._normalize_port_list([80])
        assert result == [{"type": "single", "port": 80}]

    def test_single_string_port(self):
        result = MappingModel._normalize_port_list(["8080"])
        assert result == [{"type": "single", "port": 8080}]

    def test_port_range(self):
        result = MappingModel._normalize_port_list(["8000-8100"])
        assert result == [{"type": "range", "start": 8000, "end": 8100}]

    def test_comma_separated_ports(self):
        result = MappingModel._normalize_port_list(["80,443,8080"])
        assert result == [{"type": "list", "ports": [80, 443, 8080]}]

    def test_invalid_port_out_of_range(self):
        result = MappingModel._normalize_port_list([0])
        assert result == []

    def test_invalid_port_too_large(self):
        result = MappingModel._normalize_port_list([65536])
        assert result == []

    def test_invalid_range_bad_format(self):
        result = MappingModel._normalize_port_list(["abc-def"])
        assert result == []

    def test_invalid_range_start_gt_end(self):
        result = MappingModel._normalize_port_list(["9000-8000"])
        assert result == []

    def test_mixed_port_types(self):
        result = MappingModel._normalize_port_list([80, "8000-8010", "443,8443"])
        assert len(result) == 3
        types = [r["type"] for r in result]
        assert "single" in types
        assert "range" in types
        assert "list" in types

    def test_invalid_string_port(self):
        result = MappingModel._normalize_port_list(["notaport"])
        assert result == []

    def test_invalid_comma_list_with_bad_value(self):
        result = MappingModel._normalize_port_list(["80,abc"])
        assert result == []

    def test_empty_list(self):
        result = MappingModel._normalize_port_list([])
        assert result == []


# ===========================================================================
# MappingModel._normalize_service_list Tests
# ===========================================================================

class TestNormalizeServiceList:
    """Tests for MappingModel._normalize_service_list."""

    def _make_mapping_db(self, services=None, first_service=None):
        db = MagicMock(name="db")
        query_mock = MagicMock()
        svc_list = services or []
        select_result = _FakeSelectResult(svc_list, first_item=first_service)
        query_mock.select = MagicMock(return_value=select_result)
        db.return_value = query_mock
        db.services = MagicMock()
        return db, select_result

    def test_all_services_keyword(self):
        svc1 = MagicMock()
        svc2 = MagicMock()
        db, _ = self._make_mapping_db(services=[svc1, svc2])
        result = MappingModel._normalize_service_list(db, ["all"], cluster_id=1)
        assert len(result) == 1
        assert result[0]["type"] == "all"
        assert result[0]["count"] == 2

    def test_all_breaks_after_first(self):
        db, _ = self._make_mapping_db(services=[MagicMock()])
        # "all" + more items; only "all" entry should exist
        result = MappingModel._normalize_service_list(db, ["all", 5], cluster_id=1)
        assert len(result) == 1
        assert result[0]["type"] == "all"

    def test_collection_reference(self):
        svc = MagicMock()
        svc.id = 10
        db, select_result = self._make_mapping_db(services=[svc])
        result = MappingModel._normalize_service_list(db, ["collection:webservers"], cluster_id=1)
        assert len(result) == 1
        assert result[0]["type"] == "collection"
        assert result[0]["name"] == "webservers"
        assert 10 in result[0]["service_ids"]

    def test_collection_empty_skips_entry(self):
        db, _ = self._make_mapping_db(services=[])
        result = MappingModel._normalize_service_list(db, ["collection:empty"], cluster_id=1)
        assert result == []

    def test_individual_service_id_int(self):
        svc = MagicMock()
        svc.name = "my-svc"
        svc.ip_fqdn = "10.0.0.1"
        svc.port = 80
        db, select_result = self._make_mapping_db(first_service=svc)
        result = MappingModel._normalize_service_list(db, [5], cluster_id=1)
        assert len(result) == 1
        assert result[0]["type"] == "service"
        assert result[0]["id"] == 5

    def test_individual_service_id_string_digit(self):
        svc = MagicMock()
        svc.name = "str-svc"
        svc.ip_fqdn = "10.0.0.2"
        svc.port = 443
        db, select_result = self._make_mapping_db(first_service=svc)
        result = MappingModel._normalize_service_list(db, ["7"], cluster_id=1)
        assert len(result) == 1
        assert result[0]["id"] == 7

    def test_individual_service_not_found_skips(self):
        db, select_result = self._make_mapping_db(first_service=None)
        result = MappingModel._normalize_service_list(db, [42], cluster_id=1)
        assert result == []


# ===========================================================================
# MappingModel.create_mapping Tests
# ===========================================================================

class TestMappingModelCreateMapping:
    """Tests for MappingModel.create_mapping."""

    def _make_mapping_create_db(self, source_normalized=True, dest_normalized=True):
        db = MagicMock(name="db")
        query_mock = MagicMock()
        svc = MagicMock()
        svc.name = "svc"
        svc.ip_fqdn = "10.0.0.1"
        svc.port = 80
        items = [svc] if source_normalized else []
        select_result = _FakeSelectResult(items, first_item=svc if source_normalized else None)
        query_mock.select = MagicMock(return_value=select_result)
        db.return_value = query_mock
        db.services = MagicMock()
        db.mappings = MagicMock()
        db.mappings.insert = MagicMock(return_value=42)
        return db

    def test_create_mapping_success(self):
        db = self._make_mapping_create_db()
        mapping_id = MappingModel.create_mapping(
            db=db,
            name="test-mapping",
            source_services=[1],
            dest_services=[2],
            ports=[80],
            cluster_id=1,
            created_by=1,
        )
        assert mapping_id == 42
        db.mappings.insert.assert_called_once()

    def test_create_mapping_default_protocols(self):
        db = self._make_mapping_create_db()
        MappingModel.create_mapping(
            db=db,
            name="proto-mapping",
            source_services=[1],
            dest_services=[2],
            ports=[80],
            cluster_id=1,
            created_by=1,
        )
        call_kwargs = db.mappings.insert.call_args[1]
        assert call_kwargs["protocols"] == ["tcp"]

    def test_create_mapping_raises_on_empty_sources(self):
        db = self._make_mapping_create_db(source_normalized=False)
        with pytest.raises(ValueError, match="No valid source services"):
            MappingModel.create_mapping(
                db=db,
                name="fail-mapping",
                source_services=[999],
                dest_services=[1],
                ports=[80],
                cluster_id=1,
                created_by=1,
            )

    def test_create_mapping_raises_on_empty_ports(self):
        db = self._make_mapping_create_db()
        with pytest.raises(ValueError, match="No valid ports"):
            MappingModel.create_mapping(
                db=db,
                name="no-ports",
                source_services=[1],
                dest_services=[2],
                ports=[0],  # invalid port out of range
                cluster_id=1,
                created_by=1,
            )


# ===========================================================================
# MappingModel._service_matches Tests
# ===========================================================================

class TestServiceMatches:
    """Tests for MappingModel._service_matches."""

    def test_matches_all_type(self):
        db = MagicMock()
        result = MappingModel._service_matches(db, 5, [{"type": "all"}], 1)
        assert result is True

    def test_matches_service_type_by_id(self):
        db = MagicMock()
        result = MappingModel._service_matches(
            db, 5, [{"type": "service", "id": 5}], 1
        )
        assert result is True

    def test_no_match_different_service_id(self):
        db = MagicMock()
        result = MappingModel._service_matches(
            db, 5, [{"type": "service", "id": 99}], 1
        )
        assert result is False

    def test_matches_collection_type(self):
        db = MagicMock()
        svc = MagicMock()
        svc.collection = "web"
        db.services.__getitem__ = MagicMock(return_value=svc)
        result = MappingModel._service_matches(
            db, 5, [{"type": "collection", "name": "web"}], 1
        )
        assert result is True

    def test_no_match_collection_mismatch(self):
        db = MagicMock()
        svc = MagicMock()
        svc.collection = "other"
        db.services.__getitem__ = MagicMock(return_value=svc)
        result = MappingModel._service_matches(
            db, 5, [{"type": "collection", "name": "web"}], 1
        )
        assert result is False

    def test_empty_refs_returns_false(self):
        db = MagicMock()
        result = MappingModel._service_matches(db, 5, [], 1)
        assert result is False


# ===========================================================================
# MappingModel._port_matches Tests
# ===========================================================================

class TestPortMatches:
    """Tests for MappingModel._port_matches."""

    def test_matches_single(self):
        assert MappingModel._port_matches(80, [{"type": "single", "port": 80}]) is True

    def test_no_match_single(self):
        assert MappingModel._port_matches(443, [{"type": "single", "port": 80}]) is False

    def test_matches_range(self):
        assert MappingModel._port_matches(
            8050, [{"type": "range", "start": 8000, "end": 8100}]
        ) is True

    def test_no_match_range(self):
        assert MappingModel._port_matches(
            9000, [{"type": "range", "start": 8000, "end": 8100}]
        ) is False

    def test_matches_list(self):
        assert MappingModel._port_matches(
            443, [{"type": "list", "ports": [80, 443, 8080]}]
        ) is True

    def test_no_match_list(self):
        assert MappingModel._port_matches(
            9999, [{"type": "list", "ports": [80, 443, 8080]}]
        ) is False

    def test_empty_refs_returns_false(self):
        assert MappingModel._port_matches(80, []) is False


# ===========================================================================
# MappingModel.get_cluster_mappings Tests
# ===========================================================================

class TestGetClusterMappings:
    """Tests for MappingModel.get_cluster_mappings."""

    def _make_mappings_db(self, mappings_list=None, user_record=None, user_services=None):
        db = MagicMock(name="db")

        mapping_mocks = []
        for m in (mappings_list or []):
            mm = MagicMock()
            mm.id = m.get("id", 1)
            mm.name = m.get("name", "map")
            mm.description = m.get("description", None)
            mm.source_services = m.get("source_services", [])
            mm.dest_services = m.get("dest_services", [])
            mm.protocols = m.get("protocols", ["tcp"])
            mm.ports = m.get("ports", [])
            mm.auth_required = m.get("auth_required", True)
            mm.priority = m.get("priority", 100)
            mm.created_at = m.get("created_at", datetime(2025, 1, 1))
            mapping_mocks.append(mm)

        query_mock = MagicMock()
        select_result = _FakeSelectResult(mapping_mocks)
        query_mock.select = MagicMock(return_value=select_result)
        db.return_value = query_mock

        db.mappings = MagicMock()
        db.auth_user = MagicMock()
        if user_record is not None:
            db.auth_user.__getitem__ = MagicMock(return_value=user_record)
        else:
            db.auth_user.__getitem__ = MagicMock(return_value=None)

        db.user_service_assignments = MagicMock()

        return db, query_mock

    def test_returns_all_for_admin(self):
        user = MagicMock()
        user.get = MagicMock(return_value=True)  # is_admin
        db, _ = self._make_mappings_db(
            mappings_list=[{"id": 1, "name": "m1"}],
            user_record=user,
        )
        result = MappingModel.get_cluster_mappings(db, cluster_id=1, user_id=1)
        assert isinstance(result, list)

    def test_returns_all_without_user_filter(self):
        db, _ = self._make_mappings_db(mappings_list=[{"id": 1, "name": "m1"}])
        result = MappingModel.get_cluster_mappings(db, cluster_id=1)
        assert isinstance(result, list)
        assert len(result) == 1

    def test_filters_non_admin_user_by_services(self):
        # Non-admin user with access to service 10
        user = MagicMock()
        user.get = MagicMock(return_value=False)  # not admin

        # A mapping where source_services includes service 10
        accessible_mapping = {
            "id": 1,
            "name": "accessible",
            "source_services": [{"type": "service", "id": 10}],
            "dest_services": [],
        }
        inaccessible_mapping = {
            "id": 2,
            "name": "inaccessible",
            "source_services": [{"type": "service", "id": 99}],
            "dest_services": [{"type": "service", "id": 88}],
        }

        db = MagicMock(name="db")
        db.auth_user = MagicMock()
        db.auth_user.__getitem__ = MagicMock(return_value=user)

        # user_service_assignments returns service_id=10
        assignment = MagicMock()
        assignment.service_id = 10
        ua_select = _FakeSelectResult([assignment])
        ua_query = MagicMock()
        ua_query.select = MagicMock(return_value=ua_select)

        m1 = MagicMock()
        m1.id = 1
        m1.name = "accessible"
        m1.description = None
        m1.source_services = [{"type": "service", "id": 10}]
        m1.dest_services = []
        m1.protocols = ["tcp"]
        m1.ports = []
        m1.auth_required = True
        m1.priority = 100
        m1.created_at = datetime(2025, 1, 1)

        m2 = MagicMock()
        m2.id = 2
        m2.name = "inaccessible"
        m2.description = None
        m2.source_services = [{"type": "service", "id": 99}]
        m2.dest_services = [{"type": "service", "id": 88}]
        m2.protocols = ["tcp"]
        m2.ports = []
        m2.auth_required = True
        m2.priority = 100
        m2.created_at = datetime(2025, 1, 1)

        all_select = _FakeSelectResult([m1, m2])

        all_query = MagicMock()
        all_query.select = MagicMock(return_value=all_select)

        call_count = [0]

        def db_call(*args, **kwargs):
            call_count[0] += 1
            if call_count[0] == 1:
                return ua_query  # user_service_assignments query
            return all_query  # mappings query

        # Use side_effect on the MagicMock itself (not __call__)
        db.side_effect = db_call
        db.mappings = MagicMock()
        db.user_service_assignments = MagicMock()

        result = MappingModel.get_cluster_mappings(db, cluster_id=1, user_id=5)
        # Should include accessible (id=1) but not inaccessible (id=2)
        ids = [r["id"] for r in result]
        assert 1 in ids
        assert 2 not in ids


# ===========================================================================
# Pydantic Model Validation Tests
# ===========================================================================

class TestCreateServiceRequestValidation:
    """Pydantic validation for CreateServiceRequest."""

    def test_valid_request(self):
        req = CreateServiceRequest(
            name="my-service",
            ip_fqdn="10.0.0.1",
            port=8080,
            cluster_id=1,
        )
        assert req.name == "my-service"

    def test_name_too_short(self):
        with pytest.raises(Exception):
            CreateServiceRequest(name="ab", ip_fqdn="x", port=80, cluster_id=1)

    def test_name_invalid_chars(self):
        with pytest.raises(Exception):
            CreateServiceRequest(name="my service!", ip_fqdn="x", port=80, cluster_id=1)

    def test_port_out_of_range(self):
        with pytest.raises(Exception):
            CreateServiceRequest(name="svc", ip_fqdn="x", port=0, cluster_id=1)

    def test_invalid_protocol(self):
        with pytest.raises(Exception):
            CreateServiceRequest(
                name="svc", ip_fqdn="x", port=80, cluster_id=1, protocol="ftp"
            )

    def test_invalid_auth_type(self):
        with pytest.raises(Exception):
            CreateServiceRequest(
                name="svc", ip_fqdn="x", port=80, cluster_id=1, auth_type="oauth"
            )

    def test_name_normalized_to_lower(self):
        req = CreateServiceRequest(name="My-Service", ip_fqdn="x", port=80, cluster_id=1)
        assert req.name == "my-service"


class TestSetServiceAuthRequestValidation:
    """Pydantic validation for SetServiceAuthRequest."""

    def test_valid_jwt_auth(self):
        req = SetServiceAuthRequest(auth_type="jwt", jwt_algorithm="HS256")
        assert req.auth_type == "jwt"

    def test_invalid_auth_type(self):
        with pytest.raises(Exception):
            SetServiceAuthRequest(auth_type="oauth")

    def test_invalid_jwt_algorithm(self):
        with pytest.raises(Exception):
            SetServiceAuthRequest(auth_type="jwt", jwt_algorithm="RS256")


class TestCreateMappingRequestValidation:
    """Pydantic validation for CreateMappingRequest."""

    def test_valid_request(self):
        req = CreateMappingRequest(
            name="my-mapping",
            source_services=[1],
            dest_services=[2],
            cluster_id=1,
            ports=[80],
        )
        assert req.name == "my-mapping"

    def test_name_too_short(self):
        with pytest.raises(Exception):
            CreateMappingRequest(
                name="ab", source_services=[1], dest_services=[2],
                cluster_id=1, ports=[80]
            )

    def test_invalid_protocol(self):
        with pytest.raises(Exception):
            CreateMappingRequest(
                name="my-map", source_services=[1], dest_services=[2],
                cluster_id=1, ports=[80], protocols=["ftp"]
            )

    def test_empty_ports(self):
        with pytest.raises(Exception):
            CreateMappingRequest(
                name="my-map", source_services=[1], dest_services=[2],
                cluster_id=1, ports=[]
            )

    def test_invalid_priority(self):
        with pytest.raises(Exception):
            CreateMappingRequest(
                name="my-map", source_services=[1], dest_services=[2],
                cluster_id=1, ports=[80], priority=0
            )

    def test_port_range_validation(self):
        req = CreateMappingRequest(
            name="my-map", source_services=[1], dest_services=[2],
            cluster_id=1, ports=["8000-9000"]
        )
        assert "8000-9000" in req.ports

    def test_port_list_validation(self):
        req = CreateMappingRequest(
            name="my-map", source_services=[1], dest_services=[2],
            cluster_id=1, ports=["80,443"]
        )
        assert "80,443" in req.ports

    def test_name_normalized_to_lower(self):
        req = CreateMappingRequest(
            name="My-Mapping", source_services=[1], dest_services=[2],
            cluster_id=1, ports=[80]
        )
        assert req.name == "my-mapping"
