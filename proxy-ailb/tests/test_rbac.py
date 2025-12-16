"""
Comprehensive tests for MarchProxy AILB RBAC System
Tests Role-Based Access Control, authentication, and authorization
"""

import pytest
import json
from datetime import datetime, timedelta
import sys
import os

# Add parent directory to path for imports
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

from app.auth.rbac import (
    RBACManager,
    Role,
    Permission,
    UserContext,
    hash_password,
    verify_password,
    AuthenticationError,
    AuthorizationError,
    ROLE_PERMISSIONS
)


class TestPasswordHashing:
    """Test password hashing and verification functions"""

    def test_hash_password_creates_different_hashes(self):
        """Test that same password creates different hashes with different salts"""
        password = "SecurePassword123!"
        hash1, salt1 = hash_password(password)
        hash2, salt2 = hash_password(password)

        # Hashes should be different due to different salts
        assert hash1 != hash2
        assert salt1 != salt2

    def test_hash_password_with_provided_salt(self):
        """Test that same salt produces same hash"""
        password = "TestPassword"
        hash1, salt1 = hash_password(password)
        hash2, salt2 = hash_password(password, salt1)

        # Using same salt should produce same hash
        assert hash1 == hash2
        assert salt1 == salt2

    def test_verify_password_success(self):
        """Test successful password verification"""
        password = "MySecurePassword123"
        password_hash, salt = hash_password(password)

        # Should verify successfully
        assert verify_password(password, password_hash, salt) is True

    def test_verify_password_failure_wrong_password(self):
        """Test password verification fails with wrong password"""
        password = "CorrectPassword"
        wrong_password = "WrongPassword"
        password_hash, salt = hash_password(password)

        # Should fail verification
        assert verify_password(wrong_password, password_hash, salt) is False

    def test_verify_password_failure_wrong_salt(self):
        """Test password verification fails with wrong salt"""
        password = "TestPassword"
        password_hash, _ = hash_password(password)
        wrong_salt = "wrong_salt_value"

        # Should fail verification
        assert verify_password(password, password_hash, wrong_salt) is False


class TestRBACManagerCreation:
    """Test RBACManager initialization"""

    def test_rbac_manager_creation_minimal(self):
        """Test creating RBACManager with minimal configuration"""
        manager = RBACManager(jwt_secret="test-secret-key")

        assert manager.jwt_secret == "test-secret-key"
        assert manager.redis is None
        assert manager.config == {}
        assert manager.token_expiry_hours == 24
        assert manager.api_key_prefix == "mp"

    def test_rbac_manager_creation_with_config(self):
        """Test creating RBACManager with custom configuration"""
        config = {
            "token_expiry_hours": 48,
            "api_key_prefix": "custom"
        }
        manager = RBACManager(
            jwt_secret="secret",
            config=config
        )

        assert manager.token_expiry_hours == 48
        assert manager.api_key_prefix == "custom"

    def test_rbac_manager_in_memory_storage(self):
        """Test that RBACManager initializes empty storage"""
        manager = RBACManager(jwt_secret="secret")

        assert len(manager._users) == 0
        assert len(manager._api_keys) == 0


