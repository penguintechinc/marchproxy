# AILB Billing & Cost Tracking Module

Comprehensive cost tracking and budget management system for the AI Load Balancer (AILB).

## Overview

This module provides:
- **Cost calculation** for AI/LLM API requests across different providers
- **Usage tracking** with detailed records per virtual key
- **Budget management** with configurable limits and alerts
- **Spending analytics** with aggregated reports by model, provider, and key
- **Pre-request budget checks** to prevent overspending

## Architecture

### Components

1. **models.py** - Pydantic data models
   - `CostConfig` - Model pricing configuration (per 1K tokens)
   - `UsageRecord` - Individual request usage tracking
   - `BudgetConfig` - Budget limits and alert thresholds
   - `SpendSummary` - Aggregated spending reports
   - `BudgetStatusResponse` - Current budget status

2. **tracker.py** - Core tracking logic
   - `CostTracker` class - Main tracking engine
   - `MODEL_PRICING` dict - Default pricing for common models
   - In-memory storage (TODO: PostgreSQL migration)

3. **routes.py** - FastAPI REST endpoints
   - GET `/api/billing/spend` - Get spend summary
   - GET `/api/billing/spend/{key_id}` - Get key-specific spend
   - GET `/api/billing/budget/{key_id}` - Get budget status
   - PUT `/api/billing/budget/{key_id}` - Set/update budget
   - GET `/api/billing/pricing` - Get model pricing
   - PUT `/api/billing/pricing` - Update pricing (admin)
   - POST `/api/billing/budget/{key_id}/check` - Pre-check budget

## Usage Examples

### Basic Cost Calculation

```python
from app.billing.tracker import CostTracker

tracker = CostTracker()

# Calculate cost for a request
cost = tracker.calculate_cost(
    model="gpt-4",
    input_tokens=1000,
    output_tokens=500
)
# Returns: 0.06 (USD)
```

### Recording Usage

```python
# Record usage after serving a request
record = tracker.record_usage(
    key_id="key_abc123",
    model="gpt-4",
    provider="openai",
    input_tokens=1000,
    output_tokens=500,
    request_id="req_xyz789",
    user_id="user_demo"
)

print(f"Cost: ${record.cost:.4f}")
```

### Setting Budgets

```python
from app.billing.models import BudgetConfig, BudgetPeriod

# Set monthly budget
budget = BudgetConfig(
    key_id="key_abc123",
    max_budget=100.0,          # $100/month
    alert_threshold=0.8,        # Alert at 80%
    period=BudgetPeriod.MONTHLY,
    enabled=True,
    alert_email="admin@example.com"
)

tracker.set_budget("key_abc123", budget)
```

### Checking Budget Status

```python
# Get current budget status
status = tracker.get_budget_status("key_abc123")

print(f"Current Spend: ${status.current_spend:.2f}")
print(f"Remaining: ${status.budget_remaining:.2f}")
print(f"Used: {status.budget_used_percent:.1f}%")
print(f"Status: {status.status.value}")
print(f"Can make request: {status.can_make_request}")
```

### Pre-checking Budget

```python
# Check if budget allows a request before making it
estimated_cost = tracker.calculate_cost("gpt-4", 1000, 500)

if tracker.check_budget("key_abc123", estimated_cost):
    # Proceed with request
    response = await make_api_call(...)
    tracker.record_usage(...)
else:
    # Budget exceeded, reject request
    raise HTTPException(402, "Budget exceeded")
```

### Getting Spending Reports

```python
from app.billing.models import BudgetPeriod

# Get monthly spending for a key
summary = tracker.get_spend("key_abc123", BudgetPeriod.MONTHLY)

print(f"Total Cost: ${summary.total_cost:.2f}")
print(f"Total Requests: {summary.total_requests}")
print(f"Total Tokens: {summary.total_tokens:,}")

# Breakdown by model
for model, cost in summary.by_model.items():
    print(f"  {model}: ${cost:.2f}")

# Breakdown by provider
for provider, cost in summary.by_provider.items():
    print(f"  {provider}: ${cost:.2f}")
```

