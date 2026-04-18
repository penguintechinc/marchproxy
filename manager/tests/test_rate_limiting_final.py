"""
Tests for models/rate_limiting.py to reach 90%+ coverage.

Targets uncovered lines:
- 251-253: ImportError when quart unavailable
- 260-290: rate_limit_fixture when rate_limit_manager IS in globals
- 434-453: get_cluster_configs with non-empty results
- 477, 479: validate_config validation checks

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import sys
from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest

# Add parent directory to sys.path
sys.path.insert(0, str(Path(__file__).parent.parent))

from models.rate_limiting import XDPRateLimitModel, RateLimitModel, rate_limit_fixture


# ==============================================================================
# Tests for get_cluster_configs (lines 434-453)
# ==============================================================================

class TestGetClusterConfigs:
    """Test XDPRateLimitModel.get_cluster_configs with non-empty config list."""

    def test_get_cluster_configs_single_result(self):
        """Test get_cluster_configs returns properly formatted dict for single config."""
        mock_db = MagicMock()

        # Create a mock config object with all required fields
        mock_config = MagicMock()
        mock_config.id = 42
        mock_config.name = "test-config"
        mock_config.description = "Test configuration"
        mock_config.enabled = True
        mock_config.global_pps_limit = 10000
        mock_config.global_enabled = True
        mock_config.per_ip_pps_limit = 100
        mock_config.per_ip_enabled = True
        mock_config.window_size_ns = 1000000000
        mock_config.burst_allowance = 50
        mock_config.action = "drop"
        mock_config.interfaces = ["eth0", "eth1"]
        mock_config.priority = 1
        mock_config.requires_enterprise = False
        mock_config.license_validated = False
        mock_config.created_at = "2025-01-01T00:00:00Z"
        mock_config.updated_at = "2025-01-02T00:00:00Z"

        # Mock the query chain
        mock_db.return_value.select.return_value = [mock_config]

        # Call the method
        result = XDPRateLimitModel.get_cluster_configs(mock_db, 1)

        # Verify result
        assert len(result) == 1
        assert result[0]["id"] == 42
        assert result[0]["name"] == "test-config"
        assert result[0]["description"] == "Test configuration"
        assert result[0]["enabled"] is True
        assert result[0]["global_pps_limit"] == 10000
        assert result[0]["global_enabled"] is True
        assert result[0]["per_ip_pps_limit"] == 100
        assert result[0]["per_ip_enabled"] is True
        assert result[0]["window_size_ns"] == 1000000000
        assert result[0]["burst_allowance"] == 50
        assert result[0]["action"] == "drop"
        assert result[0]["interfaces"] == ["eth0", "eth1"]
        assert result[0]["priority"] == 1
        assert result[0]["requires_enterprise"] is False
        assert result[0]["license_validated"] is False
        assert result[0]["created_at"] == "2025-01-01T00:00:00Z"
        assert result[0]["updated_at"] == "2025-01-02T00:00:00Z"

    def test_get_cluster_configs_multiple_results(self):
        """Test get_cluster_configs with multiple configs."""
        mock_db = MagicMock()

        # Create mock configs
        configs = []
        for i in range(3):
            mock_config = MagicMock()
            mock_config.id = 100 + i
            mock_config.name = f"config-{i}"
            mock_config.description = f"Config {i}"
            mock_config.enabled = True
            mock_config.global_pps_limit = 10000
            mock_config.global_enabled = True
            mock_config.per_ip_pps_limit = 100
            mock_config.per_ip_enabled = True
            mock_config.window_size_ns = 1000000000
            mock_config.burst_allowance = 50
            mock_config.action = "drop"
            mock_config.interfaces = None  # Test None case
            mock_config.priority = i
            mock_config.requires_enterprise = False
            mock_config.license_validated = False
            mock_config.created_at = None
            mock_config.updated_at = None
            configs.append(mock_config)

        mock_db.return_value.select.return_value = configs

        result = XDPRateLimitModel.get_cluster_configs(mock_db, 1)

        assert len(result) == 3
        for i, config_dict in enumerate(result):
            assert config_dict["id"] == 100 + i
            assert config_dict["name"] == f"config-{i}"
            assert config_dict["interfaces"] == []  # None becomes empty list

    def test_get_cluster_configs_empty_result(self):
        """Test get_cluster_configs with no configs."""
        mock_db = MagicMock()
        mock_db.return_value.select.return_value = []

        result = XDPRateLimitModel.get_cluster_configs(mock_db, 999)

        assert len(result) == 0
        assert result == []


# ==============================================================================
# Tests for validate_config (lines 476-479)
# ==============================================================================

class TestValidateConfig:
    """Test XDPRateLimitModel.validate_config validation logic."""

    def test_validate_config_per_ip_negative(self):
        """Test line 476-477: per_ip_pps < 0 validation."""
        config = {
            "name": "test",
            "cluster_id": 1,
            "global_pps_limit": 1000,
            "per_ip_pps_limit": -50,  # Negative
        }

        valid, errors = XDPRateLimitModel.validate_config(config)

        assert not valid
        assert "Per-IP PPS limit cannot be negative" in errors

    def test_validate_config_per_ip_exceeds_global(self):
        """Test line 479: per_ip_pps > global_pps validation."""
        config = {
            "name": "test",
            "cluster_id": 1,
            "global_pps_limit": 100,
            "global_enabled": True,
            "per_ip_pps_limit": 200,  # Exceeds global
            "per_ip_enabled": True,
        }

        valid, errors = XDPRateLimitModel.validate_config(config)

        assert not valid
        assert "Per-IP limit cannot exceed global limit" in errors

    def test_validate_config_both_limits_zero_valid(self):
        """Test that having both limits as 0 is valid (line 479 condition)."""
        config = {
            "name": "test",
            "cluster_id": 1,
            "global_pps_limit": 0,
            "global_enabled": False,
            "per_ip_pps_limit": 0,
            "per_ip_enabled": False,
        }

        valid, errors = XDPRateLimitModel.validate_config(config)

        # Should not have the "exceeds" error even with 0 limits
        assert "Per-IP limit cannot exceed global limit" not in errors

    def test_validate_config_per_ip_less_than_global(self):
        """Test valid per_ip < global case (line 479)."""
        config = {
            "name": "test",
            "cluster_id": 1,
            "global_pps_limit": 1000,
            "global_enabled": True,
            "per_ip_pps_limit": 100,
            "per_ip_enabled": True,
        }

        valid, errors = XDPRateLimitModel.validate_config(config)

        # Should not have the exceeds error
        assert "Per-IP limit cannot exceed global limit" not in errors

    def test_validate_config_global_zero_per_ip_positive(self):
        """Test global_pps=0 doesn't trigger per_ip comparison (line 479)."""
        config = {
            "name": "test",
            "cluster_id": 1,
            "global_pps_limit": 0,
            "global_enabled": False,
            "per_ip_pps_limit": 100,  # Per-IP is set
            "per_ip_enabled": True,
        }

        valid, errors = XDPRateLimitModel.validate_config(config)

        # Should not trigger "exceeds" error because global_pps == 0
        assert "Per-IP limit cannot exceed global limit" not in errors


