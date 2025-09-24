"""
MarchProxy - Main py4web Application

This is the main application entry point for the MarchProxy management system.
It handles all core functionality including authentication, cluster management,
service configuration, and API endpoints for proxy servers.
"""

import os

# Load version from .version file
try:
    version_file = os.path.join(os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(__file__)))), '.version')
    with open(version_file, 'r') as f:
        VERSION = f.read().strip()
except FileNotFoundError:
    VERSION = 'v0.1.0.1'  # Default version if file not found
from py4web import action, request, abort, redirect, URL, Field
from py4web.utils.cors import CORS
from py4web.utils.form import Form, FormStyleBulma
from py4web.utils.publisher import Publisher
from py4web.utils.auth import Auth
from py4web.utils.mailer import Mailer
from py4web.core import Template

from pydal import DAL, Field

# Initialize database connection
db = None
auth = None
mailer = None

def init_app():
    """Initialize the application components"""
    global db, auth, mailer
    
    # Database connection
    db_uri = os.environ.get('DB_URI', 'sqlite://storage.db')
    # Convert postgresql:// to postgres:// for PyDAL compatibility
    if db_uri.startswith('postgresql://'):
        db_uri = db_uri.replace('postgresql://', 'postgres://', 1)
    db = DAL(db_uri, pool_size=10, migrate=True, fake_migrate_all=False)
    
    # Authentication system
    auth = Auth(db)
    
    # Optional mailer (for notifications)
    mailer = Mailer(
        server=os.environ.get('SMTP_SERVER', 'localhost'),
        sender=os.environ.get('SMTP_SENDER', 'noreply@marchproxy.local'),
        login=os.environ.get('SMTP_LOGIN', ''),
        password=os.environ.get('SMTP_PASSWORD', '')
    ) if os.environ.get('SMTP_SERVER') else None
    
    return db, auth, mailer

# Initialize on import
db, auth, mailer = init_app()

# Import models after database initialization
from . import models

def initialize_default_cluster():
    """Initialize default cluster for Community edition if not exists"""
    try:
        from .common import cluster_manager, auth as marchproxy_auth
        
        # Check if we already have a default cluster
        default_cluster = db(db.clusters.is_default == True).select().first()
        
        if not default_cluster:
            # Create default cluster
            cluster_id = cluster_manager.create_default_cluster()
            
            # Create default admin user if not exists
            admin_user = db(db.auth_user.is_admin == True).select().first()
            if not admin_user:
                admin_password_hash = marchproxy_auth.hash_password('admin123')  # Change in production
                
                admin_id = db.auth_user.insert(
                    username='admin',
                    password_hash=admin_password_hash,
                    email='admin@marchproxy.local',
                    is_admin=True,
                    is_active=True,
                    email_verified=True
                )
                
                # Assign admin to default cluster
                db.user_cluster_assignments.insert(
                    user_id=admin_id,
                    cluster_id=cluster_id,
                    role='admin',
                    assigned_by=admin_id
                )
            
            print(f"[INIT] Created default cluster (ID: {cluster_id}) for Community edition")
        else:
            print(f"[INIT] Default cluster already exists (ID: {default_cluster.id})")
            
    except Exception as e:
        print(f"[ERROR] Failed to initialize default cluster: {e}")

# Initialize default cluster on startup (skip if explicitly disabled)
if not os.environ.get('SKIP_BOOTSTRAP', '').lower() == 'true':
    initialize_default_cluster()