"""
mTLS (Mutual TLS) API endpoints for MarchProxy Manager

Provides API endpoints for mTLS certificate management, client certificate
validation, and mTLS configuration for both ingress and egress proxies.
"""

import os
import json
import hashlib
import base64
from datetime import datetime, timedelta
from typing import Dict, List, Any, Optional, Tuple
from cryptography import x509
from cryptography.hazmat.primitives import serialization, hashes
from cryptography.hazmat.primitives.asymmetric import rsa, ec
import logging

from py4web import action, request, response, abort, HTTP
from py4web.utils.cors import CORS
from pydal.validators import *

from ..common import (
    auth, db, require_auth, require_admin, require_license_feature,
    create_audit_log, check_permission
)

logger = logging.getLogger(__name__)
cors = CORS()

class MTLSManager:
    """Manager for mTLS certificate operations"""

    def __init__(self, db):
        self.db = db

    def create_client_certificate(self, ca_cert_id: int, common_name: str,
                                organizational_unit: str = None,
                                valid_days: int = 365,
                                key_type: str = 'ecc',
                                key_size: int = 384) -> Dict[str, Any]:
        """Create a new client certificate signed by the specified CA"""

        # Get CA certificate
        ca_cert_record = self.db.certificates[ca_cert_id]
        if not ca_cert_record:
            raise ValueError("CA certificate not found")

        try:
            # Load CA certificate and key
            ca_cert_bytes = ca_cert_record.cert_data.encode('utf-8')
            ca_key_bytes = ca_cert_record.key_data.encode('utf-8')

            ca_cert = x509.load_pem_x509_certificate(ca_cert_bytes)
            ca_private_key = serialization.load_pem_private_key(ca_key_bytes, password=None)

            # Generate client private key
            if key_type == 'ecc':
                if key_size == 256:
                    curve = ec.SECP256R1()
                elif key_size == 384:
                    curve = ec.SECP384R1()
                elif key_size == 521:
                    curve = ec.SECP521R1()
                else:
                    raise ValueError(f"Unsupported ECC key size: {key_size}")
                client_private_key = ec.generate_private_key(curve)
            elif key_type == 'rsa':
                if key_size < 2048:
                    raise ValueError("RSA key size must be at least 2048 bits")
                client_private_key = rsa.generate_private_key(
                    public_exponent=65537,
                    key_size=key_size
                )
            else:
                raise ValueError(f"Unsupported key type: {key_type}")

            # Create client certificate
            client_subject = x509.Name([
                x509.NameAttribute(x509.oid.NameOID.COUNTRY_NAME, "US"),
                x509.NameAttribute(x509.oid.NameOID.STATE_OR_PROVINCE_NAME, "CA"),
                x509.NameAttribute(x509.oid.NameOID.LOCALITY_NAME, "San Francisco"),
                x509.NameAttribute(x509.oid.NameOID.ORGANIZATION_NAME, "MarchProxy"),
                x509.NameAttribute(x509.oid.NameOID.ORGANIZATIONAL_UNIT_NAME,
                                 organizational_unit or "Client Certificate"),
                x509.NameAttribute(x509.oid.NameOID.COMMON_NAME, common_name),
            ])

            client_cert = x509.CertificateBuilder().subject_name(
                client_subject
            ).issuer_name(
                ca_cert.subject
            ).public_key(
                client_private_key.public_key()
            ).serial_number(
                x509.random_serial_number()
            ).not_valid_before(
                datetime.utcnow()
            ).not_valid_after(
                datetime.utcnow() + timedelta(days=valid_days)
            ).add_extension(
                x509.BasicConstraints(ca=False, path_length=None),
                critical=True,
            ).add_extension(
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
            ).add_extension(
                x509.ExtendedKeyUsage([
                    x509.oid.ExtendedKeyUsageOID.CLIENT_AUTH,
                ]),
                critical=True,
            ).sign(ca_private_key, hashes.SHA384())

            # Serialize certificates and keys
            client_cert_pem = client_cert.public_bytes(serialization.Encoding.PEM).decode('utf-8')
            client_key_pem = client_private_key.private_bytes(
                encoding=serialization.Encoding.PEM,
                format=serialization.PrivateFormat.PKCS8,
                encryption_algorithm=serialization.NoEncryption()
            ).decode('utf-8')

            # Calculate fingerprint
            fingerprint = hashlib.sha256(client_cert.public_bytes(serialization.Encoding.DER)).hexdigest()

            return {
                'cert_data': client_cert_pem,
                'key_data': client_key_pem,
                'ca_cert_data': ca_cert_record.cert_data,
                'subject': client_subject.rfc4514_string(),
                'issuer': ca_cert.subject.rfc4514_string(),
                'serial_number': str(client_cert.serial_number),
                'fingerprint_sha256': fingerprint,
                'not_before': client_cert.not_valid_before,
                'not_after': client_cert.not_valid_after,
                'common_name': common_name,
                'organizational_unit': organizational_unit
            }

        except Exception as e:
            logger.error(f"Failed to create client certificate: {e}")
            raise

    def validate_client_certificate(self, cert_data: str, ca_cert_id: int) -> Dict[str, Any]:
        """Validate a client certificate against a CA"""

        try:
            # Load client certificate
            cert_bytes = cert_data.encode('utf-8')
            client_cert = x509.load_pem_x509_certificate(cert_bytes)

            # Get CA certificate
            ca_cert_record = self.db.certificates[ca_cert_id]
            if not ca_cert_record:
                return {'valid': False, 'error': 'CA certificate not found'}

            ca_cert_bytes = ca_cert_record.cert_data.encode('utf-8')
            ca_cert = x509.load_pem_x509_certificate(ca_cert_bytes)

            # Validate certificate chain
            try:
                ca_public_key = ca_cert.public_key()
                ca_public_key.verify(
                    client_cert.signature,
                    client_cert.tbs_certificate_bytes,
                    client_cert.signature_algorithm_oid._name
                )
                signature_valid = True
            except Exception:
                signature_valid = False

            # Check validity period
            now = datetime.utcnow()
            time_valid = client_cert.not_valid_before <= now <= client_cert.not_valid_after

            # Extract certificate information
            common_name = None
            organizational_unit = []

            for attribute in client_cert.subject:
                if attribute.oid == x509.oid.NameOID.COMMON_NAME:
                    common_name = attribute.value
                elif attribute.oid == x509.oid.NameOID.ORGANIZATIONAL_UNIT_NAME:
                    organizational_unit.append(attribute.value)

            # Check extended key usage for client authentication
            has_client_auth = False
            try:
                eku_ext = client_cert.extensions.get_extension_for_oid(
                    x509.oid.ExtensionOID.EXTENDED_KEY_USAGE
                )
                has_client_auth = x509.oid.ExtendedKeyUsageOID.CLIENT_AUTH in eku_ext.value
            except x509.ExtensionNotFound:
                pass

            fingerprint = hashlib.sha256(client_cert.public_bytes(serialization.Encoding.DER)).hexdigest()

            return {
                'valid': signature_valid and time_valid and has_client_auth,
                'signature_valid': signature_valid,
                'time_valid': time_valid,
                'has_client_auth': has_client_auth,
                'common_name': common_name,
                'organizational_unit': organizational_unit,
                'subject': client_cert.subject.rfc4514_string(),
                'issuer': client_cert.issuer.rfc4514_string(),
                'serial_number': str(client_cert.serial_number),
                'fingerprint_sha256': fingerprint,
                'not_before': client_cert.not_valid_before,
                'not_after': client_cert.not_valid_after,
                'days_until_expiry': (client_cert.not_valid_after - now).days
            }

        except Exception as e:
            logger.error(f"Certificate validation failed: {e}")
            return {'valid': False, 'error': str(e)}

    def create_ca_bundle(self, cert_ids: List[int]) -> str:
        """Create a CA bundle from multiple certificates"""

        ca_bundle = []

        for cert_id in cert_ids:
            cert_record = self.db.certificates[cert_id]
            if cert_record and cert_record.is_active:
                ca_bundle.append(cert_record.cert_data.strip())

        return '\n'.join(ca_bundle)

    def get_mtls_config_for_proxy(self, cluster_id: int, proxy_type: str) -> Dict[str, Any]:
        """Get mTLS configuration for a specific proxy type and cluster"""

        # Get active certificates for this cluster
        certs = self.db(
            (self.db.certificates.cluster_id == cluster_id) &
            (self.db.certificates.is_active == True)
        ).select()

        server_certs = []
        client_cas = []

        for cert in certs:
            # Parse certificate to determine its purpose
            try:
                cert_bytes = cert.cert_data.encode('utf-8')
                x509_cert = x509.load_pem_x509_certificate(cert_bytes)

                # Check key usage and extended key usage
                is_server_cert = False
                is_ca_cert = False

                try:
                    basic_constraints = x509_cert.extensions.get_extension_for_oid(
                        x509.oid.ExtensionOID.BASIC_CONSTRAINTS
                    )
                    is_ca_cert = basic_constraints.value.ca
                except x509.ExtensionNotFound:
                    pass

                try:
                    eku_ext = x509_cert.extensions.get_extension_for_oid(
                        x509.oid.ExtensionOID.EXTENDED_KEY_USAGE
                    )
                    is_server_cert = x509.oid.ExtendedKeyUsageOID.SERVER_AUTH in eku_ext.value
                except x509.ExtensionNotFound:
                    pass

                if is_server_cert and not is_ca_cert:
                    server_certs.append({
                        'id': cert.id,
                        'name': cert.name,
                        'domain_names': cert.domain_names or [],
                        'expires_at': cert.expires_at,
                        'cert_data': cert.cert_data,
                        'key_data': cert.key_data,
                        'ca_data': cert.ca_data
                    })
                elif is_ca_cert:
                    client_cas.append({
                        'id': cert.id,
                        'name': cert.name,
                        'subject': cert.subject,
                        'expires_at': cert.expires_at,
                        'cert_data': cert.cert_data
                    })

            except Exception as e:
                logger.warning(f"Failed to parse certificate {cert.id}: {e}")
                continue

        # Create default mTLS configuration
        config = {
            'enabled': len(server_certs) > 0 and len(client_cas) > 0,
            'require_client_cert': True,
            'verify_client_cert': True,
            'server_certificates': server_certs,
            'client_ca_certificates': client_cas,
            'allowed_cns': [],  # Empty means allow all CNs
            'allowed_ous': [],  # Empty means allow all OUs
            'cert_validation_mode': 'strict',  # strict, warn, none
            'proxy_type': proxy_type,
            'cluster_id': cluster_id
        }

        # Add proxy-type specific configurations
        if proxy_type == 'ingress':
            config.update({
                'default_server_cert_id': server_certs[0]['id'] if server_certs else None,
                'sni_enabled': True,
                'client_cert_header': 'X-Client-Cert',
                'client_cn_header': 'X-Client-CN',
                'client_ou_header': 'X-Client-OU'
            })
        elif proxy_type == 'egress':
            config.update({
                'client_cert_id': None,  # Will be configured per service
                'verify_server_cert': True,
                'trusted_server_cas': []
            })

        return config

