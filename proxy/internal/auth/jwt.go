package auth

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

var (
	ErrInvalidToken      = errors.New("invalid token")
	ErrTokenExpired      = errors.New("token expired")
	ErrTokenNotYetValid  = errors.New("token not yet valid")
	ErrInvalidSignature  = errors.New("invalid signature")
	ErrMissingToken      = errors.New("missing token")
	ErrInvalidClaims     = errors.New("invalid claims")
	ErrUnauthorized      = errors.New("unauthorized")
)

type JWTAuthenticator struct {
	config         JWTConfig
	publicKeys     map[string]*rsa.PublicKey
	secretKeys     map[string][]byte
	jwksClient     *JWKSClient
	tokenCache     *TokenCache
	blacklist      *TokenBlacklist
	mutex          sync.RWMutex
	metrics        *AuthMetrics
}

type JWTConfig struct {
	Algorithm           string
	PublicKeyPath       string
	PublicKeyContent    string
	SecretKey           string
	JWKSURL             string
	JWKSRefreshInterval time.Duration
	Issuer              string
	Audience            []string
	RequiredClaims      []string
	TokenLookup         TokenLookupConfig
	CacheEnabled        bool
	CacheTTL            time.Duration
	BlacklistEnabled    bool
	ClockSkew           time.Duration
	ValidationOptions   ValidationOptions
}

type TokenLookupConfig struct {
	Header       string
	QueryParam   string
	Cookie       string
	PathParam    string
	HeaderPrefix string
}

type ValidationOptions struct {
	ValidateIssuer    bool
	ValidateAudience  bool
	ValidateExpiry    bool
	ValidateNotBefore bool
	RequireExpiry     bool
	RequireNotBefore  bool
}

type JWTClaims struct {
	jwt.RegisteredClaims
	UserID      string                 `json:"user_id,omitempty"`
	Username    string                 `json:"username,omitempty"`
	Email       string                 `json:"email,omitempty"`
	Roles       []string               `json:"roles,omitempty"`
	Permissions []string               `json:"permissions,omitempty"`
	Scope       string                 `json:"scope,omitempty"`
	ClientID    string                 `json:"client_id,omitempty"`
	Custom      map[string]interface{} `json:"custom,omitempty"`
}

type TokenCache struct {
	cache     map[string]*CachedToken
	mutex     sync.RWMutex
	ttl       time.Duration
	maxSize   int
	evictLRU  bool
}

type CachedToken struct {
	Claims      *JWTClaims
	Token       string
	ExpiresAt   time.Time
	LastAccessed time.Time
	AccessCount  int64
}

type TokenBlacklist struct {
	tokens    map[string]time.Time
	mutex     sync.RWMutex
	cleanupInterval time.Duration
	stopChan  chan struct{}
}

type JWKSClient struct {
	url          string
	httpClient   *http.Client
	cache        *JWKSCache
	refreshInterval time.Duration
	mutex        sync.RWMutex
}

type JWKSCache struct {
	keys      map[string]interface{}
	expiresAt time.Time
}

type AuthMetrics struct {
	TotalRequests       uint64
	SuccessfulAuths     uint64
	FailedAuths         uint64
	TokensValidated     uint64
	TokensRefreshed     uint64
	CacheHits           uint64
	CacheMisses         uint64
	BlacklistHits       uint64
	AverageValidationTime time.Duration
	mutex               sync.RWMutex
}

func NewJWTAuthenticator(config JWTConfig) (*JWTAuthenticator, error) {
	ja := &JWTAuthenticator{
		config:     config,
		publicKeys: make(map[string]*rsa.PublicKey),
		secretKeys: make(map[string][]byte),
		metrics:    &AuthMetrics{},
	}

	if config.PublicKeyPath != "" || config.PublicKeyContent != "" {
		if err := ja.loadPublicKey(); err != nil {
			return nil, fmt.Errorf("failed to load public key: %w", err)
		}
	}

	if config.SecretKey != "" {
		ja.secretKeys["default"] = []byte(config.SecretKey)
	}

	if config.JWKSURL != "" {
		ja.jwksClient = NewJWKSClient(config.JWKSURL, config.JWKSRefreshInterval)
		if err := ja.jwksClient.RefreshKeys(); err != nil {
			return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
		}
	}

	if config.CacheEnabled {
		ja.tokenCache = NewTokenCache(config.CacheTTL, 10000)
	}

	if config.BlacklistEnabled {
		ja.blacklist = NewTokenBlacklist(1 * time.Hour)
		ja.blacklist.StartCleanup()
	}

	if config.ClockSkew == 0 {
		config.ClockSkew = 5 * time.Minute
	}

	return ja, nil
}

