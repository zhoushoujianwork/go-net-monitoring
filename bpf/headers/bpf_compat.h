#ifndef __BPF_COMPAT_H__
#define __BPF_COMPAT_H__

// macOS 兼容的 eBPF 头文件定义

// 基础类型定义
typedef unsigned char __u8;
typedef unsigned short __u16;
typedef unsigned int __u32;
typedef unsigned long long __u64;

// BPF Map 类型定义
#define BPF_MAP_TYPE_PERCPU_ARRAY 6

// XDP 动作定义
#define XDP_ABORTED 0
#define XDP_DROP    1
#define XDP_PASS    2
#define XDP_TX      3
#define XDP_REDIRECT 4

// 以太网协议类型
#define ETH_P_IP 0x0800

// IP 协议类型
#define IPPROTO_TCP 6
#define IPPROTO_UDP 17

// 网络字节序转换
#define bpf_htons(x) __builtin_bswap16(x)
#define bpf_ntohs(x) __builtin_bswap16(x)

// BPF 辅助函数声明
static void *(*bpf_map_lookup_elem)(void *map, const void *key) = (void *) 1;

// 简化的原子操作（仅用于编译测试）
#define __sync_fetch_and_add(ptr, val) ({ \
    *(ptr) += (val); \
})

// Section 定义
#define SEC(name) __attribute__((section(name), used))

// Map 定义宏
#define __uint(name, val) int (*name)[val]
#define __type(name, val) typeof(val) *name

// XDP 上下文结构
struct xdp_md {
    __u32 data;
    __u32 data_end;
    __u32 data_meta;
    __u32 ingress_ifindex;
    __u32 rx_queue_index;
};

// 以太网头部
struct ethhdr {
    unsigned char h_dest[6];
    unsigned char h_source[6];
    __u16 h_proto;
} __attribute__((packed));

// IP 头部
struct iphdr {
    __u8 ihl:4,
         version:4;
    __u8 tos;
    __u16 tot_len;
    __u16 id;
    __u16 frag_off;
    __u8 ttl;
    __u8 protocol;
    __u16 check;
    __u32 saddr;
    __u32 daddr;
} __attribute__((packed));

#endif /* __BPF_COMPAT_H__ */
