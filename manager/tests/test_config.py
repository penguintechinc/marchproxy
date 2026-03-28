"""
Unit tests for manager/config/settings.py

Tests ConfigManager initialization, get/set config, database/Redis config
helpers, cache management, and default initialization.
No real database connections are used.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import os
from unittest.mock import MagicMock, call, patch

import pytest

from config.settings import ConfigManager


pytestmark = pytest.mark.unit


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _make_config_manager(mock_db: MagicMock) -> ConfigManager:
    """Instantiate ConfigManager with a mock DB that won't actually define tables."""
    with patch.object(ConfigManager, "_ensure_config_table"):
        mgr = ConfigManager(mock_db)
    # Expose db as attribute (already set in __init__)
    return mgr


# ---------------------------------------------------------------------------
# TestConfigManagerInit
# ---------------------------------------------------------------------------


class TestConfigManagerInit:
    def test_ensure_config_table_called_on_init(self, mock_db):
        with patch.object(ConfigManager, "_ensure_config_table") as mock_ensure:
            ConfigManager(mock_db)
        mock_ensure.assert_called_once()

    def test_cache_starts_empty(self, mock_db):
        mgr = _make_config_manager(mock_db)
        assert mgr._config_cache == {}

    def test_db_assigned(self, mock_db):
        mgr = _make_config_manager(mock_db)
        assert mgr.db is mock_db


# ---------------------------------------------------------------------------
# TestGetConfig
# ---------------------------------------------------------------------------


class TestGetConfig:
    def test_returns_cached_value_without_db_query(self, mock_db):
        mgr = _make_config_manager(mock_db)
        # Pre-populate cache with a recent timestamp
        import time
        mgr._config_cache["my_key"] = "cached_value"
        mgr._last_cache_update = time.time()  # fresh cache

        result = mgr.get_config("my_key")

        assert result == "cached_value"
        # DB should NOT have been queried
        mock_db.assert_not_called()

    def test_queries_db_when_not_cached(self, mock_db):
        mgr = _make_config_manager(mock_db)
        # Stale cache (last update time = 0 means expired)
        mgr._last_cache_update = 0

        config_row = MagicMock()
        config_row.value = "db_value"
        mock_db.return_value.select.return_value.first.return_value = config_row

        result = mgr.get_config("some_key")

        assert result == "db_value"

    def test_db_json_value_parsed(self, mock_db):
        mgr = _make_config_manager(mock_db)
        mgr._last_cache_update = 0

        config_row = MagicMock()
        config_row.value = '{"enabled": true, "count": 5}'
        mock_db.return_value.select.return_value.first.return_value = config_row

        result = mgr.get_config("json_key")

        assert result == {"enabled": True, "count": 5}

    def test_falls_back_to_env_var_when_db_empty(self, mock_db):
        mgr = _make_config_manager(mock_db)
        mgr._last_cache_update = 0
        mock_db.return_value.select.return_value.first.return_value = None

        with patch.dict(os.environ, {"MY_ENV_KEY": "env_value"}):
            result = mgr.get_config("my_env_key")

        assert result == "env_value"

    def test_falls_back_to_default_when_all_miss(self, mock_db):
        mgr = _make_config_manager(mock_db)
        mgr._last_cache_update = 0
        mock_db.return_value.select.return_value.first.return_value = None

        # Ensure env var is absent
        env_key = "DEFINITELY_NOT_SET_XYZ_789"
        os.environ.pop(env_key, None)

        result = mgr.get_config(env_key.lower(), default="fallback")

        assert result == "fallback"

    def test_db_string_value_not_parsed_as_json(self, mock_db):
        mgr = _make_config_manager(mock_db)
        mgr._last_cache_update = 0

        config_row = MagicMock()
        config_row.value = "plain string"
        mock_db.return_value.select.return_value.first.return_value = config_row

        result = mgr.get_config("str_key")

        assert result == "plain string"


# ---------------------------------------------------------------------------
# TestSetConfig
# ---------------------------------------------------------------------------


