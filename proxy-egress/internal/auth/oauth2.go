package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type OAuth2Authenticator struct {
	config       OAuth2Config
	httpClient   *http.Client
	stateStore   *StateStore
	tokenStore   *OAuth2TokenStore
	metrics      *OAuth2Metrics
	mutex        sync.RWMutex
}

type OAuth2Config struct {
	ClientID             string
	ClientSecret         string
	AuthorizationURL     string
	TokenURL             string
	UserInfoURL          string
	RedirectURL          string
	Scopes               []string
	ResponseType         string
	GrantType            string
	AccessType           string
	ProviderName         string
	UsePKCE              bool
	PKCEChallengeMethod  string
	StateExpiry          time.Duration
	TokenCacheEnabled    bool
	TokenCacheTTL        time.Duration
	AutoRefreshTokens    bool
	RefreshThreshold     time.Duration
	CustomParameters     map[string]string
	UserInfoMapping      UserInfoMapping
}

type UserInfoMapping struct {
	UserID      string
	Username    string
	Email       string
	Name        string
	Picture     string
	Roles       string
	Groups      string
}

type OAuth2Token struct {
	AccessToken      string    `json:"access_token"`
	RefreshToken     string    `json:"refresh_token,omitempty"`
	TokenType        string    `json:"token_type"`
	ExpiresIn        int       `json:"expires_in,omitempty"`
	ExpiresAt        time.Time `json:"expires_at"`
	Scope            string    `json:"scope,omitempty"`
	IDToken          string    `json:"id_token,omitempty"`
	State            string    `json:"state,omitempty"`
	CodeVerifier     string    `json:"-"`
}

type OAuth2UserInfo struct {
	ID          string                 `json:"id"`
	Username    string                 `json:"username"`
	Email       string                 `json:"email"`
	Name        string                 `json:"name"`
	Picture     string                 `json:"picture"`
	Roles       []string               `json:"roles"`
	Groups      []string               `json:"groups"`
	Attributes  map[string]interface{} `json:"attributes"`
}

type StateStore struct {
	states map[string]*StateData
	mutex  sync.RWMutex
	expiry time.Duration
}

type StateData struct {
	State         string
	CodeVerifier  string
	Nonce         string
	RedirectURL   string
	CreatedAt     time.Time
	ExpiresAt     time.Time
	UserData      map[string]string
}

type OAuth2TokenStore struct {
	tokens map[string]*OAuth2Token
	users  map[string]*OAuth2UserInfo
	mutex  sync.RWMutex
	ttl    time.Duration
}

type OAuth2Metrics struct {
	AuthorizationRequests uint64
	TokenExchanges        uint64
	TokenRefreshes        uint64
	UserInfoRequests      uint64
	SuccessfulAuths       uint64
	FailedAuths           uint64
	CacheHits             uint64
	CacheMisses           uint64
	AverageLatency        time.Duration
	mutex                 sync.RWMutex
}

type PKCEChallenge struct {
	CodeVerifier string
	CodeChallenge string
	Method        string
}

type OAuth2Provider interface {
	GetAuthorizationURL(state string, pkce *PKCEChallenge) string
	ExchangeToken(ctx context.Context, code string, pkce *PKCEChallenge) (*OAuth2Token, error)
	RefreshToken(ctx context.Context, refreshToken string) (*OAuth2Token, error)
	GetUserInfo(ctx context.Context, accessToken string) (*OAuth2UserInfo, error)
	RevokeToken(ctx context.Context, token string) error
}

func NewOAuth2Authenticator(config OAuth2Config) (*OAuth2Authenticator, error) {
	if config.ClientID == "" || config.ClientSecret == "" {
		return nil, errors.New("client ID and secret are required")
	}

	if config.StateExpiry == 0 {
		config.StateExpiry = 10 * time.Minute
	}

	if config.TokenCacheTTL == 0 {
		config.TokenCacheTTL = 1 * time.Hour
	}

	if config.RefreshThreshold == 0 {
		config.RefreshThreshold = 5 * time.Minute
	}

	oa := &OAuth2Authenticator{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		stateStore: NewStateStore(config.StateExpiry),
		metrics:    &OAuth2Metrics{},
	}

	if config.TokenCacheEnabled {
		oa.tokenStore = NewOAuth2TokenStore(config.TokenCacheTTL)
	}

	go oa.stateStore.StartCleanup()
	if oa.tokenStore != nil {
		go oa.tokenStore.StartCleanup()
	}

	return oa, nil
}

