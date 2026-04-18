#!/usr/bin/env python3
"""
Integration tests for models/media_settings.py using real SQLite database.

Tests actual database interactions to improve branch coverage.
"""

import pytest
from datetime import datetime
from pydal import DAL, Field
from models.media_settings import MediaSettingsModel, MediaStreamModel


@pytest.fixture
def test_db():
    """Create an in-memory SQLite database for testing."""
    db = DAL("sqlite:memory:")

    # Define users table first (referenced by media_settings)
    db.define_table(
        "users",
        Field("id", type="integer"),
    )

    # Create a test user
    test_user_id = db.users.insert(id=1)

    MediaSettingsModel.define_table(db)
    MediaStreamModel.define_table(db)
    db.test_user_id = test_user_id
    return db


class TestMediaSettingsModelWithRealDB:
    """Integration tests for MediaSettingsModel with real database."""

    def test_get_settings_returns_none_when_empty(self, test_db):
        """Test get_settings returns None when no records exist (lines 35-37)."""
        result = MediaSettingsModel.get_settings(test_db)
        assert result is None

    def test_get_settings_returns_dict_when_exists(self, test_db):
        """Test get_settings returns dict when record exists (lines 35-48)."""
        # Create a setting
        test_db.media_settings.insert(
            admin_max_resolution=1080,
            admin_max_bitrate_kbps=5000,
            enforce_codec="h264",
            transcode_ladder_enabled=True,
            transcode_ladder_resolutions=[360, 540, 720, 1080],
            updated_by=test_db.test_user_id,
        )
        test_db.commit()

        result = MediaSettingsModel.get_settings(test_db)

        assert result is not None
        assert result["admin_max_resolution"] == 1080
        assert result["admin_max_bitrate_kbps"] == 5000
        assert result["enforce_codec"] == "h264"

    def test_get_settings_uses_default_ladder_when_null(self, test_db):
        """Test get_settings applies default ladder when NULL (line 45)."""
        test_db.media_settings.insert(
            admin_max_resolution=1080,
            admin_max_bitrate_kbps=None,
            enforce_codec=None,
            transcode_ladder_enabled=True,
            transcode_ladder_resolutions=None,  # NULL
            updated_by=test_db.test_user_id,
        )
        test_db.commit()

        result = MediaSettingsModel.get_settings(test_db)

        assert result["transcode_ladder_resolutions"] == [360, 540, 720, 1080]

    def test_update_settings_creates_new_record(self, test_db):
        """Test update_settings creates new record when none exists (lines 85-107)."""
        result = MediaSettingsModel.update_settings(
            test_db,
            updated_by=test_db.test_user_id,
            admin_max_resolution=1440,
            admin_max_bitrate_kbps=8000,
            enforce_codec="h265",
        )

        assert result is not None
        assert result["admin_max_resolution"] == 1440
        assert result["enforce_codec"] == "h265"

    def test_update_settings_with_zero_resolution_clears_to_none(self, test_db):
        """Test update creates record with resolution=0 cleared to None (lines 88-92)."""
        result = MediaSettingsModel.update_settings(
            test_db,
            updated_by=test_db.test_user_id,
            admin_max_resolution=0,  # Should be cleared
        )

        assert result["admin_max_resolution"] is None

    def test_update_settings_updates_existing_record(self, test_db):
        """Test update_settings updates existing record (lines 63-84)."""
        # Create initial setting
        test_db.media_settings.insert(
            admin_max_resolution=720,
            admin_max_bitrate_kbps=3000,
            enforce_codec="h264",
            transcode_ladder_enabled=True,
            transcode_ladder_resolutions=[360, 720],
            updated_by=test_db.test_user_id,
        )
        test_db.commit()

        # Update it
        result = MediaSettingsModel.update_settings(
            test_db,
            updated_by=test_db.test_user_id,
            admin_max_resolution=1080,
            admin_max_bitrate_kbps=5000,
        )

        assert result["admin_max_resolution"] == 1080
        assert result["admin_max_bitrate_kbps"] == 5000
        assert result["updated_by"] == test_db.test_user_id

    def test_update_settings_with_zero_bitrate_clears_to_none(self, test_db):
        """Test update sets bitrate=0 to None (lines 93-96)."""
        # Create initial
        test_db.media_settings.insert(
            admin_max_bitrate_kbps=5000,
            updated_by=test_db.test_user_id,
        )
        test_db.commit()

        # Update with 0
        result = MediaSettingsModel.update_settings(
            test_db,
            updated_by=test_db.test_user_id,
            admin_max_bitrate_kbps=0,
        )

        assert result["admin_max_bitrate_kbps"] is None

    def test_update_settings_with_empty_codec_clears_to_none(self, test_db):
        """Test update sets codec='' to None (line 98)."""
        test_db.media_settings.insert(
            enforce_codec="h264",
            updated_by=test_db.test_user_id,
        )
        test_db.commit()

        result = MediaSettingsModel.update_settings(
            test_db,
            updated_by=test_db.test_user_id,
            enforce_codec="",
        )

        assert result["enforce_codec"] is None

    def test_update_settings_defaults_transcode_ladder_enabled(self, test_db):
        """Test creation defaults transcode_ladder_enabled to True (line 100)."""
        result = MediaSettingsModel.update_settings(
            test_db,
            updated_by=test_db.test_user_id,
        )

        assert result["transcode_ladder_enabled"] is True

    def test_update_settings_defaults_ladder_resolutions(self, test_db):
        """Test creation uses default ladder (line 102)."""
        result = MediaSettingsModel.update_settings(
            test_db,
            updated_by=test_db.test_user_id,
        )

        assert result["transcode_ladder_resolutions"] == [360, 540, 720, 1080]

    def test_clear_admin_override_clears_resolution(self, test_db):
        """Test clear_admin_override sets resolution to None (lines 113-119)."""
        # Create setting
        test_db.media_settings.insert(
            admin_max_resolution=1440,
            updated_by=test_db.test_user_id,
        )
        test_db.commit()

        result = MediaSettingsModel.clear_admin_override(test_db, updated_by=test_db.test_user_id)

        assert result["admin_max_resolution"] is None
        assert result["updated_by"] == test_db.test_user_id

    def test_clear_admin_override_handles_empty_db(self, test_db):
        """Test clear_admin_override returns None when no record (line 113)."""
        result = MediaSettingsModel.clear_admin_override(test_db, updated_by=1)

        assert result is None