class TestSetConfig:
    def test_insert_called_when_no_existing_row(self, mock_db):
        mgr = _make_config_manager(mock_db)
        # No existing row
        mock_db.return_value.select.return_value.first.return_value = None
        mock_db.system_config = MagicMock()
        mock_db.system_config.insert = MagicMock()

        mgr.set_config("new_key", "new_value")

        mock_db.system_config.insert.assert_called_once()

    def test_update_called_when_existing_row(self, mock_db):
        mgr = _make_config_manager(mock_db)
        existing = MagicMock()
        mock_db.return_value.select.return_value.first.return_value = existing

        mgr.set_config("existing_key", "updated_value")

        existing.update_record.assert_called_once()

    def test_cache_updated_after_set(self, mock_db):
        mgr = _make_config_manager(mock_db)
        mock_db.return_value.select.return_value.first.return_value = None
        mock_db.system_config = MagicMock()

        mgr.set_config("cached_key", "cached_val")

        assert mgr._config_cache.get("cached_key") == "cached_val"

    def test_dict_value_serialized_to_json(self, mock_db):
        mgr = _make_config_manager(mock_db)
        mock_db.return_value.select.return_value.first.return_value = None
        mock_db.system_config = MagicMock()

        mgr.set_config("dict_key", {"a": 1, "b": 2})

        call_kwargs = mock_db.system_config.insert.call_args[1]
        import json
        assert json.loads(call_kwargs["value"]) == {"a": 1, "b": 2}

    def test_returns_true_on_success(self, mock_db):
        mgr = _make_config_manager(mock_db)
        mock_db.return_value.select.return_value.first.return_value = None
        mock_db.system_config = MagicMock()

        result = mgr.set_config("k", "v")

        assert result is True


# ---------------------------------------------------------------------------
# TestGetDatabaseConfig
# ---------------------------------------------------------------------------


class TestGetDatabaseConfig:
    def test_returns_dict_with_required_keys(self, mock_db):
        mgr = _make_config_manager(mock_db)

        # get_config falls back to env/default for all keys
        mgr._last_cache_update = 0
        mock_db.return_value.select.return_value.first.return_value = None

        result = mgr.get_database_config()

        assert isinstance(result, dict)
        for key in ("host", "port", "database", "username", "password", "pool_size"):
            assert key in result, f"Missing key: {key}"

    def test_port_is_integer(self, mock_db):
        mgr = _make_config_manager(mock_db)
        mgr._last_cache_update = 0
        mock_db.return_value.select.return_value.first.return_value = None

        result = mgr.get_database_config()

        assert isinstance(result["port"], int)

    def test_pool_size_is_integer(self, mock_db):
        mgr = _make_config_manager(mock_db)
        mgr._last_cache_update = 0
        mock_db.return_value.select.return_value.first.return_value = None

        result = mgr.get_database_config()

        assert isinstance(result["pool_size"], int)


# ---------------------------------------------------------------------------
# TestGetRedisConfig
# ---------------------------------------------------------------------------


class TestGetRedisConfig:
    def test_returns_dict_with_required_keys(self, mock_db):
        mgr = _make_config_manager(mock_db)
        mgr._last_cache_update = 0
        mock_db.return_value.select.return_value.first.return_value = None

        result = mgr.get_redis_config()

        assert isinstance(result, dict)
        for key in ("host", "port", "password", "database"):
            assert key in result, f"Missing key: {key}"

    def test_port_is_integer(self, mock_db):
        mgr = _make_config_manager(mock_db)
        mgr._last_cache_update = 0
        mock_db.return_value.select.return_value.first.return_value = None

        result = mgr.get_redis_config()

        assert isinstance(result["port"], int)

    def test_database_is_integer(self, mock_db):
        mgr = _make_config_manager(mock_db)
        mgr._last_cache_update = 0
        mock_db.return_value.select.return_value.first.return_value = None

        result = mgr.get_redis_config()

        assert isinstance(result["database"], int)


# ---------------------------------------------------------------------------
# TestClearCache
# ---------------------------------------------------------------------------


class TestClearCache:
    def test_cache_empty_after_clear(self, mock_db):
        mgr = _make_config_manager(mock_db)
        mgr._config_cache = {"key1": "val1", "key2": "val2"}

        mgr.clear_cache()

        assert mgr._config_cache == {}

    def test_last_cache_update_reset(self, mock_db):
        mgr = _make_config_manager(mock_db)
        import time
        mgr._last_cache_update = time.time()

        mgr.clear_cache()

        assert mgr._last_cache_update == 0

    def test_get_config_queries_db_after_clear(self, mock_db):
        mgr = _make_config_manager(mock_db)
        import time
        mgr._config_cache = {"k": "old"}
        mgr._last_cache_update = time.time()  # still fresh

        mgr.clear_cache()

        # Now the cache is stale; DB should be consulted
        config_row = MagicMock()
        config_row.value = "fresh_value"
        mock_db.return_value.select.return_value.first.return_value = config_row

        result = mgr.get_config("k")

        assert result == "fresh_value"
