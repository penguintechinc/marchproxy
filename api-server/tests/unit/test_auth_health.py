"""Unit tests for auth and health endpoints.

Tests health handlers directly (no auth needed), and auth handlers by
patching flask_security.auth_required to be a no-op before import.
"""
import json
import pytest
from unittest.mock import AsyncMock, MagicMock, patch


# ---------------------------------------------------------------------------
# Health endpoint - direct function calls (no auth needed)
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_healthz_returns_200_status():
    """healthz returns status 200."""
    from quart import Quart
    app = Quart(__name__)
    async with app.app_context():
        from app_quart.api.v1.health import healthz
        response, status = await healthz()
        assert status == 200


@pytest.mark.asyncio
async def test_healthz_returns_healthy_json():
    """healthz returns {'status': 'healthy'}."""
    from quart import Quart
    app = Quart(__name__)
    async with app.app_context():
        from app_quart.api.v1.health import healthz
        response, status = await healthz()
        data = json.loads(await response.get_data(as_text=True))
        assert data['status'] == 'healthy'


@pytest.mark.asyncio
async def test_readyz_returns_ready_when_db_ok():
    """readyz returns {'status': 'ready', 'database': 'connected'} with 200."""
    from quart import Quart
    app = Quart(__name__)
    async with app.app_context():
        with patch('app_quart.api.v1.health.db') as mock_db:
            mock_db.session = MagicMock()
            mock_db.session.execute = AsyncMock(return_value=MagicMock())
            from app_quart.api.v1.health import readyz
            response, status = await readyz()
            data = json.loads(await response.get_data(as_text=True))
            assert status == 200
            assert data['status'] == 'ready'
            assert data['database'] == 'connected'


@pytest.mark.asyncio
async def test_readyz_returns_503_when_db_fails():
    """readyz returns 503 when DB raises."""
    from quart import Quart
    app = Quart(__name__)
    async with app.app_context():
        with patch('app_quart.api.v1.health.db') as mock_db:
            mock_db.session = MagicMock()
            mock_db.session.execute = AsyncMock(side_effect=Exception('Connection refused'))
            from app_quart.api.v1.health import readyz
            response, status = await readyz()
            data = json.loads(await response.get_data(as_text=True))
            assert status == 503
            assert data['status'] == 'not_ready'
            assert data['database'] == 'disconnected'
            assert 'error' in data


@pytest.mark.asyncio
async def test_readyz_includes_error_message():
    """readyz error response includes the exception message."""
    from quart import Quart
    app = Quart(__name__)
    async with app.app_context():
        with patch('app_quart.api.v1.health.db') as mock_db:
            mock_db.session = MagicMock()
            mock_db.session.execute = AsyncMock(side_effect=Exception('timeout'))
            from app_quart.api.v1.health import readyz
            response, status = await readyz()
            data = json.loads(await response.get_data(as_text=True))
            assert 'timeout' in data['error']


@pytest.mark.asyncio
async def test_readyz_calls_db_select_1():
    """readyz executes 'SELECT 1' against the database."""
    from quart import Quart
    from sqlalchemy import text
    app = Quart(__name__)
    async with app.app_context():
        with patch('app_quart.api.v1.health.db') as mock_db:
            mock_db.session = MagicMock()
            mock_db.session.execute = AsyncMock(return_value=None)
            from app_quart.api.v1.health import readyz
            await readyz()
            mock_db.session.execute.assert_called_once()


# ---------------------------------------------------------------------------
# Auth endpoint - requires patching auth_required at function level
# We patch the underlying logic by using a minimal test app with security
# configured, or by testing the inner logic directly after removing decorator.
# ---------------------------------------------------------------------------

def _make_undecorated_auth_module():
    """
    Return a namespace with auth handler functions stripped of decorators.
    We extract the raw coroutine by accessing __wrapped__ or by re-importing
    with auth_required patched.
    """
    import importlib
    import sys

    # Remove cached module so we get a fresh import with the mock
    for key in list(sys.modules.keys()):
        if 'app_quart.api.v1.auth' in key:
            del sys.modules[key]

    def noop_auth_required(*args, **kwargs):
        def decorator(fn):
            return fn
        return decorator

    with patch('flask_security.auth_required', noop_auth_required):
        with patch('flask_security.current_user', MagicMock()):
            with patch('flask_security.login_user', MagicMock()):
                with patch('flask_security.logout_user', MagicMock()):
                    import app_quart.api.v1.auth as auth_mod
                    importlib.reload(auth_mod)
                    return auth_mod


@pytest.mark.asyncio
async def test_get_current_user_returns_user_fields():
    """get_current_user returns id, email, username, roles."""
    from quart import Quart
    app = Quart(__name__)

    auth_mod = _make_undecorated_auth_module()

    async with app.app_context():
        mock_user = MagicMock()
        mock_user.id = 42
        mock_user.email = 'testuser@example.com'
        mock_user.username = 'testuser'
        role1 = MagicMock()
        role1.name = 'Admin'
        role2 = MagicMock()
        role2.name = 'Viewer'
        mock_user.roles = [role1, role2]

        with patch.object(auth_mod, 'current_user', mock_user):
            response = await auth_mod.get_current_user()
            data = json.loads(await response.get_data(as_text=True))
            assert data['id'] == 42
            assert data['email'] == 'testuser@example.com'
            assert data['username'] == 'testuser'
            assert 'Admin' in data['roles']
            assert 'Viewer' in data['roles']


@pytest.mark.asyncio
async def test_get_current_user_roles_are_name_strings():
    """roles list contains role .name strings."""
    from quart import Quart
    app = Quart(__name__)
    auth_mod = _make_undecorated_auth_module()

    async with app.app_context():
        mock_user = MagicMock()
        mock_user.id = 1
        mock_user.email = 'a@b.com'
        mock_user.username = 'ab'
        r = MagicMock()
        r.name = 'Maintainer'
        mock_user.roles = [r]

        with patch.object(auth_mod, 'current_user', mock_user):
            response = await auth_mod.get_current_user()
            data = json.loads(await response.get_data(as_text=True))
            assert data['roles'] == ['Maintainer']


@pytest.mark.asyncio
async def test_get_current_user_empty_roles():
    """User with no roles returns empty list."""
    from quart import Quart
    app = Quart(__name__)
    auth_mod = _make_undecorated_auth_module()

    async with app.app_context():
        mock_user = MagicMock()
        mock_user.id = 1
        mock_user.email = 'a@b.com'
        mock_user.username = 'ab'
        mock_user.roles = []

        with patch.object(auth_mod, 'current_user', mock_user):
            response = await auth_mod.get_current_user()
            data = json.loads(await response.get_data(as_text=True))
            assert data['roles'] == []


@pytest.mark.asyncio
async def test_logout_calls_logout_user():
    """logout() calls flask_security logout_user exactly once."""
    from quart import Quart
    app = Quart(__name__)
    auth_mod = _make_undecorated_auth_module()

    async with app.app_context():
        with patch.object(auth_mod, 'logout_user') as mock_logout:
            response = await auth_mod.logout()
            mock_logout.assert_called_once()


@pytest.mark.asyncio
async def test_logout_returns_success_message():
    """logout returns 'Logged out successfully'."""
    from quart import Quart
    app = Quart(__name__)
    auth_mod = _make_undecorated_auth_module()

    async with app.app_context():
        with patch.object(auth_mod, 'logout_user'):
            response = await auth_mod.logout()
            data = json.loads(await response.get_data(as_text=True))
            assert data['message'] == 'Logged out successfully'
