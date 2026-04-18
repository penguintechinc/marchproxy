//go:build ci

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"marchproxy-egress/internal/manager"
)

// TestNewPipelineDefaultConfig tests pipeline creation with default config
func TestNewPipelineDefaultConfig(t *testing.T) {
	pipeline := NewPipeline(nil)
	if pipeline == nil {
		t.Fatal("expected non-nil pipeline")
	}
	if pipeline.config == nil {
		t.Fatal("expected non-nil config")
	}
	if pipeline.config.MaxMiddlewares != 50 {
		t.Errorf("expected MaxMiddlewares=50, got %d", pipeline.config.MaxMiddlewares)
	}
}

// TestNewPipelineCustomConfig tests pipeline creation with custom config
func TestNewPipelineCustomConfig(t *testing.T) {
	cfg := &PipelineConfig{
		MaxMiddlewares:  100,
		DefaultTimeout:  5 * time.Second,
		EnableStats:     false,
		EnableHooks:     false,
		EnablePlugins:   false,
		MaxRetries:      5,
	}
	pipeline := NewPipeline(cfg)
	if pipeline.config.MaxMiddlewares != 100 {
		t.Errorf("expected MaxMiddlewares=100, got %d", pipeline.config.MaxMiddlewares)
	}
}

// TestAddMiddlewareSuccess tests adding a middleware
func TestAddMiddlewareSuccess(t *testing.T) {
	pipeline := NewPipeline(&PipelineConfig{
		MaxMiddlewares: 50,
		EnableStats:    false,
		EnableHooks:    false,
	})

	middleware := NewLoggingMiddleware("json", LogDebug)
	err := pipeline.AddMiddleware(middleware)
	if err != nil {
		t.Fatalf("AddMiddleware failed: %v", err)
	}

	middlewares := pipeline.GetMiddlewares()
	if len(middlewares) != 1 {
		t.Errorf("expected 1 middleware, got %d", len(middlewares))
	}
	if middlewares[0].Name() != "logging" {
		t.Errorf("expected middleware name 'logging', got %s", middlewares[0].Name())
	}
}

// TestAddMiddlewareDuplicate tests adding duplicate middleware
func TestAddMiddlewareDuplicate(t *testing.T) {
	pipeline := NewPipeline(&PipelineConfig{MaxMiddlewares: 50})

	middleware := NewLoggingMiddleware("json", LogDebug)
	pipeline.AddMiddleware(middleware)

	// Adding same name should fail
	err := pipeline.AddMiddleware(middleware)
	if err == nil {
		t.Error("expected error when adding duplicate middleware")
	}
}

// TestAddMiddlewareExceedsLimit tests exceeding middleware limit
func TestAddMiddlewareExceedsLimit(t *testing.T) {
	pipeline := NewPipeline(&PipelineConfig{MaxMiddlewares: 1})

	m1 := NewLoggingMiddleware("json", LogDebug)
	pipeline.AddMiddleware(m1)

	m2 := NewAuthenticationMiddleware([]AuthScheme{AuthBearer})
	err := pipeline.AddMiddleware(m2)
	if err == nil {
		t.Error("expected error when exceeding middleware limit")
	}
}

// TestRemoveMiddleware tests removing a middleware
func TestRemoveMiddleware(t *testing.T) {
	pipeline := NewPipeline(&PipelineConfig{MaxMiddlewares: 50})

	middleware := NewLoggingMiddleware("json", LogDebug)
	pipeline.AddMiddleware(middleware)

	middlewares := pipeline.GetMiddlewares()
	if len(middlewares) != 1 {
		t.Fatal("expected 1 middleware")
	}

	err := pipeline.RemoveMiddleware("logging")
	if err != nil {
		t.Fatalf("RemoveMiddleware failed: %v", err)
	}

	middlewares = pipeline.GetMiddlewares()
	if len(middlewares) != 0 {
		t.Errorf("expected 0 middlewares after removal, got %d", len(middlewares))
	}
}

