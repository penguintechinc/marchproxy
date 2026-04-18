//go:build ci

package routing

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"marchproxy-egress/internal/manager"
)

// TestRoundRobinBalancerComprehensive tests round-robin load balancing
func TestRoundRobinBalancerComprehensive(t *testing.T) {
	services := []*manager.Service{
		{Name: "service-1", Host: "host1", Port: 8001},
		{Name: "service-2", Host: "host2", Port: 8002},
		{Name: "service-3", Host: "host3", Port: 8003},
	}

	balancer := &RoundRobinBalancer{current: 0}
	req := httptest.NewRequest("GET", "http://localhost/test", nil)
	ctx := &RoutingContext{}

	// Verify round-robin sequence
	results := make([]int, 9)
	for i := 0; i < 9; i++ {
		selected := balancer.SelectService(services, req, ctx)
		if selected == nil {
			t.Fatal("expected selected service")
		}
		// Find index
		for j, s := range services {
			if s.Name == selected.Name {
				results[i] = j
				break
			}
		}
	}

	// Verify cycling pattern: 0, 1, 2, 0, 1, 2, 0, 1, 2
	expected := []int{0, 1, 2, 0, 1, 2, 0, 1, 2}
	for i, v := range results {
		if v != expected[i] {
			t.Errorf("index %d: expected %d, got %d", i, expected[i], v)
		}
	}
}

// TestRoundRobinBalancerSingleService tests round-robin with single service
func TestRoundRobinBalancerSingleService(t *testing.T) {
	services := []*manager.Service{
		{Name: "service-1", Host: "host1", Port: 8001},
	}

	balancer := &RoundRobinBalancer{}
	req := httptest.NewRequest("GET", "http://localhost/test", nil)
	ctx := &RoutingContext{}

	for i := 0; i < 5; i++ {
		selected := balancer.SelectService(services, req, ctx)
		if selected == nil {
			t.Fatal("expected selected service")
		}
		if selected.Name != "service-1" {
			t.Errorf("expected service-1, got %s", selected.Name)
		}
	}
}

// TestRoundRobinBalancerEmptyServices tests round-robin with no services
func TestRoundRobinBalancerEmptyServices(t *testing.T) {
	balancer := &RoundRobinBalancer{}
	req := httptest.NewRequest("GET", "http://localhost/test", nil)
	ctx := &RoutingContext{}

	selected := balancer.SelectService(nil, req, ctx)
	if selected != nil {
		t.Error("expected nil for empty services")
	}
}

// TestWeightedBalancer tests weighted load balancing
func TestWeightedBalancer(t *testing.T) {
	balancer := &WeightedBalancer{
		weights: map[int]int{
			0: 5, // 50%
			1: 3, // 30%
			2: 2, // 20%
		},
	}

	// Test that weights are structure is accessible
	if balancer.weights[0] != 5 {
		t.Errorf("expected weight 5, got %d", balancer.weights[0])
	}
	if balancer.weights[1] != 3 {
		t.Errorf("expected weight 3, got %d", balancer.weights[1])
	}
}

// TestPathConditionRegex tests regex path matching
func TestPathConditionRegex(t *testing.T) {
	pattern := `^/api/v[0-9]+/.*$`
	regex, err := regexp.Compile(pattern)
	if err != nil {
		t.Fatalf("failed to compile regex: %v", err)
	}

	pc := &PathCondition{
		Pattern: pattern,
		Type:    PathRegex,
		Regex:   regex,
	}

	tests := []struct {
		path      string
		shouldMatch bool
	}{
		{"/api/v1/users", true},
		{"/api/v2/products", true},
		{"/api/users", false},
		{"/web/v1/users", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://localhost"+tt.path, nil)
			ctx := &RoutingContext{}

			result := pc.Match(req, ctx)
			if result != tt.shouldMatch {
				t.Errorf("got %v, want %v", result, tt.shouldMatch)
			}
		})
	}
}

