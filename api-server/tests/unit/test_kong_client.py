"""Unit tests for app_quart/services/kong_client.py."""
import pytest
from unittest.mock import AsyncMock, MagicMock, patch


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------

@pytest.fixture
def mock_response():
    """Build a mock httpx response."""
    def _make(json_data=None, text_data=None, status_code=200):
        resp = MagicMock()
        resp.status_code = status_code
        if json_data is not None:
            resp.json.return_value = json_data
        if text_data is not None:
            resp.text = text_data
        resp.raise_for_status = MagicMock()
        return resp
    return _make


@pytest.fixture
def client_with_mock(mock_response):
    """Return a KongClient whose internal httpx.AsyncClient is fully mocked."""
    from app_quart.services.kong_client import KongClient

    with patch('httpx.AsyncClient') as MockAsyncClient:
        inner = MagicMock()
        inner.get = AsyncMock()
        inner.post = AsyncMock()
        inner.patch = AsyncMock()
        inner.delete = AsyncMock()
        inner.aclose = AsyncMock()
        MockAsyncClient.return_value = inner

        client = KongClient(base_url='http://mock-kong:8001')
        yield client, inner, mock_response


# ---------------------------------------------------------------------------
# __init__ / close
# ---------------------------------------------------------------------------

def test_kong_client_uses_provided_base_url():
    from app_quart.services.kong_client import KongClient
    with patch('httpx.AsyncClient') as MockAC:
        MockAC.return_value = MagicMock()
        c = KongClient(base_url='http://custom:9999')
        assert c.base_url == 'http://custom:9999'


def test_kong_client_uses_config_url_when_none_provided():
    from app_quart.services.kong_client import KongClient
    from app_quart.config import config
    with patch('httpx.AsyncClient') as MockAC:
        MockAC.return_value = MagicMock()
        c = KongClient()
        assert c.base_url == config.KONG_ADMIN_URL


@pytest.mark.asyncio
async def test_close_calls_aclose(client_with_mock):
    client, inner, _ = client_with_mock
    await client.close()
    inner.aclose.assert_called_once()


# ---------------------------------------------------------------------------
# Status
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_get_status(client_with_mock):
    client, inner, mock_response = client_with_mock
    inner.get.return_value = mock_response({'server': 'kong', 'version': '3.0'})
    result = await client.get_status()
    inner.get.assert_called_once_with('/status')
    assert result['version'] == '3.0'


# ---------------------------------------------------------------------------
# Services
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_list_services(client_with_mock):
    client, inner, mr = client_with_mock
    inner.get.return_value = mr({'data': [], 'total': 0})
    result = await client.list_services(offset=5, size=50)
    inner.get.assert_called_once_with('/services', params={'offset': 5, 'size': 50})
    assert result['total'] == 0


@pytest.mark.asyncio
async def test_get_service(client_with_mock):
    client, inner, mr = client_with_mock
    inner.get.return_value = mr({'id': 'abc', 'name': 'svc1'})
    result = await client.get_service('abc')
    inner.get.assert_called_once_with('/services/abc')
    assert result['name'] == 'svc1'


@pytest.mark.asyncio
async def test_create_service(client_with_mock):
    client, inner, mr = client_with_mock
    inner.post.return_value = mr({'id': 'abc', 'name': 'svc1', 'host': 'example.com'})
    data = {'name': 'svc1', 'host': 'example.com', 'protocol': 'http'}
    result = await client.create_service(data)
    inner.post.assert_called_once_with('/services', json=data)
    assert result['id'] == 'abc'


@pytest.mark.asyncio
async def test_update_service(client_with_mock):
    client, inner, mr = client_with_mock
    inner.patch.return_value = mr({'id': 'abc', 'name': 'updated'})
    result = await client.update_service('abc', {'name': 'updated'})
    inner.patch.assert_called_once_with('/services/abc', json={'name': 'updated'})
    assert result['name'] == 'updated'


@pytest.mark.asyncio
async def test_delete_service(client_with_mock):
    client, inner, mr = client_with_mock
    inner.delete.return_value = mr(status_code=204)
    await client.delete_service('abc')
    inner.delete.assert_called_once_with('/services/abc')


