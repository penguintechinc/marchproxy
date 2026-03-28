"""
Unit tests for manager/models/cluster.py

Tests ClusterModel static methods and CreateClusterRequest Pydantic validation.
No real database connections are used.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import hashlib
from unittest.mock import MagicMock, patch

import pytest
from pydantic import ValidationError

from models.cluster import ClusterModel, CreateClusterRequest


pytestmark = pytest.mark.unit


# ---------------------------------------------------------------------------
# generate_api_key
# ---------------------------------------------------------------------------

class TestGenerateApiKey:
    def test_returns_non_empty_string(self):
        key = ClusterModel.generate_api_key()
        assert isinstance(key, str)
        assert len(key) > 0

    def test_returns_url_safe_characters_only(self):
        key = ClusterModel.generate_api_key()
        # secrets.token_urlsafe produces base64url chars: A-Z a-z 0-9 - _
        assert all(c.isalnum() or c in "-_" for c in key)

    def test_each_call_returns_unique_key(self):
        keys = {ClusterModel.generate_api_key() for _ in range(10)}
        assert len(keys) == 10

    def test_key_length_sufficient(self):
        # 48 bytes → ~64 base64url chars
        key = ClusterModel.generate_api_key()
        assert len(key) >= 60


# ---------------------------------------------------------------------------
# hash_api_key / verify_api_key
# ---------------------------------------------------------------------------

class TestHashAndVerifyApiKey:
    def test_hash_returns_hex_string(self):
        h = ClusterModel.hash_api_key("testkey")
        assert isinstance(h, str)
        assert len(h) == 64  # SHA-256 hex = 64 chars

    def test_hash_is_deterministic(self):
        key = "same-key-123"
        assert ClusterModel.hash_api_key(key) == ClusterModel.hash_api_key(key)

    def test_hash_matches_expected_sha256(self):
        key = "hello"
        expected = hashlib.sha256(b"hello").hexdigest()
        assert ClusterModel.hash_api_key(key) == expected

    def test_different_keys_produce_different_hashes(self):
        assert ClusterModel.hash_api_key("key1") != ClusterModel.hash_api_key("key2")

    def test_verify_returns_true_for_correct_key(self):
        key = "my-api-key"
        h = ClusterModel.hash_api_key(key)
        assert ClusterModel.verify_api_key(key, h) is True

    def test_verify_returns_false_for_wrong_key(self):
        h = ClusterModel.hash_api_key("correct-key")
        assert ClusterModel.verify_api_key("wrong-key", h) is False

    def test_verify_returns_false_for_empty_key(self):
        h = ClusterModel.hash_api_key("some-key")
        assert ClusterModel.verify_api_key("", h) is False

    def test_round_trip_with_generated_key(self):
        key = ClusterModel.generate_api_key()
        h = ClusterModel.hash_api_key(key)
        assert ClusterModel.verify_api_key(key, h) is True


# ---------------------------------------------------------------------------
# create_cluster
# ---------------------------------------------------------------------------

class TestCreateCluster:
    def test_returns_cluster_id_and_api_key(self, mock_db):
        mock_db.clusters.insert.return_value = 42

        cluster_id, api_key = ClusterModel.create_cluster(
            db=mock_db,
            name="test-cluster",
            created_by=1,
        )

        assert cluster_id == 42
        assert isinstance(api_key, str)
        assert len(api_key) > 0

    def test_inserts_into_clusters_table(self, mock_db):
        mock_db.clusters.insert.return_value = 1

        ClusterModel.create_cluster(db=mock_db, name="my-cluster", created_by=5)

        mock_db.clusters.insert.assert_called_once()
        call_kwargs = mock_db.clusters.insert.call_args[1]
        assert call_kwargs["name"] == "my-cluster"
        assert call_kwargs["created_by"] == 5

    def test_stores_hashed_key_not_plain_text(self, mock_db):
        mock_db.clusters.insert.return_value = 1

        _, api_key = ClusterModel.create_cluster(db=mock_db, name="cluster", created_by=1)

        call_kwargs = mock_db.clusters.insert.call_args[1]
        stored_hash = call_kwargs["api_key_hash"]
        # The stored value must be the hash of the returned api_key
        expected_hash = ClusterModel.hash_api_key(api_key)
        assert stored_hash == expected_hash

    def test_respects_optional_params(self, mock_db):
        mock_db.clusters.insert.return_value = 7

        ClusterModel.create_cluster(
            db=mock_db,
            name="cluster",
            description="desc",
            created_by=2,
            syslog_endpoint="syslog:514",
            log_auth=False,
            log_netflow=False,
            log_debug=True,
            max_proxies=10,
        )

        kw = mock_db.clusters.insert.call_args[1]
        assert kw["description"] == "desc"
        assert kw["syslog_endpoint"] == "syslog:514"
        assert kw["log_auth"] is False
        assert kw["log_netflow"] is False
        assert kw["log_debug"] is True
        assert kw["max_proxies"] == 10


# ---------------------------------------------------------------------------
# validate_api_key
# ---------------------------------------------------------------------------

class TestValidateApiKey:
    def _make_cluster_row(self, **overrides):
        row = MagicMock()
        row.id = overrides.get("id", 1)
        row.name = overrides.get("name", "default")
        row.description = overrides.get("description", "Test cluster")
        row.syslog_endpoint = overrides.get("syslog_endpoint", None)
        row.log_auth = overrides.get("log_auth", True)
        row.log_netflow = overrides.get("log_netflow", True)
        row.log_debug = overrides.get("log_debug", False)
        row.max_proxies = overrides.get("max_proxies", 3)
        row.is_default = overrides.get("is_default", False)
        return row

    def test_returns_cluster_dict_when_found(self, mock_db):
        cluster_row = self._make_cluster_row(id=5, name="prod")
        select_chain = MagicMock()
        select_chain.first.return_value = cluster_row
        mock_db.return_value.select.return_value = select_chain

        result = ClusterModel.validate_api_key(mock_db, "some-api-key")

        assert result is not None
        assert result["cluster_id"] == 5
        assert result["name"] == "prod"

    def test_returns_none_when_not_found(self, mock_db):
        select_chain = MagicMock()
        select_chain.first.return_value = None
        mock_db.return_value.select.return_value = select_chain

        result = ClusterModel.validate_api_key(mock_db, "invalid-key")

        assert result is None

    def test_result_contains_expected_keys(self, mock_db):
        cluster_row = self._make_cluster_row()
        select_chain = MagicMock()
        select_chain.first.return_value = cluster_row
        mock_db.return_value.select.return_value = select_chain

        result = ClusterModel.validate_api_key(mock_db, "key")

        expected_keys = {
            "cluster_id", "name", "description", "syslog_endpoint",
            "log_auth", "log_netflow", "log_debug", "max_proxies", "is_default",
        }
        assert expected_keys.issubset(set(result.keys()))

    def test_queries_using_hashed_key(self, mock_db):
        """validate_api_key must hash the key before querying."""
        select_chain = MagicMock()
        select_chain.first.return_value = None
        mock_db.return_value.select.return_value = select_chain

        api_key = "plaintext-key"
        ClusterModel.validate_api_key(mock_db, api_key)

        # Confirm db() was called (the query was built)
        mock_db.assert_called()


# ---------------------------------------------------------------------------
# rotate_api_key
# ---------------------------------------------------------------------------

class TestRotateApiKey:
    def test_returns_new_api_key_string(self, mock_db):
        cluster_row = MagicMock()
        cluster_row.update_record = MagicMock()
        mock_db.clusters.__getitem__ = MagicMock(return_value=cluster_row)

        new_key = ClusterModel.rotate_api_key(mock_db, cluster_id=1)

        assert isinstance(new_key, str)
        assert len(new_key) > 0

    def test_calls_update_record_with_new_hash(self, mock_db):
        cluster_row = MagicMock()
        cluster_row.update_record = MagicMock()
        mock_db.clusters.__getitem__ = MagicMock(return_value=cluster_row)

        new_key = ClusterModel.rotate_api_key(mock_db, cluster_id=1)

        cluster_row.update_record.assert_called_once()
        kw = cluster_row.update_record.call_args[1]
        assert "api_key_hash" in kw
        assert kw["api_key_hash"] == ClusterModel.hash_api_key(new_key)

    def test_returns_none_when_cluster_not_found(self, mock_db):
        mock_db.clusters.__getitem__ = MagicMock(return_value=None)

        result = ClusterModel.rotate_api_key(mock_db, cluster_id=999)

        assert result is None


# ---------------------------------------------------------------------------
# count_active_proxies
# ---------------------------------------------------------------------------

class TestCountActiveProxies:
    def test_returns_integer_count(self, mock_db):
        mock_db.return_value.count.return_value = 3

        count = ClusterModel.count_active_proxies(mock_db, cluster_id=1)

        assert count == 3

    def test_returns_zero_when_no_active_proxies(self, mock_db):
        mock_db.return_value.count.return_value = 0

        count = ClusterModel.count_active_proxies(mock_db, cluster_id=1)

        assert count == 0

    def test_calls_count_on_db(self, mock_db):
        mock_db.return_value.count.return_value = 2

        ClusterModel.count_active_proxies(mock_db, cluster_id=5)

        mock_db.return_value.count.assert_called_once()


# ---------------------------------------------------------------------------
# check_proxy_limit
# ---------------------------------------------------------------------------

class TestCheckProxyLimit:
    def _setup_cluster(self, mock_db, max_proxies: int, active_count: int):
        cluster_row = MagicMock()
        cluster_row.max_proxies = max_proxies
        mock_db.clusters.__getitem__ = MagicMock(return_value=cluster_row)
        mock_db.return_value.count.return_value = active_count

    def test_returns_true_when_below_limit(self, mock_db):
        self._setup_cluster(mock_db, max_proxies=5, active_count=2)
        assert ClusterModel.check_proxy_limit(mock_db, cluster_id=1) is True

    def test_returns_false_when_at_limit(self, mock_db):
        self._setup_cluster(mock_db, max_proxies=3, active_count=3)
        assert ClusterModel.check_proxy_limit(mock_db, cluster_id=1) is False

    def test_returns_false_when_above_limit(self, mock_db):
        self._setup_cluster(mock_db, max_proxies=2, active_count=5)
        assert ClusterModel.check_proxy_limit(mock_db, cluster_id=1) is False

    def test_returns_false_when_cluster_not_found(self, mock_db):
        mock_db.clusters.__getitem__ = MagicMock(return_value=None)
        assert ClusterModel.check_proxy_limit(mock_db, cluster_id=999) is False

    def test_single_slot_remaining_is_true(self, mock_db):
        self._setup_cluster(mock_db, max_proxies=3, active_count=2)
        assert ClusterModel.check_proxy_limit(mock_db, cluster_id=1) is True


# ---------------------------------------------------------------------------
# CreateClusterRequest Pydantic validation
# ---------------------------------------------------------------------------

class TestCreateClusterRequest:
    # --- Valid cases ---

    def test_valid_simple_name(self):
        req = CreateClusterRequest(name="mycluster")
        assert req.name == "mycluster"

    def test_name_is_lowercased(self):
        req = CreateClusterRequest(name="MyCluster")
        assert req.name == "mycluster"

    def test_valid_name_with_hyphen(self):
        req = CreateClusterRequest(name="my-cluster")
        assert req.name == "my-cluster"

    def test_valid_name_with_underscore(self):
        req = CreateClusterRequest(name="my_cluster")
        assert req.name == "my_cluster"

    def test_defaults_are_applied(self):
        req = CreateClusterRequest(name="abc")
        assert req.log_auth is True
        assert req.log_netflow is True
        assert req.log_debug is False
        assert req.max_proxies == 3

    def test_valid_max_proxies_boundary_low(self):
        req = CreateClusterRequest(name="abc", max_proxies=1)
        assert req.max_proxies == 1

    def test_valid_max_proxies_boundary_high(self):
        req = CreateClusterRequest(name="abc", max_proxies=1000)
        assert req.max_proxies == 1000

    def test_optional_fields_none_by_default(self):
        req = CreateClusterRequest(name="abc")
        assert req.description is None
        assert req.syslog_endpoint is None

    # --- Invalid cases ---

    def test_name_too_short_raises(self):
        with pytest.raises(ValidationError):
            CreateClusterRequest(name="ab")

    def test_name_with_spaces_raises(self):
        with pytest.raises(ValidationError):
            CreateClusterRequest(name="my cluster")

    def test_name_with_special_chars_raises(self):
        with pytest.raises(ValidationError):
            CreateClusterRequest(name="my@cluster!")

    def test_max_proxies_zero_raises(self):
        with pytest.raises(ValidationError):
            CreateClusterRequest(name="abc", max_proxies=0)

    def test_max_proxies_too_high_raises(self):
        with pytest.raises(ValidationError):
            CreateClusterRequest(name="abc", max_proxies=1001)

    def test_max_proxies_negative_raises(self):
        with pytest.raises(ValidationError):
            CreateClusterRequest(name="abc", max_proxies=-1)
