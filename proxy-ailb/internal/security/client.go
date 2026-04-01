// Package security provides prompt injection detection and content scanning.
package security

import (
	"log/slog"
	"regexp"
	"strings"
)

// ThreatType classifies detected security threats.
type ThreatType string

const (
	ThreatPromptInjection    ThreatType = "prompt_injection"
	ThreatJailbreak          ThreatType = "jailbreak"
	ThreatDataExtraction     ThreatType = "data_extraction"
	ThreatSystemPromptLeak   ThreatType = "system_prompt_leak"
	ThreatCredentialHarvest  ThreatType = "credential_harvesting"
)

// Action defines the response to a detected threat.
type Action string

const (
	ActionLog      Action = "log"
	ActionSanitize Action = "sanitize"
	ActionBlock    Action = "block"
)

// Detection represents a detected security threat.
type Detection struct {
	ThreatType ThreatType
	Confidence float64
	Action     Action
	Message    string
}

// Scanner scans prompts for security threats using pattern matching.
type Scanner struct {
	policy   string
	patterns map[ThreatType][]*regexp.Regexp
	actions  map[ThreatType]Action
}

// NewScanner creates a prompt security scanner with the given policy.
// Supported policies: "strict", "balanced", "permissive".
func NewScanner(policy string) *Scanner {
	s := &Scanner{
		policy:   policy,
		patterns: make(map[ThreatType][]*regexp.Regexp),
		actions:  make(map[ThreatType]Action),
	}

	// Compile threat patterns.
	injectionPatterns := []string{
		`(?i)ignore\s+(?:previous|all|above)\s+(?:instructions?|prompts?|rules?)`,
		`(?i)forget\s+(?:previous|all|above)\s+(?:instructions?|prompts?|rules?)`,
		`(?i)system\s*:\s*(?:you\s+are\s+now|new\s+instructions?)`,
		`(?i)override\s+(?:previous|all|system)\s+(?:instructions?|rules?)`,
		`(?i)disregard\s+(?:previous|all|system)\s+(?:instructions?|rules?)`,
	}
	jailbreakPatterns := []string{
		`(?i)pretend\s+(?:you\s+are|to\s+be)`,
		`(?i)bypass\s+(?:your|the)\s+(?:safety|guidelines|restrictions)`,
		`(?i)break\s+(?:your|the)\s+(?:rules|guidelines|restrictions)`,
		`(?i)you\s+(?:can|should)\s+(?:do|say|ignore)\s+anything`,
	}
	extractionPatterns := []string{
		`(?i)(?:show|tell|give|reveal)\s+me\s+(?:your|the)\s+(?:system\s+)?(?:prompt|instructions?)`,
		`(?i)what\s+(?:are\s+)?your\s+(?:initial\s+)?(?:instructions?|prompt)`,
	}
	credentialPatterns := []string{
		`(?i)(?:api[_\s]*key|password|token|secret)\s*[:=]\s*["']?[\w\-]{20,}`,
		`sk-[a-zA-Z0-9]{20,}`,
	}

	s.compilePatterns(ThreatPromptInjection, injectionPatterns)
	s.compilePatterns(ThreatJailbreak, jailbreakPatterns)
	s.compilePatterns(ThreatDataExtraction, extractionPatterns)
	s.compilePatterns(ThreatCredentialHarvest, credentialPatterns)

	// Set actions based on policy.
	switch policy {
	case "strict":
		s.actions[ThreatPromptInjection] = ActionBlock
		s.actions[ThreatJailbreak] = ActionBlock
		s.actions[ThreatDataExtraction] = ActionBlock
		s.actions[ThreatCredentialHarvest] = ActionBlock
	case "permissive":
		s.actions[ThreatPromptInjection] = ActionSanitize
		s.actions[ThreatJailbreak] = ActionLog
		s.actions[ThreatDataExtraction] = ActionSanitize
		s.actions[ThreatCredentialHarvest] = ActionBlock
	default: // balanced
		s.actions[ThreatPromptInjection] = ActionBlock
		s.actions[ThreatJailbreak] = ActionSanitize
		s.actions[ThreatDataExtraction] = ActionBlock
		s.actions[ThreatCredentialHarvest] = ActionBlock
	}

	return s
}

func (s *Scanner) compilePatterns(threat ThreatType, patterns []string) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			slog.Warn("failed to compile security pattern", "pattern", p, "error", err)
			continue
		}
		compiled = append(compiled, re)
	}
	s.patterns[threat] = compiled
}

// ScanMessages scans all messages for threats.
// Returns detections and whether the request should be blocked.
func (s *Scanner) ScanMessages(messages []struct{ Role, Content string }) ([]Detection, bool) {
	var detections []Detection
	shouldBlock := false

	for _, msg := range messages {
		dets := s.scanText(msg.Content)
		detections = append(detections, dets...)
		for _, d := range dets {
			if d.Action == ActionBlock {
				shouldBlock = true
			}
		}
	}

	return detections, shouldBlock
}

// ScanPrompt scans a single prompt string for threats.
func (s *Scanner) ScanPrompt(prompt string) ([]Detection, bool) {
	dets := s.scanText(prompt)
	shouldBlock := false
	for _, d := range dets {
		if d.Action == ActionBlock {
			shouldBlock = true
		}
	}
	return dets, shouldBlock
}

func (s *Scanner) scanText(text string) []Detection {
	var detections []Detection

	for threatType, patterns := range s.patterns {
		matchCount := 0
		for _, re := range patterns {
			if re.MatchString(text) {
				matchCount++
			}
		}

		threshold := 1
		if s.policy == "permissive" {
			threshold = 2
		}

		if matchCount >= threshold {
			confidence := float64(matchCount) / 5.0
			if confidence > 1.0 {
				confidence = 1.0
			}

			action := s.actions[threatType]
			detections = append(detections, Detection{
				ThreatType: threatType,
				Confidence: confidence,
				Action:     action,
				Message:    string(threatType) + ": " + strings.Repeat("*", matchCount) + " pattern(s) matched",
			})

			slog.Warn("security threat detected",
				"type", threatType,
				"confidence", confidence,
				"action", action,
				"matches", matchCount,
			)
		}
	}

	return detections
}