### Filtered Summaries

```python
# Get all OpenAI spending across all keys
openai_summary = tracker.get_summary(
    period=BudgetPeriod.MONTHLY,
    provider="openai"
)

# Get GPT-4 spending for a specific key
gpt4_summary = tracker.get_summary(
    period=BudgetPeriod.WEEKLY,
    key_id="key_abc123",
    model="gpt-4"
)
```

## API Endpoints

### Authentication

All endpoints require authentication via `X-User-ID` header (development).
In production, this will be replaced with proper JWT/OAuth authentication.

Admin endpoints additionally require `X-Admin: true` header.

### GET /api/billing/spend

Get aggregated spending summary with optional filters.

**Query Parameters:**
- `period` - Budget period (hourly, daily, weekly, monthly, total)
- `key_id` - Filter by virtual key
- `provider` - Filter by provider (openai, anthropic, etc.)
- `model` - Filter by model name
- `start_date` - Custom start date (ISO 8601)
- `end_date` - Custom end date (ISO 8601)

**Example:**
```bash
curl -H "X-User-ID: user123" \
  "http://localhost:8080/api/billing/spend?period=monthly&provider=openai"
```

### GET /api/billing/spend/{key_id}

Get spending for a specific virtual key.

**Example:**
```bash
curl -H "X-User-ID: user123" \
  "http://localhost:8080/api/billing/spend/key_abc123?period=weekly"
```

### GET /api/billing/budget/{key_id}

Get current budget status for a key.

**Example:**
```bash
curl -H "X-User-ID: user123" \
  "http://localhost:8080/api/billing/budget/key_abc123"
```

**Response:**
```json
{
  "key_id": "key_abc123",
  "current_spend": 75.50,
  "budget_remaining": 24.50,
  "budget_used_percent": 75.5,
  "status": "warning",
  "alert_triggered": false,
  "can_make_request": true,
  "period_start": "2025-12-01T00:00:00Z",
  "period_end": "2025-12-31T23:59:59Z",
  "estimated_requests_remaining": 245
}
```

### PUT /api/billing/budget/{key_id}

Set or update budget configuration.

**Example:**
```bash
curl -X PUT -H "X-User-ID: user123" \
  -H "Content-Type: application/json" \
  -d '{
    "key_id": "key_abc123",
    "max_budget": 100.0,
    "alert_threshold": 0.8,
    "period": "monthly",
    "enabled": true
  }' \
  "http://localhost:8080/api/billing/budget/key_abc123"
```

### GET /api/billing/pricing

Get current pricing for all models.

**Example:**
```bash
curl -H "X-User-ID: user123" \
  "http://localhost:8080/api/billing/pricing"
```

### PUT /api/billing/pricing (Admin Only)

Update model pricing.

**Example:**
```bash
curl -X PUT -H "X-User-ID: admin" -H "X-Admin: true" \
  "http://localhost:8080/api/billing/pricing?model_name=gpt-4&provider=openai&input_price_per_1k=0.03&output_price_per_1k=0.06"
```

### POST /api/billing/budget/{key_id}/check

Pre-check if budget allows a request.

**Example:**
```bash
curl -X POST -H "X-User-ID: user123" \
  "http://localhost:8080/api/billing/budget/key_abc123/check?estimated_tokens=1500&model=gpt-4"
```

## Supported Models & Pricing

### OpenAI Models

| Model | Input (per 1K) | Output (per 1K) |
|-------|----------------|-----------------|
| gpt-4 | $0.03 | $0.06 |
| gpt-4-turbo | $0.01 | $0.03 |
| gpt-3.5-turbo | $0.0005 | $0.0015 |
| gpt-3.5-turbo-16k | $0.003 | $0.004 |

### Anthropic Claude Models

| Model | Input (per 1K) | Output (per 1K) |
|-------|----------------|-----------------|
| claude-3-opus | $0.015 | $0.075 |
| claude-3-sonnet | $0.003 | $0.015 |
| claude-3-haiku | $0.00025 | $0.00125 |
| claude-2.1 | $0.008 | $0.024 |

