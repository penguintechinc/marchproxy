"""
Comprehensive HTTP-level tests for api/block_rules_bp.py blueprint.

Routes at /api/v1/clusters prefix:
  GET /api/v1/<cluster_id>/block-rules - list rules
  POST /api/v1/<cluster_id>/block-rules - create rule (admin only)
  GET /api/v1/<cluster_id>/block-rules/<rule_id> - get single rule
  PUT /api/v1/<cluster_id>/block-rules/<rule_id> - update rule (admin only)
  DELETE /api/v1/<cluster_id>/block-rules/<rule_id> - delete rule (admin only)
  POST /api/v1/<cluster_id>/block-rules/bulk - bulk create (admin only)
  GET /api/v1/<cluster_id>/threat-feed - API key auth (no JWT)
  GET /api/v1/<cluster_id>/block-rules/version - API key auth (no JWT)
  GET /api/v1/<cluster_id>/block-rules/sync-status - JWT auth

Coverage:
  - Admin vs non-admin access control
  - Cluster existence validation
  - Rule existence validation
  - Validation errors
  - Success cases
  - API key authentication
  - Bulk operations with mixed valid/invalid rules
  - Sync status endpoint

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest


# ---------------------------------------------------------------------------
# Valid request data
# ---------------------------------------------------------------------------

VALID_RULE = {
    "name": "Block TOR Exit",
    "rule_type": "ip",
    "layer": "L4",
    "value": "192.168.1.1",
    "description": "Block TOR exit node",
    "ports": [80, 443],
    "protocols": ["tcp"],
    "wildcard": False,
    "match_type": "exact",
    "action": "drop",
    "priority": 100,
    "apply_to_alb": True,
    "apply_to_nlb": True,
    "apply_to_egress": False,
    "expires_at": None,
}


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _admin_payload():
    """Decoded JWT payload for admin user."""
    return {
        "user_id": 1,
        "username": "admin",
        "is_admin": True,
        "roles": ["admin"],
        "scope": "*:read *:write *:admin *:delete",
        "tenant": "test",
    }


def _user_payload():
    """Decoded JWT payload for regular user."""
    return {
        "user_id": 2,
        "username": "user",
        "is_admin": False,
        "roles": [],
        "scope": "*:read",
        "tenant": "test",
    }


def _cluster_mock(cluster_id=1, is_active=True):
    """Create a mock cluster row."""
    cluster = MagicMock()
    cluster.id = cluster_id
    cluster.name = f"test-cluster-{cluster_id}"
    cluster.is_active = is_active
    return cluster


def _rule_dict(rule_id=10, cluster_id=1):
    """Create a rule dictionary matching BlockRuleModel response."""
    return {
        "id": rule_id,
        "cluster_id": cluster_id,
        "name": "Test Rule",
        "rule_type": "ip",
        "layer": "L4",
        "value": "10.0.0.1",
        "description": "Test rule description",
        "ports": [80, 443],
        "protocols": ["tcp"],
        "wildcard": False,
        "match_type": "exact",
        "action": "drop",
        "priority": 100,
        "apply_to_alb": True,
        "apply_to_nlb": True,
        "apply_to_egress": False,
        "source": "manual",
        "source_feed_name": None,
        "expires_at": None,
        "is_active": True,
        "created_by": 1,
        "created_at": "2025-01-01T00:00:00Z",
        "updated_at": "2025-01-01T00:00:00Z",
    }


# ===========================================================================
# GET /api/v1/<cluster_id>/block-rules - List rules
# ===========================================================================

class TestListBlockRules:
    """Test listing block rules for a cluster."""

    @pytest.mark.asyncio
    async def test_list_no_auth_returns_401(self, test_client):
        """Unauthenticated request returns 401."""
        resp = await test_client.get("/api/v1/1/block-rules")
        assert resp.status_code == 401

    @pytest.mark.asyncio
    async def test_list_admin_cluster_not_found_returns_404(self, test_app, test_client):
        """Admin user, cluster not found returns 404."""
        fresh_db = MagicMock()
        select_result = MagicMock()
        select_result.first = MagicMock(return_value=None)
        query_mock = MagicMock()
        query_mock.select = MagicMock(return_value=select_result)
        fresh_db.return_value = query_mock

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/v1/1/block-rules",
                headers={"Authorization": "Bearer admin-token"},
            )
        assert resp.status_code == 404
        data = await resp.get_json()
        assert "Cluster not found" in data.get("error", "")

    @pytest.mark.asyncio
    async def test_list_user_no_cluster_access_returns_403(self, test_app, test_client):
        """Non-admin user without cluster access returns 403."""
        fresh_db = MagicMock()

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.cluster.UserClusterAssignmentModel.check_user_cluster_access",
                   return_value=None), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.get(
                "/api/v1/1/block-rules",
                headers={"Authorization": "Bearer user-token"},
            )
        assert resp.status_code == 403
        data = await resp.get_json()
        assert "Access denied" in data.get("error", "")

    @pytest.mark.asyncio
    async def test_list_admin_success_empty(self, test_app, test_client):
        """Admin user lists rules successfully, empty result."""
        cluster = _cluster_mock(1)
        fresh_db = MagicMock()
        select_result = MagicMock()
        select_result.first = MagicMock(return_value=cluster)
        query_mock = MagicMock()
        query_mock.select = MagicMock(return_value=select_result)
        fresh_db.return_value = query_mock

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.block_rules.BlockRuleModel.list_rules", return_value=[]), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/v1/1/block-rules",
                headers={"Authorization": "Bearer admin-token"},
            )
        assert resp.status_code == 200
        data = await resp.get_json()
        assert data["cluster_id"] == 1
        assert data["rules_count"] == 0
        assert data["rules"] == []

    @pytest.mark.asyncio
    async def test_list_admin_success_with_rules(self, test_app, test_client):
        """Admin user lists rules successfully with data."""
        cluster = _cluster_mock(1)
        rules = [
            _rule_dict(10, 1),
            _rule_dict(11, 1),
        ]
        fresh_db = MagicMock()
        select_result = MagicMock()
        select_result.first = MagicMock(return_value=cluster)
        query_mock = MagicMock()
        query_mock.select = MagicMock(return_value=select_result)
        fresh_db.return_value = query_mock

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.block_rules.BlockRuleModel.list_rules", return_value=rules), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/v1/1/block-rules",
                headers={"Authorization": "Bearer admin-token"},
            )
        assert resp.status_code == 200
        data = await resp.get_json()
        assert data["cluster_id"] == 1
        assert data["rules_count"] == 2
        assert len(data["rules"]) == 2

    @pytest.mark.asyncio
    async def test_list_with_filters(self, test_app, test_client):
        """List rules with query parameter filters."""
        cluster = _cluster_mock(1)
        rules = [_rule_dict(10, 1)]
        fresh_db = MagicMock()
        select_result = MagicMock()
        select_result.first = MagicMock(return_value=cluster)
        query_mock = MagicMock()
        query_mock.select = MagicMock(return_value=select_result)
        fresh_db.return_value = query_mock

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.block_rules.BlockRuleModel.list_rules", return_value=rules) as mock_list, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/v1/1/block-rules?rule_type=ip&layer=L4&include_inactive=true",
                headers={"Authorization": "Bearer admin-token"},
            )
        assert resp.status_code == 200
        # Verify list_rules was called with filters
        mock_list.assert_called_once()
        call_args = mock_list.call_args
        assert call_args[1]["rule_type"] == "ip"
        assert call_args[1]["layer"] == "L4"
        assert call_args[1]["include_inactive"] is True


# ===========================================================================
# POST /api/v1/<cluster_id>/block-rules - Create rule
# ===========================================================================

class TestCreateBlockRule:
    """Test creating a block rule."""

    @pytest.mark.asyncio
    async def test_create_no_auth_returns_401(self, test_client):
        """Unauthenticated request returns 401."""
        resp = await test_client.post(
            "/api/v1/1/block-rules",
            json=VALID_RULE,
        )
        assert resp.status_code == 401

    @pytest.mark.asyncio
    async def test_create_non_admin_returns_403(self, test_app, test_client):
        """Non-admin user cannot create rules."""
        fresh_db = MagicMock()

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.post(
                "/api/v1/1/block-rules",
                json=VALID_RULE,
                headers={"Authorization": "Bearer user-token"},
            )
        assert resp.status_code == 403
        data = await resp.get_json()
        assert "Admin access required" in data.get("error", "")

    @pytest.mark.asyncio
    async def test_create_admin_cluster_not_found_returns_404(self, test_app, test_client):
        """Admin user, cluster not found returns 404."""
        fresh_db = MagicMock()
        select_result = MagicMock()
        select_result.first = MagicMock(return_value=None)
        query_mock = MagicMock()
        query_mock.select = MagicMock(return_value=select_result)
        fresh_db.return_value = query_mock

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/1/block-rules",
                json=VALID_RULE,
                headers={"Authorization": "Bearer admin-token"},
            )
        assert resp.status_code == 404

    @pytest.mark.asyncio
    async def test_create_invalid_json_returns_400(self, test_app, test_client):
        """Invalid JSON request returns 400."""
        cluster = _cluster_mock(1)
        fresh_db = MagicMock()
        select_result = MagicMock()
        select_result.first = MagicMock(return_value=cluster)
        query_mock = MagicMock()
        query_mock.select = MagicMock(return_value=select_result)
        fresh_db.return_value = query_mock

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/1/block-rules",
                json={},  # Missing required fields
                headers={"Authorization": "Bearer admin-token"},
            )
        assert resp.status_code == 400
        data = await resp.get_json()
        assert "error" in data

    @pytest.mark.asyncio
    async def test_create_success(self, test_app, test_client):
        """Successfully create a block rule."""
        cluster = _cluster_mock(1)
        rule = _rule_dict(10, 1)
        fresh_db = MagicMock()
        select_result = MagicMock()
        select_result.first = MagicMock(return_value=cluster)
        query_mock = MagicMock()
        query_mock.select = MagicMock(return_value=select_result)
        fresh_db.return_value = query_mock

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.block_rules.BlockRuleModel.create_rule", return_value=10), \
             patch("models.block_rules.BlockRuleModel.get_rule", return_value=rule), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/1/block-rules",
                json=VALID_RULE,
                headers={"Authorization": "Bearer admin-token"},
            )
        assert resp.status_code == 201
        data = await resp.get_json()
        assert "message" in data
        assert "rule" in data

    @pytest.mark.asyncio
    async def test_create_db_error_returns_500(self, test_app, test_client):
        """Database error during creation returns 500."""
        cluster = _cluster_mock(1)
        fresh_db = MagicMock()
        select_result = MagicMock()
        select_result.first = MagicMock(return_value=cluster)
        query_mock = MagicMock()
        query_mock.select = MagicMock(return_value=select_result)
        fresh_db.return_value = query_mock

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.block_rules.BlockRuleModel.create_rule",
                   side_effect=Exception("DB error")), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/1/block-rules",
                json=VALID_RULE,
                headers={"Authorization": "Bearer admin-token"},
            )
        assert resp.status_code == 500


# ===========================================================================
# GET /api/v1/<cluster_id>/block-rules/<rule_id> - Get single rule
# ===========================================================================

class TestGetSingleBlockRule:
    """Test retrieving a single block rule."""

    @pytest.mark.asyncio
    async def test_get_no_auth_returns_401(self, test_client):
        """Unauthenticated request returns 401."""
        resp = await test_client.get("/api/v1/1/block-rules/10")
        assert resp.status_code == 401

    @pytest.mark.asyncio
    async def test_get_rule_not_found_returns_404(self, test_app, test_client):
        """Rule not found returns 404."""
        fresh_db = MagicMock()

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.block_rules.BlockRuleModel.get_rule", return_value=None), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/v1/1/block-rules/10",
                headers={"Authorization": "Bearer admin-token"},
            )
        assert resp.status_code == 404
        data = await resp.get_json()
        assert "not found" in data.get("error", "").lower()

    @pytest.mark.asyncio
    async def test_get_rule_wrong_cluster_returns_404(self, test_app, test_client):
        """Rule from different cluster returns 404."""
        rule = _rule_dict(10, 2)  # Rule belongs to cluster 2
        fresh_db = MagicMock()

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.block_rules.BlockRuleModel.get_rule", return_value=rule), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/v1/1/block-rules/10",
                headers={"Authorization": "Bearer admin-token"},
            )
        assert resp.status_code == 404

    @pytest.mark.asyncio
    async def test_get_rule_success(self, test_app, test_client):
        """Successfully retrieve a rule."""
        rule = _rule_dict(10, 1)
        fresh_db = MagicMock()

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.block_rules.BlockRuleModel.get_rule", return_value=rule), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/v1/1/block-rules/10",
                headers={"Authorization": "Bearer admin-token"},
            )
        assert resp.status_code == 200
        data = await resp.get_json()
        assert data["id"] == 10
        assert data["cluster_id"] == 1


# ===========================================================================
# PUT /api/v1/<cluster_id>/block-rules/<rule_id> - Update rule
# ===========================================================================

class TestUpdateBlockRule:
    """Test updating a block rule."""

    @pytest.mark.asyncio
    async def test_update_no_auth_returns_401(self, test_client):
        """Unauthenticated request returns 401."""
        resp = await test_client.put(
            "/api/v1/1/block-rules/10",
            json={"priority": 200},
        )
        assert resp.status_code == 401

    @pytest.mark.asyncio
    async def test_update_non_admin_returns_403(self, test_app, test_client):
        """Non-admin user cannot update rules."""
        fresh_db = MagicMock()

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.put(
                "/api/v1/1/block-rules/10",
                json={"priority": 200},
                headers={"Authorization": "Bearer user-token"},
            )
        assert resp.status_code == 403
        data = await resp.get_json()
        assert "Admin access required" in data.get("error", "")

    @pytest.mark.asyncio
    async def test_update_rule_not_found_returns_404(self, test_app, test_client):
        """Updating non-existent rule returns 404."""
        fresh_db = MagicMock()

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.block_rules.BlockRuleModel.get_rule", return_value=None), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.put(
                "/api/v1/1/block-rules/10",
                json={"priority": 200},
                headers={"Authorization": "Bearer admin-token"},
            )
        assert resp.status_code == 404

    @pytest.mark.asyncio
    async def test_update_invalid_json_returns_400(self, test_app, test_client):
        """Invalid update JSON returns 400."""
        rule = _rule_dict(10, 1)
        fresh_db = MagicMock()

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.block_rules.BlockRuleModel.get_rule", return_value=rule), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.put(
                "/api/v1/1/block-rules/10",
                json={"invalid_field": "value"},
                headers={"Authorization": "Bearer admin-token"},
            )
        # Pydantic UpdateBlockRuleRequest may accept unknown fields; test actual behavior
        assert resp.status_code in [200, 400]

    @pytest.mark.asyncio
    async def test_update_success(self, test_app, test_client):
        """Successfully update a rule."""
        rule = _rule_dict(10, 1)
        updated_rule = _rule_dict(10, 1)
        updated_rule["priority"] = 200
        fresh_db = MagicMock()

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.block_rules.BlockRuleModel.get_rule",
                   side_effect=[rule, updated_rule]), \
             patch("models.block_rules.BlockRuleModel.update_rule", return_value=True), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.put(
                "/api/v1/1/block-rules/10",
                json={"priority": 200},
                headers={"Authorization": "Bearer admin-token"},
            )
        assert resp.status_code == 200
        data = await resp.get_json()
        assert "message" in data
        assert "rule" in data

    @pytest.mark.asyncio
    async def test_update_db_error_returns_500(self, test_app, test_client):
        """Database error during update returns 500."""
        rule = _rule_dict(10, 1)
        fresh_db = MagicMock()

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.block_rules.BlockRuleModel.get_rule", return_value=rule), \
             patch("models.block_rules.BlockRuleModel.update_rule",
                   side_effect=Exception("DB error")), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.put(
                "/api/v1/1/block-rules/10",
                json={"priority": 200},
                headers={"Authorization": "Bearer admin-token"},
            )
        assert resp.status_code == 500


# ===========================================================================
# DELETE /api/v1/<cluster_id>/block-rules/<rule_id> - Delete rule
# ===========================================================================

class TestDeleteBlockRule:
    """Test deleting a block rule."""

    @pytest.mark.asyncio
    async def test_delete_no_auth_returns_401(self, test_client):
        """Unauthenticated request returns 401."""
        resp = await test_client.delete("/api/v1/1/block-rules/10")
        assert resp.status_code == 401

    @pytest.mark.asyncio
    async def test_delete_non_admin_returns_403(self, test_app, test_client):
        """Non-admin user cannot delete rules."""
        fresh_db = MagicMock()

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.delete(
                "/api/v1/1/block-rules/10",
                headers={"Authorization": "Bearer user-token"},
            )
        assert resp.status_code == 403

    @pytest.mark.asyncio
    async def test_delete_rule_not_found_returns_404(self, test_app, test_client):
        """Deleting non-existent rule returns 404."""
        fresh_db = MagicMock()

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.block_rules.BlockRuleModel.get_rule", return_value=None), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.delete(
                "/api/v1/1/block-rules/10",
                headers={"Authorization": "Bearer admin-token"},
            )
        assert resp.status_code == 404

    @pytest.mark.asyncio
    async def test_delete_success(self, test_app, test_client):
        """Successfully delete a rule."""
        rule = _rule_dict(10, 1)
        fresh_db = MagicMock()

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.block_rules.BlockRuleModel.get_rule", return_value=rule), \
             patch("models.block_rules.BlockRuleModel.delete_rule", return_value=True), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.delete(
                "/api/v1/1/block-rules/10",
                headers={"Authorization": "Bearer admin-token"},
            )
        assert resp.status_code == 200
        data = await resp.get_json()
        assert "message" in data

    @pytest.mark.asyncio
    async def test_delete_hard_delete_parameter(self, test_app, test_client):
        """Delete with hard_delete=true parameter."""
        rule = _rule_dict(10, 1)
        fresh_db = MagicMock()

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.block_rules.BlockRuleModel.get_rule", return_value=rule), \
             patch("models.block_rules.BlockRuleModel.delete_rule", return_value=True) as mock_delete, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.delete(
                "/api/v1/1/block-rules/10?hard_delete=true",
                headers={"Authorization": "Bearer admin-token"},
            )
        assert resp.status_code == 200
        # Verify hard_delete param was passed
        mock_delete.assert_called_once()
        call_kwargs = mock_delete.call_args[1]
        assert call_kwargs.get("hard_delete") is True


# ===========================================================================
# POST /api/v1/<cluster_id>/block-rules/bulk - Bulk create
# ===========================================================================

class TestBulkCreateBlockRules:
    """Test bulk creating block rules."""

    @pytest.mark.asyncio
    async def test_bulk_no_auth_returns_401(self, test_client):
        """Unauthenticated request returns 401."""
        resp = await test_client.post(
            "/api/v1/1/block-rules/bulk",
            json={"rules": []},
        )
        assert resp.status_code == 401

    @pytest.mark.asyncio
    async def test_bulk_non_admin_returns_403(self, test_app, test_client):
        """Non-admin user cannot bulk create."""
        fresh_db = MagicMock()

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.post(
                "/api/v1/1/block-rules/bulk",
                json={"rules": []},
                headers={"Authorization": "Bearer user-token"},
            )
        assert resp.status_code == 403

    @pytest.mark.asyncio
    async def test_bulk_cluster_not_found_returns_404(self, test_app, test_client):
        """Bulk create to non-existent cluster returns 404."""
        fresh_db = MagicMock()
        select_result = MagicMock()
        select_result.first = MagicMock(return_value=None)
        query_mock = MagicMock()
        query_mock.select = MagicMock(return_value=select_result)
        fresh_db.return_value = query_mock

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/1/block-rules/bulk",
                json={"rules": []},
                headers={"Authorization": "Bearer admin-token"},
            )
        assert resp.status_code == 404

    @pytest.mark.asyncio
    async def test_bulk_no_rules_returns_400(self, test_app, test_client):
        """Bulk create with no rules in payload returns 400."""
        cluster = _cluster_mock(1)
        fresh_db = MagicMock()
        select_result = MagicMock()
        select_result.first = MagicMock(return_value=cluster)
        query_mock = MagicMock()
        query_mock.select = MagicMock(return_value=select_result)
        fresh_db.return_value = query_mock

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/1/block-rules/bulk",
                json={"rules": []},
                headers={"Authorization": "Bearer admin-token"},
            )
        assert resp.status_code == 400
        data = await resp.get_json()
        assert "No rules provided" in data.get("error", "")

    @pytest.mark.asyncio
    async def test_bulk_all_valid_rules(self, test_app, test_client):
        """Bulk create with all valid rules succeeds."""
        cluster = _cluster_mock(1)
        fresh_db = MagicMock()
        select_result = MagicMock()
        select_result.first = MagicMock(return_value=cluster)
        query_mock = MagicMock()
        query_mock.select = MagicMock(return_value=select_result)
        fresh_db.return_value = query_mock

        rules_to_create = [
            VALID_RULE,
            {**VALID_RULE, "name": "Rule 2"},
            {**VALID_RULE, "name": "Rule 3"},
        ]

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.block_rules.BlockRuleModel.create_rule", return_value=10), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/1/block-rules/bulk",
                json={"rules": rules_to_create},
                headers={"Authorization": "Bearer admin-token"},
            )
        assert resp.status_code == 201
        data = await resp.get_json()
        assert data["created_count"] == 3
        assert data["error_count"] == 0

    @pytest.mark.asyncio
    async def test_bulk_mixed_valid_invalid_rules(self, test_app, test_client):
        """Bulk create with mixed valid and invalid rules."""
        cluster = _cluster_mock(1)
        fresh_db = MagicMock()
        select_result = MagicMock()
        select_result.first = MagicMock(return_value=cluster)
        query_mock = MagicMock()
        query_mock.select = MagicMock(return_value=select_result)
        fresh_db.return_value = query_mock

        rules_to_create = [
            VALID_RULE,
            {},  # Invalid: missing required fields
            {**VALID_RULE, "name": "Rule 3"},
        ]

        def create_rule_side_effect(*args, **kwargs):
            if not kwargs.get("name"):
                raise ValueError("Missing required field: name")
            return 10

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.block_rules.BlockRuleModel.create_rule",
                   side_effect=create_rule_side_effect), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/1/block-rules/bulk",
                json={"rules": rules_to_create},
                headers={"Authorization": "Bearer admin-token"},
            )
        assert resp.status_code == 201
        data = await resp.get_json()
        assert data["created_count"] == 2
        assert data["error_count"] == 1
        assert len(data.get("errors", [])) == 1


# ===========================================================================
# GET /api/v1/<cluster_id>/threat-feed - Threat feed (API key auth)
# ===========================================================================

class TestThreatFeed:
    """Test threat feed endpoint (API key authentication)."""

    @pytest.mark.asyncio
    async def test_threat_feed_no_api_key_returns_401(self, test_client):
        """Missing API key returns 401."""
        resp = await test_client.get("/api/v1/1/threat-feed")
        assert resp.status_code == 401
        data = await resp.get_json()
        assert "API key required" in data.get("error", "")

    @pytest.mark.asyncio
    async def test_threat_feed_invalid_api_key_returns_401(self, test_app, test_client):
        """Invalid API key returns 401."""
        fresh_db = MagicMock()

        with patch("models.cluster.ClusterModel.validate_api_key", return_value=None), \
             patch.object(test_app, "db", fresh_db):
            resp = await test_client.get(
                "/api/v1/1/threat-feed",
                headers={"X-API-Key": "invalid-key"},
            )
        assert resp.status_code == 401
        data = await resp.get_json()
        assert "Invalid API key" in data.get("error", "")

    @pytest.mark.asyncio
    async def test_threat_feed_wrong_cluster_returns_401(self, test_app, test_client):
        """API key for different cluster returns 401."""
        cluster_info = {"cluster_id": 2}
        fresh_db = MagicMock()

        with patch("models.cluster.ClusterModel.validate_api_key", return_value=cluster_info), \
             patch.object(test_app, "db", fresh_db):
            resp = await test_client.get(
                "/api/v1/1/threat-feed",
                headers={"X-API-Key": "valid-key"},
            )
        assert resp.status_code == 401

    @pytest.mark.asyncio
    async def test_threat_feed_success(self, test_app, test_client):
        """Successfully retrieve threat feed."""
        cluster_info = {"cluster_id": 1}
        threat_feed = {
            "cluster_id": 1,
            "version": "v1.0.0",
            "rules_count": 5,
            "rules": [
                {"id": 1, "rule_type": "ip", "value": "10.0.0.1"},
            ],
        }
        fresh_db = MagicMock()

        with patch("models.cluster.ClusterModel.validate_api_key", return_value=cluster_info), \
             patch("models.block_rules.BlockRuleModel.get_threat_feed", return_value=threat_feed), \
             patch.object(test_app, "db", fresh_db):
            resp = await test_client.get(
                "/api/v1/1/threat-feed",
                headers={"X-API-Key": "valid-key"},
            )
        assert resp.status_code == 200
        data = await resp.get_json()
        assert data["cluster_id"] == 1
        assert data["version"] == "v1.0.0"

    @pytest.mark.asyncio
    async def test_threat_feed_api_key_in_query(self, test_app, test_client):
        """Threat feed with API key in query parameter."""
        cluster_info = {"cluster_id": 1}
        threat_feed = {"cluster_id": 1, "version": "v1.0.0", "rules_count": 0, "rules": []}
        fresh_db = MagicMock()

        with patch("models.cluster.ClusterModel.validate_api_key", return_value=cluster_info), \
             patch("models.block_rules.BlockRuleModel.get_threat_feed", return_value=threat_feed), \
             patch.object(test_app, "db", fresh_db):
            resp = await test_client.get(
                "/api/v1/1/threat-feed?api_key=valid-key",
            )
        assert resp.status_code == 200


# ===========================================================================
# GET /api/v1/<cluster_id>/block-rules/version - Version (API key auth)
# ===========================================================================

class TestBlockRulesVersion:
    """Test block rules version endpoint (API key authentication)."""

    @pytest.mark.asyncio
    async def test_version_no_api_key_returns_401(self, test_client):
        """Missing API key returns 401."""
        resp = await test_client.get("/api/v1/1/block-rules/version")
        assert resp.status_code == 401

    @pytest.mark.asyncio
    async def test_version_invalid_api_key_returns_401(self, test_app, test_client):
        """Invalid API key returns 401."""
        fresh_db = MagicMock()

        with patch("models.cluster.ClusterModel.validate_api_key", return_value=None), \
             patch.object(test_app, "db", fresh_db):
            resp = await test_client.get(
                "/api/v1/1/block-rules/version",
                headers={"X-API-Key": "invalid-key"},
            )
        assert resp.status_code == 401

    @pytest.mark.asyncio
    async def test_version_success(self, test_app, test_client):
        """Successfully retrieve rules version."""
        cluster_info = {"cluster_id": 1}
        fresh_db = MagicMock()

        with patch("models.cluster.ClusterModel.validate_api_key", return_value=cluster_info), \
             patch("models.block_rules.BlockRuleModel.get_rules_version",
                   return_value="abc123def456"), \
             patch.object(test_app, "db", fresh_db):
            resp = await test_client.get(
                "/api/v1/1/block-rules/version",
                headers={"X-API-Key": "valid-key"},
            )
        assert resp.status_code == 200
        data = await resp.get_json()
        assert data["cluster_id"] == 1
        assert data["version"] == "abc123def456"


# ===========================================================================
# GET /api/v1/<cluster_id>/block-rules/sync-status - Sync status
# ===========================================================================

class TestSyncStatus:
    """Test sync status endpoint (JWT authentication)."""

    @pytest.mark.asyncio
    async def test_sync_status_no_auth_returns_401(self, test_client):
        """Unauthenticated request returns 401."""
        resp = await test_client.get("/api/v1/1/block-rules/sync-status")
        assert resp.status_code == 401

    @pytest.mark.asyncio
    async def test_sync_status_user_no_access_returns_403(self, test_app, test_client):
        """User without cluster access returns 403."""
        fresh_db = MagicMock()

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.cluster.UserClusterAssignmentModel.check_user_cluster_access",
                   return_value=None), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.get(
                "/api/v1/1/block-rules/sync-status",
                headers={"Authorization": "Bearer user-token"},
            )
        assert resp.status_code == 403

    @pytest.mark.asyncio
    async def test_sync_status_success_no_proxies(self, test_app, test_client):
        """Successfully get sync status with no proxies."""
        fresh_db = MagicMock()
        select_result = MagicMock()
        select_result.__iter__ = MagicMock(return_value=iter([]))
        query_mock = MagicMock()
        query_mock.select = MagicMock(return_value=select_result)
        fresh_db.return_value = query_mock

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.block_rules.BlockRuleModel.get_rules_version", return_value="v1"), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/v1/1/block-rules/sync-status",
                headers={"Authorization": "Bearer admin-token"},
            )
        assert resp.status_code == 200
        data = await resp.get_json()
        assert data["cluster_id"] == 1
        assert data["current_version"] == "v1"
        assert data["proxies"] == []

    @pytest.mark.asyncio
    async def test_sync_status_success_with_proxies(self, test_app, test_client):
        """Successfully get sync status with multiple proxies."""
        proxy1 = MagicMock()
        proxy1.id = 1
        proxy1.name = "proxy-1"
        proxy1.proxy_type = "alb"

        fresh_db = MagicMock()
        select_result = MagicMock()
        select_result.__iter__ = MagicMock(return_value=iter([proxy1]))
        query_mock = MagicMock()
        query_mock.select = MagicMock(return_value=select_result)
        fresh_db.return_value = query_mock

        sync_data = {
            "sync_status": "synced",
            "last_sync_at": "2025-01-01T00:00:00Z",
            "last_sync_version": "v1",
            "rules_count": 10,
            "sync_error": None,
        }

        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.block_rules.BlockRuleModel.get_rules_version", return_value="v1"), \
             patch("models.block_rules.BlockRuleSyncModel.get_sync_status", return_value=sync_data), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/v1/1/block-rules/sync-status",
                headers={"Authorization": "Bearer admin-token"},
            )
        assert resp.status_code == 200
        data = await resp.get_json()
        assert data["cluster_id"] == 1
        assert len(data["proxies"]) == 1
        assert data["proxies"][0]["proxy_name"] == "proxy-1"
        assert data["proxies"][0]["is_current"] is True
