"""Unit tests for Kong API route handlers.

Tests all handler functions directly by patching:
- flask_security.auth_required → no-op
- KongClient → AsyncMock
- db.session → MagicMock
- AuditService.log → AsyncMock
- current_user → MagicMock
"""
import importlib
import json
import sys
import pytest
from unittest.mock import AsyncMock, MagicMock, patch


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _fresh_import(module_path: str):
    """Import a module with caches cleared, auth_required patched as no-op."""
    # Remove stale cached modules
    for key in list(sys.modules.keys()):
        if module_path.replace('.', '/') in key.replace('.', '/') or key == module_path:
            del sys.modules[key]

    def noop_auth_required(*args, **kwargs):
        def decorator(fn):
            return fn
        return decorator

    with patch('flask_security.auth_required', noop_auth_required):
        mod = importlib.import_module(module_path)
        importlib.reload(mod)
        return mod


def _mock_client():
    """Return a mock KongClient instance with async methods."""
    client = MagicMock()
    for method in [
        'list_services', 'get_service', 'create_service', 'update_service', 'delete_service',
        'list_routes', 'get_route', 'create_route', 'update_route', 'delete_route',
        'list_upstreams', 'get_upstream', 'create_upstream', 'update_upstream', 'delete_upstream',
        'list_targets', 'create_target', 'delete_target',
        'list_consumers', 'get_consumer', 'create_consumer', 'update_consumer', 'delete_consumer',
        'list_plugins', 'get_enabled_plugins', 'get_plugin_schema', 'get_plugin',
        'create_plugin', 'update_plugin', 'delete_plugin',
        'list_certificates', 'get_certificate', 'create_certificate',
        'update_certificate', 'delete_certificate',
        'list_snis', 'create_sni', 'delete_sni',
        'list_config', 'get_config', 'post_config', 'get_status',
        'close',
    ]:
        setattr(client, method, AsyncMock())
    return client


def _make_mock_user(user_id=1, email='admin@test.com'):
    user = MagicMock()
    user.id = user_id
    user.email = email
    return user


def _make_mock_db():
    db = MagicMock()
    db.session = MagicMock()
    db.session.add = MagicMock()
    db.session.delete = MagicMock()
    db.session.commit = AsyncMock()
    return db


# ---------------------------------------------------------------------------
# Quart app context fixture
# ---------------------------------------------------------------------------

@pytest.fixture
def quart_app():
    from quart import Quart
    app = Quart(__name__)
    app.config['TESTING'] = True
    return app


# ===========================================================================
# Kong SERVICES handlers
# ===========================================================================

@pytest.mark.asyncio
async def test_list_kong_services(quart_app):
    mock_client = _mock_client()
    mock_client.list_services.return_value = {'data': [{'id': 's1'}], 'total': 1}

    services_mod = _fresh_import('app_quart.api.v1.kong.services')

    async with quart_app.test_request_context('/api/v1/kong/services', method='GET'):
        with patch.object(services_mod, 'KongClient', return_value=mock_client):
            response = await services_mod.list_kong_services()
            data = json.loads(await response.get_data(as_text=True))
            assert data['total'] == 1
            mock_client.list_services.assert_called_once()
            mock_client.close.assert_called_once()


@pytest.mark.asyncio
async def test_list_kong_services_default_params(quart_app):
    mock_client = _mock_client()
    mock_client.list_services.return_value = {'data': [], 'total': 0}
    services_mod = _fresh_import('app_quart.api.v1.kong.services')

    async with quart_app.test_request_context('/api/v1/kong/services', method='GET'):
        with patch.object(services_mod, 'KongClient', return_value=mock_client):
            await services_mod.list_kong_services()
            mock_client.list_services.assert_called_once_with(offset=0, size=100)


@pytest.mark.asyncio
async def test_get_kong_service(quart_app):
    mock_client = _mock_client()
    mock_client.get_service.return_value = {'id': 'svc1', 'name': 'test-service'}
    services_mod = _fresh_import('app_quart.api.v1.kong.services')

    async with quart_app.test_request_context('/api/v1/kong/services/svc1', method='GET'):
        with patch.object(services_mod, 'KongClient', return_value=mock_client):
            response = await services_mod.get_kong_service('svc1')
            data = json.loads(await response.get_data(as_text=True))
            assert data['name'] == 'test-service'
            mock_client.get_service.assert_called_once_with('svc1')


@pytest.mark.asyncio
async def test_create_kong_service(quart_app):
    mock_client = _mock_client()
    kong_result = {
        'id': 'new-svc', 'name': 'my-service', 'host': 'example.com',
        'protocol': 'http', 'port': 80
    }
    mock_client.create_service.return_value = kong_result
    services_mod = _fresh_import('app_quart.api.v1.kong.services')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()
    audit_mock = AsyncMock()

    async with quart_app.test_request_context(
        '/api/v1/kong/services',
        method='POST',
        json={'name': 'my-service', 'host': 'example.com'}
    ):
        with patch.object(services_mod, 'KongClient', return_value=mock_client):
            with patch.object(services_mod, 'db', mock_db):
                with patch.object(services_mod, 'current_user', mock_user):
                    with patch.object(services_mod.AuditService, 'log', audit_mock):
                        response, status = await services_mod.create_kong_service()
                        assert status == 201
                        data = json.loads(await response.get_data(as_text=True))
                        assert data['id'] == 'new-svc'
                        mock_db.session.add.assert_called_once()
                        mock_db.session.commit.assert_called_once()
                        audit_mock.assert_called_once()


@pytest.mark.asyncio
async def test_update_kong_service(quart_app):
    mock_client = _mock_client()
    mock_client.get_service.return_value = {'id': 'svc1', 'name': 'old'}
    mock_client.update_service.return_value = {'id': 'svc1', 'name': 'updated'}
    services_mod = _fresh_import('app_quart.api.v1.kong.services')
    mock_db = _make_mock_db()
    mock_db.session.query = MagicMock()
    mock_user = _make_mock_user()
    audit_mock = AsyncMock()

    # KongService.query.filter_by needs to work
    mock_db_service = MagicMock()
    services_mod.KongService = MagicMock()
    services_mod.KongService.query = MagicMock()
    services_mod.KongService.query.filter_by.return_value.first.return_value = mock_db_service

    async with quart_app.test_request_context(
        '/api/v1/kong/services/svc1',
        method='PATCH',
        json={'name': 'updated'}
    ):
        with patch.object(services_mod, 'KongClient', return_value=mock_client):
            with patch.object(services_mod, 'db', mock_db):
                with patch.object(services_mod, 'current_user', mock_user):
                    with patch.object(services_mod.AuditService, 'log', audit_mock):
                        response = await services_mod.update_kong_service('svc1')
                        data = json.loads(await response.get_data(as_text=True))
                        assert data['name'] == 'updated'
                        mock_client.update_service.assert_called_once_with('svc1', {'name': 'updated'})
                        audit_mock.assert_called_once()


@pytest.mark.asyncio
async def test_update_kong_service_no_db_record(quart_app):
    """update_service still succeeds when no local DB record exists."""
    mock_client = _mock_client()
    mock_client.get_service.return_value = {'id': 's1', 'name': 'old'}
    mock_client.update_service.return_value = {'id': 's1', 'name': 'new'}
    services_mod = _fresh_import('app_quart.api.v1.kong.services')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()

    services_mod.KongService = MagicMock()
    services_mod.KongService.query.filter_by.return_value.first.return_value = None

    async with quart_app.test_request_context(
        '/api/v1/kong/services/s1', method='PATCH', json={'name': 'new'}
    ):
        with patch.object(services_mod, 'KongClient', return_value=mock_client):
            with patch.object(services_mod, 'db', mock_db):
                with patch.object(services_mod, 'current_user', mock_user):
                    with patch.object(services_mod.AuditService, 'log', AsyncMock()):
                        response = await services_mod.update_kong_service('s1')
                        data = json.loads(await response.get_data(as_text=True))
                        assert data['id'] == 's1'
                        mock_db.session.commit.assert_not_called()


