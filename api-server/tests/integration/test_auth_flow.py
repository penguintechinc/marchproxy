"""
Integration tests for authentication flow.
"""
import pytest
from httpx import AsyncClient
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.user import User


@pytest.mark.asyncio
class TestAuthFlow:
    """Test complete authentication flow."""

    async def test_user_registration(self, async_client: AsyncClient):
        """Test user registration endpoint."""
        response = await async_client.post(
            "/api/v1/auth/register",
            json={
                "email": "newuser@test.com",
                "username": "newuser",
                "full_name": "New User",
                "password": "NewUser123!"
            }
        )

        assert response.status_code == 201
        data = response.json()
        assert data["email"] == "newuser@test.com"
        assert data["username"] == "newuser"
        assert "id" in data
        assert "hashed_password" not in data

    async def test_duplicate_registration(self, async_client: AsyncClient, admin_user: User):
        """Test duplicate user registration fails."""
        response = await async_client.post(
            "/api/v1/auth/register",
            json={
                "email": admin_user.email,
                "username": "different",
                "full_name": "Test",
                "password": "Test123!"
            }
        )

        assert response.status_code == 400
        assert "already exists" in response.json()["detail"].lower()

    async def test_login_success(self, async_client: AsyncClient, admin_user: User):
        """Test successful login."""
        response = await async_client.post(
            "/api/v1/auth/login",
            data={
                "username": admin_user.email,
                "password": "Admin123!"
            }
        )

        assert response.status_code == 200
        data = response.json()
        assert "access_token" in data
        assert "refresh_token" in data
        assert data["token_type"] == "bearer"

    async def test_login_invalid_credentials(self, async_client: AsyncClient):
        """Test login with invalid credentials."""
        response = await async_client.post(
            "/api/v1/auth/login",
            data={
                "username": "nonexistent@test.com",
                "password": "wrongpassword"
            }
        )

        assert response.status_code == 401
        assert "incorrect" in response.json()["detail"].lower()

    async def test_login_inactive_user(self, async_client: AsyncClient, db_session: AsyncSession):
        """Test login with inactive user account."""
        from app.services.auth_service import AuthService

        auth_service = AuthService(db_session)

        # Create inactive user
        user = User(
            email="inactive@test.com",
            username="inactive",
            full_name="Inactive User",
            hashed_password=auth_service.get_password_hash("Test123!"),
            is_active=False,
            is_superuser=False
        )
        db_session.add(user)
        await db_session.commit()

        response = await async_client.post(
            "/api/v1/auth/login",
            data={
                "username": "inactive@test.com",
                "password": "Test123!"
            }
        )

        assert response.status_code == 400
        assert "inactive" in response.json()["detail"].lower()

    async def test_get_current_user(self, async_client: AsyncClient, auth_headers: dict):
        """Test getting current user information."""
        response = await async_client.get(
            "/api/v1/auth/me",
            headers=auth_headers
        )

        assert response.status_code == 200
        data = response.json()
        assert data["email"] == "admin@test.com"
        assert data["is_superuser"] is True

    async def test_get_current_user_unauthorized(self, async_client: AsyncClient):
        """Test getting current user without authentication."""
        response = await async_client.get("/api/v1/auth/me")
        assert response.status_code == 401

    async def test_refresh_token(self, async_client: AsyncClient, admin_user: User):
        """Test token refresh flow."""
        # Login first
        login_response = await async_client.post(
            "/api/v1/auth/login",
            data={
                "username": admin_user.email,
                "password": "Admin123!"
            }
        )

        refresh_token = login_response.json()["refresh_token"]

        # Refresh token
        response = await async_client.post(
            "/api/v1/auth/refresh",
            json={"refresh_token": refresh_token}
        )

        assert response.status_code == 200
        data = response.json()
        assert "access_token" in data
        assert data["token_type"] == "bearer"

    async def test_change_password(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        admin_user: User
    ):
        """Test password change."""
        response = await async_client.post(
            "/api/v1/auth/change-password",
            headers=auth_headers,
            json={
                "old_password": "Admin123!",
                "new_password": "NewAdmin123!"
            }
        )

        assert response.status_code == 200

        # Try logging in with new password
        login_response = await async_client.post(
            "/api/v1/auth/login",
            data={
                "username": admin_user.email,
                "password": "NewAdmin123!"
            }
        )

        assert login_response.status_code == 200

    async def test_2fa_enrollment(self, async_client: AsyncClient, auth_headers: dict):
        """Test 2FA enrollment."""
        response = await async_client.post(
            "/api/v1/auth/2fa/enroll",
            headers=auth_headers
        )

        assert response.status_code == 200
        data = response.json()
        assert "secret" in data
        assert "qr_code" in data

    async def test_2fa_verify(
        self,
        async_client: AsyncClient,
        auth_headers: dict,
        db_session: AsyncSession,
        admin_user: User
    ):
        """Test 2FA verification."""
        import pyotp

        # Enroll first
        enroll_response = await async_client.post(
            "/api/v1/auth/2fa/enroll",
            headers=auth_headers
        )
        secret = enroll_response.json()["secret"]

        # Generate TOTP code
        totp = pyotp.TOTP(secret)
        code = totp.now()

        # Verify code
        response = await async_client.post(
            "/api/v1/auth/2fa/verify",
            headers=auth_headers,
            json={"code": code}
        )

        assert response.status_code == 200
        data = response.json()
        assert data["enabled"] is True

    async def test_logout(self, async_client: AsyncClient, auth_headers: dict):
        """Test logout endpoint."""
        response = await async_client.post(
            "/api/v1/auth/logout",
            headers=auth_headers
        )

        assert response.status_code == 200
        assert response.json()["message"] == "Successfully logged out"
