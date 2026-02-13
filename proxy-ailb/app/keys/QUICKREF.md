# Virtual Key Management - Quick Reference

## Key Format

```
sk-mp-{key_id}-{secret}
```

Example: `sk-mp-a1b2c3d4e5f67890-xYz123AbC456DeF789GhI012JkL345MnO678PqR`

## Import

```python
from app.keys import KeyManager
from app.keys.models import KeyCreate, KeyUpdate, KeyStatus
```

## Create Key

```python
manager = KeyManager()

key_data = KeyCreate(
    name="My Key",
    user_id="user_123",
    expires_days=365,          # Optional
    max_budget=100.0,          # Optional, USD
    tpm_limit=10000,           # Optional, tokens/minute
    rpm_limit=60,              # Optional, requests/minute
    allowed_models=["gpt-4"]   # Optional, default ["*"]
)

api_key, virtual_key = manager.generate_key(key_data)
# Save api_key - it's only shown once!
```

## Validate Key

```python
result = manager.validate_key(api_key)

if result.valid:
    print(f"Key ID: {result.key_id}")
    print(f"Rate limits: {result.rate_limit_info}")
else:
    print(f"Invalid: {result.error}")
```

## Record Usage

```python
manager.record_usage(
    key_id="abc123",
    tokens=1500,
    cost=0.045,
    model="gpt-4",
    provider="openai"
)
```

## List Keys

```python
# All keys for user
keys = manager.list_keys(user_id="user_123")

# Filter by status
active_keys = manager.list_keys(
    user_id="user_123",
    status=KeyStatus.ACTIVE
)
```

## Update Key

```python
update = KeyUpdate(
    name="Updated Name",
    max_budget=200.0,
    is_active=False
)

manager.update_key(key_id, update)
```

## Rotate Key

```python
new_api_key, updated_key = manager.rotate_key(key_id)
# Old key is now invalid
# Save new_api_key - only shown once!
```

## Delete Key

```python
manager.delete_key(key_id)  # Soft delete (deactivates)
```

## Get Usage Stats

```python
stats = manager.get_usage_stats(key_id, days=30)

print(f"Total tokens: {stats['total_tokens']}")
print(f"Total cost: ${stats['total_cost']}")
print(f"Model breakdown: {stats['model_breakdown']}")
```

## REST API Endpoints

```bash
# Create key
POST /api/keys
Header: X-User-ID: user_123
Body: {"name": "My Key", "max_budget": 100}

# List keys
GET /api/keys
Header: X-User-ID: user_123

# Get key details
GET /api/keys/{key_id}
Header: X-User-ID: user_123

# Update key
PUT /api/keys/{key_id}
Header: X-User-ID: user_123
Body: {"name": "New Name", "max_budget": 200}

# Delete key
DELETE /api/keys/{key_id}
Header: X-User-ID: user_123

# Rotate key
POST /api/keys/{key_id}/rotate
Header: X-User-ID: user_123

# Get usage
GET /api/keys/{key_id}/usage?days=30
Header: X-User-ID: user_123

# Validate key (internal)
POST /api/keys/validate
Header: Authorization: Bearer sk-mp-...
```

## Using Keys with AILB

```bash
curl -X POST "http://localhost:8080/v1/chat/completions" \
  -H "Authorization: Bearer sk-mp-abc123..." \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

## Error Codes

- `401`: Invalid/expired/budget exceeded
- `403`: Access denied (not key owner)
- `404`: Key not found
- `429`: Rate limit exceeded

## Status Values

- `active`: Key is valid and usable
- `inactive`: Key is deactivated
- `expired`: Key expiration date passed
- `revoked`: Budget exceeded

## Files

```
/home/penguin/code/MarchProxy/proxy-ailb/app/keys/
├── models.py          # Pydantic models
├── manager.py         # Core logic
├── routes.py          # FastAPI endpoints
├── __init__.py        # Module exports
├── README.md          # Full documentation
├── INTEGRATION.md     # Integration guide
├── SUMMARY.md         # Implementation summary
├── QUICKREF.md        # This file
├── example.py         # Full examples
└── test_basic.py      # Basic test
```

## Next Steps

1. Install: `pip install -r requirements.txt`
2. Test: `python3 app/keys/test_basic.py`
3. Example: `python3 app/keys/example.py`
4. Integrate: See INTEGRATION.md
