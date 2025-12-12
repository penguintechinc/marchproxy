# Phase 5: Enhanced Go Proxy L3/L4 with Enterprise Features

## Overview

Phase 5 creates `proxy-l3l4` - an enhanced version of `proxy-egress` with enterprise-grade L3/L4 networking features targeting 100+ Gbps throughput and sub-millisecond latency (p99 <1ms).

## Performance Targets

- **Throughput**: 100+ Gbps per proxy instance
- **Latency**: p50 <100μs, p99 <1ms, p99.9 <5ms
- **Connections**: 10M+ concurrent connections
- **Packet Rate**: 100M+ packets per second
- **CPU Efficiency**: <5% CPU at 10 Gbps

## Enterprise Features

### 1. QoS Traffic Shaping
- **Token bucket rate limiting** - Per-service bandwidth controls
- **Priority queues** - P0 (critical) to P3 (best-effort)
- **DSCP/ECN marking** - QoS-aware packet marking
- **Bandwidth guarantees** - Minimum bandwidth allocation

### 2. Multi-Cloud Routing
- **Cloud-aware routing** - AWS, GCP, Azure, on-premises
- **Dynamic path selection** - Latency, cost, bandwidth, geo-proximity
- **Active-active failover** - Sub-second failover times
- **Health probing** - RTT, packet loss, jitter monitoring

### 3. Deep Observability
- **OpenTelemetry tracing** - Distributed request tracing
- **Jaeger integration** - Trace visualization and analysis
- **Custom metrics** - Business-specific KPIs
- **Detailed flow analysis** - Per-connection statistics

### 4. NUMA Optimization
- **CPU affinity** - Pin workers to NUMA nodes
- **Memory locality** - NUMA-local buffer allocation
- **Interrupt handling** - Distribute interrupts across cores
- **Cache optimization** - Minimize cache line bouncing

## Implementation Strategy

### Step 1: Directory Creation

```bash
# Copy proxy-egress as baseline
cp -r /home/penguin/code/MarchProxy/proxy-egress /home/penguin/code/MarchProxy/proxy-l3l4

# Navigate to new directory
cd /home/penguin/code/MarchProxy/proxy-l3l4

# Update module name in go.mod
sed -i 's/marchproxy-egress/marchproxy-l3l4/g' go.mod

# Update all import paths
find . -name "*.go" -type f -exec sed -i 's/marchproxy-egress/marchproxy-l3l4/g' {} \;

# Update binary name in main.go
sed -i 's/marchproxy-egress/marchproxy-l3l4/g' cmd/proxy/main.go

# Create new enterprise feature directories
mkdir -p internal/qos
mkdir -p internal/multicloud
mkdir -p internal/observability
mkdir -p internal/numa

# Update .version file
echo "v1.0.0.$(date +%s)" > .version
```

### Step 2: New Directory Structure

```
proxy-l3l4/
├── cmd/
│   └── proxy/
│       └── main.go                 # Updated for L3/L4 features
├── internal/
│   ├── qos/                        # QoS and Traffic Shaping
│   │   ├── token_bucket.go         # Token bucket algorithm
│   │   ├── priority_queue.go       # P0-P3 priority queues
│   │   ├── bandwidth_limiter.go    # Per-service bandwidth limits
│   │   ├── dscp_marker.go          # DSCP/ECN marking
│   │   └── qos_test.go             # Unit tests
│   ├── multicloud/                 # Multi-Cloud Routing
│   │   ├── route_table.go          # Cloud route management
│   │   ├── health_probe.go         # RTT measurement
│   │   ├── routing_algorithm.go    # Path selection algorithms
│   │   ├── failover.go             # Failover logic
│   │   └── multicloud_test.go      # Unit tests
│   ├── observability/              # Deep Observability
│   │   ├── otel_tracer.go          # OpenTelemetry integration
│   │   ├── jaeger_exporter.go      # Jaeger exporter
│   │   ├── custom_metrics.go       # Custom business metrics
│   │   └── observability_test.go   # Unit tests
│   └── numa/                       # NUMA Optimization
│       ├── numa_affinity.go        # Thread affinity management
│       ├── memory_allocation.go    # NUMA-aware allocation
│       ├── interrupt_handler.go    # Interrupt distribution
│       └── numa_test.go            # Unit tests
├── Dockerfile                      # Docker build for L3/L4 proxy
├── go.mod                          # Module definition (marchproxy-l3l4)
├── go.sum                          # Dependency checksums
└── .version                        # Version file

```

