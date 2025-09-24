# Technical Architecture Deep Dive

This document provides a comprehensive technical overview of MarchProxy's architecture, design patterns, and implementation details.

## System Architecture Overview

MarchProxy follows a distributed microservices architecture with clear separation of concerns:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            MarchProxy System                                │
│                                                                             │
│  ┌───────────────────────┐    ┌─────────────────────────────────────────┐  │
│  │      Manager          │    │            Proxy Cluster               │  │
│  │   (Control Plane)     │    │         (Data Plane)                   │  │
│  │                       │    │                                         │  │
│  │ ┌─────────────────┐   │    │ ┌─────────┐ ┌─────────┐ ┌─────────┐   │  │
│  │ │   Web Server    │   │    │ │ Proxy 1 │ │ Proxy 2 │ │ Proxy N │   │  │
│  │ │   (py4web)      │   │    │ │  (Go)   │ │  (Go)   │ │  (Go)   │   │  │
│  │ └─────────────────┘   │    │ └─────────┘ └─────────┘ └─────────┘   │  │
│  │ ┌─────────────────┐   │    │                                         │  │
│  │ │   API Server    │◄──┼────┼─────────────┐                           │  │
│  │ │   (REST/JSON)   │   │    │             │                           │  │
│  │ └─────────────────┘   │    │             │                           │  │
│  │ ┌─────────────────┐   │    │ ┌───────────▼─────────────────────────┐ │  │
│  │ │  Auth Service   │   │    │ │         eBPF Layer                  │ │  │
│  │ │  (JWT/SAML)     │   │    │ │                                     │ │  │
│  │ └─────────────────┘   │    │ │ ┌─────────┐ ┌─────────┐ ┌─────────┐ │ │  │
│  │ ┌─────────────────┐   │    │ │ │   XDP   │ │   TC    │ │  Maps   │ │ │  │
│  │ │ License Service │   │    │ │ │Programs │ │Programs │ │ & State │ │ │  │
│  │ └─────────────────┘   │    │ │ └─────────┘ └─────────┘ └─────────┘ │ │  │
│  │ ┌─────────────────┐   │    │ └─────────────────────────────────────┘ │  │
│  │ │   PostgreSQL    │   │    │                                         │  │
│  │ │   Database      │   │    │ ┌─────────────────────────────────────┐ │  │
│  │ └─────────────────┘   │    │ │      Hardware Acceleration          │ │  │
│  └───────────────────────┘    │ │                                     │ │  │
│                               │ │ ┌─────────┐ ┌─────────┐ ┌─────────┐ │ │  │
│  ┌───────────────────────┐    │ │ │  DPDK   │ │ AF_XDP  │ │ SR-IOV  │ │ │  │
│  │   Observability       │    │ │ │(kernel  │ │(zero-   │ │(hardware│ │ │  │
│  │                       │    │ │ │ bypass) │ │ copy)   │ │isolation)│ │ │  │
│  │ ┌─────────────────┐   │    │ │ └─────────┘ └─────────┘ └─────────┘ │ │  │
│  │ │   Prometheus    │   │    │ └─────────────────────────────────────┘ │  │
│  │ └─────────────────┘   │    └─────────────────────────────────────────┘  │
│  │ ┌─────────────────┐   │                                                │
│  │ │    Grafana      │   │                                                │
│  │ └─────────────────┘   │                                                │
│  │ ┌─────────────────┐   │                                                │
│  │ │   ELK Stack     │   │                                                │
│  │ └─────────────────┘   │                                                │
│  └───────────────────────┘                                                │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Component Architecture

### Manager Component (Control Plane)

The Manager is built using Python with py4web framework and follows a layered architecture:

```
┌─────────────────────────────────────────────────────────────────┐
│                      Manager Architecture                       │
│                                                                 │
│ ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐   │
│ │  Presentation   │ │   Presentation  │ │   Presentation  │   │
│ │     Layer       │ │     Layer       │ │     Layer       │   │
│ │                 │ │                 │ │                 │   │
│ │  ┌───────────┐  │ │  ┌───────────┐  │ │  ┌───────────┐  │   │
│ │  │    Web    │  │ │  │    API    │  │ │  │    CLI    │  │   │
│ │  │    UI     │  │ │  │   Server  │  │ │  │Interface  │  │   │
│ │  │(HTML/JS)  │  │ │  │(REST/JSON)│  │ │  │  (Click)  │  │   │
│ │  └───────────┘  │ │  └───────────┘  │ │  └───────────┘  │   │
│ └─────────────────┘ └─────────────────┘ └─────────────────┘   │
│           │                   │                   │           │
│           └───────────────────┼───────────────────┘           │
│                               │                               │
│ ┌─────────────────────────────▼─────────────────────────────┐ │
│ │                    Business Logic Layer                   │ │
│ │                                                           │ │
│ │ ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐   │ │
│ │ │    Auth     │ │   Service   │ │      License        │   │ │
│ │ │  Service    │ │ Management  │ │     Service         │   │ │
│ │ │             │ │             │ │                     │   │ │
│ │ │ • JWT       │ │ • CRUD Ops  │ │ • Validation        │   │ │
│ │ │ • SAML      │ │ • Mapping   │ │ • Feature Gates     │   │ │
│ │ │ • OAuth2    │ │ • Clusters  │ │ • Limits            │   │ │
│ │ │ • 2FA       │ │ • Users     │ │ • Reporting         │   │ │
│ │ └─────────────┘ └─────────────┘ └─────────────────────┘   │ │
│ │                                                           │ │
│ │ ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐   │ │
│ │ │Certificate  │ │   Config    │ │      Proxy          │   │ │
│ │ │ Management  │ │ Validation  │ │    Management       │   │ │
│ │ │             │ │             │ │                     │   │ │
│ │ │ • CA Mgmt   │ │ • Schema    │ │ • Registration      │   │ │
│ │ │ • Wildcard  │ │ • Validation│ │ • Heartbeat         │   │ │
│ │ │ • Auto-Renew│ │ • Sync      │ │ • Health Check      │   │ │
│ │ │ • Vault Intg│ │ • Rollback  │ │ • Load Balancing    │   │ │
│ │ └─────────────┘ └─────────────┘ └─────────────────────┘   │ │
│ └─────────────────────────────────────────────────────────┘ │
│                               │                             │
│ ┌─────────────────────────────▼─────────────────────────────┐ │
│ │                    Data Access Layer                      │ │
│ │                                                           │ │
│ │ ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐   │ │
│ │ │  Database   │ │   Cache     │ │      External       │   │ │
│ │ │   Access    │ │    Layer    │ │      Services       │   │ │
│ │ │             │ │             │ │                     │   │ │
│ │ │ • pydal ORM │ │ • Redis     │ │ • License Server    │   │ │
│ │ │ • Migrations│ │ • Memory    │ │ • Vault/Infisical   │   │ │
│ │ │ • Pooling   │ │ • LRU       │ │ • SAML/OAuth2 IdP   │   │ │
│ │ │ • Validation│ │ • TTL       │ │ • Monitoring        │   │ │
│ │ └─────────────┘ └─────────────┘ └─────────────────────┘   │ │
│ └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

#### Database Schema Design

MarchProxy uses a normalized database schema with proper indexing and constraints:

```sql
-- Core entities
CREATE TABLE clusters (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    api_key VARCHAR(255) UNIQUE NOT NULL,
    syslog_endpoint VARCHAR(255),
    log_auth BOOLEAN DEFAULT true,
    log_netflow BOOLEAN DEFAULT false,
    log_debug BOOLEAN DEFAULT false,
    is_active BOOLEAN DEFAULT true,
    created_by INTEGER REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255),
    is_admin BOOLEAN DEFAULT false,
    totp_secret VARCHAR(32),
    auth_provider VARCHAR(50) DEFAULT 'local',
    external_id VARCHAR(255),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    last_login TIMESTAMP
);

