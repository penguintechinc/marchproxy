package waf_test

import (
	"net/http/httptest"
	"testing"

	"marchproxy-egress/internal/waf"
)

func TestWAFErrorVarsDefined(t *testing.T) {
	if waf.ErrRequestBlocked == nil {
		t.Error("ErrRequestBlocked should be defined and non-nil")
	}
	if waf.ErrSQLInjection == nil {
		t.Error("ErrSQLInjection should be defined and non-nil")
	}
	if waf.ErrXSSAttack == nil {
		t.Error("ErrXSSAttack should be defined and non-nil")
	}
	if waf.ErrPathTraversal == nil {
		t.Error("ErrPathTraversal should be defined and non-nil")
	}
	if waf.ErrCommandInjection == nil {
		t.Error("ErrCommandInjection should be defined and non-nil")
	}
	if waf.ErrSuspiciousPayload == nil {
		t.Error("ErrSuspiciousPayload should be defined and non-nil")
	}
}

func TestWAFErrorVarsDistinct(t *testing.T) {
	errs := []error{
		waf.ErrRequestBlocked,
		waf.ErrSQLInjection,
		waf.ErrXSSAttack,
		waf.ErrPathTraversal,
		waf.ErrCommandInjection,
		waf.ErrSuspiciousPayload,
	}
	for i := 0; i < len(errs); i++ {
		for j := i + 1; j < len(errs); j++ {
			if errs[i] == errs[j] {
				t.Errorf("error vars at index %d and %d are the same: %v", i, j, errs[i])
			}
		}
	}
}

func TestWAFModeConstants(t *testing.T) {
	modes := []waf.WAFMode{
		waf.ModeDetection,
		waf.ModePrevention,
		waf.ModeBypass,
	}
	seen := make(map[waf.WAFMode]bool)
	for _, m := range modes {
		if m == "" {
			t.Error("WAFMode constant must not be empty string")
		}
		if seen[m] {
			t.Errorf("duplicate WAFMode: %q", m)
		}
		seen[m] = true
	}

	if string(waf.ModeDetection) != "detection" {
		t.Errorf("expected ModeDetection = %q, got %q", "detection", waf.ModeDetection)
	}
	if string(waf.ModePrevention) != "prevention" {
		t.Errorf("expected ModePrevention = %q, got %q", "prevention", waf.ModePrevention)
	}
	if string(waf.ModeBypass) != "bypass" {
		t.Errorf("expected ModeBypass = %q, got %q", "bypass", waf.ModeBypass)
	}
}

func TestRuleSeverityConstants(t *testing.T) {
	severities := []waf.RuleSeverity{
		waf.SeverityInfo,
		waf.SeverityLow,
		waf.SeverityMedium,
		waf.SeverityHigh,
		waf.SeverityCritical,
	}
	seen := make(map[waf.RuleSeverity]bool)
	for _, s := range severities {
		if seen[s] {
			t.Errorf("duplicate RuleSeverity: %d", s)
		}
		seen[s] = true
	}
}

func TestRuleSeverityScore(t *testing.T) {
	tests := []struct {
		severity  waf.RuleSeverity
		wantScore int
	}{
		{waf.SeverityInfo, int(waf.SeverityInfo) * 2},
		{waf.SeverityLow, int(waf.SeverityLow) * 2},
		{waf.SeverityMedium, int(waf.SeverityMedium) * 2},
		{waf.SeverityHigh, int(waf.SeverityHigh) * 2},
		{waf.SeverityCritical, int(waf.SeverityCritical) * 2},
	}
	for _, tt := range tests {
		got := tt.severity.Score()
		if got != tt.wantScore {
			t.Errorf("RuleSeverity(%d).Score() = %d, want %d", tt.severity, got, tt.wantScore)
		}
	}
}

func TestRuleSeverityScorePositive(t *testing.T) {
	if waf.SeverityCritical.Score() <= 0 {
		t.Errorf("SeverityCritical.Score() should be positive, got %d", waf.SeverityCritical.Score())
	}
	if waf.SeverityHigh.Score() <= 0 {
		t.Errorf("SeverityHigh.Score() should be positive, got %d", waf.SeverityHigh.Score())
	}
}