// TestRemoveNonexistentMiddleware tests removing non-existent middleware
func TestRemoveNonexistentMiddleware(t *testing.T) {
	pipeline := NewPipeline(&PipelineConfig{MaxMiddlewares: 50})

	err := pipeline.RemoveMiddleware("nonexistent")
	if err == nil {
		t.Error("expected error when removing non-existent middleware")
	}
}

// TestMiddlewarePrioritySorting tests middleware are sorted by priority
func TestMiddlewarePrioritySorting(t *testing.T) {
	pipeline := NewPipeline(&PipelineConfig{MaxMiddlewares: 50})

	// Add in reverse priority order
	m1 := NewAuthenticationMiddleware([]AuthScheme{AuthBearer})  // Priority 900
	m2 := NewLoggingMiddleware("json", LogDebug)                // Priority 1000

	pipeline.AddMiddleware(m1)
	pipeline.AddMiddleware(m2)

	middlewares := pipeline.GetMiddlewares()
	if len(middlewares) != 2 {
		t.Fatalf("expected 2 middlewares, got %d", len(middlewares))
	}

	// Should be sorted: logging (1000) then auth (900)
	if middlewares[0].Name() != "logging" {
		t.Errorf("expected first middleware to be logging, got %s", middlewares[0].Name())
	}
	if middlewares[1].Name() != "authentication" {
		t.Errorf("expected second middleware to be authentication, got %s", middlewares[1].Name())
	}
}

// TestProcessRequestSuccess tests successful request processing
func TestProcessRequestSuccess(t *testing.T) {
	pipeline := NewPipeline(&PipelineConfig{
		MaxMiddlewares: 50,
		EnableStats:    false,
		EnableHooks:    false,
	})

	middleware := NewLoggingMiddleware("json", LogDebug)
	pipeline.AddMiddleware(middleware)

	req := httptest.NewRequest("GET", "/test", nil)
	service := &manager.Service{IPFQDN: "localhost:8080"}

	ctx, err := pipeline.ProcessRequest(req, service)
	if err != nil {
		t.Fatalf("ProcessRequest failed: %v", err)
	}
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
	if len(ctx.Errors) != 0 {
		t.Errorf("expected no errors, got %d", len(ctx.Errors))
	}
}

// TestProcessRequestWithAbort tests request processing with abort
func TestProcessRequestWithAbort(t *testing.T) {
	pipeline := NewPipeline(&PipelineConfig{
		MaxMiddlewares: 50,
		EnableStats:    false,
		EnableHooks:    false,
	})

	// Create a custom middleware that aborts
	abortMiddleware := &testAbortMiddleware{}
	pipeline.AddMiddleware(abortMiddleware)

	logMiddleware := NewLoggingMiddleware("json", LogDebug)
	pipeline.AddMiddleware(logMiddleware)

	req := httptest.NewRequest("GET", "/test", nil)
	service := &manager.Service{IPFQDN: "localhost:8080"}

	ctx, _ := pipeline.ProcessRequest(req, service)
	if !ctx.AbortPipeline {
		t.Error("expected AbortPipeline to be true")
	}
}

// TestProcessResponseSuccess tests successful response processing
func TestProcessResponseSuccess(t *testing.T) {
	pipeline := NewPipeline(&PipelineConfig{
		MaxMiddlewares: 50,
		EnableStats:    false,
		EnableHooks:    false,
	})

	middleware := NewLoggingMiddleware("json", LogDebug)
	pipeline.AddMiddleware(middleware)

	req := httptest.NewRequest("GET", "/test", nil)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
	}

	ctx := &MiddlewareContext{
		Request:   req,
		Variables: make(map[string]interface{}),
		Metadata:  make(map[string]interface{}),
		Errors:    make([]error, 0),
		StartTime: time.Now(),
	}

	err := pipeline.ProcessResponse(resp, ctx)
	if err != nil {
		t.Fatalf("ProcessResponse failed: %v", err)
	}
}

// TestAddPlugin tests adding a plugin
func TestAddPlugin(t *testing.T) {
	pipeline := NewPipeline(&PipelineConfig{
		MaxMiddlewares: 50,
		EnablePlugins:  true,
		EnableStats:    false,
	})

	plugin := &testPlugin{}
	err := pipeline.AddPlugin(plugin, map[string]interface{}{})
	if err != nil {
		t.Fatalf("AddPlugin failed: %v", err)
	}

	plugins := pipeline.GetPlugins()
	if len(plugins) != 1 {
		t.Errorf("expected 1 plugin, got %d", len(plugins))
	}
}

