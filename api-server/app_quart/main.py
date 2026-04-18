"""Quart application factory for MarchProxy API server."""
from quart import Quart # noqa: F401
from quart_cors import cors # noqa: F401


def create_app() -> Quart:
    """Create and configure the Quart application.

    Returns:
        Quart: Configured application instance.
    """
    app = Quart(__name__)

    # Load configuration
    from app_quart.config import config # noqa: F401
    app.config['SECRET_KEY'] = config.SECRET_KEY
    app.config['SQLALCHEMY_DATABASE_URI'] = config.DATABASE_URL.replace(
        '+asyncpg',
        ''
    )
    app.config['SQLALCHEMY_TRACK_MODIFICATIONS'] = False
    app.config['SECURITY_PASSWORD_SALT'] = config.SECURITY_PASSWORD_SALT
    app.config['SECURITY_TOKEN_AUTHENTICATION_HEADER'] = 'Authorization'
    app.config['SECURITY_TOKEN_AUTHENTICATION_KEY'] = 'auth_token'
    app.config['WTF_CSRF_ENABLED'] = False

    # Initialize extensions
    from app_quart.extensions import db, security # noqa: F401
    from app_quart.models.user import Role, User # noqa: F401
    from flask_security import SQLAlchemyUserDatastore # noqa: F401

    db.init_app(app)
    user_datastore = SQLAlchemyUserDatastore(db, User, Role)
    security.init_app(app, user_datastore)

    # CORS
    app = cors(app, allow_origin=config.CORS_ORIGINS)

    # Register blueprints
    from app_quart.api.blueprints import register_blueprints # noqa: F401
    register_blueprints(app)

    return app


if __name__ == '__main__':
    app = create_app()
    app.run(host='0.0.0.0', port=5000, debug=True)