CREATE TABLE services (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    ip_fqdn VARCHAR(255) NOT NULL,
    collection VARCHAR(50) NOT NULL,
    cluster_id INTEGER REFERENCES clusters(id),
    auth_type VARCHAR(20) DEFAULT 'none',
    token_base64 TEXT,
    jwt_secret VARCHAR(255),
    jwt_expiry INTEGER DEFAULT 3600,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(name, cluster_id)
);

CREATE TABLE mappings (
    id SERIAL PRIMARY KEY,
    source_services TEXT[] NOT NULL,
    dest_services TEXT[] NOT NULL,
    cluster_id INTEGER REFERENCES clusters(id),
    protocols TEXT[] NOT NULL,
    ports INTEGER[] NOT NULL,
    auth_required BOOLEAN DEFAULT false,
    comments TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_services_cluster ON services(cluster_id);
CREATE INDEX idx_services_collection ON services(collection);
CREATE INDEX idx_mappings_cluster ON mappings(cluster_id);
CREATE INDEX idx_users_auth_provider ON users(auth_provider);
CREATE INDEX idx_users_external_id ON users(external_id);
```

### Proxy Component (Data Plane)

The Proxy is built in Go with a multi-layered architecture optimized for performance:

```
┌─────────────────────────────────────────────────────────────────┐
│                      Proxy Architecture                         │
│                                                                 │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │                    Application Layer                        │ │
│ │                                                             │ │
│ │ ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐     │ │
│ │ │    HTTP     │ │    Admin    │ │      Metrics        │     │ │
│ │ │   Server    │ │  Interface  │ │      Server         │     │ │
│ │ │             │ │             │ │                     │     │ │
│ │ │ • REST API  │ │ • Health    │ │ • Prometheus        │     │ │
│ │ │ • WebUI     │ │ • Config    │ │ • Custom Metrics    │     │ │
│ │ │ • Auth      │ │ • Debug     │ │ • Performance       │     │ │
│ │ └─────────────┘ └─────────────┘ └─────────────────────┘     │ │
│ └─────────────────────────────────────────────────────────────┘ │
│                               │                                 │
│ ┌─────────────────────────────▼─────────────────────────────────┐ │
│ │                    Middleware Layer                          │ │
│ │                                                               │ │
│ │ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────┐ │ │
│ │ │     Auth    │ │Rate Limiting│ │   Circuit   │ │   WAF   │ │ │
│ │ │ Middleware  │ │ Middleware  │ │   Breaker   │ │Middleware│ │ │
│ │ │             │ │             │ │             │ │         │ │ │
│ │ │ • JWT Val   │ │ • Token Bkt │ │ • Fail Fast │ │ • SQLi  │ │ │
│ │ │ • API Key   │ │ • Sliding   │ │ • Recovery  │ │ • XSS   │ │ │
│ │ │ • mTLS      │ │ • Per-IP    │ │ • Metrics   │ │ • CSRF  │ │ │
│ │ └─────────────┘ └─────────────┘ └─────────────┘ └─────────┘ │ │
│ └─────────────────────────────────────────────────────────────┘ │
│                               │                                 │
│ ┌─────────────────────────────▼─────────────────────────────────┐ │
│ │                   Proxy Core Layer                           │ │
│ │                                                               │ │
│ │ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────┐ │ │
│ │ │   Protocol  │ │ Load Balance│ │ Connection  │ │Service  │ │ │
│ │ │   Handlers  │ │  & Routing  │ │   Pooling   │ │Discovery│ │ │
│ │ │             │ │             │ │             │ │         │ │ │
│ │ │ • HTTP/HTTPS│ │ • Round Robin│ │ • Pool Mgmt │ │ • Config│ │ │
│ │ │ • WebSocket │ │ • Least Conn │ │ • Health    │ │ • Cache │ │ │
│ │ │ • QUIC/H3   │ │ • IP Hash   │ │ • Timeouts  │ │ • Sync  │ │ │
│ │ │ • TCP/UDP   │ │ • Weighted  │ │ • Keepalive │ │ • Watch │ │ │
│ │ └─────────────┘ └─────────────┘ └─────────────┘ └─────────┘ │ │
│ └─────────────────────────────────────────────────────────────┘ │
│                               │                                 │
│ ┌─────────────────────────────▼─────────────────────────────────┐ │
│ │                    eBPF Acceleration Layer                   │ │
│ │                                                               │ │
│ │ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────┐ │ │
│ │ │     XDP     │ │     TC      │ │    Maps     │ │Programs │ │ │
│ │ │  Programs   │ │  Programs   │ │   & State   │ │ Loader  │ │ │
│ │ │             │ │             │ │             │ │         │ │ │
│ │ │ • Filtering │ │ • QoS       │ │ • LRU Cache │ │ • Verify│ │ │
│ │ │ • Rate Lmt  │ │ • Shaping   │ │ • LPM Trie  │ │ • Load  │ │ │
│ │ │ • DDoS Prot │ │ • Redirect  │ │ • Hash Map  │ │ • Update│ │ │
│ │ │ • Load Bal  │ │ • Mirror    │ │ • Ring Buf  │ │ • Watch │ │ │
│ │ └─────────────┘ └─────────────┘ └─────────────┘ └─────────┘ │ │
│ └─────────────────────────────────────────────────────────────┘ │
│                               │                                 │
│ ┌─────────────────────────────▼─────────────────────────────────┐ │
│ │                Hardware Acceleration Layer                   │ │
│ │                                                               │ │
│ │ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────┐ │ │
│ │ │    DPDK     │ │   AF_XDP    │ │   SR-IOV    │ │  NUMA   │ │ │
│ │ │             │ │             │ │             │ │Optimized│ │ │
│ │ │ • PMD       │ │ • Zero Copy │ │ • Hardware  │ │         │ │ │
│ │ │ • Huge Pages│ │ • Batching  │ │ • Isolation │ │ • CPU   │ │ │
│ │ │ • CPU Cores │ │ • Polling   │ │ • Passthru  │ │ • Memory│ │ │
│ │ │ • Memory    │ │ • Lock Free │ │ • VF Mgmt   │ │ • Cache │ │ │
│ │ └─────────────┘ └─────────────┘ └─────────────┘ └─────────┘ │ │
│ └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

## Design Patterns and Principles

### 1. Microservices Architecture

MarchProxy follows microservices principles with:

- **Service Independence**: Each component can be developed, deployed, and scaled independently
- **Technology Diversity**: Manager (Python) and Proxy (Go) use optimal technologies for their roles
- **Failure Isolation**: Component failures don't cascade to the entire system
- **Horizontal Scaling**: Proxy components can be scaled based on load

### 2. Event-Driven Architecture

```go
// Event system for configuration changes
type EventType string

const (
    ServiceCreated EventType = "service.created"
    ServiceUpdated EventType = "service.updated"
    ServiceDeleted EventType = "service.deleted"
    MappingCreated EventType = "mapping.created"
    ClusterUpdated EventType = "cluster.updated"
)

type Event struct {
    ID        string    `json:"id"`
    Type      EventType `json:"type"`
    Source    string    `json:"source"`
    Data      any       `json:"data"`
    Timestamp time.Time `json:"timestamp"`
    Version   string    `json:"version"`
}

type EventBus interface {
    Publish(event Event) error
    Subscribe(eventType EventType, handler EventHandler) error
    Unsubscribe(eventType EventType, handlerID string) error
}

type EventHandler func(event Event) error
```

### 3. Command Query Responsibility Segregation (CQRS)

Separate read and write operations for optimal performance:

```go
// Command side - writes
type ServiceCommand interface {
    CreateService(ctx context.Context, req CreateServiceRequest) (*Service, error)
    UpdateService(ctx context.Context, id int, req UpdateServiceRequest) (*Service, error)
    DeleteService(ctx context.Context, id int) error
}

// Query side - reads
type ServiceQuery interface {
    GetService(ctx context.Context, id int) (*Service, error)
    ListServices(ctx context.Context, filters ServiceFilters) ([]Service, error)
    GetServicesByCluster(ctx context.Context, clusterID int) ([]Service, error)
}
```

### 4. Repository Pattern

Abstract data access with repository interfaces:

```go
type ServiceRepository interface {
    Create(ctx context.Context, service *Service) error
    GetByID(ctx context.Context, id int) (*Service, error)
    GetByName(ctx context.Context, name string, clusterID int) (*Service, error)
    List(ctx context.Context, filters ServiceFilters) ([]Service, error)
    Update(ctx context.Context, service *Service) error
    Delete(ctx context.Context, id int) error
}

type PostgreSQLServiceRepository struct {
    db *sql.DB
}

func (r *PostgreSQLServiceRepository) Create(ctx context.Context, service *Service) error {
    query := `
        INSERT INTO services (name, ip_fqdn, collection, cluster_id, auth_type, created_at)
        VALUES ($1, $2, $3, $4, $5, NOW())
        RETURNING id, created_at
    `
    return r.db.QueryRowContext(ctx, query,
        service.Name, service.IPFQDN, service.Collection,
        service.ClusterID, service.AuthType,
    ).Scan(&service.ID, &service.CreatedAt)
}
```

### 5. Middleware Pattern

Composable request processing pipeline:

```go
type Middleware func(http.Handler) http.Handler

func ChainMiddleware(middlewares ...Middleware) Middleware {
    return func(next http.Handler) http.Handler {
        for i := len(middlewares) - 1; i >= 0; i-- {
            next = middlewares[i](next)
        }
        return next
    }
}

// Authentication middleware
func AuthMiddleware(authService AuthService) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := extractToken(r)
            if token == "" {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }

            user, err := authService.ValidateToken(r.Context(), token)
            if err != nil {
                http.Error(w, "Invalid token", http.StatusUnauthorized)
                return
            }

            ctx := context.WithValue(r.Context(), "user", user)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// Rate limiting middleware
func RateLimitMiddleware(limiter RateLimiter) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            clientIP := getClientIP(r)

            if !limiter.Allow(clientIP) {
                http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

## eBPF Integration Architecture

### eBPF Program Structure

```c
// XDP rate limiting program
#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/tcp.h>
#include <linux/udp.h>
#include <bpf/bpf_helpers.h>

// BPF maps for state management
struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __type(key, __u32);    // Source IP
    __type(value, struct rate_limit_entry);
    __uint(max_entries, 65536);
} rate_limit_map SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __type(key, __u32);
    __type(value, struct rate_limit_config);
    __uint(max_entries, 1);
} config_map SEC(".maps");

