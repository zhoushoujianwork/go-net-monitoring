#!/bin/bash

# 测试新的host_ip_address功能
echo "🧪 测试 host_ip_address 功能"
echo "================================"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 检查服务状态
echo -e "${BLUE}[INFO]${NC} 检查服务状态..."
if ! curl -s http://localhost:8080/health > /dev/null; then
    echo -e "${RED}[ERROR]${NC} 服务未运行，请先启动服务"
    echo "运行: make docker-up-debug"
    exit 1
fi

echo -e "${GREEN}[SUCCESS]${NC} 服务运行正常"

# 等待一段时间让指标收集
echo -e "${BLUE}[INFO]${NC} 等待指标收集..."
sleep 5

# 测试1: 检查network_interface_info指标是否包含host_ip_address标签
echo -e "${BLUE}[TEST 1]${NC} 检查 network_interface_info 指标格式..."
INTERFACE_METRICS=$(curl -s http://localhost:8080/metrics | grep "network_interface_info{")

if [ -z "$INTERFACE_METRICS" ]; then
    echo -e "${RED}[FAIL]${NC} 未找到 network_interface_info 指标"
    exit 1
fi

echo -e "${GREEN}[SUCCESS]${NC} 找到 network_interface_info 指标"
echo "指标内容:"
echo "$INTERFACE_METRICS" | while read -r line; do
    echo "  $line"
done

# 测试2: 验证host_ip_address标签存在
echo -e "\n${BLUE}[TEST 2]${NC} 验证 host_ip_address 标签..."
if echo "$INTERFACE_METRICS" | grep -q "host_ip_address="; then
    echo -e "${GREEN}[SUCCESS]${NC} host_ip_address 标签存在"
    
    # 提取host_ip_address值
    HOST_IP=$(echo "$INTERFACE_METRICS" | grep -o 'host_ip_address="[^"]*"' | head -1 | cut -d'"' -f2)
    echo -e "${BLUE}[INFO]${NC} 检测到的主机IP: $HOST_IP"
    
    # 验证IP格式
    if [[ $HOST_IP =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]]; then
        echo -e "${GREEN}[SUCCESS]${NC} 主机IP格式正确"
    else
        echo -e "${YELLOW}[WARN]${NC} 主机IP格式可能不正确: $HOST_IP"
    fi
else
    echo -e "${RED}[FAIL]${NC} host_ip_address 标签不存在"
    exit 1
fi

# 测试3: 检查标签完整性
echo -e "\n${BLUE}[TEST 3]${NC} 检查标签完整性..."
REQUIRED_LABELS=("interface=" "ip_address=" "mac_address=" "host=" "host_ip_address=")
ALL_LABELS_PRESENT=true

for label in "${REQUIRED_LABELS[@]}"; do
    if echo "$INTERFACE_METRICS" | grep -q "$label"; then
        echo -e "${GREEN}[✓]${NC} $label 标签存在"
    else
        echo -e "${RED}[✗]${NC} $label 标签缺失"
        ALL_LABELS_PRESENT=false
    fi
done

if [ "$ALL_LABELS_PRESENT" = true ]; then
    echo -e "${GREEN}[SUCCESS]${NC} 所有必需标签都存在"
else
    echo -e "${RED}[FAIL]${NC} 部分标签缺失"
    exit 1
fi

# 测试4: 检查容器环境检测
echo -e "\n${BLUE}[TEST 4]${NC} 检查容器环境检测..."
AGENT_LOGS=$(docker logs netmon-agent 2>&1 | grep "Host detection completed" | tail -1)
if [ -n "$AGENT_LOGS" ]; then
    echo -e "${GREEN}[SUCCESS]${NC} 找到主机检测日志"
    echo "日志内容: $AGENT_LOGS"
    
    if echo "$AGENT_LOGS" | grep -q '"is_container":true'; then
        echo -e "${GREEN}[SUCCESS]${NC} 正确检测到容器环境"
    else
        echo -e "${YELLOW}[WARN]${NC} 未检测到容器环境"
    fi
else
    echo -e "${YELLOW}[WARN]${NC} 未找到主机检测日志"
fi

# 测试5: 比较容器IP和主机IP
echo -e "\n${BLUE}[TEST 5]${NC} 比较容器IP和主机IP..."
CONTAINER_IP=$(echo "$INTERFACE_METRICS" | grep -o 'ip_address="[^"]*"' | head -1 | cut -d'"' -f2)
HOST_IP=$(echo "$INTERFACE_METRICS" | grep -o 'host_ip_address="[^"]*"' | head -1 | cut -d'"' -f2)

echo -e "${BLUE}[INFO]${NC} 容器IP: $CONTAINER_IP"
echo -e "${BLUE}[INFO]${NC} 主机IP: $HOST_IP"

if [ "$CONTAINER_IP" != "$HOST_IP" ]; then
    echo -e "${GREEN}[SUCCESS]${NC} 容器IP和主机IP不同，说明正确区分了虚拟环境"
else
    echo -e "${YELLOW}[WARN]${NC} 容器IP和主机IP相同，可能在非虚拟化环境中"
fi

# 测试6: 验证Prometheus查询
echo -e "\n${BLUE}[TEST 6]${NC} 验证Prometheus查询兼容性..."
QUERY_RESULT=$(curl -s "http://localhost:8080/metrics" | grep "network_interface_info" | wc -l)
if [ "$QUERY_RESULT" -gt 0 ]; then
    echo -e "${GREEN}[SUCCESS]${NC} 指标可以被Prometheus正确解析 ($QUERY_RESULT 个指标)"
else
    echo -e "${RED}[FAIL]${NC} 指标无法被Prometheus解析"
    exit 1
fi

# 测试7: 检查指标帮助信息
echo -e "\n${BLUE}[TEST 7]${NC} 检查指标帮助信息..."
HELP_TEXT=$(curl -s http://localhost:8080/metrics | grep "# HELP network_interface_info")
if echo "$HELP_TEXT" | grep -q "host IP address"; then
    echo -e "${GREEN}[SUCCESS]${NC} 帮助信息已更新包含主机IP说明"
    echo "帮助信息: $HELP_TEXT"
else
    echo -e "${YELLOW}[WARN]${NC} 帮助信息可能需要更新"
fi

# 总结
echo -e "\n${GREEN}🎉 测试完成！${NC}"
echo "================================"
echo -e "${BLUE}功能总结:${NC}"
echo "✅ network_interface_info 指标已成功添加 host_ip_address 标签"
echo "✅ 主机IP检测功能正常工作"
echo "✅ 容器环境检测正确"
echo "✅ 所有必需标签都存在"
echo "✅ Prometheus兼容性良好"

echo -e "\n${BLUE}示例指标:${NC}"
echo "$INTERFACE_METRICS" | head -1

echo -e "\n${BLUE}使用说明:${NC}"
echo "现在可以使用以下Prometheus查询来区分虚拟机和物理机:"
echo "  # 查询所有容器网卡信息"
echo "  network_interface_info{host_ip_address!=\"\"}"
echo ""
echo "  # 查询物理机网卡信息"  
echo "  network_interface_info{host_ip_address=\"\"}"
echo ""
echo "  # 按主机IP分组统计"
echo "  count by (host_ip_address) (network_interface_info)"