// TestPathConditionWildcard tests wildcard path matching
func TestPathConditionWildcard(t *testing.T) {
	pc := &PathCondition{
		Pattern:       "/api/*/users",
		Type:          PathWildcard,
		CaseSensitive: true,
	}

	tests := []struct {
		path      string
		shouldMatch bool
	}{
		{"/api/v1/users", true},
		{"/api/v2/users", true},
		{"/api/admin/users", true},
		{"/api/users", false},
		{"/api/v1/products", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://localhost"+tt.path, nil)
			ctx := &RoutingContext{}

			result := pc.Match(req, ctx)
			if result != tt.shouldMatch {
				t.Errorf("got %v, want %v", result, tt.shouldMatch)
			}
		})
	}
}

// TestHeaderConditionVariants tests different header matching types
func TestHeaderConditionVariants(t *testing.T) {
	tests := []struct {
		name      string
		condition *HeaderCondition
		headers   map[string]string
		should    bool
	}{
		{
			name: "exact match",
			condition: &HeaderCondition{
				Name:  "X-Custom",
				Value: "exact-value",
				Type:  HeaderExact,
			},
			headers: map[string]string{"X-Custom": "exact-value"},
			should:  true,
		},
		{
			name: "contains match",
			condition: &HeaderCondition{
				Name:  "X-Custom",
				Value: "partial",
				Type:  HeaderContains,
			},
			headers: map[string]string{"X-Custom": "partial-value"},
			should:  true,
		},
		{
			name: "present check",
			condition: &HeaderCondition{
				Name: "X-Custom",
				Type: HeaderPresent,
			},
			headers: map[string]string{"X-Custom": "any"},
			should:  true,
		},
		{
			name: "absent check",
			condition: &HeaderCondition{
				Name: "X-Missing",
				Type: HeaderAbsent,
			},
			headers: map[string]string{"X-Custom": "any"},
			should:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://localhost/", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			ctx := &RoutingContext{}

			result := tt.condition.Match(req, ctx)
			if result != tt.should {
				t.Errorf("got %v, want %v", result, tt.should)
			}
		})
	}
}

// TestQueryConditionVariants tests query parameter matching
func TestQueryConditionVariants(t *testing.T) {
	tests := []struct {
		name       string
		condition  *QueryCondition
		queryStr   string
		shouldMatch bool
	}{
		{
			name: "exact match",
			condition: &QueryCondition{
				Name:  "page",
				Value: "1",
				Type:  QueryExact,
			},
			queryStr:    "page=1",
			shouldMatch: true,
		},
		{
			name: "present check",
			condition: &QueryCondition{
				Name: "filter",
				Type: QueryPresent,
			},
			queryStr:    "filter=active",
			shouldMatch: true,
		},
		{
			name: "absent check",
			condition: &QueryCondition{
				Name: "filter",
				Type: QueryAbsent,
			},
			queryStr:    "page=1",
			shouldMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://localhost/?"+tt.queryStr, nil)
			ctx := &RoutingContext{}

			result := tt.condition.Match(req, ctx)
			if result != tt.shouldMatch {
				t.Errorf("got %v, want %v", result, tt.shouldMatch)
			}
		})
	}
}

// TestMethodConditionComprehensive tests HTTP method matching variants
func TestMethodConditionComprehensive(t *testing.T) {
	mc := &MethodCondition{
		Methods: []string{"GET", "HEAD", "OPTIONS"},
	}

	tests := []struct {
		method      string
		shouldMatch bool
	}{
		{"GET", true},
		{"HEAD", true},
		{"OPTIONS", true},
		{"POST", false},
		{"DELETE", false},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "http://localhost/", nil)
			ctx := &RoutingContext{}

			result := mc.Match(req, ctx)
			if result != tt.shouldMatch {
				t.Errorf("got %v, want %v", result, tt.shouldMatch)
			}
		})
	}
}

