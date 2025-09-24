# In-Code Documentation Standards

This document defines the comprehensive documentation standards for MarchProxy codebase to ensure consistency, maintainability, and clarity across all components.

## Overview

MarchProxy follows strict documentation standards to maintain high code quality and developer productivity. All code must be self-documenting with clear, comprehensive documentation that explains not just what the code does, but why it does it.

## General Principles

### 1. Documentation-First Development
- Write documentation before implementing features
- Update documentation with every code change
- Treat documentation as part of the code review process

### 2. Clarity and Completeness
- Explain the "why" not just the "what"
- Include examples for complex functions
- Document edge cases and error conditions
- Use clear, concise language

### 3. Consistency
- Follow language-specific conventions
- Use consistent terminology across the codebase
- Maintain uniform formatting

## Language-Specific Standards

### Go Code Documentation

#### Package Documentation

Every Go package must have comprehensive package documentation:

```go
// Package auth provides authentication and authorization services for MarchProxy.
//
// This package implements multiple authentication methods including JWT tokens,
// API keys, SAML SSO, and OAuth2 integration. It supports both local authentication
// and integration with external identity providers.
//
// Key Features:
//   - JWT token validation with configurable algorithms
//   - API key management with rotation support
//   - SAML 2.0 SSO integration for enterprise authentication
//   - OAuth2 support for Google, Microsoft, and custom providers
//   - Multi-factor authentication with TOTP
//   - Session management with secure cookie handling
//
// Basic Usage:
//
//	auth := auth.NewService(config)
//	token, err := auth.ValidateJWT(tokenString)
//	if err != nil {
//	    log.Error("Authentication failed", "error", err)
//	    return
//	}
//
// Configuration:
//
//	config := auth.Config{
//	    JWTSecret:     "your-secret-key",
//	    SessionMaxAge: time.Hour * 24,
//	    SAMLEnabled:   true,
//	}
//
// Thread Safety:
//
// All functions in this package are safe for concurrent use unless
// otherwise noted. The AuthService maintains internal state that is
// protected by appropriate synchronization primitives.
package auth
```

#### Function Documentation

All exported functions must have godoc comments:

```go
// ValidateJWT validates a JWT token and returns the associated user claims.
//
// This function performs comprehensive JWT validation including:
//   - Signature verification using the configured secret/key
//   - Expiration time validation with configurable clock skew
//   - Issuer and audience validation if configured
//   - Custom claim validation for MarchProxy-specific fields
//
// Parameters:
//   - tokenString: The JWT token as a string (without "Bearer " prefix)
//
// Returns:
//   - *UserClaims: Parsed and validated user claims
//   - error: Validation error with specific failure reason
//
// Possible errors:
//   - ErrInvalidToken: Token format is invalid or corrupted
//   - ErrTokenExpired: Token has expired (check exp claim)
//   - ErrInvalidSignature: Token signature verification failed
//   - ErrInvalidIssuer: Token issuer doesn't match expected value
//   - ErrInvalidAudience: Token audience doesn't match expected value
//   - ErrMissingClaims: Required custom claims are missing
//
// Example:
//
//	tokenString := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
//	claims, err := auth.ValidateJWT(tokenString)
//	if err != nil {
//	    if errors.Is(err, auth.ErrTokenExpired) {
//	        return handleTokenRefresh()
//	    }
//	    return handleAuthError(err)
//	}
//
//	userID := claims.UserID
//	roles := claims.Roles
//
// Thread Safety:
//
// This function is safe for concurrent use and does not modify any
// shared state. Multiple goroutines can call this function simultaneously.
func (s *Service) ValidateJWT(tokenString string) (*UserClaims, error) {
    // Implementation...
}
```

#### Struct Documentation

Document all exported structs and their fields:

```go
// Config holds the configuration for the authentication service.
//
// This structure defines all parameters needed to configure authentication
// behavior, including JWT settings, session management, and external
// identity provider integration.
//
// Field validation is performed during service initialization. Invalid
// configurations will result in an error when creating the service.
type Config struct {
    // JWTSecret is the secret key used for JWT token signing and validation.
    // Must be at least 256 bits (32 bytes) for HS256 algorithm.
    // This field is required for JWT authentication to work.
    JWTSecret string `json:"jwt_secret" yaml:"jwt_secret" validate:"required,min=32"`

    // JWTAlgorithm specifies the algorithm used for JWT signing.
    // Supported algorithms: HS256, HS384, HS512, RS256, RS384, RS512
    // Default: HS256
    JWTAlgorithm string `json:"jwt_algorithm" yaml:"jwt_algorithm" validate:"oneof=HS256 HS384 HS512 RS256 RS384 RS512"`

    // TokenExpiry defines how long JWT tokens remain valid.
    // After this duration, tokens will be rejected during validation.
    // Recommended: 1 hour for security, longer for user convenience.
    // Default: 1 hour
    TokenExpiry time.Duration `json:"token_expiry" yaml:"token_expiry"`

    // SessionMaxAge controls HTTP session cookie lifetime.
    // Sessions will be automatically invalidated after this duration.
    // Should be longer than TokenExpiry to allow token refresh.
    // Default: 24 hours
    SessionMaxAge time.Duration `json:"session_max_age" yaml:"session_max_age"`

    // SAMLConfig contains SAML SSO configuration (Enterprise feature).
    // This field is optional and only used when SAML authentication is enabled.
    // Requires valid Enterprise license with saml_authentication feature.
    SAMLConfig *SAMLConfig `json:"saml_config,omitempty" yaml:"saml_config,omitempty"`

    // OAuth2Providers defines external OAuth2 identity providers.
    // Each provider must have unique name and valid configuration.
    // Supports: google, microsoft, github, custom
    OAuth2Providers map[string]*OAuth2Config `json:"oauth2_providers,omitempty" yaml:"oauth2_providers,omitempty"`

    // EnableMFA controls whether multi-factor authentication is required.
    // When enabled, users must configure TOTP after first login.
    // Default: false (for backward compatibility)
    EnableMFA bool `json:"enable_mfa" yaml:"enable_mfa"`

    // TOTPIssuer is the issuer name displayed in authenticator apps.
    // Should be set to your organization or application name.
    // Example: "MarchProxy Production"
    TOTPIssuer string `json:"totp_issuer" yaml:"totp_issuer"`
}
```

#### Interface Documentation

Document interfaces with usage examples:

