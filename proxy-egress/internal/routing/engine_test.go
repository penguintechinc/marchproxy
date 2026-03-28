package routing_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"marchproxy-egress/internal/routing"
)

func TestNewRoutingEngineNotNil(t *testing.T) {
	engine := routing.NewRoutingEngine(nil)
	if engine == nil {
		t.Fatal("NewRoutingEngine(nil) should return non-nil engine")
	}
}

func TestNewRoutingEngineWithConfig(t *testing.T) {
	cfg := &routing.RoutingConfig{
		EnableTrie:         true,
		CaseSensitivePaths: false,
		MaxTrieDepth:       10,
		DefaultTimeout:     30 * time.Second,
		EnableStats:        false,
		MaxRoutes:          500,
		EnableCaching:      false,
	}
	engine := routing.NewRoutingEngine(cfg)
	if engine == nil {
		t.Fatal("NewRoutingEngine with config should return non-nil engine")
	}
}

func TestRouteStructFields(t *testing.T) {
	route := &routing.Route{
		ID:       "route-1",
		Priority: 100,
		Name:     "test-route",
		Enabled:  true,
	}

	if route.ID != "route-1" {
		t.Errorf("expected ID = %q, got %q", "route-1", route.ID)
	}
	if route.Priority != 100 {
		t.Errorf("expected Priority = 100, got %d", route.Priority)
	}
	if route.Name != "test-route" {
		t.Errorf("expected Name = %q, got %q", "test-route", route.Name)
	}
	if !route.Enabled {
		t.Error("expected Enabled = true")
	}
}

func TestPathMatchTypeConstants(t *testing.T) {
	// Verify constants are distinct
	types := []routing.PathMatchType{
		routing.PathExact,
		routing.PathPrefix,
		routing.PathRegex,
		routing.PathWildcard,
	}
	seen := make(map[routing.PathMatchType]bool)
	for _, pt := range types {
		if seen[pt] {
			t.Errorf("duplicate PathMatchType: %d", pt)
		}
		seen[pt] = true
	}
}

func TestHeaderMatchTypeConstants(t *testing.T) {
	types := []routing.HeaderMatchType{
		routing.HeaderExact,
		routing.HeaderRegex,
		routing.HeaderPresent,
		routing.HeaderAbsent,
		routing.HeaderContains,
	}
	seen := make(map[routing.HeaderMatchType]bool)
	for _, ht := range types {
		if seen[ht] {
			t.Errorf("duplicate HeaderMatchType: %d", ht)
		}
		seen[ht] = true
	}
}

func TestGetRoutesEmpty(t *testing.T) {
	engine := routing.NewRoutingEngine(nil)
	routes := engine.GetRoutes()
	if routes == nil {
		t.Fatal("GetRoutes should return non-nil slice")
	}
	if len(routes) != 0 {
		t.Errorf("expected 0 routes, got %d", len(routes))
	}
}

func TestGetStatsInitial(t *testing.T) {
	cfg := &routing.RoutingConfig{
		EnableStats: false,
		MaxRoutes:   100,
	}
	engine := routing.NewRoutingEngine(cfg)
	stats := engine.GetStats()
	if stats == nil {
		t.Fatal("GetStats should return non-nil stats")
	}
	if stats.TotalRequests != 0 {
		t.Errorf("expected 0 total requests initially, got %d", stats.TotalRequests)
	}
}

func TestAddRouteEmptyID(t *testing.T) {
	engine := routing.NewRoutingEngine(nil)
	route := &routing.Route{
		ID:      "",
		Name:    "test",
		Enabled: true,
		Conditions: []routing.Condition{
			&routing.MethodCondition{Methods: []string{"GET"}},
		},
	}
	err := engine.AddRoute(route)
	if err == nil {
		t.Error("expected error when adding route with empty ID")
	}
}

func TestAddRouteEmptyName(t *testing.T) {
	engine := routing.NewRoutingEngine(nil)
	route := &routing.Route{
		ID:      "r1",
		Name:    "",
		Enabled: true,
		Conditions: []routing.Condition{
			&routing.MethodCondition{Methods: []string{"GET"}},
		},
	}
	err := engine.AddRoute(route)
	if err == nil {
		t.Error("expected error when adding route with empty name")
	}
}

