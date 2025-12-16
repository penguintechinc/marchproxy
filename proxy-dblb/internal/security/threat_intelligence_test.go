package security

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// TestNewThreatIntelligenceEngine tests the creation of a new ThreatIntelligenceEngine
func TestNewThreatIntelligenceEngine(t *testing.T) {
	logger := logrus.New()
	config := &ThreatIntelConfig{
		Enabled:        true,
		UpdateInterval: 1 * time.Hour,
		Feeds: map[string]*Feed{
			"feed1": {
				Name:    "Feed 1",
				Type:    "json",
				URL:     "http://example.com/feed1",
				Enabled: true,
			},
		},
		AutoBlock: AutoBlockConfig{
			Enabled:             false,
			ConfidenceThreshold: 0.7,
		},
		RetentionDays: 30,
	}

	engine := NewThreatIntelligenceEngine(config, nil, logger)

	if engine == nil {
		t.Fatal("NewThreatIntelligenceEngine returned nil")
	}

	if engine.config == nil {
		t.Error("config not set")
	}

	if engine.indicators == nil {
		t.Error("indicators map not initialized")
	}

	if engine.ipThreats == nil {
		t.Error("ipThreats map not initialized")
	}

	if engine.domainThreats == nil {
		t.Error("domainThreats map not initialized")
	}

	if len(engine.feedStats) != 1 {
		t.Errorf("feedStats count = %d, want 1", len(engine.feedStats))
	}

	engine.Close()
}

// TestIsIPThreat tests checking if an IP is a known threat
func TestIsIPThreat(t *testing.T) {
	logger := logrus.New()
	config := &ThreatIntelConfig{
		Enabled:        true,
		UpdateInterval: 1 * time.Hour,
		Feeds:          map[string]*Feed{},
		AutoBlock: AutoBlockConfig{
			Enabled: false,
		},
		RetentionDays: 30,
	}

	engine := NewThreatIntelligenceEngine(config, nil, logger)
	defer engine.Close()

	// Add a test IP threat
	threat := &ThreatIndicator{
		ID:         "test_ip_1",
		Type:       "ip",
		Value:      "192.168.1.100",
		Confidence: 0.9,
		Severity:   "high",
		Categories: []string{"malware"},
		Sources:    []string{"test"},
		FirstSeen:  time.Now(),
		LastSeen:   time.Now(),
		Blocked:    false,
		Tags:       []string{},
		Metadata:   make(map[string]interface{}),
	}

	engine.processIndicator(threat)

	// Test IP that should be a threat
	if !engine.IsIPThreat("192.168.1.100") {
		t.Error("IsIPThreat returned false for known threat IP")
	}

	// Test case-insensitivity
	if !engine.IsIPThreat("192.168.1.100") {
		t.Error("IsIPThreat failed on case variation")
	}

	// Test IP that should not be a threat
	if engine.IsIPThreat("192.168.1.101") {
		t.Error("IsIPThreat returned true for unknown IP")
	}

	// Test with expired threat
	expiredTime := time.Now().Add(-1 * time.Hour)
	expiredThreat := &ThreatIndicator{
		ID:         "test_ip_expired",
		Type:       "ip",
		Value:      "10.0.0.1",
		Confidence: 0.9,
		Severity:   "high",
		Categories: []string{"malware"},
		Sources:    []string{"test"},
		FirstSeen:  time.Now(),
		LastSeen:   time.Now(),
		ExpiresAt:  &expiredTime,
		Blocked:    false,
		Tags:       []string{},
		Metadata:   make(map[string]interface{}),
	}

	engine.processIndicator(expiredThreat)

	if engine.IsIPThreat("10.0.0.1") {
		t.Error("IsIPThreat returned true for expired threat")
	}
}