```go
// AuthService defines the interface for authentication operations.
//
// This interface abstracts authentication functionality to allow for
// different implementations (local, LDAP, external services) while
// maintaining a consistent API throughout the application.
//
// Implementations must be thread-safe and handle concurrent requests
// appropriately. All methods should include proper context handling
// for request timeout and cancellation.
//
// Error Handling:
//
// All methods return structured errors that can be checked using
// errors.Is() for specific error types. This allows callers to
// handle different failure scenarios appropriately.
//
// Example Implementation:
//
//	type LocalAuthService struct {
//	    config Config
//	    db     database.DB
//	}
//
//	func (s *LocalAuthService) ValidateJWT(ctx context.Context, token string) (*UserClaims, error) {
//	    // Implementation for local JWT validation
//	}
//
// Example Usage:
//
//	authSvc := auth.NewLocalService(config, db)
//	claims, err := authSvc.ValidateJWT(ctx, tokenString)
//	if err != nil {
//	    return handleAuthError(err)
//	}
type AuthService interface {
    // ValidateJWT validates a JWT token and returns user claims.
    //
    // The context should include request timeout and any tracing information.
    // Returns ErrInvalidToken, ErrTokenExpired, or other specific auth errors.
    ValidateJWT(ctx context.Context, token string) (*UserClaims, error)

    // ValidateAPIKey validates an API key and returns associated permissions.
    //
    // API keys are long-lived credentials used for programmatic access.
    // The returned permissions define what operations the key can perform.
    ValidateAPIKey(ctx context.Context, apiKey string) (*APIKeyPermissions, error)

    // CreateSession creates a new user session after successful authentication.
    //
    // Sessions are stored server-side and referenced by secure cookies.
    // The userID must reference a valid user in the system.
    CreateSession(ctx context.Context, userID int, metadata SessionMetadata) (*Session, error)

    // InvalidateSession invalidates an existing user session.
    //
    // This is called during logout or when session security is compromised.
    // The sessionID should be the unique identifier for the session.
    InvalidateSession(ctx context.Context, sessionID string) error
}
```

#### Error Documentation

Document custom error types:

```go
// AuthError represents authentication-related errors with specific types
// that can be checked using errors.Is() for appropriate error handling.
//
// Error types follow a hierarchical structure:
//   - ErrAuthentication: Base authentication error
//     - ErrInvalidCredentials: Wrong username/password
//     - ErrAccountLocked: Account temporarily locked
//     - ErrAccountDisabled: Account permanently disabled
//   - ErrAuthorization: Base authorization error
//     - ErrInsufficientPermissions: User lacks required permissions
//     - ErrResourceNotFound: Requested resource doesn't exist
//
// Example usage:
//
//	err := authService.Login(username, password)
//	if err != nil {
//	    if errors.Is(err, auth.ErrInvalidCredentials) {
//	        return "Invalid username or password"
//	    }
//	    if errors.Is(err, auth.ErrAccountLocked) {
//	        return "Account temporarily locked. Try again later."
//	    }
//	    return "Authentication failed"
//	}
type AuthError struct {
    // Type specifies the specific error type for programmatic handling
    Type AuthErrorType

    // Message provides a human-readable error description
    Message string

    // Code is a stable error code that won't change between versions
    Code string

    // Details contains additional context about the error
    Details map[string]interface{}

    // Underlying wraps the original error that caused this auth error
    Underlying error
}

// Error implements the error interface
func (e *AuthError) Error() string {
    if e.Message != "" {
        return e.Message
    }
    return string(e.Type)
}

// Is implements error checking for errors.Is()
func (e *AuthError) Is(target error) bool {
    t, ok := target.(*AuthError)
    if !ok {
        return false
    }
    return e.Type == t.Type
}

// Unwrap returns the underlying error for errors.Unwrap()
func (e *AuthError) Unwrap() error {
    return e.Underlying
}

// Predefined error instances for common cases
var (
    // ErrInvalidCredentials indicates wrong username or password
    ErrInvalidCredentials = &AuthError{
        Type: AuthErrorInvalidCredentials,
        Code: "INVALID_CREDENTIALS",
        Message: "Invalid username or password",
    }

    // ErrTokenExpired indicates JWT token has expired
    ErrTokenExpired = &AuthError{
        Type: AuthErrorTokenExpired,
        Code: "TOKEN_EXPIRED",
        Message: "Authentication token has expired",
    }

    // ErrInsufficientPermissions indicates user lacks required permissions
    ErrInsufficientPermissions = &AuthError{
        Type: AuthErrorInsufficientPermissions,
        Code: "INSUFFICIENT_PERMISSIONS",
        Message: "User lacks required permissions for this operation",
    }
)
```

### Python Code Documentation

#### Module Documentation

Every Python module must have comprehensive docstrings:

```python
"""Authentication and authorization module for MarchProxy Manager.

This module provides comprehensive authentication services including user
management, session handling, JWT token operations, and integration with
external identity providers like SAML and OAuth2.

Classes:
    AuthService: Main authentication service class
    SAMLProvider: SAML SSO integration
    OAuth2Provider: OAuth2 authentication provider
    JWTManager: JWT token creation and validation

Functions:
    hash_password: Secure password hashing with bcrypt
    verify_password: Password verification
    generate_totp_secret: TOTP secret generation for 2FA

Constants:
    DEFAULT_TOKEN_EXPIRY: Default JWT token expiration (1 hour)
    MAX_LOGIN_ATTEMPTS: Maximum failed login attempts before lockout
    LOCKOUT_DURATION: Account lockout duration in seconds

Example:
    Basic authentication setup:

    >>> auth = AuthService(db, config)
    >>> user = auth.authenticate_user("admin", "password")
    >>> if user:
    ...     token = auth.create_jwt_token(user.id)
    ...     print(f"Login successful: {token}")

Requirements:
    - py4web framework for web integration
    - pydal for database operations
    - passlib for password hashing
    - pyotp for TOTP generation
    - python-saml for SAML integration (optional)

License:
    AGPL v3 - See LICENSE file for details

Author:
    MarchProxy Development Team

Version:
    0.1.1
"""

from typing import Optional, Dict, Any, List, Union
import logging
from datetime import datetime, timedelta

# Module-level constants with documentation
DEFAULT_TOKEN_EXPIRY = 3600  # 1 hour in seconds
MAX_LOGIN_ATTEMPTS = 5      # Maximum failed attempts before lockout
LOCKOUT_DURATION = 900     # 15 minutes in seconds

logger = logging.getLogger(__name__)
```

#### Class Documentation

Document all classes with comprehensive docstrings:

```python
class AuthService:
    """Provides authentication and authorization services for MarchProxy.

    This class handles all authentication operations including user login,
    JWT token management, session handling, and integration with external
    identity providers. It supports both local authentication and enterprise
    SSO solutions.

    The service is designed to be thread-safe and can handle concurrent
    authentication requests. It integrates with the py4web framework for
    web authentication and provides REST API endpoints for programmatic access.

    Attributes:
        db (DAL): Database connection for user and session storage
        config (AuthConfig): Authentication configuration parameters
        jwt_manager (JWTManager): JWT token creation and validation
        session_store (SessionStore): Session persistence and management

    Example:
        Basic service initialization:

        >>> config = AuthConfig(
        ...     jwt_secret="secure-secret-key",
        ...     session_timeout=3600,
        ...     enable_2fa=True
        ... )
        >>> auth = AuthService(db, config)
        >>>
        >>> # Authenticate user
        >>> user = auth.authenticate_user("admin", "password", "123456")
        >>> if user:
        ...     session = auth.create_session(user.id)
        ...     print(f"Session created: {session.id}")

    Enterprise Features:
        When configured with appropriate licenses, the service supports:
        - SAML 2.0 SSO integration
        - OAuth2 authentication (Google, Microsoft, custom)
        - SCIM user provisioning
        - Advanced audit logging

    Thread Safety:
        All public methods are thread-safe and can be called concurrently.
        Internal state is protected using appropriate locking mechanisms.

    Raises:
        AuthenticationError: For authentication failures
        AuthorizationError: For authorization failures
        ConfigurationError: For invalid configuration
        DatabaseError: For database connectivity issues
    """

    def __init__(self, db: Any, config: 'AuthConfig') -> None:
        """Initialize the authentication service.

        Args:
            db: Database connection (pydal DAL instance)
            config: Authentication configuration object

        Raises:
            ConfigurationError: If configuration is invalid
            DatabaseError: If database connection fails

        Example:
            >>> config = AuthConfig(jwt_secret="my-secret")
            >>> auth = AuthService(db, config)
        """
        self.db = db
        self.config = config
        self.jwt_manager = JWTManager(config.jwt_secret, config.jwt_algorithm)
        self.session_store = SessionStore(db)

        # Validate configuration
        self._validate_config()

        # Initialize external providers if configured
        self._init_saml_provider()
        self._init_oauth2_providers()

        logger.info("AuthService initialized", extra={
            "jwt_algorithm": config.jwt_algorithm,
            "session_timeout": config.session_timeout,
            "mfa_enabled": config.enable_mfa
        })
```

#### Function Documentation

All functions must have detailed docstrings:

```python
def authenticate_user(
    self,
    username: str,
    password: str,
    totp_code: Optional[str] = None,
    user_agent: Optional[str] = None,
    ip_address: Optional[str] = None
) -> Optional['User']:
    """Authenticate a user with username, password, and optional 2FA.

    This method performs comprehensive user authentication including:
    - Username and password validation
    - Account status verification (active, not locked)
    - TOTP verification if 2FA is enabled
    - Failed attempt tracking and account lockout
    - Audit logging for security monitoring

    The authentication process follows these steps:
    1. Validate input parameters
    2. Check for account lockout
    3. Verify username exists and account is active
    4. Validate password hash
    5. Verify TOTP code if 2FA is enabled
    6. Update last login timestamp
    7. Log authentication event

    Args:
        username: User's login name (case-insensitive)
        password: Plain text password for verification
        totp_code: 6-digit TOTP code for 2FA (required if enabled)
        user_agent: Client user agent string for audit logging
        ip_address: Client IP address for audit logging

    Returns:
        User object if authentication succeeds, None if it fails

    Raises:
        AuthenticationError: For authentication failures with specific reasons
        ValueError: For invalid input parameters
        DatabaseError: For database connectivity issues

    Example:
        Basic authentication:

        >>> user = auth.authenticate_user("admin", "password123")
        >>> if user:
        ...     print(f"Welcome {user.username}")
        ... else:
        ...     print("Authentication failed")

        With 2FA:

        >>> user = auth.authenticate_user(
        ...     "admin",
        ...     "password123",
        ...     totp_code="123456"
        ... )

        With audit information:

        >>> user = auth.authenticate_user(
        ...     "admin",
        ...     "password123",
        ...     user_agent="Mozilla/5.0...",
        ...     ip_address="192.168.1.100"
        ... )

    Security Considerations:
        - Passwords are never logged or stored in plain text
        - Failed attempts are tracked to prevent brute force attacks
        - Account lockout is enforced after MAX_LOGIN_ATTEMPTS failures
        - All authentication events are logged for audit purposes

    Performance Notes:
        - Password verification uses bcrypt which is intentionally slow
        - Database queries are optimized with proper indexing
        - TOTP verification includes network time tolerance

    Side Effects:
        - Updates user's last_login timestamp on success
        - Increments failed_attempts counter on failure
        - May lock account if too many failures occur
        - Creates audit log entry for the authentication attempt
    """
    # Validate input parameters
    if not username or not password:
        raise ValueError("Username and password are required")

    if len(username) > 255:
        raise ValueError("Username too long")

    # Implementation continues...
```

#### Configuration Documentation

Document configuration classes thoroughly:

```python
class AuthConfig:
    """Configuration settings for the authentication service.

    This class encapsulates all configuration parameters needed for
    authentication service operation. It provides validation, defaults,
    and helper methods for configuration management.

    All configuration values can be set through environment variables,
    configuration files, or direct assignment. Environment variables
    take precedence over file settings.

    Attributes:
        jwt_secret (str): Secret key for JWT signing (required)
        jwt_algorithm (str): Algorithm for JWT signing (default: HS256)
        jwt_expiry (int): JWT token expiry in seconds (default: 3600)
        session_timeout (int): Session timeout in seconds (default: 86400)
        enable_mfa (bool): Enable multi-factor authentication (default: False)
        totp_issuer (str): TOTP issuer name for authenticator apps
        password_min_length (int): Minimum password length (default: 8)
        max_login_attempts (int): Max failed attempts before lockout (default: 5)
        lockout_duration (int): Account lockout duration in seconds (default: 900)

    Enterprise Attributes:
        saml_enabled (bool): Enable SAML SSO (requires license)
        saml_metadata_url (str): SAML IdP metadata URL
        oauth2_providers (Dict): OAuth2 provider configurations
        scim_enabled (bool): Enable SCIM provisioning

    Example:
        Basic configuration:

        >>> config = AuthConfig(
        ...     jwt_secret="your-secret-key-here",
        ...     jwt_expiry=7200,  # 2 hours
        ...     enable_mfa=True
        ... )

        Enterprise configuration:

        >>> config = AuthConfig(
        ...     jwt_secret="secret",
        ...     saml_enabled=True,
        ...     saml_metadata_url="https://idp.company.com/metadata",
        ...     oauth2_providers={
        ...         "google": {
        ...             "client_id": "google-client-id",
        ...             "client_secret": "google-client-secret"
        ...         }
        ...     }
        ... )

        From environment variables:

        >>> # Set AUTH_JWT_SECRET=secret in environment
        >>> config = AuthConfig.from_environment()

    Validation:
        Configuration validation is performed during initialization:
        - jwt_secret must be at least 32 characters for security
        - jwt_algorithm must be a supported algorithm
        - Numeric values must be within reasonable ranges
        - URLs must be valid and accessible

    Thread Safety:
        Configuration objects are immutable after creation and can be
        safely shared between threads.
    """

    def __init__(
        self,
        jwt_secret: str,
        jwt_algorithm: str = "HS256",
        jwt_expiry: int = 3600,
        session_timeout: int = 86400,
        enable_mfa: bool = False,
        totp_issuer: str = "MarchProxy",
        password_min_length: int = 8,
        max_login_attempts: int = 5,
        lockout_duration: int = 900,
        **kwargs
    ) -> None:
        """Initialize authentication configuration.

        Args:
            jwt_secret: Secret key for JWT token signing and validation
            jwt_algorithm: Algorithm for JWT operations (HS256, HS384, HS512, RS256, etc.)
            jwt_expiry: Token expiry time in seconds
            session_timeout: HTTP session timeout in seconds
            enable_mfa: Whether to require multi-factor authentication
            totp_issuer: Issuer name displayed in authenticator apps
            password_min_length: Minimum required password length
            max_login_attempts: Maximum failed login attempts before lockout
            lockout_duration: Account lockout duration in seconds
            **kwargs: Additional configuration parameters for enterprise features

        Raises:
            ValueError: If configuration parameters are invalid

        Example:
            >>> config = AuthConfig(
            ...     jwt_secret="very-secure-secret-key-here",
            ...     jwt_expiry=7200,
            ...     enable_mfa=True
            ... )
        """
        self.jwt_secret = jwt_secret
        self.jwt_algorithm = jwt_algorithm
        self.jwt_expiry = jwt_expiry
        self.session_timeout = session_timeout
        self.enable_mfa = enable_mfa
        self.totp_issuer = totp_issuer
        self.password_min_length = password_min_length
        self.max_login_attempts = max_login_attempts
        self.lockout_duration = lockout_duration

        # Enterprise features
        self.saml_enabled = kwargs.get('saml_enabled', False)
        self.saml_metadata_url = kwargs.get('saml_metadata_url')
        self.oauth2_providers = kwargs.get('oauth2_providers', {})
        self.scim_enabled = kwargs.get('scim_enabled', False)

        # Validate configuration
        self.validate()
```

### eBPF Program Documentation

Document eBPF programs with comprehensive comments:

