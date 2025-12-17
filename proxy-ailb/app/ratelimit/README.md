# Rate Limiting Module

Token-based and request-based rate limiting for the AILB proxy using a sliding window algorithm.

## Features

- **Sliding Window Algorithm**: Accurate rate limiting without burst allowances
- **Dual Limits**: Both TPM (Tokens Per Minute) and RPM (Requests Per Minute)
- **Per-Key Configuration**: Different limits for different API keys
- **FastAPI Middleware**: Automatic rate limiting with proper HTTP headers
- **Thread-Safe**: Safe for concurrent requests
- **429 Responses**: Standard HTTP rate limit exceeded responses with Retry-After
- **Transparent Headers**: X-RateLimit-* headers on all responses

## Components

### 1. models.py

Pydantic models for configuration and status:

```python
from app.ratelimit import RateLimitConfig, RateLimitStatus

# Configure rate limits
config = RateLimitConfig(
    tpm_limit=10000,      # 10k tokens per minute
    rpm_limit=60,         # 60 requests per minute
    window_seconds=60,    # 60 second window
    enabled=True
)

# Check status
status = RateLimitStatus(
    current_tpm=5000,
    current_rpm=30,
    reset_at=datetime.utcnow(),
    is_limited=False,
    remaining_tpm=5000,
    remaining_rpm=30
)
```

### 2. limiter.py

Core rate limiting logic with sliding window:

```python
from app.ratelimit import RateLimiter, RateLimitConfig

# Initialize limiter
limiter = RateLimiter(redis_client=None)

# Set custom config for an API key
config = RateLimitConfig(tpm_limit=20000, rpm_limit=100)
limiter.set_config("api-key-123", config)

# Check if request is allowed
allowed, status = limiter.check_limit("api-key-123", tokens=500)
if not allowed:
    print(f"Rate limited: {status.limit_reason}")

# Record successful request
limiter.record_request("api-key-123", tokens=500)

# Get current status
status = limiter.get_status("api-key-123")
print(f"Current TPM: {status.current_tpm}/{config.tpm_limit}")
print(f"Current RPM: {status.current_rpm}/{config.rpm_limit}")

# Reset limits for a key
limiter.reset("api-key-123")
```

### 3. middleware.py

FastAPI middleware for automatic rate limiting:

```python
from fastapi import FastAPI
from app.ratelimit import RateLimiter, RateLimitMiddleware

app = FastAPI()
limiter = RateLimiter()

# Add middleware
app.add_middleware(
    RateLimitMiddleware,
    limiter=limiter,
    exempt_paths=["/healthz", "/metrics", "/docs"]
)

# Middleware automatically:
# - Extracts API key from Authorization/X-API-Key header
# - Checks rate limits before request
# - Records token usage after request
# - Returns 429 if rate limited
# - Adds X-RateLimit-* headers to all responses
```

## Integration with main.py

Add to your FastAPI application:

```python
from app.ratelimit import RateLimiter, RateLimitMiddleware, RateLimitConfig

# In AILBServer.__init__()
self.rate_limiter = RateLimiter(
    redis_client=None,  # TODO: Pass Redis client when available
    config={
        'default_tpm_limit': 10000,
        'default_rpm_limit': 60,
        'default_window_seconds': 60,
        'rate_limiting_enabled': True
    }
)

# In FastAPI app setup (after app creation)
app.add_middleware(
    RateLimitMiddleware,
    limiter=ailb_server.rate_limiter,
    exempt_paths=["/healthz", "/metrics"]
)
```

## API Key Extraction

The middleware extracts API keys from:

1. **Authorization Header**: `Authorization: Bearer <api-key>`
2. **X-API-Key Header**: `X-API-Key: <api-key>`
3. **Query Parameter**: `?api_key=<api-key>`

## Response Headers

All responses include rate limit headers:

```
X-RateLimit-Limit-TPM: 10000
X-RateLimit-Limit-RPM: 60
X-RateLimit-Remaining-TPM: 5000
X-RateLimit-Remaining-RPM: 30
X-RateLimit-Reset: 2025-12-16T12:01:00Z
X-RateLimit-Window: 60s
```

## 429 Response Format

When rate limited:

```json
{
  "error": {
    "message": "Rate limit exceeded: tpm_exceeded",
    "type": "rate_limit_exceeded",
    "code": "tpm_exceeded"
  },
  "current_tpm": 10500,
  "current_rpm": 45,
  "limit_tpm": 10000,
  "limit_rpm": 60,
  "reset_at": "2025-12-16T12:01:00Z",
  "retry_after_seconds": 30
}
```

