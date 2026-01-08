package cache

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type CacheEngine struct {
	stores      map[string]Store
	policies    map[string]Policy
	keyGen      KeyGenerator
	mutex       sync.RWMutex
	metrics     *Metrics
	config      Config
	defaultTTL  time.Duration
}

type Store interface {
	Get(ctx context.Context, key string) (*CacheEntry, error)
	Set(ctx context.Context, key string, entry *CacheEntry, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
	Exists(ctx context.Context, key string) (bool, error)
	Keys(ctx context.Context, pattern string) ([]string, error)
	Size(ctx context.Context) (int64, error)
	Stats(ctx context.Context) (StoreStats, error)
}

type Policy interface {
	ShouldCache(req *http.Request, resp *http.Response) bool
	GetTTL(req *http.Request, resp *http.Response) time.Duration
	GenerateKey(req *http.Request) string
	ShouldInvalidate(req *http.Request) bool
	GetTags(req *http.Request, resp *http.Response) []string
}

type KeyGenerator interface {
	Generate(req *http.Request) string
	GenerateWithParams(method, url string, headers map[string]string, body []byte) string
}

type CacheEntry struct {
	Key          string            `json:"key"`
	Value        []byte            `json:"value"`
	Headers      map[string]string `json:"headers"`
	StatusCode   int               `json:"status_code"`
	ContentType  string            `json:"content_type"`
	Tags         []string          `json:"tags"`
	CreatedAt    time.Time         `json:"created_at"`
	ExpiresAt    time.Time         `json:"expires_at"`
	AccessCount  int64             `json:"access_count"`
	LastAccessed time.Time         `json:"last_accessed"`
	Size         int64             `json:"size"`
	Compressed   bool              `json:"compressed"`
}

func (ce *CacheEntry) IsExpired() bool {
	return time.Now().After(ce.ExpiresAt)
}

func (ce *CacheEntry) IsStale(staleTime time.Duration) bool {
	return time.Since(ce.LastAccessed) > staleTime
}

func (ce *CacheEntry) Touch() {
	ce.LastAccessed = time.Now()
	ce.AccessCount++
}

type Config struct {
	DefaultStore   string
	DefaultPolicy  string
	DefaultTTL     time.Duration
	MaxSize        int64
	CompressionEnabled bool
	CompressionMinSize int64
	StaleWhileRevalidate bool
	StaleIfError     bool
	PurgeEnabled     bool
	MetricsEnabled   bool
}

type Metrics struct {
	Hits                uint64
	Misses              uint64
	Sets                uint64
	Deletes             uint64
	Evictions           uint64
	Errors              uint64
	TotalRequests       uint64
	TotalResponseTime   time.Duration
	AverageResponseTime time.Duration
	mutex               sync.RWMutex
}

type StoreStats struct {
	Size        int64     `json:"size"`
	KeyCount    int64     `json:"key_count"`
	HitRate     float64   `json:"hit_rate"`
	Memory      int64     `json:"memory_usage"`
	LastEviction time.Time `json:"last_eviction"`
}

func NewCacheEngine(config Config) *CacheEngine {
	ce := &CacheEngine{
		stores:     make(map[string]Store),
		policies:   make(map[string]Policy),
		keyGen:     NewDefaultKeyGenerator(),
		metrics:    &Metrics{},
		config:     config,
		defaultTTL: config.DefaultTTL,
	}

	if ce.defaultTTL == 0 {
		ce.defaultTTL = 5 * time.Minute
	}

	return ce
}

func (ce *CacheEngine) RegisterStore(name string, store Store) {
	ce.mutex.Lock()
	defer ce.mutex.Unlock()
	ce.stores[name] = store
}

func (ce *CacheEngine) RegisterPolicy(name string, policy Policy) {
	ce.mutex.Lock()
	defer ce.mutex.Unlock()
	ce.policies[name] = policy
}

func (ce *CacheEngine) Get(ctx context.Context, req *http.Request) (*CacheEntry, error) {
	start := time.Now()
	defer func() {
		ce.updateMetrics(time.Since(start))
	}()

	store := ce.getStore()
	policy := ce.getPolicy()
	key := policy.GenerateKey(req)

	entry, err := store.Get(ctx, key)
	if err != nil {
		ce.metrics.recordMiss()
		return nil, err
	}

	if entry == nil {
		ce.metrics.recordMiss()
		return nil, nil
	}

	if entry.IsExpired() {
		ce.metrics.recordMiss()
		store.Delete(ctx, key)
		return nil, nil
	}

	entry.Touch()
	store.Set(ctx, key, entry, time.Until(entry.ExpiresAt))
	ce.metrics.recordHit()

	return entry, nil
}

func (ce *CacheEngine) Set(ctx context.Context, req *http.Request, resp *http.Response, body []byte) error {
	store := ce.getStore()
	policy := ce.getPolicy()

	if !policy.ShouldCache(req, resp) {
		return nil
	}

	key := policy.GenerateKey(req)
	ttl := policy.GetTTL(req, resp)
	if ttl == 0 {
		ttl = ce.defaultTTL
	}

	headers := make(map[string]string)
	for name, values := range resp.Header {
		headers[name] = strings.Join(values, ", ")
	}

	entry := &CacheEntry{
		Key:          key,
		Value:        body,
		Headers:      headers,
		StatusCode:   resp.StatusCode,
		ContentType:  resp.Header.Get("Content-Type"),
		Tags:         policy.GetTags(req, resp),
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(ttl),
		AccessCount:  1,
		LastAccessed: time.Now(),
		Size:         int64(len(body)),
		Compressed:   false,
	}

	if ce.config.CompressionEnabled && entry.Size > ce.config.CompressionMinSize {
		compressed, err := ce.compress(body)
		if err == nil && len(compressed) < len(body) {
			entry.Value = compressed
			entry.Compressed = true
			entry.Size = int64(len(compressed))
		}
	}

	err := store.Set(ctx, key, entry, ttl)
	if err != nil {
		ce.metrics.recordError()
		return err
	}

	ce.metrics.recordSet()
	return nil
}

func (ce *CacheEngine) Delete(ctx context.Context, key string) error {
	store := ce.getStore()
	err := store.Delete(ctx, key)
	if err != nil {
		ce.metrics.recordError()
		return err
	}
	ce.metrics.recordDelete()
	return nil
}

func (ce *CacheEngine) DeleteByTags(ctx context.Context, tags []string) error {
	store := ce.getStore()

	keys, err := store.Keys(ctx, "*")
	if err != nil {
		return err
	}

	var keysToDelete []string
	for _, key := range keys {
		entry, err := store.Get(ctx, key)
		if err != nil || entry == nil {
			continue
		}

		for _, tag := range tags {
			for _, entryTag := range entry.Tags {
				if tag == entryTag {
					keysToDelete = append(keysToDelete, key)
					break
				}
			}
		}
	}

	for _, key := range keysToDelete {
		if err := store.Delete(ctx, key); err != nil {
			ce.metrics.recordError()
		} else {
			ce.metrics.recordDelete()
		}
	}

	return nil
}

func (ce *CacheEngine) Clear(ctx context.Context) error {
	store := ce.getStore()
	return store.Clear(ctx)
}

func (ce *CacheEngine) InvalidateByPattern(ctx context.Context, pattern string) error {
	store := ce.getStore()

	keys, err := store.Keys(ctx, pattern)
	if err != nil {
		return err
	}

	for _, key := range keys {
		if err := store.Delete(ctx, key); err != nil {
			ce.metrics.recordError()
		} else {
			ce.metrics.recordDelete()
		}
	}

	return nil
}

func (ce *CacheEngine) Exists(ctx context.Context, req *http.Request) (bool, error) {
	store := ce.getStore()
	policy := ce.getPolicy()
	key := policy.GenerateKey(req)
	return store.Exists(ctx, key)
}

func (ce *CacheEngine) GetStats(ctx context.Context) (StoreStats, error) {
	store := ce.getStore()
	return store.Stats(ctx)
}

func (ce *CacheEngine) GetMetrics() *Metrics {
	ce.metrics.mutex.RLock()
	defer ce.metrics.mutex.RUnlock()

	// Return a copy without the mutex to avoid copying a lock value
	return &Metrics{
		Hits:                ce.metrics.Hits,
		Misses:              ce.metrics.Misses,
		Sets:                ce.metrics.Sets,
		Deletes:             ce.metrics.Deletes,
		Evictions:           ce.metrics.Evictions,
		Errors:              ce.metrics.Errors,
		TotalRequests:       ce.metrics.TotalRequests,
		TotalResponseTime:   ce.metrics.TotalResponseTime,
		AverageResponseTime: ce.metrics.AverageResponseTime,
	}
}

func (ce *CacheEngine) getStore() Store {
	ce.mutex.RLock()
	defer ce.mutex.RUnlock()

	if store, exists := ce.stores[ce.config.DefaultStore]; exists {
		return store
	}

	for _, store := range ce.stores {
		return store
	}

	return NewMemoryStore(MemoryStoreConfig{})
}

func (ce *CacheEngine) getPolicy() Policy {
	ce.mutex.RLock()
	defer ce.mutex.RUnlock()

	if policy, exists := ce.policies[ce.config.DefaultPolicy]; exists {
		return policy
	}

	for _, policy := range ce.policies {
		return policy
	}

	return NewDefaultPolicy(DefaultPolicyConfig{})
}

func (ce *CacheEngine) compress(data []byte) ([]byte, error) {
	return data, nil
}

func (ce *CacheEngine) updateMetrics(duration time.Duration) {
	ce.metrics.mutex.Lock()
	defer ce.metrics.mutex.Unlock()

	ce.metrics.TotalRequests++
	ce.metrics.TotalResponseTime += duration
	ce.metrics.AverageResponseTime = ce.metrics.TotalResponseTime / time.Duration(ce.metrics.TotalRequests)
}

func (m *Metrics) recordHit() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.Hits++
}

