#!/usr/bin/env python3
"""
Comprehensive tests for models/media_settings.py focused on branch coverage.

Tests uncovered branches and edge cases in MediaSettingsModel and MediaStreamModel.
"""

import pytest
from unittest.mock import MagicMock, patch, call
from datetime import datetime
from models.media_settings import (
    MediaSettingsModel,
    MediaStreamModel,
    UpdateMediaSettingsRequest,
)


# ===== MediaSettingsModel Tests =====


class TestMediaSettingsModelUpdateBranches:
    """Test update_settings branches with proper mocking."""

    def test_update_existing_with_zero_resolution_clears_it(self, mock_db):
        """Test updating existing settings with resolution=0 clears to None (lines 70-72)."""
        settings_row = MagicMock()
        settings_row.update_record = MagicMock()

        mock_db.return_value.select.return_value.first.return_value = settings_row

        with patch.object(MediaSettingsModel, "get_settings", return_value={"id": 1}):
            MediaSettingsModel.update_settings(
                mock_db, updated_by=1, admin_max_resolution=0
            )

        # Verify update_record was called
        settings_row.update_record.assert_called_once()
        call_kwargs = settings_row.update_record.call_args[1]
        assert call_kwargs["admin_max_resolution"] is None

    def test_update_existing_with_positive_resolution_keeps_it(self, mock_db):
        """Test updating existing settings preserves positive resolution."""
        settings_row = MagicMock()
        settings_row.update_record = MagicMock()

        mock_db.return_value.select.return_value.first.return_value = settings_row

        with patch.object(MediaSettingsModel, "get_settings", return_value={"id": 1}):
            MediaSettingsModel.update_settings(
                mock_db, updated_by=1, admin_max_resolution=1440
            )

        call_kwargs = settings_row.update_record.call_args[1]
        assert call_kwargs["admin_max_resolution"] == 1440

    def test_update_existing_with_zero_bitrate_clears_it(self, mock_db):
        """Test bitrate=0 clears to None (lines 74-76)."""
        settings_row = MagicMock()
        settings_row.update_record = MagicMock()

        mock_db.return_value.select.return_value.first.return_value = settings_row

        with patch.object(MediaSettingsModel, "get_settings", return_value={"id": 1}):
            MediaSettingsModel.update_settings(
                mock_db, updated_by=1, admin_max_bitrate_kbps=0
            )

        call_kwargs = settings_row.update_record.call_args[1]
        assert call_kwargs["admin_max_bitrate_kbps"] is None

    def test_update_existing_with_empty_codec_clears_it(self, mock_db):
        """Test codec='' clears to None (lines 78)."""
        settings_row = MagicMock()
        settings_row.update_record = MagicMock()

        mock_db.return_value.select.return_value.first.return_value = settings_row

        with patch.object(MediaSettingsModel, "get_settings", return_value={"id": 1}):
            MediaSettingsModel.update_settings(mock_db, updated_by=1, enforce_codec="")

        call_kwargs = settings_row.update_record.call_args[1]
        assert call_kwargs["enforce_codec"] is None

    def test_update_existing_skips_unspecified_fields(self, mock_db):
        """Test unspecified fields are not in update_record call."""
        settings_row = MagicMock()
        settings_row.update_record = MagicMock()

        mock_db.return_value.select.return_value.first.return_value = settings_row

        with patch.object(MediaSettingsModel, "get_settings", return_value={"id": 1}):
            MediaSettingsModel.update_settings(
                mock_db, updated_by=1, admin_max_resolution=1080
            )

        call_kwargs = settings_row.update_record.call_args[1]
        # These should not be in the call
        assert "admin_max_bitrate_kbps" not in call_kwargs or \
               call_kwargs.get("admin_max_bitrate_kbps") is None

    def test_create_new_with_zero_resolution_clears_it(self, mock_db):
        """Test creating new settings with resolution=0 clears to None (lines 88-92)."""
        mock_db.return_value.select.return_value.first.return_value = None
        mock_db.media_settings.insert = MagicMock(return_value=1)

        with patch.object(MediaSettingsModel, "get_settings", return_value={"id": 1}):
            MediaSettingsModel.update_settings(
                mock_db, updated_by=1, admin_max_resolution=0
            )

        call_kwargs = mock_db.media_settings.insert.call_args[1]
        assert call_kwargs["admin_max_resolution"] is None

    def test_create_new_with_positive_resolution_keeps_it(self, mock_db):
        """Test creating new settings preserves positive resolution (line 89)."""
        mock_db.return_value.select.return_value.first.return_value = None
        mock_db.media_settings.insert = MagicMock(return_value=1)

        with patch.object(MediaSettingsModel, "get_settings", return_value={"id": 1}):
            MediaSettingsModel.update_settings(
                mock_db, updated_by=1, admin_max_resolution=2160
            )

        call_kwargs = mock_db.media_settings.insert.call_args[1]
        assert call_kwargs["admin_max_resolution"] == 2160

    def test_create_new_with_zero_bitrate_clears_it(self, mock_db):
        """Test bitrate=0 clears to None in creation (lines 93-96)."""
        mock_db.return_value.select.return_value.first.return_value = None
        mock_db.media_settings.insert = MagicMock(return_value=1)

        with patch.object(MediaSettingsModel, "get_settings", return_value={"id": 1}):
            MediaSettingsModel.update_settings(
                mock_db, updated_by=1, admin_max_bitrate_kbps=0
            )

        call_kwargs = mock_db.media_settings.insert.call_args[1]
        assert call_kwargs["admin_max_bitrate_kbps"] is None

    def test_create_new_with_empty_codec_clears_it(self, mock_db):
        """Test codec='' clears to None in creation (line 98)."""
        mock_db.return_value.select.return_value.first.return_value = None
        mock_db.media_settings.insert = MagicMock(return_value=1)

        with patch.object(MediaSettingsModel, "get_settings", return_value={"id": 1}):
            MediaSettingsModel.update_settings(
                mock_db, updated_by=1, enforce_codec=""
            )

        call_kwargs = mock_db.media_settings.insert.call_args[1]
        assert call_kwargs["enforce_codec"] is None

    def test_create_new_defaults_transcode_ladder_enabled_true(self, mock_db):
        """Test creation defaults transcode_ladder_enabled to True (line 100)."""
        mock_db.return_value.select.return_value.first.return_value = None
        mock_db.media_settings.insert = MagicMock(return_value=1)

        with patch.object(MediaSettingsModel, "get_settings", return_value={"id": 1}):
            MediaSettingsModel.update_settings(mock_db, updated_by=1)

        call_kwargs = mock_db.media_settings.insert.call_args[1]
        assert call_kwargs["transcode_ladder_enabled"] is True

    def test_create_new_defaults_ladder_resolutions(self, mock_db):
        """Test creation uses default ladder when not specified (line 102)."""
        mock_db.return_value.select.return_value.first.return_value = None
        mock_db.media_settings.insert = MagicMock(return_value=1)

        with patch.object(MediaSettingsModel, "get_settings", return_value={"id": 1}):
            MediaSettingsModel.update_settings(mock_db, updated_by=1)

        call_kwargs = mock_db.media_settings.insert.call_args[1]
        assert call_kwargs["transcode_ladder_resolutions"] == [360, 540, 720, 1080]


