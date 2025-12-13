# MarchProxy gRPC Proto Definitions - File Index

Quick navigation and file reference for the MarchProxy gRPC proto implementation.

## Quick Start

**New to the project?** Start here:
1. Read [QUICKSTART.md](QUICKSTART.md) - 5-minute quick reference
2. Review [ARCHITECTURE_DIAGRAM.md](ARCHITECTURE_DIAGRAM.md) - Visual architecture
3. Generate code: `../scripts/gen-proto.sh`

**Need comprehensive docs?**
- [README.md](README.md) - Complete documentation with examples

**Implementation details?**
- [PHASE1_PROTO_IMPLEMENTATION.md](PHASE1_PROTO_IMPLEMENTATION.md) - Full implementation summary

## File Organization

### Proto Definitions (980 lines)

| File | Lines | Purpose |
|------|-------|---------|
| [marchproxy/types.proto](marchproxy/types.proto) | 232 | Common types, enums, and shared messages |
| [marchproxy/module.proto](marchproxy/module.proto) | 363 | ModuleService interface (25 RPCs) |
| [marchproxy/nlb.proto](marchproxy/nlb.proto) | 385 | NLBService interface (21 RPCs) |

### Documentation (1,599 lines)

| File | Lines | Purpose |
|------|-------|---------|
| [README.md](README.md) | 371 | Comprehensive documentation and examples |
| [QUICKSTART.md](QUICKSTART.md) | 341 | Quick reference guide |
| [ARCHITECTURE_DIAGRAM.md](ARCHITECTURE_DIAGRAM.md) | 463 | Visual architecture diagrams |
| [PHASE1_PROTO_IMPLEMENTATION.md](PHASE1_PROTO_IMPLEMENTATION.md) | 424 | Implementation summary |

### Scripts (262 lines)

| File | Lines | Purpose |
|------|-------|---------|
| [../scripts/gen-proto.sh](../scripts/gen-proto.sh) | 151 | Generate Go and Python code |
| [../scripts/validate-proto.sh](../scripts/validate-proto.sh) | 111 | Validate proto files |

### Configuration

| File | Purpose |
|------|---------|
| [.gitignore](.gitignore) | Exclude generated files from git |

## Total Statistics

- **Total Lines**: 2,841
- **Proto Files**: 3
- **Documentation Files**: 4
- **Scripts**: 2
- **Services**: 2 (ModuleService, NLBService)
- **RPC Methods**: 46 total
- **Message Types**: ~145
- **Enumerations**: 4

## Proto Files Detailed

### types.proto (232 lines)

**Enumerations**:
- `StatusCode` - Operation status codes (9 values)
- `ModuleType` - Module types: ALB, DBLB, AILB, RTMP, NLB (6 values)
- `Protocol` - Protocols: TCP, UDP, HTTP, HTTPS, MySQL, RTMP, etc. (17 values)
- `HealthStatus` - Health states (7 values)

**Core Message Types**:
- `ModuleInstance` - Module instance information
- `Route` - Traffic routing configuration
- `RateLimit` - Rate limiting configuration
- `MetricDataPoint` - Individual metric
- `ModuleMetrics` - Metrics collection
- `ModuleStats` - Comprehensive statistics

**Configuration Types**:
- `ScalingConfig` - Auto-scaling configuration
- `BlueGreenConfig` - Blue/green deployment
- `HealthCheckConfig` - Health check settings
- `ConfigUpdate` - Configuration updates

**Utility Types**:
- `Error`, `Response`, `Pagination`, `Filter`, `Query`

### module.proto (363 lines)

**Service**: `ModuleService` - 25 RPCs in 7 categories

**Categories**:
1. Lifecycle (3): GetStatus, Reload, Shutdown
2. Traffic Routing (4): CanHandle, GetRoutes, UpdateRoutes, DeleteRoute
3. Rate Limiting (3): GetRateLimits, SetRateLimit, RemoveRateLimit
4. Scaling (3): GetMetrics, Scale, GetInstances
5. Blue/Green (4): SetTrafficWeight, GetActiveVersion, Rollback, PromoteVersion
6. Health (3): HealthCheck, GetStats, StreamMetrics
7. Configuration (2): GetConfig, UpdateConfig

**Message Types**: ~50 request/response pairs

### nlb.proto (385 lines)

**Service**: `NLBService` - 21 RPCs in 6 categories

**Categories**:
1. Registration (5): RegisterModule, UnregisterModule, Heartbeat, ListModules, GetModuleInfo
2. Routing (4): UpdateRouting, GetRoutingTable, RouteRequest, ValidateRoute
3. Metrics (4): ReportMetrics, GetNLBMetrics, GetModuleMetrics, StreamNLBMetrics
4. Health (2): CheckHealth, GetNLBStatus
5. Configuration (2): UpdateNLBConfig, GetNLBConfig
6. Load Balancing (3): RebalanceLoad, GetLoadDistribution, TriggerScaling

**Message Types**: ~42 request/response pairs

## Documentation Files Detailed

### README.md (371 lines)

**Sections**:
- Overview and architecture
- Proto file descriptions
- Code generation instructions
- Usage examples (Go and Python)
- Architecture flows
- Best practices
- Testing guide

