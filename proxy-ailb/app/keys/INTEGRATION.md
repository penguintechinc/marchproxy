# Virtual Key Management Integration Guide

This guide shows how to integrate the virtual key management system with the AILB main application.

## Quick Start

### 1. Add Routes to Main Application

Edit `/home/penguin/code/MarchProxy/proxy-ailb/main.py`:

```python
from app.keys.routes import router as keys_router

# Add after creating the FastAPI app
app.include_router(keys_router)
```

### 2. Initialize KeyManager at Startup

Add to the `AILBServer` class in `main.py`:

```python
from app.keys.manager import KeyManager

class AILBServer:
    def __init__(self):
        # ... existing code ...
        self.key_manager = None

    async def startup(self):
        # ... existing initialization ...

        # Initialize key manager
        self.key_manager = KeyManager(
            redis_client=None,  # TODO: Add Redis client if available
            config={}
        )
        logger.info("Key manager initialized")
```

### 3. Add Key Validation Middleware

Add middleware to validate API keys on incoming requests:

```python
from fastapi import Request, HTTPException
from app.keys.models import KeyValidationResult

@app.middleware("http")
async def validate_api_key_middleware(request: Request, call_next):
    """Validate API key for protected endpoints"""

    # Skip validation for health checks and key management endpoints
    if request.url.path in ["/healthz", "/metrics", "/api/keys/validate"]:
        return await call_next(request)

    # Get authorization header
    auth_header = request.headers.get("Authorization", "")

    if auth_header.startswith("Bearer "):
        api_key = auth_header[7:]

        # Validate key
        result: KeyValidationResult = ailb_server.key_manager.validate_key(api_key)

        if not result.valid:
            return JSONResponse(
                status_code=401,
                content={
                    "error": "Invalid API key",
                    "detail": result.error
                }
            )

        # Attach key info to request state
        request.state.key_id = result.key_id
        request.state.key_data = result.key_data
        request.state.rate_limit_info = result.rate_limit_info

    response = await call_next(request)
    return response
```

### 4. Track Usage in Chat Completions Endpoint

Update the `/v1/chat/completions` endpoint to record usage:

```python
@app.post("/v1/chat/completions")
async def chat_completions(
    request: Request,
    authorization: Optional[str] = Header(None),
    x_preferred_model: Optional[str] = Header(None, alias="X-Preferred-Model")
):
    # ... existing request handling ...

    # Get response from LLM
    response_text, usage_info = await ailb_server.request_router.route_request(
        model=model,
        messages=messages,
        **{k: v for k, v in body.items() if k not in ['messages', 'model']}
    )

    # Record usage if key is present
    if hasattr(request.state, "key_id"):
        # Calculate cost (example - adjust based on your pricing)
        input_tokens = usage_info.get('input_tokens', 0)
        output_tokens = usage_info.get('output_tokens', 0)
        cost = (input_tokens * 0.00001) + (output_tokens * 0.00003)  # Example rates

        # Record usage
        ailb_server.key_manager.record_usage(
            key_id=request.state.key_id,
            tokens=usage_info.get('total_tokens', 0),
            cost=cost,
            model=model,
            provider=usage_info.get('provider', 'unknown'),
            request_id=f"req_{int(time.time())}"
        )

    # ... return response ...
```

### 5. Add Rate Limit Headers to Response

Add rate limit information to response headers:

```python
@app.middleware("http")
async def add_rate_limit_headers(request: Request, call_next):
    response = await call_next(request)

    # Add rate limit headers if key was validated
    if hasattr(request.state, "rate_limit_info"):
        rate_info = request.state.rate_limit_info

        if rate_info:
            tpm = rate_info.get('tpm', {})
            rpm = rate_info.get('rpm', {})

            response.headers["X-RateLimit-Limit-TPM"] = str(tpm.get('limit', 'unlimited'))
            response.headers["X-RateLimit-Remaining-TPM"] = str(
                tpm.get('limit', 0) - tpm.get('current', 0)
            )
            response.headers["X-RateLimit-Limit-RPM"] = str(rpm.get('limit', 'unlimited'))
            response.headers["X-RateLimit-Remaining-RPM"] = str(
                rpm.get('limit', 0) - rpm.get('current', 0)
            )

    return response
```

## API Usage Examples

### Creating a Virtual Key

```bash
curl -X POST "http://localhost:8080/api/keys" \
  -H "Content-Type: application/json" \
  -H "X-User-ID: user_123" \
  -d '{
    "name": "My Production Key",
    "user_id": "user_123",
    "expires_days": 365,
    "allowed_models": ["gpt-4", "claude-3-opus"],
    "max_budget": 100.0,
    "tpm_limit": 10000,
    "rpm_limit": 60
  }'
```

