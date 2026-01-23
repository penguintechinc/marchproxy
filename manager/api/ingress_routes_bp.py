"""
Ingress routes and routing configuration API Blueprint for MarchProxy Manager (Quart)

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from quart import Blueprint, request, current_app, jsonify
from pydantic import ValidationError, BaseModel
import logging
from datetime import datetime
from typing import Optional, Dict, Any, List
from middleware.auth import require_auth

logger = logging.getLogger(__name__)

ingress_routes_bp = Blueprint("ingress_routes", __name__, url_prefix="/api/v1/ingress-routes")


class CreateIngressRouteRequest(BaseModel):
    name: str
    cluster_id: int
    source_port: int
    dest_service_id: int
    protocol: str = "tcp"
    enabled: bool = True
    description: Optional[str] = None


class UpdateIngressRouteRequest(BaseModel):
    name: Optional[str] = None
    source_port: Optional[int] = None
    dest_service_id: Optional[int] = None
    protocol: Optional[str] = None
    enabled: Optional[bool] = None
    description: Optional[str] = None


class IngressRouteResponse(BaseModel):
    id: int
    name: str
    cluster_id: int
    source_port: int
    dest_service_id: int
    protocol: str
    enabled: bool
    description: Optional[str]
    created_at: datetime


@ingress_routes_bp.route("", methods=["GET", "POST"])
async def routes_list():
    """List all ingress routes or create new route"""
    db = current_app.db

    if request.method == "GET":

        @require_auth()
        async def get_routes(user_data):
            cluster_id = request.args.get("cluster_id", type=int)
            if not cluster_id:
                return jsonify({"error": "cluster_id parameter required"}), 400

            # Get all ingress routes for cluster
            try:
                routes = db(
                    (db.ingress_routes.cluster_id == cluster_id)
                    & (db.ingress_routes.is_active == True)
                ).select(orderby=db.ingress_routes.source_port)

                result = []
                for route in routes:
                    result.append(
                        IngressRouteResponse(
                            id=route.id,
                            name=route.name,
                            cluster_id=route.cluster_id,
                            source_port=route.source_port,
                            dest_service_id=route.dest_service_id,
                            protocol=route.protocol,
                            enabled=route.enabled,
                            description=route.description,
                            created_at=route.created_at,
                        ).dict()
                    )

                return jsonify({"routes": result}), 200
            except Exception as e:
                logger.error(f"Error fetching routes: {str(e)}")
                return (
                    jsonify({"error": "Failed to fetch routes", "details": str(e)}),
                    500,
                )

        return await get_routes(user_data={})

    elif request.method == "POST":

        @require_auth(admin_required=True)
        async def create_route_handler(user_data):
            try:
                data_json = await request.get_json()
                data = CreateIngressRouteRequest(**data_json)
            except ValidationError as e:
                return jsonify({"error": "Validation error", "details": str(e)}), 400

            try:
                # Validate destination service exists in cluster
                dest_service = (
                    db(
                        (db.services.id == data.dest_service_id)
                        & (db.services.cluster_id == data.cluster_id)
                        & (db.services.is_active == True)
                    )
                    .select()
                    .first()
                )

                if not dest_service:
                    return (
                        jsonify({"error": "Destination service not found in cluster"}),
                        404,
                    )

                # Check for port conflict
                existing = (
                    db(
                        (db.ingress_routes.cluster_id == data.cluster_id)
                        & (db.ingress_routes.source_port == data.source_port)
                        & (db.ingress_routes.protocol == data.protocol)
                        & (db.ingress_routes.is_active == True)
                    )
                    .select()
                    .first()
                )

                if existing:
                    return (
                        jsonify(
                            {"error": f"Port {data.source_port}/{data.protocol} already in use"}
                        ),
                        409,
                    )

                # Create route
                route_id = db.ingress_routes.insert(
                    name=data.name,
                    cluster_id=data.cluster_id,
                    source_port=data.source_port,
                    dest_service_id=data.dest_service_id,
                    protocol=data.protocol,
                    enabled=data.enabled,
                    description=data.description,
                    created_by=user_data["user_id"],
                    created_at=datetime.utcnow(),
                )

                route = db.ingress_routes[route_id]
                response = IngressRouteResponse(
                    id=route.id,
                    name=route.name,
                    cluster_id=route.cluster_id,
                    source_port=route.source_port,
                    dest_service_id=route.dest_service_id,
                    protocol=route.protocol,
                    enabled=route.enabled,
                    description=route.description,
                    created_at=route.created_at,
                )
                return jsonify(response.dict()), 201

            except Exception as e:
                logger.error(f"Error creating ingress route: {str(e)}")
                return (
                    jsonify({"error": "Failed to create route", "details": str(e)}),
                    500,
                )

        return await create_route_handler(user_data={})


@ingress_routes_bp.route("/<int:route_id>", methods=["GET", "PUT", "DELETE"])
async def route_detail(route_id):
    """Get, update or delete an ingress route"""
    db = current_app.db

    @require_auth()
    async def handler(user_data):
        try:
            route = db.ingress_routes[route_id]
            if not route or not route.is_active:
                return jsonify({"error": "Route not found"}), 404

            if request.method == "GET":
                response = IngressRouteResponse(
                    id=route.id,
                    name=route.name,
                    cluster_id=route.cluster_id,
                    source_port=route.source_port,
                    dest_service_id=route.dest_service_id,
                    protocol=route.protocol,
                    enabled=route.enabled,
                    description=route.description,
                    created_at=route.created_at,
                )
                return jsonify(response.dict()), 200

            elif request.method == "PUT":
                # Admin only
                user = db.auth_user[user_data["user_id"]]
                if not user.is_admin:
                    return jsonify({"error": "Admin access required"}), 403

                try:
                    data_json = await request.get_json()
                    data = UpdateIngressRouteRequest(**data_json)
                except ValidationError as e:
                    return (
                        jsonify({"error": "Validation error", "details": str(e)}),
                        400,
                    )

                update_data = {}

                # Validate new destination service if provided
                if data.dest_service_id:
                    dest_service = (
                        db(
                            (db.services.id == data.dest_service_id)
                            & (db.services.cluster_id == route.cluster_id)
                            & (db.services.is_active == True)
                        )
                        .select()
                        .first()
                    )
                    if not dest_service:
                        return jsonify({"error": "Destination service not found"}), 404
                    update_data["dest_service_id"] = data.dest_service_id

                # Check for port conflict if port is being changed
                if data.source_port and data.source_port != route.source_port:
                    protocol = data.protocol or route.protocol
                    existing = (
                        db(
                            (db.ingress_routes.cluster_id == route.cluster_id)
                            & (db.ingress_routes.source_port == data.source_port)
                            & (db.ingress_routes.protocol == protocol)
                            & (db.ingress_routes.id != route_id)
                            & (db.ingress_routes.is_active == True)
                        )
                        .select()
                        .first()
                    )
                    if existing:
                        return (
                            jsonify(
                                {"error": f"Port {data.source_port}/{protocol} already in use"}
                            ),
                            409,
                        )
                    update_data["source_port"] = data.source_port

                if data.name:
                    update_data["name"] = data.name
                if data.protocol:
                    update_data["protocol"] = data.protocol
                if data.enabled is not None:
                    update_data["enabled"] = data.enabled
                if data.description is not None:
                    update_data["description"] = data.description

                if update_data:
                    update_data["updated_at"] = datetime.utcnow()
                    route.update_record(**update_data)

                # Fetch updated route
                updated_route = db.ingress_routes[route_id]
                response = IngressRouteResponse(
                    id=updated_route.id,
                    name=updated_route.name,
                    cluster_id=updated_route.cluster_id,
                    source_port=updated_route.source_port,
                    dest_service_id=updated_route.dest_service_id,
                    protocol=updated_route.protocol,
                    enabled=updated_route.enabled,
                    description=updated_route.description,
                    created_at=updated_route.created_at,
                )
                return jsonify(response.dict()), 200

            elif request.method == "DELETE":
                # Admin only
                user = db.auth_user[user_data["user_id"]]
                if not user.is_admin:
                    return jsonify({"error": "Admin access required"}), 403

                route.update_record(is_active=False, updated_at=datetime.utcnow())
                return jsonify({"message": "Route deleted"}), 204

        except Exception as e:
            logger.error(f"Error in route detail handler: {str(e)}")
            return jsonify({"error": "Internal server error", "details": str(e)}), 500

    return await handler(user_data={})


@ingress_routes_bp.route("/by-port/<int:port>", methods=["GET"])
@require_auth()
async def get_route_by_port(port, user_data):
    """Get ingress route by source port"""
    db = current_app.db
    cluster_id = request.args.get("cluster_id", type=int)

    if not cluster_id:
        return jsonify({"error": "cluster_id parameter required"}), 400

    try:
        route = (
            db(
                (db.ingress_routes.cluster_id == cluster_id)
                & (db.ingress_routes.source_port == port)
                & (db.ingress_routes.is_active == True)
            )
            .select()
            .first()
        )

        if not route:
            return jsonify({"error": "Route not found"}), 404

        response = IngressRouteResponse(
            id=route.id,
            name=route.name,
            cluster_id=route.cluster_id,
            source_port=route.source_port,
            dest_service_id=route.dest_service_id,
            protocol=route.protocol,
            enabled=route.enabled,
            description=route.description,
            created_at=route.created_at,
        )
        return jsonify(response.dict()), 200

    except Exception as e:
        logger.error(f"Error fetching route by port: {str(e)}")
        return jsonify({"error": "Failed to fetch route", "details": str(e)}), 500


@ingress_routes_bp.route("/status/<int:route_id>", methods=["PUT"])
@require_auth(admin_required=True)
async def update_route_status(route_id, user_data):
    """Enable or disable an ingress route"""
    db = current_app.db

    try:
        data_json = await request.get_json()
        enabled = data_json.get("enabled")

        if enabled is None:
            return jsonify({"error": "enabled parameter required"}), 400

        if not isinstance(enabled, bool):
            return jsonify({"error": "enabled must be boolean"}), 400

        route = db.ingress_routes[route_id]
        if not route or not route.is_active:
            return jsonify({"error": "Route not found"}), 404

        route.update_record(enabled=enabled, updated_at=datetime.utcnow())

        response = IngressRouteResponse(
            id=route.id,
            name=route.name,
            cluster_id=route.cluster_id,
            source_port=route.source_port,
            dest_service_id=route.dest_service_id,
            protocol=route.protocol,
            enabled=route.enabled,
            description=route.description,
            created_at=route.created_at,
        )
        return jsonify(response.dict()), 200

    except Exception as e:
        logger.error(f"Error updating route status: {str(e)}")
        return (
            jsonify({"error": "Failed to update route status", "details": str(e)}),
            500,
        )


@ingress_routes_bp.route("/validate", methods=["POST"])
@require_auth(admin_required=True)
async def validate_route_config(user_data):
    """Validate ingress route configuration"""
    db = current_app.db

    try:
        data_json = await request.get_json()
        data = CreateIngressRouteRequest(**data_json)

        # Validate cluster exists
        cluster = db.clusters[data.cluster_id]
        if not cluster or not cluster.is_active:
            return jsonify({"error": "Cluster not found"}), 404

        # Validate destination service
        dest_service = (
            db(
                (db.services.id == data.dest_service_id)
                & (db.services.cluster_id == data.cluster_id)
                & (db.services.is_active == True)
            )
            .select()
            .first()
        )

        if not dest_service:
            return jsonify({"error": "Destination service not found"}), 404

        # Check for port conflict
        existing = (
            db(
                (db.ingress_routes.cluster_id == data.cluster_id)
                & (db.ingress_routes.source_port == data.source_port)
                & (db.ingress_routes.protocol == data.protocol)
                & (db.ingress_routes.is_active == True)
            )
            .select()
            .first()
        )

        if existing:
            return (
                jsonify(
                    {
                        "valid": False,
                        "error": f"Port {data.source_port}/{data.protocol} already in use",
                    }
                ),
                200,
            )

        return jsonify({"valid": True, "message": "Route configuration is valid"}), 200

    except ValidationError as e:
        return (
            jsonify({"valid": False, "error": "Validation error", "details": str(e)}),
            200,
        )
    except Exception as e:
        logger.error(f"Error validating route config: {str(e)}")
        return jsonify({"error": "Failed to validate route", "details": str(e)}), 500