```c
// SPDX-License-Identifier: GPL-2.0
/*
 * MarchProxy XDP Rate Limiting Program
 *
 * This eBPF program implements high-performance packet-level rate limiting
 * using the eXpress Data Path (XDP) framework. It operates at the network
 * driver level, providing maximum performance by filtering packets before
 * they enter the kernel networking stack.
 *
 * Features:
 * - Per-source-IP rate limiting using token bucket algorithm
 * - Configurable rate limits and burst sizes
 * - Global rate limiting for DDoS protection
 * - Detailed statistics collection
 * - Zero-copy operation for maximum performance
 *
 * Algorithm:
 * The program uses a token bucket algorithm for rate limiting:
 * 1. Each source IP gets a bucket with configurable capacity
 * 2. Tokens are added to buckets at a configured rate
 * 3. Each packet consumes one token from the bucket
 * 4. Packets are dropped when bucket is empty
 * 5. LRU map ensures memory usage stays bounded
 *
 * Performance:
 * - Processes millions of packets per second
 * - Constant-time lookup using hash maps
 * - Lock-free operation using atomic operations
 * - Minimal CPU overhead per packet
 *
 * Maps:
 * - rate_limit_map: Per-IP token bucket state (LRU_HASH)
 * - config_map: Global configuration (ARRAY)
 * - stats_map: Performance statistics (ARRAY)
 *
 * Configuration:
 * The program is configured through the config_map which contains:
 * - rate_limit: Maximum packets per second per IP
 * - burst_size: Maximum tokens in bucket (burst capacity)
 * - time_window: Time window for rate calculation (nanoseconds)
 * - global_rate_limit: Global rate limit across all IPs
 *
 * Statistics:
 * The program maintains detailed statistics in stats_map:
 * - total_packets: Total packets processed
 * - dropped_packets: Packets dropped due to rate limiting
 * - allowed_packets: Packets allowed through
 * - rate_limit_hits: Number of rate limit violations
 *
 * Author: MarchProxy Development Team
 * Version: 0.1.1
 * License: GPL-2.0
 */

#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/if_packet.h>
#include <linux/ip.h>
#include <linux/tcp.h>
#include <linux/udp.h>
#include <linux/icmp.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

/*
 * Configuration structure for rate limiting parameters.
 * This structure is stored in config_map and can be updated
 * from userspace to change rate limiting behavior.
 */
struct rate_limit_config {
    __u32 rate_limit;        /* Packets per second per IP */
    __u32 burst_size;        /* Maximum tokens in bucket */
    __u32 time_window;       /* Time window in nanoseconds */
    __u32 global_rate_limit; /* Global packets per second limit */
    __u32 enabled;           /* 1 if rate limiting enabled, 0 if disabled */
};

/*
 * Per-IP rate limiting state using token bucket algorithm.
 * Each source IP address has an entry in the rate_limit_map
 * that tracks the current token bucket state.
 */
struct rate_limit_entry {
    __u64 last_time;         /* Last packet timestamp (nanoseconds) */
    __u32 tokens;            /* Current tokens in bucket */
    __u32 packets;           /* Total packets seen from this IP */
    __u32 dropped;           /* Packets dropped from this IP */
};

/*
 * Statistics structure for monitoring rate limiting performance.
 * These counters help administrators understand the effectiveness
 * of rate limiting and tune parameters appropriately.
 */
struct rate_limit_stats {
    __u64 total_packets;     /* Total packets processed */
    __u64 dropped_packets;   /* Total packets dropped */
    __u64 allowed_packets;   /* Total packets allowed */
    __u64 rate_limit_hits;   /* Number of rate limit violations */
    __u64 config_updates;    /* Number of configuration updates */
    __u64 map_errors;        /* Number of map operation errors */
};

/*
 * BPF Maps for storing rate limiting state and configuration.
 * These maps provide the interface between the eBPF program
 * and userspace control applications.
 */

/* Per-IP rate limiting state using LRU eviction */
struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __type(key, __u32);                    /* Source IP address */
    __type(value, struct rate_limit_entry);
    __uint(max_entries, 65536);             /* Maximum tracked IPs */
    __uint(map_flags, BPF_F_NO_COMMON_LRU);
} rate_limit_map SEC(".maps");

/* Global configuration (single entry) */
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __type(key, __u32);
    __type(value, struct rate_limit_config);
    __uint(max_entries, 1);
} config_map SEC(".maps");

/* Performance statistics (single entry) */
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __type(key, __u32);
    __type(value, struct rate_limit_stats);
    __uint(max_entries, 1);
} stats_map SEC(".maps");

/*
 * Helper function to update statistics counters.
 * This function atomically increments various counters
 * to track rate limiting performance.
 *
 * @counter_type: Type of counter to increment (0=total, 1=dropped, 2=allowed, 3=hits)
 */
static __always_inline void update_stats(__u32 counter_type) {
    __u32 key = 0;
    struct rate_limit_stats *stats = bpf_map_lookup_elem(&stats_map, &key);

    if (!stats) {
        /* Initialize stats if not present */
        struct rate_limit_stats init_stats = {0};
        bpf_map_update_elem(&stats_map, &key, &init_stats, BPF_ANY);
        stats = bpf_map_lookup_elem(&stats_map, &key);
        if (!stats)
            return;
    }

    /* Atomically increment the appropriate counter */
    switch (counter_type) {
    case 0: /* Total packets */
        __sync_fetch_and_add(&stats->total_packets, 1);
        break;
    case 1: /* Dropped packets */
        __sync_fetch_and_add(&stats->dropped_packets, 1);
        break;
    case 2: /* Allowed packets */
        __sync_fetch_and_add(&stats->allowed_packets, 1);
        break;
    case 3: /* Rate limit hits */
        __sync_fetch_and_add(&stats->rate_limit_hits, 1);
        break;
    }
}

/*
 * Token bucket algorithm implementation.
 * This function implements the core rate limiting logic using
 * a token bucket algorithm for smooth rate limiting with burst support.
 *
 * @src_ip: Source IP address for this packet
 * @config: Rate limiting configuration
 * @now: Current timestamp in nanoseconds
 *
 * Returns: 1 if packet should be allowed, 0 if it should be dropped
 */
static __always_inline int token_bucket_check(__u32 src_ip,
                                             struct rate_limit_config *config,
                                             __u64 now) {
    /* Look up existing rate limit entry for this IP */
    struct rate_limit_entry *entry = bpf_map_lookup_elem(&rate_limit_map, &src_ip);

    if (!entry) {
        /* First packet from this IP - create new entry */
        struct rate_limit_entry new_entry = {
            .last_time = now,
            .tokens = config->burst_size - 1,  /* Consume one token for this packet */
            .packets = 1,
            .dropped = 0
        };

        /* Store new entry in map */
        if (bpf_map_update_elem(&rate_limit_map, &src_ip, &new_entry, BPF_ANY) < 0) {
            /* Map update failed - allow packet but don't track */
            return 1;
        }

        return 1; /* Allow first packet */
    }

    /* Calculate time difference since last packet */
    __u64 time_diff = now - entry->last_time;

    /* Add tokens based on time elapsed and configured rate */
    if (time_diff > 0) {
        /* Calculate tokens to add: (time_diff * rate_limit) / 1_second_in_ns */
        __u64 tokens_to_add = (time_diff * config->rate_limit) / 1000000000ULL;

        /* Cap tokens at burst size */
        __u32 new_tokens = entry->tokens + (__u32)tokens_to_add;
        if (new_tokens > config->burst_size) {
            new_tokens = config->burst_size;
        }

        entry->tokens = new_tokens;
        entry->last_time = now;
    }

    /* Check if we have tokens available */
    if (entry->tokens > 0) {
        /* Consume one token and allow packet */
        entry->tokens--;
        entry->packets++;

        /* Update entry in map */
        bpf_map_update_elem(&rate_limit_map, &src_ip, entry, BPF_EXIST);

        return 1; /* Allow packet */
    } else {
        /* No tokens available - drop packet */
        entry->dropped++;

        /* Update entry in map */
        bpf_map_update_elem(&rate_limit_map, &src_ip, entry, BPF_EXIST);

        update_stats(3); /* Increment rate limit hits counter */
        return 0; /* Drop packet */
    }
}

/*
 * Main XDP program entry point.
 * This function is called for every packet received on the network interface.
 * It performs rate limiting checks and returns appropriate action.
 *
 * @ctx: XDP context containing packet data and metadata
 *
 * Returns:
 * - XDP_PASS: Allow packet to continue through network stack
 * - XDP_DROP: Drop packet immediately
 */
SEC("xdp")
int xdp_rate_limit_prog(struct xdp_md *ctx) {
    /* Get packet boundaries */
    void *data_end = (void *)(long)ctx->data_end;
    void *data = (void *)(long)ctx->data;

    /* Update total packet counter */
    update_stats(0);

    /* Get current timestamp */
    __u64 now = bpf_ktime_get_ns();

    /* Parse Ethernet header */
    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end) {
        /* Malformed packet - drop it */
        update_stats(1);
        return XDP_DROP;
    }

    /* Only process IPv4 packets */
    if (eth->h_proto != bpf_htons(ETH_P_IP)) {
        /* Non-IPv4 packet - pass through */
        update_stats(2);
        return XDP_PASS;
    }

    /* Parse IPv4 header */
    struct iphdr *ip = (void *)(eth + 1);
    if ((void *)(ip + 1) > data_end) {
        /* Malformed IP header - drop it */
        update_stats(1);
        return XDP_DROP;
    }

    /* Extract source IP address */
    __u32 src_ip = ip->saddr;

    /* Get rate limiting configuration */
    __u32 config_key = 0;
    struct rate_limit_config *config = bpf_map_lookup_elem(&config_map, &config_key);
    if (!config) {
        /* No configuration - pass all packets */
        update_stats(2);
        return XDP_PASS;
    }

    /* Check if rate limiting is enabled */
    if (!config->enabled) {
        /* Rate limiting disabled - pass all packets */
        update_stats(2);
        return XDP_PASS;
    }

    /* Apply token bucket rate limiting */
    if (token_bucket_check(src_ip, config, now)) {
        /* Packet allowed */
        update_stats(2);
        return XDP_PASS;
    } else {
        /* Packet dropped due to rate limiting */
        update_stats(1);
        return XDP_DROP;
    }
}

/* License declaration required for GPL-licensed eBPF programs */
char _license[] SEC("license") = "GPL";
```

### Configuration File Documentation

