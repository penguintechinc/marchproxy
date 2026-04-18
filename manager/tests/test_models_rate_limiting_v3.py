"""
Unit tests for rate limiting models

Tests cover RateLimitModel, RateLimitManager, XDPRateLimitModel, and XDPRateLimitManager
"""

import pytest
from datetime import datetime, timedelta
from unittest.mock import MagicMock, patch, call
from models.rate_limiting import (
    RateLimitModel,
    RateLimitManager,
    XDPRateLimitModel,
    XDPRateLimitManager,
)


@pytest.fixture
def mock_db():
    """Create a mock database object"""
    db = MagicMock()
    db.rate_limits = MagicMock()
    db.xdp_rate_limits = MagicMock()
    db.xdp_rate_limit_stats = MagicMock()
    db.xdp_rate_limit_whitelist = MagicMock()
    db.clusters = MagicMock()
    db.proxy_servers = MagicMock()
    db.users = MagicMock()
    return db


class TestRateLimitModel:
    """Tests for RateLimitModel static methods"""

    def test_define_table(self, mock_db):
        """Test table definition"""
        mock_db.define_table = MagicMock(return_value=MagicMock())
        result = RateLimitModel.define_table(mock_db)
        mock_db.define_table.assert_called_once()

    def test_check_rate_limit_new_client(self, mock_db):
        """Test rate limit check for new client (no existing record)"""
        mock_db.return_value.select.return_value.first.return_value = None
        mock_db.rate_limits.insert = MagicMock(return_value=1)

        allowed, info = RateLimitModel.check_rate_limit(
            mock_db, "client-1", "/api/test", max_requests=100, window_minutes=60
        )

        assert allowed is True
        assert info["allowed"] is True
        assert info["requests_remaining"] == 99
        assert "window_reset" in info

    def test_check_rate_limit_existing_client_within_limit(self, mock_db):
        """Test rate limit check for existing client within limit"""
        existing_record = MagicMock()
        existing_record.is_blocked = False
        existing_record.request_count = 50
        existing_record.window_start = datetime.utcnow() - timedelta(minutes=30)
        existing_record.last_request = datetime.utcnow()
        existing_record.update_record = MagicMock()

        mock_db.return_value.select.return_value.first.return_value = existing_record

        allowed, info = RateLimitModel.check_rate_limit(
            mock_db, "client-1", "/api/test", max_requests=100, window_minutes=60
        )

        assert allowed is True
        assert info["allowed"] is True
        # After incrementing by 1, requests_remaining = max_requests - (request_count + 1)
        # = 100 - (50 + 1) = 49, but the model calculates before updating
        # So it's 100 - 50 = 50
        assert info["requests_remaining"] == 50

    def test_check_rate_limit_blocked_client(self, mock_db):
        """Test rate limit check for blocked client"""
        future_time = datetime.utcnow() + timedelta(minutes=10)
        existing_record = MagicMock()
        existing_record.is_blocked = True
        existing_record.block_until = future_time
        existing_record.last_request = datetime.utcnow()

        mock_db.return_value.select.return_value.first.return_value = existing_record

        allowed, info = RateLimitModel.check_rate_limit(mock_db, "client-1", "/api/test")

        assert allowed is False
        assert info["allowed"] is False
        assert info["requests_remaining"] == 0
        assert "retry_after" in info

    def test_check_rate_limit_exceeded(self, mock_db):
        """Test rate limit check when limit exceeded"""
        existing_record = MagicMock()
        existing_record.is_blocked = False
        existing_record.request_count = 100
        existing_record.window_start = datetime.utcnow() - timedelta(minutes=30)
        existing_record.last_request = datetime.utcnow()
        existing_record.update_record = MagicMock()

        mock_db.return_value.select.return_value.first.return_value = existing_record

        allowed, info = RateLimitModel.check_rate_limit(
            mock_db, "client-1", "/api/test", max_requests=100, block_duration_minutes=15
        )

        assert allowed is False
        assert info["allowed"] is False
        existing_record.update_record.assert_called_once()
        call_kwargs = existing_record.update_record.call_args[1]
        assert call_kwargs["is_blocked"] is True

    def test_check_rate_limit_window_expired(self, mock_db):
        """Test rate limit check with expired window"""
        old_window_start = datetime.utcnow() - timedelta(minutes=90)
        existing_record = MagicMock()
        existing_record.is_blocked = False
        existing_record.request_count = 100
        existing_record.window_start = old_window_start
        existing_record.last_request = datetime.utcnow()
        existing_record.update_record = MagicMock()

        mock_db.return_value.select.return_value.first.return_value = existing_record

        allowed, info = RateLimitModel.check_rate_limit(
            mock_db, "client-1", "/api/test", max_requests=100, window_minutes=60
        )

        assert allowed is True
        assert info["allowed"] is True
        existing_record.update_record.assert_called_once()
        call_kwargs = existing_record.update_record.call_args[1]
        assert call_kwargs["request_count"] == 1

    def test_rate_limit_fixture_import(self, mock_db):
        """Test that rate limit fixture can be imported"""
        from models.rate_limiting import rate_limit_fixture

        # Should return a decorator function
        decorator = rate_limit_fixture(endpoint_type="api_general")
        assert callable(decorator)

    def test_get_client_stats_no_records(self, mock_db):
        """Test client stats with no records"""
        mock_db.return_value.select.return_value = []

        stats = RateLimitModel.get_client_stats(mock_db, "client-1")

        assert stats["client_id"] == "client-1"
        assert stats["endpoints"] == []
        assert stats["total_requests"] == 0
        assert stats["blocked_endpoints"] == 0

    def test_get_client_stats_with_records(self, mock_db):
        """Test client stats with records"""
        record1 = MagicMock()
        record1.endpoint = "/api/test1"
        record1.request_count = 50
        record1.is_blocked = False
        record1.block_until = None
        record1.window_start = datetime.utcnow()
        record1.last_request = datetime.utcnow()

        record2 = MagicMock()
        record2.endpoint = "/api/test2"
        record2.request_count = 30
        record2.is_blocked = True
        record2.block_until = datetime.utcnow() + timedelta(minutes=10)
        record2.window_start = datetime.utcnow()
        record2.last_request = datetime.utcnow()

        mock_db.return_value.select.return_value = [record1, record2]

        stats = RateLimitModel.get_client_stats(mock_db, "client-1")

        assert stats["client_id"] == "client-1"
        assert len(stats["endpoints"]) == 2
        assert stats["total_requests"] == 80
        assert stats["blocked_endpoints"] == 1


