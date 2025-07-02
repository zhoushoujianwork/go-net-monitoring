#!/bin/bash

echo "=== 网络监控能力测试 ==="

echo "1. 检查宿主机网络接口："
ip link show | grep -E "^[0-9]+:" | awk '{print $2}' | sed 's/://'

echo -e "\n2. 测试容器网络监控能力："

# 启动一个临时容器来测试网络监控
echo "启动测试容器..."
docker run --rm -it \
  --network host \
  --privileged \
  --cap-add=NET_ADMIN \
  --cap-add=NET_RAW \
  zhoushoujian/go-net-monitoring:latest \
  sh -c "
    echo '检查容器内网络接口：'
    ip link show | grep -E '^[0-9]+:' | awk '{print \$2}' | sed 's/://'
    
    echo -e '\n检查是否可以使用tcpdump：'
    if command -v tcpdump >/dev/null 2>&1; then
        echo 'tcpdump 可用'
        echo '测试抓包能力（5秒）：'
        timeout 5s tcpdump -i eth0 -c 5 2>/dev/null || echo '抓包失败'
    else
        echo 'tcpdump 不可用'
    fi
    
    echo -e '\n检查是否可以使用ss命令：'
    if command -v ss >/dev/null 2>&1; then
        echo 'ss 命令可用'
        echo '当前网络连接数：'
        ss -tuln | wc -l
    else
        echo 'ss 命令不可用'
    fi
    
    echo -e '\n检查网络统计：'
    if [ -f /proc/net/dev ]; then
        echo '网络接口统计：'
        cat /proc/net/dev | head -5
    fi
  "
