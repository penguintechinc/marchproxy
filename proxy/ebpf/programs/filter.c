// MarchProxy eBPF Filter Program
// High-performance packet filtering for simple proxy rules

#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/tcp.h>
#include <linux/udp.h>
#include <linux/icmp.h>
#include <linux/in.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

// Map definitions for rule storage and statistics
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, __u32);     // Source IP
    __type(value, __u32);   // Allowed destination count
    __uint(max_entries, 10000);
} source_allow_map SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, struct rule_key);
    __type(value, struct rule_value);
    __uint(max_entries, 10000);
} proxy_rules SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __type(key, __u32);
    __type(value, __u64);
    __uint(max_entries, 256);
} stats_map SEC(".maps");

// Rule key structure
struct rule_key {
    __u32 src_ip;
    __u32 dst_ip;
    __u16 dst_port;
    __u8  protocol;
    __u8  pad;
};

// Rule value structure
struct rule_value {
    __u8  action;        // 0 = drop, 1 = allow, 2 = redirect
    __u8  auth_required; // 0 = no auth, 1 = auth required
    __u16 redirect_port; // Port for redirection
    __u32 redirect_ip;   // IP for redirection
    __u64 rule_id;       // Rule identifier for logging
};

// Statistics counters
enum {
    STAT_PACKETS_PROCESSED = 0,
    STAT_PACKETS_ALLOWED = 1,
    STAT_PACKETS_DROPPED = 2,
    STAT_PACKETS_REDIRECTED = 3,
    STAT_PACKETS_TO_USERSPACE = 4,
    STAT_AUTH_REQUIRED = 5,
};

// Helper function to update statistics
static __always_inline void update_stat(__u32 key) {
    __u64 *counter = bpf_map_lookup_elem(&stats_map, &key);
    if (counter) {
        __sync_fetch_and_add(counter, 1);
    }
}

// Parse Ethernet header
static __always_inline int parse_eth(void *data, void *data_end, __u16 *eth_type) {
    struct ethhdr *eth = data;
    
    if ((void *)(eth + 1) > data_end)
        return -1;
    
    *eth_type = bpf_ntohs(eth->h_proto);
    return sizeof(*eth);
}

// Parse IP header
static __always_inline int parse_ip(void *data, void *data_end, struct iphdr **iph) {
    struct iphdr *ip = data;
    
    if ((void *)(ip + 1) > data_end)
        return -1;
    
    // Check IP version and header length
    if (ip->version != 4 || ip->ihl < 5)
        return -1;
    
    int ip_len = ip->ihl * 4;
    if ((void *)ip + ip_len > data_end)
        return -1;
    
    *iph = ip;
    return ip_len;
}

// Main XDP program for fast packet processing
SEC("xdp")
int marchproxy_filter(struct xdp_md *ctx) {
    void *data_end = (void *)(long)ctx->data_end;
    void *data = (void *)(long)ctx->data;
    
    __u16 eth_type;
    struct iphdr *ip;
    struct rule_key key = {};
    
    // Update processed packets counter
    update_stat(STAT_PACKETS_PROCESSED);
    
    // Parse Ethernet header
    int offset = parse_eth(data, data_end, &eth_type);
    if (offset < 0 || eth_type != ETH_P_IP) {
        // Not IPv4, pass to userspace
        update_stat(STAT_PACKETS_TO_USERSPACE);
        return XDP_PASS;
    }
    
    // Parse IP header
    int ip_offset = parse_ip(data + offset, data_end, &ip);
    if (ip_offset < 0) {
        // Invalid IP header, drop
        update_stat(STAT_PACKETS_DROPPED);
        return XDP_DROP;
    }
    
    // Fill rule key
    key.src_ip = ip->saddr;
    key.dst_ip = ip->daddr;
    key.protocol = ip->protocol;
    
    // Parse transport layer for port information
    void *transport = data + offset + ip_offset;
    
    switch (ip->protocol) {
        case IPPROTO_TCP: {
            struct tcphdr *tcp = transport;
            if ((void *)(tcp + 1) > data_end) {
                update_stat(STAT_PACKETS_TO_USERSPACE);
                return XDP_PASS;
            }
            key.dst_port = bpf_ntohs(tcp->dest);
            break;
        }
        case IPPROTO_UDP: {
            struct udphdr *udp = transport;
            if ((void *)(udp + 1) > data_end) {
                update_stat(STAT_PACKETS_TO_USERSPACE);
                return XDP_PASS;
            }
            key.dst_port = bpf_ntohs(udp->dest);
            break;
        }
        case IPPROTO_ICMP: {
            // ICMP doesn't have ports, set to 0
            key.dst_port = 0;
            break;
        }
        default:
            // Unknown protocol, pass to userspace
            update_stat(STAT_PACKETS_TO_USERSPACE);
            return XDP_PASS;
    }
    
    // Lookup rule in map
    struct rule_value *rule = bpf_map_lookup_elem(&proxy_rules, &key);
    if (!rule) {
        // No specific rule, check source allowlist
        __u32 *src_allowed = bpf_map_lookup_elem(&source_allow_map, &key.src_ip);
        if (!src_allowed) {
            // Source not in allowlist, drop
            update_stat(STAT_PACKETS_DROPPED);
            return XDP_DROP;
        }
        // Source allowed, pass to userspace for detailed processing
        update_stat(STAT_PACKETS_TO_USERSPACE);
        return XDP_PASS;
    }
    
    // Process based on rule action
    switch (rule->action) {
        case 0: // Drop
            update_stat(STAT_PACKETS_DROPPED);
            return XDP_DROP;
            
        case 1: // Allow
            if (rule->auth_required) {
                // Authentication required, pass to userspace
                update_stat(STAT_AUTH_REQUIRED);
                return XDP_PASS;
            } else {
                // Direct allow
                update_stat(STAT_PACKETS_ALLOWED);
                return XDP_PASS;
            }
            
        case 2: // Redirect
            // Redirection logic would be implemented here
            // For now, pass to userspace for complex redirection
            update_stat(STAT_PACKETS_REDIRECTED);
            return XDP_PASS;
            
        default:
            // Unknown action, pass to userspace
            update_stat(STAT_PACKETS_TO_USERSPACE);
            return XDP_PASS;
    }
}

// TC program for egress traffic filtering
SEC("tc")
int marchproxy_egress(struct __sk_buff *skb) {
    void *data_end = (void *)(long)skb->data_end;
    void *data = (void *)(long)skb->data;
    
    __u16 eth_type;
    struct iphdr *ip;
    
    // Update processed packets counter
    update_stat(STAT_PACKETS_PROCESSED);
    
    // Parse Ethernet header
    int offset = parse_eth(data, data_end, &eth_type);
    if (offset < 0 || eth_type != ETH_P_IP) {
        return TC_ACT_OK; // Pass non-IP traffic
    }
    
    // Parse IP header
    int ip_offset = parse_ip(data + offset, data_end, &ip);
    if (ip_offset < 0) {
        return TC_ACT_SHOT; // Drop malformed packets
    }
    
    // For egress, we mainly collect statistics and apply rate limiting
    // More complex logic would go here
    
    return TC_ACT_OK;
}

// Socket filter program for connection tracking
SEC("socket")
int marchproxy_socket_filter(struct __sk_buff *skb) {
    // This would implement connection state tracking
    // and provide detailed flow information to userspace
    return 0; // Allow all for now
}

char _license[] SEC("license") = "GPL";