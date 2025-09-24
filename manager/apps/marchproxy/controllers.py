"""
MarchProxy Controllers

Main controller logic for the MarchProxy management system.
Handles web routes, API endpoints, and business logic.
"""

import os
import json
import hashlib
from datetime import datetime, timedelta
from typing import Dict, List, Any

from py4web import action, request, response, redirect, URL, abort, HTTP, Field
from py4web.utils.cors import CORS
from py4web.utils.form import Form, FormStyleBulma
from py4web.utils.publisher import Publisher
from pydal.validators import *

# Import application components
from . import db
from .common import (
    auth, license_manager, cluster_manager,
    require_auth, require_admin, require_license_feature,
    create_audit_log, COMMUNITY_MAX_PROXIES
)

# Enable CORS for API endpoints
cors = CORS(origin="*", headers="*", methods="*")

#
# Core Application Routes
#

@action('index')
@action.uses('index.html')  
def index():
    """Main landing page"""
    user = auth.get_current_user()
    if user:
        redirect(URL('dashboard'))
    
    return dict(
        title="MarchProxy Management",
        user=None
    )

@action('healthz')
def healthz():
    """Health check endpoint"""
    try:
        # Check database connectivity
        db.executesql("SELECT 1")
        
        # Check license server connectivity (if Enterprise)
        license_key = os.environ.get('LICENSE_KEY')
        license_status = "community"
        if license_key:
            try:
                validation = license_manager.validate_license(license_key)
                license_status = "valid" if validation.get('valid') else "invalid"
            except:
                license_status = "error"
        
        return dict(
            status="healthy",
            timestamp=datetime.utcnow().isoformat(),
            version="1.0.0",
            database="connected",
            license=license_status
        )
    except Exception as e:
        response.status = 500
        return dict(
            status="unhealthy", 
            error=str(e),
            timestamp=datetime.utcnow().isoformat()
        )

@action('metrics')
def metrics():
    """Prometheus metrics endpoint"""
    from prometheus_client import generate_latest, CONTENT_TYPE_LATEST
    
    # This would include various metrics
    # For now, return basic structure
    response.headers['Content-Type'] = CONTENT_TYPE_LATEST
    return generate_latest()

#
# Authentication Routes
#

@action('auth/login', method=['GET', 'POST'])
@action.uses('auth/login.html')
def login():
    """User login"""
    form = Form([
        Field('username', requires=IS_NOT_EMPTY()),
        Field('password', 'password', requires=IS_NOT_EMPTY()),
        Field('totp_token', length=6, label='2FA Code (if enabled)')
    ], submit_value='Login')
    
    if form.accepted:
        # Authenticate user
        result = auth.authenticate_user(
            form.vars.username, 
            form.vars.password, 
            form.vars.totp_token
        )
        
        if result:
            if result.get('error') == '2fa_required':
                form.errors['totp_token'] = 'Invalid or missing 2FA code'
                return dict(form=form)
            
            # Successful login - set session
            auth.set_user_session(result)
            create_audit_log('login', 'user', str(result['id']))
            redirect(URL('dashboard'))
        else:
            form.errors['password'] = 'Invalid username or password'
            create_audit_log('login_failed', 'user', form.vars.username, success=False)
    
    return dict(form=form)

@action('auth/logout')
def logout():
    """User logout"""
    user_id = auth.get_current_user_id()
    if user_id:
        create_audit_log('logout', 'user', str(user_id))
    
    auth.clear_user_session()
    redirect(URL('index'))

@action('auth/setup-2fa', method=['GET', 'POST'])
@action.uses('auth/setup_2fa.html')
@require_auth
def setup_2fa():
    """Setup 2FA for current user"""
    user_id = auth.get_current_user_id()
    user = db.auth_user[user_id]
    
    if request.method == 'POST':
        token = request.json.get('token')
        if auth.enable_2fa(user_id, token):
            create_audit_log('2fa_enabled', 'user', str(user_id))
            return dict(success=True, message='2FA enabled successfully')
        else:
            return dict(success=False, message='Invalid token')
    else:
        secret, qr_code = auth.setup_2fa(user_id)
        return dict(secret=secret, qr_code=qr_code)

#
# Dashboard and Main UI
#

