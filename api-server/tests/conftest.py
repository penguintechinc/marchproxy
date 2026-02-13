"""
Pytest configuration and fixtures for API server tests.
"""
import asyncio
import os
from typing import AsyncGenerator, Generator
from datetime import datetime, timedelta

import pytest
from fastapi.testclient import TestClient
from httpx import AsyncClient
from sqlalchemy import create_engine
from sqlalchemy.orm import Session, sessionmaker
from sqlalchemy.ext.asyncio import AsyncSession, create_async_engine, async_sessionmaker

from app.main import app
from app.core.database import Base, get_db
from app.dependencies import get_current_user
from app.models.user import User
from app.models.cluster import Cluster
from app.services.auth_service import AuthService


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
async def engine():
    """Create test database engine."""
    engine = create_async_engine(TEST_DATABASE_URL, echo=False)

    # Create all tables
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.drop_all)
        await conn.run_sync(Base.metadata.create_all)

    yield engine

    # Drop all tables after tests
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.drop_all)

    await engine.dispose()


@pytest.fixture
async def db_session(engine) -> AsyncGenerator[AsyncSession, None]:
    """Create database session for each test."""
    async_session = async_sessionmaker(
        engine, class_=AsyncSession, expire_on_commit=False
    )

    async with async_session() as session:
        yield session
        await session.rollback()


@pytest.fixture
def client(db_session: AsyncSession) -> TestClient:
    """Create test client with database session override."""

    async def override_get_db():
        yield db_session

    app.dependency_overrides[get_db] = override_get_db

    with TestClient(app) as test_client:
        yield test_client

    app.dependency_overrides.clear()


@pytest.fixture
async def async_client(db_session: AsyncSession) -> AsyncGenerator[AsyncClient, None]:
    """Create async test client."""

    async def override_get_db():
        yield db_session

    app.dependency_overrides[get_db] = override_get_db

    async with AsyncClient(app=app, base_url="http://test") as ac:
        yield ac

    app.dependency_overrides.clear()


@pytest.fixture
async def admin_user(db_session: AsyncSession) -> User:
    """Create admin user for testing."""
    auth_service = AuthService(db_session)

    user = User(
        email="admin@test.com",
        username="admin",
        full_name="Admin User",
        hashed_password=auth_service.get_password_hash("Admin123!"),
        is_active=True,
        is_superuser=True,
        email_verified=True,
        totp_secret=None
    )

    db_session.add(user)
    await db_session.commit()
    await db_session.refresh(user)

    return user


@pytest.fixture
async def regular_user(db_session: AsyncSession) -> User:
    """Create regular user for testing."""
    auth_service = AuthService(db_session)

    user = User(
        email="user@test.com",
        username="testuser",
        full_name="Test User",
        hashed_password=auth_service.get_password_hash("User123!"),
        is_active=True,
        is_superuser=False,
        email_verified=True,
        totp_secret=None
    )

    db_session.add(user)
    await db_session.commit()
    await db_session.refresh(user)

    return user


@pytest.fixture
async def admin_token(admin_user: User) -> str:
    """Generate JWT token for admin user."""
    from app.services.auth_service import AuthService

    access_token = AuthService.create_access_token(
        data={"sub": admin_user.email}
    )
    return access_token


@pytest.fixture
async def user_token(regular_user: User) -> str:
    """Generate JWT token for regular user."""
    from app.services.auth_service import AuthService

    access_token = AuthService.create_access_token(
        data={"sub": regular_user.email}
    )
    return access_token


@pytest.fixture
async def test_cluster(db_session: AsyncSession, admin_user: User) -> Cluster:
    """Create test cluster."""
    cluster = Cluster(
        name="test-cluster",
        description="Test cluster",
        tier="community",
        api_key="test-api-key-12345",
        max_proxies=3,
        created_by_id=admin_user.id,
        is_active=True
    )

    db_session.add(cluster)
    await db_session.commit()
    await db_session.refresh(cluster)

    return cluster


@pytest.fixture
def auth_headers(admin_token: str) -> dict:
    """Generate authentication headers."""
    return {"Authorization": f"Bearer {admin_token}"}


@pytest.fixture
def user_auth_headers(user_token: str) -> dict:
    """Generate authentication headers for regular user."""
    return {"Authorization": f"Bearer {user_token}"}