func (oa *OAuth2Authenticator) GetAuthorizationURL(redirectURL string, state string, userData map[string]string) (string, error) {
	start := time.Now()
	defer oa.updateMetrics(start)

	if state == "" {
		state = oa.generateState()
	}

	stateData := &StateData{
		State:       state,
		RedirectURL: redirectURL,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(oa.config.StateExpiry),
		UserData:    userData,
	}

	params := url.Values{}
	params.Set("client_id", oa.config.ClientID)
	params.Set("redirect_uri", oa.config.RedirectURL)
	params.Set("response_type", oa.config.ResponseType)
	params.Set("state", state)

	if len(oa.config.Scopes) > 0 {
		params.Set("scope", strings.Join(oa.config.Scopes, " "))
	}

	if oa.config.AccessType != "" {
		params.Set("access_type", oa.config.AccessType)
	}

	if oa.config.UsePKCE {
		pkce := oa.generatePKCE()
		stateData.CodeVerifier = pkce.CodeVerifier
		params.Set("code_challenge", pkce.CodeChallenge)
		params.Set("code_challenge_method", pkce.Method)
	}

	for key, value := range oa.config.CustomParameters {
		params.Set(key, value)
	}

	oa.stateStore.Set(state, stateData)
	oa.metrics.recordAuthorizationRequest()

	authURL, _ := url.Parse(oa.config.AuthorizationURL)
	authURL.RawQuery = params.Encode()

	return authURL.String(), nil
}

func (oa *OAuth2Authenticator) ExchangeAuthorizationCode(ctx context.Context, code string, state string) (*OAuth2Token, error) {
	start := time.Now()
	defer oa.updateMetrics(start)

	stateData := oa.stateStore.Get(state)
	if stateData == nil {
		oa.metrics.recordFailure()
		return nil, errors.New("invalid state")
	}

	oa.stateStore.Delete(state)

	params := url.Values{}
	params.Set("grant_type", "authorization_code")
	params.Set("code", code)
	params.Set("redirect_uri", oa.config.RedirectURL)
	params.Set("client_id", oa.config.ClientID)
	params.Set("client_secret", oa.config.ClientSecret)

	if oa.config.UsePKCE && stateData.CodeVerifier != "" {
		params.Set("code_verifier", stateData.CodeVerifier)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", oa.config.TokenURL, strings.NewReader(params.Encode()))
	if err != nil {
		oa.metrics.recordFailure()
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := oa.httpClient.Do(req)
	if err != nil {
		oa.metrics.recordFailure()
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		oa.metrics.recordFailure()
		return nil, fmt.Errorf("token exchange failed: %s", string(body))
	}

	var token OAuth2Token
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		oa.metrics.recordFailure()
		return nil, err
	}

	if token.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	}

	if oa.config.TokenCacheEnabled && token.AccessToken != "" {
		oa.tokenStore.SetToken(token.AccessToken, &token)
	}

	oa.metrics.recordTokenExchange()
	oa.metrics.recordSuccess()

	return &token, nil
}

func (oa *OAuth2Authenticator) RefreshAccessToken(ctx context.Context, refreshToken string) (*OAuth2Token, error) {
	start := time.Now()
	defer oa.updateMetrics(start)

	params := url.Values{}
	params.Set("grant_type", "refresh_token")
	params.Set("refresh_token", refreshToken)
	params.Set("client_id", oa.config.ClientID)
	params.Set("client_secret", oa.config.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", oa.config.TokenURL, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := oa.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token refresh failed: %s", string(body))
	}

	var token OAuth2Token
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}

	if token.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	}

	if oa.config.TokenCacheEnabled && token.AccessToken != "" {
		oa.tokenStore.SetToken(token.AccessToken, &token)
	}

	oa.metrics.recordTokenRefresh()

	return &token, nil
}

