package cache

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/MarchProxy/proxy/internal/middleware"
)

type CacheMiddleware struct {
	engine     *CacheEngine
	config     CacheMiddlewareConfig
	enabled    bool
	statistics *MiddlewareStatistics
}

type CacheMiddlewareConfig struct {
	EnableResponseCaching bool
	EnableRequestCaching  bool
	BypassOnError        bool
	StaleWhileRevalidate bool
	StaleIfError         bool
	GracePeriod          time.Duration
	MaxStaleAge          time.Duration
	VaryHeaders          []string
	IgnorePaths          []string
	IgnoreQueryParams    []string
	CompressResponses    bool
	CompressionMinSize   int64
	ServeStaleTimeout    time.Duration
}

type MiddlewareStatistics struct {
	CacheHits              uint64
	CacheMisses            uint64
	CacheErrors            uint64
	StaleServed            uint64
	BackgroundRefreshes    uint64
	CompressionSaved       int64
	TotalCachedSize        int64
	AverageResponseTime    time.Duration
	CacheHitRatio          float64
}

func NewCacheMiddleware(engine *CacheEngine, config CacheMiddlewareConfig) *CacheMiddleware {
	return &CacheMiddleware{
		engine:     engine,
		config:     config,
		enabled:    true,
		statistics: &MiddlewareStatistics{},
	}
}

func (cm *CacheMiddleware) Name() string {
	return "cache"
}

func (cm *CacheMiddleware) Priority() int {
	return 100
}

func (cm *CacheMiddleware) ProcessRequest(req *http.Request, ctx *middleware.MiddlewareContext) error {
	if !cm.enabled || !cm.shouldProcessRequest(req) {
		return nil
	}

	start := time.Now()
	defer func() {
		cm.updateResponseTime(time.Since(start))
	}()

	if cm.config.EnableRequestCaching {
		cachedEntry, err := cm.engine.Get(req.Context(), req)
		if err != nil {
			cm.statistics.CacheErrors++
			if !cm.config.BypassOnError {
				return fmt.Errorf("cache get error: %w", err)
			}
			return nil
		}

		if cachedEntry != nil {
			cm.statistics.CacheHits++
			
			if cachedEntry.IsExpired() && cm.config.StaleWhileRevalidate {
				ctx.SetData("cache_stale_entry", cachedEntry)
				ctx.SetData("cache_needs_refresh", true)
				return nil
			}

			response := cm.createResponseFromCache(cachedEntry)
			ctx.SetData("cache_hit", true)
			ctx.SetData("cache_response", response)
			ctx.SetData("cache_entry", cachedEntry)
			
			ctx.StopProcessing()
			return nil
		}

		cm.statistics.CacheMisses++
		ctx.SetData("cache_miss", true)
	}

	return nil
}

