"""Kong Configuration Import/Export API endpoints."""
import yaml
import hashlib
from quart import jsonify, request
from flask_security import auth_required, current_user
from app_quart.api.v1 import v1_bp
from app_quart.services.kong_client import KongClient
from app_quart.services.audit import AuditService
from app_quart.extensions import db
from app_quart.models.kong import KongConfigHistory


@v1_bp.route('/kong/config', methods=['GET'])
@auth_required('token')
async def get_kong_config():
    """Export current Kong configuration as YAML."""
    client = KongClient()
    try:
        # Fetch all entities from Kong
        services = await client.list_services()
        routes = await client.list_routes()
        upstreams = await client.list_upstreams()
        consumers = await client.list_consumers()
        plugins = await client.list_plugins()
        certificates = await client.list_certificates()

        # Build declarative config
        config = {
            '_format_version': '3.0',
            'services': services.get('data', []),
            'routes': routes.get('data', []),
            'upstreams': upstreams.get('data', []),
            'consumers': consumers.get('data', []),
            'plugins': plugins.get('data', []),
            'certificates': certificates.get('data', [])
        }

        yaml_content = yaml.dump(config, default_flow_style=False, sort_keys=False)

        return yaml_content, 200, {'Content-Type': 'text/yaml'}
    finally:
        await client.close()


@v1_bp.route('/kong/config/validate', methods=['POST'])
@auth_required('token')
async def validate_kong_config():
    """Validate a Kong configuration YAML without applying it."""
    content_type = request.content_type or ''

    if 'yaml' in content_type or 'text/plain' in content_type:
        yaml_content = (await request.get_data()).decode('utf-8')
    else:
        data = await request.get_json()
        yaml_content = data.get('config', '')

    try:
        parsed = yaml.safe_load(yaml_content)

        # Validate format version
        format_version = parsed.get('_format_version')
        if not format_version:
            return jsonify({'valid': False, 'error': 'Missing _format_version'}), 400

        # Count entities
        stats = {
            'services': len(parsed.get('services', [])),
            'routes': len(parsed.get('routes', [])),
            'upstreams': len(parsed.get('upstreams', [])),
            'consumers': len(parsed.get('consumers', [])),
            'plugins': len(parsed.get('plugins', [])),
            'certificates': len(parsed.get('certificates', []))
        }

        return jsonify({
            'valid': True,
            'format_version': format_version,
            'stats': stats
        })
    except yaml.YAMLError as e:
        return jsonify({'valid': False, 'error': f'Invalid YAML: {str(e)}'}), 400


@v1_bp.route('/kong/config/preview', methods=['POST'])
@auth_required('token')
async def preview_kong_config():
    """Preview changes that would be made by applying a config."""
    content_type = request.content_type or ''

    if 'yaml' in content_type or 'text/plain' in content_type:
        yaml_content = (await request.get_data()).decode('utf-8')
    else:
        data = await request.get_json()
        yaml_content = data.get('config', '')

    try:
        new_config = yaml.safe_load(yaml_content)
    except yaml.YAMLError as e:
        return jsonify({'error': f'Invalid YAML: {str(e)}'}), 400

    client = KongClient()
    try:
        # Fetch current state
        current_services = await client.list_services()
        current_routes = await client.list_routes()
        current_upstreams = await client.list_upstreams()
        current_consumers = await client.list_consumers()
        current_plugins = await client.list_plugins()

        def diff_entities(current_list, new_list, key='name'):
            current_names = {e.get(key) for e in current_list}
            new_names = {e.get(key) for e in new_list}

            return {
                'added': list(new_names - current_names),
                'removed': list(current_names - new_names),
                'unchanged': list(current_names & new_names)
            }

        preview = {
            'services': diff_entities(
                current_services.get('data', []),
                new_config.get('services', [])
            ),
            'routes': diff_entities(
                current_routes.get('data', []),
                new_config.get('routes', [])
            ),
            'upstreams': diff_entities(
                current_upstreams.get('data', []),
                new_config.get('upstreams', [])
            ),
            'consumers': diff_entities(
                current_consumers.get('data', []),
                new_config.get('consumers', []),
                key='username'
            ),
            'plugins': diff_entities(
                current_plugins.get('data', []),
                new_config.get('plugins', []),
                key='id'
            )
        }

        return jsonify(preview)
    finally:
        await client.close()