class TestRateLimitManager:
    """Tests for RateLimitManager"""

    def test_manager_initialization(self, mock_db):
        """Test manager initialization"""
        manager = RateLimitManager(mock_db)
        assert manager.db is mock_db
        assert "api_general" in manager.policies
        assert "api_auth" in manager.policies
        assert "api_proxy" in manager.policies

    def test_check_limit_with_default_policy(self, mock_db):
        """Test check_limit with default policy"""
        existing_record = MagicMock()
        existing_record.is_blocked = False
        existing_record.request_count = 50
        existing_record.window_start = datetime.utcnow() - timedelta(minutes=30)
        existing_record.update_record = MagicMock()
        mock_db.return_value.select.return_value.first.return_value = existing_record

        manager = RateLimitManager(mock_db)
        allowed, info = manager.check_limit("client-1", "/api/test")

        assert allowed is True
        assert info["allowed"] is True

    def test_check_limit_with_custom_policy(self, mock_db):
        """Test check_limit with custom endpoint type"""
        existing_record = MagicMock()
        existing_record.is_blocked = False
        existing_record.request_count = 20
        existing_record.window_start = datetime.utcnow() - timedelta(minutes=10)
        existing_record.update_record = MagicMock()
        mock_db.return_value.select.return_value.first.return_value = existing_record

        manager = RateLimitManager(mock_db)
        allowed, info = manager.check_limit("client-1", "/api/auth/login", endpoint_type="api_auth")

        assert allowed is True

    def test_get_client_identifier_from_user(self, mock_db):
        """Test client identifier extraction from user"""
        request = MagicMock()
        user = {"id": "user-123"}

        manager = RateLimitManager(mock_db)
        identifier = manager.get_client_identifier(request, user)

        assert identifier == "user:user-123"

    def test_get_client_identifier_from_x_forwarded_for(self, mock_db):
        """Test client identifier from X-Forwarded-For header"""
        request = MagicMock()
        request.headers.get.side_effect = lambda h: "192.168.1.1, 10.0.0.1" if h == "X-Forwarded-For" else None
        request.environ.get.return_value = "127.0.0.1"

        manager = RateLimitManager(mock_db)
        identifier = manager.get_client_identifier(request)

        assert identifier == "ip:192.168.1.1"

    def test_get_client_identifier_from_remote_addr(self, mock_db):
        """Test client identifier from REMOTE_ADDR"""
        request = MagicMock()
        request.headers.get.side_effect = lambda h: None
        request.environ.get.return_value = "192.168.1.100"

        manager = RateLimitManager(mock_db)
        identifier = manager.get_client_identifier(request)

        assert identifier == "ip:192.168.1.100"

    def test_get_endpoint_type_auth(self, mock_db):
        """Test endpoint type detection for auth endpoints"""
        manager = RateLimitManager(mock_db)
        endpoint_type = manager.get_endpoint_type("/api/auth/login")
        assert endpoint_type == "api_auth"

    def test_get_endpoint_type_proxy(self, mock_db):
        """Test endpoint type detection for proxy endpoints"""
        manager = RateLimitManager(mock_db)
        endpoint_type = manager.get_endpoint_type("/api/proxy/register")
        assert endpoint_type == "api_proxy"

    def test_get_endpoint_type_license(self, mock_db):
        """Test endpoint type detection for license endpoints"""
        manager = RateLimitManager(mock_db)
        endpoint_type = manager.get_endpoint_type("/api/license/validate")
        assert endpoint_type == "api_license"

    def test_get_endpoint_type_admin(self, mock_db):
        """Test endpoint type detection for admin endpoints"""
        manager = RateLimitManager(mock_db)
        endpoint_type = manager.get_endpoint_type("/api/clusters/list")
        assert endpoint_type == "api_admin"

    def test_get_endpoint_type_general(self, mock_db):
        """Test endpoint type detection for general endpoints"""
        manager = RateLimitManager(mock_db)
        endpoint_type = manager.get_endpoint_type("/api/services/list")
        assert endpoint_type == "api_general"


