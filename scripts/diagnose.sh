#!/bin/bash

echo "🔍 诊断Agent启动问题..."

# 构建项目
echo "📦 构建项目..."
make build

echo ""
echo "🧪 测试1: 使用测试模式启动Agent"
echo "这将避免网络捕获问题..."

# 启动Server
echo "启动Server..."
./bin/server --config configs/server.yaml &
SERVER_PID=$!
sleep 2

# 测试模式启动Agent
echo "启动Agent (测试模式)..."
TEST_MODE=true timeout 15s ./bin/agent --config configs/agent-simple.yaml &
AGENT_PID=$!

# 等待并观察
sleep 12

# 检查进程状态
if kill -0 $AGENT_PID 2>/dev/null; then
    echo "✅ Agent在测试模式下运行正常"
    kill $AGENT_PID 2>/dev/null
else
    echo "❌ Agent在测试模式下也出现问题"
fi

# 清理Server
kill $SERVER_PID 2>/dev/null

echo ""
echo "🧪 测试2: 检查网络权限"
echo "测试是否能够打开网络接口..."

# 测试网络接口访问
if sudo timeout 5s tcpdump -i en0 -c 1 >/dev/null 2>&1; then
    echo "✅ 网络接口访问正常"
else
    echo "❌ 网络接口访问有问题"
fi

echo ""
echo "🧪 测试3: 检查配置文件"
echo "验证配置文件解析..."

# 创建配置测试程序
cat > /tmp/config_test.go << 'EOF'
package main

import (
    "fmt"
    "log"
    "os"
    "time"
    "gopkg.in/yaml.v3"
)

type AgentConfig struct {
    Server struct {
        Host string `yaml:"host"`
        Port int    `yaml:"port"`
    } `yaml:"server"`
    Monitor struct {
        Interface      string        `yaml:"interface"`
        Protocols      []string      `yaml:"protocols"`
        ReportInterval time.Duration `yaml:"report_interval"`
        BufferSize     int           `yaml:"buffer_size"`
    } `yaml:"monitor"`
    Reporter struct {
        ServerURL   string        `yaml:"server_url"`
        Timeout     time.Duration `yaml:"timeout"`
        RetryCount  int           `yaml:"retry_count"`
        RetryDelay  time.Duration `yaml:"retry_delay"`
        BatchSize   int           `yaml:"batch_size"`
    } `yaml:"reporter"`
    Log struct {
        Level  string `yaml:"level"`
        Format string `yaml:"format"`
        Output string `yaml:"output"`
    } `yaml:"log"`
}

func main() {
    data, err := os.ReadFile("configs/agent-simple.yaml")
    if err != nil {
        log.Fatalf("读取配置文件失败: %v", err)
    }

    var config AgentConfig
    if err := yaml.Unmarshal(data, &config); err != nil {
        log.Fatalf("解析配置文件失败: %v", err)
    }

    fmt.Printf("配置解析成功:\n")
    fmt.Printf("  ReportInterval: %v\n", config.Monitor.ReportInterval)
    fmt.Printf("  Timeout: %v\n", config.Reporter.Timeout)
    fmt.Printf("  RetryDelay: %v\n", config.Reporter.RetryDelay)
    
    if config.Monitor.ReportInterval <= 0 {
        fmt.Printf("❌ ReportInterval无效: %v\n", config.Monitor.ReportInterval)
    } else {
        fmt.Printf("✅ ReportInterval有效: %v\n", config.Monitor.ReportInterval)
    }
}
EOF

cd /Users/mikas/Documents/opentelemetryJaeger/go-net-monitoring
if go run /tmp/config_test.go; then
    echo "✅ 配置文件解析正常"
else
    echo "❌ 配置文件解析有问题"
fi

rm /tmp/config_test.go

echo ""
echo "🔍 诊断完成"
echo "如果测试模式正常但真实模式有问题，可能是网络权限或libpcap问题"
echo "如果配置解析有问题，需要修复配置文件格式"
