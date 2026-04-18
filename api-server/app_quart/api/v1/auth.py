"""Authentication endpoints."""
from app_quart.api.v1 import v1_bp # noqa: F401
from app_quart.extensions import db # noqa: F401
from flask_security import auth_required, current_user, login_user, logout_user # noqa: F401
from quart import jsonify, request # noqa: F401


@v1_bp.route('/auth/me', methods=['GET'])
@auth_required('token')
async def get_current_user():
    """Get current authenticated user."""
    return jsonify({
        'id': current_user.id,
        'email': current_user.email,
        'username': current_user.username,
        'roles': [r.name for r in current_user.roles]
    })


@v1_bp.route('/auth/logout', methods=['POST'])
@auth_required('token')
async def logout():
    """Logout current user."""
    logout_user()
    return jsonify({'message': 'Logged out successfully'})
