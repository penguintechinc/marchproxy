# Proxy Configuration Guide

This guide covers comprehensive configuration options for the MarchProxy Proxy component.

## Overview

The MarchProxy Proxy is a high-performance Go application with eBPF integration that handles:

- Multi-protocol traffic proxying (TCP, UDP, ICMP, HTTP/HTTPS, WebSocket, QUIC)
- Advanced performance acceleration (eBPF, XDP, AF_XDP, DPDK)
- Authentication and authorization
- Rate limiting and security features
- Real-time configuration synchronization with Manager

## Configuration File Structure

The proxy uses a YAML configuration file with the following sections:

```yaml
# /etc/marchproxy/proxy.yaml
manager:         # Manager connection settings
server:          # Proxy server configuration
performance:     # Performance and acceleration settings
network:         # Network interface configuration
security:        # Security and authentication settings
logging:         # Logging configuration
monitoring:      # Health checks and metrics
protocols:       # Protocol-specific settings
cache:           # Caching configuration
circuit_breaker: # Circuit breaker settings
rate_limiting:   # Rate limiting configuration
```

## Manager Connection

### Basic Manager Connection

```yaml
manager:
  # Manager server details
  url: "http://localhost:8000"           # Manager URL
  api_key: "${CLUSTER_API_KEY}"          # Cluster API key from environment
  cluster_id: 1                          # Cluster ID (auto-assigned during registration)

  # Connection settings
  timeout: 30s                           # Request timeout
  retry_attempts: 3                      # Retry attempts on failure
  retry_interval: 10s                    # Delay between retries
  retry_backoff: 2.0                     # Exponential backoff multiplier

  # Configuration sync
  config_refresh_interval: 60s           # Configuration refresh interval
  config_refresh_jitter: 30s             # Random jitter for refresh timing
  force_refresh_on_error: true           # Force refresh on configuration errors
```

### Advanced Manager Connection

```yaml
manager:
  # High availability
  endpoints:                             # Multiple manager endpoints
    - "https://manager-1.company.com:8000"
    - "https://manager-2.company.com:8000"
    - "https://manager-3.company.com:8000"

  failover:
    enabled: true                        # Enable automatic failover
    health_check_interval: 30s           # Health check interval
    failure_threshold: 3                 # Failures before failover
    recovery_threshold: 2                # Successes before recovery

  # TLS configuration for manager connection
  tls:
    enabled: true                        # Enable TLS for manager communication
    verify_certificate: true             # Verify manager certificate
    ca_file: "/etc/ssl/certs/ca.crt"     # CA certificate file
    cert_file: "/etc/ssl/certs/proxy.crt" # Client certificate (optional)
    key_file: "/etc/ssl/private/proxy.key" # Client private key (optional)

  # Authentication
  auth:
    type: "api_key"                      # Authentication type (api_key, jwt)
    api_key: "${CLUSTER_API_KEY}"        # API key
    api_key_header: "X-Cluster-API-Key"  # API key header name
```

## Server Configuration

### Basic Server Settings

```yaml
server:
  # Port configuration
  proxy_port: 8080                       # Main proxy port
  metrics_port: 8081                     # Metrics endpoint port
  admin_port: 8082                       # Admin interface port (optional)
  health_port: 8083                      # Health check port (optional)

  # Process configuration
  worker_threads: 0                      # Worker threads (0 = auto-detect)
  max_connections: 10000                 # Maximum concurrent connections
  connection_timeout: 300s               # Connection timeout

  # Buffer settings
  buffer_size: 65536                     # Network buffer size (64KB)
  read_buffer_size: 32768               # Read buffer size (32KB)
  write_buffer_size: 32768              # Write buffer size (32KB)

  # Graceful shutdown
  shutdown_timeout: 30s                 # Graceful shutdown timeout
  drain_timeout: 10s                    # Connection drain timeout
```

### Advanced Server Settings

