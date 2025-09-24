"""
MarchProxy Common Components

Shared utilities, authentication, and common functionality
used across the MarchProxy management system.
"""

import os
import uuid
import hashlib
import secrets
import json
import logging
from datetime import datetime, timedelta
from typing import Optional, Dict, List, Any
from functools import wraps

import jwt
import pyotp
import qrcode
import io
import base64
import bcrypt
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives.kdf.pbkdf2 import PBKDF2HMAC

from py4web import request, response, redirect, URL, abort, HTTP, Session
from py4web.utils.auth import Auth
from py4web.utils.form import Form
from pydal import Field

# Import database from models  
from . import db
from .syslog import cluster_syslog_manager

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Constants
COMMUNITY_MAX_PROXIES = 3
JWT_ALGORITHM = 'HS256'
TOTP_ISSUER = 'MarchProxy'

class MarchProxyAuth:
    """Enhanced authentication system for MarchProxy"""
    
    def __init__(self, db):
        self.db = db
        self.jwt_secret = os.environ.get('JWT_SECRET', 'change-me-in-production')
    
    def hash_password(self, password: str) -> str:
        """Hash a password using bcrypt"""
        salt = bcrypt.gensalt()
        return bcrypt.hashpw(password.encode('utf-8'), salt).decode('utf-8')
    
    def verify_password(self, password: str, hashed: str) -> bool:
        """Verify a password against its hash"""
        return bcrypt.checkpw(password.encode('utf-8'), hashed.encode('utf-8'))
    
    def authenticate_user(self, username: str, password: str, totp_token: str = None) -> Optional[Dict]:
        """Authenticate user with username/password and optional 2FA"""
        user = self.db((self.db.auth_user.username == username) & 
                      (self.db.auth_user.is_active == True)).select().first()
        
        if not user or not user.password_hash:
            return None
            
        # Verify password
        if not self.verify_password(password, user.password_hash):
            return None
            
        # Check 2FA if enabled
        if user.totp_enabled:
            if not totp_token or not self.verify_2fa(user.id, totp_token):
                return {'error': '2fa_required', 'user_id': user.id}
        
        # Update login tracking
        self.db(self.db.auth_user.id == user.id).update(
            last_login=datetime.utcnow(),
            login_count=self.db.auth_user.login_count + 1
        )
        
        return user.as_dict()
    
    def get_user_session(self):
        """Get user session information"""
        return request.app_name + '_user' in request.session
    
    def set_user_session(self, user_data: Dict):
        """Set user session information"""
        request.session[request.app_name + '_user'] = user_data
    
    def clear_user_session(self):
        """Clear user session"""
        if request.app_name + '_user' in request.session:
            del request.session[request.app_name + '_user']
    
    def generate_api_key(self, name: str, cluster_id: Optional[int] = None, 
                        user_id: Optional[int] = None, permissions: List[str] = None) -> tuple:
        """Generate a new API key"""
        # Generate secure random key
        key = secrets.token_urlsafe(32)
        key_hash = hashlib.sha256(key.encode()).hexdigest()
        key_prefix = key[:8]
        
        # Store in database
        api_key_id = self.db.api_keys.insert(
            key_name=name,
            key_hash=key_hash,
            key_prefix=key_prefix,
            cluster_id=cluster_id,
            user_id=user_id,
            permissions=permissions or [],
            created_by=self.get_current_user_id()
        )
        
        return key, api_key_id
    
    def validate_api_key(self, key: str) -> Optional[Dict]:
        """Validate an API key and return key information"""
        key_hash = hashlib.sha256(key.encode()).hexdigest()
        
        api_key = self.db(
            (self.db.api_keys.key_hash == key_hash) & 
            (self.db.api_keys.is_active == True)
        ).select().first()
        
        if not api_key:
            return None
        
        # Check expiry
        if api_key.expires_at and api_key.expires_at < datetime.utcnow():
            return None
        
        # Update usage tracking
        self.db(self.db.api_keys.id == api_key.id).update(
            last_used=datetime.utcnow(),
            usage_count=self.db.api_keys.usage_count + 1
        )
        
        return api_key.as_dict()
    
    def generate_cluster_api_key(self, cluster_id: int) -> str:
        """Generate API key specifically for cluster proxy authentication"""
        cluster = self.db.clusters[cluster_id]
        if not cluster:
            raise ValueError("Cluster not found")
        
        # Generate new API key
        api_key = secrets.token_urlsafe(32)
        api_key_hash = hashlib.sha256(api_key.encode()).hexdigest()
        
        # Update cluster with new API key
        self.db(self.db.clusters.id == cluster_id).update(
            api_key=api_key_hash,
            api_key_created_at=datetime.utcnow(),
            api_key_rotated_at=datetime.utcnow() if cluster.api_key else None
        )
        
        return api_key
    
    def setup_2fa(self, user_id: int) -> tuple:
        """Setup 2FA for a user"""
        user = self.db.auth_user[user_id]
        if not user:
            raise ValueError("User not found")
        
        # Generate TOTP secret
        secret = pyotp.random_base32()
        
        # Create TOTP URI for QR code
        totp_uri = pyotp.totp.TOTP(secret).provisioning_uri(
            name=user.username,
            issuer_name=TOTP_ISSUER
        )
        
        # Generate QR code
        qr = qrcode.QRCode(version=1, box_size=10, border=5)
        qr.add_data(totp_uri)
        qr.make(fit=True)
        
        img = qr.make_image(fill_color="black", back_color="white")
        img_buffer = io.BytesIO()
        img.save(img_buffer, format='PNG')
        img_buffer.seek(0)
        
        qr_code_data = base64.b64encode(img_buffer.getvalue()).decode()
        
        # Update user with TOTP secret (not enabled until verified)
        self.db(self.db.auth_user.id == user_id).update(
            totp_secret=secret
        )
        
        return secret, qr_code_data
    
    def verify_2fa(self, user_id: int, token: str) -> bool:
        """Verify 2FA token"""
        user = self.db.auth_user[user_id]
        if not user or not user.totp_secret:
            return False
        
        totp = pyotp.TOTP(user.totp_secret)
        return totp.verify(token, valid_window=1)  # Allow 1 window for clock skew
    
    def enable_2fa(self, user_id: int, token: str) -> bool:
        """Enable 2FA after successful verification"""
        if self.verify_2fa(user_id, token):
            # Generate backup codes
            backup_codes = [secrets.token_hex(4) for _ in range(10)]
            
            self.db(self.db.auth_user.id == user_id).update(
                totp_enabled=True,
                backup_codes=json.dumps(backup_codes)
            )
            return True
        return False
    
    def get_current_user_id(self) -> Optional[int]:
        """Get current authenticated user ID"""
        user_data = request.session.get(request.app_name + '_user', {})
        return user_data.get('id')
    
    def get_current_user(self) -> Optional[Dict]:
        """Get current authenticated user data"""
        user_id = self.get_current_user_id()
        if user_id:
            return self.db.auth_user[user_id]
        return None

