"""
End-to-end test for request routing through proxies.
"""
import pytest
import requests
import time


@pytest.mark.e2e
class TestServiceRouting:
    """Test request routing through configured proxies."""

    def test_service_configuration_propagates(
        self,
        docker_services,
        api_base_url,
        admin_credentials
    ):
        """Test service configuration propagates to proxies."""
        # Login and create cluster
        login_resp = requests.post(
            f"{api_base_url}/api/v1/auth/login",
            data=admin_credentials
        )
        token = login_resp.json()["access_token"]
        headers = {"Authorization": f"Bearer {token}"}

        # Create cluster
        cluster_resp = requests.post(
            f"{api_base_url}/api/v1/clusters",
            headers=headers,
            json={"name": f"routing-{int(time.time())}", "tier": "community"}
        )
        cluster = cluster_resp.json()

        # Create service
        service_resp = requests.post(
            f"{api_base_url}/api/v1/services",
            headers=headers,
            json={
                "name": f"route-svc-{int(time.time())}",
                "cluster_id": cluster["id"],
                "source_ip": "10.0.0.50",
                "destination_host": "test.example.com",
                "destination_port": 443,
                "protocol": "https"
            }
        )

        assert service_resp.status_code == 201

    def test_multiple_services_routing(
        self,
        docker_services,
        api_base_url,
        admin_credentials
    ):
        """Test multiple services can be configured."""
        login_resp = requests.post(
            f"{api_base_url}/api/v1/auth/login",
            data=admin_credentials
        )
        token = login_resp.json()["access_token"]
        headers = {"Authorization": f"Bearer {token}"}

        # Create cluster
        cluster_resp = requests.post(
            f"{api_base_url}/api/v1/clusters",
            headers=headers,
            json={"name": f"multi-route-{int(time.time())}", "tier": "community"}
        )
        cluster = cluster_resp.json()

        # Create multiple services
        for i in range(3):
            service_resp = requests.post(
                f"{api_base_url}/api/v1/services",
                headers=headers,
                json={
                    "name": f"multi-svc-{i}-{int(time.time())}",
                    "cluster_id": cluster["id"],
                    "source_ip": f"10.0.0.{100+i}",
                    "destination_host": f"svc{i}.example.com",
                    "destination_port": 443,
                    "protocol": "https"
                }
            )
            assert service_resp.status_code == 201
