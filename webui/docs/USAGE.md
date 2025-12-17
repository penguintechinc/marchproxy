# User Guide - MarchProxy WebUI

## Getting Started

### Accessing the UI

**Default URL**: `http://localhost:3000`

**Production URL**: Set by your deployment environment

### First Login

1. Navigate to the login page
2. Enter credentials provided by your administrator
3. If 2FA enabled, enter TOTP code from authenticator app
4. Click "Login"
5. You'll be redirected to the dashboard

## Dashboard

The dashboard is your central hub for monitoring the MarchProxy system.

### Dashboard Overview

**Summary Statistics**
- **Total Clusters**: Number of clusters you have access to
- **Total Services**: Count of service definitions across all clusters
- **Total Proxies**: Number of active proxy instances
- **Active Connections**: Real-time connection count

**Recent Activity**
- Latest configuration changes
- Proxy status changes
- Service updates
- User actions (admin users only)

**Status Indicators**
- Green: Healthy/online
- Yellow: Degraded/warning
- Red: Offline/critical

**Charts**
- Cluster utilization over time
- Proxy CPU/memory usage trends
- Connection count trends
- Service request rates

### Dashboard Actions

**Refresh Data**
- Charts auto-update every 5 seconds
- Click refresh icon for immediate update
- Real-time updates available in top-right corner

**Export Dashboard Data**
- Click "Export" button
- Choose format: CSV, JSON, or PDF
- Opens new browser tab with data

## Services Management

### Understanding Services

A **Service** defines:
- What traffic to proxy (port/protocol)
- Where to send traffic (upstream server)
- Authentication requirements
- Status and health

### Viewing Services

1. Click **Services** in sidebar
2. Table shows all services with:
   - Name and description
   - Status (active/inactive)
   - Port configuration
   - Protocol (TCP/UDP/ICMP)
   - Upstream server
   - Creation date

### Creating a Service

1. Click **Create Service** button
2. Fill form fields:
   - **Name**: Display name (required)
   - **Description**: Service purpose (optional)
   - **Port**: Single port (8080), range (8000-8100), or list (8080,8081,8082)
   - **Protocol**: Select TCP, UDP, or ICMP
   - **Upstream URL**: Where to forward traffic
   - **Cluster**: Which cluster owns this service
   - **Authentication**: Enable for service auth tokens

3. Click **Create**
4. Success notification appears
5. Redirected to service details page

### Editing a Service

1. Click service name or edit icon
2. Modify fields as needed
3. Click **Update**
4. Changes applied immediately

### Managing Service Authentication

**Rotate Token**
1. Open service details
2. Click **Rotate Token** button
3. Confirm action
4. New token generated
5. Old token invalidated immediately
6. Applications must update their configuration

**View Token**
1. Open service details
2. Token visible in "Authentication" section (masked by default)
3. Click eye icon to reveal full token
4. Click copy icon to copy to clipboard

### Deleting a Service

1. Click service row
2. Click **Delete** button
3. Confirmation dialog appears
4. Click **Confirm Delete**
5. Service removed (waiting for UI implementation)

## Clusters Management

### Understanding Clusters

A **Cluster** is a group of proxy servers that work together. Features:
- Isolated configuration per cluster
- Separate API keys for each cluster
- Services scoped to specific clusters
- Community tier: 1 default cluster only
- Enterprise tier: Unlimited clusters

### Viewing Clusters

1. Click **Clusters** in sidebar
2. Table shows:
   - Cluster name
   - Status
   - Proxy count
   - Service count
   - License tier
   - Created date

### Creating a Cluster (Enterprise)

1. Click **Create Cluster** button
2. Fill form:
   - **Name**: Unique cluster identifier
   - **Description**: Cluster purpose
   - **License Tier**: Enterprise only
3. Click **Create**
4. New cluster appears in table
5. Automatic API key generated

### Managing Cluster API Key

**View Current Key**
1. Open cluster details
2. API key visible in "Configuration" section
3. Masked by default for security
4. Click eye icon to reveal

**Rotate API Key**
1. Open cluster details
2. Click **Rotate API Key** button
3. Confirm rotation
4. New key generated immediately
5. Old key becomes invalid
6. Proxies must update config to use new key

