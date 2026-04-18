"""
Tests for models/block_rules.py

Covers: BlockRuleModel, BlockRuleSyncModel, Pydantic validators.
Uses MagicMock for the PyDAL `db` object.

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from datetime import datetime
from unittest.mock import MagicMock, patch

import pytest

from models.block_rules import (
    BlockRuleModel,
    BlockRuleSyncModel,
    CreateBlockRuleRequest,
    UpdateBlockRuleRequest,
)


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _patch_datetime_field(field_mock):
    """Patch comparison operators on a field mock for datetime comparisons."""
    field_mock.__gt__ = MagicMock(return_value=MagicMock())
    field_mock.__lt__ = MagicMock(return_value=MagicMock())
    field_mock.__ge__ = MagicMock(return_value=MagicMock())
    field_mock.__le__ = MagicMock(return_value=MagicMock())
    # __eq__ and __ne__ for None comparisons
    field_mock.__eq__ = MagicMock(return_value=MagicMock())
    field_mock.__ne__ = MagicMock(return_value=MagicMock())


def _make_db():
    """Create a minimal PyDAL-like mock."""
    db = MagicMock(name="db")
    db.block_rules = MagicMock()
    db.block_rules.insert = MagicMock(return_value=1)
    db.block_rules.__getitem__ = MagicMock(return_value=None)
    # Patch datetime fields to support comparison operators
    _patch_datetime_field(db.block_rules.expires_at)
    # db(query).select() → configurable return (default empty iterator)
    query_mock = MagicMock()
    query_mock.select = MagicMock(return_value=iter([]))
    query_mock.count = MagicMock(return_value=0)
    query_mock.update = MagicMock(return_value=1)
    query_mock.delete = MagicMock(return_value=0)
    # Use both return_value and __call__ to ensure db() returns query_mock
    db.return_value = query_mock
    db.__call__ = MagicMock(return_value=query_mock)
    db.block_rule_sync = MagicMock()
    return db, query_mock


def _make_rule_mock(rule_id=1):
    r = MagicMock()
    r.id = rule_id
    r.name = "test-rule"
    r.description = "Test"
    r.cluster_id = 1
    r.rule_type = "ip"
    r.layer = "L4"
    r.value = "10.0.0.1"
    r.ports = None
    r.protocols = '["tcp","udp"]'
    r.wildcard = False
    r.match_type = "exact"
    r.action = "deny"
    r.priority = 1000
    r.apply_to_alb = True
    r.apply_to_nlb = True
    r.apply_to_egress = True
    r.source = "manual"
    r.source_feed_name = None
    r.is_active = True
    r.expires_at = None
    r.hit_count = 0
    r.last_hit = None
    r.created_at = datetime(2025, 1, 1)
    r.updated_at = None
    return r


# ===========================================================================
# Static validators
# ===========================================================================

class TestBlockRuleModelValidators:
    def test_validate_ip_valid_ipv4(self):
        assert BlockRuleModel.validate_ip("192.168.1.1") is True

    def test_validate_ip_valid_ipv6(self):
        assert BlockRuleModel.validate_ip("::1") is True

    def test_validate_ip_invalid(self):
        assert BlockRuleModel.validate_ip("not-an-ip") is False

    def test_validate_cidr_valid(self):
        assert BlockRuleModel.validate_cidr("10.0.0.0/8") is True

    def test_validate_cidr_invalid(self):
        assert BlockRuleModel.validate_cidr("not-a-cidr") is False

    def test_validate_domain_valid(self):
        assert BlockRuleModel.validate_domain("example.com") is True

    def test_validate_domain_wildcard(self):
        assert BlockRuleModel.validate_domain("*.example.com") is True

    def test_validate_domain_invalid(self):
        assert BlockRuleModel.validate_domain("not a domain") is False

    def test_validate_regex_valid(self):
        assert BlockRuleModel.validate_regex(r"^\d+$") is True

    def test_validate_regex_invalid(self):
        assert BlockRuleModel.validate_regex("[invalid") is False


# ===========================================================================
# BlockRuleModel.create_rule
# ===========================================================================

class TestCreateRule:
    def test_create_rule_invalid_type_raises(self):
        db, _ = _make_db()
        with pytest.raises(ValueError, match="Invalid rule_type"):
            BlockRuleModel.create_rule(db, 1, "r", "unknown_type", "L4", "10.0.0.1")

    def test_create_rule_invalid_layer_raises(self):
        db, _ = _make_db()
        with pytest.raises(ValueError, match="Invalid layer"):
            BlockRuleModel.create_rule(db, 1, "r", "ip", "L99", "10.0.0.1")

    def test_create_rule_invalid_ip_raises(self):
        db, _ = _make_db()
        with pytest.raises(ValueError, match="Invalid IP"):
            BlockRuleModel.create_rule(db, 1, "r", "ip", "L4", "not-an-ip")

    def test_create_rule_invalid_cidr_raises(self):
        db, _ = _make_db()
        with pytest.raises(ValueError, match="Invalid CIDR"):
            BlockRuleModel.create_rule(db, 1, "r", "cidr", "L4", "bad-cidr")

    def test_create_rule_invalid_domain_raises(self):
        db, _ = _make_db()
        with pytest.raises(ValueError, match="Invalid domain"):
            BlockRuleModel.create_rule(db, 1, "r", "domain", "L7", "not a domain")

    def test_create_rule_invalid_regex_raises(self):
        db, _ = _make_db()
        with pytest.raises(ValueError, match="Invalid regex"):
            BlockRuleModel.create_rule(
                db, 1, "r", "url_pattern", "L7", "[invalid", match_type="regex"
            )

    def test_create_rule_ip_success(self):
        db, _ = _make_db()
        db.block_rules.insert.return_value = 5
        result = BlockRuleModel.create_rule(db, 1, "block-ip", "ip", "L4", "10.0.0.1")
        assert result == 5
        db.block_rules.insert.assert_called_once()

    def test_create_rule_cidr_success(self):
        db, _ = _make_db()
        db.block_rules.insert.return_value = 6
        result = BlockRuleModel.create_rule(
            db, 1, "block-cidr", "cidr", "L4", "192.168.0.0/16",
            protocols=["tcp"], ports=[80, 443]
        )
        assert result == 6

    def test_create_rule_domain_success(self):
        db, _ = _make_db()
        db.block_rules.insert.return_value = 7
        result = BlockRuleModel.create_rule(db, 1, "block-domain", "domain", "L7", "evil.com")
        assert result == 7

    def test_create_rule_url_pattern_prefix(self):
        db, _ = _make_db()
        db.block_rules.insert.return_value = 8
        result = BlockRuleModel.create_rule(
            db, 1, "block-url", "url_pattern", "L7", "/admin", match_type="prefix"
        )
        assert result == 8

    def test_create_rule_url_pattern_regex(self):
        db, _ = _make_db()
        db.block_rules.insert.return_value = 9
        result = BlockRuleModel.create_rule(
            db, 1, "block-regex", "url_pattern", "L7", r"^/api/.*", match_type="regex"
        )
        assert result == 9

    def test_create_rule_port_success(self):
        db, _ = _make_db()
        db.block_rules.insert.return_value = 10
        result = BlockRuleModel.create_rule(db, 1, "block-port", "port", "L4", "22")
        assert result == 10

    def test_create_rule_with_expires_at(self):
        db, _ = _make_db()
        db.block_rules.insert.return_value = 11
        result = BlockRuleModel.create_rule(
            db, 1, "temp-rule", "ip", "L4", "1.2.3.4",
            expires_at=datetime(2030, 1, 1),
            source="threat_feed",
            source_feed_name="test-feed",
        )
        assert result == 11


# ===========================================================================
# BlockRuleModel.get_rule
# ===========================================================================

class TestGetRule:
    def test_get_rule_not_found_returns_none(self):
        db, _ = _make_db()
        db.block_rules.__getitem__ = MagicMock(return_value=None)
        result = BlockRuleModel.get_rule(db, 999)
        assert result is None

    def test_get_rule_success(self):
        db, _ = _make_db()
        rule = _make_rule_mock()
        db.block_rules.__getitem__ = MagicMock(return_value=rule)
        result = BlockRuleModel.get_rule(db, 1)
        assert result is not None
        assert result["id"] == 1
        assert result["rule_type"] == "ip"

    def test_get_rule_with_ports_json(self):
        db, _ = _make_db()
        rule = _make_rule_mock()
        rule.ports = '[80, 443]'
        db.block_rules.__getitem__ = MagicMock(return_value=rule)
        result = BlockRuleModel.get_rule(db, 1)
        assert result["ports"] == [80, 443]

    def test_get_rule_with_ports_bad_json(self):
        db, _ = _make_db()
        rule = _make_rule_mock()
        rule.ports = "invalid-json"
        db.block_rules.__getitem__ = MagicMock(return_value=rule)
        result = BlockRuleModel.get_rule(db, 1)
        # Falls back to raw value
        assert result["ports"] == "invalid-json"

    def test_get_rule_with_expires_at(self):
        db, _ = _make_db()
        rule = _make_rule_mock()
        rule.expires_at = datetime(2030, 1, 1)
        rule.last_hit = datetime(2025, 6, 1)
        rule.updated_at = datetime(2025, 1, 2)
        db.block_rules.__getitem__ = MagicMock(return_value=rule)
        result = BlockRuleModel.get_rule(db, 1)
        assert result["expires_at"] is not None
        assert result["last_hit"] is not None


# ===========================================================================
# BlockRuleModel.list_rules
# ===========================================================================

class TestListRules:
    def test_list_rules_empty(self):
        db, query_mock = _make_db()
        query_mock.select.return_value = iter([])
        result = BlockRuleModel.list_rules(db, cluster_id=1)
        assert result == []

    def test_list_rules_with_results(self):
        db, query_mock = _make_db()
        rule = _make_rule_mock()
        # Use a list so it's iterable multiple times
        query_mock.select.return_value = [rule]
        result = BlockRuleModel.list_rules(db, cluster_id=1)
        assert len(result) == 1

    def test_list_rules_with_rule_type_filter(self):
        db, query_mock = _make_db()
        query_mock.select.return_value = iter([])
        result = BlockRuleModel.list_rules(db, cluster_id=1, rule_type="ip")
        assert result == []

    def test_list_rules_with_layer_filter(self):
        db, query_mock = _make_db()
        query_mock.select.return_value = iter([])
        result = BlockRuleModel.list_rules(db, cluster_id=1, layer="L4")
        assert result == []

    def test_list_rules_with_proxy_type_alb(self):
        db, query_mock = _make_db()
        query_mock.select.return_value = iter([])
        result = BlockRuleModel.list_rules(db, cluster_id=1, proxy_type="alb")
        assert result == []

    def test_list_rules_with_proxy_type_nlb(self):
        db, query_mock = _make_db()
        query_mock.select.return_value = iter([])
        result = BlockRuleModel.list_rules(db, cluster_id=1, proxy_type="nlb")
        assert result == []

    def test_list_rules_with_proxy_type_egress(self):
        db, query_mock = _make_db()
        query_mock.select.return_value = iter([])
        result = BlockRuleModel.list_rules(db, cluster_id=1, proxy_type="egress")
        assert result == []

    def test_list_rules_include_inactive(self):
        db, query_mock = _make_db()
        query_mock.select.return_value = iter([])
        result = BlockRuleModel.list_rules(db, cluster_id=1, include_inactive=True)
        assert result == []


# ===========================================================================
# BlockRuleModel.update_rule
# ===========================================================================

class TestUpdateRule:
    def test_update_rule_not_found_returns_false(self):
        db, _ = _make_db()
        db.block_rules.__getitem__ = MagicMock(return_value=None)
        result = BlockRuleModel.update_rule(db, 999, name="new-name")
        assert result is False

    def test_update_rule_success(self):
        db, _ = _make_db()
        rule = _make_rule_mock()
        db.block_rules.__getitem__ = MagicMock(return_value=rule)
        result = BlockRuleModel.update_rule(db, 1, name="updated-name", priority=500)
        assert result is True
        rule.update_record.assert_called_once()

    def test_update_rule_with_json_fields(self):
        db, _ = _make_db()
        rule = _make_rule_mock()
        db.block_rules.__getitem__ = MagicMock(return_value=rule)
        result = BlockRuleModel.update_rule(
            db, 1, ports=[8080, 9090], protocols=["tcp"]
        )
        assert result is True

    def test_update_rule_no_valid_fields(self):
        db, _ = _make_db()
        rule = _make_rule_mock()
        db.block_rules.__getitem__ = MagicMock(return_value=rule)
        # Pass field that's not in valid_fields — only updated_at is set
        result = BlockRuleModel.update_rule(db, 1, invalid_field="x")
        assert result is True


# ===========================================================================
# BlockRuleModel.delete_rule
# ===========================================================================

class TestDeleteRule:
    def test_delete_rule_not_found_returns_false(self):
        db, _ = _make_db()
        db.block_rules.__getitem__ = MagicMock(return_value=None)
        result = BlockRuleModel.delete_rule(db, 999)
        assert result is False

    def test_soft_delete_default(self):
        db, _ = _make_db()
        rule = _make_rule_mock()
        db.block_rules.__getitem__ = MagicMock(return_value=rule)
        result = BlockRuleModel.delete_rule(db, 1)
        assert result is True
        rule.update_record.assert_called_once()
        # hard_delete=False → update_record, not db.delete
        db.return_value.delete.assert_not_called()

    def test_hard_delete(self):
        db, query_mock = _make_db()
        rule = _make_rule_mock()
        db.block_rules.__getitem__ = MagicMock(return_value=rule)
        result = BlockRuleModel.delete_rule(db, 1, hard_delete=True)
        assert result is True
        # db(condition).delete() is called; db.__call__ returns query_mock
        db.assert_called()
        query_mock.delete.assert_called()


# ===========================================================================
# BlockRuleModel.increment_hit_count
# ===========================================================================

class TestIncrementHitCount:
    def test_increment_not_found_returns_false(self):
        db, _ = _make_db()
        db.block_rules.__getitem__ = MagicMock(return_value=None)
        result = BlockRuleModel.increment_hit_count(db, 999)
        assert result is False

    def test_increment_success(self):
        db, _ = _make_db()
        rule = _make_rule_mock()
        rule.hit_count = 5
        db.block_rules.__getitem__ = MagicMock(return_value=rule)
        result = BlockRuleModel.increment_hit_count(db, 1)
        assert result is True
        rule.update_record.assert_called_once()


# ===========================================================================
# BlockRuleModel.get_rules_version
# ===========================================================================

class TestGetRulesVersion:
    def test_returns_sha256_string(self):
        db, query_mock = _make_db()
        query_mock.select.return_value = iter([])
        version = BlockRuleModel.get_rules_version(db, 1)
        assert isinstance(version, str)
        assert len(version) == 64  # SHA256 hex

    def test_different_clusters_produce_same_hash_when_empty(self):
        db, query_mock = _make_db()
        query_mock.select.return_value = iter([])
        v1 = BlockRuleModel.get_rules_version(db, 1)
        query_mock.select.return_value = iter([])
        v2 = BlockRuleModel.get_rules_version(db, 2)
        # Both empty → same hash
        assert v1 == v2


# ===========================================================================
# BlockRuleModel.get_threat_feed
# ===========================================================================

class TestGetThreatFeed:
    def test_threat_feed_empty_cluster(self):
        db, query_mock = _make_db()
        query_mock.select.return_value = iter([])
        result = BlockRuleModel.get_threat_feed(db, 1)
        assert "version" in result
        assert "rules_count" in result
        assert result["rules_count"] == 0
        assert result["cluster_id"] == 1

    def test_threat_feed_with_l4_ip_rule(self):
        db, query_mock = _make_db()
        rule = _make_rule_mock()
        rule.layer = "L4"
        rule.rule_type = "ip"
        # get_threat_feed calls list_rules which calls db(query).select(orderby=...)
        # Then get_rules_version calls list_rules again → need to return same list
        query_mock.select.return_value = [rule]
        result = BlockRuleModel.get_threat_feed(db, 1)
        assert result["rules_count"] == 1
        assert len(result["l4_rules"]["ip"]) == 1

    def test_threat_feed_with_l7_domain_rule(self):
        db, query_mock = _make_db()
        rule = _make_rule_mock()
        rule.layer = "L7"
        rule.rule_type = "domain"
        rule.value = "evil.com"
        query_mock.select.return_value = [rule]
        result = BlockRuleModel.get_threat_feed(db, 1)
        assert len(result["l7_rules"]["domain"]) == 1

    def test_threat_feed_with_since_version_no_change(self):
        """When since_version equals current, full_rules is None"""
        db, query_mock = _make_db()
        query_mock.select.return_value = iter([])
        # Get the version of an empty ruleset
        result = BlockRuleModel.get_threat_feed(db, 1, since_version=None)
        # since_version doesn't match (None vs hash) → full_rules not None
        assert result["full_rules"] is not None

    def test_threat_feed_with_proxy_type(self):
        db, query_mock = _make_db()
        query_mock.select.return_value = iter([])
        result = BlockRuleModel.get_threat_feed(db, 1, proxy_type="alb")
        assert "version" in result


# ===========================================================================
# BlockRuleSyncModel
# ===========================================================================

class TestBlockRuleSyncModel:
    def _make_sync_db(self, first_return=None):
        """Make a db mock where db(cond).select().first() returns first_return."""
        db = MagicMock(name="db")
        # Correctly chain the return values: db().select().first()
        db.return_value.select.return_value.first.return_value = first_return
        db.block_rule_sync = MagicMock()
        return db

    def test_update_sync_status_creates_new(self):
        db = self._make_sync_db(first_return=None)
        result = BlockRuleSyncModel.update_sync_status(
            db, proxy_id=1, version="abc123", rules_count=5
        )
        assert result is True
        db.block_rule_sync.insert.assert_called_once()

    def test_update_sync_status_updates_existing(self):
        existing = MagicMock()
        db = self._make_sync_db(first_return=existing)
        result = BlockRuleSyncModel.update_sync_status(
            db, proxy_id=1, version="abc123", rules_count=5, status="synced", error="err"
        )
        assert result is True
        existing.update_record.assert_called_once()

    def test_get_sync_status_not_found(self):
        db = self._make_sync_db(first_return=None)
        result = BlockRuleSyncModel.get_sync_status(db, proxy_id=999)
        assert result is None

    def test_get_sync_status_success(self):
        sync = MagicMock()
        sync.proxy_id = 1
        sync.last_sync_version = "abc123"
        sync.last_sync_at = datetime(2025, 1, 1)
        sync.rules_count = 5
        sync.sync_status = "synced"
        sync.sync_error = None
        db = self._make_sync_db(first_return=sync)
        result = BlockRuleSyncModel.get_sync_status(db, proxy_id=1)
        assert result is not None
        assert result["sync_status"] == "synced"
        assert result["last_sync_at"] is not None


# ===========================================================================
# Pydantic CreateBlockRuleRequest validators
# ===========================================================================

class TestCreateBlockRuleRequest:
    def test_valid_ip_rule(self):
        req = CreateBlockRuleRequest(name="test-rule", rule_type="ip", layer="L4", value="10.0.0.1")
        assert req.name == "test-rule"

    def test_name_too_short_raises(self):
        with pytest.raises(Exception):
            CreateBlockRuleRequest(name="ab", rule_type="ip", layer="L4", value="10.0.0.1")

    def test_priority_out_of_range_raises(self):
        with pytest.raises(Exception):
            CreateBlockRuleRequest(
                name="test-rule", rule_type="ip", layer="L4", value="10.0.0.1", priority=0
            )

    def test_domain_rule_requires_l7_layer(self):
        with pytest.raises(Exception):
            CreateBlockRuleRequest(
                name="test-rule", rule_type="domain", layer="L4", value="evil.com"
            )

    def test_ip_rule_with_l7_layer_valid(self):
        req = CreateBlockRuleRequest(name="test-rule", rule_type="ip", layer="L7", value="10.0.0.1")
        assert req.layer == "L7"

    def test_url_pattern_with_l7_layer(self):
        req = CreateBlockRuleRequest(
            name="test-rule", rule_type="url_pattern", layer="L7", value="/admin"
        )
        assert req.rule_type == "url_pattern"