func TestAddRouteNoConditions(t *testing.T) {
	engine := routing.NewRoutingEngine(nil)
	route := &routing.Route{
		ID:         "r1",
		Name:       "test",
		Enabled:    true,
		Conditions: []routing.Condition{},
	}
	err := engine.AddRoute(route)
	if err == nil {
		t.Error("expected error when adding route with no conditions")
	}
}

func TestRemoveNonExistentRoute(t *testing.T) {
	engine := routing.NewRoutingEngine(nil)
	err := engine.RemoveRoute("does-not-exist")
	if err == nil {
		t.Error("expected error when removing non-existent route")
	}
}

func TestMethodConditionMatch(t *testing.T) {
	cond := &routing.MethodCondition{Methods: []string{"GET", "POST"}}

	tests := []struct {
		method string
		want   bool
	}{
		{"GET", true},
		{"POST", true},
		{"DELETE", false},
		{"PUT", false},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, "/", nil)
		ctx := &routing.RoutingContext{
			Variables: make(map[string]string),
			Metadata:  make(map[string]interface{}),
		}
		got := cond.Match(req, ctx)
		if got != tt.want {
			t.Errorf("MethodCondition.Match(%q) = %v, want %v", tt.method, got, tt.want)
		}
	}
}

func TestMethodConditionString(t *testing.T) {
	cond := &routing.MethodCondition{Methods: []string{"GET"}}
	s := cond.String()
	if s == "" {
		t.Error("MethodCondition.String() should not be empty")
	}
}