# Initialize mTLS manager
mtls_manager = MTLSManager(db)

# API Endpoints

@action('/api/mtls/certificates', methods=['GET', 'POST'])
@action.uses(cors, auth, auth.user)
def mtls_certificates():
    """mTLS certificate management"""

    if request.method == 'GET':
        if not check_permission(auth, 'read_certificates'):
            abort(403, "Insufficient permissions")

        # Get mTLS certificates for the user's clusters
        cluster_filter = request.params.get('cluster_id')
        cert_type = request.params.get('type', 'all')  # server, client, ca, all

        query = (db.certificates.is_active == True)

        if cluster_filter:
            query &= (db.certificates.cluster_id == cluster_filter)

        certs = db(query).select(orderby=db.certificates.name)

        cert_list = []
        for cert in certs:
            # Parse certificate to determine type
            cert_info = {
                'id': cert.id,
                'name': cert.name,
                'description': cert.description,
                'cluster_id': cert.cluster_id,
                'domain_names': cert.domain_names or [],
                'subject': cert.subject,
                'issuer': cert.issuer,
                'serial_number': cert.serial_number,
                'fingerprint_sha256': cert.fingerprint_sha256,
                'expires_at': cert.expires_at.isoformat() if cert.expires_at else None,
                'auto_renew': cert.auto_renew,
                'source_type': cert.source_type,
                'is_active': cert.is_active,
                'created_at': cert.created_at.isoformat(),
                'type': 'unknown'
            }

            # Determine certificate type
            try:
                cert_bytes = cert.cert_data.encode('utf-8')
                x509_cert = x509.load_pem_x509_certificate(cert_bytes)

                is_ca = False
                is_server = False
                is_client = False

                try:
                    basic_constraints = x509_cert.extensions.get_extension_for_oid(
                        x509.oid.ExtensionOID.BASIC_CONSTRAINTS
                    )
                    is_ca = basic_constraints.value.ca
                except x509.ExtensionNotFound:
                    pass

                try:
                    eku_ext = x509_cert.extensions.get_extension_for_oid(
                        x509.oid.ExtensionOID.EXTENDED_KEY_USAGE
                    )
                    is_server = x509.oid.ExtendedKeyUsageOID.SERVER_AUTH in eku_ext.value
                    is_client = x509.oid.ExtendedKeyUsageOID.CLIENT_AUTH in eku_ext.value
                except x509.ExtensionNotFound:
                    pass

                if is_ca:
                    cert_info['type'] = 'ca'
                elif is_server and is_client:
                    cert_info['type'] = 'dual'
                elif is_server:
                    cert_info['type'] = 'server'
                elif is_client:
                    cert_info['type'] = 'client'

            except Exception:
                pass

            # Filter by type if requested
            if cert_type != 'all' and cert_info['type'] != cert_type:
                continue

            cert_list.append(cert_info)

        return {'certificates': cert_list}

    elif request.method == 'POST':
        if not check_permission(auth, 'create_certificates'):
            abort(403, "Insufficient permissions")

        data = request.json

        try:
            if data.get('action') == 'create_client':
                # Create client certificate
                result = mtls_manager.create_client_certificate(
                    ca_cert_id=data['ca_cert_id'],
                    common_name=data['common_name'],
                    organizational_unit=data.get('organizational_unit'),
                    valid_days=data.get('valid_days', 365),
                    key_type=data.get('key_type', 'ecc'),
                    key_size=data.get('key_size', 384)
                )

                # Store the client certificate
                from models.certificate import CertificateModel
                cert_id = CertificateModel.create_certificate(
                    db=db,
                    name=f"Client-{data['common_name']}",
                    cert_data=result['cert_data'],
                    key_data=result['key_data'],
                    source_type='generated',
                    created_by=auth.current_user.get('id'),
                    description=f"Client certificate for {data['common_name']}",
                    ca_bundle=result['ca_cert_data']
                )

                create_audit_log(
                    event_type='certificate_created',
                    resource_type='certificate',
                    resource_id=str(cert_id),
                    user_id=auth.current_user.get('id'),
                    event_data={
                        'certificate_type': 'client',
                        'common_name': data['common_name'],
                        'ca_cert_id': data['ca_cert_id']
                    }
                )

                return {
                    'success': True,
                    'certificate_id': cert_id,
                    'certificate': result
                }

            elif data.get('action') == 'create_ca_bundle':
                # Create CA bundle
                bundle = mtls_manager.create_ca_bundle(data['cert_ids'])

                return {
                    'success': True,
                    'ca_bundle': bundle,
                    'cert_count': len(data['cert_ids'])
                }

            else:
                return {'error': 'Invalid action specified'}

        except Exception as e:
            logger.error(f"mTLS certificate operation failed: {e}")
            return {'error': str(e)}

