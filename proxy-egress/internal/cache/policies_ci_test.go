//go:build ci

package cache

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestDefaultPolicyShouldCache tests ShouldCache logic
func TestDefaultPolicyShouldCache(t *testing.T) {
	config := DefaultPolicyConfig{
		DefaultTTL:           5 * time.Minute,
		CacheableStatusCodes: []int{200, 203, 206},
		CacheableMethods:     []string{"GET", "HEAD"},
		RespectCacheControl:  true,
	}
	policy := NewDefaultPolicy(config)

	tests := []struct {
		name           string
		method         string
		statusCode     int
		cacheControl   string
		expectedResult bool
	}{
		{
			name:           "GET 200 should cache",
			method:         "GET",
			statusCode:     200,
			cacheControl:   "",
			expectedResult: true,
		},
		{
			name:           "POST should not cache",
			method:         "POST",
			statusCode:     200,
			cacheControl:   "",
			expectedResult: false,
		},
		{
			name:           "200 should cache",
			method:         "GET",
			statusCode:     200,
			cacheControl:   "",
			expectedResult: true,
		},
		{
			name:           "400 should not cache",
			method:         "GET",
			statusCode:     400,
			cacheControl:   "",
			expectedResult: false,
		},
		{
			name:           "no-cache directive should not cache",
			method:         "GET",
			statusCode:     200,
			cacheControl:   "no-cache",
			expectedResult: false,
		},
		{
			name:           "no-store directive should not cache",
			method:         "GET",
			statusCode:     200,
			cacheControl:   "no-store",
			expectedResult: false,
		},
		{
			name:           "private directive should not cache",
			method:         "GET",
			statusCode:     200,
			cacheControl:   "private",
			expectedResult: false,
		},
		{
			name:           "HEAD 200 should cache",
			method:         "HEAD",
			statusCode:     200,
			cacheControl:   "",
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(tt.method, "http://example.com", nil)
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Header: http.Header{
					"Cache-Control": []string{tt.cacheControl},
				},
			}

			result := policy.ShouldCache(req, resp)
			if result != tt.expectedResult {
				t.Errorf("Expected %v, got %v", tt.expectedResult, result)
			}
		})
	}
}

// TestDefaultPolicyGetTTL tests TTL extraction
func TestDefaultPolicyGetTTL(t *testing.T) {
	config := DefaultPolicyConfig{
		DefaultTTL:          5 * time.Minute,
		MaxTTL:              24 * time.Hour,
		RespectCacheControl: true,
	}
	policy := NewDefaultPolicy(config)

	tests := []struct {
		name           string
		cacheControl   string
		expires        string
		expectedTTL    time.Duration
	}{
		{
			name:           "max-age directive",
			cacheControl:   "max-age=3600",
			expires:        "",
			expectedTTL:    time.Hour,
		},
		{
			name:           "s-maxage directive",
			cacheControl:   "s-maxage=1800",
			expires:        "",
			expectedTTL:    30 * time.Minute,
		},
		{
			name:           "max-age exceeds MaxTTL",
			cacheControl:   "max-age=100000",
			expires:        "",
			expectedTTL:    24 * time.Hour, // Capped at MaxTTL
		},
		{
			name:         "no cache control uses default",
			cacheControl: "",
			expires:      "",
			expectedTTL:  5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "http://example.com", nil)
			resp := &http.Response{
				Header: http.Header{
					"Cache-Control": []string{tt.cacheControl},
					"Expires":       []string{tt.expires},
				},
			}

			ttl := policy.GetTTL(req, resp)
			if ttl != tt.expectedTTL {
				t.Errorf("Expected %v, got %v", tt.expectedTTL, ttl)
			}
		})
	}
}

