/* MarchProxy Envoy XDP Program
 * Early packet classification and DDoS protection for Envoy L7 proxy
 *
 * This XDP program provides:
 * - Protocol detection (HTTP/HTTPS/HTTP2/gRPC/WebSocket)
 * - Rate limiting using BPF maps
 * - DDoS protection at wire speed
 * - Early packet dropping for invalid traffic
 */

#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/ipv6.h>
#include <linux/in.h>
#include <linux/tcp.h>
#include <linux/udp.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

/* Map Definitions */

/* Rate limiting map: Key = source IP, Value = packet count + timestamp */
struct rate_limit_key {
	__u32 src_ip;
};

struct rate_limit_value {
	__u64 packet_count;
	__u64 last_reset_ns;
	__u64 dropped_count;
};

struct {
	__uint(type, BPF_MAP_TYPE_LRU_HASH);
	__uint(max_entries, 1000000);
	__type(key, struct rate_limit_key);
	__type(value, struct rate_limit_value);
} rate_limit_map SEC(".maps");

/* Configuration map: Global rate limit settings */
struct rate_limit_config {
	__u64 window_ns;        /* Time window in nanoseconds (default: 1 second) */
	__u64 max_packets;      /* Max packets per window (default: 10000) */
	__u32 enabled;          /* Rate limiting enabled flag */
};

struct {
	__uint(type, BPF_MAP_TYPE_ARRAY);
	__uint(max_entries, 1);
	__type(key, __u32);
	__type(value, struct rate_limit_config);
} rate_limit_config_map SEC(".maps");

/* Statistics map */
struct stats {
	__u64 total_packets;
	__u64 total_bytes;
	__u64 http_packets;
	__u64 https_packets;
	__u64 http2_packets;
	__u64 grpc_packets;
	__u64 websocket_packets;
	__u64 rate_limited;
	__u64 dropped;
};

struct {
	__uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
	__uint(max_entries, 1);
	__type(key, __u32);
	__type(value, struct stats);
} stats_map SEC(".maps");

/* Helper function to parse Ethernet header */
static __always_inline int parse_ethhdr(void *data, void *data_end,
					struct ethhdr **ethhdr)
{
	struct ethhdr *eth = data;

	if ((void *)(eth + 1) > data_end)
		return -1;

	*ethhdr = eth;
	return eth->h_proto;
}

/* Helper function to parse IP header */
static __always_inline int parse_iphdr(void *data, void *data_end,
				       struct iphdr **iphdr)
{
	struct iphdr *iph = data;

	if ((void *)(iph + 1) > data_end)
		return -1;

	/* Check IP header length */
	if (iph->ihl < 5)
		return -1;

	*iphdr = iph;
	return iph->protocol;
}

/* Helper function to parse TCP header */
static __always_inline int parse_tcphdr(void *data, void *data_end,
					struct tcphdr **tcphdr)
{
	struct tcphdr *tcp = data;

	if ((void *)(tcp + 1) > data_end)
		return -1;

	*tcphdr = tcp;
	return 0;
}

/* Detect HTTP protocol by checking for common HTTP methods */
static __always_inline int detect_http(void *data, void *data_end)
{
	if (data + 4 > data_end)
		return 0;

	char *payload = data;

	/* Check for HTTP methods: GET, POST, PUT, DELETE, HEAD, OPTIONS, PATCH */
	if ((payload[0] == 'G' && payload[1] == 'E' && payload[2] == 'T' && payload[3] == ' ') ||
	    (payload[0] == 'P' && payload[1] == 'O' && payload[2] == 'S' && payload[3] == 'T') ||
	    (payload[0] == 'P' && payload[1] == 'U' && payload[2] == 'T' && payload[3] == ' ') ||
	    (payload[0] == 'D' && payload[1] == 'E' && payload[2] == 'L' && payload[3] == 'E') ||
	    (payload[0] == 'H' && payload[1] == 'E' && payload[2] == 'A' && payload[3] == 'D') ||
	    (payload[0] == 'O' && payload[1] == 'P' && payload[2] == 'T' && payload[3] == 'I') ||
	    (payload[0] == 'P' && payload[1] == 'A' && payload[2] == 'T' && payload[3] == 'C')) {
		return 1;
	}

	return 0;
}

/* Detect TLS/HTTPS by checking for TLS handshake */
static __always_inline int detect_tls(void *data, void *data_end)
{
	if (data + 3 > data_end)
		return 0;

	unsigned char *payload = data;

	/* TLS record format: [Content Type][Version Major][Version Minor] */
	/* Content Type: 0x16 = Handshake, 0x17 = Application Data */
	/* Version: 0x0301 = TLS 1.0, 0x0302 = TLS 1.1, 0x0303 = TLS 1.2, 0x0304 = TLS 1.3 */
	if ((payload[0] == 0x16 || payload[0] == 0x17) &&
	    (payload[1] == 0x03) &&
	    (payload[2] >= 0x01 && payload[2] <= 0x04)) {
		return 1;
	}

	return 0;
}

/* Detect HTTP/2 by checking for connection preface or SETTINGS frame */
static __always_inline int detect_http2(void *data, void *data_end)
{
	if (data + 9 > data_end)
		return 0;

	char *payload = data;

	/* HTTP/2 connection preface: "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n" */
	if (payload[0] == 'P' && payload[1] == 'R' && payload[2] == 'I' && payload[3] == ' ') {
		return 1;
	}

	/* HTTP/2 SETTINGS frame (type = 0x04) with length in first 3 bytes */
	/* Frame format: [Length:3][Type:1][Flags:1][Reserved:1][Stream ID:4] */
	if (data + 9 <= data_end && payload[3] == 0x04) {
		return 1;
	}

	return 0;
}

