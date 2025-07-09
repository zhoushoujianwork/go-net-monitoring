#!/bin/sh

# Docker entrypoint script

set -e

# 显示启动信息
echo "=== 网络监控系统启动 ==="
echo "组件: ${COMPONENT:-unknown}"
echo "版本: ${VERSION:-dev}"
echo "时间: $(date)"
echo "=========================="

# 根据组件类型启动不同的程序
case "${COMPONENT}" in
    "server")
        echo "启动Server..."
        exec ./server --config configs/server.yaml
        ;;
    "agent-ebpf")
        echo "启动eBPF Agent..."
        exec ./agent-ebpf --config configs/agent.yaml
        ;;
    *)
        echo "错误: 未知的组件类型: ${COMPONENT}"
        echo "支持的组件: server, agent, agent-ebpf"
        exit 1
        ;;
esac