# ---------------------------------------------------------------------------
# Routes
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_list_routes(client_with_mock):
    client, inner, mr = client_with_mock
    inner.get.return_value = mr({'data': [{'id': 'r1'}], 'total': 1})
    result = await client.list_routes(offset=0, size=10)
    inner.get.assert_called_once_with('/routes', params={'offset': 0, 'size': 10})
    assert result['total'] == 1


@pytest.mark.asyncio
async def test_get_route(client_with_mock):
    client, inner, mr = client_with_mock
    inner.get.return_value = mr({'id': 'r1', 'name': 'route1'})
    result = await client.get_route('r1')
    inner.get.assert_called_once_with('/routes/r1')
    assert result['name'] == 'route1'


@pytest.mark.asyncio
async def test_create_route(client_with_mock):
    client, inner, mr = client_with_mock
    inner.post.return_value = mr({'id': 'r1', 'name': 'route1'})
    data = {'name': 'route1', 'paths': ['/api']}
    result = await client.create_route(data)
    inner.post.assert_called_once_with('/routes', json=data)
    assert result['id'] == 'r1'


@pytest.mark.asyncio
async def test_update_route(client_with_mock):
    client, inner, mr = client_with_mock
    inner.patch.return_value = mr({'id': 'r1', 'name': 'updated-route'})
    result = await client.update_route('r1', {'name': 'updated-route'})
    inner.patch.assert_called_once_with('/routes/r1', json={'name': 'updated-route'})
    assert result['name'] == 'updated-route'


@pytest.mark.asyncio
async def test_delete_route(client_with_mock):
    client, inner, mr = client_with_mock
    inner.delete.return_value = mr(status_code=204)
    await client.delete_route('r1')
    inner.delete.assert_called_once_with('/routes/r1')


# ---------------------------------------------------------------------------
# Upstreams
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_list_upstreams(client_with_mock):
    client, inner, mr = client_with_mock
    inner.get.return_value = mr({'data': [], 'total': 0})
    result = await client.list_upstreams()
    inner.get.assert_called_once_with('/upstreams')
    assert 'data' in result


@pytest.mark.asyncio
async def test_get_upstream(client_with_mock):
    client, inner, mr = client_with_mock
    inner.get.return_value = mr({'id': 'u1', 'name': 'upstream1'})
    result = await client.get_upstream('u1')
    inner.get.assert_called_once_with('/upstreams/u1')
    assert result['name'] == 'upstream1'


@pytest.mark.asyncio
async def test_create_upstream(client_with_mock):
    client, inner, mr = client_with_mock
    inner.post.return_value = mr({'id': 'u1', 'name': 'upstream1'})
    data = {'name': 'upstream1'}
    result = await client.create_upstream(data)
    inner.post.assert_called_once_with('/upstreams', json=data)
    assert result['id'] == 'u1'


@pytest.mark.asyncio
async def test_update_upstream(client_with_mock):
    client, inner, mr = client_with_mock
    inner.patch.return_value = mr({'id': 'u1', 'name': 'updated'})
    result = await client.update_upstream('u1', {'name': 'updated'})
    inner.patch.assert_called_once_with('/upstreams/u1', json={'name': 'updated'})
    assert result['name'] == 'updated'


@pytest.mark.asyncio
async def test_delete_upstream(client_with_mock):
    client, inner, mr = client_with_mock
    inner.delete.return_value = mr(status_code=204)
    await client.delete_upstream('u1')
    inner.delete.assert_called_once_with('/upstreams/u1')


# ---------------------------------------------------------------------------
# Targets
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_list_targets(client_with_mock):
    client, inner, mr = client_with_mock
    inner.get.return_value = mr({'data': [{'id': 't1'}], 'total': 1})
    result = await client.list_targets('u1')
    inner.get.assert_called_once_with('/upstreams/u1/targets')
    assert result['total'] == 1


@pytest.mark.asyncio
async def test_create_target(client_with_mock):
    client, inner, mr = client_with_mock
    inner.post.return_value = mr({'id': 't1', 'target': '10.0.0.1:80'})
    data = {'target': '10.0.0.1:80', 'weight': 100}
    result = await client.create_target('u1', data)
    inner.post.assert_called_once_with('/upstreams/u1/targets', json=data)
    assert result['target'] == '10.0.0.1:80'


