"""Kong Admin API client."""
import httpx
from typing import Optional, Dict, Any, List
from app_quart.config import config


class KongClient:
    """HTTP client for Kong Admin API."""

    def __init__(self, base_url: Optional[str] = None):
        self.base_url = base_url or config.KONG_ADMIN_URL
        self._client = httpx.AsyncClient(
            base_url=self.base_url,
            timeout=30.0,
            headers={'Content-Type': 'application/json'}
        )

    async def close(self):
        await self._client.aclose()

    # Status
    async def get_status(self) -> Dict[str, Any]:
        response = await self._client.get('/status')
        response.raise_for_status()
        return response.json()

    # Services
    async def list_services(self, offset: int = 0, size: int = 100) -> Dict[str, Any]:
        response = await self._client.get('/services', params={'offset': offset, 'size': size})
        response.raise_for_status()
        return response.json()

    async def get_service(self, id_or_name: str) -> Dict[str, Any]:
        response = await self._client.get(f'/services/{id_or_name}')
        response.raise_for_status()
        return response.json()

    async def create_service(self, data: Dict[str, Any]) -> Dict[str, Any]:
        response = await self._client.post('/services', json=data)
        response.raise_for_status()
        return response.json()

    async def update_service(self, id_or_name: str, data: Dict[str, Any]) -> Dict[str, Any]:
        response = await self._client.patch(f'/services/{id_or_name}', json=data)
        response.raise_for_status()
        return response.json()

    async def delete_service(self, id_or_name: str) -> None:
        response = await self._client.delete(f'/services/{id_or_name}')
        response.raise_for_status()

    # Routes
    async def list_routes(self, offset: int = 0, size: int = 100) -> Dict[str, Any]:
        response = await self._client.get('/routes', params={'offset': offset, 'size': size})
        response.raise_for_status()
        return response.json()

    async def get_route(self, id_or_name: str) -> Dict[str, Any]:
        response = await self._client.get(f'/routes/{id_or_name}')
        response.raise_for_status()
        return response.json()

    async def create_route(self, data: Dict[str, Any]) -> Dict[str, Any]:
        response = await self._client.post('/routes', json=data)
        response.raise_for_status()
        return response.json()

    async def update_route(self, id_or_name: str, data: Dict[str, Any]) -> Dict[str, Any]:
        response = await self._client.patch(f'/routes/{id_or_name}', json=data)
        response.raise_for_status()
        return response.json()

    async def delete_route(self, id_or_name: str) -> None:
        response = await self._client.delete(f'/routes/{id_or_name}')
        response.raise_for_status()

    # Upstreams
    async def list_upstreams(self) -> Dict[str, Any]:
        response = await self._client.get('/upstreams')
        response.raise_for_status()
        return response.json()

    async def get_upstream(self, id_or_name: str) -> Dict[str, Any]:
        response = await self._client.get(f'/upstreams/{id_or_name}')
        response.raise_for_status()
        return response.json()

    async def create_upstream(self, data: Dict[str, Any]) -> Dict[str, Any]:
        response = await self._client.post('/upstreams', json=data)
        response.raise_for_status()
        return response.json()

    async def update_upstream(self, id_or_name: str, data: Dict[str, Any]) -> Dict[str, Any]:
        response = await self._client.patch(f'/upstreams/{id_or_name}', json=data)
        response.raise_for_status()
        return response.json()

    async def delete_upstream(self, id_or_name: str) -> None:
        response = await self._client.delete(f'/upstreams/{id_or_name}')
        response.raise_for_status()

    # Targets
    async def list_targets(self, upstream_id: str) -> Dict[str, Any]:
        response = await self._client.get(f'/upstreams/{upstream_id}/targets')
        response.raise_for_status()
        return response.json()

    async def create_target(self, upstream_id: str, data: Dict[str, Any]) -> Dict[str, Any]:
        response = await self._client.post(f'/upstreams/{upstream_id}/targets', json=data)
        response.raise_for_status()
        return response.json()

    async def delete_target(self, upstream_id: str, target_id: str) -> None:
        response = await self._client.delete(f'/upstreams/{upstream_id}/targets/{target_id}')
        response.raise_for_status()

    # Consumers
    async def list_consumers(self) -> Dict[str, Any]:
        response = await self._client.get('/consumers')
        response.raise_for_status()
        return response.json()

    async def get_consumer(self, id_or_username: str) -> Dict[str, Any]:
        response = await self._client.get(f'/consumers/{id_or_username}')
        response.raise_for_status()
        return response.json()

    async def create_consumer(self, data: Dict[str, Any]) -> Dict[str, Any]:
        response = await self._client.post('/consumers', json=data)
        response.raise_for_status()
        return response.json()

    async def update_consumer(self, id_or_username: str, data: Dict[str, Any]) -> Dict[str, Any]:
        response = await self._client.patch(f'/consumers/{id_or_username}', json=data)
        response.raise_for_status()
        return response.json()

    async def delete_consumer(self, id_or_username: str) -> None:
        response = await self._client.delete(f'/consumers/{id_or_username}')
        response.raise_for_status()

    # Plugins
    async def list_plugins(self) -> Dict[str, Any]:
        response = await self._client.get('/plugins')
        response.raise_for_status()
        return response.json()

    async def get_enabled_plugins(self) -> Dict[str, Any]:
        response = await self._client.get('/plugins/enabled')
        response.raise_for_status()
        return response.json()

    async def get_plugin_schema(self, plugin_name: str) -> Dict[str, Any]:
        response = await self._client.get(f'/plugins/schema/{plugin_name}')
        response.raise_for_status()
        return response.json()

    async def get_plugin(self, plugin_id: str) -> Dict[str, Any]:
        response = await self._client.get(f'/plugins/{plugin_id}')
        response.raise_for_status()
        return response.json()

    async def create_plugin(self, data: Dict[str, Any]) -> Dict[str, Any]:
        response = await self._client.post('/plugins', json=data)
        response.raise_for_status()
        return response.json()

    async def update_plugin(self, plugin_id: str, data: Dict[str, Any]) -> Dict[str, Any]:
        response = await self._client.patch(f'/plugins/{plugin_id}', json=data)
        response.raise_for_status()
        return response.json()

    async def delete_plugin(self, plugin_id: str) -> None:
        response = await self._client.delete(f'/plugins/{plugin_id}')
        response.raise_for_status()

    # Certificates
    async def list_certificates(self) -> Dict[str, Any]:
        response = await self._client.get('/certificates')
        response.raise_for_status()
        return response.json()

    async def get_certificate(self, cert_id: str) -> Dict[str, Any]:
        response = await self._client.get(f'/certificates/{cert_id}')
        response.raise_for_status()
        return response.json()

    async def create_certificate(self, data: Dict[str, Any]) -> Dict[str, Any]:
        response = await self._client.post('/certificates', json=data)
        response.raise_for_status()
        return response.json()

    async def update_certificate(self, cert_id: str, data: Dict[str, Any]) -> Dict[str, Any]:
        response = await self._client.patch(f'/certificates/{cert_id}', json=data)
        response.raise_for_status()
        return response.json()

    async def delete_certificate(self, cert_id: str) -> None:
        response = await self._client.delete(f'/certificates/{cert_id}')
        response.raise_for_status()

    # SNIs
    async def list_snis(self) -> Dict[str, Any]:
        response = await self._client.get('/snis')
        response.raise_for_status()
        return response.json()

    async def create_sni(self, data: Dict[str, Any]) -> Dict[str, Any]:
        response = await self._client.post('/snis', json=data)
        response.raise_for_status()
        return response.json()

    async def delete_sni(self, sni_id: str) -> None:
        response = await self._client.delete(f'/snis/{sni_id}')
        response.raise_for_status()

    # Declarative Config
    async def get_config(self) -> str:
        response = await self._client.get('/config')
        response.raise_for_status()
        return response.text

    async def post_config(self, yaml_config: str) -> Dict[str, Any]:
        response = await self._client.post(
            '/config',
            content=yaml_config,
            headers={'Content-Type': 'text/yaml'}
        )
        response.raise_for_status()
        return response.json()