class TestMediaSettingsModelGetSettingsBranches:
    """Test get_settings branches."""

    def test_get_settings_returns_none_when_empty(self, mock_db):
        """Test get_settings returns None when no record (lines 35-37)."""
        mock_db.return_value.select.return_value.first.return_value = None

        result = MediaSettingsModel.get_settings(mock_db)

        assert result is None

    def test_get_settings_uses_default_ladder_when_null(self, mock_db):
        """Test get_settings uses default ladder when NULL (line 45)."""
        settings_row = MagicMock()
        settings_row.id = 1
        settings_row.admin_max_resolution = 1080
        settings_row.admin_max_bitrate_kbps = 5000
        settings_row.enforce_codec = "h264"
        settings_row.transcode_ladder_enabled = True
        settings_row.transcode_ladder_resolutions = None
        settings_row.updated_by = 1
        settings_row.updated_at = datetime.utcnow()

        mock_db.return_value.select.return_value.first.return_value = settings_row

        result = MediaSettingsModel.get_settings(mock_db)

        assert result["transcode_ladder_resolutions"] == [360, 540, 720, 1080]

    def test_get_settings_preserves_custom_ladder(self, mock_db):
        """Test get_settings preserves custom ladder when set."""
        settings_row = MagicMock()
        settings_row.id = 1
        settings_row.admin_max_resolution = None
        settings_row.admin_max_bitrate_kbps = None
        settings_row.enforce_codec = None
        settings_row.transcode_ladder_enabled = True
        settings_row.transcode_ladder_resolutions = [480, 1080]
        settings_row.updated_by = 1
        settings_row.updated_at = datetime.utcnow()

        mock_db.return_value.select.return_value.first.return_value = settings_row

        result = MediaSettingsModel.get_settings(mock_db)

        assert result["transcode_ladder_resolutions"] == [480, 1080]


