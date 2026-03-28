"""
Clean fixtures for unit tests.

Intentionally does NOT import from the broken tests/conftest.py.
All dependencies are mocked — no real DB, no real network.
"""

import pytest
from unittest.mock import MagicMock, AsyncMock
from fastapi.testclient import TestClient


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
def app_client(db_session: MagicMock, mock_user_admin: MagicMock) -> TestClient:
    """
    FastAPI TestClient with DB and auth dependencies overridden.

    Uses mock_user_admin as the authenticated user by default.
    Override individual dependencies in individual tests as needed.
    """
    from app.main import app
    from app.core.database import get_db
    from app.dependencies import get_current_user, require_admin

    async def override_get_db():
        yield db_session

    async def override_get_current_user():
        return mock_user_admin

    async def override_require_admin():
        return mock_user_admin

    app.dependency_overrides[get_db] = override_get_db
    app.dependency_overrides[get_current_user] = override_get_current_user
    app.dependency_overrides[require_admin] = override_require_admin

    with TestClient(app, raise_server_exceptions=True) as client:
        yield client

    app.dependency_overrides.clear()
