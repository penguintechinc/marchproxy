# Role-Based Access Control (RBAC) Guide

Complete guide to using the MarchProxy RBAC system with OAuth2-style scoped permissions.

## Overview

MarchProxy implements a comprehensive RBAC system with:
- **OAuth2-style scoped permissions**: Fine-grained access control
- **Three permission levels**: Global, Cluster, Service
- **Default roles**: Admin, Maintainer, Viewer, Cluster Admin, Service Owner
- **Custom roles**: Create application-specific roles
- **Permission caching**: High-performance permission checks

## Architecture

### Permission Scopes

```
Global (System-wide)
├── global:admin - Full system access
├── global:users:read - Read all users
├── global:users:write - Manage all users
├── global:clusters:read - Read all clusters
├── global:clusters:write - Manage all clusters
├── global:services:read - Read all services
└── global:services:write - Manage all services

Cluster (Cluster-specific)
├── cluster:read - Read cluster details
├── cluster:write - Update cluster
├── cluster:services:read - Read cluster services
└── cluster:services:write - Manage cluster services

Service (Service-specific)
├── service:read - Read service details
├── service:write - Update service
├── service:proxies:read - Read service proxies
└── service:certs:write - Manage certificates
```

### Default Roles

| Role | Scope | Description | Permissions |
|------|-------|-------------|-------------|
| **Admin** | Global | Full system access | All permissions |
| **Maintainer** | Global | Read/write, no user mgmt | Cluster/service read/write |
| **Viewer** | Global | Read-only access | All read permissions |
| **Cluster Admin** | Cluster | Manage specific cluster | Full cluster permissions |
| **Service Owner** | Service | Manage specific service | Full service permissions |

## Decorator Reference

**All RBAC decorators are imported from `middleware.rbac`:**

```python
from middleware.rbac import (
    requires_permission,
    requires_role,
    requires_any_permission,
    requires_all_permissions
)
```

### Decorator Signatures

#### `@requires_permission(permission, resource_type=None, resource_id_param=None)`

Check a single specific permission. Optionally scope to a resource (cluster or service).

**Decorator Arguments:**
- `permission` (str): Permission to require (e.g., `Permissions.GLOBAL_ADMIN`)
- `resource_type` (str, optional): `'cluster'` or `'service'` for scoped checks
- `resource_id_param` (str, optional): Name of route parameter containing resource ID

**Returns:** 401 if unauthenticated, 403 if insufficient permissions

#### `@requires_role(role_name, scope=PermissionScope.GLOBAL, resource_id_param=None)`

Check if user has a specific role at a given scope.

**Decorator Arguments:**
- `role_name` (str): Role name (e.g., `'admin'`, `'cluster_admin'`)
- `scope` (PermissionScope): `GLOBAL`, `CLUSTER`, or `SERVICE`
- `resource_id_param` (str, optional): Name of route parameter for scoped role checks

**Returns:** 401 if unauthenticated, 403 if role not assigned

#### `@requires_any_permission(*permissions)`

Check if user has ANY of the specified permissions (OR logic).

**Decorator Arguments:**
- `*permissions` (str...): One or more permission strings to check

**Returns:** 401 if unauthenticated, 403 if no matching permission found

#### `@requires_all_permissions(*permissions)`

Check if user has ALL specified permissions (AND logic).

**Decorator Arguments:**
- `*permissions` (str...): All permission strings required

**Returns:** 401 if unauthenticated, 403 if any permission missing

---

## Usage Guide

### 1. Protecting Routes with Decorators

#### Require Specific Permission (Global Scope)

```python
from middleware.rbac import requires_permission
from models.rbac import Permissions  # Import permission constants

@app.route('/api/v1/clusters', methods=['POST'])
@requires_permission(Permissions.GLOBAL_CLUSTER_WRITE)
async def create_cluster():
    """Only users with global:clusters:write can access"""
    return jsonify({'message': 'Cluster created'})
```

#### Require Permission with Resource Scope

```python
@app.route('/api/v1/clusters/<int:cluster_id>', methods=['PUT'])
@requires_permission(
    Permissions.CLUSTER_WRITE,
    resource_type='cluster',
    resource_id_param='cluster_id'
)
async def update_cluster(cluster_id: int):
    """Checks if user has cluster:write permission for this specific cluster"""
    return jsonify({'message': f'Cluster {cluster_id} updated'})
```