### Step 3: Module and Dependencies Update

**Updated go.mod additions:**

```go
require (
    go.opentelemetry.io/otel v1.31.0
    go.opentelemetry.io/otel/exporters/jaeger v1.31.0
    go.opentelemetry.io/otel/sdk v1.31.0
    go.opentelemetry.io/otel/trace v1.31.0
    github.com/klauspost/compress v1.17.0  // High-performance compression
    golang.org/x/time v0.8.0                // Rate limiting
)
```

## Feature Implementation Details

### QoS Traffic Shaping

#### Token Bucket Algorithm (internal/qos/token_bucket.go)

```go
package qos

import (
    "sync"
    "time"
)

// TokenBucket implements token bucket rate limiting
type TokenBucket struct {
    capacity    int64         // Maximum tokens
    tokens      int64         // Current tokens
    refillRate  int64         // Tokens per second
    lastRefill  time.Time     // Last refill time
    mu          sync.Mutex
}

// NewTokenBucket creates a new token bucket
func NewTokenBucket(capacity, refillRate int64) *TokenBucket {
    return &TokenBucket{
        capacity:   capacity,
        tokens:     capacity,
        refillRate: refillRate,
        lastRefill: time.Now(),
    }
}

// Take attempts to consume tokens
func (tb *TokenBucket) Take(tokens int64) bool {
    tb.mu.Lock()
    defer tb.mu.Unlock()

    tb.refill()

    if tb.tokens >= tokens {
        tb.tokens -= tokens
        return true
    }

    return false
}

// refill adds tokens based on elapsed time
func (tb *TokenBucket) refill() {
    now := time.Now()
    elapsed := now.Sub(tb.lastRefill)

    tokensToAdd := int64(elapsed.Seconds() * float64(tb.refillRate))
    if tokensToAdd > 0 {
        tb.tokens = min(tb.capacity, tb.tokens+tokensToAdd)
        tb.lastRefill = now
    }
}
```

#### Priority Queue (internal/qos/priority_queue.go)

```go
package qos

import (
    "container/heap"
    "sync"
)

// Priority levels
const (
    PriorityCritical = 0  // P0 - Real-time traffic
    PriorityHigh     = 1  // P1 - Interactive traffic
    PriorityMedium   = 2  // P2 - Bulk transfer
    PriorityLow      = 3  // P3 - Best effort
)

// Packet represents a network packet with priority
type Packet struct {
    Data     []byte
    Priority int
    Sequence uint64
}

// PriorityQueue implements a priority-based packet queue
type PriorityQueue struct {
    queues   [4]*packetQueue  // P0-P3 queues
    mu       sync.RWMutex
    sequence uint64
}

type packetQueue []Packet

func (pq packetQueue) Len() int           { return len(pq) }
func (pq packetQueue) Less(i, j int) bool { return pq[i].Sequence < pq[j].Sequence }
func (pq packetQueue) Swap(i, j int)      { pq[i], pq[j] = pq[j], pq[i] }

func (pq *packetQueue) Push(x interface{}) {
    *pq = append(*pq, x.(Packet))
}

func (pq *packetQueue) Pop() interface{} {
    old := *pq
    n := len(old)
    packet := old[n-1]
    *pq = old[0 : n-1]
    return packet
}

// NewPriorityQueue creates a new priority queue
func NewPriorityQueue() *PriorityQueue {
    pq := &PriorityQueue{}
    for i := 0; i < 4; i++ {
        pq.queues[i] = &packetQueue{}
        heap.Init(pq.queues[i])
    }
    return pq
}

// Enqueue adds a packet to the appropriate priority queue
func (pq *PriorityQueue) Enqueue(data []byte, priority int) {
    pq.mu.Lock()
    defer pq.mu.Unlock()

    if priority < 0 || priority > 3 {
        priority = PriorityLow
    }

    packet := Packet{
        Data:     data,
        Priority: priority,
        Sequence: pq.sequence,
    }
    pq.sequence++

    heap.Push(pq.queues[priority], packet)
}

// Dequeue retrieves the highest priority packet
func (pq *PriorityQueue) Dequeue() (Packet, bool) {
    pq.mu.Lock()
    defer pq.mu.Unlock()

    // Check queues in priority order
    for i := 0; i < 4; i++ {
        if pq.queues[i].Len() > 0 {
            packet := heap.Pop(pq.queues[i]).(Packet)
            return packet, true
        }
    }

    return Packet{}, false
}
```

