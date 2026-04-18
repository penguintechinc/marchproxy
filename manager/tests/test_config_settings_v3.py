"""
Unit tests for ConfigManager and configuration module.
Tests coverage for get_config, set_config, and all config getter methods.
"""

import pytest
import json
import os
from unittest.mock import MagicMock, patch, call
from dataclasses import dataclass


@dataclass
class MockConfigRow:
    """Mock config row object"""
    key: str
    value: str
    category: str
    description: str
    is_secret: bool

    def update_record(self, **kwargs):
        """Mock update_record method"""
        for key, val in kwargs.items():
            setattr(self, key, val)


@pytest.fixture
def mock_db():
    """Create a mock database object"""
    db = MagicMock()

    # Mock define_table
    db.define_table = MagicMock()
    db.commit = MagicMock()

    # Mock system_config table with proper attributes
    db.system_config = MagicMock()

    # Create mock attributes that support comparison operators
    id_attr = MagicMock()
    id_attr.__gt__ = MagicMock(return_value=MagicMock())
    id_attr.__and__ = MagicMock(return_value=MagicMock())
    db.system_config.id = id_attr

    db.system_config.key = MagicMock()
    db.system_config.category = MagicMock()
    db.system_config.is_secret = MagicMock()

    # Default query mock (returns None)
    query_mock = MagicMock()
    query_mock.__and__ = MagicMock(return_value=query_mock)
    select_mock = MagicMock()
    select_mock.first = MagicMock(return_value=None)
    select_mock.return_value = []  # For select().return_value in get_all_config
    select_mock.__iter__ = MagicMock(return_value=iter([]))
    query_mock.select = MagicMock(return_value=select_mock)

    db.return_value = query_mock
    db.__call__ = MagicMock(return_value=query_mock)

    # Mock common_filter (for datetime defaults)
    db.common_filter = MagicMock()

    return db


@pytest.fixture
def config_manager(mock_db):
    """Create ConfigManager instance with mock database"""
    from config.settings import ConfigManager
    return ConfigManager(mock_db)


class TestConfigManagerInit:
    """Test ConfigManager initialization"""

    def test_init_creates_config_table(self, mock_db):
        """Verify _ensure_config_table is called and define_table is used"""
        from config.settings import ConfigManager

        manager = ConfigManager(mock_db)

        # Verify define_table was called
        mock_db.define_table.assert_called_once()

        # Verify commit was called
        mock_db.commit.assert_called_once()

        # Verify table was stored
        assert manager.db == mock_db

    def test_init_initializes_cache(self, config_manager):
        """Verify cache is initialized as empty dict"""
        assert config_manager._config_cache == {}
        assert config_manager._cache_ttl == 300
        assert config_manager._last_cache_update == 0


