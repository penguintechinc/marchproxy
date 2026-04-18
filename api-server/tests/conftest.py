"""
Pytest configuration and fixtures for API server tests.
"""
import asyncio # noqa: F401, # noqa: F401
import os # noqa: F401, # noqa: F401
from datetime import datetime, timedelta # noqa: F401
from typing import AsyncGenerator, Generator # noqa: F401

import pytest # noqa: F401, # noqa: F401
from app_quart.extensions import db # noqa: F401
from app_quart.main import create_app # noqa: F401
from app_quart.models.user import Role, User # noqa: F401
from httpx import AsyncClient # noqa: F401
from sqlalchemy import create_engine # noqa: F401
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker, create_async_engine # noqa: F401
from sqlalchemy.orm import Session, sessionmaker # noqa: F401

# Test database URL - use test database
TEST_DATABASE_URL = os.getenv(
    "TEST_DATABASE_URL",
    "postgresql+asyncpg://marchproxy:marchproxy@localhost:5432/marchproxy_test"
)


@pytest.fixture(scope="session")
def event_loop() -> Generator:
    """Create event loop for async tests."""
    loop = asyncio.get_event_loop_policy().new_event_loop()
    yield loop
    loop.close()


@pytest.fixture(scope="session")
async def app():
    """Create Quart application for testing."""
    test_app = create_app()
    test_app.config["TESTING"] = True
    return test_app


@pytest.fixture
async def db_session() -> AsyncGenerator[AsyncSession, None]:
    """Create database session for each test."""
    # Use the app's database connection
    async with db.engine.begin() as conn:
        await conn.run_sync(db.Model.metadata.drop_all)
        await conn.run_sync(db.Model.metadata.create_all)

    async_session = async_sessionmaker(
        db.engine, class_=AsyncSession, expire_on_commit=False
    )

    async with async_session() as session:
        yield session
        await session.rollback()

    async with db.engine.begin() as conn:
        await conn.run_sync(db.Model.metadata.drop_all)


@pytest.fixture
async def async_client(app) -> AsyncGenerator[AsyncClient, None]:
    """Create async test client for Quart app."""
    async with AsyncClient(app=app, base_url="http://test") as ac:
        yield ac


@pytest.fixture
async def admin_user(db_session: AsyncSession) -> User:
    """Create admin user for testing."""
    user = User(
        email="admin@test.com",
        username="admin",
        first_name="Admin",
        last_name="User",
        password="Admin123!",
        active=True,
        fs_uniquifier="admin-uniquifier"
    )

    db_session.add(user)
    await db_session.commit()
    await db_session.refresh(user)

    return user


@pytest.fixture
async def regular_user(db_session: AsyncSession) -> User:
    """Create regular user for testing."""
    user = User(
        email="user@test.com",
        username="testuser",
        first_name="Test",
        last_name="User",
        password="User123!",
        active=True,
        fs_uniquifier="user-uniquifier"
    )

    db_session.add(user)
    await db_session.commit()
    await db_session.refresh(user)

    return user


@pytest.fixture
async def admin_token(admin_user: User) -> str:
    """Generate JWT token for admin user."""
    # Use Flask-Security to generate token
    from flask_security import generate_confirmation_token
    token = generate_confirmation_token(admin_user.email)
    return token


@pytest.fixture
async def user_token(regular_user: User) -> str:
    """Generate JWT token for regular user."""
    from flask_security import generate_confirmation_token
    token = generate_confirmation_token(regular_user.email)
    return token


@pytest.fixture
def auth_headers(admin_token: str) -> dict:
    """Generate authentication headers."""
    return {"Authorization": f"Bearer {admin_token}"}


@pytest.fixture
def user_auth_headers(user_token: str) -> dict:
    """Generate authentication headers for regular user."""
    return {"Authorization": f"Bearer {user_token}"}
