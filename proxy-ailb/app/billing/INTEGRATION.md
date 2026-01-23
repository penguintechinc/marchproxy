# Billing Module Integration Guide

Step-by-step guide to integrate billing and cost tracking into the AILB application.

## 1. Register Billing Routes

Edit `/home/penguin/code/MarchProxy/proxy-ailb/main.py`:

```python
# Add import at top
from app.billing.routes import router as billing_router

# After creating FastAPI app, add router
app = FastAPI(
    title="AILB - AI Load Balancer",
    description="Intelligent AI/LLM proxy with routing, memory, and RAG support",
    version="1.0.0",
    lifespan=lifespan
)

# Register billing routes
app.include_router(billing_router)
```

## 2. Initialize Cost Tracker in Server

Edit `AILBServer.__init__()` in `main.py`:

```python
from app.billing.tracker import CostTracker

class AILBServer:
    def __init__(self):
        self.connectors: Dict[str, Any] = {}
        self.request_router = None
        self.memory_manager = None
        self.rag_manager = None
        self.grpc_server_task = None
        self.cost_tracker = CostTracker()  # Add this line

        # ... rest of init
```

## 3. Record Usage in Chat Completions

Edit the `/v1/chat/completions` endpoint in `main.py`:

```python
@app.post("/v1/chat/completions")
async def chat_completions(
    request: Request,
    authorization: Optional[str] = Header(None),
    x_preferred_model: Optional[str] = Header(None, alias="X-Preferred-Model")
):
    start_time = time.time()

    try:
        # Parse request
        body = await request.json()
        messages = body.get("messages", [])
        model = body.get("model") or x_preferred_model or "gpt-3.5-turbo"

        # Get API key from authorization header
        api_key = None
        if authorization:
            api_key = authorization.replace("Bearer ", "")

        # TODO: Validate API key and get key_id
        # For now, use a placeholder
        key_id = "key_from_api_key_validation"

        # Pre-check budget if we have a key_id
        if key_id:
            # Estimate token count (rough estimate)
            estimated_input_tokens = sum(len(m.get('content', '').split()) * 1.3
                                        for m in messages)
            estimated_output_tokens = 500  # Conservative estimate
            estimated_cost = ailb_server.cost_tracker.calculate_cost(
                model=model,
                input_tokens=int(estimated_input_tokens),
                output_tokens=estimated_output_tokens
            )

            # Check budget
            if not ailb_server.cost_tracker.check_budget(key_id, estimated_cost):
                raise HTTPException(
                    status_code=402,
                    detail="Budget exceeded for this API key"
                )

        # ... existing memory and RAG enhancement code ...

        # Route request to appropriate LLM provider
        response_text, usage_info = await ailb_server.request_router.route_request(
            model=model,
            messages=messages,
            **{k: v for k, v in body.items() if k not in ['messages', 'model', 'session_id']}
        )

        # Record usage if we have a key_id
        if key_id and usage_info:
            try:
                # Determine provider from model or router
                provider = "openai"  # TODO: Get from router
                if "claude" in model.lower():
                    provider = "anthropic"
                elif "ollama" in model.lower():
                    provider = "ollama"

                ailb_server.cost_tracker.record_usage(
                    key_id=key_id,
                    model=model,
                    provider=provider,
                    input_tokens=usage_info.get('input_tokens', 0),
                    output_tokens=usage_info.get('output_tokens', 0),
                    request_id=f"chatcmpl-{int(time.time())}",
                    session_id=body.get('session_id'),
                    metadata={
                        'route': '/v1/chat/completions',
                        'duration_ms': (time.time() - start_time) * 1000
                    }
                )
            except Exception as e:
                logger.error(f"Failed to record usage: {e}")
                # Don't fail the request if usage recording fails

        # Return OpenAI-compatible response
        return {
            "id": f"chatcmpl-{int(time.time())}",
            "object": "chat.completion",
            "created": int(time.time()),
            "model": model,
            "choices": [{
                "index": 0,
                "message": {
                    "role": "assistant",
                    "content": response_text
                },
                "finish_reason": "stop"
            }],
            "usage": {
                "prompt_tokens": usage_info.get('input_tokens', 0),
                "completion_tokens": usage_info.get('output_tokens', 0),
                "total_tokens": usage_info.get('total_tokens', 0)
            }
        }

    except HTTPException:
        raise
    except Exception as e:
        logger.error("Chat completion failed", error=str(e))
        raise HTTPException(status_code=500, detail=str(e))
```