**Important**: All proxies using old key will disconnect after rotation

### Cluster Status

- **Active**: Healthy, ready for proxies
- **Inactive**: Disabled, proxies cannot register
- **Degraded**: Some issues detected, reduced functionality

## Proxies Monitoring

### Understanding Proxies

**Proxy** is a running instance that:
- Registers with a cluster
- Forwards traffic based on services
- Sends heartbeat and metrics
- Displays status and health

### Proxy Fleet Status

1. Click **Proxies** in sidebar
2. Overview shows:
   - Total proxies
   - Online/offline count
   - Average metrics across fleet

### Proxy Details

Click proxy row to see:

**Basic Info**
- Hostname/IP address
- Version
- Cluster assignment
- Registration time

**Status**
- Current status: Online/Offline/Degraded
- Last heartbeat time
- Uptime percentage

**Metrics** (Real-time, updated every 5 seconds)
- CPU Usage: % of available
- Memory Usage: % of available
- Connections: Active concurrent connections
- Throughput: Bytes in/out per second
- Error Rate: % of failed requests

**Services**
- Services proxied by this instance
- Connections per service
- Last request time

### Proxy Status Colors

- **Green**: Online and healthy
- **Yellow**: Degraded (high resource usage or errors)
- **Red**: Offline or critical issue
- **Gray**: Unknown state

### Deregistering a Proxy

1. Click proxy row
2. Click **Deregister** button
3. Confirm action
4. Proxy removed from cluster
5. Traffic immediately fails over to other proxies

## Certificates Management

### Understanding Certificates

Certificates enable:
- TLS/HTTPS for services
- Mutual TLS (mTLS) between services
- Client certificate validation
- Secure communication encryption

### Viewing Certificates

1. Click **Certificates** in sidebar
2. Table shows:
   - Certificate name
   - Subject and issuer
   - Expiration date
   - Status: Valid/Expired/Expiring Soon
   - Source: Upload/Infisical/Vault
   - Auto-renew status

### Certificate Expiration Alerts

**Expiring Soon** (< 30 days)
- Highlighted in yellow
- Email alert sent to admins
- "Renew" button enabled

**Expired**
- Highlighted in red
- Services using this cert may fail
- Must renew immediately

### Uploading a Certificate

1. Click **Upload Certificate** button
2. Choose source:
   - **Direct Upload**: PEM format file
   - **Infisical**: Select secret path
   - **HashiCorp Vault**: Specify vault path
3. Fill details:
   - **Name**: Display name
   - **Auto-renew**: Enable automatic renewal (if supported)
4. Click **Upload**
5. Certificate appears in table

### Renewing a Certificate

**Manual Renewal**
1. Click certificate row
2. Click **Renew** button
3. System renews certificate
4. New cert replaces old one
5. Services automatically use new cert

**Batch Renewal**
1. Select multiple expiring certificates
2. Click **Batch Renew**
3. All selected certs renewed together
4. Progress shown in dialog

### Certificate Sources

**Direct Upload**
- Upload PEM format files
- Manual renewal required
- No integration needed

**Infisical**
- Automatic retrieval from Infisical
- Requires API credentials
- Auto-renewal via Infisical schedule

**HashiCorp Vault**
- PKI engine integration
- Dynamic certificate generation
- Auto-renewal supported

## Modules & Routing

### Understanding Modules

**Module** is a custom routing rule set that:
- Defines request routes and rules
- Supports conditional routing
- Enables traffic manipulation
- Provides plugins and middleware

### Module Manager

1. Click **Modules** > **Manager** in sidebar
2. View all modules:
   - Name and version
   - Type: Routing, Plugin, Middleware
   - Status: Enabled/Disabled
   - Health: Healthy/Degraded/Unhealthy
   - Routes count

**Module Actions**
- Click module to view details
- Enable/disable module
- View routes
- Check health

### Route Editor

1. Click **Modules** > **Routes** in sidebar
2. Visual route editor shows:
   - Route paths
   - Request methods
   - Upstream targets
   - Enabled/disabled status

**Editing Routes**
1. Click route to edit
2. Change path, method, or target
3. Save changes
4. Revalidates configuration