@v1_bp.route('/kong/config', methods=['POST'])
@auth_required('token')
async def apply_kong_config():
    """Apply a Kong configuration YAML."""
    content_type = request.content_type or ''

    if 'yaml' in content_type or 'text/plain' in content_type:
        yaml_content = (await request.get_data()).decode('utf-8')
    else:
        data = await request.get_json()
        yaml_content = data.get('config', '')

    try:
        parsed = yaml.safe_load(yaml_content)
    except yaml.YAMLError as e:
        return jsonify({'error': f'Invalid YAML: {str(e)}'}), 400

    client = KongClient()
    try:
        # Apply to Kong
        result = await client.post_config(yaml_content)

        # Calculate hash for deduplication
        config_hash = hashlib.sha256(yaml_content.encode()).hexdigest()

        # Mark previous configs as not current
        KongConfigHistory.query.filter_by(is_current=True).update({'is_current': False})

        # Save to history
        history = KongConfigHistory(
            config_yaml=yaml_content,
            config_hash=config_hash,
            description=request.args.get('description', 'Config upload'),
            applied_by=current_user.id,
            is_current=True,
            services_count=len(parsed.get('services', [])),
            routes_count=len(parsed.get('routes', [])),
            plugins_count=len(parsed.get('plugins', []))
        )
        db.session.add(history)
        await db.session.commit()

        await AuditService.log(
            user_id=current_user.id,
            user_email=current_user.email,
            action='apply_config',
            entity_type='kong_config',
            entity_id=str(history.id),
            new_value={'hash': config_hash, 'stats': {
                'services': history.services_count,
                'routes': history.routes_count,
                'plugins': history.plugins_count
            }}
        )

        return jsonify({
            'success': True,
            'history_id': history.id,
            'hash': config_hash
        })
    finally:
        await client.close()


@v1_bp.route('/kong/config/history', methods=['GET'])
@auth_required('token')
async def list_config_history():
    """List Kong configuration history."""
    limit = request.args.get('limit', 20, type=int)
    offset = request.args.get('offset', 0, type=int)

    query = KongConfigHistory.query.order_by(KongConfigHistory.applied_at.desc())
    total = query.count()
    configs = query.offset(offset).limit(limit).all()

    return jsonify({
        'data': [{
            'id': c.id,
            'description': c.description,
            'applied_at': c.applied_at.isoformat() if c.applied_at else None,
            'applied_by': c.applied_by,
            'is_current': c.is_current,
            'services_count': c.services_count,
            'routes_count': c.routes_count,
            'plugins_count': c.plugins_count,
            'hash': c.config_hash
        } for c in configs],
        'total': total,
        'offset': offset,
        'limit': limit
    })


@v1_bp.route('/kong/config/history/<int:history_id>', methods=['GET'])
@auth_required('token')
async def get_config_history(history_id: int):
    """Get a specific configuration from history."""
    config = KongConfigHistory.query.get_or_404(history_id)

    return config.config_yaml, 200, {'Content-Type': 'text/yaml'}


@v1_bp.route('/kong/config/rollback/<int:history_id>', methods=['POST'])
@auth_required('token')
async def rollback_config(history_id: int):
    """Rollback to a previous configuration."""
    config = KongConfigHistory.query.get_or_404(history_id)

    client = KongClient()
    try:
        # Apply the historical config
        await client.post_config(config.config_yaml)

        # Update current flags
        KongConfigHistory.query.filter_by(is_current=True).update({'is_current': False})
        config.is_current = True
        await db.session.commit()

        await AuditService.log(
            user_id=current_user.id,
            user_email=current_user.email,
            action='rollback_config',
            entity_type='kong_config',
            entity_id=str(history_id)
        )

        return jsonify({'success': True, 'rolled_back_to': history_id})
    finally:
        await client.close()


@v1_bp.route('/kong/status', methods=['GET'])
@auth_required('token')
async def get_kong_status():
    """Get Kong gateway status."""
    client = KongClient()
    try:
        status = await client.get_status()
        return jsonify(status)
    finally:
        await client.close()