func (oa *OAuth2Authenticator) GetUserInfo(ctx context.Context, accessToken string) (*OAuth2UserInfo, error) {
	start := time.Now()
	defer oa.updateMetrics(start)

	if oa.config.TokenCacheEnabled {
		if userInfo := oa.tokenStore.GetUserInfo(accessToken); userInfo != nil {
			oa.metrics.recordCacheHit()
			return userInfo, nil
		}
		oa.metrics.recordCacheMiss()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", oa.config.UserInfoURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := oa.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("user info request failed: %s", string(body))
	}

	var rawUserInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawUserInfo); err != nil {
		return nil, err
	}

	userInfo := oa.mapUserInfo(rawUserInfo)

	if oa.config.TokenCacheEnabled {
		oa.tokenStore.SetUserInfo(accessToken, userInfo)
	}

	oa.metrics.recordUserInfoRequest()

	return userInfo, nil
}

func (oa *OAuth2Authenticator) ValidateToken(ctx context.Context, accessToken string) (bool, error) {
	if oa.config.TokenCacheEnabled {
		if token := oa.tokenStore.GetToken(accessToken); token != nil {
			if time.Now().Before(token.ExpiresAt) {
				return true, nil
			}
			
			if oa.config.AutoRefreshTokens && token.RefreshToken != "" {
				if time.Until(token.ExpiresAt) < oa.config.RefreshThreshold {
					newToken, err := oa.RefreshAccessToken(ctx, token.RefreshToken)
					if err == nil {
						*token = *newToken
						return true, nil
					}
				}
			}
		}
	}

	userInfo, err := oa.GetUserInfo(ctx, accessToken)
	if err != nil {
		return false, err
	}

	return userInfo != nil, nil
}

func (oa *OAuth2Authenticator) RevokeToken(ctx context.Context, token string) error {
	if oa.config.TokenCacheEnabled {
		oa.tokenStore.DeleteToken(token)
	}

	return nil
}

func (oa *OAuth2Authenticator) generateState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func (oa *OAuth2Authenticator) generatePKCE() *PKCEChallenge {
	verifier := make([]byte, 32)
	rand.Read(verifier)
	
	codeVerifier := base64.RawURLEncoding.EncodeToString(verifier)
	
	method := oa.config.PKCEChallengeMethod
	if method == "" {
		method = "S256"
	}
	
	var codeChallenge string
	if method == "S256" {
		codeChallenge = base64.RawURLEncoding.EncodeToString([]byte(codeVerifier))
	} else {
		codeChallenge = codeVerifier
	}
	
	return &PKCEChallenge{
		CodeVerifier:  codeVerifier,
		CodeChallenge: codeChallenge,
		Method:        method,
	}
}

func (oa *OAuth2Authenticator) mapUserInfo(raw map[string]interface{}) *OAuth2UserInfo {
	mapping := oa.config.UserInfoMapping
	userInfo := &OAuth2UserInfo{
		Attributes: raw,
	}

	if mapping.UserID != "" {
		if id, ok := raw[mapping.UserID].(string); ok {
			userInfo.ID = id
		}
	}

	if mapping.Username != "" {
		if username, ok := raw[mapping.Username].(string); ok {
			userInfo.Username = username
		}
	}

	if mapping.Email != "" {
		if email, ok := raw[mapping.Email].(string); ok {
			userInfo.Email = email
		}
	}

	if mapping.Name != "" {
		if name, ok := raw[mapping.Name].(string); ok {
			userInfo.Name = name
		}
	}

	if mapping.Picture != "" {
		if picture, ok := raw[mapping.Picture].(string); ok {
			userInfo.Picture = picture
		}
	}

	if mapping.Roles != "" {
		if roles, ok := raw[mapping.Roles].([]interface{}); ok {
			for _, role := range roles {
				if r, ok := role.(string); ok {
					userInfo.Roles = append(userInfo.Roles, r)
				}
			}
		}
	}

	if mapping.Groups != "" {
		if groups, ok := raw[mapping.Groups].([]interface{}); ok {
			for _, group := range groups {
				if g, ok := group.(string); ok {
					userInfo.Groups = append(userInfo.Groups, g)
				}
			}
		}
	}

	return userInfo
}