Response:
```json
{
  "key": "sk-mp-a1b2c3d4e5f67890-xYz123AbC456DeF789GhI012JkL345MnO678PqR",
  "key_data": {
    "id": "a1b2c3d4e5f67890",
    "name": "My Production Key",
    "status": "active",
    "max_budget": 100.0,
    "spent": 0.0,
    "budget_remaining": 100.0,
    ...
  }
}
```

### Using the Virtual Key

```bash
curl -X POST "http://localhost:8080/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-mp-a1b2c3d4e5f67890-xYz123..." \
  -d '{
    "model": "gpt-4",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ]
  }'
```

### Checking Key Usage

```bash
curl -X GET "http://localhost:8080/api/keys/a1b2c3d4e5f67890/usage?days=30" \
  -H "X-User-ID: user_123"
```

## Environment Variables

Add to your `.env` or Docker configuration:

```bash
# Virtual Key Settings
VIRTUAL_KEY_PREFIX=sk-mp
VIRTUAL_KEY_DEFAULT_BUDGET=100.0
VIRTUAL_KEY_DEFAULT_TPM=10000
VIRTUAL_KEY_DEFAULT_RPM=60

# Redis for persistence (optional)
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_DB=0
```

## Database Migration (TODO)

When ready to migrate from in-memory to PostgreSQL:

1. Create database schema:

```sql
CREATE TABLE virtual_keys (
    id VARCHAR(32) PRIMARY KEY,
    key_hash VARCHAR(64) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    team_id VARCHAR(64),
    created_at TIMESTAMP NOT NULL,
    expires_at TIMESTAMP,
    is_active BOOLEAN DEFAULT true,
    allowed_models JSONB DEFAULT '["*"]',
    max_budget DECIMAL(10,2),
    spent DECIMAL(10,2) DEFAULT 0,
    tpm_limit INTEGER,
    rpm_limit INTEGER,
    metadata JSONB DEFAULT '{}',
    last_used TIMESTAMP,
    total_requests INTEGER DEFAULT 0
);

CREATE TABLE key_usage (
    id SERIAL PRIMARY KEY,
    key_id VARCHAR(32) REFERENCES virtual_keys(id),
    timestamp TIMESTAMP NOT NULL,
    tokens INTEGER NOT NULL,
    cost DECIMAL(10,4) NOT NULL,
    model VARCHAR(100) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    request_id VARCHAR(100)
);

CREATE INDEX idx_keys_user ON virtual_keys(user_id);
CREATE INDEX idx_keys_team ON virtual_keys(team_id);
CREATE INDEX idx_keys_status ON virtual_keys(is_active, expires_at);
CREATE INDEX idx_usage_key ON key_usage(key_id, timestamp);
CREATE INDEX idx_usage_time ON key_usage(timestamp);
```

2. Update `KeyManager` to use database queries instead of in-memory dict
3. Add connection pooling (asyncpg or psycopg3)
4. Update persistence methods

## Security Considerations

1. **Never log full API keys** - Only log key IDs
2. **Use HTTPS in production** - API keys should never be transmitted over HTTP
3. **Implement key rotation** - Encourage users to rotate keys regularly
4. **Set expiration dates** - Default to 1 year maximum
5. **Monitor for abuse** - Track usage patterns and alert on anomalies
6. **Rate limiting** - Always set TPM/RPM limits for production keys

## Testing

Run the basic test:

```bash
cd /home/penguin/code/MarchProxy/proxy-ailb
pip install -r requirements.txt
python3 app/keys/test_basic.py
```

Run the full example:

```bash
python3 app/keys/example.py
```

## Monitoring

Add Prometheus metrics for key usage:

```python
from prometheus_client import Counter, Histogram, Gauge

key_requests_total = Counter(
    'ailb_key_requests_total',
    'Total requests per virtual key',
    ['key_id', 'model']
)

key_tokens_total = Counter(
    'ailb_key_tokens_total',
    'Total tokens consumed per virtual key',
    ['key_id', 'model']
)

key_cost_total = Counter(
    'ailb_key_cost_total',
    'Total cost per virtual key in USD',
    ['key_id']
)

key_validation_failures = Counter(
    'ailb_key_validation_failures_total',
    'Total key validation failures',
    ['reason']
)
```

## Troubleshooting

### "Key not found" error
- Check that the key was created successfully
- Verify the key hasn't been deleted/deactivated
- Ensure the key is being passed correctly in the Authorization header

### Rate limit exceeded
- Check the key's TPM/RPM limits
- Wait for the next minute to reset counters
- Increase limits if legitimate traffic

### Budget exceeded
- Check total spend against max_budget
- Increase budget or create new key
- Review usage patterns for unexpected costs

## Support

For issues or questions:
- Check the README.md in this directory
- Review the example.py for usage patterns
- Examine the test_basic.py for integration examples
