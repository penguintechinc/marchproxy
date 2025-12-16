# Compliance Policy for MarchProxy
# Validates compliance with SOC2, HIPAA, PCI-DSS requirements

package marchproxy.compliance

import rego.v1

# SOC2 Compliance Checks
soc2_compliant if {
    # CC6.1: Logical and physical access controls
    authentication_required
    audit_trail_intact
    encryption_enabled
}

# HIPAA Compliance Checks
hipaa_compliant if {
    # 164.312(a)(1): Access control
    authentication_required
    # 164.312(b): Audit controls
    audit_trail_intact
    # 164.312(e)(1): Transmission security
    encryption_enabled
    # 164.308(a)(5)(ii)(C): Log-in monitoring
    login_monitoring_enabled
}

# PCI-DSS Compliance Checks
pci_dss_compliant if {
    # Requirement 8: Identify and authenticate access
    strong_authentication
    # Requirement 10: Track and monitor all access
    audit_trail_intact
    comprehensive_logging
    # Requirement 10.5: Secure audit trails
    audit_trail_immutable
}

# Authentication required for all access
authentication_required if {
    input.user != ""
}

authentication_required if {
    input.service != ""
}

authentication_required if {
    input.certificate != null
}

# Strong authentication (2FA, mTLS, etc.)
strong_authentication if {
    input.metadata.auth_method == "mtls"
}

strong_authentication if {
    input.metadata.auth_method == "jwt"
    input.metadata.jwt_validated == true
}

# Audit trail integrity
audit_trail_intact if {
    input.metadata.audit_chain_valid == true
}

# Audit trail immutability
audit_trail_immutable if {
    input.metadata.audit_append_only == true
}

# Encryption enabled
encryption_enabled if {
    input.metadata.tls_enabled == true
}

# Comprehensive logging
comprehensive_logging if {
    input.metadata.logging_level == "debug"
}

comprehensive_logging if {
    input.metadata.logging_level == "info"
}

# Login monitoring enabled
login_monitoring_enabled if {
    input.metadata.auth_logging == true
}

# Compliance violations
violations contains violation if {
    not authentication_required
    violation := {
        "severity": "critical",
        "standard": "all",
        "requirement": "authentication",
        "description": "Unauthenticated access attempt",
    }
}

violations contains violation if {
    not audit_trail_intact
    violation := {
        "severity": "critical",
        "standard": "all",
        "requirement": "audit_integrity",
        "description": "Audit trail integrity compromised",
    }
}

violations contains violation if {
    not encryption_enabled
    violation := {
        "severity": "high",
        "standard": "all",
        "requirement": "encryption",
        "description": "Unencrypted transmission",
    }
}

violations contains violation if {
    input.metadata.failed_login_attempts > 5
    violation := {
        "severity": "high",
        "standard": "PCI-DSS",
        "requirement": "access_control",
        "description": "Excessive failed login attempts",
    }
}

# Deny if there are critical violations
deny if {
    some violation in violations
    violation.severity == "critical"
}

# Allow with compliance status
allow := {
    "allowed": count(violations) == 0,
    "soc2_compliant": soc2_compliant,
    "hipaa_compliant": hipaa_compliant,
    "pci_dss_compliant": pci_dss_compliant,
    "violations": violations,
}