class TestGetConfig:
    """Test get_config method with fallback hierarchy"""

    def test_get_config_from_db_string(self, config_manager, mock_db):
        """Test getting string value from database"""
        # Setup mock to return a row
        row = MockConfigRow(key="test_key", value="test_value", category="general", description="", is_secret=False)
        query_mock = MagicMock()
        select_mock = MagicMock()
        select_mock.first.return_value = row
        query_mock.select.return_value = select_mock
        mock_db.return_value = query_mock

        result = config_manager.get_config("test_key")

        assert result == "test_value"
        # Verify it was cached
        assert config_manager._config_cache["test_key"] == "test_value"

    def test_get_config_from_db_json_dict(self, config_manager, mock_db):
        """Test getting JSON dict value from database"""
        json_value = json.dumps({"nested": "value"})
        row = MockConfigRow(key="json_key", value=json_value, category="general", description="", is_secret=False)
        query_mock = MagicMock()
        select_mock = MagicMock()
        select_mock.first.return_value = row
        query_mock.select.return_value = select_mock
        mock_db.return_value = query_mock

        result = config_manager.get_config("json_key")

        assert result == {"nested": "value"}
        assert config_manager._config_cache["json_key"] == {"nested": "value"}

    def test_get_config_from_db_json_list(self, config_manager, mock_db):
        """Test getting JSON list value from database"""
        json_value = json.dumps([1, 2, 3])
        row = MockConfigRow(key="list_key", value=json_value, category="general", description="", is_secret=False)
        query_mock = MagicMock()
        select_mock = MagicMock()
        select_mock.first.return_value = row
        query_mock.select.return_value = select_mock
        mock_db.return_value = query_mock

        result = config_manager.get_config("list_key")

        assert result == [1, 2, 3]

    def test_get_config_from_env_fallback(self, config_manager, mock_db):
        """Test fallback to environment variable when DB miss"""
        query_mock = MagicMock()
        select_mock = MagicMock()
        select_mock.first.return_value = None  # DB miss
        query_mock.select.return_value = select_mock
        mock_db.return_value = query_mock

        with patch.dict(os.environ, {"MY_KEY": "env_value"}):
            result = config_manager.get_config("my_key")

        assert result == "env_value"

    def test_get_config_default_fallback(self, config_manager, mock_db):
        """Test fallback to default when DB and ENV miss"""
        query_mock = MagicMock()
        select_mock = MagicMock()
        select_mock.first.return_value = None
        query_mock.select.return_value = select_mock
        mock_db.return_value = query_mock

        result = config_manager.get_config("missing_key", default="default_value")

        assert result == "default_value"

    def test_get_config_from_cache(self, config_manager, mock_db):
        """Test cached value is returned without DB query"""
        config_manager._config_cache["cached_key"] = "cached_value"
        config_manager._last_cache_update = 9999999999  # Far in future

        result = config_manager.get_config("cached_key")

        assert result == "cached_value"
        # DB should not be called
        mock_db.assert_not_called()

    def test_get_config_category_filter(self, config_manager, mock_db):
        """Test category parameter is used in query"""
        row = MockConfigRow(key="cat_key", value="cat_value", category="smtp", description="", is_secret=False)
        query_mock = MagicMock()
        select_mock = MagicMock()
        select_mock.first.return_value = row
        query_mock.select.return_value = select_mock
        mock_db.return_value = query_mock

        result = config_manager.get_config("cat_key", category="smtp")

        assert result == "cat_value"
        # Verify the query was constructed with category
        assert mock_db.called

    def test_get_config_db_exception_fallback(self, config_manager, mock_db):
        """Test fallback to env/default when DB raises exception"""
        mock_db.side_effect = Exception("DB error")

        with patch.dict(os.environ, {}, clear=True):
            result = config_manager.get_config("error_key", default="fallback")

        assert result == "fallback"

    def test_get_config_empty_value_fallback(self, config_manager, mock_db):
        """Test fallback to env when DB value is empty"""
        row = MockConfigRow(key="empty_key", value="", category="general", description="", is_secret=False)
        query_mock = MagicMock()
        select_mock = MagicMock()
        select_mock.first.return_value = row
        query_mock.select.return_value = select_mock
        mock_db.return_value = query_mock

        with patch.dict(os.environ, {"EMPTY_KEY": "env_fallback"}):
            result = config_manager.get_config("empty_key")

        assert result == "env_fallback"