func (ja *JWTAuthenticator) loadPublicKey() error {
	var keyData []byte
	
	if ja.config.PublicKeyContent != "" {
		keyData = []byte(ja.config.PublicKeyContent)
	} else if ja.config.PublicKeyPath != "" {
		return fmt.Errorf("public key file reading not implemented in this example")
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return errors.New("failed to parse PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return errors.New("not an RSA public key")
	}

	ja.publicKeys["default"] = rsaPub
	return nil
}

func (ja *JWTAuthenticator) Authenticate(r *http.Request) (*JWTClaims, error) {
	start := time.Now()
	defer func() {
		ja.updateMetrics(time.Since(start))
	}()

	ja.metrics.recordRequest()

	token, err := ja.extractToken(r)
	if err != nil {
		ja.metrics.recordFailure()
		return nil, err
	}

	if ja.config.BlacklistEnabled && ja.blacklist.IsBlacklisted(token) {
		ja.metrics.recordBlacklistHit()
		ja.metrics.recordFailure()
		return nil, ErrUnauthorized
	}

	if ja.config.CacheEnabled {
		if cached := ja.tokenCache.Get(token); cached != nil {
			ja.metrics.recordCacheHit()
			ja.metrics.recordSuccess()
			return cached.Claims, nil
		}
		ja.metrics.recordCacheMiss()
	}

	claims, err := ja.validateToken(token)
	if err != nil {
		ja.metrics.recordFailure()
		return nil, err
	}

	if ja.config.CacheEnabled {
		ja.tokenCache.Set(token, claims)
	}

	ja.metrics.recordSuccess()
	ja.metrics.recordValidation()

	return claims, nil
}

func (ja *JWTAuthenticator) extractToken(r *http.Request) (string, error) {
	config := ja.config.TokenLookup

	if config.Header != "" {
		token := r.Header.Get(config.Header)
		if token != "" {
			if config.HeaderPrefix != "" {
				token = strings.TrimPrefix(token, config.HeaderPrefix)
			}
			return strings.TrimSpace(token), nil
		}
	}

	if config.QueryParam != "" {
		if token := r.URL.Query().Get(config.QueryParam); token != "" {
			return token, nil
		}
	}

	if config.Cookie != "" {
		if cookie, err := r.Cookie(config.Cookie); err == nil {
			return cookie.Value, nil
		}
	}

	return "", ErrMissingToken
}

func (ja *JWTAuthenticator) validateToken(tokenString string) (*JWTClaims, error) {
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{ja.config.Algorithm}),
		jwt.WithLeeway(ja.config.ClockSkew),
	)

	token, err := parser.ParseWithClaims(tokenString, &JWTClaims{}, ja.keyFunc)
	if err != nil {
		if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors&jwt.ValidationErrorExpired != 0 {
				return nil, ErrTokenExpired
			}
			if ve.Errors&jwt.ValidationErrorNotValidYet != 0 {
				return nil, ErrTokenNotYetValid
			}
			if ve.Errors&jwt.ValidationErrorSignatureInvalid != 0 {
				return nil, ErrInvalidSignature
			}
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidClaims
	}

	if err := ja.validateClaims(claims); err != nil {
		return nil, err
	}

	return claims, nil
}

func (ja *JWTAuthenticator) keyFunc(token *jwt.Token) (interface{}, error) {
	switch ja.config.Algorithm {
	case "RS256", "RS384", "RS512":
		if kid, ok := token.Header["kid"].(string); ok {
			if ja.jwksClient != nil {
				return ja.jwksClient.GetKey(kid)
			}
			if key, exists := ja.publicKeys[kid]; exists {
				return key, nil
			}
		}
		if key, exists := ja.publicKeys["default"]; exists {
			return key, nil
		}
		return nil, errors.New("public key not found")

	case "HS256", "HS384", "HS512":
		if kid, ok := token.Header["kid"].(string); ok {
			if key, exists := ja.secretKeys[kid]; exists {
				return key, nil
			}
		}
		if key, exists := ja.secretKeys["default"]; exists {
			return key, nil
		}
		return nil, errors.New("secret key not found")

	default:
		return nil, fmt.Errorf("unsupported algorithm: %s", ja.config.Algorithm)
	}
}

