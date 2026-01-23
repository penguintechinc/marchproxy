"""
Authentication API Blueprint for MarchProxy Manager (Quart)

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

from quart import Blueprint, request, current_app, jsonify
from pydantic import ValidationError
import logging
from datetime import datetime
from models.auth import (
    UserModel,
    SessionModel,
    JWTManager,
    LoginRequest,
    RegisterRequest,
    Enable2FARequest,
    Verify2FARequest,
    TokenResponse,
    UserResponse,
)
from middleware.auth import require_auth

logger = logging.getLogger(__name__)

auth_bp = Blueprint("auth", __name__)


@auth_bp.route("/login", methods=["POST"])
async def login():
    """User login endpoint"""
    try:
        data_json = await request.get_json()
        data = LoginRequest(**data_json)
    except ValidationError as e:
        return jsonify({"error": "Validation error", "details": str(e)}), 400

    db = current_app.db
    jwt_manager = current_app.jwt_manager

    # Find user
    user = db(db.users.username == data.username).select().first()
    if not user or not UserModel.verify_password(data.password, user.password_hash):
        return jsonify({"error": "Invalid credentials"}), 401

    # Check if user is active
    if not user.is_active:
        return jsonify({"error": "Account is disabled"}), 403

    # Check 2FA if enabled
    if user.totp_enabled:
        if not data.totp_code:
            return jsonify({"error": "TOTP code required"}), 422

        if not UserModel.verify_totp(user.totp_secret, data.totp_code):
            return jsonify({"error": "Invalid TOTP code"}), 401

    # Update last login
    user.update_record(last_login=datetime.utcnow())

    # Create session
    session_id = SessionModel.create_session(
        db,
        user.id,
        ip_address=request.remote_addr,
        user_agent=request.headers.get("User-Agent"),
    )

    # Create JWT tokens
    access_payload = {
        "user_id": user.id,
        "username": user.username,
        "is_admin": user.is_admin,
        "session_id": session_id,
        "type": "access",
    }
    access_token = jwt_manager.create_token(access_payload)
    refresh_token = jwt_manager.create_refresh_token(user.id)

    response = TokenResponse(
        access_token=access_token,
        refresh_token=refresh_token,
        expires_in=jwt_manager.ttl_hours * 3600,
    )
    return jsonify(response.dict()), 200


@auth_bp.route("/logout", methods=["POST"])
@require_auth()
async def logout(user_data):
    """User logout endpoint"""
    db = current_app.db
    session_id = user_data.get("session_id")

    if session_id:
        SessionModel.destroy_session(db, session_id)

    return jsonify({"message": "Logged out successfully"}), 200


@auth_bp.route("/register", methods=["POST"])
@require_auth(admin_required=True)
async def register(user_data):
    """User registration endpoint (admin only)"""
    try:
        data_json = await request.get_json()
        data = RegisterRequest(**data_json)
    except ValidationError as e:
        return jsonify({"error": "Validation error", "details": str(e)}), 400

    db = current_app.db

    # Check if username or email already exists
    existing = (
        db((db.users.username == data.username) | (db.users.email == data.email))
        .select()
        .first()
    )

    if existing:
        return jsonify({"error": "Username or email already exists"}), 409

    # Create user
    password_hash = UserModel.hash_password(data.password)
    user_id = db.users.insert(
        username=data.username, email=data.email, password_hash=password_hash
    )

    user = db.users[user_id]
    response = UserResponse(
        id=user.id,
        username=user.username,
        email=user.email,
        is_admin=user.is_admin,
        is_active=user.is_active,
        totp_enabled=user.totp_enabled,
        auth_provider=user.auth_provider,
        created_at=user.created_at,
    )
    return jsonify(response.dict()), 201


@auth_bp.route("/refresh", methods=["POST"])
async def refresh():
    """Refresh access token"""
    data = await request.get_json()
    refresh_token = data.get("refresh_token")

    if not refresh_token:
        return jsonify({"error": "Refresh token required"}), 400

    jwt_manager = current_app.jwt_manager
    new_access_token = jwt_manager.refresh_access_token(refresh_token)

    if not new_access_token:
        return jsonify({"error": "Invalid or expired refresh token"}), 401

    return (
        jsonify(
            {
                "access_token": new_access_token,
                "token_type": "Bearer",
                "expires_in": jwt_manager.ttl_hours * 3600,
            }
        ),
        200,
    )


@auth_bp.route("/2fa/enable", methods=["POST"])
@require_auth()
async def enable_2fa(user_data):
    """Enable 2FA for user"""
    try:
        data_json = await request.get_json()
        data = Enable2FARequest(**data_json)
    except ValidationError as e:
        return jsonify({"error": "Validation error", "details": str(e)}), 400

    db = current_app.db
    user_id = user_data["user_id"]

    # Verify current password
    user = db.users[user_id]
    if not UserModel.verify_password(data.password, user.password_hash):
        return jsonify({"error": "Invalid password"}), 401

    # Generate TOTP secret
    totp_secret = UserModel.generate_totp_secret()
    totp_uri = UserModel.get_totp_uri(totp_secret, user.username)

    # Store secret (but don't enable yet)
    user.update_record(totp_secret=totp_secret)

    return (
        jsonify(
            {
                "secret": totp_secret,
                "qr_uri": totp_uri,
                "message": "Scan QR code and verify with TOTP code to complete setup",
            }
        ),
        200,
    )


@auth_bp.route("/2fa/verify", methods=["POST"])
@require_auth()
async def verify_2fa(user_data):
    """Verify and complete 2FA setup"""
    try:
        data_json = await request.get_json()
        data = Verify2FARequest(**data_json)
    except ValidationError as e:
        return jsonify({"error": "Validation error", "details": str(e)}), 400

    db = current_app.db
    user_id = user_data["user_id"]

    # Verify TOTP code
    if not UserModel.verify_totp(data.secret, data.totp_code):
        return jsonify({"error": "Invalid TOTP code"}), 400

    # Enable 2FA for user
    user = db.users[user_id]
    user.update_record(totp_enabled=True)

    return jsonify({"message": "2FA enabled successfully"}), 200


@auth_bp.route("/2fa/disable", methods=["POST"])
@require_auth()
async def disable_2fa(user_data):
    """Disable 2FA for user"""
    data = await request.get_json()
    db = current_app.db
    user_id = user_data["user_id"]

    # Verify current password
    user = db.users[user_id]
    if not UserModel.verify_password(data.get("password", ""), user.password_hash):
        return jsonify({"error": "Invalid password"}), 401

    # Verify TOTP code if 2FA is currently enabled
    if user.totp_enabled:
        totp_code = data.get("totp_code")
        if not totp_code or not UserModel.verify_totp(user.totp_secret, totp_code):
            return jsonify({"error": "Invalid TOTP code"}), 400

    # Disable 2FA
    user.update_record(totp_enabled=False, totp_secret=None)

    return jsonify({"message": "2FA disabled successfully"}), 200


@auth_bp.route("/profile", methods=["GET", "PUT"])
@require_auth()
async def profile(user_data):
    """Get/update user profile"""
    db = current_app.db
    user_id = user_data["user_id"]
    user = db.users[user_id]

    if request.method == "GET":
        response = UserResponse(
            id=user.id,
            username=user.username,
            email=user.email,
            is_admin=user.is_admin,
            is_active=user.is_active,
            totp_enabled=user.totp_enabled,
            auth_provider=user.auth_provider,
            created_at=user.created_at,
        )
        return jsonify(response.dict()), 200

    elif request.method == "PUT":
        data = await request.get_json()
        update_data = {}

        # Allow updating email
        if "email" in data:
            update_data["email"] = data["email"]

        # Allow updating password with current password verification
        if "new_password" in data:
            current_password = data.get("current_password")
            if not current_password or not UserModel.verify_password(
                current_password, user.password_hash
            ):
                return (
                    jsonify({"error": "Current password required for password change"}),
                    401,
                )

            update_data["password_hash"] = UserModel.hash_password(data["new_password"])

        if update_data:
            user.update_record(**update_data)

        return jsonify({"message": "Profile updated successfully"}), 200
