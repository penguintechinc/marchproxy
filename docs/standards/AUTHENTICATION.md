# 🔐 Authentication Standards

Part of [Development Standards](../STANDARDS.md)

## Overview

marchproxy uses **penguin-aaa** for OIDC-compliant token validation and **scope-based RBAC**. Authentication flow:

1. User logs in (credentials validated, JWT issued)
2. JWT includes OIDC-standard claims: `sub`, `iss`, `aud`, `exp`, `scope`, `tenant`, `roles`
3. Client passes JWT in `Authorization: Bearer` header
4. Services validate token signature and check required scopes (never role names)
5. Authorization middleware enforces tenant isolation and scope checks

---

## JWT Token Format

All tokens (user and service) follow OIDC standard structure:

```json
{
  "sub": "user-uuid-or-service-id",
  "iss": "https://auth-service-url",
  "aud": ["marchproxy-manager"],
  "iat": 1234567890,
  "exp": 1234567899,
  "scope": "clusters:read services:write",
  "tenant": "tenant-id",
  "roles": ["maintainer"],
  "teams": ["team-id"]
}
```

**Key claims:**
- `sub` - User or service identity (UUID)
- `scope` - Space-separated list of authorized actions (see table below)
- `tenant` - Tenant ID; required for multi-tenant isolation
- `roles` - Informational only; actual permissions determined by `scope`

---

## Standard Scopes Reference

| Scope | Resource | Action | Typical Users |
|-------|----------|--------|---|
| `clusters:read` | Clusters | View only | Viewer, Maintainer, Admin |
| `clusters:write` | Clusters | Create/edit | Maintainer, Admin |
| `clusters:admin` | Clusters | Delete, manage | Admin only |
| `services:read` | Services | View only | Viewer, Maintainer, Admin |
| `services:write` | Services | Create/edit | Maintainer, Admin |
| `services:admin` | Services | Delete, manage | Admin only |
| `users:read` | Users | View only | Admin |
| `users:write` | Users | Create/edit | Admin |
| `users:admin` | Users | Delete, manage | Admin |
| `certificates:read` | Certificates | View only | Maintainer, Admin |
| `certificates:write` | Certificates | Issue/manage | Maintainer, Admin |
| `config:read` | Configuration | View only | Maintainer, Admin |
| `config:write` | Configuration | Edit | Admin only |
| `proxies:read` | Proxies | View only | Viewer, Maintainer, Admin |
| `proxies:manage` | Proxies | Control (start/stop) | Maintainer, Admin |
| `*:admin` | All resources | Full access | Admin role only |

---

## Manager (Quart) - Decorators & Usage

**Validation occurs via `manager/middleware/auth.py` and `manager/middleware/rbac.py`**

### Basic Auth Decorator (Quart)

```python
from manager.middleware.auth import require_auth, get_current_user

# Require valid JWT token
@app.route('/api/v1/clusters')
@require_auth()
async def list_clusters():
    user = get_current_user()
    print(f"User: {user.id}, Scope: {user.scope}")
    return {'clusters': []}

# Require admin role
@app.route('/api/v1/admin/settings')
@require_auth(admin_required=True)
async def admin_settings():
    return {'settings': {}}

# License-gated feature
@app.route('/api/v1/sso/saml')
@require_auth(license_feature='saml_sso')
async def saml_config():
    return {'sso': {}}
```

### Scope-Based Authorization (Quart)

```python
from manager.middleware.rbac import requires_permission, requires_role

# Single scope required
@app.route('/api/v1/clusters/<cluster_id>')
@requires_permission('clusters:read')
async def get_cluster(cluster_id):
    return {'cluster': cluster_id}

# Multiple scopes (any one required)
@app.route('/api/v1/clusters/<cluster_id>', methods=['PUT'])
@requires_permission('clusters:write', 'clusters:admin')
async def update_cluster(cluster_id):
    return {'updated': cluster_id}

# All scopes required (AND check)
@app.route('/api/v1/critical-action')
@requires_all_permissions('clusters:admin', 'users:admin')
async def critical_action():
    return {'result': 'done'}

# Role-based with scope fallback (rarely used)
@app.route('/api/v1/sensitive')
@requires_role('admin', scope='global')
async def sensitive_endpoint():
    return {'data': {}}

# Resource-level access check
@app.route('/api/v1/clusters/<cluster_id>/status')
@requires_permission('clusters:read')
async def cluster_status(cluster_id):
    user = get_current_user()
    if not can_access_cluster(user.id, cluster_id):
        abort(403, 'No access to this cluster')
    return {'status': 'ok'}
```

---

## API-Server (FastAPI) - Dependencies & Usage

**Token validation occurs via `app/dependencies.py` and `app/core/security.py`**

### Basic User Dependency (FastAPI)

```python
from fastapi import FastAPI, Depends, HTTPException
from app.dependencies import get_current_user, require_admin
from app.models.sqlalchemy.user import User

app = FastAPI()

# Any authenticated user
@app.get('/api/v1/profile')
async def get_profile(user: User = Depends(get_current_user)):
    return {'email': user.email, 'id': user.id}

# Admin only
@app.get('/api/v1/admin/users')
async def list_users(user: User = Depends(require_admin)):
    return {'users': []}

# License-gated feature validation
@app.post('/api/v1/sso/config')
async def configure_sso(user: User = Depends(require_admin)):
    feature = request.headers.get('X-License-Feature', 'saml_sso')
    from app.dependencies import validate_license_feature
    if not validate_license_feature(feature):
        raise HTTPException(status_code=402, detail='Feature not licensed')
    return {'configured': True}
```

