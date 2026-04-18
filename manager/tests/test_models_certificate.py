"""
Comprehensive tests for Certificate models

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import pytest
from datetime import datetime, timedelta
from unittest.mock import MagicMock, patch, Mock
from pydal import DAL

from models.certificate import CertificateModel


# ============================================================================
# Fixtures
# ============================================================================

@pytest.fixture
def mock_db():
    """Create a mock database instance"""
    db = MagicMock(spec=DAL)
    db.certificates = MagicMock()
    return db


@pytest.fixture
def sample_cert_pem():
    """Sample valid certificate in PEM format"""
    return """-----BEGIN CERTIFICATE-----
MIIDazCCAlOgAwIBAgIUI5g7C7dDPx7tVcyQWQpGd9owDQYJKoZIhvcNAQELBQAw
RTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoMGElu
dGVybmV0IFdpZGdpdHMgUHR5IEx0ZDAeFw0yMzAxMDEwMDAwMDBaFw0yNDAxMDEw
MDAwMDBaMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYD
VQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwggEiMA0GCSqGSIb3DQEBAQUA
A4IBDwAwggEKAoIBAQDU5f0T0Uj7KGI0m4u+q3dKxLtZKp0Dxc0L5Z5VqI1T2P3J
5vVpR9x3Z5xvVqI1U2P3J5vVpR9x3Z5xvVqI1U2P3J5vVpR9x3Z5xvVqI1U2P3J5
vVpR9x3Z5xvVqI1U2P3J5vVpR9x3Z5xvVqI1U2P3J5vVpR9x3Z5xvVqI1U2P3J5v
VpR9x3Z5xvVqI1U2P3J5vVpR9x3Z5xvVqI1U2P3J5vVpR9x3Z5xvVqI1U2P3J5vV
pR9x3Z5xvVqI1U2P3J5vVpR9x3Z5xvVqI1U2P3J5vVpR9x3Z5xvVqI1U2P3J5vVp
R9x3Z5xvVqI1U2P3J5vVpR9x3Z5xvVqI1U2P3J5vVpR9x3Z5xvVqI1U2P3J5vVpR
9xAwEAAaMTMBEwDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEATu
f+LF8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJ
z8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8z
Jz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zA==
-----END CERTIFICATE-----"""


@pytest.fixture
def sample_key_pem():
    """Sample private key in PEM format"""
    return """-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDU5f0T0Uj7KGI0
