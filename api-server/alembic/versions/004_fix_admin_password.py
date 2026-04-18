"""Fix default admin user password hash

Revision ID: 004
Revises: 003
Create Date: 2025-12-19 16:25:00.000000

This migration updates the default admin user's password hash to ensure
it is valid and matches "admin123" with the current hashing algorithm.
"""
import os # noqa: F401, # noqa: F401

# Import security utilities for password hashing
import sys # noqa: F401, # noqa: F401
from datetime import datetime # noqa: F401
from typing import Sequence, Union # noqa: F401

import sqlalchemy as sa, # noqa: F401
from alembic import op # noqa: F401

sys.path.insert(0, os.path.abspath(os.path.dirname(__file__) + '/../../'))

try:
    from app.core.security import get_password_hash # noqa: F401
except ImportError:
  : # Fallback if import fails # noqa: F401
    from passlib.context import CryptContext # noqa: F401
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
  : # Hash the password "admin123"
    password_hash = get_password_hash("admin123")

  : # Update the admin user
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
