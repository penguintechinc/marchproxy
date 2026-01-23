"""
Media settings models for MarchProxy Manager

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from datetime import datetime
from typing import Optional, List
from pydal import DAL, Field
from pydantic import BaseModel, validator


class MediaSettingsModel:
    """Global media settings model (single row, super admin only)"""

    @staticmethod
    def define_table(db: DAL):
        """Define media_settings table in database"""
        return db.define_table(
            "media_settings",
            Field(
                "admin_max_resolution", type="integer"
            ),  # NULL = use hardware default
            Field("admin_max_bitrate_kbps", type="integer"),
            Field(
                "enforce_codec", type="string", length=10
            ),  # NULL, 'h264', 'h265', 'av1'
            Field("transcode_ladder_enabled", type="boolean", default=True),
            Field("transcode_ladder_resolutions", type="json"),  # [360, 540, 720, 1080]
            Field("updated_by", type="reference users"),
            Field("updated_at", type="datetime", default=datetime.utcnow),
        )

    @staticmethod
    def get_settings(db: DAL) -> Optional[dict]:
        """Get global media settings (single row)"""
        settings = db(db.media_settings.id > 0).select().first()
        if not settings:
            return None
        return {
            "id": settings.id,
            "admin_max_resolution": settings.admin_max_resolution,
            "admin_max_bitrate_kbps": settings.admin_max_bitrate_kbps,
            "enforce_codec": settings.enforce_codec,
            "transcode_ladder_enabled": settings.transcode_ladder_enabled,
            "transcode_ladder_resolutions": settings.transcode_ladder_resolutions
            or [360, 540, 720, 1080],
            "updated_by": settings.updated_by,
            "updated_at": settings.updated_at,
        }

    @staticmethod
    def update_settings(
        db: DAL,
        updated_by: int,
        admin_max_resolution: Optional[int] = None,
        admin_max_bitrate_kbps: Optional[int] = None,
        enforce_codec: Optional[str] = None,
        transcode_ladder_enabled: Optional[bool] = None,
        transcode_ladder_resolutions: Optional[List[int]] = None,
    ) -> dict:
        """Update or create global media settings"""
        settings = db(db.media_settings.id > 0).select().first()

        if settings:
            # Update existing settings
            update_data = {
                "updated_by": updated_by,
                "updated_at": datetime.utcnow(),
            }
            if admin_max_resolution is not None:
                update_data["admin_max_resolution"] = (
                    admin_max_resolution if admin_max_resolution > 0 else None
                )
            if admin_max_bitrate_kbps is not None:
                update_data["admin_max_bitrate_kbps"] = (
                    admin_max_bitrate_kbps if admin_max_bitrate_kbps > 0 else None
                )
            if enforce_codec is not None:
                update_data["enforce_codec"] = (
                    enforce_codec if enforce_codec != "" else None
                )
            if transcode_ladder_enabled is not None:
                update_data["transcode_ladder_enabled"] = transcode_ladder_enabled
            if transcode_ladder_resolutions is not None:
                update_data[
                    "transcode_ladder_resolutions"
                ] = transcode_ladder_resolutions

            settings.update_record(**update_data)
        else:
            # Create new settings
            db.media_settings.insert(
                admin_max_resolution=(
                    admin_max_resolution
                    if admin_max_resolution and admin_max_resolution > 0
                    else None
                ),
                admin_max_bitrate_kbps=(
                    admin_max_bitrate_kbps
                    if admin_max_bitrate_kbps and admin_max_bitrate_kbps > 0
                    else None
                ),
                enforce_codec=(
                    enforce_codec if enforce_codec and enforce_codec != "" else None
                ),
                transcode_ladder_enabled=(
                    transcode_ladder_enabled
                    if transcode_ladder_enabled is not None
                    else True
                ),
                transcode_ladder_resolutions=transcode_ladder_resolutions
                or [360, 540, 720, 1080],
                updated_by=updated_by,
                updated_at=datetime.utcnow(),
            )

        return MediaSettingsModel.get_settings(db)

    @staticmethod
    def clear_admin_override(db: DAL, updated_by: int) -> dict:
        """Clear admin resolution override (revert to hardware default)"""
        settings = db(db.media_settings.id > 0).select().first()
        if settings:
            settings.update_record(
                admin_max_resolution=None,
                updated_by=updated_by,
                updated_at=datetime.utcnow(),
            )
        return MediaSettingsModel.get_settings(db)


class MediaStreamModel:
    """Model for tracking active media streams"""

    @staticmethod
    def define_table(db: DAL):
        """Define media_streams table in database"""
        return db.define_table(
            "media_streams",
            Field("stream_key", type="string", unique=True, required=True, length=100),
            Field(
                "protocol", type="string", required=True, length=20
            ),  # rtmp, srt, webrtc
            Field("codec", type="string", length=10),  # h264, h265, av1
            Field("resolution", type="string", length=20),  # e.g., "1920x1080"
            Field("bitrate_kbps", type="integer"),
            Field(
                "status", type="string", default="active", length=20
            ),  # active, idle, error
            Field("client_ip", type="string", length=45),
            Field("user_agent", type="string", length=255),
            Field("started_at", type="datetime", default=datetime.utcnow),
            Field("ended_at", type="datetime"),
            Field("bytes_in", type="bigint", default=0),
            Field("bytes_out", type="bigint", default=0),
            Field("metadata", type="json"),
        )

    @staticmethod
    def create_stream(
        db: DAL,
        stream_key: str,
        protocol: str,
        client_ip: str,
        codec: str = None,
        resolution: str = None,
        bitrate_kbps: int = None,
        user_agent: str = None,
    ) -> int:
        """Register new active stream"""
        return db.media_streams.insert(
            stream_key=stream_key,
            protocol=protocol,
            codec=codec,
            resolution=resolution,
            bitrate_kbps=bitrate_kbps,
            status="active",
            client_ip=client_ip,
            user_agent=user_agent,
        )

    @staticmethod
    def end_stream(db: DAL, stream_key: str, bytes_in: int = 0, bytes_out: int = 0):
        """Mark stream as ended"""
        stream = db(db.media_streams.stream_key == stream_key).select().first()
        if stream:
            stream.update_record(
                status="idle",
                ended_at=datetime.utcnow(),
                bytes_in=bytes_in,
                bytes_out=bytes_out,
            )

    @staticmethod
    def get_active_streams(db: DAL) -> list:
        """Get all active streams"""
        streams = db(db.media_streams.status == "active").select()
        return [dict(s) for s in streams]

    @staticmethod
    def get_stream(db: DAL, stream_key: str) -> Optional[dict]:
        """Get stream by key"""
        stream = db(db.media_streams.stream_key == stream_key).select().first()
        if stream:
            return dict(stream)
        return None


# Pydantic request/response models


class UpdateMediaSettingsRequest(BaseModel):
    """Request to update global media settings"""

    admin_max_resolution: Optional[int] = None
    admin_max_bitrate_kbps: Optional[int] = None
    enforce_codec: Optional[str] = None
    transcode_ladder_enabled: Optional[bool] = None
    transcode_ladder_resolutions: Optional[List[int]] = None

    @validator("admin_max_resolution")
    def validate_resolution(cls, v):
        if v is not None and v != 0:
            valid = [360, 480, 540, 720, 1080, 1440, 2160, 4320]
            if v not in valid:
                raise ValueError(f"Resolution must be one of: {valid}")
        return v

    @validator("enforce_codec")
    def validate_codec(cls, v):
        if v is not None and v != "":
            valid = ["h264", "h265", "av1"]
            if v not in valid:
                raise ValueError(f"Codec must be one of: {valid}")
        return v

    @validator("transcode_ladder_resolutions")
    def validate_ladder(cls, v):
        if v is not None:
            valid = [360, 480, 540, 720, 1080, 1440, 2160, 4320]
            for res in v:
                if res not in valid:
                    raise ValueError(f"Ladder resolution must be one of: {valid}")
        return v


class MediaSettingsResponse(BaseModel):
    """Response for media settings"""

    admin_max_resolution: Optional[int]
    admin_max_bitrate_kbps: Optional[int]
    enforce_codec: Optional[str]
    transcode_ladder_enabled: bool
    transcode_ladder_resolutions: List[int]
    updated_at: Optional[datetime]


class HardwareCapabilitiesResponse(BaseModel):
    """Response for hardware capabilities"""

    gpu_type: str
    gpu_model: Optional[str]
    vram_gb: int
    hardware_max_resolution: int
    admin_max_resolution: Optional[int]
    effective_max_resolution: int
    av1_supported: bool
    supports_8k: bool
    supports_4k: bool


class MediaCapabilitiesResponse(BaseModel):
    """Combined media capabilities response"""

    settings: MediaSettingsResponse
    hardware: HardwareCapabilitiesResponse


class CreateRestreamRequest(BaseModel):
    """Request to create restream destination"""

    platform: str  # twitch, youtube, facebook, custom
    rtmp_url: str
    stream_key: str
    quality: str  # 360p, 720p, 1080p, etc.
    enabled: bool = True


class MediaStreamResponse(BaseModel):
    """Response for media stream info"""

    id: int
    stream_key: str
    protocol: str
    codec: Optional[str]
    resolution: Optional[str]
    bitrate_kbps: Optional[int]
    status: str
    client_ip: str
    started_at: datetime
    bytes_in: int
    bytes_out: int