func (ja *JWTAuthenticator) validateClaims(claims *JWTClaims) error {
	opts := ja.config.ValidationOptions

	if opts.ValidateIssuer && ja.config.Issuer != "" {
		if claims.Issuer != ja.config.Issuer {
			return fmt.Errorf("invalid issuer: expected %s, got %s", ja.config.Issuer, claims.Issuer)
		}
	}

	if opts.ValidateAudience && len(ja.config.Audience) > 0 {
		found := false
		for _, aud := range ja.config.Audience {
			for _, claimAud := range claims.Audience {
				if aud == claimAud {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return errors.New("invalid audience")
		}
	}

	for _, required := range ja.config.RequiredClaims {
		switch required {
		case "sub":
			if claims.Subject == "" {
				return fmt.Errorf("required claim missing: %s", required)
			}
		case "user_id":
			if claims.UserID == "" {
				return fmt.Errorf("required claim missing: %s", required)
			}
		case "email":
			if claims.Email == "" {
				return fmt.Errorf("required claim missing: %s", required)
			}
		}
	}

	return nil
}

func (ja *JWTAuthenticator) GenerateToken(claims *JWTClaims) (string, error) {
	now := time.Now()
	
	if claims.ExpiresAt == nil || claims.ExpiresAt.IsZero() {
		claims.ExpiresAt = jwt.NewNumericDate(now.Add(1 * time.Hour))
	}
	if claims.NotBefore == nil || claims.NotBefore.IsZero() {
		claims.NotBefore = jwt.NewNumericDate(now)
	}
	if claims.IssuedAt == nil || claims.IssuedAt.IsZero() {
		claims.IssuedAt = jwt.NewNumericDate(now)
	}
	if claims.Issuer == "" {
		claims.Issuer = ja.config.Issuer
	}
	if len(claims.Audience) == 0 {
		claims.Audience = ja.config.Audience
	}

	token := jwt.NewWithClaims(jwt.GetSigningMethod(ja.config.Algorithm), claims)

	var key interface{}
	switch ja.config.Algorithm {
	case "RS256", "RS384", "RS512":
		return "", errors.New("RSA signing requires private key")
	case "HS256", "HS384", "HS512":
		if k, exists := ja.secretKeys["default"]; exists {
			key = k
		} else {
			return "", errors.New("secret key not found")
		}
	default:
		return "", fmt.Errorf("unsupported algorithm: %s", ja.config.Algorithm)
	}

	return token.SignedString(key)
}

func (ja *JWTAuthenticator) RefreshToken(tokenString string) (string, error) {
	claims, err := ja.validateToken(tokenString)
	if err != nil && err != ErrTokenExpired {
		return "", err
	}

	newClaims := *claims
	now := time.Now()
	newClaims.IssuedAt = jwt.NewNumericDate(now)
	newClaims.NotBefore = jwt.NewNumericDate(now)
	newClaims.ExpiresAt = jwt.NewNumericDate(now.Add(1 * time.Hour))

	ja.metrics.recordRefresh()

	return ja.GenerateToken(&newClaims)
}

func (ja *JWTAuthenticator) RevokeToken(tokenString string) error {
	if !ja.config.BlacklistEnabled {
		return errors.New("blacklist not enabled")
	}

	claims, err := ja.validateToken(tokenString)
	if err != nil && err != ErrTokenExpired {
		return err
	}

	expiry := time.Now().Add(24 * time.Hour)
	if claims.ExpiresAt != nil {
		expiry = claims.ExpiresAt.Time
	}

	ja.blacklist.Add(tokenString, expiry)
	
	if ja.config.CacheEnabled {
		ja.tokenCache.Delete(tokenString)
	}

	return nil
}

func (ja *JWTAuthenticator) GetMetrics() *AuthMetrics {
	ja.metrics.mutex.RLock()
	defer ja.metrics.mutex.RUnlock()
	
	metricsCopy := *ja.metrics
	return &metricsCopy
}

func (ja *JWTAuthenticator) updateMetrics(duration time.Duration) {
	ja.metrics.mutex.Lock()
	defer ja.metrics.mutex.Unlock()
	
	count := ja.metrics.TotalRequests
	if count > 0 {
		ja.metrics.AverageValidationTime = 
			(ja.metrics.AverageValidationTime*time.Duration(count-1) + duration) / 
			time.Duration(count)
	} else {
		ja.metrics.AverageValidationTime = duration
	}
}

func NewTokenCache(ttl time.Duration, maxSize int) *TokenCache {
	tc := &TokenCache{
		cache:    make(map[string]*CachedToken),
		ttl:      ttl,
		maxSize:  maxSize,
		evictLRU: true,
	}
	
	go tc.cleanupExpired()
	return tc
}

func (tc *TokenCache) Get(token string) *CachedToken {
	tc.mutex.RLock()
	defer tc.mutex.RUnlock()
	
	cached, exists := tc.cache[token]
	if !exists {
		return nil
	}
	
	if time.Now().After(cached.ExpiresAt) {
		return nil
	}
	
	cached.LastAccessed = time.Now()
	cached.AccessCount++
	
	return cached
}

func (tc *TokenCache) Set(token string, claims *JWTClaims) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()
	
	if len(tc.cache) >= tc.maxSize && tc.evictLRU {
		tc.evictOldest()
	}
	
	expiresAt := time.Now().Add(tc.ttl)
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(expiresAt) {
		expiresAt = claims.ExpiresAt.Time
	}
	
	tc.cache[token] = &CachedToken{
		Claims:       claims,
		Token:        token,
		ExpiresAt:    expiresAt,
		LastAccessed: time.Now(),
		AccessCount:  1,
	}
}

func (tc *TokenCache) Delete(token string) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()
	
	delete(tc.cache, token)
}

