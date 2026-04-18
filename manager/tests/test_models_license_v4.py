"""
Comprehensive unit tests for license.py models - Coverage improvement v4

Tests focus on uncovered branches and edge cases:
- LicenseValidator.check_feature_enabled()
- LicenseValidator.get_proxy_limit()
- LicenseValidator._convert_features_to_dict()
- LicenseValidator enforce_proxy_limits()
- LicenseManager get_license_status_sync()
- Cache expiration logic
- Feature mapping

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import pytest
from datetime import datetime, timedelta
from unittest.mock import MagicMock, patch, AsyncMock

from models.license import (
    LicenseCacheModel,
    LicenseValidator,
    LicenseManager,
)


@pytest.fixture
def mock_db():
    """Create a mock database instance"""
    db = MagicMock()
    db.license_cache = MagicMock()
    db.proxy_servers = MagicMock()
    db.auth_user = MagicMock()
    db.clusters = MagicMock()
    db.services = MagicMock()
    return db


@pytest.fixture
def license_validator():
    """Create a LicenseValidator instance"""
    return LicenseValidator()


class TestLicenseValidatorCheckFeatureEnabled:
    """Tests for LicenseValidator.check_feature_enabled()"""

    def test_check_feature_enabled_invalid_license_denies_all(self, license_validator):
        """Invalid license denies all features"""
        license_data = {"is_valid": False, "features": {}}
        assert license_validator.check_feature_enabled(license_data, "basic_proxy") is False
        assert license_validator.check_feature_enabled(license_data, "multi_cluster") is False

    def test_check_feature_enabled_valid_community_feature(self, license_validator):
        """Valid license enables community features"""
        license_data = {"is_valid": True, "is_enterprise": False, "features": {}}
        assert license_validator.check_feature_enabled(license_data, "basic_proxy") is True
        assert license_validator.check_feature_enabled(license_data, "tcp_proxy") is True
        assert license_validator.check_feature_enabled(license_data, "udp_proxy") is True
        assert license_validator.check_feature_enabled(license_data, "icmp_proxy") is True
        assert license_validator.check_feature_enabled(license_data, "basic_auth") is True
        assert license_validator.check_feature_enabled(license_data, "api_tokens") is True
        assert license_validator.check_feature_enabled(license_data, "single_cluster") is True

    def test_check_feature_enabled_enterprise_feature_no_enterprise(self, license_validator):
        """Enterprise features denied without enterprise license"""
        license_data = {"is_valid": True, "is_enterprise": False, "features": {}}
        assert license_validator.check_feature_enabled(license_data, "multi_cluster") is False
        assert license_validator.check_feature_enabled(license_data, "saml_auth") is False
        assert license_validator.check_feature_enabled(license_data, "oauth2_auth") is False

    def test_check_feature_enabled_enterprise_with_features(self, license_validator):
        """Enterprise license enables specific features"""
        license_data = {
            "is_valid": True,
            "is_enterprise": True,
            "features": {"multi_cluster": True, "saml_auth": True},
        }
        assert license_validator.check_feature_enabled(license_data, "multi_cluster") is True
        assert license_validator.check_feature_enabled(license_data, "saml_auth") is True
        assert license_validator.check_feature_enabled(license_data, "oauth2_auth") is False

    def test_check_feature_enabled_enterprise_without_feature_entitlement(
        self, license_validator
    ):
        """Enterprise features can be disabled"""
        license_data = {
            "is_valid": True,
            "is_enterprise": True,
            "features": {"multi_cluster": False, "saml_auth": True},
        }
        assert license_validator.check_feature_enabled(license_data, "multi_cluster") is False
        assert license_validator.check_feature_enabled(license_data, "saml_auth") is True

    def test_check_feature_enabled_missing_features_dict(self, license_validator):
        """Missing features dict defaults to empty"""
        license_data = {"is_valid": True, "is_enterprise": True}
        assert license_validator.check_feature_enabled(license_data, "multi_cluster") is False

    def test_check_feature_enabled_all_community_features(self, license_validator):
        """All community features recognized"""
        license_data = {"is_valid": True, "features": {}}
        community_features = [
            "basic_proxy",
            "tcp_proxy",
            "udp_proxy",
            "icmp_proxy",
            "basic_auth",
            "api_tokens",
            "single_cluster",
        ]
        for feature in community_features:
            assert license_validator.check_feature_enabled(license_data, feature) is True


class TestLicenseValidatorGetProxyLimit:
    """Tests for LicenseValidator.get_proxy_limit()"""

    def test_get_proxy_limit_invalid_license(self, license_validator):
        """Invalid license returns community default (3)"""
        license_data = {"is_valid": False}
        assert license_validator.get_proxy_limit(license_data) == 3

    def test_get_proxy_limit_no_is_valid_field(self, license_validator):
        """Missing is_valid field defaults to False"""
        license_data = {}
        assert license_validator.get_proxy_limit(license_data) == 3

    def test_get_proxy_limit_valid_enterprise(self, license_validator):
        """Valid enterprise license with proxy limit"""
        license_data = {"is_valid": True, "max_proxies": 50}
        assert license_validator.get_proxy_limit(license_data) == 50

    def test_get_proxy_limit_community(self, license_validator):
        """Community license with lower limit"""
        license_data = {"is_valid": True, "max_proxies": 5}
        assert license_validator.get_proxy_limit(license_data) == 5

    def test_get_proxy_limit_zero(self, license_validator):
        """License with zero proxies"""
        license_data = {"is_valid": True, "max_proxies": 0}
        assert license_validator.get_proxy_limit(license_data) == 0

    def test_get_proxy_limit_missing_max_proxies(self, license_validator):
        """Valid license without max_proxies defaults to 3"""
        license_data = {"is_valid": True}
        assert license_validator.get_proxy_limit(license_data) == 3


class TestConvertFeaturesToDict:
    """Tests for _convert_features_to_dict()"""

    def test_convert_features_empty_list(self, license_validator):
        """Empty features list returns empty dict"""
        result = license_validator._convert_features_to_dict([])
        assert result == {}

    def test_convert_features_single_feature_entitled(self, license_validator):
        """Single entitled feature"""
        features_list = [{"name": "multi_cluster", "entitled": True}]
        result = license_validator._convert_features_to_dict(features_list)
        assert result == {"multi_cluster": True}

    def test_convert_features_single_feature_not_entitled(self, license_validator):
        """Single non-entitled feature"""
        features_list = [{"name": "multi_cluster", "entitled": False}]
        result = license_validator._convert_features_to_dict(features_list)
        assert result == {"multi_cluster": False}

    def test_convert_features_multiple(self, license_validator):
        """Multiple features with mixed entitlements"""
        features_list = [
            {"name": "multi_cluster", "entitled": True},
            {"name": "saml_auth", "entitled": False},
            {"name": "oauth2_auth", "entitled": True},
            {"name": "xdp_rate_limiting", "entitled": True},
        ]
        result = license_validator._convert_features_to_dict(features_list)
        assert result == {
            "multi_cluster": True,
            "saml_auth": False,
            "oauth2_auth": True,
            "xdp_rate_limiting": True,
        }

    def test_convert_features_missing_name(self, license_validator):
        """Feature with missing name"""
        features_list = [{"entitled": True}]
        result = license_validator._convert_features_to_dict(features_list)
        assert "" in result
        assert result[""] is True

    def test_convert_features_missing_entitled(self, license_validator):
        """Feature with missing entitled field defaults to False"""
        features_list = [{"name": "feature1"}]
        result = license_validator._convert_features_to_dict(features_list)
        assert result == {"feature1": False}

    def test_convert_features_all_missing(self, license_validator):
        """Feature object with all fields missing"""
        features_list = [{}]
        result = license_validator._convert_features_to_dict(features_list)
        assert result == {"": False}


class TestEnforceProxyLimits:
    """Tests for LicenseValidator.enforce_proxy_limits()"""

    def test_enforce_proxy_limits_no_cache(self, mock_db, license_validator):
        """No cached license returns False"""
        with patch.object(
            LicenseCacheModel,
            "get_cached_validation",
            return_value=None,
        ):
            result = license_validator.enforce_proxy_limits(mock_db, "unknown-key")
            assert result is False


class TestLicenseManagerInit:
    """Tests for LicenseManager initialization"""

    def test_license_manager_with_key(self, mock_db):
        """LicenseManager initializes with license key"""
        manager = LicenseManager(mock_db, "test-key-123")
        assert manager.license_key == "test-key-123"
        assert manager.db == mock_db
        assert manager.validator is not None

    def test_license_manager_without_key(self, mock_db):
        """LicenseManager initializes without license key (community)"""
        manager = LicenseManager(mock_db)
        assert manager.license_key is None
        assert manager.db == mock_db
        assert manager.validator is not None


class TestLicenseCacheModelOperations:
    """Tests for LicenseCacheModel cache operations"""

    def test_cache_validation_new_entry(self, mock_db):
        """Cache new validation result"""
        db_query = MagicMock()
        db_query.select.return_value.first.return_value = None

        mock_db.side_effect = lambda *args, **kwargs: db_query

        result = LicenseCacheModel.cache_validation(
            mock_db,
            "test-key",
            {"tier": "enterprise", "max_proxies": 10, "features": {}},
            is_valid=True,
        )

        assert result is True
        mock_db.license_cache.insert.assert_called_once()

    def test_cache_validation_enterprise_sets_keepalive(self, mock_db):
        """Enterprise license sets keepalive timestamp"""
        mock_db.return_value = MagicMock()
        mock_db.return_value.select.return_value.first.return_value = None

        result = LicenseCacheModel.cache_validation(
            mock_db,
            "test-key",
            {"tier": "enterprise", "max_proxies": 10, "features": {}},
            is_valid=True,
        )

        assert result is True
        call_kwargs = mock_db.license_cache.insert.call_args[1]
        assert call_kwargs["last_keepalive"] is not None

    def test_cache_validation_community_no_keepalive(self, mock_db):
        """Community license doesn't set keepalive"""
        mock_db.return_value = MagicMock()
        mock_db.return_value.select.return_value.first.return_value = None

        result = LicenseCacheModel.cache_validation(
            mock_db,
            "test-key",
            {"tier": "community", "max_proxies": 3, "features": {}},
            is_valid=True,
        )

        assert result is True
        call_kwargs = mock_db.license_cache.insert.call_args[1]
        assert call_kwargs["last_keepalive"] is None

    def test_cache_validation_update_existing(self, mock_db):
        """Update existing cache entry"""
        existing_entry = MagicMock()
        existing_entry.validation_count = 5

        db_query = MagicMock()
        db_query.select.return_value.first.return_value = existing_entry

        mock_db.side_effect = lambda *args, **kwargs: db_query

        result = LicenseCacheModel.cache_validation(
            mock_db,
            "test-key",
            {"tier": "enterprise", "max_proxies": 20, "features": {}},
            is_valid=True,
        )

        assert result is True
        existing_entry.update_record.assert_called_once()
        call_kwargs = existing_entry.update_record.call_args[1]
        assert call_kwargs["validation_count"] == 6

    def test_get_cached_validation_none(self, mock_db):
        """Get non-existent cache returns None"""
        db_query = MagicMock()
        db_query.select.return_value.first.return_value = None

        mock_db.side_effect = lambda *args, **kwargs: db_query

        result = LicenseCacheModel.get_cached_validation(mock_db, "nonexistent")

        assert result is None

    def test_get_cached_validation_fresh(self, mock_db):
        """Get fresh cache returns data"""
        cache_entry = MagicMock()
        cache_entry.last_validated = datetime.utcnow() - timedelta(minutes=30)
        cache_entry.is_valid = True
        cache_entry.is_enterprise = False
        cache_entry.max_proxies = 10
        cache_entry.features = {}
        cache_entry.validation_data = {}
        cache_entry.expires_at = None
        cache_entry.last_keepalive = None
        cache_entry.keepalive_count = 0

        db_query = MagicMock()
        db_query.select.return_value.first.return_value = cache_entry

        mock_db.side_effect = lambda *args, **kwargs: db_query

        result = LicenseCacheModel.get_cached_validation(mock_db, "fresh-key")

        assert result is not None
        assert result["is_valid"] is True
        assert result["max_proxies"] == 10

    def test_get_cached_validation_expired(self, mock_db):
        """Get expired cache returns None"""
        cache_entry = MagicMock()
        cache_entry.last_validated = datetime.utcnow() - timedelta(hours=2)
        cache_entry.is_valid = True
        cache_entry.is_enterprise = False

        db_query = MagicMock()
        db_query.select.return_value.first.return_value = cache_entry

        mock_db.side_effect = lambda *args, **kwargs: db_query

        result = LicenseCacheModel.get_cached_validation(mock_db, "expired-key")

        assert result is None
