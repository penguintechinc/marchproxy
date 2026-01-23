package threat

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestNewAccessController(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	ac := NewAccessController(false, logger)
	if ac == nil {
		t.Fatal("NewAccessController returned nil")
	}

	if ac.Count() != 0 {
		t.Errorf("Expected 0 rules, got %d", ac.Count())
	}
}

func TestAccessController_AddRule_Domain(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ac := NewAccessController(false, logger)

	rule := &AccessControlRule{
		ID:              "test-1",
		TargetType:      "domain",
		TargetPattern:   "api.example.com",
		Mode:            AccessControlModeAllow,
		AllowedServices: []string{"service-a", "service-b"},
		RequireAuth:     true,
		Category:        "api",
		Source:          "test",
		CreatedAt:       time.Now(),
	}

	err := ac.AddRule(rule)
	if err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	if ac.Count() != 1 {
		t.Errorf("Expected 1 rule, got %d", ac.Count())
	}
}

func TestAccessController_AddRule_IP(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ac := NewAccessController(false, logger)

	rule := &AccessControlRule{
		ID:            "test-ip",
		TargetType:    "ip",
		TargetPattern: "192.168.1.100",
		Mode:          AccessControlModeDeny,
		RequireAuth:   false,
		Category:      "internal",
		Source:        "test",
		CreatedAt:     time.Now(),
	}

	err := ac.AddRule(rule)
	if err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	if ac.Count() != 1 {
		t.Errorf("Expected 1 rule, got %d", ac.Count())
	}
}

func TestAccessController_AddRule_EmptyID(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ac := NewAccessController(false, logger)

	rule := &AccessControlRule{
		ID:            "",
		TargetType:    "domain",
		TargetPattern: "api.example.com",
		Mode:          AccessControlModeAllow,
		CreatedAt:     time.Now(),
	}

	err := ac.AddRule(rule)
	if err == nil {
		t.Error("Expected error for empty ID")
	}
}

func TestAccessController_AddRule_InvalidType(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ac := NewAccessController(false, logger)

	rule := &AccessControlRule{
		ID:            "test-invalid",
		TargetType:    "invalid",
		TargetPattern: "something",
		Mode:          AccessControlModeAllow,
		CreatedAt:     time.Now(),
	}

	err := ac.AddRule(rule)
	if err == nil {
		t.Error("Expected error for invalid target type")
	}
}

func TestAccessController_Check_NoRules_DefaultAllow(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ac := NewAccessController(false, logger) // defaultRequireAuth = false

	decision := ac.Check("api.example.com", "domain", nil)
	if decision == nil {
		t.Fatal("Check returned nil")
	}

	if !decision.Allowed {
		t.Error("Expected default allow when no rules match")
	}
}

func TestAccessController_Check_NoRules_DefaultRequireAuth(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ac := NewAccessController(true, logger) // defaultRequireAuth = true

	// No authentication provided
	decision := ac.Check("api.example.com", "domain", nil)
	if decision.Allowed {
		t.Error("Expected deny when auth required by default but not provided")
	}
	if !decision.RequiresAuth {
		t.Error("Expected RequiresAuth to be true")
	}
}

func TestAccessController_Check_RequireAuth_Unauthenticated(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ac := NewAccessController(false, logger)

	rule := &AccessControlRule{
		ID:            "test-auth",
		TargetType:    "domain",
		TargetPattern: "api.example.com",
		Mode:          AccessControlModeAllow,
		RequireAuth:   true,
		CreatedAt:     time.Now(),
	}
	ac.AddRule(rule)

	// No authentication
	decision := ac.Check("api.example.com", "domain", nil)
	if decision.Allowed {
		t.Error("Expected deny when auth required but not provided")
	}
	if !decision.RequiresAuth {
		t.Error("Expected RequiresAuth to be true")
	}
}

func TestAccessController_Check_RequireAuth_Authenticated(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ac := NewAccessController(false, logger)

	rule := &AccessControlRule{
		ID:            "test-auth",
		TargetType:    "domain",
		TargetPattern: "api.example.com",
		Mode:          AccessControlModeAllow,
		RequireAuth:   true,
		CreatedAt:     time.Now(),
	}
	ac.AddRule(rule)

	svc := &ServiceContext{
		ServiceID:     "svc-123",
		ServiceName:   "my-service",
		TokenID:       "tok-abc",
		Authenticated: true,
	}

	decision := ac.Check("api.example.com", "domain", svc)
	if !decision.Allowed {
		t.Error("Expected allow when authenticated")
	}
}

