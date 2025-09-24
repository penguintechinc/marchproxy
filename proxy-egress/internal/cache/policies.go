package cache

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

type DefaultPolicy struct {
	config DefaultPolicyConfig
}

type DefaultPolicyConfig struct {
	DefaultTTL           time.Duration
	MaxTTL               time.Duration
	CacheableStatusCodes []int
	CacheableMethods     []string
	IgnoreHeaders        []string
	RequiredHeaders      []string
	VaryHeaders          []string
	RespectCacheControl  bool
	TagExtractors        []TagExtractor
}

type TagExtractor func(req *http.Request, resp *http.Response) []string

type ConditionalPolicy struct {
	conditions []CacheCondition
	basePolicy Policy
}

type CacheCondition struct {
	Matcher ConditionMatcher
	Policy  Policy
}

type ConditionMatcher func(req *http.Request, resp *http.Response) bool

type HeaderBasedPolicy struct {
	headerRules map[string]HeaderRule
	defaultTTL  time.Duration
}

type HeaderRule struct {
	TTL           time.Duration
	ShouldCache   bool
	Tags          []string
	VaryOn        []string
}

type PathBasedPolicy struct {
	pathRules  map[string]PathRule
	patterns   []PathPattern
	defaultTTL time.Duration
}

type PathRule struct {
	TTL         time.Duration
	ShouldCache bool
	Tags        []string
	Methods     []string
}

type PathPattern struct {
	Pattern string
	Rule    PathRule
}

type TimeBasedPolicy struct {
	schedules []CacheSchedule
	basePolicy Policy
}

type CacheSchedule struct {
	StartTime   time.Time
	EndTime     time.Time
	DaysOfWeek  []time.Weekday
	TTL         time.Duration
	ShouldCache bool
}

func NewDefaultPolicy(config DefaultPolicyConfig) *DefaultPolicy {
	if config.DefaultTTL == 0 {
		config.DefaultTTL = 5 * time.Minute
	}
	if config.MaxTTL == 0 {
		config.MaxTTL = 24 * time.Hour
	}
	if len(config.CacheableStatusCodes) == 0 {
		config.CacheableStatusCodes = []int{200, 203, 206, 300, 301, 410}
	}
	if len(config.CacheableMethods) == 0 {
		config.CacheableMethods = []string{"GET", "HEAD"}
	}

	return &DefaultPolicy{config: config}
}

func (dp *DefaultPolicy) ShouldCache(req *http.Request, resp *http.Response) bool {
	if !dp.isMethodCacheable(req.Method) {
		return false
	}

	if !dp.isStatusCacheable(resp.StatusCode) {
		return false
	}

	if dp.config.RespectCacheControl {
		cacheControl := resp.Header.Get("Cache-Control")
		if strings.Contains(cacheControl, "no-cache") || 
		   strings.Contains(cacheControl, "no-store") || 
		   strings.Contains(cacheControl, "private") {
			return false
		}
	}

	for _, header := range dp.config.RequiredHeaders {
		if resp.Header.Get(header) == "" {
			return false
		}
	}

	return true
}

func (dp *DefaultPolicy) GetTTL(req *http.Request, resp *http.Response) time.Duration {
	if dp.config.RespectCacheControl {
		if ttl := dp.extractTTLFromCacheControl(resp); ttl > 0 {
			if ttl > dp.config.MaxTTL {
				return dp.config.MaxTTL
			}
			return ttl
		}
	}

	if expires := resp.Header.Get("Expires"); expires != "" {
		if expireTime, err := time.Parse(time.RFC1123, expires); err == nil {
			ttl := time.Until(expireTime)
			if ttl > 0 && ttl <= dp.config.MaxTTL {
				return ttl
			}
		}
	}

	return dp.config.DefaultTTL
}

func (dp *DefaultPolicy) GenerateKey(req *http.Request) string {
	builder := NewCacheKeyBuilder()
	builder.AddComponent(req.Method)
	builder.AddURL(req.URL)
	
	if len(dp.config.VaryHeaders) > 0 {
		builder.AddHeaders(req.Header, dp.config.VaryHeaders)
	}

	return builder.Build()
}