// TestDefaultPolicyGenerateKey tests cache key generation
func TestDefaultPolicyGenerateKey(t *testing.T) {
	policy := NewDefaultPolicy(DefaultPolicyConfig{
		VaryHeaders: []string{"Accept", "Authorization"},
	})

	req1, _ := http.NewRequest("GET", "http://example.com/api/users", nil)
	req1.Header.Set("Accept", "application/json")

	req2, _ := http.NewRequest("GET", "http://example.com/api/users", nil)
	req2.Header.Set("Accept", "application/xml")

	key1 := policy.GenerateKey(req1)
	key2 := policy.GenerateKey(req2)

	if key1 == key2 {
		t.Error("Different Accept headers should generate different keys")
	}

	// Same request should generate same key
	key1Again := policy.GenerateKey(req1)
	if key1 != key1Again {
		t.Error("Same request should generate same key")
	}
}

// TestDefaultPolicyShouldInvalidate tests invalidation detection
func TestDefaultPolicyShouldInvalidate(t *testing.T) {
	policy := NewDefaultPolicy(DefaultPolicyConfig{})

	tests := []struct {
		method            string
		shouldInvalidate   bool
	}{
		{"GET", false},
		{"HEAD", false},
		{"POST", true},
		{"PUT", true},
		{"PATCH", true},
		{"DELETE", true},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			req, _ := http.NewRequest(tt.method, "http://example.com", nil)
			result := policy.ShouldInvalidate(req)
			if result != tt.shouldInvalidate {
				t.Errorf("Expected %v, got %v", tt.shouldInvalidate, result)
			}
		})
	}
}

// TestDefaultPolicyGetTags tests tag extraction
func TestDefaultPolicyGetTags(t *testing.T) {
	extractors := []TagExtractor{
		UserAgentTagExtractor,
		ContentTypeTagExtractor,
	}

	policy := NewDefaultPolicy(DefaultPolicyConfig{
		TagExtractors: extractors,
	})

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 Chrome")

	resp := &http.Response{
		StatusCode: 200,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
	}

	tags := policy.GetTags(req, resp)

	if len(tags) == 0 {
		t.Error("Expected tags to be generated")
	}

	// Should include content-type tag
	hasContentTypeTag := false
	for _, tag := range tags {
		if strings.HasPrefix(tag, "content-type:") {
			hasContentTypeTag = true
		}
	}

	if !hasContentTypeTag {
		t.Error("Expected content-type tag")
	}
}

// TestConditionalPolicy tests conditional caching
func TestConditionalPolicy(t *testing.T) {
	basePolicy := NewDefaultPolicy(DefaultPolicyConfig{
		DefaultTTL: 5 * time.Minute,
	})

	specialPolicy := NewDefaultPolicy(DefaultPolicyConfig{
		DefaultTTL: 1 * time.Hour,
	})

	// Matcher that returns true for specific paths
	matcher := func(req *http.Request, resp *http.Response) bool {
		return strings.HasPrefix(req.URL.Path, "/api/")
	}

	conditional := NewConditionalPolicy(
		[]CacheCondition{
			{
				Matcher: matcher,
				Policy:  specialPolicy,
			},
		},
		basePolicy,
	)

	// Test API request
	apiReq, _ := http.NewRequest("GET", "http://example.com/api/users", nil)
	resp := &http.Response{
		StatusCode: 200,
		Header:     http.Header{},
	}

	ttl := conditional.GetTTL(apiReq, resp)
	if ttl != 1*time.Hour {
		t.Errorf("Expected 1 hour TTL for API, got %v", ttl)
	}

	// Test non-API request
	pageReq, _ := http.NewRequest("GET", "http://example.com/page", nil)
	ttl = conditional.GetTTL(pageReq, resp)
	if ttl != 5*time.Minute {
		t.Errorf("Expected 5 minute TTL for page, got %v", ttl)
	}
}