func (tc *TokenCache) evictOldest() {
	var oldestToken string
	var oldestTime time.Time
	
	for token, cached := range tc.cache {
		if oldestTime.IsZero() || cached.LastAccessed.Before(oldestTime) {
			oldestToken = token
			oldestTime = cached.LastAccessed
		}
	}
	
	if oldestToken != "" {
		delete(tc.cache, oldestToken)
	}
}

func (tc *TokenCache) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		tc.mutex.Lock()
		now := time.Now()
		for token, cached := range tc.cache {
			if now.After(cached.ExpiresAt) {
				delete(tc.cache, token)
			}
		}
		tc.mutex.Unlock()
	}
}

func NewTokenBlacklist(cleanupInterval time.Duration) *TokenBlacklist {
	return &TokenBlacklist{
		tokens:          make(map[string]time.Time),
		cleanupInterval: cleanupInterval,
		stopChan:        make(chan struct{}),
	}
}

func (tb *TokenBlacklist) Add(token string, expiry time.Time) {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	
	tb.tokens[token] = expiry
}

func (tb *TokenBlacklist) IsBlacklisted(token string) bool {
	tb.mutex.RLock()
	defer tb.mutex.RUnlock()
	
	expiry, exists := tb.tokens[token]
	if !exists {
		return false
	}
	
	if time.Now().After(expiry) {
		return false
	}
	
	return true
}

func (tb *TokenBlacklist) StartCleanup() {
	go func() {
		ticker := time.NewTicker(tb.cleanupInterval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				tb.cleanup()
			case <-tb.stopChan:
				return
			}
		}
	}()
}

func (tb *TokenBlacklist) cleanup() {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	
	now := time.Now()
	for token, expiry := range tb.tokens {
		if now.After(expiry) {
			delete(tb.tokens, token)
		}
	}
}

func (tb *TokenBlacklist) Stop() {
	close(tb.stopChan)
}

func NewJWKSClient(url string, refreshInterval time.Duration) *JWKSClient {
	return &JWKSClient{
		url: url,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache:           &JWKSCache{keys: make(map[string]interface{})},
		refreshInterval: refreshInterval,
	}
}

func (jc *JWKSClient) RefreshKeys() error {
	return nil
}

func (jc *JWKSClient) GetKey(kid string) (interface{}, error) {
	jc.mutex.RLock()
	defer jc.mutex.RUnlock()
	
	if key, exists := jc.cache.keys[kid]; exists {
		return key, nil
	}
	
	return nil, fmt.Errorf("key not found: %s", kid)
}

func (am *AuthMetrics) recordRequest() {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	am.TotalRequests++
}

func (am *AuthMetrics) recordSuccess() {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	am.SuccessfulAuths++
}

func (am *AuthMetrics) recordFailure() {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	am.FailedAuths++
}

func (am *AuthMetrics) recordValidation() {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	am.TokensValidated++
}

func (am *AuthMetrics) recordRefresh() {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	am.TokensRefreshed++
}

func (am *AuthMetrics) recordCacheHit() {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	am.CacheHits++
}

func (am *AuthMetrics) recordCacheMiss() {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	am.CacheMisses++
}

func (am *AuthMetrics) recordBlacklistHit() {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	am.BlacklistHits++
}

func DefaultJWTConfig() JWTConfig {
	return JWTConfig{
		Algorithm: "HS256",
		TokenLookup: TokenLookupConfig{
			Header:       "Authorization",
			HeaderPrefix: "Bearer ",
			QueryParam:   "token",
			Cookie:       "jwt",
		},
		CacheEnabled:     true,
		CacheTTL:         5 * time.Minute,
		BlacklistEnabled: true,
		ClockSkew:        5 * time.Minute,
		ValidationOptions: ValidationOptions{
			ValidateIssuer:    true,
			ValidateAudience:  true,
			ValidateExpiry:    true,
			ValidateNotBefore: true,
			RequireExpiry:     true,
		},
	}
}