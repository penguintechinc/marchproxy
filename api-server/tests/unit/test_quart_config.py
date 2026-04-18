"""Unit tests for app_quart/config.py."""
import os
import pytest
from unittest.mock import patch


def test_config_kong_admin_url_default():
    from app_quart.config import Config
    c = Config()
    assert c.KONG_ADMIN_URL == 'http://kong:8001'


def test_config_kong_admin_url_is_string():
    from app_quart.config import Config
    c = Config()
    assert isinstance(c.KONG_ADMIN_URL, str)
    assert 'http' in c.KONG_ADMIN_URL


def test_config_secret_key_default():
    from app_quart.config import Config
    c = Config()
    assert isinstance(c.SECRET_KEY, str)
    assert len(c.SECRET_KEY) > 0


def test_config_database_url_default():
    from app_quart.config import Config
    c = Config()
    assert 'marchproxy' in c.DATABASE_URL
    assert 'postgresql' in c.DATABASE_URL


def test_config_database_url_contains_protocol():
    from app_quart.config import Config
    c = Config()
    assert 'postgresql' in c.DATABASE_URL or 'sqlite' in c.DATABASE_URL or 'mysql' in c.DATABASE_URL


def test_config_debug_default_false():
    from app_quart.config import Config
    with patch.dict(os.environ, {}, clear=False):
        # Remove DEBUG env var if it was set
        env = {k: v for k, v in os.environ.items() if k != 'DEBUG'}
        env['DEBUG'] = 'false'
        with patch.dict(os.environ, env, clear=True):
            c = Config()
            assert c.DEBUG is False


def test_config_debug_is_bool():
    from app_quart.config import Config
    c = Config()
    assert isinstance(c.DEBUG, bool)


def test_config_cors_origins_default():
    from app_quart.config import Config
    with patch.dict(os.environ, {'CORS_ORIGINS': 'http://localhost:3000'}):
        c = Config()
        assert isinstance(c.CORS_ORIGINS, list)
        assert 'http://localhost:3000' in c.CORS_ORIGINS


def test_config_cors_origins_multiple():
    with patch.dict(os.environ, {'CORS_ORIGINS': 'http://localhost:3000,http://localhost:4000'}):
        from app_quart.config import Config
        c = Config()
        assert len(c.CORS_ORIGINS) == 2
        assert 'http://localhost:3000' in c.CORS_ORIGINS
        assert 'http://localhost:4000' in c.CORS_ORIGINS


def test_config_jwt_expires_default():
    from app_quart.config import Config
    c = Config()
    assert c.JWT_ACCESS_TOKEN_EXPIRES == 3600


def test_config_jwt_expires_is_int():
    from app_quart.config import Config
    c = Config()
    assert isinstance(c.JWT_ACCESS_TOKEN_EXPIRES, int)
    assert c.JWT_ACCESS_TOKEN_EXPIRES > 0


def test_config_redis_url_default():
    from app_quart.config import Config
    c = Config()
    assert 'redis' in c.REDIS_URL


def test_config_security_password_salt_default():
    from app_quart.config import Config
    c = Config()
    assert isinstance(c.SECURITY_PASSWORD_SALT, str)
    assert len(c.SECURITY_PASSWORD_SALT) > 0


def test_config_log_level_default():
    from app_quart.config import Config
    c = Config()
    assert c.LOG_LEVEL == 'INFO'


def test_config_module_singleton():
    from app_quart.config import config
    assert config is not None
    assert hasattr(config, 'KONG_ADMIN_URL')
    assert hasattr(config, 'DATABASE_URL')
    assert hasattr(config, 'SECRET_KEY')