class TestUserManagement:
    """Test user creation and retrieval"""

    def test_add_user_success(self):
        """Test successfully adding a user"""
        manager = RBACManager(jwt_secret="test-secret")
        user_id = manager.add_user(
            username="testuser",
            password="SecurePassword123",
            role=Role.USER,
            organization_id="org-123"
        )

        # Should return a user ID
        assert user_id is not None
        assert len(user_id) > 0

    def test_add_user_with_metadata(self):
        """Test adding user with metadata"""
        manager = RBACManager(jwt_secret="test-secret")
        metadata = {"department": "Engineering", "team": "Backend"}
        user_id = manager.add_user(
            username="admin1",
            password="AdminPass123",
            role=Role.ADMIN,
            metadata=metadata
        )

        user = manager.get_user("admin1")
        assert user is not None
        assert user["metadata"]["department"] == "Engineering"

    def test_add_user_with_managed_orgs(self):
        """Test adding resource manager with managed organizations"""
        manager = RBACManager(jwt_secret="test-secret")
        user_id = manager.add_user(
            username="resource_mgr",
            password="Password123",
            role=Role.RESOURCE_MANAGER,
            organization_id="org-1",
            managed_orgs=["org-1", "org-2", "org-3"]
        )

        user = manager.get_user("resource_mgr")
        assert user["managed_orgs"] == ["org-1", "org-2", "org-3"]

    def test_get_user_success(self):
        """Test successfully retrieving a user"""
        manager = RBACManager(jwt_secret="test-secret")
        manager.add_user(
            username="testuser",
            password="TestPass123",
            role=Role.USER
        )

        user = manager.get_user("testuser")
        assert user is not None
        assert user["username"] == "testuser"
        assert user["role"] == "user"

    def test_get_user_not_found(self):
        """Test retrieving non-existent user returns None"""
        manager = RBACManager(jwt_secret="test-secret")
        user = manager.get_user("nonexistent")
        assert user is None

    def test_add_multiple_users(self):
        """Test adding multiple users"""
        manager = RBACManager(jwt_secret="test-secret")

        manager.add_user("user1", "Pass1", Role.USER)
        manager.add_user("user2", "Pass2", Role.AUDITOR)
        manager.add_user("user3", "Pass3", Role.ADMIN)

        assert manager.get_user("user1") is not None
        assert manager.get_user("user2") is not None
        assert manager.get_user("user3") is not None


class TestUserAuthentication:
    """Test user authentication"""

    def test_authenticate_user_success(self):
        """Test successful user authentication"""
        manager = RBACManager(jwt_secret="test-secret")
        manager.add_user(
            username="testuser",
            password="CorrectPassword",
            role=Role.USER,
            organization_id="org-1"
        )

        context = manager.authenticate_user("testuser", "CorrectPassword")

        assert context is not None
        assert context.username == "testuser"
        assert context.role == Role.USER
        assert context.organization_id == "org-1"

    def test_authenticate_user_invalid_password(self):
        """Test authentication fails with invalid password"""
        manager = RBACManager(jwt_secret="test-secret")
        manager.add_user(
            username="testuser",
            password="CorrectPassword",
            role=Role.USER
        )

        with pytest.raises(AuthenticationError):
            manager.authenticate_user("testuser", "WrongPassword")

    def test_authenticate_user_not_found(self):
        """Test authentication fails for non-existent user"""
        manager = RBACManager(jwt_secret="test-secret")

        with pytest.raises(AuthenticationError):
            manager.authenticate_user("nonexistent", "anypassword")

    def test_authenticate_disabled_user(self):
        """Test authentication fails for disabled user"""
        manager = RBACManager(jwt_secret="test-secret")
        manager.add_user(
            username="disableduser",
            password="Password123",
            role=Role.USER
        )

        # Disable the user
        user = manager.get_user("disableduser")
        user["enabled"] = False

        with pytest.raises(AuthenticationError):
            manager.authenticate_user("disableduser", "Password123")


