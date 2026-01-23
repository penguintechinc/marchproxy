"""
Block rules API Blueprint for MarchProxy Manager (Quart)

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from quart import Blueprint, request, current_app, jsonify
from pydantic import ValidationError
import logging
from datetime import datetime
from models.block_rules import (
    BlockRuleModel,
    BlockRuleSyncModel,
    CreateBlockRuleRequest,
    UpdateBlockRuleRequest,
    BlockRuleResponse,
)
from models.cluster import ClusterModel, UserClusterAssignmentModel
from middleware.auth import require_auth

logger = logging.getLogger(__name__)

block_rules_bp = Blueprint("block_rules", __name__, url_prefix="/api/v1/clusters")


@block_rules_bp.route("/<int:cluster_id>/block-rules", methods=["GET", "POST"])
@require_auth()
async def manage_block_rules(user_data, cluster_id):
    """List or create block rules for a cluster"""
    db = current_app.db
    user = user_data

    if request.method == "GET":
        # Check access to cluster
        if not user["is_admin"]:
            user_role = UserClusterAssignmentModel.check_user_cluster_access(
                db, user["user_id"], cluster_id
            )
            if not user_role:
                return jsonify({"error": "Access denied to cluster"}), 403

        # Verify cluster exists
        cluster = (
            db((db.clusters.id == cluster_id) & (db.clusters.is_active == True)).select().first()
        )
        if not cluster:
            return jsonify({"error": "Cluster not found"}), 404

        # Get query parameters
        rule_type = request.args.get("rule_type")
        layer = request.args.get("layer")
        proxy_type = request.args.get("proxy_type")
        include_inactive = request.args.get("include_inactive", "false").lower() == "true"

        rules = BlockRuleModel.list_rules(
            db,
            cluster_id,
            include_inactive=include_inactive,
            rule_type=rule_type,
            layer=layer,
            proxy_type=proxy_type,
        )

        return (
            jsonify({"cluster_id": cluster_id, "rules_count": len(rules), "rules": rules}),
            200,
        )

    elif request.method == "POST":
        # Check authentication - admin required
        if not user["is_admin"]:
            return jsonify({"error": "Admin access required"}), 403

        # Verify cluster exists
        cluster = (
            db((db.clusters.id == cluster_id) & (db.clusters.is_active == True)).select().first()
        )
        if not cluster:
            return jsonify({"error": "Cluster not found"}), 404

        try:
            data_json = await request.get_json()
            data = CreateBlockRuleRequest(**data_json)
        except ValidationError as e:
            return jsonify({"error": "Validation error", "details": str(e)}), 400

        try:
            rule_id = BlockRuleModel.create_rule(
                db,
                cluster_id,
                name=data.name,
                rule_type=data.rule_type,
                layer=data.layer,
                value=data.value,
                created_by=user["user_id"],
                description=data.description,
                ports=data.ports,
                protocols=data.protocols,
                wildcard=data.wildcard,
                match_type=data.match_type,
                action=data.action,
                priority=data.priority,
                apply_to_alb=data.apply_to_alb,
                apply_to_nlb=data.apply_to_nlb,
                apply_to_egress=data.apply_to_egress,
                expires_at=data.expires_at,
            )

            rule = BlockRuleModel.get_rule(db, rule_id)
            return (
                jsonify({"message": "Block rule created successfully", "rule": rule}),
                201,
            )

        except ValueError as e:
            return jsonify({"error": str(e)}), 400
        except Exception as e:
            logger.error(f"Block rule creation failed: {e}")
            return jsonify({"error": "Failed to create block rule"}), 500


@block_rules_bp.route(
    "/<int:cluster_id>/block-rules/<int:rule_id>", methods=["GET", "PUT", "DELETE"]
)
@require_auth()
async def manage_single_block_rule(user_data, cluster_id, rule_id):
    """Get, update, or delete a specific block rule"""
    db = current_app.db
    user = user_data

    if request.method == "GET":
        # Check access to cluster
        if not user["is_admin"]:
            user_role = UserClusterAssignmentModel.check_user_cluster_access(
                db, user["user_id"], cluster_id
            )
            if not user_role:
                return jsonify({"error": "Access denied to cluster"}), 403

        rule = BlockRuleModel.get_rule(db, rule_id)
        if not rule or rule["cluster_id"] != int(cluster_id):
            return jsonify({"error": "Block rule not found"}), 404

        return jsonify(rule), 200

    elif request.method == "PUT":
        # Check authentication - admin required
        if not user["is_admin"]:
            return jsonify({"error": "Admin access required"}), 403

        # Verify rule exists and belongs to cluster
        rule = BlockRuleModel.get_rule(db, rule_id)
        if not rule or rule["cluster_id"] != int(cluster_id):
            return jsonify({"error": "Block rule not found"}), 404

        try:
            data_json = await request.get_json()
            data = UpdateBlockRuleRequest(**data_json)
        except ValidationError as e:
            return jsonify({"error": "Validation error", "details": str(e)}), 400

        try:
            success = BlockRuleModel.update_rule(db, rule_id, **data.dict(exclude_unset=True))
            if not success:
                return jsonify({"error": "Failed to update block rule"}), 500

            updated_rule = BlockRuleModel.get_rule(db, rule_id)
            return (
                jsonify({"message": "Block rule updated successfully", "rule": updated_rule}),
                200,
            )

        except ValueError as e:
            return jsonify({"error": str(e)}), 400
        except Exception as e:
            logger.error(f"Block rule update failed: {e}")
            return jsonify({"error": "Failed to update block rule"}), 500

    elif request.method == "DELETE":
        # Check authentication - admin required
        if not user["is_admin"]:
            return jsonify({"error": "Admin access required"}), 403

        # Verify rule exists and belongs to cluster
        rule = BlockRuleModel.get_rule(db, rule_id)
        if not rule or rule["cluster_id"] != int(cluster_id):
            return jsonify({"error": "Block rule not found"}), 404

        # Check for hard delete parameter
        hard_delete = request.args.get("hard_delete", "false").lower() == "true"

        try:
            success = BlockRuleModel.delete_rule(db, rule_id, hard_delete=hard_delete)
            if not success:
                return jsonify({"error": "Failed to delete block rule"}), 500

            return jsonify({"message": "Block rule deleted successfully"}), 200

        except Exception as e:
            logger.error(f"Block rule deletion failed: {e}")
            return jsonify({"error": "Failed to delete block rule"}), 500


@block_rules_bp.route("/<int:cluster_id>/block-rules/bulk", methods=["POST"])
@require_auth(admin_required=True)
async def bulk_create_block_rules(user_data, cluster_id):
    """Bulk create block rules (for threat feed imports)"""
    db = current_app.db
    user = user_data

    # Verify cluster exists
    cluster = db((db.clusters.id == cluster_id) & (db.clusters.is_active == True)).select().first()
    if not cluster:
        return jsonify({"error": "Cluster not found"}), 404

    try:
        data_json = await request.get_json()
    except Exception as e:
        return jsonify({"error": "Invalid JSON"}), 400

    rules_data = data_json.get("rules", [])
    if not rules_data:
        return jsonify({"error": "No rules provided"}), 400

    created_count = 0
    errors = []

    for idx, rule_data in enumerate(rules_data):
        try:
            data = CreateBlockRuleRequest(**rule_data)
            BlockRuleModel.create_rule(
                db,
                cluster_id,
                name=data.name,
                rule_type=data.rule_type,
                layer=data.layer,
                value=data.value,
                created_by=user["user_id"],
                description=data.description,
                ports=data.ports,
                protocols=data.protocols,
                wildcard=data.wildcard,
                match_type=data.match_type,
                action=data.action,
                priority=data.priority,
                apply_to_alb=data.apply_to_alb,
                apply_to_nlb=data.apply_to_nlb,
                apply_to_egress=data.apply_to_egress,
                expires_at=data.expires_at,
                source=rule_data.get("source", "api"),
                source_feed_name=rule_data.get("source_feed_name"),
            )
            created_count += 1
        except (ValidationError, ValueError) as e:
            errors.append(
                {
                    "index": idx,
                    "rule": rule_data.get("name", f"rule_{idx}"),
                    "error": str(e),
                }
            )

    return (
        jsonify(
            {
                "message": f"Created {created_count} block rules",
                "created_count": created_count,
                "error_count": len(errors),
                "errors": errors if errors else None,
            }
        ),
        201,
    )


@block_rules_bp.route("/<int:cluster_id>/threat-feed", methods=["GET"])
async def get_threat_feed(cluster_id):
    """Get threat feed for proxy consumption (API key authenticated)"""
    db = current_app.db

    # This endpoint uses API key authentication instead of JWT
    api_key = request.headers.get("X-API-Key") or request.args.get("api_key")
    if not api_key:
        return jsonify({"error": "API key required"}), 401

    # Validate cluster API key
    cluster_info = ClusterModel.validate_api_key(db, api_key)
    if not cluster_info or cluster_info["cluster_id"] != int(cluster_id):
        return jsonify({"error": "Invalid API key for cluster"}), 401

    # Get query parameters
    proxy_type = request.args.get("proxy_type")  # alb, nlb, egress
    since_version = request.args.get("since_version")  # For delta updates

    # Get threat feed
    threat_feed = BlockRuleModel.get_threat_feed(
        db, cluster_id, proxy_type=proxy_type, since_version=since_version
    )

    # Record proxy sync if proxy_id provided
    proxy_id = request.args.get("proxy_id")
    if proxy_id:
        try:
            BlockRuleSyncModel.update_sync_status(
                db,
                int(proxy_id),
                version=threat_feed["version"],
                rules_count=threat_feed["rules_count"],
                status="synced",
            )
        except Exception as e:
            logger.warning(f"Failed to update sync status: {e}")

    return jsonify(threat_feed), 200


@block_rules_bp.route("/<int:cluster_id>/block-rules/version", methods=["GET"])
async def get_rules_version(cluster_id):
    """Get current rules version hash for change detection"""
    db = current_app.db

    # This endpoint uses API key authentication
    api_key = request.headers.get("X-API-Key") or request.args.get("api_key")
    if not api_key:
        return jsonify({"error": "API key required"}), 401

    # Validate cluster API key
    cluster_info = ClusterModel.validate_api_key(db, api_key)
    if not cluster_info or cluster_info["cluster_id"] != int(cluster_id):
        return jsonify({"error": "Invalid API key for cluster"}), 401

    proxy_type = request.args.get("proxy_type")
    version = BlockRuleModel.get_rules_version(db, cluster_id, proxy_type)

    return jsonify({"cluster_id": cluster_id, "version": version}), 200


@block_rules_bp.route("/<int:cluster_id>/block-rules/sync-status", methods=["GET"])
@require_auth()
async def get_sync_status(user_data, cluster_id):
    """Get sync status for all proxies in cluster"""
    db = current_app.db
    user = user_data

    # Check access to cluster
    if not user["is_admin"]:
        user_role = UserClusterAssignmentModel.check_user_cluster_access(
            db, user["user_id"], cluster_id
        )
        if not user_role:
            return jsonify({"error": "Access denied to cluster"}), 403

    # Get all proxies in cluster and their sync status
    proxies = db(
        (db.proxy_servers.cluster_id == cluster_id) & (db.proxy_servers.status == "active")
    ).select()

    current_version = BlockRuleModel.get_rules_version(db, cluster_id)

    sync_statuses = []
    for proxy in proxies:
        sync = BlockRuleSyncModel.get_sync_status(db, proxy.id)
        sync_statuses.append(
            {
                "proxy_id": proxy.id,
                "proxy_name": proxy.name,
                "proxy_type": proxy.proxy_type,
                "sync_status": sync["sync_status"] if sync else "never_synced",
                "last_sync_at": sync["last_sync_at"] if sync else None,
                "last_sync_version": sync["last_sync_version"] if sync else None,
                "is_current": (sync["last_sync_version"] == current_version if sync else False),
                "rules_count": sync["rules_count"] if sync else 0,
                "sync_error": sync["sync_error"] if sync else None,
            }
        )

    return (
        jsonify(
            {
                "cluster_id": cluster_id,
                "current_version": current_version,
                "proxies": sync_statuses,
            }
        ),
        200,
    )
