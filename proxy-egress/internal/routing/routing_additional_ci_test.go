//go:build ci
// +build ci

package routing

import (
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"marchproxy-egress/internal/manager"
)

// TestPathCondition_ExactMatch tests exact path matching
func TestPathCondition_ExactMatch(t *testing.T) {
	tests := []struct {
		name           string
		pattern        string
		requestPath    string
		caseSensitive  bool
		shouldMatch    bool
	}{
		{
			name:          "exact match",
			pattern:       "/api/users",
			requestPath:   "/api/users",
			caseSensitive: true,
			shouldMatch:   true,
		},
		{
			name:          "case sensitive mismatch",
			pattern:       "/api/users",
			requestPath:   "/api/Users",
			caseSensitive: true,
			shouldMatch:   false,
		},
		{
			name:          "case insensitive match",
			pattern:       "/api/users",
			requestPath:   "/api/Users",
			caseSensitive: false,
			shouldMatch:   true,
		},
		{
			name:          "partial path no match",
			pattern:       "/api/users",
			requestPath:   "/api/users/123",
			caseSensitive: true,
			shouldMatch:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := &PathCondition{
				Pattern:       tt.pattern,
				Type:          PathExact,
				CaseSensitive: tt.caseSensitive,
			}

			req := httptest.NewRequest("GET", tt.requestPath, nil)
			ctx := &RoutingContext{}

			result := pc.Match(req, ctx)
			if result != tt.shouldMatch {
				t.Errorf("got %v, want %v", result, tt.shouldMatch)
			}
		})
	}
}

// TestPathCondition_PrefixMatch tests prefix path matching
func TestPathCondition_PrefixMatch(t *testing.T) {
	tests := []struct {
		name          string
		pattern       string
		requestPath   string
		shouldMatch   bool
	}{
		{
			name:        "prefix match",
			pattern:     "/api",
			requestPath: "/api/users",
			shouldMatch: true,
		},
		{
			name:        "exact match also prefix",
			pattern:     "/api/users",
			requestPath: "/api/users",
			shouldMatch: true,
		},
		{
			name:        "no match different prefix",
			pattern:     "/api",
			requestPath: "/web/users",
			shouldMatch: false,
		},
		{
			name:        "root prefix",
			pattern:     "/",
			requestPath: "/any/path",
			shouldMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := &PathCondition{
				Pattern:       tt.pattern,
				Type:          PathPrefix,
				CaseSensitive: true,
			}

			req := httptest.NewRequest("GET", tt.requestPath, nil)
			ctx := &RoutingContext{}

			result := pc.Match(req, ctx)
			if result != tt.shouldMatch {
				t.Errorf("got %v, want %v", result, tt.shouldMatch)
			}
		})
	}
}

// TestPathCondition_RegexMatch tests regex path matching
func TestPathCondition_RegexMatch(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		requestPath string
		shouldMatch bool
	}{
		{
			name:        "simple regex",
			pattern:     "^/api/users/[0-9]+$",
			requestPath: "/api/users/123",
			shouldMatch: true,
		},
		{
			name:        "regex no match",
			pattern:     "^/api/users/[0-9]+$",
			requestPath: "/api/users/abc",
			shouldMatch: false,
		},
		{
			name:        "regex with groups",
			pattern:     "^/api/(users|posts)/[0-9]+$",
			requestPath: "/api/posts/456",
			shouldMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			regex, err := regexp.Compile(tt.pattern)
			if err != nil {
				t.Fatalf("invalid regex: %v", err)
			}

			pc := &PathCondition{
				Pattern: tt.pattern,
				Type:    PathRegex,
				Regex:   regex,
			}

			req := httptest.NewRequest("GET", tt.requestPath, nil)
			ctx := &RoutingContext{}

			result := pc.Match(req, ctx)
			if result != tt.shouldMatch {
				t.Errorf("got %v, want %v", result, tt.shouldMatch)
			}
		})
	}
}