class TestJWTTokens:
    """Test JWT token generation and verification"""

    def test_generate_jwt_token_success(self):
        """Test successful JWT token generation"""
        manager = RBACManager(jwt_secret="test-secret-key")
        context = UserContext(
            user_id="user-123",
            username="testuser",
            role=Role.USER,
            organization_id="org-1"
        )

        token = manager.generate_jwt_token(context)

        assert token is not None
        assert len(token) > 0
        # JWT has 3 parts separated by dots
        assert token.count('.') == 2

    def test_generate_jwt_token_with_custom_expiry(self):
        """Test JWT token generation with custom expiry"""
        manager = RBACManager(jwt_secret="test-secret")
        context = UserContext(
            user_id="user-456",
            username="admin",
            role=Role.ADMIN
        )

        token = manager.generate_jwt_token(context, expires_hours=72)
        assert token is not None

    def test_verify_jwt_token_success(self):
        """Test successful JWT token verification"""
        manager = RBACManager(jwt_secret="test-secret")
        original_context = UserContext(
            user_id="user-789",
            username="testuser",
            role=Role.USER,
            organization_id="org-1"
        )

        token = manager.generate_jwt_token(original_context)
        verified_context = manager.verify_jwt_token(token)

        assert verified_context.user_id == original_context.user_id
        assert verified_context.username == original_context.username
        assert verified_context.role == original_context.role

    def test_verify_jwt_token_invalid_signature(self):
        """Test verification fails with tampered token"""
        manager = RBACManager(jwt_secret="test-secret")
        context = UserContext(
            user_id="user-999",
            username="testuser",
            role=Role.USER
        )

        token = manager.generate_jwt_token(context)

        # Tamper with signature
        parts = token.split('.')
        tampered_token = parts[0] + '.' + parts[1] + '.invalidsignature'

        with pytest.raises(AuthenticationError):
            manager.verify_jwt_token(tampered_token)

    def test_verify_jwt_token_expired(self):
        """Test verification fails with expired token"""
        manager = RBACManager(jwt_secret="test-secret")
        context = UserContext(
            user_id="user-100",
            username="testuser",
            role=Role.USER
        )

        # Generate token with -1 hours expiry (already expired)
        import base64
        import hmac
        import hashlib

        now = datetime.utcnow()
        expires = now - timedelta(hours=1)  # Expired

        payload = {
            "user_id": context.user_id,
            "username": context.username,
            "role": context.role.value,
            "organization_id": context.organization_id,
            "managed_orgs": [],
            "exp": int(expires.timestamp()),
            "iat": int(now.timestamp()),
            "iss": "marchproxy-ailb"
        }

        header = {"alg": "HS256", "typ": "JWT"}
        header_b64 = base64.urlsafe_b64encode(
            json.dumps(header).encode()
        ).rstrip(b'=').decode()
        payload_b64 = base64.urlsafe_b64encode(
            json.dumps(payload).encode()
        ).rstrip(b'=').decode()

        message = f"{header_b64}.{payload_b64}"
        signature = hmac.new(
            manager.jwt_secret.encode(),
            message.encode(),
            hashlib.sha256
        ).digest()
        signature_b64 = base64.urlsafe_b64encode(signature).rstrip(b'=').decode()

        expired_token = f"{header_b64}.{payload_b64}.{signature_b64}"

        with pytest.raises(AuthenticationError):
            manager.verify_jwt_token(expired_token)

    def test_verify_jwt_token_invalid_format(self):
        """Test verification fails with invalid token format"""
        manager = RBACManager(jwt_secret="test-secret")

        with pytest.raises(AuthenticationError):
            manager.verify_jwt_token("invalid.token")

        with pytest.raises(AuthenticationError):
            manager.verify_jwt_token("invalid")