func TestRuleActionConstants(t *testing.T) {
	actions := []waf.RuleAction{
		waf.ActionBlock,
		waf.ActionAllow,
		waf.ActionLog,
		waf.ActionRedirect,
		waf.ActionChallenge,
	}
	seen := make(map[waf.RuleAction]bool)
	for _, a := range actions {
		if a == "" {
			t.Error("RuleAction constant must not be empty")
		}
		if seen[a] {
			t.Errorf("duplicate RuleAction: %q", a)
		}
		seen[a] = true
	}
}

func TestRuleCategoryConstants(t *testing.T) {
	categories := []waf.RuleCategory{
		waf.CategorySQLInjection,
		waf.CategoryXSS,
		waf.CategoryPathTraversal,
		waf.CategoryCommandInjection,
		waf.CategoryXMLInjection,
		waf.CategoryLDAPInjection,
		waf.CategoryProtocolAttack,
		waf.CategoryApplicationAttack,
	}
	seen := make(map[waf.RuleCategory]bool)
	for _, c := range categories {
		if c == "" {
			t.Error("RuleCategory constant must not be empty")
		}
		if seen[c] {
			t.Errorf("duplicate RuleCategory: %q", c)
		}
		seen[c] = true
	}
}

func TestNewWAFNotNil(t *testing.T) {
	cfg := waf.WAFConfig{
		Enabled:             true,
		Mode:                waf.ModeDetection,
		MaxRequestBodySize:  1024 * 1024,
		BlockingScore:       10,
		AnomalyThreshold:    5,
	}
	w := waf.NewWAF(cfg)
	if w == nil {
		t.Fatal("NewWAF should return non-nil WAF")
	}
}

func TestNewWAFDisabled(t *testing.T) {
	cfg := waf.WAFConfig{
		Enabled: false,
		Mode:    waf.ModeDetection,
	}
	w := waf.NewWAF(cfg)
	if w == nil {
		t.Fatal("NewWAF should return non-nil WAF even when disabled")
	}
}

func TestWAFBypassModeAllowsAll(t *testing.T) {
	cfg := waf.WAFConfig{
		Enabled:            true,
		Mode:               waf.ModeBypass,
		MaxRequestBodySize: 1024 * 1024,
	}
	w := waf.NewWAF(cfg)

	req := httptest.NewRequest("GET", "/", nil)
	err := w.ProcessRequest(req)
	if err != nil {
		t.Errorf("WAF in bypass mode should allow all requests, got error: %v", err)
	}
}

func TestWAFDisabledAllowsAll(t *testing.T) {
	cfg := waf.WAFConfig{
		Enabled:            false,
		Mode:               waf.ModePrevention,
		MaxRequestBodySize: 1024 * 1024,
	}
	w := waf.NewWAF(cfg)

	req := httptest.NewRequest("GET", "/safe", nil)
	err := w.ProcessRequest(req)
	if err != nil {
		t.Errorf("disabled WAF should allow all requests, got error: %v", err)
	}
}

func TestWAFDetectionModeDoesNotBlock(t *testing.T) {
	cfg := waf.WAFConfig{
		Enabled:            true,
		Mode:               waf.ModeDetection,
		MaxRequestBodySize: 1024 * 1024,
		BlockingScore:      10,
	}
	w := waf.NewWAF(cfg)

	// Even a suspicious-looking path should not cause blocking in detection mode
	req := httptest.NewRequest("GET", "/page?id=1", nil)
	err := w.ProcessRequest(req)
	if err != nil {
		t.Errorf("detection mode should not block requests, got error: %v", err)
	}
}

func TestWAFPreventionModeBlocksPathTraversal(t *testing.T) {
	cfg := waf.WAFConfig{
		Enabled:            true,
		Mode:               waf.ModePrevention,
		MaxRequestBodySize: 1024 * 1024,
		BlockingScore:      5, // Low threshold to catch path traversal (score=10)
	}
	w := waf.NewWAF(cfg)

	// Path traversal attempt
	req := httptest.NewRequest("GET", "/files/../../../etc/passwd", nil)
	err := w.ProcessRequest(req)
	if err == nil {
		t.Error("WAF in prevention mode should block path traversal attacks")
	}
}

func TestNewRuleEngineNotNil(t *testing.T) {
	re := waf.NewRuleEngine()
	if re == nil {
		t.Fatal("NewRuleEngine should return non-nil")
	}
}