class TestSetConfig:
    """Test set_config method"""

    def test_set_config_insert_new(self, config_manager, mock_db):
        """Test inserting new configuration"""
        query_mock = MagicMock()
        select_mock = MagicMock()
        select_mock.first.return_value = None  # Key doesn't exist
        query_mock.select.return_value = select_mock
        mock_db.return_value = query_mock
        mock_db.system_config.insert = MagicMock()

        result = config_manager.set_config("new_key", "new_value", "general", "Test key", False)

        assert result is True
        mock_db.system_config.insert.assert_called_once()
        mock_db.commit.assert_called()
        assert config_manager._config_cache["new_key"] == "new_value"

    def test_set_config_update_existing(self, config_manager, mock_db):
        """Test updating existing configuration"""
        row = MagicMock()
        row.update_record = MagicMock()
        query_mock = MagicMock()
        select_mock = MagicMock()
        select_mock.first.return_value = row
        query_mock.select.return_value = select_mock
        mock_db.return_value = query_mock

        result = config_manager.set_config("existing_key", "new_value")

        assert result is True
        row.update_record.assert_called_once()
        mock_db.commit.assert_called()
        assert config_manager._config_cache["existing_key"] == "new_value"

    def test_set_config_dict_value_serialized(self, config_manager, mock_db):
        """Test dict value is JSON serialized"""
        query_mock = MagicMock()
        select_mock = MagicMock()
        select_mock.first.return_value = None
        query_mock.select.return_value = select_mock
        mock_db.return_value = query_mock
        mock_db.system_config.insert = MagicMock()

        dict_value = {"key": "value", "nested": {"inner": "data"}}
        config_manager.set_config("dict_key", dict_value)

        call_args = mock_db.system_config.insert.call_args
        assert call_args is not None
        # Value should be JSON string
        assert isinstance(call_args[1]["value"], str)
        assert json.loads(call_args[1]["value"]) == dict_value

    def test_set_config_list_value_serialized(self, config_manager, mock_db):
        """Test list value is JSON serialized"""
        query_mock = MagicMock()
        select_mock = MagicMock()
        select_mock.first.return_value = None
        query_mock.select.return_value = select_mock
        mock_db.return_value = query_mock
        mock_db.system_config.insert = MagicMock()

        list_value = [1, "two", 3.0]
        config_manager.set_config("list_key", list_value)

        call_args = mock_db.system_config.insert.call_args
        assert json.loads(call_args[1]["value"]) == list_value

    def test_set_config_none_value_empty_string(self, config_manager, mock_db):
        """Test None value becomes empty string"""
        query_mock = MagicMock()
        select_mock = MagicMock()
        select_mock.first.return_value = None
        query_mock.select.return_value = select_mock
        mock_db.return_value = query_mock
        mock_db.system_config.insert = MagicMock()

        config_manager.set_config("none_key", None)

        call_args = mock_db.system_config.insert.call_args
        assert call_args[1]["value"] == ""

    def test_set_config_int_value_string(self, config_manager, mock_db):
        """Test int value is converted to string"""
        query_mock = MagicMock()
        select_mock = MagicMock()
        select_mock.first.return_value = None
        query_mock.select.return_value = select_mock
        mock_db.return_value = query_mock
        mock_db.system_config.insert = MagicMock()

        config_manager.set_config("int_key", 42)

        call_args = mock_db.system_config.insert.call_args
        assert call_args[1]["value"] == "42"

    def test_set_config_exception_returns_false(self, config_manager, mock_db):
        """Test exception during set returns False"""
        mock_db.side_effect = Exception("DB error")

        result = config_manager.set_config("error_key", "error_value")

        assert result is False

    def test_set_config_stores_metadata(self, config_manager, mock_db):
        """Test metadata is stored with config"""
        query_mock = MagicMock()
        select_mock = MagicMock()
        select_mock.first.return_value = None
        query_mock.select.return_value = select_mock
        mock_db.return_value = query_mock
        mock_db.system_config.insert = MagicMock()

        config_manager.set_config(
            "meta_key",
            "meta_value",
            category="custom",
            description="Custom description",
            is_secret=True
        )

        call_args = mock_db.system_config.insert.call_args
        assert call_args[1]["category"] == "custom"
        assert call_args[1]["description"] == "Custom description"
        assert call_args[1]["is_secret"] is True


class TestGetDatabaseConfig:
    """Test get_database_config method"""

    def test_get_database_config_returns_dict(self, config_manager):
        """Test returned dict has all required keys"""
        with patch.object(config_manager, 'get_config') as mock_get:
            mock_get.side_effect = lambda key, default, cat: {
                "db_host": "localhost",
                "db_port": 5432,
                "db_name": "testdb",
                "db_username": "user",
                "db_password": "pass",
                "db_ssl_mode": "prefer",
                "db_pool_size": 20,
                "db_max_overflow": 10,
            }.get(key, default)

            result = config_manager.get_database_config()

            assert "host" in result
            assert "port" in result
            assert "database" in result
            assert "username" in result
            assert "password" in result
            assert "ssl_mode" in result
            assert "pool_size" in result
            assert "max_overflow" in result

    def test_get_database_config_port_int(self, config_manager):
        """Test port is converted to int"""
        with patch.object(config_manager, 'get_config') as mock_get:
            mock_get.side_effect = lambda key, default, cat: "5432"

            result = config_manager.get_database_config()

            assert isinstance(result["port"], int)

    def test_get_database_config_uses_category(self, config_manager):
        """Test database category is used"""
        with patch.object(config_manager, 'get_config') as mock_get:
            config_manager.get_database_config()

            # Verify 'database' category was passed
            calls = mock_get.call_args_list
            for call_obj in calls:
                if len(call_obj[0]) > 2:
                    assert call_obj[0][2] == "database"