func (m *Metrics) recordMiss() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.Misses++
}

func (m *Metrics) recordSet() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.Sets++
}

func (m *Metrics) recordDelete() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.Deletes++
}

func (m *Metrics) recordEviction() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.Evictions++
}

func (m *Metrics) recordError() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.Errors++
}

func (m *Metrics) GetHitRate() float64 {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	total := m.Hits + m.Misses
	if total == 0 {
		return 0.0
	}
	return float64(m.Hits) / float64(total) * 100.0
}

type DefaultKeyGenerator struct{}

func NewDefaultKeyGenerator() *DefaultKeyGenerator {
	return &DefaultKeyGenerator{}
}

func (kg *DefaultKeyGenerator) Generate(req *http.Request) string {
	return kg.GenerateWithParams(
		req.Method,
		req.URL.String(),
		kg.extractHeaders(req),
		nil,
	)
}

func (kg *DefaultKeyGenerator) GenerateWithParams(method, urlStr string, headers map[string]string, body []byte) string {
	hasher := md5.New()

	hasher.Write([]byte(method))
	hasher.Write([]byte(urlStr))

	var headerKeys []string
	for key := range headers {
		headerKeys = append(headerKeys, key)
	}
	sort.Strings(headerKeys)

	for _, key := range headerKeys {
		hasher.Write([]byte(key))
		hasher.Write([]byte(headers[key]))
	}

	if body != nil {
		hasher.Write(body)
	}

	return hex.EncodeToString(hasher.Sum(nil))
}