// TestHostConditionVariants tests host matching
func TestHostConditionVariants(t *testing.T) {
	tests := []struct {
		name       string
		condition  *HostCondition
		hostHeader string
		shouldMatch bool
	}{
		{
			name: "exact host match",
			condition: &HostCondition{
				Hosts: []string{"example.com", "api.example.com"},
				Type:  HostExact,
			},
			hostHeader:  "api.example.com",
			shouldMatch: true,
		},
		{
			name: "wildcard host match",
			condition: &HostCondition{
				Hosts: []string{"*.example.com"},
				Type:  HostWildcard,
			},
			hostHeader:  "api.example.com",
			shouldMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://localhost/", nil)
			req.Host = tt.hostHeader
			ctx := &RoutingContext{}

			result := tt.condition.Match(req, ctx)
			if result != tt.shouldMatch {
				t.Errorf("got %v, want %v", result, tt.shouldMatch)
			}
		})
	}
}

// TestTimeConditionComprehensive tests time-based routing scenarios
func TestTimeConditionComprehensive(t *testing.T) {
	req := httptest.NewRequest("GET", "http://localhost/", nil)
	ctx := &RoutingContext{}

	// Test 1: Empty condition (no day/time restrictions) should match
	tc := &TimeCondition{
		Days:     []time.Weekday{},
		TimeZone: time.UTC,
	}

	result := tc.Match(req, ctx)
	if !result {
		t.Error("expected time condition with empty constraints to match")
	}

	// Test 2: All days specified should include current day
	allDays := []time.Weekday{
		time.Sunday, time.Monday, time.Tuesday, time.Wednesday,
		time.Thursday, time.Friday, time.Saturday,
	}

	tc2 := &TimeCondition{
		Days:     allDays,
		TimeZone: time.UTC,
	}

	result = tc2.Match(req, ctx)
	if !result {
		t.Error("expected time condition with all days to match")
	}

	// Test 3: Time range condition
	tc3 := &TimeCondition{
		StartTime: time.Date(0, 0, 0, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(0, 0, 0, 23, 59, 59, 0, time.UTC),
		TimeZone:  time.UTC,
	}

	result = tc3.Match(req, ctx)
	// Should match since we're always in 24-hour range
	if !result {
		t.Error("expected 24-hour time range to match")
	}
}

// TestRewriteActionComprehensive tests path rewriting patterns
func TestRewriteActionComprehensive(t *testing.T) {
	pattern := `/api/v1/(.*)`
	regex, err := regexp.Compile(pattern)
	if err != nil {
		t.Fatalf("failed to compile regex: %v", err)
	}

	ra := &RewriteAction{
		Pattern:     pattern,
		Replacement: `/backend/$1`,
		Regex:       regex,
	}

	req := httptest.NewRequest("GET", "http://localhost/api/v1/users", nil)
	ctx := &RoutingContext{
		Variables: make(map[string]string),
		Metadata:  make(map[string]interface{}),
	}

	err = ra.Execute(req, ctx)
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}

	if req.URL.Path != "/backend/users" {
		t.Errorf("expected path /backend/users, got %s", req.URL.Path)
	}
}

// TestHeaderActionComprehensive tests header modification operations
func TestHeaderActionComprehensive(t *testing.T) {
	tests := []struct {
		name      string
		operation HeaderOperation
		headerName string
		value     string
	}{
		{
			name:      "set header",
			operation: HeaderSet,
			headerName: "X-Custom",
			value:     "new-value",
		},
		{
			name:      "add header",
			operation: HeaderAdd,
			headerName: "X-Added",
			value:     "value",
		},
		{
			name:      "remove header",
			operation: HeaderRemove,
			headerName: "X-Remove",
			value:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://localhost/", nil)
			if tt.operation == HeaderRemove {
				req.Header.Set("X-Remove", "value")
			}

			ha := &HeaderAction{
				Operation: tt.operation,
				Name:      tt.headerName,
				Value:     tt.value,
			}

			ctx := &RoutingContext{
				Variables: make(map[string]string),
				Metadata:  make(map[string]interface{}),
			}
			if err := ha.Execute(req, ctx); err != nil {
				t.Errorf("Execute failed: %v", err)
			}
		})
	}
}

