"""
Performance tests for API server load handling.
"""
import pytest
import requests
import concurrent.futures
import time
from statistics import mean, median


@pytest.mark.performance
class TestAPIPerformance:
    """Test API server performance under load."""

    def test_health_endpoint_response_time(self, api_base_url):
        """Test health endpoint responds quickly."""
        response_times = []

        for _ in range(100):
            start = time.time()
            response = requests.get(f"{api_base_url}/healthz")
            duration = time.time() - start

            assert response.status_code == 200
            response_times.append(duration)

        avg_time = mean(response_times)
        median_time = median(response_times)

        # Should respond in under 100ms on average
        assert avg_time < 0.1, f"Average response time {avg_time}s exceeds 100ms"
        assert median_time < 0.05, f"Median response time {median_time}s exceeds 50ms"

    def test_concurrent_requests_handling(self, api_base_url):
        """Test API can handle concurrent requests."""
        def make_request():
            return requests.get(f"{api_base_url}/healthz")

        # Send 100 concurrent requests
        with concurrent.futures.ThreadPoolExecutor(max_workers=10) as executor:
            futures = [executor.submit(make_request) for _ in range(100)]
            results = [f.result() for f in concurrent.futures.as_completed(futures)]

        # All should succeed
        assert all(r.status_code == 200 for r in results)

    def test_authentication_performance(self, api_base_url):
        """Test authentication endpoint performance."""
        response_times = []

        for _ in range(50):
            start = time.time()
            response = requests.post(
                f"{api_base_url}/api/v1/auth/login",
                data={
                    "username": "admin@test.com",
                    "password": "Admin123!"
                }
            )
            duration = time.time() - start

            assert response.status_code == 200
            response_times.append(duration)

        avg_time = mean(response_times)

        # Authentication should complete in under 500ms
        assert avg_time < 0.5, f"Average auth time {avg_time}s exceeds 500ms"

    def test_list_operations_performance(self, api_base_url, admin_credentials):
        """Test list operations performance."""
        # Login first
        login_resp = requests.post(
            f"{api_base_url}/api/v1/auth/login",
            data=admin_credentials
        )
        token = login_resp.json()["access_token"]
        headers = {"Authorization": f"Bearer {token}"}

        endpoints = [
            "/api/v1/clusters",
            "/api/v1/services",
            "/api/v1/proxies"
        ]

        for endpoint in endpoints:
            response_times = []

            for _ in range(20):
                start = time.time()
                response = requests.get(f"{api_base_url}{endpoint}", headers=headers)
                duration = time.time() - start

                assert response.status_code == 200
                response_times.append(duration)

            avg_time = mean(response_times)

            # List operations should complete in under 200ms
            assert avg_time < 0.2, f"{endpoint} average time {avg_time}s exceeds 200ms"

    def test_metrics_endpoint_performance(self, api_base_url):
        """Test metrics endpoint doesn't slow down under load."""
        response_times = []

        for _ in range(50):
            start = time.time()
            response = requests.get(f"{api_base_url}/metrics")
            duration = time.time() - start

            assert response.status_code == 200
            response_times.append(duration)

        avg_time = mean(response_times)

        # Metrics should be fast even under load
        assert avg_time < 0.1, f"Metrics average time {avg_time}s exceeds 100ms"
