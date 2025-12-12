package zerotrust

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// ComplianceReporter generates compliance reports for SOC2, HIPAA, PCI-DSS
type ComplianceReporter struct {
	auditLogger *AuditLogger
	logger      *logrus.Logger
}

// ComplianceReport represents a compliance report
type ComplianceReport struct {
	ReportID      string                 `json:"report_id"`
	Standard      string                 `json:"standard"` // SOC2, HIPAA, PCI-DSS
	GeneratedAt   time.Time              `json:"generated_at"`
	StartTime     time.Time              `json:"start_time"`
	EndTime       time.Time              `json:"end_time"`
	TotalEvents   int                    `json:"total_events"`
	Summary       *ComplianceSummary     `json:"summary"`
	Findings      []*ComplianceFinding   `json:"findings"`
	Recommendations []string             `json:"recommendations"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// ComplianceSummary provides high-level compliance metrics
type ComplianceSummary struct {
	AccessAttempts      int     `json:"access_attempts"`
	SuccessfulAccess    int     `json:"successful_access"`
	FailedAccess        int     `json:"failed_access"`
	FailureRate         float64 `json:"failure_rate"`
	UniqueUsers         int     `json:"unique_users"`
	UniqueServices      int     `json:"unique_services"`
	PolicyViolations    int     `json:"policy_violations"`
	CertificateIssues   int     `json:"certificate_issues"`
	ChainIntegrityValid bool    `json:"chain_integrity_valid"`
}

// ComplianceFinding represents a compliance finding or issue
type ComplianceFinding struct {
	Severity    string    `json:"severity"` // critical, high, medium, low
	Category    string    `json:"category"`
	Description string    `json:"description"`
	Count       int       `json:"count"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	Examples    []string  `json:"examples,omitempty"`
}

// NewComplianceReporter creates a new compliance reporter
func NewComplianceReporter(auditLogger *AuditLogger, logger *logrus.Logger) *ComplianceReporter {
	return &ComplianceReporter{
		auditLogger: auditLogger,
		logger:      logger,
	}
}

// GenerateSOC2Report generates a SOC2 compliance report
func (cr *ComplianceReporter) GenerateSOC2Report(startTime, endTime time.Time) (*ComplianceReport, error) {
	events, err := cr.auditLogger.GetEvents(startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve audit events: %w", err)
	}

	report := &ComplianceReport{
		ReportID:    fmt.Sprintf("SOC2-%d", time.Now().Unix()),
		Standard:    "SOC2",
		GeneratedAt: time.Now(),
		StartTime:   startTime,
		EndTime:     endTime,
		TotalEvents: len(events),
		Summary:     &ComplianceSummary{},
		Findings:    []*ComplianceFinding{},
		Recommendations: []string{},
		Metadata:    make(map[string]interface{}),
	}

	// Analyze events for SOC2 trust principles
	cr.analyzeSOC2(events, report)

	return report, nil
}

// GenerateHIPAAReport generates a HIPAA compliance report
func (cr *ComplianceReporter) GenerateHIPAAReport(startTime, endTime time.Time) (*ComplianceReport, error) {
	events, err := cr.auditLogger.GetEvents(startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve audit events: %w", err)
	}

	report := &ComplianceReport{
		ReportID:    fmt.Sprintf("HIPAA-%d", time.Now().Unix()),
		Standard:    "HIPAA",
		GeneratedAt: time.Now(),
		StartTime:   startTime,
		EndTime:     endTime,
		TotalEvents: len(events),
		Summary:     &ComplianceSummary{},
		Findings:    []*ComplianceFinding{},
		Recommendations: []string{},
		Metadata:    make(map[string]interface{}),
	}

	// Analyze events for HIPAA requirements
	cr.analyzeHIPAA(events, report)

	return report, nil
}

// GeneratePCIDSSReport generates a PCI-DSS compliance report
func (cr *ComplianceReporter) GeneratePCIDSSReport(startTime, endTime time.Time) (*ComplianceReport, error) {
	events, err := cr.auditLogger.GetEvents(startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve audit events: %w", err)
	}

	report := &ComplianceReport{
		ReportID:    fmt.Sprintf("PCIDSS-%d", time.Now().Unix()),
		Standard:    "PCI-DSS",
		GeneratedAt: time.Now(),
		StartTime:   startTime,
		EndTime:     endTime,
		TotalEvents: len(events),
		Summary:     &ComplianceSummary{},
		Findings:    []*ComplianceFinding{},
		Recommendations: []string{},
		Metadata:    make(map[string]interface{}),
	}

	// Analyze events for PCI-DSS requirements
	cr.analyzePCIDSS(events, report)

	return report, nil
}

