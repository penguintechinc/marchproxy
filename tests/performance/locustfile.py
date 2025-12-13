"""
Locust load testing for MarchProxy API Server.

Usage:
    locust -f locustfile.py --host=http://localhost:8000
"""
from locust import HttpUser, task, between
import random
import json


class MarchProxyUser(HttpUser):
    """Simulated user for load testing."""

    wait_time = between(1, 3)
    token = None
    cluster_id = None

    def on_start(self):
        """Login and setup before tasks."""
        # Login
        response = self.client.post(
            "/api/v1/auth/login",
            data={
                "username": "admin@test.com",
                "password": "Admin123!"
            }
        )

        if response.status_code == 200:
            self.token = response.json()["access_token"]

            # Get or create a test cluster
            headers = {"Authorization": f"Bearer {self.token}"}
            clusters_response = self.client.get(
                "/api/v1/clusters",
                headers=headers
            )

            if clusters_response.status_code == 200:
                clusters = clusters_response.json()
                if clusters:
                    self.cluster_id = clusters[0]["id"]

    @task(5)
    def list_clusters(self):
        """List all clusters."""
        if self.token:
            self.client.get(
                "/api/v1/clusters",
                headers={"Authorization": f"Bearer {self.token}"}
            )

    @task(5)
    def list_services(self):
        """List all services."""
        if self.token:
            self.client.get(
                "/api/v1/services",
                headers={"Authorization": f"Bearer {self.token}"}
            )

    @task(5)
    def list_proxies(self):
        """List all proxies."""
        if self.token:
            self.client.get(
                "/api/v1/proxies",
                headers={"Authorization": f"Bearer {self.token}"}
            )

    @task(3)
    def get_cluster_details(self):
        """Get specific cluster details."""
        if self.token and self.cluster_id:
            self.client.get(
                f"/api/v1/clusters/{self.cluster_id}",
                headers={"Authorization": f"Bearer {self.token}"}
            )

    @task(2)
    def health_check(self):
        """Check health endpoint."""
        self.client.get("/healthz")

    @task(2)
    def metrics_endpoint(self):
        """Check metrics endpoint."""
        self.client.get("/metrics")

    @task(1)
    def create_service(self):
        """Create a new service."""
        if self.token and self.cluster_id:
            service_data = {
                "name": f"load-test-service-{random.randint(1000, 9999)}",
                "cluster_id": self.cluster_id,
                "source_ip": f"10.0.0.{random.randint(1, 254)}",
                "destination_host": "loadtest.example.com",
                "destination_port": random.choice([80, 443, 8080, 8443]),
                "protocol": random.choice(["http", "https", "tcp"])
            }

            self.client.post(
                "/api/v1/services",
                headers={"Authorization": f"Bearer {self.token}"},
                json=service_data
            )


class ProxyHeartbeatUser(HttpUser):
    """Simulated proxy sending heartbeats."""

    wait_time = between(5, 10)
    api_key = "test-cluster-api-key-12345"
    proxy_id = None

    def on_start(self):
        """Register proxy on start."""
        response = self.client.post(
            "/api/v1/proxies/register",
            headers={"X-Cluster-API-Key": self.api_key},
            json={
                "hostname": f"load-test-proxy-{random.randint(1000, 9999)}",
                "ip_address": f"192.168.1.{random.randint(1, 254)}",
                "version": "v1.0.0",
                "capabilities": ["l7", "tls"]
            }
        )

        if response.status_code in [200, 201]:
            self.proxy_id = response.json()["id"]

    @task
    def send_heartbeat(self):
        """Send proxy heartbeat."""
        if self.proxy_id:
            self.client.post(
                f"/api/v1/proxies/{self.proxy_id}/heartbeat",
                headers={"X-Cluster-API-Key": self.api_key},
                json={
                    "cpu_usage": random.uniform(10, 90),
                    "memory_usage": random.uniform(30, 80),
                    "active_connections": random.randint(50, 500),
                    "bytes_in": random.randint(1000000, 10000000),
                    "bytes_out": random.randint(1000000, 10000000)
                }
            )