func (dp *DefaultPolicy) ShouldInvalidate(req *http.Request) bool {
	invalidatingMethods := []string{"POST", "PUT", "PATCH", "DELETE"}
	for _, method := range invalidatingMethods {
		if req.Method == method {
			return true
		}
	}
	return false
}

func (dp *DefaultPolicy) GetTags(req *http.Request, resp *http.Response) []string {
	var tags []string

	for _, extractor := range dp.config.TagExtractors {
		tags = append(tags, extractor(req, resp)...)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "" {
		tags = append(tags, "content-type:"+strings.Split(contentType, ";")[0])
	}

	tags = append(tags, "method:"+req.Method)
	tags = append(tags, "status:"+strconv.Itoa(resp.StatusCode))

	return tags
}

func (dp *DefaultPolicy) isMethodCacheable(method string) bool {
	for _, m := range dp.config.CacheableMethods {
		if m == method {
			return true
		}
	}
	return false
}

func (dp *DefaultPolicy) isStatusCacheable(status int) bool {
	for _, s := range dp.config.CacheableStatusCodes {
		if s == status {
			return true
		}
	}
	return false
}

func (dp *DefaultPolicy) extractTTLFromCacheControl(resp *http.Response) time.Duration {
	cacheControl := resp.Header.Get("Cache-Control")
	if cacheControl == "" {
		return 0
	}

	directives := strings.Split(cacheControl, ",")
	for _, directive := range directives {
		directive = strings.TrimSpace(directive)
		if strings.HasPrefix(directive, "max-age=") {
			if seconds, err := strconv.Atoi(strings.TrimPrefix(directive, "max-age=")); err == nil {
				return time.Duration(seconds) * time.Second
			}
		}
		if strings.HasPrefix(directive, "s-maxage=") {
			if seconds, err := strconv.Atoi(strings.TrimPrefix(directive, "s-maxage=")); err == nil {
				return time.Duration(seconds) * time.Second
			}
		}
	}

	return 0
}

func NewConditionalPolicy(conditions []CacheCondition, basePolicy Policy) *ConditionalPolicy {
	return &ConditionalPolicy{
		conditions: conditions,
		basePolicy: basePolicy,
	}
}

func (cp *ConditionalPolicy) ShouldCache(req *http.Request, resp *http.Response) bool {
	policy := cp.selectPolicy(req, resp)
	return policy.ShouldCache(req, resp)
}

func (cp *ConditionalPolicy) GetTTL(req *http.Request, resp *http.Response) time.Duration {
	policy := cp.selectPolicy(req, resp)
	return policy.GetTTL(req, resp)
}

func (cp *ConditionalPolicy) GenerateKey(req *http.Request) string {
	return cp.basePolicy.GenerateKey(req)
}

func (cp *ConditionalPolicy) ShouldInvalidate(req *http.Request) bool {
	return cp.basePolicy.ShouldInvalidate(req)
}

func (cp *ConditionalPolicy) GetTags(req *http.Request, resp *http.Response) []string {
	policy := cp.selectPolicy(req, resp)
	return policy.GetTags(req, resp)
}

func (cp *ConditionalPolicy) selectPolicy(req *http.Request, resp *http.Response) Policy {
	for _, condition := range cp.conditions {
		if condition.Matcher(req, resp) {
			return condition.Policy
		}
	}
	return cp.basePolicy
}

func NewHeaderBasedPolicy(rules map[string]HeaderRule, defaultTTL time.Duration) *HeaderBasedPolicy {
	return &HeaderBasedPolicy{
		headerRules: rules,
		defaultTTL:  defaultTTL,
	}
}

func (hbp *HeaderBasedPolicy) ShouldCache(req *http.Request, resp *http.Response) bool {
	for header, rule := range hbp.headerRules {
		if resp.Header.Get(header) != "" {
			return rule.ShouldCache
		}
	}
	return true
}

func (hbp *HeaderBasedPolicy) GetTTL(req *http.Request, resp *http.Response) time.Duration {
	for header, rule := range hbp.headerRules {
		if resp.Header.Get(header) != "" {
			return rule.TTL
		}
	}
	return hbp.defaultTTL
}

func (hbp *HeaderBasedPolicy) GenerateKey(req *http.Request) string {
	builder := NewCacheKeyBuilder()
	builder.AddComponent(req.Method)
	builder.AddURL(req.URL)

	for header, rule := range hbp.headerRules {
		if len(rule.VaryOn) > 0 {
			builder.AddHeaders(req.Header, rule.VaryOn)
		}
		if value := req.Header.Get(header); value != "" {
			builder.AddParam("header_"+strings.ToLower(header), value)
		}
	}

	return builder.Build()
}

func (hbp *HeaderBasedPolicy) ShouldInvalidate(req *http.Request) bool {
	return req.Method != "GET" && req.Method != "HEAD"
}

func (hbp *HeaderBasedPolicy) GetTags(req *http.Request, resp *http.Response) []string {
	var tags []string

	for header, rule := range hbp.headerRules {
		if resp.Header.Get(header) != "" {
			tags = append(tags, rule.Tags...)
			tags = append(tags, "header:"+strings.ToLower(header))
		}
	}

	return tags
}

func NewPathBasedPolicy(rules map[string]PathRule, patterns []PathPattern, defaultTTL time.Duration) *PathBasedPolicy {
	return &PathBasedPolicy{
		pathRules:  rules,
		patterns:   patterns,
		defaultTTL: defaultTTL,
	}
}

func (pbp *PathBasedPolicy) ShouldCache(req *http.Request, resp *http.Response) bool {
	rule := pbp.findRule(req.URL.Path, req.Method)
	return rule.ShouldCache
}

func (pbp *PathBasedPolicy) GetTTL(req *http.Request, resp *http.Response) time.Duration {
	rule := pbp.findRule(req.URL.Path, req.Method)
	if rule.TTL > 0 {
		return rule.TTL
	}
	return pbp.defaultTTL
}

func (pbp *PathBasedPolicy) GenerateKey(req *http.Request) string {
	builder := NewCacheKeyBuilder()
	builder.AddComponent(req.Method)
	builder.AddURL(req.URL)
	return builder.Build()
}

func (pbp *PathBasedPolicy) ShouldInvalidate(req *http.Request) bool {
	return req.Method != "GET" && req.Method != "HEAD"
}

func (pbp *PathBasedPolicy) GetTags(req *http.Request, resp *http.Response) []string {
	rule := pbp.findRule(req.URL.Path, req.Method)
	tags := make([]string, len(rule.Tags))
	copy(tags, rule.Tags)
	tags = append(tags, "path:"+req.URL.Path)
	return tags
}

func (pbp *PathBasedPolicy) findRule(path, method string) PathRule {
	if rule, exists := pbp.pathRules[path]; exists {
		if pbp.methodMatches(method, rule.Methods) {
			return rule
		}
	}

	for _, pattern := range pbp.patterns {
		if pbp.pathMatches(path, pattern.Pattern) && pbp.methodMatches(method, pattern.Rule.Methods) {
			return pattern.Rule
		}
	}

	return PathRule{
		TTL:         pbp.defaultTTL,
		ShouldCache: true,
		Tags:        []string{},
		Methods:     []string{"GET", "HEAD"},
	}
}

func (pbp *PathBasedPolicy) pathMatches(path, pattern string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(path, prefix)
	}
	return path == pattern
}

