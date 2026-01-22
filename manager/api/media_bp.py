"""
Media module API Blueprint for MarchProxy Manager (Quart)

Provides endpoints for managing media streaming configuration,
active streams, and restreaming destinations.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from quart import Blueprint, request, current_app, jsonify
from pydantic import ValidationError
import logging
from datetime import datetime
from models.media_settings import (
    MediaSettingsModel,
    MediaStreamModel,
    MediaSettingsResponse,
    MediaStreamResponse,
    CreateRestreamRequest,
)
from middleware.auth import require_auth

logger = logging.getLogger(__name__)

media_bp = Blueprint("media", __name__, url_prefix="/api/v1/modules/rtmp")


@media_bp.route("/config", methods=["GET", "PUT"])
@require_auth()
async def media_config(user_data):
    """Get or update media module configuration"""
    db = current_app.db

    if request.method == "GET":
        # Get current settings
        settings = MediaSettingsModel.get_settings(db)
        if not settings:
            # Return defaults if no settings exist
            settings = {
                "admin_max_resolution": None,
                "admin_max_bitrate_kbps": None,
                "enforce_codec": None,
                "transcode_ladder_enabled": True,
                "transcode_ladder_resolutions": [360, 540, 720, 1080],
                "updated_at": None,
            }

        return jsonify({"config": settings, "status": "ok"}), 200

    elif request.method == "PUT":
        # Only admins can update config
        if not user_data.get("is_admin", False):
            return jsonify({"error": "Admin access required"}), 403

        try:
            data_json = await request.get_json()
            # Validate transcode ladder if provided
            if "transcode_ladder_resolutions" in data_json:
                valid_res = [360, 480, 540, 720, 1080, 1440, 2160, 4320]
                for res in data_json["transcode_ladder_resolutions"]:
                    if res not in valid_res:
                        return jsonify({
                            "error": f"Invalid resolution {res}. Valid: {valid_res}"
                        }), 400

        except Exception as e:
            return jsonify({"error": "Invalid JSON", "details": str(e)}), 400

        try:
            settings = MediaSettingsModel.update_settings(
                db,
                updated_by=user_data["user_id"],
                transcode_ladder_enabled=data_json.get("transcode_ladder_enabled"),
                transcode_ladder_resolutions=data_json.get(
                    "transcode_ladder_resolutions"
                ),
            )

            return jsonify({"config": settings, "status": "updated"}), 200

        except Exception as e:
            logger.error(f"Error updating media config: {str(e)}")
            return jsonify({"error": "Failed to update config"}), 500


@media_bp.route("/streams", methods=["GET"])
@require_auth()
async def list_streams(user_data):
    """List all active media streams"""
    db = current_app.db

    try:
        streams = MediaStreamModel.get_active_streams(db)

        result = []
        for stream in streams:
            started_at = stream["started_at"]
            result.append({
                "id": stream["id"],
                "stream_key": stream["stream_key"],
                "protocol": stream["protocol"],
                "codec": stream["codec"],
                "resolution": stream["resolution"],
                "bitrate_kbps": stream["bitrate_kbps"],
                "status": stream["status"],
                "client_ip": stream["client_ip"],
                "started_at": started_at.isoformat() if started_at else None,
                "bytes_in": stream["bytes_in"],
                "bytes_out": stream["bytes_out"],
            })

        return jsonify({"streams": result, "count": len(result)}), 200

    except Exception as e:
        logger.error(f"Error listing streams: {str(e)}")
        return jsonify({"error": "Failed to list streams"}), 500


@media_bp.route("/streams/<stream_key>", methods=["GET", "DELETE"])
@require_auth()
async def stream_detail(user_data, stream_key):
    """Get or stop a specific stream"""
    db = current_app.db

    if request.method == "GET":
        stream = MediaStreamModel.get_stream(db, stream_key)
        if not stream:
            return jsonify({"error": "Stream not found"}), 404

        started_at = stream["started_at"]
        ended_at = stream.get("ended_at")
        return jsonify({
            "stream": {
                "id": stream["id"],
                "stream_key": stream["stream_key"],
                "protocol": stream["protocol"],
                "codec": stream["codec"],
                "resolution": stream["resolution"],
                "bitrate_kbps": stream["bitrate_kbps"],
                "status": stream["status"],
                "client_ip": stream["client_ip"],
                "started_at": started_at.isoformat() if started_at else None,
                "ended_at": ended_at.isoformat() if ended_at else None,
                "bytes_in": stream["bytes_in"],
                "bytes_out": stream["bytes_out"],
            }
        }), 200

    elif request.method == "DELETE":
        # Only admins can stop streams
        if not user_data.get("is_admin", False):
            return jsonify({"error": "Admin access required"}), 403

        stream = MediaStreamModel.get_stream(db, stream_key)
        if not stream:
            return jsonify({"error": "Stream not found"}), 404

        try:
            # Mark stream as ended in DB
            MediaStreamModel.end_stream(db, stream_key)

            # TODO: Send gRPC command to proxy-rtmp to actually stop the stream

            logger.info(f"Stream {stream_key} stopped by user {user_data['user_id']}")
            return jsonify({"status": "stopped", "stream_key": stream_key}), 200

        except Exception as e:
            logger.error(f"Error stopping stream: {str(e)}")
            return jsonify({"error": "Failed to stop stream"}), 500


@media_bp.route("/capabilities", methods=["GET"])
@require_auth()
async def get_capabilities(user_data):
    """Get media module hardware capabilities and current limits"""
    db = current_app.db

    # Get admin settings
    settings = MediaSettingsModel.get_settings(db)

    # TODO: Query actual hardware capabilities from proxy-rtmp via gRPC
    # For now, return mock data
    hardware = {
        "gpu_type": "nvidia",
        "gpu_model": "NVIDIA GeForce RTX 4080",
        "vram_gb": 16,
        "hardware_max_resolution": 4320,  # 8K
        "av1_supported": True,
        "supports_8k": True,
        "supports_4k": True,
    }

    # Calculate effective max resolution
    admin_max = settings.get("admin_max_resolution") if settings else None
    hardware_max = hardware["hardware_max_resolution"]
    effective_max = min(admin_max, hardware_max) if admin_max else hardware_max

    ladder_default = [360, 540, 720, 1080]
    return jsonify({
        "hardware": hardware,
        "settings": {
            "admin_max_resolution": admin_max,
            "enforce_codec": settings.get("enforce_codec") if settings else None,
            "transcode_ladder_enabled": (
                settings.get("transcode_ladder_enabled", True) if settings else True
            ),
            "transcode_ladder_resolutions": (
                settings.get("transcode_ladder_resolutions", ladder_default)
                if settings
                else ladder_default
            ),
        },
        "effective_max_resolution": effective_max,
    }), 200


@media_bp.route("/streams/<stream_key>/restream", methods=["GET", "POST", "DELETE"])
@require_auth()
async def manage_restream(user_data, stream_key):
    """Manage restreaming destinations for a stream"""
    db = current_app.db

    # Check stream exists
    stream = MediaStreamModel.get_stream(db, stream_key)
    if not stream:
        return jsonify({"error": "Stream not found"}), 404

    if request.method == "GET":
        # TODO: Get restream destinations from database
        # For now, return empty list
        return jsonify({"stream_key": stream_key, "destinations": []}), 200

    elif request.method == "POST":
        if not user_data.get("is_admin", False):
            return jsonify({"error": "Admin access required"}), 403

        try:
            data_json = await request.get_json()
            restream = CreateRestreamRequest(**data_json)
        except ValidationError as e:
            return jsonify({"error": "Validation error", "details": str(e)}), 400

        # TODO: Store restream config and notify proxy-rtmp via gRPC
        logger.info(f"Restream created for {stream_key} to {restream.platform}")

        return jsonify({
            "status": "created",
            "stream_key": stream_key,
            "destination": {
                "platform": restream.platform,
                "quality": restream.quality,
                "enabled": restream.enabled,
            },
        }), 201

    elif request.method == "DELETE":
        if not user_data.get("is_admin", False):
            return jsonify({"error": "Admin access required"}), 403

        # TODO: Remove restream destination
        return jsonify({"status": "deleted", "stream_key": stream_key}), 200


@media_bp.route("/stats", methods=["GET"])
@require_auth()
async def get_stats(user_data):
    """Get media module statistics"""
    db = current_app.db

    try:
        active_streams = MediaStreamModel.get_active_streams(db)

        # Calculate totals
        total_bytes_in = sum(s.get("bytes_in", 0) for s in active_streams)
        total_bytes_out = sum(s.get("bytes_out", 0) for s in active_streams)

        # Group by protocol
        by_protocol = {}
        for stream in active_streams:
            proto = stream.get("protocol", "unknown")
            by_protocol[proto] = by_protocol.get(proto, 0) + 1

        return jsonify({
            "stats": {
                "active_streams": len(active_streams),
                "total_bytes_in": total_bytes_in,
                "total_bytes_out": total_bytes_out,
                "by_protocol": by_protocol,
                "timestamp": datetime.utcnow().isoformat(),
            }
        }), 200

    except Exception as e:
        logger.error(f"Error getting stats: {str(e)}")
        return jsonify({"error": "Failed to get stats"}), 500