// TestPathCondition_WildcardMatch tests wildcard path matching
func TestPathCondition_WildcardMatch(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		requestPath string
		shouldMatch bool
	}{
		{
			name:        "simple wildcard star",
			pattern:     "/api/*",
			requestPath: "/api/users",
			shouldMatch: true,
		},
		{
			name:        "question mark wildcard",
			pattern:     "/api/users/?",
			requestPath: "/api/users/1",
			shouldMatch: true,
		},
		{
			name:        "wildcard no match",
			pattern:     "/api/*.txt",
			requestPath: "/api/users",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := &PathCondition{
				Pattern:       tt.pattern,
				Type:          PathWildcard,
				CaseSensitive: true,
			}

			req := httptest.NewRequest("GET", tt.requestPath, nil)
			ctx := &RoutingContext{}

			result := pc.Match(req, ctx)
			if result != tt.shouldMatch {
				t.Errorf("got %v, want %v", result, tt.shouldMatch)
			}
		})
	}
}

// TestHeaderCondition_Exact tests exact header matching
func TestHeaderCondition_Exact(t *testing.T) {
	tests := []struct {
		name           string
		headerName     string
		headerValue    string
		conditionName  string
		conditionValue string
		shouldMatch    bool
	}{
		{
			name:           "exact header match",
			headerName:     "X-Api-Key",
			headerValue:    "secret123",
			conditionName:  "X-Api-Key",
			conditionValue: "secret123",
			shouldMatch:    true,
		},
		{
			name:           "header mismatch",
			headerName:     "X-Api-Key",
			headerValue:    "wrong-key",
			conditionName:  "X-Api-Key",
			conditionValue: "secret123",
			shouldMatch:    false,
		},
		{
			name:           "missing header",
			headerName:     "X-Api-Key",
			headerValue:    "",
			conditionName:  "X-Api-Key",
			conditionValue: "secret123",
			shouldMatch:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.headerValue != "" {
				req.Header.Set(tt.headerName, tt.headerValue)
			}

			hc := &HeaderCondition{
				Name:  tt.conditionName,
				Value: tt.conditionValue,
				Type:  HeaderExact,
			}

			ctx := &RoutingContext{}
			result := hc.Match(req, ctx)
			if result != tt.shouldMatch {
				t.Errorf("got %v, want %v", result, tt.shouldMatch)
			}
		})
	}
}

// TestHeaderCondition_Present tests header presence checking
func TestHeaderCondition_Present(t *testing.T) {
	tests := []struct {
		name        string
		headerName  string
		headerValue string
		shouldMatch bool
	}{
		{
			name:        "header present",
			headerName:  "Authorization",
			headerValue: "Bearer token",
			shouldMatch: true,
		},
		{
			name:        "header absent",
			headerName:  "Authorization",
			headerValue: "",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.headerValue != "" {
				req.Header.Set(tt.headerName, tt.headerValue)
			}

			hc := &HeaderCondition{
				Name: tt.headerName,
				Type: HeaderPresent,
			}

			ctx := &RoutingContext{}
			result := hc.Match(req, ctx)
			if result != tt.shouldMatch {
				t.Errorf("got %v, want %v", result, tt.shouldMatch)
			}
		})
	}
}

// TestHeaderCondition_Contains tests header substring matching
func TestHeaderCondition_Contains(t *testing.T) {
	tests := []struct {
		name           string
		headerValue    string
		searchValue    string
		shouldMatch    bool
	}{
		{
			name:        "contains match",
			headerValue: "Bearer token123abc",
			searchValue: "token123",
			shouldMatch: true,
		},
		{
			name:        "contains no match",
			headerValue: "Bearer xyz",
			searchValue: "token",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("Authorization", tt.headerValue)

			hc := &HeaderCondition{
				Name:  "Authorization",
				Value: tt.searchValue,
				Type:  HeaderContains,
			}

			ctx := &RoutingContext{}
			result := hc.Match(req, ctx)
			if result != tt.shouldMatch {
				t.Errorf("got %v, want %v", result, tt.shouldMatch)
			}
		})
	}
}

