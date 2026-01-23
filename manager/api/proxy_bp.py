"""
Proxy registration and management API Blueprint for MarchProxy Manager (Quart)

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from quart import Blueprint, request, current_app, jsonify
from pydantic import ValidationError
import logging
from models.proxy import (
    ProxyServerModel,
    ProxyMetricsModel,
    ProxyRegistrationRequest,
    ProxyHeartbeatRequest,
    ProxyConfigRequest,
    ProxyResponse,
    ProxyStatsResponse,
    ProxyMetricsResponse,
)
from models.cluster import ClusterModel, UserClusterAssignmentModel
from middleware.auth import require_auth

logger = logging.getLogger(__name__)

proxy_bp = Blueprint("proxy", __name__, url_prefix="/api/v1/proxy")


@proxy_bp.route("/register", methods=["POST"])
async def register():
    """Register new proxy server"""
    try:
        data_json = await request.get_json()
        data = ProxyRegistrationRequest(**data_json)
    except ValidationError as e:
        return jsonify({"error": "Validation error", "details": str(e)}), 400

    db = current_app.db

    # Register proxy with cluster API key validation
    proxy_id = ProxyServerModel.register_proxy(
        db,
        name=data.name,
        hostname=data.hostname,
        cluster_api_key=data.cluster_api_key,
        proxy_type=data.proxy_type,
        ip_address=data.ip_address,
        port=data.port,
        version=data.version,
        capabilities=data.capabilities,
    )

    if not proxy_id:
        return (
            jsonify(
                {
                    "error": "Registration failed - invalid API key or proxy limit exceeded"
                }
            ),
            400,
        )

    registered_proxy = db.proxy_servers[proxy_id]
    return (
        jsonify(
            {
                "proxy": ProxyResponse(
                    id=registered_proxy.id,
                    name=registered_proxy.name,
                    hostname=registered_proxy.hostname,
                    ip_address=registered_proxy.ip_address,
                    port=registered_proxy.port,
                    cluster_id=registered_proxy.cluster_id,
                    status=registered_proxy.status,
                    version=registered_proxy.version,
                    license_validated=registered_proxy.license_validated,
                    last_seen=registered_proxy.last_seen,
                    registered_at=registered_proxy.registered_at,
                ).dict(),
                "message": "Proxy registered successfully",
            }
        ),
        201,
    )


@proxy_bp.route("/heartbeat", methods=["POST"])
async def heartbeat():
    """Proxy heartbeat endpoint"""
    try:
        data_json = await request.get_json()
        data = ProxyHeartbeatRequest(**data_json)
    except ValidationError as e:
        return jsonify({"error": "Validation error", "details": str(e)}), 400

    db = current_app.db

    # Update heartbeat
    status_data = {}
    if data.version:
        status_data["version"] = data.version
    if data.capabilities:
        status_data["capabilities"] = data.capabilities
    if data.config_version:
        status_data["config_version"] = data.config_version

    success = ProxyServerModel.update_heartbeat(
        db, data.proxy_name, data.cluster_api_key, status_data
    )

    if not success:
        return (
            jsonify({"error": "Heartbeat failed - invalid API key or proxy not found"}),
            400,
        )

    # Record metrics if provided
    if data.metrics:
        found_proxy = db(db.proxy_servers.name == data.proxy_name).select().first()
        if found_proxy:
            ProxyMetricsModel.record_metrics(db, found_proxy.id, data.metrics)

    return jsonify({"message": "Heartbeat recorded successfully"}), 200


@proxy_bp.route("/config", methods=["POST"])
async def get_config():
    """Get proxy configuration"""
    try:
        data_json = await request.get_json()
        data = ProxyConfigRequest(**data_json)
    except ValidationError as e:
        return jsonify({"error": "Validation error", "details": str(e)}), 400

    db = current_app.db

    config = ProxyServerModel.get_proxy_config(
        db, data.proxy_name, data.cluster_api_key
    )

    if not config:
        return (
            jsonify(
                {
                    "error": "Configuration retrieval failed - invalid API key or proxy not found"
                }
            ),
            400,
        )

    return jsonify(config), 200


@proxy_bp.route("/proxies", methods=["GET"])
@require_auth()
async def list_proxies(user_data):
    """List all proxies (authenticated endpoint)"""
    db = current_app.db
    user_id = user_data["user_id"]
    is_admin = user_data.get("is_admin", False)
    cluster_id = request.args.get("cluster_id")

    if is_admin:
        # Admin can see all proxies
        if cluster_id:
            proxies = ProxyServerModel.get_cluster_proxies(db, int(cluster_id))
        else:
            # Get all proxies across clusters
            all_proxies = db(db.proxy_servers).select()
            proxies = [
                {
                    "id": p.id,
                    "name": p.name,
                    "hostname": p.hostname,
                    "ip_address": p.ip_address,
                    "port": p.port,
                    "cluster_id": p.cluster_id,
                    "status": p.status,
                    "version": p.version,
                    "license_validated": p.license_validated,
                    "last_seen": p.last_seen,
                    "registered_at": p.registered_at,
                }
                for p in all_proxies
            ]
    else:
        # Regular user can only see proxies in their assigned clusters
        user_clusters = UserClusterAssignmentModel.get_user_clusters(db, user_id)
        cluster_ids = [uc["cluster_id"] for uc in user_clusters]

        if cluster_id and int(cluster_id) in cluster_ids:
            proxies = ProxyServerModel.get_cluster_proxies(db, int(cluster_id))
        else:
            # Get proxies from all user's clusters
            proxies = []
            for cid in cluster_ids:
                proxies.extend(ProxyServerModel.get_cluster_proxies(db, cid))

    return jsonify({"proxies": proxies}), 200


@proxy_bp.route("/proxies/<int:proxy_id>", methods=["GET"])
@require_auth()
async def get_proxy(user_data, proxy_id):
    """Get proxy details"""
    db = current_app.db
    user_id = user_data["user_id"]
    is_admin = user_data.get("is_admin", False)

    found_proxy = db.proxy_servers[proxy_id]
    if not found_proxy:
        return jsonify({"error": "Proxy not found"}), 404

    # Check access to proxy cluster
    if not is_admin:
        user_role = UserClusterAssignmentModel.check_user_cluster_access(
            db, user_id, found_proxy.cluster_id
        )
        if not user_role:
            return jsonify({"error": "Access denied to proxy"}), 403

    return (
        jsonify(
            ProxyResponse(
                id=found_proxy.id,
                name=found_proxy.name,
                hostname=found_proxy.hostname,
                ip_address=found_proxy.ip_address,
                port=found_proxy.port,
                cluster_id=found_proxy.cluster_id,
                status=found_proxy.status,
                version=found_proxy.version,
                license_validated=found_proxy.license_validated,
                last_seen=found_proxy.last_seen,
                registered_at=found_proxy.registered_at,
            ).dict()
        ),
        200,
    )


@proxy_bp.route("/proxies/stats", methods=["GET"])
@require_auth()
async def get_stats(user_data):
    """Get proxy statistics"""
    db = current_app.db
    user_id = user_data["user_id"]
    is_admin = user_data.get("is_admin", False)
    cluster_id = request.args.get("cluster_id")

    if cluster_id:
        cluster_id = int(cluster_id)
        # Check access to cluster
        if not is_admin:
            user_role = UserClusterAssignmentModel.check_user_cluster_access(
                db, user_id, cluster_id
            )
            if not user_role:
                return jsonify({"error": "Access denied to cluster"}), 403

    stats = ProxyServerModel.get_proxy_stats(db, cluster_id)
    return jsonify(ProxyStatsResponse(**stats).dict()), 200


@proxy_bp.route("/proxies/<int:proxy_id>/metrics", methods=["GET"])
@require_auth()
async def get_metrics(user_data, proxy_id):
    """Get proxy metrics"""
    db = current_app.db
    user_id = user_data["user_id"]
    is_admin = user_data.get("is_admin", False)

    found_proxy = db.proxy_servers[proxy_id]
    if not found_proxy:
        return jsonify({"error": "Proxy not found"}), 404

    # Check access to proxy cluster
    if not is_admin:
        user_role = UserClusterAssignmentModel.check_user_cluster_access(
            db, user_id, found_proxy.cluster_id
        )
        if not user_role:
            return jsonify({"error": "Access denied to proxy"}), 403

    hours = int(request.args.get("hours", 24))
    metrics = ProxyMetricsModel.get_metrics(db, proxy_id, hours)

    return (
        jsonify(
            {
                "proxy_id": proxy_id,
                "metrics": [
                    ProxyMetricsResponse(
                        proxy_id=metric["proxy_id"],
                        timestamp=metric["timestamp"],
                        cpu_usage=metric["cpu_usage"],
                        memory_usage=metric["memory_usage"],
                        connections_active=metric["connections_active"],
                        connections_total=metric["connections_total"],
                        bytes_sent=metric["bytes_sent"],
                        bytes_received=metric["bytes_received"],
                        requests_per_second=metric["requests_per_second"],
                        latency_avg=metric["latency_avg"],
                        latency_p95=metric["latency_p95"],
                        errors_per_second=metric["errors_per_second"],
                    ).dict()
                    for metric in metrics
                ],
            }
        ),
        200,
    )


@proxy_bp.route("/proxies/cleanup", methods=["POST"])
@require_auth(admin_required=True)
async def cleanup_stale(user_data):
    """Cleanup stale proxy registrations (admin only)"""
    db = current_app.db

    data_json = await request.get_json()
    timeout_minutes = int(data_json.get("timeout_minutes", 10))
    cleaned_count = ProxyServerModel.cleanup_stale_proxies(db, timeout_minutes)

    return (
        jsonify(
            {
                "message": f"Cleaned up {cleaned_count} stale proxy registrations",
                "cleaned_count": cleaned_count,
            }
        ),
        200,
    )
