"""
Cluster management API Blueprint for MarchProxy Manager (Quart)

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import logging # noqa: F401, # noqa: F401
from datetime import datetime # noqa: F401

from middleware.auth import require_auth # noqa: F401
from models.cluster import ( # noqa: F401
    AssignUserToClusterRequest,
    ClusterModel,
    ClusterResponse,
    CreateClusterRequest,
    UpdateClusterRequest,
    UserClusterAssignmentModel,
)
from penguintechinc_utils import get_logger # noqa: F401
from pydantic import ValidationError # noqa: F401
from quart import Blueprint, current_app, jsonify, request # noqa: F401

logger = get_logger(__name__)

clusters_bp = Blueprint("clusters", __name__, url_prefix="/api/v1/clusters")


@clusters_bp.route("", methods=["GET", "POST"])
async def clusters_list(): # noqa: C901
    """List all clusters or create new cluster"""
    db = current_app.db

    if request.method == "GET":

        @require_auth()
        async def get_clusters_handler(user_data):
            user = user_data

            if user.get("is_admin") or "*:admin" in user.get("scope", []) or "admin" in user.get("roles", []):
              # Admin sees all clusters
                all_clusters = db(db.clusters.is_active == True).select( # noqa: E712
                    orderby=db.clusters.name
                )
            else:
              # Regular user sees only assigned clusters
                user_clusters = UserClusterAssignmentModel.get_user_clusters(db, user["user_id"])
                cluster_ids = [uc["cluster_id"] for uc in user_clusters]
                all_clusters = db(
                    (db.clusters.id.belongs(cluster_ids))
                    & (db.clusters.is_active == True)  # noqa: E712
                ).select(orderby=db.clusters.name)

            result = []
            for cluster in all_clusters:
                active_proxies = ClusterModel.count_active_proxies(db, cluster.id)
                result.append(
                    ClusterResponse(
                        id=cluster.id,
                        name=cluster.name,
                        description=cluster.description,
                        syslog_endpoint=cluster.syslog_endpoint,
                        log_auth=cluster.log_auth,
                        log_netflow=cluster.log_netflow,
                        log_debug=cluster.log_debug,
                        is_active=cluster.is_active,
                        is_default=cluster.is_default,
                        max_proxies=cluster.max_proxies,
                        active_proxies=active_proxies,
                        created_at=cluster.created_at,
                        updated_at=cluster.updated_at,
                    ).dict()
                )

            return jsonify({"clusters": result}), 200

        return await get_clusters_handler()

    elif request.method == "POST":

        @require_auth(admin_required=True)
        async def create_cluster_handler(user_data):
            try:
                data_json = await request.get_json()
                data = CreateClusterRequest(**data_json)
            except ValidationError as e:
                return jsonify({"error": "Validation error", "details": str(e)}), 400

          # Check if cluster name already exists
            existing = db(db.clusters.name == data.name).select().first()
            if existing:
                return jsonify({"error": "Cluster name already exists"}), 409

          # Create cluster
            try:
                cluster_id, api_key = ClusterModel.create_cluster(
                    db,
                    name=data.name,
                    description=data.description,
                    created_by=user_data["user_id"],
                    syslog_endpoint=data.syslog_endpoint,
                    log_auth=data.log_auth,
                    log_netflow=data.log_netflow,
                    log_debug=data.log_debug,
                    max_proxies=data.max_proxies,
                )

                cluster = db.clusters[cluster_id]
                return (
                    jsonify(
                        {
                            "cluster": ClusterResponse(
                                id=cluster.id,
                                name=cluster.name,
                                description=cluster.description,
                                syslog_endpoint=cluster.syslog_endpoint,
                                log_auth=cluster.log_auth,
                                log_netflow=cluster.log_netflow,
                                log_debug=cluster.log_debug,
                                is_active=cluster.is_active,
                                is_default=cluster.is_default,
                                max_proxies=cluster.max_proxies,
                                active_proxies=0,
                                created_at=cluster.created_at,
                                updated_at=cluster.updated_at,
                            ).dict(),
                            "api_key": api_key,
                            "message": "Cluster created successfully",
                        }
                    ),
                    201,
                )

            except Exception as e:
                logger.error(f"Cluster creation failed: {e}")
                return jsonify({"error": "Failed to create cluster"}), 500

        return await create_cluster_handler(user_data=None)


@clusters_bp.route("/<cluster_id>", methods=["GET", "PUT"])
async def cluster_detail(cluster_id): # noqa: C901
    """Get or update cluster details"""
    db = current_app.db

    if request.method == "GET":

        @require_auth()
        async def get_cluster_handler(user_data):
            user = user_data

          # Check access to cluster
            is_admin = user.get("is_admin") or "*:admin" in user.get("scope", []) or "admin" in user.get("roles", [])
            if not is_admin:
                user_role = UserClusterAssignmentModel.check_user_cluster_access(
                    db, user["user_id"], cluster_id
                )
                if not user_role:
                    return jsonify({"error": "Access denied to cluster"}), 403

            cluster = (
                db((db.clusters.id == cluster_id) & (db.clusters.is_active == True))  # noqa: E712
                .select()
                .first()
            )
            if not cluster:
                return jsonify({"error": "Cluster not found"}), 404

            active_proxies = ClusterModel.count_active_proxies(db, cluster.id)

            return (
                jsonify(
                    ClusterResponse(
                        id=cluster.id,
                        name=cluster.name,
                        description=cluster.description,
                        syslog_endpoint=cluster.syslog_endpoint,
                        log_auth=cluster.log_auth,
                        log_netflow=cluster.log_netflow,
                        log_debug=cluster.log_debug,
                        is_active=cluster.is_active,
                        is_default=cluster.is_default,
                        max_proxies=cluster.max_proxies,
                        active_proxies=active_proxies,
                        created_at=cluster.created_at,
                        updated_at=cluster.updated_at,
                    ).dict()
                ),
                200,
            )

        return await get_cluster_handler(user_data=None)

    elif request.method == "PUT":

        @require_auth(admin_required=True)
        async def update_cluster_handler(user_data):
            try:
                data_json = await request.get_json()
                data = UpdateClusterRequest(**data_json)
            except ValidationError as e:
                return jsonify({"error": "Validation error", "details": str(e)}), 400

            cluster = db.clusters[cluster_id]
            if not cluster:
                return jsonify({"error": "Cluster not found"}), 404

          # Update cluster
            update_data = {"updated_at": datetime.utcnow()}

            if data.name is not None:
              # Check name uniqueness
                existing = (
                    db((db.clusters.name == data.name) & (db.clusters.id != cluster_id))
                    .select()
                    .first()
                )
                if existing:
                    return jsonify({"error": "Cluster name already exists"}), 409
                update_data["name"] = data.name

            if data.description is not None:
                update_data["description"] = data.description
            if data.syslog_endpoint is not None:
                update_data["syslog_endpoint"] = data.syslog_endpoint
            if data.log_auth is not None:
                update_data["log_auth"] = data.log_auth
            if data.log_netflow is not None:
                update_data["log_netflow"] = data.log_netflow
            if data.log_debug is not None:
                update_data["log_debug"] = data.log_debug
            if data.max_proxies is not None:
                update_data["max_proxies"] = data.max_proxies

            cluster.update_record(**update_data)

            active_proxies = ClusterModel.count_active_proxies(db, cluster.id)

            return (
                jsonify(
                    ClusterResponse(
                        id=cluster.id,
                        name=cluster.name,
                        description=cluster.description,
                        syslog_endpoint=cluster.syslog_endpoint,
                        log_auth=cluster.log_auth,
                        log_netflow=cluster.log_netflow,
                        log_debug=cluster.log_debug,
                        is_active=cluster.is_active,
                        is_default=cluster.is_default,
                        max_proxies=cluster.max_proxies,
                        active_proxies=active_proxies,
                        created_at=cluster.created_at,
                        updated_at=cluster.updated_at,
                    ).dict()
                ),
                200,
            )

        return await update_cluster_handler(user_data=None)


@clusters_bp.route("/<cluster_id>/rotate-key", methods=["POST"])
@require_auth(admin_required=True)
async def rotate_api_key(cluster_id, user_data):
    """Rotate cluster API key"""
    db = current_app.db

    cluster = db.clusters[cluster_id]
    if not cluster:
        return jsonify({"error": "Cluster not found"}), 404

    new_api_key = ClusterModel.rotate_api_key(db, cluster_id)
    if not new_api_key:
        return jsonify({"error": "Failed to rotate API key"}), 500

    return (
        jsonify({"api_key": new_api_key, "message": "API key rotated successfully"}),
        200,
    )


@clusters_bp.route("/<cluster_id>/logging", methods=["PUT"])
@require_auth(admin_required=True)
async def update_logging_config(cluster_id, user_data):
    """Update cluster logging configuration"""
    db = current_app.db

    data = await request.get_json()

    success = ClusterModel.update_logging_config(
        db,
        cluster_id,
        syslog_endpoint=data.get("syslog_endpoint"),
        log_auth=data.get("log_auth"),
        log_netflow=data.get("log_netflow"),
        log_debug=data.get("log_debug"),
    )

    if not success:
        return jsonify({"error": "Cluster not found"}), 404

    return jsonify({"message": "Logging configuration updated successfully"}), 200


@clusters_bp.route("/<cluster_id>/assign-user", methods=["POST"])
@require_auth(admin_required=True)
async def assign_user(cluster_id, user_data):
    """Assign user to cluster"""
    db = current_app.db

    try:
        data_json = await request.get_json()
        data = AssignUserToClusterRequest(**data_json)
    except ValidationError as e:
        return jsonify({"error": "Validation error", "details": str(e)}), 400

  # Check if cluster exists
    cluster = db.clusters[cluster_id]
    if not cluster:
        return jsonify({"error": "Cluster not found"}), 404

  # Check if user exists
    user = db.users[data.user_id]
    if not user:
        return jsonify({"error": "User not found"}), 404

  # Assign user to cluster
    success = UserClusterAssignmentModel.assign_user_to_cluster(
        db, data.user_id, cluster_id, data.role, user_data["user_id"]
    )

    if success:
        return jsonify({"message": "User assigned to cluster successfully"}), 200
    else:
        return jsonify({"error": "Failed to assign user to cluster"}), 500


@clusters_bp.route("/config/<cluster_id>", methods=["GET"])
async def get_config(cluster_id):
    """Get cluster configuration for proxy (API key authenticated)"""
    db = current_app.db

  # This endpoint uses API key authentication instead of JWT
    api_key = request.headers.get("X-API-Key") or request.args.get("api_key")
    if not api_key:
        return jsonify({"error": "API key required"}), 401

  # Validate cluster API key
    cluster_info = ClusterModel.validate_api_key(db, api_key)
    if not cluster_info or cluster_info["cluster_id"] != int(cluster_id):
        return jsonify({"error": "Invalid API key for cluster"}), 401

  # Get cluster configuration
    config = ClusterModel.get_cluster_config(db, cluster_id)
    if not config:
        return jsonify({"error": "Cluster configuration not found"}), 404

    return jsonify(config), 200