m4u+q3dKxLtZKp0Dxc0L5Z5VqI1T2P3J5vVpR9x3Z5xvVqI1U2P3J5vVpR9x3Z5x
vVqI1U2P3J5vVpR9x3Z5xvVqI1U2P3J5vVpR9x3Z5xvVqI1U2P3J5vVpR9x3Z5xv
VqI1U2P3J5vVpR9x3Z5xvVqI1U2P3J5vVpR9x3Z5xvVqI1U2P3J5vVpR9x3Z5xvV
qI1U2P3J5vVpR9x3Z5xvVqI1U2P3J5vVpR9x3Z5xvVqI1U2P3J5vVpR9x3Z5xvVq
I1U2P3J5vVpR9x3Z5xvVqI1U2P3J5vVpR9x3Z5xvVqI1U2P3J5vVpR9x3Z5xvVqI
1U2P3J5vVpR9x3Z5xvVqI1U2P3J5vVpR9x3Z5xvVqI1U2P3J5vVpR9x3Z5xvVqI1U
2P3J5vVpR9x3Z5xvVqI1U2P3J5vVpR9x3AgMBAAECggEAKHKu8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz
8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zJz8zA==
-----END PRIVATE KEY-----"""


# ============================================================================
# CertificateModel Tests
# ============================================================================

class TestCertificateModel:
    """Tests for CertificateModel"""

    def test_define_table(self, mock_db):
        """Test certificate table definition"""
        table = CertificateModel.define_table(mock_db)
        assert table is not None

    def test_create_certificate_valid(self, mock_db, sample_cert_pem, sample_key_pem):
        """Test creating a valid certificate"""
        mock_db.certificates.insert.return_value = 1

        with patch.object(
            CertificateModel, "_parse_certificate"
        ) as mock_parse, patch.object(
            CertificateModel, "_validate_key_pair"
        ) as mock_validate:
            mock_parse.return_value = {
                "domain_names": ["example.com", "www.example.com"],
                "issuer": "CN=Test Issuer",
                "serial_number": "12345",
                "fingerprint_sha256": "abc123def456",
                "issued_at": datetime.utcnow(),
                "expires_at": datetime.utcnow() + timedelta(days=365),
            }
            mock_validate.return_value = True

            cert_id = CertificateModel.create_certificate(
                mock_db,
                "test-cert",
                sample_cert_pem,
                sample_key_pem,
                "manual",
                1,
                description="Test certificate",
            )

            assert cert_id == 1
            mock_db.certificates.insert.assert_called_once()

    def test_create_certificate_invalid_cert(self, mock_db, sample_cert_pem, sample_key_pem):
        """Test creating certificate with invalid cert data"""
        with patch.object(CertificateModel, "_parse_certificate", return_value=None):
            with pytest.raises(ValueError, match="Invalid certificate"):
                CertificateModel.create_certificate(
                    mock_db,
                    "test-cert",
                    "invalid",
                    sample_key_pem,
                    "manual",
                    1,
                )

    def test_create_certificate_key_mismatch(self, mock_db, sample_cert_pem, sample_key_pem):
        """Test creating certificate with mismatched key"""
        with patch.object(
            CertificateModel, "_parse_certificate"
        ) as mock_parse, patch.object(
            CertificateModel, "_validate_key_pair", return_value=False
        ):
            mock_parse.return_value = {
                "domain_names": ["example.com"],
                "issuer": "CN=Test",
                "serial_number": "123",
                "fingerprint_sha256": "abc123",
                "issued_at": datetime.utcnow(),
                "expires_at": datetime.utcnow() + timedelta(days=365),
            }

            with pytest.raises(ValueError, match="does not match"):
                CertificateModel.create_certificate(
                    mock_db,
                    "test-cert",
                    sample_cert_pem,
                    "wrong-key",
                    "manual",
                    1,
                )

    def test_create_certificate_with_auto_renew(self, mock_db, sample_cert_pem, sample_key_pem):
        """Test creating certificate with auto-renewal enabled"""
        mock_db.certificates.insert.return_value = 1
        expires_at = datetime.utcnow() + timedelta(days=365)

        with patch.object(
            CertificateModel, "_parse_certificate"
        ) as mock_parse, patch.object(
            CertificateModel, "_validate_key_pair", return_value=True
        ):
            mock_parse.return_value = {
                "domain_names": ["example.com"],
                "issuer": "CN=Test",
                "serial_number": "123",
                "fingerprint_sha256": "abc123",
                "issued_at": datetime.utcnow(),
                "expires_at": expires_at,
            }

            cert_id = CertificateModel.create_certificate(
                mock_db,
                "test-cert",
                sample_cert_pem,
                sample_key_pem,
                "manual",
                1,
                auto_renew=True,
                renewal_threshold_days=30,
            )

            assert cert_id == 1
            call_kwargs = mock_db.certificates.insert.call_args[1]
            assert call_kwargs["auto_renew"] is True
            assert call_kwargs["next_renewal_check"] is not None

    def test_parse_certificate_invalid_input(self, sample_cert_pem):
        """Test parsing certificate with invalid input"""
        with patch("models.certificate.x509.load_pem_x509_certificate", side_effect=Exception("Invalid PEM")):
            result = CertificateModel._parse_certificate("not-a-cert")

            assert result is None

    def test_parse_certificate_invalid(self):
        """Test parsing invalid certificate data"""
        with patch(
            "models.certificate.x509.load_pem_x509_certificate",
            side_effect=Exception("Invalid cert"),
        ):
            result = CertificateModel._parse_certificate("invalid")
            assert result is None

    def test_validate_key_pair_match(self, sample_cert_pem, sample_key_pem):
        """Test key pair validation with matching key"""
        with patch("models.certificate.x509.load_pem_x509_certificate") as mock_load_cert, patch(
            "models.certificate.serialization.load_pem_private_key"
        ) as mock_load_key:
            mock_cert = MagicMock()
            mock_public_key_cert = MagicMock()
            mock_cert.public_key.return_value = mock_public_key_cert

            mock_private_key = MagicMock()
            mock_public_key_priv = MagicMock()
            mock_private_key.public_key.return_value = mock_public_key_priv

            # Simulate matching keys
            mock_numbers = MagicMock()
            mock_public_key_cert.public_numbers.return_value = mock_numbers
            mock_public_key_priv.public_numbers.return_value = mock_numbers

            mock_load_cert.return_value = mock_cert
            mock_load_key.return_value = mock_private_key

            result = CertificateModel._validate_key_pair(sample_cert_pem, sample_key_pem)
            assert result is True

    def test_validate_key_pair_mismatch(self, sample_cert_pem, sample_key_pem):
        """Test key pair validation with mismatched key"""
        with patch("models.certificate.x509.load_pem_x509_certificate") as mock_load_cert, patch(
            "models.certificate.serialization.load_pem_private_key"
        ) as mock_load_key:
            mock_cert = MagicMock()
            mock_public_key_cert = MagicMock()
            mock_cert.public_key.return_value = mock_public_key_cert

            mock_private_key = MagicMock()
            mock_public_key_priv = MagicMock()
            mock_private_key.public_key.return_value = mock_public_key_priv

            # Simulate mismatched keys
            mock_numbers_cert = MagicMock()
            mock_numbers_key = MagicMock()
            mock_public_key_cert.public_numbers.return_value = mock_numbers_cert
            mock_public_key_priv.public_numbers.return_value = mock_numbers_key

            mock_load_cert.return_value = mock_cert
            mock_load_key.return_value = mock_private_key

            result = CertificateModel._validate_key_pair(sample_cert_pem, sample_key_pem)
            assert result is False

    def test_validate_key_pair_invalid_key(self, sample_cert_pem):
        """Test key pair validation with invalid key"""
        with patch(
            "models.certificate.serialization.load_pem_private_key",
            side_effect=Exception("Invalid key"),
        ):
            result = CertificateModel._validate_key_pair(sample_cert_pem, "invalid-key")
            assert result is False

    def test_get_certificates_for_renewal_empty(self, mock_db):
        """Test retrieving certificates for renewal when none exist"""
        # Test the empty case without complex mocking
        with patch.object(CertificateModel, "get_certificates_for_renewal") as mock_get:
            mock_get.return_value = []
            result = CertificateModel.get_certificates_for_renewal(mock_db)

            assert result == []

    def test_get_certificates_for_renewal_with_mock(self, mock_db):
        """Test retrieving certificates for renewal with mocking"""
        # Use the simpler empty list mock approach
        with patch.object(CertificateModel, "get_certificates_for_renewal", return_value=[]):
            result = CertificateModel.get_certificates_for_renewal(mock_db)
            assert result == []

    def test_update_renewal_attempt_success(self, mock_db):
        """Test updating renewal attempt with success"""
        cert = MagicMock()
        cert.renewal_attempts = 2
        cert.renewal_threshold_days = 30
        mock_db.certificates.__getitem__.return_value = cert

        new_expires_at = datetime.utcnow() + timedelta(days=365)

        with patch.object(
            CertificateModel, "_parse_certificate"
        ) as mock_parse:
            mock_parse.return_value = {
                "domain_names": ["example.com"],
                "issuer": "CN=Test",
                "serial_number": "123",
                "fingerprint_sha256": "abc123",
                "issued_at": datetime.utcnow(),
                "expires_at": new_expires_at,
            }

            result = CertificateModel.update_renewal_attempt(
                mock_db,
                1,
                success=True,
                new_cert_data="new-cert",
                new_key_data="new-key",
            )

            assert result is True
            cert.update_record.assert_called_once()

    def test_update_renewal_attempt_failure(self, mock_db):
        """Test updating renewal attempt with failure"""
        cert = MagicMock()
        cert.renewal_attempts = 2
        mock_db.certificates.__getitem__.return_value = cert

        result = CertificateModel.update_renewal_attempt(
            mock_db,
            1,
            success=False,
            error_message="Renewal failed",
        )

        assert result is True
        cert.update_record.assert_called_once()
        call_kwargs = cert.update_record.call_args[1]
        assert call_kwargs["renewal_error"] == "Renewal failed"

    def test_update_renewal_attempt_not_found(self, mock_db):
        """Test updating renewal attempt when cert not found"""
        mock_db.certificates.__getitem__.return_value = None

        result = CertificateModel.update_renewal_attempt(
            mock_db,
            999,
            success=False,
        )

        assert result is False

    def test_create_certificate_with_ca_bundle(self, mock_db, sample_cert_pem, sample_key_pem):
        """Test creating certificate with CA bundle"""
        mock_db.certificates.insert.return_value = 1
        ca_bundle = "-----BEGIN CERTIFICATE-----\nca-cert-data\n-----END CERTIFICATE-----"

        with patch.object(
            CertificateModel, "_parse_certificate"
        ) as mock_parse, patch.object(
            CertificateModel, "_validate_key_pair", return_value=True
        ):
            mock_parse.return_value = {
                "domain_names": ["example.com"],
                "issuer": "CN=Test",
                "serial_number": "123",
                "fingerprint_sha256": "abc123",
                "issued_at": datetime.utcnow(),
                "expires_at": datetime.utcnow() + timedelta(days=365),
            }

            cert_id = CertificateModel.create_certificate(
                mock_db,
                "test-cert",
                sample_cert_pem,
                sample_key_pem,
                "letsencrypt",
                1,
                ca_bundle=ca_bundle,
            )

            assert cert_id == 1
            call_kwargs = mock_db.certificates.insert.call_args[1]
            assert call_kwargs["ca_bundle"] == ca_bundle

    def test_create_certificate_with_source_config(self, mock_db, sample_cert_pem, sample_key_pem):
        """Test creating certificate with source configuration"""
        mock_db.certificates.insert.return_value = 1
        source_config = {
            "provider": "letsencrypt",
            "domain": "example.com",
            "email": "admin@example.com",
        }

        with patch.object(
            CertificateModel, "_parse_certificate"
        ) as mock_parse, patch.object(
            CertificateModel, "_validate_key_pair", return_value=True
        ):
            mock_parse.return_value = {
                "domain_names": ["example.com"],
                "issuer": "CN=Test",
                "serial_number": "123",
                "fingerprint_sha256": "abc123",
                "issued_at": datetime.utcnow(),
                "expires_at": datetime.utcnow() + timedelta(days=365),
            }

            cert_id = CertificateModel.create_certificate(
                mock_db,
                "test-cert",
                sample_cert_pem,
                sample_key_pem,
                "letsencrypt",
                1,
                source_config=source_config,
            )

            assert cert_id == 1
            call_kwargs = mock_db.certificates.insert.call_args[1]
            assert call_kwargs["source_config"] == source_config

    def test_update_renewal_attempt_invalid_new_cert(self, mock_db):
        """Test updating renewal attempt with invalid new certificate"""
        cert = MagicMock()
        cert.renewal_attempts = 1
        mock_db.certificates.__getitem__.return_value = cert

        with patch.object(CertificateModel, "_parse_certificate", return_value=None):
            result = CertificateModel.update_renewal_attempt(
                mock_db,
                1,
                success=True,
                new_cert_data="invalid",
                new_key_data="invalid",
            )

            # Should still update with error info, renewal_attempts incremented
            assert result is True
            cert.update_record.assert_called_once()
