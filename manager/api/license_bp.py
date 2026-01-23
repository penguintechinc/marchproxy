"""
License management API Blueprint for MarchProxy Manager (Quart)

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from quart import Blueprint, request, current_app, jsonify
from pydantic import ValidationError, BaseModel
import logging
import httpx
from datetime import datetime, timedelta
from typing import Optional, Dict, Any
from models.license import LicenseCacheModel
from middleware.auth import require_auth

logger = logging.getLogger(__name__)

license_bp = Blueprint("license", __name__, url_prefix="/api/v1/license")


class ValidateLicenseRequest(BaseModel):
    license_key: str


class LicenseStatusResponse(BaseModel):
    is_valid: bool
    is_enterprise: bool
    max_proxies: int
    tier: str
    features: Dict[str, Any]
    expires_at: Optional[datetime] = None
    error: Optional[str] = None


class LicenseKeepaliveRequest(BaseModel):
    license_key: str
    usage_stats: Optional[Dict[str, Any]] = None


@license_bp.route("/validate", methods=["POST"])
@require_auth(admin_required=True)
async def validate_license(user_data):
    """Validate license key with license server"""
    db = current_app.db

    try:
        data_json = await request.get_json()
        data = ValidateLicenseRequest(**data_json)
    except ValidationError as e:
        return jsonify({"error": "Validation error", "details": str(e)}), 400

    try:
        license_key = data.license_key
        license_server_url = current_app.config.get(
            "LICENSE_SERVER_URL", "https://license.penguintech.io"
        )

        # Check cache first
        cached = LicenseCacheModel.get_cached_validation(db, license_key)
        if cached:
            response = LicenseStatusResponse(
                is_valid=cached["is_valid"],
                is_enterprise=cached["is_enterprise"],
                max_proxies=cached["max_proxies"],
                tier="enterprise" if cached["is_enterprise"] else "community",
                features=cached.get("features", {}),
                expires_at=cached.get("expires_at"),
            )
            return jsonify(response.dict(exclude_none=True)), 200

        # Call license server
        async with httpx.AsyncClient(verify=True) as client:
            resp = await client.post(
                f"{license_server_url}/api/v2/validate",
                json={"license_key": license_key},
                timeout=10.0,
            )

        if resp.status_code == 200:
            validation_data = resp.json()
            is_valid = validation_data.get("valid", False)
            is_enterprise = validation_data.get("tier") == "enterprise"
            expires_at = None

            if validation_data.get("expires_at"):
                expires_at = datetime.fromisoformat(validation_data["expires_at"])

            # Cache the validation
            LicenseCacheModel.cache_validation(
                db, license_key, validation_data, is_valid, expires_at
            )

            response = LicenseStatusResponse(
                is_valid=is_valid,
                is_enterprise=is_enterprise,
                max_proxies=validation_data.get("max_proxies", 3),
                tier="enterprise" if is_enterprise else "community",
                features=validation_data.get("features", {}),
                expires_at=expires_at,
            )
            return jsonify(response.dict(exclude_none=True)), 200
        else:
            error_msg = resp.json().get("error", "License validation failed")
            validation_data = {"error": error_msg}

            # Cache the failed validation
            LicenseCacheModel.cache_validation(db, license_key, validation_data, False)

            response = LicenseStatusResponse(
                is_valid=False,
                is_enterprise=False,
                max_proxies=0,
                tier="community",
                features={},
                error=error_msg,
            )
            return jsonify(response.dict(exclude_none=True)), 400

    except httpx.RequestError as e:
        logger.error(f"License server connection error: {str(e)}")
        return jsonify({"error": "License server unavailable", "details": str(e)}), 503
    except Exception as e:
        logger.error(f"Error validating license: {str(e)}")
        return jsonify({"error": "Failed to validate license", "details": str(e)}), 500


@license_bp.route("/status", methods=["GET"])
@require_auth(admin_required=True)
async def get_license_status(user_data):
    """Get current license status"""
    db = current_app.db

    try:
        license_key = request.args.get("license_key")
        if not license_key:
            return jsonify({"error": "license_key parameter required"}), 400

        cached = LicenseCacheModel.get_cached_validation(db, license_key)
        if not cached:
            return jsonify({"error": "License not validated yet"}), 404

        is_enterprise = cached["is_enterprise"]
        is_valid = cached["is_valid"]

        # Check for missed keepalives
        if is_enterprise and cached.get("last_keepalive"):
            keepalive_cutoff = datetime.utcnow() - timedelta(hours=24)
            if cached["last_keepalive"] < keepalive_cutoff:
                is_valid = False

        response = LicenseStatusResponse(
            is_valid=is_valid,
            is_enterprise=is_enterprise,
            max_proxies=cached.get("max_proxies", 3),
            tier="enterprise" if is_enterprise else "community",
            features=cached.get("features", {}),
            expires_at=cached.get("expires_at"),
            error=cached.get("error_message"),
        )
        return jsonify(response.dict(exclude_none=True)), 200

    except Exception as e:
        logger.error(f"Error getting license status: {str(e)}")
        return (
            jsonify({"error": "Failed to get license status", "details": str(e)}),
            500,
        )


@license_bp.route("/keepalive", methods=["POST"])
@require_auth(admin_required=True)
async def send_keepalive(user_data):
    """Send keepalive to license server (enterprise only)"""
    db = current_app.db

    try:
        data_json = await request.get_json()
        data = LicenseKeepaliveRequest(**data_json)
    except ValidationError as e:
        return jsonify({"error": "Validation error", "details": str(e)}), 400

    try:
        license_key = data.license_key
        license_server_url = current_app.config.get(
            "LICENSE_SERVER_URL", "https://license.penguintech.io"
        )

        # Get cached license
        cached = LicenseCacheModel.get_cached_validation(db, license_key)
        if not cached or not cached["is_enterprise"]:
            return jsonify({"error": "License is not enterprise"}), 400

        # Send keepalive
        payload = {
            "license_key": license_key,
            "product_name": current_app.config.get("PRODUCT_NAME", "marchproxy"),
            "usage_stats": data.usage_stats or {},
        }

        async with httpx.AsyncClient(verify=True) as client:
            resp = await client.post(
                f"{license_server_url}/api/v2/keepalive", json=payload, timeout=10.0
            )

        if resp.status_code == 200:
            # Update keepalive timestamp in cache
            if cached.get("id"):
                db.license_cache[cached["id"]].update_record(
                    last_keepalive=datetime.utcnow(),
                    keepalive_count=cached.get("keepalive_count", 0) + 1,
                )

            return (
                jsonify(
                    {
                        "message": "Keepalive sent successfully",
                        "next_keepalive_due": (
                            datetime.utcnow() + timedelta(hours=24)
                        ).isoformat(),
                    }
                ),
                200,
            )
        else:
            error_msg = resp.json().get("error", "Keepalive failed")
            return jsonify({"error": "Keepalive failed", "details": error_msg}), 400

    except httpx.RequestError as e:
        logger.error(f"License server connection error: {str(e)}")
        return jsonify({"error": "License server unavailable", "details": str(e)}), 503
    except Exception as e:
        logger.error(f"Error sending keepalive: {str(e)}")
        return jsonify({"error": "Failed to send keepalive", "details": str(e)}), 500


@license_bp.route("/features", methods=["GET"])
@require_auth(admin_required=True)
async def check_features(user_data):
    """Check available features for license"""
    db = current_app.db

    try:
        license_key = request.args.get("license_key")
        if not license_key:
            return jsonify({"error": "license_key parameter required"}), 400

        cached = LicenseCacheModel.get_cached_validation(db, license_key)
        if not cached:
            return jsonify({"error": "License not validated yet"}), 404

        features = cached.get("features", {})
        return (
            jsonify(
                {
                    "license_key": license_key,
                    "tier": "enterprise" if cached["is_enterprise"] else "community",
                    "features": features,
                }
            ),
            200,
        )

    except Exception as e:
        logger.error(f"Error checking features: {str(e)}")
        return jsonify({"error": "Failed to check features", "details": str(e)}), 500
