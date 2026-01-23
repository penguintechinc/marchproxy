"""Kong Plugins API endpoints."""
from quart import jsonify, request
from flask_security import auth_required, current_user
from app_quart.api.v1 import v1_bp
from app_quart.services.kong_client import KongClient
from app_quart.services.audit import AuditService
from app_quart.extensions import db
from app_quart.models.kong import KongPlugin


@v1_bp.route('/kong/plugins', methods=['GET'])
@auth_required('token')
async def list_kong_plugins():
    """List all Kong plugins."""
    client = KongClient()
    try:
        result = await client.list_plugins()
        return jsonify(result)
    finally:
        await client.close()


@v1_bp.route('/kong/plugins/enabled', methods=['GET'])
@auth_required('token')
async def list_enabled_plugins():
    """List all enabled plugin types."""
    client = KongClient()
    try:
        result = await client.get_enabled_plugins()
        return jsonify(result)
    finally:
        await client.close()


@v1_bp.route('/kong/plugins/schema/<plugin_name>', methods=['GET'])
@auth_required('token')
async def get_plugin_schema(plugin_name: str):
    """Get the configuration schema for a plugin."""
    client = KongClient()
    try:
        result = await client.get_plugin_schema(plugin_name)
        return jsonify(result)
    finally:
        await client.close()


@v1_bp.route('/kong/plugins/<plugin_id>', methods=['GET'])
@auth_required('token')
async def get_kong_plugin(plugin_id: str):
    """Get a specific Kong plugin."""
    client = KongClient()
    try:
        result = await client.get_plugin(plugin_id)
        return jsonify(result)
    finally:
        await client.close()


@v1_bp.route('/kong/plugins', methods=['POST'])
@auth_required('token')
async def create_kong_plugin():
    """Create a new Kong plugin."""
    data = await request.get_json()

    client = KongClient()
    try:
        kong_result = await client.create_plugin(data)

        db_plugin = KongPlugin(
            kong_id=kong_result.get('id'),
            name=kong_result.get('name'),
            config=kong_result.get('config'),
            enabled=kong_result.get('enabled', True),
            protocols=kong_result.get('protocols'),
            tags=kong_result.get('tags'),
            created_by=current_user.id
        )
        db.session.add(db_plugin)
        await db.session.commit()

        await AuditService.log(
            user_id=current_user.id,
            user_email=current_user.email,
            action='create',
            entity_type='kong_plugin',
            entity_id=kong_result.get('id'),
            entity_name=kong_result.get('name'),
            new_value=kong_result
        )

        return jsonify(kong_result), 201
    finally:
        await client.close()


@v1_bp.route('/kong/plugins/<plugin_id>', methods=['PATCH'])
@auth_required('token')
async def update_kong_plugin(plugin_id: str):
    """Update a Kong plugin."""
    data = await request.get_json()

    client = KongClient()
    try:
        old_value = await client.get_plugin(plugin_id)
        kong_result = await client.update_plugin(plugin_id, data)

        db_plugin = KongPlugin.query.filter_by(kong_id=plugin_id).first()
        if db_plugin:
            for key, value in kong_result.items():
                if hasattr(db_plugin, key):
                    setattr(db_plugin, key, value)
            await db.session.commit()

        await AuditService.log(
            user_id=current_user.id,
            user_email=current_user.email,
            action='update',
            entity_type='kong_plugin',
            entity_id=plugin_id,
            entity_name=kong_result.get('name'),
            old_value=old_value,
            new_value=kong_result
        )

        return jsonify(kong_result)
    finally:
        await client.close()


@v1_bp.route('/kong/plugins/<plugin_id>', methods=['DELETE'])
@auth_required('token')
async def delete_kong_plugin(plugin_id: str):
    """Delete a Kong plugin."""
    client = KongClient()
    try:
        old_value = await client.get_plugin(plugin_id)
        await client.delete_plugin(plugin_id)

        db_plugin = KongPlugin.query.filter_by(kong_id=plugin_id).first()
        if db_plugin:
            db.session.delete(db_plugin)
            await db.session.commit()

        await AuditService.log(
            user_id=current_user.id,
            user_email=current_user.email,
            action='delete',
            entity_type='kong_plugin',
            entity_id=plugin_id,
            entity_name=old_value.get('name'),
            old_value=old_value
        )

        return '', 204
    finally:
        await client.close()
