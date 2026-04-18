"""Unit tests for app/core/config.py"""
import os # noqa: F401, # noqa: F401
from unittest.mock import patch # noqa: F401

import pytest # noqa: F401, # noqa: F401


def test_settings_has_algorithm():
    from app.core.config import settings # noqa: F401
    assert settings.ALGORITHM == "HS256"


def test_settings_access_token_expire_minutes_default():
    from app.core.config import settings # noqa: F401
    assert settings.ACCESS_TOKEN_EXPIRE_MINUTES == 30


def test_settings_refresh_token_expire_days_default():
    from app.core.config import settings # noqa: F401
    assert settings.REFRESH_TOKEN_EXPIRE_DAYS == 7


def test_settings_community_max_proxies_default():
    from app.core.config import settings # noqa: F401
    assert settings.COMMUNITY_MAX_PROXIES == 3


def test_settings_release_mode_default_false():
    from app.core.config import settings # noqa: F401
    assert settings.RELEASE_MODE is False


def test_settings_secret_key_exists():
    from app.core.config import settings # noqa: F401
    assert isinstance(settings.SECRET_KEY, str)
    assert len(settings.SECRET_KEY) >= 32


def test_settings_algorithm_is_string():
    from app.core.config import settings # noqa: F401
    assert isinstance(settings.ALGORITHM, str)


def test_cors_origins_returns_list():
    from app.core.config import settings # noqa: F401
    origins = settings.CORS_ORIGINS
    assert isinstance(origins, list)


def test_cors_origins_non_empty_by_default():
    from app.core.config import settings # noqa: F401
    origins = settings.CORS_ORIGINS
    assert len(origins) > 0


def test_cors_origins_parses_comma_separated():
    from app.core.config import Settings # noqa: F401
    s = Settings(CORS_ORIGINS_STR="http://localhost:3000,http://localhost:4000")
    origins = s.CORS_ORIGINS
    assert "http://localhost:3000" in origins
    assert "http://localhost:4000" in origins
    assert len(origins) == 2


def test_cors_origins_empty_string_gives_fallback():
    from app.core.config import Settings # noqa: F401
    s = Settings(CORS_ORIGINS_STR="")
    origins = s.CORS_ORIGINS
 # Empty string falls back to default list
    assert isinstance(origins, list)
    assert len(origins) > 0


def test_settings_app_name():
    from app.core.config import settings # noqa: F401
    assert isinstance(settings.APP_NAME, str)
    assert len(settings.APP_NAME) > 0


def test_settings_product_name():
    from app.core.config import settings # noqa: F401
    assert settings.PRODUCT_NAME == "marchproxy"


def test_settings_license_server_url():
    from app.core.config import settings # noqa: F401
    assert "penguintech.io" in settings.LICENSE_SERVER_URL


def test_settings_default_page_size():
    from app.core.config import settings # noqa: F401
    assert settings.DEFAULT_PAGE_SIZE == 20
    assert settings.MAX_PAGE_SIZE == 100
