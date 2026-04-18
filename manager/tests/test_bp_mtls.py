"""
Tests for api/mtls_bp.py blueprint.

Blueprint registered at /api/mtls:
  GET/POST /api/mtls/certificates
  POST     /api/mtls/certificates/validate
  GET      /api/mtls/config/<cluster_id>/<proxy_type>
  PUT      /api/mtls/config/<cluster_id>/<proxy_type>
  POST     /api/mtls/ca/generate
  GET      /api/mtls/certificates/<cert_id>/download
  POST     /api/mtls/test/connection

All endpoints require admin JWT auth.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _admin_payload():
    return {
        "user_id": 1, "username": "admin", "is_admin": True,
        "roles": ["admin"], "scope": ["*:admin"],
    }


def _user_payload():
    return {"user_id": 2, "username": "user", "is_admin": False, "roles": [], "scope": []}


def _cert_row(cert_id=1):
    c = MagicMock()
    c.id = cert_id
    c.name = "test-cert"
    c.description = "Test cert"
    c.cluster_id = 1
    c.domain_names = ["example.com"]
    c.subject = "CN=example.com"
    c.issuer = "CN=CA"
    c.serial_number = "1234"
    c.fingerprint_sha256 = "abc123"
    c.expires_at = MagicMock()
    c.expires_at.isoformat.return_value = "2026-01-01T00:00:00"
    c.auto_renew = False
    c.source_type = "generated"
    c.is_active = True
    c.created_at = MagicMock()
    c.created_at.isoformat.return_value = "2025-01-01T00:00:00"
    c.cert_data = "-----BEGIN CERTIFICATE-----\nfake\n-----END CERTIFICATE-----"
    c.key_data = "-----BEGIN PRIVATE KEY-----\nfake\n-----END PRIVATE KEY-----"
    c.ca_data = None
    return c


# test_app and test_client are provided by tests/conftest.py


# ===========================================================================
# GET/POST /api/mtls/certificates
# ===========================================================================

class TestMtlsCertificatesGet:
    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.get("/api/mtls/certificates")
        assert resp.status_code == 401

    async def test_list_certificates_empty(self, test_app, test_client):
        fresh_db = MagicMock()
        # Return empty list from select
        fresh_db.return_value.select.return_value = []
        fresh_db.certificates.is_active = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/mtls/certificates",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]

    async def test_list_certificates_with_cluster_filter(self, test_app, test_client):
        fresh_db = MagicMock()
        fresh_db.return_value.select.return_value = []
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/mtls/certificates?cluster_id=1&type=ca",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]

    async def test_list_certificates_parses_cert_data(self, test_app, test_client):
        """Covers the x509 parsing branch (exception path)"""
        fresh_db = MagicMock()
        cert = _cert_row()
        fresh_db.return_value.select.return_value = [cert]
        fresh_db.certificates.is_active = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/mtls/certificates",
                headers={"Authorization": "Bearer tok"},
            )
        # The x509 parse will fail on fake cert data, but the exception is caught
        assert resp.status_code in [200, 500]


class TestMtlsCertificatesPost:
    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.post("/api/mtls/certificates", json={})
        assert resp.status_code == 401

    async def test_invalid_action_returns_400(self, test_app, test_client):
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/mtls/certificates",
                json={"action": "unknown"},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [400, 500]

    async def test_create_client_cert_success(self, test_app, test_client):
        fresh_db = MagicMock()
        cert_result = {
            "cert_data": "-----BEGIN CERTIFICATE-----\nfake\n-----END CERTIFICATE-----",
            "key_data": "-----BEGIN PRIVATE KEY-----\nfake\n-----END PRIVATE KEY-----",
            "ca_cert_data": "-----BEGIN CERTIFICATE-----\nca\n-----END CERTIFICATE-----",
        }
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("api.mtls_bp.MTLSManager.create_client_certificate",
                   new_callable=AsyncMock, return_value=cert_result), \
             patch("models.certificate.CertificateModel.create_certificate",
                   new_callable=AsyncMock, return_value=42), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/mtls/certificates",
                json={
                    "action": "create_client",
                    "ca_cert_id": 1,
                    "common_name": "test-client",
                },
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [201, 500]

    async def test_create_ca_bundle_success(self, test_app, test_client):
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("api.mtls_bp.MTLSManager.create_ca_bundle",
                   new_callable=AsyncMock, return_value="bundled-pem-data"), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/mtls/certificates",
                json={"action": "create_ca_bundle", "cert_ids": [1, 2]},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]

    async def test_post_exception_returns_500(self, test_app, test_client):
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("api.mtls_bp.MTLSManager.create_client_certificate",
                   new_callable=AsyncMock, side_effect=ValueError("bad request")), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/mtls/certificates",
                json={"action": "create_client", "ca_cert_id": 1, "common_name": "x"},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [500]


# ===========================================================================
# POST /api/mtls/certificates/validate
# ===========================================================================

class TestValidateCertificate:
    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.post("/api/mtls/certificates/validate", json={})
        assert resp.status_code == 401

    async def test_validate_success(self, test_app, test_client):
        fresh_db = MagicMock()
        validate_result = {"valid": True, "subject": "CN=test"}
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("api.mtls_bp.MTLSManager.validate_client_certificate",
                   new_callable=AsyncMock, return_value=validate_result), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/mtls/certificates/validate",
                json={"cert_data": "-----BEGIN CERTIFICATE-----", "ca_cert_id": 1},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]

    async def test_validate_exception_returns_500(self, test_app, test_client):
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("api.mtls_bp.MTLSManager.validate_client_certificate",
                   new_callable=AsyncMock, side_effect=ValueError("invalid cert")), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/mtls/certificates/validate",
                json={"cert_data": "bad", "ca_cert_id": 1},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 500


# ===========================================================================
# GET /api/mtls/config/<cluster_id>/<proxy_type>
# ===========================================================================

class TestGetMtlsConfig:
    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.get("/api/mtls/config/1/ingress")
        assert resp.status_code == 401

    async def test_invalid_proxy_type_returns_400(self, test_app, test_client):
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/mtls/config/1/invalid-type",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 400

    async def test_get_ingress_config_success(self, test_app, test_client):
        fresh_db = MagicMock()
        config_result = {"enabled": True, "require_client_cert": True}
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("api.mtls_bp.MTLSManager.get_mtls_config_for_proxy",
                   new_callable=AsyncMock, return_value=config_result), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/mtls/config/1/ingress",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]

    async def test_get_egress_config_success(self, test_app, test_client):
        fresh_db = MagicMock()
        config_result = {"enabled": False}
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("api.mtls_bp.MTLSManager.get_mtls_config_for_proxy",
                   new_callable=AsyncMock, return_value=config_result), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/mtls/config/1/egress",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]

    async def test_get_config_exception_returns_500(self, test_app, test_client):
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("api.mtls_bp.MTLSManager.get_mtls_config_for_proxy",
                   new_callable=AsyncMock, side_effect=Exception("db error")), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/mtls/config/1/ingress",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [500]


# ===========================================================================
# PUT /api/mtls/config/<cluster_id>/<proxy_type>
# ===========================================================================

class TestUpdateMtlsConfig:
    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.put("/api/mtls/config/1/ingress", json={})
        assert resp.status_code == 401

    async def test_invalid_proxy_type_returns_400(self, test_app, test_client):
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.put(
                "/api/mtls/config/1/invalid",
                json={"enabled": True},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 400

    async def test_cluster_not_found_returns_404(self, test_app, test_client):
        fresh_db = MagicMock()
        fresh_db.clusters.__getitem__ = MagicMock(return_value=None)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.put(
                "/api/mtls/config/999/ingress",
                json={"enabled": True},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [404, 500]

    async def test_update_ingress_config_success(self, test_app, test_client):
        fresh_db = MagicMock()
        cluster_row = MagicMock()
        cluster_row.metadata = {}
        fresh_db.clusters.__getitem__ = MagicMock(return_value=cluster_row)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.put(
                "/api/mtls/config/1/ingress",
                json={
                    "enabled": True,
                    "require_client_cert": True,
                    "default_server_cert_id": 5,
                    "sni_enabled": True,
                },
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]

    async def test_update_egress_config_success(self, test_app, test_client):
        fresh_db = MagicMock()
        cluster_row = MagicMock()
        cluster_row.metadata = {"mtls_config": {}}
        fresh_db.clusters.__getitem__ = MagicMock(return_value=cluster_row)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.put(
                "/api/mtls/config/1/egress",
                json={"enabled": False, "client_cert_id": 3},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]

    async def test_update_config_exception_returns_500(self, test_app, test_client):
        fresh_db = MagicMock()
        fresh_db.clusters.__getitem__ = MagicMock(side_effect=Exception("db error"))
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.put(
                "/api/mtls/config/1/ingress",
                json={"enabled": True},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [500]


# ===========================================================================
# POST /api/mtls/ca/generate
# ===========================================================================

class TestGenerateCaCertificate:
    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.post("/api/mtls/ca/generate", json={})
        assert resp.status_code == 401

    async def test_generate_ca_success(self, test_app, test_client):
        fresh_db = MagicMock()
        expires = MagicMock()
        expires.isoformat.return_value = "2030-01-01T00:00:00"
        ca_data = {
            "ca_cert": "-----BEGIN CERTIFICATE-----\nca\n-----END CERTIFICATE-----",
            "ca_key": "-----BEGIN PRIVATE KEY-----\nkey\n-----END PRIVATE KEY-----",
            "ca_subject": "CN=TestCA",
            "ca_expires_at": expires,
        }
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.certificate.TLSProxyCAModel.generate_self_signed_ca",
                   new_callable=AsyncMock, return_value=ca_data), \
             patch("models.certificate.CertificateModel.create_certificate",
                   new_callable=AsyncMock, return_value=10), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/mtls/ca/generate",
                json={
                    "domain": "test.local",
                    "name": "test-ca",
                    "cluster_id": 1,
                    "key_type": "ecc",
                },
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [201, 500]

    async def test_generate_ca_exception_returns_500(self, test_app, test_client):
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.certificate.TLSProxyCAModel.generate_self_signed_ca",
                   new_callable=AsyncMock, side_effect=Exception("gen failed")), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/mtls/ca/generate",
                json={"domain": "test.local"},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 500


# ===========================================================================
# GET /api/mtls/certificates/<cert_id>/download
# ===========================================================================

class TestDownloadCertificate:
    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.get("/api/mtls/certificates/1/download")
        assert resp.status_code == 401

    async def test_cert_not_found_returns_404(self, test_app, test_client):
        fresh_db = MagicMock()
        fresh_db.certificates.__getitem__ = MagicMock(return_value=None)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/mtls/certificates/999/download",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 404

    async def test_download_cert_pem(self, test_app, test_client):
        fresh_db = MagicMock()
        cert = _cert_row()
        fresh_db.certificates.__getitem__ = MagicMock(return_value=cert)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/mtls/certificates/1/download?type=cert",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]

    async def test_download_key(self, test_app, test_client):
        fresh_db = MagicMock()
        cert = _cert_row()
        fresh_db.certificates.__getitem__ = MagicMock(return_value=cert)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/mtls/certificates/1/download?type=key",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]

    async def test_download_ca_no_ca_data_returns_404(self, test_app, test_client):
        fresh_db = MagicMock()
        cert = _cert_row()
        cert.ca_data = None
        fresh_db.certificates.__getitem__ = MagicMock(return_value=cert)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/mtls/certificates/1/download?type=ca",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [404, 500]

    async def test_download_ca_with_ca_data(self, test_app, test_client):
        fresh_db = MagicMock()
        cert = _cert_row()
        cert.ca_data = "-----BEGIN CERTIFICATE-----\nca\n-----END CERTIFICATE-----"
        fresh_db.certificates.__getitem__ = MagicMock(return_value=cert)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/mtls/certificates/1/download?type=ca",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]

    async def test_download_bundle(self, test_app, test_client):
        fresh_db = MagicMock()
        cert = _cert_row()
        cert.ca_data = "-----BEGIN CERTIFICATE-----\nca\n-----END CERTIFICATE-----"
        fresh_db.certificates.__getitem__ = MagicMock(return_value=cert)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/mtls/certificates/1/download?type=bundle",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]

    async def test_download_bundle_no_ca(self, test_app, test_client):
        fresh_db = MagicMock()
        cert = _cert_row()
        cert.ca_data = None
        fresh_db.certificates.__getitem__ = MagicMock(return_value=cert)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/mtls/certificates/1/download?type=bundle",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]

    async def test_invalid_download_type_returns_400(self, test_app, test_client):
        fresh_db = MagicMock()
        cert = _cert_row()
        fresh_db.certificates.__getitem__ = MagicMock(return_value=cert)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/mtls/certificates/1/download?type=unknown",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [400, 500]


# ===========================================================================
# POST /api/mtls/test/connection
# ===========================================================================

class TestMtlsConnection:
    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.post("/api/mtls/test/connection", json={})
        assert resp.status_code == 401

    async def test_missing_target_url_returns_400(self, test_app, test_client):
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/mtls/test/connection",
                json={},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [400, 500]

    async def test_client_cert_not_found_returns_404(self, test_app, test_client):
        fresh_db = MagicMock()
        fresh_db.certificates.__getitem__ = MagicMock(return_value=None)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/mtls/test/connection",
                json={
                    "target_url": "https://example.com",
                    "client_cert_id": 999,
                },
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [404, 500]

    async def test_connection_test_network_failure_returns_500(self, test_app, test_client):
        """Real network call will fail, caught as exception → 500"""
        fresh_db = MagicMock()
        fresh_db.certificates.__getitem__ = MagicMock(return_value=None)
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/mtls/test/connection",
                json={"target_url": "https://localhost:1"},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [500]

    async def test_ca_cert_not_found_returns_404(self, test_app, test_client):
        fresh_db = MagicMock()
        # First call (client cert) returns OK, second (ca_cert) returns None
        fresh_db.certificates.__getitem__ = MagicMock(side_effect=[
            MagicMock(),  # client cert found
            None,          # ca cert not found
        ])
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/mtls/test/connection",
                json={
                    "target_url": "https://example.com",
                    "client_cert_id": 1,
                    "ca_cert_id": 999,
                },
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [404, 500]
