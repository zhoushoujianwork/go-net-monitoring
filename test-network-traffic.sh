#!/bin/bash

echo "=== 网络流量测试脚本 ==="
echo "生成一些网络活动来测试入站和出站流量统计..."

# 生成出站流量
echo "1. 生成出站流量..."
curl -s https://httpbin.org/get > /dev/null 2>&1 &
curl -s https://api.github.com/users/octocat > /dev/null 2>&1 &
curl -s https://www.google.com > /dev/null 2>&1 &

# 等待请求完成
sleep 5

echo "2. 等待数据收集..."
sleep 15

echo "3. 检查发送字节统计..."
curl -s http://localhost:8080/metrics | grep "network_domain_bytes_sent_total" | grep -v " 0$" | head -5

echo ""
echo "4. 检查接收字节统计..."
curl -s http://localhost:8080/metrics | grep "network_domain_bytes_received_total" | grep -v " 0$" | head -5

echo ""
echo "5. 检查域名访问统计..."
curl -s http://localhost:8080/metrics | grep "network_domains_accessed_total" | grep -v " 0$" | head -5

echo ""
echo "=== 测试完成 ==="