struct rate_limit_entry {
    __u64 last_time;
    __u32 tokens;
    __u32 packets;
};

struct rate_limit_config {
    __u32 rate_limit;      // packets per second
    __u32 burst_size;      // burst capacity
    __u32 time_window;     // time window in nanoseconds
};

SEC("xdp")
int xdp_rate_limit(struct xdp_md *ctx) {
    void *data_end = (void *)(long)ctx->data_end;
    void *data = (void *)(long)ctx->data;

    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return XDP_PASS;

    if (eth->h_proto != bpf_htons(ETH_P_IP))
        return XDP_PASS;

    struct iphdr *ip = (void *)(eth + 1);
    if ((void *)(ip + 1) > data_end)
        return XDP_PASS;

    __u32 src_ip = ip->saddr;
    __u64 now = bpf_ktime_get_ns();

    // Get configuration
    __u32 config_key = 0;
    struct rate_limit_config *config = bpf_map_lookup_elem(&config_map, &config_key);
    if (!config)
        return XDP_PASS;

    // Get or create rate limit entry
    struct rate_limit_entry *entry = bpf_map_lookup_elem(&rate_limit_map, &src_ip);
    if (!entry) {
        struct rate_limit_entry new_entry = {
            .last_time = now,
            .tokens = config->burst_size - 1,
            .packets = 1
        };
        bpf_map_update_elem(&rate_limit_map, &src_ip, &new_entry, BPF_ANY);
        return XDP_PASS;
    }

    // Token bucket algorithm
    __u64 time_diff = now - entry->last_time;
    if (time_diff > config->time_window) {
        // Refill tokens
        __u32 new_tokens = (time_diff * config->rate_limit) / 1000000000; // Convert to seconds
        entry->tokens = (entry->tokens + new_tokens > config->burst_size) ?
                       config->burst_size : entry->tokens + new_tokens;
        entry->last_time = now;
    }

    // Check if we have tokens
    if (entry->tokens > 0) {
        entry->tokens--;
        entry->packets++;
        bpf_map_update_elem(&rate_limit_map, &src_ip, entry, BPF_EXIST);
        return XDP_PASS;
    }

    // Rate limit exceeded
    return XDP_DROP;
}