#### Require Specific Role (Global Scope)

```python
from middleware.rbac import requires_role
from models.rbac import PermissionScope

@app.route('/api/v1/admin/dashboard')
@requires_role('admin', scope=PermissionScope.GLOBAL)
async def admin_dashboard():
    """Only users with Admin role (global scope) can access"""
    return jsonify({'message': 'Admin dashboard'})
```

#### Require Scoped Role (Cluster-Specific)

```python
@app.route('/api/v1/clusters/<int:cluster_id>/manage', methods=['POST'])
@requires_role('cluster_admin', scope=PermissionScope.CLUSTER, resource_id_param='cluster_id')
async def manage_cluster(cluster_id: int):
    """Only users with cluster_admin role for this specific cluster can access"""
    return jsonify({'message': f'Cluster {cluster_id} managed'})
```

#### Require ANY of Multiple Permissions (OR Logic)

```python
from middleware.rbac import requires_any_permission

@app.route('/api/v1/resources')
@requires_any_permission(
    Permissions.GLOBAL_ADMIN,
    Permissions.GLOBAL_CLUSTER_READ
)
async def list_resources():
    """Users with either permission can access"""
    return jsonify({'resources': []})
```

#### Require ALL Permissions (AND Logic)

```python
from middleware.rbac import requires_all_permissions

@app.route('/api/v1/critical-operation')
@requires_all_permissions(
    Permissions.GLOBAL_CLUSTER_WRITE,
    Permissions.GLOBAL_SERVICE_WRITE
)
async def critical_operation():
    """Requires both permissions"""
    return jsonify({'status': 'success'})
```

### 2. Programmatic Permission Checks

#### Check Permission in Code

```python
from models.rbac import RBACModel, Permissions

async def my_function():
    user_id = g.user_id
    db = g.db

    # Check global permission
    if RBACModel.has_permission(db, user_id, Permissions.GLOBAL_ADMIN):
        # User is admin
        pass

    # Check cluster-specific permission
    if RBACModel.has_permission(
        db, user_id,
        Permissions.CLUSTER_WRITE,
        'cluster',
        cluster_id
    ):
        # User can write to this cluster
        pass
```

#### Get All User Permissions

```python
permissions = RBACModel.get_user_permissions(db, user_id)
# Returns:
# {
#     'global': ['global:admin', 'global:users:read', ...],
#     'cluster': {'123': ['cluster:read', 'cluster:write'], ...},
#     'service': {'456': ['service:read', ...], ...}
# }
```

### 3. Managing Roles

#### Assign Role to User

```python
from models.rbac import RBACModel, PermissionScope

# Assign global role
RBACModel.assign_role(
    db,
    user_id=user_id,
    role_name='admin',
    scope=PermissionScope.GLOBAL,
    granted_by=admin_user_id
)

# Assign cluster-scoped role
RBACModel.assign_role(
    db,
    user_id=user_id,
    role_name='cluster_admin',
    scope=PermissionScope.CLUSTER,
    resource_id=cluster_id,
    granted_by=admin_user_id
)

# Assign service-scoped role
RBACModel.assign_role(
    db,
    user_id=user_id,
    role_name='service_owner',
    scope=PermissionScope.SERVICE,
    resource_id=service_id,
    granted_by=admin_user_id
)
```

#### Revoke Role from User

```python
# Revoke global role
RBACModel.revoke_role(db, user_id, 'admin')

# Revoke scoped role
RBACModel.revoke_role(db, user_id, 'cluster_admin', resource_id=cluster_id)
```

#### Get User Roles

```python
roles = RBACModel.get_user_roles(db, user_id)
# Returns list of role assignments with scope and resource info
```

### 4. Creating Custom Roles

#### Define Custom Role

```python
# Custom role for billing management
custom_role_id = db.roles.insert(
    name='billing_manager',
    display_name='Billing Manager',
    description='Manages billing and invoices',
    scope=PermissionScope.GLOBAL.value,
    permissions=[
        'global:billing:read',
        'global:billing:write',
        'global:invoices:read',
        'global:invoices:write',
    ],
    is_system=False,
    is_active=True
)
db.commit()
```

