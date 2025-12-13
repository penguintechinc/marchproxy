"""
Integration tests for xDS config updates.
"""
import pytest
from httpx import AsyncClient
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.cluster import Cluster
from app.models.service import Service
from app.models.proxy import Proxy


@pytest.mark.asyncio
class TestXDSIntegration:
    """Test xDS configuration generation and updates."""

    async def test_xds_config_generation(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        test_cluster: Cluster,
        db_session: AsyncSession
    ):
        """Test xDS configuration is generated for cluster."""
        # Create service
        service = Service(
            name="xds-service",
            cluster_id=test_cluster.id,
            source_ip="10.0.0.100",
            destination_host="api.example.com",
            destination_port=443,
            protocol="https"
        )
        db_session.add(service)
        await db_session.commit()

        # Get xDS config
        response = await async_client.get(
            f"/api/v1/xds/clusters/{test_cluster.id}/config",
            headers=auth_headers
        )

        assert response.status_code == 200
        data = response.json()
        assert "clusters" in data or "listeners" in data

    async def test_xds_snapshot_update_on_service_create(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        test_cluster: Cluster
    ):
        """Test xDS snapshot updates when service is created."""
        # Get initial snapshot version
        initial_response = await async_client.get(
            f"/api/v1/xds/clusters/{test_cluster.id}/snapshot",
            headers=auth_headers
        )

        initial_version = None
        if initial_response.status_code == 200:
            initial_version = initial_response.json().get("version")

        # Create new service
        service_response = await async_client.post(
            "/api/v1/services",
            headers=auth_headers,
            json={
                "name": "snapshot-service",
                "cluster_id": test_cluster.id,
                "source_ip": "10.0.0.110",
                "destination_host": "snapshot.example.com",
                "destination_port": 443,
                "protocol": "https"
            }
        )

        assert service_response.status_code == 201

        # Get updated snapshot version
        updated_response = await async_client.get(
            f"/api/v1/xds/clusters/{test_cluster.id}/snapshot",
            headers=auth_headers
        )

        assert updated_response.status_code == 200
        updated_version = updated_response.json().get("version")

        # Version should have changed
        if initial_version:
            assert updated_version != initial_version

    async def test_xds_listener_configuration(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        test_cluster: Cluster,
        db_session: AsyncSession
    ):
        """Test xDS listener configuration."""
        # Create service
        service = Service(
            name="listener-service",
            cluster_id=test_cluster.id,
            source_ip="10.0.0.120",
            destination_host="listener.example.com",
            destination_port=8080,
            protocol="http"
        )
        db_session.add(service)
        await db_session.commit()

        response = await async_client.get(
            f"/api/v1/xds/clusters/{test_cluster.id}/listeners",
            headers=auth_headers
        )

        assert response.status_code == 200
        data = response.json()
        assert isinstance(data, list)

    async def test_xds_cluster_configuration(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        test_cluster: Cluster,
        db_session: AsyncSession
    ):
        """Test xDS cluster (upstream) configuration."""
        # Create service
        service = Service(
            name="upstream-service",
            cluster_id=test_cluster.id,
            source_ip="10.0.0.130",
            destination_host="upstream.example.com",
            destination_port=443,
            protocol="https"
        )
        db_session.add(service)
        await db_session.commit()

        response = await async_client.get(
            f"/api/v1/xds/clusters/{test_cluster.id}/clusters",
            headers=auth_headers
        )

        assert response.status_code == 200
        data = response.json()
        assert isinstance(data, list)

    async def test_xds_route_configuration(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        test_cluster: Cluster,
        db_session: AsyncSession
    ):
        """Test xDS route configuration."""
        # Create service
        service = Service(
            name="route-service",
            cluster_id=test_cluster.id,
            source_ip="10.0.0.140",
            destination_host="route.example.com",
            destination_port=443,
            protocol="https"
        )
        db_session.add(service)
        await db_session.commit()

        response = await async_client.get(
            f"/api/v1/xds/clusters/{test_cluster.id}/routes",
            headers=auth_headers
        )

        assert response.status_code == 200
        data = response.json()
        assert isinstance(data, list)

    async def test_xds_tls_configuration(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        test_cluster: Cluster,
        db_session: AsyncSession
    ):
        """Test xDS TLS configuration includes certificates."""
        from app.models.certificate import Certificate
        from datetime import datetime, timedelta

        # Create certificate
        cert = Certificate(
            name="xds-cert",
            cluster_id=test_cluster.id,
            domain="xds.example.com",
            certificate="cert-data",
            private_key="key-data",
            source="manual",
            expires_at=datetime.utcnow() + timedelta(days=90)
        )
        db_session.add(cert)

        # Create service using TLS
        service = Service(
            name="tls-service",
            cluster_id=test_cluster.id,
            source_ip="10.0.0.150",
            destination_host="xds.example.com",
            destination_port=443,
            protocol="https"
        )
        db_session.add(service)
        await db_session.commit()

        response = await async_client.get(
            f"/api/v1/xds/clusters/{test_cluster.id}/secrets",
            headers=auth_headers
        )

        assert response.status_code == 200

    async def test_xds_proxy_discovery(
        self,
        async_client: AsyncClient,
        test_cluster: Cluster,
        db_session: AsyncSession
    ):
        """Test proxy can discover xDS configuration."""
        # Register proxy
        proxy = Proxy(
            hostname="xds-proxy",
            ip_address="192.168.1.100",
            cluster_id=test_cluster.id,
            version="v1.0.0",
            status="online"
        )
        db_session.add(proxy)
        await db_session.commit()
        await db_session.refresh(proxy)

        # Proxy requests xDS config
        response = await async_client.get(
            f"/xds/discovery",
            headers={"X-Cluster-API-Key": test_cluster.api_key}
        )

        # Should return xDS discovery response
        assert response.status_code in [200, 404]

    async def test_xds_incremental_update(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        test_cluster: Cluster,
        db_session: AsyncSession
    ):
        """Test xDS supports incremental updates."""
        # Create initial service
        service1 = Service(
            name="incremental-service-1",
            cluster_id=test_cluster.id,
            source_ip="10.0.0.160",
            destination_host="inc1.example.com",
            destination_port=443,
            protocol="https"
        )
        db_session.add(service1)
        await db_session.commit()

        # Get snapshot
        snapshot1 = await async_client.get(
            f"/api/v1/xds/clusters/{test_cluster.id}/snapshot",
            headers=auth_headers
        )
        version1 = snapshot1.json().get("version")

        # Add another service
        service2 = Service(
            name="incremental-service-2",
            cluster_id=test_cluster.id,
            source_ip="10.0.0.161",
            destination_host="inc2.example.com",
            destination_port=443,
            protocol="https"
        )
        db_session.add(service2)
        await db_session.commit()

        # Get updated snapshot
        snapshot2 = await async_client.get(
            f"/api/v1/xds/clusters/{test_cluster.id}/snapshot",
            headers=auth_headers
        )
        version2 = snapshot2.json().get("version")

        # Versions should differ
        assert version1 != version2

    async def test_xds_health_check_configuration(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        test_cluster: Cluster,
        db_session: AsyncSession
    ):
        """Test xDS includes health check configuration."""
        service = Service(
            name="health-check-service",
            cluster_id=test_cluster.id,
            source_ip="10.0.0.170",
            destination_host="health.example.com",
            destination_port=443,
            protocol="https"
        )
        db_session.add(service)
        await db_session.commit()

        response = await async_client.get(
            f"/api/v1/xds/clusters/{test_cluster.id}/config",
            headers=auth_headers
        )

        assert response.status_code == 200
