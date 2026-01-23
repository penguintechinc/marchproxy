"""
Service management API Blueprint for MarchProxy Manager (Quart)

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from quart import Blueprint, request, current_app, jsonify
from pydantic import ValidationError
import logging
from datetime import datetime
from models.service import (
    ServiceModel,
    UserServiceAssignmentModel,
    CreateServiceRequest,
    UpdateServiceRequest,
    ServiceResponse,
    ServiceAuthResponse,
    AssignUserToServiceRequest,
    SetServiceAuthRequest,
    CreateJwtTokenRequest,
)
from middleware.auth import require_auth

logger = logging.getLogger(__name__)

services_bp = Blueprint("services", __name__, url_prefix="/api/v1/services")


@services_bp.route("", methods=["GET", "POST"])
async def services_list():
    """List all services or create new service"""
    db = current_app.db

    if request.method == "GET":

        @require_auth()
        async def get_services(user_data):
            cluster_id = request.args.get("cluster_id", type=int)
            if not cluster_id:
                return jsonify({"error": "cluster_id parameter required"}), 400

            user_id = user_data.get("user_id")
            services_list = ServiceModel.get_cluster_services(db, cluster_id, user_id)

            result = []
            for svc in services_list:
                result.append(
                    ServiceResponse(
                        id=svc["id"],
                        name=svc["name"],
                        ip_fqdn=svc["ip_fqdn"],
                        port=svc["port"],
                        protocol=svc["protocol"],
                        collection=svc["collection"],
                        cluster_id=svc.get("cluster_id", cluster_id),
                        auth_type=svc["auth_type"],
                        tls_enabled=svc["tls_enabled"],
                        health_check_enabled=svc["health_check_enabled"],
                        created_at=svc["created_at"],
                    ).dict()
                )

            return jsonify({"services": result}), 200

        return await get_services(user_data={})

    elif request.method == "POST":

        @require_auth(admin_required=True)
        async def create_service_handler(user_data):
            try:
                data_json = await request.get_json()
                data = CreateServiceRequest(**data_json)
            except ValidationError as e:
                return jsonify({"error": "Validation error", "details": str(e)}), 400

            try:
                service_id = ServiceModel.create_service(
                    db,
                    name=data.name,
                    ip_fqdn=data.ip_fqdn,
                    port=data.port,
                    cluster_id=data.cluster_id,
                    created_by=user_data["user_id"],
                    protocol=data.protocol,
                    collection=data.collection,
                    auth_type=data.auth_type,
                    tls_enabled=data.tls_enabled,
                    tls_verify=data.tls_verify,
                )

                service = db.services[service_id]
                response = ServiceResponse(
                    id=service.id,
                    name=service.name,
                    ip_fqdn=service.ip_fqdn,
                    port=service.port,
                    protocol=service.protocol,
                    collection=service.collection,
                    cluster_id=service.cluster_id,
                    auth_type=service.auth_type,
                    tls_enabled=service.tls_enabled,
                    health_check_enabled=service.health_check_enabled,
                    created_at=service.created_at,
                )
                return jsonify(response.dict()), 201
            except Exception as e:
                logger.error(f"Error creating service: {str(e)}")
                return (
                    jsonify({"error": "Failed to create service", "details": str(e)}),
                    500,
                )

        return await create_service_handler(user_data={})


@services_bp.route("/<int:service_id>", methods=["GET", "PUT", "DELETE"])
async def service_detail(service_id):
    """Get, update or delete a service"""
    db = current_app.db

    @require_auth()
    async def handler(user_data):
        service = db.services[service_id]
        if not service:
            return jsonify({"error": "Service not found"}), 404

        if request.method == "GET":
            config = ServiceModel.get_service_config(db, service_id)
            if not config:
                return jsonify({"error": "Service not found"}), 404

            return jsonify(config), 200

        elif request.method == "PUT":
            # Admin only
            user = db.auth_user[user_data["user_id"]]
            if not user.is_admin:
                return jsonify({"error": "Admin access required"}), 403

            try:
                data_json = await request.get_json()
                data = UpdateServiceRequest(**data_json)
            except ValidationError as e:
                return jsonify({"error": "Validation error", "details": str(e)}), 400

            update_data = {}
            if data.name:
                update_data["name"] = data.name
            if data.ip_fqdn:
                update_data["ip_fqdn"] = data.ip_fqdn
            if data.port:
                update_data["port"] = data.port
            if data.protocol:
                update_data["protocol"] = data.protocol
            if data.collection:
                update_data["collection"] = data.collection
            if data.auth_type:
                update_data["auth_type"] = data.auth_type
            if data.tls_enabled is not None:
                update_data["tls_enabled"] = data.tls_enabled
            if data.tls_verify is not None:
                update_data["tls_verify"] = data.tls_verify
            if data.health_check_enabled is not None:
                update_data["health_check_enabled"] = data.health_check_enabled
            if data.health_check_path:
                update_data["health_check_path"] = data.health_check_path
            if data.health_check_interval:
                update_data["health_check_interval"] = data.health_check_interval

            if update_data:
                update_data["updated_at"] = datetime.utcnow()
                service.update_record(**update_data)

            response = ServiceResponse(
                id=service.id,
                name=service.name,
                ip_fqdn=service.ip_fqdn,
                port=service.port,
                protocol=service.protocol,
                collection=service.collection,
                cluster_id=service.cluster_id,
                auth_type=service.auth_type,
                tls_enabled=service.tls_enabled,
                health_check_enabled=service.health_check_enabled,
                created_at=service.created_at,
            )
            return jsonify(response.dict()), 200

        elif request.method == "DELETE":
            # Admin only
            user = db.auth_user[user_data["user_id"]]
            if not user.is_admin:
                return jsonify({"error": "Admin access required"}), 403

            service.update_record(is_active=False)
            return jsonify({"message": "Service deleted"}), 204

    return await handler(user_data={})


@services_bp.route("/<int:service_id>/auth", methods=["POST"])
@require_auth(admin_required=True)
async def set_service_auth(service_id, user_data):
    """Set authentication method for service"""
    db = current_app.db
    service = db.services[service_id]

    if not service:
        return jsonify({"error": "Service not found"}), 404

    try:
        data_json = await request.get_json()
        data = SetServiceAuthRequest(**data_json)
    except ValidationError as e:
        return jsonify({"error": "Validation error", "details": str(e)}), 400

    try:
        if data.auth_type == "base64":
            token = ServiceModel.set_base64_auth(db, service_id)
            response = ServiceAuthResponse(service_id=service_id, auth_type="base64", token=token)
        elif data.auth_type == "jwt":
            jwt_secret = ServiceModel.set_jwt_auth(
                db,
                service_id,
                expiry_seconds=data.jwt_expiry,
                algorithm=data.jwt_algorithm,
            )
            response = ServiceAuthResponse(
                service_id=service_id,
                auth_type="jwt",
                jwt_secret=jwt_secret,
                jwt_expiry=data.jwt_expiry,
            )
        elif data.auth_type == "none":
            service.update_record(
                auth_type="none",
                token_base64=None,
                jwt_secret=None,
                updated_at=datetime.utcnow(),
            )
            response = ServiceAuthResponse(service_id=service_id, auth_type="none")
        else:
            return jsonify({"error": "Invalid auth type"}), 400

        return jsonify(response.dict()), 200
    except Exception as e:
        logger.error(f"Error setting service auth: {str(e)}")
        return (
            jsonify({"error": "Failed to set authentication", "details": str(e)}),
            500,
        )


@services_bp.route("/<int:service_id>/auth/rotate", methods=["POST"])
@require_auth(admin_required=True)
async def rotate_service_jwt(service_id, user_data):
    """Rotate JWT secret for service"""
    db = current_app.db
    service = db.services[service_id]

    if not service or service.auth_type != "jwt":
        return jsonify({"error": "Service not found or not using JWT auth"}), 404

    try:
        new_secret = ServiceModel.rotate_jwt_secret(db, service_id)
        if not new_secret:
            return jsonify({"error": "Failed to rotate JWT secret"}), 500

        response = ServiceAuthResponse(
            service_id=service_id,
            auth_type="jwt",
            jwt_secret=new_secret,
            jwt_expiry=service.jwt_expiry,
        )
        return jsonify(response.dict()), 200
    except Exception as e:
        logger.error(f"Error rotating JWT secret: {str(e)}")
        return jsonify({"error": "Failed to rotate JWT secret", "details": str(e)}), 500


@services_bp.route("/<int:service_id>/token", methods=["POST"])
@require_auth(admin_required=True)
async def create_service_token(service_id, user_data):
    """Create JWT token for service"""
    db = current_app.db
    service = db.services[service_id]

    if not service or service.auth_type != "jwt":
        return jsonify({"error": "Service not found or not using JWT auth"}), 404

    try:
        data_json = await request.get_json()
        data = CreateJwtTokenRequest(**data_json)
    except (ValidationError, Exception) as e:
        data = CreateJwtTokenRequest(service_id=service_id)

    try:
        token = ServiceModel.create_jwt_token(db, service_id, data.additional_claims)
        if not token:
            return jsonify({"error": "Failed to create token"}), 500

        return jsonify({"token": token, "service_id": service_id}), 200
    except Exception as e:
        logger.error(f"Error creating service token: {str(e)}")
        return jsonify({"error": "Failed to create token", "details": str(e)}), 500


@services_bp.route("/<int:service_id>/assign", methods=["POST"])
@require_auth(admin_required=True)
async def assign_user_to_service(service_id, user_data):
    """Assign user to service"""
    db = current_app.db

    try:
        data_json = await request.get_json()
        data = AssignUserToServiceRequest(**data_json)
    except ValidationError as e:
        return jsonify({"error": "Validation error", "details": str(e)}), 400

    try:
        result = UserServiceAssignmentModel.assign_user_to_service(
            db,
            user_id=data.user_id,
            service_id=service_id,
            assigned_by=user_data["user_id"],
        )

        if result:
            return jsonify({"message": "User assigned to service"}), 200
        else:
            return jsonify({"error": "Failed to assign user"}), 500
    except Exception as e:
        logger.error(f"Error assigning user to service: {str(e)}")
        return jsonify({"error": "Failed to assign user", "details": str(e)}), 500


@services_bp.route("/<int:service_id>/unassign/<int:user_id>", methods=["DELETE"])
@require_auth(admin_required=True)
async def remove_user_from_service(service_id, user_id, user_data):
    """Remove user from service"""
    db = current_app.db

    try:
        result = UserServiceAssignmentModel.remove_user_from_service(db, user_id, service_id)
        if result:
            return jsonify({"message": "User removed from service"}), 200
        else:
            return jsonify({"error": "Failed to remove user"}), 500
    except Exception as e:
        logger.error(f"Error removing user from service: {str(e)}")
        return jsonify({"error": "Failed to remove user", "details": str(e)}), 500
