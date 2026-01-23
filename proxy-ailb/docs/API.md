# AILB API Reference

## Overview

AILB provides two API interfaces:
- **HTTP API** (Port 8080): OpenAI-compatible REST endpoints for LLM requests
- **gRPC API** (Port 50051): ModuleService for MarchProxy integration

## HTTP API Endpoints

### Chat Completions

**Endpoint:** `POST /v1/chat/completions`

OpenAI-compatible chat completion endpoint with intelligent provider routing, conversation memory, and RAG support.

**Request:**
```json
{
  "model": "gpt-4",
  "messages": [
    {"role": "user", "content": "Hello!"}
  ],
  "session_id": "optional-session-id",
  "rag_collection": "optional-collection-name",
  "rag_top_k": 3,
  "temperature": 0.7,
  "max_tokens": 1000
}
```

**Response:**
```json
{
  "id": "chatcmpl-1234567890",
  "object": "chat.completion",
  "created": 1234567890,
  "model": "gpt-4",
  "choices": [{
    "index": 0,
    "message": {
      "role": "assistant",
      "content": "Response text here"
    },
    "finish_reason": "stop"
  }],
  "usage": {
    "prompt_tokens": 15,
    "completion_tokens": 25,
    "total_tokens": 40
  }
}
```

**Headers:**
- `Authorization: Bearer <api_key>` - Optional API key for usage tracking
- `X-Session-ID: <session-id>` - Optional session ID for memory
- `X-Preferred-Model: <model>` - Alternative to `model` in body

**Query Parameters:**
- `model` - LLM model to use (required)
- `messages` - Array of message objects (required)
- `session_id` - Conversation session identifier (optional)
- `rag_collection` - RAG knowledge base collection (optional)
- `rag_top_k` - Number of RAG results to retrieve (optional, default: 3)

**Status Codes:**
- `200` - Success
- `401` - Invalid API key
- `402` - Budget exceeded
- `400` - Invalid request
- `500` - Server error

---

### List Models

**Endpoint:** `GET /v1/models`

List all available models from configured providers.

**Response:**
```json
{
  "object": "list",
  "data": [
    {"id": "gpt-4", "object": "model", "provider": "openai"},
    {"id": "claude-3-opus", "object": "model", "provider": "anthropic"},
    {"id": "mistral-7b", "object": "model", "provider": "ollama"}
  ]
}
```

---

### Routing Statistics

**Endpoint:** `GET /api/routing/stats`

Get detailed routing statistics and provider metrics.

**Response:**
```json
{
  "total_requests": 1500,
  "successful_requests": 1485,
  "failed_requests": 15,
  "providers": {
    "openai": {
      "requests": 750,
      "success_rate": 0.99,
      "avg_latency_ms": 250,
      "last_error": null
    },
    "anthropic": {
      "requests": 600,
      "success_rate": 0.98,
      "avg_latency_ms": 300,
      "last_error": null
    },
    "ollama": {
      "requests": 150,
      "success_rate": 0.95,
      "avg_latency_ms": 150,
      "last_error": "Connection timeout"
    }
  }
}
```

---

## Virtual Keys API

Base path: `/api/keys`

### Create API Key

**Endpoint:** `POST /api/keys`

Generate a new virtual API key with optional quotas.

**Headers:**
- `X-User-ID: <user-id>` - User identifier (required)

**Request:**
```json
{
  "name": "My API Key",
  "user_id": "user-123",
  "monthly_quota_dollars": 100.0,
  "monthly_quota_tokens": 1000000,
  "rate_limit_rpm": 60,
  "rate_limit_tpm": 90000
}
```

**Response:**
```json
{
  "key": "sk-ailb_abcdef1234567890",
  "key_data": {
    "id": "key-123",
    "user_id": "user-123",
    "name": "My API Key",
    "created_at": "2024-01-15T10:30:00Z",
    "monthly_quota_dollars": 100.0,
    "monthly_quota_tokens": 1000000,
    "rate_limit_rpm": 60,
    "rate_limit_tpm": 90000,
    "last_used": null
  }
}
```

---

### List Keys

**Endpoint:** `GET /api/keys`

List all API keys for the authenticated user.

**Headers:**
- `X-User-ID: <user-id>` - User identifier (required)

**Query Parameters:**
- `limit` - Max keys to return (default: 100)
- `offset` - Pagination offset (default: 0)
- `status` - Filter by status (active|revoked|expired)

**Response:**
```json
{
  "keys": [
    {
      "id": "key-123",
      "name": "Production Key",
      "created_at": "2024-01-15T10:30:00Z",
      "last_used": "2024-01-20T14:22:30Z",
      "status": "active"
    }
  ],
  "total": 1,
  "limit": 100,
  "offset": 0
}
```

---

### Get Key Details

**Endpoint:** `GET /api/keys/{key_id}`

Get detailed information for a specific API key.

**Headers:**
- `X-User-ID: <user-id>` - User identifier (required)

**Response:**
```json
{
  "id": "key-123",
  "user_id": "user-123",
  "name": "Production Key",
  "created_at": "2024-01-15T10:30:00Z",
  "last_used": "2024-01-20T14:22:30Z",
  "status": "active",
  "monthly_quota_dollars": 100.0,
  "monthly_quota_tokens": 1000000,
  "rate_limit_rpm": 60,
  "rate_limit_tpm": 90000
}
```

