"""Unit tests for app_quart/main.py and api/blueprints.py.

These tests verify create_app produces a properly configured application.
They must run in isolation due to blueprint singleton state.
"""
import sys
import importlib
import pytest
from unittest.mock import AsyncMock, MagicMock, patch


def _fresh_create_app():
    """Create fresh app, clearing v1_bp endpoint cache to avoid conflicts."""
    # Clear the blueprint's deferred functions to allow re-registration
    from app_quart.api.v1 import v1_bp
    # Manually clear registered endpoints to allow re-use in tests
    v1_bp.deferred_functions = []

    with patch('app_quart.extensions.db') as mock_db:
        mock_db.init_app = MagicMock()
        with patch('flask_security.Security') as MockSecurity:
            sec_instance = MagicMock()
            sec_instance.init_app = MagicMock()
            MockSecurity.return_value = sec_instance
            with patch('flask_security.SQLAlchemyUserDatastore'):
                from app_quart.main import create_app
                return create_app()


# ---------------------------------------------------------------------------
# Blueprints
# ---------------------------------------------------------------------------

def test_register_blueprints_function_exists():
    """register_blueprints is importable."""
    from app_quart.api.blueprints import register_blueprints
    assert callable(register_blueprints)


def test_register_blueprints_imports_v1_bp():
    """register_blueprints uses the v1 blueprint."""
    import inspect
    from app_quart.api import blueprints
    source = inspect.getsource(blueprints)
    assert 'v1_bp' in source
    assert '/api/v1' in source


# ---------------------------------------------------------------------------
# create_app
# ---------------------------------------------------------------------------

def test_create_app_returns_quart_instance():
    """create_app returns a Quart application instance."""
    from quart import Quart
    app = _fresh_create_app()
    assert isinstance(app, Quart)


def test_create_app_sets_secret_key():
    """create_app configures SECRET_KEY from config."""
    app = _fresh_create_app()
    assert 'SECRET_KEY' in app.config
    assert isinstance(app.config['SECRET_KEY'], str)


def test_create_app_disables_csrf():
    """create_app disables WTF_CSRF_ENABLED."""
    app = _fresh_create_app()
    assert app.config['WTF_CSRF_ENABLED'] is False


def test_create_app_sets_sqlalchemy_uri():
    """create_app sets SQLALCHEMY_DATABASE_URI without +asyncpg."""
    app = _fresh_create_app()
    uri = app.config['SQLALCHEMY_DATABASE_URI']
    assert '+asyncpg' not in uri


def test_create_app_sets_sqlalchemy_track_modifications_false():
    """create_app sets SQLALCHEMY_TRACK_MODIFICATIONS to False."""
    app = _fresh_create_app()
    assert app.config['SQLALCHEMY_TRACK_MODIFICATIONS'] is False


def test_create_app_sets_security_token_header():
    """create_app sets SECURITY_TOKEN_AUTHENTICATION_HEADER."""
    app = _fresh_create_app()
    assert app.config['SECURITY_TOKEN_AUTHENTICATION_HEADER'] == 'Authorization'


def test_create_app_has_url_map():
    """create_app produces an app with a URL map."""
    app = _fresh_create_app()
    rules = list(app.url_map._rules)
    assert len(rules) > 0