// TestIsDomainThreat tests checking if a domain is a known threat
func TestIsDomainThreat(t *testing.T) {
	logger := logrus.New()
	config := &ThreatIntelConfig{
		Enabled:        true,
		UpdateInterval: 1 * time.Hour,
		Feeds:          map[string]*Feed{},
		AutoBlock: AutoBlockConfig{
			Enabled: false,
		},
		RetentionDays: 30,
	}

	engine := NewThreatIntelligenceEngine(config, nil, logger)
	defer engine.Close()

	// Add a test domain threat
	threat := &ThreatIndicator{
		ID:         "test_domain_1",
		Type:       "domain",
		Value:      "malicious.example.com",
		Confidence: 0.95,
		Severity:   "critical",
		Categories: []string{"phishing", "malware"},
		Sources:    []string{"test"},
		FirstSeen:  time.Now(),
		LastSeen:   time.Now(),
		Blocked:    false,
		Tags:       []string{},
		Metadata:   make(map[string]interface{}),
	}

	engine.processIndicator(threat)

	// Test domain that should be a threat
	if !engine.IsDomainThreat("malicious.example.com") {
		t.Error("IsDomainThreat returned false for known threat domain")
	}

	// Test case-insensitivity
	if !engine.IsDomainThreat("MALICIOUS.EXAMPLE.COM") {
		t.Error("IsDomainThreat failed on case variation")
	}

	// Test domain that should not be a threat
	if engine.IsDomainThreat("legitimate.example.com") {
		t.Error("IsDomainThreat returned true for unknown domain")
	}

	// Test with expired threat
	expiredTime := time.Now().Add(-1 * time.Hour)
	expiredThreat := &ThreatIndicator{
		ID:         "test_domain_expired",
		Type:       "domain",
		Value:      "expired.malware.com",
		Confidence: 0.9,
		Severity:   "high",
		Categories: []string{"malware"},
		Sources:    []string{"test"},
		FirstSeen:  time.Now(),
		LastSeen:   time.Now(),
		ExpiresAt:  &expiredTime,
		Blocked:    false,
		Tags:       []string{},
		Metadata:   make(map[string]interface{}),
	}

	engine.processIndicator(expiredThreat)

	if engine.IsDomainThreat("expired.malware.com") {
		t.Error("IsDomainThreat returned true for expired threat")
	}
}

// TestAddCustomThreat tests adding custom threat indicators
func TestAddCustomThreat(t *testing.T) {
	logger := logrus.New()
	config := &ThreatIntelConfig{
		Enabled:        true,
		UpdateInterval: 1 * time.Hour,
		Feeds:          map[string]*Feed{},
		AutoBlock: AutoBlockConfig{
			Enabled: false,
		},
		RetentionDays: 30,
	}

	engine := NewThreatIntelligenceEngine(config, nil, logger)
	defer engine.Close()

	// Add custom IP threat
	err := engine.AddCustomThreat("ip", "203.0.113.10", 0.8, []string{"botnet"})
	if err != nil {
		t.Errorf("AddCustomThreat failed: %v", err)
	}

	if !engine.IsIPThreat("203.0.113.10") {
		t.Error("Custom IP threat not added properly")
	}

	// Add custom domain threat
	err = engine.AddCustomThreat("domain", "badness.net", 0.9, []string{"c2", "malware"})
	if err != nil {
		t.Errorf("AddCustomThreat failed: %v", err)
	}

	if !engine.IsDomainThreat("badness.net") {
		t.Error("Custom domain threat not added properly")
	}

	// Verify threat details
	threat := engine.GetThreat("203.0.113.10")
	if threat == nil {
		t.Fatal("GetThreat returned nil for custom threat")
	}

	if threat.Confidence != 0.8 {
		t.Errorf("Confidence mismatch: got %f, want 0.8", threat.Confidence)
	}

	if len(threat.Categories) != 1 || threat.Categories[0] != "botnet" {
		t.Errorf("Categories mismatch: got %v, want [botnet]", threat.Categories)
	}
}

// TestRemoveCustomThreat tests removing threat indicators
func TestRemoveCustomThreat(t *testing.T) {
	logger := logrus.New()
	config := &ThreatIntelConfig{
		Enabled:        true,
		UpdateInterval: 1 * time.Hour,
		Feeds:          map[string]*Feed{},
		AutoBlock: AutoBlockConfig{
			Enabled: false,
		},
		RetentionDays: 30,
	}

	engine := NewThreatIntelligenceEngine(config, nil, logger)
	defer engine.Close()

	// Add threat
	engine.AddCustomThreat("ip", "192.0.2.50", 0.8, []string{"malware"})
	if !engine.IsIPThreat("192.0.2.50") {
		t.Fatal("Threat not added")
	}

	// Remove threat
	removed := engine.RemoveThreat("192.0.2.50")
	if !removed {
		t.Error("RemoveThreat returned false")
	}

	// Verify removal
	if engine.IsIPThreat("192.0.2.50") {
		t.Error("Threat still exists after removal")
	}

	// Test removing non-existent threat
	removed = engine.RemoveThreat("10.0.0.99")
	if removed {
		t.Error("RemoveThreat returned true for non-existent threat")
	}

	// Add and remove domain threat
	engine.AddCustomThreat("domain", "remove-me.com", 0.7, []string{"phishing"})
	if !engine.IsDomainThreat("remove-me.com") {
		t.Fatal("Domain threat not added")
	}

	removed = engine.RemoveThreat("remove-me.com")
	if !removed {
		t.Error("RemoveThreat for domain returned false")
	}

	if engine.IsDomainThreat("remove-me.com") {
		t.Error("Domain threat still exists after removal")
	}
}

