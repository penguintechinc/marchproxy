// MarchProxy Simple eBPF packet filter
// Simplified version compatible with standard kernel headers

#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/tcp.h>
#include <linux/udp.h>
#include <linux/icmp.h>
#include <linux/in.h>
#include <linux/pkt_cls.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

#define MAX_RULES 512

// Simple rule structure
struct filter_rule {
    __u32 src_ip;
    __u32 dst_ip;
    __u16 src_port;
    __u16 dst_port;
    __u8 protocol;  // IPPROTO_TCP, IPPROTO_UDP, etc.
    __u8 action;    // 0 = drop, 1 = allow, 2 = redirect to userspace
};

// Statistics structure
struct filter_stats {
    __u64 total_packets;
    __u64 allowed_packets;
    __u64 dropped_packets;
    __u64 redirected_packets;
};

// Maps
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, MAX_RULES);
    __type(key, __u32);  // rule ID
    __type(value, struct filter_rule);
} rules_map SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 1);
    __type(key, __u32);
    __type(value, struct filter_stats);
} stats_map SEC(".maps");

// Parse ethernet header
static __always_inline int parse_ethernet(void *data, void *data_end, __u16 *eth_type) {
    struct ethhdr *eth = data;
    
    if ((void *)(eth + 1) > data_end)
        return -1;
        
    *eth_type = bpf_ntohs(eth->h_proto);
    return sizeof(struct ethhdr);
}

// Parse IP header
static __always_inline int parse_ip(void *data, int offset, void *data_end, 
                                   struct iphdr **ip_hdr) {
    struct iphdr *ip = data + offset;
    
    if ((void *)(ip + 1) > data_end)
        return -1;
        
    if (ip->version != 4 || ip->ihl < 5)
        return -1;
        
    *ip_hdr = ip;
    return ip->ihl * 4;
}

// Update statistics
static __always_inline void update_stats(__u8 action) {
    __u32 key = 0;
    struct filter_stats *stats = bpf_map_lookup_elem(&stats_map, &key);
    
    if (!stats)
        return;
        
    __sync_fetch_and_add(&stats->total_packets, 1);
    
    if (action == 0)
        __sync_fetch_and_add(&stats->dropped_packets, 1);
    else if (action == 1) 
        __sync_fetch_and_add(&stats->allowed_packets, 1);
    else if (action == 2)
        __sync_fetch_and_add(&stats->redirected_packets, 1);
}

// Main filter program
SEC("tc")
int marchproxy_filter(struct __sk_buff *skb) {
    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;
    __u16 eth_type;
    struct iphdr *ip;
    
    // Parse ethernet header
    int eth_len = parse_ethernet(data, data_end, &eth_type);
    if (eth_len < 0 || eth_type != ETH_P_IP) {
        update_stats(1); // allow non-IP traffic
        return TC_ACT_OK;
    }
    
    // Parse IP header
    int ip_len = parse_ip(data, eth_len, data_end, &ip);
    if (ip_len < 0) {
        update_stats(0); // drop malformed packets
        return TC_ACT_SHOT;
    }
    
    __u32 src_ip = bpf_ntohl(ip->saddr);
    __u32 dst_ip = bpf_ntohl(ip->daddr);
    __u8 protocol = ip->protocol;
    __u16 src_port = 0, dst_port = 0;
    
    // Parse transport layer for port info
    int transport_offset = eth_len + ip_len;
    
    if (protocol == IPPROTO_TCP) {
        struct tcphdr *tcp = data + transport_offset;
        if ((void *)(tcp + 1) > data_end) {
            update_stats(0);
            return TC_ACT_SHOT;
        }
        src_port = bpf_ntohs(tcp->source);
        dst_port = bpf_ntohs(tcp->dest);
    } else if (protocol == IPPROTO_UDP) {
        struct udphdr *udp = data + transport_offset;
        if ((void *)(udp + 1) > data_end) {
            update_stats(0);
            return TC_ACT_SHOT;
        }
        src_port = bpf_ntohs(udp->source);
        dst_port = bpf_ntohs(udp->dest);
    }
    
    // Check rules (simplified - iterate through first few rules)
    for (__u32 rule_id = 0; rule_id < 32; rule_id++) {
        struct filter_rule *rule = bpf_map_lookup_elem(&rules_map, &rule_id);
        if (!rule)
            continue;
            
        // Check if rule matches
        int matches = 1;
        
        if (rule->src_ip != 0 && rule->src_ip != src_ip)
            matches = 0;
        if (rule->dst_ip != 0 && rule->dst_ip != dst_ip) 
            matches = 0;
        if (rule->protocol != 0 && rule->protocol != protocol)
            matches = 0;
        if (rule->src_port != 0 && rule->src_port != src_port)
            matches = 0;
        if (rule->dst_port != 0 && rule->dst_port != dst_port)
            matches = 0;
            
        if (matches) {
            update_stats(rule->action);
            
            if (rule->action == 0)
                return TC_ACT_SHOT;    // drop
            else if (rule->action == 1)
                return TC_ACT_OK;      // allow
            else if (rule->action == 2)
                return TC_ACT_OK;      // redirect to userspace
        }
    }
    
    // Default action: allow and redirect to userspace for complex processing
    update_stats(2);
    return TC_ACT_OK;
}

char _license[] SEC("license") = "GPL";
__u32 _version SEC("version") = 1;