class TestMediaStreamModelWithRealDB:
    """Integration tests for MediaStreamModel with real database."""

    def test_create_stream_inserts_record(self, test_db):
        """Test create_stream inserts a record (lines 156-166)."""
        stream_id = MediaStreamModel.create_stream(
            test_db,
            stream_key="test-stream",
            protocol="rtmp",
            client_ip="192.168.1.1",
            codec="h264",
            resolution="1920x1080",
            bitrate_kbps=5000,
            user_agent="Chrome/120",
        )

        assert stream_id is not None

        # Verify record was inserted
        stream = test_db(test_db.media_streams.stream_key == "test-stream").select().first()
        assert stream is not None
        assert stream.status == "active"
        assert stream.codec == "h264"

    def test_create_stream_with_minimal_params(self, test_db):
        """Test create_stream with minimal parameters."""
        stream_id = MediaStreamModel.create_stream(
            test_db,
            stream_key="minimal-stream",
            protocol="srt",
            client_ip="10.0.0.1",
        )

        assert stream_id is not None

        stream = test_db(test_db.media_streams.stream_key == "minimal-stream").select().first()
        assert stream.codec is None
        assert stream.resolution is None

    def test_end_stream_updates_status(self, test_db):
        """Test end_stream updates status to idle (lines 172-178)."""
        # Create stream
        stream_id = MediaStreamModel.create_stream(
            test_db,
            stream_key="end-test",
            protocol="rtmp",
            client_ip="192.168.1.1",
        )

        # End it
        MediaStreamModel.end_stream(
            test_db,
            stream_key="end-test",
            bytes_in=1000,
            bytes_out=5000,
        )

        # Verify
        stream = test_db(test_db.media_streams.stream_key == "end-test").select().first()
        assert stream.status == "idle"
        assert stream.bytes_in == 1000
        assert stream.bytes_out == 5000
        assert stream.ended_at is not None

    def test_end_stream_with_missing_stream(self, test_db):
        """Test end_stream gracefully handles missing stream (line 172)."""
        # Should not raise
        MediaStreamModel.end_stream(test_db, stream_key="nonexistent")

    def test_get_active_streams_returns_list(self, test_db):
        """Test get_active_streams returns active streams (lines 182-184)."""
        # Create active streams
        MediaStreamModel.create_stream(
            test_db, stream_key="stream1", protocol="rtmp", client_ip="192.168.1.1"
        )
        MediaStreamModel.create_stream(
            test_db, stream_key="stream2", protocol="srt", client_ip="10.0.0.1"
        )

        result = MediaStreamModel.get_active_streams(test_db)

        assert isinstance(result, list)
        assert len(result) == 2

    def test_get_active_streams_empty(self, test_db):
        """Test get_active_streams returns empty list when none active."""
        result = MediaStreamModel.get_active_streams(test_db)

        assert result == []

    def test_get_stream_returns_dict_when_found(self, test_db):
        """Test get_stream returns dict when found (lines 189-191)."""
        MediaStreamModel.create_stream(
            test_db,
            stream_key="found-stream",
            protocol="rtmp",
            client_ip="192.168.1.1",
        )

        result = MediaStreamModel.get_stream(test_db, stream_key="found-stream")

        assert result is not None
        assert result["stream_key"] == "found-stream"

    def test_get_stream_returns_none_when_not_found(self, test_db):
        """Test get_stream returns None when not found (lines 189, 192)."""
        result = MediaStreamModel.get_stream(test_db, stream_key="missing")

        assert result is None