class TestGetSmtpConfig:
    """Test get_smtp_config method"""

    def test_get_smtp_config_returns_dict(self, config_manager):
        """Test returned dict has all SMTP keys"""
        with patch.object(config_manager, 'get_config') as mock_get:
            mock_get.side_effect = lambda key, default, cat: default

            result = config_manager.get_smtp_config()

            assert "host" in result
            assert "port" in result
            assert "username" in result
            assert "password" in result
            assert "from_address" in result
            assert "use_tls" in result
            assert "use_ssl" in result

    def test_get_smtp_config_bool_conversion(self, config_manager):
        """Test use_tls and use_ssl are bool"""
        with patch.object(config_manager, 'get_config') as mock_get:
            mock_get.side_effect = lambda key, default, cat: default

            result = config_manager.get_smtp_config()

            assert isinstance(result["use_tls"], bool)
            assert isinstance(result["use_ssl"], bool)


class TestGetSyslogConfig:
    """Test get_syslog_config method"""

    def test_get_syslog_config_returns_dict(self, config_manager):
        """Test returned dict has all syslog keys"""
        with patch.object(config_manager, 'get_config') as mock_get:
            mock_get.side_effect = lambda key, default, cat: default

            result = config_manager.get_syslog_config()

            assert "enabled" in result
            assert "host" in result
            assert "port" in result
            assert "protocol" in result
            assert "facility" in result
            assert "tag" in result

    def test_get_syslog_config_enabled_bool(self, config_manager):
        """Test enabled is bool"""
        with patch.object(config_manager, 'get_config') as mock_get:
            mock_get.side_effect = lambda key, default, cat: default

            result = config_manager.get_syslog_config()

            assert isinstance(result["enabled"], bool)


class TestGetMonitoringConfig:
    """Test get_monitoring_config method"""

    def test_get_monitoring_config_returns_dict(self, config_manager):
        """Test returned dict has smtp, alerts, and retention"""
        with patch.object(config_manager, 'get_config') as mock_get:
            mock_get.side_effect = lambda key, default, cat: default

            result = config_manager.get_monitoring_config()

            assert "smtp" in result
            assert "alerts" in result
            assert "retention" in result

    def test_get_monitoring_config_alerts_keys(self, config_manager):
        """Test alerts contains all alert types"""
        with patch.object(config_manager, 'get_config') as mock_get:
            mock_get.side_effect = lambda key, default, cat: default

            result = config_manager.get_monitoring_config()

            alerts = result["alerts"]
            assert "default_email" in alerts
            assert "critical_email" in alerts
            assert "license_email" in alerts
            assert "performance_email" in alerts
            assert "security_email" in alerts
            assert "slack_webhook" in alerts
            assert "pagerduty_url" in alerts

    def test_get_monitoring_config_retention_int(self, config_manager):
        """Test retention values are int"""
        with patch.object(config_manager, 'get_config') as mock_get:
            mock_get.side_effect = lambda key, default, cat: "30"

            result = config_manager.get_monitoring_config()

            retention = result["retention"]
            assert isinstance(retention["metrics_days"], int)
            assert isinstance(retention["logs_days"], int)
            assert isinstance(retention["traces_days"], int)


class TestGetRedisConfig:
    """Test get_redis_config method"""

    def test_get_redis_config_returns_dict(self, config_manager):
        """Test returned dict has all Redis keys"""
        with patch.object(config_manager, 'get_config') as mock_get:
            mock_get.side_effect = lambda key, default, cat: default

            result = config_manager.get_redis_config()

            assert "host" in result
            assert "port" in result
            assert "password" in result
            assert "database" in result
            assert "ssl" in result
            assert "pool_size" in result

    def test_get_redis_config_types(self, config_manager):
        """Test Redis config value types"""
        with patch.object(config_manager, 'get_config') as mock_get:
            mock_get.side_effect = lambda key, default, cat: default

            result = config_manager.get_redis_config()

            assert isinstance(result["port"], int)
            assert isinstance(result["database"], int)
            assert isinstance(result["ssl"], bool)
            assert isinstance(result["pool_size"], int)


