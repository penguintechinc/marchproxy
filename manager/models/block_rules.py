"""
Block rules models for MarchProxy Manager

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import hashlib
import json
import re
from datetime import datetime
from typing import Any, Dict, List, Literal, Optional

from pydal import DAL, Field
from pydantic import BaseModel, validator


class BlockRuleModel:
    """Block rule model for threat intelligence and traffic control"""

    VALID_RULE_TYPES = ["ip", "cidr", "domain", "url_pattern", "port"]
    VALID_LAYERS = ["L4", "L7"]
    # Actions:
    # - 'deny': Active rejection - sends ICMP unreachable/TCP RST/HTTP 403 (for egress)
    # - 'drop': Silent drop - no response sent (for ingress security)
    # - 'allow': Explicit allow (whitelist)
    # - 'log': Log only, don't block
    VALID_ACTIONS = ["deny", "drop", "allow", "log"]
    VALID_MATCH_TYPES = ["exact", "prefix", "suffix", "regex", "contains"]
    VALID_SOURCES = ["manual", "threat_feed", "api"]

    @staticmethod
    def define_table(db: DAL):
        """Define block_rules table in database"""
        return db.define_table(
            "block_rules",
            Field("name", type="string", required=True, length=100),
            Field("description", type="text"),
            Field("cluster_id", type="reference clusters", required=True),
            Field("rule_type", type="string", required=True, length=20),
            Field("layer", type="string", required=True, length=10),
            Field("value", type="string", required=True, length=255),
            Field("ports", type="json"),
            Field("protocols", type="json"),
            Field("wildcard", type="boolean", default=False),
            Field("match_type", type="string", default="exact", length=20),
            Field("action", type="string", default="deny", length=20),
            Field("priority", type="integer", default=1000),
            Field("apply_to_alb", type="boolean", default=True),
            Field("apply_to_nlb", type="boolean", default=True),
            Field("apply_to_egress", type="boolean", default=True),
            Field("source", type="string", default="manual", length=20),
            Field("source_feed_name", type="string", length=100),
            Field("expires_at", type="datetime"),
            Field("is_active", type="boolean", default=True),
            Field("created_by", type="reference users"),
            Field("created_at", type="datetime", default=datetime.utcnow),
            Field("updated_at", type="datetime", update=datetime.utcnow),
        )

    @staticmethod
    def validate_ip(value: str) -> bool:
        """Validate IPv4 or IPv6 address"""
        import ipaddress

        try:
            ipaddress.ip_address(value)
            return True
        except ValueError:
            return False

    @staticmethod
    def validate_cidr(value: str) -> bool:
        """Validate CIDR notation"""
        import ipaddress

        try:
            ipaddress.ip_network(value, strict=False)
            return True
        except ValueError:
            return False

    @staticmethod
    def validate_domain(value: str) -> bool:
        """Validate domain name"""
        # Allow wildcards like *.example.com
        pattern = r"^(\*\.)?([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$"
        return bool(re.match(pattern, value))

    @staticmethod
    def validate_regex(pattern: str) -> bool:
        """Validate regex pattern"""
        try:
            re.compile(pattern)
            return True
        except re.error:
            return False

    @staticmethod
    def create_rule(
        db: DAL,
        cluster_id: int,
        name: str,
        rule_type: str,
        layer: str,
        value: str,
        created_by: int = None,
        description: str = None,
        ports: list = None,
        protocols: list = None,
        wildcard: bool = False,
        match_type: str = "exact",
        action: str = "deny",
        priority: int = 1000,
        apply_to_alb: bool = True,
        apply_to_nlb: bool = True,
        apply_to_egress: bool = True,
        source: str = "manual",
        source_feed_name: str = None,
        expires_at: datetime = None,
    ) -> int:
        """
        Create a new block rule.

        Action types:
        - 'deny': Active rejection with response (ICMP unreachable/TCP RST/HTTP 403)
                  Recommended for egress proxies so services know they're blocked.
        - 'drop': Silent drop with no response.
                  Recommended for ingress proxies to not reveal information to attackers.
        - 'allow': Explicit whitelist entry.
        - 'log': Log the traffic but don't block it.

        For egress proxy rules, 'deny' is the default so blocked services get feedback.
        For ingress proxy rules (ALB/NLB), 'drop' is typically preferred for security.
        """

        # Validate rule type
        if rule_type not in BlockRuleModel.VALID_RULE_TYPES:
            raise ValueError(f"Invalid rule_type: {rule_type}")

        # Validate layer
        if layer not in BlockRuleModel.VALID_LAYERS:
            raise ValueError(f"Invalid layer: {layer}")

        # Validate value based on rule type
        if rule_type == "ip":
            if not BlockRuleModel.validate_ip(value):
                raise ValueError(f"Invalid IP address: {value}")
        elif rule_type == "cidr":
            if not BlockRuleModel.validate_cidr(value):
                raise ValueError(f"Invalid CIDR: {value}")
        elif rule_type == "domain":
            if not BlockRuleModel.validate_domain(value):
                raise ValueError(f"Invalid domain: {value}")
        elif rule_type == "url_pattern" and match_type == "regex":
            if not BlockRuleModel.validate_regex(value):
                raise ValueError(f"Invalid regex pattern: {value}")

        # Set default protocols if not provided
        if protocols is None:
            protocols = ["tcp", "udp"]

        rule_id = db.block_rules.insert(
            name=name,
            description=description,
            cluster_id=cluster_id,
            rule_type=rule_type,
            layer=layer,
            value=value,
            ports=json.dumps(ports) if ports else None,
            protocols=json.dumps(protocols),
            wildcard=wildcard,
            match_type=match_type,
            action=action,
            priority=priority,
            apply_to_alb=apply_to_alb,
            apply_to_nlb=apply_to_nlb,
            apply_to_egress=apply_to_egress,
            source=source,
            source_feed_name=source_feed_name,
            expires_at=expires_at,
            created_by=created_by,
        )

        return rule_id

    @staticmethod
    def get_rule(db: DAL, rule_id: int) -> Optional[Dict[str, Any]]:
        """Get a block rule by ID"""
        rule = db.block_rules[rule_id]
        if not rule:
            return None

        return BlockRuleModel._format_rule(rule)

    @staticmethod
    def list_rules(
        db: DAL,
        cluster_id: int,
        include_inactive: bool = False,
        rule_type: str = None,
        layer: str = None,
        proxy_type: str = None,
    ) -> List[Dict[str, Any]]:
        """List block rules for a cluster"""
        query = db.block_rules.cluster_id == cluster_id

        if not include_inactive:
            query &= db.block_rules.is_active == True  # noqa: E712
            # Exclude expired rules
            query &= (db.block_rules.expires_at == None) | (  # noqa: E711
                db.block_rules.expires_at > datetime.utcnow()
            )

        if rule_type:
            query &= db.block_rules.rule_type == rule_type

        if layer:
            query &= db.block_rules.layer == layer

        if proxy_type:
            if proxy_type == "alb":
                query &= db.block_rules.apply_to_alb == True  # noqa: E712
            elif proxy_type == "nlb":
                query &= db.block_rules.apply_to_nlb == True  # noqa: E712
            elif proxy_type == "egress":
                query &= db.block_rules.apply_to_egress == True  # noqa: E712

        rules = db(query).select(orderby=db.block_rules.priority)
        return [BlockRuleModel._format_rule(rule) for rule in rules]

    @staticmethod
    def update_rule(db: DAL, rule_id: int, **kwargs) -> bool:
        """Update a block rule"""
        rule = db.block_rules[rule_id]
        if not rule:
            return False

        # Filter only valid update fields
        valid_fields = [
            "name",
            "description",
            "value",
            "ports",
            "protocols",
            "wildcard",
            "match_type",
            "action",
            "priority",
            "apply_to_alb",
            "apply_to_nlb",
            "apply_to_egress",
            "is_active",
            "expires_at",
        ]

        update_data = {"updated_at": datetime.utcnow()}
        for field in valid_fields:
            if field in kwargs and kwargs[field] is not None:
                if field in ["ports", "protocols"]:
                    update_data[field] = json.dumps(kwargs[field])
                else:
                    update_data[field] = kwargs[field]

        rule.update_record(**update_data)
        return True

    @staticmethod
    def delete_rule(db: DAL, rule_id: int, hard_delete: bool = False) -> bool:
        """Delete a block rule (soft delete by default)"""
        rule = db.block_rules[rule_id]
        if not rule:
            return False

        if hard_delete:
            db(db.block_rules.id == rule_id).delete()
        else:
            rule.update_record(is_active=False, updated_at=datetime.utcnow())

        return True

    @staticmethod
    def increment_hit_count(db: DAL, rule_id: int) -> bool:
        """Increment rule hit count and update last_hit timestamp"""
        rule = db.block_rules[rule_id]
        if not rule:
            return False

        rule.update_record(hit_count=rule.hit_count + 1, last_hit=datetime.utcnow())
        return True

    @staticmethod
    def get_rules_version(db: DAL, cluster_id: int, proxy_type: str = None) -> str:
        """Get SHA256 hash of current rules for change detection"""
        rules = BlockRuleModel.list_rules(db, cluster_id, proxy_type=proxy_type)
        rules_json = json.dumps(rules, sort_keys=True, default=str)
        return hashlib.sha256(rules_json.encode()).hexdigest()

    @staticmethod
    def get_threat_feed(
        db: DAL, cluster_id: int, proxy_type: str = None, since_version: str = None
    ) -> Dict[str, Any]:
        """Get threat feed data for proxy consumption"""
        rules = BlockRuleModel.list_rules(db, cluster_id, proxy_type=proxy_type)
        current_version = BlockRuleModel.get_rules_version(db, cluster_id, proxy_type)

        # Group rules by layer and type for efficient proxy processing
        l4_rules = {"ip": [], "cidr": [], "port": []}
        l7_rules = {"domain": [], "url_pattern": []}

        for rule in rules:
            if rule["layer"] == "L4":
                if rule["rule_type"] in l4_rules:
                    l4_rules[rule["rule_type"]].append(rule)
            else:  # L7
                if rule["rule_type"] in l7_rules:
                    l7_rules[rule["rule_type"]].append(rule)

        return {
            "version": current_version,
            "generated_at": datetime.utcnow().isoformat(),
            "cluster_id": cluster_id,
            "rules_count": len(rules),
            "l4_rules": l4_rules,
            "l7_rules": l7_rules,
            "full_rules": rules if since_version != current_version else None,
        }

    @staticmethod
    def _format_rule(rule) -> Dict[str, Any]:
        """Format rule row as dictionary"""
        ports = None
        if rule.ports:
            try:
                ports = json.loads(rule.ports) if isinstance(rule.ports, str) else rule.ports
            except (json.JSONDecodeError, TypeError):
                ports = rule.ports

        protocols = ["tcp", "udp"]
        if rule.protocols:
            try:
                protocols = (
                    json.loads(rule.protocols)
                    if isinstance(rule.protocols, str)
                    else rule.protocols
                )
            except (json.JSONDecodeError, TypeError):
                protocols = rule.protocols

        return {
            "id": rule.id,
            "name": rule.name,
            "description": rule.description,
            "cluster_id": rule.cluster_id,
            "rule_type": rule.rule_type,
            "layer": rule.layer,
            "value": rule.value,
            "ports": ports,
            "protocols": protocols,
            "wildcard": rule.wildcard,
            "match_type": rule.match_type,
            "action": rule.action,
            "priority": rule.priority,
            "apply_to_alb": rule.apply_to_alb,
            "apply_to_nlb": rule.apply_to_nlb,
            "apply_to_egress": rule.apply_to_egress,
            "source": rule.source,
            "source_feed_name": rule.source_feed_name,
            "is_active": rule.is_active,
            "expires_at": rule.expires_at.isoformat() if rule.expires_at else None,
            "hit_count": rule.hit_count,
            "last_hit": rule.last_hit.isoformat() if rule.last_hit else None,
            "created_at": rule.created_at.isoformat() if rule.created_at else None,
            "updated_at": rule.updated_at.isoformat() if rule.updated_at else None,
        }


class BlockRuleSyncModel:
    """Block rule sync tracking model"""

    @staticmethod
    def update_sync_status(
        db: DAL,
        proxy_id: int,
        version: str,
        rules_count: int,
        status: str = "synced",
        error: str = None,
    ) -> bool:
        """Update sync status for a proxy"""
        existing = db(db.block_rule_sync.proxy_id == proxy_id).select().first()

        if existing:
            existing.update_record(
                last_sync_version=version,
                last_sync_at=datetime.utcnow(),
                rules_count=rules_count,
                sync_status=status,
                sync_error=error,
            )
        else:
            db.block_rule_sync.insert(
                proxy_id=proxy_id,
                last_sync_version=version,
                rules_count=rules_count,
                sync_status=status,
                sync_error=error,
            )

        return True

    @staticmethod
    def get_sync_status(db: DAL, proxy_id: int) -> Optional[Dict[str, Any]]:
        """Get sync status for a proxy"""
        sync = db(db.block_rule_sync.proxy_id == proxy_id).select().first()
        if not sync:
            return None

        return {
            "proxy_id": sync.proxy_id,
            "last_sync_version": sync.last_sync_version,
            "last_sync_at": (sync.last_sync_at.isoformat() if sync.last_sync_at else None),
            "rules_count": sync.rules_count,
            "sync_status": sync.sync_status,
            "sync_error": sync.sync_error,
        }


# Pydantic models for request/response validation
class CreateBlockRuleRequest(BaseModel):
    """
    Request model for creating block rules.

    Action types:
    - 'deny': Active rejection with response (ICMP unreachable/TCP RST/HTTP 403)
              Use for egress proxies so services know they're blocked.
    - 'drop': Silent drop with no response.
              Use for ingress proxies (ALB/NLB) to not reveal info to attackers.
    - 'allow': Explicit whitelist entry.
    - 'log': Log the traffic but don't block it.
    """

    name: str
    description: Optional[str] = None
    rule_type: Literal["ip", "cidr", "domain", "url_pattern", "port"]
    layer: Literal["L4", "L7"]
    value: str
    ports: Optional[List[int]] = None
    protocols: Optional[List[str]] = None
    wildcard: bool = False
    match_type: Literal["exact", "prefix", "suffix", "regex", "contains"] = "exact"
    # 'deny' = active rejection (egress), 'drop' = silent drop (ingress)
    action: Literal["deny", "drop", "allow", "log"] = "deny"
    priority: int = 1000
    apply_to_alb: bool = True
    apply_to_nlb: bool = True
    apply_to_egress: bool = True
    expires_at: Optional[datetime] = None

    @validator("name")
    def validate_name(cls, v):
        if len(v) < 3:
            raise ValueError("Rule name must be at least 3 characters long")
        if len(v) > 255:
            raise ValueError("Rule name must be at most 255 characters")
        return v

    @validator("priority")
    def validate_priority(cls, v):
        if v < 1 or v > 100000:
            raise ValueError("Priority must be between 1 and 100000")
        return v

    @validator("layer")
    def validate_layer_for_rule_type(cls, v, values):
        rule_type = values.get("rule_type")
        # L7 rules must use L7 layer
        if rule_type in ["domain", "url_pattern"] and v != "L7":
            raise ValueError(f"{rule_type} rules must use L7 layer")
        # ip, cidr, port rules work with L4
        if rule_type in ["ip", "cidr", "port"] and v not in ["L4", "L7"]:
            raise ValueError(f"{rule_type} rules require L4 or L7 layer")
        return v

    @validator("action")
    def validate_action_recommendation(cls, v, values):
        """
        Validate action and provide guidance on best practices.
        - Egress proxy: 'deny' recommended (services get feedback)
        - Ingress proxy (ALB/NLB): 'drop' recommended (security)
        """
        # This validator just ensures valid action, actual enforcement
        # happens at the proxy level based on proxy type
        return v


class UpdateBlockRuleRequest(BaseModel):
    name: Optional[str] = None
    description: Optional[str] = None
    value: Optional[str] = None
    ports: Optional[List[int]] = None
    protocols: Optional[List[str]] = None
    wildcard: Optional[bool] = None
    match_type: Optional[Literal["exact", "prefix", "suffix", "regex", "contains"]] = None
    action: Optional[Literal["deny", "drop", "allow", "log"]] = None
    priority: Optional[int] = None
    apply_to_alb: Optional[bool] = None
    apply_to_nlb: Optional[bool] = None
    apply_to_egress: Optional[bool] = None
    is_active: Optional[bool] = None
    expires_at: Optional[datetime] = None


class BlockRuleResponse(BaseModel):
    id: int
    name: str
    description: Optional[str]
    cluster_id: int
    rule_type: str
    layer: str
    value: str
    ports: Optional[List[int]]
    protocols: List[str]
    wildcard: bool
    match_type: str
    action: str
    priority: int
    apply_to_alb: bool
    apply_to_nlb: bool
    apply_to_egress: bool
    source: str
    source_feed_name: Optional[str]
    is_active: bool
    expires_at: Optional[str]
    hit_count: int
    last_hit: Optional[str]
    created_at: str
    updated_at: Optional[str]


class ThreatFeedResponse(BaseModel):
    version: str
    generated_at: str
    cluster_id: int
    rules_count: int
    l4_rules: Dict[str, List[Dict[str, Any]]]
    l7_rules: Dict[str, List[Dict[str, Any]]]
    full_rules: Optional[List[Dict[str, Any]]]