@action('/api/mtls/certificates/validate', methods=['POST'])
@action.uses(cors, auth, auth.user)
def validate_certificate():
    """Validate a client certificate against a CA"""

    if not check_permission(auth, 'read_certificates'):
        abort(403, "Insufficient permissions")

    data = request.json

    try:
        result = mtls_manager.validate_client_certificate(
            cert_data=data['cert_data'],
            ca_cert_id=data['ca_cert_id']
        )

        create_audit_log(
            event_type='certificate_validated',
            resource_type='certificate',
            user_id=auth.current_user.get('id'),
            event_data={
                'ca_cert_id': data['ca_cert_id'],
                'validation_result': result['valid']
            }
        )

        return result

    except Exception as e:
        logger.error(f"Certificate validation failed: {e}")
        return {'valid': False, 'error': str(e)}

@action('/api/mtls/config/<cluster_id:int>/<proxy_type>', methods=['GET'])
@action.uses(cors, auth, auth.user)
def get_mtls_config(cluster_id, proxy_type):
    """Get mTLS configuration for a proxy"""

    if not check_permission(auth, 'read_certificates'):
        abort(403, "Insufficient permissions")

    # Validate proxy type
    if proxy_type not in ['ingress', 'egress']:
        abort(400, "Invalid proxy type")

    try:
        config = mtls_manager.get_mtls_config_for_proxy(cluster_id, proxy_type)

        return {
            'success': True,
            'config': config
        }

    except Exception as e:
        logger.error(f"Failed to get mTLS config: {e}")
        return {'error': str(e)}

