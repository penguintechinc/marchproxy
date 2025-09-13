// MarchProxy eBPF packet filtering program
// High-performance packet filtering and forwarding for MarchProxy

#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/tcp.h>
#include <linux/udp.h>
#include <linux/icmp.h>
#include <linux/in.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

#define MAX_SERVICES 1024
#define MAX_MAPPINGS 512
#define MAX_PORTS 16

// Service definition structure
struct service {
    __u32 id;
    __be32 ip_addr;        // Network byte order
    __u16 port;            // Host byte order
    __u8 auth_required;    // 0 = no auth, 1 = auth required
    __u8 auth_type;        // 0 = none, 1 = base64, 2 = jwt
    __u32 flags;           // Additional service flags
};

// Mapping definition structure  
struct mapping {
    __u32 id;
    __u32 source_services[MAX_PORTS];  // Source service IDs
    __u32 dest_services[MAX_PORTS];    // Destination service IDs
    __u16 ports[MAX_PORTS];            // Allowed ports
    __u8 protocols;                    // Bitmask: 1=TCP, 2=UDP, 4=ICMP
    __u8 auth_required;                // Authentication requirement
    __u8 priority;                     // Routing priority (higher = preferred)
    __u8 port_count;                   // Number of valid ports
    __u8 src_count;                    // Number of source services
    __u8 dest_count;                   // Number of dest services
};

// Connection tracking structure
struct connection_key {
    __be32 src_ip;
    __be32 dst_ip;
    __u16 src_port;
    __u16 dst_port;
    __u8 protocol;
};

struct connection_value {
    __u64 packets;
    __u64 bytes;
    __u64 timestamp;
    __u32 service_id;
    __u8 authenticated;
};

// Statistics structure
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

// eBPF Maps

// Service configuration map
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, MAX_SERVICES);
    __type(key, __u32);         // service ID
    __type(value, struct service);
} services_map SEC(".maps");

// Mapping configuration map
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, MAX_MAPPINGS);
    __type(key, __u32);         // mapping ID
    __type(value, struct mapping);
} mappings_map SEC(".maps");

// Connection tracking map
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 65536);
    __type(key, struct connection_key);
    __type(value, struct connection_value);
} connections_map SEC(".maps");

// Statistics map
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 1);
    __type(key, __u32);
    __type(value, struct proxy_stats);
} stats_map SEC(".maps");

// Helper function to parse Ethernet header
static __always_inline int parse_eth_hdr(struct __sk_buff *skb, __u16 *eth_proto) {
    void *data_end = (void *)(long)skb->data_end;
    void *data = (void *)(long)skb->data;
    struct ethhdr *eth = data;

    if ((void *)(eth + 1) > data_end)
        return -1;

    *eth_proto = bpf_ntohs(eth->h_proto);
    return sizeof(struct ethhdr);
}

// Helper function to parse IP header
static __always_inline int parse_ip_hdr(struct __sk_buff *skb, int offset, 
                                       struct iphdr **ip_hdr) {
    void *data_end = (void *)(long)skb->data_end;
    void *data = (void *)(long)skb->data;
    struct iphdr *ip = data + offset;

    if ((void *)(ip + 1) > data_end)
        return -1;

    // Verify IP version and header length
    if (ip->version != 4)
        return -1;

    if (ip->ihl < 5)
        return -1;

    *ip_hdr = ip;
    return ip->ihl * 4;
}

// Helper function to find matching mapping
static __always_inline struct mapping *find_mapping(__be32 dst_ip, __u16 dst_port, __u8 protocol) {
    struct mapping *mapping;
    __u32 map_id;

    // Iterate through mappings (simplified - in production, use more efficient lookup)
    for (map_id = 1; map_id <= MAX_MAPPINGS; map_id++) {
        mapping = bpf_map_lookup_elem(&mappings_map, &map_id);
        if (!mapping)
            continue;

        // Check protocol match
        __u8 proto_mask = 0;
        if (protocol == IPPROTO_TCP) proto_mask = 1;
        else if (protocol == IPPROTO_UDP) proto_mask = 2;
        else if (protocol == IPPROTO_ICMP) proto_mask = 4;

        if (!(mapping->protocols & proto_mask))
            continue;

        // Check port match
        int i;
        int port_matched = 0;
        for (i = 0; i < mapping->port_count && i < MAX_PORTS; i++) {
            if (mapping->ports[i] == dst_port) {
                port_matched = 1;
                break;
            }
        }

        if (port_matched)
            return mapping;
    }

    return NULL;
}

// Helper function to find destination service
static __always_inline struct service *find_dest_service(struct mapping *mapping) {
    if (!mapping || mapping->dest_count == 0)
        return NULL;

    // Simple round-robin selection (use hash or other algorithm in production)
    __u32 service_id = mapping->dest_services[0];
    return bpf_map_lookup_elem(&services_map, &service_id);
}