char _license[] SEC("license") = "GPL";
```

### Go-eBPF Integration

```go
package ebpf

import (
    "github.com/cilium/ebpf"
    "github.com/cilium/ebpf/link"
    "github.com/cilium/ebpf/rlimit"
)

type EBPFManager struct {
    collection *ebpf.Collection
    links      []link.Link
    maps       map[string]*ebpf.Map
}

func NewEBPFManager() (*EBPFManager, error) {
    // Remove memory limit for eBPF
    if err := rlimit.RemoveMemlock(); err != nil {
        return nil, fmt.Errorf("failed to remove memory limit: %w", err)
    }

    // Load pre-compiled eBPF programs
    spec, err := ebpf.LoadCollectionSpec("proxy.o")
    if err != nil {
        return nil, fmt.Errorf("failed to load eBPF spec: %w", err)
    }

    collection, err := ebpf.NewCollection(spec)
    if err != nil {
        return nil, fmt.Errorf("failed to create eBPF collection: %w", err)
    }

    return &EBPFManager{
        collection: collection,
        maps:       make(map[string]*ebpf.Map),
    }, nil
}

func (m *EBPFManager) AttachXDP(ifaceName string) error {
    iface, err := net.InterfaceByName(ifaceName)
    if err != nil {
        return fmt.Errorf("failed to get interface %s: %w", ifaceName, err)
    }

    // Attach XDP program
    l, err := link.AttachXDP(link.XDPOptions{
        Program:   m.collection.Programs["xdp_rate_limit"],
        Interface: iface.Index,
        Flags:     link.XDPGenericMode, // Use generic mode for compatibility
    })
    if err != nil {
        return fmt.Errorf("failed to attach XDP program: %w", err)
    }

    m.links = append(m.links, l)
    return nil
}

