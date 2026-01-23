# Rate Limiting Architecture

## Overview

The rate limiting system uses a **sliding window algorithm** to provide accurate, distributed-ready rate limiting for the AILB proxy.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         Client Request                          │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                     FastAPI Application                          │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │               RateLimitMiddleware                         │  │
│  │  1. Extract API key from request                         │  │
│  │  2. Check rate limits (pre-request)                      │  │
│  │  3. Process request if allowed                           │  │
│  │  4. Record token usage (post-request)                    │  │
│  │  5. Add X-RateLimit-* headers                            │  │
│  └───────────────┬───────────────────────────────────────────┘  │
│                  │                                               │
│                  ▼                                               │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                    RateLimiter                            │  │
│  │                                                           │  │
│  │  ┌─────────────────────────────────────────────────────┐ │  │
│  │  │           Sliding Window Algorithm                  │ │  │
│  │  │  • Track requests in time window                    │ │  │
│  │  │  • Count tokens and requests                        │ │  │
│  │  │  • Automatic cleanup of old entries                 │ │  │
│  │  └─────────────────────────────────────────────────────┘ │  │
│  │                                                           │  │
│  │  ┌─────────────────────────────────────────────────────┐ │  │
│  │  │         Per-Key Configuration                       │ │  │
│  │  │  • TPM (Tokens Per Minute) limits                   │ │  │
│  │  │  • RPM (Requests Per Minute) limits                 │ │  │
│  │  │  • Window size (seconds)                            │ │  │
│  │  │  • Enable/disable flag                              │ │  │
│  │  └─────────────────────────────────────────────────────┘ │  │
│  │                                                           │  │
│  └───────────────┬───────────────────────────────────────────┘  │
│                  │                                               │
└──────────────────┼───────────────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Storage Backend                               │
│                                                                  │
│  ┌──────────────────────┐         ┌────────────────────────┐    │
│  │   In-Memory Store    │         │   Redis (Optional)     │    │
│  │  • Fast access       │ ◄────► │  • Distributed         │    │
│  │  • Thread-safe       │         │  • Persistent          │    │
│  │  • Development mode  │         │  • Production mode     │    │
│  └──────────────────────┘         └────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

## Component Flow

### Request Flow (Successful)

```
1. Client Request
   ↓
2. RateLimitMiddleware.dispatch()
   ├─> Extract API key from headers
   ├─> Check if path is exempt
   └─> Get rate limit config for key
   ↓
3. RateLimiter.check_limit(key_id, tokens=0)
   ├─> Get or create window for key
   ├─> Clean up expired requests
   ├─> Calculate current TPM/RPM
   ├─> Compare against limits
   └─> Return (allowed=True, status)
   ↓
4. Process Request (call_next)
   ↓
5. Extract token usage from response
   ↓
6. RateLimiter.record_request(key_id, tokens)
   ├─> Add request record to window
   ├─> Update token/request counters
   └─> Persist to Redis (if available)
   ↓
7. Add rate limit headers to response
   ↓
8. Return response to client
```

### Request Flow (Rate Limited)

```
1. Client Request
   ↓
2. RateLimitMiddleware.dispatch()
   ↓
3. RateLimiter.check_limit(key_id, tokens=0)
   ├─> Window shows TPM or RPM exceeded
   └─> Return (allowed=False, status)
   ↓
4. Create 429 Response
   ├─> Calculate Retry-After
   ├─> Build error JSON
   └─> Add rate limit headers
   ↓
5. Return 429 to client
```

## Data Structures

### WindowData

```python
@dataclass
class WindowData:
    requests: deque[RequestRecord]  # Sliding window of requests
    total_tokens: int               # Sum of tokens in window
    total_requests: int             # Count of requests in window
```

### RequestRecord

```python
@dataclass
class RequestRecord:
    timestamp: float  # Unix timestamp
    tokens: int       # Token count for this request
```

### Storage Structure

#### In-Memory

```python
{
    "api-key-123": WindowData(
        requests=deque([
            RequestRecord(timestamp=1702742400.0, tokens=100),
            RequestRecord(timestamp=1702742410.0, tokens=150),
            RequestRecord(timestamp=1702742420.0, tokens=200),
        ]),
        total_tokens=450,
        total_requests=3
    ),
    "api-key-456": WindowData(...)
}
```

#### Redis (Future)

```
Key: ailb:ratelimit:window:api-key-123
Type: Sorted Set
Members:
  1702742400.0 -> {"tokens": 100}
  1702742410.0 -> {"tokens": 150}
  1702742420.0 -> {"tokens": 200}

Key: ailb:ratelimit:config:api-key-123
Type: Hash
Fields:
  tpm_limit: 10000
  rpm_limit: 60
  window_seconds: 60
  enabled: 1
```

## Sliding Window Algorithm

### Visual Representation

```
Current Time: 12:00:45
Window Size: 60 seconds
Window Start: 11:59:45

Timeline:
11:59:30  11:59:45  12:00:00  12:00:15  12:00:30  12:00:45
    X        |         X         X         X         NOW
  (expired)  |       (kept)    (kept)    (kept)
             |
          Window Start

Cleanup Process:
1. Calculate window_start = now - window_seconds
2. Remove all requests where timestamp < window_start
3. Update total_tokens and total_requests
```

### Algorithm Steps