@action('dashboard')
@action.uses('dashboard.html')
@require_auth
def dashboard():
    """Main dashboard"""
    user_id = auth.get_current_user_id()
    user = db.auth_user[user_id]
    
    # Get cluster information
    if user.is_admin:
        clusters = db(db.clusters.is_active == True).select()
    else:
        # Service owners see only their assigned clusters
        clusters = db(
            (db.user_cluster_assignments.user_id == user_id) & 
            (db.user_cluster_assignments.cluster_id == db.clusters.id) &
            (db.clusters.is_active == True)
        ).select(db.clusters.ALL)
    
    # Get proxy server counts per cluster
    proxy_stats = {}
    for cluster in clusters:
        proxy_count = db(
            (db.proxy_servers.cluster_id == cluster.id) &
            (db.proxy_servers.status == 'active')
        ).count()
        proxy_stats[cluster.id] = {
            'active': proxy_count,
            'max': cluster.max_proxies,
            'utilization': (proxy_count / cluster.max_proxies * 100) if cluster.max_proxies > 0 else 0
        }
    
    # License information
    license_info = {'edition': 'Community', 'valid': True}
    license_key = os.environ.get('LICENSE_KEY')
    if license_key:
        try:
            validation = license_manager.validate_license(license_key)
            license_info = {
                'edition': 'Enterprise',
                'valid': validation.get('valid', False),
                'expires_at': validation.get('expires_at'),
                'features': validation.get('features', [])
            }
        except:
            license_info['valid'] = False
    
    return dict(
        user=user,
        clusters=clusters,
        proxy_stats=proxy_stats,
        license_info=license_info
    )

#
# Cluster Management (Enterprise)
#

@action('clusters')
@action.uses('clusters/index.html')
@require_auth
def clusters():
    """Cluster management"""
    user_id = auth.get_current_user_id()
    user = db.auth_user[user_id]
    
    if user.is_admin:
        clusters = db(db.clusters.is_active == True).select()
    else:
        # Service owners see only assigned clusters  
        clusters = db(
            (db.user_cluster_assignments.user_id == user_id) &
            (db.user_cluster_assignments.cluster_id == db.clusters.id) &
            (db.clusters.is_active == True)
        ).select(db.clusters.ALL)
    
    return dict(clusters=clusters, user=user)

@action('clusters/create', method=['GET', 'POST'])
@action.uses('clusters/create.html')
@require_admin
@require_license_feature('multi_cluster')
def create_cluster():
    """Create new cluster (Enterprise only)"""
    form = Form([
        Field('name', requires=IS_NOT_EMPTY()),
        Field('description'),
        Field('syslog_endpoint', label='Syslog Endpoint (host:port)'),
        Field('log_auth', 'boolean', default=True, label='Log Authentication Events'),
        Field('log_netflow', 'boolean', default=True, label='Log Network Flow'),
        Field('log_debug', 'boolean', default=False, label='Debug Logging')
    ])
    
    if form.accepted:
        try:
            cluster_id = cluster_manager.create_cluster(
                form.vars.name,
                form.vars.description,
                auth.get_current_user_id()
            )
            
            # Update cluster with logging configuration
            db(db.clusters.id == cluster_id).update(
                syslog_endpoint=form.vars.syslog_endpoint,
                log_auth=form.vars.log_auth,
                log_netflow=form.vars.log_netflow,
                log_debug=form.vars.log_debug
            )
            
            create_audit_log('cluster_created', 'cluster', str(cluster_id), {
                'name': form.vars.name,
                'description': form.vars.description
            })
            
            redirect(URL('clusters'))
        except Exception as e:
            form.errors['name'] = str(e)
    
    return dict(form=form)

@action('clusters/<cluster_id>/rotate-key', method='POST')
@require_admin
def rotate_cluster_key(cluster_id):
    """Rotate cluster API key"""
    try:
        new_key = auth.generate_cluster_api_key(int(cluster_id))
        create_audit_log('cluster_key_rotated', 'cluster', cluster_id)
        return dict(success=True, new_key=new_key)
    except Exception as e:
        response.status = 500
        return dict(success=False, error=str(e))

#
# Service Management
#

@action('services')
@action.uses('services/index.html')
@require_auth
def services():
    """Service management"""
    user_id = auth.get_current_user_id()
    user = db.auth_user[user_id]
    
    if user.is_admin:
        # Admins see all services
        services = db(db.services.is_active == True).select(orderby=db.services.name)
    else:
        # Service owners see only assigned services
        services = db(
            (db.user_service_assignments.user_id == user_id) &
            (db.user_service_assignments.service_id == db.services.id) &
            (db.services.is_active == True)
        ).select(db.services.ALL, orderby=db.services.name)
    
    return dict(services=services, user=user)