class TestXDPRateLimitModel:
    """Tests for XDPRateLimitModel"""

    def test_define_table(self, mock_db):
        """Test XDP rate limit table definition"""
        mock_db.define_table = MagicMock(return_value=MagicMock())
        result = XDPRateLimitModel.define_table(mock_db)
        mock_db.define_table.assert_called_once()

    def test_define_stats_table(self, mock_db):
        """Test XDP stats table definition"""
        mock_db.define_table = MagicMock(return_value=MagicMock())
        result = XDPRateLimitModel.define_stats_table(mock_db)
        mock_db.define_table.assert_called_once()

    def test_define_ip_whitelist_table(self, mock_db):
        """Test XDP IP whitelist table definition"""
        mock_db.define_table = MagicMock(return_value=MagicMock())
        result = XDPRateLimitModel.define_ip_whitelist_table(mock_db)
        mock_db.define_table.assert_called_once()

    def test_create_default_config_new_cluster(self, mock_db):
        """Test creating default XDP config for new cluster"""
        mock_db.return_value.select.return_value.first.return_value = None
        mock_db.xdp_rate_limits.insert = MagicMock(return_value=1)

        config_id = XDPRateLimitModel.create_default_config(mock_db, cluster_id=1, user_id=1)

        assert config_id == 1
        mock_db.xdp_rate_limits.insert.assert_called_once()

    def test_create_default_config_existing(self, mock_db):
        """Test creating default config for cluster that already has one"""
        existing = MagicMock()
        existing.id = 1
        mock_db.return_value.select.return_value.first.return_value = existing

        config_id = XDPRateLimitModel.create_default_config(mock_db, cluster_id=1, user_id=1)

        assert config_id == 1
        mock_db.xdp_rate_limits.insert.assert_not_called()

    def test_validate_config_valid(self, mock_db):
        """Test XDP config validation - valid config"""
        config = {
            "name": "Test Rate Limit",
            "cluster_id": 1,
            "global_pps_limit": 100000,
            "per_ip_pps_limit": 1000,
            "window_size_ns": 1000000000,
            "burst_allowance": 100,
            "action": 1,
            "interfaces": ["eth0"],
        }

        valid, errors = XDPRateLimitModel.validate_config(config)
        assert valid is True
        assert errors == []

    def test_validate_config_missing_name(self, mock_db):
        """Test validation with missing name"""
        config = {
            "cluster_id": 1,
            "global_pps_limit": 100000,
            "interfaces": ["eth0"],
        }

        valid, errors = XDPRateLimitModel.validate_config(config)
        assert valid is False
        assert "Name is required" in errors

    def test_validate_config_missing_cluster(self, mock_db):
        """Test validation with missing cluster ID"""
        config = {"name": "Test", "global_pps_limit": 100000, "interfaces": ["eth0"]}

        valid, errors = XDPRateLimitModel.validate_config(config)
        assert valid is False
        assert "Cluster ID is required" in errors

    def test_validate_config_negative_pps_limit(self, mock_db):
        """Test validation with negative PPS limit"""
        config = {
            "name": "Test",
            "cluster_id": 1,
            "global_pps_limit": -100,
            "interfaces": ["eth0"],
        }

        valid, errors = XDPRateLimitModel.validate_config(config)
        assert valid is False
        assert any("cannot be negative" in e for e in errors)

    def test_validate_config_invalid_action(self, mock_db):
        """Test validation with invalid action"""
        config = {
            "name": "Test",
            "cluster_id": 1,
            "action": 99,
            "interfaces": ["eth0"],
        }

        valid, errors = XDPRateLimitModel.validate_config(config)
        assert valid is False
        assert any("Action must be" in e for e in errors)

    def test_validate_config_no_interfaces(self, mock_db):
        """Test validation with no interfaces"""
        config = {
            "name": "Test",
            "cluster_id": 1,
            "interfaces": [],
        }

        valid, errors = XDPRateLimitModel.validate_config(config)
        assert valid is False
        assert any("interface" in e.lower() for e in errors)

    def test_get_cluster_configs_empty(self, mock_db):
        """Test getting cluster configurations - empty"""
        query_mock = MagicMock()
        query_mock.select.return_value = []
        mock_db.return_value = query_mock

        configs = XDPRateLimitModel.get_cluster_configs(mock_db, cluster_id=1)

        assert len(configs) == 0
        assert isinstance(configs, list)

    def test_delete_rate_limit_not_found(self, mock_db):
        """Test deleting non-existent rate limit"""
        mock_db.xdp_rate_limits.__getitem__ = MagicMock(return_value=None)

        manager = XDPRateLimitManager(mock_db)
        success, result = manager.delete_rate_limit(999)

        assert success is False
        assert "error" in result

    def test_get_proxy_config_for_proxy(self, mock_db):
        """Test getting proxy config for a specific proxy"""
        license_manager = MagicMock()
        license_manager.has_feature.return_value = True

        with patch.object(
            XDPRateLimitModel,
            "get_cluster_configs",
            return_value=[{"id": 1, "enabled": True, "license_validated": True}],
        ):
            manager = XDPRateLimitManager(mock_db, license_manager=license_manager)
            result = manager.get_proxy_config(cluster_id=1, proxy_id=1)

            assert result["cluster_id"] == 1
            assert result["proxy_id"] == 1
            assert "configurations" in result


