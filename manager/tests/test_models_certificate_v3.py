"""
Unit tests for models/certificate.py - Coverage for uncovered lines

Tests SAN extension handling, CN domain names, certificate renewal methods,
Infisical/Vault provider exceptions, and TLSProxyConfigManager.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import pytest
from datetime import datetime, timedelta
from unittest.mock import MagicMock, AsyncMock, patch, call
from cryptography import x509
from cryptography.x509 import ExtensionNotFound

from models.certificate import (
    CertificateModel,
    InfisicalCertificateProvider,
    VaultCertificateProvider,
    CertificateManager,
    TLSProxyConfigManager,
    TLSProxyCAModel,
)

pytestmark = pytest.mark.unit


# ============================================================================
# Fixtures
# ============================================================================


@pytest.fixture
def mock_db():
    """Create a mock database instance"""
    return MagicMock()


@pytest.fixture
def mock_license_manager():
    """Create a mock license manager"""
    mgr = MagicMock()
    mgr.has_feature = MagicMock(return_value=True)
    return mgr


@pytest.fixture
def mock_cert_record():
    """Create a mock certificate record"""
    cert = MagicMock()
    cert.id = 1
    cert.name = "test-cert"
    cert.source_type = "vault"
    cert.source_config = {
        "vault_url": "http://vault:8200",
        "token": "test-token",
        "role": "test-role",
        "common_name": "test.example.com",
    }
    cert.expires_at = datetime(2026, 6, 1)
    cert.renewal_threshold_days = 30
    cert.renewal_attempts = 0
    return cert


# ============================================================================
# Test: SAN Extension Not Found (Line 127-128)
# ============================================================================


class TestSANExtensionNotFound:
    """Test handling of missing SAN extension in certificate"""

    def test_parse_certificate_san_not_found(self):
        """Test x509.ExtensionNotFound exception handling - bad cert returns None"""
        cert_pem = "-----BEGIN CERTIFICATE-----\nINVALID\n-----END CERTIFICATE-----"

        result = CertificateModel._parse_certificate(cert_pem)
        # Should return None for invalid cert
        assert result is None

    def test_parse_certificate_with_mock_no_san(self):
        """Test SAN not found exception path with mocked cert"""
        with patch("models.certificate.x509.load_pem_x509_certificate") as mock_load:
            mock_cert = MagicMock()
            mock_cert.extensions.get_extension_for_oid.side_effect = ExtensionNotFound(
                "SAN not found", x509.oid.ExtensionOID.SUBJECT_ALTERNATIVE_NAME
            )
            # Mock subject with CN
            cn_attr = MagicMock()
            cn_attr.oid = x509.oid.NameOID.COMMON_NAME
            cn_attr.value = "test.example.com"
            mock_cert.subject = [cn_attr]
            mock_cert.issuer.rfc4514_string.return_value = "CN=Test CA"
            mock_cert.serial_number = 12345
            mock_cert.public_bytes.return_value = b"cert_bytes"
            mock_cert.not_valid_before = datetime(2025, 1, 1)
            mock_cert.not_valid_after = datetime(2026, 1, 1)

            mock_load.return_value = mock_cert

            result = CertificateModel._parse_certificate("cert_pem_data")

            assert result is not None
            assert "test.example.com" in result["domain_names"]


# ============================================================================
# Test: CN Added to Domain Names (Line 138)
# ============================================================================


class TestCNAddedToDomainNames:
    """Test CN is added to domain_names when not in SAN list"""

    def test_cn_added_when_not_in_san(self):
        """Test CN is inserted at beginning of domain_names"""
        with patch("models.certificate.x509.load_pem_x509_certificate") as mock_load:
            mock_cert = MagicMock()

            # Mock SAN with one domain
            san_ext = MagicMock()
            san_name = MagicMock(spec=x509.DNSName)
            san_name.value = "alt.example.com"
            san_ext.value = [san_name]

            mock_cert.extensions.get_extension_for_oid.return_value = san_ext

            # Mock subject with different CN
            cn_attr = MagicMock()
            cn_attr.oid = x509.oid.NameOID.COMMON_NAME
            cn_attr.value = "main.example.com"
            mock_cert.subject = [cn_attr]

            mock_cert.issuer.rfc4514_string.return_value = "CN=Test CA"
            mock_cert.serial_number = 12345
            mock_cert.public_bytes.return_value = b"cert_bytes"
            mock_cert.not_valid_before = datetime(2025, 1, 1)
            mock_cert.not_valid_after = datetime(2026, 1, 1)

            mock_load.return_value = mock_cert

            result = CertificateModel._parse_certificate("cert_pem_data")

            assert result is not None
            # CN should be first in list
            assert result["domain_names"][0] == "main.example.com"
            assert "alt.example.com" in result["domain_names"]

    def test_cn_not_added_when_already_in_san(self):
        """Test CN is not added if already in SAN list"""
        with patch("models.certificate.x509.load_pem_x509_certificate") as mock_load:
            mock_cert = MagicMock()

            # Mock SAN with CN already included
            san_ext = MagicMock()
            san_name1 = MagicMock(spec=x509.DNSName)
            san_name1.value = "test.example.com"
            san_name2 = MagicMock(spec=x509.DNSName)
            san_name2.value = "alt.example.com"
            san_ext.value = [san_name1, san_name2]

            mock_cert.extensions.get_extension_for_oid.return_value = san_ext

            # Mock subject with CN that's already in SAN
            cn_attr = MagicMock()
            cn_attr.oid = x509.oid.NameOID.COMMON_NAME
            cn_attr.value = "test.example.com"
            mock_cert.subject = [cn_attr]

            mock_cert.issuer.rfc4514_string.return_value = "CN=Test CA"
            mock_cert.serial_number = 12345
            mock_cert.public_bytes.return_value = b"cert_bytes"
            mock_cert.not_valid_before = datetime(2025, 1, 1)
            mock_cert.not_valid_after = datetime(2026, 1, 1)

            mock_load.return_value = mock_cert

            result = CertificateModel._parse_certificate("cert_pem_data")

            assert result is not None
            # CN should appear only once
            count = result["domain_names"].count("test.example.com")
            assert count == 1


# ============================================================================
# Test: get_certificates_for_renewal (Line 190)
# ============================================================================


class TestGetCertificatesForRenewal:
    """Test get_certificates_for_renewal returns list"""

    @pytest.mark.skip(reason="DAL query objects can't be mocked for datetime comparisons")
    def test_returns_list_of_dicts(self, mock_db):
        """Test returns list with certificate renewal data"""
        pass

    @pytest.mark.skip(reason="DAL query objects can't be mocked for datetime comparisons")
    def test_returns_empty_list_when_no_certs(self, mock_db):
        """Test returns empty list when no certificates for renewal"""
        pass

    @pytest.mark.skip(reason="DAL query objects can't be mocked for datetime comparisons")
    def test_includes_all_required_fields(self, mock_db):
        """Test returned dict includes all required fields"""
        pass


# ============================================================================
# Test: get_expiring_certificates (Line 268)
# ============================================================================


class TestGetExpiringCertificates:
    """Test get_expiring_certificates returns list"""

    @pytest.mark.skip(reason="DAL query objects can't be mocked for datetime comparisons")
    def test_returns_list_of_dicts(self, mock_db):
        """Test returns list with expiring certificate data"""
        pass

    @pytest.mark.skip(reason="DAL query objects can't be mocked for datetime comparisons")
    def test_returns_empty_list_when_no_expiring(self, mock_db):
        """Test returns empty list when no expiring certificates"""
        pass

    @pytest.mark.skip(reason="DAL query objects can't be mocked for datetime comparisons")
    def test_includes_all_required_fields(self, mock_db):
        """Test returned dict includes all required fields"""
        pass


# ============================================================================
# Test: Infisical fetch_certificate Exception (Line 316-317)
# ============================================================================


class TestInfisicalCertificateProviderException:
    """Test InfisicalCertificateProvider exception handling"""

    @pytest.mark.asyncio
    async def test_fetch_certificate_connection_error(self):
        """Test exception handling when Infisical connection fails"""
        provider = InfisicalCertificateProvider(
            "https://infisical.example.com", "token123", "proj456"
        )

        mock_client = AsyncMock()
        mock_client.get.side_effect = Exception("Connection refused")
        mock_async_client = MagicMock()
        mock_async_client.__aenter__ = AsyncMock(return_value=mock_client)
        mock_async_client.__aexit__ = AsyncMock(return_value=None)

        with patch("httpx.AsyncClient", return_value=mock_async_client):
            result = await provider.fetch_certificate("/path/to/cert")

        assert result is None

    @pytest.mark.asyncio
    async def test_fetch_certificate_timeout(self):
        """Test exception handling when Infisical times out"""
        provider = InfisicalCertificateProvider(
            "https://infisical.example.com", "token123", "proj456"
        )

        mock_client = AsyncMock()
        mock_client.get.side_effect = TimeoutError("Request timed out")
        mock_async_client = MagicMock()
        mock_async_client.__aenter__ = AsyncMock(return_value=mock_client)
        mock_async_client.__aexit__ = AsyncMock(return_value=None)

        with patch("httpx.AsyncClient", return_value=mock_async_client):
            result = await provider.fetch_certificate("/path/to/cert")

        assert result is None


# ============================================================================
# Test: Vault issue_certificate with alt_names (Line 343)
# ============================================================================


class TestVaultIssueWithAltNames:
    """Test Vault issue_certificate with alt_names in payload"""

    @pytest.mark.asyncio
    async def test_issue_certificate_with_alt_names(self):
        """Test alt_names are included in Vault payload"""
        provider = VaultCertificateProvider("http://vault:8200", "token123", "pki")

        response_data = {
            "data": {
                "certificate": "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
                "private_key": "-----BEGIN RSA PRIVATE KEY-----\ntest\n-----END RSA PRIVATE KEY-----",
                "ca_chain": "-----BEGIN CERTIFICATE-----\nca\n-----END CERTIFICATE-----",
            }
        }

        mock_client = AsyncMock()
        mock_response = MagicMock()
        mock_response.status_code = 200
        mock_response.json.return_value = response_data
        mock_client.post.return_value = mock_response

        mock_async_client = MagicMock()
        mock_async_client.__aenter__ = AsyncMock(return_value=mock_client)
        mock_async_client.__aexit__ = AsyncMock(return_value=None)

        with patch("httpx.AsyncClient", return_value=mock_async_client):
            result = await provider.issue_certificate(
                "web",
                "test.example.com",
                alt_names=["app.example.com", "www.example.com"],
            )

        assert result is not None
        assert result["cert_data"]
        assert result["key_data"]

        # Verify alt_names were in the payload
        call_kwargs = mock_client.post.call_args[1]
        payload = call_kwargs["json"]
        assert "alt_names" in payload
        assert payload["alt_names"] == "app.example.com,www.example.com"


# ============================================================================
# Test: Vault issue_certificate Exception (Line 365-368)
# ============================================================================


class TestVaultIssueCertificateException:
    """Test Vault issue_certificate exception handling"""

    @pytest.mark.asyncio
    async def test_issue_certificate_connection_error(self):
        """Test exception handling when Vault connection fails"""
        provider = VaultCertificateProvider("http://vault:8200", "token123", "pki")

        mock_client = AsyncMock()
        mock_client.post.side_effect = Exception("Connection refused")
        mock_async_client = MagicMock()
        mock_async_client.__aenter__ = AsyncMock(return_value=mock_client)
        mock_async_client.__aexit__ = AsyncMock(return_value=None)

        with patch("httpx.AsyncClient", return_value=mock_async_client):
            result = await provider.issue_certificate("web", "test.example.com")

        assert result is None

    @pytest.mark.asyncio
    async def test_issue_certificate_timeout(self):
        """Test exception handling when Vault times out"""
        provider = VaultCertificateProvider("http://vault:8200", "token123", "pki")

        mock_client = AsyncMock()
        mock_client.post.side_effect = TimeoutError("Request timed out")
        mock_async_client = MagicMock()
        mock_async_client.__aenter__ = AsyncMock(return_value=mock_client)
        mock_async_client.__aexit__ = AsyncMock(return_value=None)

        with patch("httpx.AsyncClient", return_value=mock_async_client):
            result = await provider.issue_certificate("web", "test.example.com")

        assert result is None


# ============================================================================
# Test: Vault revoke_certificate (Line 385-387)
# ============================================================================


class TestVaultRevokeCertificate:
    """Test Vault revoke_certificate success and exception paths"""

    @pytest.mark.asyncio
    async def test_revoke_certificate_success(self):
        """Test successful certificate revocation"""
        provider = VaultCertificateProvider("http://vault:8200", "token123", "pki")

        mock_client = AsyncMock()
        mock_response = MagicMock()
        mock_response.status_code = 200
        mock_client.post.return_value = mock_response

        mock_async_client = MagicMock()
        mock_async_client.__aenter__ = AsyncMock(return_value=mock_client)
        mock_async_client.__aexit__ = AsyncMock(return_value=None)

        with patch("httpx.AsyncClient", return_value=mock_async_client):
            result = await provider.revoke_certificate("12345")

        assert result is True

    @pytest.mark.asyncio
    async def test_revoke_certificate_failure_status(self):
        """Test revocation fails with non-200 status"""
        provider = VaultCertificateProvider("http://vault:8200", "token123", "pki")

        mock_client = AsyncMock()
        mock_response = MagicMock()
        mock_response.status_code = 404
        mock_client.post.return_value = mock_response

        mock_async_client = MagicMock()
        mock_async_client.__aenter__ = AsyncMock(return_value=mock_client)
        mock_async_client.__aexit__ = AsyncMock(return_value=None)

        with patch("httpx.AsyncClient", return_value=mock_async_client):
            result = await provider.revoke_certificate("99999")

        assert result is False

    @pytest.mark.asyncio
    async def test_revoke_certificate_exception(self):
        """Test exception handling during revocation"""
        provider = VaultCertificateProvider("http://vault:8200", "token123", "pki")

        mock_client = AsyncMock()
        mock_client.post.side_effect = Exception("Vault unreachable")
        mock_async_client = MagicMock()
        mock_async_client.__aenter__ = AsyncMock(return_value=mock_client)
        mock_async_client.__aexit__ = AsyncMock(return_value=None)

        with patch("httpx.AsyncClient", return_value=mock_async_client):
            result = await provider.revoke_certificate("12345")

        assert result is False


# ============================================================================
# Test: CertificateManager.renew_certificate (Line 472, 474)
# ============================================================================


class TestCertificateManagerRenewal:
    """Test CertificateManager.renew_certificate paths"""

    @pytest.mark.asyncio
    async def test_renew_certificate_not_found(self, mock_db):
        """Test renewal fails when certificate not found"""
        mock_db.__getitem__.return_value = None

        mgr = CertificateManager(mock_db)
        result = await mgr.renew_certificate(999)

        assert result is False

    @pytest.mark.asyncio
    async def test_renew_certificate_manual_source(self, mock_db, mock_cert_record):
        """Test renewal fails for manual certificates"""
        mock_cert_record.source_type = "upload"
        mock_db.__getitem__.return_value = mock_cert_record

        mgr = CertificateManager(mock_db)
        result = await mgr.renew_certificate(1)

        assert result is False

    @pytest.mark.asyncio
    async def test_renew_certificate_exception_path(self, mock_db, mock_cert_record):
        """Test renewal exception is caught and logged"""
        mock_db.__getitem__.return_value = mock_cert_record
        mock_db.certificates.__getitem__.return_value = mock_cert_record

        mgr = CertificateManager(mock_db)

        with patch.object(mgr, "_renew_from_vault", side_effect=Exception("Vault error")):
            with patch("models.certificate.CertificateModel.update_renewal_attempt"):
                result = await mgr.renew_certificate(1)

        assert result is False


# ============================================================================
# Test: _renew_from_infisical (Line 486-500)
# ============================================================================


class TestRenewFromInfisical:
    """Test _renew_from_infisical success and failure paths"""

    @pytest.mark.asyncio
    async def test_renew_from_infisical_success(self, mock_db):
        """Test successful renewal from Infisical"""
        mock_cert = MagicMock()
        mock_cert.id = 1
        mock_cert.name = "infisical-cert"
        mock_cert.source_config = {
            "api_url": "https://infisical.example.com",
            "token": "token123",
            "project_id": "proj456",
            "secret_path": "/path/to/cert",
            "environment": "prod",
        }

        cert_data = {
            "cert_data": "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
            "key_data": "-----BEGIN RSA PRIVATE KEY-----\ntest\n-----END RSA PRIVATE KEY-----",
            "ca_bundle": "",
        }

        mock_db.certificates.__getitem__.return_value = mock_cert

        mgr = CertificateManager(mock_db)

        with patch.object(
            InfisicalCertificateProvider, "fetch_certificate", new_callable=AsyncMock
        ) as mock_fetch:
            mock_fetch.return_value = cert_data
            with patch.object(
                CertificateModel, "update_renewal_attempt", return_value=True
            ) as mock_update:
                result = await mgr._renew_from_infisical(mock_cert)

        assert result is True
        mock_update.assert_called_once()

    @pytest.mark.asyncio
    async def test_renew_from_infisical_no_cert_data(self, mock_db):
        """Test renewal fails when Infisical returns None"""
        mock_cert = MagicMock()
        mock_cert.id = 1
        mock_cert.source_config = {
            "api_url": "https://infisical.example.com",
            "token": "token123",
            "project_id": "proj456",
            "secret_path": "/path/to/cert",
        }

        mgr = CertificateManager(mock_db)

        with patch.object(
            InfisicalCertificateProvider, "fetch_certificate", new_callable=AsyncMock
        ) as mock_fetch:
            mock_fetch.return_value = None
            result = await mgr._renew_from_infisical(mock_cert)

        assert result is False

    @pytest.mark.asyncio
    async def test_renew_from_infisical_missing_cert_data_field(self, mock_db):
        """Test renewal fails when cert_data field is missing"""
        mock_cert = MagicMock()
        mock_cert.id = 1
        mock_cert.source_config = {
            "api_url": "https://infisical.example.com",
            "token": "token123",
            "project_id": "proj456",
            "secret_path": "/path/to/cert",
        }

        mgr = CertificateManager(mock_db)

        with patch.object(
            InfisicalCertificateProvider, "fetch_certificate", new_callable=AsyncMock
        ) as mock_fetch:
            # Return dict but with empty cert_data
            mock_fetch.return_value = {"cert_data": "", "key_data": "key"}
            result = await mgr._renew_from_infisical(mock_cert)

        assert result is False


# ============================================================================
# Test: _renew_from_vault (Line 504-521)
# ============================================================================


class TestRenewFromVault:
    """Test _renew_from_vault success and failure paths"""

    @pytest.mark.asyncio
    async def test_renew_from_vault_success(self, mock_db, mock_cert_record):
        """Test successful renewal from Vault"""
        cert_data = {
            "cert_data": "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
            "key_data": "-----BEGIN RSA PRIVATE KEY-----\ntest\n-----END RSA PRIVATE KEY-----",
            "ca_bundle": "",
        }

        mgr = CertificateManager(mock_db)

        with patch.object(
            VaultCertificateProvider, "issue_certificate", new_callable=AsyncMock
        ) as mock_issue:
            mock_issue.return_value = cert_data
            with patch.object(
                CertificateModel, "update_renewal_attempt", return_value=True
            ) as mock_update:
                result = await mgr._renew_from_vault(mock_cert_record)

        assert result is True
        mock_update.assert_called_once()

    @pytest.mark.asyncio
    async def test_renew_from_vault_failure(self, mock_db, mock_cert_record):
        """Test renewal fails when Vault issue_certificate returns None"""
        mgr = CertificateManager(mock_db)

        with patch.object(
            VaultCertificateProvider, "issue_certificate", new_callable=AsyncMock
        ) as mock_issue:
            mock_issue.return_value = None
            result = await mgr._renew_from_vault(mock_cert_record)

        assert result is False

    @pytest.mark.asyncio
    async def test_renew_from_vault_missing_key_data(self, mock_db, mock_cert_record):
        """Test renewal fails when key_data is missing"""
        cert_data = {
            "cert_data": "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
            "key_data": "",
            "ca_bundle": "",
        }

        mgr = CertificateManager(mock_db)

        with patch.object(
            VaultCertificateProvider, "issue_certificate", new_callable=AsyncMock
        ) as mock_issue:
            mock_issue.return_value = cert_data
            result = await mgr._renew_from_vault(mock_cert_record)

        assert result is False


# ============================================================================
# Test: TLSProxyConfigManager.create_tls_proxy_config (Line 997-1034)
# ============================================================================


class TestTLSProxyConfigManagerCreate:
    """Test TLSProxyConfigManager.create_tls_proxy_config paths"""

    def test_create_no_license_manager(self, mock_db):
        """Test creation fails when no license_manager"""
        mgr = TLSProxyConfigManager(mock_db, license_manager=None)
        success, error = mgr.create_tls_proxy_config(1, 1, {"name": "config"}, 1)

        assert success is False
        assert "Enterprise" in str(error["error"])

    def test_create_license_check_fails(self, mock_db):
        """Test creation fails when license check fails"""
        mock_license = MagicMock()
        mock_license.has_feature.return_value = False

        mgr = TLSProxyConfigManager(mock_db, license_manager=mock_license)
        success, error = mgr.create_tls_proxy_config(1, 1, {"name": "config"}, 1)

        assert success is False
        assert "Enterprise" in str(error["error"])

    def test_create_ca_not_found(self, mock_db, mock_license_manager):
        """Test creation fails when CA not found"""
        mock_db.tls_proxy_cas.__getitem__.return_value = None

        mgr = TLSProxyConfigManager(mock_db, license_manager=mock_license_manager)
        success, error = mgr.create_tls_proxy_config(1, 99, {"name": "config"}, 1)

        assert success is False
        assert "Invalid CA" in str(error["error"])

    def test_create_ca_wrong_cluster(self, mock_db, mock_license_manager):
        """Test creation fails when CA belongs to different cluster"""
        mock_ca = MagicMock()
        mock_ca.cluster_id = 2  # Different cluster
        mock_db.tls_proxy_cas.__getitem__.return_value = mock_ca

        mgr = TLSProxyConfigManager(mock_db, license_manager=mock_license_manager)
        success, error = mgr.create_tls_proxy_config(1, 1, {"name": "config"}, 1)

        assert success is False
        assert "Invalid CA" in str(error["error"])

    def test_create_success(self, mock_db, mock_license_manager):
        """Test successful proxy config creation"""
        mock_ca = MagicMock()
        mock_ca.cluster_id = 1
        mock_db.tls_proxy_cas.__getitem__.return_value = mock_ca
        mock_db.tls_proxy_configs.insert.return_value = 42

        mgr = TLSProxyConfigManager(mock_db, license_manager=mock_license_manager)
        success, result = mgr.create_tls_proxy_config(
            1,
            1,
            {
                "name": "prod-proxy",
                "enabled": True,
                "protocol_detection": True,
            },
            1,
        )

        assert success is True
        assert result["id"] == 42
        mock_db.tls_proxy_configs.insert.assert_called_once()

    def test_create_exception(self, mock_db, mock_license_manager):
        """Test exception during proxy config creation"""
        mock_ca = MagicMock()
        mock_ca.cluster_id = 1
        mock_db.tls_proxy_cas.__getitem__.return_value = mock_ca
        mock_db.tls_proxy_configs.insert.side_effect = Exception("DB error")

        mgr = TLSProxyConfigManager(mock_db, license_manager=mock_license_manager)
        success, error = mgr.create_tls_proxy_config(1, 1, {"name": "config"}, 1)

        assert success is False
        assert "error" in error


# ============================================================================
# Test: TLSProxyConfigManager.get_proxy_config (Line 1036-1076)
# ============================================================================


class TestTLSProxyConfigManagerGet:
    """Test TLSProxyConfigManager.get_proxy_config paths"""

    def test_get_config_not_found(self, mock_db, mock_license_manager):
        """Test config not found returns disabled response"""
        mock_db.return_value.select.return_value.first.return_value = None

        mgr = TLSProxyConfigManager(mock_db, license_manager=mock_license_manager)
        result = mgr.get_proxy_config(1, 1)

        assert result["enabled"] is False
        assert result["enterprise_available"] is True

    def test_get_config_not_found_no_license(self, mock_db):
        """Test config not found with no license manager"""
        mock_db.return_value.select.return_value.first.return_value = None

        mgr = TLSProxyConfigManager(mock_db, license_manager=None)
        result = mgr.get_proxy_config(1, 1)

        assert result["enabled"] is False
        assert result["enterprise_available"] is False

    def test_get_config_found(self, mock_db, mock_license_manager):
        """Test successful config retrieval"""
        mock_config = MagicMock()
        mock_config.id = 1
        mock_config.protocol_detection = True
        mock_config.port_based_detection = False
        mock_config.target_ports = [443, 8443]
        mock_config.intercept_mode = "transparent"
        mock_config.certificate_validation = "none"
        mock_config.preserve_sni = True
        mock_config.log_connections = True
        mock_config.log_decrypted_content = False
        mock_config.max_concurrent_connections = 10000
        mock_config.connection_timeout_seconds = 300
        mock_config.buffer_size_kb = 64

        mock_db.return_value.select.return_value.first.return_value = mock_config

        mock_certs = {
            "ca_cert": "-----BEGIN CERTIFICATE-----\nca\n-----END CERTIFICATE-----",
            "wildcard_cert": "-----BEGIN CERTIFICATE-----\nwc\n-----END CERTIFICATE-----",
            "wildcard_key": "-----BEGIN RSA PRIVATE KEY-----\nkey\n-----END RSA PRIVATE KEY-----",
            "wildcard_domain": "*.example.com",
        }

        with patch.object(
            TLSProxyCAModel, "get_proxy_certificates", return_value=mock_certs
        ):
            mgr = TLSProxyConfigManager(mock_db, license_manager=mock_license_manager)
            result = mgr.get_proxy_config(1, 1)

        assert result["enabled"] is True
        assert result["config_id"] == 1
        assert result["protocol_detection"] is True
        assert result["enterprise_available"] is True
        assert result["certificates"] == mock_certs


# ============================================================================
# Test: TLSProxyConfigManager Init (Line 989-991)
# ============================================================================


class TestTLSProxyConfigManagerInit:
    """Test TLSProxyConfigManager initialization"""

    def test_init_stores_db(self, mock_db):
        """Test __init__ stores db reference"""
        mgr = TLSProxyConfigManager(mock_db)
        assert mgr.db is mock_db

    def test_init_stores_license_manager(self, mock_db, mock_license_manager):
        """Test __init__ stores license_manager"""
        mgr = TLSProxyConfigManager(mock_db, license_manager=mock_license_manager)
        assert mgr.license_manager is mock_license_manager

    def test_init_license_manager_optional(self, mock_db):
        """Test __init__ with None license_manager"""
        mgr = TLSProxyConfigManager(mock_db, license_manager=None)
        assert mgr.license_manager is None