class TestPermissions:
    """Test permission checking"""

    def test_role_permissions_admin(self):
        """Test admin role has all permissions"""
        admin_perms = ROLE_PERMISSIONS[Role.ADMIN]

        # Admin should have all permissions
        assert Permission.SYSTEM_CONFIG in admin_perms
        assert Permission.USER_CREATE in admin_perms
        assert Permission.APIKEY_DELETE in admin_perms
        assert Permission.ANALYTICS_EXPORT in admin_perms

    def test_role_permissions_user(self):
        """Test user role has limited permissions"""
        user_perms = ROLE_PERMISSIONS[Role.USER]

        # User should have basic permissions
        assert Permission.SYSTEM_HEALTH in user_perms
        assert Permission.PROXY_USE in user_perms
        assert Permission.APIKEY_CREATE in user_perms

        # User should NOT have admin permissions
        assert Permission.SYSTEM_CONFIG not in user_perms
        assert Permission.USER_CREATE not in user_perms

    def test_role_permissions_auditor(self):
        """Test auditor role has read-only permissions"""
        auditor_perms = ROLE_PERMISSIONS[Role.AUDITOR]

        # Auditor can read
        assert Permission.ANALYTICS_READ in auditor_perms
        assert Permission.USER_READ in auditor_perms

        # Auditor cannot modify
        assert Permission.USER_UPDATE not in auditor_perms
        assert Permission.USER_DELETE not in auditor_perms

    def test_role_permissions_service(self):
        """Test service role has minimal permissions"""
        service_perms = ROLE_PERMISSIONS[Role.SERVICE]

        # Service can use proxy and check health
        assert Permission.SYSTEM_HEALTH in service_perms
        assert Permission.PROXY_USE in service_perms

        # Service has no admin permissions
        assert Permission.USER_CREATE not in service_perms

    def test_check_permission_granted(self):
        """Test permission check returns True when granted"""
        manager = RBACManager(jwt_secret="test-secret")
        context = UserContext(
            user_id="user-1",
            username="testuser",
            role=Role.ADMIN,
            permissions=ROLE_PERMISSIONS[Role.ADMIN]
        )

        # Admin should have system config permission
        assert manager.check_permission(
            context,
            Permission.SYSTEM_CONFIG
        ) is True

    def test_check_permission_denied(self):
        """Test permission check returns False when denied"""
        manager = RBACManager(jwt_secret="test-secret")
        context = UserContext(
            user_id="user-2",
            username="testuser",
            role=Role.USER,
            permissions=ROLE_PERMISSIONS[Role.USER]
        )

        # User should NOT have system config permission
        assert manager.check_permission(
            context,
            Permission.SYSTEM_CONFIG
        ) is False

    def test_check_permission_with_org_restriction(self):
        """Test permission check with organization restriction"""
        manager = RBACManager(jwt_secret="test-secret")
        context = UserContext(
            user_id="user-3",
            username="manager",
            role=Role.RESOURCE_MANAGER,
            organization_id="org-1",
            managed_orgs=["org-1", "org-2"],
            permissions=ROLE_PERMISSIONS[Role.RESOURCE_MANAGER]
        )

        # Should have access to managed org
        assert manager.check_permission(
            context,
            Permission.QUOTA_UPDATE,
            resource_org_id="org-1"
        ) is True

        # Should NOT have access to non-managed org
        assert manager.check_permission(
            context,
            Permission.QUOTA_UPDATE,
            resource_org_id="org-3"
        ) is False

    def test_check_permission_with_user_restriction(self):
        """Test permission check with user-level restriction"""
        manager = RBACManager(jwt_secret="test-secret")
        user_id = "user-100"
        context = UserContext(
            user_id=user_id,
            username="testuser",
            role=Role.USER,
            permissions=ROLE_PERMISSIONS[Role.USER]
        )

        # User can access own data
        assert manager.check_permission(
            context,
            Permission.APIKEY_READ,
            resource_user_id=user_id
        ) is True

        # User cannot access other's data
        assert manager.check_permission(
            context,
            Permission.APIKEY_READ,
            resource_user_id="user-200"
        ) is False

    def test_admin_always_has_permission(self):
        """Test admin always has access regardless of org/user"""
        manager = RBACManager(jwt_secret="test-secret")
        context = UserContext(
            user_id="admin-1",
            username="admin",
            role=Role.ADMIN,
            permissions=ROLE_PERMISSIONS[Role.ADMIN]
        )

        # Admin has access to any org
        assert manager.check_permission(
            context,
            Permission.SYSTEM_CONFIG,
            resource_org_id="any-org"
        ) is True

        # Admin has access to any user's data
        assert manager.check_permission(
            context,
            Permission.USER_READ,
            resource_user_id="any-user"
        ) is True