**Includes**: 15+ code examples

### QUICKSTART.md (341 lines)

**Sections**:
- Quick start commands
- Implementation checklists
- Message type reference
- Enum quick reference
- Quick examples
- Debugging tips
- Performance tips
- Troubleshooting

**Target audience**: Developers implementing modules

### ARCHITECTURE_DIAGRAM.md (463 lines)

**Sections**:
- High-level architecture diagram
- Communication patterns (5 flows)
- Service interface summaries
- Complete request lifecycle
- Routing decision tree
- Health check hierarchy
- Scaling decision flow
- Error handling flow
- Network ports
- Proto file organization

**Target audience**: System architects and designers

### PHASE1_PROTO_IMPLEMENTATION.md (424 lines)

**Sections**:
- Task completion summary
- Deliverables overview
- Statistics and metrics
- Design decisions
- Integration points
- Next steps
- Testing checklist
- Compliance verification

**Target audience**: Project managers and technical leads

## Scripts Detailed

### gen-proto.sh (151 lines)

**Purpose**: Generate Go and Python code from proto definitions

**Features**:
- Dependency checking and installation
- Generates Go code to `pkg/proto/marchproxy/`
- Generates Python code to `manager/proto/marchproxy/`
- Fixes Python import paths
- Comprehensive error handling
- Detailed output

**Usage**: `./scripts/gen-proto.sh`

### validate-proto.sh (111 lines)

**Purpose**: Validate proto files for syntax errors

**Features**:
- Full validation with protoc (if available)
- Fallback basic validation
- Checks syntax declarations
- Validates package names
- Checks brace balancing
- Colored output

**Usage**: `./scripts/validate-proto.sh`

## Common Tasks

### Generate Code
```bash
./scripts/gen-proto.sh
```

### Validate Proto Files
```bash
./scripts/validate-proto.sh
```

### View Proto Definitions
```bash
# View types
cat proto/marchproxy/types.proto

# View module service
cat proto/marchproxy/module.proto

# View NLB service
cat proto/marchproxy/nlb.proto
```

### Count RPCs
```bash
# ModuleService RPCs
grep "rpc " proto/marchproxy/module.proto | wc -l
# Output: 25

# NLBService RPCs
grep "rpc " proto/marchproxy/nlb.proto | wc -l
# Output: 21
```

### Find Message Types
```bash
# All message definitions
grep "^message " proto/marchproxy/*.proto

# All enum definitions
grep "^enum " proto/marchproxy/*.proto
```

## Integration with MarchProxy

### Module Implementation (Go)

Modules must implement `ModuleService`:
- ALB (Application Load Balancer)
- DBLB (Database Load Balancer)
- AILB (AI Load Balancer)
- RTMP (Media Streaming)

**Location**: `pkg/modules/{alb,dblb,ailb,rtmp}/`

### NLB Implementation (Python)

NLB implements `NLBService`:
- Module registration
- Routing table management
- Metrics aggregation
- Load balancing

**Location**: `manager/nlb/` (future)

### Generated Code Locations

**Go**:
```
pkg/proto/marchproxy/
├── types.pb.go
├── module.pb.go
├── module_grpc.pb.go
├── nlb.pb.go
└── nlb_grpc.pb.go
```

**Python**:
```
manager/proto/marchproxy/
├── __init__.py
├── types_pb2.py
├── module_pb2.py
├── module_pb2_grpc.py
├── nlb_pb2.py
└── nlb_pb2_grpc.py
```

## Version History

- **v1.0.0** (2025-12-13) - Initial proto definitions
  - ModuleService with 25 RPCs
  - NLBService with 21 RPCs
  - Comprehensive type system
  - Complete documentation

## Support and Resources

### Internal Documentation
- [README.md](README.md) - Comprehensive guide
- [QUICKSTART.md](QUICKSTART.md) - Quick reference
- [ARCHITECTURE_DIAGRAM.md](ARCHITECTURE_DIAGRAM.md) - Visual diagrams

### External Resources
- [Protocol Buffers](https://protobuf.dev/)
- [gRPC Documentation](https://grpc.io/docs/)
- [gRPC Go](https://grpc.io/docs/languages/go/)
- [gRPC Python](https://grpc.io/docs/languages/python/)

### Project Resources
- Main README: `../README.md`
- Project plan: `../.PLAN-micro`
- Architecture docs: `../docs/ARCHITECTURE.md`

## Contributing

When modifying proto files:
1. Update the proto file
2. Run `./scripts/validate-proto.sh`
3. Run `./scripts/gen-proto.sh`
4. Update this INDEX.md if adding new files
5. Update README.md for significant changes
6. Update PHASE1_PROTO_IMPLEMENTATION.md for statistics

## Notes

- Proto files use proto3 syntax
- All timestamps use `google.protobuf.Timestamp`
- Enum zero values are `_UNSPECIFIED`
- Field numbers never reused
- Metadata maps for extensibility
- Consistent naming: PascalCase (messages), snake_case (fields)

---

**Last Updated**: 2025-12-13
**Phase**: 1 - gRPC Proto Definitions
**Status**: Complete ✅