func (m *EBPFManager) UpdateRateLimitConfig(config RateLimitConfig) error {
    configMap := m.collection.Maps["config_map"]

    key := uint32(0)
    value := struct {
        RateLimit  uint32
        BurstSize  uint32
        TimeWindow uint32
    }{
        RateLimit:  config.RateLimit,
        BurstSize:  config.BurstSize,
        TimeWindow: uint32(config.TimeWindow.Nanoseconds()),
    }

    return configMap.Update(key, value, ebpf.UpdateAny)
}

func (m *EBPFManager) GetRateLimitStats() (map[string]RateLimitStats, error) {
    rateLimitMap := m.collection.Maps["rate_limit_map"]

    stats := make(map[string]RateLimitStats)

    iter := rateLimitMap.Iterate()
    var key uint32
    var value struct {
        LastTime uint64
        Tokens   uint32
        Packets  uint32
    }

    for iter.Next(&key, &value) {
        ip := intToIP(key)
        stats[ip.String()] = RateLimitStats{
            LastTime: time.Unix(0, int64(value.LastTime)),
            Tokens:   value.Tokens,
            Packets:  value.Packets,
        }
    }

    return stats, iter.Err()
}
```

## Performance Optimization Strategies

### 1. Zero-Copy Networking

```go
// AF_XDP zero-copy implementation
type AFXDPSocket struct {
    socket   *xsk.Socket
    umem     *xsk.Umem
    fillRing *xsk.FillQueue
    compRing *xsk.CompQueue
    rxRing   *xsk.RxQueue
    txRing   *xsk.TxQueue
}

