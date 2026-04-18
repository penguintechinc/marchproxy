"""Unit tests for app_quart models."""
import pytest
from unittest.mock import MagicMock, patch


# ---------------------------------------------------------------------------
# Kong models
# ---------------------------------------------------------------------------

def test_kong_service_model_exists():
    from app_quart.models.kong import KongService
    assert KongService is not None


def test_kong_route_model_exists():
    from app_quart.models.kong import KongRoute
    assert KongRoute is not None


def test_kong_upstream_model_exists():
    from app_quart.models.kong import KongUpstream
    assert KongUpstream is not None


def test_kong_target_model_exists():
    from app_quart.models.kong import KongTarget
    assert KongTarget is not None


def test_kong_consumer_model_exists():
    from app_quart.models.kong import KongConsumer
    assert KongConsumer is not None


def test_kong_plugin_model_exists():
    from app_quart.models.kong import KongPlugin
    assert KongPlugin is not None


def test_kong_certificate_model_exists():
    from app_quart.models.kong import KongCertificate
    assert KongCertificate is not None


def test_kong_sni_model_exists():
    from app_quart.models.kong import KongSNI
    assert KongSNI is not None


def test_kong_config_history_model_exists():
    from app_quart.models.kong import KongConfigHistory
    assert KongConfigHistory is not None


def test_kong_service_tablename():
    from app_quart.models.kong import KongService
    assert KongService.__tablename__ == 'kong_services'


def test_kong_route_tablename():
    from app_quart.models.kong import KongRoute
    assert KongRoute.__tablename__ == 'kong_routes'


def test_kong_upstream_tablename():
    from app_quart.models.kong import KongUpstream
    assert KongUpstream.__tablename__ == 'kong_upstreams'


def test_kong_target_tablename():
    from app_quart.models.kong import KongTarget
    assert KongTarget.__tablename__ == 'kong_targets'


def test_kong_consumer_tablename():
    from app_quart.models.kong import KongConsumer
    assert KongConsumer.__tablename__ == 'kong_consumers'


def test_kong_plugin_tablename():
    from app_quart.models.kong import KongPlugin
    assert KongPlugin.__tablename__ == 'kong_plugins'


def test_kong_certificate_tablename():
    from app_quart.models.kong import KongCertificate
    assert KongCertificate.__tablename__ == 'kong_certificates'


def test_kong_sni_tablename():
    from app_quart.models.kong import KongSNI
    assert KongSNI.__tablename__ == 'kong_snis'


def test_kong_config_history_tablename():
    from app_quart.models.kong import KongConfigHistory
    assert KongConfigHistory.__tablename__ == 'kong_config_history'


# ---------------------------------------------------------------------------
# User model
# ---------------------------------------------------------------------------

def test_user_model_exists():
    from app_quart.models.user import User
    assert User is not None


def test_role_model_exists():
    from app_quart.models.user import Role
    assert Role is not None


def test_user_tablename():
    from app_quart.models.user import User
    assert User.__tablename__ == 'users'


def test_role_tablename():
    from app_quart.models.user import Role
    assert Role.__tablename__ == 'roles'


def _make_user_with_roles(roles_permissions):
    """Create a mock User-like object with the has_permission method."""
    from app_quart.models.user import User
    # Call has_permission directly by binding it to a mock object
    user = MagicMock()
    mock_roles = []
    for perms in roles_permissions:
        role = MagicMock()
        role.permissions = perms
        mock_roles.append(role)
    user.roles = mock_roles
    # Bind the real method to our mock object
    user.has_permission = lambda perm: User.has_permission(user, perm)
    return user


def test_user_has_permission_true():
    """has_permission returns True when role has the permission."""
    user = _make_user_with_roles([['read', 'write']])
    assert user.has_permission('read') is True


def test_user_has_permission_false():
    """has_permission returns False when no role has the permission."""
    user = _make_user_with_roles([['read']])
    assert user.has_permission('admin') is False


def test_user_has_permission_empty_roles():
    """has_permission returns False when user has no roles."""
    user = _make_user_with_roles([])
    assert user.has_permission('read') is False


def test_user_has_permission_none_permissions():
    """has_permission handles None permissions gracefully."""
    user = _make_user_with_roles([None])
    assert user.has_permission('read') is False


def test_user_has_permission_multiple_roles():
    """has_permission checks all roles."""
    user = _make_user_with_roles([['read'], ['admin', 'write']])
    assert user.has_permission('admin') is True
    assert user.has_permission('write') is True
    assert user.has_permission('delete') is False


# ---------------------------------------------------------------------------
# Audit model
# ---------------------------------------------------------------------------

def test_audit_log_model_exists():
    from app_quart.models.audit import AuditLog
    assert AuditLog is not None


def test_audit_log_tablename():
    from app_quart.models.audit import AuditLog
    assert AuditLog.__tablename__ == 'audit_logs'


# ---------------------------------------------------------------------------
# Models __init__
# ---------------------------------------------------------------------------

def test_models_init_exports():
    from app_quart.models import User, Role, AuditLog
    assert User is not None
    assert Role is not None
    assert AuditLog is not None