// TestPathTrieBasics tests basic path trie operations
func TestPathTrieBasics(t *testing.T) {
	trie := NewPathTrie()

	route1 := &Route{ID: "route-1", Name: "users"}
	route2 := &Route{ID: "route-2", Name: "products"}

	trie.Add("/api/users", route1)
	trie.Add("/api/products", route2)

	matched := trie.Match("/api/users")
	if matched == nil || matched.ID != "route-1" {
		t.Error("expected to match route-1")
	}

	matched = trie.Match("/api/products")
	if matched == nil || matched.ID != "route-2" {
		t.Error("expected to match route-2")
	}

	matched = trie.Match("/api/notfound")
	if matched != nil {
		t.Error("expected no match for non-existent path")
	}
}

// TestPathTrieRemoval tests path trie route removal
func TestPathTrieRemoval(t *testing.T) {
	trie := NewPathTrie()

	route := &Route{ID: "route-1", Name: "test"}
	trie.Add("/api/test", route)

	matched := trie.Match("/api/test")
	if matched == nil {
		t.Fatal("expected route before removal")
	}

	trie.Remove("/api/test", route)

	matched = trie.Match("/api/test")
	if matched != nil {
		t.Error("expected no match after removal")
	}
}

// TestPathTrieParametrized tests parametrized path matching
func TestPathTrieParametrized(t *testing.T) {
	trie := NewPathTrie()

	route := &Route{ID: "route-1", Name: "user-detail"}
	trie.Add("/api/users/{id}", route)

	// Matching with parameter
	matched := trie.Match("/api/users/123")
	if matched == nil {
		t.Error("expected to match parametrized path")
	}

	matched = trie.Match("/api/users/abc")
	if matched == nil {
		t.Error("expected to match with different param value")
	}
}

// TestRoutingEngineAddRoute tests adding routes
func TestRoutingEngineAddRoute(t *testing.T) {
	engine := NewRoutingEngine(nil)

	route := &Route{
		ID:        "route-1",
		Name:      "test-route",
		Priority:  100,
		Enabled:   true,
		Conditions: []Condition{
			&PathCondition{
				Pattern: "/test",
				Type:    PathExact,
			},
		},
		Services: []*manager.Service{
			{Name: "svc1", Host: "localhost", Port: 8001},
		},
		LoadBalancer: &RoundRobinBalancer{},
	}

	err := engine.AddRoute(route)
	if err != nil {
		t.Fatalf("AddRoute failed: %v", err)
	}

	routes := engine.GetRoutes()
	if len(routes) != 1 {
		t.Errorf("expected 1 route, got %d", len(routes))
	}
}

// TestRoutingEngineAddRouteDuplicateID tests duplicate route ID detection
func TestRoutingEngineAddRouteDuplicateID(t *testing.T) {
	engine := NewRoutingEngine(nil)

	route1 := &Route{
		ID:   "duplicate-id",
		Name: "route1",
		Conditions: []Condition{
			&PathCondition{Pattern: "/api", Type: PathPrefix},
		},
	}

	route2 := &Route{
		ID:   "duplicate-id",
		Name: "route2",
		Conditions: []Condition{
			&PathCondition{Pattern: "/web", Type: PathPrefix},
		},
	}

	engine.AddRoute(route1)
	err := engine.AddRoute(route2)

	if err == nil {
		t.Error("expected error for duplicate route ID")
	}
}

// TestRoutingEngineRemoveRoute tests route removal
func TestRoutingEngineRemoveRoute(t *testing.T) {
	engine := NewRoutingEngine(nil)

	route := &Route{
		ID:   "route-1",
		Name: "test",
		Conditions: []Condition{
			&PathCondition{Pattern: "/test", Type: PathExact},
		},
	}

	engine.AddRoute(route)
	err := engine.RemoveRoute("route-1")

	if err != nil {
		t.Errorf("RemoveRoute failed: %v", err)
	}

	routes := engine.GetRoutes()
	if len(routes) != 0 {
		t.Errorf("expected 0 routes after removal, got %d", len(routes))
	}
}

