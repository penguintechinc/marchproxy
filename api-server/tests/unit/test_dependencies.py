"""Unit tests for app/dependencies.py"""
from unittest.mock import AsyncMock, MagicMock, patch # noqa: F401

import pytest # noqa: F401, # noqa: F401
from fastapi import HTTPException # noqa: F401


@pytest.mark.asyncio
async def test_require_admin_passes_for_admin():
    from app.dependencies import require_admin # noqa: F401
    user = MagicMock()
    user.is_admin = True
    result = await require_admin(user)
    assert result is user


@pytest.mark.asyncio
async def test_require_admin_raises_403_for_non_admin():
    from app.dependencies import require_admin # noqa: F401
    user = MagicMock()
    user.is_admin = False
    with pytest.raises(HTTPException) as exc_info:
        await require_admin(user)
    assert exc_info.value.status_code == 403


@pytest.mark.asyncio
async def test_require_admin_403_detail():
    from app.dependencies import require_admin # noqa: F401
    user = MagicMock()
    user.is_admin = False
    with pytest.raises(HTTPException) as exc_info:
        await require_admin(user)
    assert "Admin" in exc_info.value.detail or "admin" in exc_info.value.detail


@pytest.mark.asyncio
async def test_get_current_user_raises_401_on_invalid_token():
    from app.dependencies import get_current_user # noqa: F401
    from fastapi.security import HTTPAuthorizationCredentials # noqa: F401

    credentials = MagicMock(spec=HTTPAuthorizationCredentials)
    credentials.credentials = "invalid.jwt.token"
    db = AsyncMock()

    with patch("app.dependencies.decode_token", side_effect=Exception("invalid")):
        with pytest.raises(HTTPException) as exc_info:
            await get_current_user(credentials, db)
        assert exc_info.value.status_code == 401


@pytest.mark.asyncio
async def test_get_current_user_raises_401_when_sub_missing():
    from app.dependencies import get_current_user # noqa: F401
    from fastapi.security import HTTPAuthorizationCredentials # noqa: F401

    credentials = MagicMock(spec=HTTPAuthorizationCredentials)
    credentials.credentials = "some.token.here"
    db = AsyncMock()

 # Token decodes but has no "sub"
    with patch("app.dependencies.decode_token", return_value={"type": "access"}):
        with pytest.raises(HTTPException) as exc_info:
            await get_current_user(credentials, db)
        assert exc_info.value.status_code == 401


@pytest.mark.asyncio
async def test_get_current_user_raises_404_when_user_not_found():
    from app.dependencies import get_current_user # noqa: F401
    from fastapi.security import HTTPAuthorizationCredentials # noqa: F401

    credentials = MagicMock(spec=HTTPAuthorizationCredentials)
    credentials.credentials = "valid.token.here"
    db = AsyncMock()

 # Token decodes with sub, but user not in DB
    mock_result = MagicMock()
    mock_result.scalar_one_or_none.return_value = None
    db.execute = AsyncMock(return_value=mock_result)

    with patch("app.dependencies.decode_token", return_value={"sub": "999", "type": "access"}):
        with pytest.raises(HTTPException) as exc_info:
            await get_current_user(credentials, db)
        assert exc_info.value.status_code == 404


@pytest.mark.asyncio
async def test_get_current_user_raises_404_when_user_inactive():
    from app.dependencies import get_current_user # noqa: F401
    from fastapi.security import HTTPAuthorizationCredentials # noqa: F401

    credentials = MagicMock(spec=HTTPAuthorizationCredentials)
    credentials.credentials = "valid.token.here"
    db = AsyncMock()

 # User exists but is inactive
    inactive_user = MagicMock()
    inactive_user.is_active = False
    mock_result = MagicMock()
    mock_result.scalar_one_or_none.return_value = inactive_user
    db.execute = AsyncMock(return_value=mock_result)

    with patch("app.dependencies.decode_token", return_value={"sub": "1", "type": "access"}):
        with pytest.raises(HTTPException) as exc_info:
            await get_current_user(credentials, db)
        assert exc_info.value.status_code == 404


@pytest.mark.asyncio
async def test_get_current_user_returns_user_on_success():
    from app.dependencies import get_current_user # noqa: F401
    from fastapi.security import HTTPAuthorizationCredentials # noqa: F401

    credentials = MagicMock(spec=HTTPAuthorizationCredentials)
    credentials.credentials = "valid.token.here"
    db = AsyncMock()

    active_user = MagicMock()
    active_user.is_active = True
    mock_result = MagicMock()
    mock_result.scalar_one_or_none.return_value = active_user
    db.execute = AsyncMock(return_value=mock_result)

    with patch("app.dependencies.decode_token", return_value={"sub": "1", "type": "access"}):
        result = await get_current_user(credentials, db)
    assert result is active_user


@pytest.mark.asyncio
async def test_validate_license_feature_raises_402_without_key():
    from app.dependencies import validate_license_feature # noqa: F401

    with pytest.raises(HTTPException) as exc_info:
        await validate_license_feature("unlimited_proxies", x_license_key=None)
    assert exc_info.value.status_code == 402


@pytest.mark.asyncio
async def test_validate_license_feature_raises_402_on_invalid_license():
    from app.dependencies import validate_license_feature # noqa: F401

    mock_validation = {"valid": False, "tier": "community", "features": []}
    with patch("app.dependencies.license_manager") as mock_mgr:
        mock_mgr.validate_license = AsyncMock(return_value=mock_validation)
        with pytest.raises(HTTPException) as exc_info:
            await validate_license_feature("unlimited_proxies", x_license_key="bad-key")
        assert exc_info.value.status_code == 402


@pytest.mark.asyncio
async def test_validate_license_feature_raises_402_feature_not_in_license():
    from app.dependencies import validate_license_feature # noqa: F401

    mock_validation = {"valid": True, "tier": "enterprise", "features": ["other_feature"]}
    with patch("app.dependencies.license_manager") as mock_mgr:
        mock_mgr.validate_license = AsyncMock(return_value=mock_validation)
        with pytest.raises(HTTPException) as exc_info:
            await validate_license_feature("unlimited_proxies", x_license_key="some-key")
        assert exc_info.value.status_code == 402


@pytest.mark.asyncio
async def test_validate_license_feature_returns_true_on_valid():
    from app.dependencies import validate_license_feature # noqa: F401

    mock_validation = {
        "valid": True,
        "tier": "enterprise",
        "features": ["unlimited_proxies", "saml"]
    }
    with patch("app.dependencies.license_manager") as mock_mgr:
        mock_mgr.validate_license = AsyncMock(return_value=mock_validation)
        result = await validate_license_feature(
            "unlimited_proxies", x_license_key="valid-key"
        )
    assert result is True
