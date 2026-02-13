# Virtual Key Management System

This module provides a comprehensive virtual API key management system for the MarchProxy AILB (AI Load Balancer).

## Overview

The virtual key system allows for:
- Generation and management of API keys with `sk-mp-` prefix
- Budget tracking and enforcement (USD)
- Rate limiting (TPM - Tokens Per Minute, RPM - Requests Per Minute)
- Model access control
- Usage tracking and analytics
- Key rotation and revocation

## Components

### 1. Models (`models.py`)

Pydantic models for data validation:

- **VirtualKey**: Core key data model with validation
- **KeyCreate**: Request model for creating keys
- **KeyUpdate**: Request model for updating keys
- **KeyResponse**: Response model with computed fields
- **KeyUsage**: Usage tracking record
- **KeyValidationResult**: Result of key validation

### 2. Manager (`manager.py`)

Core business logic for key management:

- `generate_key()`: Create new virtual key with `sk-mp-{id}-{secret}` format
- `validate_key()`: Validate key and check all constraints
- `get_key()`: Retrieve key details
- `list_keys()`: List keys with filtering
- `update_key()`: Update key settings
- `delete_key()`: Soft delete (deactivate)
- `record_usage()`: Track token usage and costs
- `check_rate_limit()`: Verify TPM/RPM limits
- `rotate_key()`: Generate new secret for existing key

**Storage**: Currently uses in-memory storage (dict). Contains TODO markers for PostgreSQL migration.

### 3. Routes (`routes.py`)

FastAPI endpoints for key management:

#### Endpoints

- `POST /api/keys` - Create new virtual key
- `GET /api/keys` - List all keys for user
- `GET /api/keys/{key_id}` - Get specific key details
- `PUT /api/keys/{key_id}` - Update key settings
- `DELETE /api/keys/{key_id}` - Delete (deactivate) key
- `POST /api/keys/{key_id}/rotate` - Rotate key secret
- `GET /api/keys/{key_id}/usage` - Get usage statistics
- `POST /api/keys/validate` - Validate API key (internal)

## Usage Examples

### Creating a Key

```python
from app.keys import KeyManager
from app.keys.models import KeyCreate

manager = KeyManager()

# Create key request
key_create = KeyCreate(
    name="Production API Key",
    user_id="user_123",
    team_id="team_456",
    expires_days=365,
    allowed_models=["gpt-4", "claude-3-opus"],
    max_budget=100.0,  # $100 USD
    tpm_limit=10000,   # 10k tokens per minute
    rpm_limit=60       # 60 requests per minute
)

# Generate key
api_key, virtual_key = manager.generate_key(key_create)

print(f"API Key: {api_key}")  # sk-mp-abc123...-xyz789...
print(f"Key ID: {virtual_key.id}")
```

### Validating a Key

```python
# Validate incoming API key
result = manager.validate_key("sk-mp-abc123...-xyz789...")

if result.valid:
    print(f"Key valid: {result.key_id}")
    print(f"Rate limits: {result.rate_limit_info}")
else:
    print(f"Invalid: {result.error}")
```

### Recording Usage

```python
# Record usage after LLM request
success = manager.record_usage(
    key_id="abc123...",
    tokens=1500,
    cost=0.045,  # $0.045
    model="gpt-4",
    provider="openai",
    request_id="req_xyz"
)
```

### Listing Keys

```python
# Get all keys for a user
keys = manager.list_keys(user_id="user_123")

# Filter by status
from app.keys.models import KeyStatus
active_keys = manager.list_keys(
    user_id="user_123",
    status=KeyStatus.ACTIVE
)
```

### Rotating a Key

```python
# Rotate key secret (invalidates old key)
result = manager.rotate_key("abc123...")
if result:
    new_api_key, updated_key = result
    print(f"New API Key: {new_api_key}")
```

## API Key Format

Keys follow the format: `sk-mp-{key_id}-{secret}`

- `sk-mp`: Prefix (MarchProxy)
- `{key_id}`: 16-byte hex identifier (32 chars)
- `{secret}`: 32-byte URL-safe secret (43 chars)

Example: `sk-mp-a1b2c3d4e5f67890-xYz123AbC456DeF789GhI012JkL345MnO678PqR`

## Rate Limiting

The system tracks:
- **TPM (Tokens Per Minute)**: Total tokens consumed in current minute
- **RPM (Requests Per Minute)**: Total requests made in current minute

Rate limit data is cleaned up automatically (default: 2 hours retention).

## Budget Tracking

- Tracks total spend in USD
- Automatically blocks requests when budget exceeded
- Budget checks performed during validation

## Security Features

- Keys are hashed using SHA-256 before storage
- Full key only returned once during creation/rotation
- Constant-time comparison for validation
- Soft delete prevents accidental data loss

## TODO: PostgreSQL Integration

Current implementation uses in-memory storage. Migration to PostgreSQL needed:

1. Create database schema:
   - `virtual_keys` table
   - `key_usage` table
   - Indexes on user_id, team_id, status

2. Replace in-memory dicts with database queries
3. Implement connection pooling
4. Add database migration scripts
5. Update `_persist_key_to_redis()` with PostgreSQL persistence

## Integration with AILB

To integrate with the main AILB application:

1. Import the router in `main.py`:
```python
from app.keys.routes import router as keys_router
app.include_router(keys_router)
```

2. Add key validation middleware:
```python
from app.keys import KeyManager

key_manager = KeyManager()

@app.middleware("http")
async def validate_api_key(request: Request, call_next):
    auth = request.headers.get("Authorization")
    if auth:
        result = key_manager.validate_key(auth)
        if not result.valid:
            return JSONResponse(
                status_code=401,
                content={"error": result.error}
            )
        request.state.key_id = result.key_id
    return await call_next(request)
```

3. Record usage after requests:
```python
# In chat completions endpoint
usage_info = response.get("usage", {})
if hasattr(request.state, "key_id"):
    key_manager.record_usage(
        key_id=request.state.key_id,
        tokens=usage_info.get("total_tokens", 0),
        cost=calculated_cost,
        model=model_name,
        provider=provider_name
    )
```

## Testing

Example test scenarios:

1. Key creation and validation
2. Budget enforcement
3. Rate limiting (TPM/RPM)
4. Expiration handling
5. Key rotation
6. Model access control
7. Usage tracking accuracy

## Dependencies

- `pydantic>=2.0` - Data validation
- `fastapi>=0.100` - API framework
- No external dependencies for key generation (uses stdlib `secrets`, `hashlib`)