func TestPathConditionExactMatch(t *testing.T) {
	cond := &routing.PathCondition{
		Pattern:       "/api/v1/health",
		Type:          routing.PathExact,
		CaseSensitive: false,
	}

	tests := []struct {
		path string
		want bool
	}{
		{"/api/v1/health", true},
		{"/API/V1/HEALTH", true}, // case insensitive
		{"/api/v1/healthz", false},
		{"/api/v1", false},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, tt.path, nil)
		ctx := &routing.RoutingContext{
			Variables: make(map[string]string),
			Metadata:  make(map[string]interface{}),
		}
		got := cond.Match(req, ctx)
		if got != tt.want {
			t.Errorf("PathCondition.Match(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestPathConditionPrefixMatch(t *testing.T) {
	cond := &routing.PathCondition{
		Pattern:       "/api",
		Type:          routing.PathPrefix,
		CaseSensitive: false,
	}

	tests := []struct {
		path string
		want bool
	}{
		{"/api/v1", true},
		{"/api", true},
		{"/other", false},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, tt.path, nil)
		ctx := &routing.RoutingContext{
			Variables: make(map[string]string),
			Metadata:  make(map[string]interface{}),
		}
		got := cond.Match(req, ctx)
		if got != tt.want {
			t.Errorf("PathCondition prefix Match(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestHeaderConditionExactMatch(t *testing.T) {
	cond := &routing.HeaderCondition{
		Name:  "X-Custom-Header",
		Value: "expected-value",
		Type:  routing.HeaderExact,
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Custom-Header", "expected-value")
	ctx := &routing.RoutingContext{
		Variables: make(map[string]string),
		Metadata:  make(map[string]interface{}),
	}

	if !cond.Match(req, ctx) {
		t.Error("HeaderCondition should match exact header value")
	}

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("X-Custom-Header", "wrong-value")
	if cond.Match(req2, ctx) {
		t.Error("HeaderCondition should not match wrong header value")
	}
}

func TestHeaderConditionPresent(t *testing.T) {
	cond := &routing.HeaderCondition{
		Name: "X-Auth-Token",
		Type: routing.HeaderPresent,
	}

	ctx := &routing.RoutingContext{
		Variables: make(map[string]string),
		Metadata:  make(map[string]interface{}),
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Auth-Token", "anything")
	if !cond.Match(req, ctx) {
		t.Error("HeaderPresent should match when header exists")
	}

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	if cond.Match(req2, ctx) {
		t.Error("HeaderPresent should not match when header is absent")
	}
}

func TestHostConditionExactMatch(t *testing.T) {
	cond := &routing.HostCondition{
		Hosts: []string{"example.com", "api.example.com"},
		Type:  routing.HostExact,
	}

	ctx := &routing.RoutingContext{
		Variables: make(map[string]string),
		Metadata:  make(map[string]interface{}),
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "example.com"
	if !cond.Match(req, ctx) {
		t.Error("HostCondition should match exact host")
	}

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Host = "other.com"
	if cond.Match(req2, ctx) {
		t.Error("HostCondition should not match different host")
	}
}

func TestRoutingContextFields(t *testing.T) {
	ctx := &routing.RoutingContext{
		Variables: map[string]string{"key": "value"},
		Metadata:  map[string]interface{}{"meta": 42},
		Retries:   2,
	}
	if ctx.Variables["key"] != "value" {
		t.Errorf("expected Variables[key] = %q", "value")
	}
	if ctx.Metadata["meta"] != 42 {
		t.Errorf("expected Metadata[meta] = 42")
	}
	if ctx.Retries != 2 {
		t.Errorf("expected Retries = 2, got %d", ctx.Retries)
	}
}

func TestNewPathTrie(t *testing.T) {
	trie := routing.NewPathTrie()
	if trie == nil {
		t.Fatal("NewPathTrie should return non-nil")
	}
}

func TestPathTrieAddAndMatch(t *testing.T) {
	trie := routing.NewPathTrie()

	route := &routing.Route{
		ID:      "r1",
		Name:    "test",
		Enabled: true,
	}
	trie.Add("/api/v1/users", route)

	matched := trie.Match("/api/v1/users")
	if matched == nil {
		t.Fatal("trie.Match should find added route")
	}
	if matched.ID != "r1" {
		t.Errorf("expected matched route ID = %q, got %q", "r1", matched.ID)
	}
}

func TestPathTrieNoMatch(t *testing.T) {
	trie := routing.NewPathTrie()
	matched := trie.Match("/nonexistent")
	if matched != nil {
		t.Error("trie.Match should return nil for unregistered path")
	}
}

func TestRedirectActionExecute(t *testing.T) {
	action := &routing.RedirectAction{
		URL:        "https://example.com/new",
		StatusCode: 301,
		Permanent:  true,
	}
	req := httptest.NewRequest(http.MethodGet, "/old", nil)
	ctx := &routing.RoutingContext{
		Variables: make(map[string]string),
		Metadata:  make(map[string]interface{}),
	}
	err := action.Execute(req, ctx)
	if err != nil {
		t.Errorf("RedirectAction.Execute returned error: %v", err)
	}
	if ctx.Metadata["redirect_url"] != "https://example.com/new" {
		t.Errorf("expected redirect_url in metadata, got %v", ctx.Metadata["redirect_url"])
	}
}

func TestHeaderActionSetExecute(t *testing.T) {
	action := &routing.HeaderAction{
		Operation: routing.HeaderSet,
		Name:      "X-Custom",
		Value:     "test-value",
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := &routing.RoutingContext{
		Variables: make(map[string]string),
		Metadata:  make(map[string]interface{}),
	}
	err := action.Execute(req, ctx)
	if err != nil {
		t.Errorf("HeaderAction.Execute returned error: %v", err)
	}
	if got := req.Header.Get("X-Custom"); got != "test-value" {
		t.Errorf("expected header X-Custom = %q, got %q", "test-value", got)
	}
}

func TestHeaderActionRemoveExecute(t *testing.T) {
	action := &routing.HeaderAction{
		Operation: routing.HeaderRemove,
		Name:      "X-Remove-Me",
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Remove-Me", "some-value")
	ctx := &routing.RoutingContext{
		Variables: make(map[string]string),
		Metadata:  make(map[string]interface{}),
	}
	if err := action.Execute(req, ctx); err != nil {
		t.Errorf("HeaderAction remove returned error: %v", err)
	}
	if got := req.Header.Get("X-Remove-Me"); got != "" {
		t.Errorf("expected header to be removed, got %q", got)
	}
}