func (kg *DefaultKeyGenerator) extractHeaders(req *http.Request) map[string]string {
	headers := make(map[string]string)
	
	relevantHeaders := []string{
		"Accept",
		"Accept-Encoding",
		"Accept-Language",
		"Authorization",
		"Cache-Control",
		"User-Agent",
	}

	for _, header := range relevantHeaders {
		if value := req.Header.Get(header); value != "" {
			headers[header] = value
		}
	}

	return headers
}

type CacheKeyBuilder struct {
	components []string
	params     map[string]string
}

func NewCacheKeyBuilder() *CacheKeyBuilder {
	return &CacheKeyBuilder{
		components: make([]string, 0),
		params:     make(map[string]string),
	}
}

func (ckb *CacheKeyBuilder) AddComponent(component string) *CacheKeyBuilder {
	ckb.components = append(ckb.components, component)
	return ckb
}

func (ckb *CacheKeyBuilder) AddParam(key, value string) *CacheKeyBuilder {
	ckb.params[key] = value
	return ckb
}

func (ckb *CacheKeyBuilder) AddURL(u *url.URL) *CacheKeyBuilder {
	ckb.AddComponent(u.Path)
	
	for key, values := range u.Query() {
		if len(values) > 0 {
			ckb.AddParam(key, values[0])
		}
	}
	
	return ckb
}

