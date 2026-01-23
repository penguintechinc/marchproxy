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

## Usage Guide

### 1. Protecting Routes with Decorators

#### Require Specific Permission

```python
from middleware.rbac import requires_permission
from models.rbac import Permissions

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
    """Checks cluster-specific write permission"""
    return jsonify({'message': f'Cluster {cluster_id} updated'})
```

#### Require Specific Role

```python
from middleware.rbac import requires_role
from models.rbac import PermissionScope

@app.route('/api/v1/admin/dashboard')
@requires_role('admin', scope=PermissionScope.GLOBAL)
async def admin_dashboard():
    """Only users with Admin role can access"""
    return jsonify({'message': 'Admin dashboard'})
```

#### Require ANY of Multiple Permissions

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

#### Require ALL Permissions

```python
from middleware.rbac import requires_all_permissions

@app.route('/api/v1/critical-operation')
@requires_all_permissions(
    Permissions.GLOBAL_ADMIN,
    Permissions.GLOBAL_SETTINGS
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

```python
from middleware.rbac import (
    is_admin,
    can_manage_users,
    can_access_cluster,
    can_access_service
)

# Check if user is admin
if is_admin(user_id, db):
    # User has global admin rights
    pass

# Check if user can manage other users
if can_manage_users(user_id, db):
    # User can create/update/delete users
    pass

# Check cluster access
if can_access_cluster(user_id, cluster_id, db):
    # User can access this cluster
    pass

# Check service access
if can_access_service(user_id, service_id, db):
    # User can access this service
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
