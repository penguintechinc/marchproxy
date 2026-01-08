"""
Mapping configuration API Blueprint for MarchProxy Manager (Quart)

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from quart import Blueprint, request, current_app, jsonify
from pydantic import ValidationError
import logging
from datetime import datetime
from models.mapping import (
    MappingModel,
    CreateMappingRequest, UpdateMappingRequest, MappingResponse,
    ResolvedMappingResponse
)
from middleware.auth import require_auth

logger = logging.getLogger(__name__)

mappings_bp = Blueprint('mappings', __name__, url_prefix='/api/v1/mappings')


@mappings_bp.route('', methods=['GET', 'POST'])
async def mappings_list():
    """List all mappings or create new mapping"""
    db = current_app.db

    if request.method == 'GET':
        @require_auth()
        async def get_mappings(user_data):
            cluster_id = request.args.get('cluster_id', type=int)
            if not cluster_id:
                return jsonify({"error": "cluster_id parameter required"}), 400

            user_id = user_data.get('user_id')
            mappings_list = MappingModel.get_cluster_mappings(db, cluster_id, user_id)

            result = []
            for mapping in mappings_list:
                result.append(MappingResponse(
                    id=mapping['id'],
                    name=mapping['name'],
                    description=mapping['description'],
                    source_services=mapping['source_services'],
                    dest_services=mapping['dest_services'],
                    cluster_id=cluster_id,
                    protocols=mapping['protocols'],
                    ports=mapping['ports'],
                    auth_required=mapping['auth_required'],
                    priority=mapping['priority'],
                    created_at=mapping['created_at']
                ).dict())

            return jsonify({"mappings": result}), 200

        return await get_mappings(user_data={})

    elif request.method == 'POST':
        @require_auth(admin_required=True)
        async def create_mapping_handler(user_data):
            try:
                data_json = await request.get_json()
                data = CreateMappingRequest(**data_json)
            except ValidationError as e:
                return jsonify({"error": "Validation error", "details": str(e)}), 400

            try:
                mapping_id = MappingModel.create_mapping(
                    db,
                    name=data.name,
                    source_services=data.source_services,
                    dest_services=data.dest_services,
                    ports=data.ports,
                    cluster_id=data.cluster_id,
                    created_by=user_data['user_id'],
                    protocols=data.protocols,
                    auth_required=data.auth_required,
                    priority=data.priority,
                    description=data.description,
                    comments=data.comments
                )

                mapping = db.mappings[mapping_id]
                response = MappingResponse(
                    id=mapping.id,
                    name=mapping.name,
                    description=mapping.description,
                    source_services=mapping.source_services,
                    dest_services=mapping.dest_services,
                    cluster_id=mapping.cluster_id,
                    protocols=mapping.protocols,
                    ports=mapping.ports,
                    auth_required=mapping.auth_required,
                    priority=mapping.priority,
                    created_at=mapping.created_at
                )
                return jsonify(response.dict()), 201
            except ValueError as e:
                return jsonify({"error": str(e)}), 400
            except Exception as e:
                logger.error(f"Error creating mapping: {str(e)}")
                return jsonify({"error": "Failed to create mapping", "details": str(e)}), 500

        return await create_mapping_handler(user_data={})


@mappings_bp.route('/<int:mapping_id>', methods=['GET', 'PUT', 'DELETE'])
async def mapping_detail(mapping_id):
    """Get, update or delete a mapping"""
    db = current_app.db

    @require_auth()
    async def handler(user_data):
        mapping = db.mappings[mapping_id]
        if not mapping:
            return jsonify({"error": "Mapping not found"}), 404

        if request.method == 'GET':
            response = MappingResponse(
                id=mapping.id,
                name=mapping.name,
                description=mapping.description,
                source_services=mapping.source_services,
                dest_services=mapping.dest_services,
                cluster_id=mapping.cluster_id,
                protocols=mapping.protocols,
                ports=mapping.ports,
                auth_required=mapping.auth_required,
                priority=mapping.priority,
                created_at=mapping.created_at
            )
            return jsonify(response.dict()), 200

        elif request.method == 'PUT':
            # Admin only
            user = db.auth_user[user_data['user_id']]
            if not user.is_admin:
                return jsonify({"error": "Admin access required"}), 403

            try:
                data_json = await request.get_json()
                data = UpdateMappingRequest(**data_json)
            except ValidationError as e:
                return jsonify({"error": "Validation error", "details": str(e)}), 400

            update_data = {}
            if data.name:
                update_data['name'] = data.name
            if data.description is not None:
                update_data['description'] = data.description
            if data.source_services:
                normalized_sources = MappingModel._normalize_service_list(
                    db, data.source_services, mapping.cluster_id
                )
                if normalized_sources:
                    update_data['source_services'] = normalized_sources
            if data.dest_services:
                normalized_dests = MappingModel._normalize_service_list(
                    db, data.dest_services, mapping.cluster_id
                )
                if normalized_dests:
                    update_data['dest_services'] = normalized_dests
            if data.protocols:
                update_data['protocols'] = data.protocols
            if data.ports:
                normalized_ports = MappingModel._normalize_port_list(data.ports)
                if normalized_ports:
                    update_data['ports'] = normalized_ports
            if data.auth_required is not None:
                update_data['auth_required'] = data.auth_required
            if data.priority is not None:
                update_data['priority'] = data.priority
            if data.comments is not None:
                update_data['comments'] = data.comments

            if update_data:
                update_data['updated_at'] = datetime.utcnow()
                mapping.update_record(**update_data)

            response = MappingResponse(
                id=mapping.id,
                name=mapping.name,
                description=mapping.description,
                source_services=mapping.source_services,
                dest_services=mapping.dest_services,
                cluster_id=mapping.cluster_id,
                protocols=mapping.protocols,
                ports=mapping.ports,
                auth_required=mapping.auth_required,
                priority=mapping.priority,
                created_at=mapping.created_at
            )
            return jsonify(response.dict()), 200

        elif request.method == 'DELETE':
            # Admin only
            user = db.auth_user[user_data['user_id']]
            if not user.is_admin:
                return jsonify({"error": "Admin access required"}), 403

            mapping.update_record(is_active=False)
            return jsonify({"message": "Mapping deleted"}), 204

    return await handler(user_data={})


@mappings_bp.route('/<int:mapping_id>/resolve', methods=['GET'])
@require_auth()
async def resolve_mapping(mapping_id, user_data):
    """Resolve mapping to concrete service configurations"""
    db = current_app.db

    try:
        resolved = MappingModel.resolve_mapping_services(db, mapping_id)
        if not resolved:
            return jsonify({"error": "Mapping not found"}), 404

        response = ResolvedMappingResponse(
            id=resolved['id'],
            name=resolved['name'],
            sources=resolved['sources'],
            destinations=resolved['destinations'],
            protocols=resolved['protocols'],
            ports=resolved['ports'],
            auth_required=resolved['auth_required'],
            priority=resolved['priority']
        )
        return jsonify(response.dict()), 200
    except Exception as e:
        logger.error(f"Error resolving mapping: {str(e)}")
        return jsonify({"error": "Failed to resolve mapping", "details": str(e)}), 500


@mappings_bp.route('/match', methods=['POST'])
@require_auth()
async def find_matching_mappings(user_data):
    """Find mappings matching source, destination, protocol, and port"""
    db = current_app.db

    try:
        data_json = await request.get_json()
        source_service_id = data_json.get('source_service_id')
        dest_service_id = data_json.get('dest_service_id')
        protocol = data_json.get('protocol', 'tcp')
        port = data_json.get('port')

        if not all([source_service_id, dest_service_id, port]):
            return jsonify({
                "error": "Missing required parameters: source_service_id, dest_service_id, port"
            }), 400

        matching = MappingModel.find_matching_mappings(
            db, source_service_id, dest_service_id, protocol, port
        )

        return jsonify({"mappings": matching}), 200
    except Exception as e:
        logger.error(f"Error finding matching mappings: {str(e)}")
        return jsonify({"error": "Failed to find matching mappings", "details": str(e)}), 500