```yaml
server:
  # Performance tuning
  tcp_nodelay: true                      # Disable Nagle's algorithm
  tcp_fastopen: true                     # Enable TCP Fast Open
  reuse_port: true                       # Enable SO_REUSEPORT
  keepalive_enabled: true                # Enable TCP keepalive
  keepalive_idle: 600s                   # Keepalive idle time
  keepalive_interval: 60s                # Keepalive probe interval
  keepalive_count: 3                     # Keepalive probe count

  # Connection pooling
  connection_pool:
    enabled: true                        # Enable connection pooling
    max_idle_connections: 100            # Maximum idle connections per host
    max_connections_per_host: 200        # Maximum connections per host
    idle_timeout: 300s                   # Idle connection timeout
    max_lifetime: 3600s                  # Maximum connection lifetime

  # Request/response limits
  max_request_size: "10MB"               # Maximum request size
  max_response_size: "100MB"             # Maximum response size
  header_timeout: 10s                    # Header read timeout
  body_timeout: 30s                      # Body read timeout
```

## Performance Configuration

### eBPF Acceleration

```yaml
performance:
  # eBPF configuration
  enable_ebpf: true                      # Enable eBPF acceleration
  ebpf_program_path: "/opt/marchproxy/ebpf/proxy.o"  # eBPF program path
  ebpf_map_size: 65536                   # eBPF map size
  ebpf_log_level: 1                      # eBPF verifier log level (0-4)

  # eBPF features
  ebpf_features:
    packet_filtering: true               # Enable packet filtering
    connection_tracking: true            # Enable connection tracking
    load_balancing: true                 # Enable eBPF load balancing
    rate_limiting: true                  # Enable eBPF rate limiting
    ddos_protection: true                # Enable DDoS protection
```

### XDP Acceleration

```yaml
performance:
  # XDP configuration
  enable_xdp: true                       # Enable XDP acceleration
  xdp_mode: "native"                     # XDP mode (native, generic, offload)
  xdp_program_path: "/opt/marchproxy/ebpf/xdp.o"  # XDP program path
  xdp_flags: ["DRV_MODE"]                # XDP flags

  # XDP features
  xdp_features:
    early_packet_drop: true              # Drop packets at driver level
    zero_copy_rx: true                   # Zero-copy receive
    zero_copy_tx: true                   # Zero-copy transmit
    batch_processing: true               # Batch packet processing

  # AF_XDP configuration
  enable_af_xdp: false                   # Enable AF_XDP sockets
  af_xdp_queue_size: 2048               # AF_XDP queue size
  af_xdp_frame_size: 2048               # AF_XDP frame size
  af_xdp_completion_size: 1024          # Completion queue size
```

### DPDK Configuration

```yaml
performance:
  # DPDK configuration (Enterprise)
  enable_dpdk: false                     # Enable DPDK acceleration
  dpdk_memory: "1024MB"                  # DPDK memory allocation
  dpdk_cores: "2-7"                      # CPU cores for DPDK
  dpdk_ports: ["0000:01:00.0"]          # PCIe addresses of network interfaces

  # DPDK parameters
  dpdk_params:
    huge_pages: "1GB"                    # Huge page size
    iova_mode: "pa"                      # IOVA mode (pa, va)
    proc_type: "primary"                 # Process type
    log_level: "info"                    # DPDK log level

  # SR-IOV configuration
  sriov:
    enabled: false                       # Enable SR-IOV
    num_vfs: 8                          # Number of Virtual Functions
    vf_driver: "vfio-pci"               # VF driver
```

### CPU and Memory Optimization

```yaml
performance:
  # CPU configuration
  cpu_affinity: "auto"                   # CPU affinity (auto, cores list)
  numa_node: 0                          # Preferred NUMA node
  cpu_isolation: false                   # Isolate CPU cores

  # Memory configuration
  huge_pages:
    enabled: false                       # Enable huge pages
    size: "2MB"                         # Huge page size (2MB, 1GB)
    count: 1024                         # Number of huge pages

  # Threading
  thread_pool:
    min_threads: 4                      # Minimum worker threads
    max_threads: 32                     # Maximum worker threads
    stack_size: "8MB"                   # Thread stack size
    idle_timeout: 60s                   # Idle thread timeout
```

## Network Configuration

### Network Interface Settings