class TestAPIKeyManagement:
    """Test API key creation and management"""

    def test_create_api_key_success(self):
        """Test successful API key creation"""
        manager = RBACManager(jwt_secret="test-secret")
        manager.add_user("testuser", "Password123", Role.USER)
        context = manager.authenticate_user("testuser", "Password123")

        api_key, key_id = manager.create_api_key(
            context,
            name="TestKey"
        )

        # Should return key and key_id
        assert api_key is not None
        assert key_id is not None
        assert api_key.startswith("mp-")  # Should use default prefix

    def test_create_api_key_with_expiry(self):
        """Test API key creation with expiration"""
        manager = RBACManager(jwt_secret="test-secret")
        manager.add_user("admin", "AdminPass123", Role.ADMIN)
        context = manager.authenticate_user("admin", "AdminPass123")

        api_key, key_id = manager.create_api_key(
            context,
            name="ExpiringKey",
            expires_days=30
        )

        key_data = manager._get_api_key(key_id)
        assert key_data["expires_at"] is not None

    def test_api_key_format(self):
        """Test API key format is correct"""
        manager = RBACManager(
            jwt_secret="test-secret",
            config={"api_key_prefix": "custom"}
        )
        manager.add_user("user", "Pass", Role.USER)
        context = manager.authenticate_user("user", "Pass")

        api_key, key_id = manager.create_api_key(context, "key1")

        # Format should be: prefix-key_id-secret
        parts = api_key.split('-')
        assert parts[0] == "custom"
        assert len(parts) >= 3

    def test_authenticate_with_api_key_success(self):
        """Test successful API key authentication"""
        manager = RBACManager(jwt_secret="test-secret")
        manager.add_user("testuser", "Password123", Role.USER)
        user_context = manager.authenticate_user("testuser", "Password123")

        api_key, key_id = manager.create_api_key(
            user_context,
            name="TestKey"
        )

        # Should authenticate with API key
        api_context = manager.authenticate_api_key(api_key)

        assert api_context is not None
        assert api_context.username == "testuser"
        assert api_context.api_key_id == key_id

    def test_authenticate_with_invalid_api_key(self):
        """Test authentication fails with invalid API key"""
        manager = RBACManager(jwt_secret="test-secret")

        with pytest.raises(AuthenticationError):
            manager.authenticate_api_key("invalid-api-key")

    def test_authenticate_with_disabled_api_key(self):
        """Test authentication fails with disabled API key"""
        manager = RBACManager(jwt_secret="test-secret")
        manager.add_user("testuser", "Password123", Role.USER)
        user_context = manager.authenticate_user("testuser", "Password123")

        api_key, key_id = manager.create_api_key(
            user_context,
            name="TestKey"
        )

        # Disable the key
        key_data = manager._get_api_key(key_id)
        key_data["enabled"] = False

        with pytest.raises(AuthenticationError):
            manager.authenticate_api_key(api_key)

    def test_authenticate_with_expired_api_key(self):
        """Test authentication fails with expired API key"""
        manager = RBACManager(jwt_secret="test-secret")
        manager.add_user("testuser", "Password123", Role.USER)
        user_context = manager.authenticate_user("testuser", "Password123")

        api_key, key_id = manager.create_api_key(
            user_context,
            name="TestKey",
            expires_days=1
        )

        # Set expiry to past
        key_data = manager._get_api_key(key_id)
        key_data["expires_at"] = (
            datetime.utcnow() - timedelta(days=1)
        ).isoformat()

        with pytest.raises(AuthenticationError):
            manager.authenticate_api_key(api_key)

    def test_revoke_api_key_success(self):
        """Test successfully revoking an API key"""
        manager = RBACManager(jwt_secret="test-secret")
        manager.add_user("testuser", "Password123", Role.USER)
        user_context = manager.authenticate_user("testuser", "Password123")

        api_key, key_id = manager.create_api_key(
            user_context,
            name="TestKey"
        )

        # Revoke the key
        assert manager.revoke_api_key(key_id, user_context) is True

        # Should not authenticate with revoked key
        with pytest.raises(AuthenticationError):
            manager.authenticate_api_key(api_key)

    def test_revoke_api_key_user_can_revoke_own(self):
        """Test user can revoke their own API key"""
        manager = RBACManager(jwt_secret="test-secret")
        manager.add_user("testuser", "Password123", Role.USER)
        user_context = manager.authenticate_user("testuser", "Password123")

        api_key, key_id = manager.create_api_key(
            user_context,
            name="TestKey"
        )

        # User can revoke own key
        assert manager.revoke_api_key(key_id, user_context) is True

    def test_revoke_api_key_user_cannot_revoke_others(self):
        """Test user cannot revoke another user's API key"""
        manager = RBACManager(jwt_secret="test-secret")
        manager.add_user("user1", "Password1", Role.USER)
        manager.add_user("user2", "Password2", Role.USER)

        context1 = manager.authenticate_user("user1", "Password1")
        context2 = manager.authenticate_user("user2", "Password2")

        api_key, key_id = manager.create_api_key(context1, "key1")

        # User2 cannot revoke User1's key
        with pytest.raises(AuthorizationError):
            manager.revoke_api_key(key_id, context2)

    def test_revoke_api_key_admin_can_revoke_any(self):
        """Test admin can revoke any API key"""
        manager = RBACManager(jwt_secret="test-secret")
        manager.add_user("admin", "AdminPass", Role.ADMIN)
        manager.add_user("user", "UserPass", Role.USER)

        admin_context = manager.authenticate_user("admin", "AdminPass")
        user_context = manager.authenticate_user("user", "UserPass")

        api_key, key_id = manager.create_api_key(user_context, "userkey")

        # Admin can revoke user's key
        assert manager.revoke_api_key(key_id, admin_context) is True

    def test_list_api_keys_admin_sees_all(self):
        """Test admin can see all API keys"""
        manager = RBACManager(jwt_secret="test-secret")
        manager.add_user("admin", "AdminPass", Role.ADMIN)
        manager.add_user("user1", "Pass1", Role.USER)
        manager.add_user("user2", "Pass2", Role.USER)

        admin_context = manager.authenticate_user("admin", "AdminPass")
        context1 = manager.authenticate_user("user1", "Pass1")
        context2 = manager.authenticate_user("user2", "Pass2")

        manager.create_api_key(context1, "key1")
        manager.create_api_key(context2, "key2")

        keys = manager.list_api_keys(admin_context)
        assert len(keys) == 2

    def test_list_api_keys_user_sees_own(self):
        """Test user can see only their own API keys"""
        manager = RBACManager(jwt_secret="test-secret")
        manager.add_user("user1", "Pass1", Role.USER)
        manager.add_user("user2", "Pass2", Role.USER)

        context1 = manager.authenticate_user("user1", "Pass1")
        context2 = manager.authenticate_user("user2", "Pass2")

        manager.create_api_key(context1, "key1")
        manager.create_api_key(context1, "key2")
        manager.create_api_key(context2, "key3")

        # User1 should only see 2 keys
        keys = manager.list_api_keys(context1)
        assert len(keys) == 2

        # User2 should only see 1 key
        keys = manager.list_api_keys(context2)
        assert len(keys) == 1