// TestHeaderBasedPolicy tests header-based caching
func TestHeaderBasedPolicy(t *testing.T) {
	rules := map[string]HeaderRule{
		"Cache-Control": {
			TTL:         10 * time.Minute,
			ShouldCache: true,
		},
		"X-No-Cache": {
			TTL:         0,
			ShouldCache: false,
		},
	}

	policy := NewHeaderBasedPolicy(rules, 5*time.Minute)

	tests := []struct {
		name           string
		headers        map[string]string
		expectedCache  bool
		expectedTTL    time.Duration
	}{
		{
			name:          "Has Cache-Control header",
			headers:       map[string]string{"Cache-Control": "public"},
			expectedCache: true,
			expectedTTL:   10 * time.Minute,
		},
		{
			name:          "Has X-No-Cache header",
			headers:       map[string]string{"X-No-Cache": "true"},
			expectedCache: false,
			expectedTTL:   0,
		},
		{
			name:          "No matching headers",
			headers:       map[string]string{"Other": "value"},
			expectedCache: true,
			expectedTTL:   5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "http://example.com", nil)
			resp := &http.Response{
				StatusCode: 200,
				Header:     http.Header{},
			}

			for k, v := range tt.headers {
				resp.Header.Set(k, v)
			}

			shouldCache := policy.ShouldCache(req, resp)
			if shouldCache != tt.expectedCache {
				t.Errorf("Expected cache=%v, got %v", tt.expectedCache, shouldCache)
			}

			ttl := policy.GetTTL(req, resp)
			if ttl != tt.expectedTTL {
				t.Errorf("Expected TTL=%v, got %v", tt.expectedTTL, ttl)
			}
		})
	}
}

// TestPathBasedPolicy tests path-based caching
func TestPathBasedPolicy(t *testing.T) {
	rules := map[string]PathRule{
		"/api/users": {
			TTL:         30 * time.Minute,
			ShouldCache: true,
			Methods:     []string{"GET"},
		},
		"/admin": {
			TTL:         0,
			ShouldCache: false,
			Methods:     []string{"GET"},
		},
	}

	patterns := []PathPattern{
		{
			Pattern: "/api/*",
			Rule: PathRule{
				TTL:         15 * time.Minute,
				ShouldCache: true,
				Methods:     []string{"GET"},
			},
		},
	}

	policy := NewPathBasedPolicy(rules, patterns, 5*time.Minute)

	tests := []struct {
		name          string
		path          string
		method        string
		expectedCache bool
		expectedTTL   time.Duration
	}{
		{
			name:          "Exact path match",
			path:          "/api/users",
			method:        "GET",
			expectedCache: true,
			expectedTTL:   30 * time.Minute,
		},
		{
			name:          "Pattern match",
			path:          "/api/products",
			method:        "GET",
			expectedCache: true,
			expectedTTL:   15 * time.Minute,
		},
		{
			name:          "Admin path no cache",
			path:          "/admin",
			method:        "GET",
			expectedCache: false,
			expectedTTL:   5 * time.Minute, // Returns default TTL even when ShouldCache is false
		},
		{
			name:          "Default policy",
			path:          "/other",
			method:        "GET",
			expectedCache: true,
			expectedTTL:   5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(tt.method, "http://example.com"+tt.path, nil)
			resp := &http.Response{
				StatusCode: 200,
				Header:     http.Header{},
			}

			shouldCache := policy.ShouldCache(req, resp)
			if shouldCache != tt.expectedCache {
				t.Errorf("Expected cache=%v, got %v", tt.expectedCache, shouldCache)
			}

			ttl := policy.GetTTL(req, resp)
			if ttl != tt.expectedTTL {
				t.Errorf("Expected TTL=%v, got %v", tt.expectedTTL, ttl)
			}
		})
	}
}