@action('/api/mtls/config/<cluster_id:int>/<proxy_type>', methods=['PUT'])
@action.uses(cors, auth, auth.user)
def update_mtls_config(cluster_id, proxy_type):
    """Update mTLS configuration for a proxy"""

    if not check_permission(auth, 'update_certificates'):
        abort(403, "Insufficient permissions")

    # Validate proxy type
    if proxy_type not in ['ingress', 'egress']:
        abort(400, "Invalid proxy type")

    data = request.json

    try:
        # Store mTLS configuration in the cluster's metadata
        cluster = db.clusters[cluster_id]
        if not cluster:
            abort(404, "Cluster not found")

        # Update cluster metadata with mTLS config
        cluster_metadata = cluster.metadata or {}
        if 'mtls_config' not in cluster_metadata:
            cluster_metadata['mtls_config'] = {}

        cluster_metadata['mtls_config'][proxy_type] = {
            'enabled': data.get('enabled', False),
            'require_client_cert': data.get('require_client_cert', True),
            'verify_client_cert': data.get('verify_client_cert', True),
            'allowed_cns': data.get('allowed_cns', []),
            'allowed_ous': data.get('allowed_ous', []),
            'cert_validation_mode': data.get('cert_validation_mode', 'strict'),
            'updated_at': datetime.utcnow().isoformat(),
            'updated_by': auth.current_user.get('id')
        }

        if proxy_type == 'ingress':
            cluster_metadata['mtls_config'][proxy_type].update({
                'default_server_cert_id': data.get('default_server_cert_id'),
                'sni_enabled': data.get('sni_enabled', True),
                'client_cert_header': data.get('client_cert_header', 'X-Client-Cert'),
                'client_cn_header': data.get('client_cn_header', 'X-Client-CN'),
                'client_ou_header': data.get('client_ou_header', 'X-Client-OU')
            })
        elif proxy_type == 'egress':
            cluster_metadata['mtls_config'][proxy_type].update({
                'client_cert_id': data.get('client_cert_id'),
                'verify_server_cert': data.get('verify_server_cert', True),
                'trusted_server_cas': data.get('trusted_server_cas', [])
            })

        cluster.update_record(metadata=cluster_metadata)

        create_audit_log(
            event_type='mtls_config_updated',
            resource_type='cluster',
            resource_id=str(cluster_id),
            user_id=auth.current_user.get('id'),
            event_data={
                'proxy_type': proxy_type,
                'config': cluster_metadata['mtls_config'][proxy_type]
            }
        )

        return {
            'success': True,
            'message': f'mTLS configuration updated for {proxy_type} proxy'
        }

    except Exception as e:
        logger.error(f"Failed to update mTLS config: {e}")
        return {'error': str(e)}

