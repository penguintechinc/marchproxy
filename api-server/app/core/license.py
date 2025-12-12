"""
License validation and feature gating

Integrates with license.penguintech.io for enterprise feature enforcement.
"""

import logging
from datetime import datetime, timedelta
from enum import Enum
from typing import Optional

import httpx
from pydantic import BaseModel

from app.config import get_settings

settings = get_settings()
logger = logging.getLogger(__name__)


class LicenseTier(str, Enum):
    """License tier enumeration"""
    COMMUNITY = "community"
    ENTERPRISE = "enterprise"


class LicenseInfo(BaseModel):
    """License information model"""
    tier: LicenseTier
    max_proxies: int
    features: list[str]
    valid_until: Optional[datetime] = None
    is_valid: bool = True


class LicenseValidator:
    """License validation service"""

    def __init__(self):
        self.license_key = settings.LICENSE_KEY
        self.server_url = settings.LICENSE_SERVER_URL
        self.product_name = settings.PRODUCT_NAME
        self.release_mode = settings.RELEASE_MODE
        self._cache: Optional[LicenseInfo] = None
        self._cache_expiry: Optional[datetime] = None

    async def validate_license(self, force: bool = False) -> LicenseInfo:
        """
        Validate license key and return license information

        Args:
            force: Force validation even if cached

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

        # Check cache
        if not force and self._cache and self._cache_expiry:
            if datetime.utcnow() < self._cache_expiry:
                logger.debug("Returning cached license info")
                return self._cache

        # No license key = Community tier
        if not self.license_key:
            logger.info("No license key provided, using Community tier")
            license_info = LicenseInfo(
                tier=LicenseTier.COMMUNITY,
                max_proxies=settings.COMMUNITY_MAX_PROXIES,
                features=[],
                is_valid=True
            )
            self._cache = license_info
            self._cache_expiry = datetime.utcnow() + timedelta(hours=1)
            return license_info

        # Validate with license server
        try:
            async with httpx.AsyncClient(timeout=10.0) as client:
                response = await client.post(
                    f"{self.server_url}/api/v2/validate",
                    json={
                        "license_key": self.license_key,
                        "product": self.product_name
                    }
                )

                if response.status_code == 200:
                    data = response.json()
                    license_info = LicenseInfo(
                        tier=LicenseTier.ENTERPRISE,
                        max_proxies=data.get("max_proxies", 999999),
                        features=data.get("features", []),
                        valid_until=datetime.fromisoformat(data["valid_until"])
                        if "valid_until" in data else None,
                        is_valid=True
                    )
                    logger.info(f"License validated: {license_info.tier}")
                else:
                    logger.warning(
                        f"License validation failed: {response.status_code}"
                    )
                    license_info = LicenseInfo(
                        tier=LicenseTier.COMMUNITY,
                        max_proxies=settings.COMMUNITY_MAX_PROXIES,
                        features=[],
                        is_valid=False
                    )

        except Exception as e:
            logger.error(f"License validation error: {e}")
            # Fallback to Community on error
            license_info = LicenseInfo(
                tier=LicenseTier.COMMUNITY,
                max_proxies=settings.COMMUNITY_MAX_PROXIES,
                features=[],
                is_valid=False
            )

        # Cache for 1 hour
        self._cache = license_info
        self._cache_expiry = datetime.utcnow() + timedelta(hours=1)
        return license_info

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
