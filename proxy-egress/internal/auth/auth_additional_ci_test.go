//go:build ci
// +build ci

package auth

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestOAuth2Authenticator_NewOAuth2Authenticator tests OAuth2 creation with various configs
func TestOAuth2Authenticator_NewOAuth2Authenticator(t *testing.T) {
	tests := []struct {
		name      string
		config    OAuth2Config
		wantErr   bool
		errMsg    string
	}{
		{
			name: "valid config",
			config: OAuth2Config{
				ClientID:     "client-123",
				ClientSecret: "secret-456",
			},
			wantErr: false,
		},
		{
			name: "missing client ID",
			config: OAuth2Config{
				ClientID:     "",
				ClientSecret: "secret",
			},
			wantErr: true,
			errMsg:  "client ID",
		},
		{
			name: "missing client secret",
			config: OAuth2Config{
				ClientID:     "client",
				ClientSecret: "",
			},
			wantErr: true,
			errMsg:  "secret",
		},
		{
			name: "both missing",
			config: OAuth2Config{
				ClientID:     "",
				ClientSecret: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth, err := NewOAuth2Authenticator(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("got error %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errMsg)
				}
			}
			if !tt.wantErr && auth == nil {
				t.Error("expected non-nil authenticator")
			}
		})
	}
}

