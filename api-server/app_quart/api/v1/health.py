"""Health check endpoints."""
from quart import jsonify
from sqlalchemy import text
from app_quart.api.v1 import v1_bp
from app_quart.extensions import db


@v1_bp.route('/healthz', methods=['GET'])
async def healthz():
    """Health check endpoint."""
    return jsonify({'status': 'healthy'}), 200


@v1_bp.route('/readyz', methods=['GET'])
async def readyz():
    """Readiness check endpoint with database connectivity verification."""
    try:
        await db.session.execute(text('SELECT 1'))
        return jsonify({'status': 'ready', 'database': 'connected'}), 200
    except Exception as e:
        return jsonify({'status': 'not_ready', 'database': 'disconnected', 'error': str(e)}), 503
