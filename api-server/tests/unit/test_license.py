"""Unit tests for app/core/license.py"""
import pytest
from unittest.mock import patch, MagicMock, AsyncMock


@pytest.mark.asyncio
async def test_license_validator_dev_mode_enterprise():
    """When RELEASE_MODE=False, validator returns Enterprise tier."""
    from app.core.license import LicenseValidator, LicenseTier

    validator = LicenseValidator()
    # Default RELEASE_MODE is False in test env
    validator.release_mode = False
    license_info = await validator.validate_license()
    assert license_info.tier == LicenseTier.ENTERPRISE


@pytest.mark.asyncio
async def test_license_validator_dev_mode_all_features():
    from app.core.license import LicenseValidator

    validator = LicenseValidator()
    validator.release_mode = False
    license_info = await validator.validate_license()
    assert "all" in license_info.features


@pytest.mark.asyncio
async def test_license_validator_dev_mode_proxy_limit_unlimited():
    from app.core.license import LicenseValidator

    validator = LicenseValidator()
    validator.release_mode = False
    # check_proxy_limit is async — current_count < max_proxies
    result = await validator.check_proxy_limit(9999)
    assert result is True


@pytest.mark.asyncio
async def test_license_validator_dev_mode_proxy_limit_valid():
    from app.core.license import LicenseValidator

    validator = LicenseValidator()
    validator.release_mode = False
    result = await validator.check_proxy_limit(0)
    assert result is True


@pytest.mark.asyncio
async def test_license_validator_dev_mode_check_feature_any():
    from app.core.license import LicenseValidator

    validator = LicenseValidator()
    validator.release_mode = False
    result = await validator.check_feature("any_feature")
    assert result is True


@pytest.mark.asyncio
async def test_license_validator_dev_mode_check_feature_arbitrary():
    from app.core.license import LicenseValidator

    validator = LicenseValidator()
    validator.release_mode = False
    result = await validator.check_feature("saml_authentication")
    assert result is True


def test_license_tier_values():
    from app.core.license import LicenseTier
    assert LicenseTier.COMMUNITY.value == "community"
    assert LicenseTier.ENTERPRISE.value == "enterprise"


def test_license_info_model():
    from app.core.license import LicenseInfo, LicenseTier
    info = LicenseInfo(
        tier=LicenseTier.COMMUNITY,
        max_proxies=3,
        features=[],
        valid_until=None,
        is_valid=True
    )
    assert info.tier == LicenseTier.COMMUNITY
    assert info.max_proxies == 3
    assert info.is_valid is True
    assert info.features == []


def test_license_info_enterprise_model():
    from app.core.license import LicenseInfo, LicenseTier
    info = LicenseInfo(
        tier=LicenseTier.ENTERPRISE,
        max_proxies=999999,
        features=["all"],
        is_valid=True
    )
    assert info.tier == LicenseTier.ENTERPRISE
    assert info.max_proxies == 999999
    assert "all" in info.features


@pytest.mark.asyncio
async def test_license_validator_release_mode_no_key_community():
    """In release mode with no key, should return Community tier."""
    from app.core.license import LicenseValidator, LicenseTier

    validator = LicenseValidator()
    validator.release_mode = True
    validator.license_key = ""

    license_info = await validator.validate_license()
    assert license_info.tier == LicenseTier.COMMUNITY
    assert license_info.max_proxies == 3  # COMMUNITY_MAX_PROXIES default


@pytest.mark.asyncio
async def test_license_validator_caching():
    """Second call returns cached result without hitting validator again."""
    from app.core.license import LicenseValidator, LicenseTier

    validator = LicenseValidator()
    validator.release_mode = False

    result1 = await validator.validate_license()
    result2 = await validator.validate_license()
    # Both should be enterprise in dev mode
    assert result1.tier == LicenseTier.ENTERPRISE
    assert result2.tier == LicenseTier.ENTERPRISE


@pytest.mark.asyncio
async def test_license_manager_dev_mode_returns_dict():
    """LicenseManager.validate_license returns dict with expected keys."""
    from app.core.license import LicenseManager

    manager = LicenseManager()
    result = await manager.validate_license("any-key")
    assert isinstance(result, dict)
    assert "valid" in result
    assert "tier" in result
    assert "max_proxies" in result
    assert "features" in result