#### DSCP Marker (internal/qos/dscp_marker.go)

```go
package qos

import (
    "encoding/binary"
    "net"
)

// DSCP values for different traffic classes
const (
    DSCPBestEffort = 0x00  // Default
    DSCPLowDrop    = 0x28  // AF11
    DSCPMediumDrop = 0x30  // AF12
    DSCPHighDrop   = 0x38  // AF13
    DSCPExpedited  = 0xB8  // EF (low latency)
)

// DSCPMarker handles DSCP/ECN packet marking
type DSCPMarker struct {
    priorityToDSCP map[int]uint8
}

// NewDSCPMarker creates a new DSCP marker
func NewDSCPMarker() *DSCPMarker {
    return &DSCPMarker{
        priorityToDSCP: map[int]uint8{
            PriorityCritical: DSCPExpedited,
            PriorityHigh:     DSCPLowDrop,
            PriorityMedium:   DSCPMediumDrop,
            PriorityLow:      DSCPBestEffort,
        },
    }
}

// MarkPacket sets DSCP value in IP header
func (dm *DSCPMarker) MarkPacket(packet []byte, priority int) error {
    if len(packet) < 20 {
        return ErrPacketTooSmall
    }

    // Check IP version
    version := packet[0] >> 4
    if version != 4 {
        return ErrUnsupportedIPVersion
    }

    // Get DSCP value for priority
    dscp, ok := dm.priorityToDSCP[priority]
    if !ok {
        dscp = DSCPBestEffort
    }

    // Set DSCP in TOS field (byte 1, bits 2-7)
    tos := packet[1]
    tos = (tos & 0x03) | (dscp << 2)
    packet[1] = tos

    // Recalculate IP checksum
    packet[10] = 0  // Clear old checksum
    packet[11] = 0
    checksum := ipChecksum(packet[:20])
    binary.BigEndian.PutUint16(packet[10:12], checksum)

    return nil
}

// ipChecksum calculates IP header checksum
func ipChecksum(header []byte) uint16 {
    var sum uint32
    for i := 0; i < len(header); i += 2 {
        sum += uint32(binary.BigEndian.Uint16(header[i : i+2]))
    }
    for sum > 0xFFFF {
        sum = (sum & 0xFFFF) + (sum >> 16)
    }
    return ^uint16(sum)
}

var (
    ErrPacketTooSmall        = fmt.Errorf("packet too small for IP header")
    ErrUnsupportedIPVersion  = fmt.Errorf("unsupported IP version")
)
```

### Multi-Cloud Routing

#### Route Table (internal/multicloud/route_table.go)

```go
package multicloud

import (
    "net"
    "sync"
    "time"
)

// CloudProvider represents a cloud provider
type CloudProvider string

const (
    ProviderAWS    CloudProvider = "aws"
    ProviderGCP    CloudProvider = "gcp"
    ProviderAzure  CloudProvider = "azure"
    ProviderOnPrem CloudProvider = "onprem"
)

// Route represents a network route to a destination
type Route struct {
    ID          string
    Destination *net.IPNet
    Gateway     net.IP
    Provider    CloudProvider
    Region      string
    Cost        float64      // Cost per GB
    Latency     time.Duration
    Bandwidth   int64        // Bits per second
    Active      bool
    LastCheck   time.Time
}

// RouteTable manages routes across multiple clouds
type RouteTable struct {
    routes map[string]*Route
    mu     sync.RWMutex
}

// NewRouteTable creates a new route table
func NewRouteTable() *RouteTable {
    return &RouteTable{
        routes: make(map[string]*Route),
    }
}

// AddRoute adds a route to the table
func (rt *RouteTable) AddRoute(route *Route) {
    rt.mu.Lock()
    defer rt.mu.Unlock()
    rt.routes[route.ID] = route
}

// GetBestRoute finds the best route for a destination
func (rt *RouteTable) GetBestRoute(dest net.IP, algorithm RoutingAlgorithm) *Route {
    rt.mu.RLock()
    defer rt.mu.RUnlock()

    var candidates []*Route
    for _, route := range rt.routes {
        if route.Active && route.Destination.Contains(dest) {
            candidates = append(candidates, route)
        }
    }

    if len(candidates) == 0 {
        return nil
    }

    return algorithm.Select(candidates)
}

// UpdateRouteHealth updates route health metrics
func (rt *RouteTable) UpdateRouteHealth(routeID string, latency time.Duration, packetLoss float64) {
    rt.mu.Lock()
    defer rt.mu.Unlock()

    if route, exists := rt.routes[routeID]; exists {
        route.Latency = latency
        route.LastCheck = time.Now()

        // Mark route as inactive if packet loss is too high
        if packetLoss > 0.05 {  // 5% packet loss threshold
            route.Active = false
        }
    }
}
```

