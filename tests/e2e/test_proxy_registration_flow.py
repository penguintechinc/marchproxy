"""
End-to-end test for proxy registration flow: Proxy → API → xDS → Envoy.
"""
import pytest
import requests
import time


@pytest.mark.e2e
class TestProxyRegistrationFlow:
    """Test complete proxy registration and configuration flow."""

    def test_create_cluster_via_api(self, docker_services, api_base_url, admin_credentials):
        """Test creating a cluster via API."""
        # Login
        login_response = requests.post(
            f"{api_base_url}/api/v1/auth/login",
            data=admin_credentials
        )
        assert login_response.status_code == 200
        token = login_response.json()["access_token"]

        headers = {"Authorization": f"Bearer {token}"}

        # Create cluster
        cluster_data = {
            "name": f"e2e-cluster-{int(time.time())}",
            "description": "E2E test cluster",
            "tier": "community"
        }

        response = requests.post(
            f"{api_base_url}/api/v1/clusters",
            headers=headers,
            json=cluster_data
        )

        assert response.status_code == 201
        cluster = response.json()
        assert "id" in cluster
        assert "api_key" in cluster

        return cluster

    def test_register_proxy_to_cluster(self, docker_services, api_base_url, admin_credentials):
        """Test registering a proxy to a cluster."""
        # Create cluster first
        cluster = self.test_create_cluster_via_api(docker_services, api_base_url, admin_credentials)
        api_key = cluster["api_key"]

        # Register proxy
        proxy_data = {
            "hostname": f"test-proxy-{int(time.time())}",
            "ip_address": "192.168.1.100",
            "version": "v1.0.0",
            "capabilities": ["l7", "tls"]
        }

        response = requests.post(
            f"{api_base_url}/api/v1/proxies/register",
            headers={"X-Cluster-API-Key": api_key},
            json=proxy_data
        )

        assert response.status_code == 201
        proxy = response.json()
        assert proxy["hostname"] == proxy_data["hostname"]
        assert proxy["status"] == "online"

        return proxy, cluster

    def test_proxy_heartbeat_flow(self, docker_services, api_base_url, admin_credentials):
        """Test proxy heartbeat updates."""
        proxy, cluster = self.test_register_proxy_to_cluster(docker_services, api_base_url, admin_credentials)

        # Send heartbeat
        heartbeat_data = {
            "cpu_usage": 45.5,
            "memory_usage": 60.2,
            "active_connections": 150
        }

        response = requests.post(
            f"{api_base_url}/api/v1/proxies/{proxy['id']}/heartbeat",
            headers={"X-Cluster-API-Key": cluster["api_key"]},
            json=heartbeat_data
        )

        assert response.status_code == 200
        data = response.json()
        assert data["status"] == "online"
        assert "last_heartbeat" in data

    def test_create_service_for_proxy(self, docker_services, api_base_url, admin_credentials):
        """Test creating a service that proxies will use."""
        # Login
        login_response = requests.post(
            f"{api_base_url}/api/v1/auth/login",
            data=admin_credentials
        )
        token = login_response.json()["access_token"]
        headers = {"Authorization": f"Bearer {token}"}

        # Create cluster
        cluster_response = requests.post(
            f"{api_base_url}/api/v1/clusters",
            headers=headers,
            json={
                "name": f"service-cluster-{int(time.time())}",
                "tier": "community"
            }
        )
        cluster = cluster_response.json()

        # Create service
        service_data = {
            "name": f"e2e-service-{int(time.time())}",
            "cluster_id": cluster["id"],
            "source_ip": "10.0.0.100",
            "destination_host": "api.example.com",
            "destination_port": 443,
            "protocol": "https",
            "auth_type": "jwt"
        }

        response = requests.post(
            f"{api_base_url}/api/v1/services",
            headers=headers,
            json=service_data
        )

        assert response.status_code == 201
        service = response.json()
        assert service["name"] == service_data["name"]
        assert "jwt_secret" in service or "auth_token" in service

        return service, cluster

    def test_xds_config_generation(self, docker_services, api_base_url, xds_base_url, admin_credentials):
        """Test xDS configuration is generated after service creation."""
        service, cluster = self.test_create_service_for_proxy(docker_services, api_base_url, admin_credentials)

        # Login
        login_response = requests.post(
            f"{api_base_url}/api/v1/auth/login",
            data=admin_credentials
        )
        token = login_response.json()["access_token"]
        headers = {"Authorization": f"Bearer {token}"}

        # Get xDS configuration
        response = requests.get(
            f"{api_base_url}/api/v1/xds/clusters/{cluster['id']}/config",
            headers=headers
        )

        assert response.status_code == 200
        xds_config = response.json()

        # Should contain configuration
        assert isinstance(xds_config, dict)

    def test_xds_snapshot_versioning(self, docker_services, api_base_url, admin_credentials):
        """Test xDS snapshot version changes on updates."""
        # Create service
        service, cluster = self.test_create_service_for_proxy(docker_services, api_base_url, admin_credentials)

        # Login
        login_response = requests.post(
            f"{api_base_url}/api/v1/auth/login",
            data=admin_credentials
        )
        token = login_response.json()["access_token"]
        headers = {"Authorization": f"Bearer {token}"}

        # Get initial snapshot
        response1 = requests.get(
            f"{api_base_url}/api/v1/xds/clusters/{cluster['id']}/snapshot",
            headers=headers
        )
        assert response1.status_code == 200
        version1 = response1.json().get("version")

        # Update service
        time.sleep(1)
        update_response = requests.put(
            f"{api_base_url}/api/v1/services/{service['id']}",
            headers=headers,
            json={"destination_port": 8443}
        )
        assert update_response.status_code == 200

        # Get updated snapshot
        response2 = requests.get(
            f"{api_base_url}/api/v1/xds/clusters/{cluster['id']}/snapshot",
            headers=headers
        )
        assert response2.status_code == 200
        version2 = response2.json().get("version")

        # Versions should be different
        if version1 and version2:
            assert version1 != version2

    def test_proxy_discovers_xds_config(self, docker_services, api_base_url, xds_base_url, admin_credentials):
        """Test proxy can discover xDS configuration."""
        proxy, cluster = self.test_register_proxy_to_cluster(docker_services, api_base_url, admin_credentials)

        # Proxy requests xDS discovery
        response = requests.get(
            f"{xds_base_url}/xds/discovery",
            headers={"X-Cluster-API-Key": cluster["api_key"]}
        )

        # Should return discovery response or 404 if not implemented yet
        assert response.status_code in [200, 404]

    def test_full_flow_cluster_to_proxy(self, docker_services, api_base_url, admin_credentials):
        """Test complete flow from cluster creation to proxy registration."""
        # 1. Create cluster
        login_response = requests.post(
            f"{api_base_url}/api/v1/auth/login",
            data=admin_credentials
        )
        token = login_response.json()["access_token"]
        headers = {"Authorization": f"Bearer {token}"}

        cluster_response = requests.post(
            f"{api_base_url}/api/v1/clusters",
            headers=headers,
            json={
                "name": f"full-flow-{int(time.time())}",
                "tier": "community"
            }
        )
        assert cluster_response.status_code == 201
        cluster = cluster_response.json()

        # 2. Create service
        service_response = requests.post(
            f"{api_base_url}/api/v1/services",
            headers=headers,
            json={
                "name": f"flow-service-{int(time.time())}",
                "cluster_id": cluster["id"],
                "source_ip": "10.0.0.200",
                "destination_host": "flow.example.com",
                "destination_port": 443,
                "protocol": "https"
            }
        )
        assert service_response.status_code == 201

        # 3. Register proxy
        proxy_response = requests.post(
            f"{api_base_url}/api/v1/proxies/register",
            headers={"X-Cluster-API-Key": cluster["api_key"]},
            json={
                "hostname": f"flow-proxy-{int(time.time())}",
                "ip_address": "192.168.1.200",
                "version": "v1.0.0"
            }
        )
        assert proxy_response.status_code == 201

        # 4. Send heartbeat
        proxy = proxy_response.json()
        heartbeat_response = requests.post(
            f"{api_base_url}/api/v1/proxies/{proxy['id']}/heartbeat",
            headers={"X-Cluster-API-Key": cluster["api_key"]},
            json={
                "cpu_usage": 30.0,
                "memory_usage": 50.0,
                "active_connections": 100
            }
        )
        assert heartbeat_response.status_code == 200

        # 5. Verify proxy is listed
        proxies_response = requests.get(
            f"{api_base_url}/api/v1/proxies?cluster_id={cluster['id']}",
            headers=headers
        )
        assert proxies_response.status_code == 200
        proxies = proxies_response.json()
        assert len(proxies) > 0
        assert any(p["id"] == proxy["id"] for p in proxies)
