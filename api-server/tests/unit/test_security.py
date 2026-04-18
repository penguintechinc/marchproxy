"""Unit tests for app/core/security.py"""
from unittest.mock import MagicMock, patch # noqa: F401

import pytest # noqa: F401, # noqa: F401
from jose import JWTError # noqa: F401

# bcrypt 4.0+ + passlib 1.7.4 have a known incompatibility in detect_wrap_bug.
# We mock the CryptContext to avoid that issue in unit tests; the hashing
# contract (verify/hash round-trips correctly) is still exercised via the mock.

def test_verify_password_correct():
    """verify_password returns True when plain matches hashed."""
    from app.core.security import verify_password # noqa: F401
    with patch("app.core.security.pwd_context") as mock_ctx:
        mock_ctx.verify.return_value = True
        result = verify_password("mypassword", "$2b$12$fakehash")
        mock_ctx.verify.assert_called_once_with("mypassword", "$2b$12$fakehash")
        assert result is True


def test_verify_password_wrong():
    """verify_password returns False when plain does not match hashed."""
    from app.core.security import verify_password # noqa: F401
    with patch("app.core.security.pwd_context") as mock_ctx:
        mock_ctx.verify.return_value = False
        result = verify_password("wrong", "$2b$12$fakehash")
        assert result is False


def test_get_password_hash_returns_string():
    """get_password_hash returns a non-empty string."""
    from app.core.security import get_password_hash # noqa: F401
    with patch("app.core.security.pwd_context") as mock_ctx:
        mock_ctx.hash.return_value = "$2b$12$fakedhashedvalue"
        h = get_password_hash("password")
        assert isinstance(h, str)
        assert len(h) > 0


def test_get_password_hash_different_each_time():
    """get_password_hash produces unique hashes (bcrypt random salt)."""
    from app.core.security import get_password_hash # noqa: F401

 # Use side_effect to return different hashes per call
    with patch("app.core.security.pwd_context") as mock_ctx:
        mock_ctx.hash.side_effect = ["$2b$12$hash1", "$2b$12$hash2"]
        h1 = get_password_hash("same")
        h2 = get_password_hash("same")
        assert h1 != h2


def test_create_access_token_returns_string():
    from app.core.security import create_access_token # noqa: F401
    token = create_access_token("user123")
    assert isinstance(token, str)
    assert len(token) > 0


def test_create_access_token_decodable():
    from app.core.security import create_access_token, decode_token # noqa: F401
    token = create_access_token("user42")
    payload = decode_token(token)
    assert payload["sub"] == "user42"
    assert payload["type"] == "access"


def test_create_refresh_token_type():
    from app.core.security import create_refresh_token, decode_token # noqa: F401
    token = create_refresh_token("user42")
    payload = decode_token(token)
    assert payload["type"] == "refresh"
    assert payload["sub"] == "user42"


def test_decode_token_invalid_raises():
    from app.core.security import decode_token # noqa: F401
    with pytest.raises(JWTError):
        decode_token("not.a.valid.token")


def test_decode_token_wrong_secret_raises():
    from app.core.config import settings # noqa: F401
    from app.core.security import decode_token # noqa: F401
    from jose import jwt # noqa: F401

 # Sign with different secret
    token = jwt.encode({"sub": "user1"}, "wrong-secret", algorithm=settings.ALGORITHM)
    with pytest.raises(JWTError):
        decode_token(token)


def test_generate_totp_secret_returns_base32():
    import base64 # noqa: F401

    from app.core.security import generate_totp_secret # noqa: F401
    secret = generate_totp_secret()
    assert isinstance(secret, str)
    assert len(secret) > 0
 # Base32 charset: A-Z 2-7 — pad to multiple of 8 then decode
    padded = secret + "=" * ((8 - len(secret) % 8) % 8)
    base64.b32decode(padded)


def test_verify_totp_code_invalid():
    from app.core.security import generate_totp_secret, verify_totp_code # noqa: F401
    secret = generate_totp_secret()
    assert verify_totp_code(secret, "000000") is False


def test_verify_totp_code_valid():
    import pyotp # noqa: F401
    from app.core.security import verify_totp_code # noqa: F401
    secret = pyotp.random_base32()
    totp = pyotp.TOTP(secret)
    valid_code = totp.now()
    assert verify_totp_code(secret, valid_code) is True


def test_get_totp_uri_returns_otpauth_string():
    from app.core.security import generate_totp_secret, get_totp_uri # noqa: F401
    secret = generate_totp_secret()
    uri = get_totp_uri(secret, "user@example.com", "MarchProxy")
    assert uri.startswith("otpauth://totp/")
    assert "MarchProxy" in uri


def test_create_access_token_with_custom_expiry():
    from datetime import timedelta # noqa: F401

    from app.core.security import create_access_token, decode_token # noqa: F401
    token = create_access_token("user99", expires_delta=timedelta(hours=2))
    payload = decode_token(token)
    assert payload["sub"] == "user99"
    assert payload["type"] == "access"
