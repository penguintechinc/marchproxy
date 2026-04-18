"""
Shared test fixtures for MarchProxy Manager unit tests.

Provides mock DB, Quart app/client, and auth header fixtures.
No real DB connections are made.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import sys
from pathlib import Path
from unittest.mock import AsyncMock, MagicMock, patch

import pytest
import pytest_asyncio

# Add parent directory to sys.path so imports work from tests/
sys.path.insert(0, str(Path(__file__).parent.parent))


# ---------------------------------------------------------------------------
# Mock DB fixture
# ---------------------------------------------------------------------------

def _make_mock_db() -> MagicMock:
    """
    Build a MagicMock that mimics the PyDAL DAL interface used by models.

    Supports patterns like:
      db.clusters.insert(...)
      db(condition).select().first()
      db(condition).count()
      db(condition).update(...)
      db(condition).delete()
      db.table[id]
    """
    db = MagicMock(name="db")

    # Allow attribute access for table names to return table mocks
    def _table_attr(name: str):
        tbl = MagicMock(name=f"db.{name}")
        tbl.insert = MagicMock(return_value=1)
        tbl.__getitem__ = MagicMock(return_value=None)
        return tbl

    def _make_table_mock(name: str) -> MagicMock:
        """Create a table mock whose field attributes support comparison operators."""
        tbl = MagicMock(name=f"db.{name}")
        tbl.insert = MagicMock(return_value=1)
        tbl.__getitem__ = MagicMock(return_value=None)

        # Add field mocks that support comparison operators
        # These are needed for PyDAL queries like db.table.field > 0
        for field_name in ["id", "created_at", "updated_at"]:
            field_mock = MagicMock(name=f"db.{name}.{field_name}")
            field_mock.__gt__ = MagicMock(return_value=MagicMock())
            field_mock.__lt__ = MagicMock(return_value=MagicMock())
            field_mock.__ge__ = MagicMock(return_value=MagicMock())
            field_mock.__le__ = MagicMock(return_value=MagicMock())
            field_mock.__eq__ = MagicMock(return_value=MagicMock())
            setattr(tbl, field_name, field_mock)

        return tbl

    # Pre-create common tables so tests can configure them easily
    for table_name in [
        "clusters",
        "proxy_servers",
        "users",
        "services",
        "mappings",
        "certificates",
        "user_cluster_assignments",
        "proxy_metrics",
        "media_settings",
        "media_streams",
    ]:
        tbl = _make_table_mock(table_name)
        setattr(db, table_name, tbl)

    # Configure date/time fields that need comparison operators with datetime objects.
    # MagicMock does support __gt__ via magic methods but only when the *right* side
    # is not a datetime (Python requires __gt__ on the left object).
    # We patch them explicitly for each field used in datetime comparisons.
    from unittest.mock import MagicMock as _MM

    def _patch_datetime_field(field_mock):
        field_mock.__gt__ = _MM(return_value=_MM())
        field_mock.__lt__ = _MM(return_value=_MM())
        field_mock.__ge__ = _MM(return_value=_MM())
        field_mock.__le__ = _MM(return_value=_MM())

    _patch_datetime_field(db.proxy_servers.last_seen)
    _patch_datetime_field(db.proxy_metrics.timestamp)

    # db(condition) returns a query mock; configure .select().first() etc. per test
    query_mock = MagicMock(name="db_query")
    select_mock = MagicMock(name="db_query.select")
    select_mock.return_value = MagicMock(first=MagicMock(return_value=None), __iter__=iter([]))
    query_mock.select = select_mock
    query_mock.count = MagicMock(return_value=0)
    query_mock.update = MagicMock(return_value=1)
    query_mock.delete = MagicMock(return_value=0)
    # Allow chained __call__: db(cond)(another_cond) → same query mock
    query_mock.__call__ = MagicMock(return_value=query_mock)
    db.__call__ = MagicMock(return_value=query_mock)

    return db


@pytest.fixture
def mock_db() -> MagicMock:
    """PyDAL-like MagicMock database. No real connections."""
    return _make_mock_db()


@pytest.fixture
def db_query(mock_db) -> MagicMock:
    """Shortcut to the query mock returned by mock_db(condition)."""
    return mock_db.return_value  # type: ignore[attr-defined]


# ---------------------------------------------------------------------------
# Admin payload fixture (shared by all tests)
# ---------------------------------------------------------------------------

@pytest.fixture
def admin_payload() -> dict:
    """Decoded JWT payload for an admin user."""
    return {
        "user_id": 1,
        "sub": "1",
        "username": "admin",
        "email": "admin@test.example",
        "is_admin": True,
        "scope": "*:read *:write *:admin *:delete settings:write users:admin",
        "roles": ["admin"],
        "tenant": "test",
        "session_id": "sess-admin",
    }


@pytest.fixture
def user_payload() -> dict:
    """Decoded JWT payload for a regular (non-admin) user."""
    return {
        "user_id": 2,
        "sub": "2",
        "username": "testuser",
        "email": "user@test.example",
        "is_admin": False,
        "scope": "",
        "roles": [],
        "tenant": "test",
        "session_id": "sess-user",
    }


# ---------------------------------------------------------------------------
# Global autouse fixture: Mock _validate_token ONLY for client tests
# ---------------------------------------------------------------------------

@pytest.fixture(autouse=True)
def _autouse_mock_validate_token_for_client(request, admin_payload):
    """
    Automatically mock middleware.auth._validate_token for HTTP client tests only.

    Only patches if the test uses test_client fixture.
    This allows direct function tests to control their own mocks.
    """
    # Only apply to tests that use test_client fixture
    if "test_client" not in request.fixturenames:
        yield
        return

    with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_validate:
        mock_validate.return_value = admin_payload
        yield mock_validate


# ---------------------------------------------------------------------------
# Quart app / client fixtures
# ---------------------------------------------------------------------------

@pytest_asyncio.fixture
async def test_app():
    """
    Create a Quart test application without real DB or JWT dependencies.
    """
    mock_db_manager = MagicMock()
    mock_db_manager.initialize_schema.return_value = True
    mock_db = _make_mock_db()
    mock_db_manager.get_pydal_connection.return_value = mock_db
    mock_db_manager.db_type = "sqlite"

    with patch("database.get_db_manager", return_value=mock_db_manager):
        from quart_app import create_app

        test_config = {
            "DATABASE_URL": "sqlite:///test.db",
            "JWT_SECRET": "test-secret-for-testing-only-32chars!",
            "DB_TYPE": "sqlite",
        }
        app = create_app(config=test_config)
        app.config["TESTING"] = True

        # Attach the mock_db to the app for use in request context
        app.db = mock_db

        yield app


@pytest_asyncio.fixture
async def test_client(test_app):
    """Quart test client bound to test_app."""
    return test_app.test_client()


# ---------------------------------------------------------------------------
# Auth header fixtures
# ---------------------------------------------------------------------------

@pytest.fixture
def admin_token() -> str:
    """A mock JWT token string representing an admin user."""
    return "mock-admin-jwt-token-abcdef1234567890"


@pytest.fixture
def user_token() -> str:
    """A mock JWT token string representing a regular user."""
    return "mock-user-jwt-token-abcdef1234567890"


@pytest.fixture
def admin_headers(admin_token) -> dict:
    """HTTP headers with admin Bearer token."""
    return {"Authorization": f"Bearer {admin_token}"}


@pytest.fixture
def user_headers(user_token) -> dict:
    """HTTP headers with regular user Bearer token."""
    return {"Authorization": f"Bearer {user_token}"}
