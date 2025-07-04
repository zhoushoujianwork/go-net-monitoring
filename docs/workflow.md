# 标准工作流程

## 🎯 概述

本文档定义了go-net-monitoring项目的标准开发、构建、测试和部署流程，确保团队协作的一致性和效率。

## 🔄 开发流程

### 1. 环境准备

#### 初始设置
```bash
# 克隆项目
git clone https://github.com/zhoushoujianwork/go-net-monitoring.git
cd go-net-monitoring

# 设置开发环境
make dev-setup
```

#### 依赖检查
```bash
# 检查Go版本 (需要1.19+)
go version

# 检查Docker
docker --version
docker-compose --version

# 检查必要工具
make version
```

### 2. 开发周期

#### 2.1 创建功能分支
```bash
# 从main分支创建功能分支
git checkout main
git pull origin main
git checkout -b feature/your-feature-name
```

#### 2.2 本地开发
```bash
# 启动开发环境
make dev-run-server    # 终端1: 运行Server
make dev-run-agent     # 终端2: 运行Agent (需要sudo)

# 或使用Docker开发环境
make docker-up-debug   # 启动调试模式
make docker-logs       # 查看日志
```

#### 2.3 代码测试
```bash
# 运行单元测试
make test

# 生成覆盖率报告
make test-coverage

# 检查代码质量
go fmt ./...
go vet ./...
```

#### 2.4 构建验证
```bash
# 本地构建验证
make build

# Docker构建验证
make build-optimized

# 启动集成测试
make docker-up-debug
make health
make metrics
```

### 3. 提交流程

#### 3.1 代码提交
```bash
# 添加更改
git add .

# 提交 (使用规范的提交信息)
git commit -m "feat: 添加新功能描述"
git commit -m "fix: 修复问题描述"
git commit -m "docs: 更新文档"
git commit -m "refactor: 重构代码"
```

#### 3.2 推送和PR
```bash
# 推送到远程分支
git push origin feature/your-feature-name

# 在GitHub上创建Pull Request
# 1. 填写PR模板
# 2. 添加相关标签
# 3. 请求代码审查
```

## 🏗️ 构建流程

### 1. 标准构建 (推荐)

#### 优化构建
```bash
# 一键优化构建
make build-optimized

# 特性:
# - 构建速度提升60%
# - 镜像大小减少30%
# - 避免重复构建
# - 并行编译
```

#### 构建选项
```bash
make build-clean       # 清理缓存后构建
make build-test        # 构建并运行测试
```

### 2. 传统构建

#### 本地构建
```bash
make build             # 构建二进制文件
make build-agent       # 只构建Agent
make build-server      # 只构建Server
```

#### Docker构建
```bash
docker build -t go-net-monitoring .
```

### 3. 构建验证

```bash
# 检查构建产物
ls -la bin/

# 验证镜像
docker images go-net-monitoring

# 测试运行
make docker-up
make health
```

## 🧪 测试流程

### 1. 单元测试

```bash
# 运行所有测试
make test

# 运行特定包测试
go test ./pkg/collector/...
go test ./internal/agent/...
```

### 2. 集成测试

```bash
# 启动测试环境
make docker-up-debug

# 验证服务
make health

# 检查指标
make metrics

# 查看日志
make docker-logs-agent
make docker-logs-server
```

### 3. 性能测试

```bash
# 压力测试Server
curl -X POST http://localhost:8080/api/v1/metrics \
  -H "Content-Type: application/json" \
  -d '{"test": "data"}'

# 监控资源使用
docker stats netmon-agent netmon-server
```

## 🚀 部署流程

### 1. 开发环境部署

```bash
# 启动开发环境
make docker-up-debug

# 包含监控栈
make docker-up-monitoring
```

### 2. 测试环境部署

```bash
# 构建测试镜像
make build-optimized

# 启动测试环境
make docker-up

# 验证部署
make health
```

### 3. 生产环境部署

#### 3.1 镜像准备
```bash
# 构建生产镜像
make build-optimized

# 推送到镜像仓库
make deploy-push
```

#### 3.2 Kubernetes部署
```bash
# 部署到K8s集群
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/server-deployment.yaml
kubectl apply -f k8s/agent-daemonset.yaml
```

#### 3.3 Docker Compose部署
```bash
# 生产环境配置
docker-compose -f docker-compose.prod.yml up -d
```

## 📊 监控和维护

### 1. 健康检查

```bash
# 检查服务状态
make health

# 查看容器状态
docker ps --filter "name=netmon"

# 检查资源使用
docker stats
```

### 2. 日志管理

```bash
# 查看实时日志
make docker-logs

# 查看特定服务日志
make docker-logs-agent
make docker-logs-server

# 日志轮转 (生产环境)
docker-compose logs --tail=1000 > logs/app.log
```

### 3. 指标监控

```bash
# 查看Prometheus指标
make metrics

# 访问Grafana Dashboard
# http://localhost:3000 (admin/admin123)

# 检查关键指标
curl -s http://localhost:8080/metrics | grep network_domain
```

## 🔧 故障排查

### 1. 常见问题

#### 构建失败
```bash
# 清理并重新构建
make clean-all
make build-optimized
```

#### 服务启动失败
```bash
# 检查日志
make docker-logs

# 检查配置
docker exec netmon-agent cat /app/configs/agent.yaml
```

#### 权限问题
```bash
# Agent需要特权模式
docker run --privileged --network host ...

# 或使用sudo运行本地Agent
sudo make dev-run-agent
```

### 2. 调试技巧

```bash
# 启用调试模式
make docker-up-debug

# 进入容器调试
docker exec -it netmon-agent sh
docker exec -it netmon-server sh

# 查看网络接口
docker exec netmon-agent ip link show
```

## 📋 检查清单

### 开发前检查
- [ ] 环境设置完成 (`make dev-setup`)
- [ ] 依赖安装完成
- [ ] 功能分支已创建

### 提交前检查
- [ ] 代码测试通过 (`make test`)
- [ ] 构建验证通过 (`make build-optimized`)
- [ ] 集成测试通过 (`make health`)
- [ ] 代码格式化完成

### 部署前检查
- [ ] 镜像构建成功
- [ ] 配置文件正确
- [ ] 健康检查通过
- [ ] 监控指标正常

### 发布前检查
- [ ] 版本标签创建
- [ ] 文档更新完成
- [ ] 变更日志记录
- [ ] 回滚方案准备

## 🎯 最佳实践

### 1. 代码质量
- 使用统一的代码格式 (`go fmt`)
- 编写单元测试 (覆盖率>80%)
- 添加必要的注释和文档
- 遵循Go编程规范

### 2. 构建优化
- 优先使用 `make build-optimized`
- 利用Docker层缓存
- 定期清理构建缓存
- 监控构建时间和镜像大小

### 3. 部署安全
- 使用非root用户运行
- 限制容器权限
- 定期更新基础镜像
- 配置健康检查

### 4. 监控运维
- 设置关键指标告警
- 定期检查日志
- 监控资源使用情况
- 制定故障响应流程

这个标准流程确保了项目的高质量交付和稳定运行。