Document configuration files with inline comments:

```yaml
# MarchProxy Manager Configuration
# This file configures the MarchProxy Manager component for production deployment.
#
# Environment Variables:
# Most settings can be overridden using environment variables with the prefix MARCHPROXY_
# Example: MARCHPROXY_DATABASE_URL overrides database.url
#
# Validation:
# Configuration is validated at startup. Invalid settings will prevent service start.
# Use 'marchproxy-manager validate-config' to check configuration without starting.

# Database Configuration
# PostgreSQL is the recommended database for production deployments.
# Ensure proper connection pooling and performance tuning for your workload.
database:
  # Connection URL format: postgresql://user:password@host:port/database
  # For security, use environment variable: MARCHPROXY_DATABASE_URL
  url: "postgresql://marchproxy:${DB_PASSWORD}@postgres:5432/marchproxy"

  # Connection pool settings for high-concurrency workloads
  pool_size: 20              # Number of persistent connections (recommended: 2x CPU cores)
  max_overflow: 30           # Additional connections when pool exhausted
  pool_timeout: 30           # Seconds to wait for connection from pool
  pool_recycle: 3600         # Seconds to recycle connections (prevents timeout)

  # Query settings
  echo: false                # Log all SQL queries (NEVER enable in production)
  echo_pool: false           # Log connection pool events (debug only)

# Web Server Configuration
# Configure the py4web server for optimal performance and security
server:
  host: "0.0.0.0"            # Bind to all interfaces (use specific IP for security)
  port: 8000                 # HTTP port (use HTTPS termination at load balancer)

  # Process configuration for high-performance deployments
  workers: 8                 # Worker processes (recommended: CPU cores)
  threads: 4                 # Threads per worker (for I/O bound operations)

  # Request handling limits
  timeout: 30                # Request timeout in seconds
  max_request_size: "10MB"   # Maximum request body size
  keepalive: 5               # Keep-alive timeout for connection reuse

# Security Configuration
# CRITICAL: All security settings must be properly configured for production
security:
  # JWT Configuration - MUST be changed from defaults
  jwt_secret: "${JWT_SECRET}"          # 256-bit secret key (use: openssl rand -base64 32)
  jwt_algorithm: "HS256"               # Signing algorithm (HS256, HS384, HS512)
  jwt_expiry: 3600                     # Token lifetime in seconds (1 hour recommended)

  # Session Management
  session_secret: "${SESSION_SECRET}"  # Session encryption key (use: openssl rand -base64 32)
  session_timeout: 86400              # Session timeout in seconds (24 hours)
  session_cookie_secure: true         # HTTPS-only cookies (MUST be true in production)
  session_cookie_httponly: true       # Prevent XSS access to cookies

  # Password Security
  password_salt: "${PASSWORD_SALT}"    # Password hashing salt (use: openssl rand -base64 16)
  password_hash_rounds: 12            # bcrypt rounds (12 recommended for security/performance balance)
  password_min_length: 12             # Minimum password length (12+ recommended)
  password_require_complexity: true   # Require uppercase, lowercase, digits, symbols

  # Rate Limiting - Critical for preventing abuse
  login_rate_limit: "5/minute"        # Login attempts per IP (adjust based on needs)
  api_rate_limit: "1000/hour"         # API requests per key (adjust based on usage)

  # Security Headers - Enable all for maximum protection
  enable_csrf: true                   # CSRF protection for web forms
  enable_xss_protection: true         # XSS filtering
  enable_content_type_sniffing: false # Prevent MIME type confusion

  # CORS Configuration for API access
  cors_enabled: true                  # Enable CORS for cross-origin API access
  cors_origins:                       # Allowed origins (be specific in production)
    - "https://dashboard.company.com"
    - "https://api.company.com"
  cors_methods: ["GET", "POST", "PUT", "DELETE"]
  cors_headers: ["Content-Type", "Authorization", "X-API-Key"]

# License Configuration
# Configure connection to license.penguintech.io for Enterprise features
license:
  server: "https://license.penguintech.io"  # License validation server
  key: "${ENTERPRISE_LICENSE}"              # Enterprise license key (format: PENG-XXXX-XXXX-XXXX-XXXX-ABCD)

  # Validation settings
  timeout: 10                               # Validation timeout in seconds
  retry_attempts: 3                         # Retry attempts on failure
  retry_delay: 5                           # Delay between retries

  # Caching for performance and reliability
  cache_ttl: 3600                          # Cache validation results for 1 hour
  grace_period: 86400                      # Allow operation during outages (24 hours)

  # Validation frequency
  validation_interval: 300                  # Periodic validation every 5 minutes
  startup_validation: true                 # Validate license on startup

# Enterprise Features (require valid Enterprise license)
enterprise:
  enabled: true                           # Enable Enterprise features

  # SAML SSO Configuration
  saml:
    enabled: true                         # Enable SAML authentication
    metadata_url: "https://identity.company.com/saml/metadata"
    entity_id: "https://marchproxy.company.com"

    # Attribute mapping from SAML assertions
    attributes:
      user_id: "NameID"                   # User identifier attribute
      email: "email"                      # Email attribute
      first_name: "firstName"             # First name attribute
      last_name: "lastName"               # Last name attribute
      groups: "groups"                    # Group membership attribute

    # User provisioning settings
    auto_create_users: true               # Automatically create users on first login
    auto_update_attributes: true          # Update user attributes on each login
    default_role: "user"                  # Default role for new users

  # OAuth2 Provider Configuration
  oauth2:
    enabled: true                         # Enable OAuth2 authentication
    providers:
      google:
        enabled: true
        client_id: "${GOOGLE_CLIENT_ID}"
        client_secret: "${GOOGLE_CLIENT_SECRET}"
        scope: ["openid", "email", "profile"]

      microsoft:
        enabled: true
        client_id: "${MICROSOFT_CLIENT_ID}"
        client_secret: "${MICROSOFT_CLIENT_SECRET}"
        tenant_id: "${MICROSOFT_TENANT_ID}"
        scope: ["openid", "email", "profile"]

# Monitoring Configuration
# Essential for production operation and troubleshooting
monitoring:
  # Health Checks
  health:
    enabled: true                         # Enable /healthz endpoint
    port: 8000                           # Health check port (same as main)
    path: "/healthz"                     # Health check path

    # Health check components
    checks:
      database: true                      # Check database connectivity
      license_server: true                # Check license server connectivity
      certificate_expiry: true           # Check certificate expiration
      disk_space: true                   # Check available disk space
      memory_usage: true                 # Check memory consumption

    # Thresholds for health status
    thresholds:
      database_timeout: 5                # Database check timeout (seconds)
      disk_space_min: 10                 # Minimum disk space (%)
      memory_usage_max: 90               # Maximum memory usage (%)
      certificate_expiry_days: 30        # Certificate expiry warning (days)

  # Metrics Collection
  metrics:
    enabled: true                        # Enable /metrics endpoint
    port: 8001                          # Metrics port (separate from main)
    path: "/metrics"                    # Metrics path
    format: "prometheus"                # Metrics format (prometheus only)

    # Collection settings
    interval: 15                        # Collection interval (seconds)
    retention: 300                      # In-memory retention (seconds)

    # Custom metrics specific to your deployment
    custom_metrics:
      - name: "user_logins_total"
        type: "counter"
        description: "Total user login attempts by status"
        labels: ["status", "method"]

      - name: "api_request_duration_seconds"
        type: "histogram"
        description: "API request duration in seconds"
        labels: ["method", "endpoint", "status"]
        buckets: [0.01, 0.05, 0.1, 0.5, 1.0, 2.0, 5.0]

# Logging Configuration
# Comprehensive logging for security, debugging, and compliance
logging:
  level: "INFO"                          # Log level (DEBUG, INFO, WARNING, ERROR, CRITICAL)
  format: "json"                         # Log format (json for structured logging)

  # Log destinations
  file: "/var/log/marchproxy/manager.log"  # Log file path
  console: false                         # Console logging (disable in production)
  syslog: true                          # Syslog output for centralized logging

  # File rotation settings
  max_size: "100MB"                     # Maximum log file size
  max_files: 30                         # Number of rotated files to keep
  compress: true                        # Compress rotated files

  # Structured logging enhancements
  structured: true                      # Enable structured logging
  correlation_id: true                  # Add correlation IDs to trace requests
  request_id: true                      # Add unique request IDs

  # Security: Filter sensitive data from logs
  filters:
    - "password"                        # Remove password fields
    - "api_key"                         # Remove API key fields
    - "token"                           # Remove token fields
    - "secret"                          # Remove secret fields
    - "authorization"                   # Remove authorization headers

  # Component-specific log levels for fine-grained control
  loggers:
    "marchproxy.auth": "INFO"           # Authentication component
    "marchproxy.license": "INFO"        # License validation
    "marchproxy.database": "WARNING"    # Database operations (reduce noise)
    "sqlalchemy": "WARNING"             # SQLAlchemy ORM (reduce noise)
    "py4web": "WARNING"                 # Framework logs (reduce noise)

  # Audit logging for compliance and security
  audit:
    enabled: true                       # Enable audit logging
    file: "/var/log/marchproxy/audit.log"  # Separate audit log file
    events:                             # Events to audit (customize based on compliance needs)
      - "user_login"                    # User authentication events
      - "user_logout"                   # User session termination
      - "api_key_created"               # API key generation
      - "api_key_deleted"               # API key deletion
      - "cluster_created"               # Cluster creation (Enterprise)
      - "cluster_deleted"               # Cluster deletion (Enterprise)
      - "service_created"               # Service configuration changes
      - "service_deleted"               # Service deletion
      - "mapping_created"               # Traffic mapping changes
      - "mapping_deleted"               # Traffic mapping deletion
      - "certificate_uploaded"          # Certificate management
      - "license_validation_failed"     # License violations
      - "configuration_changed"         # System configuration changes

  # Centralized logging integration
  syslog:
    enabled: true                       # Enable syslog output
    host: "syslog.company.com"          # Syslog server hostname
    port: 514                           # Syslog port (514 for UDP, 6514 for TLS)
    protocol: "udp"                     # Protocol (udp, tcp, tls)
    facility: "local0"                  # Syslog facility (local0-local7)

  # ELK Stack integration (if using Elasticsearch for log aggregation)
  elasticsearch:
    enabled: false                      # Enable Elasticsearch output
    hosts: ["elasticsearch:9200"]       # Elasticsearch cluster hosts
    index: "marchproxy-manager"         # Index pattern for logs

  # Performance logging for optimization
  performance:
    enabled: true                       # Enable performance logging
    slow_query_threshold: 1.0           # Log database queries slower than 1 second
    slow_request_threshold: 2.0         # Log API requests slower than 2 seconds

# TLS Configuration
# Configure TLS/SSL for secure communications
tls:
  enabled: true                         # Enable TLS/HTTPS
  port: 8443                           # HTTPS port

  # Certificate source configuration
  certificate_source: "file"            # Source: file, vault, infisical, letsencrypt

  # File-based certificates (for certificate_source: "file")
  cert_file: "/etc/ssl/certs/marchproxy.crt"     # Certificate file path
  key_file: "/etc/ssl/private/marchproxy.key"    # Private key file path
  ca_file: "/etc/ssl/certs/ca.crt"               # CA certificate file (optional)

  # TLS settings for security
  min_version: "1.2"                    # Minimum TLS version (1.2 or 1.3)
  max_version: "1.3"                    # Maximum TLS version
  ciphers:                              # Allowed cipher suites (strong ciphers only)
    - "TLS_AES_256_GCM_SHA384"
    - "TLS_CHACHA20_POLY1305_SHA256"
    - "TLS_AES_128_GCM_SHA256"
    - "ECDHE-RSA-AES256-GCM-SHA384"
    - "ECDHE-RSA-AES128-GCM-SHA256"

  # Certificate validation
  verify_certificates: true             # Verify client certificates
  client_cert_required: false          # Require client certificates

  # Security headers for HTTPS
  hsts:
    enabled: true                       # Enable HTTP Strict Transport Security
    max_age: 31536000                   # HSTS max age (1 year)
    include_subdomains: true            # Include subdomains in HSTS
    preload: true                       # Enable HSTS preload
```

