// MarchProxy Working eBPF filter
#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

// Simple packet counter map
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 4);
    __type(key, __u32);
    __type(value, __u64);
} packet_count SEC(".maps");

// Basic TC filter that counts packets
SEC("tc")  
int marchproxy_filter(struct __sk_buff *skb) {
    __u32 index = 0;
    __u64 *count = bpf_map_lookup_elem(&packet_count, &index);
    
    if (count) {
        __sync_fetch_and_add(count, 1);
    }
    
    // Pass all packets through for now
    return TC_ACT_OK;
}

char _license[] SEC("license") = "GPL";
__u32 _version SEC("version") = 1;