func TestAccessController_Check_AllowedServices(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ac := NewAccessController(false, logger)

	rule := &AccessControlRule{
		ID:              "test-services",
		TargetType:      "domain",
		TargetPattern:   "api.example.com",
		Mode:            AccessControlModeAllow,
		AllowedServices: []string{"service-a", "service-b"},
		RequireAuth:     true,
		CreatedAt:       time.Now(),
	}
	ac.AddRule(rule)

	tests := []struct {
		serviceName string
		allowed     bool
	}{
		{"service-a", true},
		{"service-b", true},
		{"service-c", false},
		{"unknown", false},
	}

	for _, tt := range tests {
		svc := &ServiceContext{
			ServiceID:     "svc-123",
			ServiceName:   tt.serviceName,
			Authenticated: true,
		}

		decision := ac.Check("api.example.com", "domain", svc)
		if decision.Allowed != tt.allowed {
			t.Errorf("Service %s: expected allowed=%v, got allowed=%v", tt.serviceName, tt.allowed, decision.Allowed)
		}
	}
}

func TestAccessController_Check_AllowedTokens(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ac := NewAccessController(false, logger)

	rule := &AccessControlRule{
		ID:            "test-tokens",
		TargetType:    "domain",
		TargetPattern: "api.example.com",
		Mode:          AccessControlModeAllow,
		AllowedTokens: []string{"token-1", "token-2"},
		RequireAuth:   true,
		CreatedAt:     time.Now(),
	}
	ac.AddRule(rule)

	tests := []struct {
		tokenID string
		allowed bool
	}{
		{"token-1", true},
		{"token-2", true},
		{"token-3", false},
		{"unknown", false},
	}

	for _, tt := range tests {
		svc := &ServiceContext{
			ServiceID:     "svc-123",
			ServiceName:   "my-service",
			TokenID:       tt.tokenID,
			Authenticated: true,
		}

		decision := ac.Check("api.example.com", "domain", svc)
		if decision.Allowed != tt.allowed {
			t.Errorf("Token %s: expected allowed=%v, got allowed=%v", tt.tokenID, tt.allowed, decision.Allowed)
		}
	}
}

func TestAccessController_Check_DenyMode(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ac := NewAccessController(false, logger)

	rule := &AccessControlRule{
		ID:            "test-deny",
		TargetType:    "domain",
		TargetPattern: "blocked.example.com",
		Mode:          AccessControlModeDeny,
		CreatedAt:     time.Now(),
	}
	ac.AddRule(rule)

	decision := ac.Check("blocked.example.com", "domain", nil)
	if decision.Allowed {
		t.Error("Expected deny mode to block access")
	}
}

func TestAccessController_Check_ExpiredRule(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ac := NewAccessController(false, logger)

	rule := &AccessControlRule{
		ID:            "test-expired",
		TargetType:    "domain",
		TargetPattern: "api.example.com",
		Mode:          AccessControlModeDeny,
		CreatedAt:     time.Now().Add(-2 * time.Hour),
		ExpiresAt:     time.Now().Add(-1 * time.Hour), // Already expired
	}
	ac.AddRule(rule)

	decision := ac.Check("api.example.com", "domain", nil)
	if !decision.Allowed {
		t.Error("Expected expired rule to be ignored")
	}
}

func TestAccessController_Check_IPRule(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ac := NewAccessController(false, logger)

	rule := &AccessControlRule{
		ID:            "test-ip",
		TargetType:    "ip",
		TargetPattern: "192.168.1.100",
		Mode:          AccessControlModeDeny,
		CreatedAt:     time.Now(),
	}
	ac.AddRule(rule)

	decision := ac.Check("192.168.1.100", "ip", nil)
	if decision.Allowed {
		t.Error("Expected IP to be blocked")
	}

	decision = ac.Check("192.168.1.50", "ip", nil)
	if !decision.Allowed {
		t.Error("Expected different IP to be allowed")
	}
}