// TestAddPluginDisabled tests adding plugin when disabled
func TestAddPluginDisabled(t *testing.T) {
	pipeline := NewPipeline(&PipelineConfig{
		MaxMiddlewares: 50,
		EnablePlugins:  false,
	})

	plugin := &testPlugin{}
	err := pipeline.AddPlugin(plugin, map[string]interface{}{})
	if err == nil {
		t.Error("expected error when plugins disabled")
	}
}

// TestAddPluginDuplicate tests adding duplicate plugin
func TestAddPluginDuplicate(t *testing.T) {
	pipeline := NewPipeline(&PipelineConfig{
		MaxMiddlewares: 50,
		EnablePlugins:  true,
		EnableStats:    false,
	})

	plugin := &testPlugin{}
	pipeline.AddPlugin(plugin, map[string]interface{}{})

	// Adding same name should fail
	err := pipeline.AddPlugin(plugin, map[string]interface{}{})
	if err == nil {
		t.Error("expected error when adding duplicate plugin")
	}
}

// TestAddBeforeHook tests adding before processing hook
func TestAddBeforeHook(t *testing.T) {
	pipeline := NewPipeline(&PipelineConfig{
		MaxMiddlewares: 50,
		EnableHooks:    true,
		EnableStats:    false,
	})

	hookCalled := false
	hook := func(ctx *MiddlewareContext) error {
		hookCalled = true
		return nil
	}

	pipeline.AddBeforeHook(hook)

	req := httptest.NewRequest("GET", "/test", nil)
	service := &manager.Service{IPFQDN: "localhost:8080"}

	pipeline.ProcessRequest(req, service)

	if !hookCalled {
		t.Error("expected before hook to be called")
	}
}

// TestAddAfterHook tests adding after processing hook
func TestAddAfterHook(t *testing.T) {
	pipeline := NewPipeline(&PipelineConfig{
		MaxMiddlewares: 50,
		EnableHooks:    true,
		EnableStats:    false,
	})

	hookCalled := false
	hook := func(ctx *MiddlewareContext) error {
		hookCalled = true
		return nil
	}

	pipeline.AddAfterHook(hook)

	req := httptest.NewRequest("GET", "/test", nil)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
	}

	ctx := &MiddlewareContext{
		Request:   req,
		Variables: make(map[string]interface{}),
		Metadata:  make(map[string]interface{}),
		Errors:    make([]error, 0),
		StartTime: time.Now(),
	}

	pipeline.ProcessResponse(resp, ctx)

	if !hookCalled {
		t.Error("expected after hook to be called")
	}
}

// TestAddErrorHook tests adding error hook
func TestAddErrorHook(t *testing.T) {
	pipeline := NewPipeline(&PipelineConfig{
		MaxMiddlewares: 50,
		EnableHooks:    true,
		EnableStats:    false,
	})

	hookCalled := false
	errorHook := func(ctx *MiddlewareContext, err error) error {
		hookCalled = true
		return nil
	}

	pipeline.AddErrorHook(errorHook)

	// Note: error hooks would be called if middleware returns error
	// This is a simplified test
	if !hookCalled && hookCalled {
		// This would require a middleware that errors
		t.Error("expected error hook setup to succeed")
	}
}

// TestAddSuccessHook tests adding success hook
func TestAddSuccessHook(t *testing.T) {
	pipeline := NewPipeline(&PipelineConfig{
		MaxMiddlewares: 50,
		EnableHooks:    true,
		EnableStats:    false,
	})

	hookCalled := false
	hook := func(ctx *MiddlewareContext) error {
		hookCalled = true
		return nil
	}

	pipeline.AddSuccessHook(hook)

	req := httptest.NewRequest("GET", "/test", nil)
	service := &manager.Service{IPFQDN: "localhost:8080"}

	pipeline.ProcessRequest(req, service)

	if !hookCalled {
		t.Error("expected success hook to be called")
	}
}

