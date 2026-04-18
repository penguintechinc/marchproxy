"""
Unit tests for manager/models/proxy.py

Tests ProxyServerModel, ProxyMetricsModel, and ProxyRegistrationRequest.
No real database connections are used.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from unittest.mock import MagicMock, patch

import pytest
from pydantic import ValidationError

from models.proxy import (
    ProxyMetricsModel,
    ProxyRegistrationRequest,
    ProxyServerModel,
)


pytestmark = pytest.mark.unit


# ---------------------------------------------------------------------------
# ProxyRegistrationRequest validation
# ---------------------------------------------------------------------------


class TestProxyRegistrationRequestValidation:
    def test_valid_egress_proxy_type(self):
        req = ProxyRegistrationRequest(
            name="my-proxy",
            hostname="proxy.example.com",
            cluster_api_key="key123",
            proxy_type="egress",
        )
        assert req.proxy_type == "egress"

    def test_valid_ingress_proxy_type(self):
        req = ProxyRegistrationRequest(
            name="my-proxy",
            hostname="proxy.example.com",
            cluster_api_key="key123",
            proxy_type="ingress",
        )
        assert req.proxy_type == "ingress"

    def test_invalid_proxy_type_raises(self):
        with pytest.raises(ValidationError) as exc_info:
            ProxyRegistrationRequest(
                name="my-proxy",
                hostname="proxy.example.com",
                cluster_api_key="key123",
                proxy_type="sidecar",
            )
        assert "egress" in str(exc_info.value).lower() or "ingress" in str(exc_info.value).lower()

    def test_name_minimum_length(self):
        """Name must be at least 3 characters."""
        with pytest.raises(ValidationError):
            ProxyRegistrationRequest(
                name="ab",
                hostname="proxy.example.com",
                cluster_api_key="key123",
            )

    def test_name_exactly_three_chars_valid(self):
        req = ProxyRegistrationRequest(
            name="abc",
            hostname="proxy.example.com",
            cluster_api_key="key123",
        )
        assert req.name == "abc"

    def test_name_invalid_special_chars(self):
        with pytest.raises(ValidationError):
            ProxyRegistrationRequest(
                name="bad name!",
                hostname="proxy.example.com",
                cluster_api_key="key123",
            )

    def test_name_allows_hyphens_and_underscores(self):
        req = ProxyRegistrationRequest(
            name="my-proxy_01",
            hostname="proxy.example.com",
            cluster_api_key="key123",
        )
        assert req.name == "my-proxy_01"

    def test_name_lowercased(self):
        req = ProxyRegistrationRequest(
            name="MyProxy",
            hostname="proxy.example.com",
            cluster_api_key="key123",
        )
        assert req.name == "myproxy"

    def test_port_minimum_valid(self):
        req = ProxyRegistrationRequest(
            name="myproxy",
            hostname="proxy.example.com",
            cluster_api_key="key123",
            port=1,
        )
        assert req.port == 1

    def test_port_maximum_valid(self):
        req = ProxyRegistrationRequest(
            name="myproxy",
            hostname="proxy.example.com",
            cluster_api_key="key123",
            port=65535,
        )
        assert req.port == 65535

    def test_port_zero_invalid(self):
        with pytest.raises(ValidationError):
            ProxyRegistrationRequest(
                name="myproxy",
                hostname="proxy.example.com",
                cluster_api_key="key123",
                port=0,
            )

    def test_port_over_max_invalid(self):
        with pytest.raises(ValidationError):
            ProxyRegistrationRequest(
                name="myproxy",
                hostname="proxy.example.com",
                cluster_api_key="key123",
                port=65536,
            )

    def test_hostname_one_char_valid(self):
        req = ProxyRegistrationRequest(
            name="myproxy",
            hostname="h",
            cluster_api_key="key123",
        )
        assert req.hostname == "h"

    def test_hostname_empty_invalid(self):
        with pytest.raises(ValidationError):
            ProxyRegistrationRequest(
                name="myproxy",
                hostname="",
                cluster_api_key="key123",
            )

    def test_hostname_253_chars_valid(self):
        hostname = "a" * 253
        req = ProxyRegistrationRequest(
            name="myproxy",
            hostname=hostname,
            cluster_api_key="key123",
        )
        assert len(req.hostname) == 253

    def test_hostname_254_chars_invalid(self):
        with pytest.raises(ValidationError):
            ProxyRegistrationRequest(
                name="myproxy",
                hostname="a" * 254,
                cluster_api_key="key123",
            )

    def test_default_port(self):
        req = ProxyRegistrationRequest(
            name="myproxy",
            hostname="proxy.example.com",
            cluster_api_key="key123",
        )
        assert req.port == 8080

    def test_default_proxy_type(self):
        req = ProxyRegistrationRequest(
            name="myproxy",
            hostname="proxy.example.com",
            cluster_api_key="key123",
        )
        assert req.proxy_type == "egress"


# ---------------------------------------------------------------------------
# ProxyServerModel.register_proxy — success path
# ---------------------------------------------------------------------------


class TestProxyRegistrationSuccess:
    def test_register_new_proxy_returns_proxy_id(self, mock_db):
        """register_proxy returns the new proxy's ID on success."""
        cluster_info = {"cluster_id": 7}
        mock_db.proxy_servers.insert.return_value = 42

        # No existing proxy found
        mock_db.return_value.select.return_value.first.return_value = None

        with (
            patch("models.cluster.ClusterModel.validate_api_key", return_value=cluster_info),
            patch("models.cluster.ClusterModel.check_proxy_limit", return_value=True),
            patch("models.proxy.socket.gethostbyname", return_value="192.168.1.10"),
        ):
            result = ProxyServerModel.register_proxy(
                db=mock_db,
                name="my-proxy",
                hostname="proxy.example.com",
                cluster_api_key="valid-key",
            )

        assert result == 42
        mock_db.proxy_servers.insert.assert_called_once()

    def test_register_with_explicit_ip_skips_resolution(self, mock_db):
        """Explicit ip_address bypasses socket.gethostbyname."""
        cluster_info = {"cluster_id": 5}
        mock_db.proxy_servers.insert.return_value = 99
        mock_db.return_value.select.return_value.first.return_value = None

        with (
            patch("models.cluster.ClusterModel.validate_api_key", return_value=cluster_info),
            patch("models.cluster.ClusterModel.check_proxy_limit", return_value=True),
            patch("models.proxy.socket.gethostbyname") as mock_resolve,
        ):
            result = ProxyServerModel.register_proxy(
                db=mock_db,
                name="my-proxy",
                hostname="proxy.example.com",
                cluster_api_key="valid-key",
                ip_address="10.0.0.1",
            )

        mock_resolve.assert_not_called()
        assert result == 99

    def test_register_existing_proxy_returns_existing_id(self, mock_db):
        """When a proxy with matching name/hostname exists, update and return existing id."""
        cluster_info = {"cluster_id": 3}
        existing_proxy = MagicMock()
        existing_proxy.id = 55
        mock_db.return_value.select.return_value.first.return_value = existing_proxy

        with (
            patch("models.cluster.ClusterModel.validate_api_key", return_value=cluster_info),
            patch("models.cluster.ClusterModel.check_proxy_limit", return_value=True),
            patch("models.proxy.socket.gethostbyname", return_value="10.1.2.3"),
        ):
            result = ProxyServerModel.register_proxy(
                db=mock_db,
                name="my-proxy",
                hostname="proxy.example.com",
                cluster_api_key="valid-key",
            )

        assert result == 55
        existing_proxy.update_record.assert_called_once()


