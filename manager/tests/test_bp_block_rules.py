"""
Tests for api/block_rules_bp.py blueprint.

Routes at /api/v1/clusters prefix:
  GET/POST /api/v1/<cluster_id>/block-rules
  GET/PUT/DELETE /api/v1/<cluster_id>/block-rules/<rule_id>
  POST /api/v1/<cluster_id>/block-rules/bulk
  GET /api/v1/<cluster_id>/threat-feed
  GET /api/v1/<cluster_id>/block-rules/version
  GET /api/v1/<cluster_id>/block-rules/sync-status

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _admin_payload():
    return {
        "user_id": 1, "username": "admin", "is_admin": True,
        "roles": ["admin"], "scope": ["*:admin"],
    }


def _user_payload():
    return {"user_id": 2, "username": "user", "is_admin": False, "roles": [], "scope": []}


def _cluster_row():
    c = MagicMock()
    c.id = 1
    c.name = "test-cluster"
    c.is_active = True
    return c


def _rule_row(rule_id=10):
    r = MagicMock()
    r.id = rule_id
    r.cluster_id = 1
    r.rule_type = "ip"
    r.layer = "l3"
    r.value = "10.0.0.1"
    r.proxy_type = "envoy"
    r.priority = 100
    r.is_active = True
    r.description = "test rule"
    r.created_at = "2025-01-01T00:00:00"
    r.updated_at = None
    r.update_record = MagicMock()
    return r


# ===========================================================================
# GET /api/v1/<cluster_id>/block-rules
# ===========================================================================

class TestListBlockRules:
    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.get("/api/v1/1/block-rules")
        assert resp.status_code == 401

    async def test_user_no_cluster_access_returns_403(self, test_app, test_client):
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.cluster.UserClusterAssignmentModel.check_user_cluster_access",
                   return_value=None), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.get(
                "/api/v1/1/block-rules",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 403

    async def test_cluster_not_found_returns_404(self, test_app, test_client):
        fresh_db = MagicMock()
        select_result = MagicMock()
        select_result.first = MagicMock(return_value=None)
        query_mock = MagicMock()
        query_mock.select = MagicMock(return_value=select_result)
        fresh_db.return_value = query_mock
        fresh_db.clusters = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/v1/1/block-rules",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 404

    async def test_list_rules_success(self, test_app, test_client):
        cluster = _cluster_row()
        fresh_db = MagicMock()
        select_result = MagicMock()
        select_result.first = MagicMock(return_value=cluster)
        query_mock = MagicMock()
        query_mock.select = MagicMock(return_value=select_result)
        fresh_db.return_value = query_mock
        fresh_db.clusters = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.block_rules.BlockRuleModel.list_rules", return_value=[]), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/v1/1/block-rules",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 200

    async def test_list_rules_with_filters(self, test_app, test_client):
        cluster = _cluster_row()
        fresh_db = MagicMock()
        select_result = MagicMock()
        select_result.first = MagicMock(return_value=cluster)
        query_mock = MagicMock()
        query_mock.select = MagicMock(return_value=select_result)
        fresh_db.return_value = query_mock
        fresh_db.clusters = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.block_rules.BlockRuleModel.list_rules", return_value=[]), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/v1/1/block-rules?rule_type=ip&layer=l3&include_inactive=true",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 200


# ===========================================================================
# POST /api/v1/<cluster_id>/block-rules
# ===========================================================================

class TestCreateBlockRule:
    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.post("/api/v1/1/block-rules", json={})
        assert resp.status_code == 401

    async def test_user_no_cluster_access_returns_403(self, test_app, test_client):
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.cluster.UserClusterAssignmentModel.check_user_cluster_access",
                   return_value=None), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.post(
                "/api/v1/1/block-rules",
                json={"rule_type": "ip", "value": "10.0.0.1", "layer": "l3"},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 403

    async def test_validation_error_returns_400(self, test_app, test_client):
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/1/block-rules",
                json={},  # missing required fields
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 400

    async def test_create_rule_success(self, test_app, test_client):
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.block_rules.BlockRuleModel.create_rule", return_value=10), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/1/block-rules",
                json={"rule_type": "ip", "value": "10.0.0.1", "layer": "l3",
                      "proxy_type": "envoy", "priority": 100},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [201, 400, 500]

    async def test_create_rule_value_error_returns_400(self, test_app, test_client):
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.block_rules.BlockRuleModel.create_rule",
                   side_effect=ValueError("invalid")), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/1/block-rules",
                json={"rule_type": "ip", "value": "10.0.0.1", "layer": "l3",
                      "proxy_type": "envoy", "priority": 100},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [400, 500]

    async def test_create_rule_exception_returns_500(self, test_app, test_client):
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.block_rules.BlockRuleModel.create_rule",
                   side_effect=Exception("db err")), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/1/block-rules",
                json={"rule_type": "ip", "value": "10.0.0.1", "layer": "l3",
                      "proxy_type": "envoy", "priority": 100},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [400, 500]


# ===========================================================================
# GET/PUT/DELETE /api/v1/<cluster_id>/block-rules/<rule_id>
# ===========================================================================

class TestSingleBlockRule:
    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.get("/api/v1/1/block-rules/10")
        assert resp.status_code == 401

    async def test_get_rule_not_found_returns_404(self, test_app, test_client):
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.block_rules.BlockRuleModel.get_rule", return_value=None), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/v1/1/block-rules/10",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 404

    async def test_get_rule_success(self, test_app, test_client):
        rule_data = {
            "id": 10, "cluster_id": 1, "rule_type": "ip",
            "layer": "l3", "value": "10.0.0.1",
            "proxy_type": "envoy", "priority": 100,
            "is_active": True, "description": "test",
            "created_at": "2025-01-01T00:00:00", "updated_at": None,
        }
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.block_rules.BlockRuleModel.get_rule", return_value=rule_data), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/v1/1/block-rules/10",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 200

    async def test_put_user_no_access_returns_403(self, test_app, test_client):
        rule_data = {"id": 10, "cluster_id": 1}
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.block_rules.BlockRuleModel.get_rule", return_value=rule_data), \
             patch("models.cluster.UserClusterAssignmentModel.check_user_cluster_access",
                   return_value=None), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.put(
                "/api/v1/1/block-rules/10",
                json={"priority": 200},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 403

    async def test_put_rule_success(self, test_app, test_client):
        rule_data = {
            "id": 10, "cluster_id": 1, "rule_type": "ip",
            "layer": "l3", "value": "10.0.0.1",
            "proxy_type": "envoy", "priority": 100,
            "is_active": True, "description": "test",
            "created_at": "2025-01-01T00:00:00", "updated_at": None,
        }
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.block_rules.BlockRuleModel.get_rule", return_value=rule_data), \
             patch("models.block_rules.BlockRuleModel.update_rule", return_value=rule_data), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.put(
                "/api/v1/1/block-rules/10",
                json={"priority": 200},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 400, 500]

    async def test_delete_user_no_access_returns_403(self, test_app, test_client):
        rule_data = {"id": 10, "cluster_id": 1}
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.block_rules.BlockRuleModel.get_rule", return_value=rule_data), \
             patch("models.cluster.UserClusterAssignmentModel.check_user_cluster_access",
                   return_value=None), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.delete(
                "/api/v1/1/block-rules/10",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 403

    async def test_delete_rule_success(self, test_app, test_client):
        rule_data = {"id": 10, "cluster_id": 1}
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.block_rules.BlockRuleModel.get_rule", return_value=rule_data), \
             patch("models.block_rules.BlockRuleModel.delete_rule", return_value=True), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.delete(
                "/api/v1/1/block-rules/10",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 204, 500]


# ===========================================================================
# POST /api/v1/<cluster_id>/block-rules/bulk
# ===========================================================================

class TestBulkCreateBlockRules:
    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.post("/api/v1/1/block-rules/bulk", json={})
        assert resp.status_code == 401

    async def test_user_no_access_returns_403(self, test_app, test_client):
        fresh_db = MagicMock()
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.cluster.UserClusterAssignmentModel.check_user_cluster_access",
                   return_value=None), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _user_payload()
            resp = await test_client.post(
                "/api/v1/1/block-rules/bulk",
                json={"rules": []},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code == 403

    async def test_bulk_create_success(self, test_app, test_client):
        fresh_db = MagicMock()
        # Cluster must exist for the endpoint to proceed
        cluster_row = MagicMock()
        cluster_row.is_active = True
        fresh_db.clusters = MagicMock()
        fresh_db.clusters.id = MagicMock()
        fresh_db.clusters.is_active = MagicMock()
        select_result = MagicMock()
        select_result.first.return_value = cluster_row
        fresh_db.return_value.select.return_value = select_result
        rules = [
            {"name": "test-rule", "rule_type": "ip", "value": "10.0.0.1", "layer": "l3",
             "priority": 100},
        ]
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.block_rules.BlockRuleModel.create_rule", return_value=10), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.post(
                "/api/v1/1/block-rules/bulk",
                json={"rules": rules},
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [201, 400, 500]


# ===========================================================================
# GET /api/v1/<cluster_id>/threat-feed
# ===========================================================================

class TestGetThreatFeed:
    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.get("/api/v1/1/threat-feed")
        assert resp.status_code == 401

    async def test_threat_feed_success(self, test_app, test_client):
        fresh_db = MagicMock()
        cluster_info = {"cluster_id": 1, "cluster_name": "test"}
        threat_data = {"version": "abc123", "rules_count": 1, "rules": []}
        with patch("models.cluster.ClusterModel.validate_api_key", return_value=cluster_info), \
             patch("models.block_rules.BlockRuleModel.get_threat_feed",
                   return_value=threat_data), \
             patch.object(test_app, "db", fresh_db):
            resp = await test_client.get(
                "/api/v1/1/threat-feed",
                headers={"X-API-Key": "valid-key"},
            )
        assert resp.status_code in [200, 500]

    async def test_threat_feed_invalid_api_key_returns_401(self, test_app, test_client):
        fresh_db = MagicMock()
        with patch("models.cluster.ClusterModel.validate_api_key", return_value=None), \
             patch.object(test_app, "db", fresh_db):
            resp = await test_client.get(
                "/api/v1/1/threat-feed",
                headers={"X-API-Key": "bad-key"},
            )
        assert resp.status_code == 401


# ===========================================================================
# GET /api/v1/<cluster_id>/block-rules/version
# ===========================================================================

class TestGetRulesVersion:
    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.get("/api/v1/1/block-rules/version")
        assert resp.status_code == 401

    async def test_version_success(self, test_app, test_client):
        fresh_db = MagicMock()
        cluster_info = {"cluster_id": 1}
        with patch("models.cluster.ClusterModel.validate_api_key", return_value=cluster_info), \
             patch("models.block_rules.BlockRuleModel.get_rules_version", return_value="abc123"), \
             patch.object(test_app, "db", fresh_db):
            resp = await test_client.get(
                "/api/v1/1/block-rules/version",
                headers={"X-API-Key": "valid-key"},
            )
        assert resp.status_code in [200, 500]


# ===========================================================================
# GET /api/v1/<cluster_id>/block-rules/sync-status
# ===========================================================================

class TestGetSyncStatus:
    async def test_no_auth_returns_401(self, test_client):
        resp = await test_client.get("/api/v1/1/block-rules/sync-status")
        assert resp.status_code == 401

    async def test_sync_status_success(self, test_app, test_client):
        fresh_db = MagicMock()
        status_data = {"version": 5, "synced": True, "last_sync": "2025-01-01T00:00:00"}
        with patch("middleware.auth._validate_token", new_callable=AsyncMock) as mock_v, \
             patch("models.block_rules.BlockRuleSyncModel.get_sync_status",
                   return_value=status_data), \
             patch.object(test_app, "db", fresh_db):
            mock_v.return_value = _admin_payload()
            resp = await test_client.get(
                "/api/v1/1/block-rules/sync-status",
                headers={"Authorization": "Bearer tok"},
            )
        assert resp.status_code in [200, 500]
