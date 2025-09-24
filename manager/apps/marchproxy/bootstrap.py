"""
MarchProxy Bootstrap Script

This script initializes the MarchProxy system with:
1. Default admin user
2. Default cluster (Community edition)
3. Basic system configuration
"""

import os
import sys
from datetime import datetime

def init_imports():
    """Initialize imports to avoid circular dependency issues"""
    global db, auth, cluster_manager, license_manager, create_audit_log
    
    # Import after py4web is initialized
    from . import db
    from .common import auth, cluster_manager, license_manager, create_audit_log
    return db, auth, cluster_manager, license_manager, create_audit_log

db = auth = cluster_manager = license_manager = create_audit_log = None

def create_admin_user(username='admin', password='admin', email='admin@localhost'):
    """Create default admin user"""
    print(f"Creating admin user: {username}")
    
    # Check if admin user already exists
    existing = db(db.auth_user.username == username).select().first()
    if existing:
        print(f"Admin user {username} already exists")
        return existing.id
    
    # Hash password
    password_hash = auth.hash_password(password)
    
    # Create admin user
    admin_id = db.auth_user.insert(
        username=username,
        email=email,
        password_hash=password_hash,
        first_name='System',
        last_name='Administrator', 
        is_admin=True,
        is_active=True,
        created_at=datetime.utcnow()
    )
    
    db.commit()
    print(f"Created admin user with ID: {admin_id}")
    return admin_id

def create_default_cluster():
    """Create default cluster for Community edition"""
    print("Creating default cluster...")
    
    # Check if default cluster exists
    existing = db(db.clusters.is_default == True).select().first()
    if existing:
        print("Default cluster already exists")
        return existing.id
    
    try:
        cluster_id = cluster_manager.create_default_cluster()
        db.commit()
        print(f"Created default cluster with ID: {cluster_id}")
        return cluster_id
    except Exception as e:
        print(f"Error creating default cluster: {e}")
        return None

def setup_license_cache():
    """Setup license cache for Community edition"""
    print("Setting up license cache...")
    
    # Community edition defaults
    community_license = {
        'valid': True,
        'edition': 'Community',
        'max_proxies': 3,
        'features': []  # No enterprise features
    }
    
    # Check if Community license cache exists
    existing = db(db.license_cache.license_key == 'community').select().first()
    if not existing:
        db.license_cache.insert(
            license_key='community',
            product_name='marchproxy',
            validation_data=community_license,
            is_valid=True,
            max_proxies=3,
            current_proxies=0,
            last_validated=datetime.utcnow(),
            created_at=datetime.utcnow()
        )
        db.commit()
        print("Created Community license cache entry")
    else:
        print("License cache already exists")

def bootstrap_system():
    """Bootstrap the entire MarchProxy system"""
    print("Starting MarchProxy bootstrap process...")
    
    try:
        # Initialize imports
        global db, auth, cluster_manager, license_manager, create_audit_log
        db, auth, cluster_manager, license_manager, create_audit_log = init_imports()
        
        # 1. Create admin user
        admin_id = create_admin_user()
        
        # 2. Create default cluster
        cluster_id = create_default_cluster()
        
        # 3. Setup license cache
        setup_license_cache()
        
        # 4. Create audit log entry
        create_audit_log(
            'system_bootstrap', 
            'system', 
            'bootstrap',
            {'admin_id': admin_id, 'cluster_id': cluster_id},
            success=True
        )
        
        db.commit()
        print("Bootstrap completed successfully!")
        
        print("\nDefault credentials:")
        print("Username: admin")
        print("Password: admin")
        print("\n⚠️  Please change the default password after first login!")
        
        return True
        
    except Exception as e:
        print(f"Bootstrap failed: {e}")
        import traceback
        traceback.print_exc()
        return False

if __name__ == '__main__':
    bootstrap_system()