// MarchProxy XDP Rate Limiting Program
// High-performance packet rate limiting at the driver level

#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/tcp.h>
#include <linux/udp.h>
#include <linux/in.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

// Rate limiting configuration structure
struct rate_limit_config {
    __u32 enabled;              // Rate limiting enabled flag
    __u32 global_pps_limit;     // Global packets per second limit
    __u32 per_ip_pps_limit;     // Per-IP packets per second limit
    __u32 window_size_ns;       // Time window in nanoseconds (default: 1 second)
    __u32 burst_allowance;      // Burst packets allowed above rate
    __u32 action;               // Action: 0=PASS, 1=DROP, 2=RATE_LIMIT
};

// Per-IP rate limiting state
struct ip_rate_state {
    __u64 last_update_ns;       // Last update timestamp
    __u32 packet_count;         // Packets in current window
    __u32 total_packets;        // Total packets seen
    __u32 dropped_packets;      // Total dropped packets
    __u32 burst_tokens;         // Available burst tokens
};

// Global rate limiting state
struct global_rate_state {
    __u64 last_update_ns;       // Last update timestamp
    __u32 packet_count;         // Packets in current window
    __u32 total_packets;        // Total packets processed
    __u32 dropped_packets;      // Total dropped packets
};

// Statistics structure for monitoring
struct rate_limit_stats {
    __u64 total_packets;        // Total packets processed
    __u64 passed_packets;       // Packets passed through
    __u64 dropped_packets;      // Packets dropped
    __u64 rate_limited_ips;     // Number of IPs currently rate limited
    __u64 global_drops;         // Drops due to global rate limit
    __u64 per_ip_drops;         // Drops due to per-IP rate limit
};

// BPF Maps
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __type(key, __u32);
    __type(value, struct rate_limit_config);
    __uint(max_entries, 1);
} rate_limit_config_map SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __type(key, __u32);  // Source IP address
    __type(value, struct ip_rate_state);
    __uint(max_entries, 65536);  // Support up to 64K unique IPs
} ip_rate_state_map SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __type(key, __u32);
    __type(value, struct global_rate_state);
    __uint(max_entries, 1);
} global_rate_state_map SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __type(key, __u32);
    __type(value, struct rate_limit_stats);
    __uint(max_entries, 1);
} rate_limit_stats_map SEC(".maps");

// Enterprise license validation map (updated by user space)
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __type(key, __u32);
    __type(value, __u32);  // 1 if Enterprise features enabled, 0 otherwise
    __uint(max_entries, 1);
} enterprise_license_map SEC(".maps");

// Helper function to parse IPv4 header
static inline int parse_ipv4(struct xdp_md *ctx, __u32 *src_ip, __u32 *dst_ip) {
    void *data_end = (void *)(long)ctx->data_end;
    void *data = (void *)(long)ctx->data;

    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return -1;

    if (eth->h_proto != bpf_htons(ETH_P_IP))
        return -1;

    struct iphdr *ip = (void *)(eth + 1);
    if ((void *)(ip + 1) > data_end)
        return -1;

    *src_ip = ip->saddr;
    *dst_ip = ip->daddr;
    return 0;
}

// Rate limiting logic for a specific IP
static inline int check_ip_rate_limit(struct rate_limit_config *config,
                                     __u32 src_ip, __u64 now_ns) {
    struct ip_rate_state *state = bpf_map_lookup_elem(&ip_rate_state_map, &src_ip);
    struct ip_rate_state new_state = {0};

    if (!state) {
        // First packet from this IP
        new_state.last_update_ns = now_ns;
        new_state.packet_count = 1;
        new_state.total_packets = 1;
        new_state.burst_tokens = config->burst_allowance;
        bpf_map_update_elem(&ip_rate_state_map, &src_ip, &new_state, BPF_ANY);
        return XDP_PASS;
    }

    // Calculate time elapsed since last update
    __u64 elapsed_ns = now_ns - state->last_update_ns;

    // Reset counters if window has elapsed
    if (elapsed_ns >= config->window_size_ns) {
        state->last_update_ns = now_ns;
        state->packet_count = 1;
        state->burst_tokens = config->burst_allowance;
    } else {
        state->packet_count++;
    }

    state->total_packets++;

    // Check if rate limit exceeded
    if (state->packet_count > config->per_ip_pps_limit) {
        // Try to use burst tokens
        if (state->burst_tokens > 0) {
            state->burst_tokens--;
        } else {
            // Rate limit exceeded, drop packet
            state->dropped_packets++;
            bpf_map_update_elem(&ip_rate_state_map, &src_ip, state, BPF_ANY);
            return XDP_DROP;
        }
    }

    bpf_map_update_elem(&ip_rate_state_map, &src_ip, state, BPF_ANY);
    return XDP_PASS;
}

