"""Kong Routes API endpoints."""
from quart import jsonify, request
from flask_security import auth_required, current_user
from app_quart.api.v1 import v1_bp
from app_quart.services.kong_client import KongClient
from app_quart.services.audit import AuditService
from app_quart.extensions import db
from app_quart.models.kong import KongRoute


@v1_bp.route('/kong/routes', methods=['GET'])
@auth_required('token')
async def list_kong_routes():
    """List all Kong routes."""
    offset = request.args.get('offset', 0, type=int)
    size = request.args.get('size', 100, type=int)

    client = KongClient()
    try:
        result = await client.list_routes(offset=offset, size=size)
        return jsonify(result)
    finally:
        await client.close()


@v1_bp.route('/kong/routes/<route_id>', methods=['GET'])
@auth_required('token')
async def get_kong_route(route_id: str):
    """Get a specific Kong route."""
    client = KongClient()
    try:
        result = await client.get_route(route_id)
        return jsonify(result)
    finally:
        await client.close()


@v1_bp.route('/kong/routes', methods=['POST'])
@auth_required('token')
async def create_kong_route():
    """Create a new Kong route."""
    data = await request.get_json()

    client = KongClient()
    try:
        kong_result = await client.create_route(data)

        # Save to database
        db_route = KongRoute(
            kong_id=kong_result.get('id'),
            name=kong_result.get('name'),
            protocols=kong_result.get('protocols'),
            methods=kong_result.get('methods'),
            hosts=kong_result.get('hosts'),
            paths=kong_result.get('paths'),
            headers=kong_result.get('headers'),
            strip_path=kong_result.get('strip_path', True),
            preserve_host=kong_result.get('preserve_host', False),
            regex_priority=kong_result.get('regex_priority', 0),
            tags=kong_result.get('tags'),
            created_by=current_user.id
        )
        db.session.add(db_route)
        await db.session.commit()

        await AuditService.log(
            user_id=current_user.id,
            user_email=current_user.email,
            action='create',
            entity_type='kong_route',
            entity_id=kong_result.get('id'),
            entity_name=kong_result.get('name'),
            new_value=kong_result
        )

        return jsonify(kong_result), 201
    finally:
        await client.close()


@v1_bp.route('/kong/routes/<route_id>', methods=['PATCH'])
@auth_required('token')
async def update_kong_route(route_id: str):
    """Update a Kong route."""
    data = await request.get_json()

    client = KongClient()
    try:
        old_value = await client.get_route(route_id)
        kong_result = await client.update_route(route_id, data)

        db_route = KongRoute.query.filter_by(kong_id=route_id).first()
        if db_route:
            for key, value in kong_result.items():
                if hasattr(db_route, key):
                    setattr(db_route, key, value)
            await db.session.commit()

        await AuditService.log(
            user_id=current_user.id,
            user_email=current_user.email,
            action='update',
            entity_type='kong_route',
            entity_id=route_id,
            entity_name=kong_result.get('name'),
            old_value=old_value,
            new_value=kong_result
        )

        return jsonify(kong_result)
    finally:
        await client.close()


@v1_bp.route('/kong/routes/<route_id>', methods=['DELETE'])
@auth_required('token')
async def delete_kong_route(route_id: str):
    """Delete a Kong route."""
    client = KongClient()
    try:
        old_value = await client.get_route(route_id)
        await client.delete_route(route_id)

        db_route = KongRoute.query.filter_by(kong_id=route_id).first()
        if db_route:
            db.session.delete(db_route)
            await db.session.commit()

        await AuditService.log(
            user_id=current_user.id,
            user_email=current_user.email,
            action='delete',
            entity_type='kong_route',
            entity_id=route_id,
            entity_name=old_value.get('name'),
            old_value=old_value
        )

        return '', 204
    finally:
        await client.close()
