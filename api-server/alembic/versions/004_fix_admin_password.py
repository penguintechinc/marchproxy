"""Fix default admin user password hash

Revision ID: 004
Revises: 003
Create Date: 2025-12-19 16:25:00.000000

This migration updates the default admin user's password hash to ensure
it is valid and matches "admin123" with the current hashing algorithm.
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
revision: str = '004'
down_revision: Union[str, None] = '003'
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


def upgrade() -> None:
    """Update admin user password."""
    # Hash the password "admin123"
    password_hash = get_password_hash("admin123")

    # Update the admin user
    op.execute(
        sa.text(
            """
            UPDATE auth_user
            SET password_hash = :password_hash,
                updated_at = :updated_at
            WHERE email = :email
            """
        ),
        {
            "email": "admin@localhost.local",
            "password_hash": password_hash,
            "updated_at": datetime.utcnow(),
        }
    )


def downgrade() -> None:
    """
    No-op for downgrade since we cannot restore the previous unknown/bad hash reliably,
    and rolling back a password fix isn't usually desired. 
    However, strictly proper migrations might restore a backup, but here we just pass.
    """
    pass