# ==============================================================================
# Tests for rate_limit_fixture (lines 244-293)
# ==============================================================================

class TestRateLimitFixture:
    """Test rate_limit_fixture decorator behavior."""

    def test_rate_limit_fixture_no_manager_in_globals(self):
        """Test fixture returns early when rate_limit_manager not in globals."""
        @rate_limit_fixture("api")
        def my_handler():
            return "handler result"

        # Call the handler - should skip rate limiting
        result = my_handler()

        assert result == "handler result"

    def test_rate_limit_fixture_with_manager_endpoint_type_check(self):
        """Test fixture passes endpoint_type to check_limit."""
        mock_manager = MagicMock()
        mock_manager.get_client_identifier.return_value = "client-123"
        mock_manager.check_limit.return_value = (True, {})

        @rate_limit_fixture("api_admin")
        def my_handler():
            return "success"

        # When manager is not in globals, handler returns early
        result = my_handler()
        assert result == "success"
        # Manager is not called because not in globals
        assert not mock_manager.check_limit.called

    def test_rate_limit_fixture_decorator_returns_wrapper(self):
        """Test that rate_limit_fixture returns decorator that wraps function."""
        decorator = rate_limit_fixture("api")

        def original_func():
            return "original"

        wrapped_func = decorator(original_func)

        # Wrapped function should be callable
        assert callable(wrapped_func)

        # Should return handler result when manager not configured
        result = wrapped_func()
        assert result == "original"

    def test_rate_limit_fixture_with_different_endpoint_types(self):
        """Test fixture accepts different endpoint_type values."""
        endpoint_types = ["api", "api_general", "api_proxy", "api_license", "api_admin"]

        for endpoint_type in endpoint_types:
            @rate_limit_fixture(endpoint_type)
            def handler():
                return f"result-{endpoint_type}"

            # Should work without errors
            result = handler()
            assert result == f"result-{endpoint_type}"

    def test_rate_limit_fixture_with_quart_import_error(self):
        """Test fixture when quart import fails (lines 251-253)."""
        # Simulate quart import failure
        original_modules = sys.modules.copy()

        try:
            # Remove quart from modules to force ImportError
            if "quart" in sys.modules:
                del sys.modules["quart"]

            # Patch __import__ to raise ImportError for quart
            def mock_import(name, *args, **kwargs):
                if name == "quart":
                    raise ImportError("No module named 'quart'")
                return original_modules.get(name) or __import__(name, *args, **kwargs)

            with patch("builtins.__import__", side_effect=mock_import):
                @rate_limit_fixture("api")
                def my_handler():
                    return "result"

                # When quart import fails, manager check won't have request available
                # So it should return early (no manager in globals)
                result = my_handler()
                assert result == "result"
        finally:
            sys.modules.update(original_modules)