func (cm *CacheMiddleware) ProcessResponse(resp *http.Response, ctx *middleware.MiddlewareContext) error {
	if !cm.enabled || !cm.config.EnableResponseCaching {
		return nil
	}

	if ctx.HasData("cache_hit") {
		return nil
	}

	if !cm.shouldCacheResponse(resp) {
		return nil
	}

	body, err := cm.readAndRestoreBody(resp)
	if err != nil {
		cm.statistics.CacheErrors++
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if cm.config.CompressResponses && len(body) > int(cm.config.CompressionMinSize) {
		compressedBody, err := cm.compressBody(body)
		if err == nil && len(compressedBody) < len(body) {
			savedBytes := int64(len(body) - len(compressedBody))
			cm.statistics.CompressionSaved += savedBytes
			body = compressedBody
		}
	}

	req := ctx.Request
	if req == nil {
		return fmt.Errorf("no request found in context")
	}

	err = cm.engine.Set(req.Context(), req, resp, body)
	if err != nil {
		cm.statistics.CacheErrors++
		if !cm.config.BypassOnError {
			return fmt.Errorf("cache set error: %w", err)
		}
	}

	cm.statistics.TotalCachedSize += int64(len(body))

	if ctx.HasData("cache_needs_refresh") {
		cm.statistics.BackgroundRefreshes++
		ctx.SetData("cache_refreshed", true)
	}

	return nil
}

func (cm *CacheMiddleware) ProcessError(err error, ctx *middleware.MiddlewareContext) error {
	if !cm.enabled {
		return err
	}

	if cm.config.StaleIfError {
		if staleEntry, hasStale := ctx.GetData("cache_stale_entry").(*CacheEntry); hasStale {
			if time.Since(staleEntry.ExpiresAt) <= cm.config.MaxStaleAge {
				response := cm.createResponseFromCache(staleEntry)
				ctx.SetData("cache_stale_served", true)
				ctx.SetData("cache_response", response)
				cm.statistics.StaleServed++
				return nil
			}
		}
	}

	return err
}

func (cm *CacheMiddleware) shouldProcessRequest(req *http.Request) bool {
	if req.Method != "GET" && req.Method != "HEAD" {
		return false
	}

	for _, ignorePath := range cm.config.IgnorePaths {
		if strings.Contains(req.URL.Path, ignorePath) {
			return false
		}
	}

	cacheControl := req.Header.Get("Cache-Control")
	if strings.Contains(cacheControl, "no-cache") || strings.Contains(cacheControl, "no-store") {
		return false
	}

	return true
}

func (cm *CacheMiddleware) shouldCacheResponse(resp *http.Response) bool {
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return false
	}

	cacheControl := resp.Header.Get("Cache-Control")
	if strings.Contains(cacheControl, "no-cache") || 
	   strings.Contains(cacheControl, "no-store") || 
	   strings.Contains(cacheControl, "private") {
		return false
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") || 
	   strings.Contains(contentType, "application/stream") {
		return false
	}

	return true
}

func (cm *CacheMiddleware) createResponseFromCache(entry *CacheEntry) *http.Response {
	resp := &http.Response{
		StatusCode: entry.StatusCode,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(entry.Value)),
	}

	for key, value := range entry.Headers {
		resp.Header.Set(key, value)
	}

	resp.Header.Set("X-Cache", "HIT")
	resp.Header.Set("X-Cache-Key", entry.Key)
	resp.Header.Set("Age", fmt.Sprintf("%.0f", time.Since(entry.CreatedAt).Seconds()))

	if entry.Compressed {
		resp.Header.Set("X-Cache-Compressed", "true")
	}

	for _, tag := range entry.Tags {
		resp.Header.Add("X-Cache-Tags", tag)
	}

	return resp
}

func (cm *CacheMiddleware) readAndRestoreBody(resp *http.Response) ([]byte, error) {
	if resp.Body == nil {
		return nil, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(body))

	return body, nil
}

func (cm *CacheMiddleware) compressBody(body []byte) ([]byte, error) {
	return body, nil
}

func (cm *CacheMiddleware) updateResponseTime(duration time.Duration) {
	totalRequests := cm.statistics.CacheHits + cm.statistics.CacheMisses
	if totalRequests > 0 {
		cm.statistics.AverageResponseTime = 
			(cm.statistics.AverageResponseTime*time.Duration(totalRequests-1) + duration) / 
			time.Duration(totalRequests)
	} else {
		cm.statistics.AverageResponseTime = duration
	}

	cm.statistics.CacheHitRatio = float64(cm.statistics.CacheHits) / float64(totalRequests) * 100
}

func (cm *CacheMiddleware) Enabled() bool {
	return cm.enabled
}

func (cm *CacheMiddleware) Enable() {
	cm.enabled = true
}

func (cm *CacheMiddleware) Disable() {
	cm.enabled = false
}

func (cm *CacheMiddleware) GetStatistics() *MiddlewareStatistics {
	return cm.statistics
}

func (cm *CacheMiddleware) ResetStatistics() {
	cm.statistics = &MiddlewareStatistics{}
}

func (cm *CacheMiddleware) InvalidateByTags(ctx context.Context, tags []string) error {
	return cm.engine.DeleteByTags(ctx, tags)
}

func (cm *CacheMiddleware) InvalidateByPattern(ctx context.Context, pattern string) error {
	return cm.engine.InvalidateByPattern(ctx, pattern)
}

func (cm *CacheMiddleware) Clear(ctx context.Context) error {
	return cm.engine.Clear(ctx)
}

func (cm *CacheMiddleware) GetEngine() *CacheEngine {
	return cm.engine
}

type CacheHandler struct {
	middleware *CacheMiddleware
}

func NewCacheHandler(middleware *CacheMiddleware) *CacheHandler {
	return &CacheHandler{
		middleware: middleware,
	}
}