// TestGetThreatInfo tests getting threat details
func TestGetThreatInfo(t *testing.T) {
	logger := logrus.New()
	config := &ThreatIntelConfig{
		Enabled:        true,
		UpdateInterval: 1 * time.Hour,
		Feeds:          map[string]*Feed{},
		AutoBlock: AutoBlockConfig{
			Enabled: false,
		},
		RetentionDays: 30,
	}

	engine := NewThreatIntelligenceEngine(config, nil, logger)
	defer engine.Close()

	// Add a detailed threat
	engine.AddCustomThreat("ip", "198.51.100.1", 0.85, []string{"ransomware", "dropper"})

	// Get threat info
	threat := engine.GetThreat("198.51.100.1")
	if threat == nil {
		t.Fatal("GetThreat returned nil")
	}

	if threat.Type != "ip" {
		t.Errorf("Type mismatch: got %s, want ip", threat.Type)
	}

	if threat.Value != "198.51.100.1" {
		t.Errorf("Value mismatch: got %s, want 198.51.100.1", threat.Value)
	}

	if threat.Confidence != 0.85 {
		t.Errorf("Confidence mismatch: got %f, want 0.85", threat.Confidence)
	}

	if len(threat.Categories) != 2 {
		t.Errorf("Categories count mismatch: got %d, want 2", len(threat.Categories))
	}

	if threat.Severity != "medium" {
		t.Errorf("Severity mismatch: got %s, want medium", threat.Severity)
	}

	if len(threat.Sources) == 0 {
		t.Error("Sources are empty")
	}

	// Test case-insensitive retrieval
	threat2 := engine.GetThreat("198.51.100.1")
	if threat2 == nil {
		t.Error("GetThreat failed for lowercase lookup")
	}

	// Test non-existent threat
	threat3 := engine.GetThreat("192.168.0.1")
	if threat3 != nil {
		t.Error("GetThreat returned non-nil for non-existent threat")
	}
}

// TestGetStats tests getting threat intelligence statistics
func TestGetStats(t *testing.T) {
	logger := logrus.New()
	config := &ThreatIntelConfig{
		Enabled:        true,
		UpdateInterval: 1 * time.Hour,
		Feeds: map[string]*Feed{
			"feed1": {
				Name:    "Feed 1",
				Type:    "json",
				URL:     "http://example.com/feed1",
				Enabled: true,
			},
		},
		AutoBlock: AutoBlockConfig{
			Enabled: false,
		},
		RetentionDays: 30,
	}

	engine := NewThreatIntelligenceEngine(config, nil, logger)
	defer engine.Close()

	// Initial stats should be empty
	stats := engine.GetStats()
	if stats == nil {
		t.Fatal("GetStats returned nil")
	}

	if totalIndicators, ok := stats["total_indicators"].(int); !ok || totalIndicators != 0 {
		t.Errorf("Initial total_indicators wrong: got %v, want 0", stats["total_indicators"])
	}

	if ipThreats, ok := stats["ip_threats"].(int); !ok || ipThreats != 0 {
		t.Errorf("Initial ip_threats wrong: got %v, want 0", stats["ip_threats"])
	}

	// Add threats
	engine.AddCustomThreat("ip", "203.0.113.1", 0.8, []string{"malware"})
	engine.AddCustomThreat("ip", "203.0.113.2", 0.7, []string{"botnet"})
	engine.AddCustomThreat("domain", "evil.com", 0.9, []string{"phishing"})

	// Check updated stats
	stats = engine.GetStats()

	if totalIndicators, ok := stats["total_indicators"].(int); !ok || totalIndicators != 3 {
		t.Errorf("total_indicators after adds: got %v, want 3", stats["total_indicators"])
	}

	if ipThreats, ok := stats["ip_threats"].(int); !ok || ipThreats != 2 {
		t.Errorf("ip_threats after adds: got %v, want 2", stats["ip_threats"])
	}

	if domainThreats, ok := stats["domain_threats"].(int); !ok || domainThreats != 1 {
		t.Errorf("domain_threats after adds: got %v, want 1", stats["domain_threats"])
	}

	// Verify feed_stats exists
	if _, ok := stats["feed_stats"]; !ok {
		t.Error("feed_stats not in stats")
	}
}