@pytest.mark.asyncio
async def test_delete_kong_service(quart_app):
    mock_client = _mock_client()
    mock_client.get_service.return_value = {'id': 'svc1', 'name': 'old'}
    services_mod = _fresh_import('app_quart.api.v1.kong.services')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()
    mock_db_service = MagicMock()
    services_mod.KongService = MagicMock()
    services_mod.KongService.query.filter_by.return_value.first.return_value = mock_db_service

    async with quart_app.test_request_context(
        '/api/v1/kong/services/svc1', method='DELETE'
    ):
        with patch.object(services_mod, 'KongClient', return_value=mock_client):
            with patch.object(services_mod, 'db', mock_db):
                with patch.object(services_mod, 'current_user', mock_user):
                    with patch.object(services_mod.AuditService, 'log', AsyncMock()):
                        response, status = await services_mod.delete_kong_service('svc1')
                        assert status == 204
                        mock_client.delete_service.assert_called_once_with('svc1')
                        mock_db.session.delete.assert_called_once_with(mock_db_service)


# ===========================================================================
# Kong ROUTES handlers
# ===========================================================================

@pytest.mark.asyncio
async def test_list_kong_routes(quart_app):
    mock_client = _mock_client()
    mock_client.list_routes.return_value = {'data': [{'id': 'r1'}], 'total': 1}
    routes_mod = _fresh_import('app_quart.api.v1.kong.routes')

    async with quart_app.test_request_context("/api/v1/kong/test", method="GET"):
        with patch.object(routes_mod, 'KongClient', return_value=mock_client):
            response = await routes_mod.list_kong_routes()
            data = json.loads(await response.get_data(as_text=True))
            assert data['total'] == 1


@pytest.mark.asyncio
async def test_get_kong_route(quart_app):
    mock_client = _mock_client()
    mock_client.get_route.return_value = {'id': 'r1', 'name': 'my-route'}
    routes_mod = _fresh_import('app_quart.api.v1.kong.routes')

    async with quart_app.test_request_context("/api/v1/kong/test", method="GET"):
        with patch.object(routes_mod, 'KongClient', return_value=mock_client):
            response = await routes_mod.get_kong_route('r1')
            data = json.loads(await response.get_data(as_text=True))
            assert data['name'] == 'my-route'
            mock_client.get_route.assert_called_once_with('r1')


@pytest.mark.asyncio
async def test_create_kong_route(quart_app):
    mock_client = _mock_client()
    kong_result = {'id': 'new-route', 'name': 'my-route', 'paths': ['/api']}
    mock_client.create_route.return_value = kong_result
    routes_mod = _fresh_import('app_quart.api.v1.kong.routes')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()

    async with quart_app.test_request_context(
        '/api/v1/kong/routes', method='POST', json={'name': 'my-route', 'paths': ['/api']}
    ):
        with patch.object(routes_mod, 'KongClient', return_value=mock_client):
            with patch.object(routes_mod, 'db', mock_db):
                with patch.object(routes_mod, 'current_user', mock_user):
                    with patch.object(routes_mod.AuditService, 'log', AsyncMock()):
                        response, status = await routes_mod.create_kong_route()
                        assert status == 201
                        data = json.loads(await response.get_data(as_text=True))
                        assert data['id'] == 'new-route'


@pytest.mark.asyncio
async def test_update_kong_route(quart_app):
    mock_client = _mock_client()
    mock_client.get_route.return_value = {'id': 'r1', 'name': 'old'}
    mock_client.update_route.return_value = {'id': 'r1', 'name': 'updated'}
    routes_mod = _fresh_import('app_quart.api.v1.kong.routes')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()
    routes_mod.KongRoute = MagicMock()
    routes_mod.KongRoute.query.filter_by.return_value.first.return_value = None

    async with quart_app.test_request_context(
        '/api/v1/kong/routes/r1', method='PATCH', json={'name': 'updated'}
    ):
        with patch.object(routes_mod, 'KongClient', return_value=mock_client):
            with patch.object(routes_mod, 'db', mock_db):
                with patch.object(routes_mod, 'current_user', mock_user):
                    with patch.object(routes_mod.AuditService, 'log', AsyncMock()):
                        response = await routes_mod.update_kong_route('r1')
                        data = json.loads(await response.get_data(as_text=True))
                        assert data['name'] == 'updated'


@pytest.mark.asyncio
async def test_delete_kong_route(quart_app):
    mock_client = _mock_client()
    mock_client.get_route.return_value = {'id': 'r1', 'name': 'my-route'}
    routes_mod = _fresh_import('app_quart.api.v1.kong.routes')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()
    routes_mod.KongRoute = MagicMock()
    routes_mod.KongRoute.query.filter_by.return_value.first.return_value = None

    async with quart_app.test_request_context(
        '/api/v1/kong/routes/r1', method='DELETE'
    ):
        with patch.object(routes_mod, 'KongClient', return_value=mock_client):
            with patch.object(routes_mod, 'db', mock_db):
                with patch.object(routes_mod, 'current_user', mock_user):
                    with patch.object(routes_mod.AuditService, 'log', AsyncMock()):
                        response, status = await routes_mod.delete_kong_route('r1')
                        assert status == 204
                        mock_client.delete_route.assert_called_once_with('r1')


# ===========================================================================
# Kong UPSTREAMS & TARGETS handlers
# ===========================================================================

@pytest.mark.asyncio
async def test_list_kong_upstreams(quart_app):
    mock_client = _mock_client()
    mock_client.list_upstreams.return_value = {'data': [], 'total': 0}
    upstreams_mod = _fresh_import('app_quart.api.v1.kong.upstreams')

    async with quart_app.test_request_context("/api/v1/kong/test", method="GET"):
        with patch.object(upstreams_mod, 'KongClient', return_value=mock_client):
            response = await upstreams_mod.list_kong_upstreams()
            data = json.loads(await response.get_data(as_text=True))
            assert 'data' in data


@pytest.mark.asyncio
async def test_get_kong_upstream(quart_app):
    mock_client = _mock_client()
    mock_client.get_upstream.return_value = {'id': 'u1', 'name': 'my-upstream'}
    upstreams_mod = _fresh_import('app_quart.api.v1.kong.upstreams')

    async with quart_app.test_request_context("/api/v1/kong/test", method="GET"):
        with patch.object(upstreams_mod, 'KongClient', return_value=mock_client):
            response = await upstreams_mod.get_kong_upstream('u1')
            data = json.loads(await response.get_data(as_text=True))
            assert data['name'] == 'my-upstream'


@pytest.mark.asyncio
async def test_create_kong_upstream(quart_app):
    mock_client = _mock_client()
    kong_result = {'id': 'u1', 'name': 'my-upstream', 'algorithm': 'round-robin'}
    mock_client.create_upstream.return_value = kong_result
    upstreams_mod = _fresh_import('app_quart.api.v1.kong.upstreams')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()

    async with quart_app.test_request_context(
        '/api/v1/kong/upstreams', method='POST', json={'name': 'my-upstream'}
    ):
        with patch.object(upstreams_mod, 'KongClient', return_value=mock_client):
            with patch.object(upstreams_mod, 'db', mock_db):
                with patch.object(upstreams_mod, 'current_user', mock_user):
                    with patch.object(upstreams_mod.AuditService, 'log', AsyncMock()):
                        response, status = await upstreams_mod.create_kong_upstream()
                        assert status == 201
                        data = json.loads(await response.get_data(as_text=True))
                        assert data['id'] == 'u1'