// Update statistics
static __always_inline void update_stats(__u64 bytes, __u8 protocol, int action) {
    __u32 key = 0;
    struct proxy_stats *stats = bpf_map_lookup_elem(&stats_map, &key);
    
    if (!stats)
        return;

    __sync_fetch_and_add(&stats->total_packets, 1);
    __sync_fetch_and_add(&stats->total_bytes, bytes);

    if (protocol == IPPROTO_TCP)
        __sync_fetch_and_add(&stats->tcp_packets, 1);
    else if (protocol == IPPROTO_UDP)
        __sync_fetch_and_add(&stats->udp_packets, 1);
    else if (protocol == IPPROTO_ICMP)
        __sync_fetch_and_add(&stats->icmp_packets, 1);

    if (action == 0) // dropped
        __sync_fetch_and_add(&stats->dropped_packets, 1);
    else if (action == 1) // forwarded
        __sync_fetch_and_add(&stats->forwarded_packets, 1);
    else if (action == 2) // fallback
        __sync_fetch_and_add(&stats->fallback_to_userspace, 1);
}

// Main eBPF program for ingress traffic
SEC("tc")
int marchproxy_ingress(struct __sk_buff *skb) {
    __u16 eth_proto;
    struct iphdr *ip;
    int eth_hdr_len, ip_hdr_len;
    
    // Parse Ethernet header
    eth_hdr_len = parse_eth_hdr(skb, &eth_proto);
    if (eth_hdr_len < 0 || eth_proto != ETH_P_IP)
        return TC_ACT_OK; // Pass non-IP traffic

    // Parse IP header
    ip_hdr_len = parse_ip_hdr(skb, eth_hdr_len, &ip);
    if (ip_hdr_len < 0)
        return TC_ACT_OK; // Pass malformed IP packets

    __be32 src_ip = ip->saddr;
    __be32 dst_ip = ip->daddr;
    __u8 protocol = ip->protocol;
    __u16 src_port = 0, dst_port = 0;

    // Parse transport layer header for port information
    void *data_end = (void *)(long)skb->data_end;
    void *data = (void *)(long)skb->data;
    int transport_offset = eth_hdr_len + ip_hdr_len;

    if (protocol == IPPROTO_TCP) {
        struct tcphdr *tcp = data + transport_offset;
        if ((void *)(tcp + 1) > data_end)
            return TC_ACT_OK;
        
        src_port = bpf_ntohs(tcp->source);
        dst_port = bpf_ntohs(tcp->dest);
    } else if (protocol == IPPROTO_UDP) {
        struct udphdr *udp = data + transport_offset;
        if ((void *)(udp + 1) > data_end)
            return TC_ACT_OK;
        
        src_port = bpf_ntohs(udp->source);
        dst_port = bpf_ntohs(udp->dest);
    }

    // Find matching mapping
    struct mapping *mapping = find_mapping(dst_ip, dst_port, protocol);
    if (!mapping) {
        update_stats(skb->len, protocol, 0); // dropped
        return TC_ACT_SHOT; // Drop unmatched packets
    }

    // Find destination service
    struct service *dest_service = find_dest_service(mapping);
    if (!dest_service) {
        update_stats(skb->len, protocol, 0); // dropped
        return TC_ACT_SHOT; // Drop if no destination service
    }

    // Check if authentication is required
    if (mapping->auth_required || dest_service->auth_required) {
        // For packets requiring authentication, pass to userspace
        update_stats(skb->len, protocol, 2); // fallback
        return TC_ACT_OK; // Pass to userspace for authentication
    }

    // Track connection
    struct connection_key conn_key = {
        .src_ip = src_ip,
        .dst_ip = dst_ip,
        .src_port = src_port,
        .dst_port = dst_port,
        .protocol = protocol,
    };

    struct connection_value *conn_val = bpf_map_lookup_elem(&connections_map, &conn_key);
    if (conn_val) {
        // Update existing connection
        __sync_fetch_and_add(&conn_val->packets, 1);
        __sync_fetch_and_add(&conn_val->bytes, skb->len);
        conn_val->timestamp = bpf_ktime_get_ns();
    } else {
        // Create new connection entry
        struct connection_value new_conn = {
            .packets = 1,
            .bytes = skb->len,
            .timestamp = bpf_ktime_get_ns(),
            .service_id = dest_service->id,
            .authenticated = 0,
        };
        bpf_map_update_elem(&connections_map, &conn_key, &new_conn, BPF_ANY);
    }

    // Simple packet forwarding (modify destination IP/port)
    // In production, this would use proper packet rewriting
    update_stats(skb->len, protocol, 1); // forwarded
    
    // For now, redirect to userspace for actual forwarding
    // In a full implementation, this would modify packet headers and redirect
    return TC_ACT_OK;
}

// eBPF program for egress traffic
SEC("tc")
int marchproxy_egress(struct __sk_buff *skb) {
    // Similar logic for outbound traffic
    // For now, just pass through and collect stats
    
    __u16 eth_proto;
    int eth_hdr_len = parse_eth_hdr(skb, &eth_proto);
    if (eth_hdr_len < 0 || eth_proto != ETH_P_IP)
        return TC_ACT_OK;

    struct iphdr *ip;
    int ip_hdr_len = parse_ip_hdr(skb, eth_hdr_len, &ip);
    if (ip_hdr_len < 0)
        return TC_ACT_OK;

    update_stats(skb->len, ip->protocol, 1); // forwarded
    return TC_ACT_OK;
}

char _license[] SEC("license") = "GPL";
__u32 _version SEC("version") = 1;