```yaml
network:
  # Primary interface
  interface: "eth0"                      # Primary network interface
  bind_to_device: false                  # Bind socket to specific device

  # IP configuration
  bind_ip: "0.0.0.0"                    # IP address to bind
  external_ip: ""                       # External IP (auto-detect if empty)

  # Advanced interface settings
  interface_settings:
    mtu: 9000                           # Maximum Transmission Unit
    queue_size: 4096                    # Network queue size
    interrupt_coalescing: true          # Enable interrupt coalescing
    receive_scaling: true               # Enable Receive Side Scaling (RSS)
    transmit_scaling: true              # Enable Transmit Side Scaling (XPS)

  # Quality of Service
  qos:
    enabled: false                      # Enable QoS
    traffic_class: 0                    # Traffic class (0-7)
    dscp_marking: 0                     # DSCP marking (0-63)
```

### Load Balancing Configuration

```yaml
network:
  # Load balancing
  load_balancing:
    algorithm: "round_robin"            # Algorithm (round_robin, least_conn, ip_hash, weighted)
    health_check: true                  # Enable health checks
    health_check_interval: 30s          # Health check interval
    health_check_timeout: 5s            # Health check timeout
    health_check_retries: 3             # Health check retries

    # Failover settings
    failover:
      enabled: true                     # Enable automatic failover
      detection_threshold: 3            # Failure detection threshold
      recovery_threshold: 2             # Recovery detection threshold
      blacklist_duration: 300s          # Failed backend blacklist duration
```

## Security Configuration

### Authentication Settings

```yaml
security:
  # Authentication methods
  auth_methods: ["jwt", "api_key"]       # Supported auth methods

  # JWT configuration
  jwt:
    verify_signature: true              # Verify JWT signature
    verify_expiry: true                 # Verify JWT expiry
    clock_skew: 60s                     # Clock skew tolerance
    algorithms: ["HS256", "RS256"]      # Accepted algorithms

  # API key configuration
  api_key:
    header_name: "X-API-Key"            # API key header name
    query_param: "api_key"              # API key query parameter
    cache_ttl: 300s                     # API key cache TTL

  # Token validation
  token_validation:
    cache_enabled: true                 # Enable token cache
    cache_size: 10000                   # Token cache size
    cache_ttl: 300s                     # Token cache TTL
    validation_timeout: 5s              # Token validation timeout
```

### Rate Limiting

```yaml
security:
  # Rate limiting configuration
  rate_limiting:
    enabled: true                       # Enable rate limiting
    algorithm: "token_bucket"           # Algorithm (token_bucket, sliding_window)

    # Global rate limits
    global:
      requests_per_second: 1000         # Global RPS limit
      burst_size: 2000                  # Burst capacity

    # Per-IP rate limits
    per_ip:
      requests_per_second: 100          # Per-IP RPS limit
      burst_size: 200                   # Per-IP burst capacity
      window_size: 60s                  # Rate limiting window

    # Per-user rate limits
    per_user:
      requests_per_second: 200          # Per-user RPS limit
      burst_size: 400                   # Per-user burst capacity

    # XDP rate limiting (Enterprise)
    xdp:
      enabled: false                    # Enable XDP-based rate limiting
      pps_limit: 1000000               # Packets per second limit
      burst_packets: 2000000           # Burst packet limit
      time_window: 1s                   # Rate limiting window
```

### Web Application Firewall

```yaml
security:
  # WAF configuration
  waf:
    enabled: true                       # Enable WAF
    mode: "blocking"                    # Mode (monitoring, blocking)

    # SQL injection protection
    sql_injection:
      enabled: true                     # Enable SQL injection protection
      patterns_file: "/etc/marchproxy/sql_patterns.txt"
      block_threshold: 5                # Block threshold

    # XSS protection
    xss:
      enabled: true                     # Enable XSS protection
      patterns_file: "/etc/marchproxy/xss_patterns.txt"
      sanitize_responses: true          # Sanitize responses

    # Command injection protection
    command_injection:
      enabled: true                     # Enable command injection protection
      patterns_file: "/etc/marchproxy/cmd_patterns.txt"

    # Custom rules
    custom_rules:
      - name: "Block admin paths"
        pattern: "/admin/*"
        action: "block"
        enabled: true
```

### DDoS Protection

