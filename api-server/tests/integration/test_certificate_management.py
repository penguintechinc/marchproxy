"""
Integration tests for certificate management.
"""
import pytest
from httpx import AsyncClient
from sqlalchemy.ext.asyncio import AsyncSession
from datetime import datetime, timedelta

from app.models.cluster import Cluster
from app.models.certificate import Certificate


@pytest.mark.asyncio
class TestCertificateManagement:
    """Test certificate operations."""

    async def test_upload_certificate(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        test_cluster: Cluster
    ):
        """Test uploading a TLS certificate."""
        cert_data = """-----BEGIN CERTIFICATE-----
MIIDXTCCAkWgAwIBAgIJAKJ5VmXmZ0lBMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV
BAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBX
aWRnaXRzIFB0eSBMdGQwHhcNMjQwMTAxMDAwMDAwWhcNMjUwMTAxMDAwMDAwWjBF
MQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50
ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIB
CgKCAQEA0123456789...
-----END CERTIFICATE-----"""

        key_data = """-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDTU3Z...
-----END PRIVATE KEY-----"""

        response = await async_client.post(
            "/api/v1/certificates",
            headers=auth_headers,
            json={
                "name": "test-cert",
                "cluster_id": test_cluster.id,
                "domain": "example.com",
                "certificate": cert_data,
                "private_key": key_data,
                "source": "manual"
            }
        )

        assert response.status_code == 201
        data = response.json()
        assert data["name"] == "test-cert"
        assert data["domain"] == "example.com"
        assert "private_key" not in data  # Should not expose private key

    async def test_upload_certificate_with_chain(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        test_cluster: Cluster
    ):
        """Test uploading certificate with chain."""
        cert_data = "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----"
        key_data = "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----"
        chain_data = "-----BEGIN CERTIFICATE-----\nchain\n-----END CERTIFICATE-----"

        response = await async_client.post(
            "/api/v1/certificates",
            headers=auth_headers,
            json={
                "name": "chain-cert",
                "cluster_id": test_cluster.id,
                "domain": "chain.example.com",
                "certificate": cert_data,
                "private_key": key_data,
                "chain": chain_data,
                "source": "manual"
            }
        )

        assert response.status_code == 201

    async def test_list_certificates(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        test_cluster: Cluster,
        db_session: AsyncSession
    ):
        """Test listing certificates."""
        # Create test certificates
        for i in range(3):
            cert = Certificate(
                name=f"list-cert-{i}",
                cluster_id=test_cluster.id,
                domain=f"test{i}.example.com",
                certificate="cert-data",
                private_key="key-data",
                source="manual",
                expires_at=datetime.utcnow() + timedelta(days=90)
            )
            db_session.add(cert)
        await db_session.commit()

        response = await async_client.get(
            "/api/v1/certificates",
            headers=auth_headers
        )

        assert response.status_code == 200
        data = response.json()
        assert len(data) >= 3

    async def test_filter_certificates_by_cluster(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        test_cluster: Cluster,
        db_session: AsyncSession,
        admin_user
    ):
        """Test filtering certificates by cluster."""
        # Create another cluster
        cluster2 = Cluster(
            name="cluster2-cert",
            description="Second",
            tier="community",
            api_key="key-cert-2",
            created_by_id=admin_user.id
        )
        db_session.add(cluster2)
        await db_session.commit()

        # Create certificates in different clusters
        cert1 = Certificate(
            name="c1-cert",
            cluster_id=test_cluster.id,
            domain="c1.example.com",
            certificate="cert1",
            private_key="key1",
            source="manual",
            expires_at=datetime.utcnow() + timedelta(days=90)
        )
        cert2 = Certificate(
            name="c2-cert",
            cluster_id=cluster2.id,
            domain="c2.example.com",
            certificate="cert2",
            private_key="key2",
            source="manual",
            expires_at=datetime.utcnow() + timedelta(days=90)
        )
        db_session.add_all([cert1, cert2])
        await db_session.commit()

        response = await async_client.get(
            f"/api/v1/certificates?cluster_id={test_cluster.id}",
            headers=auth_headers
        )

        assert response.status_code == 200
        data = response.json()
        assert all(c["cluster_id"] == test_cluster.id for c in data)

    async def test_get_certificate_by_id(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        test_cluster: Cluster,
        db_session: AsyncSession
    ):
        """Test getting certificate by ID."""
        cert = Certificate(
            name="get-cert",
            cluster_id=test_cluster.id,
            domain="get.example.com",
            certificate="cert-data",
            private_key="key-data",
            source="manual",
            expires_at=datetime.utcnow() + timedelta(days=90)
        )
        db_session.add(cert)
        await db_session.commit()
        await db_session.refresh(cert)

        response = await async_client.get(
            f"/api/v1/certificates/{cert.id}",
            headers=auth_headers
        )

        assert response.status_code == 200
        data = response.json()
        assert data["id"] == cert.id
        assert data["domain"] == "get.example.com"

    async def test_certificate_expiry_warning(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        test_cluster: Cluster,
        db_session: AsyncSession
    ):
        """Test getting certificates expiring soon."""
        # Create certificate expiring in 10 days
        cert = Certificate(
            name="expiring-cert",
            cluster_id=test_cluster.id,
            domain="expiring.example.com",
            certificate="cert-data",
            private_key="key-data",
            source="manual",
            expires_at=datetime.utcnow() + timedelta(days=10)
        )
        db_session.add(cert)
        await db_session.commit()

        response = await async_client.get(
            "/api/v1/certificates/expiring?days=30",
            headers=auth_headers
        )

        assert response.status_code == 200
        data = response.json()
        assert any(c["domain"] == "expiring.example.com" for c in data)