func (pbp *PathBasedPolicy) methodMatches(method string, allowedMethods []string) bool {
	if len(allowedMethods) == 0 {
		return true
	}
	for _, m := range allowedMethods {
		if m == method {
			return true
		}
	}
	return false
}

func NewTimeBasedPolicy(schedules []CacheSchedule, basePolicy Policy) *TimeBasedPolicy {
	return &TimeBasedPolicy{
		schedules:  schedules,
		basePolicy: basePolicy,
	}
}

func (tbp *TimeBasedPolicy) ShouldCache(req *http.Request, resp *http.Response) bool {
	schedule := tbp.getCurrentSchedule()
	if schedule != nil {
		return schedule.ShouldCache
	}
	return tbp.basePolicy.ShouldCache(req, resp)
}

func (tbp *TimeBasedPolicy) GetTTL(req *http.Request, resp *http.Response) time.Duration {
	schedule := tbp.getCurrentSchedule()
	if schedule != nil {
		return schedule.TTL
	}
	return tbp.basePolicy.GetTTL(req, resp)
}

func (tbp *TimeBasedPolicy) GenerateKey(req *http.Request) string {
	return tbp.basePolicy.GenerateKey(req)
}

func (tbp *TimeBasedPolicy) ShouldInvalidate(req *http.Request) bool {
	return tbp.basePolicy.ShouldInvalidate(req)
}

