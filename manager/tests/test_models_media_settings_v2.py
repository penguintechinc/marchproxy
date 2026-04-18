#!/usr/bin/env python3
"""
Comprehensive tests for models/media_settings.py to improve coverage.

Tests for MediaSettingsModel (get_settings, update_settings, clear_admin_override)
and MediaStreamModel (create_stream, end_stream, get_active_streams, get_stream).
"""

import pytest
from unittest.mock import MagicMock, patch
from datetime import datetime
from models.media_settings import (
    MediaSettingsModel,
    MediaStreamModel,
    UpdateMediaSettingsRequest,
)


class TestMediaSettingsModelGetSettings:
    """Test MediaSettingsModel.get_settings - covers lines 33-48."""

    def test_get_settings_returns_none_when_no_settings(self, mock_db):
        """Test get_settings returns None when no record exists (line 36-37)."""
        mock_db.return_value.select.return_value.first.return_value = None

        result = MediaSettingsModel.get_settings(mock_db)

        assert result is None

    def test_get_settings_returns_dict_with_defaults(self, mock_db):
        """Test get_settings returns dict with default ladder when null (line 45)."""
        settings_row = MagicMock()
        settings_row.id = 1
        settings_row.admin_max_resolution = 1080
        settings_row.admin_max_bitrate_kbps = 5000
        settings_row.enforce_codec = "h264"
        settings_row.transcode_ladder_enabled = True
        settings_row.transcode_ladder_resolutions = None  # Test default
        settings_row.updated_by = 1
        settings_row.updated_at = datetime(2025, 1, 22, 12, 0, 0)

        mock_db.return_value.select.return_value.first.return_value = settings_row

        result = MediaSettingsModel.get_settings(mock_db)

        assert result is not None
        assert result["id"] == 1
        assert result["admin_max_resolution"] == 1080
        assert result["transcode_ladder_resolutions"] == [360, 540, 720, 1080]
        assert result["updated_by"] == 1

    def test_get_settings_returns_dict_with_custom_ladder(self, mock_db):
        """Test get_settings returns custom transcode_ladder_resolutions."""
        settings_row = MagicMock()
        settings_row.id = 1
        settings_row.admin_max_resolution = None
        settings_row.admin_max_bitrate_kbps = None
        settings_row.enforce_codec = None
        settings_row.transcode_ladder_enabled = False
        settings_row.transcode_ladder_resolutions = [480, 720, 1080]  # Custom
        settings_row.updated_by = 2
        settings_row.updated_at = datetime(2025, 1, 22, 12, 0, 0)

        mock_db.return_value.select.return_value.first.return_value = settings_row

        result = MediaSettingsModel.get_settings(mock_db)

        assert result["transcode_ladder_resolutions"] == [480, 720, 1080]
        assert result["admin_max_resolution"] is None