// TestRoutingEngineRoute tests routing a request
func TestRoutingEngineRoute(t *testing.T) {
	engine := NewRoutingEngine(nil)

	svc := &manager.Service{
		Name:   "backend",
		Host:   "localhost",
		Port:   8080,
		Scheme: "http",
	}

	route := &Route{
		ID:       "route-1",
		Name:     "api-route",
		Priority: 100,
		Enabled:  true,
		Conditions: []Condition{
			&PathCondition{
				Pattern: "/api",
				Type:    PathPrefix,
			},
		},
		Services:     []*manager.Service{svc},
		LoadBalancer: &RoundRobinBalancer{},
	}

	engine.AddRoute(route)

	req := httptest.NewRequest("GET", "http://localhost/api/users", nil)
	selectedSvc, ctx, err := engine.Route(req)

	if err != nil {
		t.Errorf("Route failed: %v", err)
	}

	if selectedSvc == nil {
		t.Error("expected selected service")
	} else if selectedSvc.Name != "backend" {
		t.Errorf("expected backend, got %s", selectedSvc.Name)
	}

	if ctx.Service != selectedSvc {
		t.Error("expected context service to match selected service")
	}
}

// TestRoutingEngineNoMatch tests unmatched requests
func TestRoutingEngineNoMatch(t *testing.T) {
	engine := NewRoutingEngine(nil)

	route := &Route{
		ID:   "route-1",
		Name: "api-route",
		Conditions: []Condition{
			&PathCondition{Pattern: "/api", Type: PathPrefix},
		},
	}

	engine.AddRoute(route)

	req := httptest.NewRequest("GET", "http://localhost/web/home", nil)
	_, _, err := engine.Route(req)

	if err == nil {
		t.Error("expected error for unmatched route")
	}
}

// TestRedirectAction tests redirect action
func TestRedirectAction(t *testing.T) {
	rda := &RedirectAction{
		URL:        "https://example.com/new",
		StatusCode: http.StatusMovedPermanently,
		Permanent:  true,
	}

	req := httptest.NewRequest("GET", "http://localhost/old", nil)
	ctx := &RoutingContext{
		Variables: make(map[string]string),
		Metadata:  make(map[string]interface{}),
	}

	err := rda.Execute(req, ctx)
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}

	if ctx.Metadata["redirect_url"] != "https://example.com/new" {
		t.Error("expected redirect URL in context")
	}

	if ctx.Metadata["redirect_status"] != http.StatusMovedPermanently {
		t.Error("expected status code in context")
	}
}

// TestWildcardMatch tests wildcard matching function
func TestWildcardMatch(t *testing.T) {
	tests := []struct {
		pattern string
		text    string
		should  bool
	}{
		{"*.example.com", "api.example.com", true},
		{"*.example.com", "v2.api.example.com", true}, // .* in regex matches multiple levels
		{"/api/*", "/api/users", true},
		{"/api/*", "/api/users/123", true}, // .* matches /users/123
		{"*", "anything", true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.pattern, tt.text), func(t *testing.T) {
			matched, err := wildcardMatch(tt.pattern, tt.text)
			if err != nil {
				t.Errorf("wildcardMatch failed: %v", err)
			}
			if matched != tt.should {
				t.Errorf("got %v, want %v", matched, tt.should)
			}
		})
	}
}

// TestRoutingStats tests statistics collection
func TestRoutingStats(t *testing.T) {
	engine := NewRoutingEngine(nil)

	route := &Route{
		ID:       "route-1",
		Name:     "test",
		Priority: 100,
		Enabled:  true,
		Conditions: []Condition{
			&PathCondition{Pattern: "/test", Type: PathExact},
		},
		Services:     []*manager.Service{{Name: "svc", Host: "localhost", Port: 8080}},
		LoadBalancer: &RoundRobinBalancer{},
	}

	engine.AddRoute(route)

	req := httptest.NewRequest("GET", "http://localhost/test", nil)
	engine.Route(req)

	stats := engine.GetStats()
	if stats.TotalRequests != 1 {
		t.Errorf("expected 1 request, got %d", stats.TotalRequests)
	}

	// Route should have been matched and routed
	if stats.RoutedRequests < 1 {
		t.Errorf("expected at least 1 routed request, got %d", stats.RoutedRequests)
	}
}

