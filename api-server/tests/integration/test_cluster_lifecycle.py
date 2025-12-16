"""
Integration tests for cluster lifecycle management.
"""
import pytest
from httpx import AsyncClient
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.cluster import Cluster


@pytest.mark.asyncio
class TestClusterLifecycle:
    """Test complete cluster CRUD operations."""

    async def test_create_cluster_community(
        self,
        async_client: AsyncClient,
        auth_headers: dict
    ):
        """Test creating a community cluster."""
        response = await async_client.post(
            "/api/v1/clusters",
            headers=auth_headers,
            json={
                "name": "test-community",
                "description": "Test community cluster",
                "tier": "community"
            }
        )

        assert response.status_code == 201
        data = response.json()
        assert data["name"] == "test-community"
        assert data["tier"] == "community"
        assert data["max_proxies"] == 3
        assert "api_key" in data
        assert data["is_active"] is True

    async def test_create_cluster_enterprise(
        self,
        async_client: AsyncClient,
        auth_headers: dict
    ):
        """Test creating an enterprise cluster."""
        response = await async_client.post(
            "/api/v1/clusters",
            headers=auth_headers,
            json={
                "name": "test-enterprise",
                "description": "Test enterprise cluster",
                "tier": "enterprise",
                "license_key": "PENG-TEST-TEST-TEST-TEST-ABCD"
            }
        )

        assert response.status_code == 201
        data = response.json()
        assert data["name"] == "test-enterprise"
        assert data["tier"] == "enterprise"
        assert data["max_proxies"] > 3

    async def test_create_cluster_invalid_tier(
        self,
        async_client: AsyncClient,
        auth_headers: dict
    ):
        """Test creating cluster with invalid tier."""
        response = await async_client.post(
            "/api/v1/clusters",
            headers=auth_headers,
            json={
                "name": "invalid-tier",
                "description": "Invalid tier",
                "tier": "invalid"
            }
        )

        assert response.status_code == 422

    async def test_create_cluster_duplicate_name(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        test_cluster: Cluster
    ):
        """Test creating cluster with duplicate name."""
        response = await async_client.post(
            "/api/v1/clusters",
            headers=auth_headers,
            json={
                "name": test_cluster.name,
                "description": "Duplicate",
                "tier": "community"
            }
        )

        assert response.status_code == 400
        assert "already exists" in response.json()["detail"].lower()

    async def test_list_clusters(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        test_cluster: Cluster
    ):
        """Test listing all clusters."""
        response = await async_client.get(
            "/api/v1/clusters",
            headers=auth_headers
        )

        assert response.status_code == 200
        data = response.json()
        assert len(data) > 0
        assert any(c["name"] == test_cluster.name for c in data)

    async def test_get_cluster_by_id(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        test_cluster: Cluster
    ):
        """Test getting cluster by ID."""
        response = await async_client.get(
            f"/api/v1/clusters/{test_cluster.id}",
            headers=auth_headers
        )

        assert response.status_code == 200
        data = response.json()
        assert data["id"] == test_cluster.id
        assert data["name"] == test_cluster.name

    async def test_get_cluster_not_found(
        self,
        async_client: AsyncClient,
        auth_headers: dict
    ):
        """Test getting non-existent cluster."""
        response = await async_client.get(
            "/api/v1/clusters/99999",
            headers=auth_headers
        )

        assert response.status_code == 404

    async def test_update_cluster(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        test_cluster: Cluster
    ):
        """Test updating cluster."""
        response = await async_client.put(
            f"/api/v1/clusters/{test_cluster.id}",
            headers=auth_headers,
            json={
                "description": "Updated description",
                "is_active": True
            }
        )

        assert response.status_code == 200
        data = response.json()
        assert data["description"] == "Updated description"

    async def test_update_cluster_name_unique(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        db_session: AsyncSession,
        admin_user
    ):
        """Test updating cluster name must be unique."""
        # Create two clusters
        cluster1 = Cluster(
            name="cluster1",
            description="First",
            tier="community",
            api_key="key1",
            created_by_id=admin_user.id
        )
        cluster2 = Cluster(
            name="cluster2",
            description="Second",
            tier="community",
            api_key="key2",
            created_by_id=admin_user.id
        )

        db_session.add_all([cluster1, cluster2])
        await db_session.commit()

        # Try to update cluster2 with cluster1's name
        response = await async_client.put(
            f"/api/v1/clusters/{cluster2.id}",
            headers=auth_headers,
            json={"name": "cluster1"}
        )

        assert response.status_code == 400

    async def test_delete_cluster(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        db_session: AsyncSession,
        admin_user
    ):
        """Test deleting cluster."""
        cluster = Cluster(
            name="delete-me",
            description="To be deleted",
            tier="community",
            api_key="delete-key",
            created_by_id=admin_user.id
        )
        db_session.add(cluster)
        await db_session.commit()
        await db_session.refresh(cluster)

        response = await async_client.delete(
            f"/api/v1/clusters/{cluster.id}",
            headers=auth_headers
        )

        assert response.status_code == 204

        # Verify deletion
        get_response = await async_client.get(
            f"/api/v1/clusters/{cluster.id}",
            headers=auth_headers
        )
        assert get_response.status_code == 404

    async def test_regenerate_api_key(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        test_cluster: Cluster
    ):
        """Test regenerating cluster API key."""
        old_key = test_cluster.api_key

        response = await async_client.post(
            f"/api/v1/clusters/{test_cluster.id}/regenerate-key",
            headers=auth_headers
        )

        assert response.status_code == 200
        data = response.json()
        assert "api_key" in data
        assert data["api_key"] != old_key

    async def test_cluster_authorization(
        self,
        async_client: AsyncClient,
        user_auth_headers: dict
    ):
        """Test cluster operations require admin privileges."""
        response = await async_client.post(
            "/api/v1/clusters",
            headers=user_auth_headers,
            json={
                "name": "unauthorized",
                "description": "Test",
                "tier": "community"
            }
        )

        # Regular users may be forbidden from creating clusters
        assert response.status_code in [403, 401]

    async def test_license_validation_enterprise(
        self,
        async_client: AsyncClient,
        auth_headers: dict
    ):
        """Test enterprise cluster requires valid license."""
        response = await async_client.post(
            "/api/v1/clusters",
            headers=auth_headers,
            json={
                "name": "enterprise-no-license",
                "description": "Enterprise without license",
                "tier": "enterprise"
            }
        )

        # Should fail without license key
        assert response.status_code in [400, 422]
