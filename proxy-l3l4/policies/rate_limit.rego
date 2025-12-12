# Rate Limiting Policy for MarchProxy
# Implements sophisticated rate limiting rules with burst handling

package marchproxy.rate_limit

import rego.v1

# Default rate limit configuration
default_rate_limit := {
    "requests_per_second": 100,
    "requests_per_minute": 1000,
    "burst_size": 50,
}

# Get rate limit for specific service
rate_limit contains result if {
    input.service != ""
    service_config := data.rate_limits[input.service]
    service_config != null
    result := service_config
}

# Apply default rate limit if no specific configuration
rate_limit contains default_rate_limit if {
    input.service != ""
    data.rate_limits[input.service] == null
}

# IP-based rate limiting
ip_rate_limit contains result if {
    input.source_ip != ""
    ip_config := data.ip_rate_limits[input.source_ip]
    ip_config != null
    result := ip_config
}

# Stricter limits for unauthenticated requests
unauthenticated_limit contains result if {
    input.user == ""
    input.service == ""
    result := {
        "requests_per_second": 10,
        "requests_per_minute": 100,
        "burst_size": 5,
    }
}

# Priority-based rate limiting
priority_limit contains result if {
    input.metadata.priority != null
    priority := input.metadata.priority
    result := data.priority_limits[priority]
}

# Block if over rate limit
deny if {
    input.metadata.current_rate != null
    limit := rate_limit[_]
    limit.requests_per_second != null
    input.metadata.current_rate > limit.requests_per_second
}

# Warning if approaching rate limit
warning contains msg if {
    input.metadata.current_rate != null
    limit := rate_limit[_]
    limit.requests_per_second != null
    threshold := limit.requests_per_second * 0.8
    input.metadata.current_rate > threshold
    msg := sprintf("Approaching rate limit: %d/%d requests per second", [
        input.metadata.current_rate,
        limit.requests_per_second
    ])
}

# Allow with rate limit info
allow := {
    "allowed": true,
    "rate_limit": rate_limit[_],
    "warnings": warning,
}