#### Health Probe (internal/multicloud/health_probe.go)

```go
package multicloud

import (
    "context"
    "net"
    "time"
)

// HealthProbe performs health checks on routes
type HealthProbe struct {
    routeTable *RouteTable
    interval   time.Duration
    timeout    time.Duration
}

// NewHealthProbe creates a new health probe
func NewHealthProbe(rt *RouteTable, interval, timeout time.Duration) *HealthProbe {
    return &HealthProbe{
        routeTable: rt,
        interval:   interval,
        timeout:    timeout,
    }
}

// Start begins health probing
func (hp *HealthProbe) Start(ctx context.Context) {
    ticker := time.NewTicker(hp.interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            hp.probeAllRoutes()
        }
    }
}

// probeAllRoutes checks health of all routes
func (hp *HealthProbe) probeAllRoutes() {
    hp.routeTable.mu.RLock()
    routes := make([]*Route, 0, len(hp.routeTable.routes))
    for _, route := range hp.routeTable.routes {
        routes = append(routes, route)
    }
    hp.routeTable.mu.RUnlock()

    for _, route := range routes {
        latency, packetLoss := hp.probeRoute(route)
        hp.routeTable.UpdateRouteHealth(route.ID, latency, packetLoss)
    }
}

// probeRoute measures RTT to a route's gateway
func (hp *HealthProbe) probeRoute(route *Route) (time.Duration, float64) {
    start := time.Now()

    conn, err := net.DialTimeout("tcp", net.JoinHostPort(route.Gateway.String(), "80"), hp.timeout)
    if err != nil {
        return hp.timeout, 1.0  // 100% packet loss
    }
    defer conn.Close()

    latency := time.Since(start)
    return latency, 0.0  // 0% packet loss for successful probe
}
```

#### Routing Algorithms (internal/multicloud/routing_algorithm.go)

