// SPDX-License-Identifier: GPL-2.0
// MarchProxy eBPF Rule Matcher
// Simple packet filtering and service rule matching

#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/tcp.h>
#include <linux/udp.h>
#include <linux/icmp.h>
#include <linux/in.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

// Service rule structure - must match Go struct in internal/ebpf/loader.go
struct service_rule {
    __u32 service_id;
    __be32 ip_addr;    // Network byte order
    __u16 port;
    __u8 protocol;     // IPPROTO_TCP, IPPROTO_UDP, IPPROTO_ICMP
    __u8 action;       // 0=drop, 1=allow, 2=userspace
};

// Statistics structure
struct ebpf_stats {
    __u64 total_packets;
    __u64 tcp_packets;
    __u64 udp_packets;
    __u64 dropped_packets;
    __u64 allowed_packets;
    __u64 userspace_packets;
};

// BPF maps for service rules and statistics
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 10000);
    __type(key, __u32);                 // Rule ID
    __type(value, struct service_rule);
} service_rules SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 1000);
    __type(key, __u32);                 // Service ID
    __type(value, __u32);               // Rule ID list start (linked)
} service_lookup SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 1);
    __type(key, __u32);
    __type(value, struct ebpf_stats);
} statistics SEC(".maps");

// Action definitions
#define ACTION_DROP       0
#define ACTION_ALLOW      1
#define ACTION_USERSPACE  2

// Helper function to update statistics
static inline void update_stats(__u64 *counter) {
    __u32 key = 0;
    struct ebpf_stats *stats = bpf_map_lookup_elem(&statistics, &key);
    if (stats) {
        __sync_fetch_and_add(counter, 1);
    }
}

// Main packet processing function
SEC("xdp")
int rule_matcher_xdp(struct xdp_md *ctx) {
    void *data_end = (void *)(long)ctx->data_end;
    void *data = (void *)(long)ctx->data;

    // Update total packet counter
    __u32 stats_key = 0;
    struct ebpf_stats *stats = bpf_map_lookup_elem(&statistics, &stats_key);
    if (stats) {
        __sync_fetch_and_add(&stats->total_packets, 1);
    }

    // Parse Ethernet header
    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return XDP_DROP;

    // Only process IPv4 packets
    if (eth->h_proto != bpf_htons(ETH_P_IP))
        return XDP_PASS;

    // Parse IP header
    struct iphdr *ip = (void *)(eth + 1);
    if ((void *)(ip + 1) > data_end)
        return XDP_DROP;

    __u8 protocol = ip->protocol;
    __be32 dst_ip = ip->daddr;
    __u16 dst_port = 0;

    // Extract destination port based on protocol
    switch (protocol) {
        case IPPROTO_TCP: {
            struct tcphdr *tcp = (void *)(ip + 1);
            if ((void *)(tcp + 1) > data_end)
                return XDP_DROP;
            dst_port = tcp->dest;
            if (stats) {
                __sync_fetch_and_add(&stats->tcp_packets, 1);
            }
            break;
        }
        case IPPROTO_UDP: {
            struct udphdr *udp = (void *)(ip + 1);
            if ((void *)(udp + 1) > data_end)
                return XDP_DROP;
            dst_port = udp->dest;
            if (stats) {
                __sync_fetch_and_add(&stats->udp_packets, 1);
            }
            break;
        }
        case IPPROTO_ICMP: {
            // ICMP doesn't have ports, use type/code as port for rule matching
            struct icmphdr *icmp = (void *)(ip + 1);
            if ((void *)(icmp + 1) > data_end)
                return XDP_DROP;
            dst_port = bpf_htons((icmp->type << 8) | icmp->code);
            break;
        }
        default:
            // Unknown protocol, pass to userspace
            return XDP_PASS;
    }

    // Look for matching service rules
    // This is a simplified linear search - could be optimized with better data structures
    struct service_rule *rule;
    __u32 rule_id;

    // Iterate through possible rule IDs (simplified approach)
    for (rule_id = 1; rule_id <= 1000; rule_id++) {
        rule = bpf_map_lookup_elem(&service_rules, &rule_id);
        if (!rule)
            continue;

        // Check if rule matches this packet
        if (rule->protocol == protocol &&
            rule->ip_addr == dst_ip &&
            rule->port == dst_port) {

            // Found a matching rule, apply action
            switch (rule->action) {
                case ACTION_DROP:
                    if (stats) {
                        __sync_fetch_and_add(&stats->dropped_packets, 1);
                    }
                    return XDP_DROP;

                case ACTION_ALLOW:
                    if (stats) {
                        __sync_fetch_and_add(&stats->allowed_packets, 1);
                    }
                    return XDP_PASS;

                case ACTION_USERSPACE:
                    if (stats) {
                        __sync_fetch_and_add(&stats->userspace_packets, 1);
                    }
                    return XDP_PASS;  // Let userspace handle it

                default:
                    return XDP_PASS;
            }
        }
    }

    // No matching rule found, pass to userspace for processing
    return XDP_PASS;
}

