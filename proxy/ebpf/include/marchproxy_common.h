// MarchProxy eBPF Common Definitions
// Shared constants and structures between eBPF program and userspace

#ifndef MARCHPROXY_COMMON_H
#define MARCHPROXY_COMMON_H

// Protocol constants
#define PROTO_TCP  1
#define PROTO_UDP  2
#define PROTO_ICMP 4

// Action constants
#define ACTION_DROP     0
#define ACTION_FORWARD  1  
#define ACTION_FALLBACK 2

// Authentication types
#define AUTH_TYPE_NONE   0
#define AUTH_TYPE_BASE64 1
#define AUTH_TYPE_JWT    2

// Map size limits
#define MAX_SERVICES 1024
#define MAX_MAPPINGS 512
#define MAX_PORTS    16
#define MAX_CONNECTIONS 65536

// Performance tuning
#define EBPF_PROG_TYPE_TC    1
#define EBPF_PROG_TYPE_XDP   2
#define EBPF_PROG_TYPE_CGROUP 3

// Map names for userspace access
#define SERVICES_MAP_NAME    "services_map"
#define MAPPINGS_MAP_NAME    "mappings_map"  
#define CONNECTIONS_MAP_NAME "connections_map"
#define STATS_MAP_NAME       "stats_map"

#ifndef __KERNEL__
// Userspace only definitions
#include <stdint.h>

typedef uint8_t  __u8;
typedef uint16_t __u16;
typedef uint32_t __u32;
typedef uint64_t __u64;
typedef uint32_t __be32;

#endif // __KERNEL__

// Service structure (must match eBPF program)
struct service {
    __u32 id;
    __be32 ip_addr;        // Network byte order
    __u16 port;            // Host byte order
    __u8 auth_required;    // 0 = no auth, 1 = auth required
    __u8 auth_type;        // AUTH_TYPE_* constants
    __u32 flags;           // Additional service flags
};

// Mapping structure (must match eBPF program)
struct mapping {
    __u32 id;
    __u32 source_services[MAX_PORTS];  // Source service IDs
    __u32 dest_services[MAX_PORTS];    // Destination service IDs
    __u16 ports[MAX_PORTS];            // Allowed ports
    __u8 protocols;                    // Bitmask: PROTO_* constants
    __u8 auth_required;                // Authentication requirement
    __u8 priority;                     // Routing priority (higher = preferred)
    __u8 port_count;                   // Number of valid ports
    __u8 src_count;                    // Number of source services
    __u8 dest_count;                   // Number of dest services
};

// Connection tracking key (must match eBPF program)
struct connection_key {
    __be32 src_ip;
    __be32 dst_ip;
    __u16 src_port;
    __u16 dst_port;
    __u8 protocol;
};

// Connection tracking value (must match eBPF program)
struct connection_value {
    __u64 packets;
    __u64 bytes;
    __u64 timestamp;
    __u32 service_id;
    __u8 authenticated;
};

// Statistics structure (must match eBPF program)
struct proxy_stats {
    __u64 total_packets;
    __u64 total_bytes;
    __u64 tcp_packets;
    __u64 udp_packets;
    __u64 icmp_packets;
    __u64 dropped_packets;
    __u64 forwarded_packets;
    __u64 auth_required;
    __u64 fallback_to_userspace;
};

#endif // MARCHPROXY_COMMON_H