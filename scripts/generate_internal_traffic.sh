#!/bin/bash

echo "🏠 生成内网流量用于测试..."

# 常见内网IP范围
INTERNAL_RANGES=(
    "192.168.1"
    "192.168.0" 
    "192.168.17"
    "10.0.0"
    "10.0.1"
    "172.16.0"
)

# 常见服务端口
PORTS=(22 80 443 3306 5432 6379 8080 9200)

echo "扫描内网服务..."

for range in "${INTERNAL_RANGES[@]}"; do
    echo "扫描 ${range}.x 网段..."
    
    for i in {1..10}; do
        ip="${range}.${i}"
        
        # 跳过本机IP
        if [ "$ip" = "192.168.17.92" ]; then
            continue
        fi
        
        # 测试几个常见端口
        for port in "${PORTS[@]}"; do
            timeout 1 nc -z "$ip" "$port" 2>/dev/null && {
                echo "✅ 发现服务: $ip:$port"
            } &
        done
        
        # 限制并发数
        if [ $((i % 5)) -eq 0 ]; then
            wait
        fi
    done
done

# 等待所有后台任务完成
wait

echo "🔍 尝试访问常见内网服务..."

# 路由器管理界面
curl -m 2 -s http://192.168.1.1 >/dev/null 2>&1 && echo "访问了路由器管理界面"
curl -m 2 -s http://192.168.0.1 >/dev/null 2>&1 && echo "访问了路由器管理界面"
curl -m 2 -s http://192.168.17.1 >/dev/null 2>&1 && echo "访问了网关"

# 常见内网服务
curl -m 2 -s http://192.168.1.100:8080 >/dev/null 2>&1 && echo "访问了内网Web服务"
curl -m 2 -s http://10.0.0.100 >/dev/null 2>&1 && echo "访问了10.x网段服务"

echo "✅ 内网流量生成完成"
