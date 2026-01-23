# AILB Usage Guide

## Quick Start

### Prerequisites
- Running AILB instance (localhost:8080)
- Python 3.7+ or any HTTP client

### Basic Chat Request

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {"role": "user", "content": "What is machine learning?"}
    ]
  }'
```

### Python Client

```python
import requests

response = requests.post(
    "http://localhost:8080/v1/chat/completions",
    json={
        "model": "gpt-3.5-turbo",
        "messages": [
            {"role": "user", "content": "Hello!"}
        ]
    }
)

print(response.json())
```

## Using API Keys

Create a virtual API key for usage tracking and budgeting:

```bash
# Create key
curl -X POST http://localhost:8080/api/keys \
  -H "Content-Type: application/json" \
  -H "X-User-ID: user-123" \
  -d '{
    "name": "My Production Key",
    "monthly_quota_dollars": 100.0,
    "rate_limit_rpm": 60
  }'

# Response includes the API key
# Store it securely - you cannot retrieve it again!
```

Use the API key in requests:

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-ailb_abcdef1234567890" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

## Conversation Memory

Enable stateful conversations with session IDs:

```bash
# First message - establish session
response1 = requests.post(
    "http://localhost:8080/v1/chat/completions",
    headers={"Authorization": "Bearer <api-key>"},
    json={
        "model": "gpt-3.5-turbo",
        "messages": [
            {"role": "user", "content": "My name is Alice"}
        ],
        "session_id": "user-alice-001"
    }
)

# Second message - context is preserved
response2 = requests.post(
    "http://localhost:8080/v1/chat/completions",
    headers={"Authorization": "Bearer <api-key>"},
    json={
        "model": "gpt-3.5-turbo",
        "messages": [
            {"role": "user", "content": "What is my name?"}
        ],
        "session_id": "user-alice-001"
    }
)

# Response will mention "Alice" from context
```

## RAG (Retrieval-Augmented Generation)

Use knowledge base collections for context-aware responses:

```python
# Query with RAG context
response = requests.post(
    "http://localhost:8080/v1/chat/completions",
    headers={"Authorization": "Bearer <api-key>"},
    json={
        "model": "gpt-4",
        "messages": [
            {"role": "user", "content": "Tell me about our company policies"}
        ],
        "rag_collection": "company_docs",
        "rag_top_k": 5
    }
)

# LLM response includes relevant context from knowledge base
```

## Provider Selection

Request specific providers or models:

```python
# Use specific model
response = requests.post(
    "http://localhost:8080/v1/chat/completions",
    json={
        "model": "claude-3-opus-20240229",
        "messages": [{"role": "user", "content": "..."}]
    }
)

# Alternative: Use X-Preferred-Model header
response = requests.post(
    "http://localhost:8080/v1/chat/completions",
    headers={"X-Preferred-Model": "gpt-4"},
    json={
        "messages": [{"role": "user", "content": "..."}]
    }
)
```

**Available Models:**
- OpenAI: gpt-4, gpt-4-turbo, gpt-3.5-turbo
- Anthropic: claude-3-opus, claude-3-sonnet, claude-3-haiku
- Ollama: mistral, neural-chat, dolphin-mixtral (configured locally)

## Managing API Keys

### List Keys
```bash
curl -X GET http://localhost:8080/api/keys \
  -H "X-User-ID: user-123"
```

### Get Key Details
```bash
curl -X GET http://localhost:8080/api/keys/key-123 \
  -H "X-User-ID: user-123"
```

### Update Key
```bash
curl -X PATCH http://localhost:8080/api/keys/key-123 \
  -H "X-User-ID: user-123" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated Name",
    "rate_limit_rpm": 120
  }'
```

### Revoke Key
```bash
curl -X DELETE http://localhost:8080/api/keys/key-123 \
  -H "X-User-ID: user-123"
```

## Billing & Cost Management

### Check Spending

```bash
# Monthly summary
curl -X GET "http://localhost:8080/api/billing/spend?period=monthly" \
  -H "X-User-ID: user-123"

# By provider
curl -X GET "http://localhost:8080/api/billing/spend?provider=openai" \
  -H "X-User-ID: user-123"