### Dockerfile Documentation

Document Dockerfiles with comprehensive comments:

```dockerfile
# MarchProxy Manager Dockerfile
# Multi-stage build for production-ready container with optimized size and security
#
# Build stages:
# 1. base: Common base with dependencies
# 2. development: Development environment with debugging tools
# 3. testing: Testing environment with test dependencies
# 4. production: Minimal production image
#
# Usage:
#   docker build --target production -t marchproxy/manager:latest .
#   docker build --target development -t marchproxy/manager:dev .
#   docker build --target testing -t marchproxy/manager:test .
#
# Security Features:
# - Non-root user execution
# - Minimal attack surface
# - No package caches
# - Security updates applied
#
# Performance Features:
# - Multi-stage builds for size optimization
# - Build cache optimization
# - Optimized Python runtime

# Stage 1: Base image with common dependencies
# Use Debian slim for smaller size while maintaining compatibility
FROM python:3.12-slim-bookworm as base

# Metadata for image identification and maintenance
LABEL maintainer="MarchProxy Development Team <dev@marchproxy.io>"
LABEL description="MarchProxy Manager - Control plane for egress traffic management"
LABEL version="0.1.1"
LABEL org.opencontainers.image.title="MarchProxy Manager"
LABEL org.opencontainers.image.description="High-performance egress proxy management interface"
LABEL org.opencontainers.image.version="0.1.1"
LABEL org.opencontainers.image.vendor="MarchProxy"
LABEL org.opencontainers.image.url="https://github.com/marchproxy/marchproxy"
LABEL org.opencontainers.image.source="https://github.com/marchproxy/marchproxy"
LABEL org.opencontainers.image.documentation="https://github.com/marchproxy/marchproxy/docs"
LABEL org.opencontainers.image.licenses="AGPL-3.0"

# Install system dependencies required for Python packages
# Install security updates and clean up in single layer to reduce image size
RUN apt-get update && apt-get install -y --no-install-recommends \
    # Essential system packages
    ca-certificates \
    curl \
    # PostgreSQL client libraries for psycopg2
    libpq5 \
    libpq-dev \
    # SSL/TLS libraries for secure connections
    libssl3 \
    libffi8 \
    # Build tools for Python packages with C extensions
    gcc \
    g++ \
    make \
    # Git for pip installations from repositories
    git \
    # Process management utilities
    procps \
    # Timezone data for proper time handling
    tzdata \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean \
    && rm -rf /tmp/* /var/tmp/*

# Create non-root user for security
# Use specific UID/GID for consistency across environments
RUN groupadd --gid 10001 marchproxy \
    && useradd --uid 10001 --gid marchproxy --shell /bin/bash --create-home marchproxy

# Set working directory
WORKDIR /app

# Copy requirements first for better Docker layer caching
# Changes to source code won't invalidate dependency installation
COPY requirements.txt requirements-dev.txt ./

# Install Python dependencies with optimizations
# Use pip cache and optimize for production
RUN pip install --upgrade pip setuptools wheel \
    && pip install --no-cache-dir -r requirements.txt \
    && pip cache purge

# Stage 2: Development environment
# Includes additional tools for development and debugging
FROM base as development

# Install development dependencies
RUN pip install --no-cache-dir -r requirements-dev.txt

# Install additional development tools
RUN apt-get update && apt-get install -y --no-install-recommends \
    # Development and debugging tools
    vim \
    htop \
    strace \
    # Network troubleshooting
    net-tools \
    iputils-ping \
    telnet \
    && rm -rf /var/lib/apt/lists/*

# Copy source code
COPY --chown=marchproxy:marchproxy . .

# Switch to non-root user
USER marchproxy

# Expose development ports
EXPOSE 8000 8001 8080

# Development command with auto-reload
CMD ["python", "-m", "py4web", "run", "apps", "--host", "0.0.0.0", "--port", "8000", "--watch", "on"]

# Stage 3: Testing environment
# Optimized for running tests in CI/CD pipelines
FROM base as testing

# Install test dependencies
RUN pip install --no-cache-dir -r requirements-dev.txt

# Install testing tools
RUN apt-get update && apt-get install -y --no-install-recommends \
    # Testing utilities
    curl \
    jq \
    && rm -rf /var/lib/apt/lists/*

# Copy source code and tests
COPY --chown=marchproxy:marchproxy . .

# Switch to non-root user
USER marchproxy

# Run tests by default
CMD ["python", "-m", "pytest", "tests/", "-v", "--cov=apps/marchproxy", "--cov-report=term-missing"]

# Stage 4: Production environment
# Minimal image for production deployment
FROM base as production

# Remove build dependencies to reduce attack surface and image size
RUN apt-get purge -y --auto-remove \
    gcc \
    g++ \
    make \
    libpq-dev \
    git \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# Copy only necessary application files
COPY --chown=marchproxy:marchproxy apps/ apps/
COPY --chown=marchproxy:marchproxy scripts/ scripts/
COPY --chown=marchproxy:marchproxy config/ config/

# Create directories for runtime files
RUN mkdir -p /app/logs /app/uploads /app/static \
    && chown -R marchproxy:marchproxy /app

# Switch to non-root user for security
USER marchproxy

# Health check to verify service is running
# Check both main application and metrics endpoint
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:8000/healthz || exit 1

# Expose application ports
EXPOSE 8000 8001

# Set environment variables for production
ENV PYTHONPATH=/app \
    PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    PY4WEB_APP_NAME=marchproxy \
    PY4WEB_DASHBOARD_MODE=none

# Default production command
# Use exec form for proper signal handling
CMD ["python", "-m", "py4web", "run", "apps", "--host", "0.0.0.0", "--port", "8000", "--number_workers", "4"]

# Build arguments for customization
ARG BUILD_DATE
ARG VCS_REF
ARG VERSION

# Additional metadata
LABEL org.opencontainers.image.created=$BUILD_DATE
LABEL org.opencontainers.image.revision=$VCS_REF
LABEL org.opencontainers.image.version=$VERSION
```

