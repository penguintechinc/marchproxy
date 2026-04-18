"""
Unit tests for cluster models

Tests cover ClusterModel and UserClusterAssignmentModel
"""

import pytest
from datetime import datetime, timedelta
from unittest.mock import MagicMock, patch
from models.cluster import (
    ClusterModel,
    UserClusterAssignmentModel,
    CreateClusterRequest,
    UpdateClusterRequest,
)


@pytest.fixture
def mock_db():
    """Create a mock database object"""
    db = MagicMock()
    db.clusters = MagicMock()
    db.proxy_servers = MagicMock()
    db.user_cluster_assignments = MagicMock()
    db.services = MagicMock()
    db.mappings = MagicMock()
    db.certificates = MagicMock()
    return db


class TestClusterModel:
    """Tests for ClusterModel"""

    def test_define_table(self, mock_db):
        """Test table definition"""
        mock_db.define_table = MagicMock(return_value=MagicMock())
        result = ClusterModel.define_table(mock_db)
        mock_db.define_table.assert_called_once()

    def test_generate_api_key(self):
        """Test API key generation"""
        key1 = ClusterModel.generate_api_key()
        key2 = ClusterModel.generate_api_key()

        assert len(key1) > 0
        assert len(key2) > 0
        assert key1 != key2

    def test_hash_api_key(self):
        """Test API key hashing"""
        api_key = "test-api-key-12345"
        hash1 = ClusterModel.hash_api_key(api_key)
        hash2 = ClusterModel.hash_api_key(api_key)

        assert hash1 == hash2
        assert len(hash1) == 64  # SHA256 hex digest length

    def test_verify_api_key_valid(self):
        """Test API key verification - valid key"""
        api_key = "test-api-key"
        api_key_hash = ClusterModel.hash_api_key(api_key)

        result = ClusterModel.verify_api_key(api_key, api_key_hash)

        assert result is True

    def test_verify_api_key_invalid(self):
        """Test API key verification - invalid key"""
        api_key = "test-api-key"
        api_key_hash = ClusterModel.hash_api_key(api_key)

        result = ClusterModel.verify_api_key("wrong-key", api_key_hash)

        assert result is False

    def test_create_cluster(self, mock_db):
        """Test cluster creation"""
        mock_db.clusters.insert = MagicMock(return_value=1)

        cluster_id, api_key = ClusterModel.create_cluster(
            mock_db,
            name="test-cluster",
            description="Test cluster",
            created_by=1,
            max_proxies=5,
        )

        assert cluster_id == 1
        assert len(api_key) > 0
        mock_db.clusters.insert.assert_called_once()

    def test_create_cluster_with_syslog(self, mock_db):
        """Test cluster creation with syslog endpoint"""
        mock_db.clusters.insert = MagicMock(return_value=1)

        cluster_id, api_key = ClusterModel.create_cluster(
            mock_db,
            name="test-cluster",
            syslog_endpoint="syslog.example.com:514",
            created_by=1,
        )

        assert cluster_id == 1
        call_kwargs = mock_db.clusters.insert.call_args[1]
        assert call_kwargs["syslog_endpoint"] == "syslog.example.com:514"

    def test_create_default_cluster_new(self, mock_db):
        """Test creating default cluster when none exists"""
        mock_db.return_value.select.return_value.first.return_value = None
        mock_db.clusters.insert = MagicMock(return_value=1)

        cluster_id, api_key = ClusterModel.create_default_cluster(mock_db, created_by=1)

        assert cluster_id == 1
        assert api_key is not None
        call_kwargs = mock_db.clusters.insert.call_args[1]
        assert call_kwargs["name"] == "default"
        assert call_kwargs["is_default"] is True

    def test_create_default_cluster_existing(self, mock_db):
        """Test creating default cluster when one already exists"""
        existing = MagicMock()
        existing.id = 1
        mock_db.return_value.select.return_value.first.return_value = existing

        cluster_id, api_key = ClusterModel.create_default_cluster(mock_db, created_by=1)

        assert cluster_id == 1
        assert api_key is None
        mock_db.clusters.insert.assert_not_called()

    def test_validate_api_key_valid(self, mock_db):
        """Test API key validation - valid key"""
        api_key = ClusterModel.generate_api_key()
        api_key_hash = ClusterModel.hash_api_key(api_key)

        cluster = MagicMock()
        cluster.id = 1
        cluster.name = "test-cluster"
        cluster.description = "Test"
        cluster.syslog_endpoint = None
        cluster.log_auth = True
        cluster.log_netflow = True
        cluster.log_debug = False
        cluster.max_proxies = 3
        cluster.is_default = False

        mock_db.return_value.select.return_value.first.return_value = cluster

        result = ClusterModel.validate_api_key(mock_db, api_key)

        assert result is not None
        assert result["cluster_id"] == 1
        assert result["name"] == "test-cluster"

    def test_validate_api_key_invalid(self, mock_db):
        """Test API key validation - invalid key"""
        mock_db.return_value.select.return_value.first.return_value = None

        result = ClusterModel.validate_api_key(mock_db, "invalid-key")

        assert result is None

    def test_rotate_api_key(self, mock_db):
        """Test API key rotation"""
        cluster = MagicMock()
        cluster.update_record = MagicMock()
        mock_db.clusters.__getitem__ = MagicMock(return_value=cluster)

        new_api_key = ClusterModel.rotate_api_key(mock_db, cluster_id=1)

        assert len(new_api_key) > 0
        cluster.update_record.assert_called_once()
        call_kwargs = cluster.update_record.call_args[1]
        assert "api_key_hash" in call_kwargs

    def test_rotate_api_key_cluster_not_found(self, mock_db):
        """Test API key rotation with non-existent cluster"""
        mock_db.clusters.__getitem__ = MagicMock(return_value=None)

        new_api_key = ClusterModel.rotate_api_key(mock_db, cluster_id=999)

        assert new_api_key is None

    def test_update_logging_config(self, mock_db):
        """Test updating logging configuration"""
        cluster = MagicMock()
        cluster.update_record = MagicMock()
        mock_db.clusters.__getitem__ = MagicMock(return_value=cluster)

        result = ClusterModel.update_logging_config(
            mock_db,
            cluster_id=1,
            syslog_endpoint="syslog.example.com:514",
            log_auth=False,
        )

        assert result is True
        cluster.update_record.assert_called_once()

    def test_update_logging_config_cluster_not_found(self, mock_db):
        """Test updating logging config for non-existent cluster"""
        mock_db.clusters.__getitem__ = MagicMock(return_value=None)

        result = ClusterModel.update_logging_config(mock_db, cluster_id=999)

        assert result is False

    def test_get_cluster_config_basic(self, mock_db):
        """Test getting cluster configuration"""
        # Create proper mock structure for cluster lookup
        cluster = MagicMock()
        cluster.id = 1
        cluster.name = "test-cluster"
        cluster.syslog_endpoint = None
        cluster.log_auth = True
        cluster.log_netflow = True
        cluster.log_debug = False

        select_result = MagicMock()
        select_result.first.return_value = cluster

        query_result = MagicMock()
        query_result.select.return_value = select_result

        # When db() is called, return our query mock
        mock_db.return_value = query_result
        # When .select() is called on later queries, return empty
        mock_db.services.select = MagicMock(return_value=[])

        result = ClusterModel.get_cluster_config(mock_db, cluster_id=1)

        assert result is not None
        assert "cluster" in result

    def test_get_cluster_config_services(self, mock_db):
        """Test getting cluster config includes services"""
        cluster = MagicMock()
        cluster.id = 1
        cluster.name = "test"
        cluster.syslog_endpoint = None
        cluster.log_auth = True
        cluster.log_netflow = True
        cluster.log_debug = False

        select_result = MagicMock()
        select_result.first.return_value = cluster
        query_result = MagicMock()
        query_result.select.return_value = select_result

        mock_db.return_value = query_result

        result = ClusterModel.get_cluster_config(mock_db, cluster_id=1)

        assert result is not None
        assert "services" in result
        assert "mappings" in result

    def test_check_proxy_limit_within_limit(self, mock_db):
        """Test proxy limit check - within limit"""
        cluster = MagicMock()
        cluster.max_proxies = 5
        mock_db.clusters.__getitem__ = MagicMock(return_value=cluster)

        with patch.object(ClusterModel, "count_active_proxies", return_value=3):
            result = ClusterModel.check_proxy_limit(mock_db, cluster_id=1)
            assert result is True

    def test_check_proxy_limit_at_limit(self, mock_db):
        """Test proxy limit check - at limit"""
        cluster = MagicMock()
        cluster.max_proxies = 3
        mock_db.clusters.__getitem__ = MagicMock(return_value=cluster)

        with patch.object(ClusterModel, "count_active_proxies", return_value=3):
            result = ClusterModel.check_proxy_limit(mock_db, cluster_id=1)
            assert result is False

    def test_check_proxy_limit_cluster_not_found(self, mock_db):
        """Test proxy limit check with non-existent cluster"""
        mock_db.clusters.__getitem__ = MagicMock(return_value=None)

        result = ClusterModel.check_proxy_limit(mock_db, cluster_id=999)

        assert result is False