**Creating Routes**
1. Click **New Route** button
2. Enter path (e.g., `/api/users`)
3. Select method: GET, POST, PATCH, DELETE, etc.
4. Enter upstream server
5. Click **Create**

### Module Health

Indicators show:
- **Healthy**: All routes responding
- **Degraded**: Some routes slow or errors
- **Unhealthy**: Most routes failing
- **Unknown**: No recent data

## Deployments & Traffic Control

### Blue-Green Deployments

**What is Blue-Green?**
- Two identical environments (blue/green)
- Traffic switches between them
- Zero-downtime deployments
- Quick rollback capability

**Current Active Environment**
- Shown on deployment page
- Blue or Green (opposite is standby)

**Deployment Process**

1. Click **Deployments** > **Blue-Green** in sidebar
2. View current state:
   - Active environment (blue/green)
   - Traffic weight (100% active, 0% standby)
   - Health of each environment

3. Deploy new version to standby environment
4. Run tests against standby
5. Gradually shift traffic:
   - Drag slider to 10/90
   - Monitor metrics
   - Shift to 50/50
   - Finally 0/100 for full switch

6. If issues detected, reverse slider
7. After 24 hours, old environment can be destroyed

### Traffic Shifting

**Manual Slider**
- Drag to adjust traffic percentage
- Changes applied immediately
- No downtime during shift
- Metrics show both versions

**Metrics During Deployment**
- Track error rates for each version
- Monitor latency
- CPU/memory usage
- Connection counts

**Automatic Rollback**
- If error rate exceeds threshold, auto-rollback
- Requires configuration
- Protects against bad deployments

## Auto-Scaling (Enterprise)

### Scaling Policies

1. Click **Scaling** > **Auto** in sidebar
2. View active policies:
   - Target service/module
   - Min/max replicas
   - Scale-up trigger (e.g., CPU > 70%)
   - Scale-down trigger (e.g., CPU < 30%)

### Creating Scaling Policy

1. Click **New Policy** button
2. Select target service
3. Configure:
   - Minimum replicas: At least this many
   - Maximum replicas: Never exceed this
   - Scale-up metric: (CPU, memory, requests/sec)
   - Scale-up threshold: (e.g., 80%)
   - Scale-down metric: Same options
   - Scale-down threshold: (e.g., 30%)
   - Cooldown period: Seconds between scaling events

4. Click **Create**

### Scaling History

View when service scaled:
- Date/time of scaling event
- Number of replicas before/after
- Trigger metric and value
- Duration of scale operation

## Observability

### Metrics Dashboard

1. Click **Observability** > **Metrics** in sidebar
2. Real-time charts showing:
   - CPU usage (all proxies)
   - Memory usage (all proxies)
   - Request rate (requests/sec)
   - Error rate (% of requests)
   - Latency (p50, p95, p99)
   - Connection count

**Time Range Selection**
- Last hour (5-second intervals)
- Last 24 hours (1-minute intervals)
- Last 7 days (1-hour intervals)
- Last 30 days (1-day intervals)

### Distributed Tracing

1. Click **Observability** > **Tracing** in sidebar
2. View spans from recent requests
3. Search by:
   - Trace ID
   - Service name
   - Operation name
   - Duration
   - Status (success/error)

**Trace Details**
- Request path through services
- Duration at each hop
- Error messages if failed
- Log entries for debugging

### Alerts

1. Click **Observability** > **Alerts** in sidebar
2. View configured alerts:
   - Alert name
   - Condition (e.g., CPU > 80%)
   - Status: Active/Resolved
   - Last triggered

**Creating Alert**
1. Click **New Alert**
2. Name alert
3. Select metric (CPU, memory, error rate, etc.)
4. Set threshold
5. Set notification (email, webhook)
6. Click **Create**

**Alert Notifications**
- Email when triggered
- Webhook for integration
- Resolved notification when metric normalizes

## Security & Compliance

### mTLS Configuration

**What is mTLS?**
- Mutual TLS authentication
- Both client and server authenticate
- Certificate-based verification
- Enhanced security for service-to-service

1. Click **Security** > **mTLS** in sidebar
2. Configure:
   - Enable/disable mTLS globally
   - CA certificate
   - Client certificates
   - Certificate validation mode

3. Services using mTLS show lock icon

### OPA Policy Editor