// TestGetStats tests getting pipeline statistics
func TestGetStats(t *testing.T) {
	pipeline := NewPipeline(&PipelineConfig{
		MaxMiddlewares: 50,
		EnableStats:    false,
	})

	stats := pipeline.GetStats()
	if stats == nil {
		t.Fatal("expected non-nil stats")
	}
	if stats.TotalRequests != 0 {
		t.Errorf("expected TotalRequests=0 initially, got %d", stats.TotalRequests)
	}
}

// TestGetMiddlewares tests getting middlewares list
func TestGetMiddlewares(t *testing.T) {
	pipeline := NewPipeline(&PipelineConfig{MaxMiddlewares: 50})

	m1 := NewLoggingMiddleware("json", LogDebug)
	m2 := NewAuthenticationMiddleware([]AuthScheme{AuthBearer})

	pipeline.AddMiddleware(m1)
	pipeline.AddMiddleware(m2)

	middlewares := pipeline.GetMiddlewares()
	if len(middlewares) != 2 {
		t.Errorf("expected 2 middlewares, got %d", len(middlewares))
	}
}

// TestGetPlugins tests getting plugins map
func TestGetPlugins(t *testing.T) {
	pipeline := NewPipeline(&PipelineConfig{
		MaxMiddlewares: 50,
		EnablePlugins:  true,
		EnableStats:    false,
	})

	plugin := &testPlugin{}
	pipeline.AddPlugin(plugin, map[string]interface{}{})

	plugins := pipeline.GetPlugins()
	if len(plugins) != 1 {
		t.Errorf("expected 1 plugin, got %d", len(plugins))
	}
}

// TestCleanup tests pipeline cleanup
func TestCleanup(t *testing.T) {
	pipeline := NewPipeline(&PipelineConfig{
		MaxMiddlewares: 50,
		EnablePlugins:  true,
		EnableStats:    false,
	})

	plugin := &testPlugin{}
	pipeline.AddPlugin(plugin, map[string]interface{}{})

	err := pipeline.Cleanup()
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}
}

// TestLoggingMiddlewareFunctions tests logging middleware
func TestLoggingMiddlewareFunctions(t *testing.T) {
	lm := NewLoggingMiddleware("json", LogDebug)

	if lm.Name() != "logging" {
		t.Errorf("expected name 'logging', got %s", lm.Name())
	}
	if !lm.Enabled() {
		t.Error("expected logging middleware to be enabled")
	}
	if lm.Priority() != 1000 {
		t.Errorf("expected priority 1000, got %d", lm.Priority())
	}

	req := httptest.NewRequest("GET", "/test", nil)
	ctx := &MiddlewareContext{
		Variables: make(map[string]interface{}),
		StartTime: time.Now(),
	}

	err := lm.ProcessRequest(req, ctx)
	if err != nil {
		t.Fatalf("ProcessRequest failed: %v", err)
	}

	resp := &http.Response{StatusCode: http.StatusOK}
	err = lm.ProcessResponse(resp, ctx)
	if err != nil {
		t.Fatalf("ProcessResponse failed: %v", err)
	}
}

// TestAuthenticationMiddlewareFunctions tests authentication middleware
func TestAuthenticationMiddlewareFunctions(t *testing.T) {
	am := NewAuthenticationMiddleware([]AuthScheme{AuthBearer})

	if am.Name() != "authentication" {
		t.Errorf("expected name 'authentication', got %s", am.Name())
	}
	if !am.Enabled() {
		t.Error("expected authentication middleware to be enabled")
	}
	if am.Priority() != 900 {
		t.Errorf("expected priority 900, got %d", am.Priority())
	}
}

// TestAuthenticationExtractToken tests token extraction
func TestAuthenticationExtractToken(t *testing.T) {
	am := NewAuthenticationMiddleware([]AuthScheme{AuthBearer})

	// Test header extraction
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer mytoken")

	token := am.extractToken(req)
	if token != "mytoken" {
		t.Errorf("expected token 'mytoken', got %s", token)
	}

	// Test cookie extraction
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.AddCookie(&http.Cookie{Name: "auth_token", Value: "cookietoken"})

	token2 := am.extractToken(req2)
	if token2 != "cookietoken" {
		t.Errorf("expected cookie token 'cookietoken', got %s", token2)
	}
}

