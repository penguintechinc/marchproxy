//go:build ci

package threat

import (
	"testing"
	"time"

	"marchproxy-egress/internal/logging"
)

func TestAccessControllerCheck(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	tests := []struct {
		name           string
		rules          []*AccessControlRule
		target         string
		targetType     string
		svc            *ServiceContext
		expectedAllow  bool
		expectedReason string
	}{
		{
			name:          "no rules defaults to allow",
			target:        "example.com",
			targetType:    "domain",
			svc:           nil,
			expectedAllow: true,
		},
		{
			name:       "exact domain match - allow rule",
			target:     "example.com",
			targetType: "domain",
			rules: []*AccessControlRule{
				{
					ID:            "rule1",
					TargetType:    "domain",
					TargetPattern: "example.com",
					Mode:          AccessControlModeAllow,
					RequireAuth:   false,
				},
			},
			svc:           nil,
			expectedAllow: true,
		},
		{
			name:       "exact domain match - deny rule",
			target:     "malicious.com",
			targetType: "domain",
			rules: []*AccessControlRule{
				{
					ID:            "rule1",
					TargetType:    "domain",
					TargetPattern: "malicious.com",
					Mode:          AccessControlModeDeny,
					RequireAuth:   false,
				},
			},
			svc:           nil,
			expectedAllow: false,
		},
		{
			name:       "IP address exact match",
			target:     "192.168.1.1",
			targetType: "ip",
			rules: []*AccessControlRule{
				{
					ID:            "rule1",
					TargetType:    "ip",
					TargetPattern: "192.168.1.1",
					Mode:          AccessControlModeDeny,
					RequireAuth:   false,
				},
			},
			svc:           nil,
			expectedAllow: false,
		},
		{
			name:       "unknown target type uses default",
			target:     "unknown.com",
			targetType: "unknown",
			svc:        nil,
			expectedAllow: true,
		},
		{
			name:       "authentication required but not provided",
			target:     "secure.com",
			targetType: "domain",
			rules: []*AccessControlRule{
				{
					ID:            "rule1",
					TargetType:    "domain",
					TargetPattern: "secure.com",
					Mode:          AccessControlModeAllow,
					RequireAuth:   true,
				},
			},
			svc:           &ServiceContext{Authenticated: false},
			expectedAllow: false,
		},
		{
			name:       "authentication required and provided",
			target:     "secure.com",
			targetType: "domain",
			rules: []*AccessControlRule{
				{
					ID:            "rule1",
					TargetType:    "domain",
					TargetPattern: "secure.com",
					Mode:          AccessControlModeAllow,
					RequireAuth:   true,
				},
			},
			svc:           &ServiceContext{Authenticated: true, ServiceID: "svc1"},
			expectedAllow: true,
		},
		{
			name:       "service allowed by ID",
			target:     "internal.com",
			targetType: "domain",
			rules: []*AccessControlRule{
				{
					ID:               "rule1",
					TargetType:       "domain",
					TargetPattern:    "internal.com",
					Mode:             AccessControlModeAllow,
					AllowedServices:  []string{"service-1"},
					RequireAuth:      true,
				},
			},
			svc: &ServiceContext{
				Authenticated: true,
				ServiceID:     "service-1",
			},
			expectedAllow: true,
		},
		{
			name:       "service not allowed",
			target:     "internal.com",
			targetType: "domain",
			rules: []*AccessControlRule{
				{
					ID:               "rule1",
					TargetType:       "domain",
					TargetPattern:    "internal.com",
					Mode:             AccessControlModeAllow,
					AllowedServices:  []string{"service-1"},
					RequireAuth:      true,
				},
			},
			svc: &ServiceContext{
				Authenticated: true,
				ServiceID:     "service-2",
			},
			expectedAllow: false,
		},
		{
			name:       "token allowed",
			target:     "premium.com",
			targetType: "domain",
			rules: []*AccessControlRule{
				{
					ID:            "rule1",
					TargetType:    "domain",
					TargetPattern: "premium.com",
					Mode:          AccessControlModeAllow,
					AllowedTokens: []string{"token-123"},
					RequireAuth:   true,
				},
			},
			svc: &ServiceContext{
				Authenticated: true,
				TokenID:       "token-123",
			},
			expectedAllow: true,
		},
		{
			name:       "token not allowed",
			target:     "premium.com",
			targetType: "domain",
			rules: []*AccessControlRule{
				{
					ID:            "rule1",
					TargetType:    "domain",
					TargetPattern: "premium.com",
					Mode:          AccessControlModeAllow,
					AllowedTokens: []string{"token-123"},
					RequireAuth:   true,
				},
			},
			svc: &ServiceContext{
				Authenticated: true,
				TokenID:       "token-456",
			},
			expectedAllow: false,
		},
		{
			name:       "expired rule treated as no match",
			target:     "expired.com",
			targetType: "domain",
			rules: []*AccessControlRule{
				{
					ID:            "rule1",
					TargetType:    "domain",
					TargetPattern: "expired.com",
					Mode:          AccessControlModeDeny,
					ExpiresAt:     time.Now().Add(-1 * time.Hour),
				},
			},
			svc:           nil,
			expectedAllow: true, // Rule expired, so defaults to allow
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ac := NewAccessController(false, logger)

			for _, rule := range tt.rules {
				if err := ac.AddRule(rule); err != nil {
					t.Fatalf("failed to add rule: %v", err)
				}
			}

			decision := ac.Check(tt.target, tt.targetType, tt.svc)

			if decision.Allowed != tt.expectedAllow {
				t.Errorf("expected allowed=%v, got %v (reason: %s)", tt.expectedAllow, decision.Allowed, decision.Reason)
			}
		})
	}
}

