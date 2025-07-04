#!/bin/bash

echo "=== 简单网络流量测试 ==="

# 清除之前的指标（重启Agent）
echo "1. 重启Agent以清除旧数据..."
cd /data/github/go-net-monitoring
docker-compose restart agent
sleep 10

echo "2. 检查初始状态..."
echo "发送字节数："
curl -s http://localhost:8080/metrics | grep "network_domain_bytes_sent_total" | grep -v " 0$" | wc -l

echo "接收字节数："
curl -s http://localhost:8080/metrics | grep "network_domain_bytes_received_total" | grep -v " 0$" | wc -l

echo ""
echo "3. 发送一个大的HTTP请求..."
# 发送一个会产生大响应的请求
curl -s "https://httpbin.org/json" > /tmp/response.json
echo "响应大小: $(wc -c < /tmp/response.json) 字节"

echo ""
echo "4. 等待数据收集..."
sleep 15

echo ""
echo "5. 检查结果..."
echo "发送字节数（非零）："
curl -s http://localhost:8080/metrics | grep "network_domain_bytes_sent_total" | grep -v " 0$" | wc -l

echo "接收字节数（非零）："
curl -s http://localhost:8080/metrics | grep "network_domain_bytes_received_total" | grep -v " 0$" | wc -l

echo ""
echo "6. 详细的接收字节统计："
curl -s http://localhost:8080/metrics | grep "network_domain_bytes_received_total" | grep -v " 0$" | head -3

echo ""
echo "=== 测试完成 ==="
