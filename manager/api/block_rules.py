"""
Block rules API endpoints for MarchProxy Manager

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from py4web import request, response
from py4web.utils.cors import enable_cors
from pydantic import ValidationError
import logging
from datetime import datetime
from ..models.block_rules import (
    BlockRuleModel, BlockRuleSyncModel,
    CreateBlockRuleRequest, UpdateBlockRuleRequest, BlockRuleResponse
)
from ..models.cluster import ClusterModel, UserClusterAssignmentModel
from .auth import _check_auth

logger = logging.getLogger(__name__)


def block_rules_api(db, jwt_manager):
    """Block rules management API endpoints"""

    @enable_cors()
    def list_block_rules(cluster_id):
        """List all block rules for a cluster"""
        if request.method == 'GET':
            # Check authentication
            auth_result = _check_auth(db, jwt_manager)
            if 'error' in auth_result:
                response.status = auth_result['status']
                return auth_result

            user = auth_result['user']

            # Check access to cluster
            if not user['is_admin']:
                user_role = UserClusterAssignmentModel.check_user_cluster_access(
                    db, user['id'], cluster_id
                )
                if not user_role:
                    response.status = 403
                    return {"error": "Access denied to cluster"}

            # Verify cluster exists
            cluster = db((db.clusters.id == cluster_id) & (db.clusters.is_active == True)).select().first()
            if not cluster:
                response.status = 404
                return {"error": "Cluster not found"}

            # Get query parameters
            rule_type = request.query.get('rule_type')
            layer = request.query.get('layer')
            proxy_type = request.query.get('proxy_type')
            include_inactive = request.query.get('include_inactive', 'false').lower() == 'true'

            rules = BlockRuleModel.list_rules(
                db, cluster_id,
                include_inactive=include_inactive,
                rule_type=rule_type,
                layer=layer,
                proxy_type=proxy_type
            )

            return {
                "cluster_id": cluster_id,
                "rules_count": len(rules),
                "rules": rules
            }

        else:
            response.status = 405
            return {"error": "Method not allowed"}

    @enable_cors()
    def create_block_rule(cluster_id):
        """Create a new block rule"""
        if request.method == 'POST':
            # Check authentication - admin required
            auth_result = _check_auth(db, jwt_manager, admin_required=True)
            if 'error' in auth_result:
                response.status = auth_result['status']
                return auth_result

            # Verify cluster exists
            cluster = db((db.clusters.id == cluster_id) & (db.clusters.is_active == True)).select().first()
            if not cluster:
                response.status = 404
                return {"error": "Cluster not found"}

            try:
                data = CreateBlockRuleRequest(**request.json)
            except ValidationError as e:
                response.status = 400
                return {"error": "Validation error", "details": str(e)}

            try:
                rule_id = BlockRuleModel.create_rule(
                    db, cluster_id,
                    name=data.name,
                    rule_type=data.rule_type,
                    layer=data.layer,
                    value=data.value,
                    created_by=auth_result['user']['id'],
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
                    expires_at=data.expires_at
                )

                rule = BlockRuleModel.get_rule(db, rule_id)
                return {
                    "message": "Block rule created successfully",
                    "rule": rule
                }

            except ValueError as e:
                response.status = 400
                return {"error": str(e)}
            except Exception as e:
                logger.error(f"Block rule creation failed: {e}")
                response.status = 500
                return {"error": "Failed to create block rule"}

        else:
            response.status = 405
            return {"error": "Method not allowed"}

    @enable_cors()
    def get_block_rule(cluster_id, rule_id):
        """Get a specific block rule"""
        if request.method == 'GET':
            # Check authentication
            auth_result = _check_auth(db, jwt_manager)
            if 'error' in auth_result:
                response.status = auth_result['status']
                return auth_result

            user = auth_result['user']

            # Check access to cluster
            if not user['is_admin']:
                user_role = UserClusterAssignmentModel.check_user_cluster_access(
                    db, user['id'], cluster_id
                )
                if not user_role:
                    response.status = 403
                    return {"error": "Access denied to cluster"}

            rule = BlockRuleModel.get_rule(db, rule_id)
            if not rule or rule['cluster_id'] != int(cluster_id):
                response.status = 404
                return {"error": "Block rule not found"}

            return rule

        else:
            response.status = 405
            return {"error": "Method not allowed"}

    @enable_cors()
    def update_block_rule(cluster_id, rule_id):
        """Update a block rule"""
        if request.method == 'PUT':
            # Check authentication - admin required
            auth_result = _check_auth(db, jwt_manager, admin_required=True)
            if 'error' in auth_result:
                response.status = auth_result['status']
                return auth_result

            # Verify rule exists and belongs to cluster
            rule = BlockRuleModel.get_rule(db, rule_id)
            if not rule or rule['cluster_id'] != int(cluster_id):
                response.status = 404
                return {"error": "Block rule not found"}

            try:
                data = UpdateBlockRuleRequest(**request.json)
            except ValidationError as e:
                response.status = 400
                return {"error": "Validation error", "details": str(e)}

            try:
                success = BlockRuleModel.update_rule(db, rule_id, **data.dict(exclude_unset=True))
                if not success:
                    response.status = 500
                    return {"error": "Failed to update block rule"}

                updated_rule = BlockRuleModel.get_rule(db, rule_id)
                return {
                    "message": "Block rule updated successfully",
                    "rule": updated_rule
                }

            except ValueError as e:
                response.status = 400
                return {"error": str(e)}
            except Exception as e:
                logger.error(f"Block rule update failed: {e}")
                response.status = 500
                return {"error": "Failed to update block rule"}

        else:
            response.status = 405
            return {"error": "Method not allowed"}

    @enable_cors()
    def delete_block_rule(cluster_id, rule_id):
        """Delete a block rule"""
        if request.method == 'DELETE':
            # Check authentication - admin required
            auth_result = _check_auth(db, jwt_manager, admin_required=True)
            if 'error' in auth_result:
                response.status = auth_result['status']
                return auth_result

            # Verify rule exists and belongs to cluster
            rule = BlockRuleModel.get_rule(db, rule_id)
            if not rule or rule['cluster_id'] != int(cluster_id):
                response.status = 404
                return {"error": "Block rule not found"}

            # Check for hard delete parameter
            hard_delete = request.query.get('hard_delete', 'false').lower() == 'true'

            try:
                success = BlockRuleModel.delete_rule(db, rule_id, hard_delete=hard_delete)
                if not success:
                    response.status = 500
                    return {"error": "Failed to delete block rule"}

                return {"message": "Block rule deleted successfully"}

            except Exception as e:
                logger.error(f"Block rule deletion failed: {e}")
                response.status = 500
                return {"error": "Failed to delete block rule"}

        else:
            response.status = 405
            return {"error": "Method not allowed"}

    @enable_cors()
    def get_threat_feed(cluster_id):
        """Get threat feed for proxy consumption (API key authenticated)"""
        if request.method == 'GET':
            # This endpoint uses API key authentication instead of JWT
            api_key = request.environ.get('HTTP_X_API_KEY') or request.query.get('api_key')
            if not api_key:
                response.status = 401
                return {"error": "API key required"}

            # Validate cluster API key
            cluster_info = ClusterModel.validate_api_key(db, api_key)
            if not cluster_info or cluster_info['cluster_id'] != int(cluster_id):
                response.status = 401
                return {"error": "Invalid API key for cluster"}

            # Get query parameters
            proxy_type = request.query.get('proxy_type')  # alb, nlb, egress
            since_version = request.query.get('since_version')  # For delta updates

            # Get threat feed
            threat_feed = BlockRuleModel.get_threat_feed(
                db, cluster_id,
                proxy_type=proxy_type,
                since_version=since_version
            )

            # Record proxy sync if proxy_id provided
            proxy_id = request.query.get('proxy_id')
            if proxy_id:
                try:
                    BlockRuleSyncModel.update_sync_status(
                        db, int(proxy_id),
                        version=threat_feed['version'],
                        rules_count=threat_feed['rules_count'],
                        status='synced'
                    )
                except Exception as e:
                    logger.warning(f"Failed to update sync status: {e}")

            return threat_feed

        else:
            response.status = 405
            return {"error": "Method not allowed"}

    @enable_cors()
    def get_rules_version(cluster_id):
        """Get current rules version hash for change detection"""
        if request.method == 'GET':
            # This endpoint uses API key authentication
            api_key = request.environ.get('HTTP_X_API_KEY') or request.query.get('api_key')
            if not api_key:
                response.status = 401
                return {"error": "API key required"}

            # Validate cluster API key
            cluster_info = ClusterModel.validate_api_key(db, api_key)
            if not cluster_info or cluster_info['cluster_id'] != int(cluster_id):
                response.status = 401
                return {"error": "Invalid API key for cluster"}

            proxy_type = request.query.get('proxy_type')
            version = BlockRuleModel.get_rules_version(db, cluster_id, proxy_type)

            return {
                "cluster_id": cluster_id,
                "version": version
            }

        else:
            response.status = 405
            return {"error": "Method not allowed"}

    @enable_cors()
    def bulk_create_block_rules(cluster_id):
        """Bulk create block rules (for threat feed imports)"""
        if request.method == 'POST':
            # Check authentication - admin required
            auth_result = _check_auth(db, jwt_manager, admin_required=True)
            if 'error' in auth_result:
                response.status = auth_result['status']
                return auth_result

            # Verify cluster exists
            cluster = db((db.clusters.id == cluster_id) & (db.clusters.is_active == True)).select().first()
            if not cluster:
                response.status = 404
                return {"error": "Cluster not found"}

            rules_data = request.json.get('rules', [])
            if not rules_data:
                response.status = 400
                return {"error": "No rules provided"}

            created_count = 0
            errors = []

            for idx, rule_data in enumerate(rules_data):
                try:
                    data = CreateBlockRuleRequest(**rule_data)
                    BlockRuleModel.create_rule(
                        db, cluster_id,
                        name=data.name,
                        rule_type=data.rule_type,
                        layer=data.layer,
                        value=data.value,
                        created_by=auth_result['user']['id'],
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
                        source=rule_data.get('source', 'api'),
                        source_feed_name=rule_data.get('source_feed_name')
                    )
                    created_count += 1
                except (ValidationError, ValueError) as e:
                    errors.append({
                        "index": idx,
                        "rule": rule_data.get('name', f'rule_{idx}'),
                        "error": str(e)
                    })

            return {
                "message": f"Created {created_count} block rules",
                "created_count": created_count,
                "error_count": len(errors),
                "errors": errors if errors else None
            }

        else:
            response.status = 405
            return {"error": "Method not allowed"}

    @enable_cors()
    def get_sync_status(cluster_id):
        """Get sync status for all proxies in cluster"""
        if request.method == 'GET':
            # Check authentication
            auth_result = _check_auth(db, jwt_manager)
            if 'error' in auth_result:
                response.status = auth_result['status']
                return auth_result

            user = auth_result['user']

            # Check access to cluster
            if not user['is_admin']:
                user_role = UserClusterAssignmentModel.check_user_cluster_access(
                    db, user['id'], cluster_id
                )
                if not user_role:
                    response.status = 403
                    return {"error": "Access denied to cluster"}

            # Get all proxies in cluster and their sync status
            proxies = db(
                (db.proxy_servers.cluster_id == cluster_id) &
                (db.proxy_servers.status == 'active')
            ).select()

            current_version = BlockRuleModel.get_rules_version(db, cluster_id)

            sync_statuses = []
            for proxy in proxies:
                sync = BlockRuleSyncModel.get_sync_status(db, proxy.id)
                sync_statuses.append({
                    'proxy_id': proxy.id,
                    'proxy_name': proxy.name,
                    'proxy_type': proxy.proxy_type,
                    'sync_status': sync['sync_status'] if sync else 'never_synced',
                    'last_sync_at': sync['last_sync_at'] if sync else None,
                    'last_sync_version': sync['last_sync_version'] if sync else None,
                    'is_current': sync['last_sync_version'] == current_version if sync else False,
                    'rules_count': sync['rules_count'] if sync else 0,
                    'sync_error': sync['sync_error'] if sync else None
                })

            return {
                "cluster_id": cluster_id,
                "current_version": current_version,
                "proxies": sync_statuses
            }

        else:
            response.status = 405
            return {"error": "Method not allowed"}

    # Return API endpoints
    return {
        'list_block_rules': list_block_rules,
        'create_block_rule': create_block_rule,
        'get_block_rule': get_block_rule,
        'update_block_rule': update_block_rule,
        'delete_block_rule': delete_block_rule,
        'get_threat_feed': get_threat_feed,
        'get_rules_version': get_rules_version,
        'bulk_create_block_rules': bulk_create_block_rules,
        'get_sync_status': get_sync_status
    }