func (tbp *TimeBasedPolicy) GetTags(req *http.Request, resp *http.Response) []string {
	return tbp.basePolicy.GetTags(req, resp)
}

func (tbp *TimeBasedPolicy) getCurrentSchedule() *CacheSchedule {
	now := time.Now()
	weekday := now.Weekday()
	timeOfDay := time.Date(0, 1, 1, now.Hour(), now.Minute(), now.Second(), 0, time.UTC)

	for _, schedule := range tbp.schedules {
		if tbp.weekdayMatches(weekday, schedule.DaysOfWeek) &&
		   tbp.timeInRange(timeOfDay, schedule.StartTime, schedule.EndTime) {
			return &schedule
		}
	}

	return nil
}

func (tbp *TimeBasedPolicy) weekdayMatches(weekday time.Weekday, allowedDays []time.Weekday) bool {
	if len(allowedDays) == 0 {
		return true
	}
	for _, day := range allowedDays {
		if day == weekday {
			return true
		}
	}
	return false
}

func (tbp *TimeBasedPolicy) timeInRange(current, start, end time.Time) bool {
	startTime := time.Date(0, 1, 1, start.Hour(), start.Minute(), start.Second(), 0, time.UTC)
	endTime := time.Date(0, 1, 1, end.Hour(), end.Minute(), end.Second(), 0, time.UTC)

	if startTime.Before(endTime) {
		return current.After(startTime) && current.Before(endTime)
	}

	return current.After(startTime) || current.Before(endTime)
}

func UserAgentTagExtractor(req *http.Request, resp *http.Response) []string {
	userAgent := req.Header.Get("User-Agent")
	if userAgent == "" {
		return nil
	}

	var tags []string
	if strings.Contains(userAgent, "Mobile") {
		tags = append(tags, "device:mobile")
	} else if strings.Contains(userAgent, "Tablet") {
		tags = append(tags, "device:tablet")
	} else {
		tags = append(tags, "device:desktop")
	}

	if strings.Contains(userAgent, "Chrome") {
		tags = append(tags, "browser:chrome")
	} else if strings.Contains(userAgent, "Firefox") {
		tags = append(tags, "browser:firefox")
	} else if strings.Contains(userAgent, "Safari") {
		tags = append(tags, "browser:safari")
	}

	return tags
}

func ContentTypeTagExtractor(req *http.Request, resp *http.Response) []string {
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		return nil
	}

	parts := strings.Split(contentType, "/")
	if len(parts) >= 2 {
		return []string{
			"content-type:" + parts[0],
			"content-subtype:" + strings.Split(parts[1], ";")[0],
		}
	}

	return []string{"content-type:" + strings.Split(contentType, ";")[0]}
}

func AuthenticationTagExtractor(req *http.Request, resp *http.Response) []string {
	if req.Header.Get("Authorization") != "" {
		return []string{"authenticated:true"}
	}
	return []string{"authenticated:false"}
}

func APIVersionTagExtractor(req *http.Request, resp *http.Response) []string {
	version := req.Header.Get("API-Version")
	if version == "" {
		version = req.URL.Query().Get("version")
	}
	if version == "" {
		if strings.HasPrefix(req.URL.Path, "/v") {
			parts := strings.Split(req.URL.Path, "/")
			if len(parts) > 1 {
				version = parts[1]
			}
		}
	}

	if version != "" {
		return []string{"api-version:" + version}
	}

	return nil
}