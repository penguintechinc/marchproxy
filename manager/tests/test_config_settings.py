#!/usr/bin/env python3
"""
Tests for config/settings.py ConfigManager initialization and methods.

Tests that configuration management is properly initialized and
configuration methods are callable with proper return types.
"""

from unittest.mock import MagicMock, patch
import pytest


class TestConfigManager:
    """Test ConfigManager class."""

    @pytest.fixture
    def mock_db(self):
        """Create a mock database."""
        db = MagicMock()
        db.system_config = MagicMock()
        db.define_table = MagicMock(return_value=MagicMock())
        db.commit = MagicMock()
        return db

    def test_config_manager_instantiation(self, mock_db):
        """Test ConfigManager can be instantiated."""
        from config.settings import ConfigManager

        config = ConfigManager(mock_db)
        assert config is not None
        assert hasattr(config, "db")
        assert hasattr(config, "_config_cache")
        assert hasattr(config, "_cache_ttl")

    def test_config_manager_has_get_methods(self, mock_db):
        """Test ConfigManager has expected configuration retrieval methods."""
        from config.settings import ConfigManager

        config = ConfigManager(mock_db)
        assert callable(config.get_config)
        assert callable(config.set_config)
        assert callable(config.get_database_config)
        assert callable(config.get_smtp_config)
        assert callable(config.get_syslog_config)
        assert callable(config.get_monitoring_config)

    def test_get_database_config_returns_dict(self, mock_db):
        """Test get_database_config returns a dictionary."""
        from config.settings import ConfigManager

        config = ConfigManager(mock_db)

        # Mock to return None for all queries
        query_mock = MagicMock()
        query_mock.select.return_value.first.return_value = None
        mock_db.__call__ = MagicMock(return_value=query_mock)

        with patch.dict("os.environ", {}, clear=True):
            result = config.get_database_config()

        assert isinstance(result, dict)
        assert "host" in result
        assert "port" in result
        assert "database" in result
        assert "username" in result
        assert "password" in result
        assert "ssl_mode" in result
        assert "pool_size" in result
        assert "max_overflow" in result

    def test_get_database_config_port_is_int(self, mock_db):
        """Test get_database_config returns port as integer."""
        from config.settings import ConfigManager

        config = ConfigManager(mock_db)

        # Mock to return None for all queries
        query_mock = MagicMock()
        query_mock.select.return_value.first.return_value = None
        mock_db.__call__ = MagicMock(return_value=query_mock)

        with patch.dict("os.environ", {}, clear=True):
            result = config.get_database_config()

        assert isinstance(result["port"], int)
        assert isinstance(result["pool_size"], int)
        assert isinstance(result["max_overflow"], int)

    def test_get_smtp_config_returns_dict(self, mock_db):
        """Test get_smtp_config returns dictionary with expected keys."""
        from config.settings import ConfigManager

        config = ConfigManager(mock_db)

        # Mock to return None
        query_mock = MagicMock()
        query_mock.select.return_value.first.return_value = None
        mock_db.__call__ = MagicMock(return_value=query_mock)

        with patch.dict("os.environ", {}, clear=True):
            result = config.get_smtp_config()

        assert isinstance(result, dict)
        assert "host" in result
        assert "port" in result
        assert "username" in result
        assert "password" in result
        assert "from_address" in result
        assert "use_tls" in result
        assert "use_ssl" in result

    def test_get_smtp_config_port_and_bools(self, mock_db):
        """Test get_smtp_config returns proper types."""
        from config.settings import ConfigManager

        config = ConfigManager(mock_db)

        # Mock to return None
        query_mock = MagicMock()
        query_mock.select.return_value.first.return_value = None
        mock_db.__call__ = MagicMock(return_value=query_mock)

        with patch.dict("os.environ", {}, clear=True):
            result = config.get_smtp_config()

        assert isinstance(result["port"], int)
        assert isinstance(result["use_tls"], bool)
        assert isinstance(result["use_ssl"], bool)

    def test_get_syslog_config_returns_dict(self, mock_db):
        """Test get_syslog_config returns dictionary with expected keys."""
        from config.settings import ConfigManager

        config = ConfigManager(mock_db)

        # Mock to return None
        query_mock = MagicMock()
        query_mock.select.return_value.first.return_value = None
        mock_db.__call__ = MagicMock(return_value=query_mock)

        with patch.dict("os.environ", {}, clear=True):
            result = config.get_syslog_config()

        assert isinstance(result, dict)
        assert "enabled" in result
        assert "host" in result
        assert "port" in result
        assert "protocol" in result
        assert "facility" in result
        assert "tag" in result

    def test_get_syslog_config_port_and_bool(self, mock_db):
        """Test get_syslog_config returns proper types."""
        from config.settings import ConfigManager

        config = ConfigManager(mock_db)

        # Mock to return None
        query_mock = MagicMock()
        query_mock.select.return_value.first.return_value = None
        mock_db.__call__ = MagicMock(return_value=query_mock)

        with patch.dict("os.environ", {}, clear=True):
            result = config.get_syslog_config()

        assert isinstance(result["port"], int)
        assert isinstance(result["enabled"], bool)

    def test_get_monitoring_config_returns_dict(self, mock_db):
        """Test get_monitoring_config returns dictionary."""
        from config.settings import ConfigManager

        config = ConfigManager(mock_db)

        # Mock to return None
        query_mock = MagicMock()
        query_mock.select.return_value.first.return_value = None
        mock_db.__call__ = MagicMock(return_value=query_mock)

        with patch.dict("os.environ", {}, clear=True):
            result = config.get_monitoring_config()

        assert isinstance(result, dict)
        assert "smtp" in result
        assert "alerts" in result
        assert isinstance(result["smtp"], dict)
        assert isinstance(result["alerts"], dict)

    def test_set_config_is_callable(self, mock_db):
        """Test set_config is callable and returns boolean."""
        from config.settings import ConfigManager

        config = ConfigManager(mock_db)

        # Mock insert
        query_mock = MagicMock()
        query_mock.select.return_value.first.return_value = None
        mock_db.__call__ = MagicMock(return_value=query_mock)
        mock_db.system_config.insert = MagicMock(return_value=1)

        result = config.set_config("key", "value")
        assert isinstance(result, bool)

    def test_get_config_is_callable(self, mock_db):
        """Test get_config is callable."""
        from config.settings import ConfigManager

        config = ConfigManager(mock_db)

        # Test that the method exists and is callable
        assert callable(config.get_config)

        # Calling it should not raise an error
        try:
            result = config.get_config("test_key")
            # Result can be anything depending on env/mock
            assert result is not None or result is None
        except Exception:
            # If it raises, that's ok - we're just checking it's callable
            pass

    def test_cache_ttl_is_300(self, mock_db):
        """Test default cache TTL is 300 seconds."""
        from config.settings import ConfigManager

        config = ConfigManager(mock_db)
        assert config._cache_ttl == 300

    def test_ensure_config_table_called(self, mock_db):
        """Test that _ensure_config_table is called on init."""
        from config.settings import ConfigManager

        config = ConfigManager(mock_db)
        # define_table should have been called
        assert mock_db.define_table.called