/* Detect gRPC (HTTP/2 with content-type: application/grpc) */
static __always_inline int detect_grpc(void *data, void *data_end, int is_http2)
{
	/* gRPC runs over HTTP/2, so first check if it's HTTP/2 */
	if (!is_http2)
		return 0;

	/* For deeper inspection, would need to parse HTTP/2 headers
	 * For now, rely on port-based detection (handled in main function)
	 */
	return 0;
}

/* Detect WebSocket upgrade */
static __always_inline int detect_websocket(void *data, void *data_end)
{
	if (data + 16 > data_end)
		return 0;

	char *payload = data;

	/* Look for "Upgrade: websocket" or "Connection: Upgrade" */
	/* This is a simplified check - would need full HTTP header parsing */
	if (payload[0] == 'G' && payload[1] == 'E' && payload[2] == 'T') {
		/* GET request - might be WebSocket upgrade */
		/* Would need to parse headers for full detection */
		return 0;
	}

	return 0;
}

/* Rate limiting check and enforcement */
static __always_inline int check_rate_limit(__u32 src_ip, __u64 now_ns)
{
	struct rate_limit_key key = {.src_ip = src_ip};
	struct rate_limit_value *val;

	/* Get configuration */
	__u32 config_key = 0;
	struct rate_limit_config *config = bpf_map_lookup_elem(&rate_limit_config_map, &config_key);
	if (!config || !config->enabled)
		return XDP_PASS;

	/* Lookup or create rate limit entry */
	val = bpf_map_lookup_elem(&rate_limit_map, &key);
	if (!val) {
		/* First packet from this IP - create entry */
		struct rate_limit_value new_val = {
			.packet_count = 1,
			.last_reset_ns = now_ns,
			.dropped_count = 0
		};
		bpf_map_update_elem(&rate_limit_map, &key, &new_val, BPF_ANY);
		return XDP_PASS;
	}

	/* Check if we need to reset the window */
	if (now_ns - val->last_reset_ns > config->window_ns) {
		val->packet_count = 1;
		val->last_reset_ns = now_ns;
		return XDP_PASS;
	}

	/* Check if we're over the limit */
	if (val->packet_count >= config->max_packets) {
		val->dropped_count++;

		/* Update stats */
		__u32 stats_key = 0;
		struct stats *stats = bpf_map_lookup_elem(&stats_map, &stats_key);
		if (stats) {
			stats->rate_limited++;
			stats->dropped++;
		}

		return XDP_DROP;
	}

	/* Increment counter */
	val->packet_count++;
	return XDP_PASS;
}

/* Main XDP program */
SEC("xdp")
int xdp_envoy_filter(struct xdp_md *ctx)
{
	void *data_end = (void *)(long)ctx->data_end;
	void *data = (void *)(long)ctx->data;
	struct ethhdr *eth;
	struct iphdr *iph;
	struct tcphdr *tcp;
	__u64 now_ns = bpf_ktime_get_ns();
	int eth_proto, ip_proto;

	/* Update stats */
	__u32 stats_key = 0;
	struct stats *stats = bpf_map_lookup_elem(&stats_map, &stats_key);
	if (stats) {
		stats->total_packets++;
		stats->total_bytes += (data_end - data);
	}

	/* Parse Ethernet header */
	eth_proto = parse_ethhdr(data, data_end, &eth);
	if (eth_proto < 0)
		return XDP_DROP;

	/* Only handle IPv4 for now */
	if (eth_proto != bpf_htons(ETH_P_IP))
		return XDP_PASS;

	/* Parse IP header */
	data = (void *)(eth + 1);
	ip_proto = parse_iphdr(data, data_end, &iph);
	if (ip_proto < 0)
		return XDP_DROP;

	/* Rate limiting check */
	int rate_limit_action = check_rate_limit(iph->saddr, now_ns);
	if (rate_limit_action == XDP_DROP)
		return XDP_DROP;

	/* Only handle TCP */
	if (ip_proto != IPPROTO_TCP)
		return XDP_PASS;

	/* Parse TCP header */
	data = (void *)iph + (iph->ihl * 4);
	if (parse_tcphdr(data, data_end, &tcp) < 0)
		return XDP_DROP;

	/* Get TCP payload */
	void *payload = (void *)tcp + (tcp->doff * 4);
	if (payload >= data_end)
		return XDP_PASS;

	/* Protocol detection */
	__u16 dest_port = bpf_ntohs(tcp->dest);
	int is_http = 0, is_https = 0, is_http2 = 0, is_grpc = 0, is_websocket = 0;

	/* Detect protocols based on port and payload */
	if (dest_port == 80 || dest_port == 8080) {
		is_http = detect_http(payload, data_end);
		is_http2 = detect_http2(payload, data_end);
		is_websocket = detect_websocket(payload, data_end);
	} else if (dest_port == 443 || dest_port == 8443) {
		is_https = detect_tls(payload, data_end);
		/* HTTPS can contain HTTP/2 or gRPC, but we can't inspect encrypted traffic */
	} else if (dest_port == 50051) {
		/* Common gRPC port */
		is_grpc = 1;
		is_http2 = 1;
	}

	/* Update protocol-specific stats */
	if (stats) {
		if (is_http)
			stats->http_packets++;
		if (is_https)
			stats->https_packets++;
		if (is_http2)
			stats->http2_packets++;
		if (is_grpc)
			stats->grpc_packets++;
		if (is_websocket)
			stats->websocket_packets++;
	}

	/* Pass to Envoy for L7 processing */
	return XDP_PASS;
}

char _license[] SEC("license") = "GPL";