```go
package multicloud

import (
    "math"
    "time"
)

// RoutingAlgorithm defines interface for route selection
type RoutingAlgorithm interface {
    Select(routes []*Route) *Route
}

// LatencyBasedRouting selects route with lowest latency
type LatencyBasedRouting struct{}

func (l *LatencyBasedRouting) Select(routes []*Route) *Route {
    if len(routes) == 0 {
        return nil
    }

    best := routes[0]
    for _, route := range routes[1:] {
        if route.Latency < best.Latency {
            best = route
        }
    }
    return best
}

// CostBasedRouting selects route with lowest cost
type CostBasedRouting struct{}

func (c *CostBasedRouting) Select(routes []*Route) *Route {
    if len(routes) == 0 {
        return nil
    }

    best := routes[0]
    for _, route := range routes[1:] {
        if route.Cost < best.Cost {
            best = route
        }
    }
    return best
}

// WeightedRouting selects route based on weighted score
type WeightedRouting struct {
    LatencyWeight   float64
    CostWeight      float64
    BandwidthWeight float64
}

func (w *WeightedRouting) Select(routes []*Route) *Route {
    if len(routes) == 0 {
        return nil
    }

    // Normalize metrics
    maxLatency := maxLatencyFromRoutes(routes)
    maxCost := maxCostFromRoutes(routes)
    maxBandwidth := maxBandwidthFromRoutes(routes)

    best := routes[0]
    bestScore := w.calculateScore(best, maxLatency, maxCost, maxBandwidth)

    for _, route := range routes[1:] {
        score := w.calculateScore(route, maxLatency, maxCost, maxBandwidth)
        if score < bestScore {
            best = route
            bestScore = score
        }
    }

    return best
}

func (w *WeightedRouting) calculateScore(route *Route, maxLatency time.Duration, maxCost, maxBandwidth float64) float64 {
    latencyScore := float64(route.Latency) / float64(maxLatency)
    costScore := route.Cost / maxCost
    bandwidthScore := 1.0 - (float64(route.Bandwidth) / maxBandwidth)

    return w.LatencyWeight*latencyScore + w.CostWeight*costScore + w.BandwidthWeight*bandwidthScore
}

func maxLatencyFromRoutes(routes []*Route) time.Duration {
    max := routes[0].Latency
    for _, r := range routes[1:] {
        if r.Latency > max {
            max = r.Latency
        }
    }
    return max
}

func maxCostFromRoutes(routes []*Route) float64 {
    max := routes[0].Cost
    for _, r := range routes[1:] {
        if r.Cost > max {
            max = r.Cost
        }
    }
    return max
}

func maxBandwidthFromRoutes(routes []*Route) float64 {
    max := float64(routes[0].Bandwidth)
    for _, r := range routes[1:] {
        if float64(r.Bandwidth) > max {
            max = float64(r.Bandwidth)
        }
    }
    return max
}
```

### Deep Observability

#### OpenTelemetry Tracer (internal/observability/otel_tracer.go)

```go
package observability

import (
    "context"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

// TracerManager manages OpenTelemetry tracing
type TracerManager struct {
    tracer trace.Tracer
}

// NewTracerManager creates a new tracer manager
func NewTracerManager(serviceName string) *TracerManager {
    return &TracerManager{
        tracer: otel.Tracer(serviceName),
    }
}

// StartSpan creates a new trace span
func (tm *TracerManager) StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
    return tm.tracer.Start(ctx, name, trace.WithAttributes(attrs...))
}

// TraceConnection traces a proxy connection
func (tm *TracerManager) TraceConnection(ctx context.Context, srcIP, dstIP string, srcPort, dstPort int) (context.Context, trace.Span) {
    attrs := []attribute.KeyValue{
        attribute.String("src.ip", srcIP),
        attribute.String("dst.ip", dstIP),
        attribute.Int("src.port", srcPort),
        attribute.Int("dst.port", dstPort),
    }
    return tm.StartSpan(ctx, "proxy.connection", attrs...)
}
```

### NUMA Optimization

#### NUMA Affinity (internal/numa/numa_affinity.go)

```go
//go:build linux

package numa

import (
    "fmt"
    "runtime"
    "syscall"
)

// SetCPUAffinity pins the current thread to specific CPUs
func SetCPUAffinity(cpus []int) error {
    var cpuSet syscall.CPUSet
    for _, cpu := range cpus {
        cpuSet.Set(cpu)
    }

    runtime.LockOSThread()
    return syscall.SchedSetaffinity(0, &cpuSet)
}

// GetNUMANode returns the NUMA node for a CPU
func GetNUMANode(cpu int) (int, error) {
    // Implementation would read from /sys/devices/system/node/node*/cpulist
    // Simplified for example
    return cpu / (runtime.NumCPU() / 2), nil
}
```

## Backward Compatibility

All enterprise features are **optional** and disabled by default. proxy-l3l4 maintains full backward compatibility with proxy-egress:

- Same CLI flags and environment variables
- Same configuration format
- Same manager API integration
- Same metrics and health check endpoints
- Graceful degradation when features are disabled

## Docker Integration

### Dockerfile Updates

```dockerfile
FROM debian:12-slim as builder

RUN apt-get update && apt-get install -y \
    golang-1.24 \
    git \
    make \
    gcc \
    libbpf-dev

WORKDIR /build
COPY . .

RUN go mod download
RUN CGO_ENABLED=1 go build -o marchproxy-l3l4 cmd/proxy/main.go

FROM debian:12-slim

RUN apt-get update && apt-get install -y \
    ca-certificates \
    libbpf0 \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /build/marchproxy-l3l4 /usr/local/bin/

EXPOSE 8080 8081

ENTRYPOINT ["/usr/local/bin/marchproxy-l3l4"]
```

