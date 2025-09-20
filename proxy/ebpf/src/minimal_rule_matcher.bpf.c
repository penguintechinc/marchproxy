// SPDX-License-Identifier: GPL-2.0
// MarchProxy Minimal eBPF Rule Matcher
// Ultra-minimal packet filtering without problematic headers

// Define required types first
typedef unsigned char       __u8;
typedef unsigned short      __u16;
typedef unsigned int        __u32;
typedef unsigned long long  __u64;
typedef __u32               __be32;
typedef __u16               __be16;

#include <bpf/bpf_helpers.h>

// Network constants
#define ETH_P_IP        0x0800
#define IPPROTO_TCP     6
#define IPPROTO_UDP     17
#define IPPROTO_ICMP    1

// Simplified network structures
struct ethhdr {
    __u8    h_dest[6];
    __u8    h_source[6];
    __be16  h_proto;
} __attribute__((packed));

struct iphdr {
    __u8    version_ihl;    // version(4) + ihl(4)
    __u8    tos;
    __be16  tot_len;
    __be16  id;
    __be16  frag_off;
    __u8    ttl;
    __u8    protocol;
    __u16   check;
    __be32  saddr;
    __be32  daddr;
} __attribute__((packed));

struct tcphdr {
    __be16  source;
    __be16  dest;
    __be32  seq;
    __be32  ack_seq;
    __u16   flags;
    __be16  window;
    __u16   check;
    __u16   urg_ptr;
} __attribute__((packed));

struct udphdr {
    __be16  source;
    __be16  dest;
    __be16  len;
    __u16   check;
} __attribute__((packed));

// Simple service rule structure
struct service_rule {
    __u32 service_id;
    __be32 ip_addr;
    __u16 port;
    __u8 protocol;
    __u8 action;        // 0=drop, 1=allow, 2=userspace
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

// BPF maps
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 1000);
    __type(key, __u32);
    __type(value, struct service_rule);
} service_rules SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 1);
    __type(key, __u32);
    __type(value, struct ebpf_stats);
} statistics SEC(".maps");

// Helper macros for endianness
#define bpf_htons(x) __builtin_bswap16(x)
#define bpf_ntohs(x) __builtin_bswap16(x)

// Action definitions
#define ACTION_DROP       0
#define ACTION_ALLOW      1
#define ACTION_USERSPACE  2

// Simple XDP program
SEC("xdp")
int minimal_rule_matcher(struct xdp_md *ctx) {
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
    if (protocol == IPPROTO_TCP) {
        struct tcphdr *tcp = (void *)(ip + 1);
        if ((void *)(tcp + 1) > data_end)
            return XDP_DROP;
        dst_port = tcp->dest;
        if (stats) {
            __sync_fetch_and_add(&stats->tcp_packets, 1);
        }
    } else if (protocol == IPPROTO_UDP) {
        struct udphdr *udp = (void *)(ip + 1);
        if ((void *)(udp + 1) > data_end)
            return XDP_DROP;
        dst_port = udp->dest;
        if (stats) {
            __sync_fetch_and_add(&stats->udp_packets, 1);
        }
    } else {
        // Non-TCP/UDP traffic, pass to userspace
        return XDP_PASS;
    }

    // Look for matching service rules (simple linear search)
    struct service_rule *rule;
    __u32 rule_id;

    // Check first 50 rules for performance
    for (rule_id = 1; rule_id <= 50; rule_id++) {
        rule = bpf_map_lookup_elem(&service_rules, &rule_id);
        if (!rule)
            continue;

        // Check if rule matches this packet
        if (rule->protocol == protocol &&
            rule->ip_addr == dst_ip &&
            rule->port == dst_port) {

            // Found a matching rule, apply action
            if (rule->action == ACTION_DROP) {
                if (stats) {
                    __sync_fetch_and_add(&stats->dropped_packets, 1);
                }
                return XDP_DROP;
            } else if (rule->action == ACTION_ALLOW) {
                if (stats) {
                    __sync_fetch_and_add(&stats->allowed_packets, 1);
                }
                return XDP_PASS;
            } else {
                if (stats) {
                    __sync_fetch_and_add(&stats->userspace_packets, 1);
                }
                return XDP_PASS;
            }
        }
    }

    // No matching rule found, pass to userspace
    return XDP_PASS;
}

char _license[] SEC("license") = "GPL";