## 4. Integration with Virtual Keys Module

Edit `/home/penguin/code/MarchProxy/proxy-ailb/app/keys/manager.py`:

```python
# Add import
from app.billing.tracker import get_cost_tracker

class KeyManager:
    def __init__(self):
        # ... existing code ...
        self.cost_tracker = get_cost_tracker()

    def get_key_with_budget_check(self, key_id: str) -> Tuple[VirtualKey, bool]:
        """
        Get key and check if budget allows requests

        Returns:
            Tuple of (VirtualKey, can_make_request)
        """
        key = self.get_key(key_id)
        if not key:
            return None, False

        # Check budget status
        budget_status = self.cost_tracker.get_budget_status(key_id)

        return key, budget_status.can_make_request
```

## 5. Add Budget Info to Key Response

Edit `/home/penguin/code/MarchProxy/proxy-ailb/app/keys/routes.py`:

```python
from app.billing.tracker import get_cost_tracker

@router.get("/{key_id}", response_model=KeyResponse)
async def get_key(
    key_id: str,
    user_id: str = Depends(get_current_user),
    key_manager: KeyManager = Depends(get_key_manager)
):
    """Get details for a specific virtual key"""
    try:
        virtual_key = key_manager.get_key(key_id)

        if not virtual_key:
            raise HTTPException(status_code=404, detail="Key not found")

        if virtual_key.user_id != user_id:
            raise HTTPException(status_code=403, detail="Access denied")

        # Get current spending
        cost_tracker = get_cost_tracker()
        budget_status = cost_tracker.get_budget_status(key_id)

        # Update spent amount in response
        virtual_key.spent = budget_status.current_spend

        return KeyResponse.from_virtual_key(virtual_key)

    except HTTPException:
        raise
    except Exception as e:
        logger.error("Failed to get key: %s", str(e))
        raise HTTPException(status_code=500, detail="Failed to get key")
```

## 6. Update Key Usage Endpoint

Edit the key usage endpoint to use billing data:

```python
@router.get("/{key_id}/usage")
async def get_key_usage(
    key_id: str,
    days: int = Query(30, ge=1, le=365),
    user_id: str = Depends(get_current_user),
    key_manager: KeyManager = Depends(get_key_manager)
):
    """Get usage statistics for a virtual key"""
    try:
        virtual_key = key_manager.get_key(key_id)

        if not virtual_key:
            raise HTTPException(status_code=404, detail="Key not found")

        if virtual_key.user_id != user_id:
            raise HTTPException(status_code=403, detail="Access denied")

        # Get spending summary from billing module
        from app.billing.tracker import get_cost_tracker
        from app.billing.models import BudgetPeriod
        from datetime import datetime, timedelta

        cost_tracker = get_cost_tracker()

        # Determine period based on days
        if days <= 1:
            period = BudgetPeriod.DAILY
        elif days <= 7:
            period = BudgetPeriod.WEEKLY
        else:
            period = BudgetPeriod.MONTHLY

        # Get custom date range
        end_date = datetime.utcnow()
        start_date = end_date - timedelta(days=days)

        summary = cost_tracker.get_spend(
            key_id=key_id,
            period=period,
            start_date=start_date,
            end_date=end_date
        )

        return {
            "key_id": key_id,
            "period_days": days,
            "total_cost": summary.total_cost,
            "total_requests": summary.total_requests,
            "total_tokens": summary.total_tokens,
            "by_model": summary.by_model,
            "by_provider": summary.by_provider,
            "period_start": summary.period_start.isoformat(),
            "period_end": summary.period_end.isoformat()
        }

    except HTTPException:
        raise
    except Exception as e:
        logger.error("Failed to get usage stats: %s", str(e))
        raise HTTPException(status_code=500, detail="Failed to get usage statistics")
```