# ---------------------------------------------------------------------------
# ProxyServerModel.register_proxy — failure paths
# ---------------------------------------------------------------------------


class TestProxyRegistrationFailure:
    def test_invalid_cluster_key_returns_none(self, mock_db):
        with patch("models.cluster.ClusterModel.validate_api_key", return_value=None):
            result = ProxyServerModel.register_proxy(
                db=mock_db,
                name="my-proxy",
                hostname="proxy.example.com",
                cluster_api_key="bad-key",
            )
        assert result is None

    def test_proxy_limit_exceeded_returns_none(self, mock_db):
        cluster_info = {"cluster_id": 2}
        with (
            patch("models.cluster.ClusterModel.validate_api_key", return_value=cluster_info),
            patch("models.cluster.ClusterModel.check_proxy_limit", return_value=False),
        ):
            result = ProxyServerModel.register_proxy(
                db=mock_db,
                name="my-proxy",
                hostname="proxy.example.com",
                cluster_api_key="valid-key",
            )
        assert result is None

    def test_hostname_resolution_failure_returns_none(self, mock_db):
        import socket as _socket

        cluster_info = {"cluster_id": 1}
        with (
            patch("models.cluster.ClusterModel.validate_api_key", return_value=cluster_info),
            patch("models.cluster.ClusterModel.check_proxy_limit", return_value=True),
            patch("models.proxy.socket.gethostbyname", side_effect=_socket.gaierror),
        ):
            result = ProxyServerModel.register_proxy(
                db=mock_db,
                name="my-proxy",
                hostname="nonexistent.invalid",
                cluster_api_key="valid-key",
            )
        assert result is None


