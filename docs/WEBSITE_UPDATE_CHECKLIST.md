# Website Update Checklist for Dual Proxy Architecture

## 🎯 Key Changes to Communicate

### Major Features Added
- ✅ **Ingress Proxy v1.0.0**: Complete reverse proxy with load balancing
- ✅ **Dual Proxy Architecture**: Both ingress and egress in single deployment
- ✅ **mTLS Authentication**: Mutual TLS with ECC P-384 cryptography
- ✅ **Certificate Management**: Automated CA generation and lifecycle management
- ✅ **Enhanced Security**: Client certificate validation and strong cipher suites

### License Clarification
- **Community Edition**: 3 total proxies (any combination of ingress/egress)
- **Enterprise Edition**: Unlimited proxies with multi-cluster support

## 📄 Pages That Need Updates

### 1. Homepage (`/`)
**Hero Section Updates:**
```
OLD: "High-Performance Egress Proxy"
NEW: "High-Performance Dual Proxy Suite"

OLD: "Manage egress traffic with eBPF acceleration"
NEW: "Complete ingress and egress traffic management with mTLS authentication"
```

**Feature Highlights:**
- Add "Ingress (Reverse) Proxy v1.0.0"
- Add "mTLS Mutual Authentication"
- Add "Automated Certificate Management"
- Update "eBPF Acceleration" to mention both proxies

### 2. Features Page (`/features`)
**New Sections to Add:**
- **Dual Proxy Architecture**
  - Ingress reverse proxy with host/path routing
  - Egress forward proxy with service mapping
  - Unified management and monitoring

- **mTLS Security**
  - Mutual TLS authentication
  - ECC P-384 cryptography
  - Automated certificate generation
  - Client certificate validation

- **Load Balancing** (Ingress)
  - Round-robin, least-connections algorithms
  - Backend health checking
  - SSL/TLS termination

### 3. Architecture Page (`/architecture`)
**Diagram Updates:**
- Replace single proxy diagram with dual proxy architecture
- Show ingress proxy handling external clients
- Show egress proxy handling outbound traffic
- Highlight mTLS certificate flow
- Show certificate management in manager

**Component Descriptions:**
- Update to describe both proxy types
- Add mTLS certificate authority section
- Explain dual proxy coordination

### 4. Pricing Page (`/pricing`)
**Community Edition Updates:**
```
OLD: "Up to 3 proxy instances"
NEW: "Up to 3 total proxy instances (any combination of ingress/egress)"

Examples:
- 1 ingress + 2 egress
- 2 ingress + 1 egress
- 3 egress only
- 3 ingress only
```

**Feature Matrix Updates:**
- Add mTLS authentication row
- Add certificate management row
- Add dual proxy architecture row

### 5. Documentation (`/docs`)
**New Sections:**
- **mTLS Setup Guide**
  - Certificate generation
  - Client certificate configuration
  - Troubleshooting mTLS issues

- **Ingress Proxy Configuration**
  - Reverse proxy setup
  - Load balancer configuration
  - SSL/TLS termination

- **Dual Proxy Deployment**
  - Docker Compose setup
  - Kubernetes deployment
  - High availability configuration

### 6. API Documentation (`/docs/api`)
**New Endpoints:**
- Certificate management APIs
- Ingress route configuration
- mTLS validation endpoints

## 🎨 Visual Assets to Update

### Architecture Diagrams
1. **Main Architecture Diagram**
   - Show both ingress and egress proxies
   - Include mTLS certificate flows
   - Display certificate management

2. **Traffic Flow Diagrams**
   - Ingress: External clients → Ingress proxy → Backend services
   - Egress: Internal services → Egress proxy → External destinations

3. **Security Diagram**
   - mTLS handshake flow
   - Certificate authority hierarchy
   - Client certificate validation

### Screenshots
- Update dashboard screenshots with dual proxy metrics
- Show certificate management interface
- Display both proxy monitoring views

## 📝 Content Updates

### Homepage Copy
```markdown
# High-Performance Dual Proxy Suite

Complete ingress and egress traffic management for enterprise data centers with advanced eBPF acceleration, mTLS authentication, and hardware optimization.

## Key Benefits
- **Complete Traffic Control**: Both inbound and outbound traffic management
- **Enterprise Security**: mTLS authentication with automated certificate management
- **Unmatched Performance**: eBPF acceleration on both proxy types
- **Production Ready**: Comprehensive monitoring and enterprise features
```

### Feature Descriptions

**Ingress Proxy (v1.0.0)**
- Reverse proxy with host/path-based routing
- Load balancing with health checking
- SSL/TLS termination with mTLS support
- DDoS protection and rate limiting

**mTLS Authentication**
- Mutual TLS with client certificate validation
- ECC P-384 cryptography with SHA-384 hashing
- Automated certificate authority management
- Strong cipher suite enforcement

**Unified Management**
- Single management interface for both proxies
- Centralized certificate management
- Coordinated monitoring and alerting
- Hot-reload configuration updates

## 🚀 Performance Metrics to Highlight

### Benchmarks
- **Combined Throughput**: 100+ Gbps with hardware acceleration
- **Ingress Performance**: Load balancing across multiple backends
- **Egress Performance**: High-speed service-to-service communication
- **mTLS Overhead**: Minimal performance impact with ECC cryptography

### Use Cases
- **Ingress**: API gateways, web application load balancing, SSL termination
- **Egress**: Service mesh communication, internet access control, data center egress

## 📋 Technical Specifications

### Supported Protocols
- **Ingress**: HTTP/HTTPS, WebSocket, TLS termination
- **Egress**: TCP, UDP, ICMP, HTTP/HTTPS, WebSocket

### Security Features
- **mTLS**: Mutual authentication with client certificates
- **Encryption**: ECC P-384, RSA 4096+, strong cipher suites
- **Validation**: Real-time certificate validation and revocation

### Deployment Options
- **Single Node**: Both proxies on same server
- **Distributed**: Separate ingress and egress servers
- **High Availability**: Multiple instances with load balancing

## 📞 Call-to-Action Updates

### Homepage CTA
```
OLD: "Deploy MarchProxy"
NEW: "Deploy Dual Proxy Suite"

OLD: "Get started with egress management"
NEW: "Complete traffic management solution"
```

### Download/Install
- Update installation commands for dual proxy
- Include certificate generation in quick start
- Highlight mTLS configuration steps

## 🔗 Navigation Updates

### Main Menu
- "Features" → Highlight dual proxy
- "Architecture" → Update with dual proxy diagram
- "Documentation" → Add mTLS and ingress guides
- "Pricing" → Clarify proxy limits

### Footer Links
- Add "mTLS Guide"
- Add "Ingress Configuration"
- Add "Certificate Management"

## ✅ Validation Checklist

Before publishing:
- [ ] All proxy limits clearly explained (3 total for Community)
- [ ] Dual proxy architecture prominently featured
- [ ] mTLS benefits and setup clearly documented
- [ ] Performance metrics updated for both proxies
- [ ] Screenshots show latest dual proxy interface
- [ ] API documentation includes certificate management
- [ ] Installation guides updated for new architecture
- [ ] Security documentation covers mTLS thoroughly

## 📈 SEO Keywords to Target

Primary:
- "dual proxy architecture"
- "mTLS authentication"
- "ingress egress proxy"
- "enterprise proxy suite"

Secondary:
- "reverse proxy load balancer"
- "mutual TLS authentication"
- "eBPF proxy acceleration"
- "certificate management"