class TestMediaSettingsModelUpdateSettings:
    """Test MediaSettingsModel.update_settings - covers lines 51-107."""

    def test_update_settings_creates_new_when_none_exists(self, mock_db):
        """Test update_settings creates new record when none exists (lines 85-105)."""
        mock_db.return_value.select.return_value.first.return_value = None
        mock_db.media_settings.insert = MagicMock(return_value=1)

        with patch.object(
            MediaSettingsModel,
            "get_settings",
            return_value={
                "id": 1,
                "admin_max_resolution": 1080,
                "admin_max_bitrate_kbps": 5000,
                "enforce_codec": "h264",
                "transcode_ladder_enabled": True,
                "transcode_ladder_resolutions": [360, 540, 720, 1080],
                "updated_by": 1,
                "updated_at": datetime.utcnow(),
            },
        ):
            result = MediaSettingsModel.update_settings(
                mock_db,
                updated_by=1,
                admin_max_resolution=1080,
                admin_max_bitrate_kbps=5000,
                enforce_codec="h264",
            )

        assert result is not None
        assert mock_db.media_settings.insert.called

    def test_update_settings_handles_zero_resolution(self, mock_db):
        """Test update_settings converts 0 to None for resolution (lines 70-72)."""
        existing = MagicMock()
        existing.update_record = MagicMock()

        mock_db.return_value.select.return_value.first.return_value = existing

        with patch.object(MediaSettingsModel, "get_settings", return_value={}):
            MediaSettingsModel.update_settings(
                mock_db, updated_by=1, admin_max_resolution=0
            )

        call_kwargs = existing.update_record.call_args[1]
        assert call_kwargs["admin_max_resolution"] is None

    def test_update_settings_handles_zero_bitrate(self, mock_db):
        """Test update_settings converts 0 to None for bitrate (lines 73-76)."""
        existing = MagicMock()
        existing.update_record = MagicMock()

        mock_db.return_value.select.return_value.first.return_value = existing

        with patch.object(MediaSettingsModel, "get_settings", return_value={}):
            MediaSettingsModel.update_settings(
                mock_db, updated_by=1, admin_max_bitrate_kbps=0
            )

        call_kwargs = existing.update_record.call_args[1]
        assert call_kwargs["admin_max_bitrate_kbps"] is None

    def test_update_settings_handles_empty_codec(self, mock_db):
        """Test update_settings converts empty string to None for codec (lines 77-78)."""
        existing = MagicMock()
        existing.update_record = MagicMock()

        mock_db.return_value.select.return_value.first.return_value = existing

        with patch.object(MediaSettingsModel, "get_settings", return_value={}):
            MediaSettingsModel.update_settings(mock_db, updated_by=1, enforce_codec="")

        call_kwargs = existing.update_record.call_args[1]
        assert call_kwargs["enforce_codec"] is None

    def test_update_settings_updates_transcode_ladder(self, mock_db):
        """Test update_settings updates transcode_ladder_resolutions (lines 81-82)."""
        existing = MagicMock()
        existing.update_record = MagicMock()

        mock_db.return_value.select.return_value.first.return_value = existing

        with patch.object(MediaSettingsModel, "get_settings", return_value={}):
            MediaSettingsModel.update_settings(
                mock_db,
                updated_by=1,
                transcode_ladder_resolutions=[480, 720, 1080],
            )

        call_kwargs = existing.update_record.call_args[1]
        assert call_kwargs["transcode_ladder_resolutions"] == [480, 720, 1080]

    def test_update_settings_preserves_positive_resolution(self, mock_db):
        """Test positive resolution values are preserved."""
        existing = MagicMock()
        existing.update_record = MagicMock()

        mock_db.return_value.select.return_value.first.return_value = existing

        with patch.object(MediaSettingsModel, "get_settings", return_value={}):
            MediaSettingsModel.update_settings(
                mock_db, updated_by=1, admin_max_resolution=1440
            )

        call_kwargs = existing.update_record.call_args[1]
        assert call_kwargs["admin_max_resolution"] == 1440

    def test_update_settings_creates_with_default_ladder(self, mock_db):
        """Test new settings creation uses default ladder (line 102)."""
        mock_db.return_value.select.return_value.first.return_value = None
        mock_db.media_settings.insert = MagicMock(return_value=1)

        with patch.object(MediaSettingsModel, "get_settings", return_value={}):
            MediaSettingsModel.update_settings(mock_db, updated_by=1)

        call_kwargs = mock_db.media_settings.insert.call_args[1]
        assert call_kwargs["transcode_ladder_resolutions"] == [360, 540, 720, 1080]

    def test_update_settings_creates_with_default_transcode_enabled(self, mock_db):
        """Test new settings creation defaults transcode_ladder_enabled to True (line 100)."""
        mock_db.return_value.select.return_value.first.return_value = None
        mock_db.media_settings.insert = MagicMock(return_value=1)

        with patch.object(MediaSettingsModel, "get_settings", return_value={}):
            MediaSettingsModel.update_settings(mock_db, updated_by=1)

        call_kwargs = mock_db.media_settings.insert.call_args[1]
        assert call_kwargs["transcode_ladder_enabled"] is True

    def test_update_settings_updates_all_fields(self, mock_db):
        """Test update_settings updates existing record with all fields (line 84)."""
        existing = MagicMock()
        existing.update_record = MagicMock()

        mock_db.return_value.select.return_value.first.return_value = existing

        with patch.object(MediaSettingsModel, "get_settings", return_value={}):
            MediaSettingsModel.update_settings(
                mock_db,
                updated_by=2,
                admin_max_resolution=2160,
                admin_max_bitrate_kbps=8000,
                enforce_codec="h265",
                transcode_ladder_enabled=False,
                transcode_ladder_resolutions=[720, 1080],
            )

        assert existing.update_record.called
        call_kwargs = existing.update_record.call_args[1]
        assert call_kwargs["updated_by"] == 2
        assert call_kwargs["admin_max_resolution"] == 2160