// Global rate limiting logic
static inline int check_global_rate_limit(struct rate_limit_config *config, __u64 now_ns) {
    __u32 key = 0;
    struct global_rate_state *state = bpf_map_lookup_elem(&global_rate_state_map, &key);
    struct global_rate_state new_state = {0};

    if (!state) {
        // Initialize global state
        new_state.last_update_ns = now_ns;
        new_state.packet_count = 1;
        new_state.total_packets = 1;
        bpf_map_update_elem(&global_rate_state_map, &key, &new_state, BPF_ANY);
        return XDP_PASS;
    }

    // Calculate time elapsed since last update
    __u64 elapsed_ns = now_ns - state->last_update_ns;

    // Reset counters if window has elapsed
    if (elapsed_ns >= config->window_size_ns) {
        state->last_update_ns = now_ns;
        state->packet_count = 1;
    } else {
        state->packet_count++;
    }

    state->total_packets++;

    // Check if global rate limit exceeded
    if (state->packet_count > config->global_pps_limit) {
        state->dropped_packets++;
        bpf_map_update_elem(&global_rate_state_map, &key, state, BPF_ANY);
        return XDP_DROP;
    }

    bpf_map_update_elem(&global_rate_state_map, &key, state, BPF_ANY);
    return XDP_PASS;
}

// Update statistics
static inline void update_stats(__u32 action, bool global_drop, bool ip_drop) {
    __u32 key = 0;
    struct rate_limit_stats *stats = bpf_map_lookup_elem(&rate_limit_stats_map, &key);
    struct rate_limit_stats new_stats = {0};

    if (!stats) {
        stats = &new_stats;
    }

    stats->total_packets++;

    if (action == XDP_DROP) {
        stats->dropped_packets++;
        if (global_drop)
            stats->global_drops++;
        if (ip_drop)
            stats->per_ip_drops++;
    } else {
        stats->passed_packets++;
    }

    bpf_map_update_elem(&rate_limit_stats_map, &key, stats, BPF_ANY);
}

// Main XDP program entry point
SEC("xdp")
int xdp_rate_limiter(struct xdp_md *ctx) {
    // Check if Enterprise license is active
    __u32 license_key = 0;
    __u32 *enterprise_enabled = bpf_map_lookup_elem(&enterprise_license_map, &license_key);
    if (!enterprise_enabled || *enterprise_enabled == 0) {
        // Rate limiting requires Enterprise license
        return XDP_PASS;
    }

    // Get rate limiting configuration
    __u32 config_key = 0;
    struct rate_limit_config *config = bpf_map_lookup_elem(&rate_limit_config_map, &config_key);
    if (!config || !config->enabled) {
        return XDP_PASS;
    }

    // Parse packet to extract source IP
    __u32 src_ip, dst_ip;
    if (parse_ipv4(ctx, &src_ip, &dst_ip) < 0) {
        // Not IPv4 or parse error, pass through
        return XDP_PASS;
    }

    // Get current timestamp
    __u64 now_ns = bpf_ktime_get_ns();

    int action = XDP_PASS;
    bool global_drop = false;
    bool ip_drop = false;

    // Check global rate limit first
    if (config->global_pps_limit > 0) {
        action = check_global_rate_limit(config, now_ns);
        if (action == XDP_DROP) {
            global_drop = true;
            goto update_and_return;
        }
    }

    // Check per-IP rate limit
    if (config->per_ip_pps_limit > 0) {
        action = check_ip_rate_limit(config, src_ip, now_ns);
        if (action == XDP_DROP) {
            ip_drop = true;
        }
    }

update_and_return:
    // Update statistics
    update_stats(action, global_drop, ip_drop);

    return action;
}

char _license[] SEC("license") = "GPL";