func (oa *OAuth2Authenticator) updateMetrics(start time.Time) {
	duration := time.Since(start)
	oa.metrics.mutex.Lock()
	defer oa.metrics.mutex.Unlock()
	
	count := oa.metrics.AuthorizationRequests + oa.metrics.TokenExchanges + 
			oa.metrics.TokenRefreshes + oa.metrics.UserInfoRequests
	
	if count > 0 {
		oa.metrics.AverageLatency = 
			(oa.metrics.AverageLatency*time.Duration(count-1) + duration) / 
			time.Duration(count)
	} else {
		oa.metrics.AverageLatency = duration
	}
}

func NewStateStore(expiry time.Duration) *StateStore {
	return &StateStore{
		states: make(map[string]*StateData),
		expiry: expiry,
	}
}

func (ss *StateStore) Set(state string, data *StateData) {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()
	ss.states[state] = data
}

func (ss *StateStore) Get(state string) *StateData {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()
	
	data, exists := ss.states[state]
	if !exists {
		return nil
	}
	
	if time.Now().After(data.ExpiresAt) {
		return nil
	}
	
	return data
}

func (ss *StateStore) Delete(state string) {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()
	delete(ss.states, state)
}

func (ss *StateStore) StartCleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			ss.cleanup()
		}
	}()
}

func (ss *StateStore) cleanup() {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()
	
	now := time.Now()
	for state, data := range ss.states {
		if now.After(data.ExpiresAt) {
			delete(ss.states, state)
		}
	}
}

func NewOAuth2TokenStore(ttl time.Duration) *OAuth2TokenStore {
	return &OAuth2TokenStore{
		tokens: make(map[string]*OAuth2Token),
		users:  make(map[string]*OAuth2UserInfo),
		ttl:    ttl,
	}
}

func (ts *OAuth2TokenStore) SetToken(accessToken string, token *OAuth2Token) {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()
	ts.tokens[accessToken] = token
}

func (ts *OAuth2TokenStore) GetToken(accessToken string) *OAuth2Token {
	ts.mutex.RLock()
	defer ts.mutex.RUnlock()
	return ts.tokens[accessToken]
}

func (ts *OAuth2TokenStore) DeleteToken(accessToken string) {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()
	delete(ts.tokens, accessToken)
	delete(ts.users, accessToken)
}

func (ts *OAuth2TokenStore) SetUserInfo(accessToken string, userInfo *OAuth2UserInfo) {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()
	ts.users[accessToken] = userInfo
}

func (ts *OAuth2TokenStore) GetUserInfo(accessToken string) *OAuth2UserInfo {
	ts.mutex.RLock()
	defer ts.mutex.RUnlock()
	return ts.users[accessToken]
}

func (ts *OAuth2TokenStore) StartCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for range ticker.C {
			ts.cleanup()
		}
	}()
}

func (ts *OAuth2TokenStore) cleanup() {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()
	
	now := time.Now()
	for token, data := range ts.tokens {
		if now.After(data.ExpiresAt) {
			delete(ts.tokens, token)
			delete(ts.users, token)
		}
	}
}

func (om *OAuth2Metrics) recordAuthorizationRequest() {
	om.mutex.Lock()
	defer om.mutex.Unlock()
	om.AuthorizationRequests++
}

func (om *OAuth2Metrics) recordTokenExchange() {
	om.mutex.Lock()
	defer om.mutex.Unlock()
	om.TokenExchanges++
}

func (om *OAuth2Metrics) recordTokenRefresh() {
	om.mutex.Lock()
	defer om.mutex.Unlock()
	om.TokenRefreshes++
}

func (om *OAuth2Metrics) recordUserInfoRequest() {
	om.mutex.Lock()
	defer om.mutex.Unlock()
	om.UserInfoRequests++
}

func (om *OAuth2Metrics) recordSuccess() {
	om.mutex.Lock()
	defer om.mutex.Unlock()
	om.SuccessfulAuths++
}

func (om *OAuth2Metrics) recordFailure() {
	om.mutex.Lock()
	defer om.mutex.Unlock()
	om.FailedAuths++
}

func (om *OAuth2Metrics) recordCacheHit() {
	om.mutex.Lock()
	defer om.mutex.Unlock()
	om.CacheHits++
}

func (om *OAuth2Metrics) recordCacheMiss() {
	om.mutex.Lock()
	defer om.mutex.Unlock()
	om.CacheMisses++
}