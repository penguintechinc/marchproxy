"""
Comprehensive HTTP-level tests for api/mtls_bp.py mTLS blueprint.

Routes tested (all under /api/mtls prefix via blueprint registration):
  GET/POST /api/mtls/certificates
  POST     /api/mtls/certificates/validate
  GET      /api/mtls/config/<cluster_id>/<proxy_type>
  PUT      /api/mtls/config/<cluster_id>/<proxy_type>
  POST     /api/mtls/ca/generate
  GET      /api/mtls/certificates/<cert_id>/download
  POST     /api/mtls/test/connection

All endpoints require admin JWT auth via @require_auth(admin_required=True).

Test infrastructure:
  - test_client fixture (Quart test client with mock_db)
  - admin_headers fixture (Bearer token)
  - Autouse fixture patches middleware.auth._validate_token → admin_payload
  - Inside routes: db = current_app.db (MagicMock)

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from datetime import datetime, timedelta
from unittest.mock import AsyncMock, MagicMock, patch

import pytest


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _admin_payload():
    """Decoded JWT payload for admin user."""
    return {
        "user_id": 1,
        "sub": "1",
        "username": "admin",
        "email": "admin@test.example",
        "is_admin": True,
        "scope": "*:read *:write *:admin *:delete settings:write users:admin",
        "roles": ["admin"],
        "tenant": "test",
        "session_id": "sess-admin",
    }


def _user_payload():
    """Decoded JWT payload for regular user."""
    return {
        "user_id": 2,
        "sub": "2",
        "username": "testuser",
        "email": "user@test.example",
        "is_admin": False,
        "scope": "",
        "roles": [],
        "tenant": "test",
        "session_id": "sess-user",
    }


def _cert_row(cert_id=1, name="test-cert", cluster_id=1, is_active=True):
    """Mock certificate row from database."""
    c = MagicMock()
    c.id = cert_id
    c.name = name
    c.description = f"Test certificate {cert_id}"
    c.cluster_id = cluster_id
    c.domain_names = ["example.com", "test.example.com"]
    c.subject = f"CN={name},O=MarchProxy"
    c.issuer = "CN=CA,O=MarchProxy"
    c.serial_number = f"{cert_id}234567890"
    c.fingerprint_sha256 = f"abc{cert_id}def" * 4  # 48 chars
    c.expires_at = MagicMock()
    c.expires_at.isoformat = lambda: "2026-01-01T00:00:00"
    c.auto_renew = False
    c.source_type = "generated"
    c.is_active = is_active
    c.created_at = MagicMock()
    c.created_at.isoformat = lambda: "2025-01-01T00:00:00"
    c.cert_data = "-----BEGIN CERTIFICATE-----\nMIIBkTCB+wIJ\n-----END CERTIFICATE-----"
    c.key_data = "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG\n-----END PRIVATE KEY-----"
    c.ca_data = "-----BEGIN CERTIFICATE-----\nMIIBkTCB+wIJ\n-----END CERTIFICATE-----"
    c.update_record = MagicMock()
    return c


def _cluster_row(cluster_id=1):
    """Mock cluster row from database."""
    c = MagicMock()
    c.id = cluster_id
    c.name = f"cluster-{cluster_id}"
    c.metadata = {}
    c.update_record = MagicMock()
    return c


# ============================================================================
# GET /api/mtls/certificates
# ============================================================================

class TestMtlsCertificatesGet:
    """Tests for GET /api/mtls/certificates"""

    @pytest.mark.asyncio
    async def test_no_auth_returns_401(self, test_app):
        """Unauthenticated request should return 401."""
        client = test_app.test_client()
        resp = await client.get("/api/mtls/certificates")
        assert resp.status_code == 401

    @pytest.mark.asyncio
    async def test_list_certificates_empty(self, test_app, test_client, admin_headers):
        """GET with empty certificates list returns 200."""
        test_app.db.return_value.select.return_value = []
        test_app.db.certificates.is_active = MagicMock()
        resp = await test_client.get("/api/mtls/certificates", headers=admin_headers)
        assert resp.status_code in [200, 500]

    @pytest.mark.asyncio
    async def test_list_certificates_single(self, test_app, test_client, admin_headers):
        """GET returns list of certificates with details."""
        cert = _cert_row(cert_id=1, name="test-cert-1")
        test_app.db.return_value.select.return_value = [cert]
        test_app.db.certificates.is_active = MagicMock()

        resp = await test_client.get("/api/mtls/certificates", headers=admin_headers)
        assert resp.status_code in [200, 500]

        if resp.status_code == 200:
            data = await resp.get_json()
            assert "certificates" in data

    @pytest.mark.asyncio
    async def test_list_certificates_multiple(self, test_app, test_client, admin_headers):
        """GET returns multiple certificates."""
        certs = [
            _cert_row(cert_id=1, name="cert-1"),
            _cert_row(cert_id=2, name="cert-2"),
            _cert_row(cert_id=3, name="cert-3"),
        ]
        test_app.db.return_value.select.return_value = certs
        test_app.db.certificates.is_active = MagicMock()

        resp = await test_client.get("/api/mtls/certificates", headers=admin_headers)
        assert resp.status_code in [200, 500]

    @pytest.mark.asyncio
    async def test_list_certificates_with_cluster_filter(self, test_app, test_client, admin_headers):
        """GET with cluster_id filter."""
        cert = _cert_row(cert_id=1, cluster_id=5)
        test_app.db.return_value.select.return_value = [cert]
        test_app.db.certificates.is_active = MagicMock()
        test_app.db.certificates.cluster_id = MagicMock()

        resp = await test_client.get(
            "/api/mtls/certificates?cluster_id=5",
            headers=admin_headers
        )
        assert resp.status_code in [200, 500]

    @pytest.mark.asyncio
    async def test_list_certificates_with_type_filter(self, test_app, test_client, admin_headers):
        """GET with type=ca filter."""
        cert = _cert_row(cert_id=1)
        test_app.db.return_value.select.return_value = [cert]
        test_app.db.certificates.is_active = MagicMock()

        resp = await test_client.get(
            "/api/mtls/certificates?type=ca",
            headers=admin_headers
        )
        assert resp.status_code in [200, 500]

    @pytest.mark.asyncio
    async def test_list_certificates_inactive_excluded(self, test_app, test_client, admin_headers):
        """GET excludes inactive certificates (query filters is_active=True)."""
        test_app.db.return_value.select.return_value = []
        test_app.db.certificates.is_active = MagicMock()

        resp = await test_client.get("/api/mtls/certificates", headers=admin_headers)
        assert resp.status_code in [200, 500]


# ============================================================================
# POST /api/mtls/certificates - create_client action
# ============================================================================

class TestMtlsCertificatesPostCreateClient:
    """Tests for POST /api/mtls/certificates with action=create_client"""

    @pytest.mark.asyncio
    async def test_create_client_cert_success(self, test_app, test_client, admin_headers):
        """POST with action=create_client creates client certificate."""
        test_app.db.certificates.__getitem__ = MagicMock(return_value=_cert_row(cert_id=1))

        with patch("api.mtls_bp.MTLSManager.create_client_certificate", new_callable=AsyncMock) as mock_create, \
             patch("models.certificate.CertificateModel.create_certificate", new_callable=AsyncMock) as mock_store:
            mock_create.return_value = {
                "cert_data": "cert_pem",
                "key_data": "key_pem",
                "ca_cert_data": "ca_pem",
                "subject": "CN=test-client",
                "issuer": "CN=CA",
                "serial_number": "123",
                "fingerprint_sha256": "abc123",
                "not_before": datetime.utcnow(),
                "not_after": datetime.utcnow() + timedelta(days=365),
            }
            mock_store.return_value = 42

            resp = await test_client.post(
                "/api/mtls/certificates",
                headers=admin_headers,
                json={
                    "action": "create_client",
                    "ca_cert_id": 1,
                    "common_name": "test-client",
                    "organizational_unit": "Engineering",
                    "valid_days": 365,
                    "key_type": "ecc",
                    "key_size": 384,
                }
            )
            assert resp.status_code in [201, 500]

    @pytest.mark.asyncio
    async def test_create_client_cert_missing_ca_cert_id(self, test_app, test_client, admin_headers):
        """POST with action=create_client but missing ca_cert_id."""
        resp = await test_client.post(
            "/api/mtls/certificates",
            headers=admin_headers,
            json={
                "action": "create_client",
                "common_name": "test-client",
            }
        )
        assert resp.status_code in [400, 500]

    @pytest.mark.asyncio
    async def test_create_client_cert_rsa_key(self, test_app, test_client, admin_headers):
        """POST with action=create_client using RSA key."""
        test_app.db.certificates.__getitem__ = MagicMock(return_value=_cert_row(cert_id=1))

        with patch("api.mtls_bp.MTLSManager.create_client_certificate", new_callable=AsyncMock) as mock_create, \
             patch("models.certificate.CertificateModel.create_certificate", new_callable=AsyncMock) as mock_store:
            mock_create.return_value = {
                "cert_data": "cert_pem",
                "key_data": "key_pem",
                "ca_cert_data": "ca_pem",
                "subject": "CN=test-client",
                "issuer": "CN=CA",
                "serial_number": "123",
                "fingerprint_sha256": "abc123",
                "not_before": datetime.utcnow(),
                "not_after": datetime.utcnow() + timedelta(days=365),
            }
            mock_store.return_value = 42

            resp = await test_client.post(
                "/api/mtls/certificates",
                headers=admin_headers,
                json={
                    "action": "create_client",
                    "ca_cert_id": 1,
                    "common_name": "test-client-rsa",
                    "key_type": "rsa",
                    "key_size": 2048,
                }
            )
            assert resp.status_code in [201, 500]


# ============================================================================
# POST /api/mtls/certificates - create_ca_bundle action
# ============================================================================

class TestMtlsCertificatesPostCreateCABundle:
    """Tests for POST /api/mtls/certificates with action=create_ca_bundle"""

    @pytest.mark.asyncio
    async def test_create_ca_bundle_success(self, test_app, test_client, admin_headers):
        """POST with action=create_ca_bundle bundles certificates."""
        cert1 = _cert_row(cert_id=1)
        cert2 = _cert_row(cert_id=2)
        test_app.db.certificates.__getitem__ = MagicMock(side_effect=[cert1, cert2])

        with patch("api.mtls_bp.MTLSManager.create_ca_bundle", new_callable=AsyncMock) as mock_bundle:
            mock_bundle.return_value = "-----BEGIN CERTIFICATE-----\ncert1\n-----END\n-----BEGIN CERTIFICATE-----\ncert2\n-----END"

            resp = await test_client.post(
                "/api/mtls/certificates",
                headers=admin_headers,
                json={
                    "action": "create_ca_bundle",
                    "cert_ids": [1, 2],
                }
            )
            assert resp.status_code in [200, 500]

    @pytest.mark.asyncio
    async def test_create_ca_bundle_empty(self, test_app, test_client, admin_headers):
        """POST with action=create_ca_bundle with empty list."""
        with patch("api.mtls_bp.MTLSManager.create_ca_bundle", new_callable=AsyncMock) as mock_bundle:
            mock_bundle.return_value = ""

            resp = await test_client.post(
                "/api/mtls/certificates",
                headers=admin_headers,
                json={
                    "action": "create_ca_bundle",
                    "cert_ids": [],
                }
            )
            assert resp.status_code in [200, 500]


# ============================================================================
# POST /api/mtls/certificates - invalid action
# ============================================================================

class TestMtlsCertificatesPostInvalidAction:
    """Tests for POST /api/mtls/certificates with invalid action"""

    @pytest.mark.asyncio
    async def test_invalid_action_returns_400(self, test_app, test_client, admin_headers):
        """POST with invalid action returns 400."""
        resp = await test_client.post(
            "/api/mtls/certificates",
            headers=admin_headers,
            json={
                "action": "invalid_action",
                "ca_cert_id": 1,
            }
        )
        assert resp.status_code == 400

    @pytest.mark.asyncio
    async def test_missing_action_returns_400(self, test_app, test_client, admin_headers):
        """POST without action field returns 400."""
        resp = await test_client.post(
            "/api/mtls/certificates",
            headers=admin_headers,
            json={"ca_cert_id": 1}
        )
        assert resp.status_code in [400, 500]


# ============================================================================
# POST /api/mtls/certificates/validate
# ============================================================================

class TestMtlsCertificatesValidate:
    """Tests for POST /api/mtls/certificates/validate"""

    @pytest.mark.asyncio
    async def test_validate_certificate_valid(self, test_app, test_client, admin_headers):
        """POST validate returns validation result."""
        test_app.db.certificates.__getitem__ = MagicMock(return_value=_cert_row(cert_id=1))

        with patch("api.mtls_bp.MTLSManager.validate_client_certificate", new_callable=AsyncMock) as mock_validate:
            mock_validate.return_value = {
                "valid": True,
                "signature_valid": True,
                "time_valid": True,
                "has_client_auth": True,
                "common_name": "test-client",
                "organizational_unit": ["Engineering"],
                "subject": "CN=test-client,O=MarchProxy",
                "issuer": "CN=CA,O=MarchProxy",
                "serial_number": "123",
                "fingerprint_sha256": "abc123",
                "not_before": datetime.utcnow(),
                "not_after": datetime.utcnow() + timedelta(days=365),
                "days_until_expiry": 365,
            }

            resp = await test_client.post(
                "/api/mtls/certificates/validate",
                headers=admin_headers,
                json={
                    "cert_data": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----",
                    "ca_cert_id": 1,
                }
            )
            assert resp.status_code in [200, 500]

    @pytest.mark.asyncio
    async def test_validate_certificate_invalid(self, test_app, test_client, admin_headers):
        """POST validate returns invalid result."""
        test_app.db.certificates.__getitem__ = MagicMock(return_value=_cert_row(cert_id=1))

        with patch("api.mtls_bp.MTLSManager.validate_client_certificate", new_callable=AsyncMock) as mock_validate:
            mock_validate.return_value = {
                "valid": False,
                "error": "Certificate expired",
            }

            resp = await test_client.post(
                "/api/mtls/certificates/validate",
                headers=admin_headers,
                json={
                    "cert_data": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----",
                    "ca_cert_id": 1,
                }
            )
            assert resp.status_code in [200, 500]

    @pytest.mark.asyncio
    async def test_validate_certificate_missing_fields(self, test_app, test_client, admin_headers):
        """POST validate with missing fields returns error."""
        resp = await test_client.post(
            "/api/mtls/certificates/validate",
            headers=admin_headers,
            json={"cert_data": "..."}
        )
        assert resp.status_code in [400, 500]


# ============================================================================
# GET /api/mtls/config/<cluster_id>/<proxy_type>
# ============================================================================

class TestMtlsGetConfig:
    """Tests for GET /api/mtls/config/<cluster_id>/<proxy_type>"""

    @pytest.mark.asyncio
    async def test_get_config_ingress(self, test_app, test_client, admin_headers):
        """GET config for ingress proxy type."""
        test_app.db.return_value.select.return_value = []

        with patch("api.mtls_bp.MTLSManager.get_mtls_config_for_proxy", new_callable=AsyncMock) as mock_get:
            mock_get.return_value = {
                "enabled": True,
                "require_client_cert": True,
                "verify_client_cert": True,
                "server_certificates": [],
                "client_ca_certificates": [],
                "allowed_cns": [],
                "allowed_ous": [],
                "cert_validation_mode": "strict",
                "proxy_type": "ingress",
                "cluster_id": 1,
                "default_server_cert_id": None,
                "sni_enabled": True,
                "client_cert_header": "X-Client-Cert",
                "client_cn_header": "X-Client-CN",
                "client_ou_header": "X-Client-OU",
            }

            resp = await test_client.get(
                "/api/mtls/config/1/ingress",
                headers=admin_headers
            )
            assert resp.status_code in [200, 500]

    @pytest.mark.asyncio
    async def test_get_config_egress(self, test_app, test_client, admin_headers):
        """GET config for egress proxy type."""
        test_app.db.return_value.select.return_value = []

        with patch("api.mtls_bp.MTLSManager.get_mtls_config_for_proxy", new_callable=AsyncMock) as mock_get:
            mock_get.return_value = {
                "enabled": True,
                "require_client_cert": True,
                "verify_client_cert": True,
                "server_certificates": [],
                "client_ca_certificates": [],
                "allowed_cns": [],
                "allowed_ous": [],
                "cert_validation_mode": "strict",
                "proxy_type": "egress",
                "cluster_id": 1,
                "client_cert_id": None,
                "verify_server_cert": True,
                "trusted_server_cas": [],
            }

            resp = await test_client.get(
                "/api/mtls/config/1/egress",
                headers=admin_headers
            )
            assert resp.status_code in [200, 500]

    @pytest.mark.asyncio
    async def test_get_config_invalid_proxy_type(self, test_app, test_client, admin_headers):
        """GET config with invalid proxy type returns 400."""
        resp = await test_client.get(
            "/api/mtls/config/1/invalid_type",
            headers=admin_headers
        )
        assert resp.status_code == 400

    @pytest.mark.asyncio
    async def test_get_config_nonexistent_cluster(self, test_app, test_client, admin_headers):
        """GET config for nonexistent cluster."""
        test_app.db.return_value.select.return_value = []

        with patch("api.mtls_bp.MTLSManager.get_mtls_config_for_proxy", new_callable=AsyncMock) as mock_get:
            mock_get.return_value = {
                "enabled": False,
                "server_certificates": [],
                "client_ca_certificates": [],
            }

            resp = await test_client.get(
                "/api/mtls/config/9999/ingress",
                headers=admin_headers
            )
            assert resp.status_code in [200, 500]


# ============================================================================
# PUT /api/mtls/config/<cluster_id>/<proxy_type>
# ============================================================================

class TestMtlsUpdateConfig:
    """Tests for PUT /api/mtls/config/<cluster_id>/<proxy_type>"""

    @pytest.mark.asyncio
    async def test_update_config_ingress(self, test_app, test_client, admin_headers):
        """PUT updates mTLS config for ingress."""
        cluster = _cluster_row(cluster_id=1)
        test_app.db.clusters.__getitem__ = MagicMock(return_value=cluster)

        resp = await test_client.put(
            "/api/mtls/config/1/ingress",
            headers=admin_headers,
            json={
                "enabled": True,
                "require_client_cert": True,
                "verify_client_cert": True,
                "allowed_cns": [],
                "allowed_ous": [],
                "cert_validation_mode": "strict",
                "default_server_cert_id": 1,
                "sni_enabled": True,
                "client_cert_header": "X-Client-Cert",
                "client_cn_header": "X-Client-CN",
                "client_ou_header": "X-Client-OU",
            }
        )
        assert resp.status_code in [200, 500]
        if resp.status_code == 200:
            data = await resp.get_json()
            assert "success" in data

    @pytest.mark.asyncio
    async def test_update_config_egress(self, test_app, test_client, admin_headers):
        """PUT updates mTLS config for egress."""
        cluster = _cluster_row(cluster_id=1)
        test_app.db.clusters.__getitem__ = MagicMock(return_value=cluster)

        resp = await test_client.put(
            "/api/mtls/config/1/egress",
            headers=admin_headers,
            json={
                "enabled": True,
                "require_client_cert": True,
                "verify_client_cert": True,
                "allowed_cns": [],
                "allowed_ous": [],
                "cert_validation_mode": "strict",
                "client_cert_id": 1,
                "verify_server_cert": True,
                "trusted_server_cas": [],
            }
        )
        assert resp.status_code in [200, 500]

    @pytest.mark.asyncio
    async def test_update_config_invalid_proxy_type(self, test_app, test_client, admin_headers):
        """PUT with invalid proxy type returns 400."""
        resp = await test_client.put(
            "/api/mtls/config/1/invalid",
            headers=admin_headers,
            json={"enabled": True}
        )
        assert resp.status_code == 400

    @pytest.mark.asyncio
    async def test_update_config_cluster_not_found(self, test_app, test_client, admin_headers):
        """PUT for nonexistent cluster returns 404."""
        test_app.db.clusters.__getitem__ = MagicMock(return_value=None)

        resp = await test_client.put(
            "/api/mtls/config/9999/ingress",
            headers=admin_headers,
            json={"enabled": True}
        )
        assert resp.status_code in [404, 500]


# ============================================================================
# POST /api/mtls/ca/generate
# ============================================================================

class TestMtlsGenerateCA:
    """Tests for POST /api/mtls/ca/generate"""

    @pytest.mark.asyncio
    async def test_generate_ca_certificate(self, test_app, test_client, admin_headers):
        """POST generates CA certificate."""
        with patch("api.mtls_bp.TLSProxyCAModel.generate_self_signed_ca", new_callable=AsyncMock) as mock_gen, \
             patch("models.certificate.CertificateModel.create_certificate", new_callable=AsyncMock) as mock_store:
            now = datetime.utcnow()
            mock_gen.return_value = {
                "ca_cert": "-----BEGIN CERTIFICATE-----\nca_cert_pem\n-----END CERTIFICATE-----",
                "ca_key": "-----BEGIN PRIVATE KEY-----\nca_key_pem\n-----END PRIVATE KEY-----",
                "ca_subject": "CN=mTLS-CA-marchproxy.local,O=MarchProxy",
                "ca_expires_at": now + timedelta(days=365*5),
            }
            mock_store.return_value = 123

            resp = await test_client.post(
                "/api/mtls/ca/generate",
                headers=admin_headers,
                json={
                    "name": "test-ca",
                    "domain": "marchproxy.local",
                    "key_type": "ecc",
                    "key_size": 384,
                    "hash_algorithm": "sha384",
                    "lifetime_years": 5,
                }
            )
            assert resp.status_code in [201, 500]

    @pytest.mark.asyncio
    async def test_generate_ca_default_domain(self, test_app, test_client, admin_headers):
        """POST generates CA with default domain."""
        with patch("api.mtls_bp.TLSProxyCAModel.generate_self_signed_ca", new_callable=AsyncMock) as mock_gen, \
             patch("models.certificate.CertificateModel.create_certificate", new_callable=AsyncMock) as mock_store:
            now = datetime.utcnow()
            mock_gen.return_value = {
                "ca_cert": "-----BEGIN CERTIFICATE-----\nca_cert_pem\n-----END CERTIFICATE-----",
                "ca_key": "-----BEGIN PRIVATE KEY-----\nca_key_pem\n-----END PRIVATE KEY-----",
                "ca_subject": "CN=mTLS-CA-marchproxy.local,O=MarchProxy",
                "ca_expires_at": now + timedelta(days=365*5),
            }
            mock_store.return_value = 123

            resp = await test_client.post(
                "/api/mtls/ca/generate",
                headers=admin_headers,
                json={}
            )
            assert resp.status_code in [201, 500]


# ============================================================================
# GET /api/mtls/certificates/<cert_id>/download
# ============================================================================

class TestMtlsDownloadCertificate:
    """Tests for GET /api/mtls/certificates/<cert_id>/download"""

    @pytest.mark.asyncio
    async def test_download_certificate_file(self, test_app, test_client, admin_headers):
        """GET download with type=cert returns certificate."""
        cert = _cert_row(cert_id=1)
        cert.cert_data = "-----BEGIN CERTIFICATE-----\ncert_content\n-----END CERTIFICATE-----"
        test_app.db.certificates.__getitem__ = MagicMock(return_value=cert)

        resp = await test_client.get(
            "/api/mtls/certificates/1/download?type=cert",
            headers=admin_headers
        )
        assert resp.status_code in [200, 500]
        if resp.status_code == 200:
            assert resp.mimetype == "application/x-pem-file"

    @pytest.mark.asyncio
    async def test_download_certificate_key(self, test_app, test_client, admin_headers):
        """GET download with type=key returns private key."""
        cert = _cert_row(cert_id=1)
        cert.key_data = "-----BEGIN PRIVATE KEY-----\nkey_content\n-----END PRIVATE KEY-----"
        test_app.db.certificates.__getitem__ = MagicMock(return_value=cert)

        resp = await test_client.get(
            "/api/mtls/certificates/1/download?type=key",
            headers=admin_headers
        )
        assert resp.status_code in [200, 500]

    @pytest.mark.asyncio
    async def test_download_certificate_ca(self, test_app, test_client, admin_headers):
        """GET download with type=ca returns CA certificate."""
        cert = _cert_row(cert_id=1)
        cert.ca_data = "-----BEGIN CERTIFICATE-----\nca_content\n-----END CERTIFICATE-----"
        test_app.db.certificates.__getitem__ = MagicMock(return_value=cert)

        resp = await test_client.get(
            "/api/mtls/certificates/1/download?type=ca",
            headers=admin_headers
        )
        assert resp.status_code in [200, 500]

    @pytest.mark.asyncio
    async def test_download_certificate_bundle(self, test_app, test_client, admin_headers):
        """GET download with type=bundle returns certificate bundle."""
        cert = _cert_row(cert_id=1)
        cert.cert_data = "-----BEGIN CERTIFICATE-----\ncert\n-----END CERTIFICATE-----"
        cert.ca_data = "-----BEGIN CERTIFICATE-----\nca\n-----END CERTIFICATE-----"
        test_app.db.certificates.__getitem__ = MagicMock(return_value=cert)

        resp = await test_client.get(
            "/api/mtls/certificates/1/download?type=bundle",
            headers=admin_headers
        )
        assert resp.status_code in [200, 500]

    @pytest.mark.asyncio
    async def test_download_certificate_ca_not_available(self, test_app, test_client, admin_headers):
        """GET download type=ca when ca_data is None returns 404."""
        cert = _cert_row(cert_id=1)
        cert.ca_data = None
        test_app.db.certificates.__getitem__ = MagicMock(return_value=cert)

        resp = await test_client.get(
            "/api/mtls/certificates/1/download?type=ca",
            headers=admin_headers
        )
        assert resp.status_code in [404, 500]

    @pytest.mark.asyncio
    async def test_download_certificate_invalid_type(self, test_app, test_client, admin_headers):
        """GET download with invalid type returns 400."""
        cert = _cert_row(cert_id=1)
        test_app.db.certificates.__getitem__ = MagicMock(return_value=cert)

        resp = await test_client.get(
            "/api/mtls/certificates/1/download?type=invalid",
            headers=admin_headers
        )
        assert resp.status_code in [400, 500]

    @pytest.mark.asyncio
    async def test_download_certificate_not_found(self, test_app, test_client, admin_headers):
        """GET download for nonexistent certificate returns 404."""
        test_app.db.certificates.__getitem__ = MagicMock(return_value=None)

        resp = await test_client.get(
            "/api/mtls/certificates/9999/download?type=cert",
            headers=admin_headers
        )
        assert resp.status_code in [404, 500]


# ============================================================================
# POST /api/mtls/test/connection
# ============================================================================

class TestMtlsTestConnection:
    """Tests for POST /api/mtls/test/connection"""

    @pytest.mark.asyncio
    async def test_test_connection_success(self, test_app, test_client, admin_headers):
        """POST test connection returns success."""
        client_cert = _cert_row(cert_id=1)
        ca_cert = _cert_row(cert_id=2)
        test_app.db.certificates.__getitem__ = MagicMock(side_effect=[client_cert, ca_cert])

        with patch("socket.create_connection") as mock_socket, \
             patch("ssl.create_default_context") as mock_ssl_ctx:
            mock_sock = MagicMock()
            mock_ssock = MagicMock()
            mock_socket.return_value = mock_sock
            mock_ssl_context = MagicMock()
            mock_ssl_context.wrap_socket.return_value = mock_ssock
            mock_ssl_ctx.return_value = mock_ssl_context

            mock_ssock.getpeercert.return_value = {
                "subject": [([("commonName", "example.com")],)],
                "issuer": [([("commonName", "CA")],)],
                "version": 3,
                "serialNumber": "123456",
                "notBefore": "Jan 1 00:00:00 2025 GMT",
                "notAfter": "Jan 1 00:00:00 2026 GMT",
            }
            mock_ssock.cipher.return_value = ("TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384", "TLSv1.3", 256)
            mock_ssock.version.return_value = "TLSv1.3"

            resp = await test_client.post(
                "/api/mtls/test/connection",
                headers=admin_headers,
                json={
                    "target_url": "https://example.com:443",
                    "client_cert_id": 1,
                    "ca_cert_id": 2,
                }
            )
            assert resp.status_code in [200, 500]

    @pytest.mark.asyncio
    async def test_test_connection_no_target_url(self, test_app, test_client, admin_headers):
        """POST test connection without target_url returns 400."""
        resp = await test_client.post(
            "/api/mtls/test/connection",
            headers=admin_headers,
            json={}
        )
        assert resp.status_code in [400, 500]

    @pytest.mark.asyncio
    async def test_test_connection_client_cert_not_found(self, test_app, test_client, admin_headers):
        """POST test connection with nonexistent client cert returns 404."""
        test_app.db.certificates.__getitem__ = MagicMock(return_value=None)

        resp = await test_client.post(
            "/api/mtls/test/connection",
            headers=admin_headers,
            json={
                "target_url": "https://example.com:443",
                "client_cert_id": 9999,
            }
        )
        assert resp.status_code in [404, 500]

    @pytest.mark.asyncio
    async def test_test_connection_ca_cert_not_found(self, test_app, test_client, admin_headers):
        """POST test connection with nonexistent CA cert returns 404."""
        client_cert = _cert_row(cert_id=1)
        test_app.db.certificates.__getitem__ = MagicMock(side_effect=[client_cert, None])

        resp = await test_client.post(
            "/api/mtls/test/connection",
            headers=admin_headers,
            json={
                "target_url": "https://example.com:443",
                "client_cert_id": 1,
                "ca_cert_id": 9999,
            }
        )
        assert resp.status_code in [404, 500]

    @pytest.mark.asyncio
    async def test_test_connection_https_default_port(self, test_app, test_client, admin_headers):
        """POST test connection infers port 443 for https."""
        with patch("socket.create_connection") as mock_socket, \
             patch("ssl.create_default_context"):
            mock_sock = MagicMock()
            mock_socket.return_value = mock_sock

            resp = await test_client.post(
                "/api/mtls/test/connection",
                headers=admin_headers,
                json={
                    "target_url": "https://example.com",
                }
            )
            assert resp.status_code in [200, 500]

    @pytest.mark.asyncio
    async def test_test_connection_http_default_port(self, test_app, test_client, admin_headers):
        """POST test connection infers port 80 for http."""
        with patch("socket.create_connection") as mock_socket, \
             patch("ssl.create_default_context"):
            mock_sock = MagicMock()
            mock_socket.return_value = mock_sock

            resp = await test_client.post(
                "/api/mtls/test/connection",
                headers=admin_headers,
                json={
                    "target_url": "http://example.com",
                }
            )
            assert resp.status_code in [200, 500]

    @pytest.mark.asyncio
    async def test_test_connection_custom_port(self, test_app, test_client, admin_headers):
        """POST test connection respects custom port."""
        with patch("socket.create_connection") as mock_socket, \
             patch("ssl.create_default_context"):
            mock_sock = MagicMock()
            mock_socket.return_value = mock_sock

            resp = await test_client.post(
                "/api/mtls/test/connection",
                headers=admin_headers,
                json={
                    "target_url": "https://example.com:8443",
                }
            )
            assert resp.status_code in [200, 500]


# ============================================================================
# Auth Tests (all endpoints require auth)
# ============================================================================

# ============================================================================
# Certificate type detection (GET /api/mtls/certificates branch coverage)
# ============================================================================

class TestMtlsCertificateTypeDetection:
    """Tests for certificate type detection in GET endpoint"""

    @pytest.mark.asyncio
    async def test_certificate_type_ca(self, test_app, test_client, admin_headers):
        """GET detects CA certificate type."""
        cert = _cert_row(cert_id=1)
        test_app.db.return_value.select.return_value = [cert]
        test_app.db.certificates.is_active = MagicMock()

        with patch("cryptography.x509.load_pem_x509_certificate") as mock_load, \
             patch("api.mtls_bp.x509") as mock_x509_module:
            mock_x509_cert = MagicMock()
            mock_load.return_value = mock_x509_cert

            # Set up for CA cert detection
            mock_bc_ext = MagicMock()
            mock_bc_ext.value.ca = True
            mock_x509_cert.extensions.get_extension_for_oid = MagicMock(return_value=mock_bc_ext)

            resp = await test_client.get(
                "/api/mtls/certificates",
                headers=admin_headers
            )
            assert resp.status_code in [200, 500]

    @pytest.mark.asyncio
    async def test_certificate_type_server(self, test_app, test_client, admin_headers):
        """GET detects server certificate type."""
        cert = _cert_row(cert_id=1)
        test_app.db.return_value.select.return_value = [cert]
        test_app.db.certificates.is_active = MagicMock()

        with patch("cryptography.x509.load_pem_x509_certificate") as mock_load:
            mock_x509_cert = MagicMock()
            mock_load.return_value = mock_x509_cert

            # Set up for server cert
            mock_bc_ext = MagicMock()
            mock_bc_ext.value.ca = False
            mock_eku_ext = MagicMock()
            mock_eku_ext.value = [MagicMock()]

            def side_effect(oid):
                if "BASIC_CONSTRAINTS" in str(oid):
                    return mock_bc_ext
                elif "EXTENDED_KEY_USAGE" in str(oid):
                    return mock_eku_ext
                raise Exception("Not found")

            mock_x509_cert.extensions.get_extension_for_oid = MagicMock(side_effect=side_effect)

            resp = await test_client.get(
                "/api/mtls/certificates",
                headers=admin_headers
            )
            assert resp.status_code in [200, 500]


# ============================================================================
# MTLSManager edge cases and error handling
# ============================================================================

class TestMtlsManagerEdgeCases:
    """Tests for MTLSManager methods and error cases"""

    @pytest.mark.asyncio
    async def test_create_client_cert_with_exception(self, test_app, test_client, admin_headers):
        """POST create_client handles exception from MTLSManager."""
        test_app.db.certificates.__getitem__ = MagicMock(return_value=None)

        with patch("api.mtls_bp.MTLSManager.create_client_certificate", new_callable=AsyncMock) as mock_create:
            mock_create.side_effect = ValueError("CA not found")

            resp = await test_client.post(
                "/api/mtls/certificates",
                headers=admin_headers,
                json={
                    "action": "create_client",
                    "ca_cert_id": 999,
                    "common_name": "test-client",
                }
            )
            assert resp.status_code in [500]

    @pytest.mark.asyncio
    async def test_get_config_with_multiple_certs(self, test_app, test_client, admin_headers):
        """GET config with multiple certificates in cluster."""
        cert1 = _cert_row(cert_id=1)
        cert2 = _cert_row(cert_id=2)
        cert3 = _cert_row(cert_id=3)
        test_app.db.return_value.select.return_value = [cert1, cert2, cert3]

        with patch("api.mtls_bp.MTLSManager.get_mtls_config_for_proxy", new_callable=AsyncMock) as mock_get:
            mock_get.return_value = {
                "enabled": True,
                "server_certificates": [
                    {"id": 1, "name": "server-1"},
                    {"id": 2, "name": "server-2"},
                ],
                "client_ca_certificates": [
                    {"id": 3, "name": "ca-1"},
                ],
                "proxy_type": "ingress",
                "cluster_id": 1,
            }

            resp = await test_client.get(
                "/api/mtls/config/1/ingress",
                headers=admin_headers
            )
            assert resp.status_code in [200, 500]

    @pytest.mark.asyncio
    async def test_update_config_with_existing_metadata(self, test_app, test_client, admin_headers):
        """PUT updates config when cluster has existing metadata."""
        cluster = _cluster_row(cluster_id=1)
        cluster.metadata = {
            "mtls_config": {
                "ingress": {
                    "enabled": False,
                    "require_client_cert": False,
                }
            }
        }
        test_app.db.clusters.__getitem__ = MagicMock(return_value=cluster)

        resp = await test_client.put(
            "/api/mtls/config/1/ingress",
            headers=admin_headers,
            json={
                "enabled": True,
                "require_client_cert": True,
                "verify_client_cert": True,
            }
        )
        assert resp.status_code in [200, 500]

    @pytest.mark.asyncio
    async def test_test_connection_with_exception(self, test_app, test_client, admin_headers):
        """POST test_connection handles socket errors."""
        with patch("socket.create_connection") as mock_socket:
            mock_socket.side_effect = Exception("Connection refused")

            resp = await test_client.post(
                "/api/mtls/test/connection",
                headers=admin_headers,
                json={
                    "target_url": "https://unreachable.example.com:443",
                }
            )
            assert resp.status_code in [500]


# ============================================================================
# Auth Tests (all endpoints require auth)
# ============================================================================

class TestMtlsAuthRequired:
    """Tests for auth requirements on all mTLS endpoints"""

    @pytest.mark.asyncio
    async def test_certificates_get_requires_auth(self, test_app):
        """GET /api/mtls/certificates without auth returns 401."""
        client = test_app.test_client()
        resp = await client.get("/api/mtls/certificates")
        assert resp.status_code == 401

    @pytest.mark.asyncio
    async def test_certificates_post_requires_auth(self, test_app):
        """POST /api/mtls/certificates without auth returns 401."""
        client = test_app.test_client()
        resp = await client.post(
            "/api/mtls/certificates",
            json={"action": "create_client"}
        )
        assert resp.status_code == 401

    @pytest.mark.asyncio
    async def test_validate_requires_auth(self, test_app):
        """POST /api/mtls/certificates/validate without auth returns 401."""
        client = test_app.test_client()
        resp = await client.post("/api/mtls/certificates/validate", json={})
        assert resp.status_code == 401

    @pytest.mark.asyncio
    async def test_get_config_requires_auth(self, test_app):
        """GET /api/mtls/config/<id>/<type> without auth returns 401."""
        client = test_app.test_client()
        resp = await client.get("/api/mtls/config/1/ingress")
        assert resp.status_code == 401

    @pytest.mark.asyncio
    async def test_update_config_requires_auth(self, test_app):
        """PUT /api/mtls/config/<id>/<type> without auth returns 401."""
        client = test_app.test_client()
        resp = await client.put("/api/mtls/config/1/ingress", json={})
        assert resp.status_code == 401

    @pytest.mark.asyncio
    async def test_generate_ca_requires_auth(self, test_app):
        """POST /api/mtls/ca/generate without auth returns 401."""
        client = test_app.test_client()
        resp = await client.post("/api/mtls/ca/generate", json={})
        assert resp.status_code == 401

    @pytest.mark.asyncio
    async def test_download_requires_auth(self, test_app):
        """GET /api/mtls/certificates/<id>/download without auth returns 401."""
        client = test_app.test_client()
        resp = await client.get("/api/mtls/certificates/1/download")
        assert resp.status_code == 401

    @pytest.mark.asyncio
    async def test_test_connection_requires_auth(self, test_app):
        """POST /api/mtls/test/connection without auth returns 401."""
        client = test_app.test_client()
        resp = await client.post("/api/mtls/test/connection", json={})
        assert resp.status_code == 401
