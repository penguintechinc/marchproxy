"""Register all API blueprints."""
from quart import Quart


def register_blueprints(app: Quart) -> None:
    """Register all API version blueprints."""
    from app_quart.api.v1 import v1_bp
    app.register_blueprint(v1_bp, url_prefix='/api/v1')