@pytest.mark.asyncio
async def test_delete_target(client_with_mock):
    client, inner, mr = client_with_mock
    inner.delete.return_value = mr(status_code=204)
    await client.delete_target('u1', 't1')
    inner.delete.assert_called_once_with('/upstreams/u1/targets/t1')


# ---------------------------------------------------------------------------
# Consumers
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_list_consumers(client_with_mock):
    client, inner, mr = client_with_mock
    inner.get.return_value = mr({'data': [], 'total': 0})
    result = await client.list_consumers()
    inner.get.assert_called_once_with('/consumers')
    assert 'data' in result


@pytest.mark.asyncio
async def test_get_consumer(client_with_mock):
    client, inner, mr = client_with_mock
    inner.get.return_value = mr({'id': 'c1', 'username': 'bob'})
    result = await client.get_consumer('bob')
    inner.get.assert_called_once_with('/consumers/bob')
    assert result['username'] == 'bob'


@pytest.mark.asyncio
async def test_create_consumer(client_with_mock):
    client, inner, mr = client_with_mock
    inner.post.return_value = mr({'id': 'c1', 'username': 'bob'})
    data = {'username': 'bob'}
    result = await client.create_consumer(data)
    inner.post.assert_called_once_with('/consumers', json=data)
    assert result['id'] == 'c1'


@pytest.mark.asyncio
async def test_update_consumer(client_with_mock):
    client, inner, mr = client_with_mock
    inner.patch.return_value = mr({'id': 'c1', 'username': 'alice'})
    result = await client.update_consumer('c1', {'username': 'alice'})
    inner.patch.assert_called_once_with('/consumers/c1', json={'username': 'alice'})
    assert result['username'] == 'alice'


@pytest.mark.asyncio
async def test_delete_consumer(client_with_mock):
    client, inner, mr = client_with_mock
    inner.delete.return_value = mr(status_code=204)
    await client.delete_consumer('c1')
    inner.delete.assert_called_once_with('/consumers/c1')


# ---------------------------------------------------------------------------
# Plugins
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_list_plugins(client_with_mock):
    client, inner, mr = client_with_mock
    inner.get.return_value = mr({'data': [], 'total': 0})
    result = await client.list_plugins()
    inner.get.assert_called_once_with('/plugins')
    assert 'data' in result


@pytest.mark.asyncio
async def test_get_enabled_plugins(client_with_mock):
    client, inner, mr = client_with_mock
    inner.get.return_value = mr({'enabled_plugins': ['rate-limiting', 'jwt']})
    result = await client.get_enabled_plugins()
    inner.get.assert_called_once_with('/plugins/enabled')
    assert 'enabled_plugins' in result


@pytest.mark.asyncio
async def test_get_plugin_schema(client_with_mock):
    client, inner, mr = client_with_mock
    inner.get.return_value = mr({'fields': []})
    result = await client.get_plugin_schema('rate-limiting')
    inner.get.assert_called_once_with('/plugins/schema/rate-limiting')
    assert 'fields' in result


@pytest.mark.asyncio
async def test_get_plugin(client_with_mock):
    client, inner, mr = client_with_mock
    inner.get.return_value = mr({'id': 'p1', 'name': 'jwt'})
    result = await client.get_plugin('p1')
    inner.get.assert_called_once_with('/plugins/p1')
    assert result['name'] == 'jwt'


@pytest.mark.asyncio
async def test_create_plugin(client_with_mock):
    client, inner, mr = client_with_mock
    inner.post.return_value = mr({'id': 'p1', 'name': 'jwt', 'enabled': True})
    data = {'name': 'jwt', 'config': {}}
    result = await client.create_plugin(data)
    inner.post.assert_called_once_with('/plugins', json=data)
    assert result['id'] == 'p1'


@pytest.mark.asyncio
async def test_update_plugin(client_with_mock):
    client, inner, mr = client_with_mock
    inner.patch.return_value = mr({'id': 'p1', 'enabled': False})
    result = await client.update_plugin('p1', {'enabled': False})
    inner.patch.assert_called_once_with('/plugins/p1', json={'enabled': False})
    assert result['enabled'] is False


