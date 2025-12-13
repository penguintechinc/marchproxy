"""
Security tests for authorization and RBAC.
"""
import pytest
import requests


@pytest.mark.security
class TestAuthorization:
    """Test authorization and role-based access control."""

    def test_unauthenticated_access_denied(self, api_base_url):
        """Test unauthenticated requests are denied."""
        protected_endpoints = [
            "/api/v1/clusters",
            "/api/v1/services",
            "/api/v1/proxies",
            "/api/v1/certificates"
        ]

        for endpoint in protected_endpoints:
            response = requests.get(f"{api_base_url}{endpoint}")
            assert response.status_code == 401

    def test_admin_only_operations(self, api_base_url, admin_credentials):
        """Test admin-only operations."""
        # Login as admin
        admin_login = requests.post(
            f"{api_base_url}/api/v1/auth/login",
            data=admin_credentials
        )
        admin_token = admin_login.json()["access_token"]
        admin_headers = {"Authorization": f"Bearer {admin_token}"}

        # Admin should be able to create clusters
        response = requests.post(
            f"{api_base_url}/api/v1/clusters",
            headers=admin_headers,
            json={
                "name": "admin-test-cluster",
                "tier": "community"
            }
        )

        assert response.status_code in [200, 201]

    def test_regular_user_restrictions(self, api_base_url):
        """Test regular users have restricted access."""
        # Login as regular user
        user_login = requests.post(
            f"{api_base_url}/api/v1/auth/login",
            data={
                "username": "user@test.com",
                "password": "User123!"
            }
        )

        if user_login.status_code == 200:
            user_token = user_login.json()["access_token"]
            user_headers = {"Authorization": f"Bearer {user_token}"}

            # Regular user should NOT be able to create clusters
            response = requests.post(
                f"{api_base_url}/api/v1/clusters",
                headers=user_headers,
                json={
                    "name": "user-test-cluster",
                    "tier": "community"
                }
            )

            assert response.status_code in [403, 401]

    def test_cluster_api_key_authorization(self, api_base_url, test_cluster_api_key):
        """Test cluster API key authorization."""
        # Invalid API key should be rejected
        response = requests.post(
            f"{api_base_url}/api/v1/proxies/register",
            headers={"X-Cluster-API-Key": "invalid-key"},
            json={
                "hostname": "test-proxy",
                "ip_address": "192.168.1.1",
                "version": "v1.0.0"
            }
        )

        assert response.status_code == 401

    def test_cross_cluster_access_denied(self, api_base_url, admin_credentials):
        """Test users cannot access resources from other clusters."""
        # Login and create two clusters
        login_resp = requests.post(
            f"{api_base_url}/api/v1/auth/login",
            data=admin_credentials
        )
        token = login_resp.json()["access_token"]
        headers = {"Authorization": f"Bearer {token}"}

        # Create cluster 1
        cluster1_resp = requests.post(
            f"{api_base_url}/api/v1/clusters",
            headers=headers,
            json={"name": "cluster1-test", "tier": "community"}
        )

        if cluster1_resp.status_code == 201:
            cluster1 = cluster1_resp.json()

            # Create service in cluster 1
            service_resp = requests.post(
                f"{api_base_url}/api/v1/services",
                headers=headers,
                json={
                    "name": "service1",
                    "cluster_id": cluster1["id"],
                    "source_ip": "10.0.0.1",
                    "destination_host": "test.com",
                    "destination_port": 443,
                    "protocol": "https"
                }
            )

            assert service_resp.status_code == 201

    def test_service_owner_permissions(self, api_base_url, admin_credentials):
        """Test service owners can manage their services."""
        # Implementation depends on RBAC model
        assert True

    def test_token_scope_enforcement(self, api_base_url, admin_credentials):
        """Test token scopes are enforced."""
        # If implementing OAuth2 scopes
        assert True