// analyzeSOC2 analyzes events for SOC2 compliance
func (cr *ComplianceReporter) analyzeSOC2(events []*AuditEvent, report *ComplianceReport) {
	uniqueUsers := make(map[string]bool)
	uniqueServices := make(map[string]bool)
	failedAccess := 0
	successfulAccess := 0
	policyViolations := 0

	for _, event := range events {
		if event.User != "" {
			uniqueUsers[event.User] = true
		}
		if event.Service != "" {
			uniqueServices[event.Service] = true
		}

		if event.Allowed {
			successfulAccess++
		} else {
			failedAccess++
		}

		if event.EventType == "policy_evaluation" && !event.Allowed {
			policyViolations++
		}
	}

	report.Summary.AccessAttempts = len(events)
	report.Summary.SuccessfulAccess = successfulAccess
	report.Summary.FailedAccess = failedAccess
	report.Summary.UniqueUsers = len(uniqueUsers)
	report.Summary.UniqueServices = len(uniqueServices)
	report.Summary.PolicyViolations = policyViolations

	if len(events) > 0 {
		report.Summary.FailureRate = float64(failedAccess) / float64(len(events))
	}

	// Verify audit chain integrity
	valid, err := cr.auditLogger.VerifyChain()
	if err != nil {
		cr.logger.WithError(err).Error("Failed to verify audit chain")
	}
	report.Summary.ChainIntegrityValid = valid

	// SOC2 findings
	if !valid {
		report.Findings = append(report.Findings, &ComplianceFinding{
			Severity:    "critical",
			Category:    "audit_integrity",
			Description: "Audit log chain integrity compromised",
			Count:       1,
		})
	}

	if report.Summary.FailureRate > 0.10 {
		report.Findings = append(report.Findings, &ComplianceFinding{
			Severity:    "high",
			Category:    "access_control",
			Description: fmt.Sprintf("High access failure rate: %.2f%%", report.Summary.FailureRate*100),
			Count:       failedAccess,
		})
		report.Recommendations = append(report.Recommendations, "Review access control policies to reduce failure rate")
	}

	if policyViolations > 0 {
		report.Findings = append(report.Findings, &ComplianceFinding{
			Severity:    "medium",
			Category:    "policy_compliance",
			Description: "Policy violations detected",
			Count:       policyViolations,
		})
	}
}

// analyzeHIPAA analyzes events for HIPAA compliance
func (cr *ComplianceReporter) analyzeHIPAA(events []*AuditEvent, report *ComplianceReport) {
	// HIPAA requires comprehensive audit logs for PHI access
	uniqueUsers := make(map[string]bool)
	unauthorizedAccess := 0
	missingAuthentication := 0

	for _, event := range events {
		if event.User != "" {
			uniqueUsers[event.User] = true
		}

		if !event.Allowed {
			unauthorizedAccess++
		}

		if event.User == "" && event.Service == "" {
			missingAuthentication++
		}
	}

	report.Summary.AccessAttempts = len(events)
	report.Summary.UniqueUsers = len(uniqueUsers)
	report.Summary.FailedAccess = unauthorizedAccess

	// HIPAA findings
	if missingAuthentication > 0 {
		report.Findings = append(report.Findings, &ComplianceFinding{
			Severity:    "critical",
			Category:    "authentication",
			Description: "Access attempts without proper authentication",
			Count:       missingAuthentication,
		})
		report.Recommendations = append(report.Recommendations, "Enforce authentication for all PHI access")
	}

	if unauthorizedAccess > 0 {
		report.Findings = append(report.Findings, &ComplianceFinding{
			Severity:    "high",
			Category:    "unauthorized_access",
			Description: "Unauthorized access attempts detected",
			Count:       unauthorizedAccess,
		})
		report.Recommendations = append(report.Recommendations, "Investigate and address unauthorized access attempts")
	}

	// Verify audit trail completeness
	valid, _ := cr.auditLogger.VerifyChain()
	report.Summary.ChainIntegrityValid = valid

	if !valid {
		report.Findings = append(report.Findings, &ComplianceFinding{
			Severity:    "critical",
			Category:    "audit_trail",
			Description: "Audit trail integrity compromised - HIPAA violation",
			Count:       1,
		})
		report.Recommendations = append(report.Recommendations, "Restore audit trail integrity immediately")
	}
}

