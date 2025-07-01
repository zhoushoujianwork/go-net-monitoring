# 部署指南

## 系统要求

### 最低要求
- 操作系统: Linux (Ubuntu 18.04+, CentOS 7+, RHEL 7+)
- 内存: 512MB
- 磁盘: 100MB
- 网络: 支持TCP/UDP协议

### 推荐配置
- 操作系统: Linux (Ubuntu 20.04+, CentOS 8+)
- 内存: 1GB+
- 磁盘: 1GB+
- CPU: 2核心+

## 权限要求

Agent需要以下权限：
- 网络数据包捕获权限（通常需要root权限或CAP_NET_RAW能力）
- 读取网络连接信息权限（/proc/net/）
- 文件系统读写权限（日志、配置文件）

## 安装方式

### 1. 二进制安装

```bash
# 下载二进制文件
wget https://github.com/your-org/go-net-monitoring/releases/download/v1.0.0/go-net-monitoring-linux-amd64.tar.gz

# 解压
tar -xzf go-net-monitoring-linux-amd64.tar.gz

# 移动到系统目录
sudo mv go-net-monitoring /opt/
sudo ln -s /opt/go-net-monitoring/bin/agent /usr/local/bin/network-agent
sudo ln -s /opt/go-net-monitoring/bin/server /usr/local/bin/network-server
```

### 2. 源码编译

```bash
# 克隆代码
git clone https://github.com/your-org/go-net-monitoring.git
cd go-net-monitoring

# 安装依赖
make deps

# 编译
make build

# 安装
sudo make install
```

### 3. Docker部署

```bash
# 构建镜像
make docker-build

# 运行Agent
docker run -d --name network-agent \
  --privileged \
  --network host \
  -v /proc:/host/proc:ro \
  -v $(pwd)/configs:/app/configs \
  go-net-monitoring:latest agent --config /app/configs/agent.yaml

# 运行Server
docker run -d --name network-server \
  -p 8080:8080 \
  -v $(pwd)/configs:/app/configs \
  go-net-monitoring:latest server --config /app/configs/server.yaml
```

## 配置部署

### 1. 创建配置目录

```bash
sudo mkdir -p /etc/network-monitoring
sudo cp configs/agent.yaml /etc/network-monitoring/
sudo cp configs/server.yaml /etc/network-monitoring/
```

### 2. 修改配置文件

根据实际环境修改配置文件：

```bash
sudo vim /etc/network-monitoring/agent.yaml
sudo vim /etc/network-monitoring/server.yaml
```

### 3. 创建日志目录

```bash
sudo mkdir -p /var/log/network-monitoring
sudo chown -R network-monitoring:network-monitoring /var/log/network-monitoring
```

## 服务管理

### 1. Systemd服务

创建Agent服务文件：

```bash
sudo tee /etc/systemd/system/network-agent.service > /dev/null <<EOF
[Unit]
Description=Network Monitoring Agent
After=network.target

[Service]
Type=simple
User=root
Group=root
ExecStart=/usr/local/bin/network-agent --config /etc/network-monitoring/agent.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
```

创建Server服务文件：

```bash
sudo tee /etc/systemd/system/network-server.service > /dev/null <<EOF
[Unit]
Description=Network Monitoring Server
After=network.target

[Service]
Type=simple
User=network-monitoring
Group=network-monitoring
ExecStart=/usr/local/bin/network-server --config /etc/network-monitoring/server.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
```

启动服务：

```bash
sudo systemctl daemon-reload
sudo systemctl enable network-agent
sudo systemctl enable network-server
sudo systemctl start network-agent
sudo systemctl start network-server
```

### 2. 服务状态检查

```bash
# 检查服务状态
sudo systemctl status network-agent
sudo systemctl status network-server

# 查看日志
sudo journalctl -u network-agent -f
sudo journalctl -u network-server -f
```

## 监控集成

### 1. Prometheus配置

在Prometheus配置文件中添加：

```yaml
scrape_configs:
  - job_name: 'network-monitoring'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 30s
```

### 2. Grafana仪表板

导入预配置的Grafana仪表板：

```bash
# 导入仪表板JSON文件
curl -X POST \
  http://admin:admin@localhost:3000/api/dashboards/db \
  -H 'Content-Type: application/json' \
  -d @examples/grafana-dashboard.json
```

## 高可用部署

### 1. Server集群

部署多个Server实例：

```bash
# Server 1
network-server --config /etc/network-monitoring/server1.yaml

# Server 2
network-server --config /etc/network-monitoring/server2.yaml
```

### 2. 负载均衡

使用Nginx进行负载均衡：

```nginx
upstream network-servers {
    server 127.0.0.1:8080;
    server 127.0.0.1:8081;
}

server {
    listen 80;
    location / {
        proxy_pass http://network-servers;
    }
}
```

## 安全配置

### 1. TLS配置

生成证书：

```bash
# 生成私钥
openssl genrsa -out server.key 2048

# 生成证书请求
openssl req -new -key server.key -out server.csr

# 生成自签名证书
openssl x509 -req -days 365 -in server.csr -signkey server.key -out server.crt
```

配置TLS：

```yaml
http:
  enable_tls: true
  tls_cert_path: "/etc/ssl/certs/server.crt"
  tls_key_path: "/etc/ssl/private/server.key"
```

### 2. 防火墙配置

```bash
# 允许Server端口
sudo ufw allow 8080/tcp

# 限制访问来源
sudo ufw allow from 192.168.1.0/24 to any port 8080
```

## 故障排除

### 1. 常见问题

**Agent无法启动**：
- 检查是否有root权限
- 检查网络接口是否存在
- 检查配置文件语法

**数据上报失败**：
- 检查网络连接
- 检查Server地址配置
- 检查防火墙设置

**性能问题**：
- 调整缓冲区大小
- 优化过滤规则
- 检查系统资源使用

### 2. 日志分析

```bash
# 查看详细日志
sudo journalctl -u network-agent --since "1 hour ago"

# 过滤错误日志
sudo journalctl -u network-agent -p err

# 实时监控日志
sudo tail -f /var/log/network-monitoring/agent.log
```

### 3. 性能监控

```bash
# 检查进程资源使用
top -p $(pgrep network-agent)

# 检查网络连接
netstat -tulpn | grep network

# 检查磁盘使用
df -h /var/log/network-monitoring
```

## 升级指南

### 1. 备份配置

```bash
sudo cp -r /etc/network-monitoring /etc/network-monitoring.backup
```

### 2. 停止服务

```bash
sudo systemctl stop network-agent
sudo systemctl stop network-server
```

### 3. 更新二进制文件

```bash
# 下载新版本
wget https://github.com/your-org/go-net-monitoring/releases/download/v1.1.0/go-net-monitoring-linux-amd64.tar.gz

# 备份旧版本
sudo mv /usr/local/bin/network-agent /usr/local/bin/network-agent.old
sudo mv /usr/local/bin/network-server /usr/local/bin/network-server.old

# 安装新版本
tar -xzf go-net-monitoring-linux-amd64.tar.gz
sudo cp bin/agent /usr/local/bin/network-agent
sudo cp bin/server /usr/local/bin/network-server
sudo chmod +x /usr/local/bin/network-*
```

### 4. 启动服务

```bash
sudo systemctl start network-agent
sudo systemctl start network-server
```

### 5. 验证升级

```bash
network-agent version
network-server version
sudo systemctl status network-agent
sudo systemctl status network-server
```