func TestAccessController_RemoveRule(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ac := NewAccessController(false, logger)

	rule := &AccessControlRule{
		ID:            "test-1",
		TargetType:    "domain",
		TargetPattern: "api.example.com",
		Mode:          AccessControlModeDeny,
		CreatedAt:     time.Now(),
	}
	ac.AddRule(rule)

	// Verify it's blocked
	decision := ac.Check("api.example.com", "domain", nil)
	if decision.Allowed {
		t.Error("Expected domain to be blocked before removal")
	}

	// Remove the rule
	err := ac.RemoveRule("test-1")
	if err != nil {
		t.Fatalf("RemoveRule failed: %v", err)
	}

	// Verify it's no longer blocked
	decision = ac.Check("api.example.com", "domain", nil)
	if !decision.Allowed {
		t.Error("Expected domain to be allowed after rule removal")
	}
}

func TestAccessController_RemoveRule_NotFound(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ac := NewAccessController(false, logger)

	err := ac.RemoveRule("nonexistent")
	if err == nil {
		t.Error("Expected error when removing nonexistent rule")
	}
}

func TestAccessController_Clear(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ac := NewAccessController(false, logger)

	// Add multiple rules
	for i := 0; i < 5; i++ {
		rule := &AccessControlRule{
			ID:            "test-" + string(rune('0'+i)),
			TargetType:    "domain",
			TargetPattern: "domain" + string(rune('0'+i)) + ".com",
			Mode:          AccessControlModeAllow,
			CreatedAt:     time.Now(),
		}
		ac.AddRule(rule)
	}

	if ac.Count() != 5 {
		t.Errorf("Expected 5 rules, got %d", ac.Count())
	}

	ac.Clear()

	if ac.Count() != 0 {
		t.Errorf("Expected 0 rules after clear, got %d", ac.Count())
	}
}

func TestAccessController_GetRules(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ac := NewAccessController(false, logger)

	rules := []*AccessControlRule{
		{ID: "r1", TargetType: "domain", TargetPattern: "api1.com", Mode: AccessControlModeAllow, CreatedAt: time.Now()},
		{ID: "r2", TargetType: "domain", TargetPattern: "api2.com", Mode: AccessControlModeDeny, CreatedAt: time.Now()},
		{ID: "r3", TargetType: "ip", TargetPattern: "192.168.1.1", Mode: AccessControlModeAllow, CreatedAt: time.Now()},
	}

	for _, rule := range rules {
		ac.AddRule(rule)
	}

	gotRules := ac.GetRules()
	if len(gotRules) != 3 {
		t.Errorf("Expected 3 rules, got %d", len(gotRules))
	}
}

func TestAccessController_SetDefaultRequireAuth(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ac := NewAccessController(false, logger)

	// Initially should allow without auth
	decision := ac.Check("any.domain.com", "domain", nil)
	if !decision.Allowed {
		t.Error("Expected allow without auth initially")
	}

	// Change default
	ac.SetDefaultRequireAuth(true)

	// Now should require auth
	decision = ac.Check("any.domain.com", "domain", nil)
	if decision.Allowed {
		t.Error("Expected deny when default require auth is set")
	}
}

func TestAccessController_SetDefaultAllow(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ac := NewAccessController(false, logger)

	// Initially should allow
	decision := ac.Check("any.domain.com", "domain", nil)
	if !decision.Allowed {
		t.Error("Expected allow by default")
	}

	// Change default to deny
	ac.SetDefaultAllow(false)

	// Now should deny by default
	decision = ac.Check("any.domain.com", "domain", nil)
	if decision.Allowed {
		t.Error("Expected deny when default allow is false")
	}
}

func TestAccessController_Check_ServiceIDMatch(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ac := NewAccessController(false, logger)

	rule := &AccessControlRule{
		ID:              "test-services",
		TargetType:      "domain",
		TargetPattern:   "api.example.com",
		Mode:            AccessControlModeAllow,
		AllowedServices: []string{"svc-123", "svc-456"},
		RequireAuth:     true,
		CreatedAt:       time.Now(),
	}
	ac.AddRule(rule)

	// Match by ServiceID
	svc := &ServiceContext{
		ServiceID:     "svc-123",
		ServiceName:   "different-name",
		Authenticated: true,
	}

	decision := ac.Check("api.example.com", "domain", svc)
	if !decision.Allowed {
		t.Error("Expected allow when ServiceID matches")
	}
}
