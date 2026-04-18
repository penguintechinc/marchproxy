"""
Extended tests for api/admin_media_bp.py blueprint.

Covers all route handlers including error cases, auth scenarios, and business logic.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from unittest.mock import AsyncMock, MagicMock, patch
from datetime import datetime

import pytest


def _admin_payload():
    return {
        "user_id": 1,
        "username": "admin",
        "is_admin": True,
        "roles": ["admin"],
        "scope": ["*:admin"],
    }


def _user_payload():
    return {"user_id": 2, "username": "user", "is_admin": False, "roles": [], "scope": []}


# ===========================================================================
# GET /api/v1/admin/media/settings — Get media settings
# ===========================================================================


class TestAdminMediaSettings:
    async def test_get_settings_no_auth_returns_401(self, test_client):
        """GET settings without auth returns 401"""
        resp = await test_client.get("/api/v1/admin/media/settings")
        assert resp.status_code == 401

    async def test_get_settings_non_admin_returns_403(self, test_app, test_client):
        """GET settings by non-admin returns 403"""
        fresh_db = MagicMock()

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.get(
                "/api/v1/admin/media/settings",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 403

    async def test_get_settings_success(self, test_app, test_client):
        """GET settings by admin returns 200 with settings"""
        fresh_db = MagicMock()

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("api.admin_media_bp.MediaSettingsModel.get_settings", return_value=None), \
             patch("api.admin_media_bp.get_hardware_capabilities", new_callable=AsyncMock) as mock_hw, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            mock_hw.return_value = {
                "gpu_type": "nvidia",
                "gpu_model": "RTX 4080",
                "vram_gb": 16,
                "hardware_max_resolution": 4320,
                "av1_supported": True,
                "supports_8k": True,
                "supports_4k": True,
            }
            resp = await test_client.get(
                "/api/v1/admin/media/settings",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 200
        data = await resp.get_json()
        assert "settings" in data
        assert "hardware_capabilities" in data
        assert "effective_max_resolution" in data


# ===========================================================================
# PUT /api/v1/admin/media/settings — Update media settings
# ===========================================================================


class TestAdminMediaSettingsPut:
    async def test_put_settings_no_auth_returns_401(self, test_client):
        """PUT settings without auth returns 401"""
        resp = await test_client.put(
            "/api/v1/admin/media/settings",
            json={"admin_max_resolution": 1080},
        )
        assert resp.status_code == 401

    async def test_put_settings_non_admin_returns_403(self, test_app, test_client):
        """PUT settings by non-admin returns 403"""
        fresh_db = MagicMock()

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.put(
                "/api/v1/admin/media/settings",
                json={"admin_max_resolution": 1080},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 403

    async def test_put_settings_validation_error_returns_400(self, test_app, test_client):
        """PUT settings with invalid data returns 400"""
        fresh_db = MagicMock()

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.put(
                "/api/v1/admin/media/settings",
                json={"admin_max_resolution": "not-a-number"},  # Invalid
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 400

    async def test_put_settings_success(self, test_app, test_client):
        """PUT settings by admin updates settings"""
        fresh_db = MagicMock()
        updated_settings = {
            "admin_max_resolution": 1080,
            "admin_max_bitrate_kbps": 5000,
            "enforce_codec": None,
            "transcode_ladder_enabled": True,
            "transcode_ladder_resolutions": [360, 540, 720, 1080],
        }

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("api.admin_media_bp.MediaSettingsModel.update_settings", return_value=updated_settings), \
             patch("api.admin_media_bp.notify_rtmp_config_change", new_callable=AsyncMock), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.put(
                "/api/v1/admin/media/settings",
                json={
                    "admin_max_resolution": 1080,
                    "admin_max_bitrate_kbps": 5000,
                },
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 200
        data = await resp.get_json()
        assert data.get("status") == "updated"
        assert "settings" in data


# ===========================================================================
# POST /api/v1/admin/media/settings/reset — Reset media settings
# ===========================================================================


class TestAdminMediaSettingsReset:
    async def test_reset_no_auth_returns_401(self, test_client):
        """POST reset without auth returns 401"""
        resp = await test_client.post("/api/v1/admin/media/settings/reset", json={})
        assert resp.status_code == 401

    async def test_reset_non_admin_returns_403(self, test_app, test_client):
        """POST reset by non-admin returns 403"""
        fresh_db = MagicMock()

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.post(
                "/api/v1/admin/media/settings/reset",
                json={},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 403

    async def test_reset_success(self, test_app, test_client):
        """POST reset by admin resets settings"""
        fresh_db = MagicMock()
        reset_settings = {
            "admin_max_resolution": None,
            "admin_max_bitrate_kbps": None,
            "enforce_codec": None,
            "transcode_ladder_enabled": True,
            "transcode_ladder_resolutions": [360, 540, 720, 1080],
        }

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("api.admin_media_bp.MediaSettingsModel.clear_admin_override", return_value=reset_settings), \
             patch("api.admin_media_bp.notify_rtmp_config_change", new_callable=AsyncMock), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/admin/media/settings/reset",
                json={},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 200
        data = await resp.get_json()
        assert data.get("status") == "reset"
        assert "settings" in data


# ===========================================================================
# GET /api/v1/admin/media/capabilities — Get hardware capabilities
# ===========================================================================


class TestAdminMediaCapabilities:
    async def test_capabilities_no_auth_returns_401(self, test_client):
        """GET capabilities without auth returns 401"""
        resp = await test_client.get("/api/v1/admin/media/capabilities")
        assert resp.status_code == 401

    async def test_capabilities_non_admin_returns_403(self, test_app, test_client):
        """GET capabilities by non-admin returns 403"""
        fresh_db = MagicMock()

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.get(
                "/api/v1/admin/media/capabilities",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 403

    async def test_capabilities_success_with_gpu(self, test_app, test_client):
        """GET capabilities by admin returns hardware info with GPU"""
        fresh_db = MagicMock()
        hardware_caps = {
            "gpu_type": "nvidia",
            "gpu_model": "RTX 4080",
            "vram_gb": 16,
            "hardware_max_resolution": 4320,
            "av1_supported": True,
            "supports_8k": True,
            "supports_4k": True,
        }

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("api.admin_media_bp.get_hardware_capabilities", new_callable=AsyncMock) as mock_hw, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            mock_hw.return_value = hardware_caps
            resp = await test_client.get(
                "/api/v1/admin/media/capabilities",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 200
        data = await resp.get_json()
        assert "hardware" in data
        assert "resolutions" in data
        assert "supported_codecs" in data
        assert data["hardware"]["gpu_type"] == "nvidia"

    async def test_capabilities_success_no_gpu(self, test_app, test_client):
        """GET capabilities by admin returns info when no GPU"""
        fresh_db = MagicMock()
        hardware_caps = {
            "gpu_type": "none",
            "gpu_model": None,
            "vram_gb": 0,
            "hardware_max_resolution": 1080,
            "av1_supported": False,
            "supports_8k": False,
            "supports_4k": False,
        }

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("api.admin_media_bp.get_hardware_capabilities", new_callable=AsyncMock) as mock_hw, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            mock_hw.return_value = hardware_caps
            resp = await test_client.get(
                "/api/v1/admin/media/capabilities",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 200
        data = await resp.get_json()
        assert data["hardware"]["gpu_type"] == "none"
        # Check resolutions - should mark some as not supported
        assert "resolutions" in data
        assert len(data["resolutions"]) > 0


# ===========================================================================
# Helper function tests
# ===========================================================================


class TestAdminMediaHelpers:
    def test_get_resolution_label(self):
        """Test resolution label mapping"""
        from api.admin_media_bp import get_resolution_label

        assert get_resolution_label(360) == "360p"
        assert get_resolution_label(720) == "720p (HD)"
        assert get_resolution_label(1080) == "1080p (Full HD)"
        assert get_resolution_label(2160) == "2160p (4K)"
        assert get_resolution_label(4320) == "4320p (8K)"
        assert get_resolution_label(999) == "999p"

    def test_get_supported_codecs_with_gpu(self):
        """Test codec support list with GPU"""
        from api.admin_media_bp import get_supported_codecs

        hardware_caps = {
            "gpu_type": "nvidia",
            "av1_supported": True,
        }
        codecs = get_supported_codecs(hardware_caps)

        assert len(codecs) == 3
        # H.264 should always be supported
        h264 = [c for c in codecs if c["id"] == "h264"][0]
        assert h264["supported"] is True
        assert h264["hardware_accelerated"] is True
        # H.265 should be supported with GPU
        h265 = [c for c in codecs if c["id"] == "h265"][0]
        assert h265["supported"] is True
        # AV1 should be supported if av1_supported is True
        av1 = [c for c in codecs if c["id"] == "av1"][0]
        assert av1["supported"] is True

    def test_get_supported_codecs_no_gpu(self):
        """Test codec support list without GPU"""
        from api.admin_media_bp import get_supported_codecs

        hardware_caps = {
            "gpu_type": "none",
            "av1_supported": False,
        }
        codecs = get_supported_codecs(hardware_caps)

        assert len(codecs) == 3
        # H.264 should still be supported (software fallback)
        h264 = [c for c in codecs if c["id"] == "h264"][0]
        assert h264["supported"] is True
        assert h264["hardware_accelerated"] is False
        # AV1 should not be supported without GPU
        av1 = [c for c in codecs if c["id"] == "av1"][0]
        assert av1["supported"] is False
        assert av1["hardware_accelerated"] is False