@pytest.mark.asyncio
async def test_update_kong_upstream(quart_app):
    mock_client = _mock_client()
    mock_client.get_upstream.return_value = {'id': 'u1', 'name': 'old'}
    mock_client.update_upstream.return_value = {'id': 'u1', 'name': 'new'}
    upstreams_mod = _fresh_import('app_quart.api.v1.kong.upstreams')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()
    upstreams_mod.KongUpstream = MagicMock()
    upstreams_mod.KongUpstream.query.filter_by.return_value.first.return_value = None

    async with quart_app.test_request_context(
        '/api/v1/kong/upstreams/u1', method='PATCH', json={'name': 'new'}
    ):
        with patch.object(upstreams_mod, 'KongClient', return_value=mock_client):
            with patch.object(upstreams_mod, 'db', mock_db):
                with patch.object(upstreams_mod, 'current_user', mock_user):
                    with patch.object(upstreams_mod.AuditService, 'log', AsyncMock()):
                        response = await upstreams_mod.update_kong_upstream('u1')
                        data = json.loads(await response.get_data(as_text=True))
                        assert data['name'] == 'new'


@pytest.mark.asyncio
async def test_delete_kong_upstream(quart_app):
    mock_client = _mock_client()
    mock_client.get_upstream.return_value = {'id': 'u1', 'name': 'up1'}
    upstreams_mod = _fresh_import('app_quart.api.v1.kong.upstreams')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()
    upstreams_mod.KongUpstream = MagicMock()
    upstreams_mod.KongUpstream.query.filter_by.return_value.first.return_value = None

    async with quart_app.test_request_context(
        '/api/v1/kong/upstreams/u1', method='DELETE'
    ):
        with patch.object(upstreams_mod, 'KongClient', return_value=mock_client):
            with patch.object(upstreams_mod, 'db', mock_db):
                with patch.object(upstreams_mod, 'current_user', mock_user):
                    with patch.object(upstreams_mod.AuditService, 'log', AsyncMock()):
                        response, status = await upstreams_mod.delete_kong_upstream('u1')
                        assert status == 204
                        mock_client.delete_upstream.assert_called_once_with('u1')


@pytest.mark.asyncio
async def test_list_kong_targets(quart_app):
    mock_client = _mock_client()
    mock_client.list_targets.return_value = {'data': [{'id': 't1'}], 'total': 1}
    upstreams_mod = _fresh_import('app_quart.api.v1.kong.upstreams')

    async with quart_app.test_request_context("/api/v1/kong/test", method="GET"):
        with patch.object(upstreams_mod, 'KongClient', return_value=mock_client):
            response = await upstreams_mod.list_kong_targets('u1')
            data = json.loads(await response.get_data(as_text=True))
            assert data['total'] == 1
            mock_client.list_targets.assert_called_once_with('u1')


@pytest.mark.asyncio
async def test_create_kong_target(quart_app):
    mock_client = _mock_client()
    kong_result = {'id': 't1', 'target': '10.0.0.1:80', 'weight': 100}
    mock_client.create_target.return_value = kong_result
    upstreams_mod = _fresh_import('app_quart.api.v1.kong.upstreams')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()
    upstreams_mod.KongUpstream = MagicMock()
    upstreams_mod.KongUpstream.query.filter_by.return_value.first.return_value = None

    async with quart_app.test_request_context(
        '/api/v1/kong/upstreams/u1/targets',
        method='POST',
        json={'target': '10.0.0.1:80', 'weight': 100}
    ):
        with patch.object(upstreams_mod, 'KongClient', return_value=mock_client):
            with patch.object(upstreams_mod, 'db', mock_db):
                with patch.object(upstreams_mod, 'current_user', mock_user):
                    with patch.object(upstreams_mod.AuditService, 'log', AsyncMock()):
                        response, status = await upstreams_mod.create_kong_target('u1')
                        assert status == 201
                        data = json.loads(await response.get_data(as_text=True))
                        assert data['target'] == '10.0.0.1:80'


@pytest.mark.asyncio
async def test_delete_kong_target(quart_app):
    mock_client = _mock_client()
    upstreams_mod = _fresh_import('app_quart.api.v1.kong.upstreams')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()
    upstreams_mod.KongTarget = MagicMock()
    upstreams_mod.KongTarget.query.filter_by.return_value.first.return_value = None

    async with quart_app.test_request_context(
        '/api/v1/kong/upstreams/u1/targets/t1', method='DELETE'
    ):
        with patch.object(upstreams_mod, 'KongClient', return_value=mock_client):
            with patch.object(upstreams_mod, 'db', mock_db):
                with patch.object(upstreams_mod, 'current_user', mock_user):
                    with patch.object(upstreams_mod.AuditService, 'log', AsyncMock()):
                        response, status = await upstreams_mod.delete_kong_target('u1', 't1')
                        assert status == 204
                        mock_client.delete_target.assert_called_once_with('u1', 't1')


# ===========================================================================
# Kong CONSUMERS handlers
# ===========================================================================

@pytest.mark.asyncio
async def test_list_kong_consumers(quart_app):
    mock_client = _mock_client()
    mock_client.list_consumers.return_value = {'data': [], 'total': 0}
    consumers_mod = _fresh_import('app_quart.api.v1.kong.consumers')

    async with quart_app.test_request_context("/api/v1/kong/test", method="GET"):
        with patch.object(consumers_mod, 'KongClient', return_value=mock_client):
            response = await consumers_mod.list_kong_consumers()
            data = json.loads(await response.get_data(as_text=True))
            assert 'data' in data


@pytest.mark.asyncio
async def test_get_kong_consumer(quart_app):
    mock_client = _mock_client()
    mock_client.get_consumer.return_value = {'id': 'c1', 'username': 'alice'}
    consumers_mod = _fresh_import('app_quart.api.v1.kong.consumers')

    async with quart_app.test_request_context("/api/v1/kong/test", method="GET"):
        with patch.object(consumers_mod, 'KongClient', return_value=mock_client):
            response = await consumers_mod.get_kong_consumer('c1')
            data = json.loads(await response.get_data(as_text=True))
            assert data['username'] == 'alice'


@pytest.mark.asyncio
async def test_create_kong_consumer(quart_app):
    mock_client = _mock_client()
    kong_result = {'id': 'c1', 'username': 'alice', 'custom_id': None}
    mock_client.create_consumer.return_value = kong_result
    consumers_mod = _fresh_import('app_quart.api.v1.kong.consumers')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()

    async with quart_app.test_request_context(
        '/api/v1/kong/consumers', method='POST', json={'username': 'alice'}
    ):
        with patch.object(consumers_mod, 'KongClient', return_value=mock_client):
            with patch.object(consumers_mod, 'db', mock_db):
                with patch.object(consumers_mod, 'current_user', mock_user):
                    with patch.object(consumers_mod.AuditService, 'log', AsyncMock()):
                        response, status = await consumers_mod.create_kong_consumer()
                        assert status == 201
                        data = json.loads(await response.get_data(as_text=True))
                        assert data['username'] == 'alice'


