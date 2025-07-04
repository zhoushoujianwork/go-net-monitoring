# 容器化部署说明

## 🎯 为什么只支持容器化部署？

### 技术挑战

#### 1. **CGO依赖复杂性**
- Agent需要调用libpcap C库进行网络包捕获
- 不同操作系统需要不同的编译环境和依赖库
- 交叉编译CGO程序需要目标平台的C编译器

#### 2. **平台差异**
```
Windows:  gopacket使用纯Go实现 ✅
Linux:    需要libpcap开发库 ⚠️
macOS:    需要libpcap开发库 ⚠️
```

#### 3. **依赖管理困难**
- 编译时依赖: libpcap-dev, gcc, 头文件
- 运行时依赖: libpcap运行时库
- 版本兼容性问题

### 容器化优势

#### 1. **环境一致性**
```bash
# 所有平台统一的部署方式
docker run -d \
  --name netmon-agent \
  --privileged \
  --network host \
  zhoushoujian/go-net-monitoring:latest
```

#### 2. **依赖封装**
- ✅ 所有依赖都打包在镜像中
- ✅ 无需用户安装libpcap等依赖
- ✅ 避免版本冲突问题

#### 3. **部署简化**
- ✅ 一条命令启动服务
- ✅ 支持Docker Compose编排
- ✅ 支持Kubernetes部署

#### 4. **维护便利**
- ✅ 统一的构建流程
- ✅ 自动化CI/CD
- ✅ 版本管理简单

## 🚀 部署方式

### 1. Docker Compose (推荐)

```bash
# 克隆项目
git clone https://github.com/zhoushoujianwork/go-net-monitoring.git
cd go-net-monitoring

# 构建并启动
make docker-build
make docker-up

# 查看状态
make health
```

### 2. 单独容器

```bash
# Server
docker run -d \
  --name netmon-server \
  -p 8080:8080 \
  -e COMPONENT=server \
  zhoushoujian/go-net-monitoring:latest

# Agent
docker run -d \
  --name netmon-agent \
  --privileged \
  --network host \
  -e COMPONENT=agent \
  -e SERVER_URL=http://localhost:8080/api/v1/metrics \
  zhoushoujian/go-net-monitoring:latest
```

### 3. Kubernetes

```bash
# 部署到K8s集群
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/server-deployment.yaml
kubectl apply -f k8s/agent-daemonset.yaml
```

## 🔧 开发模式

### 本地开发

```bash
# 启动调试模式
make docker-up-debug

# 查看日志
make docker-logs

# 进入容器调试
make dev-shell-agent
make dev-shell-server
```

### 代码修改

```bash
# 修改代码后重新构建
make docker-build
make docker-restart
```

## 📊 对比分析

| 方面 | 二进制部署 | 容器化部署 |
|------|------------|------------|
| **依赖管理** | ❌ 复杂 | ✅ 简单 |
| **跨平台** | ❌ 困难 | ✅ 统一 |
| **部署难度** | ❌ 高 | ✅ 低 |
| **维护成本** | ❌ 高 | ✅ 低 |
| **环境一致性** | ❌ 差 | ✅ 好 |
| **资源占用** | ✅ 低 | ⚠️ 稍高 |

## 🎯 最佳实践

### 生产环境

```bash
# 使用Docker Compose
docker-compose up -d

# 或使用Kubernetes
kubectl apply -f k8s/
```

### 开发环境

```bash
# 调试模式
make docker-up-debug
make docker-logs
```

### 监控集成

```bash
# 完整监控栈
make docker-up-monitoring
# 包含Prometheus + Grafana
```

## 🔍 故障排查

### 常见问题

#### 1. 权限问题
```bash
# Agent需要特权模式
docker run --privileged ...
```

#### 2. 网络问题
```bash
# Agent需要host网络
docker run --network host ...
```

#### 3. 端口冲突
```bash
# 检查端口占用
netstat -tlnp | grep 8080
```

### 调试技巧

```bash
# 查看容器日志
docker logs netmon-agent

# 进入容器调试
docker exec -it netmon-agent /bin/sh

# 检查网络接口
docker exec netmon-agent ip link show
```

## 📝 总结

**容器化部署是当前最佳选择：**

✅ **优点:**
- 环境一致性
- 部署简化
- 依赖封装
- 维护便利

⚠️ **权衡:**
- 稍高的资源占用
- 需要Docker环境

🎯 **结论:**
容器化部署完美解决了CGO依赖和跨平台问题，是现代应用部署的最佳实践。