// TestQueryCondition_Exact tests exact query parameter matching
func TestQueryCondition_Exact(t *testing.T) {
	tests := []struct {
		name           string
		queryParam     string
		queryValue     string
		conditionValue string
		shouldMatch    bool
	}{
		{
			name:           "exact query match",
			queryParam:     "version",
			queryValue:     "2",
			conditionValue: "2",
			shouldMatch:    true,
		},
		{
			name:           "query mismatch",
			queryParam:     "version",
			queryValue:     "1",
			conditionValue: "2",
			shouldMatch:    false,
		},
		{
			name:           "missing query",
			queryParam:     "version",
			queryValue:     "",
			conditionValue: "2",
			shouldMatch:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/"
			if tt.queryValue != "" {
				url += "?" + tt.queryParam + "=" + tt.queryValue
			}

			req := httptest.NewRequest("GET", url, nil)

			qc := &QueryCondition{
				Name:  tt.queryParam,
				Value: tt.conditionValue,
				Type:  QueryExact,
			}

			ctx := &RoutingContext{}
			result := qc.Match(req, ctx)
			if result != tt.shouldMatch {
				t.Errorf("got %v, want %v", result, tt.shouldMatch)
			}
		})
	}
}

// TestQueryCondition_Present tests query parameter presence checking
func TestQueryCondition_Present(t *testing.T) {
	tests := []struct {
		name        string
		queryParam  string
		queryValue  string
		shouldMatch bool
	}{
		{
			name:        "query present",
			queryParam:  "filter",
			queryValue:  "active",
			shouldMatch: true,
		},
		{
			name:        "query absent",
			queryParam:  "filter",
			queryValue:  "",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/"
			if tt.queryValue != "" {
				url += "?" + tt.queryParam + "=" + tt.queryValue
			}

			req := httptest.NewRequest("GET", url, nil)

			qc := &QueryCondition{
				Name: tt.queryParam,
				Type: QueryPresent,
			}

			ctx := &RoutingContext{}
			result := qc.Match(req, ctx)
			if result != tt.shouldMatch {
				t.Errorf("got %v, want %v", result, tt.shouldMatch)
			}
		})
	}
}

// TestMethodCondition tests HTTP method matching
func TestMethodCondition(t *testing.T) {
	tests := []struct {
		name        string
		methods     []string
		requestMethod string
		shouldMatch bool
	}{
		{
			name:           "method match GET",
			methods:        []string{"GET", "POST"},
			requestMethod:  "GET",
			shouldMatch:    true,
		},
		{
			name:           "method match POST",
			methods:        []string{"GET", "POST"},
			requestMethod:  "POST",
			shouldMatch:    true,
		},
		{
			name:           "method no match",
			methods:        []string{"GET", "POST"},
			requestMethod:  "DELETE",
			shouldMatch:    false,
		},
		{
			name:           "single method",
			methods:        []string{"GET"},
			requestMethod:  "GET",
			shouldMatch:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.requestMethod, "/", nil)

			mc := &MethodCondition{
				Methods: tt.methods,
			}

			ctx := &RoutingContext{}
			result := mc.Match(req, ctx)
			if result != tt.shouldMatch {
				t.Errorf("got %v, want %v", result, tt.shouldMatch)
			}
		})
	}
}

// TestHostCondition_Exact tests exact host matching
func TestHostCondition_Exact(t *testing.T) {
	tests := []struct {
		name        string
		hosts       []string
		requestHost string
		shouldMatch bool
	}{
		{
			name:        "exact host match",
			hosts:       []string{"api.example.com", "api2.example.com"},
			requestHost: "api.example.com",
			shouldMatch: true,
		},
		{
			name:        "host mismatch",
			hosts:       []string{"api.example.com"},
			requestHost: "other.example.com",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://"+tt.requestHost+"/", nil)
			req.Host = tt.requestHost

			hc := &HostCondition{
				Hosts: tt.hosts,
				Type:  HostExact,
			}

			ctx := &RoutingContext{}
			result := hc.Match(req, ctx)
			if result != tt.shouldMatch {
				t.Errorf("got %v, want %v", result, tt.shouldMatch)
			}
		})
	}
}