// TestTimeBasedPolicy tests time-based caching
func TestTimeBasedPolicy(t *testing.T) {
	basePolicy := NewDefaultPolicy(DefaultPolicyConfig{
		DefaultTTL: 5 * time.Minute,
	})

	now := time.Now()
	hourAgo := now.Add(-time.Hour)
	hourLater := now.Add(time.Hour)

	schedules := []CacheSchedule{
		{
			StartTime:   hourAgo,
			EndTime:     hourLater,
			DaysOfWeek:  []time.Weekday{}, // All days
			TTL:         30 * time.Minute,
			ShouldCache: true,
		},
	}

	policy := NewTimeBasedPolicy(schedules, basePolicy)

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp := &http.Response{
		StatusCode: 200,
		Header:     http.Header{},
	}

	// Should use schedule TTL since we're within time range
	ttl := policy.GetTTL(req, resp)
	if ttl != 30*time.Minute {
		t.Errorf("Expected schedule TTL 30min, got %v", ttl)
	}
}

// TestUserAgentTagExtractor tests user agent tag extraction
func TestUserAgentTagExtractor(t *testing.T) {
	tests := []struct {
		userAgent    string
		expectedTags map[string]bool
	}{
		{
			userAgent: "Mozilla/5.0 Mobile",
			expectedTags: map[string]bool{
				"device:mobile": true,
			},
		},
		{
			userAgent: "Mozilla/5.0 Tablet",
			expectedTags: map[string]bool{
				"device:tablet": true,
			},
		},
		{
			userAgent: "Mozilla/5.0 Desktop",
			expectedTags: map[string]bool{
				"device:desktop": true,
			},
		},
		{
			userAgent: "Mozilla/5.0 Chrome",
			expectedTags: map[string]bool{
				"browser:chrome":  true,
				"device:desktop":  true,
			},
		},
		{
			userAgent: "Mozilla/5.0 Firefox",
			expectedTags: map[string]bool{
				"browser:firefox": true,
				"device:desktop":  true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.userAgent, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "http://example.com", nil)
			req.Header.Set("User-Agent", tt.userAgent)
			resp := &http.Response{Header: http.Header{}}

			tags := UserAgentTagExtractor(req, resp)

			for _, tag := range tags {
				if !tt.expectedTags[tag] {
					t.Errorf("Unexpected tag: %s", tag)
				}
			}
		})
	}
}

// TestContentTypeTagExtractor tests content type tag extraction
func TestContentTypeTagExtractor(t *testing.T) {
	tests := []struct {
		contentType  string
		expectedTags map[string]bool
	}{
		{
			contentType: "application/json",
			expectedTags: map[string]bool{
				"content-type:application":    true,
				"content-subtype:json":        true,
			},
		},
		{
			contentType: "text/html; charset=utf-8",
			expectedTags: map[string]bool{
				"content-type:text":  true,
				"content-subtype:html": true,
			},
		},
		{
			contentType: "image/png",
			expectedTags: map[string]bool{
				"content-type:image": true,
				"content-subtype:png": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "http://example.com", nil)
			resp := &http.Response{
				Header: http.Header{
					"Content-Type": []string{tt.contentType},
				},
			}

			tags := ContentTypeTagExtractor(req, resp)

			for _, tag := range tags {
				if !tt.expectedTags[tag] {
					t.Errorf("Unexpected tag: %s", tag)
				}
			}
		})
	}
}

// TestAuthenticationTagExtractor tests authentication tag extraction
func TestAuthenticationTagExtractor(t *testing.T) {
	tests := []struct {
		hasAuth      bool
		expectedTag  string
	}{
		{
			hasAuth:     true,
			expectedTag: "authenticated:true",
		},
		{
			hasAuth:     false,
			expectedTag: "authenticated:false",
		},
	}

	for _, tt := range tests {
		req, _ := http.NewRequest("GET", "http://example.com", nil)
		if tt.hasAuth {
			req.Header.Set("Authorization", "Bearer token")
		}
		resp := &http.Response{Header: http.Header{}}

		tags := AuthenticationTagExtractor(req, resp)

		if len(tags) != 1 || tags[0] != tt.expectedTag {
			t.Errorf("Expected %s, got %v", tt.expectedTag, tags)
		}
	}
}

