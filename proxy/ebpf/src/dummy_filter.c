// SPDX-License-Identifier: GPL-2.0
// Dummy eBPF filter for testing Go integration

// Just include essential eBPF functionality without kernel headers
struct {
    int type;
    int max_entries;
    int *key;
    int *value;
} dummy_map __attribute__((section("maps")));

int dummy_xdp_prog(void *ctx) __attribute__((section("xdp"))) {
    // Dummy XDP program that just passes packets
    return 2; // XDP_PASS
}

char _license[] __attribute__((section("license"))) = "GPL";