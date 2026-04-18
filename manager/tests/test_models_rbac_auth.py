"""
Tests for models/rbac.py and models/auth.py

Covers RBACModel, JWTManager, SessionModel, APITokenModel, UserModel,
and related Pydantic validation models.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from datetime import datetime, timedelta
from typing import Any
from unittest.mock import MagicMock, patch

import pytest

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


class _FakeSelectResult:
    """Minimal PyDAL select-result mock."""

    def __init__(self, items=None):
        self._items = items or []

    def first(self):
        return self._items[0] if self._items else None

    def __iter__(self):
        return iter(self._items)

    def __len__(self):
        return len(self._items)


def _make_db():
    db = MagicMock(name="db")
    query_mock = MagicMock(name="db_query")
    select_result = _FakeSelectResult()
    query_mock.select = MagicMock(return_value=select_result)
    query_mock.count = MagicMock(return_value=0)
    query_mock.update = MagicMock(return_value=1)
    query_mock.delete = MagicMock(return_value=0)
    db.return_value = query_mock
    # Table mocks
    for tname in ["roles", "user_roles", "user_permissions_cache", "users",
                  "sessions", "api_tokens", "clusters", "services"]:
        tbl = MagicMock(name=f"db.{tname}")
        tbl.insert = MagicMock(return_value=1)
        tbl.__getitem__ = MagicMock(return_value=None)
        setattr(db, tname, tbl)

    # Patch datetime comparison fields so `field > datetime(...)` works
    def _patch_datetime_field(field_mock):
        field_mock.__gt__ = MagicMock(return_value=MagicMock())
        field_mock.__lt__ = MagicMock(return_value=MagicMock())
        field_mock.__ge__ = MagicMock(return_value=MagicMock())
        field_mock.__le__ = MagicMock(return_value=MagicMock())

    _patch_datetime_field(db.sessions.expires_at)
    _patch_datetime_field(db.api_tokens.expires_at)

    db.commit = MagicMock()
    db.rollback = MagicMock()
    return db


# ===========================================================================
# UserModel tests
# ===========================================================================

class TestUserModel:

    def test_hash_password_returns_string(self):
        from models.auth import UserModel
        result = UserModel.hash_password("mypassword123")
        assert isinstance(result, str)
        assert len(result) > 20

    def test_verify_password_correct(self):
        from models.auth import UserModel
        pw = "correct-horse-battery-staple"
        h = UserModel.hash_password(pw)
        assert UserModel.verify_password(pw, h) is True

    def test_verify_password_wrong(self):
        from models.auth import UserModel
        h = UserModel.hash_password("correct")
        assert UserModel.verify_password("wrong", h) is False

    def test_generate_totp_secret_length(self):
        from models.auth import UserModel
        secret = UserModel.generate_totp_secret()
        assert len(secret) >= 16

    def test_verify_totp_with_valid_token(self):
        from models.auth import UserModel
        import pyotp
        secret = pyotp.random_base32()
        token = pyotp.TOTP(secret).now()
        assert UserModel.verify_totp(secret, token) is True

    def test_verify_totp_with_invalid_token(self):
        from models.auth import UserModel
        import pyotp
        secret = pyotp.random_base32()
        assert UserModel.verify_totp(secret, "000000") is False

    def test_get_totp_uri(self):
        from models.auth import UserModel
        import pyotp
        secret = pyotp.random_base32()
        uri = UserModel.get_totp_uri(secret, "admin", "MarchProxy")
        assert "otpauth://totp/" in uri
        assert "admin" in uri

    def test_define_table_calls_db(self):
        from models.auth import UserModel
        db = MagicMock()
        db.define_table = MagicMock(return_value=MagicMock())
        UserModel.define_table(db)
        db.define_table.assert_called_once()
        call_args = db.define_table.call_args
        assert call_args[0][0] == "users"


# ===========================================================================
# SessionModel tests
# ===========================================================================

class TestSessionModel:

    def test_generate_session_id_is_string(self):
        from models.auth import SessionModel
        sid = SessionModel.generate_session_id()
        assert isinstance(sid, str)
        assert len(sid) >= 40

    def test_generate_session_id_unique(self):
        from models.auth import SessionModel
        ids = {SessionModel.generate_session_id() for _ in range(10)}
        assert len(ids) == 10

    def test_create_session_returns_session_id(self):
        from models.auth import SessionModel
        db = _make_db()
        db.sessions.insert.return_value = 1
        sid = SessionModel.create_session(db, user_id=1, ip_address="127.0.0.1")
        assert isinstance(sid, str)
        db.sessions.insert.assert_called_once()

    def test_create_session_custom_ttl(self):
        from models.auth import SessionModel
        db = _make_db()
        db.sessions.insert.return_value = 1
        sid = SessionModel.create_session(db, user_id=1, ttl_hours=2)
        assert sid is not None

    def test_validate_session_no_session(self):
        from models.auth import SessionModel
        db = _make_db()
        # select returns empty result
        result = SessionModel.validate_session(db, "nonexistent-session")
        assert result is None

    def test_validate_session_with_valid_session(self):
        from models.auth import SessionModel
        db = _make_db()
        session_mock = MagicMock()
        session_mock.user_id = 1
        session_mock.update_record = MagicMock()
        user_mock = MagicMock()
        user_mock.id = 1
        user_mock.username = "admin"
        user_mock.email = "admin@test.com"
        user_mock.is_admin = True
        user_mock.is_active = True
        # db(cond).select().first() → session
        select_result = _FakeSelectResult([session_mock])
        query_mock = MagicMock()
        query_mock.select = MagicMock(return_value=select_result)
        db.return_value = query_mock
        db.users.__getitem__ = MagicMock(return_value=user_mock)
        result = SessionModel.validate_session(db, "valid-session-id")
        assert result is not None
        assert result["user_id"] == 1

    def test_validate_session_inactive_user(self):
        from models.auth import SessionModel
        db = _make_db()
        session_mock = MagicMock()
        session_mock.user_id = 1
        session_mock.update_record = MagicMock()
        user_mock = MagicMock()
        user_mock.is_active = False
        select_result = _FakeSelectResult([session_mock])
        query_mock = MagicMock()
        query_mock.select = MagicMock(return_value=select_result)
        db.return_value = query_mock
        db.users.__getitem__ = MagicMock(return_value=user_mock)
        result = SessionModel.validate_session(db, "valid-session-id")
        assert result is None

    def test_destroy_session_returns_true_when_deleted(self):
        from models.auth import SessionModel
        db = _make_db()
        query_mock = MagicMock()
        query_mock.delete = MagicMock(return_value=1)
        db.return_value = query_mock
        result = SessionModel.destroy_session(db, "some-session-id")
        assert result is True

    def test_destroy_session_returns_false_when_not_found(self):
        from models.auth import SessionModel
        db = _make_db()
        query_mock = MagicMock()
        query_mock.delete = MagicMock(return_value=0)
        db.return_value = query_mock
        result = SessionModel.destroy_session(db, "nonexistent-id")
        assert result is False

    def test_define_table_calls_db(self):
        from models.auth import SessionModel
        db = MagicMock()
        db.define_table = MagicMock(return_value=MagicMock())
        SessionModel.define_table(db)
        db.define_table.assert_called_once()
        assert db.define_table.call_args[0][0] == "sessions"


# ===========================================================================
# APITokenModel tests
# ===========================================================================

class TestAPITokenModel:

    def test_generate_token_returns_tuple(self):
        from models.auth import APITokenModel
        token, token_id = APITokenModel.generate_token()
        assert isinstance(token, str)
        assert isinstance(token_id, str)
        assert len(token) >= 40
        assert len(token_id) >= 30

    def test_hash_token_and_verify(self):
        from models.auth import APITokenModel
        token = "my-api-token-value"
        hashed = APITokenModel.hash_token(token)
        assert APITokenModel.verify_token(token, hashed) is True
        assert APITokenModel.verify_token("wrong-token", hashed) is False

    def test_create_token_returns_tuple(self):
        from models.auth import APITokenModel
        db = _make_db()
        db.api_tokens.insert.return_value = 1
        token, token_id = APITokenModel.create_token(db, name="test-token", user_id=1)
        assert isinstance(token, str)
        assert isinstance(token_id, str)
        db.api_tokens.insert.assert_called_once()

    def test_create_token_with_expiry(self):
        from models.auth import APITokenModel
        db = _make_db()
        db.api_tokens.insert.return_value = 1
        token, token_id = APITokenModel.create_token(
            db, name="expiring-token", user_id=1, ttl_days=7
        )
        call_kwargs = db.api_tokens.insert.call_args[1]
        assert call_kwargs["expires_at"] is not None

    def test_create_token_no_expiry(self):
        from models.auth import APITokenModel
        db = _make_db()
        db.api_tokens.insert.return_value = 1
        token, token_id = APITokenModel.create_token(db, name="perm-token")
        call_kwargs = db.api_tokens.insert.call_args[1]
        assert call_kwargs["expires_at"] is None

    def test_validate_token_no_tokens(self):
        from models.auth import APITokenModel
        db = _make_db()
        # db(condition).select() returns empty iterable
        select_result = _FakeSelectResult([])
        query_mock = MagicMock()
        query_mock.select = MagicMock(return_value=select_result)
        db.return_value = query_mock
        result = APITokenModel.validate_token(db, "nonexistent-token")
        assert result is None

    def test_validate_token_with_valid_token(self):
        from models.auth import APITokenModel
        db = _make_db()
        raw_token = "my-valid-raw-token"
        hashed = APITokenModel.hash_token(raw_token)
        token_record = MagicMock()
        token_record.token_hash = hashed
        token_record.token_id = "tid-123"
        token_record.name = "test"
        token_record.user_id = 1
        token_record.service_id = None
        token_record.cluster_id = None
        token_record.permissions = {}
        token_record.update_record = MagicMock()
        select_result = _FakeSelectResult([token_record])
        query_mock = MagicMock()
        query_mock.select = MagicMock(return_value=select_result)
        db.return_value = query_mock
        result = APITokenModel.validate_token(db, raw_token)
        assert result is not None
        assert result["token_id"] == "tid-123"
        assert result["user_id"] == 1

    def test_define_table_calls_db(self):
        from models.auth import APITokenModel
        db = MagicMock()
        db.define_table = MagicMock(return_value=MagicMock())
        APITokenModel.define_table(db)
        db.define_table.assert_called_once()
        assert db.define_table.call_args[0][0] == "api_tokens"


# ===========================================================================
# JWTManager tests
# ===========================================================================

class TestJWTManager:

    def _make_jwt_manager(self, ttl_hours=1):
        from models.auth import JWTManager
        return JWTManager(
            secret_key="test-secret-key-for-testing-32chars!",
            algorithm="HS256",
            ttl_hours=ttl_hours,
        )

    def test_create_token_returns_string(self):
        mgr = self._make_jwt_manager()
        token = mgr.create_token({"user_id": 1, "username": "admin"})
        assert isinstance(token, str)

    def test_decode_token_valid(self):
        mgr = self._make_jwt_manager()
        token = mgr.create_token({"user_id": 1, "username": "admin"})
        payload = mgr.decode_token(token)
        assert payload is not None
        assert payload["user_id"] == 1

    def test_decode_token_expired(self):
        mgr = self._make_jwt_manager(ttl_hours=-1)
        token = mgr.create_token({"user_id": 1})
        result = mgr.decode_token(token)
        assert result is None

    def test_decode_token_invalid(self):
        mgr = self._make_jwt_manager()
        result = mgr.decode_token("not-a-valid-jwt-token")
        assert result is None

    def test_create_refresh_token(self):
        mgr = self._make_jwt_manager()
        refresh = mgr.create_refresh_token(user_id=1)
        assert isinstance(refresh, str)

    def test_refresh_access_token_valid(self):
        mgr = self._make_jwt_manager()
        refresh = mgr.create_refresh_token(user_id=1)
        new_access = mgr.refresh_access_token(refresh)
        assert new_access is not None

    def test_refresh_access_token_with_wrong_type(self):
        mgr = self._make_jwt_manager()
        # access token should have type "access", not "refresh"
        access = mgr.create_token({"user_id": 1, "type": "access"})
        result = mgr.refresh_access_token(access)
        assert result is None

    def test_refresh_access_token_invalid(self):
        mgr = self._make_jwt_manager()
        result = mgr.refresh_access_token("not-a-token")
        assert result is None


# ===========================================================================
# Pydantic validation models (LoginRequest, RegisterRequest)
# ===========================================================================

class TestAuthPydanticModels:

    def test_login_request_valid(self):
        from models.auth import LoginRequest
        req = LoginRequest(username="admin", password="pass123")
        assert req.username == "admin"

    def test_login_request_with_totp(self):
        from models.auth import LoginRequest
        req = LoginRequest(username="admin", password="pass", totp_code="123456")
        assert req.totp_code == "123456"

    def test_register_request_valid(self):
        from models.auth import RegisterRequest
        req = RegisterRequest(
            username="newuser", email="user@example.com", password="Password123!"
        )
        assert req.username == "newuser"

    def test_register_request_password_too_short(self):
        from models.auth import RegisterRequest
        from pydantic import ValidationError
        with pytest.raises(ValidationError):
            RegisterRequest(username="u", email="u@e.com", password="short")


# ===========================================================================
# RBACModel tests
# ===========================================================================

class TestRBACModelDefineTables:

    def test_define_tables_calls_db_define_table(self):
        from models.rbac import RBACModel
        db = MagicMock()
        db.define_table = MagicMock(return_value=MagicMock())
        RBACModel.define_tables(db)
        assert db.define_table.call_count == 3
        table_names = [c[0][0] for c in db.define_table.call_args_list]
        assert "roles" in table_names
        assert "user_roles" in table_names
        assert "user_permissions_cache" in table_names


class TestRBACModelInitializeDefaultRoles:

    def test_creates_roles_when_not_existing(self):
        from models.rbac import RBACModel
        db = _make_db()
        # db(condition).select().first() → None (no existing role)
        select_result = _FakeSelectResult([])
        query_mock = MagicMock()
        query_mock.select = MagicMock(return_value=select_result)
        db.return_value = query_mock
        RBACModel.initialize_default_roles(db)
        assert db.roles.insert.called

    def test_skips_existing_roles(self):
        from models.rbac import RBACModel
        db = _make_db()
        existing_role = MagicMock()
        # All roles already exist
        select_result = _FakeSelectResult([existing_role])
        query_mock = MagicMock()
        query_mock.select = MagicMock(return_value=select_result)
        db.return_value = query_mock
        RBACModel.initialize_default_roles(db)
        db.roles.insert.assert_not_called()


class TestRBACModelAssignRole:

    def test_assign_role_not_found_raises(self):
        from models.rbac import RBACModel, PermissionScope
        db = _make_db()
        # Role not found
        result = _FakeSelectResult([])
        qm = MagicMock()
        qm.select = MagicMock(return_value=result)
        db.return_value = qm
        with pytest.raises(ValueError, match="not found"):
            RBACModel.assign_role(db, user_id=1, role_name="nonexistent")

    def test_assign_role_scope_mismatch_raises(self):
        from models.rbac import RBACModel, PermissionScope
        db = _make_db()
        role = MagicMock()
        role.scope = PermissionScope.CLUSTER.value  # requires cluster scope
        role.id = 1
        result = _FakeSelectResult([role])
        qm = MagicMock()
        qm.select = MagicMock(return_value=result)
        db.return_value = qm
        with pytest.raises(ValueError, match="requires scope"):
            RBACModel.assign_role(
                db, user_id=1, role_name="viewer",
                scope=PermissionScope.GLOBAL  # global != cluster
            )

    def test_assign_role_returns_existing_if_already_assigned(self):
        from models.rbac import RBACModel, PermissionScope
        db = _make_db()
        role = MagicMock()
        role.scope = PermissionScope.GLOBAL.value
        role.id = 1
        existing_assignment = MagicMock()
        existing_assignment.id = 99
        # First call finds role, second call finds existing assignment
        call_count = [0]
        def db_call_side_effect(condition):
            call_count[0] += 1
            if call_count[0] == 1:
                # First call: finding role
                r = _FakeSelectResult([role])
            else:
                # Second call: finding existing assignment
                r = _FakeSelectResult([existing_assignment])
            qm = MagicMock()
            qm.select = MagicMock(return_value=r)
            return qm
        db.side_effect = db_call_side_effect
        result = RBACModel.assign_role(db, user_id=1, role_name="admin")
        assert result == 99
        db.user_roles.insert.assert_not_called()

    def test_assign_role_creates_new_assignment(self):
        from models.rbac import RBACModel, PermissionScope
        db = _make_db()
        role = MagicMock()
        role.scope = PermissionScope.GLOBAL.value
        role.id = 1
        db.user_roles.insert.return_value = 42
        call_count = [0]
        def db_call_side_effect(condition):
            call_count[0] += 1
            if call_count[0] == 1:
                r = _FakeSelectResult([role])
            else:
                r = _FakeSelectResult([])  # no existing assignment
            qm = MagicMock()
            qm.select = MagicMock(return_value=r)
            qm.delete = MagicMock(return_value=0)
            return qm
        db.side_effect = db_call_side_effect
        result = RBACModel.assign_role(db, user_id=1, role_name="admin")
        assert result == 42
        db.user_roles.insert.assert_called_once()


class TestRBACModelRevokeRole:

    def test_revoke_role_not_found_raises(self):
        from models.rbac import RBACModel
        db = _make_db()
        result = _FakeSelectResult([])
        qm = MagicMock()
        qm.select = MagicMock(return_value=result)
        db.return_value = qm
        with pytest.raises(ValueError, match="not found"):
            RBACModel.revoke_role(db, user_id=1, role_name="nonexistent")

    def test_revoke_role_updates_assignment(self):
        from models.rbac import RBACModel
        db = _make_db()
        role = MagicMock()
        role.id = 1
        call_count = [0]
        update_mock = MagicMock()
        def db_call_side_effect(condition):
            call_count[0] += 1
            if call_count[0] == 1:
                r = _FakeSelectResult([role])
            else:
                r = _FakeSelectResult([])
            qm = MagicMock()
            qm.select = MagicMock(return_value=r)
            qm.update = update_mock
            qm.delete = MagicMock(return_value=0)
            return qm
        db.side_effect = db_call_side_effect
        RBACModel.revoke_role(db, user_id=1, role_name="admin")
        update_mock.assert_called_once_with(is_active=False)

    def test_revoke_role_with_resource_id(self):
        from models.rbac import RBACModel
        db = _make_db()
        role = MagicMock()
        role.id = 1
        update_mock = MagicMock()
        call_count = [0]
        def db_call_side_effect(condition):
            call_count[0] += 1
            if call_count[0] == 1:
                r = _FakeSelectResult([role])
            else:
                r = _FakeSelectResult([])
            qm = MagicMock()
            qm.select = MagicMock(return_value=r)
            qm.update = update_mock
            qm.delete = MagicMock(return_value=0)
            return qm
        db.side_effect = db_call_side_effect
        RBACModel.revoke_role(db, user_id=1, role_name="admin", resource_id=5)
        update_mock.assert_called_once_with(is_active=False)


class TestRBACModelGetUserPermissions:

    def test_returns_cached_permissions(self):
        from models.rbac import RBACModel
        db = _make_db()
        cache = MagicMock()
        cache.global_permissions = ["clusters:read"]
        cache.cluster_permissions = {}
        cache.service_permissions = {}
        result = _FakeSelectResult([cache])
        qm = MagicMock()
        qm.select = MagicMock(return_value=result)
        db.return_value = qm
        perms = RBACModel.get_user_permissions(db, user_id=1)
        assert "clusters:read" in perms["global"]

    def test_builds_permissions_when_no_cache(self):
        from models.rbac import RBACModel, PermissionScope
        db = _make_db()
        call_count = [0]
        def db_side_effect(condition):
            call_count[0] += 1
            qm = MagicMock()
            # First call: check cache → empty
            # Second call: get assignments
            qm.select = MagicMock(return_value=_FakeSelectResult([]))
            qm.update = MagicMock()
            qm.delete = MagicMock()
            return qm
        db.side_effect = db_side_effect
        db.user_permissions_cache.insert = MagicMock(return_value=1)
        perms = RBACModel.get_user_permissions(db, user_id=1)
        assert "global" in perms
        assert "cluster" in perms
        assert "service" in perms


class TestRBACModelHasPermission:

    def test_has_permission_via_global(self):
        from models.rbac import RBACModel
        with patch.object(RBACModel, "get_user_permissions", return_value={
            "global": ["clusters:read"],
            "cluster": {},
            "service": {},
        }):
            assert RBACModel.has_permission(
                MagicMock(), user_id=1, permission="clusters:read"
            ) is True

    def test_has_permission_via_global_admin(self):
        from models.rbac import RBACModel, Permissions
        with patch.object(RBACModel, "get_user_permissions", return_value={
            "global": [Permissions.GLOBAL_ADMIN],
            "cluster": {},
            "service": {},
        }):
            assert RBACModel.has_permission(
                MagicMock(), user_id=1, permission="anything:read"
            ) is True

    def test_has_permission_cluster_scoped(self):
        from models.rbac import RBACModel
        with patch.object(RBACModel, "get_user_permissions", return_value={
            "global": [],
            "cluster": {"5": ["clusters:write"]},
            "service": {},
        }):
            assert RBACModel.has_permission(
                MagicMock(), user_id=1, permission="clusters:write",
                resource_type="cluster", resource_id=5
            ) is True

    def test_has_permission_service_scoped(self):
        from models.rbac import RBACModel
        with patch.object(RBACModel, "get_user_permissions", return_value={
            "global": [],
            "cluster": {},
            "service": {"3": ["services:read"]},
        }):
            assert RBACModel.has_permission(
                MagicMock(), user_id=1, permission="services:read",
                resource_type="service", resource_id=3
            ) is True

    def test_has_permission_not_found_returns_false(self):
        from models.rbac import RBACModel
        with patch.object(RBACModel, "get_user_permissions", return_value={
            "global": [],
            "cluster": {},
            "service": {},
        }):
            assert RBACModel.has_permission(
                MagicMock(), user_id=1, permission="admin:delete"
            ) is False


class TestRBACModelInvalidateCache:

    def test_invalidate_permission_cache_deletes(self):
        from models.rbac import RBACModel
        db = _make_db()
        delete_mock = MagicMock()
        qm = MagicMock()
        qm.delete = delete_mock
        db.return_value = qm
        RBACModel.invalidate_permission_cache(db, user_id=1)
        delete_mock.assert_called_once()


class TestRBACModelGetUserRoles:

    def test_returns_empty_list_when_no_roles(self):
        from models.rbac import RBACModel
        db = _make_db()
        result = _FakeSelectResult([])
        qm = MagicMock()
        qm.select = MagicMock(return_value=result)
        db.return_value = qm
        roles = RBACModel.get_user_roles(db, user_id=1)
        assert roles == []

    def test_returns_role_list(self):
        from models.rbac import RBACModel
        db = _make_db()
        assignment = MagicMock()
        assignment.user_roles.id = 10
        assignment.user_roles.scope = "global"
        assignment.user_roles.resource_id = None
        assignment.user_roles.granted_at = datetime(2025, 1, 1)
        assignment.user_roles.granted_by = None
        assignment.roles.name = "admin"
        assignment.roles.display_name = "Admin"
        result = _FakeSelectResult([assignment])
        qm = MagicMock()
        qm.select = MagicMock(return_value=result)
        db.return_value = qm
        roles = RBACModel.get_user_roles(db, user_id=1)
        assert len(roles) == 1
        assert roles[0]["role_name"] == "admin"
        assert roles[0]["assignment_id"] == 10