@pytest.mark.asyncio
async def test_update_kong_consumer(quart_app):
    mock_client = _mock_client()
    mock_client.get_consumer.return_value = {'id': 'c1', 'username': 'alice'}
    mock_client.update_consumer.return_value = {'id': 'c1', 'username': 'bob'}
    consumers_mod = _fresh_import('app_quart.api.v1.kong.consumers')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()
    consumers_mod.KongConsumer = MagicMock()
    consumers_mod.KongConsumer.query.filter_by.return_value.first.return_value = None

    async with quart_app.test_request_context(
        '/api/v1/kong/consumers/c1', method='PATCH', json={'username': 'bob'}
    ):
        with patch.object(consumers_mod, 'KongClient', return_value=mock_client):
            with patch.object(consumers_mod, 'db', mock_db):
                with patch.object(consumers_mod, 'current_user', mock_user):
                    with patch.object(consumers_mod.AuditService, 'log', AsyncMock()):
                        response = await consumers_mod.update_kong_consumer('c1')
                        data = json.loads(await response.get_data(as_text=True))
                        assert data['username'] == 'bob'


@pytest.mark.asyncio
async def test_delete_kong_consumer(quart_app):
    mock_client = _mock_client()
    mock_client.get_consumer.return_value = {'id': 'c1', 'username': 'alice'}
    consumers_mod = _fresh_import('app_quart.api.v1.kong.consumers')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()
    consumers_mod.KongConsumer = MagicMock()
    consumers_mod.KongConsumer.query.filter_by.return_value.first.return_value = None

    async with quart_app.test_request_context(
        '/api/v1/kong/consumers/c1', method='DELETE'
    ):
        with patch.object(consumers_mod, 'KongClient', return_value=mock_client):
            with patch.object(consumers_mod, 'db', mock_db):
                with patch.object(consumers_mod, 'current_user', mock_user):
                    with patch.object(consumers_mod.AuditService, 'log', AsyncMock()):
                        response, status = await consumers_mod.delete_kong_consumer('c1')
                        assert status == 204
                        mock_client.delete_consumer.assert_called_once_with('c1')


# ===========================================================================
# Kong PLUGINS handlers
# ===========================================================================

@pytest.mark.asyncio
async def test_list_kong_plugins(quart_app):
    mock_client = _mock_client()
    mock_client.list_plugins.return_value = {'data': [], 'total': 0}
    plugins_mod = _fresh_import('app_quart.api.v1.kong.plugins')

    async with quart_app.test_request_context("/api/v1/kong/test", method="GET"):
        with patch.object(plugins_mod, 'KongClient', return_value=mock_client):
            response = await plugins_mod.list_kong_plugins()
            data = json.loads(await response.get_data(as_text=True))
            assert 'data' in data


@pytest.mark.asyncio
async def test_list_enabled_plugins(quart_app):
    mock_client = _mock_client()
    mock_client.get_enabled_plugins.return_value = {'enabled_plugins': ['jwt', 'rate-limiting']}
    plugins_mod = _fresh_import('app_quart.api.v1.kong.plugins')

    async with quart_app.test_request_context("/api/v1/kong/test", method="GET"):
        with patch.object(plugins_mod, 'KongClient', return_value=mock_client):
            response = await plugins_mod.list_enabled_plugins()
            data = json.loads(await response.get_data(as_text=True))
            assert 'enabled_plugins' in data


@pytest.mark.asyncio
async def test_get_plugin_schema(quart_app):
    mock_client = _mock_client()
    mock_client.get_plugin_schema.return_value = {'fields': []}
    plugins_mod = _fresh_import('app_quart.api.v1.kong.plugins')

    async with quart_app.test_request_context("/api/v1/kong/test", method="GET"):
        with patch.object(plugins_mod, 'KongClient', return_value=mock_client):
            response = await plugins_mod.get_plugin_schema('jwt')
            data = json.loads(await response.get_data(as_text=True))
            assert 'fields' in data
            mock_client.get_plugin_schema.assert_called_once_with('jwt')


@pytest.mark.asyncio
async def test_get_kong_plugin(quart_app):
    mock_client = _mock_client()
    mock_client.get_plugin.return_value = {'id': 'p1', 'name': 'jwt', 'enabled': True}
    plugins_mod = _fresh_import('app_quart.api.v1.kong.plugins')

    async with quart_app.test_request_context("/api/v1/kong/test", method="GET"):
        with patch.object(plugins_mod, 'KongClient', return_value=mock_client):
            response = await plugins_mod.get_kong_plugin('p1')
            data = json.loads(await response.get_data(as_text=True))
            assert data['name'] == 'jwt'


@pytest.mark.asyncio
async def test_create_kong_plugin(quart_app):
    mock_client = _mock_client()
    kong_result = {'id': 'p1', 'name': 'jwt', 'enabled': True, 'protocols': ['http']}
    mock_client.create_plugin.return_value = kong_result
    plugins_mod = _fresh_import('app_quart.api.v1.kong.plugins')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()

    async with quart_app.test_request_context(
        '/api/v1/kong/plugins', method='POST', json={'name': 'jwt', 'config': {}}
    ):
        with patch.object(plugins_mod, 'KongClient', return_value=mock_client):
            with patch.object(plugins_mod, 'db', mock_db):
                with patch.object(plugins_mod, 'current_user', mock_user):
                    with patch.object(plugins_mod.AuditService, 'log', AsyncMock()):
                        response, status = await plugins_mod.create_kong_plugin()
                        assert status == 201
                        data = json.loads(await response.get_data(as_text=True))
                        assert data['name'] == 'jwt'


@pytest.mark.asyncio
async def test_update_kong_plugin(quart_app):
    mock_client = _mock_client()
    mock_client.get_plugin.return_value = {'id': 'p1', 'enabled': True}
    mock_client.update_plugin.return_value = {'id': 'p1', 'enabled': False}
    plugins_mod = _fresh_import('app_quart.api.v1.kong.plugins')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()
    plugins_mod.KongPlugin = MagicMock()
    plugins_mod.KongPlugin.query.filter_by.return_value.first.return_value = None

    async with quart_app.test_request_context(
        '/api/v1/kong/plugins/p1', method='PATCH', json={'enabled': False}
    ):
        with patch.object(plugins_mod, 'KongClient', return_value=mock_client):
            with patch.object(plugins_mod, 'db', mock_db):
                with patch.object(plugins_mod, 'current_user', mock_user):
                    with patch.object(plugins_mod.AuditService, 'log', AsyncMock()):
                        response = await plugins_mod.update_kong_plugin('p1')
                        data = json.loads(await response.get_data(as_text=True))
                        assert data['enabled'] is False


@pytest.mark.asyncio
async def test_delete_kong_plugin(quart_app):
    mock_client = _mock_client()
    mock_client.get_plugin.return_value = {'id': 'p1', 'name': 'jwt'}
    plugins_mod = _fresh_import('app_quart.api.v1.kong.plugins')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()
    plugins_mod.KongPlugin = MagicMock()
    plugins_mod.KongPlugin.query.filter_by.return_value.first.return_value = None

    async with quart_app.test_request_context(
        '/api/v1/kong/plugins/p1', method='DELETE'
    ):
        with patch.object(plugins_mod, 'KongClient', return_value=mock_client):
            with patch.object(plugins_mod, 'db', mock_db):
                with patch.object(plugins_mod, 'current_user', mock_user):
                    with patch.object(plugins_mod.AuditService, 'log', AsyncMock()):
                        response, status = await plugins_mod.delete_kong_plugin('p1')
                        assert status == 204
                        mock_client.delete_plugin.assert_called_once_with('p1')


# ===========================================================================
# Kong CERTIFICATES & SNIs handlers
# ===========================================================================

@pytest.mark.asyncio
async def test_list_kong_certificates(quart_app):
    mock_client = _mock_client()
    mock_client.list_certificates.return_value = {'data': [], 'total': 0}
    certs_mod = _fresh_import('app_quart.api.v1.kong.certificates')

    async with quart_app.test_request_context("/api/v1/kong/test", method="GET"):
        with patch.object(certs_mod, 'KongClient', return_value=mock_client):
            response = await certs_mod.list_kong_certificates()
            data = json.loads(await response.get_data(as_text=True))
            assert 'data' in data


