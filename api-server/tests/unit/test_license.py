"""Unit tests for app/core/license.py (wraps penguin-licensing)"""
from datetime import datetime, timezone # noqa: F401
from unittest.mock import AsyncMock, MagicMock, patch # noqa: F401

import pytest # noqa: F401, # noqa: F401
from penguin_licensing import Feature # noqa: F401
from penguin_licensing import LicenseInfo as PenguinLicenseInfo # noqa: F401


@pytest.mark.asyncio
async def test_license_validator_dev_mode_enterprise():
    """When RELEASE_MODE=False, validator returns Enterprise tier."""
    from app.core.license import LicenseTier, LicenseValidator # noqa: F401

    validator = LicenseValidator()
    validator.release_mode = False
    license_info = await validator.validate_license()
    assert license_info.tier == LicenseTier.ENTERPRISE


@pytest.mark.asyncio
async def test_license_validator_dev_mode_all_features():
    from app.core.license import LicenseValidator # noqa: F401

    validator = LicenseValidator()
    validator.release_mode = False
    license_info = await validator.validate_license()
    assert "all" in license_info.features


@pytest.mark.asyncio
async def test_license_validator_dev_mode_proxy_limit_unlimited():
    from app.core.license import LicenseValidator # noqa: F401

    validator = LicenseValidator()
    validator.release_mode = False
    result = await validator.check_proxy_limit(9999)
    assert result is True


@pytest.mark.asyncio
async def test_license_validator_dev_mode_proxy_limit_valid():
    from app.core.license import LicenseValidator # noqa: F401

    validator = LicenseValidator()
    validator.release_mode = False
    result = await validator.check_proxy_limit(0)
    assert result is True


@pytest.mark.asyncio
async def test_license_validator_dev_mode_check_feature_any():
    from app.core.license import LicenseValidator # noqa: F401

    validator = LicenseValidator()
    validator.release_mode = False
    result = await validator.check_feature("any_feature")
    assert result is True


@pytest.mark.asyncio
async def test_license_validator_dev_mode_check_feature_arbitrary():
    from app.core.license import LicenseValidator # noqa: F401

    validator = LicenseValidator()
    validator.release_mode = False
    result = await validator.check_feature("saml_authentication")
    assert result is True


def test_license_tier_values():
    from app.core.license import LicenseTier # noqa: F401
    assert LicenseTier.COMMUNITY.value == "community"
    assert LicenseTier.ENTERPRISE.value == "enterprise"


def test_license_info_model():
    from app.core.license import LicenseInfo, LicenseTier # noqa: F401
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
    from app.core.license import LicenseInfo, LicenseTier # noqa: F401
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
    from app.core.license import LicenseTier, LicenseValidator # noqa: F401

    validator = LicenseValidator()
    validator.release_mode = True
    validator.license_key = ""

    with patch.object(
        validator._penguin_client,
        "validate",
        return_value=PenguinLicenseInfo(
            valid=True,
            customer="Community",
            product="marchproxy",
            license_version="2.0",
            license_key="",
            expires_at=datetime.max.replace(tzinfo=timezone.utc),
            issued_at=datetime.now(timezone.utc),
            tier="community",
            features=[],
            limits={},
            metadata={}
        )
    ):
        license_info = await validator.validate_license()
        assert license_info.tier == LicenseTier.COMMUNITY
        assert license_info.max_proxies == 3  # COMMUNITY_MAX_PROXIES default


@pytest.mark.asyncio
async def test_license_validator_caching():
    """Second call returns consistent result in dev mode."""
    from app.core.license import LicenseTier, LicenseValidator # noqa: F401

    validator = LicenseValidator()
    validator.release_mode = False

    result1 = await validator.validate_license()
    result2 = await validator.validate_license()
    assert result1.tier == LicenseTier.ENTERPRISE
    assert result2.tier == LicenseTier.ENTERPRISE


@pytest.mark.asyncio
async def test_license_manager_dev_mode_returns_dict():
    """LicenseManager.validate_license returns dict with expected keys."""
    from app.core.license import LicenseManager # noqa: F401

    manager = LicenseManager()
    manager.validator.release_mode = False
    result = await manager.validate_license("any-key")
    assert isinstance(result, dict)
    assert "valid" in result
    assert "tier" in result
    assert "max_proxies" in result
    assert "features" in result
