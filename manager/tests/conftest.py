"""
Shared test fixtures for MarchProxy Manager unit tests.

Provides mock DB, Quart app/client, and auth header fixtures.
No real DB connections are made.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest
import pytest_asyncio
from penguin_pytest import mock_grpc_module


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

    # Pre-create common tables so tests can configure them easily
    for table_name in [
        "clusters",
        "proxy_servers",
        "users",
        "services",
        "mappings",
        "certificates",
        "user_cluster_assignments",
    ]:
        tbl = MagicMock(name=f"db.{table_name}")
        tbl.insert = MagicMock(return_value=1)
        tbl.__getitem__ = MagicMock(return_value=None)
        setattr(db, table_name, tbl)

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
# Quart app / client fixtures
# ---------------------------------------------------------------------------

@pytest_asyncio.fixture
async def test_app():
    """
    Create a Quart test application without real DB or JWT dependencies.
    """
    mock_db_manager = MagicMock()
    mock_db_manager.initialize_schema.return_value = True
    mock_db_manager.get_pydal_connection.return_value = _make_mock_db()
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


@pytest.fixture
def admin_payload() -> dict:
    """Decoded JWT payload for an admin user."""
    return {
        "user_id": 1,
        "username": "admin",
        "email": "admin@test.example",
        "is_admin": True,
    }


@pytest.fixture
def user_payload() -> dict:
    """Decoded JWT payload for a regular (non-admin) user."""
    return {
        "user_id": 2,
        "username": "testuser",
        "email": "user@test.example",
        "is_admin": False,
    }
