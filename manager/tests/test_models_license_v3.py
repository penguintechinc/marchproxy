"""
Unit tests for license models

Tests cover LicenseCacheModel, LicenseValidator, and LicenseManager
"""

import pytest
from datetime import datetime, timedelta
from unittest.mock import MagicMock, patch, AsyncMock
from models.license import (
    LicenseCacheModel,
    LicenseValidator,
    LicenseManager,
    LicenseValidationRequest,
)
from models.cluster import ClusterModel


@pytest.fixture
def mock_db():
    """Create a mock database object"""
    db = MagicMock()
    db.license_cache = MagicMock()
    db.proxy_servers = MagicMock()
    db.auth_user = MagicMock()
    db.clusters = MagicMock()
    db.services = MagicMock()
    return db


class TestLicenseCacheModel:
    """Tests for LicenseCacheModel"""

    def test_define_table(self, mock_db):
        """Test table definition"""
        mock_db.define_table = MagicMock(return_value=MagicMock())
        result = LicenseCacheModel.define_table(mock_db)
        mock_db.define_table.assert_called_once()

    def test_cache_validation_new_entry(self, mock_db):
        """Test caching validation result - new entry"""
        mock_db.return_value.select.return_value.first.return_value = None
        mock_db.license_cache.insert = MagicMock(return_value=1)

        validation_data = {
            "tier": "enterprise",
            "max_proxies": 10,
            "features": {"xdp_rate_limiting": True},
        }

        result = LicenseCacheModel.cache_validation(
            mock_db,
            license_key="PENG-TEST-KEY",
            validation_data=validation_data,
            is_valid=True,
            expires_at=datetime.utcnow() + timedelta(days=365),
        )

        assert result is True
        mock_db.license_cache.insert.assert_called_once()

    def test_cache_validation_update_existing(self, mock_db):
        """Test caching validation result - update existing"""
        existing = MagicMock()
        existing.update_record = MagicMock()
        existing.validation_count = 5
        existing.last_keepalive = None

        mock_db.return_value.select.return_value.first.return_value = existing

        validation_data = {
            "tier": "enterprise",
            "max_proxies": 10,
            "features": {},
        }

        result = LicenseCacheModel.cache_validation(
            mock_db,
            license_key="PENG-TEST-KEY",
            validation_data=validation_data,
            is_valid=True,
        )

        assert result is True
        existing.update_record.assert_called_once()

    def test_get_cached_validation_valid(self, mock_db):
        """Test getting valid cached validation"""
        cache_entry = MagicMock()
        cache_entry.is_valid = True
        cache_entry.is_enterprise = True
        cache_entry.max_proxies = 10
        cache_entry.features = {"xdp_rate_limiting": True}
        cache_entry.validation_data = {"valid": True}
        cache_entry.expires_at = datetime.utcnow() + timedelta(days=365)
        cache_entry.last_validated = datetime.utcnow() - timedelta(minutes=30)
        cache_entry.last_keepalive = datetime.utcnow() - timedelta(hours=2)
        cache_entry.keepalive_count = 5

        mock_db.return_value.select.return_value.first.return_value = cache_entry

        result = LicenseCacheModel.get_cached_validation(mock_db, "PENG-TEST-KEY")

        assert result is not None
        assert result["is_valid"] is True
        assert result["is_enterprise"] is True

    def test_get_cached_validation_expired(self, mock_db):
        """Test getting expired cached validation"""
        cache_entry = MagicMock()
        cache_entry.last_validated = datetime.utcnow() - timedelta(hours=2)

        mock_db.return_value.select.return_value.first.return_value = cache_entry

        result = LicenseCacheModel.get_cached_validation(mock_db, "PENG-TEST-KEY")

        assert result is None

    def test_get_cached_validation_missed_keepalive(self, mock_db):
        """Test cached validation with missed keepalive"""
        cache_entry = MagicMock()
        cache_entry.is_enterprise = True
        cache_entry.is_valid = True
        cache_entry.max_proxies = 10
        cache_entry.features = {}
        cache_entry.validation_data = {"valid": True}
        cache_entry.expires_at = None
        cache_entry.last_validated = datetime.utcnow() - timedelta(minutes=30)
        cache_entry.last_keepalive = datetime.utcnow() - timedelta(hours=25)
        cache_entry.keepalive_count = 5

        mock_db.return_value.select.return_value.first.return_value = cache_entry

        result = LicenseCacheModel.get_cached_validation(mock_db, "PENG-TEST-KEY")

        assert result is not None
        assert result["is_valid"] is False
        assert result["keepalive_expired"] is True

    def test_update_keepalive(self, mock_db):
        """Test updating keepalive timestamp"""
        cache_entry = MagicMock()
        cache_entry.keepalive_count = 5
        cache_entry.update_record = MagicMock()

        mock_db.return_value.select.return_value.first.return_value = cache_entry

        result = LicenseCacheModel.update_keepalive(mock_db, "PENG-TEST-KEY")

        assert result is True
        cache_entry.update_record.assert_called_once()

    def test_check_keepalive_health_not_applicable(self, mock_db):
        """Test keepalive health check - not applicable"""
        cache_entry = MagicMock()
        cache_entry.is_enterprise = False

        mock_db.return_value.select.return_value.first.return_value = cache_entry

        result = LicenseCacheModel.check_keepalive_health(mock_db, "PENG-TEST-KEY")

        assert result["status"] == "not_applicable"

    def test_check_keepalive_health_healthy(self, mock_db):
        """Test keepalive health check - healthy"""
        cache_entry = MagicMock()
        cache_entry.is_enterprise = True
        cache_entry.last_keepalive = datetime.utcnow() - timedelta(hours=2)
        cache_entry.keepalive_count = 10

        mock_db.return_value.select.return_value.first.return_value = cache_entry

        result = LicenseCacheModel.check_keepalive_health(mock_db, "PENG-TEST-KEY")

        assert result["status"] == "healthy"

    def test_check_keepalive_health_critical(self, mock_db):
        """Test keepalive health check - critical"""
        cache_entry = MagicMock()
        cache_entry.is_enterprise = True
        cache_entry.last_keepalive = datetime.utcnow() - timedelta(hours=21)
        cache_entry.keepalive_count = 10

        mock_db.return_value.select.return_value.first.return_value = cache_entry

        result = LicenseCacheModel.check_keepalive_health(mock_db, "PENG-TEST-KEY")

        assert result["status"] == "critical"

    def test_check_keepalive_health_expired(self, mock_db):
        """Test keepalive health check - expired"""
        cache_entry = MagicMock()
        cache_entry.is_enterprise = True
        cache_entry.last_keepalive = datetime.utcnow() - timedelta(hours=25)
        cache_entry.keepalive_count = 10

        mock_db.return_value.select.return_value.first.return_value = cache_entry

        result = LicenseCacheModel.check_keepalive_health(mock_db, "PENG-TEST-KEY")

        assert result["status"] == "expired"