@action('services/create', method=['GET', 'POST'])
@action.uses('services/create.html')
@require_auth
def create_service():
    """Create new service"""
    user_id = auth.get_current_user_id()
    user = db.auth_user[user_id]
    
    # Get available clusters for user
    if user.is_admin:
        clusters = db(db.clusters.is_active == True).select()
    else:
        clusters = db(
            (db.user_cluster_assignments.user_id == user_id) &
            (db.user_cluster_assignments.cluster_id == db.clusters.id) &
            (db.clusters.is_active == True)
        ).select(db.clusters.ALL)
    
    cluster_options = [(c.id, c.name) for c in clusters]
    
    form = Form([
        Field('name', requires=IS_NOT_EMPTY()),
        Field('ip_fqdn', label='IP or FQDN', requires=IS_NOT_EMPTY()),
        Field('collection', label='Collection/Group'),
        Field('cluster_id', requires=IS_IN_SET(cluster_options), label='Cluster'),
        Field('description'),
        Field('auth_type', requires=IS_IN_SET(['none', 'base64', 'jwt']), default='none'),
        Field('tags', label='Tags (comma-separated)')
    ])
    
    if form.accepted:
        try:
            # Generate authentication credentials based on type
            token_base64 = None
            jwt_secret = None
            jwt_expiry = None
            
            if form.vars.auth_type == 'base64':
                import base64
                import secrets
                # Generate a secure Base64 token
                token_bytes = secrets.token_bytes(32)
                token_base64 = base64.b64encode(token_bytes).decode('ascii')
            elif form.vars.auth_type == 'jwt':
                # Generate JWT secret
                jwt_secret = secrets.token_urlsafe(32)
                jwt_expiry = 3600  # 1 hour default
            
            # Create service
            service_id = db.services.insert(
                name=form.vars.name,
                ip_fqdn=form.vars.ip_fqdn,
                collection=form.vars.collection,
                cluster_id=form.vars.cluster_id,
                description=form.vars.description,
                auth_type=form.vars.auth_type,
                token_base64=token_base64,
                jwt_secret=jwt_secret,
                jwt_expiry=jwt_expiry,
                tags=form.vars.tags.split(',') if form.vars.tags else [],
                created_by=user_id,
                service_owner=user_id
            )
            
            # Assign service to creator (if not admin)
            if not user.is_admin:
                db.user_service_assignments.insert(
                    user_id=user_id,
                    service_id=service_id,
                    role='owner',
                    assigned_by=user_id
                )
            
            create_audit_log('service_created', 'service', str(service_id), {
                'name': form.vars.name,
                'ip_fqdn': form.vars.ip_fqdn,
                'cluster_id': form.vars.cluster_id
            })
            
            redirect(URL('services'))
        except Exception as e:
            form.errors['name'] = str(e)
    
    return dict(form=form, clusters=clusters)

@action('services/<service_id>/rotate-auth', method='POST')
@require_auth
def rotate_service_auth(service_id):
    """Rotate service authentication credentials"""
    try:
        user_id = auth.get_current_user_id()
        user = db.auth_user[user_id]
        
        # Check service access
        service = db.services[service_id]
        if not service:
            abort(404, "Service not found")
        
        # Check permissions
        if not user.is_admin:
            # Check if user is assigned to this service
            assignment = db(
                (db.user_service_assignments.user_id == user_id) &
                (db.user_service_assignments.service_id == service_id)
            ).select().first()
            
            if not assignment:
                abort(403, "Access denied")
        
        # Generate new credentials based on auth type
        update_data = {}
        
        if service.auth_type == 'base64':
            import base64
            import secrets
            token_bytes = secrets.token_bytes(32)
            update_data['token_base64'] = base64.b64encode(token_bytes).decode('ascii')
            
        elif service.auth_type == 'jwt':
            import secrets
            update_data['jwt_secret'] = secrets.token_urlsafe(32)
            # Keep the same expiry time or reset to default
            if not service.jwt_expiry:
                update_data['jwt_expiry'] = 3600
        
        # Update service with new credentials
        db(db.services.id == service_id).update(**update_data)
        
        create_audit_log('service_auth_rotated', 'service', str(service_id), {
            'service_name': service.name,
            'auth_type': service.auth_type
        })
        
        return dict(success=True, message="Authentication credentials rotated successfully")
        
    except Exception as e:
        response.status = 500
        return dict(success=False, error=str(e))

#
# Certificate Management
#