@pytest.mark.asyncio
async def test_get_kong_certificate(quart_app):
    mock_client = _mock_client()
    mock_client.get_certificate.return_value = {'id': 'cert1', 'cert': '---BEGIN---'}
    certs_mod = _fresh_import('app_quart.api.v1.kong.certificates')

    async with quart_app.test_request_context("/api/v1/kong/test", method="GET"):
        with patch.object(certs_mod, 'KongClient', return_value=mock_client):
            response = await certs_mod.get_kong_certificate('cert1')
            data = json.loads(await response.get_data(as_text=True))
            assert data['id'] == 'cert1'


@pytest.mark.asyncio
async def test_create_kong_certificate(quart_app):
    mock_client = _mock_client()
    kong_result = {'id': 'cert1', 'cert': '---BEGIN---', 'key': '---KEY---'}
    mock_client.create_certificate.return_value = kong_result
    certs_mod = _fresh_import('app_quart.api.v1.kong.certificates')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()

    async with quart_app.test_request_context(
        '/api/v1/kong/certificates',
        method='POST',
        json={'cert': '---BEGIN---', 'key': '---KEY---'}
    ):
        with patch.object(certs_mod, 'KongClient', return_value=mock_client):
            with patch.object(certs_mod, 'db', mock_db):
                with patch.object(certs_mod, 'current_user', mock_user):
                    with patch.object(certs_mod.AuditService, 'log', AsyncMock()):
                        response, status = await certs_mod.create_kong_certificate()
                        assert status == 201
                        data = json.loads(await response.get_data(as_text=True))
                        assert data['id'] == 'cert1'


@pytest.mark.asyncio
async def test_update_kong_certificate(quart_app):
    mock_client = _mock_client()
    mock_client.update_certificate.return_value = {'id': 'cert1', 'cert': '---NEW---'}
    certs_mod = _fresh_import('app_quart.api.v1.kong.certificates')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()
    certs_mod.KongCertificate = MagicMock()
    certs_mod.KongCertificate.query.filter_by.return_value.first.return_value = None

    async with quart_app.test_request_context(
        '/api/v1/kong/certificates/cert1', method='PATCH', json={'cert': '---NEW---'}
    ):
        with patch.object(certs_mod, 'KongClient', return_value=mock_client):
            with patch.object(certs_mod, 'db', mock_db):
                with patch.object(certs_mod, 'current_user', mock_user):
                    with patch.object(certs_mod.AuditService, 'log', AsyncMock()):
                        response = await certs_mod.update_kong_certificate('cert1')
                        data = json.loads(await response.get_data(as_text=True))
                        assert data['cert'] == '---NEW---'


@pytest.mark.asyncio
async def test_delete_kong_certificate(quart_app):
    mock_client = _mock_client()
    certs_mod = _fresh_import('app_quart.api.v1.kong.certificates')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()
    certs_mod.KongCertificate = MagicMock()
    certs_mod.KongCertificate.query.filter_by.return_value.first.return_value = None

    async with quart_app.test_request_context(
        '/api/v1/kong/certificates/cert1', method='DELETE'
    ):
        with patch.object(certs_mod, 'KongClient', return_value=mock_client):
            with patch.object(certs_mod, 'db', mock_db):
                with patch.object(certs_mod, 'current_user', mock_user):
                    with patch.object(certs_mod.AuditService, 'log', AsyncMock()):
                        response, status = await certs_mod.delete_kong_certificate('cert1')
                        assert status == 204
                        mock_client.delete_certificate.assert_called_once_with('cert1')


@pytest.mark.asyncio
async def test_list_kong_snis(quart_app):
    mock_client = _mock_client()
    mock_client.list_snis.return_value = {'data': [], 'total': 0}
    certs_mod = _fresh_import('app_quart.api.v1.kong.certificates')

    async with quart_app.test_request_context("/api/v1/kong/test", method="GET"):
        with patch.object(certs_mod, 'KongClient', return_value=mock_client):
            response = await certs_mod.list_kong_snis()
            data = json.loads(await response.get_data(as_text=True))
            assert 'data' in data


@pytest.mark.asyncio
async def test_create_kong_sni(quart_app):
    mock_client = _mock_client()
    kong_result = {'id': 'sni1', 'name': 'api.example.com'}
    mock_client.create_sni.return_value = kong_result
    certs_mod = _fresh_import('app_quart.api.v1.kong.certificates')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()
    certs_mod.KongCertificate = MagicMock()
    certs_mod.KongCertificate.query.filter_by.return_value.first.return_value = None

    async with quart_app.test_request_context(
        '/api/v1/kong/snis',
        method='POST',
        json={'name': 'api.example.com', 'certificate': {'id': 'cert1'}}
    ):
        with patch.object(certs_mod, 'KongClient', return_value=mock_client):
            with patch.object(certs_mod, 'db', mock_db):
                with patch.object(certs_mod, 'current_user', mock_user):
                    with patch.object(certs_mod.AuditService, 'log', AsyncMock()):
                        response, status = await certs_mod.create_kong_sni()
                        assert status == 201
                        data = json.loads(await response.get_data(as_text=True))
                        assert data['name'] == 'api.example.com'


@pytest.mark.asyncio
async def test_delete_kong_sni(quart_app):
    mock_client = _mock_client()
    certs_mod = _fresh_import('app_quart.api.v1.kong.certificates')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()
    certs_mod.KongSNI = MagicMock()
    certs_mod.KongSNI.query.filter_by.return_value.first.return_value = None

    async with quart_app.test_request_context(
        '/api/v1/kong/snis/sni1', method='DELETE'
    ):
        with patch.object(certs_mod, 'KongClient', return_value=mock_client):
            with patch.object(certs_mod, 'db', mock_db):
                with patch.object(certs_mod, 'current_user', mock_user):
                    with patch.object(certs_mod.AuditService, 'log', AsyncMock()):
                        response, status = await certs_mod.delete_kong_sni('sni1')
                        assert status == 204
                        mock_client.delete_sni.assert_called_once_with('sni1')


# ===========================================================================
# Kong CONFIG handlers
# ===========================================================================

@pytest.mark.asyncio
async def test_get_kong_config_returns_yaml(quart_app):
    mock_client = _mock_client()
    for method in ['list_services', 'list_routes', 'list_upstreams',
                   'list_consumers', 'list_plugins', 'list_certificates']:
        getattr(mock_client, method).return_value = {'data': []}
    config_mod = _fresh_import('app_quart.api.v1.kong.config')
    mock_user = _make_mock_user()

    async with quart_app.test_request_context('/api/v1/kong/config', method='GET'):
        with patch.object(config_mod, 'KongClient', return_value=mock_client):
            with patch.object(config_mod, 'current_user', mock_user):
                yaml_content, status, headers = await config_mod.get_kong_config()
                assert status == 200
                assert '_format_version' in yaml_content
                assert headers['Content-Type'] == 'text/yaml'


@pytest.mark.asyncio
async def test_validate_kong_config_valid_yaml(quart_app):
    config_mod = _fresh_import('app_quart.api.v1.kong.config')
    mock_user = _make_mock_user()
    valid_yaml = '_format_version: "3.0"\nservices: []\nroutes: []\n'

    async with quart_app.test_request_context(
        '/api/v1/kong/config/validate',
        method='POST',
        data=valid_yaml,
        headers={'Content-Type': 'text/yaml'}
    ):
        with patch.object(config_mod, 'current_user', mock_user):
            response = await config_mod.validate_kong_config()
            data = json.loads(await response.get_data(as_text=True))
            assert data['valid'] is True
            assert data['format_version'] == '3.0'