func (ckb *CacheKeyBuilder) AddHeaders(headers http.Header, keys []string) *CacheKeyBuilder {
	for _, key := range keys {
		if value := headers.Get(key); value != "" {
			ckb.AddParam(strings.ToLower(key), value)
		}
	}
	return ckb
}

func (ckb *CacheKeyBuilder) Build() string {
	var parts []string
	
	for _, component := range ckb.components {
		parts = append(parts, component)
	}
	
	var paramKeys []string
	for key := range ckb.params {
		paramKeys = append(paramKeys, key)
	}
	sort.Strings(paramKeys)
	
	for _, key := range paramKeys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, ckb.params[key]))
	}
	
	hasher := md5.New()
	hasher.Write([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(hasher.Sum(nil))
}

type CacheResponse struct {
	StatusCode  int               `json:"status_code"`
	Headers     map[string]string `json:"headers"`
	Body        []byte            `json:"body"`
	Cached      bool              `json:"cached"`
	CacheKey    string            `json:"cache_key"`
	Age         time.Duration     `json:"age"`
	TTL         time.Duration     `json:"ttl"`
	Tags        []string          `json:"tags"`
	Compressed  bool              `json:"compressed"`
}

func (cr *CacheResponse) ToHTTPResponse() *http.Response {
	resp := &http.Response{
		StatusCode: cr.StatusCode,
		Header:     make(http.Header),
		Body:       http.NoBody,
	}

	for key, value := range cr.Headers {
		resp.Header.Set(key, value)
	}

	if cr.Cached {
		resp.Header.Set("X-Cache", "HIT")
		resp.Header.Set("X-Cache-Key", cr.CacheKey)
		resp.Header.Set("Age", strconv.Itoa(int(cr.Age.Seconds())))
	} else {
		resp.Header.Set("X-Cache", "MISS")
	}

	return resp
}

func NewCacheResponse(entry *CacheEntry) *CacheResponse {
	age := time.Since(entry.CreatedAt)
	ttl := time.Until(entry.ExpiresAt)

	return &CacheResponse{
		StatusCode: entry.StatusCode,
		Headers:    entry.Headers,
		Body:       entry.Value,
		Cached:     true,
		CacheKey:   entry.Key,
		Age:        age,
		TTL:        ttl,
		Tags:       entry.Tags,
		Compressed: entry.Compressed,
	}
}

func DefaultCacheConfig() Config {
	return Config{
		DefaultStore:         "memory",
		DefaultPolicy:        "default",
		DefaultTTL:           5 * time.Minute,
		MaxSize:              100 * 1024 * 1024, // 100MB
		CompressionEnabled:   true,
		CompressionMinSize:   1024, // 1KB
		StaleWhileRevalidate: true,
		StaleIfError:         true,
		PurgeEnabled:         true,
		MetricsEnabled:       true,
	}
}