### docker-compose.yml Addition

```yaml
services:
  proxy-l3l4:
    build:
      context: ./proxy-l3l4
      dockerfile: Dockerfile
    image: marchproxy/proxy-l3l4:latest
    container_name: marchproxy-proxy-l3l4
    environment:
      - MANAGER_URL=http://manager:8000
      - CLUSTER_API_KEY=${CLUSTER_API_KEY}
      - LOG_LEVEL=INFO
      - ENABLE_QOS=true
      - ENABLE_MULTICLOUD=true
      - ENABLE_OTEL=true
      - NUMA_OPTIMIZATION=true
    ports:
      - "9080:8080"  # Proxy port
      - "9081:8081"  # Metrics port
    networks:
      - marchproxy
    depends_on:
      - manager
    cap_add:
      - NET_ADMIN
      - SYS_ADMIN
    privileged: true  # Required for eBPF and NUMA
```

## GitHub Actions Workflow

Create `.github/workflows/proxy-l3l4-ci.yml`:

```yaml
name: Proxy L3/L4 CI/CD

on:
  push:
    branches: [main, develop]
    paths:
      - 'proxy-l3l4/**'
      - '.version'
  pull_request:
    paths:
      - 'proxy-l3l4/**'

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - name: golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          working-directory: proxy-l3l4

  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - name: Run tests
        working-directory: proxy-l3l4
        run: go test -v -race -coverprofile=coverage.out ./...

  build:
    runs-on: ubuntu-latest
    needs: [lint, test]
    steps:
      - uses: actions/checkout@v4
      - name: Build Docker image
        run: |
          docker build -t marchproxy/proxy-l3l4:${GITHUB_SHA::8} proxy-l3l4
```

## Testing Strategy

### Unit Tests
- QoS components (token bucket, priority queue, DSCP marker)
- Multi-cloud routing (route table, health probe, algorithms)
- Observability (tracer, metrics)
- NUMA optimization (affinity, allocation)

### Integration Tests
- End-to-end QoS enforcement
- Multi-cloud failover scenarios
- Distributed tracing flows
- NUMA performance validation

### Performance Tests
- 100 Gbps throughput benchmarking
- Sub-millisecond latency verification
- 10M concurrent connection testing
- CPU efficiency measurements

## Documentation Updates

Update the following documentation:

1. **README.md** - Add proxy-l3l4 to architecture overview
2. **docs/ARCHITECTURE.md** - Document L3/L4 enterprise features
3. **docs/PERFORMANCE.md** - Add performance tuning guide
4. **docs/QOS.md** - QoS configuration and usage
5. **docs/MULTICLOUD.md** - Multi-cloud routing guide
6. **docs/OBSERVABILITY.md** - Observability setup

## Success Criteria

- [ ] proxy-l3l4 builds successfully
- [ ] All unit tests pass (80%+ coverage)
- [ ] Integration tests validate enterprise features
- [ ] Performance targets met (100+ Gbps, <1ms p99)
- [ ] Docker image builds and runs
- [ ] GitHub Actions workflow passes
- [ ] Documentation complete and accurate
- [ ] Backward compatibility maintained
- [ ] No security vulnerabilities in dependencies

## Timeline

- **Week 1**: Directory setup, module updates, QoS implementation
- **Week 2**: Multi-cloud routing, observability integration
- **Week 3**: NUMA optimization, testing infrastructure
- **Week 4**: Performance validation, documentation, CI/CD

## Next Steps

1. Execute directory copy and module rename commands
2. Implement QoS traffic shaping components
3. Implement multi-cloud routing system
4. Integrate OpenTelemetry and Jaeger
5. Add NUMA optimization
6. Create comprehensive test suite
7. Build Docker image and validate
8. Update documentation
9. Submit for code review

---

**NOTE**: Due to the current Bash tool permission limitations, manual execution of the directory copy commands is required. After copying, all subsequent implementation can proceed as documented above.