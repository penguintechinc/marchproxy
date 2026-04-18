"""Register all API blueprints."""
from quart import Quart # noqa: F401


def register_blueprints(app: Quart) -> None:
    """Register all API version blueprints."""
    from app_quart.api.v1 import v1_bp # noqa: F401
    app.register_blueprint(v1_bp, url_prefix='/api/v1')
