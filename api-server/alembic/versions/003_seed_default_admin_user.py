"""Seed default admin user for initial login

Revision ID: 003
Revises: 002
Create Date: 2025-12-19 15:30:00.000000

This migration creates the default admin user for MarchProxy:
- Email: admin@localhost.local
- Password: admin123
- Role: Administrator

This is for development and testing purposes. In production, you should:
1. Delete this default user after first login
2. Create a new admin user with a strong password
3. Never share default credentials
"""
from typing import Sequence, Union
from alembic import op
import sqlalchemy as sa
from datetime import datetime

# Import security utilities for password hashing
import sys
import os
sys.path.insert(0, os.path.abspath(os.path.dirname(__file__) + '/../../'))

try:
    from app.core.security import get_password_hash
except ImportError:
    # Fallback if import fails
    from passlib.context import CryptContext
    pwd_context = CryptContext(schemes=["bcrypt"], deprecated="auto")
    def get_password_hash(password: str) -> str:
        return pwd_context.hash(password)


# revision identifiers, used by Alembic.
revision: str = '003'
down_revision: Union[str, None] = '002'
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


def upgrade() -> None:
    """Create default admin user."""
    # Hash the password
    password_hash = get_password_hash("admin123")

    # Insert default admin user
    op.execute(
        sa.text(
            """
            INSERT INTO auth_user (email, username, password_hash, first_name, last_name,
                                  totp_enabled, is_active, is_admin, is_verified,
                                  created_at, updated_at)
            VALUES
            (:email, :username, :password_hash, :first_name, :last_name,
             :totp_enabled, :is_active, :is_admin, :is_verified,
             :created_at, :updated_at)
            ON CONFLICT (email) DO NOTHING
            """
        ),
        {
            "email": "admin@localhost.local",
            "username": "admin",
            "password_hash": password_hash,
            "first_name": "Admin",
            "last_name": "User",
            "totp_enabled": False,
            "is_active": True,
            "is_admin": True,
            "is_verified": True,
            "created_at": datetime.utcnow(),
            "updated_at": datetime.utcnow(),
        }
    )


def downgrade() -> None:
    """Remove default admin user."""
    op.execute(
        sa.text(
            "DELETE FROM auth_user WHERE email = :email"
        ),
        {"email": "admin@localhost.local"}
    )