func TestAccessControllerAddRule(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	tests := []struct {
		name          string
		rule          *AccessControlRule
		shouldSucceed bool
	}{
		{
			name: "add valid domain rule",
			rule: &AccessControlRule{
				ID:            "rule1",
				TargetType:    "domain",
				TargetPattern: "example.com",
				Mode:          AccessControlModeAllow,
			},
			shouldSucceed: true,
		},
		{
			name: "add valid IP rule",
			rule: &AccessControlRule{
				ID:            "rule2",
				TargetType:    "ip",
				TargetPattern: "192.168.1.0",
				Mode:          AccessControlModeDeny,
			},
			shouldSucceed: true,
		},
		{
			name: "add valid URL rule",
			rule: &AccessControlRule{
				ID:            "rule3",
				TargetType:    "url_pattern",
				TargetPattern: "/api/v1/*",
				Mode:          AccessControlModeAllow,
			},
			shouldSucceed: true,
		},
		{
			name: "reject rule without ID",
			rule: &AccessControlRule{
				ID:            "",
				TargetType:    "domain",
				TargetPattern: "example.com",
			},
			shouldSucceed: false,
		},
		{
			name: "reject unknown target type",
			rule: &AccessControlRule{
				ID:            "rule4",
				TargetType:    "unknown",
				TargetPattern: "example.com",
			},
			shouldSucceed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ac := NewAccessController(false, logger)
			err := ac.AddRule(tt.rule)

			if (err == nil) != tt.shouldSucceed {
				t.Errorf("expected success=%v, got error=%v", tt.shouldSucceed, err)
			}
		})
	}
}

func TestAccessControllerRemoveRule(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	ac := NewAccessController(false, logger)

	rule := &AccessControlRule{
		ID:            "rule1",
		TargetType:    "domain",
		TargetPattern: "example.com",
		Mode:          AccessControlModeAllow,
	}

	if err := ac.AddRule(rule); err != nil {
		t.Fatalf("failed to add rule: %v", err)
	}

	if err := ac.RemoveRule("rule1"); err != nil {
		t.Errorf("failed to remove existing rule: %v", err)
	}

	if err := ac.RemoveRule("nonexistent"); err == nil {
		t.Error("expected error when removing nonexistent rule")
	}

	if ac.Count() != 0 {
		t.Errorf("expected count=0 after removal, got %d", ac.Count())
	}
}

