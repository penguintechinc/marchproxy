"""
Admin Media Settings API Blueprint for MarchProxy Manager (Quart)

Super admin only endpoints for global media settings management.
Controls resolution limits, codec enforcement, and transcode ladders.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from quart import Blueprint, request, current_app, jsonify
from pydantic import ValidationError
import logging
from datetime import datetime
from models.media_settings import (
    MediaSettingsModel,
    UpdateMediaSettingsRequest,
    MediaSettingsResponse,
)
from middleware.auth import require_auth

logger = logging.getLogger(__name__)

admin_media_bp = Blueprint("admin_media", __name__, url_prefix="/api/v1/admin/media")


@admin_media_bp.route("/settings", methods=["GET", "PUT"])
@require_auth(admin_required=True)
async def admin_media_settings(user_data):
    """
    Get or update global media settings (super admin only)

    These settings apply to ALL streams across all communities.
    Use with caution as they affect the entire system.
    """
    db = current_app.db

    # Verify super admin (not just cluster admin)
    if not user_data.get("is_admin", False):
        return jsonify({"error": "Super admin access required"}), 403

    if request.method == "GET":
        # Get current settings
        settings = MediaSettingsModel.get_settings(db)

        # Get hardware capabilities (from cache or gRPC)
        # TODO: Implement actual gRPC call to proxy-rtmp
        hardware_caps = await get_hardware_capabilities()

        # Calculate effective max
        admin_max = settings.get("admin_max_resolution") if settings else None
        hardware_max = hardware_caps.get("hardware_max_resolution", 1440)
        effective_max = min(admin_max, hardware_max) if admin_max else hardware_max

        ladder_default = [360, 540, 720, 1080]
        return (
            jsonify(
                {
                    "settings": {
                        "admin_max_resolution": admin_max,
                        "admin_max_bitrate_kbps": (
                            settings.get("admin_max_bitrate_kbps") if settings else None
                        ),
                        "enforce_codec": (
                            settings.get("enforce_codec") if settings else None
                        ),
                        "transcode_ladder_enabled": (
                            settings.get("transcode_ladder_enabled", True)
                            if settings
                            else True
                        ),
                        "transcode_ladder_resolutions": (
                            settings.get("transcode_ladder_resolutions", ladder_default)
                            if settings
                            else ladder_default
                        ),
                        "updated_at": (
                            settings.get("updated_at").isoformat()
                            if settings and settings.get("updated_at")
                            else None
                        ),
                    },
                    "hardware_capabilities": hardware_caps,
                    "effective_max_resolution": effective_max,
                }
            ),
            200,
        )

    elif request.method == "PUT":
        try:
            data_json = await request.get_json()
            data = UpdateMediaSettingsRequest(**data_json)
        except ValidationError as e:
            return jsonify({"error": "Validation error", "details": str(e)}), 400

        try:
            # Update settings
            settings = MediaSettingsModel.update_settings(
                db,
                updated_by=user_data["user_id"],
                admin_max_resolution=data.admin_max_resolution,
                admin_max_bitrate_kbps=data.admin_max_bitrate_kbps,
                enforce_codec=data.enforce_codec,
                transcode_ladder_enabled=data.transcode_ladder_enabled,
                transcode_ladder_resolutions=data.transcode_ladder_resolutions,
            )

            # Notify proxy-rtmp of config change via gRPC
            await notify_rtmp_config_change(settings)

            logger.info(f"Media settings updated by admin {user_data['user_id']}")

            return jsonify({
                "status": "updated",
                "settings": settings,
            }), 200

        except Exception as e:
            logger.error(f"Error updating media settings: {str(e)}")
            return jsonify({"error": "Failed to update settings"}), 500


@admin_media_bp.route("/settings/reset", methods=["POST"])
@require_auth(admin_required=True)
async def reset_admin_override(user_data):
    """
    Reset admin resolution override to hardware default

    This removes any administrator-imposed resolution limit,
    allowing the system to use the full hardware capability.
    """
    db = current_app.db

    if not user_data.get("is_admin", False):
        return jsonify({"error": "Super admin access required"}), 403

    try:
        settings = MediaSettingsModel.clear_admin_override(
            db, updated_by=user_data["user_id"]
        )

        # Notify proxy-rtmp
        await notify_rtmp_config_change(settings)

        logger.info(
            f"Media settings reset to hardware default by admin {user_data['user_id']}"
        )

        return jsonify({
            "status": "reset",
            "message": "Resolution limit reset to hardware default",
            "settings": settings,
        }), 200

    except Exception as e:
        logger.error(f"Error resetting media settings: {str(e)}")
        return jsonify({"error": "Failed to reset settings"}), 500


@admin_media_bp.route("/capabilities", methods=["GET"])
@require_auth(admin_required=True)
async def admin_capabilities(user_data):
    """
    Get detailed hardware capabilities report (super admin only)

    Includes GPU information, VRAM, supported codecs, and resolution limits.
    """
    if not user_data.get("is_admin", False):
        return jsonify({"error": "Super admin access required"}), 403

    hardware_caps = await get_hardware_capabilities()

    # Add detailed resolution support info
    resolutions = []
    for height in [360, 480, 540, 720, 1080, 1440, 2160, 4320]:
        hw_max = hardware_caps.get("hardware_max_resolution", 1440)
        res_info = {
            "height": height,
            "label": get_resolution_label(height),
            "supported": height <= hw_max,
            "requires_gpu": height > 1440,
        }

        # Add reason if not supported
        if not res_info["supported"]:
            if hardware_caps.get("gpu_type") == "none":
                res_info["disabled_reason"] = "Requires GPU hardware acceleration"
            elif height > hw_max:
                res_info["disabled_reason"] = (
                    f"GPU does not support {height}p (requires more VRAM)"
                )

        resolutions.append(res_info)

    return jsonify({
        "hardware": hardware_caps,
        "resolutions": resolutions,
        "supported_codecs": get_supported_codecs(hardware_caps),
    }), 200


async def get_hardware_capabilities() -> dict:
    """
    Get hardware capabilities from proxy-rtmp via gRPC

    TODO: Implement actual gRPC call
    For now, returns mock data
    """
    # Mock response - replace with actual gRPC call
    return {
        "gpu_type": "nvidia",
        "gpu_model": "NVIDIA GeForce RTX 4080",
        "vram_gb": 16,
        "hardware_max_resolution": 4320,  # 8K capable
        "av1_supported": True,
        "supports_8k": True,
        "supports_4k": True,
    }


async def notify_rtmp_config_change(settings: dict):
    """
    Notify proxy-rtmp module of config change via gRPC

    TODO: Implement actual gRPC call to UpdatePolicy
    """
    logger.info(f"Notifying proxy-rtmp of config change: {settings}")
    # Placeholder for gRPC call
    pass


def get_resolution_label(height: int) -> str:
    """Get human-readable label for resolution"""
    labels = {
        360: "360p",
        480: "480p (SD)",
        540: "540p",
        720: "720p (HD)",
        1080: "1080p (Full HD)",
        1440: "1440p (2K)",
        2160: "2160p (4K)",
        4320: "4320p (8K)",
    }
    return labels.get(height, f"{height}p")


def get_supported_codecs(hardware_caps: dict) -> list:
    """Get list of supported codecs based on hardware"""
    has_gpu = hardware_caps.get("gpu_type") != "none"
    av1_ok = hardware_caps.get("av1_supported", False)
    codecs = [
        {
            "name": "H.264",
            "id": "h264",
            "supported": True,
            "hardware_accelerated": has_gpu,
        },
        {
            "name": "H.265/HEVC",
            "id": "h265",
            "supported": True,
            "hardware_accelerated": has_gpu,
        },
        {
            "name": "AV1",
            "id": "av1",
            "supported": av1_ok,
            "hardware_accelerated": av1_ok,
        },
    ]
    return codecs