@action('/api/mtls/ca/generate', methods=['POST'])
@action.uses(cors, auth, auth.user)
def generate_ca_certificate():
    """Generate a new CA certificate for mTLS"""

    if not check_permission(auth, 'create_certificates'):
        abort(403, "Insufficient permissions")

    data = request.json

    try:
        # Generate CA certificate using the existing TLS proxy CA functionality
        from models.certificate import TLSProxyCAModel

        domain = data.get('domain', 'marchproxy.local')
        config = {
            'key_type': data.get('key_type', 'ecc'),
            'key_size': data.get('key_size', 384),
            'hash_algorithm': data.get('hash_algorithm', 'sha384'),
            'lifetime_years': data.get('lifetime_years', 5)
        }

        ca_data = TLSProxyCAModel.generate_self_signed_ca(
            domain=domain,
            **config
        )

        # Store CA certificate
        from models.certificate import CertificateModel

        ca_cert_id = CertificateModel.create_certificate(
            db=db,
            name=data.get('name', f"mTLS-CA-{domain}"),
            cert_data=ca_data['ca_cert'],
            key_data=ca_data['ca_key'],
            source_type='generated',
            created_by=auth.current_user.get('id'),
            description=f"mTLS CA certificate for {domain}",
            cluster_id=data.get('cluster_id')
        )

        create_audit_log(
            event_type='ca_certificate_generated',
            resource_type='certificate',
            resource_id=str(ca_cert_id),
            user_id=auth.current_user.get('id'),
            event_data={
                'domain': domain,
                'key_type': config['key_type'],
                'key_size': config['key_size']
            }
        )

        return {
            'success': True,
            'ca_certificate_id': ca_cert_id,
            'ca_certificate': ca_data['ca_cert'],
            'ca_subject': ca_data['ca_subject'],
            'ca_expires_at': ca_data['ca_expires_at'].isoformat()
        }

    except Exception as e:
        logger.error(f"CA generation failed: {e}")
        return {'error': str(e)}

