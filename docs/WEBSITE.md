# MarchProxy Website

The MarchProxy website is maintained in a separate repository as a Git submodule.

## Setting up the website submodule

The website submodule is configured with sparse-checkout to only include the `marchproxy/` and `marchproxy-docs/` directories.

```bash
# Clone the main repository with submodules
git clone --recurse-submodules https://github.com/penguintechinc/marchproxy.git

# Or if you already have the main repo, initialize the submodule
git submodule init
git submodule update

# Or add the submodule for the first time with sparse-checkout
git submodule add https://github.com/penguintechinc/website.git website
cd website
git config core.sparseCheckout true
echo "marchproxy/" > .git/info/sparse-checkout
echo "marchproxy-docs/" >> .git/info/sparse-checkout
git read-tree -m -u HEAD
cd ..
```

### Manual sparse-checkout setup (if needed)

If the submodule already exists but you want to enable sparse-checkout:

```bash
cd website
git config core.sparseCheckout true
echo "marchproxy/" > .git/info/sparse-checkout
echo "marchproxy-docs/" >> .git/info/sparse-checkout
git read-tree -m -u HEAD
cd ..
```

## Working with the website submodule

```bash
# Navigate to the website directory
cd website

# Make changes to the website
# ... edit files ...

# Commit changes to the website repository
git add .
git commit -m "Update website with new features"
git push origin main

# Go back to main repository and update submodule reference
cd ..
git add website
git commit -m "Update website submodule"
git push origin main
```

## Updating the website submodule

```bash
# Pull latest changes from the website repository
cd website
git pull origin main

# Update the main repository to point to the latest website commit
cd ..
git add website
git commit -m "Update website submodule to latest"
git push origin main
```

## Website Features to Update

With the new dual proxy architecture (ingress + egress), the website should highlight:

### Key Features to Showcase
- **Dual Proxy Architecture**: Both ingress (reverse proxy) and egress (forward proxy)
- **mTLS Authentication**: Mutual TLS with ECC P-384 cryptography
- **Certificate Management**: Automated CA generation and certificate lifecycle
- **Performance**: eBPF acceleration on both proxy types
- **Enterprise Security**: Client certificate validation and strong cryptography

### Pages to Update
- **Homepage**: Update hero section with dual proxy messaging
- **Features**: Add ingress proxy capabilities and mTLS features
- **Architecture**: Update diagrams to show both proxy types
- **Pricing**: Clarify 3 total proxy limit for Community edition
- **Documentation**: Link to new testing and deployment guides

### Technical Content
- **API Documentation**: Include mTLS certificate management endpoints
- **Installation Guide**: Update with new docker-compose configuration
- **Security Guide**: Detail mTLS setup and certificate management
- **Performance Benchmarks**: Include ingress proxy performance data

## Repository Structure (Sparse Checkout)

With sparse-checkout enabled, only these directories are included:

```
website/
├── marchproxy/               # Main MarchProxy website
│   ├── src/
│   │   ├── pages/
│   │   │   ├── index.js      # Homepage - update with dual proxy
│   │   │   ├── features.js   # Features - add ingress & mTLS
│   │   │   ├── architecture.js # Architecture - dual proxy diagram
│   │   │   └── pricing.js    # Pricing - clarify proxy limits
│   │   ├── components/
│   │   │   ├── ProxyDiagram.js # Update for dual architecture
│   │   │   └── FeatureGrid.js  # Add mTLS features
│   │   └── styles/
│   ├── public/
│   │   ├── images/
│   │   │   └── architecture/ # Update diagrams
│   │   └── docs/
│   └── package.json
└── marchproxy-docs/          # Documentation site
    ├── docs/
    │   ├── installation/
    │   ├── configuration/
    │   ├── security/         # New: mTLS guide
    │   └── api/              # API documentation
    ├── sidebars.js
    └── docusaurus.config.js
```

## Content Updates Needed

### Homepage Hero Section
```
"High-Performance Dual Proxy Suite"
"Complete ingress and egress traffic management with mTLS authentication"
```

### Feature Highlights
- ✅ Ingress (Reverse) Proxy v1.0.0
- ✅ Egress (Forward) Proxy with eBPF
- ✅ mTLS Mutual Authentication
- ✅ Automated Certificate Management
- ✅ ECC P-384 Cryptography
- ✅ Load Balancing & Health Checks

### Architecture Diagram Updates
- Show both ingress and egress proxies
- Highlight mTLS certificate flow
- Display certificate management in manager
- Show client certificate validation

For detailed implementation, work in the website repository directly and then update this submodule reference.