### Scope-Based Checks (FastAPI)

```python
from fastapi import Request

# Manual scope validation
@app.get('/api/v1/clusters')
async def list_clusters(user: User = Depends(get_current_user), request: Request = None):
    if 'clusters:read' not in user.scope.split():
        raise HTTPException(status_code=403, detail='clusters:read scope required')
    return {'clusters': []}

# OR check (multiple acceptable scopes)
@app.put('/api/v1/clusters/{cluster_id}')
async def update_cluster(
    cluster_id: str,
    user: User = Depends(get_current_user)
):
    required = {'clusters:write', 'clusters:admin'}
    user_scopes = set(user.scope.split())
    if not user_scopes & required:
        raise HTTPException(status_code=403, detail='Insufficient scope')
    return {'updated': cluster_id}

# AND check (all scopes required)
@app.delete('/api/v1/clusters/{cluster_id}')
async def delete_cluster(
    cluster_id: str,
    user: User = Depends(get_current_user)
):
    required = {'clusters:admin', 'users:admin'}
    user_scopes = set(user.scope.split())
    if not required.issubset(user_scopes):
        raise HTTPException(status_code=403, detail='All scopes required')
    return {'deleted': cluster_id}
```

---

## Two-Factor Authentication (TOTP)

API-Server supports optional TOTP 2FA via RFC 6238 standard:

```python
# In app/core/security.py
from pyotp import TOTP

# Verify TOTP code
def verify_totp_code(secret: str, code: str) -> bool:
    totp = TOTP(secret)
    return totp.verify(code)

# Generate QR code for user enrollment
def generate_totp_secret() -> str:
    return pyotp.random_base32()

def get_totp_uri(user_email: str, secret: str) -> str:
    totp = TOTP(secret)
    return totp.provisioning_uri(name=user_email, issuer_name='marchproxy')
```

Login flow with 2FA:
1. User submits email + password
2. If TOTP enabled, return challenge requiring 6-digit code
3. User scans QR code in authenticator app or enters code
4. Server validates code via `verify_totp_code()`
5. JWT issued only after successful code verification

---

## Tenant Isolation (Mandatory)

Every request must be scoped to the user's tenant. Tenant validation runs **first, before any scope check**:

```python
# Quart example
@app.before_request
async def validate_tenant():
    from quart import g, request, abort
    from manager.middleware.auth import get_current_user
    
    user = get_current_user()
    if not user:
        return  # Unauthenticated; let @require_auth handle it
    
    # Tenant mismatch = immediate 403
    request_tenant = request.headers.get('X-Tenant-ID')
    if request_tenant and request_tenant != user.tenant:
        abort(403, 'Tenant mismatch')

# All queries scoped by tenant
@app.route('/api/v1/clusters')
@require_auth()
async def list_clusters():
    user = get_current_user()
    clusters = Cluster.query.filter_by(tenant_id=user.tenant).all()
    return {'clusters': []}
```

---

## Service-to-Service Authentication

For microservice communication, use **SPIFFE/SPIRE** (preferred) or **OIDC machine JWTs**:

**SPIFFE format:**
```
spiffe://penguintech.io/{environment}/{service}
# Examples:
spiffe://penguintech.io/prod/api-server
spiffe://penguintech.io/prod/manager
```

**Machine JWT (fallback):**
Same OIDC structure as user tokens, but:
- `sub` = service name (e.g., `api-server`)
- `scope` = service-scoped permissions
- `exp` = short (15–60 minutes)

---

## Practical Testing

### Test Token-Protected Endpoint

```bash
# Get token
TOKEN=$(curl -s -X POST http://localhost:8000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@localhost.local","password":"admin123"}' | jq -r .access_token)

# Call protected endpoint with Bearer token
curl -H "Authorization: Bearer $TOKEN" http://localhost:8000/api/v1/clusters
```

### Decode JWT (for debugging)

```bash
# Install jq if needed
TOKEN="your-jwt-token-here"
echo $TOKEN | cut -d. -f2 | base64 -d | jq .
```

### Test Scope Validation

```bash
# Create token with limited scope
TOKEN=$(curl -s -X POST ... | jq -r .access_token)

# This should work (user has clusters:read)
curl -H "Authorization: Bearer $TOKEN" http://localhost:8000/api/v1/clusters

# This should fail with 403 if no clusters:write scope
curl -X POST http://localhost:8000/api/v1/clusters \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"test"}'
```

---

## Troubleshooting

| Issue | Cause | Fix |
|-------|-------|-----|
| "Token expired" | JWT exp claim in past | Issue new token; check server clock |
| "Invalid signature" | Token tampered or wrong key | Verify SECRET_KEY matches issuer |
| "Missing scope" | User lacks required scope | Assign appropriate role/scope bundle |
| "Tenant mismatch" | X-Tenant-ID header != token tenant | Remove custom tenant header or match token |
| "401 Unauthorized" | No token in header | Include `Authorization: Bearer {token}` |
| "403 Forbidden" | Token valid but insufficient scope | Add scope to user role or create new role |

---

## Key Files

- Manager (Quart): `/manager/middleware/auth.py` (token validation), `/manager/middleware/rbac.py` (scope checks)
- API-Server (FastAPI): `/api-server/app/dependencies.py` (HTTPBearer), `/api-server/app/core/security.py` (JWT ops)
- Database: User roles and scopes defined in migration `002_kong_entities.py`

---

**Next Steps:** See [Database Standards](DATABASE.md) for user/role schema, [API Standards](API_PROTOCOLS.md) for endpoint patterns.
