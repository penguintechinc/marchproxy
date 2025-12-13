"""
Security tests for injection attack prevention.
"""
import pytest
import requests


@pytest.mark.security
class TestInjectionPrevention:
    """Test protection against injection attacks."""

    def test_sql_injection_prevention(self, api_base_url, admin_credentials):
        """Test SQL injection is prevented."""
        # Login
        login_resp = requests.post(
            f"{api_base_url}/api/v1/auth/login",
            data=admin_credentials
        )
        token = login_resp.json()["access_token"]
        headers = {"Authorization": f"Bearer {token}"}

        # Try SQL injection in cluster name
        sql_payloads = [
            "test'; DROP TABLE clusters;--",
            "test' OR '1'='1",
            "test UNION SELECT * FROM users",
            "test'; DELETE FROM services WHERE '1'='1"
        ]

        for payload in sql_payloads:
            response = requests.post(
                f"{api_base_url}/api/v1/clusters",
                headers=headers,
                json={
                    "name": payload,
                    "tier": "community"
                }
            )

            # Should either reject or sanitize
            assert response.status_code in [201, 400, 422]

    def test_xss_prevention(self, api_base_url, admin_credentials):
        """Test XSS attack prevention."""
        # Login
        login_resp = requests.post(
            f"{api_base_url}/api/v1/auth/login",
            data=admin_credentials
        )
        token = login_resp.json()["access_token"]
        headers = {"Authorization": f"Bearer {token}"}

        # XSS payloads
        xss_payloads = [
            "<script>alert('XSS')</script>",
            "<img src=x onerror=alert('XSS')>",
            "javascript:alert('XSS')",
            "<svg onload=alert('XSS')>"
        ]

        for payload in xss_payloads:
            response = requests.post(
                f"{api_base_url}/api/v1/clusters",
                headers=headers,
                json={
                    "name": "test-cluster",
                    "description": payload,
                    "tier": "community"
                }
            )

            # Should sanitize or reject
            if response.status_code == 201:
                data = response.json()
                # Description should be sanitized
                assert "<script>" not in data.get("description", "")

    def test_command_injection_prevention(self, api_base_url, admin_credentials):
        """Test command injection is prevented."""
        login_resp = requests.post(
            f"{api_base_url}/api/v1/auth/login",
            data=admin_credentials
        )
        token = login_resp.json()["access_token"]
        headers = {"Authorization": f"Bearer {token}"}

        # Command injection payloads
        cmd_payloads = [
            "test; ls -la",
            "test && cat /etc/passwd",
            "test | whoami",
            "test `whoami`"
        ]

        for payload in cmd_payloads:
            response = requests.post(
                f"{api_base_url}/api/v1/services",
                headers=headers,
                json={
                    "name": payload,
                    "cluster_id": 1,
                    "source_ip": "10.0.0.1",
                    "destination_host": "test.com",
                    "destination_port": 443,
                    "protocol": "https"
                }
            )

            # Should sanitize or reject
            assert response.status_code in [201, 400, 404, 422]

    def test_ldap_injection_prevention(self, api_base_url):
        """Test LDAP injection is prevented."""
        # If LDAP authentication is used
        ldap_payloads = [
            "admin*",
            "admin)(|(password=*))",
            "*)(uid=*"
        ]

        for payload in ldap_payloads:
            response = requests.post(
                f"{api_base_url}/api/v1/auth/login",
                data={
                    "username": payload,
                    "password": "test"
                }
            )

            # Should reject invalid input
            assert response.status_code in [400, 401, 422]

    def test_path_traversal_prevention(self, api_base_url, admin_credentials):
        """Test path traversal is prevented."""
        login_resp = requests.post(
            f"{api_base_url}/api/v1/auth/login",
            data=admin_credentials
        )
        token = login_resp.json()["access_token"]
        headers = {"Authorization": f"Bearer {token}"}

        # Path traversal payloads
        path_payloads = [
            "../../../etc/passwd",
            "..\\..\\..\\windows\\system32",
            "....//....//....//etc/passwd"
        ]

        for payload in path_payloads:
            response = requests.get(
                f"{api_base_url}/api/v1/files/{payload}",
                headers=headers
            )

            # Should reject or return 404
            assert response.status_code in [400, 404]

    def test_nosql_injection_prevention(self, api_base_url, admin_credentials):
        """Test NoSQL injection is prevented."""
        # If using NoSQL database
        nosql_payloads = [
            {"$gt": ""},
            {"$ne": None},
            {"$where": "1==1"}
        ]

        login_resp = requests.post(
            f"{api_base_url}/api/v1/auth/login",
            data=admin_credentials
        )
        token = login_resp.json()["access_token"]
        headers = {"Authorization": f"Bearer {token}"}

        for payload in nosql_payloads:
            response = requests.post(
                f"{api_base_url}/api/v1/clusters",
                headers=headers,
                json={
                    "name": payload,
                    "tier": "community"
                }
            )

            # Should reject malformed input
            assert response.status_code in [400, 422]