**What are OPA Policies?**
- Open Policy Agent rules
- Define authorization logic
- Flexible policy language (Rego)
- Enterprise feature

1. Click **Security** > **Policy Editor** in sidebar
2. Write Rego policies
3. Test policies with sample input
4. Deploy to proxies
5. Proxies enforce policies

### Audit Logs

1. Click **Security** > **Audit Logs** in sidebar
2. View all system actions:
   - Who performed action
   - What action (create/update/delete)
   - When it occurred
   - Which resource
   - Source IP/user agent

**Filtering**
- By user
- By action type
- By resource type
- By date range

**Exporting**
- Download as CSV
- Download as JSON
- Verify audit chain (compliance)

### Compliance Reports

1. Click **Security** > **Compliance** in sidebar
2. Select compliance standard:
   - SOC2
   - HIPAA
   - PCI-DSS
   - ISO 27001

3. Report shows:
   - Compliance status
   - Requirements met/failing
   - Audit trail
   - Recommendations

4. Click **Export** to download report

## Settings & Administration

### Account Settings

1. Click your profile icon (top-right)
2. Select **Settings**
3. Update:
   - Display name
   - Email address
   - Timezone
   - Notification preferences
   - Theme (light/dark)

### Password Management

1. Click profile icon > **Settings**
2. Click **Change Password**
3. Enter current password
4. Enter new password (must be strong)
5. Confirm new password
6. Click **Update**

### Two-Factor Authentication (2FA)

**Enable 2FA**
1. Click profile icon > **Settings**
2. Click **Enable 2FA**
3. Scan QR code with authenticator app
4. Enter 6-digit code from app
5. Save backup codes securely
6. 2FA enabled - required at next login

**Disable 2FA**
1. Click profile icon > **Settings**
2. Click **Disable 2FA**
3. Enter password to confirm
4. 2FA disabled

### Session Management

1. Click profile icon > **Settings**
2. Scroll to "Active Sessions"
3. View all logged-in devices/browsers
4. Click "Logout" on any session to end it
5. "Logout All" ends all sessions immediately

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `?` | Show help menu |
| `d` | Go to Dashboard |
| `s` | Go to Services |
| `c` | Go to Clusters |
| `p` | Go to Proxies |
| `/` | Focus search box |
| `Ctrl+K` | Command palette (if enabled) |

## Mobile Experience

- Responsive design works on tablets/phones
- Touch-friendly interface
- Hamburger menu for narrow screens
- Tables scroll horizontally on mobile
- Forms optimized for mobile keyboards

## Dark Mode

Toggle dark/light theme:
1. Click theme icon (top-right)
2. Select Light/Dark/Auto
3. Auto follows system preference

## Exporting Data

**Export Options**
- CSV: Spreadsheet format
- JSON: Machine-readable format
- PDF: Printable reports

**How to Export**
1. Navigate to any table (services, proxies, etc.)
2. Click "Export" button
3. Choose format
4. File downloads automatically

## Help & Support

**In-App Help**
- Click "?" icon
- View contextual help
- See keyboard shortcuts
- Links to documentation

**External Resources**
- Full documentation: `/webui/docs/`
- API reference: `/webui/docs/API.md`
- Support portal: support.penguintech.io
- Email: support@penguintech.io

## Troubleshooting

### Can't Login
- Verify username/password correct
- Check Caps Lock
- Ensure 2FA code is correct if enabled
- Try clearing browser cookies

### Metrics Not Updating
- Check API connection
- Refresh page
- Check proxy heartbeat status
- Verify metrics endpoint responding

### Services Not Appearing
- Verify cluster selected
- Check service status (active/inactive)
- Verify you have access to cluster (role-based)
- Refresh page

### Performance Issues
- Clear browser cache
- Disable browser extensions
- Check internet connection
- Try different browser
- Check server logs

## Best Practices

1. **Regular Backups**: Export configuration regularly
2. **API Key Security**: Rotate keys periodically
3. **Certificate Management**: Check expiration dates proactively
4. **Monitoring**: Check dashboard daily
5. **Access Control**: Use least-privilege principles
6. **Audit Logs**: Review regularly for security
7. **Updates**: Apply updates promptly
8. **Documentation**: Document custom configurations
