"""
Unit tests for manager/database.py

Tests DatabaseManager static and instance methods without real DB connections.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import threading
from unittest.mock import MagicMock, patch

import pytest

pytestmark = pytest.mark.unit


# ---------------------------------------------------------------------------
# _mask_credentials
# ---------------------------------------------------------------------------

class TestMaskCredentials:
    """Tests for DatabaseManager._mask_credentials (static method)."""

    def setup_method(self):
        # Import here to avoid module-level import issues when DB env vars absent
        from database import DatabaseManager
        self.mask = DatabaseManager._mask_credentials

    def test_masks_password_in_postgres_url(self):
        url = "postgresql://user:secret@localhost:5432/mydb"
        masked = self.mask(url)
        assert "secret" not in masked
        assert "***" in masked

    def test_keeps_username_visible(self):
        url = "postgresql://user:secret@localhost:5432/mydb"
        masked = self.mask(url)
        assert "user" in masked

    def test_keeps_host_and_port_visible(self):
        url = "postgresql://user:secret@localhost:5432/mydb"
        masked = self.mask(url)
        assert "localhost" in masked
        assert "5432" in masked

    def test_keeps_database_name_visible(self):
        url = "postgresql://user:secret@localhost:5432/mydb"
        masked = self.mask(url)
        assert "mydb" in masked

    def test_no_password_returns_url_unchanged(self):
        url = "sqlite:///some/path/to/db.sqlite"
        assert self.mask(url) == url

    def test_mysql_url_masked(self):
        url = "mysql://admin:p@ssw0rd@db.host:3306/appdb"
        masked = self.mask(url)
        assert "p@ssw0rd" not in masked
        assert "***" in masked

    def test_empty_string_returns_unchanged(self):
        assert self.mask("") == ""

    def test_special_chars_in_password_are_masked(self):
        url = "postgresql://user:pa%40ss!word@host/db"
        masked = self.mask(url)
        # Password portion replaced
        assert "pa%40ss!word" not in masked


# ---------------------------------------------------------------------------
# _convert_url_to_pydal
# ---------------------------------------------------------------------------

class TestConvertUrlToPydal:
    """Tests for DatabaseManager._convert_url_to_pydal (static method)."""

    def setup_method(self):
        from database import DatabaseManager
        self.convert = DatabaseManager._convert_url_to_pydal

    def test_postgres_scheme_converted(self):
        url = "postgresql://user:pass@localhost:5432/mydb"
        result = self.convert(url, "postgres")
        assert result.startswith("postgres://")

    def test_postgres_with_port_includes_port(self):
        url = "postgresql://user:pass@localhost:5432/mydb"
        result = self.convert(url, "postgres")
        assert "5432" in result

    def test_postgres_without_port(self):
        url = "postgresql://user:pass@localhost/mydb"
        result = self.convert(url, "postgres")
        assert result.startswith("postgres://")
        assert "None" not in result

    def test_postgres_path_preserved(self):
        url = "postgresql://user:pass@localhost:5432/mydb"
        result = self.convert(url, "postgres")
        assert "/mydb" in result

    def test_postgres_empty_path_defaults_to_marchproxy(self):
        # URL without a path segment
        url = "postgresql://user:pass@localhost:5432"
        result = self.convert(url, "postgres")
        # Should fall back to '/marchproxy'
        assert "/marchproxy" in result

    def test_mysql_scheme_returned(self):
        url = "mysql://user:pass@localhost:3306/mydb"
        result = self.convert(url, "mysql")
        assert result.startswith("mysql://")

    def test_mysql_with_port(self):
        url = "mysql://user:pass@localhost:3306/mydb"
        result = self.convert(url, "mysql")
        assert "3306" in result

    def test_mysql_without_port(self):
        url = "mysql://user:pass@localhost/mydb"
        result = self.convert(url, "mysql")
        assert result.startswith("mysql://")

    def test_mysql_path_preserved(self):
        url = "mysql://user:pass@localhost:3306/mydb"
        result = self.convert(url, "mysql")
        assert "/mydb" in result

    def test_sqlite_scheme_returned(self):
        url = "sqlite:////tmp/test.db"
        result = self.convert(url, "sqlite")
        assert result.startswith("sqlite:///")

    def test_sqlite_path_stripped_of_leading_slash(self):
        url = "sqlite:////tmp/test.db"
        result = self.convert(url, "sqlite")
        # Leading slash of the path is stripped by lstrip('/')
        assert "tmp/test.db" in result

    def test_sqlite_empty_path_defaults_to_memory(self):
        # Simulate a URL with no path
        url = "sqlite://"
        result = self.convert(url, "sqlite")
        # When parsed path is empty, falls back to ':memory:'
        assert ":memory:" in result

    def test_unsupported_db_type_raises(self):
        with pytest.raises(ValueError, match="Unsupported"):
            self.convert("some://url", "oracle")


# ---------------------------------------------------------------------------
# DatabaseManager.__init__
# ---------------------------------------------------------------------------

class TestDatabaseManagerInit:
    """Tests for DatabaseManager initialisation."""

    def test_raises_when_database_url_missing(self):
        from database import DatabaseManager
        with patch.dict("os.environ", {}, clear=True):
            with pytest.raises(ValueError, match="DATABASE_URL"):
                DatabaseManager()

    def test_raises_when_db_type_invalid(self):
        from database import DatabaseManager
        with patch.dict("os.environ", {"DATABASE_URL": "sqlite:///test.db", "DB_TYPE": "oracle"}):
            with pytest.raises(ValueError, match="DB_TYPE"):
                DatabaseManager()

    def test_accepts_sqlite_url(self):
        from database import DatabaseManager
        with patch.dict("os.environ", {"DATABASE_URL": "sqlite:///test.db", "DB_TYPE": "sqlite"}):
            mgr = DatabaseManager()
            assert mgr.db_type == "sqlite"

    def test_accepts_postgres_url(self):
        from database import DatabaseManager
        with patch.dict("os.environ", {
            "DATABASE_URL": "postgresql://user:pass@localhost:5432/db",
            "DB_TYPE": "postgres",
        }):
            mgr = DatabaseManager()
            assert mgr.db_type == "postgres"

    def test_accepts_mysql_url(self):
        from database import DatabaseManager
        with patch.dict("os.environ", {
            "DATABASE_URL": "mysql://user:pass@localhost:3306/db",
            "DB_TYPE": "mysql",
        }):
            mgr = DatabaseManager()
            assert mgr.db_type == "mysql"

    def test_default_db_type_is_postgres(self):
        from database import DatabaseManager
        with patch.dict("os.environ", {
            "DATABASE_URL": "postgresql://user:pass@localhost:5432/db",
        }, clear=False):
            # Remove DB_TYPE if present to test default
            import os
            os.environ.pop("DB_TYPE", None)
            mgr = DatabaseManager()
            assert mgr.db_type == "postgres"

    def test_pydal_uri_is_set_on_init(self):
        from database import DatabaseManager
        with patch.dict("os.environ", {
            "DATABASE_URL": "postgresql://user:pass@localhost:5432/mydb",
            "DB_TYPE": "postgres",
        }):
            mgr = DatabaseManager()
            assert mgr.pydal_uri is not None
            assert mgr.pydal_uri.startswith("postgres://")


# ---------------------------------------------------------------------------
# health_check
# ---------------------------------------------------------------------------

class TestHealthCheck:
    """Tests for DatabaseManager.health_check (static method)."""

    def setup_method(self):
        from database import DatabaseManager
        self.health_check = DatabaseManager.health_check

    def test_returns_true_when_query_succeeds(self):
        db = MagicMock()
        db.return_value.count.return_value = 5
        # db(db.users).count() pattern
        result = self.health_check(db)
        assert result is True

    def test_returns_false_when_query_raises(self):
        db = MagicMock()
        db.return_value.count.side_effect = Exception("connection refused")
        result = self.health_check(db)
        assert result is False

    def test_calls_count_on_users_table(self):
        db = MagicMock()
        db.return_value.count.return_value = 0
        self.health_check(db)
        db.return_value.count.assert_called_once()


# ---------------------------------------------------------------------------
# close / reset_connection
# ---------------------------------------------------------------------------

class TestCloseAndReset:
    """Tests for close() and reset_connection()."""

    def _make_manager(self):
        from database import DatabaseManager
        with patch.dict("os.environ", {
            "DATABASE_URL": "sqlite:///test.db",
            "DB_TYPE": "sqlite",
        }):
            return DatabaseManager()

    def test_close_calls_db_close_when_connection_exists(self):
        mgr = self._make_manager()
        mock_db = MagicMock()
        mgr._thread_local.db = mock_db

        mgr.close()

        mock_db.close.assert_called_once()
        assert mgr._thread_local.db is None

    def test_close_is_safe_when_no_connection(self):
        mgr = self._make_manager()
        # Ensure no db in thread-local
        if hasattr(mgr._thread_local, "db"):
            delattr(mgr._thread_local, "db")
        # Should not raise
        mgr.close()

    def test_reset_connection_clears_thread_local(self):
        mgr = self._make_manager()
        mock_db = MagicMock()
        mgr._thread_local.db = mock_db

        mgr.reset_connection()

        assert not hasattr(mgr._thread_local, "db")

    def test_close_handles_error_gracefully(self):
        mgr = self._make_manager()
        mock_db = MagicMock()
        mock_db.close.side_effect = Exception("close failed")
        mgr._thread_local.db = mock_db
        # Should not raise
        mgr.close()


# ---------------------------------------------------------------------------
# get_db_manager singleton
# ---------------------------------------------------------------------------

class TestGetDbManagerSingleton:
    """Tests for module-level get_db_manager() singleton behaviour."""

    def test_returns_same_instance_on_repeated_calls(self):
        import database as db_module
        # Reset singleton before test
        original = db_module._db_manager
        db_module._db_manager = None
        try:
            with patch.dict("os.environ", {
                "DATABASE_URL": "sqlite:///test.db",
                "DB_TYPE": "sqlite",
            }):
                from database import get_db_manager
                mgr1 = get_db_manager()
                mgr2 = get_db_manager()
                assert mgr1 is mgr2
        finally:
            db_module._db_manager = original

    def test_singleton_is_database_manager_instance(self):
        import database as db_module
        from database import DatabaseManager, get_db_manager
        original = db_module._db_manager
        db_module._db_manager = None
        try:
            with patch.dict("os.environ", {
                "DATABASE_URL": "sqlite:///test.db",
                "DB_TYPE": "sqlite",
            }):
                mgr = get_db_manager()
                assert isinstance(mgr, DatabaseManager)
        finally:
            db_module._db_manager = original


# ---------------------------------------------------------------------------
# get_db convenience function
# ---------------------------------------------------------------------------

class TestGetDb:
    """Tests for module-level get_db() convenience function."""

    def test_get_db_returns_pydal_connection(self):
        import database as db_module
        from database import get_db
        mock_mgr = MagicMock()
        mock_dal = MagicMock()
        mock_mgr.get_pydal_connection.return_value = mock_dal

        original = db_module._db_manager
        db_module._db_manager = mock_mgr
        try:
            result = get_db()
            assert result is mock_dal
        finally:
            db_module._db_manager = original
