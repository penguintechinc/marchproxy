"""
Main Quart Application for MarchProxy Manager

Application factory pattern with async-first Quart framework.
Supports multi-database backends (PostgreSQL, MySQL, SQLite) via PyDAL.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import os
import logging
from datetime import datetime
from typing import Optional

from quart import Quart, jsonify, request
from quart_cors import cors
from pydal import DAL

from database import get_db_manager, DatabaseManager
from models.auth import JWTManager

# Configure logging
logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)


def create_app(config: Optional[dict] = None) -> Quart:
    """
    Application factory for MarchProxy Manager.

    Creates and configures Quart application instance with:
    - CORS support
    - Database initialization
    - JWT authentication
    - Blueprint registration
    - Error handlers
    - Lifecycle hooks

    Args:
        config: Optional configuration dictionary to override environment variables

    Returns:
        Configured Quart application instance

    Raises:
        ValueError: If required configuration is missing
        RuntimeError: If database initialization fails
    """
    app = Quart(__name__)

    # Apply CORS - get allowed origins from environment or use default
    cors_origins = os.getenv(
        "CORS_ALLOWED_ORIGINS",
        "https://marchproxy.penguintech.io,http://localhost:3000",
    )
    app = cors(app, allow_origin=cors_origins)

    # Load configuration from environment variables
    _load_config(app, config)

    # Validate required configuration
    _validate_config(app.config)

    # Initialize database
    _initialize_database(app)

    # Initialize JWT manager
    _initialize_jwt(app)

    # Register blueprints
    _register_blueprints(app)

    # Register error handlers
    _register_error_handlers(app)

    # Register lifecycle hooks
    _register_lifecycle_hooks(app)

    logger.info(
        "MarchProxy Manager application created successfully",
        extra={
            "debug": app.config.get("DEBUG", False),
            "db_type": os.getenv("DB_TYPE", "postgres"),
        },
    )

    return app


def _load_config(app: Quart, config: Optional[dict] = None) -> None:
    """
    Load configuration from environment variables and optional config dict.

    Args:
        app: Quart application instance
        config: Optional configuration dictionary to override environment variables
    """
    # Default configuration
    app.config.update(
        {
            "DATABASE_URL": os.getenv("DATABASE_URL"),
            "DB_TYPE": os.getenv("DB_TYPE", "postgres").lower(),
            "JWT_SECRET": os.getenv("JWT_SECRET"),
            "JWT_ACCESS_TOKEN_EXPIRES": int(os.getenv("JWT_ACCESS_TOKEN_EXPIRES", "3600")),
            "JWT_REFRESH_TOKEN_EXPIRES": int(os.getenv("JWT_REFRESH_TOKEN_EXPIRES", "86400")),
            "DEBUG": os.getenv("DEBUG", "false").lower() == "true",
            "LICENSE_SERVER_URL": os.getenv("LICENSE_SERVER_URL", "https://license.penguintech.io"),
            "LICENSE_KEY": os.getenv("LICENSE_KEY"),
            "ADMIN_PASSWORD": os.getenv("ADMIN_PASSWORD", "admin123"),
            "SQL_ECHO": os.getenv("SQL_ECHO", "false").lower() == "true",
        }
    )

    # Override with provided config
    if config:
        app.config.update(config)

    logger.info(
        "Configuration loaded",
        extra={
            "debug": app.config["DEBUG"],
            "jwt_access_expires": app.config["JWT_ACCESS_TOKEN_EXPIRES"],
            "jwt_refresh_expires": app.config["JWT_REFRESH_TOKEN_EXPIRES"],
            "license_configured": bool(app.config["LICENSE_KEY"]),
        },
    )


def _validate_config(config: dict) -> None:
    """
    Validate required configuration values.

    Args:
        config: Application configuration dictionary

    Raises:
        ValueError: If required configuration is missing or invalid
    """
    # Required configuration
    required = ["DATABASE_URL", "JWT_SECRET"]
    missing = [key for key in required if not config.get(key)]

    if missing:
        raise ValueError(f"Missing required configuration: {', '.join(missing)}")

    # Validate JWT_SECRET is not default
    if config["JWT_SECRET"] == "your-super-secret-jwt-key-change-in-production":
        logger.warning(
            "Using default JWT_SECRET! This is INSECURE in production. "
            "Set JWT_SECRET environment variable."
        )

    # Validate DB_TYPE
    if config["DB_TYPE"] not in ["postgres", "mysql", "sqlite"]:
        raise ValueError(
            f"DB_TYPE must be 'postgres', 'mysql', or 'sqlite', got '{config['DB_TYPE']}'"
        )

    logger.info("Configuration validation passed")


def _initialize_database(app: Quart) -> None:
    """
    Initialize database connection and schema.

    Args:
        app: Quart application instance

    Raises:
        RuntimeError: If database initialization fails
    """
    try:
        logger.info("Initializing database...")

        # Get database manager instance
        db_manager = get_db_manager()

        # Initialize schema using SQLAlchemy (idempotent)
        if not db_manager.initialize_schema():
            raise RuntimeError("Failed to initialize database schema")

        # Get PyDAL connection and define tables
        db = db_manager.get_pydal_connection()

        # Attach database to app
        app.db = db
        app.db_manager = db_manager

        logger.info("Database initialized successfully", extra={"db_type": db_manager.db_type})

    except Exception as e:
        logger.error(f"Database initialization failed: {str(e)}", exc_info=True)
        raise RuntimeError(f"Database initialization failed: {str(e)}")


def _initialize_jwt(app: Quart) -> None:
    """
    Initialize JWT manager.

    Args:
        app: Quart application instance
    """
    jwt_manager = JWTManager(
        secret_key=app.config["JWT_SECRET"],
        algorithm="HS256",
        ttl_hours=app.config["JWT_ACCESS_TOKEN_EXPIRES"] // 3600,
    )

    app.jwt_manager = jwt_manager

    logger.info(
        "JWT manager initialized",
        extra={
            "algorithm": "HS256",
            "ttl_hours": app.config["JWT_ACCESS_TOKEN_EXPIRES"] // 3600,
        },
    )


def _register_blueprints(app: Quart) -> None:
    """
    Register all API blueprints with URL prefixes.

    Blueprints are imported lazily and failures are logged but don't prevent
    application startup (allows partial deployment during development).

    Args:
        app: Quart application instance
    """
    # Import and register blueprints with proper error handling
    # This allows app to start even if some blueprints are not yet implemented

    # System endpoints (health, metrics, root)
    try:
        from api.system_bp import system_bp

        app.register_blueprint(system_bp)
        logger.info("Registered system blueprint")
    except ImportError as e:
        logger.warning(f"Failed to import system blueprint: {e}")

    # Authentication endpoints
    try:
        from api.auth_bp import auth_bp

        app.register_blueprint(auth_bp, url_prefix="/api/auth")
        logger.info("Registered auth blueprint at /api/auth")
    except ImportError as e:
        logger.warning(f"Failed to import auth blueprint: {e}")

    # Cluster management endpoints
    try:
        from api.clusters_bp import clusters_bp

        app.register_blueprint(clusters_bp, url_prefix="/api")
        logger.info("Registered clusters blueprint at /api")
    except ImportError as e:
        logger.warning(f"Failed to import clusters blueprint: {e}")

    # Proxy management endpoints
    try:
        from api.proxy_bp import proxy_bp

        app.register_blueprint(proxy_bp, url_prefix="/api")
        logger.info("Registered proxy blueprint at /api")
    except ImportError as e:
        logger.warning(f"Failed to import proxy blueprint: {e}")

    # mTLS certificate endpoints
    try:
        from api.mtls_bp import mtls_bp

        app.register_blueprint(mtls_bp, url_prefix="/api/mtls")
        logger.info("Registered mtls blueprint at /api/mtls")
    except ImportError as e:
        logger.warning(f"Failed to import mtls blueprint: {e}")

    # Block rules endpoints
    try:
        from api.block_rules_bp import block_rules_bp

        app.register_blueprint(block_rules_bp, url_prefix="/api/v1")
        logger.info("Registered block_rules blueprint at /api/v1")
    except ImportError as e:
        logger.warning(f"Failed to import block_rules blueprint: {e}")

    # Service management endpoints
    try:
        from api.services_bp import services_bp

        app.register_blueprint(services_bp, url_prefix="/api")
        logger.info("Registered services blueprint at /api")
    except ImportError as e:
        logger.warning(f"Failed to import services blueprint: {e}")

    # Mapping management endpoints
    try:
        from api.mappings_bp import mappings_bp

        app.register_blueprint(mappings_bp, url_prefix="/api")
        logger.info("Registered mappings blueprint at /api")
    except ImportError as e:
        logger.warning(f"Failed to import mappings blueprint: {e}")

    # License management endpoints
    try:
        from api.license_bp import license_bp

        app.register_blueprint(license_bp, url_prefix="/api/license")
        logger.info("Registered license blueprint at /api/license")
    except ImportError as e:
        logger.warning(f"Failed to import license blueprint: {e}")

    # Config endpoints
    try:
        from api.config_bp import config_bp

        app.register_blueprint(config_bp, url_prefix="/api/config")
        logger.info("Registered config blueprint at /api/config")
    except ImportError as e:
        logger.warning(f"Failed to import config blueprint: {e}")

    # Ingress routes endpoints
    try:
        from api.ingress_routes_bp import ingress_routes_bp

        app.register_blueprint(ingress_routes_bp, url_prefix="/api")
        logger.info("Registered ingress_routes blueprint at /api")
    except ImportError as e:
        logger.warning(f"Failed to import ingress_routes blueprint: {e}")

    # Enterprise authentication endpoints
    try:
        from api.enterprise_auth_bp import enterprise_auth_bp

        app.register_blueprint(enterprise_auth_bp, url_prefix="/api/enterprise/auth")
        logger.info("Registered enterprise_auth blueprint at /api/enterprise/auth")
    except ImportError as e:
        logger.warning(f"Failed to import enterprise_auth blueprint: {e}")

    # Roles (RBAC) endpoints
    try:
        from api.roles_bp import roles_bp

        app.register_blueprint(roles_bp)
        logger.info("Registered roles blueprint at /api/v1/roles")
    except ImportError as e:
        logger.warning(f"Failed to import roles blueprint: {e}")

    # Media module endpoints
    try:
        from api.media_bp import media_bp

        app.register_blueprint(media_bp)
        logger.info("Registered media blueprint at /api/v1/modules/rtmp")
    except ImportError as e:
        logger.warning(f"Failed to import media blueprint: {e}")

    # Admin media settings endpoints (super admin only)
    try:
        from api.admin_media_bp import admin_media_bp

        app.register_blueprint(admin_media_bp)
        logger.info("Registered admin_media blueprint at /api/v1/admin/media")
    except ImportError as e:
        logger.warning(f"Failed to import admin_media blueprint: {e}")


def _register_error_handlers(app: Quart) -> None:
    """
    Register error handlers for common HTTP errors.

    Args:
        app: Quart application instance
    """

    @app.errorhandler(400)
    async def bad_request(error):
        """Handle 400 Bad Request errors"""
        logger.warning(f"Bad request: {error}")
        return (
            jsonify({"error": "Bad Request", "message": str(error), "status_code": 400}),
            400,
        )

    @app.errorhandler(401)
    async def unauthorized(error):
        """Handle 401 Unauthorized errors"""
        logger.warning(f"Unauthorized access: {error}")
        return (
            jsonify(
                {
                    "error": "Unauthorized",
                    "message": "Authentication required",
                    "status_code": 401,
                }
            ),
            401,
        )

    @app.errorhandler(403)
    async def forbidden(error):
        """Handle 403 Forbidden errors"""
        logger.warning(f"Forbidden access: {error}")
        return (
            jsonify(
                {
                    "error": "Forbidden",
                    "message": "Insufficient permissions",
                    "status_code": 403,
                }
            ),
            403,
        )

    @app.errorhandler(404)
    async def not_found(error):
        """Handle 404 Not Found errors"""
        return (
            jsonify(
                {
                    "error": "Not Found",
                    "message": "Resource not found",
                    "status_code": 404,
                }
            ),
            404,
        )

    @app.errorhandler(500)
    async def internal_error(error):
        """Handle 500 Internal Server Error"""
        logger.error(f"Internal server error: {error}", exc_info=True)
        return (
            jsonify(
                {
                    "error": "Internal Server Error",
                    "message": "An unexpected error occurred",
                    "status_code": 500,
                }
            ),
            500,
        )

    logger.info("Error handlers registered")


def _register_lifecycle_hooks(app: Quart) -> None:
    """
    Register application lifecycle hooks.

    Args:
        app: Quart application instance
    """

    @app.before_serving
    async def before_serving():
        """Run before application starts serving requests"""
        logger.info("Application starting up...")

        # Initialize default data if needed
        await _initialize_default_data(app)

        logger.info("Application ready to serve requests")

    @app.after_serving
    async def after_serving():
        """Run after application stops serving requests"""
        logger.info("Application shutting down...")

        # Close database connections
        if hasattr(app, "db_manager"):
            app.db_manager.close()

        logger.info("Application shutdown complete")

    @app.after_request
    async def after_request(response):
        """Run after each request"""
        # Commit PyDAL transactions
        if hasattr(app, "db") and app.db:
            try:
                app.db.commit()
            except Exception as e:
                logger.error(f"Error committing database transaction: {e}")
                app.db.rollback()

        return response

    logger.info("Lifecycle hooks registered")


async def _initialize_default_data(app: Quart) -> None:
    """
    Initialize default admin user, cluster, and RBAC roles if they don't exist.

    Args:
        app: Quart application instance
    """
    try:
        from models.auth import UserModel
        from models.cluster import ClusterModel
        from models.rbac import RBACModel, PermissionScope

        db = app.db

        # Initialize RBAC tables and default roles
        try:
            # Define tables (idempotent - safe to call multiple times)
            RBACModel.define_tables(db)
            db.commit()

            # Check if roles table exists and has data
            try:
                role_count = db(db.roles).count()
                if role_count == 0:
                    # Tables exist but are empty - initialize default roles
                    RBACModel.initialize_default_roles(db)
                    db.commit()
                    logger.info("RBAC default roles initialized")
                else:
                    logger.info(f"RBAC tables already initialized ({role_count} roles exist)")
            except Exception as roles_error:
                logger.error(f"Error checking/initializing RBAC roles: {roles_error}")
                db.rollback()

        except Exception as e:
            logger.error(f"RBAC table definition failed: {e}", exc_info=True)
            db.rollback()

        # Create default admin user if not exists
        try:
            admin_user = db(db.users.username == "admin").select().first()
        except Exception as e:
            logger.error(f"Error checking for admin user: {e}")
            db.rollback()
            admin_user = None

        if not admin_user:
            admin_password = app.config["ADMIN_PASSWORD"]
            password_hash = UserModel.hash_password(admin_password)

            admin_id = db.users.insert(
                username="admin",
                email="admin@localhost.local",
                password_hash=password_hash,
                is_admin=True,
                is_active=True,
            )

            logger.info(f"Created default admin user (ID: {admin_id})")

            # Assign Admin role to default admin user
            try:
                RBACModel.assign_role(db, admin_id, "admin", scope=PermissionScope.GLOBAL)
                logger.info(f"Assigned Admin role to default admin user")
            except Exception as e:
                logger.warning(f"Could not assign admin role: {e}")

            # Create default cluster for Community edition
            cluster_id, api_key = ClusterModel.create_default_cluster(db, admin_id)
            logger.info(
                f"Created default cluster (ID: {cluster_id})",
                extra={"api_key": api_key},
            )

        db.commit()

    except Exception as e:
        logger.error(f"Failed to initialize default data: {e}", exc_info=True)


# Main entry point for development
if __name__ == "__main__":
    app = create_app()
    app.run(host="0.0.0.0", port=5000, debug=app.config["DEBUG"])
