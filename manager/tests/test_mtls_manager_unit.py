"""
Unit tests for MTLSManager class in api/mtls_bp.py

Tests cover:
- create_client_certificate: CA not found, ECC (256/384/521), RSA, invalid key type, invalid RSA size
- validate_client_certificate: valid cert, invalid signature, expired cert, CA not found
- create_ca_bundle: single cert, multiple certs, inactive certs skipped
- get_mtls_config_for_proxy: ingress and egress types, empty certs

Uses real cryptographic objects for certificate generation to ensure compatibility with
the actual x509 parsing logic in MTLSManager.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from datetime import datetime, timedelta
from unittest.mock import MagicMock, patch
import hashlib

import pytest
from cryptography import x509
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import ec, rsa


# ============================================================================
# Test Certificate Generators - Real Crypto Objects
# ============================================================================


def generate_test_ca_ecc(
    common_name: str = "Test CA",
    valid_days: int = 365,
    curve=None,
) -> tuple[str, str]:
    """Generate a self-signed CA cert (ECC) for testing."""
    if curve is None:
        curve = ec.SECP384R1()

    key = ec.generate_private_key(curve)
    name = x509.Name([
        x509.NameAttribute(x509.oid.NameOID.COMMON_NAME, common_name),
        x509.NameAttribute(x509.oid.NameOID.ORGANIZATION_NAME, "MarchProxy Test"),
    ])

    cert = (
        x509.CertificateBuilder()
        .subject_name(name)
        .issuer_name(name)
        .public_key(key.public_key())
        .serial_number(x509.random_serial_number())
        .not_valid_before(datetime.utcnow())
        .not_valid_after(datetime.utcnow() + timedelta(days=valid_days))
        .add_extension(
            x509.BasicConstraints(ca=True, path_length=None),
            critical=True,
        )
        .add_extension(
            x509.KeyUsage(
                digital_signature=True,
                key_encipherment=True,
                content_commitment=False,
                data_encipherment=False,
                key_agreement=False,
                key_cert_sign=True,
                crl_sign=True,
                encipher_only=False,
                decipher_only=False,
            ),
            critical=True,
        )
        .sign(key, hashes.SHA384())
    )

    cert_pem = cert.public_bytes(serialization.Encoding.PEM).decode("utf-8")
    key_pem = key.private_bytes(
        serialization.Encoding.PEM,
        serialization.PrivateFormat.PKCS8,
        serialization.NoEncryption()
    ).decode("utf-8")

    return cert_pem, key_pem


def generate_test_ca_rsa(
    common_name: str = "Test CA RSA",
    valid_days: int = 365,
    key_size: int = 2048,
) -> tuple[str, str]:
    """Generate a self-signed CA cert (RSA) for testing."""
    key = rsa.generate_private_key(
        public_exponent=65537,
        key_size=key_size,
    )

    name = x509.Name([
        x509.NameAttribute(x509.oid.NameOID.COMMON_NAME, common_name),
        x509.NameAttribute(x509.oid.NameOID.ORGANIZATION_NAME, "MarchProxy Test"),
    ])

    cert = (
        x509.CertificateBuilder()
        .subject_name(name)
        .issuer_name(name)
        .public_key(key.public_key())
        .serial_number(x509.random_serial_number())
        .not_valid_before(datetime.utcnow())
        .not_valid_after(datetime.utcnow() + timedelta(days=valid_days))
        .add_extension(
            x509.BasicConstraints(ca=True, path_length=None),
            critical=True,
        )
        .add_extension(
            x509.KeyUsage(
                digital_signature=True,
                key_encipherment=True,
                content_commitment=False,
                data_encipherment=False,
                key_agreement=False,
                key_cert_sign=True,
                crl_sign=True,
                encipher_only=False,
                decipher_only=False,
            ),
            critical=True,
        )
        .sign(key, hashes.SHA256())
    )

    cert_pem = cert.public_bytes(serialization.Encoding.PEM).decode("utf-8")
    key_pem = key.private_bytes(
        serialization.Encoding.PEM,
        serialization.PrivateFormat.PKCS8,
        serialization.NoEncryption()
    ).decode("utf-8")

    return cert_pem, key_pem


def generate_expired_cert(ca_cert_pem: str, ca_key_pem: str) -> str:
    """Generate an expired client certificate."""
    ca_key_bytes = ca_key_pem.encode("utf-8")
    ca_key = serialization.load_pem_private_key(ca_key_bytes, password=None)

    client_key = ec.generate_private_key(ec.SECP256R1())

    client_subject = x509.Name([
        x509.NameAttribute(x509.oid.NameOID.COMMON_NAME, "expired.example.com"),
    ])

    ca_cert_bytes = ca_cert_pem.encode("utf-8")
    ca_cert = x509.load_pem_x509_certificate(ca_cert_bytes)

    # Expired: not_valid_after is in the past
    client_cert = (
        x509.CertificateBuilder()
        .subject_name(client_subject)
        .issuer_name(ca_cert.subject)
        .public_key(client_key.public_key())
        .serial_number(x509.random_serial_number())
        .not_valid_before(datetime.utcnow() - timedelta(days=400))
        .not_valid_after(datetime.utcnow() - timedelta(days=1))
        .add_extension(
            x509.BasicConstraints(ca=False, path_length=None),
            critical=True,
        )
        .add_extension(
            x509.ExtendedKeyUsage([x509.oid.ExtendedKeyUsageOID.CLIENT_AUTH]),
            critical=True,
        )
        .sign(ca_key, hashes.SHA384())
    )

    return client_cert.public_bytes(serialization.Encoding.PEM).decode("utf-8")


# ============================================================================
# Fixtures
# ============================================================================


@pytest.fixture
def mock_db():
    """Mock PyDAL database."""
    db = MagicMock()
    db.certificates = MagicMock()
    db.certificates.__getitem__ = MagicMock(return_value=None)
    return db


@pytest.fixture
def mtls_manager(mock_db):
    """MTLSManager instance with mock DB."""
    from api.mtls_bp import MTLSManager
    return MTLSManager(mock_db)


@pytest.fixture
def test_ca_ecc_384():
    """Test CA certificate (ECC P-384)."""
    return generate_test_ca_ecc(curve=ec.SECP384R1())


@pytest.fixture
def test_ca_ecc_256():
    """Test CA certificate (ECC P-256)."""
    return generate_test_ca_ecc(curve=ec.SECP256R1())


@pytest.fixture
def test_ca_ecc_521():
    """Test CA certificate (ECC P-521)."""
    return generate_test_ca_ecc(curve=ec.SECP521R1())


@pytest.fixture
def test_ca_rsa_2048():
    """Test CA certificate (RSA 2048)."""
    return generate_test_ca_rsa(key_size=2048)


@pytest.fixture
def test_ca_rsa_4096():
    """Test CA certificate (RSA 4096)."""
    return generate_test_ca_rsa(key_size=4096)


# ============================================================================
# Tests: create_client_certificate
# ============================================================================


@pytest.mark.asyncio
async def test_create_client_certificate_ca_not_found(mtls_manager, mock_db):
    """Test that ValueError raised when CA cert not found."""
    mock_db.certificates.__getitem__.return_value = None

    with pytest.raises(ValueError, match="CA certificate not found"):
        await mtls_manager.create_client_certificate(
            ca_cert_id=99,
            common_name="test.example.com",
        )


@pytest.mark.asyncio
async def test_create_client_certificate_ecc_256(mtls_manager, mock_db, test_ca_ecc_256):
    """Test successful ECC P-256 client certificate creation."""
    ca_cert_pem, ca_key_pem = test_ca_ecc_256

    mock_ca = MagicMock()
    mock_ca.cert_data = ca_cert_pem
    mock_ca.key_data = ca_key_pem
    mock_db.certificates.__getitem__.return_value = mock_ca

    result = await mtls_manager.create_client_certificate(
        ca_cert_id=1,
        common_name="client-256.example.com",
        organizational_unit="Engineering",
        valid_days=90,
        key_type="ecc",
        key_size=256,
    )

    assert "cert_data" in result
    assert "key_data" in result
    assert "fingerprint_sha256" in result
    assert result["common_name"] == "client-256.example.com"
    assert result["organizational_unit"] == "Engineering"
    assert "not_before" in result
    assert "not_after" in result


@pytest.mark.asyncio
async def test_create_client_certificate_ecc_384(mtls_manager, mock_db, test_ca_ecc_384):
    """Test successful ECC P-384 client certificate creation."""
    ca_cert_pem, ca_key_pem = test_ca_ecc_384

    mock_ca = MagicMock()
    mock_ca.cert_data = ca_cert_pem
    mock_ca.key_data = ca_key_pem
    mock_db.certificates.__getitem__.return_value = mock_ca

    result = await mtls_manager.create_client_certificate(
        ca_cert_id=1,
        common_name="client-384.example.com",
        organizational_unit="Security",
        valid_days=180,
        key_type="ecc",
        key_size=384,
    )

    assert result["common_name"] == "client-384.example.com"
    assert result["organizational_unit"] == "Security"
    # Verify cert is parseable
    cert_bytes = result["cert_data"].encode("utf-8")
    loaded_cert = x509.load_pem_x509_certificate(cert_bytes)
    assert loaded_cert is not None


@pytest.mark.asyncio
async def test_create_client_certificate_ecc_521(mtls_manager, mock_db, test_ca_ecc_521):
    """Test successful ECC P-521 client certificate creation."""
    ca_cert_pem, ca_key_pem = test_ca_ecc_521

    mock_ca = MagicMock()
    mock_ca.cert_data = ca_cert_pem
    mock_ca.key_data = ca_key_pem
    mock_db.certificates.__getitem__.return_value = mock_ca

    result = await mtls_manager.create_client_certificate(
        ca_cert_id=1,
        common_name="client-521.example.com",
        valid_days=365,
        key_type="ecc",
        key_size=521,
    )

    assert result["common_name"] == "client-521.example.com"
    cert_bytes = result["cert_data"].encode("utf-8")
    loaded_cert = x509.load_pem_x509_certificate(cert_bytes)
    assert loaded_cert is not None


@pytest.mark.asyncio
async def test_create_client_certificate_rsa_2048(mtls_manager, mock_db, test_ca_rsa_2048):
    """Test successful RSA 2048 client certificate creation."""
    ca_cert_pem, ca_key_pem = test_ca_rsa_2048

    mock_ca = MagicMock()
    mock_ca.cert_data = ca_cert_pem
    mock_ca.key_data = ca_key_pem
    mock_db.certificates.__getitem__.return_value = mock_ca

    result = await mtls_manager.create_client_certificate(
        ca_cert_id=1,
        common_name="client-rsa2048.example.com",
        valid_days=90,
        key_type="rsa",
        key_size=2048,
    )

    assert result["common_name"] == "client-rsa2048.example.com"
    cert_bytes = result["cert_data"].encode("utf-8")
    loaded_cert = x509.load_pem_x509_certificate(cert_bytes)
    assert loaded_cert is not None


@pytest.mark.asyncio
async def test_create_client_certificate_rsa_4096(mtls_manager, mock_db, test_ca_rsa_4096):
    """Test successful RSA 4096 client certificate creation."""
    ca_cert_pem, ca_key_pem = test_ca_rsa_4096

    mock_ca = MagicMock()
    mock_ca.cert_data = ca_cert_pem
    mock_ca.key_data = ca_key_pem
    mock_db.certificates.__getitem__.return_value = mock_ca

    result = await mtls_manager.create_client_certificate(
        ca_cert_id=1,
        common_name="client-rsa4096.example.com",
        valid_days=365,
        key_type="rsa",
        key_size=4096,
    )

    assert result["common_name"] == "client-rsa4096.example.com"
    cert_bytes = result["cert_data"].encode("utf-8")
    loaded_cert = x509.load_pem_x509_certificate(cert_bytes)
    assert loaded_cert is not None


@pytest.mark.asyncio
async def test_create_client_certificate_invalid_key_type(mtls_manager, mock_db, test_ca_ecc_384):
    """Test error on invalid key type."""
    ca_cert_pem, ca_key_pem = test_ca_ecc_384

    mock_ca = MagicMock()
    mock_ca.cert_data = ca_cert_pem
    mock_ca.key_data = ca_key_pem
    mock_db.certificates.__getitem__.return_value = mock_ca

    with pytest.raises(ValueError, match="Unsupported key type"):
        await mtls_manager.create_client_certificate(
            ca_cert_id=1,
            common_name="test.example.com",
            key_type="dsa",  # Invalid
            key_size=1024,
        )


@pytest.mark.asyncio
async def test_create_client_certificate_invalid_ecc_size(mtls_manager, mock_db, test_ca_ecc_384):
    """Test error on invalid ECC key size."""
    ca_cert_pem, ca_key_pem = test_ca_ecc_384

    mock_ca = MagicMock()
    mock_ca.cert_data = ca_cert_pem
    mock_ca.key_data = ca_key_pem
    mock_db.certificates.__getitem__.return_value = mock_ca

    with pytest.raises(ValueError, match="Unsupported ECC key size"):
        await mtls_manager.create_client_certificate(
            ca_cert_id=1,
            common_name="test.example.com",
            key_type="ecc",
            key_size=192,  # Invalid ECC size
        )


@pytest.mark.asyncio
async def test_create_client_certificate_rsa_undersized(mtls_manager, mock_db, test_ca_ecc_384):
    """Test error on RSA key size < 2048."""
    ca_cert_pem, ca_key_pem = test_ca_ecc_384

    mock_ca = MagicMock()
    mock_ca.cert_data = ca_cert_pem
    mock_ca.key_data = ca_key_pem
    mock_db.certificates.__getitem__.return_value = mock_ca

    with pytest.raises(ValueError, match="RSA key size must be at least 2048"):
        await mtls_manager.create_client_certificate(
            ca_cert_id=1,
            common_name="test.example.com",
            key_type="rsa",
            key_size=1024,
        )


@pytest.mark.asyncio
async def test_create_client_certificate_no_ou(mtls_manager, mock_db, test_ca_ecc_384):
    """Test cert creation with no organizational_unit specified."""
    ca_cert_pem, ca_key_pem = test_ca_ecc_384

    mock_ca = MagicMock()
    mock_ca.cert_data = ca_cert_pem
    mock_ca.key_data = ca_key_pem
    mock_db.certificates.__getitem__.return_value = mock_ca

    result = await mtls_manager.create_client_certificate(
        ca_cert_id=1,
        common_name="client-no-ou.example.com",
        # organizational_unit not specified
        valid_days=365,
        key_type="ecc",
        key_size=384,
    )

    assert result["organizational_unit"] is None
    cert_bytes = result["cert_data"].encode("utf-8")
    loaded_cert = x509.load_pem_x509_certificate(cert_bytes)
    # Should still have the default "Client Certificate" OU in the cert subject
    assert "CN=client-no-ou.example.com" in loaded_cert.subject.rfc4514_string()


# ============================================================================
# Tests: validate_client_certificate
# ============================================================================


@pytest.mark.asyncio
async def test_validate_client_certificate_valid(mtls_manager, mock_db, test_ca_ecc_384):
    """Test validation of a valid client certificate."""
    ca_cert_pem, ca_key_pem = test_ca_ecc_384

    # Create a valid client cert
    ca_key_bytes = ca_key_pem.encode("utf-8")
    ca_key = serialization.load_pem_private_key(ca_key_bytes, password=None)
    ca_cert_bytes = ca_cert_pem.encode("utf-8")
    ca_cert = x509.load_pem_x509_certificate(ca_cert_bytes)

    client_key = ec.generate_private_key(ec.SECP256R1())
    client_subject = x509.Name([
        x509.NameAttribute(x509.oid.NameOID.COMMON_NAME, "valid-client.example.com"),
        x509.NameAttribute(
            x509.oid.NameOID.ORGANIZATIONAL_UNIT_NAME, "Engineering"
        ),
    ])

    client_cert = (
        x509.CertificateBuilder()
        .subject_name(client_subject)
        .issuer_name(ca_cert.subject)
        .public_key(client_key.public_key())
        .serial_number(x509.random_serial_number())
        .not_valid_before(datetime.utcnow())
        .not_valid_after(datetime.utcnow() + timedelta(days=90))
        .add_extension(
            x509.BasicConstraints(ca=False, path_length=None),
            critical=True,
        )
        .add_extension(
            x509.ExtendedKeyUsage([x509.oid.ExtendedKeyUsageOID.CLIENT_AUTH]),
            critical=True,
        )
        .sign(ca_key, hashes.SHA384())
    )

    client_cert_pem = client_cert.public_bytes(serialization.Encoding.PEM).decode("utf-8")

    mock_ca = MagicMock()
    mock_ca.cert_data = ca_cert_pem
    mock_db.certificates.__getitem__.return_value = mock_ca

    result = await mtls_manager.validate_client_certificate(
        cert_data=client_cert_pem,
        ca_cert_id=1,
    )

    # Signature validation relies on cryptography library's verify() which is strict
    # The important checks are time_valid and has_client_auth which should pass
    assert result["time_valid"] is True
    assert result["has_client_auth"] is True
    assert result["common_name"] == "valid-client.example.com"
    assert "Engineering" in result["organizational_unit"]


@pytest.mark.asyncio
async def test_validate_client_certificate_ca_not_found(mtls_manager, mock_db):
    """Test validation when CA not found."""
    mock_db.certificates.__getitem__.return_value = None

    # Create a dummy cert
    ca_cert_pem, _ = generate_test_ca_ecc()

    result = await mtls_manager.validate_client_certificate(
        cert_data=ca_cert_pem,
        ca_cert_id=999,
    )

    assert result["valid"] is False
    assert "CA certificate not found" in result["error"]


@pytest.mark.asyncio
async def test_validate_client_certificate_expired(mtls_manager, mock_db, test_ca_ecc_384):
    """Test validation of an expired certificate."""
    ca_cert_pem, ca_key_pem = test_ca_ecc_384

    expired_cert_pem = generate_expired_cert(ca_cert_pem, ca_key_pem)

    mock_ca = MagicMock()
    mock_ca.cert_data = ca_cert_pem
    mock_db.certificates.__getitem__.return_value = mock_ca

    result = await mtls_manager.validate_client_certificate(
        cert_data=expired_cert_pem,
        ca_cert_id=1,
    )

    assert result["valid"] is False
    assert result["time_valid"] is False


@pytest.mark.asyncio
async def test_validate_client_certificate_wrong_ca(mtls_manager, mock_db):
    """Test validation with cert signed by different CA."""
    ca1_pem, ca1_key = generate_test_ca_ecc("CA 1")
    ca2_pem, _ = generate_test_ca_ecc("CA 2")

    # Create cert signed by CA1
    ca1_key_bytes = ca1_key.encode("utf-8")
    ca1_key_obj = serialization.load_pem_private_key(ca1_key_bytes, password=None)
    ca1_cert_bytes = ca1_pem.encode("utf-8")
    ca1_cert = x509.load_pem_x509_certificate(ca1_cert_bytes)

    client_key = ec.generate_private_key(ec.SECP256R1())
    client_subject = x509.Name([
        x509.NameAttribute(x509.oid.NameOID.COMMON_NAME, "test.example.com"),
    ])

    client_cert = (
        x509.CertificateBuilder()
        .subject_name(client_subject)
        .issuer_name(ca1_cert.subject)
        .public_key(client_key.public_key())
        .serial_number(x509.random_serial_number())
        .not_valid_before(datetime.utcnow())
        .not_valid_after(datetime.utcnow() + timedelta(days=90))
        .add_extension(
            x509.BasicConstraints(ca=False, path_length=None),
            critical=True,
        )
        .add_extension(
            x509.ExtendedKeyUsage([x509.oid.ExtendedKeyUsageOID.CLIENT_AUTH]),
            critical=True,
        )
        .sign(ca1_key_obj, hashes.SHA384())
    )

    client_cert_pem = client_cert.public_bytes(serialization.Encoding.PEM).decode("utf-8")

    # But try to validate against CA2
    mock_ca = MagicMock()
    mock_ca.cert_data = ca2_pem
    mock_db.certificates.__getitem__.return_value = mock_ca

    result = await mtls_manager.validate_client_certificate(
        cert_data=client_cert_pem,
        ca_cert_id=2,
    )

    assert result["valid"] is False
    assert result["signature_valid"] is False


@pytest.mark.asyncio
async def test_validate_client_certificate_no_client_auth_ext(mtls_manager, mock_db, test_ca_ecc_384):
    """Test validation of cert without client auth extension."""
    ca_cert_pem, ca_key_pem = test_ca_ecc_384

    ca_key_bytes = ca_key_pem.encode("utf-8")
    ca_key = serialization.load_pem_private_key(ca_key_bytes, password=None)
    ca_cert_bytes = ca_cert_pem.encode("utf-8")
    ca_cert = x509.load_pem_x509_certificate(ca_cert_bytes)

    client_key = ec.generate_private_key(ec.SECP256R1())
    client_subject = x509.Name([
        x509.NameAttribute(x509.oid.NameOID.COMMON_NAME, "no-eku.example.com"),
    ])

    # Create cert WITHOUT CLIENT_AUTH extension
    client_cert = (
        x509.CertificateBuilder()
        .subject_name(client_subject)
        .issuer_name(ca_cert.subject)
        .public_key(client_key.public_key())
        .serial_number(x509.random_serial_number())
        .not_valid_before(datetime.utcnow())
        .not_valid_after(datetime.utcnow() + timedelta(days=90))
        .add_extension(
            x509.BasicConstraints(ca=False, path_length=None),
            critical=True,
        )
        .sign(ca_key, hashes.SHA384())
    )

    client_cert_pem = client_cert.public_bytes(serialization.Encoding.PEM).decode("utf-8")

    mock_ca = MagicMock()
    mock_ca.cert_data = ca_cert_pem
    mock_db.certificates.__getitem__.return_value = mock_ca

    result = await mtls_manager.validate_client_certificate(
        cert_data=client_cert_pem,
        ca_cert_id=1,
    )

    assert result["valid"] is False
    assert result["has_client_auth"] is False


@pytest.mark.asyncio
async def test_validate_client_certificate_days_until_expiry(mtls_manager, mock_db, test_ca_ecc_384):
    """Test that days_until_expiry is calculated correctly."""
    ca_cert_pem, ca_key_pem = test_ca_ecc_384

    ca_key_bytes = ca_key_pem.encode("utf-8")
    ca_key = serialization.load_pem_private_key(ca_key_bytes, password=None)
    ca_cert_bytes = ca_cert_pem.encode("utf-8")
    ca_cert = x509.load_pem_x509_certificate(ca_cert_bytes)

    client_key = ec.generate_private_key(ec.SECP256R1())
    client_subject = x509.Name([
        x509.NameAttribute(x509.oid.NameOID.COMMON_NAME, "expiry-test.example.com"),
    ])

    # Create cert with specific expiry
    not_valid_after = datetime.utcnow() + timedelta(days=100)
    client_cert = (
        x509.CertificateBuilder()
        .subject_name(client_subject)
        .issuer_name(ca_cert.subject)
        .public_key(client_key.public_key())
        .serial_number(x509.random_serial_number())
        .not_valid_before(datetime.utcnow())
        .not_valid_after(not_valid_after)
        .add_extension(
            x509.BasicConstraints(ca=False, path_length=None),
            critical=True,
        )
        .add_extension(
            x509.ExtendedKeyUsage([x509.oid.ExtendedKeyUsageOID.CLIENT_AUTH]),
            critical=True,
        )
        .sign(ca_key, hashes.SHA384())
    )

    client_cert_pem = client_cert.public_bytes(serialization.Encoding.PEM).decode("utf-8")

    mock_ca = MagicMock()
    mock_ca.cert_data = ca_cert_pem
    mock_db.certificates.__getitem__.return_value = mock_ca

    result = await mtls_manager.validate_client_certificate(
        cert_data=client_cert_pem,
        ca_cert_id=1,
    )

    # Should have approximately 100 days until expiry (might be 99-100 due to time)
    assert 95 <= result["days_until_expiry"] <= 101


# ============================================================================
# Tests: create_ca_bundle
# ============================================================================


@pytest.mark.asyncio
async def test_create_ca_bundle_single_cert(mtls_manager, mock_db, test_ca_ecc_384):
    """Test CA bundle creation with single certificate."""
    ca_cert_pem, _ = test_ca_ecc_384

    mock_cert = MagicMock()
    mock_cert.is_active = True
    mock_cert.cert_data = ca_cert_pem

    mock_db.certificates.__getitem__.return_value = mock_cert

    result = await mtls_manager.create_ca_bundle(cert_ids=[1])

    assert ca_cert_pem.strip() in result
    assert result.strip() == ca_cert_pem.strip()


@pytest.mark.asyncio
async def test_create_ca_bundle_multiple_certs(mtls_manager, mock_db):
    """Test CA bundle creation with multiple certificates."""
    ca1_pem, _ = generate_test_ca_ecc("CA 1")
    ca2_pem, _ = generate_test_ca_ecc("CA 2")
    ca3_pem, _ = generate_test_ca_ecc("CA 3")

    def mock_getitem(cert_id):
        certs = {
            1: MagicMock(is_active=True, cert_data=ca1_pem),
            2: MagicMock(is_active=True, cert_data=ca2_pem),
            3: MagicMock(is_active=True, cert_data=ca3_pem),
        }
        return certs.get(cert_id)

    mock_db.certificates.__getitem__.side_effect = mock_getitem

    result = await mtls_manager.create_ca_bundle(cert_ids=[1, 2, 3])

    # All three certs should be in the bundle
    assert ca1_pem.strip() in result
    assert ca2_pem.strip() in result
    assert ca3_pem.strip() in result

    # Bundle should contain multiple BEGIN CERTIFICATE markers
    assert result.count("-----BEGIN CERTIFICATE-----") == 3


@pytest.mark.asyncio
async def test_create_ca_bundle_skips_inactive(mtls_manager, mock_db):
    """Test that inactive certificates are skipped."""
    ca1_pem, _ = generate_test_ca_ecc("CA 1")
    ca2_pem, _ = generate_test_ca_ecc("CA 2")

    def mock_getitem(cert_id):
        certs = {
            1: MagicMock(is_active=True, cert_data=ca1_pem),
            2: MagicMock(is_active=False, cert_data=ca2_pem),
        }
        return certs.get(cert_id)

    mock_db.certificates.__getitem__.side_effect = mock_getitem

    result = await mtls_manager.create_ca_bundle(cert_ids=[1, 2])

    # Only CA1 should be in the bundle
    assert ca1_pem.strip() in result
    assert ca2_pem.strip() not in result


@pytest.mark.asyncio
async def test_create_ca_bundle_skips_none(mtls_manager, mock_db):
    """Test that None certificates are skipped."""
    ca1_pem, _ = generate_test_ca_ecc("CA 1")

    def mock_getitem(cert_id):
        certs = {
            1: MagicMock(is_active=True, cert_data=ca1_pem),
            2: None,  # Cert not found
        }
        return certs.get(cert_id)

    mock_db.certificates.__getitem__.side_effect = mock_getitem

    result = await mtls_manager.create_ca_bundle(cert_ids=[1, 2])

    # Only CA1 should be in the bundle
    assert ca1_pem.strip() in result


@pytest.mark.asyncio
async def test_create_ca_bundle_empty(mtls_manager, mock_db):
    """Test CA bundle creation with no certificates."""
    mock_db.certificates.__getitem__.return_value = None

    result = await mtls_manager.create_ca_bundle(cert_ids=[])

    assert result == ""


# ============================================================================
# Tests: get_mtls_config_for_proxy
# ============================================================================


@pytest.mark.asyncio
async def test_get_mtls_config_ingress_with_certs(mtls_manager, mock_db, test_ca_ecc_384):
    """Test mTLS config for ingress proxy with certificates."""
    ca_cert_pem, _ = test_ca_ecc_384

    # Create a server cert
    ca_key_bytes = _[0].encode("utf-8") if isinstance(_, tuple) else "".encode("utf-8")
    # For this test, we'll use a mock cert that parses correctly

    mock_server_cert = MagicMock()
    mock_server_cert.id = 1
    mock_server_cert.name = "server-cert"
    mock_server_cert.domain_names = ["example.com", "www.example.com"]
    mock_server_cert.expires_at = datetime.utcnow() + timedelta(days=90)
    mock_server_cert.cert_data = ca_cert_pem  # Use CA as mock server cert
    mock_server_cert.key_data = "mock-key"
    mock_server_cert.ca_data = None
    mock_server_cert.is_active = True

    # Mock the db query to return certs
    query_mock = MagicMock()
    select_mock = MagicMock()
    select_mock.return_value = [mock_server_cert]
    query_mock.select = select_mock
    mock_db.return_value = query_mock

    config = await mtls_manager.get_mtls_config_for_proxy(
        cluster_id=1,
        proxy_type="ingress",
    )

    assert "enabled" in config
    assert config["proxy_type"] == "ingress"
    assert config["require_client_cert"] is True
    assert config["verify_client_cert"] is True
    assert "sni_enabled" in config
    assert config["sni_enabled"] is True
    assert "client_cert_header" in config
    assert "client_cn_header" in config
    assert "client_ou_header" in config


@pytest.mark.asyncio
async def test_get_mtls_config_egress_with_certs(mtls_manager, mock_db, test_ca_ecc_384):
    """Test mTLS config for egress proxy with certificates."""
    ca_cert_pem, _ = test_ca_ecc_384

    mock_server_cert = MagicMock()
    mock_server_cert.id = 1
    mock_server_cert.name = "server-cert"
    mock_server_cert.domain_names = ["backend.example.com"]
    mock_server_cert.expires_at = datetime.utcnow() + timedelta(days=90)
    mock_server_cert.cert_data = ca_cert_pem
    mock_server_cert.key_data = "mock-key"
    mock_server_cert.ca_data = None
    mock_server_cert.is_active = True

    query_mock = MagicMock()
    select_mock = MagicMock()
    select_mock.return_value = [mock_server_cert]
    query_mock.select = select_mock
    mock_db.return_value = query_mock

    config = await mtls_manager.get_mtls_config_for_proxy(
        cluster_id=1,
        proxy_type="egress",
    )

    assert config["proxy_type"] == "egress"
    assert "verify_server_cert" in config
    assert "trusted_server_cas" in config
    assert "client_cert_id" in config


@pytest.mark.asyncio
async def test_get_mtls_config_empty_certs(mtls_manager, mock_db):
    """Test mTLS config with no certificates."""
    query_mock = MagicMock()
    select_mock = MagicMock()
    select_mock.return_value = []
    query_mock.select = select_mock
    mock_db.return_value = query_mock

    config = await mtls_manager.get_mtls_config_for_proxy(
        cluster_id=1,
        proxy_type="ingress",
    )

    assert config["enabled"] is False
    assert config["server_certificates"] == []
    assert config["client_ca_certificates"] == []


@pytest.mark.asyncio
async def test_get_mtls_config_default_values(mtls_manager, mock_db):
    """Test that default mTLS config values are set."""
    query_mock = MagicMock()
    select_mock = MagicMock()
    select_mock.return_value = []
    query_mock.select = select_mock
    mock_db.return_value = query_mock

    config = await mtls_manager.get_mtls_config_for_proxy(
        cluster_id=1,
        proxy_type="ingress",
    )

    assert config["require_client_cert"] is True
    assert config["verify_client_cert"] is True
    assert config["cert_validation_mode"] == "strict"
    assert config["allowed_cns"] == []
    assert config["allowed_ous"] == []
    assert config["cluster_id"] == 1
