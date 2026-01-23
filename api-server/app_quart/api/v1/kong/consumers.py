"""Kong Consumers API endpoints."""
from quart import jsonify, request
from flask_security import auth_required, current_user
from app_quart.api.v1 import v1_bp
from app_quart.services.kong_client import KongClient
from app_quart.services.audit import AuditService
from app_quart.extensions import db
from app_quart.models.kong import KongConsumer


@v1_bp.route('/kong/consumers', methods=['GET'])
@auth_required('token')
async def list_kong_consumers():
    """List all Kong consumers."""
    client = KongClient()
    try:
        result = await client.list_consumers()
        return jsonify(result)
    finally:
        await client.close()


@v1_bp.route('/kong/consumers/<consumer_id>', methods=['GET'])
@auth_required('token')
async def get_kong_consumer(consumer_id: str):
    """Get a specific Kong consumer."""
    client = KongClient()
    try:
        result = await client.get_consumer(consumer_id)
        return jsonify(result)
    finally:
        await client.close()


@v1_bp.route('/kong/consumers', methods=['POST'])
@auth_required('token')
async def create_kong_consumer():
    """Create a new Kong consumer."""
    data = await request.get_json()

    client = KongClient()
    try:
        kong_result = await client.create_consumer(data)

        db_consumer = KongConsumer(
            kong_id=kong_result.get('id'),
            username=kong_result.get('username'),
            custom_id=kong_result.get('custom_id'),
            tags=kong_result.get('tags'),
            created_by=current_user.id
        )
        db.session.add(db_consumer)
        await db.session.commit()

        await AuditService.log(
            user_id=current_user.id,
            user_email=current_user.email,
            action='create',
            entity_type='kong_consumer',
            entity_id=kong_result.get('id'),
            entity_name=kong_result.get('username'),
            new_value=kong_result
        )

        return jsonify(kong_result), 201
    finally:
        await client.close()


@v1_bp.route('/kong/consumers/<consumer_id>', methods=['PATCH'])
@auth_required('token')
async def update_kong_consumer(consumer_id: str):
    """Update a Kong consumer."""
    data = await request.get_json()

    client = KongClient()
    try:
        old_value = await client.get_consumer(consumer_id)
        kong_result = await client.update_consumer(consumer_id, data)

        db_consumer = KongConsumer.query.filter_by(kong_id=consumer_id).first()
        if db_consumer:
            for key, value in kong_result.items():
                if hasattr(db_consumer, key):
                    setattr(db_consumer, key, value)
            await db.session.commit()

        await AuditService.log(
            user_id=current_user.id,
            user_email=current_user.email,
            action='update',
            entity_type='kong_consumer',
            entity_id=consumer_id,
            entity_name=kong_result.get('username'),
            old_value=old_value,
            new_value=kong_result
        )

        return jsonify(kong_result)
    finally:
        await client.close()


@v1_bp.route('/kong/consumers/<consumer_id>', methods=['DELETE'])
@auth_required('token')
async def delete_kong_consumer(consumer_id: str):
    """Delete a Kong consumer."""
    client = KongClient()
    try:
        old_value = await client.get_consumer(consumer_id)
        await client.delete_consumer(consumer_id)

        db_consumer = KongConsumer.query.filter_by(kong_id=consumer_id).first()
        if db_consumer:
            db.session.delete(db_consumer)
            await db.session.commit()

        await AuditService.log(
            user_id=current_user.id,
            user_email=current_user.email,
            action='delete',
            entity_type='kong_consumer',
            entity_id=consumer_id,
            entity_name=old_value.get('username'),
            old_value=old_value
        )

        return '', 204
    finally:
        await client.close()
