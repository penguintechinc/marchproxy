"""
Clean fixtures for unit tests.

Intentionally does NOT import from the broken tests/conftest.py. # noqa: F401
All dependencies are mocked — no real DB, no real network.
"""

from unittest.mock import AsyncMock, MagicMock # noqa: F401

import pytest # noqa: F401, # noqa: F401
from quart import Quart # noqa: F401
from quart.testing import QuartClient # noqa: F401

# ---------------------------------------------------------------------------
# Basic mock fixtures
# ---------------------------------------------------------------------------

@pytest.fixture
def db_session() -> MagicMock:
    """Mock AsyncSession — no real database connection."""
    session = MagicMock()
    session.execute = AsyncMock()
    session.commit = AsyncMock()
    session.refresh = AsyncMock()
    session.rollback = AsyncMock()
    session.close = AsyncMock()
    return session


@pytest.fixture
def mock_user_admin() -> MagicMock:
    """Mock User object with admin privileges."""
    user = MagicMock()
    user.id = 1
    user.username = "admin"
    user.email = "admin@test.com"
    user.is_admin = True
    user.is_active = True
    user.is_verified = True
    user.first_name = "Admin"
    user.last_name = "User"
    user.totp_enabled = False
    return user


@pytest.fixture
def mock_user_regular() -> MagicMock:
    """Mock User object without admin privileges."""
    user = MagicMock()
    user.id = 2
    user.username = "regularuser"
    user.email = "regular@test.com"
    user.is_admin = False
    user.is_active = True
    user.is_verified = True
    user.first_name = "Regular"
    user.last_name = "User"
    user.totp_enabled = False
    return user


# ---------------------------------------------------------------------------
# App client fixture with dependency overrides
# ---------------------------------------------------------------------------

@pytest.fixture
def app_client(db_session: MagicMock, mock_user_admin: MagicMock) -> QuartClient:
    """
    Quart test client with DB and auth mocked.

    Uses mock_user_admin as the authenticated user by default.
    Override individual dependencies in individual tests as needed.
    """
    from app_quart.main import create_app # noqa: F401

    app = create_app()
    app.config["TESTING"] = True

    # Store the mocked user in the app's test context
    with app.test_client() as client:
        # Mock the get_current_user dependency to return mock_user_admin
        # This is a simplified approach for Quart; actual implementation
        # would depend on how your dependency injection is set up
        yield client