Headers:
```
HTTP/1.1 429 Too Many Requests
Retry-After: 30
X-RateLimit-Limit-TPM: 10000
X-RateLimit-Remaining-TPM: 0
...
```

## Sliding Window Algorithm

The rate limiter uses a sliding window algorithm:

1. **Window**: Tracks requests in the last N seconds (configurable)
2. **Cleanup**: Automatically removes requests older than window
3. **Accurate**: No burst allowances at window boundaries
4. **Efficient**: O(1) for most operations, O(n) for cleanup (n = requests in window)

Example with 60-second window:
```
Time:  12:00:00  12:00:30  12:01:00  12:01:30
       |---------|---------|---------|
       [  Window moves continuously  ]
```

## Configuration Options

### Per-Key Configuration

```python
# High-tier customer
premium_config = RateLimitConfig(
    tpm_limit=50000,
    rpm_limit=200,
    window_seconds=60,
    enabled=True
)
limiter.set_config("premium-key-abc", premium_config)

# Free tier
free_config = RateLimitConfig(
    tpm_limit=1000,
    rpm_limit=10,
    window_seconds=60,
    enabled=True
)
limiter.set_config("free-key-xyz", free_config)

# Unlimited (for testing)
unlimited_config = RateLimitConfig(
    tpm_limit=0,  # 0 = unlimited
    rpm_limit=0,
    window_seconds=60,
    enabled=False
)
limiter.set_config("internal-key-test", unlimited_config)
```

### Global Configuration

```python
limiter = RateLimiter(config={
    'default_tpm_limit': 10000,
    'default_rpm_limit': 60,
    'default_window_seconds': 60,
    'rate_limiting_enabled': True
})
```

## TODO: Redis Backend

Currently uses in-memory storage. For production distributed systems:

```python
import redis

redis_client = redis.Redis(
    host='localhost',
    port=6379,
    decode_responses=True
)

limiter = RateLimiter(redis_client=redis_client)
```

Redis keys structure:
```
ailb:ratelimit:config:{key_id}     # Configuration
ailb:ratelimit:window:{key_id}     # Request records (sorted set)
```

## Statistics and Monitoring

```python
# Get limiter statistics
stats = limiter.get_stats()
print(f"Total tracked keys: {stats['total_tracked_keys']}")
print(f"Active keys: {stats['active_keys']}")

# Cleanup expired windows
limiter.cleanup_expired_windows(max_idle_seconds=3600)
```

## Best Practices

1. **Set Realistic Limits**: Based on your backend capacity
2. **Different Tiers**: Use different configs for different customer tiers
3. **Monitor Usage**: Track rate limit hits to adjust limits
4. **Exempt Internal**: Exempt health checks and metrics endpoints
5. **Redis for Production**: Use Redis for distributed rate limiting
6. **Cleanup Regularly**: Run cleanup_expired_windows() periodically
7. **Log Rate Limits**: Monitor which keys are hitting limits
8. **Graceful Degradation**: Handle Redis failures gracefully

## Example: Complete Integration

```python
# main.py
from app.ratelimit import RateLimiter, RateLimitMiddleware, RateLimitConfig

class AILBServer:
    def __init__(self):
        # ... existing init ...

        # Initialize rate limiter
        self.rate_limiter = RateLimiter(
            redis_client=None,  # TODO: Add Redis
            config={
                'default_tpm_limit': 10000,
                'default_rpm_limit': 60,
                'rate_limiting_enabled': True
            }
        )

    async def startup(self):
        # ... existing startup ...

        # Configure rate limits for different tiers
        # Premium tier
        self.rate_limiter.set_config(
            "premium-key-123",
            RateLimitConfig(tpm_limit=50000, rpm_limit=200)
        )

        # Free tier
        self.rate_limiter.set_config(
            "free-key-456",
            RateLimitConfig(tpm_limit=1000, rpm_limit=10)
        )

# Add middleware to app
app.add_middleware(
    RateLimitMiddleware,
    limiter=ailb_server.rate_limiter,
    exempt_paths=["/healthz", "/metrics", "/docs"]
)

# Add rate limit status endpoint
@app.get("/api/ratelimit/status")
async def get_rate_limit_status(api_key: str):
    status = ailb_server.rate_limiter.get_status(api_key)
    return status
```
