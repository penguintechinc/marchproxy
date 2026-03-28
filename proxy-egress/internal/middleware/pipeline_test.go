package middleware_test

import (
	"testing"
	"time"

	"marchproxy-egress/internal/middleware"
)

func TestMiddlewareContextSetGet(t *testing.T) {
	ctx := &middleware.MiddlewareContext{
		Variables: make(map[string]interface{}),
		Metadata:  make(map[string]interface{}),
		Errors:    make([]error, 0),
		StartTime: time.Now(),
	}

	ctx.SetData("key", "value")
	val := ctx.GetData("key")
	if val == nil {
		t.Fatal("expected GetData to return non-nil value")
	}
	if val != "value" {
		t.Errorf("expected GetData(\"key\") = %q, got %v", "value", val)
	}
}

func TestMiddlewareContextGetMissing(t *testing.T) {
	ctx := &middleware.MiddlewareContext{
		Variables: make(map[string]interface{}),
	}
	val := ctx.GetData("nonexistent")
	if val != nil {
		t.Errorf("expected nil for missing key, got %v", val)
	}
}

func TestMiddlewareContextHasData(t *testing.T) {
	ctx := &middleware.MiddlewareContext{
		Variables: make(map[string]interface{}),
	}

	if ctx.HasData("missing") {
		t.Error("HasData should return false for missing key")
	}

	ctx.SetData("present", true)
	if !ctx.HasData("present") {
		t.Error("HasData should return true for existing key")
	}
}

func TestMiddlewareContextSetGetNilVariables(t *testing.T) {
	// MiddlewareContext with nil Variables — SetData should initialize it
	ctx := &middleware.MiddlewareContext{}
	ctx.SetData("key", 42)

	val := ctx.GetData("key")
	if val != 42 {
		t.Errorf("expected 42, got %v", val)
	}
}

func TestMiddlewareContextHasDataNilVariables(t *testing.T) {
	ctx := &middleware.MiddlewareContext{}
	if ctx.HasData("anything") {
		t.Error("HasData on nil Variables should return false")
	}
}

func TestMiddlewareContextGetDataNilVariables(t *testing.T) {
	ctx := &middleware.MiddlewareContext{}
	val := ctx.GetData("key")
	if val != nil {
		t.Errorf("GetData on nil Variables should return nil, got %v", val)
	}
}

func TestMiddlewareContextStopProcessing(t *testing.T) {
	ctx := &middleware.MiddlewareContext{
		Variables: make(map[string]interface{}),
	}

	if ctx.AbortPipeline {
		t.Error("AbortPipeline should be false initially")
	}

	ctx.StopProcessing()

	if !ctx.AbortPipeline {
		t.Error("AbortPipeline should be true after StopProcessing()")
	}
}

func TestMiddlewareContextMultipleValues(t *testing.T) {
	ctx := &middleware.MiddlewareContext{
		Variables: make(map[string]interface{}),
	}

	ctx.SetData("string-key", "hello")
	ctx.SetData("int-key", 123)
	ctx.SetData("bool-key", true)
	ctx.SetData("slice-key", []int{1, 2, 3})

	if ctx.GetData("string-key") != "hello" {
		t.Error("expected string-key = 'hello'")
	}
	if ctx.GetData("int-key") != 123 {
		t.Error("expected int-key = 123")
	}
	if ctx.GetData("bool-key") != true {
		t.Error("expected bool-key = true")
	}
}

func TestMiddlewareContextOverwrite(t *testing.T) {
	ctx := &middleware.MiddlewareContext{
		Variables: make(map[string]interface{}),
	}

	ctx.SetData("key", "first")
	ctx.SetData("key", "second")

	if ctx.GetData("key") != "second" {
		t.Errorf("expected overwritten value 'second', got %v", ctx.GetData("key"))
	}
}

func TestNewPipelineNotNil(t *testing.T) {
	p := middleware.NewPipeline(nil)
	if p == nil {
		t.Fatal("NewPipeline(nil) should return non-nil pipeline")
	}
}

func TestNewPipelineWithConfig(t *testing.T) {
	cfg := &middleware.PipelineConfig{
		MaxMiddlewares:  20,
		DefaultTimeout:  10 * time.Second,
		EnableStats:     false,
		StatsInterval:   time.Minute,
		EnableHooks:     false,
		EnablePlugins:   false,
		MaxRetries:      2,
		RetryDelay:      50 * time.Millisecond,
		EnableProfiling: false,
		EnableTracing:   false,
	}
	p := middleware.NewPipeline(cfg)
	if p == nil {
		t.Fatal("NewPipeline with config should return non-nil pipeline")
	}
}

func TestPipelineGetMiddlewaresEmpty(t *testing.T) {
	p := middleware.NewPipeline(nil)
	middlewares := p.GetMiddlewares()
	if middlewares == nil {
		t.Fatal("GetMiddlewares should return non-nil slice")
	}
	if len(middlewares) != 0 {
		t.Errorf("expected 0 middlewares initially, got %d", len(middlewares))
	}
}

func TestPipelineGetPluginsEmpty(t *testing.T) {
	p := middleware.NewPipeline(nil)
	plugins := p.GetPlugins()
	if plugins == nil {
		t.Fatal("GetPlugins should return non-nil map")
	}
	if len(plugins) != 0 {
		t.Errorf("expected 0 plugins initially, got %d", len(plugins))
	}
}

func TestPipelineGetStatsInitial(t *testing.T) {
	cfg := &middleware.PipelineConfig{
		EnableStats: false,
		MaxRetries:  1,
	}
	p := middleware.NewPipeline(cfg)
	stats := p.GetStats()
	if stats == nil {
		t.Fatal("GetStats should return non-nil stats")
	}
	if stats.TotalRequests != 0 {
		t.Errorf("expected 0 total requests initially, got %d", stats.TotalRequests)
	}
}

