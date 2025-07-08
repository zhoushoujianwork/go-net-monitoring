//go:build ignore

#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/in.h>
#include <linux/tcp.h>
#include <linux/udp.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

// 包统计结构
struct packet_stats {
    __u64 total_packets;
    __u64 total_bytes;
    __u64 tcp_packets;
    __u64 udp_packets;
    __u64 other_packets;
};

// BPF Map: 存储包统计信息
struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(max_entries, 1);
    __type(key, __u32);
    __type(value, struct packet_stats);
} packet_stats_map SEC(".maps");

// XDP 程序入口点
SEC("xdp")
int xdp_packet_monitor(struct xdp_md *ctx) {
    void *data_end = (void *)(long)ctx->data_end;
    void *data = (void *)(long)ctx->data;
    
    // 检查以太网头部
    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return XDP_PASS;
    
    // 只处理 IP 包
    if (eth->h_proto != bpf_htons(ETH_P_IP))
        return XDP_PASS;
    
    // 检查 IP 头部
    struct iphdr *ip = (void *)(eth + 1);
    if ((void *)(ip + 1) > data_end)
        return XDP_PASS;
    
    // 获取统计信息
    __u32 key = 0;
    struct packet_stats *stats = bpf_map_lookup_elem(&packet_stats_map, &key);
    if (!stats)
        return XDP_PASS;
    
    // 更新统计信息
    __u64 packet_size = data_end - data;
    __sync_fetch_and_add(&stats->total_packets, 1);
    __sync_fetch_and_add(&stats->total_bytes, packet_size);
    
    // 根据协议类型分类
    switch (ip->protocol) {
        case IPPROTO_TCP:
            __sync_fetch_and_add(&stats->tcp_packets, 1);
            break;
        case IPPROTO_UDP:
            __sync_fetch_and_add(&stats->udp_packets, 1);
            break;
        default:
            __sync_fetch_and_add(&stats->other_packets, 1);
            break;
    }
    
    return XDP_PASS;
}

char _license[] SEC("license") = "GPL";
