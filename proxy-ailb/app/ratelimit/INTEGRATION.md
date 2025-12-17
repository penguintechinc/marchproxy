# Rate Limiting Integration Guide

## Quick Start

### 1. Add Rate Limiter to AILBServer

Edit `/home/penguin/code/MarchProxy/proxy-ailb/main.py`:

```python
# Add import at top
from app.ratelimit import RateLimiter, RateLimitMiddleware, RateLimitConfig

# In AILBServer.__init__()
class AILBServer:
    def __init__(self):
        # ... existing initialization ...

        # Initialize rate limiter
        self.rate_limiter = RateLimiter(
            redis_client=None,  # TODO: Add Redis when available
            config={
                'default_tpm_limit': int(os.getenv('DEFAULT_TPM_LIMIT', '10000')),
                'default_rpm_limit': int(os.getenv('DEFAULT_RPM_LIMIT', '60')),
                'default_window_seconds': int(os.getenv('RATE_LIMIT_WINDOW', '60')),
                'rate_limiting_enabled': os.getenv('RATE_LIMITING_ENABLED', 'true').lower() == 'true'
            }
        )
```

### 2. Add Middleware to FastAPI App

```python
# After app creation (around line 160)
app = FastAPI(
    title="AILB - AI Load Balancer",
    description="Intelligent AI/LLM proxy with routing, memory, and RAG support",
    version="1.0.0",
    lifespan=lifespan
)

# Add rate limiting middleware AFTER CORS middleware
app.add_middleware(
    RateLimitMiddleware,
    limiter=ailb_server.rate_limiter,
    exempt_paths=[
        "/healthz",
        "/metrics",
        "/docs",
        "/openapi.json",
        "/redoc"
    ]
)
```

### 3. Add Rate Limit Management Endpoints

Add these endpoints to `main.py`:

```python
from app.ratelimit import RateLimitConfig

@app.get("/api/ratelimit/status/{api_key}")
async def get_rate_limit_status(api_key: str):
    """Get current rate limit status for an API key"""
    try:
        status = ailb_server.rate_limiter.get_status(api_key)
        config = ailb_server.rate_limiter.get_config(api_key)

        return {
            "api_key": api_key[:8] + "...",  # Masked for security
            "status": status.dict(),
            "config": config.dict()
        }
    except Exception as e:
        logger.error("Failed to get rate limit status", error=str(e))
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/api/ratelimit/config/{api_key}")
async def set_rate_limit_config(
    api_key: str,
    config: RateLimitConfig,
    authorization: Optional[str] = Header(None)
):
    """Set rate limit configuration for an API key (admin only)"""
    # TODO: Add proper authorization check
    try:
        ailb_server.rate_limiter.set_config(api_key, config)
        return {
            "message": "Rate limit configuration updated",
            "api_key": api_key[:8] + "...",
            "config": config.dict()
        }
    except Exception as e:
        logger.error("Failed to set rate limit config", error=str(e))
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/api/ratelimit/reset/{api_key}")
async def reset_rate_limit(
    api_key: str,
    authorization: Optional[str] = Header(None)
):
    """Reset rate limits for an API key (admin only)"""
    # TODO: Add proper authorization check
    try:
        ailb_server.rate_limiter.reset(api_key)
        return {
            "message": "Rate limits reset",
            "api_key": api_key[:8] + "..."
        }
    except Exception as e:
        logger.error("Failed to reset rate limits", error=str(e))
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/api/ratelimit/stats")
async def get_rate_limit_stats():
    """Get overall rate limiting statistics"""
    try:
        stats = ailb_server.rate_limiter.get_stats()
        return stats
    except Exception as e:
        logger.error("Failed to get rate limit stats", error=str(e))
        raise HTTPException(status_code=500, detail=str(e))
```

### 4. Update Environment Variables

Add to `.env.example` and `.env`:

```bash
# Rate Limiting Configuration
RATE_LIMITING_ENABLED=true
DEFAULT_TPM_LIMIT=10000
DEFAULT_RPM_LIMIT=60
RATE_LIMIT_WINDOW=60
```

### 5. Integration with RBAC (Optional)

If you want different rate limits based on user roles, integrate with the RBAC system:

```python
from app.auth.rbac import RBACManager, Role

# In AILBServer.startup()
async def startup(self):
    # ... existing startup code ...

    # Configure rate limits based on roles
    # Admin: High limits
    admin_config = RateLimitConfig(
        tpm_limit=100000,
        rpm_limit=500,
        enabled=True
    )

    # Regular users: Standard limits
    user_config = RateLimitConfig(
        tpm_limit=10000,
        rpm_limit=60,
        enabled=True
    )

    # Service accounts: Lower limits
    service_config = RateLimitConfig(
        tpm_limit=5000,
        rpm_limit=30,
        enabled=True
    )

    # These would be set dynamically based on API key -> user role lookup
    # TODO: Implement dynamic rate limit assignment based on RBAC
```

### 6. Integration with Token Manager

Update token counting to integrate with rate limiting:

```python
# In chat_completions endpoint
@app.post("/v1/chat/completions")
async def chat_completions(
    request: Request,
    authorization: Optional[str] = Header(None),
    x_preferred_model: Optional[str] = Header(None, alias="X-Preferred-Model")
):
    start_time = time.time()

    try:
        # Extract API key
        api_key = authorization.replace("Bearer ", "") if authorization else None

        # Parse request
        body = await request.json()
        messages = body.get("messages", [])
        model = body.get("model") or x_preferred_model or "gpt-3.5-turbo"

        # ... existing memory and RAG enhancement ...

        # Route request
        response_text, usage_info = await ailb_server.request_router.route_request(
            model=model,
            messages=messages,
            **{k: v for k, v in body.items() if k not in ['messages', 'model', 'session_id']}
        )

        # Record token usage in rate limiter
        if api_key:
            total_tokens = usage_info.get('total_tokens', 0)
            ailb_server.rate_limiter.record_request(api_key, tokens=total_tokens)

        # ... existing response formatting ...

    except Exception as e:
        logger.error("Chat completion failed", error=str(e))
        raise HTTPException(status_code=500, detail=str(e))
```

## Testing the Integration

### 1. Start the Server

```bash
cd /home/penguin/code/MarchProxy/proxy-ailb
python3 main.py
```

### 2. Test Basic Rate Limiting

```bash
# Make requests with an API key
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer test-api-key-123" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "Hello"}]
  }'

# Check rate limit headers in response:
# X-RateLimit-Limit-TPM: 10000
# X-RateLimit-Remaining-TPM: 9900
# X-RateLimit-Reset: 2025-12-16T12:01:00Z
```

### 3. Test Rate Limit Status

```bash
# Get current status
curl http://localhost:8080/api/ratelimit/status/test-api-key-123
```

### 4. Test Rate Limit Configuration

```bash
# Set custom configuration
curl -X POST http://localhost:8080/api/ratelimit/config/test-api-key-123 \
  -H "Content-Type: application/json" \
  -d '{
    "tpm_limit": 20000,
    "rpm_limit": 100,
    "window_seconds": 60,
    "enabled": true
  }'
```

### 5. Test Rate Limit Exceeded

```bash
# Make many requests quickly to hit the limit
for i in {1..100}; do
  curl -X POST http://localhost:8080/v1/chat/completions \
    -H "Authorization: Bearer test-api-key-123" \
    -H "Content-Type: application/json" \
    -d '{"model": "gpt-3.5-turbo", "messages": [{"role": "user", "content": "Test"}]}'
done

# Should eventually get 429 response:
# {
#   "error": {
#     "message": "Rate limit exceeded: rpm_exceeded",
#     "type": "rate_limit_exceeded"
#   },
#   "retry_after_seconds": 30
# }
```

## Monitoring Rate Limits

### 1. Prometheus Metrics (TODO)

Add to `/metrics` endpoint:

```python
from prometheus_client import Counter, Gauge, Histogram

# Define metrics
rate_limit_hits = Counter(
    'ailb_rate_limit_hits_total',
    'Total rate limit hits',
    ['api_key', 'limit_type']
)

rate_limit_remaining = Gauge(
    'ailb_rate_limit_remaining',
    'Remaining rate limit capacity',
    ['api_key', 'limit_type']
)
```

### 2. Logging

Rate limit events are automatically logged:

```
2025-12-16 12:00:00 [WARNING] Rate limit exceeded for key test-api: tpm_exceeded (TPM: 10500/10000, RPM: 45/60)
```

### 3. Statistics Endpoint

```bash
curl http://localhost:8080/api/ratelimit/stats

# Response:
{
  "total_tracked_keys": 50,
  "total_configured_keys": 25,
  "active_keys": 15,
  "default_config": {
    "tpm_limit": 10000,
    "rpm_limit": 60,
    "window_seconds": 60,
    "enabled": true
  }
}
```

## Production Considerations

### 1. Redis Backend

For production with multiple AILB instances:

```python
import redis

redis_client = redis.Redis(
    host=os.getenv('REDIS_HOST', 'localhost'),
    port=int(os.getenv('REDIS_PORT', '6379')),
    db=int(os.getenv('REDIS_DB', '0')),
    decode_responses=True,
    password=os.getenv('REDIS_PASSWORD')
)

self.rate_limiter = RateLimiter(
    redis_client=redis_client,
    config={...}
)
```

### 2. Periodic Cleanup

Add to startup tasks:

```python
import asyncio

async def cleanup_rate_limits():
    """Periodic cleanup of expired rate limit windows"""
    while True:
        try:
            ailb_server.rate_limiter.cleanup_expired_windows(
                max_idle_seconds=3600
            )
            await asyncio.sleep(300)  # Every 5 minutes
        except Exception as e:
            logger.error("Rate limit cleanup failed", error=str(e))

# In startup()
asyncio.create_task(cleanup_rate_limits())
```

### 3. Dynamic Configuration

Load rate limit configs from database:

```python
# TODO: Load from manager database
async def load_rate_limit_configs():
    """Load rate limit configurations from manager database"""
    # Query manager API for API key configs
    # Set configs using limiter.set_config()
    pass
```

## Troubleshooting

### Rate Limits Not Working

1. Check that middleware is added AFTER CORS middleware
2. Verify API key is being extracted correctly
3. Check logs for rate limiting events
4. Verify `RATE_LIMITING_ENABLED=true` in environment

### Headers Not Appearing

1. Ensure middleware is properly configured
2. Check that path is not in exempt_paths
3. Verify API key is being sent in request

### Incorrect Token Counting

1. Update `_extract_token_usage()` in middleware.py
2. Set `X-Token-Count` header in chat completion handler
3. Integrate with TokenManager for accurate counting

## Next Steps

1. Implement Redis backend for distributed rate limiting
2. Add Prometheus metrics for monitoring
3. Integrate with RBAC for role-based limits
4. Add admin UI for managing rate limits
5. Implement burst allowances (if needed)
6. Add rate limit bypass for internal requests
7. Add rate limit warnings (e.g., at 80% capacity)