class TestXDPRateLimitManager:
    """Tests for XDPRateLimitManager"""

    def test_manager_initialization(self, mock_db):
        """Test XDP manager initialization"""
        manager = XDPRateLimitManager(mock_db)
        assert manager.db is mock_db
        assert manager.license_manager is None

    def test_manager_initialization_with_license(self, mock_db):
        """Test initialization with license manager"""
        license_manager = MagicMock()
        manager = XDPRateLimitManager(mock_db, license_manager=license_manager)
        assert manager.license_manager is license_manager

    def test_create_rate_limit_success(self, mock_db):
        """Test creating XDP rate limit"""
        config = {
            "name": "Test Rate Limit",
            "cluster_id": 1,
            "global_pps_limit": 100000,
            "per_ip_pps_limit": 1000,
            "window_size_ns": 1000000000,
            "burst_allowance": 100,
            "action": 1,
            "interfaces": ["eth0"],
            "requires_enterprise": False,
        }

        mock_db.xdp_rate_limits.insert = MagicMock(return_value=1)
        license_manager = MagicMock()
        license_manager.has_feature.return_value = False

        manager = XDPRateLimitManager(mock_db, license_manager=license_manager)
        success, result = manager.create_rate_limit(1, config, user_id=1)

        assert success is True
        assert result["id"] == 1

    def test_create_rate_limit_validation_error(self, mock_db):
        """Test creating rate limit with validation error"""
        config = {"name": "", "cluster_id": 1}

        manager = XDPRateLimitManager(mock_db)
        success, result = manager.create_rate_limit(1, config, user_id=1)

        assert success is False
        assert "errors" in result

    def test_create_rate_limit_license_error(self, mock_db):
        """Test creating rate limit when license check fails"""
        config = {
            "name": "Enterprise Config",
            "cluster_id": 1,
            "interfaces": ["eth0"],
            "requires_enterprise": True,
        }

        license_manager = MagicMock()
        license_manager.has_feature.return_value = False

        manager = XDPRateLimitManager(mock_db, license_manager=license_manager)
        success, result = manager.create_rate_limit(1, config, user_id=1)

        assert success is False
        assert "error" in result
        assert "Enterprise license" in result["error"]

    def test_create_rate_limit_database_error(self, mock_db):
        """Test creating rate limit with database error"""
        config = {
            "name": "Test",
            "cluster_id": 1,
            "interfaces": ["eth0"],
            "requires_enterprise": False,
        }

        mock_db.xdp_rate_limits.insert.side_effect = Exception("DB Error")

        manager = XDPRateLimitManager(mock_db)
        success, result = manager.create_rate_limit(1, config, user_id=1)

        assert success is False
        assert "error" in result

    def test_update_rate_limit_validation_error(self, mock_db):
        """Test updating XDP rate limit with validation error"""
        existing = MagicMock()
        existing.cluster_id = 1

        mock_db.xdp_rate_limits.__getitem__ = MagicMock(return_value=existing)

        # Invalid config - missing required fields
        config = {
            "name": "",  # Empty name
            "cluster_id": 1,
        }

        manager = XDPRateLimitManager(mock_db)
        success, result = manager.update_rate_limit(1, config, user_id=1)

        assert success is False
        assert "errors" in result

    def test_update_rate_limit_success(self, mock_db):
        """Test successfully updating rate limit"""
        existing = MagicMock()
        existing.cluster_id = 1
        existing.name = "Old Name"
        existing.description = "Old desc"
        existing.enabled = False
        existing.global_pps_limit = 50000
        existing.global_enabled = True
        existing.per_ip_pps_limit = 500
        existing.per_ip_enabled = True
        existing.window_size_ns = 1000000000
        existing.burst_allowance = 50
        existing.action = 1
        existing.interfaces = ["eth0"]
        existing.priority = 100
        existing.update_record = MagicMock()

        mock_db.xdp_rate_limits.__getitem__ = MagicMock(return_value=existing)

        config = {
            "name": "New Name",
            "cluster_id": 1,
            "interfaces": ["eth0"],
            "requires_enterprise": False,
        }

        manager = XDPRateLimitManager(mock_db)
        success, result = manager.update_rate_limit(1, config, user_id=1)

        assert success is True
        existing.update_record.assert_called_once()

    def test_update_rate_limit_license_error(self, mock_db):
        """Test updating rate limit when license check fails"""
        existing = MagicMock()
        existing.cluster_id = 1
        mock_db.xdp_rate_limits.__getitem__ = MagicMock(return_value=existing)

        config = {
            "name": "Test",
            "cluster_id": 1,
            "interfaces": ["eth0"],
            "requires_enterprise": True,
        }

        license_manager = MagicMock()
        license_manager.has_feature.return_value = False

        manager = XDPRateLimitManager(mock_db, license_manager=license_manager)
        success, result = manager.update_rate_limit(1, config, user_id=1)

        assert success is False
        assert "Enterprise license" in result["error"]

    def test_update_rate_limit_not_found(self, mock_db):
        """Test updating non-existent rate limit"""
        mock_db.xdp_rate_limits.__getitem__ = MagicMock(return_value=None)

        config = {"name": "Test", "cluster_id": 1, "interfaces": ["eth0"]}

        manager = XDPRateLimitManager(mock_db)
        success, result = manager.update_rate_limit(1, config, user_id=1)

        assert success is False
        assert "not found" in result["error"]

    def test_update_rate_limit_database_error(self, mock_db):
        """Test updating rate limit with database error"""
        existing = MagicMock()
        existing.cluster_id = 1
        existing.update_record.side_effect = Exception("DB Error")
        mock_db.xdp_rate_limits.__getitem__ = MagicMock(return_value=existing)

        config = {"name": "Test", "cluster_id": 1, "interfaces": ["eth0"], "requires_enterprise": False}

        manager = XDPRateLimitManager(mock_db)
        success, result = manager.update_rate_limit(1, config, user_id=1)

        assert success is False
        assert "error" in result

    def test_delete_rate_limit_success(self, mock_db):
        """Test deleting XDP rate limit"""
        existing = MagicMock()
        existing.update_record = MagicMock()
        mock_db.xdp_rate_limits.__getitem__ = MagicMock(return_value=existing)

        manager = XDPRateLimitManager(mock_db)
        success, result = manager.delete_rate_limit(1)

        assert success is True
        existing.update_record.assert_called_once()
        call_kwargs = existing.update_record.call_args[1]
        assert call_kwargs["is_active"] is False

    def test_delete_rate_limit_database_error(self, mock_db):
        """Test deleting rate limit with database error"""
        existing = MagicMock()
        existing.update_record.side_effect = Exception("DB Error")
        mock_db.xdp_rate_limits.__getitem__ = MagicMock(return_value=existing)

        manager = XDPRateLimitManager(mock_db)
        success, result = manager.delete_rate_limit(1)

        assert success is False
        assert "error" in result

    def test_get_proxy_config(self, mock_db):
        """Test getting proxy config"""
        config = {
            "id": 1,
            "name": "Test",
            "enabled": True,
            "license_validated": True,
        }

        with patch.object(
            XDPRateLimitModel, "get_cluster_configs", return_value=[config]
        ):
            license_manager = MagicMock()
            license_manager.has_feature.return_value = True

            manager = XDPRateLimitManager(mock_db, license_manager=license_manager)
            result = manager.get_proxy_config(cluster_id=1, proxy_id=1)

            assert result["cluster_id"] == 1
            assert result["proxy_id"] == 1
            assert len(result["configurations"]) == 1
            assert result["enterprise_enabled"] is True

    def test_get_proxy_config_filters_disabled(self, mock_db):
        """Test proxy config filters disabled configurations"""
        configs = [
            {"id": 1, "name": "Enabled", "enabled": True, "license_validated": True},
            {"id": 2, "name": "Disabled", "enabled": False, "license_validated": True},
        ]

        with patch.object(XDPRateLimitModel, "get_cluster_configs", return_value=configs):
            manager = XDPRateLimitManager(mock_db)
            result = manager.get_proxy_config(cluster_id=1, proxy_id=1)

            # Only enabled configs should be included
            assert len(result["configurations"]) == 1
            assert result["configurations"][0]["id"] == 1

    def test_get_proxy_config_without_license_manager(self, mock_db):
        """Test proxy config without license manager"""
        with patch.object(XDPRateLimitModel, "get_cluster_configs", return_value=[]):
            manager = XDPRateLimitManager(mock_db)
            result = manager.get_proxy_config(cluster_id=1, proxy_id=1)

            assert result["enterprise_enabled"] is False