# Custom date range
curl -X GET "http://localhost:8080/api/billing/spend?start_date=2024-01-01&end_date=2024-01-31" \
  -H "X-User-ID: user-123"
```

### Set Budget Limits

```bash
curl -X POST http://localhost:8080/api/billing/budget \
  -H "X-User-ID: user-123" \
  -H "Content-Type: application/json" \
  -d '{
    "key_id": "key-123",
    "monthly_limit_dollars": 500.0,
    "monthly_limit_tokens": 10000000
  }'
```

### Check Budget Status

```bash
curl -X GET http://localhost:8080/api/billing/budget/key-123 \
  -H "X-User-ID: user-123"
```

## Advanced Features

### Routing Statistics

```bash
curl http://localhost:8080/api/routing/stats

# Response shows:
# - Total requests and success rate
# - Per-provider metrics (latency, success rate)
# - Last error for each provider
```

### Health Checks

```bash
curl http://localhost:8080/healthz

# Returns: {"status": "healthy"}
```

### Metrics (Prometheus)

```bash
curl http://localhost:8080/metrics

# Includes:
# - ailb_requests_total
# - ailb_request_latency_ms
# - ailb_tokens_processed
# - ailb_cost_total_usd
```

## Integration Patterns

### Multi-Provider Fallback

AILB automatically routes to available providers. Configure failover strategy:

```bash
export ROUTING_STRATEGY=failover
```

If primary provider fails, requests automatically route to next available provider.

### Cost-Optimized Routing

Route to cheapest provider that supports the model:

```bash
export ROUTING_STRATEGY=cost_optimized
```

Useful for cost-sensitive applications. Trades latency for reduced costs.

### Latency-Optimized Routing

Route to fastest provider based on historical latency:

```bash
export ROUTING_STRATEGY=latency_optimized
```

Useful for real-time applications.

### Load-Balanced Routing (Default)

Distribute requests evenly across providers:

```bash
export ROUTING_STRATEGY=load_balanced
```

Best for production when all providers have similar cost/latency.

## Error Handling

### Common Error Scenarios

**Invalid API Key:**
```
Status: 401
Response: {"detail": "Invalid API key: ..."}
```

**Budget Exceeded:**
```
Status: 402
Response: {"detail": "Budget exceeded. Request would exceed spending limit."}
```

**Rate Limit Hit:**
```
Status: 429
Response: {"detail": "Rate limit exceeded"}
```

**Provider Down:**
```
Status: 503
Response: {"detail": "All providers unavailable"}
```

## Python SDK Example

```python
import requests
import json

class AILBClient:
    def __init__(self, api_key: str, base_url: str = "http://localhost:8080"):
        self.api_key = api_key
        self.base_url = base_url
        self.session = requests.Session()
        self.session.headers.update({
            "Authorization": f"Bearer {api_key}"
        })

    def chat(self, model: str, messages: list, **kwargs):
        """Send chat completion request"""
        response = self.session.post(
            f"{self.base_url}/v1/chat/completions",
            json={
                "model": model,
                "messages": messages,
                **kwargs
            }
        )
        response.raise_for_status()
        return response.json()

    def get_spend_summary(self):
        """Get billing summary"""
        response = self.session.get(
            f"{self.base_url}/api/billing/spend"
        )
        response.raise_for_status()
        return response.json()

# Usage
client = AILBClient("sk-ailb_...")
response = client.chat(
    model="gpt-4",
    messages=[{"role": "user", "content": "Hello!"}],
    session_id="user-123"
)
print(response)
```

## Performance Tips

1. **Use session IDs for conversations:** Enables memory context reuse
2. **Set appropriate rate limits:** Prevents runaway costs
3. **Monitor metrics:** Track provider performance
4. **Implement retry logic:** Handle transient failures
5. **Cache results:** Avoid redundant requests
6. **Use appropriate routing strategy:** Match your requirements

## Troubleshooting

### Request Hangs
- Check provider API status
- Verify network connectivity
- Check AILB logs for errors

### High Latency
- Check provider response times (`/api/routing/stats`)
- Switch to latency-optimized routing
- Consider switching providers

### Budget Issues
- Review spending summary (`/api/billing/spend`)
- Adjust key quotas
- Monitor token usage per request

---

**Last Updated:** 2025-12-16