class TestMediaSettingsModelClearAdminOverride:
    """Test MediaSettingsModel.clear_admin_override - covers lines 110-119."""

    def test_clear_admin_override_updates_existing(self, mock_db):
        """Test clear_admin_override updates existing settings (lines 113-118)."""
        existing = MagicMock()
        existing.update_record = MagicMock()

        mock_db.return_value.select.return_value.first.return_value = existing

        with patch.object(MediaSettingsModel, "get_settings", return_value={}):
            MediaSettingsModel.clear_admin_override(mock_db, updated_by=1)

        assert existing.update_record.called
        call_kwargs = existing.update_record.call_args[1]
        assert call_kwargs["admin_max_resolution"] is None
        assert call_kwargs["updated_by"] == 1

    def test_clear_admin_override_handles_no_existing(self, mock_db):
        """Test clear_admin_override handles non-existent settings (line 113)."""
        mock_db.return_value.select.return_value.first.return_value = None

        with patch.object(MediaSettingsModel, "get_settings", return_value=None):
            result = MediaSettingsModel.clear_admin_override(mock_db, updated_by=1)

        assert result is None


class TestUpdateMediaSettingsRequestValidators:
    """Test Pydantic validators for UpdateMediaSettingsRequest - covers lines 207-230."""

    def test_validate_resolution_valid_values(self):
        """Test resolution validator accepts valid values (lines 207-213)."""
        valid_resolutions = [360, 480, 540, 720, 1080, 1440, 2160, 4320]
        for res in valid_resolutions:
            req = UpdateMediaSettingsRequest(admin_max_resolution=res)
            assert req.admin_max_resolution == res

    def test_validate_resolution_zero_allowed(self):
        """Test resolution validator allows 0 (for clearing override)."""
        req = UpdateMediaSettingsRequest(admin_max_resolution=0)
        assert req.admin_max_resolution == 0

    def test_validate_resolution_none_allowed(self):
        """Test resolution validator allows None."""
        req = UpdateMediaSettingsRequest(admin_max_resolution=None)
        assert req.admin_max_resolution is None

    def test_validate_resolution_invalid_value(self):
        """Test resolution validator rejects invalid values (line 212)."""
        with pytest.raises(ValueError, match="Resolution must be one of"):
            UpdateMediaSettingsRequest(admin_max_resolution=800)

    def test_validate_codec_valid_values(self):
        """Test codec validator accepts valid values (lines 216-221)."""
        valid_codecs = ["h264", "h265", "av1"]
        for codec in valid_codecs:
            req = UpdateMediaSettingsRequest(enforce_codec=codec)
            assert req.enforce_codec == codec

    def test_validate_codec_empty_string_allowed(self):
        """Test codec validator allows empty string (for clearing)."""
        req = UpdateMediaSettingsRequest(enforce_codec="")
        assert req.enforce_codec == ""

    def test_validate_codec_none_allowed(self):
        """Test codec validator allows None."""
        req = UpdateMediaSettingsRequest(enforce_codec=None)
        assert req.enforce_codec is None

    def test_validate_codec_invalid_value(self):
        """Test codec validator rejects invalid values (line 220)."""
        with pytest.raises(ValueError, match="Codec must be one of"):
            UpdateMediaSettingsRequest(enforce_codec="vp9")

    def test_validate_ladder_valid_values(self):
        """Test ladder validator accepts valid resolutions (lines 223-230)."""
        ladder = [360, 540, 1080]
        req = UpdateMediaSettingsRequest(transcode_ladder_resolutions=ladder)
        assert req.transcode_ladder_resolutions == ladder

    def test_validate_ladder_invalid_resolution(self):
        """Test ladder validator rejects invalid resolution (line 229)."""
        with pytest.raises(ValueError, match="Ladder resolution must be one of"):
            UpdateMediaSettingsRequest(transcode_ladder_resolutions=[360, 900, 1080])

    def test_validate_ladder_none_allowed(self):
        """Test ladder validator allows None."""
        req = UpdateMediaSettingsRequest(transcode_ladder_resolutions=None)
        assert req.transcode_ladder_resolutions is None


