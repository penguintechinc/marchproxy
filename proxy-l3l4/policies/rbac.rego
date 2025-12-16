# RBAC Policy for MarchProxy Zero-Trust
# Implements role-based access control with fine-grained permissions

package marchproxy.rbac

import rego.v1

# Default deny
default allow := false

# Allow if user has required role and permissions
allow if {
    input.user != ""
    user_roles := data.users[input.user].roles
    some role in user_roles
    role_permissions := data.roles[role].permissions
    required_permission := concat(":", [input.action, input.resource])
    required_permission in role_permissions
}

# Allow if service has required role
allow if {
    input.service != ""
    service_roles := data.services[input.service].roles
    some role in service_roles
    role_permissions := data.roles[role].permissions
    required_permission := concat(":", [input.action, input.resource])
    required_permission in role_permissions
}

# Allow for admin role (wildcard permissions)
allow if {
    input.user != ""
    user_roles := data.users[input.user].roles
    "admin" in user_roles
}

# Certificate-based authentication
allow if {
    input.certificate != null
    cert_cn := input.certificate.subject
    cert_valid := time.now_ns() > time.parse_rfc3339_ns(input.certificate.not_before)
    cert_not_expired := time.now_ns() < time.parse_rfc3339_ns(input.certificate.not_after)
    cert_valid
    cert_not_expired
    service_authorized := data.services[cert_cn] != null
    service_authorized
}

# Deny if source IP is blacklisted
deny if {
    input.source_ip != ""
    input.source_ip in data.blacklisted_ips
}

# Audit trail requirement
annotations contains {"audit_required": true} if {
    allow
}

annotations contains {"deny_reason": reason} if {
    not allow
    reason := "access denied - insufficient permissions"
}

# Rate limiting check
rate_limit contains result if {
    input.service != ""
    service_limits := data.rate_limits[input.service]
    service_limits != null
    result := {
        "limit": service_limits.requests_per_minute,
        "window": "1m",
    }
}