@pytest.mark.asyncio
async def test_validate_kong_config_invalid_yaml(quart_app):
    config_mod = _fresh_import('app_quart.api.v1.kong.config')

    async with quart_app.test_request_context(
        '/api/v1/kong/config/validate',
        method='POST',
        json={'config': '{invalid yaml:::'}
    ):
        response, status = await config_mod.validate_kong_config()
        data = json.loads(await response.get_data(as_text=True))
        assert status == 400
        assert data['valid'] is False


@pytest.mark.asyncio
async def test_validate_kong_config_missing_format_version(quart_app):
    config_mod = _fresh_import('app_quart.api.v1.kong.config')

    async with quart_app.test_request_context(
        '/api/v1/kong/config/validate',
        method='POST',
        json={'config': 'services: []\n'}
    ):
        response, status = await config_mod.validate_kong_config()
        data = json.loads(await response.get_data(as_text=True))
        assert status == 400
        assert data['valid'] is False
        assert 'format_version' in data['error']


@pytest.mark.asyncio
async def test_validate_kong_config_counts_entities(quart_app):
    config_mod = _fresh_import('app_quart.api.v1.kong.config')
    yaml_with_items = (
        '_format_version: "3.0"\n'
        'services:\n  - name: svc1\n  - name: svc2\n'
        'routes:\n  - name: r1\n'
    )

    async with quart_app.test_request_context(
        '/api/v1/kong/config/validate',
        method='POST',
        data=yaml_with_items,
        headers={'Content-Type': 'text/yaml'}
    ):
        response = await config_mod.validate_kong_config()
        data = json.loads(await response.get_data(as_text=True))
        assert data['stats']['services'] == 2
        assert data['stats']['routes'] == 1


@pytest.mark.asyncio
async def test_apply_kong_config_success(quart_app):
    mock_client = _mock_client()
    mock_client.post_config.return_value = {'result': 'ok'}
    config_mod = _fresh_import('app_quart.api.v1.kong.config')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()

    valid_yaml = '_format_version: "3.0"\nservices: []\nroutes: []\nplugins: []\n'

    # Mock KongConfigHistory
    mock_history = MagicMock()
    mock_history.id = 1
    mock_history.services_count = 0
    mock_history.routes_count = 0
    mock_history.plugins_count = 0
    config_mod.KongConfigHistory = MagicMock(return_value=mock_history)
    config_mod.KongConfigHistory.query = MagicMock()
    config_mod.KongConfigHistory.query.filter_by.return_value.update = MagicMock()

    async with quart_app.test_request_context(
        '/api/v1/kong/config',
        method='POST',
        data=valid_yaml,
        headers={'Content-Type': 'text/yaml'}
    ):
        with patch.object(config_mod, 'KongClient', return_value=mock_client):
            with patch.object(config_mod, 'db', mock_db):
                with patch.object(config_mod, 'current_user', mock_user):
                    with patch.object(config_mod.AuditService, 'log', AsyncMock()):
                        response = await config_mod.apply_kong_config()
                        data = json.loads(await response.get_data(as_text=True))
                        assert data['success'] is True
                        assert 'hash' in data


@pytest.mark.asyncio
async def test_apply_kong_config_invalid_yaml(quart_app):
    config_mod = _fresh_import('app_quart.api.v1.kong.config')

    async with quart_app.test_request_context(
        '/api/v1/kong/config',
        method='POST',
        json={'config': '{broken yaml:::'}
    ):
        response, status = await config_mod.apply_kong_config()
        assert status == 400
        data = json.loads(await response.get_data(as_text=True))
        assert 'error' in data


@pytest.mark.asyncio
async def test_get_kong_status(quart_app):
    mock_client = _mock_client()
    mock_client.get_status.return_value = {'version': '3.0', 'hostname': 'kong-1'}
    config_mod = _fresh_import('app_quart.api.v1.kong.config')
    mock_user = _make_mock_user()

    async with quart_app.test_request_context('/api/v1/kong/status', method='GET'):
        with patch.object(config_mod, 'KongClient', return_value=mock_client):
            with patch.object(config_mod, 'current_user', mock_user):
                response = await config_mod.get_kong_status()
                data = json.loads(await response.get_data(as_text=True))
                assert data['version'] == '3.0'


@pytest.mark.asyncio
async def test_preview_kong_config(quart_app):
    mock_client = _mock_client()
    for method in ['list_services', 'list_routes', 'list_upstreams',
                   'list_consumers', 'list_plugins']:
        getattr(mock_client, method).return_value = {'data': []}
    config_mod = _fresh_import('app_quart.api.v1.kong.config')
    mock_user = _make_mock_user()
    yaml_input = '_format_version: "3.0"\nservices:\n  - name: new-svc\n'

    async with quart_app.test_request_context(
        '/api/v1/kong/config/preview',
        method='POST',
        data=yaml_input,
        headers={'Content-Type': 'text/yaml'}
    ):
        with patch.object(config_mod, 'KongClient', return_value=mock_client):
            with patch.object(config_mod, 'current_user', mock_user):
                response = await config_mod.preview_kong_config()
                data = json.loads(await response.get_data(as_text=True))
                assert 'services' in data
                assert 'added' in data['services']
                assert 'new-svc' in data['services']['added']


@pytest.mark.asyncio
async def test_preview_kong_config_invalid_yaml(quart_app):
    config_mod = _fresh_import('app_quart.api.v1.kong.config')

    async with quart_app.test_request_context(
        '/api/v1/kong/config/preview',
        method='POST',
        json={'config': '{bad yaml:::'}
    ):
        response, status = await config_mod.preview_kong_config()
        assert status == 400


@pytest.mark.asyncio
async def test_list_config_history(quart_app):
    """list_config_history returns paginated history list."""
    config_mod = _fresh_import('app_quart.api.v1.kong.config')

    mock_config1 = MagicMock()
    mock_config1.id = 1
    mock_config1.description = 'Initial'
    mock_config1.applied_at = MagicMock()
    mock_config1.applied_at.isoformat.return_value = '2024-01-01T00:00:00'
    mock_config1.applied_by = 1
    mock_config1.is_current = True
    mock_config1.services_count = 3
    mock_config1.routes_count = 5
    mock_config1.plugins_count = 2
    mock_config1.config_hash = 'abc123'

    mock_query = MagicMock()
    mock_query.order_by.return_value = mock_query
    mock_query.count.return_value = 1
    mock_query.offset.return_value = mock_query
    mock_query.limit.return_value = mock_query
    mock_query.all.return_value = [mock_config1]

    config_mod.KongConfigHistory = MagicMock()
    config_mod.KongConfigHistory.query = mock_query
    config_mod.KongConfigHistory.applied_at = MagicMock()

    async with quart_app.test_request_context('/api/v1/kong/config/history', method='GET'):
        response = await config_mod.list_config_history()
        data = json.loads(await response.get_data(as_text=True))
        assert data['total'] == 1
        assert len(data['data']) == 1
        assert data['data'][0]['id'] == 1
        assert data['data'][0]['is_current'] is True


@pytest.mark.asyncio
async def test_get_config_history_returns_yaml(quart_app):
    """get_config_history returns YAML content for a history entry."""
    config_mod = _fresh_import('app_quart.api.v1.kong.config')

    mock_history = MagicMock()
    mock_history.config_yaml = '_format_version: "3.0"\nservices: []\n'
    config_mod.KongConfigHistory = MagicMock()
    config_mod.KongConfigHistory.query = MagicMock()
    config_mod.KongConfigHistory.query.get_or_404.return_value = mock_history

    async with quart_app.test_request_context('/api/v1/kong/config/history/1', method='GET'):
        yaml_content, status, headers = await config_mod.get_config_history(1)
        assert status == 200
        assert '_format_version' in yaml_content
        assert headers['Content-Type'] == 'text/yaml'


