"""
Configuration management API Blueprint for MarchProxy Manager (Quart)

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from quart import Blueprint, request, current_app, jsonify
from pydantic import ValidationError, BaseModel
import logging
from typing import Optional, Dict, Any
from datetime import datetime
from middleware.auth import require_auth

logger = logging.getLogger(__name__)

config_bp = Blueprint("config", __name__, url_prefix="/api/v1/config")


class ConfigUpdateRequest(BaseModel):
    key: str
    value: Any
    description: Optional[str] = None


class SystemConfigResponse(BaseModel):
    db_type: str
    db_host: str
    db_port: int
    db_name: str
    license_mode: str
    product_version: str
    release_mode: bool


class HealthCheckResponse(BaseModel):
    status: str
    database: str
    timestamp: datetime
    version: str


@config_bp.route("/system", methods=["GET"])
@require_auth(admin_required=True)
async def get_system_config(user_data):
    """Get system configuration"""
    try:
        import os

        db_type = os.getenv("DB_TYPE", "postgresql")
        db_host = os.getenv("DB_HOST", "localhost")
        db_port = int(os.getenv("DB_PORT", "5432"))
        db_name = os.getenv("DB_NAME", "marchproxy")
        release_mode = os.getenv("RELEASE_MODE", "false").lower() == "true"
        license_mode = "strict" if release_mode else "permissive"

        # Get version from .version file
        version = "unknown"
        try:
            with open("/home/penguin/code/MarchProxy/.version", "r") as f:
                version = f.read().strip()
        except Exception:
            pass

        response = SystemConfigResponse(
            db_type=db_type,
            db_host=db_host,
            db_port=db_port,
            db_name=db_name,
            license_mode=license_mode,
            product_version=version,
            release_mode=release_mode,
        )
        return jsonify(response.dict()), 200

    except Exception as e:
        logger.error(f"Error getting system config: {str(e)}")
        return jsonify({"error": "Failed to get system config", "details": str(e)}), 500


@config_bp.route("/health", methods=["GET"])
async def health_check():
    """Health check endpoint"""
    try:
        db = current_app.db
        timestamp = datetime.utcnow()

        # Try a simple database query
        try:
            test_query = db().select().first()
            db_status = "healthy"
        except Exception as e:
            logger.error(f"Database health check failed: {str(e)}")
            db_status = "unhealthy"

        # Get version
        version = "unknown"
        try:
            with open("/home/penguin/code/MarchProxy/.version", "r") as f:
                version = f.read().strip()
        except Exception:
            pass

        response = HealthCheckResponse(
            status="healthy" if db_status == "healthy" else "degraded",
            database=db_status,
            timestamp=timestamp,
            version=version,
        )
        return jsonify(response.dict()), 200

    except Exception as e:
        logger.error(f"Health check failed: {str(e)}")
        return jsonify({"status": "unhealthy", "error": str(e)}), 503


@config_bp.route("/license", methods=["GET", "PUT"])
async def license_config():
    """Get or update license configuration"""
    import os

    if request.method == "GET":
        try:
            license_key = os.getenv("LICENSE_KEY", "").replace(
                os.getenv("LICENSE_KEY", "")[-4:], "****"
            )
            release_mode = os.getenv("RELEASE_MODE", "false").lower() == "true"
            license_server_url = os.getenv("LICENSE_SERVER_URL", "https://license.penguintech.io")

            return (
                jsonify(
                    {
                        "license_key": (license_key if os.getenv("LICENSE_KEY") else None),
                        "release_mode": release_mode,
                        "license_server_url": license_server_url,
                        "license_mode": "strict" if release_mode else "permissive",
                    }
                ),
                200,
            )

        except Exception as e:
            logger.error(f"Error getting license config: {str(e)}")
            return (
                jsonify({"error": "Failed to get license config", "details": str(e)}),
                500,
            )

    elif request.method == "PUT":

        @require_auth(admin_required=True)
        async def update_license_config(user_data):
            try:
                data_json = await request.get_json()

                # Validate release mode if provided
                if "release_mode" in data_json:
                    release_mode = data_json["release_mode"]
                    if not isinstance(release_mode, bool):
                        return jsonify({"error": "release_mode must be boolean"}), 400

                    # In production, this would write to config file or env store
                    # For now, just log the change
                    logger.info(
                        f"License mode changed to {'strict' if release_mode else 'permissive'}"
                    )

                # License key would be stored securely (encrypted)
                if "license_key" in data_json:
                    logger.info("License key updated")

                return (
                    jsonify(
                        {
                            "message": "License configuration updated",
                            "requires_restart": True,
                        }
                    ),
                    200,
                )

            except Exception as e:
                logger.error(f"Error updating license config: {str(e)}")
                return (
                    jsonify({"error": "Failed to update license config", "details": str(e)}),
                    500,
                )

        return await update_license_config(user_data={})


@config_bp.route("/logging", methods=["GET", "PUT"])
async def logging_config():
    """Get or update logging configuration"""
    import logging

    if request.method == "GET":
        try:
            root_logger = logging.getLogger()
            current_level = logging.getLevelName(root_logger.level)

            return (
                jsonify(
                    {
                        "log_level": current_level,
                        "available_levels": [
                            "DEBUG",
                            "INFO",
                            "WARNING",
                            "ERROR",
                            "CRITICAL",
                        ],
                    }
                ),
                200,
            )

        except Exception as e:
            logger.error(f"Error getting logging config: {str(e)}")
            return (
                jsonify({"error": "Failed to get logging config", "details": str(e)}),
                500,
            )

    elif request.method == "PUT":

        @require_auth(admin_required=True)
        async def update_logging_config(user_data):
            try:
                data_json = await request.get_json()
                log_level = data_json.get("log_level", "INFO").upper()

                # Validate log level
                valid_levels = ["DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL"]
                if log_level not in valid_levels:
                    return (
                        jsonify({"error": f"Invalid log level. Must be one of: {valid_levels}"}),
                        400,
                    )

                # Set log level
                root_logger = logging.getLogger()
                root_logger.setLevel(getattr(logging, log_level))

                logger.info(f"Log level changed to {log_level}")

                return (
                    jsonify(
                        {
                            "message": "Logging configuration updated",
                            "log_level": log_level,
                        }
                    ),
                    200,
                )

            except Exception as e:
                logger.error(f"Error updating logging config: {str(e)}")
                return (
                    jsonify({"error": "Failed to update logging config", "details": str(e)}),
                    500,
                )

        return await update_logging_config(user_data={})


@config_bp.route("/database", methods=["GET"])
@require_auth(admin_required=True)
async def database_config(user_data):
    """Get database configuration"""
    try:
        import os

        db = current_app.db

        db_type = os.getenv("DB_TYPE", "postgresql")
        db_host = os.getenv("DB_HOST", "localhost")
        db_port = os.getenv("DB_PORT", "5432")
        db_name = os.getenv("DB_NAME", "marchproxy")
        db_user = os.getenv("DB_USER", "marchproxy")

        # Get database stats
        try:
            db_stats = {"tables": len(db.tables), "connected": True}
        except Exception:
            db_stats = {"connected": False}

        return (
            jsonify(
                {
                    "type": db_type,
                    "host": db_host,
                    "port": int(db_port),
                    "database": db_name,
                    "user": db_user,
                    "stats": db_stats,
                }
            ),
            200,
        )

    except Exception as e:
        logger.error(f"Error getting database config: {str(e)}")
        return (
            jsonify({"error": "Failed to get database config", "details": str(e)}),
            500,
        )


@config_bp.route("/features", methods=["GET"])
@require_auth(admin_required=True)
async def features_config(user_data):
    """Get available features based on license"""
    try:
        import os
        from models.license import LicenseCacheModel

        db = current_app.db
        license_key = os.getenv("LICENSE_KEY")
        release_mode = os.getenv("RELEASE_MODE", "false").lower() == "true"

        features = {
            "core": {
                "clusters": True,
                "services": True,
                "mappings": True,
                "proxy_servers": True,
                "authentication": True,
            },
            "enterprise": {
                "saml": False,
                "oauth2": False,
                "scim": False,
                "mfa": False,
                "rbac_advanced": False,
                "audit_logging": False,
            },
        }

        # Check license if in release mode
        if release_mode and license_key:
            cached = LicenseCacheModel.get_cached_validation(db, license_key)
            if cached and cached["is_enterprise"]:
                features["enterprise"] = cached.get("features", features["enterprise"])

        return (
            jsonify(
                {
                    "release_mode": release_mode,
                    "license_active": bool(license_key),
                    "features": features,
                }
            ),
            200,
        )

    except Exception as e:
        logger.error(f"Error getting features config: {str(e)}")
        return (
            jsonify({"error": "Failed to get features config", "details": str(e)}),
            500,
        )


@config_bp.route("/version", methods=["GET"])
async def version_config():
    """Get version information"""
    try:
        version = "unknown"
        try:
            with open("/home/penguin/code/MarchProxy/.version", "r") as f:
                version = f.read().strip()
        except Exception:
            pass

        return (
            jsonify({"version": version, "timestamp": datetime.utcnow().isoformat()}),
            200,
        )

    except Exception as e:
        logger.error(f"Error getting version config: {str(e)}")
        return jsonify({"error": "Failed to get version", "details": str(e)}), 500