// TestMultipleIndicators tests handling multiple indicators
func TestMultipleIndicators(t *testing.T) {
	logger := logrus.New()
	config := &ThreatIntelConfig{
		Enabled:        true,
		UpdateInterval: 1 * time.Hour,
		Feeds:          map[string]*Feed{},
		AutoBlock: AutoBlockConfig{
			Enabled: false,
		},
		RetentionDays: 30,
	}

	engine := NewThreatIntelligenceEngine(config, nil, logger)
	defer engine.Close()

	// Add multiple IP threats
	ips := []string{"198.51.100.1", "198.51.100.2", "198.51.100.3"}
	for _, ip := range ips {
		engine.AddCustomThreat("ip", ip, 0.8, []string{"malware"})
	}

	// Add multiple domain threats
	domains := []string{"bad1.com", "bad2.net", "bad3.org", "bad4.info"}
	for _, domain := range domains {
		engine.AddCustomThreat("domain", domain, 0.85, []string{"phishing"})
	}

	// Verify all are added
	for _, ip := range ips {
		if !engine.IsIPThreat(ip) {
			t.Errorf("IP %s not found as threat", ip)
		}
	}

	for _, domain := range domains {
		if !engine.IsDomainThreat(domain) {
			t.Errorf("Domain %s not found as threat", domain)
		}
	}

	// Check stats
	stats := engine.GetStats()
	if totalIndicators, ok := stats["total_indicators"].(int); !ok || totalIndicators != 7 {
		t.Errorf("total_indicators: got %v, want 7", stats["total_indicators"])
	}

	if ipThreats, ok := stats["ip_threats"].(int); !ok || ipThreats != 3 {
		t.Errorf("ip_threats: got %v, want 3", stats["ip_threats"])
	}

	if domainThreats, ok := stats["domain_threats"].(int); !ok || domainThreats != 4 {
		t.Errorf("domain_threats: got %v, want 4", stats["domain_threats"])
	}
}

// TestAutoBlock tests auto-blocking functionality
func TestAutoBlock(t *testing.T) {
	logger := logrus.New()
	config := &ThreatIntelConfig{
		Enabled:        true,
		UpdateInterval: 1 * time.Hour,
		Feeds:          map[string]*Feed{},
		AutoBlock: AutoBlockConfig{
			Enabled:             true,
			ConfidenceThreshold: 0.8,
			BlockDuration:       1 * time.Hour,
		},
		RetentionDays: 30,
	}

	engine := NewThreatIntelligenceEngine(config, nil, logger)
	defer engine.Close()

	// Add threat above confidence threshold - should be auto-blocked
	engine.AddCustomThreat("ip", "203.0.113.100", 0.9, []string{"malware"})

	threat := engine.GetThreat("203.0.113.100")
	if threat == nil {
		t.Fatal("Threat not found")
	}

	if !threat.Blocked {
		t.Error("Threat with high confidence should be auto-blocked")
	}

	if threat.BlockedAt == nil {
		t.Error("BlockedAt should be set for auto-blocked threat")
	}

	if threat.ExpiresAt == nil {
		t.Error("ExpiresAt should be set when block duration is configured")
	}

	// Add threat below confidence threshold - should not be auto-blocked
	engine.AddCustomThreat("ip", "203.0.113.101", 0.7, []string{"suspicious"})

	threat2 := engine.GetThreat("203.0.113.101")
	if threat2 == nil {
		t.Fatal("Threat not found")
	}

	if threat2.Blocked {
		t.Error("Threat with low confidence should not be auto-blocked")
	}
}