// TestRoundRobinBalancer tests round-robin load balancing
func TestRoundRobinBalancer(t *testing.T) {
	services := []*manager.Service{
		{ID: 1, Name: "service1"},
		{ID: 2, Name: "service2"},
		{ID: 3, Name: "service3"},
	}

	balancer := &RoundRobinBalancer{}

	selected := make([]int, 6)
	for i := 0; i < 6; i++ {
		svc := balancer.SelectService(services, nil, nil)
		if svc != nil {
			selected[i] = svc.ID
		}
	}

	// Should cycle: 1, 2, 3, 1, 2, 3
	expected := []int{1, 2, 3, 1, 2, 3}
	for i, exp := range expected {
		if selected[i] != exp {
			t.Errorf("position %d: got %d, want %d", i, selected[i], exp)
		}
	}
}

// TestRoundRobinBalancer_EmptyServices tests empty service list
func TestRoundRobinBalancer_EmptyServices(t *testing.T) {
	balancer := &RoundRobinBalancer{}
	result := balancer.SelectService([]*manager.Service{}, nil, nil)
	if result != nil {
		t.Error("expected nil for empty services")
	}
}

// TestRewriteAction tests path rewriting
func TestRewriteAction(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		replacement string
		originalPath string
		wantPath    string
	}{
		{
			name:         "simple rewrite",
			pattern:      "^/api/v1/(.*)$",
			replacement:  "/api/v2/$1",
			originalPath: "/api/v1/users",
			wantPath:     "/api/v2/users",
		},
		{
			name:         "prefix rewrite",
			pattern:      "^/old/(.*)$",
			replacement:  "/new/$1",
			originalPath: "/old/resource",
			wantPath:     "/new/resource",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			regex, err := regexp.Compile(tt.pattern)
			if err != nil {
				t.Fatalf("invalid regex: %v", err)
			}

			ra := &RewriteAction{
				Pattern:     tt.pattern,
				Replacement: tt.replacement,
				Regex:       regex,
			}

			req := httptest.NewRequest("GET", tt.originalPath, nil)
			ctx := &RoutingContext{
				Variables: make(map[string]string),
				Metadata:  make(map[string]interface{}),
			}

			err = ra.Execute(req, ctx)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if req.URL.Path != tt.wantPath {
				t.Errorf("got path %q, want %q", req.URL.Path, tt.wantPath)
			}
		})
	}
}

// TestHeaderAction tests header manipulation
func TestHeaderAction(t *testing.T) {
	tests := []struct {
		name          string
		operation     HeaderOperation
		initialValue  string
		newValue      string
		expectValue   string
		expectPresent bool
	}{
		{
			name:          "header set",
			operation:     HeaderSet,
			initialValue:  "original",
			newValue:      "Bearer token123",
			expectValue:   "Bearer token123",
			expectPresent: true,
		},
		{
			name:          "header add",
			operation:     HeaderAdd,
			initialValue:  "original",
			newValue:      "custom-header-value",
			expectValue:   "original",  // Add appends, Get returns first
			expectPresent: true,
		},
		{
			name:          "header replace",
			operation:     HeaderReplace,
			initialValue:  "original",
			newValue:      "replaced-value",
			expectValue:   "replaced-value",
			expectPresent: true,
		},
		{
			name:          "header remove",
			operation:     HeaderRemove,
			initialValue:  "original",
			newValue:      "",
			expectPresent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.initialValue != "" {
				req.Header.Set("X-Custom", tt.initialValue)
			}

			ha := &HeaderAction{
				Operation: tt.operation,
				Name:      "X-Custom",
				Value:     tt.newValue,
			}

			ctx := &RoutingContext{}
			err := ha.Execute(req, ctx)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			val := req.Header.Get("X-Custom")
			if tt.expectPresent {
				if val != tt.expectValue {
					t.Errorf("got value %q, want %q", val, tt.expectValue)
				}
			} else {
				if val != "" {
					t.Errorf("header should be removed, got %q", val)
				}
			}
		})
	}
}

// TestPathTrie_Add_Match tests path trie insertion and matching
func TestPathTrie_Add_Match(t *testing.T) {
	trie := NewPathTrie()
	route := &Route{ID: "r1", Name: "test"}

	trie.Add("/api/users", route)

	// Should match exact path
	matched := trie.Match("/api/users")
	if matched == nil {
		t.Error("expected route to match /api/users")
	}
	if matched.ID != "r1" {
		t.Errorf("got route %q, want r1", matched.ID)
	}

	// Should not match partial
	notMatched := trie.Match("/api/users/123")
	if notMatched != nil {
		t.Error("should not match /api/users/123")
	}
}