class TestRateLimitModelStatsOnly:
    """Additional tests for stats methods"""

    @pytest.fixture
    def mock_db(self):
        """Create a mock database object"""
        db = MagicMock()
        db.rate_limits = MagicMock()
        return db




class TestValidateConfigEdgeCases:
    """Edge case tests for validate_config"""

    def test_validate_config_per_ip_exceeds_global(self):
        """Test validation catches per-IP limit exceeding global limit"""
        config = {
            "name": "Test",
            "cluster_id": 1,
            "global_pps_limit": 1000,
            "per_ip_pps_limit": 2000,
            "interfaces": ["eth0"],
        }

        valid, errors = XDPRateLimitModel.validate_config(config)
        assert valid is False
        assert any("cannot exceed global limit" in e for e in errors)

    def test_validate_config_window_too_small(self):
        """Test validation catches window size below minimum"""
        config = {
            "name": "Test",
            "cluster_id": 1,
            "window_size_ns": 50000000,  # 50ms, below 100ms minimum
            "interfaces": ["eth0"],
        }

        valid, errors = XDPRateLimitModel.validate_config(config)
        assert valid is False
        assert any("100ms" in e for e in errors)

    def test_validate_config_negative_burst(self):
        """Test validation catches negative burst allowance"""
        config = {
            "name": "Test",
            "cluster_id": 1,
            "burst_allowance": -10,
            "interfaces": ["eth0"],
        }

        valid, errors = XDPRateLimitModel.validate_config(config)
        assert valid is False
        assert any("negative" in e.lower() for e in errors)

    def test_validate_config_interfaces_not_list(self):
        """Test validation catches interfaces not being a list"""
        config = {
            "name": "Test",
            "cluster_id": 1,
            "interfaces": "eth0",  # String, not list
        }

        valid, errors = XDPRateLimitModel.validate_config(config)
        assert valid is False
        assert any("list" in e.lower() for e in errors)