func TestPipelineRemoveNonExistentMiddleware(t *testing.T) {
	p := middleware.NewPipeline(nil)
	err := p.RemoveMiddleware("does-not-exist")
	if err == nil {
		t.Error("expected error when removing non-existent middleware")
	}
}

func TestNewLoggingMiddlewareNotNil(t *testing.T) {
	lm := middleware.NewLoggingMiddleware("json", middleware.LogInfo)
	if lm == nil {
		t.Fatal("NewLoggingMiddleware should return non-nil")
	}
	if lm.Name() != "logging" {
		t.Errorf("expected Name() = %q, got %q", "logging", lm.Name())
	}
	if !lm.Enabled() {
		t.Error("new LoggingMiddleware should be enabled")
	}
	if lm.Priority() <= 0 {
		t.Errorf("expected positive Priority, got %d", lm.Priority())
	}
}

func TestNewAuthenticationMiddlewareNotNil(t *testing.T) {
	am := middleware.NewAuthenticationMiddleware([]middleware.AuthScheme{middleware.AuthBearer})
	if am == nil {
		t.Fatal("NewAuthenticationMiddleware should return non-nil")
	}
	if am.Name() != "authentication" {
		t.Errorf("expected Name() = %q, got %q", "authentication", am.Name())
	}
	if !am.Enabled() {
		t.Error("new AuthenticationMiddleware should be enabled")
	}
}

func TestLogLevelConstants(t *testing.T) {
	levels := []middleware.LogLevel{
		middleware.LogDebug,
		middleware.LogInfo,
		middleware.LogWarn,
		middleware.LogError,
	}
	seen := make(map[middleware.LogLevel]bool)
	for _, l := range levels {
		if seen[l] {
			t.Errorf("duplicate LogLevel: %d", l)
		}
		seen[l] = true
	}
}

func TestAuthSchemeConstants(t *testing.T) {
	schemes := []middleware.AuthScheme{
		middleware.AuthBasic,
		middleware.AuthBearer,
		middleware.AuthJWT,
		middleware.AuthAPIKey,
		middleware.AuthOAuth2,
	}
	seen := make(map[middleware.AuthScheme]bool)
	for _, s := range schemes {
		if seen[s] {
			t.Errorf("duplicate AuthScheme: %d", s)
		}
		seen[s] = true
	}
}

func TestCompressionAlgorithmConstants(t *testing.T) {
	algos := []middleware.CompressionAlgorithm{
		middleware.CompressionGzip,
		middleware.CompressionBrotli,
		middleware.CompressionZstd,
		middleware.CompressionDeflate,
	}
	seen := make(map[middleware.CompressionAlgorithm]bool)
	for _, a := range algos {
		if seen[a] {
			t.Errorf("duplicate CompressionAlgorithm: %d", a)
		}
		seen[a] = true
	}
}

func TestCircuitStateConstants(t *testing.T) {
	states := []middleware.CircuitState{
		middleware.CircuitClosed,
		middleware.CircuitOpen,
		middleware.CircuitHalfOpen,
	}
	seen := make(map[middleware.CircuitState]bool)
	for _, s := range states {
		if seen[s] {
			t.Errorf("duplicate CircuitState: %d", s)
		}
		seen[s] = true
	}
}

func TestPipelineAddBeforeHook(t *testing.T) {
	p := middleware.NewPipeline(nil)
	called := false
	p.AddBeforeHook(func(ctx *middleware.MiddlewareContext) error {
		called = true
		return nil
	})
	// Hook registered without error — just verify no panic
	_ = called
}

func TestPipelineAddAfterHook(t *testing.T) {
	p := middleware.NewPipeline(nil)
	p.AddAfterHook(func(ctx *middleware.MiddlewareContext) error {
		return nil
	})
}

func TestPipelineAddErrorHook(t *testing.T) {
	p := middleware.NewPipeline(nil)
	p.AddErrorHook(func(ctx *middleware.MiddlewareContext, err error) error {
		return nil
	})
}

func TestPipelineAddSuccessHook(t *testing.T) {
	p := middleware.NewPipeline(nil)
	p.AddSuccessHook(func(ctx *middleware.MiddlewareContext) error {
		return nil
	})
}

func TestPipelineCleanupEmpty(t *testing.T) {
	p := middleware.NewPipeline(nil)
	err := p.Cleanup()
	if err != nil {
		t.Errorf("Cleanup on empty pipeline should not error, got %v", err)
	}
}

func TestValidationErrorFields(t *testing.T) {
	ve := middleware.ValidationError{
		Field:   "email",
		Message: "invalid email format",
		Code:    "INVALID_EMAIL",
	}
	if ve.Field != "email" {
		t.Errorf("unexpected Field: %q", ve.Field)
	}
	if ve.Message != "invalid email format" {
		t.Errorf("unexpected Message: %q", ve.Message)
	}
	if ve.Code != "INVALID_EMAIL" {
		t.Errorf("unexpected Code: %q", ve.Code)
	}
}

func TestTransformRuleFields(t *testing.T) {
	rule := middleware.TransformRule{
		Pattern:     "/api/*",
		Transformer: "json-to-xml",
		Condition:   "content-type == application/json",
		Priority:    10,
	}
	if rule.Priority != 10 {
		t.Errorf("unexpected Priority: %d", rule.Priority)
	}
	if rule.Pattern != "/api/*" {
		t.Errorf("unexpected Pattern: %q", rule.Pattern)
	}
}