#### Assign Custom Role

```python
RBACModel.assign_role(
    db,
    user_id=user_id,
    role_name='billing_manager',
    scope=PermissionScope.GLOBAL,
    granted_by=admin_id
)
```

## API Endpoints

### List Roles
```http
GET /api/v1/roles
Authorization: Bearer <token>

Response:
{
  "roles": [
    {
      "id": 1,
      "name": "admin",
      "display_name": "Admin",
      "scope": "global",
      "permissions": ["global:admin", ...]
    }
  ]
}
```

### Get Role Details
```http
GET /api/v1/roles/{role_id}
Authorization: Bearer <token>

Response:
{
  "role": {...},
  "assignments": [
    {
      "user_id": 1,
      "username": "john",
      "scope": "global",
      "granted_at": "2026-01-13T..."
    }
  ]
}
```

### Create Custom Role
```http
POST /api/v1/roles
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "custom_role",
  "display_name": "Custom Role",
  "description": "Custom role description",
  "scope": "global",
  "permissions": ["global:custom:read", "global:custom:write"]
}
```

### Assign Role to User
```http
POST /api/v1/roles/assign
Authorization: Bearer <token>
Content-Type: application/json

{
  "user_id": 123,
  "role_name": "maintainer",
  "scope": "global"
}

# For scoped role:
{
  "user_id": 123,
  "role_name": "cluster_admin",
  "scope": "cluster",
  "resource_id": 456
}
```

### Revoke Role
```http
POST /api/v1/roles/revoke
Authorization: Bearer <token>
Content-Type: application/json

{
  "user_id": 123,
  "role_name": "maintainer",
  "resource_id": null  # or cluster/service ID
}
```

### Get User Roles and Permissions
```http
GET /api/v1/roles/user/{user_id}
Authorization: Bearer <token>

Response:
{
  "user_id": 123,
  "username": "john",
  "roles": [...],
  "permissions": {
    "global": [...],
    "cluster": {...},
    "service": {...}
  }
}
```

### List Available Permissions
```http
GET /api/v1/roles/permissions

Response:
{
  "permissions": ["global:admin", ...],
  "scopes": {
    "global": [...],
    "cluster": [...],
    "service": [...]
  }
}
```

## Helper Functions

The `middleware.rbac` module provides convenient helper functions for common permission checks:

### Imported from `middleware.rbac`

```python
from middleware.rbac import (
    is_admin,
    can_manage_users,
    can_access_cluster,
    can_access_service
)
```

### Function Reference

#### `is_admin(user_id: int, db) -> bool`

Check if user is a global admin.

```python
if is_admin(user_id, db):
    # User has global:admin permission
    pass
```

#### `can_manage_users(user_id: int, db) -> bool`

Check if user can manage other users (admin or has user write permission).

```python
if can_manage_users(user_id, db):
    # User has global:admin OR global:users:write
    pass
```

#### `can_access_cluster(user_id: int, cluster_id: int, db) -> bool`

Check if user can read a specific cluster.

```python
if can_access_cluster(user_id, cluster_id, db):
    # User has cluster:read permission for this cluster
    pass
```

#### `can_access_service(user_id: int, service_id: int, db) -> bool`

Check if user can read a specific service.

```python
if can_access_service(user_id, service_id, db):
    # User has service:read permission for this service
    pass
```

## Migration Guide

### Migrating Existing Code

**Before (basic is_admin check):**
```python
@app.route('/api/admin')
async def admin_endpoint():
    user = g.current_user
    if not user.is_admin:
        abort(403)
    return jsonify({'data': 'admin data'})
```

**After (RBAC with permissions):**
```python
from middleware.rbac import requires_permission
from models.rbac import Permissions

@app.route('/api/admin')
@requires_permission(Permissions.GLOBAL_ADMIN)
async def admin_endpoint():
    # Automatic permission check
    return jsonify({'data': 'admin data'})
```

### Running the Migration

```bash
# Run RBAC migration
cd manager
python migrations/add_rbac_tables.py

# Or via database migration system
python migrate.py upgrade
```

## Authorization Model: Scope-Based (OIDC Style)

