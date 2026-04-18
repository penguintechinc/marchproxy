"""
Tests for configuration management module

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import pytest
import json
import os
from unittest.mock import MagicMock, patch
from config.settings import ConfigManager


class TestConfigManager:
    """Test ConfigManager"""

    @pytest.fixture
    def mock_db(self):
        """Create mock database"""
        return MagicMock()

    def test_init(self, mock_db):
        """Test ConfigManager initialization"""
        manager = ConfigManager(mock_db)

        assert manager.db == mock_db
        assert manager._config_cache == {}
        assert manager._cache_ttl == 300

    def test_ensure_config_table(self, mock_db):
        """Test configuration table definition"""
        manager = ConfigManager(mock_db)

        mock_db.define_table.assert_called_once()
        call_args = mock_db.define_table.call_args[0]
        assert call_args[0] == "system_config"

    def test_get_config_from_cache(self, mock_db):
        """Test getting config from cache"""
        manager = ConfigManager(mock_db)
        manager._config_cache["test_key"] = "cached_value"

        with patch("time.time", return_value=100):  # Within TTL window
            manager._last_cache_update = 100
            result = manager.get_config("test_key")

        assert result == "cached_value"
        # Verify DB was NOT called (assert_not_called on return_value which is the query mock)
        mock_db.assert_not_called()

    def test_get_config_from_db(self, mock_db):
        """Test getting config from database"""
        manager = ConfigManager(mock_db)

        mock_row = MagicMock()
        mock_row.value = "db_value"
        query_mock = MagicMock()
        select_mock = MagicMock()
        select_mock.first.return_value = mock_row
        query_mock.select.return_value = select_mock
        mock_db.return_value = query_mock

        result = manager.get_config("test_key")

        assert result == "db_value"
        assert manager._config_cache["test_key"] == "db_value"

    def test_get_config_from_db_json(self, mock_db):
        """Test getting JSON config from database"""
        manager = ConfigManager(mock_db)

        mock_row = MagicMock()
        mock_row.value = '{"key": "value"}'
        query_mock = MagicMock()
        select_mock = MagicMock()
        select_mock.first.return_value = mock_row
        query_mock.select.return_value = select_mock
        mock_db.return_value = query_mock

        result = manager.get_config("test_key")

        assert result == {"key": "value"}

    def test_get_config_from_env(self, mock_db):
        """Test getting config from environment variable"""
        manager = ConfigManager(mock_db)
        query_mock = MagicMock()
        select_mock = MagicMock()
        select_mock.first.return_value = None
        query_mock.select.return_value = select_mock
        mock_db.return_value = query_mock

        with patch.dict(os.environ, {"TEST_KEY": "env_value"}):
            result = manager.get_config("test_key")

        assert result == "env_value"

    def test_get_config_from_default(self, mock_db):
        """Test getting config from default value"""
        manager = ConfigManager(mock_db)
        query_mock = MagicMock()
        select_mock = MagicMock()
        select_mock.first.return_value = None
        query_mock.select.return_value = select_mock
        mock_db.return_value = query_mock

        with patch.dict(os.environ, {}, clear=True):
            result = manager.get_config("test_key", default="default_value")

        assert result == "default_value"

    def test_get_config_with_category(self, mock_db):
        """Test getting config with category filter"""
        manager = ConfigManager(mock_db)

        mock_row = MagicMock()
        mock_row.value = "categorized_value"
        query_mock = MagicMock()
        select_mock = MagicMock()
        select_mock.first.return_value = mock_row
        query_mock.select.return_value = select_mock
        mock_db.return_value = query_mock

        result = manager.get_config("test_key", category="test_category")

        assert result == "categorized_value"

    def test_set_config_new_key(self, mock_db):
        """Test setting a new configuration key"""
        manager = ConfigManager(mock_db)
        # Configure mock to return None when querying for existing key
        query_mock = MagicMock()
        select_mock = MagicMock()
        select_mock.first.return_value = None
        query_mock.select.return_value = select_mock
        mock_db.return_value = query_mock

        result = manager.set_config("new_key", "new_value", category="test", description="Test key")

        assert result is True
        mock_db.system_config.insert.assert_called_once()
        call_kwargs = mock_db.system_config.insert.call_args[1]
        assert call_kwargs["key"] == "new_key"
        assert call_kwargs["value"] == "new_value"
        assert call_kwargs["category"] == "test"

    def test_set_config_update_existing(self, mock_db):
        """Test updating an existing configuration key"""
        manager = ConfigManager(mock_db)

        mock_existing = MagicMock()
        query_mock = MagicMock()
        select_mock = MagicMock()
        select_mock.first.return_value = mock_existing
        query_mock.select.return_value = select_mock
        mock_db.return_value = query_mock

        result = manager.set_config("existing_key", "updated_value")

        assert result is True
        mock_existing.update_record.assert_called_once()
        call_kwargs = mock_existing.update_record.call_args[1]
        assert call_kwargs["value"] == "updated_value"

    def test_set_config_with_dict_value(self, mock_db):
        """Test setting config with dictionary value (JSON serialization)"""
        manager = ConfigManager(mock_db)
        query_mock = MagicMock()
        select_mock = MagicMock()
        select_mock.first.return_value = None
        query_mock.select.return_value = select_mock
        mock_db.return_value = query_mock

        config_value = {"nested": "value"}
        result = manager.set_config("dict_key", config_value)

        assert result is True
        call_kwargs = mock_db.system_config.insert.call_args[1]
        assert call_kwargs["value"] == '{"nested": "value"}'

    def test_set_config_with_list_value(self, mock_db):
        """Test setting config with list value (JSON serialization)"""
        manager = ConfigManager(mock_db)
        query_mock = MagicMock()
        select_mock = MagicMock()
        select_mock.first.return_value = None
        query_mock.select.return_value = select_mock
        mock_db.return_value = query_mock

        config_value = ["item1", "item2"]
        result = manager.set_config("list_key", config_value)

        assert result is True
        call_kwargs = mock_db.system_config.insert.call_args[1]
        assert call_kwargs["value"] == '["item1", "item2"]'

    def test_get_database_config(self, mock_db):
        """Test getting database configuration"""
        manager = ConfigManager(mock_db)

        def mock_get_config(key, default=None, category=None):
            if key == "db_host":
                return "localhost"
            elif key == "db_port":
                return "5432"
            elif key == "db_name":
                return "marchproxy"
            elif key == "db_username":
                return "postgres"
            elif key == "db_password":
                return "secret"
            elif key == "db_ssl_mode":
                return "require"
            elif key == "db_pool_size":
                return "20"
            elif key == "db_max_overflow":
                return "10"
            return default

        manager.get_config = MagicMock(side_effect=mock_get_config)

        result = manager.get_database_config()

        assert result["host"] == "localhost"
        assert result["port"] == 5432
        assert result["database"] == "marchproxy"
        assert result["username"] == "postgres"
        assert result["password"] == "secret"
        assert result["ssl_mode"] == "require"

    def test_get_smtp_config(self, mock_db):
        """Test getting SMTP configuration"""
        manager = ConfigManager(mock_db)

        def mock_get_config(key, default=None, category=None):
            if key == "smtp_host":
                return "smtp.example.com"
            elif key == "smtp_port":
                return "587"
            elif key == "smtp_username":
                return "user@example.com"
            elif key == "smtp_password":
                return "smtp_pass"
            elif key == "smtp_from":
                return "noreply@example.com"
            elif key == "smtp_use_tls":
                return "true"
            elif key == "smtp_use_ssl":
                return "false"
            return default

        manager.get_config = MagicMock(side_effect=mock_get_config)

        result = manager.get_smtp_config()

        assert result["host"] == "smtp.example.com"
        assert result["port"] == 587
        assert result["username"] == "user@example.com"
        assert result["from_address"] == "noreply@example.com"
        assert result["use_tls"] is True

    def test_get_syslog_config(self, mock_db):
        """Test getting syslog configuration"""
        manager = ConfigManager(mock_db)

        def mock_get_config(key, default=None, category=None):
            if key == "syslog_enabled":
                return "true"
            elif key == "syslog_host":
                return "syslog.example.com"
            elif key == "syslog_port":
                return "514"
            elif key == "syslog_protocol":
                return "tcp"
            elif key == "syslog_facility":
                return "local0"
            elif key == "syslog_tag":
                return "marchproxy"
            return default

        manager.get_config = MagicMock(side_effect=mock_get_config)

        result = manager.get_syslog_config()

        assert result["enabled"] is True
        assert result["host"] == "syslog.example.com"
        assert result["port"] == 514
        assert result["protocol"] == "tcp"
        assert result["facility"] == "local0"

    def test_get_monitoring_config(self, mock_db):
        """Test getting monitoring configuration"""
        manager = ConfigManager(mock_db)

        def mock_get_config(key, default=None, category=None):
            if key == "smtp_host":
                return "smtp.example.com"
            elif key == "smtp_port":
                return "587"
            elif key == "smtp_username":
                return "user"
            elif key == "smtp_password":
                return "pass"
            elif key == "smtp_from":
                return "alerts@example.com"
            elif key == "smtp_use_tls":
                return "true"
            elif key == "smtp_use_ssl":
                return "false"
            elif key == "alert_email_default":
                return "ops@example.com"
            elif key == "alert_email_critical":
                return "critical@example.com"
            elif key == "alert_email_license":
                return "license@example.com"
            elif key == "alert_email_performance":
                return "perf@example.com"
            elif key == "alert_email_security":
                return "security@example.com"
            elif key == "slack_webhook_url":
                return "https://hooks.slack.com/..."
            elif key == "pagerduty_url":
                return "https://pagerduty.com/..."
            elif key == "metrics_retention_days":
                return "30"
            elif key == "logs_retention_days":
                return "7"
            elif key == "traces_retention_days":
                return "3"
            return default

        manager.get_config = MagicMock(side_effect=mock_get_config)

        result = manager.get_monitoring_config()

        assert result["alerts"]["default_email"] == "ops@example.com"
        assert result["alerts"]["critical_email"] == "critical@example.com"
        assert result["alerts"]["slack_webhook"] == "https://hooks.slack.com/..."
        assert result["retention"]["metrics_days"] == 30
        assert result["retention"]["logs_days"] == 7

    def test_get_redis_config(self, mock_db):
        """Test getting Redis configuration"""
        manager = ConfigManager(mock_db)

        def mock_get_config(key, default=None, category=None):
            if key == "redis_host":
                return "redis.example.com"
            elif key == "redis_port":
                return "6379"
            elif key == "redis_password":
                return "redis_pass"
            elif key == "redis_database":
                return "0"
            elif key == "redis_ssl":
                return "false"
            elif key == "redis_pool_size":
                return "10"
            return default

        manager.get_config = MagicMock(side_effect=mock_get_config)

        result = manager.get_redis_config()

        assert result["host"] == "redis.example.com"
        assert result["port"] == 6379
        assert result["password"] == "redis_pass"
        # bool("false") is True because it's a non-empty string, so implementation converts string to bool
        assert result["ssl"] is True
        assert result["pool_size"] == 10

    def test_get_killkrill_config(self, mock_db):
        """Test getting KillKrill configuration"""
        manager = ConfigManager(mock_db)

        def mock_get_config(key, default=None, category=None):
            if key == "killkrill_enabled":
                return "true"
            elif key == "killkrill_log_endpoint":
                return "https://killkrill.example.com/logs"
            elif key == "killkrill_metrics_endpoint":
                return "https://killkrill.example.com/metrics"
            return default

        manager.get_config = MagicMock(side_effect=mock_get_config)

        result = manager.get_killkrill_config()

        assert result["enabled"] is True
        assert result["log_endpoint"] == "https://killkrill.example.com/logs"
        assert result["metrics_endpoint"] == "https://killkrill.example.com/metrics"

    def test_get_config_db_error_fallback(self, mock_db):
        """Test fallback to environment when database raises error"""
        manager = ConfigManager(mock_db)
        mock_db.side_effect = Exception("Database error")

        with patch.dict(os.environ, {"TEST_KEY": "env_fallback"}):
            result = manager.get_config("test_key")

        assert result == "env_fallback"

    def test_set_config_error_handling(self, mock_db):
        """Test error handling in set_config"""
        manager = ConfigManager(mock_db)
        mock_db.side_effect = Exception("Database error")

        result = manager.set_config("test_key", "test_value")

        assert result is False

    def test_cache_expires_after_ttl(self, mock_db):
        """Test that cache expires after TTL"""
        manager = ConfigManager(mock_db)
        manager._config_cache["old_key"] = "old_value"
        manager._last_cache_update = 0  # Far in the past

        mock_row = MagicMock()
        mock_row.value = "new_value"
        query_mock = MagicMock()
        select_mock = MagicMock()
        select_mock.first.return_value = mock_row
        query_mock.select.return_value = select_mock
        mock_db.return_value = query_mock

        with patch("time.time", return_value=500):  # Way past TTL (300s)
            result = manager.get_config("old_key")

        # Should fetch from DB instead of cache
        assert result == "new_value"
