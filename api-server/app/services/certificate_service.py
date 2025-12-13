"""
Certificate Service - Business logic for TLS certificate management

Handles certificate CRUD, Infisical/Vault integration, expiry monitoring,
and auto-renewal scheduling.
"""

import logging
import json
import httpx
from datetime import datetime, timedelta
from typing import Optional
from cryptography import x509
from cryptography.hazmat.backends import default_backend
from cryptography.hazmat.primitives import serialization

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.config import settings
from app.models.sqlalchemy.certificate import Certificate, CertificateSource

logger = logging.getLogger(__name__)


class CertificateServiceError(Exception):
    """Base exception for certificate service errors"""
    pass


class InvalidCertificateError(CertificateServiceError):
    """Raised when certificate is invalid or malformed"""
    pass


class ExternalServiceError(CertificateServiceError):
    """Raised when external service (Infisical/Vault) fails"""
    pass


class CertificateService:
    """Service for managing TLS certificates"""

    def __init__(self, db: AsyncSession):
        self.db = db

    def parse_certificate(self, cert_pem: str) -> dict:
        """
        Parse PEM certificate and extract metadata

        Args:
            cert_pem: PEM-encoded certificate

        Returns:
            Dict with certificate metadata

        Raises:
            InvalidCertificateError: If certificate is malformed
        """
        try:
            cert = x509.load_pem_x509_certificate(
                cert_pem.encode(),
                default_backend()
            )

            # Extract subject common name
            common_name = None
            for attr in cert.subject:
                if attr.oid == x509.NameOID.COMMON_NAME:
                    common_name = attr.value
                    break

            # Extract issuer
            issuer = None
            for attr in cert.issuer:
                if attr.oid == x509.NameOID.COMMON_NAME:
                    issuer = attr.value
                    break

            # Extract SANs
            san_list = []
            try:
                san_ext = cert.extensions.get_extension_for_oid(
                    x509.ExtensionOID.SUBJECT_ALTERNATIVE_NAME
                )
                san_list = [
                    name.value for name in san_ext.value
                ]
            except x509.ExtensionNotFound:
                pass

            return {
                "common_name": common_name,
                "issuer": issuer,
                "valid_from": cert.not_valid_before_utc,
                "valid_until": cert.not_valid_after_utc,
                "subject_alt_names": json.dumps(san_list) if san_list else None
            }

        except Exception as e:
            logger.error(f"Failed to parse certificate: {e}")
            raise InvalidCertificateError(f"Invalid certificate: {e}")

    async def create_certificate_upload(
        self,
        name: str,
        cert_data: str,
        key_data: str,
        ca_chain: Optional[str],
        description: Optional[str],
        auto_renew: bool,
        renew_before_days: int,
        created_by: int
    ) -> Certificate:
        """
        Create certificate from direct upload

        Args:
            name: Certificate name
            cert_data: PEM-encoded certificate
            key_data: PEM-encoded private key
            ca_chain: Optional CA chain
            description: Optional description
            auto_renew: Enable auto-renewal
            renew_before_days: Days before expiry to renew
            created_by: User ID who created it

        Returns:
            Certificate object
        """
        # Parse certificate metadata
        cert_metadata = self.parse_certificate(cert_data)

        # Create certificate record
        cert = Certificate(
            name=name,
            description=description,
            source_type=CertificateSource.UPLOAD,
            cert_data=cert_data,
            key_data=key_data,
            ca_chain=ca_chain,
            common_name=cert_metadata["common_name"],
            issuer=cert_metadata["issuer"],
            valid_from=cert_metadata["valid_from"],
            valid_until=cert_metadata["valid_until"],
            subject_alt_names=cert_metadata["subject_alt_names"],
            auto_renew=auto_renew,
            renew_before_days=renew_before_days,
            created_by=created_by,
            is_active=True
        )

        self.db.add(cert)
        await self.db.commit()
        await self.db.refresh(cert)

        logger.info(f"Certificate created: {name} (upload)")
        return cert

    async def create_certificate_infisical(
        self,
        name: str,
        secret_path: str,
        project_id: str,
        environment: str,
        description: Optional[str],
        auto_renew: bool,
        renew_before_days: int,
        created_by: int
    ) -> Certificate:
        """
        Create certificate from Infisical

        Args:
            name: Certificate name
            secret_path: Infisical secret path
            project_id: Infisical project ID
            environment: Infisical environment
            description: Optional description
            auto_renew: Enable auto-renewal
            renew_before_days: Days before expiry to renew
            created_by: User ID who created it

        Returns:
            Certificate object
        """
        # Fetch from Infisical
        cert_data, key_data, ca_chain = await self._fetch_from_infisical(
            secret_path, project_id, environment
        )

        # Parse metadata
        cert_metadata = self.parse_certificate(cert_data)

        # Create certificate record
        cert = Certificate(
            name=name,
            description=description,
            source_type=CertificateSource.INFISICAL,
            cert_data=cert_data,
            key_data=key_data,
            ca_chain=ca_chain,
            common_name=cert_metadata["common_name"],
            issuer=cert_metadata["issuer"],
            valid_from=cert_metadata["valid_from"],
            valid_until=cert_metadata["valid_until"],
            subject_alt_names=cert_metadata["subject_alt_names"],
            infisical_secret_path=secret_path,
            infisical_project_id=project_id,
            infisical_environment=environment,
            auto_renew=auto_renew,
            renew_before_days=renew_before_days,
            created_by=created_by,
            is_active=True
        )

        self.db.add(cert)
        await self.db.commit()
        await self.db.refresh(cert)

        logger.info(f"Certificate created: {name} (Infisical)")
        return cert

    async def _fetch_from_infisical(
        self,
        secret_path: str,
        project_id: str,
        environment: str
    ) -> tuple[str, str, Optional[str]]:
        """
        Fetch certificate from Infisical

        Args:
            secret_path: Path to secret in Infisical (e.g., "/certificates/tls")
            project_id: Infisical project ID
            environment: Environment name (e.g., "prod", "dev")

        Returns:
            Tuple of (cert_data, key_data, ca_chain)

        Raises:
            ExternalServiceError: If Infisical API call fails
        """
        if not settings.INFISICAL_TOKEN:
            raise ExternalServiceError(
                "Infisical integration not configured. "
                "Set INFISICAL_TOKEN environment variable."
            )

        try:
            async with httpx.AsyncClient(timeout=30.0) as client:
                headers = {
                    "Authorization": f"Bearer {settings.INFISICAL_TOKEN}",
                    "Content-Type": "application/json"
                }

                url = f"{settings.INFISICAL_URL}/api/v3/secrets/raw/{secret_path}"
                params = {
                    "workspaceId": project_id,
                    "environment": environment
                }

                logger.info(f"Fetching certificate from Infisical: {secret_path}")
                response = await client.get(url, headers=headers, params=params)

                if response.status_code == 404:
                    raise ExternalServiceError(
                        f"Secret not found in Infisical: {secret_path}"
                    )
                elif response.status_code == 401:
                    raise ExternalServiceError(
                        "Infisical authentication failed. Check INFISICAL_TOKEN."
                    )
                elif response.status_code != 200:
                    raise ExternalServiceError(
                        f"Infisical API error: HTTP {response.status_code} - {response.text}"
                    )

                data = response.json()

                secret_value = data.get("secret", {})
                if isinstance(secret_value, str):
                    try:
                        secret_value = json.loads(secret_value)
                    except json.JSONDecodeError:
                        raise ExternalServiceError(
                            "Invalid secret format in Infisical. "
                            "Expected JSON with cert, key, and optional ca_chain fields."
                        )

                cert_data = secret_value.get("cert") or secret_value.get("certificate")
                key_data = secret_value.get("key") or secret_value.get("private_key")
                ca_chain = secret_value.get("ca_chain") or secret_value.get("ca_certificate")

                if not cert_data:
                    raise ExternalServiceError(
                        "Certificate data not found in Infisical secret. "
                        "Expected 'cert' or 'certificate' field."
                    )

                if not key_data:
                    raise ExternalServiceError(
                        "Private key not found in Infisical secret. "
                        "Expected 'key' or 'private_key' field."
                    )

                logger.info(f"Successfully fetched certificate from Infisical: {secret_path}")
                return cert_data, key_data, ca_chain

        except httpx.TimeoutException as e:
            raise ExternalServiceError(
                f"Timeout connecting to Infisical: {e}"
            )
        except httpx.HTTPError as e:
            raise ExternalServiceError(
                f"HTTP error connecting to Infisical: {e}"
            )
        except Exception as e:
            logger.error(f"Unexpected error fetching from Infisical: {e}", exc_info=True)
            raise ExternalServiceError(
                f"Failed to fetch certificate from Infisical: {e}"
            )

    async def create_certificate_vault(
        self,
        name: str,
        vault_path: str,
        vault_role: str,
        common_name: str,
        description: Optional[str],
        auto_renew: bool,
        renew_before_days: int,
        created_by: int
    ) -> Certificate:
        """
        Create certificate from HashiCorp Vault PKI

        Args:
            name: Certificate name
            vault_path: Vault PKI path
            vault_role: Vault role name
            common_name: Certificate common name
            description: Optional description
            auto_renew: Enable auto-renewal
            renew_before_days: Days before expiry to renew
            created_by: User ID who created it

        Returns:
            Certificate object
        """
        # Fetch from Vault
        cert_data, key_data, ca_chain = await self._fetch_from_vault(
            vault_path, vault_role, common_name
        )

        # Parse metadata
        cert_metadata = self.parse_certificate(cert_data)

        # Create certificate record
        cert = Certificate(
            name=name,
            description=description,
            source_type=CertificateSource.VAULT,
            cert_data=cert_data,
            key_data=key_data,
            ca_chain=ca_chain,
            common_name=cert_metadata["common_name"],
            issuer=cert_metadata["issuer"],
            valid_from=cert_metadata["valid_from"],
            valid_until=cert_metadata["valid_until"],
            subject_alt_names=cert_metadata["subject_alt_names"],
            vault_path=vault_path,
            vault_role=vault_role,
            vault_common_name=common_name,
            auto_renew=auto_renew,
            renew_before_days=renew_before_days,
            created_by=created_by,
            is_active=True
        )

        self.db.add(cert)
        await self.db.commit()
        await self.db.refresh(cert)

        logger.info(f"Certificate created: {name} (Vault)")
        return cert

    async def _fetch_from_vault(
        self,
        vault_path: str,
        vault_role: str,
        common_name: str
    ) -> tuple[str, str, Optional[str]]:
        """
        Fetch certificate from HashiCorp Vault PKI secrets engine

        Issues a new certificate using the PKI secrets engine.

        Args:
            vault_path: PKI secrets engine path (e.g., "pki" or "pki_int")
            vault_role: PKI role name configured in Vault
            common_name: Certificate common name (e.g., "example.com")

        Returns:
            Tuple of (cert_data, key_data, ca_chain)

        Raises:
            ExternalServiceError: If Vault API call fails
        """
        if not settings.VAULT_TOKEN:
            raise ExternalServiceError(
                "Vault integration not configured. "
                "Set VAULT_TOKEN environment variable."
            )

        vault_addr = getattr(settings, 'VAULT_ADDR', 'http://127.0.0.1:8200')

        try:
            async with httpx.AsyncClient(timeout=30.0) as client:
                headers = {
                    "X-Vault-Token": settings.VAULT_TOKEN,
                    "Content-Type": "application/json"
                }

                # Issue certificate using PKI secrets engine
                url = f"{vault_addr}/v1/{vault_path}/issue/{vault_role}"
                payload = {
                    "common_name": common_name,
                    "ttl": "8760h",  # 1 year default
                    "format": "pem"
                }

                logger.info(f"Issuing certificate from Vault PKI: {vault_path}/issue/{vault_role}")
                response = await client.post(url, headers=headers, json=payload)

                if response.status_code == 404:
                    raise ExternalServiceError(
                        f"Vault PKI path or role not found: {vault_path}/issue/{vault_role}"
                    )
                elif response.status_code == 403:
                    raise ExternalServiceError(
                        "Vault authentication failed or insufficient permissions. "
                        "Check VAULT_TOKEN and PKI role policies."
                    )
                elif response.status_code == 400:
                    error_data = response.json()
                    errors = error_data.get("errors", ["Unknown error"])
                    raise ExternalServiceError(
                        f"Vault PKI error: {'; '.join(errors)}"
                    )
                elif response.status_code != 200:
                    raise ExternalServiceError(
                        f"Vault API error: HTTP {response.status_code} - {response.text}"
                    )

                data = response.json()
                vault_data = data.get("data", {})

                cert_data = vault_data.get("certificate")
                key_data = vault_data.get("private_key")
                ca_chain = vault_data.get("ca_chain")

                # ca_chain can be a list - join into single PEM
                if isinstance(ca_chain, list):
                    ca_chain = "\n".join(ca_chain)

                # Vault may also return issuing_ca separately
                issuing_ca = vault_data.get("issuing_ca")
                if issuing_ca and not ca_chain:
                    ca_chain = issuing_ca

                if not cert_data:
                    raise ExternalServiceError(
                        "Certificate not returned by Vault PKI"
                    )

                if not key_data:
                    raise ExternalServiceError(
                        "Private key not returned by Vault PKI"
                    )

                logger.info(f"Successfully issued certificate from Vault: {common_name}")
                return cert_data, key_data, ca_chain

        except httpx.TimeoutException as e:
            raise ExternalServiceError(
                f"Timeout connecting to Vault: {e}"
            )
        except httpx.HTTPError as e:
            raise ExternalServiceError(
                f"HTTP error connecting to Vault: {e}"
            )
        except ExternalServiceError:
            raise
        except Exception as e:
            logger.error(f"Unexpected error fetching from Vault: {e}", exc_info=True)
            raise ExternalServiceError(
                f"Failed to issue certificate from Vault: {e}"
            )

    async def get_expiring_certificates(
        self,
        days_threshold: int = 30
    ) -> list[Certificate]:
        """
        Get certificates expiring within threshold

        Args:
            days_threshold: Days until expiry

        Returns:
            List of expiring certificates
        """
        threshold_date = datetime.utcnow() + timedelta(days=days_threshold)
        stmt = select(Certificate).where(
            Certificate.valid_until <= threshold_date,
            Certificate.is_active == True  # noqa: E712
        )

        certs = (await self.db.execute(stmt)).scalars().all()
        return list(certs)

    async def renew_certificate(self, cert_id: int) -> Certificate:
        """
        Renew a certificate based on its source

        Args:
            cert_id: Certificate ID to renew

        Returns:
            Renewed certificate
        """
        stmt = select(Certificate).where(Certificate.id == cert_id)
        cert = (await self.db.execute(stmt)).scalar_one_or_none()

        if not cert:
            raise ValueError(f"Certificate {cert_id} not found")

        if cert.source_type == CertificateSource.UPLOAD:
            # Cannot auto-renew uploaded certificates
            cert.renewal_error = "Manual upload certificates cannot be auto-renewed"
            await self.db.commit()
            raise CertificateServiceError(
                "Uploaded certificates must be renewed manually"
            )

        try:
            if cert.source_type == CertificateSource.INFISICAL:
                # Re-fetch from Infisical
                cert_data, key_data, ca_chain = await self._fetch_from_infisical(
                    cert.infisical_secret_path,
                    cert.infisical_project_id,
                    cert.infisical_environment
                )
            elif cert.source_type == CertificateSource.VAULT:
                # Re-issue from Vault
                cert_data, key_data, ca_chain = await self._fetch_from_vault(
                    cert.vault_path,
                    cert.vault_role,
                    cert.vault_common_name
                )
            else:
                raise ValueError(f"Unknown source type: {cert.source_type}")

            # Update certificate
            cert_metadata = self.parse_certificate(cert_data)
            cert.cert_data = cert_data
            cert.key_data = key_data
            cert.ca_chain = ca_chain
            cert.common_name = cert_metadata["common_name"]
            cert.issuer = cert_metadata["issuer"]
            cert.valid_from = cert_metadata["valid_from"]
            cert.valid_until = cert_metadata["valid_until"]
            cert.subject_alt_names = cert_metadata["subject_alt_names"]
            cert.last_renewal = datetime.utcnow()
            cert.renewal_error = None

            await self.db.commit()
            await self.db.refresh(cert)

            logger.info(f"Certificate renewed: {cert.name}")
            return cert

        except Exception as e:
            logger.error(f"Failed to renew certificate {cert.name}: {e}")
            cert.renewal_error = str(e)
            await self.db.commit()
            raise
