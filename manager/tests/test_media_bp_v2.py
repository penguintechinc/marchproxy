"""
Comprehensive tests for /api/v1/modules/rtmp media blueprint.

Tests all endpoints with mocked PyDAL database and auth middleware.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from datetime import datetime
from unittest.mock import AsyncMock, MagicMock, patch

import pytest
import pytest_asyncio
from pydantic import ValidationError


# GET /api/v1/modules/rtmp/config - Success
@pytest.mark.asyncio
async def test_get_media_config_success(test_client, admin_headers):
    """Test GET /config returns current settings"""
    with patch("models.media_settings.MediaSettingsModel.get_settings") as mock_settings:
        mock_settings.return_value = {
            "admin_max_resolution": None,
            "admin_max_bitrate_kbps": None,
            "enforce_codec": None,
            "transcode_ladder_enabled": True,
            "transcode_ladder_resolutions": [360, 540, 720, 1080],
            "updated_at": None,
        }

        response = await test_client.get("/api/v1/modules/rtmp/config", headers=admin_headers)

        assert response.status_code == 200
        data = await response.get_json()
        assert data["status"] == "ok"
        assert data["config"]["transcode_ladder_enabled"] is True
        assert data["config"]["transcode_ladder_resolutions"] == [360, 540, 720, 1080]


# GET /api/v1/modules/rtmp/config - Default fallback
@pytest.mark.asyncio
async def test_get_media_config_defaults(test_client, admin_headers):
    """Test GET /config returns defaults when no settings exist"""
    with patch("models.media_settings.MediaSettingsModel.get_settings") as mock_settings:
        mock_settings.return_value = None

        response = await test_client.get("/api/v1/modules/rtmp/config", headers=admin_headers)

        assert response.status_code == 200
        data = await response.get_json()
        assert data["config"]["admin_max_resolution"] is None
        assert data["config"]["transcode_ladder_enabled"] is True


# PUT /api/v1/modules/rtmp/config - Admin success
@pytest.mark.asyncio
async def test_put_media_config_admin_success(test_client, admin_headers):
    """Test PUT /config updates settings for admin user"""
    with patch("models.media_settings.MediaSettingsModel.update_settings") as mock_update:
        mock_update.return_value = {
            "transcode_ladder_enabled": True,
            "transcode_ladder_resolutions": [720, 1080],
            "updated_at": datetime.utcnow(),
        }

        payload = {
            "transcode_ladder_enabled": True,
            "transcode_ladder_resolutions": [720, 1080],
        }

        response = await test_client.put(
            "/api/v1/modules/rtmp/config",
            json=payload,
            headers=admin_headers,
        )

        assert response.status_code == 200
        data = await response.get_json()
        assert data["status"] == "updated"
        assert data["config"]["transcode_ladder_resolutions"] == [720, 1080]


# PUT /api/v1/modules/rtmp/config - Non-admin 403
@pytest.mark.asyncio
async def test_put_media_config_non_admin_forbidden(test_client, user_headers):
    """Test PUT /config returns 403 for non-admin user"""
    # Patch the autouse fixture to return non-admin payload
    with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_validate:
        mock_validate.return_value = {
            "user_id": 2,
            "sub": "2",
            "is_admin": False,
            "scope": "",
            "roles": [],
            "tenant": "test",
        }

        payload = {"transcode_ladder_enabled": False}

        response = await test_client.put(
            "/api/v1/modules/rtmp/config",
            json=payload,
            headers=user_headers,
        )

        assert response.status_code == 403
        data = await response.get_json()
        assert "Admin access required" in data["error"]


# PUT /api/v1/modules/rtmp/config - Invalid resolution
@pytest.mark.asyncio
async def test_put_media_config_invalid_resolution(test_client, admin_headers):
    """Test PUT /config rejects invalid resolutions"""
    payload = {
        "transcode_ladder_enabled": True,
        "transcode_ladder_resolutions": [360, 720, 9999],
    }

    response = await test_client.put(
        "/api/v1/modules/rtmp/config",
        json=payload,
        headers=admin_headers,
    )

    assert response.status_code == 400
    data = await response.get_json()
    assert "Invalid resolution" in data["error"]


# PUT /api/v1/modules/rtmp/config - Invalid JSON
@pytest.mark.asyncio
async def test_put_media_config_invalid_json(test_client, admin_headers):
    """Test PUT /config handles malformed JSON gracefully"""
    response = await test_client.put(
        "/api/v1/modules/rtmp/config",
        data="{bad json}",
        headers={**admin_headers, "Content-Type": "application/json"},
    )

    assert response.status_code == 400
    data = await response.get_json()
    assert "Invalid JSON" in data["error"]


# GET /api/v1/modules/rtmp/streams - List active streams
@pytest.mark.asyncio
async def test_list_streams_success(test_client, admin_headers):
    """Test GET /streams returns list of active streams"""
    with patch("models.media_settings.MediaStreamModel.get_active_streams") as mock_streams:
        mock_streams.return_value = [
            {
                "id": 1,
                "stream_key": "test-key-1",
                "protocol": "rtmp",
                "codec": "h264",
                "resolution": "1080p",
                "bitrate_kbps": 4000,
                "status": "active",
                "client_ip": "192.168.1.1",
                "started_at": datetime(2025, 1, 1, 12, 0, 0),
                "bytes_in": 1000,
                "bytes_out": 2000,
            },
            {
                "id": 2,
                "stream_key": "test-key-2",
                "protocol": "rtmp",
                "codec": "vp9",
                "resolution": "4k",
                "bitrate_kbps": 8000,
                "status": "active",
                "client_ip": "192.168.1.2",
                "started_at": datetime(2025, 1, 1, 13, 0, 0),
                "bytes_in": 2000,
                "bytes_out": 4000,
            },
        ]

        response = await test_client.get("/api/v1/modules/rtmp/streams", headers=admin_headers)

        assert response.status_code == 200
        data = await response.get_json()
        assert data["count"] == 2
        assert len(data["streams"]) == 2
        assert data["streams"][0]["stream_key"] == "test-key-1"
        assert data["streams"][1]["codec"] == "vp9"


# GET /api/v1/modules/rtmp/streams - Empty list
@pytest.mark.asyncio
async def test_list_streams_empty(test_client, admin_headers):
    """Test GET /streams returns empty list when no streams active"""
    with patch("models.media_settings.MediaStreamModel.get_active_streams") as mock_streams:
        mock_streams.return_value = []

        response = await test_client.get("/api/v1/modules/rtmp/streams", headers=admin_headers)

        assert response.status_code == 200
        data = await response.get_json()
        assert data["count"] == 0
        assert data["streams"] == []


# GET /api/v1/modules/rtmp/streams - Error handling
@pytest.mark.asyncio
async def test_list_streams_error(test_client, admin_headers):
    """Test GET /streams handles database errors gracefully"""
    with patch("models.media_settings.MediaStreamModel.get_active_streams") as mock_streams:
        mock_streams.side_effect = RuntimeError("Database error")

        response = await test_client.get("/api/v1/modules/rtmp/streams", headers=admin_headers)

        assert response.status_code == 500
        data = await response.get_json()
        assert "Failed to list streams" in data["error"]


# GET /api/v1/modules/rtmp/streams/<key> - Get stream details
@pytest.mark.asyncio
async def test_get_stream_detail_success(test_client, admin_headers):
    """Test GET /streams/<key> returns stream details"""
    with patch("models.media_settings.MediaStreamModel.get_stream") as mock_stream:
        mock_stream.return_value = {
            "id": 1,
            "stream_key": "test-key",
            "protocol": "rtmp",
            "codec": "h264",
            "resolution": "1080p",
            "bitrate_kbps": 4000,
            "status": "active",
            "client_ip": "192.168.1.1",
            "started_at": datetime(2025, 1, 1, 12, 0, 0),
            "ended_at": None,
            "bytes_in": 1000,
            "bytes_out": 2000,
        }

        response = await test_client.get(
            "/api/v1/modules/rtmp/streams/test-key",
            headers=admin_headers,
        )

        assert response.status_code == 200
        data = await response.get_json()
        assert data["stream"]["stream_key"] == "test-key"
        assert data["stream"]["bitrate_kbps"] == 4000
        assert data["stream"]["ended_at"] is None


# GET /api/v1/modules/rtmp/streams/<key> - Stream not found
@pytest.mark.asyncio
async def test_get_stream_detail_not_found(test_client, admin_headers):
    """Test GET /streams/<key> returns 404 when stream not found"""
    with patch("models.media_settings.MediaStreamModel.get_stream") as mock_stream:
        mock_stream.return_value = None

        response = await test_client.get(
            "/api/v1/modules/rtmp/streams/nonexistent-key",
            headers=admin_headers,
        )

        assert response.status_code == 404
        data = await response.get_json()
        assert "Stream not found" in data["error"]


# DELETE /api/v1/modules/rtmp/streams/<key> - Stop stream (admin only)
@pytest.mark.asyncio
async def test_delete_stream_admin_success(test_client, admin_headers):
    """Test DELETE /streams/<key> stops stream for admin"""
    with patch("models.media_settings.MediaStreamModel.get_stream") as mock_get_stream, patch(
        "models.media_settings.MediaStreamModel.end_stream"
    ) as mock_end_stream:
        mock_get_stream.return_value = {"id": 1, "stream_key": "test-key"}
        mock_end_stream.return_value = None

        response = await test_client.delete(
            "/api/v1/modules/rtmp/streams/test-key",
            headers=admin_headers,
        )

        assert response.status_code == 200
        data = await response.get_json()
        assert data["status"] == "stopped"
        assert data["stream_key"] == "test-key"
        mock_end_stream.assert_called_once()


# DELETE /api/v1/modules/rtmp/streams/<key> - Non-admin forbidden
@pytest.mark.asyncio
async def test_delete_stream_non_admin_forbidden(test_client, user_headers):
    """Test DELETE /streams/<key> returns 403 for non-admin"""
    with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_validate:
        mock_validate.return_value = {
            "user_id": 2,
            "sub": "2",
            "is_admin": False,
            "scope": "",
            "roles": [],
            "tenant": "test",
        }

        response = await test_client.delete(
            "/api/v1/modules/rtmp/streams/test-key",
            headers=user_headers,
        )

        assert response.status_code == 403
        data = await response.get_json()
        assert "Admin access required" in data["error"]


# DELETE /api/v1/modules/rtmp/streams/<key> - Stream not found
@pytest.mark.asyncio
async def test_delete_stream_not_found(test_client, admin_headers):
    """Test DELETE /streams/<key> returns 404 when stream not found"""
    with patch("models.media_settings.MediaStreamModel.get_stream") as mock_stream:
        mock_stream.return_value = None

        response = await test_client.delete(
            "/api/v1/modules/rtmp/streams/nonexistent-key",
            headers=admin_headers,
        )

        assert response.status_code == 404
        data = await response.get_json()
        assert "Stream not found" in data["error"]


# GET /api/v1/modules/rtmp/capabilities - Get hardware capabilities
@pytest.mark.asyncio
async def test_get_capabilities_success(test_client, admin_headers):
    """Test GET /capabilities returns hardware info and settings"""
    with patch("models.media_settings.MediaSettingsModel.get_settings") as mock_settings:
        mock_settings.return_value = {
            "admin_max_resolution": 1080,
            "enforce_codec": "h264",
            "transcode_ladder_enabled": True,
            "transcode_ladder_resolutions": [360, 720, 1080],
        }

        response = await test_client.get(
            "/api/v1/modules/rtmp/capabilities",
            headers=admin_headers,
        )

        assert response.status_code == 200
        data = await response.get_json()
        assert "hardware" in data
        assert "settings" in data
        assert "effective_max_resolution" in data
        assert data["hardware"]["gpu_type"] == "nvidia"
        assert data["settings"]["transcode_ladder_enabled"] is True


# GET /api/v1/modules/rtmp/capabilities - No settings
@pytest.mark.asyncio
async def test_get_capabilities_no_settings(test_client, admin_headers):
    """Test GET /capabilities uses defaults when no settings exist"""
    with patch("models.media_settings.MediaSettingsModel.get_settings") as mock_settings:
        mock_settings.return_value = None

        response = await test_client.get(
            "/api/v1/modules/rtmp/capabilities",
            headers=admin_headers,
        )

        assert response.status_code == 200
        data = await response.get_json()
        assert data["settings"]["transcode_ladder_enabled"] is True
        assert data["settings"]["transcode_ladder_resolutions"] == [360, 540, 720, 1080]


# GET /api/v1/modules/rtmp/streams/<key>/restream - Get destinations
@pytest.mark.asyncio
async def test_get_restream_destinations(test_client, admin_headers):
    """Test GET /streams/<key>/restream returns restream destinations"""
    with patch("models.media_settings.MediaStreamModel.get_stream") as mock_stream:
        mock_stream.return_value = {"id": 1, "stream_key": "test-key"}

        response = await test_client.get(
            "/api/v1/modules/rtmp/streams/test-key/restream",
            headers=admin_headers,
        )

        assert response.status_code == 200
        data = await response.get_json()
        assert data["stream_key"] == "test-key"
        assert data["destinations"] == []


# GET /api/v1/modules/rtmp/streams/<key>/restream - Stream not found
@pytest.mark.asyncio
async def test_get_restream_stream_not_found(test_client, admin_headers):
    """Test GET /streams/<key>/restream returns 404 when stream not found"""
    with patch("models.media_settings.MediaStreamModel.get_stream") as mock_stream:
        mock_stream.return_value = None

        response = await test_client.get(
            "/api/v1/modules/rtmp/streams/nonexistent/restream",
            headers=admin_headers,
        )

        assert response.status_code == 404
        data = await response.get_json()
        assert "Stream not found" in data["error"]


# POST /api/v1/modules/rtmp/streams/<key>/restream - Create restream
@pytest.mark.asyncio
async def test_post_restream_success(test_client, admin_headers):
    """Test POST /streams/<key>/restream creates restream destination"""
    with patch("models.media_settings.MediaStreamModel.get_stream") as mock_stream:
        mock_stream.return_value = {"id": 1, "stream_key": "test-key"}

        payload = {
            "platform": "youtube",
            "rtmp_url": "rtmp://a.rtmp.youtube.com/live2",
            "stream_key": "test-stream-key",
            "quality": "1080p",
            "enabled": True,
        }

        response = await test_client.post(
            "/api/v1/modules/rtmp/streams/test-key/restream",
            json=payload,
            headers=admin_headers,
        )

        assert response.status_code == 201
        data = await response.get_json()
        assert data["status"] == "created"
        assert data["destination"]["platform"] == "youtube"
        assert data["destination"]["quality"] == "1080p"


# POST /api/v1/modules/rtmp/streams/<key>/restream - Non-admin forbidden
@pytest.mark.asyncio
async def test_post_restream_non_admin_forbidden(test_client, user_headers):
    """Test POST /restream returns 403 for non-admin"""
    with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_validate:
        mock_validate.return_value = {
            "user_id": 2,
            "sub": "2",
            "is_admin": False,
            "scope": "",
            "roles": [],
            "tenant": "test",
        }
        with patch("models.media_settings.MediaStreamModel.get_stream") as mock_stream:
            mock_stream.return_value = {"id": 1, "stream_key": "test-key"}

            payload = {
                "platform": "youtube",
                "rtmp_url": "rtmp://a.rtmp.youtube.com/live2",
                "stream_key": "test-stream-key",
                "quality": "1080p",
                "enabled": True,
            }

            response = await test_client.post(
                "/api/v1/modules/rtmp/streams/test-key/restream",
                json=payload,
                headers=user_headers,
            )

            assert response.status_code == 403


# POST /api/v1/modules/rtmp/streams/<key>/restream - Validation error
@pytest.mark.asyncio
async def test_post_restream_validation_error(test_client, admin_headers):
    """Test POST /restream validates request data"""
    with patch("models.media_settings.MediaStreamModel.get_stream") as mock_stream:
        mock_stream.return_value = {"id": 1, "stream_key": "test-key"}

        # Missing required field 'platform'
        payload = {
            "rtmp_url": "rtmp://a.rtmp.youtube.com/live2",
            "stream_key": "test-stream-key",
            "quality": "1080p",
            "enabled": True,
        }

        response = await test_client.post(
            "/api/v1/modules/rtmp/streams/test-key/restream",
            json=payload,
            headers=admin_headers,
        )

        assert response.status_code == 400
        data = await response.get_json()
        assert "Validation error" in data["error"]


# DELETE /api/v1/modules/rtmp/streams/<key>/restream - Delete restream
@pytest.mark.asyncio
async def test_delete_restream_success(test_client, admin_headers):
    """Test DELETE /streams/<key>/restream deletes restream destination"""
    with patch("models.media_settings.MediaStreamModel.get_stream") as mock_stream:
        mock_stream.return_value = {"id": 1, "stream_key": "test-key"}

        response = await test_client.delete(
            "/api/v1/modules/rtmp/streams/test-key/restream",
            headers=admin_headers,
        )

        assert response.status_code == 200
        data = await response.get_json()
        assert data["status"] == "deleted"
        assert data["stream_key"] == "test-key"


# DELETE /api/v1/modules/rtmp/streams/<key>/restream - Non-admin forbidden
@pytest.mark.asyncio
async def test_delete_restream_non_admin_forbidden(test_client, user_headers):
    """Test DELETE /restream returns 403 for non-admin"""
    with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_validate:
        mock_validate.return_value = {
            "user_id": 2,
            "sub": "2",
            "is_admin": False,
            "scope": "",
            "roles": [],
            "tenant": "test",
        }
        with patch("models.media_settings.MediaStreamModel.get_stream") as mock_stream:
            mock_stream.return_value = {"id": 1, "stream_key": "test-key"}

            response = await test_client.delete(
                "/api/v1/modules/rtmp/streams/test-key/restream",
                headers=user_headers,
            )

            assert response.status_code == 403


# GET /api/v1/modules/rtmp/stats - Get statistics
@pytest.mark.asyncio
async def test_get_stats_success(test_client, admin_headers):
    """Test GET /stats returns media statistics"""
    with patch("models.media_settings.MediaStreamModel.get_active_streams") as mock_streams:
        mock_streams.return_value = [
            {
                "id": 1,
                "stream_key": "test-key-1",
                "protocol": "rtmp",
                "bytes_in": 1000,
                "bytes_out": 2000,
            },
            {
                "id": 2,
                "stream_key": "test-key-2",
                "protocol": "rtmp",
                "bytes_in": 500,
                "bytes_out": 1000,
            },
        ]

        response = await test_client.get("/api/v1/modules/rtmp/stats", headers=admin_headers)

        assert response.status_code == 200
        data = await response.get_json()
        assert data["stats"]["active_streams"] == 2
        assert data["stats"]["total_bytes_in"] == 1500
        assert data["stats"]["total_bytes_out"] == 3000
        assert "timestamp" in data["stats"]


# GET /api/v1/modules/rtmp/stats - No streams
@pytest.mark.asyncio
async def test_get_stats_no_streams(test_client, admin_headers):
    """Test GET /stats returns zeros when no streams active"""
    with patch("models.media_settings.MediaStreamModel.get_active_streams") as mock_streams:
        mock_streams.return_value = []

        response = await test_client.get("/api/v1/modules/rtmp/stats", headers=admin_headers)

        assert response.status_code == 200
        data = await response.get_json()
        assert data["stats"]["active_streams"] == 0
        assert data["stats"]["total_bytes_in"] == 0
        assert data["stats"]["total_bytes_out"] == 0


# GET /api/v1/modules/rtmp/stats - Error handling
@pytest.mark.asyncio
async def test_get_stats_error(test_client, admin_headers):
    """Test GET /stats handles database errors gracefully"""
    with patch("models.media_settings.MediaStreamModel.get_active_streams") as mock_streams:
        mock_streams.side_effect = RuntimeError("Database error")

        response = await test_client.get("/api/v1/modules/rtmp/stats", headers=admin_headers)

        assert response.status_code == 500
        data = await response.get_json()
        assert "Failed to get stats" in data["error"]


# Additional test: Stream with ended_at timestamp
@pytest.mark.asyncio
async def test_get_stream_with_ended_at(test_client, admin_headers):
    """Test GET /streams/<key> shows ended_at timestamp if stream is ended"""
    with patch("models.media_settings.MediaStreamModel.get_stream") as mock_stream:
        mock_stream.return_value = {
            "id": 1,
            "stream_key": "test-key",
            "protocol": "rtmp",
            "codec": "h264",
            "resolution": "1080p",
            "bitrate_kbps": 4000,
            "status": "ended",
            "client_ip": "192.168.1.1",
            "started_at": datetime(2025, 1, 1, 12, 0, 0),
            "ended_at": datetime(2025, 1, 1, 12, 30, 0),
            "bytes_in": 1000,
            "bytes_out": 2000,
        }

        response = await test_client.get(
            "/api/v1/modules/rtmp/streams/test-key",
            headers=admin_headers,
        )

        assert response.status_code == 200
        data = await response.get_json()
        assert data["stream"]["ended_at"] is not None
        assert "2025-01-01T12:30:00" in data["stream"]["ended_at"]