# ---------------------------------------------------------------------------
# ProxyServerModel.update_heartbeat
# ---------------------------------------------------------------------------


class TestUpdateHeartbeat:
    def test_success_returns_true(self, mock_db):
        cluster_info = {"cluster_id": 1}
        proxy = MagicMock()
        mock_db.return_value.select.return_value.first.return_value = proxy

        with patch("models.cluster.ClusterModel.validate_api_key", return_value=cluster_info):
            result = ProxyServerModel.update_heartbeat(
                db=mock_db,
                proxy_name="my-proxy",
                cluster_api_key="valid-key",
                status_data={"version": "1.2.3"},
            )

        assert result is True
        proxy.update_record.assert_called_once()

    def test_invalid_cluster_key_returns_false(self, mock_db):
        with patch("models.cluster.ClusterModel.validate_api_key", return_value=None):
            result = ProxyServerModel.update_heartbeat(
                db=mock_db,
                proxy_name="my-proxy",
                cluster_api_key="bad-key",
            )
        assert result is False

    def test_proxy_not_found_returns_false(self, mock_db):
        cluster_info = {"cluster_id": 1}
        mock_db.return_value.select.return_value.first.return_value = None

        with patch("models.cluster.ClusterModel.validate_api_key", return_value=cluster_info):
            result = ProxyServerModel.update_heartbeat(
                db=mock_db,
                proxy_name="ghost-proxy",
                cluster_api_key="valid-key",
            )
        assert result is False

    def test_status_data_version_propagated(self, mock_db):
        cluster_info = {"cluster_id": 1}
        proxy = MagicMock()
        mock_db.return_value.select.return_value.first.return_value = proxy

        with patch("models.cluster.ClusterModel.validate_api_key", return_value=cluster_info):
            ProxyServerModel.update_heartbeat(
                db=mock_db,
                proxy_name="my-proxy",
                cluster_api_key="valid-key",
                status_data={"version": "2.0.0", "config_version": "abc"},
            )

        call_kwargs = proxy.update_record.call_args[1]
        assert call_kwargs.get("version") == "2.0.0"
        assert call_kwargs.get("config_version") == "abc"


# ---------------------------------------------------------------------------
# ProxyServerModel.validate_license
# ---------------------------------------------------------------------------


class TestValidateLicense:
    def test_set_license_valid_true(self, mock_db):
        proxy = MagicMock()
        mock_db.proxy_servers.__getitem__ = MagicMock(return_value=proxy)

        result = ProxyServerModel.validate_license(
            db=mock_db, proxy_id=1, license_valid=True
        )

        assert result is True
        call_kwargs = proxy.update_record.call_args[1]
        assert call_kwargs.get("license_validated") is True

    def test_set_license_valid_false(self, mock_db):
        proxy = MagicMock()
        mock_db.proxy_servers.__getitem__ = MagicMock(return_value=proxy)

        result = ProxyServerModel.validate_license(
            db=mock_db, proxy_id=1, license_valid=False
        )

        assert result is True
        call_kwargs = proxy.update_record.call_args[1]
        assert call_kwargs.get("license_validated") is False

    def test_proxy_not_found_returns_false(self, mock_db):
        mock_db.proxy_servers.__getitem__ = MagicMock(return_value=None)

        result = ProxyServerModel.validate_license(
            db=mock_db, proxy_id=999, license_valid=True
        )

        assert result is False


# ---------------------------------------------------------------------------
# ProxyServerModel.cleanup_stale_proxies
# ---------------------------------------------------------------------------


class TestCleanupStaleProxies:
    def test_returns_count_of_updated_rows(self, mock_db):
        mock_db.return_value.update.return_value = 3

        result = ProxyServerModel.cleanup_stale_proxies(db=mock_db, timeout_minutes=10)

        assert result == 3

    def test_calls_db_update(self, mock_db):
        mock_db.return_value.update.return_value = 0

        ProxyServerModel.cleanup_stale_proxies(db=mock_db)

        mock_db.return_value.update.assert_called_once_with(status="inactive")

    def test_default_timeout_is_ten_minutes(self, mock_db):
        """Verify the method accepts default timeout without error."""
        mock_db.return_value.update.return_value = 0
        # Should not raise
        result = ProxyServerModel.cleanup_stale_proxies(db=mock_db)
        assert isinstance(result, int)