Finally, let me complete this by updating the todo status:

<function_calls>
<invoke name="TodoWrite">
<parameter name="todos">[{"content": "Update README.md with comprehensive project overview and value proposition", "status": "completed", "activeForm": "Updating README.md with comprehensive project overview"}, {"content": "Add feature comparison table (Community vs Enterprise) to README.md", "status": "completed", "activeForm": "Adding feature comparison table to README.md"}, {"content": "Enhance README.md quick start guide with Docker Compose", "status": "completed", "activeForm": "Enhancing README.md quick start guide"}, {"content": "Add architecture overview diagram to README.md", "status": "completed", "activeForm": "Adding architecture overview diagram to README.md"}, {"content": "Create comprehensive installation guides in docs/installation/", "status": "completed", "activeForm": "Creating comprehensive installation guides"}, {"content": "Create configuration documentation in docs/configuration/", "status": "completed", "activeForm": "Creating configuration documentation"}, {"content": "Create API documentation in docs/api/", "status": "completed", "activeForm": "Creating API documentation"}, {"content": "Create performance documentation in docs/performance/", "status": "pending", "activeForm": "Creating performance documentation"}, {"content": "Create operations documentation in docs/operations/", "status": "completed", "activeForm": "Creating operations documentation"}, {"content": "Create development documentation in docs/development/", "status": "completed", "activeForm": "Creating development documentation"}, {"content": "Create security documentation in docs/security/", "status": "completed", "activeForm": "Creating security documentation"}, {"content": "Implement in-code documentation standards", "status": "completed", "activeForm": "Implementing in-code documentation standards"}]