@pytest.mark.asyncio
async def test_rollback_config_success(quart_app):
    """rollback_config applies historical config and marks it current."""
    mock_client = _mock_client()
    mock_client.post_config.return_value = {'result': 'ok'}
    config_mod = _fresh_import('app_quart.api.v1.kong.config')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()

    mock_history = MagicMock()
    mock_history.config_yaml = '_format_version: "3.0"\nservices: []\n'
    mock_history.is_current = False
    config_mod.KongConfigHistory = MagicMock()
    config_mod.KongConfigHistory.query = MagicMock()
    config_mod.KongConfigHistory.query.get_or_404.return_value = mock_history
    config_mod.KongConfigHistory.query.filter_by.return_value.update = MagicMock()

    async with quart_app.test_request_context('/api/v1/kong/config/rollback/1', method='POST'):
        with patch.object(config_mod, 'KongClient', return_value=mock_client):
            with patch.object(config_mod, 'db', mock_db):
                with patch.object(config_mod, 'current_user', mock_user):
                    with patch.object(config_mod.AuditService, 'log', AsyncMock()):
                        response = await config_mod.rollback_config(1)
                        data = json.loads(await response.get_data(as_text=True))
                        assert data['success'] is True
                        assert data['rolled_back_to'] == 1
                        mock_client.post_config.assert_called_once_with(mock_history.config_yaml)


# ===========================================================================
# Branch coverage: DB record exists on update (sets attrs and commits)
# ===========================================================================

@pytest.mark.asyncio
async def test_update_kong_route_with_db_record(quart_app):
    """update_kong_route updates DB record attrs when record exists."""
    mock_client = _mock_client()
    mock_client.get_route.return_value = {'id': 'r1', 'name': 'old'}
    mock_client.update_route.return_value = {'id': 'r1', 'name': 'updated', 'paths': ['/new']}
    routes_mod = _fresh_import('app_quart.api.v1.kong.routes')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()
    mock_db_route = MagicMock()
    mock_db_route.name = 'old'
    routes_mod.KongRoute = MagicMock()
    routes_mod.KongRoute.query.filter_by.return_value.first.return_value = mock_db_route

    async with quart_app.test_request_context(
        '/api/v1/kong/routes/r1', method='PATCH', json={'name': 'updated'}
    ):
        with patch.object(routes_mod, 'KongClient', return_value=mock_client):
            with patch.object(routes_mod, 'db', mock_db):
                with patch.object(routes_mod, 'current_user', mock_user):
                    with patch.object(routes_mod.AuditService, 'log', AsyncMock()):
                        await routes_mod.update_kong_route('r1')
                        mock_db.session.commit.assert_called_once()


@pytest.mark.asyncio
async def test_delete_kong_route_with_db_record(quart_app):
    """delete_kong_route deletes DB record when it exists."""
    mock_client = _mock_client()
    mock_client.get_route.return_value = {'id': 'r1', 'name': 'my-route'}
    routes_mod = _fresh_import('app_quart.api.v1.kong.routes')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()
    mock_db_route = MagicMock()
    routes_mod.KongRoute = MagicMock()
    routes_mod.KongRoute.query.filter_by.return_value.first.return_value = mock_db_route

    async with quart_app.test_request_context(
        '/api/v1/kong/routes/r1', method='DELETE'
    ):
        with patch.object(routes_mod, 'KongClient', return_value=mock_client):
            with patch.object(routes_mod, 'db', mock_db):
                with patch.object(routes_mod, 'current_user', mock_user):
                    with patch.object(routes_mod.AuditService, 'log', AsyncMock()):
                        response, status = await routes_mod.delete_kong_route('r1')
                        assert status == 204
                        mock_db.session.delete.assert_called_once_with(mock_db_route)


@pytest.mark.asyncio
async def test_update_kong_consumer_with_db_record(quart_app):
    """update_kong_consumer updates DB record when it exists."""
    mock_client = _mock_client()
    mock_client.get_consumer.return_value = {'id': 'c1', 'username': 'alice'}
    mock_client.update_consumer.return_value = {'id': 'c1', 'username': 'bob'}
    consumers_mod = _fresh_import('app_quart.api.v1.kong.consumers')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()
    mock_db_consumer = MagicMock()
    consumers_mod.KongConsumer = MagicMock()
    consumers_mod.KongConsumer.query.filter_by.return_value.first.return_value = mock_db_consumer

    async with quart_app.test_request_context(
        '/api/v1/kong/consumers/c1', method='PATCH', json={'username': 'bob'}
    ):
        with patch.object(consumers_mod, 'KongClient', return_value=mock_client):
            with patch.object(consumers_mod, 'db', mock_db):
                with patch.object(consumers_mod, 'current_user', mock_user):
                    with patch.object(consumers_mod.AuditService, 'log', AsyncMock()):
                        await consumers_mod.update_kong_consumer('c1')
                        mock_db.session.commit.assert_called_once()


@pytest.mark.asyncio
async def test_delete_kong_consumer_with_db_record(quart_app):
    """delete_kong_consumer deletes DB record when it exists."""
    mock_client = _mock_client()
    mock_client.get_consumer.return_value = {'id': 'c1', 'username': 'alice'}
    consumers_mod = _fresh_import('app_quart.api.v1.kong.consumers')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()
    mock_db_consumer = MagicMock()
    consumers_mod.KongConsumer = MagicMock()
    consumers_mod.KongConsumer.query.filter_by.return_value.first.return_value = mock_db_consumer

    async with quart_app.test_request_context(
        '/api/v1/kong/consumers/c1', method='DELETE'
    ):
        with patch.object(consumers_mod, 'KongClient', return_value=mock_client):
            with patch.object(consumers_mod, 'db', mock_db):
                with patch.object(consumers_mod, 'current_user', mock_user):
                    with patch.object(consumers_mod.AuditService, 'log', AsyncMock()):
                        response, status = await consumers_mod.delete_kong_consumer('c1')
                        assert status == 204
                        mock_db.session.delete.assert_called_once_with(mock_db_consumer)


@pytest.mark.asyncio
async def test_update_kong_plugin_with_db_record(quart_app):
    """update_kong_plugin updates DB record when it exists."""
    mock_client = _mock_client()
    mock_client.get_plugin.return_value = {'id': 'p1', 'enabled': True}
    mock_client.update_plugin.return_value = {'id': 'p1', 'enabled': False}
    plugins_mod = _fresh_import('app_quart.api.v1.kong.plugins')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()
    mock_db_plugin = MagicMock()
    plugins_mod.KongPlugin = MagicMock()
    plugins_mod.KongPlugin.query.filter_by.return_value.first.return_value = mock_db_plugin

    async with quart_app.test_request_context(
        '/api/v1/kong/plugins/p1', method='PATCH', json={'enabled': False}
    ):
        with patch.object(plugins_mod, 'KongClient', return_value=mock_client):
            with patch.object(plugins_mod, 'db', mock_db):
                with patch.object(plugins_mod, 'current_user', mock_user):
                    with patch.object(plugins_mod.AuditService, 'log', AsyncMock()):
                        await plugins_mod.update_kong_plugin('p1')
                        mock_db.session.commit.assert_called_once()