// TestAPIVersionTagExtractor tests API version tag extraction
func TestAPIVersionTagExtractor(t *testing.T) {
	tests := []struct {
		path        string
		header      string
		queryParam  string
		expectedTag string
	}{
		{
			path:        "/v1/users",
			expectedTag: "api-version:v1",
		},
		{
			path:        "/v2/products",
			expectedTag: "api-version:v2",
		},
		{
			header:      "API-Version",
			expectedTag: "api-version:v3",
		},
		{
			queryParam: "version=v4",
			path:       "/users",
			expectedTag: "api-version:v4",
		},
	}

	for _, tt := range tests {
		req, _ := http.NewRequest("GET", "http://example.com"+tt.path, nil)
		if tt.header != "" {
			req.Header.Set("API-Version", strings.TrimPrefix(tt.expectedTag, "api-version:"))
		}
		if tt.queryParam != "" {
			req.URL.RawQuery = tt.queryParam
		}
		resp := &http.Response{Header: http.Header{}}

		tags := APIVersionTagExtractor(req, resp)

		if len(tags) > 0 && tags[0] != tt.expectedTag {
			t.Errorf("Expected %s, got %v", tt.expectedTag, tags)
		}
	}
}

// TestCacheKeyBuilder tests cache key building
func TestCacheKeyBuilder(t *testing.T) {
	builder := NewCacheKeyBuilder()

	key1 := builder.
		AddComponent("GET").
		AddComponent("/api/users").
		AddParam("page", "1").
		Build()

	// Same components should generate same key
	builder2 := NewCacheKeyBuilder()
	key2 := builder2.
		AddComponent("GET").
		AddComponent("/api/users").
		AddParam("page", "1").
		Build()

	if key1 != key2 {
		t.Error("Same components should generate same key")
	}

	// Different params should generate different key
	builder3 := NewCacheKeyBuilder()
	key3 := builder3.
		AddComponent("GET").
		AddComponent("/api/users").
		AddParam("page", "2").
		Build()

	if key1 == key3 {
		t.Error("Different params should generate different keys")
	}
}

// TestDefaultPolicyStatusCodes tests various status codes
func TestDefaultPolicyStatusCodes(t *testing.T) {
	policy := NewDefaultPolicy(DefaultPolicyConfig{
		CacheableStatusCodes: []int{200, 203, 206, 300, 301, 410},
	})

	tests := []struct {
		statusCode      int
		shouldBeCacheable bool
	}{
		{200, true},
		{203, true},
		{206, true},
		{204, false},
		{301, true},
		{400, false},
		{404, false},
		{500, false},
	}

	for _, tt := range tests {
		req, _ := http.NewRequest("GET", "http://example.com", nil)
		resp := &http.Response{
			StatusCode: tt.statusCode,
			Header:     http.Header{},
		}

		result := policy.ShouldCache(req, resp)
		if result != tt.shouldBeCacheable {
			t.Errorf("Status %d: expected %v, got %v", tt.statusCode, tt.shouldBeCacheable, result)
		}
	}
}

// TestDefaultPolicyRequiredHeaders tests required headers
func TestDefaultPolicyRequiredHeaders(t *testing.T) {
	policy := NewDefaultPolicy(DefaultPolicyConfig{
		RequiredHeaders: []string{"X-Cache-Me"},
	})

	// Without required header
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp := &http.Response{
		StatusCode: 200,
		Header:     http.Header{},
	}

	if policy.ShouldCache(req, resp) {
		t.Error("Should not cache without required header")
	}

	// With required header
	resp.Header.Set("X-Cache-Me", "yes")
	if !policy.ShouldCache(req, resp) {
		t.Error("Should cache with required header")
	}
}

// TestDefaultPolicyDefaultConfig tests default configuration
func TestDefaultPolicyDefaultConfig(t *testing.T) {
	policy := NewDefaultPolicy(DefaultPolicyConfig{})

	if policy.config.DefaultTTL != 5*time.Minute {
		t.Errorf("Expected default TTL 5min, got %v", policy.config.DefaultTTL)
	}

	if policy.config.MaxTTL != 24*time.Hour {
		t.Errorf("Expected max TTL 24h, got %v", policy.config.MaxTTL)
	}

	if len(policy.config.CacheableStatusCodes) == 0 {
		t.Error("Expected cacheable status codes to be set")
	}

	if len(policy.config.CacheableMethods) == 0 {
		t.Error("Expected cacheable methods to be set")
	}
}

