//go:build ci

package main

import (
	"net/http"
	"testing"

	"marchproxy-ingress/internal/manager"
)

func TestFindMatchingRouteWithMultipleVirtualHosts(t *testing.T) {
	proxy := &IngressProxy{
		clusterConfig: &manager.ClusterConfig{
			VirtualHosts: []manager.VirtualHost{
				{
					Hostname: "api.example.com",
					Backend:  "api-backend",
				},
				{
					Hostname: "www.example.com",
					Backend:  "web-backend",
				},
				{
					Hostname: "*.internal.local",
					Backend:  "internal-backend",
				},
			},
		},
	}

	tests := []struct {
		host    string
		want    int // index of expected vhost
		wantNil bool
	}{
		{"api.example.com", 0, false},
		{"www.example.com", 1, false},
		{"service.internal.local", 2, false},
		{"unknown.example.com", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "http://"+tt.host+"/", nil)
			req.Host = tt.host

			got := proxy.findMatchingRoute(req)
			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %v", got)
				}
			} else {
				if got == nil {
					t.Errorf("expected non-nil route")
				} else if got.Backend != proxy.clusterConfig.VirtualHosts[tt.want].Backend {
					t.Errorf("got backend %q, want %q", got.Backend, proxy.clusterConfig.VirtualHosts[tt.want].Backend)
				}
			}
		})
	}
}

func TestMatchesHostPatternEdgeCases(t *testing.T) {
	proxy := &IngressProxy{}

	tests := []struct {
		host    string
		pattern string
		want    bool
		name    string
	}{
		// Exact matches
		{"example.com", "example.com", true, "exact domain"},
		{"api.example.com", "api.example.com", true, "exact subdomain"},

		// Wildcard domain
		{"api.example.com", "*.example.com", true, "wildcard subdomain match"},
		{"nested.api.example.com", "*.example.com", true, "wildcard subdomain deep match (ends with .example.com)"},
		{"example.com", "*.example.com", true, "wildcard domain exact"},

		// Catch-all
		{"anything.anything", "*", true, "catch-all matches"},

		// Empty pattern
		{"anything", "", true, "empty pattern matches all"},
		{"anything", "*", true, "asterisk pattern matches all"},

		// Non-matches
		{"other.com", "example.com", false, "different domain"},
		{"fake.example.com", "*.fake.com", false, "reversed domain"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := proxy.matchesHostPattern(tt.host, tt.pattern); got != tt.want {
				t.Errorf("matchesHostPattern(%q, %q) = %v, want %v", tt.host, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestMatchesPathPatternEdgeCases(t *testing.T) {
	proxy := &IngressProxy{}

	tests := []struct {
		path    string
		pattern string
		want    bool
		name    string
	}{
		// Prefix matches
		{"/api/v1/users", "/api/*", true, "prefix match"},
		{"/api/v1/users/123", "/api/*", true, "deep prefix match"},
		{"/api", "/api/*", false, "exact path no match (needs /)"},

		// Exact matches
		{"/api/v1/users", "/api/v1/users", true, "exact path"},
		{"/api/v1/users", "/api/v1", false, "partial path no match"},

		// Wildcard root
		{"/anything", "/*", true, "wildcard root"},
		{"/anything/nested", "/*", true, "nested path wildcard root"},

		// Default patterns
		{"/", "/", true, "root path exact"},
		{"/anything", "/", true, "root path matches all"},
		{"/anything", "", true, "empty pattern catch-all"},

		// Non-matches
		{"/api/v1/users", "/v1/*", false, "different prefix"},
		{"/api/v2/users", "/api/v1/*", false, "deeper prefix no match"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := proxy.matchesPathPattern(tt.path, tt.pattern); got != tt.want {
				t.Errorf("matchesPathPattern(%q, %q) = %v, want %v", tt.path, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestFindMatchingRouteWithPathRules(t *testing.T) {
	proxy := &IngressProxy{
		clusterConfig: &manager.ClusterConfig{
			VirtualHosts: []manager.VirtualHost{
				{
					Hostname: "api.example.com",
					Backend:  "api-backend",
					RoutingRules: []manager.RoutingRule{
						{PathPattern: "/v1/*"},
						{PathPattern: "/v2/*"},
					},
				},
			},
		},
	}

	tests := []struct {
		host string
		path string
		want bool
	}{
		{"api.example.com", "/v1/users", true},
		{"api.example.com", "/v2/products", true},
		{"api.example.com", "/v3/orders", true},
		{"api.example.com", "/unknown", true},
		{"other.com", "/v1/users", false},
	}

	for _, tt := range tests {
		t.Run(tt.host+tt.path, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "http://"+tt.host+tt.path, nil)
			req.Host = tt.host

			got := proxy.findMatchingRoute(req)
			if (got != nil) != tt.want {
				t.Errorf("findMatchingRoute() got match=%v, want=%v", got != nil, tt.want)
			}
		})
	}
}

func TestSelectBackendWithMultipleEndpoints(t *testing.T) {
	proxy := &IngressProxy{
		clusterConfig: &manager.ClusterConfig{
			Backends: []manager.Backend{
				{
					Name: "api-backend",
					Endpoints: []manager.BackendEndpoint{
						{Host: "api-1.internal", Port: 8080},
						{Host: "api-2.internal", Port: 8080},
						{Host: "api-3.internal", Port: 8080},
					},
				},
			},
		},
	}

	vhost := &manager.VirtualHost{
		Backend: "api-backend",
	}

	url, err := proxy.selectBackend(vhost)
	if err != nil {
		t.Errorf("selectBackend() error = %v", err)
	}
	if url == nil {
		t.Error("selectBackend() returned nil URL")
	}
}

func TestSelectBackendEmptyEndpoints(t *testing.T) {
	proxy := &IngressProxy{
		clusterConfig: &manager.ClusterConfig{
			Backends: []manager.Backend{
				{
					Name:      "empty-backend",
					Endpoints: []manager.BackendEndpoint{},
				},
			},
		},
	}

	vhost := &manager.VirtualHost{
		Backend: "empty-backend",
	}

	url, err := proxy.selectBackend(vhost)
	if err == nil {
		t.Error("expected error for empty endpoints")
	}
	if url != nil {
		t.Errorf("expected nil URL, got %v", url)
	}
}

func TestSelectBackendNoBackendConfig(t *testing.T) {
	proxy := &IngressProxy{
		clusterConfig: &manager.ClusterConfig{
			Backends: []manager.Backend{},
		},
	}

	vhost := &manager.VirtualHost{
		Backend: "nonexistent",
	}

	_, err := proxy.selectBackend(vhost)
	if err == nil {
		t.Error("expected error for nonexistent backend")
	}
}