class TestUserContext:
    """Test UserContext dataclass"""

    def test_user_context_creation(self):
        """Test creating UserContext"""
        context = UserContext(
            user_id="user-1",
            username="testuser",
            role=Role.USER,
            organization_id="org-1"
        )

        assert context.user_id == "user-1"
        assert context.username == "testuser"
        assert context.role == Role.USER

    def test_user_context_has_permission(self):
        """Test has_permission method"""
        permissions = {Permission.PROXY_USE, Permission.SYSTEM_HEALTH}
        context = UserContext(
            user_id="user-2",
            username="testuser",
            role=Role.USER,
            permissions=permissions
        )

        assert context.has_permission(Permission.PROXY_USE) is True
        assert context.has_permission(Permission.USER_CREATE) is False

    def test_user_context_to_dict(self):
        """Test converting UserContext to dictionary"""
        permissions = {Permission.PROXY_USE, Permission.SYSTEM_HEALTH}
        context = UserContext(
            user_id="user-3",
            username="testuser",
            role=Role.USER,
            organization_id="org-1",
            managed_orgs=["org-1"],
            permissions=permissions,
            metadata={"key": "value"}
        )

        context_dict = context.to_dict()

        assert context_dict["user_id"] == "user-3"
        assert context_dict["username"] == "testuser"
        assert context_dict["role"] == "user"
        assert context_dict["organization_id"] == "org-1"
        assert len(context_dict["permissions"]) == 2
        assert context_dict["metadata"]["key"] == "value"


class TestRBACStatistics:
    """Test RBAC statistics and management"""

    def test_get_stats_empty(self):
        """Test stats for empty RBAC manager"""
        manager = RBACManager(jwt_secret="test-secret")
        stats = manager.get_stats()

        assert stats["total_users"] == 0
        assert stats["total_api_keys"] == 0
        assert stats["active_api_keys"] == 0

    def test_get_stats_with_users(self):
        """Test stats with multiple users"""
        manager = RBACManager(jwt_secret="test-secret")
        manager.add_user("user1", "Pass1", Role.USER)
        manager.add_user("admin1", "Pass2", Role.ADMIN)
        manager.add_user("auditor1", "Pass3", Role.AUDITOR)

        stats = manager.get_stats()

        assert stats["total_users"] == 3
        assert stats["users_by_role"]["user"] == 1
        assert stats["users_by_role"]["admin"] == 1
        assert stats["users_by_role"]["auditor"] == 1

    def test_get_stats_with_api_keys(self):
        """Test stats with API keys"""
        manager = RBACManager(jwt_secret="test-secret")
        manager.add_user("user1", "Pass1", Role.USER)
        context = manager.authenticate_user("user1", "Pass1")

        manager.create_api_key(context, "key1")
        manager.create_api_key(context, "key2")

        stats = manager.get_stats()

        assert stats["total_api_keys"] == 2
        assert stats["active_api_keys"] == 2