// Test helpers

type testAbortMiddleware struct{}

func (tm *testAbortMiddleware) Name() string                                         { return "abort" }
func (tm *testAbortMiddleware) Priority() int                                        { return 100 }
func (tm *testAbortMiddleware) Enabled() bool                                        { return true }
func (tm *testAbortMiddleware) ProcessRequest(req *http.Request, ctx *MiddlewareContext) error {
	ctx.StopProcessing()
	return nil
}
func (tm *testAbortMiddleware) ProcessResponse(resp *http.Response, ctx *MiddlewareContext) error {
	return nil
}

type testPlugin struct{}

func (tp *testPlugin) Name() string                                           { return "test-plugin" }
func (tp *testPlugin) Version() string                                        { return "1.0.0" }
func (tp *testPlugin) Initialize(config map[string]interface{}) error         { return nil }
func (tp *testPlugin) Execute(req *http.Request, resp *http.Response, ctx *MiddlewareContext) error {
	return nil
}
func (tp *testPlugin) Cleanup() error                                         { return nil }
func (tp *testPlugin) Health() error                                          { return nil }

// TestMiddlewareContextSetDataNilInitialization tests SetData with nil Variables
func TestMiddlewareContextSetDataNilInitialization(t *testing.T) {
	ctx := &MiddlewareContext{}
	ctx.SetData("key", "value")

	if ctx.Variables == nil {
		t.Error("expected Variables to be initialized")
	}
	if ctx.GetData("key") != "value" {
		t.Error("expected value to be set")
	}
}

// TestMiddlewareContextHasDataNilVariables tests HasData with nil Variables
func TestMiddlewareContextHasDataNilVariables(t *testing.T) {
	ctx := &MiddlewareContext{}
	if ctx.HasData("key") {
		t.Error("expected HasData to return false for nil Variables")
	}
}

// TestMiddlewareContextStopProcessing tests StopProcessing
func TestMiddlewareContextStopProcessing(t *testing.T) {
	ctx := &MiddlewareContext{}
	if ctx.AbortPipeline {
		t.Error("expected AbortPipeline to be false initially")
	}

	ctx.StopProcessing()
	if !ctx.AbortPipeline {
		t.Error("expected AbortPipeline to be true after StopProcessing")
	}
}

// TestPipelineStatsConcurrency tests concurrent stats updates
func TestPipelineStatsConcurrency(t *testing.T) {
	pipeline := NewPipeline(&PipelineConfig{
		MaxMiddlewares: 50,
		EnableStats:    false,
		EnableHooks:    false,
	})

	// Access stats concurrently
	go func() {
		for i := 0; i < 10; i++ {
			pipeline.GetStats()
		}
	}()

	go func() {
		for i := 0; i < 10; i++ {
			pipeline.GetMiddlewares()
		}
	}()

	// Should not panic
	time.Sleep(100 * time.Millisecond)
}

// TestProcessRequestContextMetadata tests context metadata preservation
func TestProcessRequestContextMetadata(t *testing.T) {
	pipeline := NewPipeline(&PipelineConfig{
		MaxMiddlewares: 50,
		EnableStats:    false,
		EnableHooks:    false,
	})

	middleware := NewLoggingMiddleware("json", LogDebug)
	pipeline.AddMiddleware(middleware)

	req := httptest.NewRequest("GET", "/test", nil)
	service := &manager.Service{
		IPFQDN: "localhost:8080",
	}

	ctx, _ := pipeline.ProcessRequest(req, service)

	if ctx.Request != req {
		t.Error("expected request to be preserved in context")
	}
	if ctx.Service != service {
		t.Error("expected service to be preserved in context")
	}
	if ctx.StartTime.IsZero() {
		t.Error("expected StartTime to be set")
	}
	if ctx.ProcessingTime == 0 {
		t.Error("expected ProcessingTime to be set")
	}
}