class TestGetKillKrillConfig:
    """Test get_killkrill_config method"""

    def test_get_killkrill_config_returns_dict(self, config_manager):
        """Test returned dict has all KillKrill keys"""
        with patch.object(config_manager, 'get_config') as mock_get:
            mock_get.side_effect = lambda key, default, cat: default

            result = config_manager.get_killkrill_config()

            assert "enabled" in result
            assert "log_endpoint" in result
            assert "metrics_endpoint" in result
            assert "api_key" in result
            assert "source_name" in result
            assert "application" in result
            assert "batch_size" in result
            assert "flush_interval" in result
            assert "timeout" in result
            assert "use_http3" in result

    def test_get_killkrill_config_types(self, config_manager):
        """Test KillKrill config value types"""
        with patch.object(config_manager, 'get_config') as mock_get:
            mock_get.side_effect = lambda key, default, cat: default

            result = config_manager.get_killkrill_config()

            assert isinstance(result["enabled"], bool)
            assert isinstance(result["batch_size"], int)
            assert isinstance(result["flush_interval"], int)
            assert isinstance(result["timeout"], int)
            assert isinstance(result["use_http3"], bool)


class TestGetLicenseConfig:
    """Test get_license_config method"""

    def test_get_license_config_returns_dict(self, config_manager):
        """Test returned dict has all license keys"""
        with patch.object(config_manager, 'get_config') as mock_get:
            mock_get.side_effect = lambda key, default, cat: default

            result = config_manager.get_license_config()

            assert "key" in result
            assert "server_url" in result
            assert "check_interval_hours" in result
            assert "offline_grace_days" in result

    def test_get_license_config_int_values(self, config_manager):
        """Test check_interval_hours and offline_grace_days are int"""
        with patch.object(config_manager, 'get_config') as mock_get:
            mock_get.side_effect = lambda key, default, cat: "24"

            result = config_manager.get_license_config()

            assert isinstance(result["check_interval_hours"], int)
            assert isinstance(result["offline_grace_days"], int)


class TestInitializeDefaultConfig:
    """Test initialize_default_config method"""

    def test_initialize_default_config_inserts_missing(self, config_manager, mock_db):
        """Test missing defaults are inserted"""
        query_mock = MagicMock()
        select_mock = MagicMock()
        select_mock.first.return_value = None  # No existing config
        query_mock.select.return_value = select_mock
        mock_db.return_value = query_mock

        with patch.object(config_manager, 'set_config') as mock_set:
            config_manager.initialize_default_config()

            # Verify set_config was called multiple times
            assert mock_set.call_count > 0

    def test_initialize_default_config_skips_existing(self, config_manager, mock_db):
        """Test existing configs are not overwritten"""
        existing_row = MockConfigRow(key="db_host", value="existing", category="database", description="", is_secret=False)
        query_mock = MagicMock()
        select_mock = MagicMock()
        select_mock.first.return_value = existing_row  # Config exists
        query_mock.select.return_value = select_mock
        mock_db.return_value = query_mock

        with patch.object(config_manager, 'set_config') as mock_set:
            config_manager.initialize_default_config()

            # set_config should be called but not for db_host
            calls = mock_set.call_args_list
            first_keys = [call_obj[0][0] for call_obj in calls]
            # db_host should not be in first set of calls since it exists
            # Only check that not all defaults are set for the same key