class TestMediaSettingsModelClearAdminOverrideBranches:
    """Test clear_admin_override branches."""

    def test_clear_admin_override_updates_when_exists(self, mock_db):
        """Test clear_admin_override updates existing record (lines 113-118)."""
        existing = MagicMock()
        existing.update_record = MagicMock()

        mock_db.return_value.select.return_value.first.return_value = existing

        with patch.object(MediaSettingsModel, "get_settings", return_value={"id": 1}):
            MediaSettingsModel.clear_admin_override(mock_db, updated_by=1)

        existing.update_record.assert_called_once()
        call_kwargs = existing.update_record.call_args[1]
        assert call_kwargs["admin_max_resolution"] is None
        assert call_kwargs["updated_by"] == 1

    def test_clear_admin_override_returns_none_when_not_exists(self, mock_db):
        """Test clear_admin_override returns None when no record (line 113)."""
        mock_db.return_value.select.return_value.first.return_value = None

        with patch.object(MediaSettingsModel, "get_settings", return_value=None):
            result = MediaSettingsModel.clear_admin_override(mock_db, updated_by=1)

        assert result is None


class TestUpdateMediaSettingsRequestValidation:
    """Test Pydantic validators for UpdateMediaSettingsRequest."""

    def test_resolve_validator_accepts_valid_resolutions(self, mock_db):
        """Test resolution validator accepts valid values (lines 207-213)."""
        for res in [360, 480, 540, 720, 1080, 1440, 2160, 4320]:
            req = UpdateMediaSettingsRequest(admin_max_resolution=res)
            assert req.admin_max_resolution == res

    def test_resolve_validator_rejects_invalid(self, mock_db):
        """Test resolution validator rejects invalid (line 212)."""
        with pytest.raises(ValueError):
            UpdateMediaSettingsRequest(admin_max_resolution=800)

    def test_codec_validator_accepts_valid(self, mock_db):
        """Test codec validator accepts valid (lines 216-221)."""
        for codec in ["h264", "h265", "av1"]:
            req = UpdateMediaSettingsRequest(enforce_codec=codec)
            assert req.enforce_codec == codec

    def test_codec_validator_rejects_invalid(self, mock_db):
        """Test codec validator rejects invalid (line 220)."""
        with pytest.raises(ValueError):
            UpdateMediaSettingsRequest(enforce_codec="vp9")

    def test_ladder_validator_accepts_valid(self, mock_db):
        """Test ladder validator accepts valid (lines 223-230)."""
        req = UpdateMediaSettingsRequest(
            transcode_ladder_resolutions=[360, 720, 1080]
        )
        assert req.transcode_ladder_resolutions == [360, 720, 1080]

    def test_ladder_validator_rejects_invalid_resolution(self, mock_db):
        """Test ladder validator rejects invalid resolution (line 229)."""
        with pytest.raises(ValueError):
            UpdateMediaSettingsRequest(transcode_ladder_resolutions=[360, 900])


# ===== MediaStreamModel Tests =====