class LicenseManager:
    """License validation and management for Enterprise features"""
    
    def __init__(self, db):
        self.db = db
        self.license_server_url = os.environ.get('LICENSE_SERVER_URL', 'https://license.penguintech.io')
        self.product_name = 'marchproxy'
    
    def validate_license(self, license_key: str) -> Dict:
        """Validate license with license server"""
        import requests
        
        try:
            # Check cache first
            cached = self.db(
                (self.db.license_cache.license_key == license_key) &
                (self.db.license_cache.expires_at > datetime.utcnow())
            ).select().first()
            
            if cached and cached.is_valid:
                return cached.validation_data
            
            # Validate with license server
            response = requests.post(
                f"{self.license_server_url}/api/v2/validate",
                json={
                    'license_key': license_key,
                    'product_name': self.product_name
                },
                timeout=10
            )
            
            if response.status_code == 200:
                validation_data = response.json()
                
                # Update cache
                self.db.license_cache.update_or_insert(
                    (self.db.license_cache.license_key == license_key),
                    license_key=license_key,
                    validation_data=validation_data,
                    is_valid=validation_data.get('valid', False),
                    expires_at=datetime.utcnow() + timedelta(hours=24),
                    features=validation_data.get('features', []),
                    max_proxies=validation_data.get('limits', {}).get('proxies', COMMUNITY_MAX_PROXIES),
                    last_validated=datetime.utcnow()
                )
                
                return validation_data
            else:
                raise Exception(f"License validation failed: {response.status_code}")
                
        except Exception as e:
            logger.error(f"License validation error: {e}")
            
            # Check for grace period
            cached = self.db(self.db.license_cache.license_key == license_key).select().first()
            if cached and cached.in_grace_period and cached.grace_period_until > datetime.utcnow():
                return cached.validation_data
            
            return {'valid': False, 'error': str(e)}
    
    def is_enterprise_feature_enabled(self, feature: str, license_key: str = None) -> bool:
        """Check if an enterprise feature is enabled"""
        if not license_key:
            # Check environment or default license
            license_key = os.environ.get('LICENSE_KEY')
            if not license_key:
                return False  # Community edition
        
        validation_data = self.validate_license(license_key)
        features = validation_data.get('features', [])
        
        feature_map = {
            'multi_cluster': 'unlimited_proxies',
            'saml_auth': 'saml_authentication',
            'oauth2_auth': 'oauth2_authentication', 
            'advanced_auth': 'advanced_auth'
        }
        
        required_feature = feature_map.get(feature, feature)
        return required_feature in features
    
    def get_proxy_limit(self, license_key: str = None) -> int:
        """Get the maximum number of proxies allowed"""
        if not license_key:
            license_key = os.environ.get('LICENSE_KEY')
            if not license_key:
                return COMMUNITY_MAX_PROXIES  # Community edition
        
        validation_data = self.validate_license(license_key)
        return validation_data.get('limits', {}).get('proxies', COMMUNITY_MAX_PROXIES)