// TestExtractTTLFromCacheControl tests TTL extraction from Cache-Control
func TestExtractTTLFromCacheControl(t *testing.T) {
	policy := NewDefaultPolicy(DefaultPolicyConfig{})

	tests := []struct {
		cacheControl string
		expectedTTL  time.Duration
	}{
		{"max-age=3600", 3600 * time.Second},
		{"max-age=60", 60 * time.Second},
		{"s-maxage=1800", 1800 * time.Second},
		{"public, max-age=3600", 3600 * time.Second},
		{"no-cache", 0},
		{"", 0},
	}

	for _, tt := range tests {
		resp := &http.Response{
			Header: http.Header{
				"Cache-Control": []string{tt.cacheControl},
			},
		}

		ttl := policy.extractTTLFromCacheControl(resp)
		if ttl != tt.expectedTTL {
			t.Errorf("Cache-Control '%s': expected %v, got %v", tt.cacheControl, tt.expectedTTL, ttl)
		}
	}
}

// TestPathPatternMatching tests path pattern matching
func TestPathPatternMatching(t *testing.T) {
	policy := NewPathBasedPolicy(map[string]PathRule{}, nil, 5*time.Minute)

	tests := []struct {
		pattern string
		path    string
		matches bool
	}{
		{"/api/*", "/api/users", true},
		{"/api/*", "/api/products", true},
		{"*", "/anything", true},
		{"/exact", "/exact", true},
		{"/exact", "/exact/path", false},
	}

	for _, tt := range tests {
		result := policy.pathMatches(tt.path, tt.pattern)
		if result != tt.matches {
			t.Errorf("Pattern '%s' vs path '%s': expected %v, got %v", tt.pattern, tt.path, tt.matches, result)
		}
	}
}

// TestMethodMatching tests method matching logic
func TestMethodMatching(t *testing.T) {
	policy := NewPathBasedPolicy(map[string]PathRule{}, nil, 5*time.Minute)

	tests := []struct {
		method         string
		allowedMethods []string
		matches        bool
	}{
		{"GET", []string{"GET", "HEAD"}, true},
		{"POST", []string{"GET", "HEAD"}, false},
		{"GET", []string{}, true}, // Empty allowed methods means all
		{"POST", []string{}, true},
	}

	for _, tt := range tests {
		result := policy.methodMatches(tt.method, tt.allowedMethods)
		if result != tt.matches {
			t.Errorf("Method '%s' vs %v: expected %v, got %v", tt.method, tt.allowedMethods, tt.matches, result)
		}
	}
}

// TestConditionalPolicyShouldInvalidate tests invalidation in conditional policy
func TestConditionalPolicyShouldInvalidate(t *testing.T) {
	basePolicy := NewDefaultPolicy(DefaultPolicyConfig{})
	conditional := NewConditionalPolicy([]CacheCondition{}, basePolicy)

	req, _ := http.NewRequest("POST", "http://example.com", nil)
	if !conditional.ShouldInvalidate(req) {
		t.Error("POST request should trigger invalidation")
	}

	req, _ = http.NewRequest("GET", "http://example.com", nil)
	if conditional.ShouldInvalidate(req) {
		t.Error("GET request should not trigger invalidation")
	}
}

// TestHeaderBasedPolicyShouldInvalidate tests invalidation in header-based policy
func TestHeaderBasedPolicyShouldInvalidate(t *testing.T) {
	policy := NewHeaderBasedPolicy(map[string]HeaderRule{}, 5*time.Minute)

	req, _ := http.NewRequest("DELETE", "http://example.com", nil)
	if !policy.ShouldInvalidate(req) {
		t.Error("DELETE request should trigger invalidation")
	}

	req, _ = http.NewRequest("GET", "http://example.com", nil)
	if policy.ShouldInvalidate(req) {
		t.Error("GET request should not trigger invalidation")
	}
}

