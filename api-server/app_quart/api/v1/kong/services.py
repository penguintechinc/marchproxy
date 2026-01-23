"""Kong Services API endpoints."""
from quart import jsonify, request
from flask_security import auth_required, current_user
from app_quart.api.v1 import v1_bp
from app_quart.services.kong_client import KongClient
from app_quart.services.audit import AuditService
from app_quart.extensions import db
from app_quart.models.kong import KongService


@v1_bp.route('/kong/services', methods=['GET'])
@auth_required('token')
async def list_kong_services():
    """List all Kong services."""
    offset = request.args.get('offset', 0, type=int)
    size = request.args.get('size', 100, type=int)

    client = KongClient()
    try:
        result = await client.list_services(offset=offset, size=size)
        return jsonify(result)
    finally:
        await client.close()


@v1_bp.route('/kong/services/<service_id>', methods=['GET'])
@auth_required('token')
async def get_kong_service(service_id: str):
    """Get a specific Kong service."""
    client = KongClient()
    try:
        result = await client.get_service(service_id)
        return jsonify(result)
    finally:
        await client.close()


@v1_bp.route('/kong/services', methods=['POST'])
@auth_required('token')
async def create_kong_service():
    """Create a new Kong service."""
    data = await request.get_json()

    client = KongClient()
    try:
        # Create in Kong
        kong_result = await client.create_service(data)

        # Save to database for audit
        db_service = KongService(
            kong_id=kong_result.get('id'),
            name=kong_result.get('name'),
            protocol=kong_result.get('protocol', 'http'),
            host=kong_result.get('host'),
            port=kong_result.get('port', 80),
            path=kong_result.get('path'),
            retries=kong_result.get('retries', 5),
            connect_timeout=kong_result.get('connect_timeout', 60000),
            write_timeout=kong_result.get('write_timeout', 60000),
            read_timeout=kong_result.get('read_timeout', 60000),
            enabled=kong_result.get('enabled', True),
            tags=kong_result.get('tags'),
            created_by=current_user.id
        )
        db.session.add(db_service)
        await db.session.commit()

        # Audit log
        await AuditService.log(
            user_id=current_user.id,
            user_email=current_user.email,
            action='create',
            entity_type='kong_service',
            entity_id=kong_result.get('id'),
            entity_name=kong_result.get('name'),
            new_value=kong_result
        )

        return jsonify(kong_result), 201
    finally:
        await client.close()


@v1_bp.route('/kong/services/<service_id>', methods=['PATCH'])
@auth_required('token')
async def update_kong_service(service_id: str):
    """Update a Kong service."""
    data = await request.get_json()

    client = KongClient()
    try:
        # Get old value for audit
        old_value = await client.get_service(service_id)

        # Update in Kong
        kong_result = await client.update_service(service_id, data)

        # Update in database
        db_service = KongService.query.filter_by(kong_id=service_id).first()
        if db_service:
            for key, value in kong_result.items():
                if hasattr(db_service, key):
                    setattr(db_service, key, value)
            await db.session.commit()

        # Audit log
        await AuditService.log(
            user_id=current_user.id,
            user_email=current_user.email,
            action='update',
            entity_type='kong_service',
            entity_id=service_id,
            entity_name=kong_result.get('name'),
            old_value=old_value,
            new_value=kong_result
        )

        return jsonify(kong_result)
    finally:
        await client.close()


@v1_bp.route('/kong/services/<service_id>', methods=['DELETE'])
@auth_required('token')
async def delete_kong_service(service_id: str):
    """Delete a Kong service."""
    client = KongClient()
    try:
        # Get old value for audit
        old_value = await client.get_service(service_id)

        # Delete from Kong
        await client.delete_service(service_id)

        # Delete from database
        db_service = KongService.query.filter_by(kong_id=service_id).first()
        if db_service:
            db.session.delete(db_service)
            await db.session.commit()

        # Audit log
        await AuditService.log(
            user_id=current_user.id,
            user_email=current_user.email,
            action='delete',
            entity_type='kong_service',
            entity_id=service_id,
            entity_name=old_value.get('name'),
            old_value=old_value
        )

        return '', 204
    finally:
        await client.close()