class TestMediaStreamModelCreateStreamBranches:
    """Test create_stream method branches."""

    def test_create_stream_with_all_fields(self, mock_db):
        """Test create_stream with all optional parameters (lines 156-166)."""
        mock_db.media_streams.insert = MagicMock(return_value=42)

        stream_id = MediaStreamModel.create_stream(
            mock_db,
            stream_key="stream1",
            protocol="rtmp",
            client_ip="192.168.1.1",
            codec="h264",
            resolution="1920x1080",
            bitrate_kbps=5000,
            user_agent="Chrome/120",
        )

        assert stream_id == 42
        call_kwargs = mock_db.media_streams.insert.call_args[1]
        assert call_kwargs["stream_key"] == "stream1"
        assert call_kwargs["protocol"] == "rtmp"
        assert call_kwargs["status"] == "active"
        assert call_kwargs["codec"] == "h264"
        assert call_kwargs["resolution"] == "1920x1080"
        assert call_kwargs["bitrate_kbps"] == 5000

    def test_create_stream_with_minimal_fields(self, mock_db):
        """Test create_stream with only required fields."""
        mock_db.media_streams.insert = MagicMock(return_value=1)

        stream_id = MediaStreamModel.create_stream(
            mock_db,
            stream_key="stream2",
            protocol="srt",
            client_ip="10.0.0.1",
        )

        assert stream_id == 1
        call_kwargs = mock_db.media_streams.insert.call_args[1]
        assert call_kwargs["codec"] is None
        assert call_kwargs["resolution"] is None
        assert call_kwargs["bitrate_kbps"] is None


class TestMediaStreamModelEndStreamBranches:
    """Test end_stream method branches."""

    def test_end_stream_updates_when_found(self, mock_db):
        """Test end_stream updates record when found (lines 172-178)."""
        stream = MagicMock()
        stream.update_record = MagicMock()

        mock_db.return_value.select.return_value.first.return_value = stream

        MediaStreamModel.end_stream(
            mock_db, stream_key="stream1", bytes_in=1000, bytes_out=5000
        )

        stream.update_record.assert_called_once()
        call_kwargs = stream.update_record.call_args[1]
        assert call_kwargs["status"] == "idle"
        assert call_kwargs["bytes_in"] == 1000
        assert call_kwargs["bytes_out"] == 5000
        assert "ended_at" in call_kwargs

    def test_end_stream_skips_when_not_found(self, mock_db):
        """Test end_stream skips update when stream not found (line 172)."""
        mock_db.return_value.select.return_value.first.return_value = None

        # Should not raise
        MediaStreamModel.end_stream(mock_db, stream_key="missing")

    def test_end_stream_uses_default_bytes(self, mock_db):
        """Test end_stream uses default bytes when not specified."""
        stream = MagicMock()
        stream.update_record = MagicMock()

        mock_db.return_value.select.return_value.first.return_value = stream

        MediaStreamModel.end_stream(mock_db, stream_key="stream1")

        call_kwargs = stream.update_record.call_args[1]
        assert call_kwargs["bytes_in"] == 0
        assert call_kwargs["bytes_out"] == 0


class TestMediaStreamModelGetActiveStreamsBranches:
    """Test get_active_streams method branches."""

    def test_get_active_streams_returns_list(self, mock_db):
        """Test get_active_streams returns list of dicts (lines 182-184)."""
        stream1 = MagicMock()
        stream1.__iter__ = MagicMock(return_value=iter([]))
        stream2 = MagicMock()
        stream2.__iter__ = MagicMock(return_value=iter([]))

        mock_db.return_value.select.return_value = [stream1, stream2]

        result = MediaStreamModel.get_active_streams(mock_db)

        assert isinstance(result, list)
        assert len(result) == 2

    def test_get_active_streams_returns_empty_when_none(self, mock_db):
        """Test get_active_streams returns empty list when no streams."""
        mock_db.return_value.select.return_value = []

        result = MediaStreamModel.get_active_streams(mock_db)

        assert result == []


class TestMediaStreamModelGetStreamBranches:
    """Test get_stream method branches."""

    def test_get_stream_returns_dict_when_found(self, mock_db):
        """Test get_stream returns dict when found (lines 189-191)."""
        stream = MagicMock()
        stream.__iter__ = MagicMock(return_value=iter([]))

        mock_db.return_value.select.return_value.first.return_value = stream

        result = MediaStreamModel.get_stream(mock_db, stream_key="found")

        assert result is not None

    def test_get_stream_returns_none_when_not_found(self, mock_db):
        """Test get_stream returns None when not found (lines 189, 192)."""
        mock_db.return_value.select.return_value.first.return_value = None

        result = MediaStreamModel.get_stream(mock_db, stream_key="missing")

        assert result is None