@action('certificates')
@action.uses('certificates/index.html')
@require_auth
def certificates():
    """Certificate management"""
    user_id = auth.get_current_user_id()
    user = db.auth_user[user_id]
    
    if user.is_admin:
        certificates = db(db.certificates.is_active == True).select(orderby=db.certificates.name)
    else:
        # Service owners see certificates for their clusters
        certificates = db(
            (db.user_cluster_assignments.user_id == user_id) &
            (db.user_cluster_assignments.cluster_id == db.certificates.cluster_id) &
            (db.certificates.is_active == True)
        ).select(db.certificates.ALL, orderby=db.certificates.name)
    
    return dict(certificates=certificates, user=user)

@action('certificates/upload', method=['GET', 'POST'])
@action.uses('certificates/upload.html')
@require_auth
def upload_certificate():
    """Upload certificate manually"""
    user_id = auth.get_current_user_id()
    user = db.auth_user[user_id]
    
    # Get available clusters for user
    if user.is_admin:
        clusters = db(db.clusters.is_active == True).select()
    else:
        clusters = db(
            (db.user_cluster_assignments.user_id == user_id) &
            (db.user_cluster_assignments.cluster_id == db.clusters.id) &
            (db.clusters.is_active == True)
        ).select(db.clusters.ALL)
    
    cluster_options = [(c.id, c.name) for c in clusters]
    
    form = Form([
        Field('name', requires=IS_NOT_EMPTY()),
        Field('cluster_id', requires=IS_IN_SET(cluster_options), label='Cluster'),
        Field('cert_data', 'text', requires=IS_NOT_EMPTY(), label='Certificate (PEM format)'),
        Field('key_data', 'text', requires=IS_NOT_EMPTY(), label='Private Key (PEM format)'),
        Field('description', 'text', label='Description'),
        Field('auto_renew', 'boolean', default=False, label='Auto-renew (if supported)'),
    ])
    
    if form.accepted:
        try:
            # Parse certificate to extract information
            cert_info = parse_certificate(form.vars.cert_data)
            
            certificate_id = db.certificates.insert(
                name=form.vars.name,
                cluster_id=form.vars.cluster_id,
                cert_data=form.vars.cert_data,
                key_data=form.vars.key_data,
                description=form.vars.description,
                source_type='upload',
                auto_renew=form.vars.auto_renew,
                subject=cert_info.get('subject', ''),
                issuer=cert_info.get('issuer', ''),
                not_before=cert_info.get('not_before'),
                not_after=cert_info.get('not_after'),
                san_domains=cert_info.get('san_domains', []),
                created_by=user_id
            )
            
            create_audit_log('certificate_uploaded', 'certificate', str(certificate_id), {
                'name': form.vars.name,
                'cluster_id': form.vars.cluster_id,
                'subject': cert_info.get('subject', '')
            })
            
            redirect(URL('certificates'))
        except Exception as e:
            form.errors['cert_data'] = str(e)
    
    return dict(form=form, clusters=clusters)

def parse_certificate(cert_data: str) -> dict:
    """Parse certificate to extract information"""
    try:
        from cryptography import x509
        from cryptography.hazmat.backends import default_backend
        
        # Load certificate
        cert = x509.load_pem_x509_certificate(cert_data.encode(), default_backend())
        
        # Extract information
        cert_info = {
            'subject': cert.subject.rfc4514_string(),
            'issuer': cert.issuer.rfc4514_string(),
            'not_before': cert.not_valid_before,
            'not_after': cert.not_valid_after,
            'san_domains': []
        }
        
        # Extract SAN domains
        try:
            san_ext = cert.extensions.get_extension_for_oid(x509.oid.ExtensionOID.SUBJECT_ALTERNATIVE_NAME)
            cert_info['san_domains'] = [name.value for name in san_ext.value]
        except x509.ExtensionNotFound:
            pass
        
        return cert_info
        
    except Exception as e:
        raise ValueError(f"Invalid certificate format: {e}")

@action('certificates/<cert_id>/delete', method='POST')
@require_auth
def delete_certificate(cert_id):
    """Delete certificate"""
    try:
        user_id = auth.get_current_user_id()
        user = db.auth_user[user_id]
        
        cert = db.certificates[cert_id]
        if not cert:
            abort(404, "Certificate not found")
        
        # Check permissions
        if not user.is_admin:
            # Check if user has access to the cluster
            assignment = db(
                (db.user_cluster_assignments.user_id == user_id) &
                (db.user_cluster_assignments.cluster_id == cert.cluster_id)
            ).select().first()
            
            if not assignment:
                abort(403, "Access denied")
        
        # Mark as inactive instead of deleting
        db(db.certificates.id == cert_id).update(is_active=False)
        
        create_audit_log('certificate_deleted', 'certificate', str(cert_id), {
            'name': cert.name,
            'cluster_id': cert.cluster_id
        })
        
        return dict(success=True, message="Certificate deleted successfully")
        
    except Exception as e:
        response.status = 500
        return dict(success=False, error=str(e))