## 7. Add Budget Alerts (Optional)

Create an alert checking task in `main.py`:

```python
import asyncio

async def check_budget_alerts():
    """Background task to check budget alerts"""
    while True:
        try:
            # Check budgets every 5 minutes
            await asyncio.sleep(300)

            # TODO: Get all keys with budgets
            # For each key, check if alert threshold reached
            # Send alert email if needed

            # Example:
            # for key_id in keys_with_budgets:
            #     status = ailb_server.cost_tracker.get_budget_status(key_id)
            #     if status.alert_triggered and not status.alert_sent:
            #         await send_budget_alert(key_id, status)

        except Exception as e:
            logger.error(f"Budget alert check failed: {e}")

# In startup()
async def startup(self):
    # ... existing startup code ...

    # Start budget alert checker
    self.alert_task = asyncio.create_task(check_budget_alerts())
```

## 8. Environment Variables

Add to `.env`:

```bash
# Billing Configuration
BILLING_ENABLED=true
BILLING_DEFAULT_CURRENCY=USD
BILLING_ALERT_FROM_EMAIL=billing@example.com
BILLING_ALERT_SMTP_HOST=smtp.example.com
BILLING_ALERT_SMTP_PORT=587
```

## 9. Testing the Integration

### Test Cost Tracking

```bash
# Make a request with an API key
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-test-123" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'

# Check spending
curl -H "X-User-ID: user123" \
  http://localhost:8080/api/billing/spend/key_test_123?period=daily
```

### Test Budget Enforcement

```bash
# Set a low budget
curl -X PUT -H "X-User-ID: user123" \
  -H "Content-Type: application/json" \
  -d '{
    "key_id": "key_test_123",
    "max_budget": 0.01,
    "alert_threshold": 0.5,
    "period": "daily",
    "enabled": true
  }' \
  http://localhost:8080/api/billing/budget/key_test_123

# Check budget status
curl -H "X-User-ID: user123" \
  http://localhost:8080/api/billing/budget/key_test_123

# Try to make requests until budget exceeded (should get 402 error)
```

## 10. Database Migration (TODO)

When ready to migrate from in-memory to PostgreSQL:

1. Create database tables (see README.md)
2. Update `tracker.py` to use database instead of in-memory lists
3. Add database connection to `AILBServer.__init__()`
4. Implement data migration script for existing in-memory data

## Complete Integration Checklist

- [ ] Import billing router in main.py
- [ ] Initialize CostTracker in AILBServer
- [ ] Add budget pre-check to chat completions endpoint
- [ ] Record usage after serving requests
- [ ] Integrate with virtual keys module
- [ ] Update key usage endpoint to use billing data
- [ ] Add environment variables
- [ ] Test cost tracking with real requests
- [ ] Test budget enforcement
- [ ] Set up budget alerts (optional)
- [ ] Plan database migration (for production)

## Troubleshooting

### Usage not recording

- Check that key_id is being extracted correctly from API key
- Verify usage_info contains token counts from LLM provider
- Check logs for recording errors (non-fatal)

### Budget checks failing incorrectly

- Verify budget period dates are correct
- Check that usage records have correct timestamps
- Ensure budget is enabled in config

### Pricing incorrect

- Update MODEL_PRICING dict in tracker.py
- Or use PUT /api/billing/pricing endpoint (admin)
- Verify model name matches exactly (case-insensitive)

## Next Steps

1. Complete API key validation integration
2. Add user authentication to billing endpoints
3. Implement budget alert email sending
4. Create admin dashboard for cost analytics
5. Plan PostgreSQL migration strategy
6. Add more detailed usage metrics (latency, errors, etc.)