// TestPathTrie_Add_Wildcard tests path trie with parameters
func TestPathTrie_Add_Wildcard(t *testing.T) {
	trie := NewPathTrie()
	route := &Route{ID: "r1", Name: "user-detail"}

	trie.Add("/users/{id}", route)

	// Should match parameterized path
	matched := trie.Match("/users/123")
	if matched == nil {
		t.Error("expected route to match /users/123")
	}

	// Should also match other values
	matched2 := trie.Match("/users/abc")
	if matched2 == nil {
		t.Error("expected route to match /users/abc")
	}
}

// TestPathTrie_Remove tests route removal from trie
func TestPathTrie_Remove(t *testing.T) {
	trie := NewPathTrie()
	route := &Route{ID: "r1", Name: "test"}

	trie.Add("/api/test", route)

	// Verify it exists
	matched := trie.Match("/api/test")
	if matched == nil {
		t.Fatal("route should exist initially")
	}

	// Remove it
	trie.Remove("/api/test", route)

	// Should no longer match
	matched = trie.Match("/api/test")
	if matched != nil {
		t.Error("route should be removed")
	}
}

// TestTimeCondition tests time-based routing
func TestTimeCondition(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		condition  TimeCondition
		shouldMatch bool
	}{
		{
			name: "no time constraints",
			condition: TimeCondition{
				StartTime: time.Time{},
				EndTime:   time.Time{},
			},
			shouldMatch: true,
		},
		{
			name: "within time window",
			condition: TimeCondition{
				StartTime: now.Add(-1 * time.Hour),
				EndTime:   now.Add(1 * time.Hour),
			},
			shouldMatch: true,
		},
		{
			name: "outside time window",
			condition: TimeCondition{
				StartTime: now.Add(-3 * time.Hour),
				EndTime:   now.Add(-1 * time.Hour),
			},
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			ctx := &RoutingContext{}

			result := tt.condition.Match(req, ctx)
			if result != tt.shouldMatch {
				t.Errorf("got %v, want %v", result, tt.shouldMatch)
			}
		})
	}
}

// TestRoutingEngine_RouteSelection tests route selection and service routing
func TestRoutingEngine_RouteSelection(t *testing.T) {
	engine := NewRoutingEngine(nil)

	route := &Route{
		ID:       "r1",
		Name:     "test-route",
		Priority: 10,
		Enabled:  true,
		Conditions: []Condition{
			&MethodCondition{Methods: []string{"GET"}},
			&PathCondition{Pattern: "/api/test", Type: PathExact},
		},
		Services: []*manager.Service{
			{ID: 1, Name: "backend-1"},
		},
		LoadBalancer: &RoundRobinBalancer{},
		Statistics:  &RouteStats{},
	}

	err := engine.AddRoute(route)
	if err != nil {
		t.Fatalf("failed to add route: %v", err)
	}

	// Test routing
	req := httptest.NewRequest("GET", "/api/test", nil)
	service, ctx, err := engine.Route(req)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if service == nil {
		t.Error("expected non-nil service")
	}
	if ctx == nil {
		t.Error("expected non-nil context")
	}
	if service != nil && service.ID != 1 {
		t.Errorf("got service %d, want 1", service.ID)
	}
}

// TestRoutingEngine_NoMatchingRoute tests unmatched routing
func TestRoutingEngine_NoMatchingRoute(t *testing.T) {
	engine := NewRoutingEngine(nil)

	route := &Route{
		ID:       "r1",
		Name:     "test-route",
		Priority: 10,
		Enabled:  true,
		Conditions: []Condition{
			&PathCondition{Pattern: "/api/test", Type: PathExact},
		},
		Services: []*manager.Service{
			{ID: 1, Name: "backend-1"},
		},
		LoadBalancer: &RoundRobinBalancer{},
	}

	engine.AddRoute(route)

	// Request different path
	req := httptest.NewRequest("GET", "/other/path", nil)
	service, _, err := engine.Route(req)

	if err == nil {
		t.Error("expected error for unmatched route")
	}
	if service != nil {
		t.Error("expected nil service for unmatched route")
	}
}