class TestUserClusterAssignmentModel:
    """Tests for UserClusterAssignmentModel"""

    def test_define_table(self, mock_db):
        """Test table definition"""
        mock_db.define_table = MagicMock(return_value=MagicMock())
        result = UserClusterAssignmentModel.define_table(mock_db)
        mock_db.define_table.assert_called_once()

    def test_assign_user_to_cluster_new(self, mock_db):
        """Test assigning user to cluster - new assignment"""
        mock_db.return_value.select.return_value.first.return_value = None
        mock_db.user_cluster_assignments.insert = MagicMock(return_value=1)

        result = UserClusterAssignmentModel.assign_user_to_cluster(
            mock_db, user_id=1, cluster_id=1, role="admin", assigned_by=2
        )

        assert result is True
        mock_db.user_cluster_assignments.insert.assert_called_once()

    def test_assign_user_to_cluster_existing(self, mock_db):
        """Test assigning user to cluster - existing assignment"""
        existing = MagicMock()
        existing.update_record = MagicMock()
        mock_db.return_value.select.return_value.first.return_value = existing

        result = UserClusterAssignmentModel.assign_user_to_cluster(
            mock_db, user_id=1, cluster_id=1, role="service_owner", assigned_by=2
        )

        assert result is True
        existing.update_record.assert_called_once()

    def test_get_user_clusters(self, mock_db):
        """Test getting clusters for user"""
        assignment = MagicMock()
        assignment.user_cluster_assignments.role = "admin"
        assignment.user_cluster_assignments.assigned_at = datetime.utcnow()
        assignment.clusters.id = 1
        assignment.clusters.name = "test-cluster"
        assignment.clusters.description = "Test"

        mock_db.return_value.select.return_value = [assignment]

        result = UserClusterAssignmentModel.get_user_clusters(mock_db, user_id=1)

        assert len(result) == 1
        assert result[0]["cluster_id"] == 1

    def test_check_user_cluster_access_allowed(self, mock_db):
        """Test checking user cluster access - allowed"""
        assignment = MagicMock()
        assignment.role = "admin"
        mock_db.return_value.select.return_value.first.return_value = assignment

        role = UserClusterAssignmentModel.check_user_cluster_access(
            mock_db, user_id=1, cluster_id=1
        )

        assert role == "admin"

    def test_check_user_cluster_access_denied(self, mock_db):
        """Test checking user cluster access - denied"""
        mock_db.return_value.select.return_value.first.return_value = None

        role = UserClusterAssignmentModel.check_user_cluster_access(
            mock_db, user_id=1, cluster_id=1
        )

        assert role is None