#
# Mapping Configuration System
#

@action('mappings')
@action.uses('mappings/index.html')
@require_auth
def mappings():
    """Mapping management"""
    user_id = auth.get_current_user_id()
    user = db.auth_user[user_id]
    
    if user.is_admin:
        mappings = db(db.mappings.is_active == True).select(
            orderby=db.mappings.priority | db.mappings.name
        )
    else:
        # Service owners see mappings for their clusters
        mappings = db(
            (db.user_cluster_assignments.user_id == user_id) &
            (db.user_cluster_assignments.cluster_id == db.mappings.cluster_id) &
            (db.mappings.is_active == True)
        ).select(db.mappings.ALL, orderby=db.mappings.priority | db.mappings.name)
    
    return dict(mappings=mappings, user=user)

@action('mappings/create', method=['GET', 'POST'])
@action.uses('mappings/create.html')
@require_auth
def create_mapping():
    """Create new mapping"""
    user_id = auth.get_current_user_id()
    user = db.auth_user[user_id]
    
    # Get available clusters for user
    if user.is_admin:
        clusters = db(db.clusters.is_active == True).select()
    else:
        clusters = db(
            (db.user_cluster_assignments.user_id == user_id) &
            (db.user_cluster_assignments.cluster_id == db.clusters.id) &
            (db.clusters.is_active == True)
        ).select(db.clusters.ALL)
    
    cluster_options = [(c.id, c.name) for c in clusters]
    
    # Get services for cluster selection (will be filtered by JavaScript)
    if user.is_admin:
        services = db(db.services.is_active == True).select()
    else:
        services = db(
            (db.user_service_assignments.user_id == user_id) &
            (db.user_service_assignments.service_id == db.services.id) &
            (db.services.is_active == True)
        ).select(db.services.ALL)
    
    form = Form([
        Field('name', requires=IS_NOT_EMPTY()),
        Field('cluster_id', requires=IS_IN_SET(cluster_options), label='Cluster'),
        Field('source_services', 'list:integer', label='Source Services', 
              comment='Services that can initiate connections'),
        Field('dest_services', 'list:integer', label='Destination Services',
              comment='Services that can receive connections'),
        Field('protocols', 'list:string', default=['TCP'], label='Protocols'),
        Field('ports', label='Ports (e.g., "80", "443", "8000-8999")', requires=IS_NOT_EMPTY()),
        Field('auth_required', 'boolean', default=True, label='Require Authentication'),
        Field('priority', 'integer', default=100, label='Priority (lower = higher priority)'),
        Field('timeout', 'integer', default=30, label='Connection Timeout (seconds)'),
        Field('description', 'text', label='Description'),
        Field('comments', 'text', label='Comments')
    ])
    
    if form.accepted:
        try:
            # Validate source and destination services are in the same cluster
            cluster_id = form.vars.cluster_id
            
            source_services = form.vars.source_services or []
            dest_services = form.vars.dest_services or []
            
            # Check if services belong to selected cluster
            for service_id in source_services + dest_services:
                service = db.services[service_id]
                if not service or service.cluster_id != cluster_id:
                    form.errors['source_services'] = "All services must belong to the selected cluster"
                    break
            
            if not form.errors:
                mapping_id = db.mappings.insert(
                    name=form.vars.name,
                    cluster_id=cluster_id,
                    source_services=source_services,
                    dest_services=dest_services,
                    protocols=form.vars.protocols,
                    ports=form.vars.ports,
                    auth_required=form.vars.auth_required,
                    priority=form.vars.priority,
                    timeout=form.vars.timeout,
                    description=form.vars.description,
                    comments=form.vars.comments,
                    created_by=user_id,
                    approval_status='approved' if user.is_admin else 'pending'
                )
                
                create_audit_log('mapping_created', 'mapping', str(mapping_id), {
                    'name': form.vars.name,
                    'cluster_id': cluster_id,
                    'source_services': len(source_services),
                    'dest_services': len(dest_services)
                })
                
                redirect(URL('mappings'))
                
        except Exception as e:
            form.errors['name'] = str(e)
    
    return dict(form=form, clusters=clusters, services=services)