```yaml
security:
  # DDoS protection
  ddos_protection:
    enabled: true                       # Enable DDoS protection

    # Connection limits
    connection_limits:
      max_connections_per_ip: 100       # Max connections per IP
      max_new_connections_per_second: 10 # Max new connections per second per IP
      connection_timeout: 30s           # Connection timeout

    # Packet filtering
    packet_filtering:
      drop_invalid_packets: true        # Drop malformed packets
      drop_fragmented_packets: false    # Drop fragmented packets
      max_packet_size: 65536           # Maximum packet size

    # SYN flood protection
    syn_flood:
      enabled: true                     # Enable SYN flood protection
      threshold: 100                    # SYN packet threshold
      window: 10s                       # Time window
```

## Protocol Configuration

### HTTP/HTTPS Settings

```yaml
protocols:
  http:
    # HTTP configuration
    enabled: true                       # Enable HTTP support
    max_header_size: 8192              # Maximum header size
    max_body_size: "100MB"             # Maximum body size
    request_timeout: 30s               # Request timeout

    # HTTP/2 support
    http2:
      enabled: true                     # Enable HTTP/2
      max_concurrent_streams: 1000      # Max concurrent streams
      initial_window_size: 65536        # Initial window size

    # Compression
    compression:
      enabled: true                     # Enable compression
      algorithms: ["gzip", "deflate", "br", "zstd"]  # Supported algorithms
      min_size: 1024                    # Minimum size to compress
      level: 6                          # Compression level (1-9)

  https:
    # HTTPS configuration
    enabled: true                       # Enable HTTPS support

    # TLS settings
    tls:
      min_version: "1.2"                # Minimum TLS version
      max_version: "1.3"                # Maximum TLS version
      ciphers: "ECDHE-RSA-AES256-GCM-SHA384:ECDHE-RSA-AES128-GCM-SHA256"

    # Certificate management
    certificates:
      source: "manager"                 # Certificate source (manager, file, vault)
      auto_reload: true                 # Auto-reload certificates
      reload_interval: 3600s            # Certificate reload interval
```

### WebSocket Settings

```yaml
protocols:
  websocket:
    # WebSocket configuration
    enabled: true                       # Enable WebSocket support
    max_message_size: "10MB"           # Maximum message size
    ping_interval: 30s                 # Ping interval
    pong_timeout: 10s                  # Pong timeout

    # Connection settings
    handshake_timeout: 10s             # Handshake timeout
    read_buffer_size: 4096             # Read buffer size
    write_buffer_size: 4096            # Write buffer size

    # Extensions
    extensions:
      compression: true                 # Enable per-message compression
      auto_fragment: true               # Enable auto-fragmentation
```

### QUIC/HTTP3 Settings

```yaml
protocols:
  quic:
    # QUIC configuration
    enabled: false                      # Enable QUIC support
    port: 8443                         # QUIC port

    # Connection settings
    max_idle_timeout: 300s             # Maximum idle timeout
    max_stream_data: "10MB"            # Maximum stream data
    max_connection_data: "100MB"       # Maximum connection data
    max_streams: 100                   # Maximum concurrent streams

    # Performance settings
    initial_rtt: 100ms                 # Initial RTT estimate
    congestion_control: "cubic"        # Congestion control algorithm
```

## Logging Configuration

### Basic Logging

```yaml
logging:
  # Log level (trace, debug, info, warn, error, fatal)
  level: "info"

  # Log format (text, json)
  format: "json"

  # Log destinations
  file: "/var/log/marchproxy/proxy.log"  # Log file path
  console: true                          # Enable console output
  syslog: false                         # Enable syslog output

  # File rotation
  max_size: "100MB"                     # Maximum log file size
  max_files: 10                         # Number of rotated files
  max_age: 30                           # Maximum age in days
  compress: true                        # Compress rotated files
```

### Advanced Logging