---

### Update Key

**Endpoint:** `PATCH /api/keys/{key_id}`

Update API key settings.

**Headers:**
- `X-User-ID: <user-id>` - User identifier (required)

**Request:**
```json
{
  "name": "Updated Name",
  "monthly_quota_dollars": 200.0,
  "monthly_quota_tokens": 2000000,
  "rate_limit_rpm": 120,
  "rate_limit_tpm": 180000
}
```

---

### Revoke Key

**Endpoint:** `DELETE /api/keys/{key_id}`

Revoke an API key (irreversible).

**Headers:**
- `X-User-ID: <user-id>` - User identifier (required)

**Status Codes:**
- `204` - Successfully revoked
- `404` - Key not found
- `409` - Key already revoked

---

## Billing API

Base path: `/api/billing`

### Get Spend Summary

**Endpoint:** `GET /api/billing/spend`

Get spending summary for a user or organization.

**Headers:**
- `X-User-ID: <user-id>` - User identifier (required)

**Query Parameters:**
- `period` - Budget period (monthly|daily|custom) - default: monthly
- `key_id` - Filter by specific API key (optional)
- `provider` - Filter by provider (openai|anthropic|ollama) (optional)
- `model` - Filter by specific model (optional)
- `start_date` - Custom period start (ISO 8601) (optional)
- `end_date` - Custom period end (ISO 8601) (optional)

**Response:**
```json
{
  "period": "monthly",
  "start_date": "2024-01-01T00:00:00Z",
  "end_date": "2024-01-31T23:59:59Z",
  "total_spend": 45.67,
  "total_tokens": 250000,
  "providers": {
    "openai": {
      "spend": 30.50,
      "tokens": 150000,
      "requests": 300
    },
    "anthropic": {
      "spend": 15.17,
      "tokens": 100000,
      "requests": 200
    }
  },
  "by_model": {
    "gpt-4": {
      "spend": 25.00,
      "tokens": 100000
    },
    "claude-3-opus": {
      "spend": 15.17,
      "tokens": 100000
    }
  }
}
```

---

### Set Budget

**Endpoint:** `POST /api/billing/budget`

Set monthly budget limit for API key.

**Headers:**
- `X-User-ID: <user-id>` - User identifier (required)

**Request:**
```json
{
  "key_id": "key-123",
  "monthly_limit_dollars": 500.0,
  "monthly_limit_tokens": 10000000
}
```

**Response:**
```json
{
  "key_id": "key-123",
  "monthly_limit_dollars": 500.0,
  "monthly_limit_tokens": 10000000,
  "current_spend": 45.67,
  "current_tokens": 250000,
  "remaining_budget": 454.33
}
```

---

### Get Budget Status

**Endpoint:** `GET /api/billing/budget/{key_id}`

Check current budget status for an API key.

**Headers:**
- `X-User-ID: <user-id>` - User identifier (required)

**Response:**
```json
{
  "key_id": "key-123",
  "monthly_limit_dollars": 500.0,
  "monthly_limit_tokens": 10000000,
  "current_spend": 45.67,
  "current_tokens": 250000,
  "remaining_budget": 454.33,
  "status": "healthy"
}
```

---

## Health & Metrics Endpoints

### Health Check

**Endpoint:** `GET /healthz`

Kubernetes-style health check.

**Response:**
```json
{
  "status": "healthy"
}
```

**Status Codes:**
- `200` - Healthy
- `503` - Unhealthy

---

### Prometheus Metrics

**Endpoint:** `GET /metrics`

Prometheus-compatible metrics endpoint.

**Metrics Exposed:**
- `ailb_requests_total` - Total requests by provider
- `ailb_request_success_rate` - Success rate by provider
- `ailb_request_latency_ms` - Latency percentiles (p50, p95, p99)
- `ailb_active_sessions` - Active conversation sessions
- `ailb_tokens_processed` - Tokens processed by provider
- `ailb_cost_total_usd` - Total cost by provider
- `ailb_memory_usage_bytes` - Memory system status
- `ailb_rag_queries_total` - RAG queries processed

---

## Error Responses

All error responses follow a standard format:

```json
{
  "detail": "Error description",
  "type": "error_type",
  "status_code": 400
}
```

**Common Error Codes:**
- `400` - Bad Request
- `401` - Unauthorized (invalid API key)
- `402` - Payment Required (budget exceeded)
- `403` - Forbidden
- `404` - Not Found
- `429` - Too Many Requests (rate limited)
- `500` - Internal Server Error
- `503` - Service Unavailable (provider down)

---

## gRPC ModuleService API

Implements MarchProxy ModuleService for orchestration integration.

**Service Methods:**

1. **GetStatus()** - Module health and operational status
2. **GetMetrics()** - Performance and routing metrics
3. **SetTrafficWeight()** - Blue/green deployment control
4. **ApplyRateLimit()** - Rate limiting configuration
5. **GetRoutes()** - Active route information
6. **Reload()** - Configuration reloading

See `/home/penguin/code/MarchProxy/proto` for protocol buffer definitions.