### Ollama (Self-hosted)

All Ollama models are priced at $0.00 (free, self-hosted).

## Budget Periods

- **HOURLY** - Budget resets every hour
- **DAILY** - Budget resets every day at midnight UTC
- **WEEKLY** - Budget resets every Monday at midnight UTC
- **MONTHLY** - Budget resets on the 1st of each month
- **TOTAL** - Lifetime budget (never resets)

## Budget Status

- **OK** - Normal operation, under budget
- **WARNING** - Alert threshold reached (default 80%)
- **EXCEEDED** - Budget limit reached, requests blocked

## TODO: Database Migration

Currently using in-memory storage. Migration to PostgreSQL planned:

### Tables Required

```sql
-- Cost configurations
CREATE TABLE cost_configs (
    id SERIAL PRIMARY KEY,
    model_name VARCHAR(255) NOT NULL,
    provider VARCHAR(100) NOT NULL,
    input_price_per_1k DECIMAL(10, 6) NOT NULL,
    output_price_per_1k DECIMAL(10, 6) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    effective_date TIMESTAMP DEFAULT NOW(),
    notes TEXT,
    UNIQUE(provider, model_name)
);

-- Usage records
CREATE TABLE usage_records (
    id UUID PRIMARY KEY,
    key_id VARCHAR(255) NOT NULL,
    model VARCHAR(255) NOT NULL,
    provider VARCHAR(100) NOT NULL,
    input_tokens INTEGER NOT NULL,
    output_tokens INTEGER NOT NULL,
    total_tokens INTEGER NOT NULL,
    cost DECIMAL(10, 6) NOT NULL,
    timestamp TIMESTAMP DEFAULT NOW(),
    request_id VARCHAR(255),
    session_id VARCHAR(255),
    user_id VARCHAR(255),
    metadata JSONB,
    INDEX idx_key_timestamp (key_id, timestamp),
    INDEX idx_provider (provider),
    INDEX idx_model (model)
);

-- Budget configurations
CREATE TABLE budget_configs (
    key_id VARCHAR(255) PRIMARY KEY,
    max_budget DECIMAL(10, 2) NOT NULL,
    alert_threshold DECIMAL(3, 2) DEFAULT 0.8,
    period VARCHAR(20) NOT NULL,
    enabled BOOLEAN DEFAULT TRUE,
    alert_email VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP
);
```

## Integration with AILB

### Main Application Integration

Add to `/home/penguin/code/MarchProxy/proxy-ailb/main.py`:

```python
from app.billing.routes import router as billing_router

# Register billing routes
app.include_router(billing_router)
```

### Request Router Integration

Modify request router to record usage:

```python
from app.billing.tracker import get_cost_tracker

# After serving request
tracker = get_cost_tracker()
tracker.record_usage(
    key_id=api_key_id,
    model=model_used,
    provider=provider_name,
    input_tokens=usage_info['input_tokens'],
    output_tokens=usage_info['output_tokens'],
    request_id=request_id
)
```

### Pre-request Budget Check

Add to request validation:

```python
# Before routing request
tracker = get_cost_tracker()
estimated_cost = tracker.calculate_cost(
    model=requested_model,
    input_tokens=estimate_input_tokens(messages),
    output_tokens=500  # Conservative estimate
)

if not tracker.check_budget(key_id, estimated_cost):
    raise HTTPException(402, "Budget exceeded")
```

## Testing

Run basic tests:

```bash
cd /home/penguin/code/MarchProxy/proxy-ailb
python3 -m app.billing.test_basic
```

Run example:

```bash
cd /home/penguin/code/MarchProxy/proxy-ailb
python3 -m app.billing.example
```

## File Structure

```
app/billing/
├── __init__.py          # Module exports
├── models.py            # Pydantic models
├── tracker.py           # Core tracking logic
├── routes.py            # FastAPI endpoints
├── example.py           # Usage examples
├── test_basic.py        # Basic tests
└── README.md            # This file
```

## License

This module is part of MarchProxy and follows the project's licensing terms.