// TestOAuth2Authenticator_GetAuthorizationURL tests URL generation with and without PKCE
func TestOAuth2Authenticator_GetAuthorizationURL(t *testing.T) {
	config := OAuth2Config{
		ClientID:         "client-123",
		ClientSecret:     "secret",
		AuthorizationURL: "https://auth.example.com/authorize",
		RedirectURL:      "https://app.example.com/callback",
		ResponseType:     "code",
		Scopes:           []string{"openid", "profile"},
		UsePKCE:          false,
	}

	auth, err := NewOAuth2Authenticator(config)
	if err != nil {
		t.Fatalf("failed to create authenticator: %v", err)
	}

	url, err := auth.GetAuthorizationURL("https://app.example.com/callback", "", nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !strings.Contains(url, "client_id=client-123") {
		t.Error("URL should contain client_id")
	}
	if !strings.Contains(url, "response_type=code") {
		t.Error("URL should contain response_type")
	}
	if !strings.Contains(url, "scope=openid+profile") {
		t.Error("URL should contain scopes")
	}
}

// TestOAuth2Authenticator_GetAuthorizationURL_WithPKCE tests PKCE flow
func TestOAuth2Authenticator_GetAuthorizationURL_WithPKCE(t *testing.T) {
	config := OAuth2Config{
		ClientID:         "client-123",
		ClientSecret:     "secret",
		AuthorizationURL: "https://auth.example.com/authorize",
		RedirectURL:      "https://app.example.com/callback",
		UsePKCE:          true,
		PKCEChallengeMethod: "S256",
	}

	auth, err := NewOAuth2Authenticator(config)
	if err != nil {
		t.Fatalf("failed to create authenticator: %v", err)
	}

	url, err := auth.GetAuthorizationURL("https://app.example.com/callback", "", nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !strings.Contains(url, "code_challenge=") {
		t.Error("URL should contain code_challenge for PKCE")
	}
	if !strings.Contains(url, "code_challenge_method=") {
		t.Error("URL should contain code_challenge_method")
	}
}

// TestOAuth2Authenticator_ExchangeAuthorizationCode_InvalidState tests error handling
func TestOAuth2Authenticator_ExchangeAuthorizationCode_InvalidState(t *testing.T) {
	config := OAuth2Config{
		ClientID:      "client-123",
		ClientSecret:  "secret",
		TokenURL:      "https://auth.example.com/token",
		RedirectURL:   "https://app.example.com/callback",
	}

	auth, err := NewOAuth2Authenticator(config)
	if err != nil {
		t.Fatalf("failed to create authenticator: %v", err)
	}

	ctx := context.Background()
	token, err := auth.ExchangeAuthorizationCode(ctx, "code123", "invalid-state")
	if err == nil {
		t.Error("expected error for invalid state")
	}
	if token != nil {
		t.Error("expected nil token for invalid state")
	}
}

// TestOAuth2Authenticator_RefreshAccessToken_NetworkError tests error handling
func TestOAuth2Authenticator_RefreshAccessToken_NetworkError(t *testing.T) {
	config := OAuth2Config{
		ClientID:     "client-123",
		ClientSecret: "secret",
		TokenURL:     "https://invalid-host-12345.example.com/token",
	}

	auth, err := NewOAuth2Authenticator(config)
	if err != nil {
		t.Fatalf("failed to create authenticator: %v", err)
	}

	ctx := context.Background()
	token, err := auth.RefreshAccessToken(ctx, "refresh-token-123")
	if err == nil {
		t.Error("expected error for network failure")
	}
	if token != nil {
		t.Error("expected nil token on network error")
	}
}

// TestOAuth2Authenticator_GetUserInfo_CacheHit tests caching behavior
func TestOAuth2Authenticator_GetUserInfo_CacheHit(t *testing.T) {
	config := OAuth2Config{
		ClientID:         "client-123",
		ClientSecret:     "secret",
		UserInfoURL:      "https://api.example.com/userinfo",
		TokenCacheEnabled: true,
		TokenCacheTTL:     1 * time.Hour,
	}

	auth, err := NewOAuth2Authenticator(config)
	if err != nil {
		t.Fatalf("failed to create authenticator: %v", err)
	}

	// Manually insert cached user info
	userInfo := &OAuth2UserInfo{
		ID:       "user-123",
		Email:    "user@example.com",
		Username: "testuser",
	}
	auth.tokenStore.SetUserInfo("token-abc", userInfo)

	// Retrieve from cache (should not make HTTP request)
	ctx := context.Background()
	retrieved, err := auth.GetUserInfo(ctx, "token-abc")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if retrieved == nil {
		t.Error("expected user info from cache")
	}
	if retrieved.ID != "user-123" {
		t.Errorf("got ID %q, want user-123", retrieved.ID)
	}
}

// TestOAuth2Authenticator_ValidateToken tests token validation
func TestOAuth2Authenticator_ValidateToken(t *testing.T) {
	config := OAuth2Config{
		ClientID:      "client-123",
		ClientSecret:  "secret",
		UserInfoURL:   "https://api.example.com/userinfo",
	}

	auth, err := NewOAuth2Authenticator(config)
	if err != nil {
		t.Fatalf("failed to create authenticator: %v", err)
	}

	// ValidateToken should fail without cache and network request
	ctx := context.Background()
	valid, err := auth.ValidateToken(ctx, "token-xyz")
	if err == nil {
		t.Error("expected error when validating with unreachable server")
	}
	if valid {
		t.Error("expected token validation to fail")
	}
}

// TestOAuth2Authenticator_RevokeToken tests token revocation
func TestOAuth2Authenticator_RevokeToken(t *testing.T) {
	config := OAuth2Config{
		ClientID:         "client-123",
		ClientSecret:     "secret",
		TokenCacheEnabled: true,
		TokenCacheTTL:     1 * time.Hour,
	}

	auth, err := NewOAuth2Authenticator(config)
	if err != nil {
		t.Fatalf("failed to create authenticator: %v", err)
	}

	// Store a token
	token := &OAuth2Token{
		AccessToken: "token-abc",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	}
	auth.tokenStore.SetToken("token-abc", token)

	// Revoke it
	ctx := context.Background()
	err = auth.RevokeToken(ctx, "token-abc")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify it's removed
	retrieved := auth.tokenStore.GetToken("token-abc")
	if retrieved != nil {
		t.Error("expected token to be removed after revocation")
	}
}

// TestStateStore_Cleanup tests state cleanup on expiry
func TestStateStore_Cleanup(t *testing.T) {
	store := NewStateStore(100 * time.Millisecond)

	// Add an expired state
	state := &StateData{
		State:     "expired-state",
		ExpiresAt: time.Now().Add(-1 * time.Second),
	}
	store.Set("expired-state", state)

	// Trigger cleanup
	store.cleanup()

	// Verify it's cleaned up
	retrieved := store.Get("expired-state")
	if retrieved != nil {
		t.Error("expected expired state to be cleaned up")
	}
}

// TestOAuth2TokenStore_Cleanup tests token store cleanup
func TestOAuth2TokenStore_Cleanup(t *testing.T) {
	store := NewOAuth2TokenStore(100 * time.Millisecond)

	// Add an expired token
	token := &OAuth2Token{
		AccessToken: "expired-token",
		ExpiresAt:   time.Now().Add(-1 * time.Second),
	}
	store.SetToken("expired-token", token)

	// Trigger cleanup
	store.cleanup()

	// Verify it's removed
	retrieved := store.GetToken("expired-token")
	if retrieved != nil {
		t.Error("expected expired token to be cleaned up")
	}
}

// TestOAuth2Metrics_RecordOperations tests metrics recording
func TestOAuth2Metrics_RecordOperations(t *testing.T) {
	metrics := &OAuth2Metrics{}

	metrics.recordAuthorizationRequest()
	if metrics.AuthorizationRequests != 1 {
		t.Errorf("got %d auth requests, want 1", metrics.AuthorizationRequests)
	}

	metrics.recordTokenExchange()
	if metrics.TokenExchanges != 1 {
		t.Errorf("got %d token exchanges, want 1", metrics.TokenExchanges)
	}

	metrics.recordTokenRefresh()
	if metrics.TokenRefreshes != 1 {
		t.Errorf("got %d token refreshes, want 1", metrics.TokenRefreshes)
	}

	metrics.recordUserInfoRequest()
	if metrics.UserInfoRequests != 1 {
		t.Errorf("got %d user info requests, want 1", metrics.UserInfoRequests)
	}

	metrics.recordSuccess()
	if metrics.SuccessfulAuths != 1 {
		t.Errorf("got %d successful auths, want 1", metrics.SuccessfulAuths)
	}

	metrics.recordFailure()
	if metrics.FailedAuths != 1 {
		t.Errorf("got %d failed auths, want 1", metrics.FailedAuths)
	}

	metrics.recordCacheHit()
	if metrics.CacheHits != 1 {
		t.Errorf("got %d cache hits, want 1", metrics.CacheHits)
	}

	metrics.recordCacheMiss()
	if metrics.CacheMisses != 1 {
		t.Errorf("got %d cache misses, want 1", metrics.CacheMisses)
	}
}

// TestOAuth2Authenticator_mapUserInfo tests user info field mapping
func TestOAuth2Authenticator_mapUserInfo(t *testing.T) {
	config := OAuth2Config{
		ClientID:     "client",
		ClientSecret: "secret",
		UserInfoMapping: UserInfoMapping{
			UserID:   "sub",
			Username: "name",
			Email:    "email",
			Picture:  "picture_url",
			Roles:    "roles",
			Groups:   "groups",
		},
	}

	auth, err := NewOAuth2Authenticator(config)
	if err != nil {
		t.Fatalf("failed to create authenticator: %v", err)
	}

	rawUserInfo := map[string]interface{}{
		"sub":          "user-123",
		"name":         "John Doe",
		"email":        "john@example.com",
		"picture_url":  "https://example.com/pic.jpg",
		"roles":        []interface{}{"admin", "user"},
		"groups":       []interface{}{"engineering", "leadership"},
	}

	userInfo := auth.mapUserInfo(rawUserInfo)

	if userInfo.ID != "user-123" {
		t.Errorf("got ID %q, want user-123", userInfo.ID)
	}
	if userInfo.Username != "John Doe" {
		t.Errorf("got username %q, want John Doe", userInfo.Username)
	}
	if userInfo.Email != "john@example.com" {
		t.Errorf("got email %q, want john@example.com", userInfo.Email)
	}
	if userInfo.Picture != "https://example.com/pic.jpg" {
		t.Errorf("got picture %q, want https://example.com/pic.jpg", userInfo.Picture)
	}
	if len(userInfo.Roles) != 2 {
		t.Errorf("got %d roles, want 2", len(userInfo.Roles))
	}
	if len(userInfo.Groups) != 2 {
		t.Errorf("got %d groups, want 2", len(userInfo.Groups))
	}
}

// TestOAuth2Authenticator_ExchangeAuthorizationCode_WithMockServer tests full exchange flow
func TestOAuth2Authenticator_ExchangeAuthorizationCode_WithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, `{"access_token":"token-abc","expires_in":3600,"token_type":"Bearer"}`)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := OAuth2Config{
		ClientID:      "client-123",
		ClientSecret:  "secret",
		TokenURL:      server.URL + "/token",
		RedirectURL:   "https://app.example.com/callback",
	}

	auth, err := NewOAuth2Authenticator(config)
	if err != nil {
		t.Fatalf("failed to create authenticator: %v", err)
	}

	// First, get authorization URL with specific state to store it
	testState := "test-state-12345"
	authURL, err := auth.GetAuthorizationURL("https://app.example.com/callback", testState, nil)
	if err != nil {
		t.Fatalf("failed to get auth URL: %v", err)
	}

	// Verify state is in URL
	if !strings.Contains(authURL, "state="+testState) {
		t.Fatalf("state not found in auth URL: %s", authURL)
	}

	// Now exchange code with valid state
	ctx := context.Background()
	token, err := auth.ExchangeAuthorizationCode(ctx, "auth-code-123", testState)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if token == nil {
		t.Error("expected non-nil token")
	}
	if token != nil && token.AccessToken != "token-abc" {
		t.Errorf("got token %q, want token-abc", token.AccessToken)
	}
}

