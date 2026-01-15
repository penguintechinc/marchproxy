"""
Database migration: Add RBAC tables

Adds role-based access control tables with OAuth2-style scoped permissions.

Migration ID: add_rbac_tables
Created: 2026-01-13
"""

import logging
from datetime import datetime

from models.rbac import RBACModel, DEFAULT_ROLES

logger = logging.getLogger(__name__)


def upgrade(db):
    """Add RBAC tables and initialize default roles"""
    logger.info("Starting RBAC tables migration...")

    # Define RBAC tables
    RBACModel.define_tables(db)
    logger.info("RBAC tables defined")

    # Initialize default roles
    RBACModel.initialize_default_roles(db)
    logger.info("Default roles initialized")

    # Migrate existing admin users to Admin role
    admin_users = db(db.users.is_admin == True).select()
    for user in admin_users:
        try:
            from models.rbac import PermissionScope
            RBACModel.assign_role(
                db,
                user.id,
                'admin',
                scope=PermissionScope.GLOBAL,
                granted_by=None  # System migration
            )
            logger.info(f"Assigned Admin role to existing admin user: {user.username}")
        except Exception as e:
            logger.warning(f"Could not assign admin role to {user.username}: {e}")

    # Migrate existing service owners to Service Owner role
    service_assignments = db(db.user_service_assignments.role == 'owner').select()
    for assignment in service_assignments:
        try:
            from models.rbac import PermissionScope
            RBACModel.assign_role(
                db,
                assignment.user_id,
                'service_owner',
                scope=PermissionScope.SERVICE,
                resource_id=assignment.service_id,
                granted_by=None  # System migration
            )
            logger.info(
                f"Assigned Service Owner role for service {assignment.service_id} "
                f"to user {assignment.user_id}"
            )
        except Exception as e:
            logger.warning(f"Could not assign service owner role: {e}")

    db.commit()
    logger.info("RBAC migration completed successfully")


def downgrade(db):
    """Remove RBAC tables (WARNING: This will delete all role data)"""
    logger.warning("Rolling back RBAC migration - this will delete all role data")

    # Drop tables in reverse order
    tables_to_drop = [
        'user_permissions_cache',
        'user_roles',
        'roles'
    ]

    for table_name in tables_to_drop:
        if table_name in db.tables:
            db[table_name].drop()
            logger.info(f"Dropped table: {table_name}")

    db.commit()
    logger.info("RBAC migration rollback completed")


if __name__ == '__main__':
    """Run migration standalone"""
    import sys
    import os

    # Add parent directory to path
    sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

    from pydal import DAL
    import os

    # Get database URL from environment
    db_url = os.getenv('DATABASE_URL', 'sqlite://storage.db')

    # Initialize database
    db = DAL(db_url, folder='databases', migrate=True)

    # Import models to define tables
    from models.auth import UserModel, SessionModel
    UserModel.define_table(db)
    SessionModel.define_table(db)

    # Run migration
    choice = input("Run migration? (upgrade/downgrade): ").strip().lower()

    if choice == 'upgrade':
        upgrade(db)
        print("✓ RBAC tables added successfully")
    elif choice == 'downgrade':
        confirm = input("⚠️  This will DELETE all role data. Continue? (yes/no): ")
        if confirm.lower() == 'yes':
            downgrade(db)
            print("✓ RBAC tables removed")
        else:
            print("Cancelled")
    else:
        print("Invalid choice. Use 'upgrade' or 'downgrade'")

    db.close()