```python
def check_limit(key_id, tokens):
    now = time.time()
    window_start = now - window_seconds

    # 1. Get or create window
    window = get_window(key_id)

    # 2. Clean up expired requests
    while window.requests and window.requests[0].timestamp < window_start:
        old = window.requests.popleft()
        window.total_tokens -= old.tokens
        window.total_requests -= 1

    # 3. Check limits
    would_exceed_tpm = (window.total_tokens + tokens) > tpm_limit
    would_exceed_rpm = (window.total_requests + 1) > rpm_limit

    # 4. Determine if allowed
    allowed = not (would_exceed_tpm or would_exceed_rpm)

    return allowed
```

## Thread Safety

### Lock Strategy

```python
class RateLimiter:
    def __init__(self):
        self._lock = Lock()  # Thread-safe access to windows

    def check_limit(self, key_id, tokens):
        with self._lock:
            # All window operations are atomic
            window = self._windows.get(key_id)
            self._cleanup_window(window)
            # ... check and return
```

### Concurrent Access Pattern

```
Thread 1: check_limit("key-1") ─┐
Thread 2: check_limit("key-2") ─┤─► Parallel (different keys)
Thread 3: check_limit("key-3") ─┘

Thread 4: check_limit("key-1") ─┐
Thread 5: check_limit("key-1") ─┤─► Sequential (same key)
Thread 6: record_request("key-1")┘
```

## Performance Characteristics

### Time Complexity

- `check_limit()`: O(n) where n = requests in window (typically < 100)
- `record_request()`: O(1)
- `get_status()`: O(n) where n = requests in window
- `cleanup_window()`: O(m) where m = expired requests

### Space Complexity

- Per key: O(r) where r = max requests in window
- Total: O(k * r) where k = number of active keys

### Optimization Strategies

1. **Window Size**: Smaller windows = fewer requests to track
2. **Cleanup**: Remove expired requests proactively
3. **Redis**: Offload storage for distributed systems
4. **TTL**: Automatic cleanup of inactive keys

## Configuration Patterns

### Tiered Rate Limits

```python
# Free Tier
free_config = RateLimitConfig(
    tpm_limit=1000,
    rpm_limit=10,
    window_seconds=60
)

# Pro Tier
pro_config = RateLimitConfig(
    tpm_limit=10000,
    rpm_limit=60,
    window_seconds=60
)

# Enterprise Tier
enterprise_config = RateLimitConfig(
    tpm_limit=100000,
    rpm_limit=500,
    window_seconds=60
)

# Internal/Admin
admin_config = RateLimitConfig(
    tpm_limit=0,  # Unlimited
    rpm_limit=0,
    enabled=False
)
```

### Dynamic Configuration

```python
# Load from database
def load_rate_limit_for_key(api_key):
    user = lookup_user_by_key(api_key)
    tier = user.subscription_tier

    if tier == "free":
        return free_config
    elif tier == "pro":
        return pro_config
    elif tier == "enterprise":
        return enterprise_config
    else:
        return default_config
```

## Monitoring and Observability

### Metrics to Track

```python
# Rate limit hits
rate_limit_hits_total{api_key="...", limit_type="tpm"}
rate_limit_hits_total{api_key="...", limit_type="rpm"}

# Current usage
rate_limit_usage_current{api_key="...", metric="tpm"}
rate_limit_usage_current{api_key="...", metric="rpm"}

# Remaining capacity
rate_limit_remaining{api_key="...", metric="tpm"}
rate_limit_remaining{api_key="...", metric="rpm"}

# Window statistics
rate_limit_window_size_seconds{api_key="..."}
rate_limit_active_windows_total
```

### Alerts

```yaml
# Alert when rate limits are frequently exceeded
- alert: HighRateLimitHits
  expr: rate(rate_limit_hits_total[5m]) > 10
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "High rate limit hits for {{ $labels.api_key }}"

# Alert when usage is consistently high
- alert: HighRateLimitUsage
  expr: rate_limit_usage_current / rate_limit_limit > 0.9
  for: 10m
  labels:
    severity: info
  annotations:
    summary: "Rate limit usage above 90% for {{ $labels.api_key }}"
```

## Security Considerations

### API Key Extraction

```python
# Secure extraction order
1. Authorization: Bearer <key>  # Standard OAuth2/OpenAI style
2. X-API-Key: <key>             # Alternative header
3. ?api_key=<key>               # Query param (less secure, use cautiously)
```

### Key Masking in Logs

```python
# Always mask API keys in logs
logger.info("Rate limit check for key %s", api_key[:8] + "...")
# Output: "Rate limit check for key mp-abcd1..."
```

### Exempt Paths

```python
# Always exempt these from rate limiting
exempt_paths = [
    "/healthz",      # Health checks
    "/metrics",      # Prometheus metrics
    "/docs",         # API documentation
    "/openapi.json", # OpenAPI spec
]
```

## Future Enhancements

1. **Redis Backend**: Distributed rate limiting across multiple instances
2. **Burst Allowances**: Allow short bursts above limits
3. **Predictive Limits**: Warn users before hitting limits
4. **Cost-Based Limits**: Rate limit based on estimated costs
5. **Geographic Limits**: Different limits per region
6. **Time-Based Limits**: Different limits by time of day
7. **Adaptive Limits**: Automatically adjust based on system load
8. **Rate Limit Bypass**: Special tokens for urgent requests
