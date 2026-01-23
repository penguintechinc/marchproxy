"""
TLS Certificate management models for MarchProxy Manager

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import ssl
import socket
import base64
import httpx
import hashlib
from datetime import datetime, timedelta
from typing import Optional, Dict, Any, List, Tuple
from cryptography import x509
from cryptography.hazmat.primitives import serialization, hashes
from cryptography.hazmat.primitives.asymmetric import rsa
from pydal import DAL, Field
from pydantic import BaseModel, validator
import logging

logger = logging.getLogger(__name__)


class CertificateModel:
    """Certificate model for TLS certificate management"""

    @staticmethod
    def define_table(db: DAL):
        """Define certificate table in database"""
        return db.define_table(
            "certificates",
            Field("name", type="string", unique=True, required=True, length=100),
            Field("description", type="text"),
            Field("cert_data", type="text", required=True),
            Field("key_data", type="text", required=True),
            Field("ca_bundle", type="text"),
            Field("source_type", type="string", required=True, length=20),
            Field("source_config", type="json"),
            Field("domain_names", type="json"),
            Field("issuer", type="string", length=255),
            Field("serial_number", type="string", length=100),
            Field("fingerprint_sha256", type="string", length=64),
            Field("auto_renew", type="boolean", default=False),
            Field("renewal_threshold_days", type="integer", default=30),
            Field("issued_at", type="datetime"),
            Field("expires_at", type="datetime"),
            Field("next_renewal_check", type="datetime"),
            Field("renewal_attempts", type="integer", default=0),
            Field("last_renewal_attempt", type="datetime"),
            Field("renewal_error", type="text"),
            Field("is_active", type="boolean", default=True),
            Field("created_by", type="reference users", required=True),
            Field("created_at", type="datetime", default=datetime.utcnow),
            Field("updated_at", type="datetime", update=datetime.utcnow),
            Field("metadata", type="json"),
        )

    @staticmethod
    def create_certificate(
        db: DAL,
        name: str,
        cert_data: str,
        key_data: str,
        source_type: str,
        created_by: int,
        description: str = None,
        ca_bundle: str = None,
        source_config: Dict = None,
        auto_renew: bool = False,
        renewal_threshold_days: int = 30,
    ) -> int:
        """Create new certificate record"""

        # Parse certificate to extract metadata
        cert_info = CertificateModel._parse_certificate(cert_data)
        if not cert_info:
            raise ValueError("Invalid certificate data")

        # Validate private key matches certificate
        if not CertificateModel._validate_key_pair(cert_data, key_data):
            raise ValueError("Private key does not match certificate")

        # Calculate next renewal check time
        next_renewal_check = None
        if auto_renew and cert_info["expires_at"]:
            check_date = cert_info["expires_at"] - timedelta(
                days=renewal_threshold_days
            )
            next_renewal_check = max(check_date, datetime.utcnow() + timedelta(days=1))

        cert_id = db.certificates.insert(
            name=name,
            description=description,
            cert_data=cert_data,
            key_data=key_data,
            ca_bundle=ca_bundle,
            source_type=source_type,
            source_config=source_config or {},
            domain_names=cert_info["domain_names"],
            issuer=cert_info["issuer"],
            serial_number=cert_info["serial_number"],
            fingerprint_sha256=cert_info["fingerprint_sha256"],
            auto_renew=auto_renew,
            renewal_threshold_days=renewal_threshold_days,
            issued_at=cert_info["issued_at"],
            expires_at=cert_info["expires_at"],
            next_renewal_check=next_renewal_check,
            created_by=created_by,
        )

        return cert_id

    @staticmethod
    def _parse_certificate(cert_data: str) -> Optional[Dict[str, Any]]:
        """Parse certificate and extract metadata"""
        try:
            # Handle PEM format
            cert_bytes = cert_data.encode("utf-8")
            cert = x509.load_pem_x509_certificate(cert_bytes)

            # Extract domain names from SAN extension
            domain_names = []
            try:
                san_ext = cert.extensions.get_extension_for_oid(
                    x509.oid.ExtensionOID.SUBJECT_ALTERNATIVE_NAME
                )
                for name in san_ext.value:
                    if isinstance(name, x509.DNSName):
                        domain_names.append(name.value)
            except x509.ExtensionNotFound:
                pass

            # Add subject CN if not in SAN
            cn = None
            for attribute in cert.subject:
                if attribute.oid == x509.oid.NameOID.COMMON_NAME:
                    cn = attribute.value
                    break

            if cn and cn not in domain_names:
                domain_names.insert(0, cn)

            # Calculate fingerprint
            fingerprint = hashlib.sha256(
                cert.public_bytes(serialization.Encoding.DER)
            ).hexdigest()

            return {
                "domain_names": domain_names,
                "issuer": cert.issuer.rfc4514_string(),
                "serial_number": str(cert.serial_number),
                "fingerprint_sha256": fingerprint,
                "issued_at": cert.not_valid_before,
                "expires_at": cert.not_valid_after,
            }

        except Exception as e:
            logger.error(f"Certificate parsing failed: {e}")
            return None

    @staticmethod
    def _validate_key_pair(cert_data: str, key_data: str) -> bool:
        """Validate that private key matches certificate"""
        try:
            cert_bytes = cert_data.encode("utf-8")
            key_bytes = key_data.encode("utf-8")

            cert = x509.load_pem_x509_certificate(cert_bytes)
            private_key = serialization.load_pem_private_key(key_bytes, password=None)

            # Compare public key from cert with public key derived from private key
            cert_public_key = cert.public_key()
            derived_public_key = private_key.public_key()

            # Get public key numbers for comparison
            cert_numbers = cert_public_key.public_numbers()
            derived_numbers = derived_public_key.public_numbers()

            return cert_numbers == derived_numbers

        except Exception as e:
            logger.error(f"Key pair validation failed: {e}")
            return False

    @staticmethod
    def get_certificates_for_renewal(db: DAL) -> List[Dict[str, Any]]:
        """Get certificates that need renewal checking"""
        now = datetime.utcnow()
        certs = db(
            (db.certificates.auto_renew == True)
            & (db.certificates.is_active == True)
            & (db.certificates.next_renewal_check <= now)
        ).select()

        return [
            {
                "id": cert.id,
                "name": cert.name,
                "source_type": cert.source_type,
                "source_config": cert.source_config,
                "expires_at": cert.expires_at,
                "renewal_threshold_days": cert.renewal_threshold_days,
                "renewal_attempts": cert.renewal_attempts,
            }
            for cert in certs
        ]

    @staticmethod
    def update_renewal_attempt(
        db: DAL,
        cert_id: int,
        success: bool,
        new_cert_data: str = None,
        new_key_data: str = None,
        error_message: str = None,
    ) -> bool:
        """Update certificate renewal attempt status"""
        cert = db.certificates[cert_id]
        if not cert:
            return False

        update_data = {
            "last_renewal_attempt": datetime.utcnow(),
            "renewal_attempts": cert.renewal_attempts + 1,
        }

        if success and new_cert_data and new_key_data:
            # Parse new certificate
            cert_info = CertificateModel._parse_certificate(new_cert_data)
            if cert_info:
                # Update certificate data
                update_data.update(
                    {
                        "cert_data": new_cert_data,
                        "key_data": new_key_data,
                        "domain_names": cert_info["domain_names"],
                        "issuer": cert_info["issuer"],
                        "serial_number": cert_info["serial_number"],
                        "fingerprint_sha256": cert_info["fingerprint_sha256"],
                        "issued_at": cert_info["issued_at"],
                        "expires_at": cert_info["expires_at"],
                        "renewal_error": None,
                        "renewal_attempts": 0,
                    }
                )

                # Schedule next renewal check
                next_check = cert_info["expires_at"] - timedelta(
                    days=cert.renewal_threshold_days
                )
                update_data["next_renewal_check"] = max(
                    next_check, datetime.utcnow() + timedelta(days=1)
                )

        else:
            # Renewal failed
            update_data["renewal_error"] = error_message

            # Schedule next attempt (exponential backoff)
            backoff_days = min(1 * (2**cert.renewal_attempts), 7)  # Max 7 days
            update_data["next_renewal_check"] = datetime.utcnow() + timedelta(
                days=backoff_days
            )

        cert.update_record(**update_data)
        return True

    @staticmethod
    def get_expiring_certificates(db: DAL, days: int = 30) -> List[Dict[str, Any]]:
        """Get certificates expiring within specified days"""
        cutoff_date = datetime.utcnow() + timedelta(days=days)
        certs = db(
            (db.certificates.expires_at <= cutoff_date)
            & (db.certificates.is_active == True)
        ).select(orderby=db.certificates.expires_at)

        return [
            {
                "id": cert.id,
                "name": cert.name,
                "domain_names": cert.domain_names,
                "expires_at": cert.expires_at,
                "days_until_expiry": (cert.expires_at - datetime.utcnow()).days,
                "auto_renew": cert.auto_renew,
                "source_type": cert.source_type,
            }
            for cert in certs
        ]


class InfisicalCertificateProvider:
    """Infisical certificate management integration"""

    def __init__(self, api_url: str, token: str, project_id: str):
        self.api_url = api_url.rstrip("/")
        self.token = token
        self.project_id = project_id
        self.timeout = 30.0

    async def fetch_certificate(
        self, secret_path: str, environment: str = "prod"
    ) -> Optional[Dict[str, str]]:
        """Fetch certificate from Infisical"""
        try:
            async with httpx.AsyncClient(timeout=self.timeout) as client:
                response = await client.get(
                    f"{self.api_url}/api/v3/secrets/{secret_path}",
                    headers={
                        "Authorization": f"Bearer {self.token}",
                        "Content-Type": "application/json",
                    },
                    params={"workspaceId": self.project_id, "environment": environment},
                )

                if response.status_code == 200:
                    data = response.json()
                    secret = data.get("secret", {})

                    return {
                        "cert_data": secret.get("certificate", ""),
                        "key_data": secret.get("private_key", ""),
                        "ca_bundle": secret.get("ca_bundle", ""),
                    }

        except Exception as e:
            logger.error(f"Infisical certificate fetch failed: {e}")

        return None


class VaultCertificateProvider:
    """HashiCorp Vault PKI certificate management"""

    def __init__(self, vault_url: str, token: str, pki_path: str = "pki"):
        self.vault_url = vault_url.rstrip("/")
        self.token = token
        self.pki_path = pki_path
        self.timeout = 30.0

    async def issue_certificate(
        self,
        role: str,
        common_name: str,
        alt_names: List[str] = None,
        ttl: str = "720h",
    ) -> Optional[Dict[str, str]]:
        """Issue new certificate from Vault PKI"""
        try:
            payload = {"common_name": common_name, "ttl": ttl, "format": "pem"}

            if alt_names:
                payload["alt_names"] = ",".join(alt_names)

            async with httpx.AsyncClient(timeout=self.timeout) as client:
                response = await client.post(
                    f"{self.vault_url}/v1/{self.pki_path}/issue/{role}",
                    headers={
                        "X-Vault-Token": self.token,
                        "Content-Type": "application/json",
                    },
                    json=payload,
                )

                if response.status_code == 200:
                    data = response.json()
                    cert_data = data.get("data", {})

                    return {
                        "cert_data": cert_data.get("certificate", ""),
                        "key_data": cert_data.get("private_key", ""),
                        "ca_bundle": cert_data.get("ca_chain", ""),
                    }

        except Exception as e:
            logger.error(f"Vault certificate issue failed: {e}")

        return None

    async def revoke_certificate(self, serial_number: str) -> bool:
        """Revoke certificate in Vault"""
        try:
            async with httpx.AsyncClient(timeout=self.timeout) as client:
                response = await client.post(
                    f"{self.vault_url}/v1/{self.pki_path}/revoke",
                    headers={
                        "X-Vault-Token": self.token,
                        "Content-Type": "application/json",
                    },
                    json={"serial_number": serial_number},
                )

                return response.status_code == 200

        except Exception as e:
            logger.error(f"Vault certificate revocation failed: {e}")
            return False


class CertificateManager:
    """Certificate management service"""

    def __init__(self, db: DAL):
        self.db = db

    def create_from_infisical(
        self,
        name: str,
        infisical_config: Dict[str, str],
        created_by: int,
        auto_renew: bool = True,
    ) -> Optional[int]:
        """Create certificate from Infisical"""
        # Validate Infisical configuration
        required_fields = ["api_url", "token", "project_id", "secret_path"]
        if not all(field in infisical_config for field in required_fields):
            raise ValueError("Missing required Infisical configuration fields")

        return CertificateModel.create_certificate(
            self.db,
            name,
            "",
            "",  # cert_data and key_data will be fetched
            source_type="infisical",
            created_by=created_by,
            source_config=infisical_config,
            auto_renew=auto_renew,
        )

    def create_from_vault(
        self,
        name: str,
        vault_config: Dict[str, str],
        created_by: int,
        auto_renew: bool = True,
    ) -> Optional[int]:
        """Create certificate from Vault PKI"""
        # Validate Vault configuration
        required_fields = ["vault_url", "token", "role", "common_name"]
        if not all(field in vault_config for field in required_fields):
            raise ValueError("Missing required Vault configuration fields")

        return CertificateModel.create_certificate(
            self.db,
            name,
            "",
            "",  # cert_data and key_data will be issued
            source_type="vault",
            created_by=created_by,
            source_config=vault_config,
            auto_renew=auto_renew,
        )

    def create_from_upload(
        self,
        name: str,
        cert_data: str,
        key_data: str,
        created_by: int,
        ca_bundle: str = None,
    ) -> int:
        """Create certificate from direct upload"""
        return CertificateModel.create_certificate(
            self.db,
            name,
            cert_data,
            key_data,
            source_type="upload",
            created_by=created_by,
            ca_bundle=ca_bundle,
            auto_renew=False,
        )

    async def renew_certificate(self, cert_id: int) -> bool:
        """Attempt to renew certificate based on source type"""
        cert = self.db.certificates[cert_id]
        if not cert:
            return False

        try:
            if cert.source_type == "infisical":
                return await self._renew_from_infisical(cert)
            elif cert.source_type == "vault":
                return await self._renew_from_vault(cert)
            else:
                # Manual certificates cannot be auto-renewed
                return False

        except Exception as e:
            logger.error(f"Certificate renewal failed for {cert.name}: {e}")
            CertificateModel.update_renewal_attempt(
                self.db, cert_id, False, error_message=str(e)
            )
            return False

    async def _renew_from_infisical(self, cert) -> bool:
        """Renew certificate from Infisical"""
        config = cert.source_config
        provider = InfisicalCertificateProvider(
            config["api_url"], config["token"], config["project_id"]
        )

        cert_data = await provider.fetch_certificate(
            config["secret_path"], config.get("environment", "prod")
        )

        if cert_data and cert_data["cert_data"] and cert_data["key_data"]:
            return CertificateModel.update_renewal_attempt(
                self.db, cert.id, True, cert_data["cert_data"], cert_data["key_data"]
            )

        return False

    async def _renew_from_vault(self, cert) -> bool:
        """Renew certificate from Vault PKI"""
        config = cert.source_config
        provider = VaultCertificateProvider(
            config["vault_url"], config["token"], config.get("pki_path", "pki")
        )

        cert_data = await provider.issue_certificate(
            config["role"],
            config["common_name"],
            config.get("alt_names", []),
            config.get("ttl", "720h"),
        )

        if cert_data and cert_data["cert_data"] and cert_data["key_data"]:
            return CertificateModel.update_renewal_attempt(
                self.db, cert.id, True, cert_data["cert_data"], cert_data["key_data"]
            )

        return False


# Pydantic models for request/response validation
class CreateCertificateRequest(BaseModel):
    name: str
    description: Optional[str] = None
    source_type: str
    cert_data: Optional[str] = None
    key_data: Optional[str] = None
    ca_bundle: Optional[str] = None
    source_config: Optional[Dict[str, Any]] = None
    auto_renew: bool = False
    renewal_threshold_days: int = 30

    @validator("name")
    def validate_name(cls, v):
        if len(v) < 3:
            raise ValueError("Certificate name must be at least 3 characters long")
        return v

    @validator("source_type")
    def validate_source_type(cls, v):
        if v not in ["upload", "infisical", "vault"]:
            raise ValueError("Source type must be one of: upload, infisical, vault")
        return v

    @validator("renewal_threshold_days")
    def validate_threshold(cls, v):
        if not (1 <= v <= 90):
            raise ValueError("Renewal threshold must be between 1 and 90 days")
        return v


class CertificateResponse(BaseModel):
    id: int
    name: str
    description: Optional[str]
    domain_names: List[str]
    issuer: str
    source_type: str
    auto_renew: bool
    issued_at: datetime
    expires_at: datetime
    days_until_expiry: int
    is_active: bool
    created_at: datetime


class CertificateRenewalResponse(BaseModel):
    certificate_id: int
    success: bool
    message: str
    new_expires_at: Optional[datetime]


# TLS Proxy CA Management for Enterprise


class TLSProxyCAModel:
    """Certificate Authority management for TLS proxying (Enterprise feature)"""

    @staticmethod
    def define_table(db: DAL):
        """Define TLS proxy CA table"""
        return db.define_table(
            "tls_proxy_cas",
            Field("name", type="string", unique=True, required=True, length=100),
            Field("description", type="text"),
            Field("cluster_id", type="reference clusters", required=True),
            # CA Certificate and Key
            Field("ca_cert_data", type="text", required=True),
            Field("ca_key_data", type="text", required=True),
            Field("ca_cert_chain", type="text"),  # Full chain if intermediate CA
            # CA Metadata
            Field("ca_subject", type="string", length=255),
            Field("ca_serial_number", type="string", length=100),
            Field("ca_fingerprint_sha256", type="string", length=64),
            Field("ca_issued_at", type="datetime"),
            Field("ca_expires_at", type="datetime"),
            # Wildcard Certificate for proxying
            Field("wildcard_cert_data", type="text", required=True),
            Field("wildcard_key_data", type="text", required=True),
            Field(
                "wildcard_domain", type="string", required=True, length=255
            ),  # e.g., *.company.com
            Field("wildcard_issued_at", type="datetime"),
            Field("wildcard_expires_at", type="datetime"),
            Field("wildcard_serial_number", type="string", length=100),
            # Generation Configuration
            Field(
                "key_type", type="string", default="ecc", length=20
            ),  # 'ecc' or 'rsa'
            Field(
                "key_size", type="integer", default=384
            ),  # ECC curve size or RSA key size
            Field("hash_algorithm", type="string", default="sha512", length=20),
            Field("lifetime_years", type="integer", default=10),
            # Usage Configuration
            Field("enabled", type="boolean", default=False),
            Field("auto_generated", type="boolean", default=False),
            Field("requires_enterprise", type="boolean", default=True),
            Field("license_validated", type="boolean", default=False),
            # Management
            Field("created_by", type="reference users", required=True),
            Field("created_at", type="datetime", default=datetime.utcnow),
            Field("updated_at", type="datetime", update=datetime.utcnow),
            Field("is_active", type="boolean", default=True),
            Field("metadata", type="json"),
        )

    @staticmethod
    def define_proxy_config_table(db: DAL):
        """Define TLS proxy configuration table"""
        return db.define_table(
            "tls_proxy_configs",
            Field("name", type="string", required=True, length=100),
            Field("cluster_id", type="reference clusters", required=True),
            Field("ca_id", type="reference tls_proxy_cas", required=True),
            # Protocol Detection Settings
            Field("enabled", type="boolean", default=False),
            Field(
                "protocol_detection", type="boolean", default=True
            ),  # Use protocol fingerprinting
            Field(
                "port_based_detection", type="boolean", default=False
            ),  # Fallback to port detection
            Field(
                "target_ports", type="json"
            ),  # List of ports to monitor (if port-based)
            # TLS Proxy Behavior
            Field(
                "intercept_mode", type="string", default="transparent", length=20
            ),  # transparent, explicit
            Field(
                "certificate_validation", type="string", default="none", length=20
            ),  # none, warn, strict
            Field("preserve_sni", type="boolean", default=True),
            Field("log_connections", type="boolean", default=True),
            Field(
                "log_decrypted_content", type="boolean", default=False
            ),  # SECURITY: disabled by default
            # Performance Settings
            Field("max_concurrent_connections", type="integer", default=10000),
            Field("connection_timeout_seconds", type="integer", default=300),
            Field("buffer_size_kb", type="integer", default=64),
            # Enterprise Features
            Field("requires_enterprise", type="boolean", default=True),
            Field("license_validated", type="boolean", default=False),
            # Management
            Field("created_by", type="reference users", required=True),
            Field("created_at", type="datetime", default=datetime.utcnow),
            Field("updated_at", type="datetime", update=datetime.utcnow),
            Field("is_active", type="boolean", default=True),
            Field("priority", type="integer", default=100),
        )

    @staticmethod
    def generate_self_signed_ca(
        domain: str,
        key_type: str = "ecc",
        key_size: int = 384,
        hash_algorithm: str = "sha512",
        lifetime_years: int = 10,
    ) -> Dict[str, str]:
        """Generate self-signed CA and wildcard certificate with modern crypto"""
        from cryptography.hazmat.primitives.asymmetric import ec, rsa
        from cryptography.hazmat.primitives import serialization, hashes
        from cryptography import x509
        from cryptography.x509.oid import NameOID, ExtendedKeyUsageOID
        import ipaddress

        try:
            # Generate CA private key
            if key_type == "ecc":
                if key_size == 256:
                    curve = ec.SECP256R1()
                elif key_size == 384:
                    curve = ec.SECP384R1()
                elif key_size == 521:
                    curve = ec.SECP521R1()
                else:
                    raise ValueError(f"Unsupported ECC key size: {key_size}")
                ca_private_key = ec.generate_private_key(curve)
            elif key_type == "rsa":
                if key_size < 2048:
                    raise ValueError("RSA key size must be at least 2048 bits")
                ca_private_key = rsa.generate_private_key(
                    public_exponent=65537, key_size=key_size
                )
            else:
                raise ValueError(f"Unsupported key type: {key_type}")

            # Select hash algorithm
            if hash_algorithm == "sha256":
                hash_alg = hashes.SHA256()
            elif hash_algorithm == "sha384":
                hash_alg = hashes.SHA384()
            elif hash_algorithm == "sha512":
                hash_alg = hashes.SHA512()
            else:
                raise ValueError(f"Unsupported hash algorithm: {hash_algorithm}")

            # Generate CA certificate
            ca_subject = x509.Name(
                [
                    x509.NameAttribute(NameOID.COUNTRY_NAME, "US"),
                    x509.NameAttribute(NameOID.STATE_OR_PROVINCE_NAME, "CA"),
                    x509.NameAttribute(NameOID.LOCALITY_NAME, "San Francisco"),
                    x509.NameAttribute(
                        NameOID.ORGANIZATION_NAME, "MarchProxy Enterprise"
                    ),
                    x509.NameAttribute(
                        NameOID.ORGANIZATIONAL_UNIT_NAME, "TLS Proxy CA"
                    ),
                    x509.NameAttribute(
                        NameOID.COMMON_NAME, f"MarchProxy TLS Proxy CA ({domain})"
                    ),
                ]
            )

            ca_cert = (
                x509.CertificateBuilder()
                .subject_name(ca_subject)
                .issuer_name(ca_subject)  # Self-signed
                .public_key(ca_private_key.public_key())
                .serial_number(x509.random_serial_number())
                .not_valid_before(datetime.utcnow())
                .not_valid_after(
                    datetime.utcnow() + timedelta(days=365 * lifetime_years)
                )
                .add_extension(
                    x509.BasicConstraints(ca=True, path_length=0),
                    critical=True,
                )
                .add_extension(
                    x509.KeyUsage(
                        digital_signature=True,
                        key_cert_sign=True,
                        crl_sign=True,
                        content_commitment=False,
                        data_encipherment=False,
                        key_agreement=False,
                        key_encipherment=False,
                        encipher_only=False,
                        decipher_only=False,
                    ),
                    critical=True,
                )
                .add_extension(
                    x509.ExtendedKeyUsage(
                        [
                            ExtendedKeyUsageOID.CLIENT_AUTH,
                            ExtendedKeyUsageOID.SERVER_AUTH,
                        ]
                    ),
                    critical=True,
                )
                .sign(ca_private_key, hash_alg)
            )

            # Generate wildcard certificate private key
            if key_type == "ecc":
                wildcard_private_key = ec.generate_private_key(curve)
            else:
                wildcard_private_key = rsa.generate_private_key(
                    public_exponent=65537, key_size=key_size
                )

            # Generate wildcard certificate
            wildcard_domain = f"*.{domain}"
            wildcard_subject = x509.Name(
                [
                    x509.NameAttribute(NameOID.COUNTRY_NAME, "US"),
                    x509.NameAttribute(NameOID.STATE_OR_PROVINCE_NAME, "CA"),
                    x509.NameAttribute(NameOID.LOCALITY_NAME, "San Francisco"),
                    x509.NameAttribute(
                        NameOID.ORGANIZATION_NAME, "MarchProxy Enterprise"
                    ),
                    x509.NameAttribute(NameOID.ORGANIZATIONAL_UNIT_NAME, "TLS Proxy"),
                    x509.NameAttribute(NameOID.COMMON_NAME, wildcard_domain),
                ]
            )

            wildcard_cert = (
                x509.CertificateBuilder()
                .subject_name(wildcard_subject)
                .issuer_name(ca_subject)
                .public_key(wildcard_private_key.public_key())
                .serial_number(x509.random_serial_number())
                .not_valid_before(datetime.utcnow())
                .not_valid_after(
                    datetime.utcnow() + timedelta(days=365 * lifetime_years)
                )
                .add_extension(
                    x509.BasicConstraints(ca=False, path_length=None),
                    critical=True,
                )
                .add_extension(
                    x509.KeyUsage(
                        digital_signature=True,
                        key_encipherment=True,
                        content_commitment=False,
                        data_encipherment=False,
                        key_agreement=False,
                        key_cert_sign=False,
                        crl_sign=False,
                        encipher_only=False,
                        decipher_only=False,
                    ),
                    critical=True,
                )
                .add_extension(
                    x509.ExtendedKeyUsage(
                        [
                            ExtendedKeyUsageOID.SERVER_AUTH,
                            ExtendedKeyUsageOID.CLIENT_AUTH,
                        ]
                    ),
                    critical=True,
                )
                .add_extension(
                    x509.SubjectAlternativeName(
                        [
                            x509.DNSName(wildcard_domain),
                            x509.DNSName(domain),  # Also include base domain
                        ]
                    ),
                    critical=False,
                )
                .sign(ca_private_key, hash_alg)
            )

            # Serialize certificates and keys
            ca_cert_pem = ca_cert.public_bytes(serialization.Encoding.PEM).decode(
                "utf-8"
            )
            ca_key_pem = ca_private_key.private_bytes(
                encoding=serialization.Encoding.PEM,
                format=serialization.PrivateFormat.PKCS8,
                encryption_algorithm=serialization.NoEncryption(),
            ).decode("utf-8")

            wildcard_cert_pem = wildcard_cert.public_bytes(
                serialization.Encoding.PEM
            ).decode("utf-8")
            wildcard_key_pem = wildcard_private_key.private_bytes(
                encoding=serialization.Encoding.PEM,
                format=serialization.PrivateFormat.PKCS8,
                encryption_algorithm=serialization.NoEncryption(),
            ).decode("utf-8")

            return {
                "ca_cert": ca_cert_pem,
                "ca_key": ca_key_pem,
                "wildcard_cert": wildcard_cert_pem,
                "wildcard_key": wildcard_key_pem,
                "ca_subject": ca_subject.rfc4514_string(),
                "ca_serial": str(ca_cert.serial_number),
                "ca_fingerprint": hashlib.sha256(
                    ca_cert.public_bytes(serialization.Encoding.DER)
                ).hexdigest(),
                "ca_issued_at": ca_cert.not_valid_before,
                "ca_expires_at": ca_cert.not_valid_after,
                "wildcard_serial": str(wildcard_cert.serial_number),
                "wildcard_issued_at": wildcard_cert.not_valid_before,
                "wildcard_expires_at": wildcard_cert.not_valid_after,
            }

        except Exception as e:
            logger.error(f"Failed to generate CA and wildcard certificate: {e}")
            raise

    @staticmethod
    def create_tls_proxy_ca(
        db: DAL,
        cluster_id: int,
        name: str,
        domain: str,
        user_id: int,
        ca_data: Dict[str, str] = None,
        config: Dict[str, Any] = None,
    ) -> Optional[int]:
        """Create TLS proxy CA configuration"""
        try:
            # Generate CA if not provided
            if ca_data is None:
                config = config or {}
                ca_data = TLSProxyCAModel.generate_self_signed_ca(
                    domain=domain,
                    key_type=config.get("key_type", "ecc"),
                    key_size=config.get("key_size", 384),
                    hash_algorithm=config.get("hash_algorithm", "sha512"),
                    lifetime_years=config.get("lifetime_years", 10),
                )
                auto_generated = True
            else:
                auto_generated = False

            # Insert CA configuration
            ca_id = db.tls_proxy_cas.insert(
                name=name,
                description=f"TLS Proxy CA for domain {domain}",
                cluster_id=cluster_id,
                ca_cert_data=ca_data["ca_cert"],
                ca_key_data=ca_data["ca_key"],
                ca_subject=ca_data["ca_subject"],
                ca_serial_number=ca_data["ca_serial"],
                ca_fingerprint_sha256=ca_data["ca_fingerprint"],
                ca_issued_at=ca_data["ca_issued_at"],
                ca_expires_at=ca_data["ca_expires_at"],
                wildcard_cert_data=ca_data["wildcard_cert"],
                wildcard_key_data=ca_data["wildcard_key"],
                wildcard_domain=f"*.{domain}",
                wildcard_issued_at=ca_data["wildcard_issued_at"],
                wildcard_expires_at=ca_data["wildcard_expires_at"],
                wildcard_serial_number=ca_data["wildcard_serial"],
                key_type=config.get("key_type", "ecc") if config else "ecc",
                key_size=config.get("key_size", 384) if config else 384,
                hash_algorithm=(
                    config.get("hash_algorithm", "sha512") if config else "sha512"
                ),
                lifetime_years=config.get("lifetime_years", 10) if config else 10,
                auto_generated=auto_generated,
                requires_enterprise=True,
                license_validated=True,
                created_by=user_id,
            )

            logger.info(f"Created TLS proxy CA {ca_id} for cluster {cluster_id}")
            return ca_id

        except Exception as e:
            logger.error(f"Failed to create TLS proxy CA: {e}")
            return None

    @staticmethod
    def get_cluster_ca(db: DAL, cluster_id: int) -> Optional[Dict[str, Any]]:
        """Get active TLS proxy CA for a cluster"""
        ca = (
            db(
                (db.tls_proxy_cas.cluster_id == cluster_id)
                & (db.tls_proxy_cas.enabled == True)
                & (db.tls_proxy_cas.is_active == True)
            )
            .select()
            .first()
        )

        if not ca:
            return None

        return {
            "id": ca.id,
            "name": ca.name,
            "description": ca.description,
            "wildcard_domain": ca.wildcard_domain,
            "ca_subject": ca.ca_subject,
            "ca_expires_at": ca.ca_expires_at,
            "wildcard_expires_at": ca.wildcard_expires_at,
            "key_type": ca.key_type,
            "hash_algorithm": ca.hash_algorithm,
            "auto_generated": ca.auto_generated,
            "created_at": ca.created_at,
        }

    @staticmethod
    def get_proxy_certificates(db: DAL, cluster_id: int) -> Optional[Dict[str, str]]:
        """Get certificates for proxy deployment"""
        ca = (
            db(
                (db.tls_proxy_cas.cluster_id == cluster_id)
                & (db.tls_proxy_cas.enabled == True)
                & (db.tls_proxy_cas.is_active == True)
            )
            .select()
            .first()
        )

        if not ca:
            return None

        return {
            "ca_cert": ca.ca_cert_data,
            "wildcard_cert": ca.wildcard_cert_data,
            "wildcard_key": ca.wildcard_key_data,
            "wildcard_domain": ca.wildcard_domain,
        }


class TLSProxyConfigManager:
    """Manager for TLS proxy configuration"""

    def __init__(self, db: DAL, license_manager=None):
        self.db = db
        self.license_manager = license_manager

    def create_tls_proxy_config(
        self, cluster_id: int, ca_id: int, config: Dict[str, Any], user_id: int
    ) -> Tuple[bool, Any]:
        """Create TLS proxy configuration"""
        try:
            # Validate Enterprise license
            if not self.license_manager or not self.license_manager.has_feature(
                "tls_proxy"
            ):
                return False, {"error": "TLS proxying requires Enterprise license"}

            # Validate CA exists and belongs to cluster
            ca = self.db.tls_proxy_cas[ca_id]
            if not ca or ca.cluster_id != cluster_id:
                return False, {"error": "Invalid CA for this cluster"}

            # Create configuration
            config_id = self.db.tls_proxy_configs.insert(
                name=config["name"],
                cluster_id=cluster_id,
                ca_id=ca_id,
                enabled=config.get("enabled", False),
                protocol_detection=config.get("protocol_detection", True),
                port_based_detection=config.get("port_based_detection", False),
                target_ports=config.get("target_ports", [443, 8443]),
                intercept_mode=config.get("intercept_mode", "transparent"),
                certificate_validation=config.get("certificate_validation", "none"),
                preserve_sni=config.get("preserve_sni", True),
                log_connections=config.get("log_connections", True),
                log_decrypted_content=config.get("log_decrypted_content", False),
                max_concurrent_connections=config.get(
                    "max_concurrent_connections", 10000
                ),
                connection_timeout_seconds=config.get(
                    "connection_timeout_seconds", 300
                ),
                buffer_size_kb=config.get("buffer_size_kb", 64),
                requires_enterprise=True,
                license_validated=True,
                created_by=user_id,
            )

            logger.info(
                f"Created TLS proxy config {config_id} for cluster {cluster_id}"
            )
            return True, {"id": config_id}

        except Exception as e:
            logger.error(f"Failed to create TLS proxy config: {e}")
            return False, {"error": str(e)}

    def get_proxy_config(self, cluster_id: int, proxy_id: int) -> Dict[str, Any]:
        """Get TLS proxy configuration for a specific proxy"""
        # Get TLS proxy configuration
        config = (
            self.db(
                (self.db.tls_proxy_configs.cluster_id == cluster_id)
                & (self.db.tls_proxy_configs.enabled == True)
                & (self.db.tls_proxy_configs.is_active == True)
            )
            .select()
            .first()
        )

        if not config:
            return {
                "enabled": False,
                "enterprise_available": (
                    self.license_manager.has_feature("tls_proxy")
                    if self.license_manager
                    else False
                ),
            }

        # Get CA certificates
        certificates = TLSProxyCAModel.get_proxy_certificates(self.db, cluster_id)

        return {
            "enabled": True,
            "config_id": config.id,
            "protocol_detection": config.protocol_detection,
            "port_based_detection": config.port_based_detection,
            "target_ports": config.target_ports or [],
            "intercept_mode": config.intercept_mode,
            "certificate_validation": config.certificate_validation,
            "preserve_sni": config.preserve_sni,
            "log_connections": config.log_connections,
            "log_decrypted_content": config.log_decrypted_content,
            "max_concurrent_connections": config.max_concurrent_connections,
            "connection_timeout_seconds": config.connection_timeout_seconds,
            "buffer_size_kb": config.buffer_size_kb,
            "certificates": certificates,
            "enterprise_available": True,
        }