class TestCreateClusterRequest:
    """Tests for CreateClusterRequest Pydantic model"""

    def test_valid_request(self):
        """Test valid cluster creation request"""
        request = CreateClusterRequest(
            name="test-cluster",
            description="Test cluster",
            log_auth=True,
            log_netflow=True,
            log_debug=False,
            max_proxies=5,
        )

        assert request.name == "test-cluster"
        assert request.max_proxies == 5

    def test_name_normalization(self):
        """Test name gets lowercased"""
        request = CreateClusterRequest(name="TEST-CLUSTER")

        assert request.name == "test-cluster"

    def test_name_too_short(self):
        """Test validation for short names"""
        with pytest.raises(ValueError, match="at least 3 characters"):
            CreateClusterRequest(name="ab")

    def test_name_invalid_chars(self):
        """Test validation for invalid characters"""
        with pytest.raises(ValueError, match="alphanumeric"):
            CreateClusterRequest(name="test@cluster")

    def test_name_with_hyphens_underscores(self):
        """Test valid names with hyphens and underscores"""
        request = CreateClusterRequest(name="test-cluster_123")

        assert request.name == "test-cluster_123"

    def test_max_proxies_too_low(self):
        """Test max proxies validation - too low"""
        with pytest.raises(ValueError, match="between 1 and 1000"):
            CreateClusterRequest(name="test", max_proxies=0)

    def test_max_proxies_too_high(self):
        """Test max proxies validation - too high"""
        with pytest.raises(ValueError, match="between 1 and 1000"):
            CreateClusterRequest(name="test", max_proxies=1001)


class TestUpdateClusterRequest:
    """Tests for UpdateClusterRequest Pydantic model"""

    def test_all_optional_fields(self):
        """Test that all fields are optional"""
        request = UpdateClusterRequest()

        assert request.name is None
        assert request.description is None
        assert request.log_auth is None

    def test_partial_update(self):
        """Test partial update"""
        request = UpdateClusterRequest(log_auth=False, max_proxies=10)

        assert request.log_auth is False
        assert request.max_proxies == 10
        assert request.name is None

    def test_max_proxies_validation(self):
        """Test max proxies validation in update"""
        with pytest.raises(ValueError):
            UpdateClusterRequest(max_proxies=1001)