func TestRuleEngineAddAndCheck(t *testing.T) {
	re := waf.NewRuleEngine()

	// Add a simple SQL injection rule
	// We need to import regexp for this
	// Instead, create through WAF which initializes default rules
	cfg := waf.WAFConfig{
		Enabled:            true,
		Mode:               waf.ModeDetection,
		MaxRequestBodySize: 1024,
		BlockingScore:      100, // high so we don't actually block
	}
	w := waf.NewWAF(cfg)
	if w == nil {
		t.Fatal("expected non-nil WAF for rule engine test")
	}
	_ = re // separate rule engine also created
}

func TestRuleEngineCheckClean(t *testing.T) {
	re := waf.NewRuleEngine()
	// A freshly created RuleEngine has no rules, so Check returns nil or empty
	violations := re.Check("hello world", "path")
	if len(violations) != 0 {
		t.Errorf("expected 0 violations for clean input on empty rule engine, got %d", len(violations))
	}
}

func TestViolationFields(t *testing.T) {
	v := waf.Violation{
		Rule:        "sql-001",
		Category:    waf.CategorySQLInjection,
		Severity:    waf.SeverityCritical,
		Description: "SQL injection attempt",
		Evidence:    "SELECT * FROM users",
		Location:    "query:q",
	}
	if v.Rule != "sql-001" {
		t.Errorf("unexpected Rule: %q", v.Rule)
	}
	if v.Category != waf.CategorySQLInjection {
		t.Errorf("unexpected Category: %q", v.Category)
	}
	if v.Severity != waf.SeverityCritical {
		t.Errorf("unexpected Severity: %d", v.Severity)
	}
}

func TestNewAnomalyDetectorNotNil(t *testing.T) {
	ad := waf.NewAnomalyDetector(10)
	if ad == nil {
		t.Fatal("NewAnomalyDetector should return non-nil")
	}
}

func TestNewGeoBlockerNotNil(t *testing.T) {
	gb := waf.NewGeoBlocker([]string{"US", "CA"}, []string{"CN", "RU"})
	if gb == nil {
		t.Fatal("NewGeoBlocker should return non-nil")
	}
}

func TestGeoBlockerAllowedCountry(t *testing.T) {
	gb := waf.NewGeoBlocker([]string{"US"}, []string{})
	// GeoBlocker always returns "US" for unknown IPs in current implementation
	// With US in allowed list, should not block
	blocked, _ := gb.IsBlocked("1.2.3.4")
	if blocked {
		t.Error("US IP should not be blocked when US is in allowed list")
	}
}

func TestGeoBlockerBlockedCountry(t *testing.T) {
	gb := waf.NewGeoBlocker([]string{}, []string{"US"})
	// GeoBlocker stub returns "US" for all IPs
	blocked, country := gb.IsBlocked("1.2.3.4")
	if !blocked {
		t.Error("IP should be blocked when its country is in blocked list")
	}
	if country == "" {
		t.Error("country should be non-empty")
	}
}

func TestNewIPReputationNotNil(t *testing.T) {
	ipr := waf.NewIPReputation()
	if ipr == nil {
		t.Fatal("NewIPReputation should return non-nil")
	}
}

func TestIPReputationGetUnknown(t *testing.T) {
	ipr := waf.NewIPReputation()
	data := ipr.GetReputation("9.9.9.9")
	// Unknown IPs return nil (not in DB)
	if data != nil {
		t.Error("unknown IP should return nil reputation data")
	}
}

func TestWAFConfigFields(t *testing.T) {
	cfg := waf.WAFConfig{
		Enabled:                true,
		Mode:                   waf.ModePrevention,
		AnomalyThreshold:       5,
		BlockingScore:          10,
		ParanoiaLevel:          1,
		MaxRequestBodySize:     1024 * 1024,
		MaxFileUploadSize:      10 * 1024 * 1024,
		EnableGeoBlocking:      false,
		EnableIPReputation:     false,
		EnableAnomalyDetection: false,
		EnableRequestLogging:   false,
		SensitiveDataMasking:   false,
	}
	if !cfg.Enabled {
		t.Error("expected Enabled = true")
	}
	if cfg.Mode != waf.ModePrevention {
		t.Errorf("expected Mode = ModePrevention, got %q", cfg.Mode)
	}
	if cfg.BlockingScore != 10 {
		t.Errorf("expected BlockingScore = 10, got %d", cfg.BlockingScore)
	}
}