func NewAFXDPSocket(ifaceName string, queueID int) (*AFXDPSocket, error) {
    // Create UMEM for zero-copy buffers
    umem, err := xsk.NewUmem(xsk.UmemConfig{
        FrameCount: 4096,
        FrameSize:  2048,
        FillSize:   2048,
        CompSize:   1024,
    })
    if err != nil {
        return nil, err
    }

    // Create AF_XDP socket
    socket, err := xsk.NewSocket(ifaceName, queueID, umem, xsk.SocketConfig{
        RxSize: 2048,
        TxSize: 2048,
    })
    if err != nil {
        umem.Close()
        return nil, err
    }

    return &AFXDPSocket{
        socket:   socket,
        umem:     umem,
        fillRing: socket.FillQueue(),
        compRing: socket.CompQueue(),
        rxRing:   socket.RxQueue(),
        txRing:   socket.TxQueue(),
    }, nil
}

func (s *AFXDPSocket) ProcessPackets() error {
    for {
        // Receive packets
        n := s.rxRing.Receive()
        if n == 0 {
            continue
        }

        // Process packets in batch
        for i := 0; i < n; i++ {
            desc := s.rxRing.GetDescriptor(i)
            packet := s.umem.GetFrame(desc.Addr)

            // Process packet without copying
            if shouldForward(packet) {
                // Forward to TX ring
                s.txRing.Transmit(desc.Addr, desc.Len)
            }
        }

        // Release processed packets
        s.rxRing.Release(n)
    }
}
```

### 2. Lock-Free Data Structures

```go
// Lock-free ring buffer for high-performance queuing
type LockFreeRingBuffer struct {
    buffer   []unsafe.Pointer
    mask     uint64
    readPos  uint64
    writePos uint64
}

func NewLockFreeRingBuffer(size uint64) *LockFreeRingBuffer {
    // Ensure size is power of 2
    if size&(size-1) != 0 {
        panic("size must be power of 2")
    }

    return &LockFreeRingBuffer{
        buffer: make([]unsafe.Pointer, size),
        mask:   size - 1,
    }
}

func (rb *LockFreeRingBuffer) Enqueue(item unsafe.Pointer) bool {
    writePos := atomic.LoadUint64(&rb.writePos)
    readPos := atomic.LoadUint64(&rb.readPos)

    // Check if buffer is full
    if writePos-readPos >= uint64(len(rb.buffer)) {
        return false
    }

    // Try to reserve slot
    if !atomic.CompareAndSwapUint64(&rb.writePos, writePos, writePos+1) {
        return false
    }

    // Write item
    rb.buffer[writePos&rb.mask] = item
    return true
}

func (rb *LockFreeRingBuffer) Dequeue() unsafe.Pointer {
    readPos := atomic.LoadUint64(&rb.readPos)
    writePos := atomic.LoadUint64(&rb.writePos)

    // Check if buffer is empty
    if readPos >= writePos {
        return nil
    }

    // Try to reserve slot
    if !atomic.CompareAndSwapUint64(&rb.readPos, readPos, readPos+1) {
        return nil
    }

    // Read item
    item := rb.buffer[readPos&rb.mask]
    rb.buffer[readPos&rb.mask] = nil // Clear for GC
    return item
}
```

### 3. Memory Pool Management

```go
// Memory pool for reducing GC pressure
type MemoryPool struct {
    pools map[int]*sync.Pool
    mu    sync.RWMutex
}

func NewMemoryPool() *MemoryPool {
    return &MemoryPool{
        pools: make(map[int]*sync.Pool),
    }
}

func (mp *MemoryPool) Get(size int) []byte {
    // Round up to next power of 2
    poolSize := nextPowerOf2(size)

    mp.mu.RLock()
    pool, exists := mp.pools[poolSize]
    mp.mu.RUnlock()

    if !exists {
        mp.mu.Lock()
        if pool, exists = mp.pools[poolSize]; !exists {
            pool = &sync.Pool{
                New: func() interface{} {
                    return make([]byte, poolSize)
                },
            }
            mp.pools[poolSize] = pool
        }
        mp.mu.Unlock()
    }

    buf := pool.Get().([]byte)
    return buf[:size] // Slice to requested size
}