class TestLicenseValidator:
    """Tests for LicenseValidator"""

    def test_initialization(self):
        """Test validator initialization"""
        validator = LicenseValidator()

        assert validator.license_server_url == "https://license.penguintech.io"
        assert validator.timeout == 30.0
        assert validator.grace_period_hours == 24

    def test_custom_license_server(self):
        """Test validator with custom license server"""
        validator = LicenseValidator(license_server_url="https://custom.license.io")

        assert validator.license_server_url == "https://custom.license.io"

    @pytest.mark.asyncio
    async def test_validate_license_cached(self, mock_db):
        """Test license validation with cached result"""
        cache_entry = MagicMock()
        cache_entry.is_valid = True
        cache_entry.is_enterprise = True
        cache_entry.max_proxies = 10
        cache_entry.features = {}
        cache_entry.validation_data = {}
        cache_entry.expires_at = None
        cache_entry.last_validated = datetime.utcnow() - timedelta(minutes=30)
        cache_entry.last_keepalive = datetime.utcnow() - timedelta(hours=2)

        mock_db.return_value.select.return_value.first.return_value = cache_entry

        with patch.object(
            LicenseCacheModel,
            "get_cached_validation",
            return_value={"is_valid": True, "is_enterprise": True},
        ):
            validator = LicenseValidator()
            result = await validator.validate_license(mock_db, "PENG-TEST-KEY")

            assert result["is_valid"] is True

    def test_convert_features_to_dict(self):
        """Test feature list to dict conversion"""
        validator = LicenseValidator()

        features_list = [
            {"name": "xdp_rate_limiting", "entitled": True},
            {"name": "saml_auth", "entitled": False},
        ]

        result = validator._convert_features_to_dict(features_list)

        assert result["xdp_rate_limiting"] is True
        assert result["saml_auth"] is False

    def test_check_feature_enabled_community(self):
        """Test feature check - community feature"""
        validator = LicenseValidator()

        license_data = {
            "is_valid": True,
            "is_enterprise": False,
            "features": {},
        }

        result = validator.check_feature_enabled(license_data, "basic_proxy")

        assert result is True

    def test_check_feature_enabled_enterprise(self):
        """Test feature check - enterprise feature"""
        validator = LicenseValidator()

        license_data = {
            "is_valid": True,
            "is_enterprise": True,
            "features": {"xdp_rate_limiting": True},
        }

        result = validator.check_feature_enabled(license_data, "xdp_rate_limiting")

        assert result is True

    def test_check_feature_disabled(self):
        """Test feature check - disabled feature"""
        validator = LicenseValidator()

        license_data = {
            "is_valid": True,
            "is_enterprise": True,
            "features": {"xdp_rate_limiting": False},
        }

        result = validator.check_feature_enabled(license_data, "xdp_rate_limiting")

        assert result is False

    def test_check_feature_community_only(self):
        """Test enterprise feature on community license"""
        validator = LicenseValidator()

        license_data = {
            "is_valid": True,
            "is_enterprise": False,
            "features": {},
        }

        result = validator.check_feature_enabled(license_data, "xdp_rate_limiting")

        assert result is False

    def test_get_proxy_limit_invalid_license(self):
        """Test proxy limit with invalid license"""
        validator = LicenseValidator()

        license_data = {"is_valid": False}

        limit = validator.get_proxy_limit(license_data)

        assert limit == 3  # Community default

    def test_get_proxy_limit_enterprise(self):
        """Test proxy limit with enterprise license"""
        validator = LicenseValidator()

        license_data = {
            "is_valid": True,
            "is_enterprise": True,
            "max_proxies": 100,
        }

        limit = validator.get_proxy_limit(license_data)

        assert limit == 100

    def test_convert_features_empty(self):
        """Test feature conversion with empty list"""
        validator = LicenseValidator()

        features_dict = validator._convert_features_to_dict([])

        assert features_dict == {}

    def test_convert_features_mixed(self):
        """Test feature conversion with mixed features"""
        validator = LicenseValidator()

        features = [
            {"name": "feature1", "entitled": True},
            {"name": "feature2", "entitled": True},
            {"name": "feature3", "entitled": False},
        ]

        result = validator._convert_features_to_dict(features)

        assert len(result) == 3
        assert result["feature1"] is True
        assert result["feature3"] is False