// TestOAuth2Authenticator_CustomParameters tests custom parameter passing
func TestOAuth2Authenticator_CustomParameters(t *testing.T) {
	config := OAuth2Config{
		ClientID:         "client-123",
		ClientSecret:     "secret",
		AuthorizationURL: "https://auth.example.com/authorize",
		RedirectURL:      "https://app.example.com/callback",
		CustomParameters: map[string]string{
			"prompt":    "login",
			"ui_locales": "en-US",
		},
	}

	auth, err := NewOAuth2Authenticator(config)
	if err != nil {
		t.Fatalf("failed to create authenticator: %v", err)
	}

	url, err := auth.GetAuthorizationURL("https://app.example.com/callback", "", nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !strings.Contains(url, "prompt=login") {
		t.Error("URL should contain custom parameter prompt=login")
	}
	if !strings.Contains(url, "ui_locales=en-US") {
		t.Error("URL should contain custom parameter ui_locales=en-US")
	}
}

// TestOAuth2Authenticator_StateExpiry tests state store expiration
func TestOAuth2Authenticator_StateExpiry(t *testing.T) {
	config := OAuth2Config{
		ClientID:     "client-123",
		ClientSecret: "secret",
		StateExpiry:  100 * time.Millisecond,
	}

	auth, err := NewOAuth2Authenticator(config)
	if err != nil {
		t.Fatalf("failed to create authenticator: %v", err)
	}

	// Get auth URL with explicit state
	testState := "expiring-state-123"
	_, err = auth.GetAuthorizationURL("https://app.example.com/callback", testState, nil)
	if err != nil {
		t.Fatalf("failed to get auth URL: %v", err)
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Try to exchange with expired state
	ctx := context.Background()
	token, err := auth.ExchangeAuthorizationCode(ctx, "code", testState)
	if err == nil {
		t.Error("expected error for expired state")
	}
	if token != nil {
		t.Error("expected nil token for expired state")
	}
}