```yaml
logging:
  # Component-specific log levels
  loggers:
    "proxy.auth": "debug"               # Authentication debug logs
    "proxy.ebpf": "info"                # eBPF logs
    "proxy.xdp": "info"                 # XDP logs
    "proxy.network": "warn"             # Network logs
    "proxy.health": "error"             # Health check logs

  # Access logging
  access_log:
    enabled: true                       # Enable access logging
    file: "/var/log/marchproxy/access.log"  # Access log file
    format: "combined"                  # Log format (combined, common, custom)
    include_request_body: false         # Include request body
    include_response_body: false        # Include response body

  # Audit logging
  audit_log:
    enabled: true                       # Enable audit logging
    file: "/var/log/marchproxy/audit.log"   # Audit log file
    events:                             # Events to audit
      - "authentication_success"
      - "authentication_failure"
      - "rate_limit_exceeded"
      - "security_violation"

  # Performance logging
  performance_log:
    enabled: false                      # Enable performance logging
    file: "/var/log/marchproxy/performance.log"
    metrics:                            # Metrics to log
      - "request_duration"
      - "queue_depth"
      - "connection_count"
      - "memory_usage"
```

### Centralized Logging

```yaml
logging:
  # Syslog configuration
  syslog:
    enabled: true                       # Enable syslog
    network: "udp"                      # Protocol (udp, tcp)
    address: "syslog.company.com:514"   # Syslog server address
    facility: "local0"                  # Syslog facility
    tag: "marchproxy-proxy"             # Syslog tag

  # Structured logging
  structured:
    enabled: true                       # Enable structured logging
    fields:                             # Additional fields
      service: "marchproxy-proxy"
      environment: "production"
      cluster_id: "${CLUSTER_ID}"
      node_id: "${NODE_ID}"
```

## Monitoring Configuration

### Health Checks

```yaml
monitoring:
  # Health check configuration
  health:
    enabled: true                       # Enable health checks
    port: 8083                         # Health check port
    path: "/healthz"                   # Health check path

    # Health check components
    checks:
      manager_connectivity: true        # Check manager connectivity
      ebpf_programs: true               # Check eBPF program status
      network_interfaces: true         # Check network interface status
      memory_usage: true                # Check memory usage
      cpu_usage: true                   # Check CPU usage
      disk_space: true                  # Check disk space

    # Health check thresholds
    thresholds:
      manager_timeout: 5s               # Manager connectivity timeout
      memory_threshold: 90              # Memory usage threshold (%)
      cpu_threshold: 95                 # CPU usage threshold (%)
      disk_threshold: 90                # Disk usage threshold (%)
```

### Metrics Collection

```yaml
monitoring:
  # Metrics configuration
  metrics:
    enabled: true                       # Enable metrics collection
    port: 8081                         # Metrics port
    path: "/metrics"                   # Metrics path
    format: "prometheus"               # Metrics format

    # Collection settings
    interval: 15s                      # Collection interval
    retention: 300s                    # Metrics retention

    # Custom metrics
    custom_metrics:
      - name: "proxy_connections_active"
        type: "gauge"
        description: "Number of active connections"
      - name: "proxy_requests_total"
        type: "counter"
        description: "Total number of requests"
      - name: "proxy_request_duration"
        type: "histogram"
        description: "Request duration in seconds"
```

## Cache Configuration

### Memory Cache

```yaml
cache:
  # Cache configuration
  enabled: true                         # Enable caching
  type: "memory"                        # Cache type (memory, redis)

  # Memory cache settings
  memory:
    max_size: "1GB"                     # Maximum cache size
    max_entries: 100000                 # Maximum number of entries
    ttl: 300s                          # Default TTL
    cleanup_interval: 60s              # Cleanup interval

    # Eviction policy
    eviction_policy: "lru"              # Eviction policy (lru, lfu, fifo)
    eviction_threshold: 0.9             # Eviction threshold (90%)

  # Cache keys
  key_patterns:
    - pattern: "/api/*"                 # API responses
      ttl: 60s
    - pattern: "/static/*"              # Static resources
      ttl: 3600s
```

### Redis Cache

```yaml
cache:
  # Redis cache settings
  redis:
    enabled: false                      # Enable Redis cache
    addresses: ["redis:6379"]           # Redis server addresses
    password: ""                        # Redis password
    database: 0                         # Redis database number

    # Connection pool
    pool_size: 10                       # Connection pool size
    min_idle_connections: 2             # Minimum idle connections
    dial_timeout: 5s                   # Connection timeout
    read_timeout: 3s                   # Read timeout
    write_timeout: 3s                  # Write timeout

    # Cluster settings
    cluster:
      enabled: false                    # Enable cluster mode
      max_redirects: 3                  # Maximum redirects
      read_only: false                  # Read-only mode
```