// TestPathBasedPolicyShouldInvalidate tests invalidation in path-based policy
func TestPathBasedPolicyShouldInvalidate(t *testing.T) {
	policy := NewPathBasedPolicy(map[string]PathRule{}, nil, 5*time.Minute)

	req, _ := http.NewRequest("PUT", "http://example.com", nil)
	if !policy.ShouldInvalidate(req) {
		t.Error("PUT request should trigger invalidation")
	}

	req, _ = http.NewRequest("HEAD", "http://example.com", nil)
	if policy.ShouldInvalidate(req) {
		t.Error("HEAD request should not trigger invalidation")
	}
}

// TestPathBasedPolicyGetTags tests tag extraction in path-based policy
func TestPathBasedPolicyGetTags(t *testing.T) {
	rules := map[string]PathRule{
		"/api/users": {
			Tags: []string{"users", "api"},
		},
	}

	policy := NewPathBasedPolicy(rules, nil, 5*time.Minute)

	req, _ := http.NewRequest("GET", "http://example.com/api/users", nil)
	resp := &http.Response{Header: http.Header{}}

	tags := policy.GetTags(req, resp)

	if len(tags) == 0 {
		t.Error("Expected tags")
	}

	// Should include path tag
	hasPathTag := false
	for _, tag := range tags {
		if tag == "path:/api/users" {
			hasPathTag = true
		}
	}

	if !hasPathTag {
		t.Error("Expected path tag")
	}
}

// TestDefaultPolicyGenerateKeyWithVaryHeaders tests key generation with vary headers
func TestDefaultPolicyGenerateKeyWithVaryHeaders(t *testing.T) {
	policy := NewDefaultPolicy(DefaultPolicyConfig{
		VaryHeaders: []string{"Accept", "Accept-Encoding"},
	})

	req1, _ := http.NewRequest("GET", "http://example.com/api/data", nil)
	req1.Header.Set("Accept", "application/json")
	req1.Header.Set("Accept-Encoding", "gzip")

	req2, _ := http.NewRequest("GET", "http://example.com/api/data", nil)
	req2.Header.Set("Accept", "application/json")
	req2.Header.Set("Accept-Encoding", "deflate")

	key1 := policy.GenerateKey(req1)
	key2 := policy.GenerateKey(req2)

	if key1 == key2 {
		t.Error("Different Accept-Encoding should generate different keys")
	}
}

// BenchmarkDefaultPolicyShouldCache benchmarks should cache logic
func BenchmarkDefaultPolicyShouldCache(b *testing.B) {
	policy := NewDefaultPolicy(DefaultPolicyConfig{
		DefaultTTL:           5 * time.Minute,
		CacheableStatusCodes: []int{200, 203, 206},
		CacheableMethods:     []string{"GET", "HEAD"},
	})

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp := &http.Response{
		StatusCode: 200,
		Header:     http.Header{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		policy.ShouldCache(req, resp)
	}
}

// BenchmarkDefaultPolicyGenerateKey benchmarks key generation
func BenchmarkDefaultPolicyGenerateKey(b *testing.B) {
	policy := NewDefaultPolicy(DefaultPolicyConfig{
		VaryHeaders: []string{"Accept", "Authorization"},
	})

	req, _ := http.NewRequest("GET", "http://example.com/api/users?page=1&size=10", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer token")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		policy.GenerateKey(req)
	}
}

// BenchmarkCacheKeyBuilder benchmarks key builder
func BenchmarkCacheKeyBuilder(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder := NewCacheKeyBuilder()
		builder.AddComponent("GET").
			AddComponent("/api/users").
			AddParam("page", strconv.Itoa(i)).
			AddParam("size", "10").
			Build()
	}
}