@action('mappings/<mapping_id>/approve', method='POST')
@require_admin
def approve_mapping(mapping_id):
    """Approve mapping (admin only)"""
    try:
        mapping = db.mappings[mapping_id]
        if not mapping:
            abort(404, "Mapping not found")
        
        db(db.mappings.id == mapping_id).update(
            approval_status='approved',
            approved_by=auth.get_current_user_id(),
            approved_at=datetime.utcnow()
        )
        
        create_audit_log('mapping_approved', 'mapping', str(mapping_id), {
            'name': mapping.name,
            'cluster_id': mapping.cluster_id
        })
        
        return dict(success=True, message="Mapping approved successfully")
        
    except Exception as e:
        response.status = 500
        return dict(success=False, error=str(e))

@action('mappings/<mapping_id>/delete', method='POST')
@require_auth
def delete_mapping(mapping_id):
    """Delete mapping"""
    try:
        user_id = auth.get_current_user_id()
        user = db.auth_user[user_id]
        
        mapping = db.mappings[mapping_id]
        if not mapping:
            abort(404, "Mapping not found")
        
        # Check permissions
        if not user.is_admin:
            # Check if user has access to the cluster
            assignment = db(
                (db.user_cluster_assignments.user_id == user_id) &
                (db.user_cluster_assignments.cluster_id == mapping.cluster_id)
            ).select().first()
            
            if not assignment:
                abort(403, "Access denied")
        
        # Mark as inactive instead of deleting
        db(db.mappings.id == mapping_id).update(is_active=False)
        
        create_audit_log('mapping_deleted', 'mapping', str(mapping_id), {
            'name': mapping.name,
            'cluster_id': mapping.cluster_id
        })
        
        return dict(success=True, message="Mapping deleted successfully")
        
    except Exception as e:
        response.status = 500
        return dict(success=False, error=str(e))

#
# License Management (Enterprise)
#

@action('license')
@action.uses('license/index.html')
@require_admin
def license_management():
    """License management interface"""
    license_key = os.environ.get('LICENSE_KEY')
    license_info = {'edition': 'Community', 'valid': True}
    
    if license_key:
        try:
            validation = license_manager.validate_license(license_key)
            license_info = {
                'edition': 'Enterprise',
                'valid': validation.get('valid', False),
                'expires_at': validation.get('expires_at'),
                'features': validation.get('features', []),
                'limits': validation.get('limits', {}),
                'license_key': license_key[:8] + '...' + license_key[-8:] if len(license_key) > 16 else license_key
            }
        except Exception as e:
            license_info = {
                'edition': 'Enterprise',
                'valid': False,
                'error': str(e),
                'license_key': license_key[:8] + '...' + license_key[-8:] if len(license_key) > 16 else license_key
            }
    
    # Get current proxy counts per cluster
    proxy_counts = {}
    clusters = db(db.clusters.is_active == True).select()
    for cluster in clusters:
        proxy_count = db(
            (db.proxy_servers.cluster_id == cluster.id) &
            (db.proxy_servers.status == 'active')
        ).count()
        proxy_counts[cluster.id] = {
            'name': cluster.name,
            'count': proxy_count,
            'max': cluster.max_proxies
        }
    
    return dict(license_info=license_info, proxy_counts=proxy_counts)

@action('api/license/validate', method='POST')
@require_admin
def validate_license_api():
    """Validate license via API"""
    try:
        data = request.json
        license_key = data.get('license_key')
        
        if not license_key:
            abort(400, "License key required")
        
        # Validate with license server
        validation = license_manager.validate_license(license_key)
        
        if validation.get('valid'):
            return dict(
                success=True,
                valid=True,
                features=validation.get('features', []),
                limits=validation.get('limits', {}),
                expires_at=validation.get('expires_at'),
                message="License is valid"
            )
        else:
            return dict(
                success=True,
                valid=False,
                error=validation.get('error', 'Invalid license'),
                message="License validation failed"
            )
            
    except Exception as e:
        response.status = 500
        return dict(success=False, error=str(e))