class ClusterManager:
    """Cluster management for Enterprise multi-cluster support"""
    
    def __init__(self, db, auth, license_manager):
        self.db = db
        self.auth = auth
        self.license_manager = license_manager
    
    def create_default_cluster(self) -> int:
        """Create default cluster for Community edition"""
        # Check if default cluster exists
        default_cluster = self.db(self.db.clusters.is_default == True).select().first()
        if default_cluster:
            return default_cluster.id
        
        # Create default cluster with API key
        api_key = secrets.token_urlsafe(32)
        api_key_hash = hashlib.sha256(api_key.encode()).hexdigest()
        
        cluster_id = self.db.clusters.insert(
            name='Default',
            description='Default cluster for Community edition',
            is_default=True,
            api_key=api_key_hash,
            max_proxies=COMMUNITY_MAX_PROXIES,
            created_by=1  # System user
        )
        
        return cluster_id
    
    def create_cluster(self, name: str, description: str = '', user_id: int = None) -> int:
        """Create new cluster (Enterprise only)"""
        license_key = os.environ.get('LICENSE_KEY')
        if not self.license_manager.is_enterprise_feature_enabled('multi_cluster', license_key):
            raise ValueError("Multi-cluster support requires Enterprise license")
        
        # Generate API key for cluster
        api_key = secrets.token_urlsafe(32)
        api_key_hash = hashlib.sha256(api_key.encode()).hexdigest()
        
        # Get proxy limit from license
        max_proxies = self.license_manager.get_proxy_limit(license_key)
        
        cluster_id = self.db.clusters.insert(
            name=name,
            description=description,
            api_key=api_key_hash,
            max_proxies=max_proxies,
            created_by=user_id or self.auth.get_current_user_id()
        )
        
        return cluster_id

# Initialize global instances
auth = MarchProxyAuth(db)
license_manager = LicenseManager(db) 
cluster_manager = ClusterManager(db, auth, license_manager)

# Utility functions
def require_auth(f):
    """Decorator to require authentication"""
    @wraps(f)
    def wrapper(*args, **kwargs):
        user_id = auth.get_current_user_id()
        if not user_id:
            redirect(URL('auth/login'))
        return f(*args, **kwargs)
    return wrapper

def require_admin(f):
    """Decorator to require admin privileges"""
    @wraps(f)
    def wrapper(*args, **kwargs):
        user_id = auth.get_current_user_id()
        if not user_id:
            redirect(URL('auth/login'))
        
        user = db.auth_user[user_id]
        if not user or not user.is_admin:
            abort(403, "Admin privileges required")
        
        return f(*args, **kwargs)
    return wrapper

def require_license_feature(feature):
    """Decorator to require specific license feature"""
    def decorator(f):
        @wraps(f)
        def wrapper(*args, **kwargs):
            if not license_manager.is_enterprise_feature_enabled(feature):
                abort(403, f"Feature '{feature}' requires Enterprise license")
            return f(*args, **kwargs)
        return wrapper
    return decorator

def create_audit_log(event_type: str, resource_type: str = None, resource_id: str = None, 
                    event_data: Dict = None, success: bool = True, error_message: str = None):
    """Create audit log entry and send to syslog if configured"""
    user_id = auth.get_current_user_id()
    user = db.auth_user[user_id] if user_id else None
    
    db.audit_log.insert(
        event_type=event_type,
        resource_type=resource_type,
        resource_id=resource_id,
        user_id=user_id,
        event_data=event_data or {},
        ip_address=request.environ.get('REMOTE_ADDR'),
        user_agent=request.environ.get('HTTP_USER_AGENT'),
        success=success,
        error_message=error_message
    )
    
    # Send to cluster syslog if authentication event and cluster has logging enabled
    if event_type.startswith(('login', 'logout', '2fa')) and user:
        # Get user's clusters or default cluster
        if user.is_admin:
            clusters = db(
                (db.clusters.is_active == True) & 
                (db.clusters.log_auth == True) &
                (db.clusters.syslog_endpoint != None)
            ).select()
        else:
            clusters = db(
                (db.user_cluster_assignments.user_id == user_id) &
                (db.user_cluster_assignments.cluster_id == db.clusters.id) &
                (db.clusters.is_active == True) &
                (db.clusters.log_auth == True) &
                (db.clusters.syslog_endpoint != None)
            ).select(db.clusters.ALL)
        
        # Log to each cluster's syslog
        for cluster in clusters:
            try:
                cluster_syslog_manager.log_to_cluster(
                    cluster.id,
                    cluster.syslog_endpoint,
                    'auth',
                    event_type=event_type,
                    username=user.username,
                    user_id=user_id,
                    ip_address=request.environ.get('REMOTE_ADDR'),
                    success=success,
                    details=event_data or {}
                )
            except Exception as e:
                logger.warning(f"Failed to log to cluster {cluster.id} syslog: {e}")