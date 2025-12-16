"""
Pytest configuration for end-to-end tests.
"""
import os
import time
import pytest
import subprocess
import requests
from typing import Generator


@pytest.fixture(scope="session")
def docker_compose_file():
    """Path to docker-compose file."""
    return os.path.join(
        os.path.dirname(__file__),
        "../../docker-compose.test.yml"
    )


@pytest.fixture(scope="session")
def docker_services(docker_compose_file) -> Generator:
    """
    Start all Docker services and wait for them to be ready.
    """
    # Start services
    subprocess.run(
        ["docker-compose", "-f", docker_compose_file, "up", "-d"],
        check=True
    )

    # Wait for services to be healthy
    max_retries = 30
    retry_interval = 2

    services = {
        "api-server": "http://localhost:8000/healthz",
        "webui": "http://localhost:5173",
        "xds-server": "http://localhost:9000/healthz"
    }

    for service_name, health_url in services.items():
        print(f"Waiting for {service_name} to be ready...")
        for i in range(max_retries):
            try:
                response = requests.get(health_url, timeout=5)
                if response.status_code == 200:
                    print(f"{service_name} is ready!")
                    break
            except requests.exceptions.RequestException:
                pass

            if i == max_retries - 1:
                raise RuntimeError(f"{service_name} failed to start")

            time.sleep(retry_interval)

    yield

    # Teardown - stop services
    subprocess.run(
        ["docker-compose", "-f", docker_compose_file, "down", "-v"],
        check=True
    )


@pytest.fixture
def api_base_url():
    """API server base URL."""
    return os.getenv("API_BASE_URL", "http://localhost:8000")


@pytest.fixture
def webui_base_url():
    """WebUI base URL."""
    return os.getenv("WEBUI_BASE_URL", "http://localhost:5173")


@pytest.fixture
def xds_base_url():
    """xDS server base URL."""
    return os.getenv("XDS_BASE_URL", "http://localhost:9000")


@pytest.fixture
def admin_credentials():
    """Admin user credentials for testing."""
    return {
        "email": "admin@test.com",
        "password": "Admin123!"
    }


@pytest.fixture
def test_cluster_api_key():
    """Test cluster API key."""
    return "test-cluster-api-key-12345"