class TestMediaStreamModelCreateStream:
    """Test MediaStreamModel.create_stream - covers lines 146-166."""

    def test_create_stream_with_all_params(self, mock_db):
        """Test create_stream with all optional parameters."""
        mock_db.media_streams.insert = MagicMock(return_value=1)

        stream_id = MediaStreamModel.create_stream(
            mock_db,
            stream_key="test-key",
            protocol="rtmp",
            client_ip="192.168.1.1",
            codec="h264",
            resolution="1920x1080",
            bitrate_kbps=5000,
            user_agent="Firefox/120",
        )

        assert stream_id == 1
        assert mock_db.media_streams.insert.called
        call_kwargs = mock_db.media_streams.insert.call_args[1]
        assert call_kwargs["stream_key"] == "test-key"
        assert call_kwargs["status"] == "active"
        assert call_kwargs["codec"] == "h264"

    def test_create_stream_with_minimal_params(self, mock_db):
        """Test create_stream with only required parameters."""
        mock_db.media_streams.insert = MagicMock(return_value=1)

        stream_id = MediaStreamModel.create_stream(
            mock_db,
            stream_key="min-key",
            protocol="srt",
            client_ip="10.0.0.1",
        )

        assert stream_id == 1
        call_kwargs = mock_db.media_streams.insert.call_args[1]
        assert call_kwargs["codec"] is None
        assert call_kwargs["resolution"] is None


class TestMediaStreamModelEndStream:
    """Test MediaStreamModel.end_stream - covers lines 169-178."""

    def test_end_stream_updates_status_and_bytes(self):
        """Test end_stream updates record with status, ended_at, bytes (lines 173-178)."""
        stream = MagicMock()
        stream.update_record = MagicMock()

        mock_db = MagicMock()
        mock_db.return_value.select.return_value.first.return_value = stream

        MediaStreamModel.end_stream(
            mock_db, stream_key="test-key", bytes_in=1000, bytes_out=5000
        )

        assert stream.update_record.called
        call_kwargs = stream.update_record.call_args[1]
        assert call_kwargs["status"] == "idle"
        assert call_kwargs["bytes_in"] == 1000
        assert call_kwargs["bytes_out"] == 5000
        assert "ended_at" in call_kwargs

    def test_end_stream_with_default_bytes(self):
        """Test end_stream with default bytes values."""
        stream = MagicMock()
        stream.update_record = MagicMock()

        mock_db = MagicMock()
        mock_db.return_value.select.return_value.first.return_value = stream

        MediaStreamModel.end_stream(mock_db, stream_key="test-key")

        call_kwargs = stream.update_record.call_args[1]
        assert call_kwargs["bytes_in"] == 0
        assert call_kwargs["bytes_out"] == 0

    def test_end_stream_handles_missing_stream(self, mock_db):
        """Test end_stream gracefully handles non-existent stream."""
        mock_db.return_value.select.return_value.first.return_value = None

        # Should not raise
        MediaStreamModel.end_stream(mock_db, stream_key="missing")


class TestMediaStreamModelGetActiveStreams:
    """Test MediaStreamModel.get_active_streams - covers lines 181-184."""

    def test_get_active_streams_returns_list_of_dicts(self, mock_db):
        """Test get_active_streams returns list of dicts (lines 183-184)."""
        stream1 = MagicMock()
        stream1.__iter__ = MagicMock(return_value=iter([]))
        stream2 = MagicMock()
        stream2.__iter__ = MagicMock(return_value=iter([]))

        mock_db.return_value.select.return_value = [stream1, stream2]

        result = MediaStreamModel.get_active_streams(mock_db)

        assert isinstance(result, list)
        assert len(result) == 2

    def test_get_active_streams_returns_empty_list(self, mock_db):
        """Test get_active_streams returns empty list when no active streams."""
        mock_db.return_value.select.return_value = []

        result = MediaStreamModel.get_active_streams(mock_db)

        assert result == []


class TestMediaStreamModelGetStream:
    """Test MediaStreamModel.get_stream - covers lines 187-192."""

    def test_get_stream_returns_dict_when_found(self):
        """Test get_stream returns dict when stream found (lines 190-191)."""
        stream = MagicMock()
        stream.__iter__ = MagicMock(return_value=iter([]))

        mock_db = MagicMock()
        mock_db.return_value.select.return_value.first.return_value = stream

        result = MediaStreamModel.get_stream(mock_db, stream_key="found-key")

        assert result is not None

    def test_get_stream_returns_none_when_not_found(self, mock_db):
        """Test get_stream returns None when stream not found (lines 189, 192)."""
        mock_db.return_value.select.return_value.first.return_value = None

        result = MediaStreamModel.get_stream(mock_db, stream_key="missing-key")

        assert result is None