class TestLicenseManager:
    """Tests for LicenseManager"""

    def test_manager_initialization(self, mock_db):
        """Test manager initialization"""
        manager = LicenseManager(mock_db, license_key="PENG-TEST-KEY")

        assert manager.db is mock_db
        assert manager.license_key == "PENG-TEST-KEY"

    def test_manager_community_edition(self, mock_db):
        """Test manager with community edition (no license key)"""
        manager = LicenseManager(mock_db)

        assert manager.license_key is None

    def test_license_manager_initialization(self, mock_db):
        """Test license manager initialization"""
        manager = LicenseManager(mock_db, license_key="PENG-TEST-TEST-TEST-TEST-ABCD")

        assert manager.license_key == "PENG-TEST-TEST-TEST-TEST-ABCD"
        assert manager.validator is not None

    def test_license_manager_community(self, mock_db):
        """Test license manager for community edition"""
        manager = LicenseManager(mock_db)

        assert manager.license_key is None
        assert manager.validator is not None

    def test_schedule_keepalive_community(self, mock_db):
        """Test scheduling keepalive for community edition"""
        manager = LicenseManager(mock_db)
        # Should return immediately for community
        result = manager.schedule_keepalive(interval_hours=1)
        # No return value expected, just verify no error
        assert result is None

    def test_get_keepalive_health_community(self, mock_db):
        """Test keepalive health check for community"""
        manager = LicenseManager(mock_db)
        health = manager.get_keepalive_health()

        assert health["status"] == "not_applicable"
        assert "Community" in health.get("message", "")

    @pytest.mark.asyncio
    async def test_check_proxy_registration_community_within_limit(self, mock_db):
        """Test proxy registration check - community within limit"""
        cluster = MagicMock()
        cluster.max_proxies = 5
        mock_db.clusters.__getitem__ = MagicMock(return_value=cluster)

        query_mock = MagicMock()
        query_mock.count.return_value = 3
        mock_db.return_value = query_mock

        with patch.object(ClusterModel, "check_proxy_limit", return_value=True):
            manager = LicenseManager(mock_db)
            result = await manager.check_proxy_registration(cluster_id=1)

            assert result is True

    @pytest.mark.asyncio
    async def test_check_proxy_registration_community_exceeds_limit(self, mock_db):
        """Test proxy registration check - community exceeds limit"""
        cluster = MagicMock()
        cluster.max_proxies = 3
        mock_db.clusters.__getitem__ = MagicMock(return_value=cluster)

        query_mock = MagicMock()
        query_mock.count.return_value = 3
        mock_db.return_value = query_mock

        with patch.object(ClusterModel, "check_proxy_limit", return_value=False):
            manager = LicenseManager(mock_db)
            result = await manager.check_proxy_registration(cluster_id=1)

            assert result is False

    @pytest.mark.asyncio
    async def test_get_available_features_community(self, mock_db):
        """Test available features - community"""
        manager = LicenseManager(mock_db)
        features = await manager.get_available_features()

        assert "basic_proxy" in features
        assert "tcp_proxy" in features
        assert "xdp_rate_limiting" not in features

    @pytest.mark.asyncio
    async def test_check_feature_enabled_community(self, mock_db):
        """Test feature check - community"""
        manager = LicenseManager(mock_db)
        result = await manager.check_feature_enabled("basic_proxy")

        assert result is True

    @pytest.mark.asyncio
    async def test_check_feature_disabled_community(self, mock_db):
        """Test enterprise feature disabled in community"""
        manager = LicenseManager(mock_db)
        result = await manager.check_feature_enabled("xdp_rate_limiting")

        assert result is False

    def test_get_keepalive_health_community(self, mock_db):
        """Test keepalive health - community edition"""
        manager = LicenseManager(mock_db)
        health = manager.get_keepalive_health()

        assert health["status"] == "not_applicable"