// TestConcurrentAccess tests concurrent access to the engine
func TestConcurrentAccess(t *testing.T) {
	logger := logrus.New()
	config := &ThreatIntelConfig{
		Enabled:        true,
		UpdateInterval: 1 * time.Hour,
		Feeds:          map[string]*Feed{},
		AutoBlock: AutoBlockConfig{
			Enabled: false,
		},
		RetentionDays: 30,
	}

	engine := NewThreatIntelligenceEngine(config, nil, logger)
	defer engine.Close()

	// Run concurrent operations
	done := make(chan bool, 3)

	// Goroutine 1: Add threats
	go func() {
		for i := 0; i < 10; i++ {
			ip := "203.0.113." + string(rune(10+i))
			engine.AddCustomThreat("ip", ip, 0.8, []string{"malware"})
		}
		done <- true
	}()

	// Goroutine 2: Check threats
	go func() {
		for i := 0; i < 20; i++ {
			engine.IsIPThreat("203.0.113.10")
			engine.IsDomainThreat("example.com")
		}
		done <- true
	}()

	// Goroutine 3: Get stats
	go func() {
		for i := 0; i < 10; i++ {
			engine.GetStats()
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// Verify data is consistent
	stats := engine.GetStats()
	if totalIndicators, ok := stats["total_indicators"].(int); !ok || totalIndicators != 10 {
		t.Errorf("total_indicators after concurrent adds: got %v, want 10", stats["total_indicators"])
	}
}

// TestThreatIndicatorFields tests that threat indicator fields are properly set
func TestThreatIndicatorFields(t *testing.T) {
	logger := logrus.New()
	config := &ThreatIntelConfig{
		Enabled:        true,
		UpdateInterval: 1 * time.Hour,
		Feeds:          map[string]*Feed{},
		AutoBlock: AutoBlockConfig{
			Enabled: false,
		},
		RetentionDays: 30,
	}

	engine := NewThreatIntelligenceEngine(config, nil, logger)
	defer engine.Close()

	categories := []string{"malware", "trojan", "rootkit"}
	engine.AddCustomThreat("ip", "192.0.2.1", 0.92, categories)

	threat := engine.GetThreat("192.0.2.1")
	if threat == nil {
		t.Fatal("Threat not found")
	}

	// Check ID format
	if threat.ID == "" {
		t.Error("Threat ID is empty")
	}

	// Check type
	if threat.Type != "ip" {
		t.Errorf("Type mismatch: got %s, want ip", threat.Type)
	}

	// Check value
	if threat.Value != "192.0.2.1" {
		t.Errorf("Value mismatch: got %s, want 192.0.2.1", threat.Value)
	}

	// Check confidence
	if threat.Confidence != 0.92 {
		t.Errorf("Confidence mismatch: got %f, want 0.92", threat.Confidence)
	}

	// Check categories
	if len(threat.Categories) != 3 {
		t.Errorf("Categories count: got %d, want 3", len(threat.Categories))
	}

	// Check severity
	if threat.Severity == "" {
		t.Error("Severity should be set")
	}

	// Check sources
	if len(threat.Sources) == 0 {
		t.Error("Sources should not be empty")
	}

	// Check times
	if threat.FirstSeen.IsZero() {
		t.Error("FirstSeen should be set")
	}

	if threat.LastSeen.IsZero() {
		t.Error("LastSeen should be set")
	}

	// Check tags
	if len(threat.Tags) == 0 {
		t.Error("Tags should be set for manual threats")
	}

	// Check metadata
	if threat.Metadata == nil {
		t.Error("Metadata should not be nil")
	}
}

// TestEmptyAndNilCases tests edge cases with empty and nil values
func TestEmptyAndNilCases(t *testing.T) {
	logger := logrus.New()
	config := &ThreatIntelConfig{
		Enabled:        true,
		UpdateInterval: 1 * time.Hour,
		Feeds:          map[string]*Feed{},
		AutoBlock: AutoBlockConfig{
			Enabled: false,
		},
		RetentionDays: 30,
	}

	engine := NewThreatIntelligenceEngine(config, nil, logger)
	defer engine.Close()

	// Test empty string queries
	if engine.IsIPThreat("") {
		t.Error("IsIPThreat should return false for empty string")
	}

	if engine.IsDomainThreat("") {
		t.Error("IsDomainThreat should return false for empty string")
	}

	if engine.GetThreat("") != nil {
		t.Error("GetThreat should return nil for empty string")
	}

	// Test remove with empty string
	removed := engine.RemoveThreat("")
	if removed {
		t.Error("RemoveThreat should return false for empty string")
	}

	// Test add custom threat with empty categories
	err := engine.AddCustomThreat("ip", "198.51.100.50", 0.5, nil)
	if err != nil {
		t.Errorf("AddCustomThreat with nil categories failed: %v", err)
	}

	threat := engine.GetThreat("198.51.100.50")
	if threat == nil {
		t.Fatal("Threat should be added even with nil categories")
	}
}

// TestEngineClosing tests that the engine can be properly closed
func TestEngineClosing(t *testing.T) {
	logger := logrus.New()
	config := &ThreatIntelConfig{
		Enabled:        true,
		UpdateInterval: 100 * time.Millisecond,
		Feeds:          map[string]*Feed{},
		AutoBlock: AutoBlockConfig{
			Enabled: false,
		},
		RetentionDays: 30,
	}

	engine := NewThreatIntelligenceEngine(config, nil, logger)

	// Add some threats
	engine.AddCustomThreat("ip", "203.0.113.1", 0.8, []string{"malware"})

	// Close engine
	err := engine.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Context should be cancelled
	select {
	case <-engine.ctx.Done():
		// Expected
	default:
		t.Error("Engine context should be cancelled after Close()")
	}

	// Give background goroutines time to exit
	time.Sleep(200 * time.Millisecond)
}
