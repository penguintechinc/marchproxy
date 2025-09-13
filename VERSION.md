# MarchProxy Versioning System

## Current Version
See `.version` file for the current version number.

## Version Format: `vMajor.Minor.Patch.Build`

### Components:

#### Major (Breaking Changes)
Increment when:
- Removing or changing existing API endpoints
- Database schema changes that require migration
- Breaking configuration file format changes
- Removing features or functionality
- Incompatible protocol changes
- License model changes (Community → Enterprise features)

#### Minor (New Features)
Increment when:
- Adding new API endpoints
- Adding new features or capabilities
- Adding new protocol support
- Adding new authentication methods
- Adding new clustering capabilities
- Adding backward-compatible database fields
- Significant performance improvements

#### Patch (Bug Fixes & Minor Updates)
Increment when:
- Bug fixes
- Security patches
- Performance optimizations
- Minor UI improvements
- Documentation updates
- Configuration additions (backward compatible)
- Minor feature enhancements

#### Build (Epoch Timestamp)
Automatically set to current Unix timestamp (epoch64) when:
- Any code change between GitHub releases
- Development progress tracking
- Testing and debugging iterations
- CI/CD pipeline runs
- Pre-release candidates
- Provides chronological ordering and uniqueness

## Version History

### v0.1.0.1757705800 (Current)
- Initial project structure created
- Foundation setup completed  
- Database schema implemented
- Docker environment configured
- Basic proxy and manager structure
- Epoch timestamp build system implemented
- Development phase - not production ready
- Build: 1757705800 = Thu Sep 12 2025 14:36:40 GMT

## Release Guidelines

### When to Update Version:

1. **During Development:**
   - Build number automatically set to current epoch timestamp
   - Use `./scripts/update-version.sh` for version updates
   - Provides automatic chronological ordering

2. **For Pull Requests:**
   - Update patch/minor/major as appropriate using the script
   - Build timestamp shows when version was last updated

3. **For Releases:**
   - Tag release with version (without build timestamp)
   - Example: `v1.0.0` (not `v1.0.0.1757705800`)

### Version Update Examples:

- `v0.1.0.1757705800` → `v0.1.0.1757706000`: Code changes, automatic timestamp
- `./scripts/update-version.sh patch`: v0.1.0.EPOCH → v0.1.1.EPOCH (bug fixes)
- `./scripts/update-version.sh minor`: v0.1.0.EPOCH → v0.2.0.EPOCH (new features)
- `./scripts/update-version.sh major`: v0.1.0.EPOCH → v1.0.0.EPOCH (breaking changes)
- `./scripts/update-version.sh 1 0 0`: Set to v1.0.0.EPOCH (first production release)

## Integration Points

### Go Application
Version is embedded and updated by script:
```go
var Version = "v0.1.0.1757705800" // Updated by update-version.sh
```

### Python Application
Version loaded dynamically from .version file:
```python
with open('.version', 'r') as f:
    VERSION = f.read().strip()  # e.g., "v0.1.0.1757705800"
```

### Version Update Script
Automatic version management:
```bash
./scripts/update-version.sh          # Update build timestamp only
./scripts/update-version.sh patch    # Increment patch version
./scripts/update-version.sh minor    # Increment minor version
./scripts/update-version.sh major    # Increment major version
./scripts/update-version.sh 1 2 3    # Set specific version
```

### Docker Images
Tagged with version (release versions omit epoch):
```bash
# Development builds (with epoch)
docker build -t marchproxy/manager:v0.1.0.1757705800 .

# Release builds (without epoch)  
docker build -t marchproxy/manager:v0.1.0 .
docker build -t marchproxy/proxy:v0.1.0 .
```

### API Responses
Version included in:
- `/healthz` endpoint responses
- `/api/version` endpoint
- Prometheus metrics labels

## Automation

### Pre-commit Hook (Future)
```bash
#!/bin/bash
# Increment build number automatically
current=$(cat .version)
# Logic to increment build number
echo $new_version > .version
```

### CI/CD Integration
- GitHub Actions read `.version` file
- Docker images tagged with version
- Release notes generated from version changes

## Version Compatibility Matrix

| Manager Version | Proxy Version | Compatible |
|----------------|---------------|------------|
| v0.1.x         | v0.1.x        | ✓          |
| v0.2.x         | v0.1.x        | ✗          |
| v1.0.x         | v1.0.x        | ✓          |

## Special Considerations

### Community vs Enterprise
- Community features: vX.X.X.X
- Enterprise features: vX.X.X.X-enterprise
- Both use same version numbering
- Feature flags determine availability

### Development vs Production
- Development: Include build number
- Production: Omit build number in tags
- Alpha/Beta: vX.X.X-alpha.BUILD
- Release Candidate: vX.X.X-rc.BUILD

## Version Update Checklist

- [ ] Update `.version` file
- [ ] Update VERSION.md history
- [ ] Update embedded versions in code
- [ ] Update Docker image tags
- [ ] Update documentation references
- [ ] Tag git commit (for releases)
- [ ] Update CHANGELOG.md (for releases)