@pytest.mark.asyncio
async def test_delete_plugin(client_with_mock):
    client, inner, mr = client_with_mock
    inner.delete.return_value = mr(status_code=204)
    await client.delete_plugin('p1')
    inner.delete.assert_called_once_with('/plugins/p1')


# ---------------------------------------------------------------------------
# Certificates
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_list_certificates(client_with_mock):
    client, inner, mr = client_with_mock
    inner.get.return_value = mr({'data': [], 'total': 0})
    result = await client.list_certificates()
    inner.get.assert_called_once_with('/certificates')
    assert 'data' in result


@pytest.mark.asyncio
async def test_get_certificate(client_with_mock):
    client, inner, mr = client_with_mock
    inner.get.return_value = mr({'id': 'cert1', 'cert': '---BEGIN---'})
    result = await client.get_certificate('cert1')
    inner.get.assert_called_once_with('/certificates/cert1')
    assert result['id'] == 'cert1'


@pytest.mark.asyncio
async def test_create_certificate(client_with_mock):
    client, inner, mr = client_with_mock
    inner.post.return_value = mr({'id': 'cert1', 'cert': '---BEGIN---', 'key': '---KEY---'})
    data = {'cert': '---BEGIN---', 'key': '---KEY---'}
    result = await client.create_certificate(data)
    inner.post.assert_called_once_with('/certificates', json=data)
    assert result['id'] == 'cert1'


@pytest.mark.asyncio
async def test_update_certificate(client_with_mock):
    client, inner, mr = client_with_mock
    inner.patch.return_value = mr({'id': 'cert1', 'cert': '---NEW---'})
    result = await client.update_certificate('cert1', {'cert': '---NEW---'})
    inner.patch.assert_called_once_with('/certificates/cert1', json={'cert': '---NEW---'})
    assert result['cert'] == '---NEW---'


@pytest.mark.asyncio
async def test_delete_certificate(client_with_mock):
    client, inner, mr = client_with_mock
    inner.delete.return_value = mr(status_code=204)
    await client.delete_certificate('cert1')
    inner.delete.assert_called_once_with('/certificates/cert1')


# ---------------------------------------------------------------------------
# SNIs
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_list_snis(client_with_mock):
    client, inner, mr = client_with_mock
    inner.get.return_value = mr({'data': [], 'total': 0})
    result = await client.list_snis()
    inner.get.assert_called_once_with('/snis')
    assert 'data' in result


@pytest.mark.asyncio
async def test_create_sni(client_with_mock):
    client, inner, mr = client_with_mock
    inner.post.return_value = mr({'id': 'sni1', 'name': 'api.example.com'})
    data = {'name': 'api.example.com', 'certificate': {'id': 'cert1'}}
    result = await client.create_sni(data)
    inner.post.assert_called_once_with('/snis', json=data)
    assert result['name'] == 'api.example.com'


@pytest.mark.asyncio
async def test_delete_sni(client_with_mock):
    client, inner, mr = client_with_mock
    inner.delete.return_value = mr(status_code=204)
    await client.delete_sni('sni1')
    inner.delete.assert_called_once_with('/snis/sni1')


# ---------------------------------------------------------------------------
# Declarative Config
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_get_config(client_with_mock):
    client, inner, mr = client_with_mock
    resp = mr(text_data='_format_version: "3.0"\nservices: []\n')
    inner.get.return_value = resp
    result = await client.get_config()
    inner.get.assert_called_once_with('/config')
    assert '_format_version' in result


@pytest.mark.asyncio
async def test_post_config(client_with_mock):
    client, inner, mr = client_with_mock
    inner.post.return_value = mr({'result': 'ok'})
    yaml_str = '_format_version: "3.0"\nservices: []\n'
    result = await client.post_config(yaml_str)
    inner.post.assert_called_once()
    call_kwargs = inner.post.call_args
    assert call_kwargs[0][0] == '/config'
    assert call_kwargs[1]['content'] == yaml_str
    assert result['result'] == 'ok'