// TC (Traffic Control) ingress program for more complex processing
SEC("tc")
int rule_matcher_tc_ingress(struct __sk_buff *skb) {
    void *data_end = (void *)(long)skb->data_end;
    void *data = (void *)(long)skb->data;

    // Update total packet counter
    __u32 stats_key = 0;
    struct ebpf_stats *stats = bpf_map_lookup_elem(&statistics, &stats_key);
    if (stats) {
        __sync_fetch_and_add(&stats->total_packets, 1);
    }

    // Parse Ethernet header
    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return TC_ACT_SHOT;

    // Only process IPv4 packets
    if (eth->h_proto != bpf_htons(ETH_P_IP))
        return TC_ACT_OK;

    // Parse IP header
    struct iphdr *ip = (void *)(eth + 1);
    if ((void *)(ip + 1) > data_end)
        return TC_ACT_SHOT;

    __u8 protocol = ip->protocol;
    __be32 dst_ip = ip->daddr;
    __u16 dst_port = 0;

    // Extract destination port based on protocol
    switch (protocol) {
        case IPPROTO_TCP: {
            struct tcphdr *tcp = (void *)(ip + 1);
            if ((void *)(tcp + 1) > data_end)
                return TC_ACT_SHOT;
            dst_port = tcp->dest;
            if (stats) {
                __sync_fetch_and_add(&stats->tcp_packets, 1);
            }
            break;
        }
        case IPPROTO_UDP: {
            struct udphdr *udp = (void *)(ip + 1);
            if ((void *)(udp + 1) > data_end)
                return TC_ACT_SHOT;
            dst_port = udp->dest;
            if (stats) {
                __sync_fetch_and_add(&stats->udp_packets, 1);
            }
            break;
        }
        case IPPROTO_ICMP: {
            struct icmphdr *icmp = (void *)(ip + 1);
            if ((void *)(icmp + 1) > data_end)
                return TC_ACT_SHOT;
            dst_port = bpf_htons((icmp->type << 8) | icmp->code);
            break;
        }
        default:
            return TC_ACT_OK;
    }

    // Look for matching service rules (same logic as XDP)
    struct service_rule *rule;
    __u32 rule_id;

    for (rule_id = 1; rule_id <= 1000; rule_id++) {
        rule = bpf_map_lookup_elem(&service_rules, &rule_id);
        if (!rule)
            continue;

        if (rule->protocol == protocol &&
            rule->ip_addr == dst_ip &&
            rule->port == dst_port) {

            switch (rule->action) {
                case ACTION_DROP:
                    if (stats) {
                        __sync_fetch_and_add(&stats->dropped_packets, 1);
                    }
                    return TC_ACT_SHOT;

                case ACTION_ALLOW:
                    if (stats) {
                        __sync_fetch_and_add(&stats->allowed_packets, 1);
                    }
                    return TC_ACT_OK;

                case ACTION_USERSPACE:
                    if (stats) {
                        __sync_fetch_and_add(&stats->userspace_packets, 1);
                    }
                    return TC_ACT_OK;

                default:
                    return TC_ACT_OK;
            }
        }
    }

    // No rule found, allow to userspace
    return TC_ACT_OK;
}

char _license[] SEC("license") = "GPL";