@pytest.mark.asyncio
async def test_delete_kong_plugin_with_db_record(quart_app):
    """delete_kong_plugin deletes DB record when it exists."""
    mock_client = _mock_client()
    mock_client.get_plugin.return_value = {'id': 'p1', 'name': 'jwt'}
    plugins_mod = _fresh_import('app_quart.api.v1.kong.plugins')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()
    mock_db_plugin = MagicMock()
    plugins_mod.KongPlugin = MagicMock()
    plugins_mod.KongPlugin.query.filter_by.return_value.first.return_value = mock_db_plugin

    async with quart_app.test_request_context(
        '/api/v1/kong/plugins/p1', method='DELETE'
    ):
        with patch.object(plugins_mod, 'KongClient', return_value=mock_client):
            with patch.object(plugins_mod, 'db', mock_db):
                with patch.object(plugins_mod, 'current_user', mock_user):
                    with patch.object(plugins_mod.AuditService, 'log', AsyncMock()):
                        response, status = await plugins_mod.delete_kong_plugin('p1')
                        assert status == 204
                        mock_db.session.delete.assert_called_once_with(mock_db_plugin)


@pytest.mark.asyncio
async def test_update_kong_upstream_with_db_record(quart_app):
    """update_kong_upstream updates DB record when it exists."""
    mock_client = _mock_client()
    mock_client.get_upstream.return_value = {'id': 'u1', 'name': 'old'}
    mock_client.update_upstream.return_value = {'id': 'u1', 'name': 'new'}
    upstreams_mod = _fresh_import('app_quart.api.v1.kong.upstreams')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()
    mock_db_upstream = MagicMock()
    upstreams_mod.KongUpstream = MagicMock()
    upstreams_mod.KongUpstream.query.filter_by.return_value.first.return_value = mock_db_upstream

    async with quart_app.test_request_context(
        '/api/v1/kong/upstreams/u1', method='PATCH', json={'name': 'new'}
    ):
        with patch.object(upstreams_mod, 'KongClient', return_value=mock_client):
            with patch.object(upstreams_mod, 'db', mock_db):
                with patch.object(upstreams_mod, 'current_user', mock_user):
                    with patch.object(upstreams_mod.AuditService, 'log', AsyncMock()):
                        await upstreams_mod.update_kong_upstream('u1')
                        mock_db.session.commit.assert_called_once()


@pytest.mark.asyncio
async def test_delete_kong_upstream_with_db_record(quart_app):
    """delete_kong_upstream deletes DB record when it exists."""
    mock_client = _mock_client()
    mock_client.get_upstream.return_value = {'id': 'u1', 'name': 'up1'}
    upstreams_mod = _fresh_import('app_quart.api.v1.kong.upstreams')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()
    mock_db_upstream = MagicMock()
    upstreams_mod.KongUpstream = MagicMock()
    upstreams_mod.KongUpstream.query.filter_by.return_value.first.return_value = mock_db_upstream

    async with quart_app.test_request_context(
        '/api/v1/kong/upstreams/u1', method='DELETE'
    ):
        with patch.object(upstreams_mod, 'KongClient', return_value=mock_client):
            with patch.object(upstreams_mod, 'db', mock_db):
                with patch.object(upstreams_mod, 'current_user', mock_user):
                    with patch.object(upstreams_mod.AuditService, 'log', AsyncMock()):
                        response, status = await upstreams_mod.delete_kong_upstream('u1')
                        assert status == 204
                        mock_db.session.delete.assert_called_once_with(mock_db_upstream)


@pytest.mark.asyncio
async def test_update_kong_certificate_with_db_record(quart_app):
    """update_kong_certificate updates DB record when it exists."""
    mock_client = _mock_client()
    mock_client.update_certificate.return_value = {'id': 'cert1', 'cert': '---NEW---'}
    certs_mod = _fresh_import('app_quart.api.v1.kong.certificates')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()
    mock_db_cert = MagicMock()
    certs_mod.KongCertificate = MagicMock()
    certs_mod.KongCertificate.query.filter_by.return_value.first.return_value = mock_db_cert

    async with quart_app.test_request_context(
        '/api/v1/kong/certificates/cert1', method='PATCH', json={'cert': '---NEW---'}
    ):
        with patch.object(certs_mod, 'KongClient', return_value=mock_client):
            with patch.object(certs_mod, 'db', mock_db):
                with patch.object(certs_mod, 'current_user', mock_user):
                    with patch.object(certs_mod.AuditService, 'log', AsyncMock()):
                        await certs_mod.update_kong_certificate('cert1')
                        mock_db.session.commit.assert_called_once()


@pytest.mark.asyncio
async def test_delete_kong_certificate_with_db_record(quart_app):
    """delete_kong_certificate deletes DB record when it exists."""
    mock_client = _mock_client()
    certs_mod = _fresh_import('app_quart.api.v1.kong.certificates')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()
    mock_db_cert = MagicMock()
    certs_mod.KongCertificate = MagicMock()
    certs_mod.KongCertificate.query.filter_by.return_value.first.return_value = mock_db_cert

    async with quart_app.test_request_context(
        '/api/v1/kong/certificates/cert1', method='DELETE'
    ):
        with patch.object(certs_mod, 'KongClient', return_value=mock_client):
            with patch.object(certs_mod, 'db', mock_db):
                with patch.object(certs_mod, 'current_user', mock_user):
                    with patch.object(certs_mod.AuditService, 'log', AsyncMock()):
                        response, status = await certs_mod.delete_kong_certificate('cert1')
                        assert status == 204
                        mock_db.session.delete.assert_called_once_with(mock_db_cert)


@pytest.mark.asyncio
async def test_delete_kong_sni_with_db_record(quart_app):
    """delete_kong_sni deletes DB record when it exists."""
    mock_client = _mock_client()
    certs_mod = _fresh_import('app_quart.api.v1.kong.certificates')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()
    mock_db_sni = MagicMock()
    certs_mod.KongSNI = MagicMock()
    certs_mod.KongSNI.query.filter_by.return_value.first.return_value = mock_db_sni

    async with quart_app.test_request_context(
        '/api/v1/kong/snis/sni1', method='DELETE'
    ):
        with patch.object(certs_mod, 'KongClient', return_value=mock_client):
            with patch.object(certs_mod, 'db', mock_db):
                with patch.object(certs_mod, 'current_user', mock_user):
                    with patch.object(certs_mod.AuditService, 'log', AsyncMock()):
                        response, status = await certs_mod.delete_kong_sni('sni1')
                        assert status == 204
                        mock_db.session.delete.assert_called_once_with(mock_db_sni)


@pytest.mark.asyncio
async def test_create_kong_target_with_db_upstream(quart_app):
    """create_kong_target sets upstream_id from DB upstream when it exists."""
    mock_client = _mock_client()
    kong_result = {'id': 't1', 'target': '10.0.0.1:80', 'weight': 100}
    mock_client.create_target.return_value = kong_result
    upstreams_mod = _fresh_import('app_quart.api.v1.kong.upstreams')
    mock_db = _make_mock_db()
    mock_user = _make_mock_user()
    mock_db_upstream = MagicMock()
    mock_db_upstream.id = 42
    upstreams_mod.KongUpstream = MagicMock()
    upstreams_mod.KongUpstream.query.filter_by.return_value.first.return_value = mock_db_upstream

    async with quart_app.test_request_context(
        '/api/v1/kong/upstreams/u1/targets',
        method='POST',
        json={'target': '10.0.0.1:80'}
    ):
        with patch.object(upstreams_mod, 'KongClient', return_value=mock_client):
            with patch.object(upstreams_mod, 'db', mock_db):
                with patch.object(upstreams_mod, 'current_user', mock_user):
                    with patch.object(upstreams_mod.AuditService, 'log', AsyncMock()):
                        response, status = await upstreams_mod.create_kong_target('u1')
                        assert status == 201
                        # KongTarget should have been created with upstream_id=42
                        mock_db.session.add.assert_called_once()