class TestLicenseValidationRequest:
    """Tests for LicenseValidationRequest Pydantic model"""

    def test_valid_license_key(self):
        """Test valid license key format"""
        request = LicenseValidationRequest(license_key="PENG-TEST-TEST-TEST-TEST-ABCD")

        assert request.license_key == "PENG-TEST-TEST-TEST-TEST-ABCD"

    def test_license_key_valid_format(self):
        """Test license key validation"""
        # Test with valid uppercase format
        request = LicenseValidationRequest(license_key="PENG-AAAA-AAAA-AAAA-AAAA-ABCD")

        # Verify it was created successfully
        assert request.license_key is not None
        assert request.license_key.startswith("PENG-")

    def test_license_key_invalid_prefix(self):
        """Test validation with invalid prefix"""
        with pytest.raises(ValueError):
            LicenseValidationRequest(license_key="TEST-TEST-TEST-TEST-TEST-ABCD")

    def test_license_key_wrong_length(self):
        """Test validation with wrong length"""
        with pytest.raises(ValueError):
            LicenseValidationRequest(license_key="PENG-SHORT")

    def test_force_refresh_default(self):
        """Test force_refresh defaults to False"""
        request = LicenseValidationRequest(license_key="PENG-TEST-TEST-TEST-TEST-ABCD")

        assert request.force_refresh is False

    def test_force_refresh_true(self):
        """Test force_refresh can be set to True"""
        request = LicenseValidationRequest(
            license_key="PENG-TEST-TEST-TEST-TEST-ABCD", force_refresh=True
        )

        assert request.force_refresh is True