MarchProxy RBAC is built on **scope-based authorization** (OIDC/OAuth2 style), NOT role-name-based. This is critical for security:

### Key Principles

1. **Scopes are the source of truth for authorization decisions**
   - Application code checks scopes (e.g., `'global:clusters:write'`)
   - Roles are pre-bundled scope sets for convenience
   - Roles are informational only — never branch on role names in code

2. **Roles as Scope Bundles**
   - Each role (Admin, Maintainer, Viewer, etc.) carries a fixed set of scopes
   - When a role is assigned to a user, those scopes are granted
   - Permission checks happen against scopes, not role names

3. **Three Scope Levels**
   - **Global**: System-wide permissions (`global:clusters:write`, `global:users:admin`)
   - **Cluster**: Scoped to a specific cluster (`cluster:read`, `cluster:write`)
   - **Service**: Scoped to a specific service (`service:read`, `service:write`)

### Authorization Middleware Pattern

All decorators (`@requires_permission`, `@requires_role`, etc.) ultimately check scopes:

```python
# These all check scopes, not role names
@requires_permission('global:clusters:write')    # Direct scope check
@requires_role('admin')                          # Checks if user's scopes include admin's scope bundle
@requires_any_permission(scope1, scope2)         # Checks if user has any of these scopes
```

### ❌ What NOT to Do

Never write authorization logic based on role names:

```python
# WRONG - checking role names
if user.role == 'admin':
    # Allow operation
    
if user.role_name == 'maintainer':
    # Allow operation
```

This is insecure because roles can change, be renamed, or be misused. Always check scopes instead.

### ✅ What TO Do

Always check scopes in your middleware/decorators:

```python
# RIGHT - checking scopes
@requires_permission(Permissions.GLOBAL_ADMIN)
async def admin_operation():
    pass

# Or programmatically
if RBACModel.has_permission(db, user_id, Permissions.GLOBAL_ADMIN):
    # Allow operation
    pass
```

---

## Best Practices

1. **Use Most Specific Permission**: Always use the most specific permission required
   ```python
   # Good
   @requires_permission(Permissions.CLUSTER_WRITE, 'cluster', 'cluster_id')

   # Avoid (too broad)
   @requires_permission(Permissions.GLOBAL_ADMIN)
   ```

2. **Scope Permissions Appropriately**: Use scoped permissions for resources
   ```python
   # Cluster-specific operations use cluster scope
   @requires_permission(Permissions.CLUSTER_WRITE, 'cluster', 'cluster_id')

   # Service-specific operations use service scope
   @requires_permission(Permissions.SERVICE_WRITE, 'service', 'service_id')
   ```

3. **Cache Performance**: Permission checks are cached automatically
   - Cache invalidated on role changes
   - Cache per-user for performance

4. **Custom Permissions**: Create custom permissions for app-specific features
   ```python
   # Define in Permissions class
   class Permissions:
       CUSTOM_FEATURE = "global:custom:feature"

   # Use in decorator
   @requires_permission(Permissions.CUSTOM_FEATURE)
   ```

5. **Audit Logging**: Track permission changes
   ```python
   # Role assignments include granted_by and granted_at
   # Use for audit trails
   ```

## Troubleshooting

### Permission Denied Errors
- Check user has correct role assigned
- Verify role has required permissions
- Check scope matches (global vs cluster vs service)
- Invalidate permission cache if stale

### Cache Issues
```python
# Manually invalidate cache
RBACModel.invalidate_permission_cache(db, user_id)
```

### Debug Permission Checks
```python
# Get user permissions
perms = RBACModel.get_user_permissions(db, user_id)
print(f"User permissions: {perms}")

# Check specific permission
has_perm = RBACModel.has_permission(db, user_id, Permissions.CLUSTER_WRITE)
print(f"Has permission: {has_perm}")
```

## Performance Considerations

- **Permission caching**: Permissions cached in database table
- **Lazy loading**: Permissions loaded only when needed
- **Efficient queries**: Single query for all user permissions
- **Cache invalidation**: Automatic on role changes

## Security Notes

- All permission checks require authentication first
- 401 returned for unauthenticated requests
- 403 returned for insufficient permissions
- Audit log tracks who granted permissions
- System roles cannot be modified/deleted
