"""Kong Upstreams and Targets API endpoints."""
from quart import jsonify, request
from flask_security import auth_required, current_user
from app_quart.api.v1 import v1_bp
from app_quart.services.kong_client import KongClient
from app_quart.services.audit import AuditService
from app_quart.extensions import db
from app_quart.models.kong import KongUpstream, KongTarget


# Upstreams
@v1_bp.route('/kong/upstreams', methods=['GET'])
@auth_required('token')
async def list_kong_upstreams():
    """List all Kong upstreams."""
    client = KongClient()
    try:
        result = await client.list_upstreams()
        return jsonify(result)
    finally:
        await client.close()


@v1_bp.route('/kong/upstreams/<upstream_id>', methods=['GET'])
@auth_required('token')
async def get_kong_upstream(upstream_id: str):
    """Get a specific Kong upstream."""
    client = KongClient()
    try:
        result = await client.get_upstream(upstream_id)
        return jsonify(result)
    finally:
        await client.close()


@v1_bp.route('/kong/upstreams', methods=['POST'])
@auth_required('token')
async def create_kong_upstream():
    """Create a new Kong upstream."""
    data = await request.get_json()

    client = KongClient()
    try:
        kong_result = await client.create_upstream(data)

        db_upstream = KongUpstream(
            kong_id=kong_result.get('id'),
            name=kong_result.get('name'),
            algorithm=kong_result.get('algorithm', 'round-robin'),
            hash_on=kong_result.get('hash_on', 'none'),
            hash_fallback=kong_result.get('hash_fallback', 'none'),
            slots=kong_result.get('slots', 10000),
            healthchecks=kong_result.get('healthchecks'),
            tags=kong_result.get('tags'),
            created_by=current_user.id
        )
        db.session.add(db_upstream)
        await db.session.commit()

        await AuditService.log(
            user_id=current_user.id,
            user_email=current_user.email,
            action='create',
            entity_type='kong_upstream',
            entity_id=kong_result.get('id'),
            entity_name=kong_result.get('name'),
            new_value=kong_result
        )

        return jsonify(kong_result), 201
    finally:
        await client.close()


@v1_bp.route('/kong/upstreams/<upstream_id>', methods=['PATCH'])
@auth_required('token')
async def update_kong_upstream(upstream_id: str):
    """Update a Kong upstream."""
    data = await request.get_json()

    client = KongClient()
    try:
        old_value = await client.get_upstream(upstream_id)
        kong_result = await client.update_upstream(upstream_id, data)

        db_upstream = KongUpstream.query.filter_by(kong_id=upstream_id).first()
        if db_upstream:
            for key, value in kong_result.items():
                if hasattr(db_upstream, key):
                    setattr(db_upstream, key, value)
            await db.session.commit()

        await AuditService.log(
            user_id=current_user.id,
            user_email=current_user.email,
            action='update',
            entity_type='kong_upstream',
            entity_id=upstream_id,
            entity_name=kong_result.get('name'),
            old_value=old_value,
            new_value=kong_result
        )

        return jsonify(kong_result)
    finally:
        await client.close()


@v1_bp.route('/kong/upstreams/<upstream_id>', methods=['DELETE'])
@auth_required('token')
async def delete_kong_upstream(upstream_id: str):
    """Delete a Kong upstream."""
    client = KongClient()
    try:
        old_value = await client.get_upstream(upstream_id)
        await client.delete_upstream(upstream_id)

        db_upstream = KongUpstream.query.filter_by(kong_id=upstream_id).first()
        if db_upstream:
            db.session.delete(db_upstream)
            await db.session.commit()

        await AuditService.log(
            user_id=current_user.id,
            user_email=current_user.email,
            action='delete',
            entity_type='kong_upstream',
            entity_id=upstream_id,
            entity_name=old_value.get('name'),
            old_value=old_value
        )

        return '', 204
    finally:
        await client.close()


# Targets
@v1_bp.route('/kong/upstreams/<upstream_id>/targets', methods=['GET'])
@auth_required('token')
async def list_kong_targets(upstream_id: str):
    """List all targets for an upstream."""
    client = KongClient()
    try:
        result = await client.list_targets(upstream_id)
        return jsonify(result)
    finally:
        await client.close()


@v1_bp.route('/kong/upstreams/<upstream_id>/targets', methods=['POST'])
@auth_required('token')
async def create_kong_target(upstream_id: str):
    """Create a new target for an upstream."""
    data = await request.get_json()

    client = KongClient()
    try:
        kong_result = await client.create_target(upstream_id, data)

        # Find the database upstream
        db_upstream = KongUpstream.query.filter_by(kong_id=upstream_id).first()

        db_target = KongTarget(
            kong_id=kong_result.get('id'),
            upstream_id=db_upstream.id if db_upstream else None,
            target=kong_result.get('target'),
            weight=kong_result.get('weight', 100),
            tags=kong_result.get('tags'),
            created_by=current_user.id
        )
        db.session.add(db_target)
        await db.session.commit()

        await AuditService.log(
            user_id=current_user.id,
            user_email=current_user.email,
            action='create',
            entity_type='kong_target',
            entity_id=kong_result.get('id'),
            entity_name=kong_result.get('target'),
            new_value=kong_result
        )

        return jsonify(kong_result), 201
    finally:
        await client.close()


@v1_bp.route('/kong/upstreams/<upstream_id>/targets/<target_id>', methods=['DELETE'])
@auth_required('token')
async def delete_kong_target(upstream_id: str, target_id: str):
    """Delete a target from an upstream."""
    client = KongClient()
    try:
        await client.delete_target(upstream_id, target_id)

        db_target = KongTarget.query.filter_by(kong_id=target_id).first()
        if db_target:
            db.session.delete(db_target)
            await db.session.commit()

        await AuditService.log(
            user_id=current_user.id,
            user_email=current_user.email,
            action='delete',
            entity_type='kong_target',
            entity_id=target_id
        )

        return '', 204
    finally:
        await client.close()
