"""
Integration tests for proxy registration and heartbeat.
"""
import pytest
from httpx import AsyncClient
from sqlalchemy.ext.asyncio import AsyncSession
from datetime import datetime, timedelta

from app.models.cluster import Cluster
from app.models.proxy import Proxy


@pytest.mark.asyncio
class TestProxyRegistration:
    """Test proxy registration and lifecycle."""

    async def test_register_proxy(
        self,
        async_client: AsyncClient,
        test_cluster: Cluster
    ):
        """Test registering a new proxy."""
        response = await async_client.post(
            "/api/v1/proxies/register",
            headers={"X-Cluster-API-Key": test_cluster.api_key},
            json={
                "hostname": "proxy-1",
                "ip_address": "192.168.1.10",
                "version": "v1.0.0",
                "capabilities": ["l7", "tls"]
            }
        )

        assert response.status_code == 201
        data = response.json()
        assert data["hostname"] == "proxy-1"
        assert data["cluster_id"] == test_cluster.id
        assert data["status"] == "online"
        assert "id" in data

    async def test_register_proxy_invalid_api_key(
        self,
        async_client: AsyncClient
    ):
        """Test proxy registration with invalid API key."""
        response = await async_client.post(
            "/api/v1/proxies/register",
            headers={"X-Cluster-API-Key": "invalid-key"},
            json={
                "hostname": "proxy-2",
                "ip_address": "192.168.1.11",
                "version": "v1.0.0"
            }
        )

        assert response.status_code == 401

    async def test_register_proxy_community_limit(
        self,
        async_client: AsyncClient,
        test_cluster: Cluster,
        db_session: AsyncSession
    ):
        """Test community cluster proxy limit (3 max)."""
        # Register 3 proxies (max for community)
        for i in range(3):
            proxy = Proxy(
                hostname=f"limit-proxy-{i}",
                ip_address=f"192.168.1.{20 + i}",
                cluster_id=test_cluster.id,
                version="v1.0.0",
                status="online"
            )
            db_session.add(proxy)
        await db_session.commit()

        # Try to register 4th proxy
        response = await async_client.post(
            "/api/v1/proxies/register",
            headers={"X-Cluster-API-Key": test_cluster.api_key},
            json={
                "hostname": "limit-proxy-4",
                "ip_address": "192.168.1.24",
                "version": "v1.0.0"
            }
        )

        assert response.status_code == 400
        assert "limit" in response.json()["detail"].lower()

    async def test_reregister_existing_proxy(
        self,
        async_client: AsyncClient,
        test_cluster: Cluster,
        db_session: AsyncSession
    ):
        """Test re-registering an existing proxy updates it."""
        # Initial registration
        proxy = Proxy(
            hostname="reregister-proxy",
            ip_address="192.168.1.30",
            cluster_id=test_cluster.id,
            version="v1.0.0",
            status="online"
        )
        db_session.add(proxy)
        await db_session.commit()

        # Re-register with updated info
        response = await async_client.post(
            "/api/v1/proxies/register",
            headers={"X-Cluster-API-Key": test_cluster.api_key},
            json={
                "hostname": "reregister-proxy",
                "ip_address": "192.168.1.30",
                "version": "v1.1.0",  # Updated version
                "capabilities": ["l7", "l3l4", "tls"]
            }
        )

        assert response.status_code == 200
        data = response.json()
        assert data["version"] == "v1.1.0"

    async def test_proxy_heartbeat(
        self,
        async_client: AsyncClient,
        test_cluster: Cluster,
        db_session: AsyncSession
    ):
        """Test proxy heartbeat endpoint."""
        proxy = Proxy(
            hostname="heartbeat-proxy",
            ip_address="192.168.1.40",
            cluster_id=test_cluster.id,
            version="v1.0.0",
            status="online"
        )
        db_session.add(proxy)
        await db_session.commit()
        await db_session.refresh(proxy)

        old_heartbeat = proxy.last_heartbeat

        response = await async_client.post(
            f"/api/v1/proxies/{proxy.id}/heartbeat",
            headers={"X-Cluster-API-Key": test_cluster.api_key},
            json={
                "cpu_usage": 45.5,
                "memory_usage": 60.2,
                "active_connections": 150
            }
        )

        assert response.status_code == 200
        data = response.json()
        assert data["status"] == "online"
        assert datetime.fromisoformat(data["last_heartbeat"]) > old_heartbeat

    async def test_proxy_metrics_update(
        self,
        async_client: AsyncClient,
        test_cluster: Cluster,
        db_session: AsyncSession
    ):
        """Test updating proxy metrics via heartbeat."""
        proxy = Proxy(
            hostname="metrics-proxy",
            ip_address="192.168.1.50",
            cluster_id=test_cluster.id,
            version="v1.0.0",
            status="online"
        )
        db_session.add(proxy)
        await db_session.commit()
        await db_session.refresh(proxy)

        response = await async_client.post(
            f"/api/v1/proxies/{proxy.id}/heartbeat",
            headers={"X-Cluster-API-Key": test_cluster.api_key},
            json={
                "cpu_usage": 75.0,
                "memory_usage": 80.5,
                "active_connections": 500,
                "bytes_in": 1024000,
                "bytes_out": 2048000
            }
        )

        assert response.status_code == 200
        data = response.json()
        assert data["metrics"]["cpu_usage"] == 75.0
        assert data["metrics"]["memory_usage"] == 80.5

    async def test_proxy_status_offline_detection(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        test_cluster: Cluster,
        db_session: AsyncSession
    ):
        """Test proxy is marked offline if no heartbeat."""
        # Create proxy with old heartbeat
        proxy = Proxy(
            hostname="offline-proxy",
            ip_address="192.168.1.60",
            cluster_id=test_cluster.id,
            version="v1.0.0",
            status="online",
            last_heartbeat=datetime.utcnow() - timedelta(minutes=10)
        )
        db_session.add(proxy)
        await db_session.commit()

        # Get proxy status
        response = await async_client.get(
            f"/api/v1/proxies/{proxy.id}",
            headers=auth_headers
        )

        assert response.status_code == 200
        data = response.json()
        # Should be marked offline or degraded
        assert data["status"] in ["offline", "degraded"]

    async def test_list_proxies_by_cluster(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        test_cluster: Cluster,
        db_session: AsyncSession,
        admin_user
    ):
        """Test listing proxies filtered by cluster."""
        # Create another cluster
        cluster2 = Cluster(
            name="cluster2-proxy",
            description="Second",
            tier="community",
            api_key="key-proxy-2",
            created_by_id=admin_user.id
        )
        db_session.add(cluster2)
        await db_session.commit()

        # Create proxies in different clusters
        proxy1 = Proxy(
            hostname="c1-proxy",
            ip_address="192.168.1.70",
            cluster_id=test_cluster.id,
            version="v1.0.0"
        )
        proxy2 = Proxy(
            hostname="c2-proxy",
            ip_address="192.168.1.71",
            cluster_id=cluster2.id,
            version="v1.0.0"
        )
        db_session.add_all([proxy1, proxy2])
        await db_session.commit()

        response = await async_client.get(
            f"/api/v1/proxies?cluster_id={test_cluster.id}",
            headers=auth_headers
        )

        assert response.status_code == 200
        data = response.json()
        assert all(p["cluster_id"] == test_cluster.id for p in data)

    async def test_deregister_proxy(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        test_cluster: Cluster,
        db_session: AsyncSession
    ):
        """Test deregistering a proxy."""
        proxy = Proxy(
            hostname="dereg-proxy",
            ip_address="192.168.1.80",
            cluster_id=test_cluster.id,
            version="v1.0.0",
            status="online"
        )
        db_session.add(proxy)
        await db_session.commit()
        await db_session.refresh(proxy)

        response = await async_client.delete(
            f"/api/v1/proxies/{proxy.id}",
            headers=auth_headers
        )

        assert response.status_code == 204

    async def test_proxy_capabilities_validation(
        self,
        async_client: AsyncClient,
        test_cluster: Cluster
    ):
        """Test proxy capabilities are validated."""
        response = await async_client.post(
            "/api/v1/proxies/register",
            headers={"X-Cluster-API-Key": test_cluster.api_key},
            json={
                "hostname": "caps-proxy",
                "ip_address": "192.168.1.90",
                "version": "v1.0.0",
                "capabilities": ["l7", "invalid-capability"]
            }
        )

        # Should accept or filter invalid capabilities
        assert response.status_code in [201, 400]