// analyzePCIDSS analyzes events for PCI-DSS compliance
func (cr *ComplianceReporter) analyzePCIDSS(events []*AuditEvent, report *ComplianceReport) {
	// PCI-DSS requires strict access controls and monitoring
	uniqueUsers := make(map[string]bool)
	failedAccess := 0
	suspiciousActivity := 0
	multipleFailures := make(map[string]int)

	for _, event := range events {
		if event.User != "" {
			uniqueUsers[event.User] = true
		}

		if !event.Allowed {
			failedAccess++
			multipleFailures[event.User]++
		}
	}

	// Check for repeated failed access (potential brute force)
	for user, failures := range multipleFailures {
		if failures > 5 {
			suspiciousActivity++
			report.Findings = append(report.Findings, &ComplianceFinding{
				Severity:    "high",
				Category:    "suspicious_activity",
				Description: fmt.Sprintf("Multiple failed access attempts from user: %s", user),
				Count:       failures,
			})
		}
	}

	report.Summary.AccessAttempts = len(events)
	report.Summary.UniqueUsers = len(uniqueUsers)
	report.Summary.FailedAccess = failedAccess

	if len(events) > 0 {
		report.Summary.FailureRate = float64(failedAccess) / float64(len(events))
	}

	// PCI-DSS requires immutable audit logs
	valid, _ := cr.auditLogger.VerifyChain()
	report.Summary.ChainIntegrityValid = valid

	if !valid {
		report.Findings = append(report.Findings, &ComplianceFinding{
			Severity:    "critical",
			Category:    "audit_logging",
			Description: "Audit log tampering detected - PCI-DSS Requirement 10.5 violation",
			Count:       1,
		})
	}

	if suspiciousActivity > 0 {
		report.Recommendations = append(report.Recommendations, "Implement automated alerting for suspicious access patterns")
	}
}

// ExportReportJSON exports report to JSON file
func (cr *ComplianceReporter) ExportReportJSON(report *ComplianceReport, outputPath string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}

	cr.logger.WithFields(logrus.Fields{
		"report_id": report.ReportID,
		"standard":  report.Standard,
		"path":      outputPath,
	}).Info("Exported compliance report")

	return nil
}

// ExportReportHTML exports report to HTML file
func (cr *ComplianceReporter) ExportReportHTML(report *ComplianceReport, outputPath string) error {
	html := cr.generateHTML(report)

	if err := os.WriteFile(outputPath, []byte(html), 0644); err != nil {
		return fmt.Errorf("failed to write HTML report: %w", err)
	}

	cr.logger.WithFields(logrus.Fields{
		"report_id": report.ReportID,
		"standard":  report.Standard,
		"path":      outputPath,
	}).Info("Exported HTML compliance report")

	return nil
}

// generateHTML generates HTML representation of the report
func (cr *ComplianceReporter) generateHTML(report *ComplianceReport) string {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>%s Compliance Report - %s</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        h1 { color: #1E3A8A; }
        table { border-collapse: collapse; width: 100%%; margin: 20px 0; }
        th, td { border: 1px solid #ddd; padding: 12px; text-align: left; }
        th { background-color: #1E3A8A; color: white; }
        .critical { color: #DC2626; font-weight: bold; }
        .high { color: #EA580C; font-weight: bold; }
        .medium { color: #D97706; }
        .low { color: #16A34A; }
    </style>
</head>
<body>
    <h1>%s Compliance Report</h1>
    <p><strong>Report ID:</strong> %s</p>
    <p><strong>Generated:</strong> %s</p>
    <p><strong>Period:</strong> %s to %s</p>

    <h2>Summary</h2>
    <table>
        <tr><th>Metric</th><th>Value</th></tr>
        <tr><td>Total Events</td><td>%d</td></tr>
        <tr><td>Successful Access</td><td>%d</td></tr>
        <tr><td>Failed Access</td><td>%d</td></tr>
        <tr><td>Failure Rate</td><td>%.2f%%</td></tr>
        <tr><td>Unique Users</td><td>%d</td></tr>
        <tr><td>Policy Violations</td><td>%d</td></tr>
        <tr><td>Chain Integrity</td><td>%s</td></tr>
    </table>

    <h2>Findings</h2>
    <table>
        <tr><th>Severity</th><th>Category</th><th>Description</th><th>Count</th></tr>`,
		report.Standard,
		report.ReportID,
		report.Standard,
		report.ReportID,
		report.GeneratedAt.Format(time.RFC3339),
		report.StartTime.Format(time.RFC3339),
		report.EndTime.Format(time.RFC3339),
		report.TotalEvents,
		report.Summary.SuccessfulAccess,
		report.Summary.FailedAccess,
		report.Summary.FailureRate*100,
		report.Summary.UniqueUsers,
		report.Summary.PolicyViolations,
		func() string { if report.Summary.ChainIntegrityValid { return "Valid" } else { return "Compromised" } }(),
	)

	for _, finding := range report.Findings {
		html += fmt.Sprintf(`
        <tr>
            <td class="%s">%s</td>
            <td>%s</td>
            <td>%s</td>
            <td>%d</td>
        </tr>`,
			finding.Severity,
			finding.Severity,
			finding.Category,
			finding.Description,
			finding.Count,
		)
	}

	html += `
    </table>

    <h2>Recommendations</h2>
    <ul>`

	for _, rec := range report.Recommendations {
		html += fmt.Sprintf("<li>%s</li>", rec)
	}

	html += `
    </ul>
</body>
</html>`

	return html
}
