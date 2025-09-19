// MarchProxy Minimal eBPF filter - Compatible with available headers
#include <stddef.h>
#include <linux/pkt_cls.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

// Simple packet counter
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 4);
    __type(key, __u32);
    __type(value, __u64);
} packet_count SEC(".maps");

// Basic filter that counts packets by type
SEC("tc")
int marchproxy_minimal(struct __sk_buff *skb) {
    __u32 index = 0;
    __u64 *count = bpf_map_lookup_elem(&packet_count, &index);
    
    if (count) {
        __sync_fetch_and_add(count, 1);
    }
    
    // For now, just pass all packets through
    return TC_ACT_OK;
}

char _license[] SEC("license") = "GPL";
__u32 _version SEC("version") = 1;