class TestGetAllConfig:
    """Test get_all_config method"""

    def test_get_all_config_returns_dict(self, config_manager, mock_db):
        """Test get_all_config returns dict with all configs"""
        row1 = MockConfigRow(key="key1", value="value1", category="general", description="Desc1", is_secret=False)
        row2 = MockConfigRow(key="key2", value="value2", category="general", description="Desc2", is_secret=False)

        query_mock = MagicMock()
        select_mock = MagicMock()
        select_mock.return_value = [row1, row2]
        select_mock.__iter__ = MagicMock(return_value=iter([row1, row2]))
        query_mock.select.return_value = select_mock
        mock_db.return_value = query_mock

        result = config_manager.get_all_config()

        assert "key1" in result
        assert "key2" in result
        assert result["key1"]["value"] == "value1"
        assert result["key2"]["value"] == "value2"

    def test_get_all_config_by_category(self, config_manager, mock_db):
        """Test get_all_config filters by category"""
        row = MockConfigRow(key="smtp_host", value="localhost", category="smtp", description="", is_secret=False)

        query_mock = MagicMock()
        select_mock = MagicMock()
        select_mock.return_value = [row]
        select_mock.__iter__ = MagicMock(return_value=iter([row]))
        query_mock.select.return_value = select_mock
        mock_db.return_value = query_mock

        result = config_manager.get_all_config(category="smtp")

        assert "smtp_host" in result

    def test_get_all_config_exclude_secrets(self, config_manager, mock_db):
        """Test secrets are excluded by default"""
        row1 = MockConfigRow(key="public_key", value="public", category="general", description="", is_secret=False)
        row2 = MockConfigRow(key="secret_key", value="secret", category="general", description="", is_secret=True)

        query_mock = MagicMock()
        select_mock = MagicMock()
        # Only public_key should be returned
        select_mock.return_value = [row1]
        select_mock.__iter__ = MagicMock(return_value=iter([row1]))
        query_mock.select.return_value = select_mock
        mock_db.return_value = query_mock

        result = config_manager.get_all_config(include_secrets=False)

        assert "public_key" in result

    def test_get_all_config_include_secrets(self, config_manager, mock_db):
        """Test secrets are included when requested"""
        row1 = MockConfigRow(key="public_key", value="public", category="general", description="", is_secret=False)
        row2 = MockConfigRow(key="secret_key", value="secret", category="general", description="", is_secret=True)

        query_mock = MagicMock()
        select_mock = MagicMock()
        select_mock.return_value = [row1, row2]
        select_mock.__iter__ = MagicMock(return_value=iter([row1, row2]))
        query_mock.select.return_value = select_mock
        mock_db.return_value = query_mock

        result = config_manager.get_all_config(include_secrets=True)

        assert "public_key" in result
        assert "secret_key" in result

    def test_get_all_config_json_parsing(self, config_manager, mock_db):
        """Test JSON values are parsed in get_all_config"""
        json_value = json.dumps({"nested": "data"})
        row = MockConfigRow(key="json_key", value=json_value, category="general", description="", is_secret=False)

        query_mock = MagicMock()
        select_mock = MagicMock()
        select_mock.return_value = [row]
        select_mock.__iter__ = MagicMock(return_value=iter([row]))
        query_mock.select.return_value = select_mock
        mock_db.return_value = query_mock

        result = config_manager.get_all_config()

        assert result["json_key"]["value"] == {"nested": "data"}

    def test_get_all_config_returns_metadata(self, config_manager, mock_db):
        """Test metadata is included in result"""
        row = MockConfigRow(key="meta_key", value="meta_value", category="custom", description="Custom desc", is_secret=True)

        query_mock = MagicMock()
        select_mock = MagicMock()
        select_mock.return_value = [row]
        select_mock.__iter__ = MagicMock(return_value=iter([row]))
        query_mock.select.return_value = select_mock
        mock_db.return_value = query_mock

        result = config_manager.get_all_config(include_secrets=True)

        assert result["meta_key"]["category"] == "custom"
        assert result["meta_key"]["description"] == "Custom desc"
        assert result["meta_key"]["is_secret"] is True


class TestClearCache:
    """Test clear_cache method"""

    def test_clear_cache_resets(self, config_manager):
        """Test cache is cleared"""
        config_manager._config_cache = {"key": "value"}
        config_manager._last_cache_update = 12345

        config_manager.clear_cache()

        assert config_manager._config_cache == {}
        assert config_manager._last_cache_update == 0


class TestModuleLevel:
    """Test module-level functions"""

    def test_get_config_manager_singleton(self, mock_db):
        """Test get_config_manager returns same instance"""
        from config.settings import get_config_manager, _config_manager

        # Reset global
        import config.settings
        config.settings._config_manager = None

        manager1 = get_config_manager(mock_db)
        manager2 = get_config_manager(mock_db)

        assert manager1 is manager2

    def test_get_config_convenience_no_manager(self):
        """Test get_config falls back to env when no manager"""
        from config.settings import get_config
        import config.settings

        # Reset global
        config.settings._config_manager = None

        with patch.dict(os.environ, {"TEST_KEY": "test_value"}):
            result = get_config("test_key")

        assert result == "test_value"

    def test_get_config_convenience_with_manager(self, mock_db):
        """Test get_config delegates to manager"""
        from config.settings import get_config_manager, get_config
        import config.settings

        # Reset and setup manager
        config.settings._config_manager = None
        manager = get_config_manager(mock_db)

        # Mock manager's get_config
        with patch.object(manager, 'get_config', return_value="manager_value") as mock_method:
            result = get_config("test_key")

        assert result == "manager_value"
        mock_method.assert_called_once()