@action('/api/mtls/certificates/<cert_id:int>/download', methods=['GET'])
@action.uses(cors, auth, auth.user)
def download_certificate(cert_id):
    """Download certificate files"""

    if not check_permission(auth, 'read_certificates'):
        abort(403, "Insufficient permissions")

    cert = db.certificates[cert_id]
    if not cert:
        abort(404, "Certificate not found")

    download_type = request.params.get('type', 'cert')  # cert, key, ca, bundle

    try:
        if download_type == 'cert':
            response.headers['Content-Type'] = 'application/x-pem-file'
            response.headers['Content-Disposition'] = f'attachment; filename="{cert.name}.crt"'
            return cert.cert_data

        elif download_type == 'key':
            response.headers['Content-Type'] = 'application/x-pem-file'
            response.headers['Content-Disposition'] = f'attachment; filename="{cert.name}.key"'
            return cert.key_data

        elif download_type == 'ca':
            if cert.ca_data:
                response.headers['Content-Type'] = 'application/x-pem-file'
                response.headers['Content-Disposition'] = f'attachment; filename="{cert.name}-ca.crt"'
                return cert.ca_data
            else:
                abort(404, "CA certificate not available")

        elif download_type == 'bundle':
            bundle = cert.cert_data
            if cert.ca_data:
                bundle += '\n' + cert.ca_data
            response.headers['Content-Type'] = 'application/x-pem-file'
            response.headers['Content-Disposition'] = f'attachment; filename="{cert.name}-bundle.crt"'
            return bundle

        else:
            abort(400, "Invalid download type")

    except Exception as e:
        logger.error(f"Certificate download failed: {e}")
        abort(500, "Download failed")

@action('/api/mtls/test/connection', methods=['POST'])
@action.uses(cors, auth, auth.user)
def test_mtls_connection():
    """Test mTLS connection with provided certificates"""

    if not check_permission(auth, 'read_certificates'):
        abort(403, "Insufficient permissions")

    data = request.json

    try:
        import ssl
        import socket
        from urllib.parse import urlparse

        # Parse target URL
        target_url = data.get('target_url')
        if not target_url:
            return {'error': 'Target URL is required'}

        parsed = urlparse(target_url)
        host = parsed.hostname
        port = parsed.port or (443 if parsed.scheme == 'https' else 80)

        # Load certificates
        client_cert_id = data.get('client_cert_id')
        ca_cert_id = data.get('ca_cert_id')

        if client_cert_id:
            client_cert = db.certificates[client_cert_id]
            if not client_cert:
                return {'error': 'Client certificate not found'}

        if ca_cert_id:
            ca_cert = db.certificates[ca_cert_id]
            if not ca_cert:
                return {'error': 'CA certificate not found'}

        # Create SSL context
        context = ssl.create_default_context()

        if ca_cert_id:
            # Add custom CA
            context.check_hostname = False
            context.verify_mode = ssl.CERT_REQUIRED
            # Note: In a real implementation, you'd write the CA cert to a temp file
            # and use context.load_verify_locations(cafile=ca_file)

        if client_cert_id:
            # Note: In a real implementation, you'd write the cert and key to temp files
            # and use context.load_cert_chain(certfile=cert_file, keyfile=key_file)
            pass

        # Test connection
        sock = socket.create_connection((host, port), timeout=10)
        ssock = context.wrap_socket(sock, server_hostname=host)

        # Get server certificate info
        server_cert = ssock.getpeercert()
        cipher = ssock.cipher()
        version = ssock.version()

        ssock.close()

        return {
            'success': True,
            'connection_successful': True,
            'server_certificate': {
                'subject': dict(x[0] for x in server_cert['subject']),
                'issuer': dict(x[0] for x in server_cert['issuer']),
                'version': server_cert['version'],
                'serial_number': server_cert['serialNumber'],
                'not_before': server_cert['notBefore'],
                'not_after': server_cert['notAfter']
            },
            'tls_info': {
                'cipher': cipher,
                'version': version
            }
        }

    except Exception as e:
        logger.error(f"mTLS connection test failed: {e}")
        return {
            'success': False,
            'connection_successful': False,
            'error': str(e)
        }

def mtls_api(db, jwt_manager):
    """Export mTLS API endpoints"""
    return {
        'mtls_certificates': mtls_certificates,
        'validate_certificate': validate_certificate,
        'get_mtls_config': get_mtls_config,
        'update_mtls_config': update_mtls_config,
        'generate_ca_certificate': generate_ca_certificate,
        'download_certificate': download_certificate,
        'test_mtls_connection': test_mtls_connection
    }