@action('api/license-status')
@cors
def api_license_status():
    """Get license status for proxy servers"""
    try:
        # Validate API key
        api_key = request.headers.get('X-API-Key')
        if not api_key:
            abort(401, "API key required")
        
        api_key_hash = hashlib.sha256(api_key.encode()).hexdigest()
        cluster = db(db.clusters.api_key == api_key_hash).select().first()
        
        if not cluster:
            abort(401, "Invalid API key")
        
        # Get license information
        license_key = os.environ.get('LICENSE_KEY')
        license_status = {
            'edition': 'Community',
            'valid': True,
            'proxy_limit': COMMUNITY_MAX_PROXIES,
            'features': []
        }
        
        if license_key:
            try:
                validation = license_manager.validate_license(license_key)
                license_status = {
                    'edition': 'Enterprise',
                    'valid': validation.get('valid', False),
                    'proxy_limit': validation.get('limits', {}).get('proxies', COMMUNITY_MAX_PROXIES),
                    'features': validation.get('features', []),
                    'expires_at': validation.get('expires_at')
                }
            except Exception as e:
                license_status['error'] = str(e)
                license_status['valid'] = False
        
        # Get current proxy count for this cluster
        current_proxies = db(
            (db.proxy_servers.cluster_id == cluster.id) &
            (db.proxy_servers.status.belongs(['active', 'pending']))
        ).count()
        
        license_status.update({
            'cluster_id': cluster.id,
            'cluster_name': cluster.name,
            'current_proxies': current_proxies,
            'max_proxies': cluster.max_proxies,
            'can_register': current_proxies < cluster.max_proxies
        })
        
        return license_status
        
    except Exception as e:
        response.status = 500
        return dict(error=str(e))

#
# API Endpoints for Proxy Servers
#

@action('api/proxy/register', method='POST')
@cors
def api_proxy_register():
    """Proxy registration endpoint"""
    try:
        data = request.json
        
        # Validate API key
        api_key = request.headers.get('X-API-Key') or data.get('api_key')
        if not api_key:
            abort(401, "API key required")
        
        # Find cluster by API key hash
        api_key_hash = hashlib.sha256(api_key.encode()).hexdigest()
        cluster = db(db.clusters.api_key == api_key_hash).select().first()
        
        if not cluster:
            abort(401, "Invalid API key")
        
        # Check proxy limit
        current_proxies = db(
            (db.proxy_servers.cluster_id == cluster.id) &
            (db.proxy_servers.status.belongs(['active', 'pending']))
        ).count()
        
        if current_proxies >= cluster.max_proxies:
            abort(429, f"Proxy limit exceeded ({cluster.max_proxies})")
        
        # Register or update proxy
        proxy_name = data.get('name')
        hostname = data.get('hostname')
        
        if not proxy_name or not hostname:
            abort(400, "Name and hostname required")
        
        # Check for existing proxy
        existing = db(
            (db.proxy_servers.name == proxy_name) &
            (db.proxy_servers.cluster_id == cluster.id)
        ).select().first()
        
        if existing:
            # Update existing proxy
            db(db.proxy_servers.id == existing.id).update(
                hostname=hostname,
                status='active',
                last_seen=datetime.utcnow(),
                version=data.get('version'),
                capabilities=data.get('capabilities', [])
            )
            proxy_id = existing.id
        else:
            # Create new proxy
            proxy_id = db.proxy_servers.insert(
                name=proxy_name,
                hostname=hostname,
                cluster_id=cluster.id,
                status='active',
                last_seen=datetime.utcnow(),
                version=data.get('version'),
                capabilities=data.get('capabilities', [])
            )
        
        # Update cluster proxy count
        proxy_count = db(
            (db.proxy_servers.cluster_id == cluster.id) &
            (db.proxy_servers.status == 'active')
        ).count()
        
        db(db.clusters.id == cluster.id).update(proxy_count=proxy_count)
        
        create_audit_log('proxy_registered', 'proxy', str(proxy_id), {
            'name': proxy_name,
            'hostname': hostname,
            'cluster_id': cluster.id
        })
        
        return dict(
            success=True,
            proxy_id=proxy_id,
            cluster_name=cluster.name,
            message="Proxy registered successfully"
        )
        
    except Exception as e:
        response.status = 500
        return dict(success=False, error=str(e))

