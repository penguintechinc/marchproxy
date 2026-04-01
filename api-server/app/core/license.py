"""
License validation and feature gating

Wraps penguin-licensing package for license.penguintech.io integration.
Provides backward-compatible interface to existing code.
"""

import logging
from penguintechinc_utils import get_logger
from datetime import datetime, timedelta
from enum import Enum
from typing import Optional

from pydantic import BaseModel
from penguin_licensing import LicenseClient, get_license_client

from app.core.config import settings
logger = get_logger(__name__)


class LicenseTier(str, Enum):
    """License tier enumeration"""
    COMMUNITY = "community"
    ENTERPRISE = "enterprise"


class LicenseInfo(BaseModel):
    """License information model (backward compatible wrapper)"""
    tier: LicenseTier
    max_proxies: int
    features: list[str]
    valid_until: Optional[datetime] = None
    is_valid: bool = True


class LicenseValidator:
    """
    License validation service wrapping penguin-licensing package.

    Maintains backward compatibility with existing async/dict interface
    while delegating to penguin-licensing's LicenseClient.
    """

    def __init__(self):
        self.license_key = settings.LICENSE_KEY
        self.server_url = settings.LICENSE_SERVER_URL
        self.product_name = settings.PRODUCT_NAME
        self.release_mode = settings.RELEASE_MODE

        # Initialize penguin-licensing client
        self._penguin_client = LicenseClient(
            license_key=self.license_key or None,
            product=self.product_name,
            base_url=self.server_url,
        )

    async def validate_license(self, force: bool = False) -> LicenseInfo:
        """
        Validate license key and return license information

        Args:
            force: Force validation even if cached (passed to penguin-licensing)

        Returns:
            LicenseInfo object

        Note:
            In development mode (RELEASE_MODE=False), all features are enabled
        """
        # Development mode bypass
        if not self.release_mode:
            logger.debug("Development mode: All features enabled")
            return LicenseInfo(
                tier=LicenseTier.ENTERPRISE,
                max_proxies=999999,
                features=["all"],
                is_valid=True
            )

        # Delegate to penguin-licensing client (synchronous call)
        try:
            penguin_info = self._penguin_client.validate(force_refresh=force)

            # Extract features from penguin-licensing response
            feature_names = [f.name for f in penguin_info.features if f.entitled]

            license_info = LicenseInfo(
                tier=LicenseTier(penguin_info.tier),
                max_proxies=penguin_info.limits.get("max_proxies", 999999)
                    if penguin_info.tier == "enterprise"
                    else settings.COMMUNITY_MAX_PROXIES,
                features=feature_names,
                valid_until=penguin_info.expires_at,
                is_valid=penguin_info.valid
            )

            if penguin_info.valid:
                logger.info(f"License validated: {license_info.tier}")
            else:
                logger.warning(f"License validation failed: {penguin_info.message}")

            return license_info

        except Exception as e:
            logger.error(f"License validation error: {e}")
            # Fallback to Community on error
            return LicenseInfo(
                tier=LicenseTier.COMMUNITY,
                max_proxies=settings.COMMUNITY_MAX_PROXIES,
                features=[],
                is_valid=False
            )

    async def check_feature(self, feature_name: str) -> bool:
        """
        Check if a specific feature is enabled

        Args:
            feature_name: Name of the feature to check

        Returns:
            True if feature is available, False otherwise
        """
        license_info = await self.validate_license()

        # Development mode or "all" features
        if "all" in license_info.features:
            return True

        return feature_name in license_info.features

    async def check_proxy_limit(self, current_count: int) -> bool:
        """
        Check if adding another proxy would exceed license limits

        Args:
            current_count: Current number of active proxies

        Returns:
            True if within limits, False if limit exceeded
        """
        license_info = await self.validate_license()
        return current_count < license_info.max_proxies


# Global license validator instance
license_validator = LicenseValidator()


class LicenseManager:
    """
    License Manager - Wrapper for LicenseValidator with async/dict interface

    Provides backward compatibility with dependencies.py expectations
    """

    def __init__(self):
        self.validator = license_validator

    async def validate_license(self, license_key: str) -> dict:
        """
        Validate license key and return dict result

        Args:
            license_key: License key to validate

        Returns:
            Dictionary with validation results
        """
        # Create a temporary client with the provided license key
        temp_client = LicenseClient(
            license_key=license_key,
            product=self.validator.product_name,
            base_url=self.validator.server_url,
        )

        try:
            penguin_info = temp_client.validate(force_refresh=True)
            feature_names = [f.name for f in penguin_info.features if f.entitled]

            return {
                "valid": penguin_info.valid,
                "tier": penguin_info.tier,
                "max_proxies": penguin_info.limits.get("max_proxies", 999999)
                    if penguin_info.tier == "enterprise"
                    else settings.COMMUNITY_MAX_PROXIES,
                "features": feature_names,
                "valid_until": penguin_info.expires_at.isoformat()
                    if penguin_info.expires_at else None
            }
        except Exception as e:
            logger.error(f"License validation error: {e}")
            return {
                "valid": False,
                "tier": "community",
                "max_proxies": settings.COMMUNITY_MAX_PROXIES,
                "features": [],
                "valid_until": None
            }
