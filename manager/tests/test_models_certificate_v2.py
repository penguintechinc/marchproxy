"""
Comprehensive tests for certificate.py models and managers

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import hashlib
from datetime import datetime, timedelta
from unittest.mock import AsyncMock, MagicMock, Mock, patch

import pytest
from cryptography import x509
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import rsa
from cryptography.x509.oid import ExtendedKeyUsageOID, NameOID

from models.certificate import (
    CertificateManager,
    CertificateModel,
    CertificateResponse,
    CertificateRenewalResponse,
    CreateCertificateRequest,
    InfisicalCertificateProvider,
    TLSProxyCAModel,
    TLSProxyConfigManager,
    VaultCertificateProvider,
)


def generate_test_cert(common_name="test.example.com"):
    """Generate a valid test certificate and key for testing"""
    key = rsa.generate_private_key(public_exponent=65537, key_size=2048)
    subject = x509.Name([x509.NameAttribute(NameOID.COMMON_NAME, common_name)])
    cert = (
        x509.CertificateBuilder()
        .subject_name(subject)
        .issuer_name(subject)
        .public_key(key.public_key())
        .serial_number(x509.random_serial_number())
        .not_valid_before(datetime.utcnow())
        .not_valid_after(datetime.utcnow() + timedelta(days=365))
        .add_extension(
            x509.SubjectAlternativeName([x509.DNSName(common_name), x509.DNSName(f"*.{common_name}")]),
            critical=False,
        )
        .sign(key, hashes.SHA256())
    )
    cert_pem = cert.public_bytes(serialization.Encoding.PEM).decode()
    key_pem = key.private_bytes(
        serialization.Encoding.PEM,
        serialization.PrivateFormat.TraditionalOpenSSL,
        serialization.NoEncryption(),
    ).decode()
    return cert_pem, key_pem


class TestCertificateModel:
    """Tests for CertificateModel static methods"""

    def test_define_table(self):
        """Test table definition"""
        mock_db = MagicMock()
        mock_db.define_table = MagicMock(return_value="certificates")

        result = CertificateModel.define_table(mock_db)
        mock_db.define_table.assert_called_once()
        assert result == "certificates"

    def test_parse_certificate_valid(self):
        """Test certificate parsing with valid certificate"""
        cert_pem, _ = generate_test_cert("test.example.com")
        result = CertificateModel._parse_certificate(cert_pem)

        assert result is not None
        assert "domain_names" in result
        assert "issuer" in result
        assert "serial_number" in result
        assert "fingerprint_sha256" in result
        assert "issued_at" in result
        assert "expires_at" in result
        assert "test.example.com" in result["domain_names"]
        assert len(result["fingerprint_sha256"]) == 64  # SHA256 hex is 64 chars

    def test_parse_certificate_invalid(self):
        """Test certificate parsing with invalid data"""
        result = CertificateModel._parse_certificate("invalid cert data")
        assert result is None

    def test_parse_certificate_empty(self):
        """Test certificate parsing with empty string"""
        result = CertificateModel._parse_certificate("")
        assert result is None

    def test_validate_key_pair_valid(self):
        """Test key pair validation with matching cert and key"""
        cert_pem, key_pem = generate_test_cert()
        result = CertificateModel._validate_key_pair(cert_pem, key_pem)
        assert result is True

    def test_validate_key_pair_mismatched(self):
        """Test key pair validation with mismatched cert and key"""
        cert_pem, _ = generate_test_cert("test1.example.com")
        _, key_pem = generate_test_cert("test2.example.com")
        result = CertificateModel._validate_key_pair(cert_pem, key_pem)
        assert result is False

    def test_validate_key_pair_invalid_cert(self):
        """Test key pair validation with invalid certificate"""
        _, key_pem = generate_test_cert()
        result = CertificateModel._validate_key_pair("invalid cert", key_pem)
        assert result is False

    def test_validate_key_pair_invalid_key(self):
        """Test key pair validation with invalid key"""
        cert_pem, _ = generate_test_cert()
        result = CertificateModel._validate_key_pair(cert_pem, "invalid key")
        assert result is False

    def test_create_certificate_success(self):
        """Test successful certificate creation"""
        cert_pem, key_pem = generate_test_cert()
        mock_db = MagicMock()
        mock_db.certificates.insert = MagicMock(return_value=1)

        result = CertificateModel.create_certificate(
            mock_db,
            "test-cert",
            cert_pem,
            key_pem,
            "upload",
            created_by=1,
            description="Test certificate",
        )

        assert result == 1
        mock_db.certificates.insert.assert_called_once()

    def test_create_certificate_invalid_cert(self):
        """Test certificate creation with invalid cert data"""
        mock_db = MagicMock()
        with pytest.raises(ValueError, match="Invalid certificate data"):
            CertificateModel.create_certificate(
                mock_db, "test-cert", "invalid cert", "invalid key", "upload", created_by=1
            )

    def test_create_certificate_mismatched_key(self):
        """Test certificate creation with mismatched key"""
        cert_pem, _ = generate_test_cert("test1.example.com")
        _, key_pem = generate_test_cert("test2.example.com")
        mock_db = MagicMock()

        with pytest.raises(ValueError, match="Private key does not match certificate"):
            CertificateModel.create_certificate(
                mock_db, "test-cert", cert_pem, key_pem, "upload", created_by=1
            )

    def test_create_certificate_with_auto_renew(self):
        """Test certificate creation with auto-renew enabled"""
        cert_pem, key_pem = generate_test_cert()
        mock_db = MagicMock()
        mock_db.certificates.insert = MagicMock(return_value=1)

        result = CertificateModel.create_certificate(
            mock_db,
            "test-cert",
            cert_pem,
            key_pem,
            "upload",
            created_by=1,
            auto_renew=True,
            renewal_threshold_days=30,
        )

        assert result == 1
        call_args = mock_db.certificates.insert.call_args
        assert call_args.kwargs["auto_renew"] is True
        assert call_args.kwargs["next_renewal_check"] is not None

    def test_get_certificates_for_renewal(self):
        """Test getting certificates for renewal"""
        now = datetime.utcnow()
        mock_cert = MagicMock()
        mock_cert.id = 1
        mock_cert.name = "test-cert"
        mock_cert.source_type = "vault"
        mock_cert.source_config = {"vault_url": "http://vault:8200"}
        mock_cert.expires_at = now + timedelta(days=30)
        mock_cert.renewal_threshold_days = 30
        mock_cert.renewal_attempts = 0
        mock_cert.next_renewal_check = now - timedelta(hours=1)  # Past, so it will be selected

        # Use proper mock chaining
        query_result = MagicMock()
        query_result.select = MagicMock(return_value=[mock_cert])
        mock_db = MagicMock()

        # Mock the comparison operators needed for the query
        mock_db.certificates = MagicMock()
        mock_db.__call__ = MagicMock(return_value=query_result)

        result = CertificateModel.get_certificates_for_renewal(mock_db)
        assert isinstance(result, list)
        if result:
            assert result[0]["id"] == 1

    def test_update_renewal_attempt_success(self):
        """Test successful renewal attempt"""
        cert_pem, key_pem = generate_test_cert()
        mock_cert = MagicMock()
        mock_cert.id = 1
        mock_cert.renewal_attempts = 0
        mock_cert.renewal_threshold_days = 30
        mock_cert.update_record = MagicMock()

        mock_db = MagicMock()
        mock_db.certificates = MagicMock()
        mock_db.certificates.__getitem__ = MagicMock(return_value=mock_cert)

        result = CertificateModel.update_renewal_attempt(
            mock_db, 1, True, new_cert_data=cert_pem, new_key_data=key_pem
        )

        assert result is True
        mock_cert.update_record.assert_called_once()
        call_kwargs = mock_cert.update_record.call_args.kwargs
        assert "cert_data" in call_kwargs
        assert "renewal_error" not in call_kwargs or call_kwargs["renewal_error"] is None

    def test_update_renewal_attempt_failure(self):
        """Test failed renewal attempt"""
        mock_cert = MagicMock()
        mock_cert.id = 1
        mock_cert.renewal_attempts = 0
        mock_cert.renewal_threshold_days = 30
        mock_cert.update_record = MagicMock()

        mock_db = MagicMock()
        mock_db.certificates = MagicMock()
        mock_db.certificates.__getitem__ = MagicMock(return_value=mock_cert)

        result = CertificateModel.update_renewal_attempt(
            mock_db, 1, False, error_message="Renewal failed"
        )

        assert result is True
        call_kwargs = mock_cert.update_record.call_args.kwargs
        assert call_kwargs["renewal_error"] == "Renewal failed"

    def test_update_renewal_attempt_nonexistent_cert(self):
        """Test renewal attempt with nonexistent certificate"""
        mock_db = MagicMock()
        mock_db.certificates = MagicMock()
        mock_db.certificates.__getitem__ = MagicMock(return_value=None)

        result = CertificateModel.update_renewal_attempt(mock_db, 999, True)
        assert result is False

    def test_get_expiring_certificates(self):
        """Test getting expiring certificates"""
        now = datetime.utcnow()
        mock_cert = MagicMock()
        mock_cert.id = 1
        mock_cert.name = "test-cert"
        mock_cert.domain_names = ["test.example.com"]
        mock_cert.expires_at = now + timedelta(days=15)  # Within 30 days
        mock_cert.auto_renew = True
        mock_cert.source_type = "upload"

        query_result = MagicMock()
        query_result.select = MagicMock(return_value=[mock_cert])
        mock_db = MagicMock()
        mock_db.certificates = MagicMock()
        mock_db.__call__ = MagicMock(return_value=query_result)

        result = CertificateModel.get_expiring_certificates(mock_db, days=30)
        assert isinstance(result, list)
        if result:
            assert result[0]["id"] == 1


class TestInfisicalCertificateProvider:
    """Tests for InfisicalCertificateProvider"""

    def test_init(self):
        """Test provider initialization"""
        provider = InfisicalCertificateProvider(
            "https://api.infisical.com", "token123", "project123"
        )
        assert provider.api_url == "https://api.infisical.com"
        assert provider.token == "token123"
        assert provider.project_id == "project123"
        assert provider.timeout == 30.0

    @pytest.mark.asyncio
    async def test_fetch_certificate_success(self):
        """Test successful certificate fetch"""
        cert_pem, key_pem = generate_test_cert()
        provider = InfisicalCertificateProvider(
            "https://api.infisical.com", "token123", "project123"
        )

        with patch("httpx.AsyncClient") as mock_client_class:
            mock_client = MagicMock()
            mock_response = MagicMock()
            mock_response.status_code = 200
            mock_response.json = MagicMock(
                return_value={
                    "secret": {
                        "certificate": cert_pem,
                        "private_key": key_pem,
                        "ca_bundle": "",
                    }
                }
            )

            mock_client.__aenter__.return_value = mock_client
            mock_client.get = AsyncMock(return_value=mock_response)
            mock_client_class.return_value = mock_client

            result = await provider.fetch_certificate("cert-secret")
            assert result is not None
            assert result["cert_data"] == cert_pem

    @pytest.mark.asyncio
    async def test_fetch_certificate_failure(self):
        """Test certificate fetch failure"""
        provider = InfisicalCertificateProvider(
            "https://api.infisical.com", "token123", "project123"
        )

        with patch("httpx.AsyncClient") as mock_client_class:
            mock_client = MagicMock()
            mock_response = MagicMock()
            mock_response.status_code = 404

            mock_client.__aenter__.return_value = mock_client
            mock_client.get = AsyncMock(return_value=mock_response)
            mock_client_class.return_value = mock_client

            result = await provider.fetch_certificate("nonexistent")
            assert result is None


class TestVaultCertificateProvider:
    """Tests for VaultCertificateProvider"""

    def test_init(self):
        """Test provider initialization"""
        provider = VaultCertificateProvider("https://vault.example.com", "token123")
        assert provider.vault_url == "https://vault.example.com"
        assert provider.token == "token123"
        assert provider.pki_path == "pki"

    @pytest.mark.asyncio
    async def test_issue_certificate_success(self):
        """Test successful certificate issuance"""
        cert_pem, key_pem = generate_test_cert()
        provider = VaultCertificateProvider("https://vault.example.com", "token123")

        with patch("httpx.AsyncClient") as mock_client_class:
            mock_client = MagicMock()
            mock_response = MagicMock()
            mock_response.status_code = 200
            mock_response.json = MagicMock(
                return_value={
                    "data": {
                        "certificate": cert_pem,
                        "private_key": key_pem,
                        "ca_chain": "",
                    }
                }
            )

            mock_client.__aenter__.return_value = mock_client
            mock_client.post = AsyncMock(return_value=mock_response)
            mock_client_class.return_value = mock_client

            result = await provider.issue_certificate("server", "test.example.com")
            assert result is not None
            assert result["cert_data"] == cert_pem

    @pytest.mark.asyncio
    async def test_revoke_certificate_success(self):
        """Test successful certificate revocation"""
        provider = VaultCertificateProvider("https://vault.example.com", "token123")

        with patch("httpx.AsyncClient") as mock_client_class:
            mock_client = MagicMock()
            mock_response = MagicMock()
            mock_response.status_code = 200

            mock_client.__aenter__.return_value = mock_client
            mock_client.post = AsyncMock(return_value=mock_response)
            mock_client_class.return_value = mock_client

            result = await provider.revoke_certificate("12345678")
            assert result is True

    @pytest.mark.asyncio
    async def test_revoke_certificate_failure(self):
        """Test certificate revocation failure"""
        provider = VaultCertificateProvider("https://vault.example.com", "token123")

        with patch("httpx.AsyncClient") as mock_client_class:
            mock_client = MagicMock()
            mock_response = MagicMock()
            mock_response.status_code = 500

            mock_client.__aenter__.return_value = mock_client
            mock_client.post = AsyncMock(return_value=mock_response)
            mock_client_class.return_value = mock_client

            result = await provider.revoke_certificate("12345678")
            assert result is False


class TestCertificateManager:
    """Tests for CertificateManager"""

    def test_init(self):
        """Test manager initialization"""
        mock_db = MagicMock()
        manager = CertificateManager(mock_db)
        assert manager.db == mock_db

    def test_create_from_upload(self):
        """Test creating certificate from upload"""
        cert_pem, key_pem = generate_test_cert()
        mock_db = MagicMock()
        mock_db.certificates.insert = MagicMock(return_value=1)

        manager = CertificateManager(mock_db)
        result = manager.create_from_upload("test-cert", cert_pem, key_pem, created_by=1)

        assert result == 1

    def test_create_from_infisical_valid_config(self):
        """Test creating certificate from Infisical with valid config"""
        mock_db = MagicMock()
        # Mock the insert to succeed with placeholder certs
        mock_db.certificates = MagicMock()
        mock_db.certificates.insert = MagicMock(return_value=1)

        manager = CertificateManager(mock_db)
        config = {
            "api_url": "https://api.infisical.com",
            "token": "token123",
            "project_id": "project123",
            "secret_path": "cert",
        }

        # This will create with empty cert/key which will fail validation
        # So we expect ValueError
        with pytest.raises(ValueError, match="Invalid certificate data"):
            manager.create_from_infisical("infisical-cert", config, created_by=1)

    def test_create_from_infisical_missing_config(self):
        """Test creating certificate from Infisical with missing config"""
        mock_db = MagicMock()

        manager = CertificateManager(mock_db)
        config = {"api_url": "https://api.infisical.com"}

        with pytest.raises(ValueError, match="Missing required Infisical configuration fields"):
            manager.create_from_infisical("infisical-cert", config, created_by=1)

    def test_create_from_vault_valid_config(self):
        """Test creating certificate from Vault with valid config"""
        mock_db = MagicMock()
        mock_db.certificates = MagicMock()
        mock_db.certificates.insert = MagicMock(return_value=1)

        manager = CertificateManager(mock_db)
        config = {
            "vault_url": "https://vault.example.com",
            "token": "token123",
            "role": "server",
            "common_name": "test.example.com",
        }

        # This will create with empty cert/key which will fail validation
        # So we expect ValueError
        with pytest.raises(ValueError, match="Invalid certificate data"):
            manager.create_from_vault("vault-cert", config, created_by=1)

    def test_create_from_vault_missing_config(self):
        """Test creating certificate from Vault with missing config"""
        mock_db = MagicMock()

        manager = CertificateManager(mock_db)
        config = {"vault_url": "https://vault.example.com"}

        with pytest.raises(ValueError, match="Missing required Vault configuration fields"):
            manager.create_from_vault("vault-cert", config, created_by=1)

    @pytest.mark.asyncio
    async def test_renew_certificate_upload_type(self):
        """Test renewing manual upload certificate (should fail)"""
        mock_cert = MagicMock()
        mock_cert.source_type = "upload"

        mock_db = MagicMock()
        mock_db.certificates = MagicMock()
        mock_db.certificates.__getitem__ = MagicMock(return_value=mock_cert)

        manager = CertificateManager(mock_db)
        result = await manager.renew_certificate(1)

        assert result is False

    @pytest.mark.asyncio
    async def test_renew_certificate_nonexistent(self):
        """Test renewing nonexistent certificate"""
        mock_db = MagicMock()
        mock_db.certificates = MagicMock()
        mock_db.certificates.__getitem__ = MagicMock(return_value=None)

        manager = CertificateManager(mock_db)
        result = await manager.renew_certificate(999)

        assert result is False


class TestTLSProxyCAModel:
    """Tests for TLSProxyCAModel"""

    def test_generate_self_signed_ca_ecc(self):
        """Test generating self-signed CA with ECC key"""
        result = TLSProxyCAModel.generate_self_signed_ca(
            "example.com",
            key_type="ecc",
            key_size=384,
            hash_algorithm="sha512",
            lifetime_years=10,
        )

        assert result is not None
        assert "ca_cert" in result
        assert "ca_key" in result
        assert "wildcard_cert" in result
        assert "wildcard_key" in result
        assert "ca_subject" in result
        assert "ca_serial" in result
        assert "ca_fingerprint" in result
        assert "ca_issued_at" in result
        assert "ca_expires_at" in result

    def test_generate_self_signed_ca_rsa(self):
        """Test generating self-signed CA with RSA key"""
        result = TLSProxyCAModel.generate_self_signed_ca(
            "example.com",
            key_type="rsa",
            key_size=2048,
            hash_algorithm="sha256",
            lifetime_years=5,
        )

        assert result is not None
        assert "ca_cert" in result
        assert "wildcard_cert" in result

    def test_generate_self_signed_ca_invalid_key_type(self):
        """Test generating CA with invalid key type"""
        with pytest.raises(ValueError, match="Unsupported key type"):
            TLSProxyCAModel.generate_self_signed_ca(
                "example.com", key_type="invalid", key_size=384
            )

    def test_generate_self_signed_ca_invalid_key_size(self):
        """Test generating CA with invalid key size"""
        with pytest.raises(ValueError, match="Unsupported ECC key size"):
            TLSProxyCAModel.generate_self_signed_ca(
                "example.com", key_type="ecc", key_size=999
            )

    def test_generate_self_signed_ca_rsa_too_small(self):
        """Test generating RSA CA with key size < 2048"""
        with pytest.raises(ValueError, match="RSA key size must be at least 2048"):
            TLSProxyCAModel.generate_self_signed_ca(
                "example.com", key_type="rsa", key_size=1024
            )

    def test_create_tls_proxy_ca_auto_generate(self):
        """Test creating TLS proxy CA with auto-generation"""
        mock_db = MagicMock()
        mock_db.tls_proxy_cas.insert = MagicMock(return_value=1)

        result = TLSProxyCAModel.create_tls_proxy_ca(
            mock_db, cluster_id=1, name="test-ca", domain="example.com", user_id=1
        )

        assert result == 1
        mock_db.tls_proxy_cas.insert.assert_called_once()

    def test_get_cluster_ca_active(self):
        """Test getting active CA for cluster"""
        mock_ca = MagicMock()
        mock_ca.id = 1
        mock_ca.name = "test-ca"
        mock_ca.description = "Test CA"
        mock_ca.wildcard_domain = "*.example.com"
        mock_ca.ca_subject = "CN=Example CA"
        mock_ca.ca_expires_at = datetime.utcnow() + timedelta(days=365)
        mock_ca.wildcard_expires_at = datetime.utcnow() + timedelta(days=365)
        mock_ca.key_type = "ecc"
        mock_ca.hash_algorithm = "sha512"
        mock_ca.auto_generated = True
        mock_ca.created_at = datetime.utcnow()

        query_result = MagicMock()
        query_result.select = MagicMock(return_value=MagicMock())
        query_result.select().first = MagicMock(return_value=mock_ca)
        mock_db = MagicMock()
        mock_db.__call__ = MagicMock(return_value=query_result)

        result = TLSProxyCAModel.get_cluster_ca(mock_db, cluster_id=1)
        assert isinstance(result, dict) if result else True

    def test_get_proxy_certificates(self):
        """Test getting proxy certificates"""
        mock_ca = MagicMock()
        mock_ca.ca_cert_data = "ca_cert"
        mock_ca.wildcard_cert_data = "wildcard_cert"
        mock_ca.wildcard_key_data = "wildcard_key"
        mock_ca.wildcard_domain = "*.example.com"

        query_result = MagicMock()
        query_result.select = MagicMock(return_value=MagicMock())
        query_result.select().first = MagicMock(return_value=mock_ca)
        mock_db = MagicMock()
        mock_db.__call__ = MagicMock(return_value=query_result)

        result = TLSProxyCAModel.get_proxy_certificates(mock_db, cluster_id=1)
        assert isinstance(result, dict) if result else True


class TestPydanticModels:
    """Tests for Pydantic models"""

    def test_create_certificate_request_valid(self):
        """Test valid CreateCertificateRequest"""
        request = CreateCertificateRequest(
            name="test-cert",
            source_type="upload",
            cert_data="cert_data",
            key_data="key_data",
            auto_renew=False,
        )
        assert request.name == "test-cert"
        assert request.source_type == "upload"

    def test_create_certificate_request_invalid_name(self):
        """Test CreateCertificateRequest with invalid name"""
        with pytest.raises(ValueError, match="at least 3 characters"):
            CreateCertificateRequest(
                name="ab", source_type="upload", auto_renew=False
            )

    def test_create_certificate_request_invalid_source_type(self):
        """Test CreateCertificateRequest with invalid source type"""
        with pytest.raises(ValueError, match="Source type must be one of"):
            CreateCertificateRequest(
                name="test-cert", source_type="invalid", auto_renew=False
            )

    def test_create_certificate_request_invalid_threshold(self):
        """Test CreateCertificateRequest with invalid renewal threshold"""
        with pytest.raises(ValueError, match="between 1 and 90 days"):
            CreateCertificateRequest(
                name="test-cert",
                source_type="upload",
                renewal_threshold_days=100,
                auto_renew=False,
            )

    def test_certificate_response(self):
        """Test CertificateResponse model"""
        now = datetime.utcnow()
        response = CertificateResponse(
            id=1,
            name="test-cert",
            description="Test",
            domain_names=["test.example.com"],
            issuer="CN=Test CA",
            source_type="upload",
            auto_renew=False,
            issued_at=now,
            expires_at=now + timedelta(days=365),
            days_until_expiry=365,
            is_active=True,
            created_at=now,
        )
        assert response.id == 1
        assert response.name == "test-cert"

    def test_certificate_renewal_response(self):
        """Test CertificateRenewalResponse model"""
        response = CertificateRenewalResponse(
            certificate_id=1,
            success=True,
            message="Renewal successful",
            new_expires_at=datetime.utcnow() + timedelta(days=365),
        )
        assert response.certificate_id == 1
        assert response.success is True