func (mp *MemoryPool) Put(buf []byte) {
    poolSize := cap(buf)

    mp.mu.RLock()
    pool, exists := mp.pools[poolSize]
    mp.mu.RUnlock()

    if exists {
        // Clear buffer before returning to pool
        for i := range buf {
            buf[i] = 0
        }
        pool.Put(buf[:cap(buf)])
    }
}

func nextPowerOf2(n int) int {
    n--
    n |= n >> 1
    n |= n >> 2
    n |= n >> 4
    n |= n >> 8
    n |= n >> 16
    n++
    return n
}
```

## Configuration Management

### Hot Configuration Reloading

```go
type ConfigManager struct {
    config     atomic.Value // stores *Config
    watchers   []ConfigWatcher
    mu         sync.RWMutex
    updateChan chan *Config
}

type ConfigWatcher interface {
    OnConfigUpdate(old, new *Config) error
}

func (cm *ConfigManager) UpdateConfig(newConfig *Config) error {
    oldConfig := cm.GetConfig()

    // Validate new configuration
    if err := newConfig.Validate(); err != nil {
        return fmt.Errorf("invalid configuration: %w", err)
    }

    // Notify watchers before update
    cm.mu.RLock()
    for _, watcher := range cm.watchers {
        if err := watcher.OnConfigUpdate(oldConfig, newConfig); err != nil {
            cm.mu.RUnlock()
            return fmt.Errorf("watcher rejected config update: %w", err)
        }
    }
    cm.mu.RUnlock()

    // Atomically update configuration
    cm.config.Store(newConfig)

    // Notify async processors
    select {
    case cm.updateChan <- newConfig:
    default:
        // Channel full, skip notification
    }

    return nil
}

func (cm *ConfigManager) GetConfig() *Config {
    return cm.config.Load().(*Config)
}

func (cm *ConfigManager) RegisterWatcher(watcher ConfigWatcher) {
    cm.mu.Lock()
    cm.watchers = append(cm.watchers, watcher)
    cm.mu.Unlock()
}
```

## Error Handling and Resilience

### Circuit Breaker Implementation

```go
type CircuitBreaker struct {
    name           string
    maxFailures    int
    resetTimeout   time.Duration
    state          int32 // 0=closed, 1=open, 2=half-open
    failures       int32
    lastFailTime   int64
    mutex          sync.RWMutex
    onStateChange  func(name string, from, to State)
}

const (
    StateClosed State = iota
    StateOpen
    StateHalfOpen
)

func (cb *CircuitBreaker) Call(fn func() (interface{}, error)) (interface{}, error) {
    state := cb.getState()

    switch state {
    case StateOpen:
        if cb.canAttemptReset() {
            cb.setState(StateHalfOpen)
        } else {
            return nil, ErrCircuitBreakerOpen
        }
    case StateHalfOpen:
        // Allow limited calls in half-open state
    case StateClosed:
        // Normal operation
    }

    result, err := fn()

    if err != nil {
        cb.recordFailure()
        return nil, err
    }

    cb.recordSuccess()
    return result, nil
}

func (cb *CircuitBreaker) recordFailure() {
    failures := atomic.AddInt32(&cb.failures, 1)
    atomic.StoreInt64(&cb.lastFailTime, time.Now().UnixNano())

    if failures >= int32(cb.maxFailures) {
        cb.setState(StateOpen)
    }
}

func (cb *CircuitBreaker) recordSuccess() {
    atomic.StoreInt32(&cb.failures, 0)
    if cb.getState() == StateHalfOpen {
        cb.setState(StateClosed)
    }
}
```

This completes the comprehensive technical architecture documentation for MarchProxy, covering all major design patterns, implementation details, and performance optimizations.