func TestAccessControllerClear(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	ac := NewAccessController(false, logger)

	rules := []*AccessControlRule{
		{ID: "r1", TargetType: "domain", TargetPattern: "a.com", Mode: AccessControlModeAllow},
		{ID: "r2", TargetType: "ip", TargetPattern: "1.1.1.1", Mode: AccessControlModeAllow},
		{ID: "r3", TargetType: "url", TargetPattern: "/test", Mode: AccessControlModeAllow},
	}

	for _, rule := range rules {
		if err := ac.AddRule(rule); err != nil {
			t.Fatalf("failed to add rule: %v", err)
		}
	}

	if ac.Count() != 3 {
		t.Errorf("expected count=3, got %d", ac.Count())
	}

	ac.Clear()

	if ac.Count() != 0 {
		t.Errorf("expected count=0 after clear, got %d", ac.Count())
	}
}

func TestAccessControllerCount(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	ac := NewAccessController(false, logger)

	if ac.Count() != 0 {
		t.Errorf("expected count=0 initially, got %d", ac.Count())
	}

	for i := 0; i < 5; i++ {
		rule := &AccessControlRule{
			ID:            "rule" + string(rune(i)),
			TargetType:    "domain",
			TargetPattern: "example" + string(rune(i)) + ".com",
		}
		ac.AddRule(rule)
	}

	if ac.Count() != 5 {
		t.Errorf("expected count=5, got %d", ac.Count())
	}
}

func TestAccessControllerDefaultBehavior(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	tests := []struct {
		name                  string
		defaultRequireAuth    bool
		defaultAllow          bool
		svc                   *ServiceContext
		expectedAllow         bool
		expectedRequiresAuth  bool
	}{
		{
			name:                 "default allow when auth not required",
			defaultRequireAuth:   false,
			defaultAllow:         true,
			svc:                  nil,
			expectedAllow:        true,
			expectedRequiresAuth: false,
		},
		{
			name:                 "default require auth",
			defaultRequireAuth:   true,
			defaultAllow:         true,
			svc:                  nil,
			expectedAllow:        false,
			expectedRequiresAuth: true,
		},
		{
			name:                 "default require auth with authenticated service",
			defaultRequireAuth:   true,
			defaultAllow:         true,
			svc:                  &ServiceContext{Authenticated: true},
			expectedAllow:        true,
			expectedRequiresAuth: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ac := NewAccessController(tt.defaultRequireAuth, logger)
			ac.SetDefaultAllow(tt.defaultAllow)

			decision := ac.Check("example.com", "domain", tt.svc)

			if decision.Allowed != tt.expectedAllow {
				t.Errorf("expected allowed=%v, got %v", tt.expectedAllow, decision.Allowed)
			}
			if decision.RequiresAuth != tt.expectedRequiresAuth {
				t.Errorf("expected requiresAuth=%v, got %v", tt.expectedRequiresAuth, decision.RequiresAuth)
			}
		})
	}
}

func TestAccessControllerGetRules(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	ac := NewAccessController(false, logger)

	rules := []*AccessControlRule{
		{ID: "r1", TargetType: "domain", TargetPattern: "a.com"},
		{ID: "r2", TargetType: "ip", TargetPattern: "1.1.1.1"},
		{ID: "r3", TargetType: "url", TargetPattern: "/test"},
	}

	for _, rule := range rules {
		ac.AddRule(rule)
	}

	retrieved := ac.GetRules()

	if len(retrieved) != 3 {
		t.Errorf("expected 3 rules, got %d", len(retrieved))
	}

	// Verify all rule IDs are present
	idMap := make(map[string]bool)
	for _, rule := range retrieved {
		idMap[rule.ID] = true
	}

	for _, rule := range rules {
		if !idMap[rule.ID] {
			t.Errorf("rule %s not found in retrieved rules", rule.ID)
		}
	}
}
