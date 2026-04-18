"""
Final coverage tests for api/block_rules_bp.py - targeting 90%+ coverage.

Focuses on uncovered lines:
- Line 121: ValueError in create_rule
- Lines 139-143: Non-admin GET single rule access denied
- Lines 164-165: ValidationError in PUT update
- Line 170: update_rule returns False
- Line 179: ValueError in update_rule
- Line 200: delete_rule returns False
- Lines 204-206: Exception in delete_rule
- Lines 225-226: JSON parse exception in bulk create
- Lines 310-319: proxy_id in threat-feed endpoint

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from unittest.mock import AsyncMock, MagicMock, patch
import pytest
from pydantic import ValidationError


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _admin_payload():
    """Admin user payload for auth."""
    return {
        "user_id": 1,
        "username": "admin",
        "is_admin": True,
        "roles": ["admin"],
        "scope": "*:read *:write *:admin",
        "tenant": "test",
    }


def _user_payload():
    """Non-admin user payload for auth."""
    return {
        "user_id": 2,
        "username": "user",
        "is_admin": False,
        "roles": [],
        "scope": "read",
        "tenant": "test",
    }


def _rule_dict(rule_id=10, cluster_id=1):
    """Mock block rule as dictionary (as returned by model)."""
    return {
        "id": rule_id,
        "cluster_id": cluster_id,
        "rule_type": "ip",
        "layer": "l3",
        "value": "10.0.0.1",
        "name": "test-rule",
        "proxy_type": "envoy",
        "priority": 100,
        "is_active": True,
        "description": "test rule",
        "created_at": "2025-01-01T00:00:00",
        "updated_at": None,
    }


# ===========================================================================
# Line 121: ValueError in create_block_rule
# ===========================================================================

@pytest.mark.asyncio
async def test_create_block_rule_value_error(test_app, test_client, admin_headers):
    """
    Test POST /api/v1/<cluster_id>/block-rules when BlockRuleModel.create_rule
    raises ValueError (line 120-121). Should return 400.
    """
    # Setup: Cluster exists, request validates OK
    select_result = MagicMock()
    select_result.first = MagicMock(return_value=MagicMock(is_active=True))
    query_mock = MagicMock()
    query_mock.select = MagicMock(return_value=select_result)
    test_app.db.return_value = query_mock

    # Patch create_rule to raise ValueError after validation passes
    with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_auth, \
         patch("api.block_rules_bp.CreateBlockRuleRequest") as mock_req_class, \
         patch("api.block_rules_bp.BlockRuleModel.create_rule",
               side_effect=ValueError("Invalid port specification")):
        mock_auth.return_value = _admin_payload()

        # Create mock request that validates OK
        mock_req_obj = MagicMock()
        mock_req_obj.name = "test-rule"
        mock_req_obj.rule_type = "ip"
        mock_req_obj.layer = "l4"
        mock_req_obj.value = "192.168.1.1"
        mock_req_obj.ports = None
        mock_req_obj.protocols = None
        mock_req_obj.wildcard = False
        mock_req_obj.match_type = "exact"
        mock_req_obj.action = "drop"
        mock_req_obj.priority = 100
        mock_req_obj.apply_to_alb = False
        mock_req_obj.apply_to_nlb = False
        mock_req_obj.apply_to_egress = False
        mock_req_obj.expires_at = None
        mock_req_obj.description = None
        mock_req_class.return_value = mock_req_obj

        resp = await test_client.post(
            "/api/v1/1/block-rules",
            json={
                "rule_type": "ip",
                "layer": "l4",
                "value": "192.168.1.1",
                "name": "test-rule",
            },
            headers=admin_headers,
        )

    assert resp.status_code == 400
    data = await resp.get_json()
    assert "error" in data
    assert "Invalid port specification" in data["error"]


# ===========================================================================
# Lines 139-143: Non-admin GET single rule - check_user_cluster_access fails
# ===========================================================================

@pytest.mark.asyncio
async def test_get_single_rule_non_admin_no_access(test_app, test_client, admin_headers):
    """
    Test GET /api/v1/<cluster_id>/block-rules/<rule_id> with non-admin user
    who has no cluster access. Should return 403.
    Lines 139-143 test the non-admin access check path.
    """
    # Patch _validate_token to return non-admin, check_user_cluster_access returns None (no access)
    with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_auth, \
         patch("api.block_rules_bp.UserClusterAssignmentModel.check_user_cluster_access",
               return_value=None):
        mock_auth.return_value = _user_payload()

        resp = await test_client.get(
            "/api/v1/1/block-rules/10",
            headers=admin_headers,
        )

    assert resp.status_code == 403
    data = await resp.get_json()
    assert "Access denied" in data["error"]


# ===========================================================================
# Lines 164-165: ValidationError in PUT update
# ===========================================================================

@pytest.mark.asyncio
async def test_update_block_rule_not_found(test_app, test_client, admin_headers):
    """
    Test PUT /api/v1/<cluster_id>/block-rules/<rule_id> when rule doesn't exist.
    Should return 404 (lines 157-159).
    """
    # Mock rule doesn't exist (get_rule returns None)
    with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_auth, \
         patch("api.block_rules_bp.BlockRuleModel.get_rule",
               return_value=None):  # Rule not found
        mock_auth.return_value = _admin_payload()

        resp = await test_client.put(
            "/api/v1/1/block-rules/999",
            json={"action": "drop"},
            headers=admin_headers,
        )

    assert resp.status_code == 404
    data = await resp.get_json()
    assert "not found" in data["error"].lower()


# ===========================================================================
# Line 170: update_rule returns False
# ===========================================================================

@pytest.mark.asyncio
async def test_update_block_rule_returns_false(test_app, test_client, admin_headers):
    """
    Test PUT /api/v1/clusters/<cluster_id>/block-rules/<rule_id> when update_rule
    returns False (DB update failed) - line 170. Should return 500.
    """
    with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_auth, \
         patch("api.block_rules_bp.BlockRuleModel.get_rule",
               return_value=_rule_dict(10, 1)), \
         patch("api.block_rules_bp.UpdateBlockRuleRequest") as mock_req_class, \
         patch("api.block_rules_bp.BlockRuleModel.update_rule", return_value=False):
        mock_auth.return_value = _admin_payload()

        # Mock the request object to have valid data
        mock_req_obj = MagicMock()
        mock_req_obj.dict = MagicMock(return_value={"action": "drop"})
        mock_req_class.return_value = mock_req_obj

        resp = await test_client.put(
            "/api/v1/1/block-rules/10",
            json={"action": "drop"},
            headers=admin_headers,
        )

    assert resp.status_code == 500
    data = await resp.get_json()
    assert "Failed to update" in data["error"]


# ===========================================================================
# Line 179: ValueError in update_rule
# ===========================================================================

@pytest.mark.asyncio
async def test_update_block_rule_value_error(test_app, test_client, admin_headers):
    """
    Test PUT /api/v1/clusters/<cluster_id>/block-rules/<rule_id> when update_rule
    raises ValueError (line 179). Should return 400.
    """
    with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_auth, \
         patch("api.block_rules_bp.BlockRuleModel.get_rule",
               return_value=_rule_dict(10, 1)), \
         patch("api.block_rules_bp.UpdateBlockRuleRequest") as mock_req_class, \
         patch("api.block_rules_bp.BlockRuleModel.update_rule",
               side_effect=ValueError("Cannot update immutable field")):
        mock_auth.return_value = _admin_payload()

        mock_req_obj = MagicMock()
        mock_req_obj.dict = MagicMock(return_value={"rule_type": "hostname"})
        mock_req_class.return_value = mock_req_obj

        resp = await test_client.put(
            "/api/v1/1/block-rules/10",
            json={"rule_type": "hostname"},
            headers=admin_headers,
        )

    assert resp.status_code == 400
    data = await resp.get_json()
    assert "error" in data


# ===========================================================================
# Line 200: delete_rule returns False
# ===========================================================================

@pytest.mark.asyncio
async def test_delete_block_rule_returns_false(test_app, test_client, admin_headers):
    """
    Test DELETE /api/v1/clusters/<cluster_id>/block-rules/<rule_id> when delete_rule
    returns False (DB delete failed) - line 200. Should return 500.
    """
    with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_auth, \
         patch("api.block_rules_bp.BlockRuleModel.get_rule",
               return_value=_rule_dict(10, 1)), \
         patch("api.block_rules_bp.BlockRuleModel.delete_rule", return_value=False):
        mock_auth.return_value = _admin_payload()

        resp = await test_client.delete(
            "/api/v1/1/block-rules/10",
            headers=admin_headers,
        )

    assert resp.status_code == 500
    data = await resp.get_json()
    assert "Failed to delete" in data["error"]


# ===========================================================================
# Lines 204-206: Exception in delete_rule
# ===========================================================================

@pytest.mark.asyncio
async def test_delete_block_rule_exception(test_app, test_client, admin_headers):
    """
    Test DELETE /api/v1/clusters/<cluster_id>/block-rules/<rule_id> when delete_rule
    raises an exception (lines 204-206). Should return 500.
    """
    with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_auth, \
         patch("api.block_rules_bp.BlockRuleModel.get_rule",
               return_value=_rule_dict(10, 1)), \
         patch("api.block_rules_bp.BlockRuleModel.delete_rule",
               side_effect=Exception("Database connection lost")):
        mock_auth.return_value = _admin_payload()

        resp = await test_client.delete(
            "/api/v1/1/block-rules/10",
            headers=admin_headers,
        )

    assert resp.status_code == 500
    data = await resp.get_json()
    assert "Failed to delete" in data["error"]


# ===========================================================================
# Lines 225-226: JSON parse exception in bulk create
# ===========================================================================

@pytest.mark.asyncio
async def test_bulk_create_json_parse_error(test_app, test_client, admin_headers):
    """
    Test POST /api/v1/clusters/<cluster_id>/block-rules/bulk with malformed JSON.
    request.get_json() raises exception (lines 225-226). Should return 400.
    """
    # Don't mock cluster check since we'll fail earlier on get_json
    # The blueprint has admin_required=True, so auth is already patched
    # We need to patch at the request level in the blueprint function

    async def mock_get_json_fail():
        raise Exception("Invalid JSON payload")

    with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_auth:
        mock_auth.return_value = _admin_payload()

        # We'll send a request that triggers get_json, which will fail
        # Since we can't directly patch inside the blueprint, we send bad data
        # that will cause an error when the client parses it
        resp = await test_client.post(
            "/api/v1/1/block-rules/bulk",
            data=b"invalid json {{{",  # Invalid JSON bytes
            headers={**admin_headers, "Content-Type": "application/json"},
        )

    assert resp.status_code == 400
    data = await resp.get_json()
    assert "Invalid JSON" in data["error"]


# ===========================================================================
# Lines 310-319: proxy_id in threat-feed endpoint
# ===========================================================================

@pytest.mark.asyncio
async def test_threat_feed_with_proxy_id_updates_sync_status(test_app, test_client):
    """
    Test GET /api/v1/clusters/<cluster_id>/threat-feed?proxy_id=5
    Should call BlockRuleSyncModel.update_sync_status when proxy_id is provided.
    """
    # Setup: Mock cluster validation
    cluster_info = MagicMock()
    cluster_info.__getitem__ = MagicMock(side_effect=lambda k: 1 if k == "cluster_id" else None)

    # Mock threat feed response
    threat_feed = {
        "version": "v1",
        "rules_count": 5,
        "rules": [
            {"id": 1, "rule_type": "ip", "layer": "l3", "value": "10.0.0.1"}
        ],
    }

    with patch("api.block_rules_bp.ClusterModel.validate_api_key",
               return_value=cluster_info), \
         patch("api.block_rules_bp.BlockRuleModel.get_threat_feed",
               return_value=threat_feed), \
         patch("api.block_rules_bp.BlockRuleSyncModel.update_sync_status") as mock_sync:

        resp = await test_client.get(
            "/api/v1/1/threat-feed?proxy_id=5",
            headers={"X-API-Key": "test-api-key"},
        )

    assert resp.status_code == 200
    # Verify update_sync_status was called with proxy_id=5, version, rules_count, status="synced"
    mock_sync.assert_called_once()
    call_args = mock_sync.call_args
    # First arg is db, second is proxy_id (int)
    assert call_args[0][1] == 5  # proxy_id
    assert "version" in call_args[1] or call_args[0] > 1  # version in kwargs or positional


@pytest.mark.asyncio
async def test_threat_feed_with_proxy_id_sync_failure_logs_warning(test_app, test_client):
    """
    Test GET /api/v1/<cluster_id>/threat-feed?proxy_id=5 when update_sync_status
    raises an exception. Should log warning but still return 200 (lines 310-319).
    """
    cluster_info = MagicMock()
    cluster_info.__getitem__ = MagicMock(side_effect=lambda k: 1 if k == "cluster_id" else None)

    threat_feed = {
        "version": "v1",
        "rules_count": 2,
        "rules": [],
    }

    with patch("api.block_rules_bp.ClusterModel.validate_api_key",
               return_value=cluster_info), \
         patch("api.block_rules_bp.BlockRuleModel.get_threat_feed",
               return_value=threat_feed), \
         patch("api.block_rules_bp.BlockRuleSyncModel.update_sync_status",
               side_effect=Exception("Sync DB error")):

        resp = await test_client.get(
            "/api/v1/1/threat-feed?proxy_id=5",
            headers={"X-API-Key": "test-api-key"},
        )

    # Should still return 200 even if sync update fails (warning is logged)
    assert resp.status_code == 200
    data = await resp.get_json()
    assert data["rules_count"] == 2


# ===========================================================================
# Additional tests for remaining uncovered lines
# ===========================================================================

@pytest.mark.asyncio
async def test_delete_rule_non_admin_no_access(test_app, test_client, admin_headers):
    """
    Test DELETE /api/v1/<cluster_id>/block-rules/<rule_id> with non-admin who lacks access.
    Should return 403 (line 187-188 admin check follows similar pattern to line 186-187 for auth).
    """
    with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_auth:
        mock_auth.return_value = _user_payload()  # Non-admin

        resp = await test_client.delete(
            "/api/v1/1/block-rules/10",
            headers=admin_headers,
        )

    assert resp.status_code == 403
    data = await resp.get_json()
    assert "Admin access required" in data["error"]


@pytest.mark.asyncio
async def test_create_rule_cluster_not_found(test_app, test_client, admin_headers):
    """
    Test POST /api/v1/<cluster_id>/block-rules when cluster doesn't exist.
    Should return 404 (line 84).
    """
    # Mock cluster doesn't exist
    select_result = MagicMock()
    select_result.first = MagicMock(return_value=None)  # No cluster
    query_mock = MagicMock()
    query_mock.select = MagicMock(return_value=select_result)
    test_app.db.return_value = query_mock

    with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_auth:
        mock_auth.return_value = _admin_payload()

        resp = await test_client.post(
            "/api/v1/999/block-rules",
            json={"rule_type": "ip", "layer": "l3", "value": "10.0.0.1", "name": "test"},
            headers=admin_headers,
        )

    assert resp.status_code == 404
    data = await resp.get_json()
    assert "Cluster not found" in data["error"]


@pytest.mark.asyncio
async def test_bulk_create_cluster_not_found(test_app, test_client, admin_headers):
    """
    Test POST /api/v1/<cluster_id>/block-rules/bulk when cluster doesn't exist.
    Should return 404 (line 221).
    """
    # Mock cluster doesn't exist
    select_result = MagicMock()
    select_result.first = MagicMock(return_value=None)
    query_mock = MagicMock()
    query_mock.select = MagicMock(return_value=select_result)
    test_app.db.return_value = query_mock

    with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_auth:
        mock_auth.return_value = _admin_payload()

        resp = await test_client.post(
            "/api/v1/999/block-rules/bulk",
            json={"rules": [{"rule_type": "ip", "layer": "l3", "value": "10.0.0.1", "name": "test"}]},
            headers=admin_headers,
        )

    assert resp.status_code == 404
    data = await resp.get_json()
    assert "Cluster not found" in data["error"]


@pytest.mark.asyncio
async def test_bulk_create_no_rules_provided(test_app, test_client, admin_headers):
    """
    Test POST /api/v1/<cluster_id>/block-rules/bulk with empty rules list.
    Should return 400 (line 230).
    """
    # Mock cluster exists
    select_result = MagicMock()
    select_result.first = MagicMock(return_value=MagicMock(is_active=True))
    query_mock = MagicMock()
    query_mock.select = MagicMock(return_value=select_result)
    test_app.db.return_value = query_mock

    with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_auth:
        mock_auth.return_value = _admin_payload()

        resp = await test_client.post(
            "/api/v1/1/block-rules/bulk",
            json={"rules": []},  # Empty rules list
            headers=admin_headers,
        )

    assert resp.status_code == 400
    data = await resp.get_json()
    assert "No rules provided" in data["error"]


@pytest.mark.asyncio
async def test_create_rule_exception_handler(test_app, test_client, admin_headers):
    """
    Test POST /api/v1/<cluster_id>/block-rules when an unexpected exception occurs.
    Should return 500 (lines 122-124).
    """
    # Mock cluster exists
    select_result = MagicMock()
    select_result.first = MagicMock(return_value=MagicMock(is_active=True))
    query_mock = MagicMock()
    query_mock.select = MagicMock(return_value=select_result)
    test_app.db.return_value = query_mock

    with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_auth, \
         patch("api.block_rules_bp.CreateBlockRuleRequest") as mock_req_class, \
         patch("api.block_rules_bp.BlockRuleModel.create_rule",
               side_effect=Exception("Unexpected database error")):
        mock_auth.return_value = _admin_payload()

        mock_req_obj = MagicMock()
        mock_req_obj.name = "test"
        mock_req_obj.rule_type = "ip"
        mock_req_obj.layer = "l3"
        mock_req_obj.value = "10.0.0.1"
        mock_req_obj.ports = None
        mock_req_obj.protocols = None
        mock_req_obj.wildcard = False
        mock_req_obj.match_type = "exact"
        mock_req_obj.action = "drop"
        mock_req_obj.priority = 100
        mock_req_obj.apply_to_alb = False
        mock_req_obj.apply_to_nlb = False
        mock_req_obj.apply_to_egress = False
        mock_req_obj.expires_at = None
        mock_req_obj.description = None
        mock_req_class.return_value = mock_req_obj

        resp = await test_client.post(
            "/api/v1/1/block-rules",
            json={"rule_type": "ip", "layer": "l3", "value": "10.0.0.1", "name": "test"},
            headers=admin_headers,
        )

    assert resp.status_code == 500
    data = await resp.get_json()
    assert "error" in data


@pytest.mark.asyncio
async def test_update_exception_handler(test_app, test_client, admin_headers):
    """
    Test PUT /api/v1/<cluster_id>/block-rules/<rule_id> when generic exception occurs.
    Should return 500 (lines 180-182).
    """
    with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_auth, \
         patch("api.block_rules_bp.BlockRuleModel.get_rule",
               return_value=_rule_dict(10, 1)), \
         patch("api.block_rules_bp.UpdateBlockRuleRequest") as mock_req_class, \
         patch("api.block_rules_bp.BlockRuleModel.update_rule",
               side_effect=Exception("DB connection timeout")):
        mock_auth.return_value = _admin_payload()

        mock_req_obj = MagicMock()
        mock_req_obj.dict = MagicMock(return_value={})
        mock_req_class.return_value = mock_req_obj

        resp = await test_client.put(
            "/api/v1/1/block-rules/10",
            json={"action": "drop"},
            headers=admin_headers,
        )

    assert resp.status_code == 500
    data = await resp.get_json()
    assert "error" in data


@pytest.mark.asyncio
async def test_get_rules_with_filters(test_app, test_client, admin_headers):
    """
    Test GET /api/v1/<cluster_id>/block-rules with filters.
    This exercises the filter parameters (line 53-65).
    """
    # Mock cluster exists and returns rules
    cluster_mock = MagicMock(is_active=True)
    select_result = MagicMock()
    select_result.first = MagicMock(return_value=cluster_mock)
    query_mock = MagicMock()
    query_mock.select = MagicMock(return_value=select_result)
    test_app.db.return_value = query_mock

    with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_auth, \
         patch("api.block_rules_bp.BlockRuleModel.list_rules",
               return_value=[_rule_dict(1, 1), _rule_dict(2, 1)]):
        mock_auth.return_value = _admin_payload()

        resp = await test_client.get(
            "/api/v1/1/block-rules?rule_type=ip&layer=l3&include_inactive=true",
            headers=admin_headers,
        )

    assert resp.status_code == 200
    data = await resp.get_json()
    assert data["rules_count"] == 2
    assert len(data["rules"]) == 2