func (ch *CacheHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		ch.handleInvalidation(w, r)
	case "DELETE":
		ch.handleClear(w, r)
	case "GET":
		ch.handleStats(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (ch *CacheHandler) handleInvalidation(w http.ResponseWriter, r *http.Request) {
	action := r.URL.Query().Get("action")
	
	switch action {
	case "tags":
		tags := r.URL.Query()["tag"]
		if len(tags) == 0 {
			http.Error(w, "No tags specified", http.StatusBadRequest)
			return
		}
		
		err := ch.middleware.InvalidateByTags(r.Context(), tags)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalidation failed: %v", err), http.StatusInternalServerError)
			return
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("Invalidated entries with tags: %v", tags)))
		
	case "pattern":
		pattern := r.URL.Query().Get("pattern")
		if pattern == "" {
			http.Error(w, "No pattern specified", http.StatusBadRequest)
			return
		}
		
		err := ch.middleware.InvalidateByPattern(r.Context(), pattern)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalidation failed: %v", err), http.StatusInternalServerError)
			return
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("Invalidated entries matching pattern: %s", pattern)))
		
	default:
		http.Error(w, "Invalid action. Use 'tags' or 'pattern'", http.StatusBadRequest)
	}
}

func (ch *CacheHandler) handleClear(w http.ResponseWriter, r *http.Request) {
	err := ch.middleware.Clear(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Clear failed: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Cache cleared successfully"))
}

func (ch *CacheHandler) handleStats(w http.ResponseWriter, r *http.Request) {
	stats := ch.middleware.GetStatistics()
	
	response := fmt.Sprintf(`{
		"cache_hits": %d,
		"cache_misses": %d,
		"cache_errors": %d,
		"stale_served": %d,
		"background_refreshes": %d,
		"compression_saved_bytes": %d,
		"total_cached_size_bytes": %d,
		"average_response_time_ms": %.2f,
		"cache_hit_ratio_percent": %.2f
	}`,
		stats.CacheHits,
		stats.CacheMisses,
		stats.CacheErrors,
		stats.StaleServed,
		stats.BackgroundRefreshes,
		stats.CompressionSaved,
		stats.TotalCachedSize,
		float64(stats.AverageResponseTime.Nanoseconds())/1000000.0,
		stats.CacheHitRatio,
	)
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(response))
}

func DefaultCacheMiddlewareConfig() CacheMiddlewareConfig {
	return CacheMiddlewareConfig{
		EnableResponseCaching: true,
		EnableRequestCaching:  true,
		BypassOnError:        true,
		StaleWhileRevalidate: true,
		StaleIfError:         true,
		GracePeriod:          30 * time.Second,
		MaxStaleAge:          1 * time.Hour,
		VaryHeaders:          []string{"Accept", "Accept-Encoding", "Authorization"},
		IgnorePaths:          []string{"/health", "/metrics", "/admin"},
		IgnoreQueryParams:    []string{"_t", "timestamp", "cb"},
		CompressResponses:    true,
		CompressionMinSize:   1024,
		ServeStaleTimeout:    5 * time.Second,
	}
}

type CacheWarmup struct {
	middleware *CacheMiddleware
	client     *http.Client
	urls       []string
	config     WarmupConfig
}

type WarmupConfig struct {
	Concurrency      int
	RequestTimeout   time.Duration
	FailureThreshold float64
	RetryCount       int
	RetryDelay       time.Duration
}

func NewCacheWarmup(middleware *CacheMiddleware, config WarmupConfig) *CacheWarmup {
	return &CacheWarmup{
		middleware: middleware,
		client: &http.Client{
			Timeout: config.RequestTimeout,
		},
		config: config,
	}
}

func (cw *CacheWarmup) WarmupURLs(ctx context.Context, urls []string) error {
	semaphore := make(chan struct{}, cw.config.Concurrency)
	results := make(chan error, len(urls))

	for _, url := range urls {
		go func(u string) {
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			err := cw.warmupURL(ctx, u)
			results <- err
		}(url)
	}

	var errors []error
	for i := 0; i < len(urls); i++ {
		if err := <-results; err != nil {
			errors = append(errors, err)
		}
	}

	failureRate := float64(len(errors)) / float64(len(urls))
	if failureRate > cw.config.FailureThreshold {
		return fmt.Errorf("warmup failed: %d/%d URLs failed (%.2f%% > %.2f%% threshold)", 
			len(errors), len(urls), failureRate*100, cw.config.FailureThreshold*100)
	}

	return nil
}

func (cw *CacheWarmup) warmupURL(ctx context.Context, url string) error {
	var lastErr error
	
	for attempt := 0; attempt <= cw.config.RetryCount; attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(cw.config.RetryDelay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			lastErr = err
			continue
		}

		resp, err := cw.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			return nil
		}

		lastErr = fmt.Errorf("HTTP %d for URL %s", resp.StatusCode, url)
	}

	return lastErr
}