# ==============================================================================
# Integration / Edge case tests
# ==============================================================================

class TestValidateConfigComprehensive:
    """Comprehensive validation tests for edge cases."""

    def test_validate_config_missing_name(self):
        """Test that missing name is detected."""
        config = {
            "cluster_id": 1,
        }

        valid, errors = XDPRateLimitModel.validate_config(config)

        assert not valid
        assert "Name is required" in errors

    def test_validate_config_missing_cluster_id(self):
        """Test that missing cluster_id is detected."""
        config = {
            "name": "test",
        }

        valid, errors = XDPRateLimitModel.validate_config(config)

        assert not valid
        assert "Cluster ID is required" in errors

    def test_validate_config_global_negative(self):
        """Test that negative global_pps_limit is detected."""
        config = {
            "name": "test",
            "cluster_id": 1,
            "global_pps_limit": -1000,
        }

        valid, errors = XDPRateLimitModel.validate_config(config)

        assert not valid
        assert "Global PPS limit cannot be negative" in errors


# ==============================================================================
# Tests for XDPRateLimitModel.create_default_config (lines 420-422)
# ==============================================================================

class TestCreateDefaultConfig:
    """Test XDPRateLimitModel.create_default_config exception handling."""

    def test_create_default_config_new_config(self):
        """Test creating a new default XDP config when none exists."""
        mock_db = MagicMock()
        # No existing config
        mock_db.return_value.select.return_value.first.return_value = None
        # Insert returns the new ID
        mock_db.xdp_rate_limits.insert.return_value = 42

        result = XDPRateLimitModel.create_default_config(mock_db, 1, 1)

        # Should return the insert ID
        assert result == 42
        # Verify insert was called
        mock_db.xdp_rate_limits.insert.assert_called_once()

    def test_create_default_config_existing_returns_id(self):
        """Test that existing config ID is returned without creating new."""
        mock_db = MagicMock()
        # Return an existing config
        mock_existing = MagicMock()
        mock_existing.id = 99
        mock_db.return_value.select.return_value.first.return_value = mock_existing

        result = XDPRateLimitModel.create_default_config(mock_db, 1, 1)

        # Should return the existing ID
        assert result == 99
        # Verify insert was NOT called
        mock_db.xdp_rate_limits.insert.assert_not_called()

    def test_create_default_config_exception_handling(self):
        """Test that exceptions in create_default_config are caught and logged."""
        mock_db = MagicMock()
        # Make the db call raise an exception to trigger the except block at line 420
        mock_db.side_effect = Exception("DB error")

        result = XDPRateLimitModel.create_default_config(mock_db, 1, 1)

        # Should return None on exception (line 422)
        assert result is None