# ---------------------------------------------------------------------------
# ProxyServerModel.get_proxy_stats
# ---------------------------------------------------------------------------


class TestGetProxyStats:
    def test_returns_dict_with_required_keys(self, mock_db):
        # mock_db.proxy_servers is a table mock; .count() called on it
        mock_db.proxy_servers.count = MagicMock(return_value=10)
        mock_db.return_value.count.return_value = 5

        result = ProxyServerModel.get_proxy_stats(db=mock_db)

        assert isinstance(result, dict)
        for key in ("total", "active", "inactive", "pending"):
            assert key in result

    def test_returns_integer_values(self, mock_db):
        # total: db.proxy_servers.count()
        mock_db.proxy_servers.count = MagicMock(return_value=4)
        # active/inactive/pending: db.proxy_servers(condition).count()
        mock_db.proxy_servers.return_value.count.return_value = 2

        result = ProxyServerModel.get_proxy_stats(db=mock_db)

        for key in ("total", "active", "inactive", "pending"):
            assert isinstance(result[key], int)

    def test_with_cluster_id_filter(self, mock_db):
        """Passing cluster_id should not raise and still return the expected shape."""
        mock_db.return_value.count.return_value = 1

        result = ProxyServerModel.get_proxy_stats(db=mock_db, cluster_id=3)

        assert isinstance(result, dict)
        assert "total" in result


# ---------------------------------------------------------------------------
# ProxyMetricsModel
# ---------------------------------------------------------------------------


class TestProxyMetricsModel:
    def test_record_metrics_calls_insert(self, mock_db):
        mock_db.proxy_metrics = MagicMock()
        mock_db.proxy_metrics.insert.return_value = 1

        metrics = {
            "cpu_usage": 45.2,
            "memory_usage": 60.1,
            "connections_active": 100,
            "connections_total": 5000,
            "bytes_sent": 1024 * 1024,
            "bytes_received": 512 * 1024,
            "requests_per_second": 200.5,
            "latency_avg": 1.5,
            "latency_p95": 3.2,
            "errors_per_second": 0.1,
        }

        result = ProxyMetricsModel.record_metrics(db=mock_db, proxy_id=7, metrics=metrics)

        mock_db.proxy_metrics.insert.assert_called_once()
        call_kwargs = mock_db.proxy_metrics.insert.call_args[1]
        assert call_kwargs.get("proxy_id") == 7
        assert call_kwargs.get("cpu_usage") == 45.2

    def test_record_metrics_returns_insert_value(self, mock_db):
        mock_db.proxy_metrics = MagicMock()
        mock_db.proxy_metrics.insert.return_value = 42

        result = ProxyMetricsModel.record_metrics(db=mock_db, proxy_id=1, metrics={})

        assert result == 42

    def test_record_metrics_with_empty_metrics(self, mock_db):
        """record_metrics should not raise when metrics dict is empty."""
        mock_db.proxy_metrics = MagicMock()
        mock_db.proxy_metrics.insert.return_value = 1

        # Should not raise
        ProxyMetricsModel.record_metrics(db=mock_db, proxy_id=1, metrics={})

        mock_db.proxy_metrics.insert.assert_called_once()

    def test_get_metrics_returns_list(self, mock_db):
        row1 = MagicMock()
        row1.__iter__ = MagicMock(return_value=iter([("cpu_usage", 30.0)]))
        metrics_rows = [row1]

        select_result = MagicMock()
        select_result.__iter__ = MagicMock(return_value=iter(metrics_rows))
        mock_db.return_value.select.return_value = select_result

        # dict(metric) is called per row; patch to return a real dict
        with patch("models.proxy.dict", side_effect=lambda x: {"cpu_usage": 30.0}):
            result = ProxyMetricsModel.get_metrics(db=mock_db, proxy_id=5)

        assert isinstance(result, list)

    def test_get_metrics_default_hours(self, mock_db):
        """get_metrics with default hours=24 should not raise."""
        select_result = MagicMock()
        select_result.__iter__ = MagicMock(return_value=iter([]))
        mock_db.return_value.select.return_value = select_result

        result = ProxyMetricsModel.get_metrics(db=mock_db, proxy_id=1)

        assert isinstance(result, list)