## Circuit Breaker Configuration

```yaml
circuit_breaker:
  # Circuit breaker settings
  enabled: true                         # Enable circuit breaker

  # Failure detection
  failure_threshold: 5                  # Failure threshold
  success_threshold: 3                  # Success threshold for recovery
  timeout: 30s                         # Request timeout
  reset_timeout: 60s                   # Circuit reset timeout

  # Monitoring
  monitor_interval: 10s                # Monitoring interval
  half_open_max_requests: 10           # Max requests in half-open state

  # Fallback configuration
  fallback:
    enabled: true                       # Enable fallback responses
    response_code: 503                  # Fallback HTTP status code
    response_body: "Service temporarily unavailable"
    content_type: "text/plain"
```

## Environment Variables Override

All configuration options can be overridden using environment variables:

```bash
# Manager connection
export PROXY_MANAGER_URL="http://manager:8000"
export CLUSTER_API_KEY="your-cluster-api-key"

# Server configuration
export PROXY_SERVER_PROXY_PORT="8080"
export PROXY_SERVER_METRICS_PORT="8081"
export PROXY_SERVER_WORKER_THREADS="8"

# Performance settings
export PROXY_PERFORMANCE_ENABLE_EBPF="true"
export PROXY_PERFORMANCE_ENABLE_XDP="true"
export PROXY_PERFORMANCE_ENABLE_DPDK="false"

# Network configuration
export PROXY_NETWORK_INTERFACE="eth0"
export PROXY_NETWORK_BIND_IP="0.0.0.0"

# Security settings
export PROXY_SECURITY_RATE_LIMITING_ENABLED="true"
export PROXY_SECURITY_WAF_ENABLED="true"

# Logging configuration
export PROXY_LOGGING_LEVEL="info"
export PROXY_LOGGING_FORMAT="json"
```

## Configuration Examples

### High-Performance Configuration

```yaml
# high-performance.yaml - Maximum performance configuration
manager:
  url: "http://manager:8000"
  config_refresh_interval: 30s

server:
  proxy_port: 8080
  worker_threads: 0                     # Auto-detect
  max_connections: 100000
  buffer_size: 131072                   # 128KB buffers

performance:
  enable_ebpf: true
  enable_xdp: true
  xdp_mode: "native"
  enable_af_xdp: true

  cpu_affinity: "auto"
  numa_node: 0

  huge_pages:
    enabled: true
    size: "2MB"
    count: 2048

network:
  interface: "eth0"
  interface_settings:
    mtu: 9000
    queue_size: 8192
    interrupt_coalescing: true
    receive_scaling: true

security:
  rate_limiting:
    enabled: true
    algorithm: "token_bucket"
    global:
      requests_per_second: 100000

logging:
  level: "warn"
  format: "json"
  file: "/var/log/marchproxy/proxy.log"

monitoring:
  metrics:
    enabled: true
    interval: 5s
```

### Security-Focused Configuration

```yaml
# security.yaml - Security-focused configuration
security:
  auth_methods: ["jwt"]

  jwt:
    verify_signature: true
    verify_expiry: true
    clock_skew: 30s

  rate_limiting:
    enabled: true
    per_ip:
      requests_per_second: 10
      burst_size: 20

  waf:
    enabled: true
    mode: "blocking"
    sql_injection:
      enabled: true
      block_threshold: 1
    xss:
      enabled: true
      sanitize_responses: true
    command_injection:
      enabled: true

  ddos_protection:
    enabled: true
    connection_limits:
      max_connections_per_ip: 10
      max_new_connections_per_second: 2

protocols:
  https:
    tls:
      min_version: "1.3"
      ciphers: "TLS_AES_256_GCM_SHA384:TLS_CHACHA20_POLY1305_SHA256"

logging:
  level: "info"
  audit_log:
    enabled: true
    events:
      - "authentication_success"
      - "authentication_failure"
      - "rate_limit_exceeded"
      - "security_violation"
      - "suspicious_activity"
```

This completes the comprehensive Proxy configuration documentation.