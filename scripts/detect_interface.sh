#!/bin/bash

echo "🔍 检测Mac网络接口..."

echo ""
echo "📡 所有网络接口:"
ifconfig | grep -E "^[a-z]" | grep -v "lo0" | cut -d: -f1

echo ""
echo "🌐 活跃的网络接口 (有IP地址):"
for interface in $(ifconfig | grep -E "^en[0-9]:" | cut -d: -f1); do
    ip=$(ifconfig $interface | grep "inet " | grep -v "127.0.0.1" | awk '{print $2}')
    if [ ! -z "$ip" ]; then
        echo "  $interface: $ip"
    fi
done

echo ""
echo "🚀 默认路由接口:"
default_interface=$(route -n get default | grep interface | awk '{print $2}')
default_ip=$(ifconfig $default_interface | grep "inet " | grep -v "127.0.0.1" | awk '{print $2}')
echo "  $default_interface: $default_ip"

echo ""
echo "💡 建议配置:"
echo "  推荐使用接口: $default_interface"
echo "  本机IP地址: $default_ip"
echo "  在agent.yaml中设置:"
echo "    monitor:"
echo "      interface: \"$default_interface\""
echo "    filters:"
echo "      ignore_ips:"
echo "        - \"127.0.0.1\""
echo "        - \"::1\""
echo "        - \"$default_ip\""

echo ""
echo "🔧 测试网络接口权限:"
if [ "$EUID" -eq 0 ]; then
    echo "  ✅ 当前以root权限运行，可以进行网络监控"
else
    echo "  ⚠️  需要sudo权限进行网络数据包捕获"
    echo "  运行命令: sudo $0"
fi