class TestIntegrationScenarios:
    """Test real-world integration scenarios"""

    def test_complete_auth_flow(self):
        """Test complete authentication flow: create user, authenticate, create token"""
        manager = RBACManager(jwt_secret="test-secret-key")

        # Step 1: Create user
        user_id = manager.add_user(
            username="newuser",
            password="MySecurePassword123",
            role=Role.USER,
            organization_id="org-1"
        )
        assert user_id is not None

        # Step 2: Authenticate
        user_context = manager.authenticate_user("newuser", "MySecurePassword123")
        assert user_context.username == "newuser"

        # Step 3: Generate JWT token
        token = manager.generate_jwt_token(user_context)
        assert token is not None

        # Step 4: Verify token
        verified_context = manager.verify_jwt_token(token)
        assert verified_context.user_id == user_context.user_id

    def test_api_key_flow(self):
        """Test complete API key flow"""
        manager = RBACManager(jwt_secret="test-secret")

        # Create user
        manager.add_user("apiuser", "Password123", Role.USER)
        user_context = manager.authenticate_user("apiuser", "Password123")

        # Create API key
        api_key, key_id = manager.create_api_key(
            user_context,
            name="integration-key"
        )

        # Authenticate with API key
        api_context = manager.authenticate_api_key(api_key)
        assert api_context.username == "apiuser"
        assert api_context.api_key_id == key_id

        # List keys
        keys = manager.list_api_keys(user_context)
        assert len(keys) == 1

        # Revoke key
        manager.revoke_api_key(key_id, user_context)

        # Try to authenticate with revoked key
        with pytest.raises(AuthenticationError):
            manager.authenticate_api_key(api_key)

    def test_resource_manager_access_control(self):
        """Test resource manager can only access assigned organizations"""
        manager = RBACManager(jwt_secret="test-secret")

        # Create resource manager
        manager.add_user(
            "resource_mgr",
            "Password123",
            Role.RESOURCE_MANAGER,
            organization_id="org-1",
            managed_orgs=["org-1", "org-2"]
        )
        mgr_context = manager.authenticate_user("resource_mgr", "Password123")

        # Should have access to managed orgs
        assert manager.check_permission(
            mgr_context,
            Permission.QUOTA_UPDATE,
            resource_org_id="org-1"
        ) is True

        assert manager.check_permission(
            mgr_context,
            Permission.QUOTA_UPDATE,
            resource_org_id="org-2"
        ) is True

        # Should NOT have access to non-managed org
        assert manager.check_permission(
            mgr_context,
            Permission.QUOTA_UPDATE,
            resource_org_id="org-3"
        ) is False

    def test_multiple_api_keys_per_user(self):
        """Test user can have multiple API keys"""
        manager = RBACManager(jwt_secret="test-secret")
        manager.add_user("multikey_user", "Password123", Role.USER)
        user_context = manager.authenticate_user("multikey_user", "Password123")

        # Create multiple keys
        key1, id1 = manager.create_api_key(user_context, "production")
        key2, id2 = manager.create_api_key(user_context, "staging")
        key3, id3 = manager.create_api_key(user_context, "development")

        # All should authenticate
        ctx1 = manager.authenticate_api_key(key1)
        ctx2 = manager.authenticate_api_key(key2)
        ctx3 = manager.authenticate_api_key(key3)

        assert ctx1.api_key_id == id1
        assert ctx2.api_key_id == id2
        assert ctx3.api_key_id == id3

        # All should be listed
        keys = manager.list_api_keys(user_context)
        assert len(keys) == 3


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
