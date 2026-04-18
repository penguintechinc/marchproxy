"""
Tests for license_bp.py and admin_media_bp.py coverage.

Covers httpx calls (license validation, keepalive) and admin media endpoints
with non-admin checks and exception handling.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import pytest
from datetime import datetime, timedelta
from unittest.mock import AsyncMock, MagicMock, patch

from models.license import LicenseCacheModel


# ============================================================================
# LICENSE_BP TESTS - httpx mocking patterns
# ============================================================================


@pytest.mark.asyncio
async def test_validate_license_200_response(test_client, admin_headers, mock_db):
    """Covers lines 75-104: successful license validation from server."""
    # Setup mock response
    mock_response = MagicMock()
    mock_response.status_code = 200
    mock_response.json.return_value = {
        "valid": True,
        "tier": "enterprise",
        "max_proxies": 10,
        "features": {"sso": True},
        "expires_at": "2027-01-01T00:00:00",
    }

    # Mock AsyncClient context manager
    mock_client = AsyncMock()
    mock_client.post.return_value = mock_response
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=None)

    # Patch httpx.AsyncClient, cache retrieval, and cache_validation
    with patch("api.license_bp.httpx.AsyncClient", return_value=mock_client), \
         patch.object(LicenseCacheModel, "get_cached_validation", return_value=None), \
         patch.object(LicenseCacheModel, "cache_validation") as mock_cache:
        response = await test_client.post(
            "/api/license/validate",
            json={"license_key": "PENG-test-key-123"},
            headers=admin_headers,
        )

    assert response.status_code == 200
    assert mock_client.post.called


@pytest.mark.asyncio
async def test_validate_license_non_200_response(test_client, admin_headers):
    """Covers lines 105-120: failed license validation cached."""
    # Setup mock response with non-200 status
    mock_response = MagicMock()
    mock_response.status_code = 400
    mock_response.json.return_value = {"error": "Invalid license key"}

    mock_client = AsyncMock()
    mock_client.post.return_value = mock_response
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=None)

    with patch("api.license_bp.httpx.AsyncClient", return_value=mock_client), \
         patch.object(LicenseCacheModel, "get_cached_validation", return_value=None), \
         patch.object(LicenseCacheModel, "cache_validation") as mock_cache:
        response = await test_client.post(
            "/api/license/validate",
            json={"license_key": "BAD-key"},
            headers=admin_headers,
        )

    assert response.status_code == 400
    assert mock_cache.called


@pytest.mark.asyncio
async def test_validate_license_request_error(test_client, admin_headers):
    """Covers lines 122-124: httpx.RequestError handling."""
    import httpx
    # Mock httpx.RequestError
    mock_client = AsyncMock()
    mock_client.post.side_effect = httpx.RequestError(
        "Connection refused"
    )
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=None)

    with patch("api.license_bp.httpx.AsyncClient", return_value=mock_client), \
         patch.object(LicenseCacheModel, "get_cached_validation", return_value=None):
        response = await test_client.post(
            "/api/license/validate",
            json={"license_key": "test-key"},
            headers=admin_headers,
        )

    assert response.status_code == 503
    data = await response.get_json()
    assert "License server unavailable" in data.get("error", "")


@pytest.mark.asyncio
async def test_validate_license_generic_exception(test_client, admin_headers):
    """Covers lines 125-127: generic Exception handling."""
    mock_client = AsyncMock()
    mock_client.post.side_effect = RuntimeError("Unexpected error")
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=None)

    with patch("api.license_bp.httpx.AsyncClient", return_value=mock_client), \
         patch.object(LicenseCacheModel, "get_cached_validation", return_value=None):
        response = await test_client.post(
            "/api/license/validate",
            json={"license_key": "test-key"},
            headers=admin_headers,
        )

    assert response.status_code == 500
    data = await response.get_json()
    assert "Failed to validate license" in data.get("error", "")


@pytest.mark.asyncio
async def test_keepalive_non_200_response(test_client, admin_headers, mock_db):
    """Covers lines 226-227: keepalive non-200 response."""
    # Setup cached license
    cached_data = {
        "id": 1,
        "is_valid": True,
        "is_enterprise": True,
        "max_proxies": 10,
        "keepalive_count": 5,
    }

    # Mock license server response (non-200)
    mock_response = MagicMock()
    mock_response.status_code = 400
    mock_response.json.return_value = {"error": "Keepalive failed"}

    mock_client = AsyncMock()
    mock_client.post.return_value = mock_response
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=None)

    with patch("api.license_bp.httpx.AsyncClient", return_value=mock_client), \
         patch.object(LicenseCacheModel, "get_cached_validation", return_value=cached_data):
        response = await test_client.post(
            "/api/license/keepalive",
            json={"license_key": "PENG-test-key"},
            headers=admin_headers,
        )

    assert response.status_code == 400


@pytest.mark.asyncio
async def test_keepalive_request_error(test_client, admin_headers, mock_db):
    """Covers lines 229-231: keepalive httpx.RequestError."""
    import httpx
    cached_data = {"is_enterprise": True}

    mock_client = AsyncMock()
    mock_client.post.side_effect = httpx.RequestError(
        "Connection lost"
    )
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=None)

    with patch("api.license_bp.httpx.AsyncClient", return_value=mock_client), \
         patch.object(LicenseCacheModel, "get_cached_validation", return_value=cached_data):
        response = await test_client.post(
            "/api/license/keepalive",
            json={"license_key": "test-key"},
            headers=admin_headers,
        )

    assert response.status_code == 503


@pytest.mark.asyncio
async def test_keepalive_generic_exception(test_client, admin_headers, mock_db):
    """Covers lines 232-234: keepalive generic Exception."""
    cached_data = {"is_enterprise": True}

    mock_client = AsyncMock()
    mock_client.post.side_effect = ValueError("Config error")
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=None)

    with patch("api.license_bp.httpx.AsyncClient", return_value=mock_client), \
         patch.object(LicenseCacheModel, "get_cached_validation", return_value=cached_data):
        response = await test_client.post(
            "/api/license/keepalive",
            json={"license_key": "test-key"},
            headers=admin_headers,
        )

    assert response.status_code == 500


# ============================================================================
# ADMIN_MEDIA_BP TESTS - Non-admin checks and exception handling
# ============================================================================


@pytest.mark.asyncio
async def test_admin_media_settings_non_admin_get_403(test_client, user_headers):
    """Covers line 37: non-admin GET /settings returns 403."""
    user_payload = {
        "user_id": 2,
        "username": "user",
        "is_admin": False,
        "scope": "",
        "roles": [],
        "tenant": "test",
        "session_id": "s2",
    }

    with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mv:
        mv.return_value = user_payload
        response = await test_client.get(
            "/api/v1/admin/media/settings",
            headers=user_headers,
        )

    assert response.status_code == 403
    data = await response.get_json()
    assert "Admin access required" in data.get("error", "")


@pytest.mark.asyncio
async def test_admin_media_settings_exception_500(test_client, admin_headers, mock_db):
    """Covers lines 117-119: exception in PUT /settings returns 500."""
    # Mock MediaSettingsModel.update_settings to raise exception
    with patch("api.admin_media_bp.MediaSettingsModel.update_settings") as mock_update:
        mock_update.side_effect = RuntimeError("Database error")

        response = await test_client.put(
            "/api/v1/admin/media/settings",
            json={
                "admin_max_resolution": 1080,
                "admin_max_bitrate_kbps": 5000,
            },
            headers=admin_headers,
        )

    assert response.status_code == 500
    data = await response.get_json()
    assert "Failed to update settings" in data.get("error", "")


@pytest.mark.asyncio
async def test_admin_media_settings_reset_non_admin_403(test_client, user_headers):
    """Covers line 134: non-admin POST /settings/reset returns 403."""
    user_payload = {
        "user_id": 2,
        "username": "user",
        "is_admin": False,
        "scope": "",
        "roles": [],
        "tenant": "test",
        "session_id": "s2",
    }

    with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mv:
        mv.return_value = user_payload
        response = await test_client.post(
            "/api/v1/admin/media/settings/reset",
            headers=user_headers,
        )

    assert response.status_code == 403
    data = await response.get_json()
    assert "Admin access required" in data.get("error", "")


@pytest.mark.asyncio
async def test_admin_media_settings_reset_exception_500(test_client, admin_headers):
    """Covers lines 155-157: exception in reset handler returns 500."""
    with patch("api.admin_media_bp.MediaSettingsModel.clear_admin_override") as mock_clear:
        mock_clear.side_effect = ValueError("Invalid operation")

        response = await test_client.post(
            "/api/v1/admin/media/settings/reset",
            headers=admin_headers,
        )

    assert response.status_code == 500
    data = await response.get_json()
    assert "Failed to reset settings" in data.get("error", "")


@pytest.mark.asyncio
async def test_admin_media_capabilities_non_admin_403(test_client, user_headers):
    """Covers line 169: non-admin GET /capabilities returns 403."""
    user_payload = {
        "user_id": 2,
        "username": "user",
        "is_admin": False,
        "scope": "",
        "roles": [],
        "tenant": "test",
        "session_id": "s2",
    }

    with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mv:
        mv.return_value = user_payload
        response = await test_client.get(
            "/api/v1/admin/media/capabilities",
            headers=user_headers,
        )

    assert response.status_code == 403
    data = await response.get_json()
    assert "Admin access required" in data.get("error", "")


@pytest.mark.asyncio
async def test_admin_media_capabilities_admin_200(test_client, admin_headers):
    """Covers lines 182-213: admin GET /capabilities returns 200 with full response."""
    with patch("api.admin_media_bp.get_hardware_capabilities", new_callable=AsyncMock) as mock_hw:
        mock_hw.return_value = {
            "gpu_type": "nvidia",
            "gpu_model": "RTX 4080",
            "vram_gb": 16,
            "hardware_max_resolution": 4320,
            "av1_supported": True,
            "supports_8k": True,
            "supports_4k": True,
        }

        response = await test_client.get(
            "/api/v1/admin/media/capabilities",
            headers=admin_headers,
        )

    assert response.status_code == 200
    data = await response.get_json()
    assert "hardware" in data
    assert "resolutions" in data
    assert "supported_codecs" in data
    assert data["hardware"]["gpu_type"] == "nvidia"


@pytest.mark.asyncio
async def test_get_hardware_capabilities_return_dict(test_client, admin_headers):
    """Covers line 213: get_hardware_capabilities returns dict."""
    from api.admin_media_bp import get_hardware_capabilities

    result = await get_hardware_capabilities()

    assert isinstance(result, dict)
    assert "gpu_type" in result
    assert "gpu_model" in result
    assert "vram_gb" in result
    assert "hardware_max_resolution" in result


@pytest.mark.asyncio
async def test_notify_rtmp_config_change_logs(test_client, admin_headers):
    """Covers line 230: notify_rtmp_config_change logs settings."""
    from api.admin_media_bp import notify_rtmp_config_change

    test_settings = {
        "admin_max_resolution": 1080,
        "transcode_ladder_enabled": True,
    }

    with patch("api.admin_media_bp.logger") as mock_logger:
        await notify_rtmp_config_change(test_settings)

        # Verify logging was called
        assert mock_logger.info.called
        call_args = mock_logger.info.call_args
        assert "Notifying proxy-rtmp" in str(call_args)


@pytest.mark.asyncio
async def test_admin_media_settings_disabled_reason_non_gpu(test_client, admin_headers):
    """Covers lines 188-189: disabled_reason for non-GPU hardware."""
    with patch("api.admin_media_bp.get_hardware_capabilities", new_callable=AsyncMock) as mock_hw:
        mock_hw.return_value = {
            "gpu_type": "none",
            "hardware_max_resolution": 1080,
            "av1_supported": False,
            "supports_8k": False,
            "supports_4k": False,
        }

        response = await test_client.get(
            "/api/v1/admin/media/capabilities",
            headers=admin_headers,
        )

    assert response.status_code == 200
    data = await response.get_json()
    resolutions = data["resolutions"]

    # Find 4K resolution
    res_4k = [r for r in resolutions if r["height"] == 2160][0]
    assert not res_4k["supported"]
    assert "disabled_reason" in res_4k
    assert "GPU" in res_4k["disabled_reason"]


@pytest.mark.asyncio
async def test_validate_license_invalid_json(test_client, admin_headers):
    """Test invalid JSON request to validate endpoint."""
    response = await test_client.post(
        "/api/license/validate",
        json={"wrong_field": "value"},
        headers=admin_headers,
    )

    assert response.status_code == 400


@pytest.mark.asyncio
async def test_keepalive_non_enterprise_400(test_client, admin_headers, mock_db):
    """Test keepalive with non-enterprise license returns 400."""
    # Return a cached non-enterprise license
    cached_data = {"is_enterprise": False}

    with patch.object(LicenseCacheModel, "get_cached_validation", return_value=cached_data):
        response = await test_client.post(
            "/api/license/keepalive",
            json={"license_key": "community-key"},
            headers=admin_headers,
        )

    # Should get 400 because license is not enterprise
    assert response.status_code == 400


@pytest.mark.asyncio
async def test_admin_media_get_settings_with_mock_data(test_client, admin_headers):
    """Test GET /settings returns hardware capabilities and effective max."""
    with patch("api.admin_media_bp.MediaSettingsModel.get_settings") as mock_get, \
         patch("api.admin_media_bp.get_hardware_capabilities", new_callable=AsyncMock) as mock_hw:

        mock_get.return_value = {
            "admin_max_resolution": 1080,
            "admin_max_bitrate_kbps": 5000,
            "enforce_codec": "h264",
            "transcode_ladder_enabled": True,
            "transcode_ladder_resolutions": [360, 540, 720, 1080],
            "updated_at": datetime.utcnow(),
        }
        mock_hw.return_value = {
            "gpu_type": "nvidia",
            "hardware_max_resolution": 2160,
            "av1_supported": True,
        }

        response = await test_client.get(
            "/api/v1/admin/media/settings",
            headers=admin_headers,
        )

    assert response.status_code == 200
    data = await response.get_json()
    assert data["settings"]["admin_max_resolution"] == 1080
    assert data["effective_max_resolution"] == 1080  # min of 1080 and 2160


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
