// MarchProxy Complete eBPF packet filter
#include <linux/bpf.h>
#include <linux/pkt_cls.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/tcp.h>
#include <linux/udp.h>
#include <linux/in.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

// Statistics map
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 8);
    __type(key, __u32);
    __type(value, __u64);
} stats_map SEC(".maps");

// Service rules map (simplified)
struct service_rule {
    __u32 service_id;
    __be32 ip_addr;
    __u16 port;
    __u8 protocol;
    __u8 action;  // 0=drop, 1=allow, 2=userspace
};

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 1024);
    __type(key, __u32);  // rule ID
    __type(value, struct service_rule);
} rules_map SEC(".maps");

// Statistics indices
#define STAT_TOTAL_PACKETS     0
#define STAT_TCP_PACKETS       1
#define STAT_UDP_PACKETS       2
#define STAT_DROPPED_PACKETS   3
#define STAT_ALLOWED_PACKETS   4
#define STAT_USERSPACE_PACKETS 5

static __always_inline void update_stat(__u32 stat_type) {
    __u64 *count = bpf_map_lookup_elem(&stats_map, &stat_type);
    if (count) {
        __sync_fetch_and_add(count, 1);
    }
}

static __always_inline int parse_ethernet(void *data, void *data_end, __u16 *eth_type) {
    struct ethhdr *eth = data;
    
    if ((void *)(eth + 1) > data_end)
        return -1;
    
    *eth_type = bpf_ntohs(eth->h_proto);
    return sizeof(struct ethhdr);
}

static __always_inline int parse_ip(void *data, int offset, void *data_end, struct iphdr **ip_hdr) {
    struct iphdr *ip = data + offset;
    
    if ((void *)(ip + 1) > data_end)
        return -1;
    
    if (ip->version != 4 || ip->ihl < 5)
        return -1;
    
    *ip_hdr = ip;
    return ip->ihl * 4;
}

SEC("tc")
int marchproxy_filter(struct __sk_buff *skb) {
    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;
    struct iphdr *ip;
    __u16 eth_type;
    
    // Update total packet count
    update_stat(STAT_TOTAL_PACKETS);
    
    // Parse Ethernet header
    int eth_len = parse_ethernet(data, data_end, &eth_type);
    if (eth_len < 0 || eth_type != ETH_P_IP) {
        // Allow non-IP traffic
        update_stat(STAT_ALLOWED_PACKETS);
        return TC_ACT_OK;
    }
    
    // Parse IP header
    int ip_len = parse_ip(data, eth_len, data_end, &ip);
    if (ip_len < 0) {
        // Drop malformed IP packets
        update_stat(STAT_DROPPED_PACKETS);
        return TC_ACT_SHOT;
    }
    
    __u8 protocol = ip->protocol;
    __u16 src_port = 0, dst_port = 0;
    
    // Update protocol-specific stats
    if (protocol == IPPROTO_TCP) {
        update_stat(STAT_TCP_PACKETS);
        
        struct tcphdr *tcp = data + eth_len + ip_len;
        if ((void *)(tcp + 1) > data_end) {
            update_stat(STAT_DROPPED_PACKETS);
            return TC_ACT_SHOT;
        }
        
        src_port = bpf_ntohs(tcp->source);
        dst_port = bpf_ntohs(tcp->dest);
        
    } else if (protocol == IPPROTO_UDP) {
        update_stat(STAT_UDP_PACKETS);
        
        struct udphdr *udp = data + eth_len + ip_len;
        if ((void *)(udp + 1) > data_end) {
            update_stat(STAT_DROPPED_PACKETS);
            return TC_ACT_SHOT;
        }
        
        src_port = bpf_ntohs(udp->source);
        dst_port = bpf_ntohs(udp->dest);
    }
    
    // Check first few rules for matching (simplified lookup)
    for (__u32 rule_id = 0; rule_id < 32; rule_id++) {
        struct service_rule *rule = bpf_map_lookup_elem(&rules_map, &rule_id);
        if (!rule)
            continue;
        
        // Check if rule matches packet
        if (rule->protocol != 0 && rule->protocol != protocol)
            continue;
        
        if (rule->port != 0 && rule->port != dst_port)
            continue;
        
        if (rule->ip_addr != 0 && rule->ip_addr != ip->daddr)
            continue;
        
        // Rule matches - take action
        if (rule->action == 0) {
            // Drop packet
            update_stat(STAT_DROPPED_PACKETS);
            return TC_ACT_SHOT;
        } else if (rule->action == 1) {
            // Allow packet
            update_stat(STAT_ALLOWED_PACKETS);
            return TC_ACT_OK;
        } else if (rule->action == 2) {
            // Send to userspace for complex processing
            update_stat(STAT_USERSPACE_PACKETS);
            return TC_ACT_OK;
        }
    }
    
    // Default: send to userspace for processing
    update_stat(STAT_USERSPACE_PACKETS);
    return TC_ACT_OK;
}

char _license[] SEC("license") = "GPL";
__u32 _version SEC("version") = 1;