// TestMultipleConditions tests routes with multiple conditions
func TestMultipleConditions(t *testing.T) {
	engine := NewRoutingEngine(nil)

	route := &Route{
		ID:       "multi-cond",
		Name:     "complex-route",
		Priority: 100,
		Enabled:  true,
		Conditions: []Condition{
			&PathCondition{Pattern: "/api", Type: PathPrefix},
			&MethodCondition{Methods: []string{"POST", "PUT"}},
			&HeaderCondition{Name: "Content-Type", Value: "application/json", Type: HeaderExact},
		},
		Services:     []*manager.Service{{Name: "api", Host: "localhost", Port: 8080}},
		LoadBalancer: &RoundRobinBalancer{},
	}

	engine.AddRoute(route)

	// Should match
	req := httptest.NewRequest("POST", "http://localhost/api/users", nil)
	req.Header.Set("Content-Type", "application/json")

	selectedSvc, _, err := engine.Route(req)
	if err != nil || selectedSvc == nil {
		t.Error("expected route to match all conditions")
	}

	// Should not match (wrong method)
	req2 := httptest.NewRequest("GET", "http://localhost/api/users", nil)
	req2.Header.Set("Content-Type", "application/json")

	_, _, err2 := engine.Route(req2)
	if err2 == nil {
		t.Error("expected route to not match on method")
	}
}

// TestSetServiceAction tests setting service via action
func TestSetServiceAction(t *testing.T) {
	svc := &manager.Service{
		Name:   "selected-service",
		Host:   "localhost",
		Port:   9000,
		Scheme: "http",
	}

	ssa := &SetServiceAction{
		ServiceID: 42,
		Service:   svc,
	}

	req := httptest.NewRequest("GET", "http://localhost/", nil)
	ctx := &RoutingContext{
		Variables: make(map[string]string),
		Metadata:  make(map[string]interface{}),
	}

	err := ssa.Execute(req, ctx)
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}

	if ctx.Service != svc {
		t.Error("expected service to be set")
	}

	if ctx.Metadata["service_id"] != 42 {
		t.Error("expected service ID in metadata")
	}
}

// TestConditionString tests String() representations
func TestConditionString(t *testing.T) {
	tests := []struct {
		cond   Condition
		hasStr bool
	}{
		{&PathCondition{Pattern: "/api", Type: PathExact}, true},
		{&HeaderCondition{Name: "X-Custom", Value: "test", Type: HeaderExact}, true},
		{&QueryCondition{Name: "page", Value: "1", Type: QueryExact}, true},
		{&MethodCondition{Methods: []string{"GET"}}, true},
		{&HostCondition{Hosts: []string{"example.com"}, Type: HostExact}, true},
		{&TimeCondition{}, true},
	}

	for _, tt := range tests {
		str := tt.cond.String()
		if tt.hasStr && str == "" {
			t.Errorf("expected non-empty string for %T", tt.cond)
		}
	}
}

// TestActionString tests String() representations for actions
func TestActionString(t *testing.T) {
	tests := []struct {
		action Action
		hasStr bool
	}{
		{&RewriteAction{Pattern: "/old", Replacement: "/new"}, true},
		{&RedirectAction{URL: "https://example.com", StatusCode: 301}, true},
		{&HeaderAction{Name: "X-Custom", Value: "test", Operation: HeaderSet}, true},
		{&SetServiceAction{ServiceID: 1}, true},
	}

	for _, tt := range tests {
		str := tt.action.String()
		if tt.hasStr && str == "" {
			t.Errorf("expected non-empty string for %T", tt.action)
		}
	}
}

// TestLoadBalancerString tests load balancer string representation
func TestLoadBalancerString(t *testing.T) {
	balancer := &RoundRobinBalancer{}
	str := balancer.String()
	if str == "" {
		t.Error("expected non-empty string")
	}
}
