"""
Integration tests for service lifecycle management.
"""
import pytest
from httpx import AsyncClient
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.cluster import Cluster
from app.models.service import Service


@pytest.mark.asyncio
class TestServiceLifecycle:
    """Test complete service CRUD operations."""

    async def test_create_service(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        test_cluster: Cluster
    ):
        """Test creating a service."""
        response = await async_client.post(
            "/api/v1/services",
            headers=auth_headers,
            json={
                "name": "test-service",
                "cluster_id": test_cluster.id,
                "source_ip": "10.0.0.100",
                "destination_host": "api.example.com",
                "destination_port": 443,
                "protocol": "https",
                "auth_type": "jwt"
            }
        )

        assert response.status_code == 201
        data = response.json()
        assert data["name"] == "test-service"
        assert data["cluster_id"] == test_cluster.id
        assert data["destination_host"] == "api.example.com"
        assert "auth_token" in data or "jwt_secret" in data

    async def test_create_service_with_port_range(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        test_cluster: Cluster
    ):
        """Test creating service with port range."""
        response = await async_client.post(
            "/api/v1/services",
            headers=auth_headers,
            json={
                "name": "port-range-service",
                "cluster_id": test_cluster.id,
                "source_ip": "10.0.0.101",
                "destination_host": "server.example.com",
                "destination_ports": "8080-8090",
                "protocol": "tcp"
            }
        )

        assert response.status_code == 201
        data = response.json()
        assert "8080-8090" in data["destination_ports"]

    async def test_create_service_with_multiple_ports(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        test_cluster: Cluster
    ):
        """Test creating service with comma-separated ports."""
        response = await async_client.post(
            "/api/v1/services",
            headers=auth_headers,
            json={
                "name": "multi-port-service",
                "cluster_id": test_cluster.id,
                "source_ip": "10.0.0.102",
                "destination_host": "multi.example.com",
                "destination_ports": "80,443,8080",
                "protocol": "tcp"
            }
        )

        assert response.status_code == 201
        data = response.json()
        assert "80" in data["destination_ports"]
        assert "443" in data["destination_ports"]

    async def test_create_service_invalid_protocol(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        test_cluster: Cluster
    ):
        """Test creating service with invalid protocol."""
        response = await async_client.post(
            "/api/v1/services",
            headers=auth_headers,
            json={
                "name": "invalid-protocol",
                "cluster_id": test_cluster.id,
                "source_ip": "10.0.0.103",
                "destination_host": "test.example.com",
                "destination_port": 443,
                "protocol": "invalid"
            }
        )

        assert response.status_code == 422

    async def test_create_service_duplicate_name_in_cluster(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        test_cluster: Cluster,
        db_session: AsyncSession
    ):
        """Test creating service with duplicate name in same cluster."""
        # Create first service
        service1 = Service(
            name="duplicate-service",
            cluster_id=test_cluster.id,
            source_ip="10.0.0.104",
            destination_host="test.example.com",
            destination_port=443,
            protocol="https"
        )
        db_session.add(service1)
        await db_session.commit()

        # Try to create duplicate
        response = await async_client.post(
            "/api/v1/services",
            headers=auth_headers,
            json={
                "name": "duplicate-service",
                "cluster_id": test_cluster.id,
                "source_ip": "10.0.0.105",
                "destination_host": "other.example.com",
                "destination_port": 443,
                "protocol": "https"
            }
        )

        assert response.status_code == 400

    async def test_list_services(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        test_cluster: Cluster,
        db_session: AsyncSession
    ):
        """Test listing all services."""
        # Create test services
        for i in range(3):
            service = Service(
                name=f"list-service-{i}",
                cluster_id=test_cluster.id,
                source_ip=f"10.0.0.{110 + i}",
                destination_host=f"test{i}.example.com",
                destination_port=443,
                protocol="https"
            )
            db_session.add(service)
        await db_session.commit()

        response = await async_client.get(
            "/api/v1/services",
            headers=auth_headers
        )

        assert response.status_code == 200
        data = response.json()
        assert len(data) >= 3

    async def test_filter_services_by_cluster(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        test_cluster: Cluster,
        db_session: AsyncSession,
        admin_user
    ):
        """Test filtering services by cluster."""
        # Create another cluster
        cluster2 = Cluster(
            name="cluster2",
            description="Second cluster",
            tier="community",
            api_key="key2",
            created_by_id=admin_user.id
        )
        db_session.add(cluster2)
        await db_session.commit()

        # Create services in different clusters
        service1 = Service(
            name="cluster1-service",
            cluster_id=test_cluster.id,
            source_ip="10.0.0.120",
            destination_host="c1.example.com",
            destination_port=443,
            protocol="https"
        )
        service2 = Service(
            name="cluster2-service",
            cluster_id=cluster2.id,
            source_ip="10.0.0.121",
            destination_host="c2.example.com",
            destination_port=443,
            protocol="https"
        )
        db_session.add_all([service1, service2])
        await db_session.commit()

        response = await async_client.get(
            f"/api/v1/services?cluster_id={test_cluster.id}",
            headers=auth_headers
        )

        assert response.status_code == 200
        data = response.json()
        assert all(s["cluster_id"] == test_cluster.id for s in data)

    async def test_get_service_by_id(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        test_cluster: Cluster,
        db_session: AsyncSession
    ):
        """Test getting service by ID."""
        service = Service(
            name="get-service",
            cluster_id=test_cluster.id,
            source_ip="10.0.0.130",
            destination_host="get.example.com",
            destination_port=443,
            protocol="https"
        )
        db_session.add(service)
        await db_session.commit()
        await db_session.refresh(service)

        response = await async_client.get(
            f"/api/v1/services/{service.id}",
            headers=auth_headers
        )

        assert response.status_code == 200
        data = response.json()
        assert data["id"] == service.id
        assert data["name"] == "get-service"

    async def test_update_service(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        test_cluster: Cluster,
        db_session: AsyncSession
    ):
        """Test updating service."""
        service = Service(
            name="update-service",
            cluster_id=test_cluster.id,
            source_ip="10.0.0.140",
            destination_host="update.example.com",
            destination_port=443,
            protocol="https"
        )
        db_session.add(service)
        await db_session.commit()
        await db_session.refresh(service)

        response = await async_client.put(
            f"/api/v1/services/{service.id}",
            headers=auth_headers,
            json={
                "destination_host": "updated.example.com",
                "destination_port": 8443
            }
        )

        assert response.status_code == 200
        data = response.json()
        assert data["destination_host"] == "updated.example.com"
        assert data["destination_port"] == 8443

    async def test_delete_service(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        test_cluster: Cluster,
        db_session: AsyncSession
    ):
        """Test deleting service."""
        service = Service(
            name="delete-service",
            cluster_id=test_cluster.id,
            source_ip="10.0.0.150",
            destination_host="delete.example.com",
            destination_port=443,
            protocol="https"
        )
        db_session.add(service)
        await db_session.commit()
        await db_session.refresh(service)

        response = await async_client.delete(
            f"/api/v1/services/{service.id}",
            headers=auth_headers
        )

        assert response.status_code == 204

    async def test_rotate_auth_token(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        test_cluster: Cluster,
        db_session: AsyncSession
    ):
        """Test rotating service auth token."""
        service = Service(
            name="rotate-token-service",
            cluster_id=test_cluster.id,
            source_ip="10.0.0.160",
            destination_host="rotate.example.com",
            destination_port=443,
            protocol="https",
            auth_type="token",
            auth_token="old-token"
        )
        db_session.add(service)
        await db_session.commit()
        await db_session.refresh(service)

        response = await async_client.post(
            f"/api/v1/services/{service.id}/rotate-token",
            headers=auth_headers
        )

        assert response.status_code == 200
        data = response.json()
        assert "auth_token" in data
        assert data["auth_token"] != "old-token"