@action('api/config/<cluster_id>')
@cors
def api_get_config(cluster_id):
    """Get cluster configuration for proxy"""
    try:
        # Validate API key
        api_key = request.headers.get('X-API-Key')
        if not api_key:
            abort(401, "API key required")
        
        # Validate cluster and API key
        api_key_hash = hashlib.sha256(api_key.encode()).hexdigest()
        cluster = db(
            (db.clusters.id == cluster_id) &
            (db.clusters.api_key == api_key_hash) &
            (db.clusters.is_active == True)
        ).select().first()
        
        if not cluster:
            abort(401, "Invalid API key or cluster")
        
        # Get services for this cluster
        services = db(
            (db.services.cluster_id == cluster_id) &
            (db.services.is_active == True)
        ).select()
        
        # Get active mappings for this cluster
        mappings = db(
            (db.mappings.cluster_id == cluster_id) &
            (db.mappings.is_active == True) &
            (db.mappings.approval_status == 'approved')
        ).select()
        
        # Get certificates for this cluster
        certificates = db(
            (db.certificates.cluster_id == cluster_id) &
            (db.certificates.is_active == True)
        ).select()
        
        # Build configuration
        config = {
            'cluster': {
                'id': cluster.id,
                'name': cluster.name,
                'description': cluster.description
            },
            'logging': {
                'syslog_endpoint': cluster.syslog_endpoint,
                'log_auth': cluster.log_auth,
                'log_netflow': cluster.log_netflow,
                'log_debug': cluster.log_debug
            },
            'services': [],
            'mappings': [],
            'certificates': []
        }
        
        # Add services
        for service in services:
            service_config = {
                'id': service.id,
                'name': service.name,
                'ip_fqdn': service.ip_fqdn,
                'collection': service.collection,
                'auth_type': service.auth_type
            }
            
            # Add authentication details based on type
            if service.auth_type == 'base64' and service.token_base64:
                service_config['auth_token'] = service.token_base64
            elif service.auth_type == 'jwt' and service.jwt_secret:
                service_config['jwt_secret'] = service.jwt_secret
                service_config['jwt_expiry'] = service.jwt_expiry
            
            config['services'].append(service_config)
        
        # Add mappings
        for mapping in mappings:
            config['mappings'].append({
                'id': mapping.id,
                'name': mapping.name,
                'source_services': mapping.source_services,
                'dest_services': mapping.dest_services,
                'protocols': mapping.protocols,
                'ports': mapping.ports,
                'auth_required': mapping.auth_required,
                'auth_type': mapping.auth_type,
                'priority': mapping.priority,
                'timeout': mapping.timeout
            })
        
        # Add certificates
        for cert in certificates:
            config['certificates'].append({
                'id': cert.id,
                'name': cert.name,
                'subject': cert.subject,
                'not_after': cert.not_after.isoformat() if cert.not_after else None,
                'san_domains': cert.san_domains
                # Note: cert_data and key_data would be provided via separate secure endpoint
            })
        
        # Generate config hash for version tracking
        config_json = json.dumps(config, sort_keys=True)
        config_hash = hashlib.sha256(config_json.encode()).hexdigest()
        
        config['version'] = config_hash
        config['generated_at'] = datetime.utcnow().isoformat()
        
        # Cache configuration
        db.config_cache.update_or_insert(
            (db.config_cache.cluster_id == cluster_id),
            cluster_id=cluster_id,
            config_hash=config_hash,
            config_data=config,
            generated_at=datetime.utcnow(),
            expires_at=datetime.utcnow() + timedelta(minutes=30)
        )
        
        return config
        
    except Exception as e:
        response.status = 500
        return dict(error=str(e))

@action('api/proxy/heartbeat', method='POST')
@cors
def api_proxy_heartbeat():
    """Proxy heartbeat/status update"""
    try:
        data = request.json
        
        # Validate API key
        api_key = request.headers.get('X-API-Key')
        if not api_key:
            abort(401, "API key required")
        
        api_key_hash = hashlib.sha256(api_key.encode()).hexdigest()
        cluster = db(db.clusters.api_key == api_key_hash).select().first()
        
        if not cluster:
            abort(401, "Invalid API key")
        
        proxy_name = data.get('name')
        if not proxy_name:
            abort(400, "Proxy name required")
        
        # Update proxy status
        proxy = db(
            (db.proxy_servers.name == proxy_name) &
            (db.proxy_servers.cluster_id == cluster.id)
        ).select().first()
        
        if proxy:
            db(db.proxy_servers.id == proxy.id).update(
                last_seen=datetime.utcnow(),
                status='active',
                cpu_usage=data.get('cpu_usage'),
                memory_usage=data.get('memory_usage'),
                connection_count=data.get('connections', 0),
                bytes_transferred=data.get('bytes_transferred', 0)
            )
            
            return dict(success=True, status="active")
        else:
            abort(404, "Proxy not found")
            
    except Exception as e:
        response.status = 500
        return dict(success=False, error=str(e))