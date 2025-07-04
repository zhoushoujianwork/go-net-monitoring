# Docker构建网络问题解决方案

## 🚨 问题描述

在使用`scripts/build-docker.sh`构建多平台Docker镜像时，可能遇到网络连接超时问题：

```
ERROR: failed to solve: DeadlineExceeded: DeadlineExceeded: alpine:3.19: 
failed to resolve source metadata for docker.io/library/alpine:3.19: 
failed to do request: Head "https://registry-1.docker.io/v2/library/alpine/manifests/3.19": 
dial tcp 31.13.95.33:443: i/o timeout
```

## 🔍 问题原因

1. **网络连接问题**: 无法连接到Docker Hub (registry-1.docker.io)
2. **DNS解析问题**: 域名解析失败或缓慢
3. **防火墙限制**: 网络防火墙阻止了HTTPS连接
4. **地理位置限制**: 某些地区访问Docker Hub较慢

## 🔧 解决方案

### 方案1: 使用本地构建 (推荐)

```bash
# 使用简化的本地构建，避免网络问题
make docker-build-local
```

**特点:**
- ✅ 避免多平台构建的网络复杂性
- ✅ 只构建当前平台 (linux/amd64)
- ✅ 构建速度快
- ✅ 适合本地开发和测试

### 方案2: 使用网络优化构建

```bash
# 使用网络优化的构建脚本
make docker-build-fixed

# 或者手动修复网络问题
make docker-build-fixed --fix-network
```

**特点:**
- 🔧 自动配置Docker镜像源
- 🔧 预拉取基础镜像
- 🔧 网络超时处理
- 🔧 支持多平台构建

### 方案3: 手动配置Docker镜像源

#### 3.1 配置daemon.json

```bash
# 编辑Docker配置文件
sudo vim /etc/docker/daemon.json
```

添加以下内容：
```json
{
  "registry-mirrors": [
    "https://docker.mirrors.ustc.edu.cn",
    "https://hub-mirror.c.163.com",
    "https://mirror.baidubce.com"
  ],
  "insecure-registries": [],
  "experimental": false,
  "features": {
    "buildkit": true
  }
}
```

#### 3.2 重启Docker服务

```bash
# 重启Docker服务
sudo systemctl restart docker

# 验证配置
docker info | grep -A 10 "Registry Mirrors"
```

### 方案4: 预拉取基础镜像

```bash
# 手动拉取基础镜像
docker pull golang:1.21-alpine
docker pull alpine:3.19

# 然后再构建
make docker-build
```

### 方案5: 使用代理

```bash
# 设置HTTP代理
export HTTP_PROXY=http://your-proxy:port
export HTTPS_PROXY=http://your-proxy:port

# 构建镜像
make docker-build
```

## 📋 可用构建命令

| 命令 | 说明 | 适用场景 |
|------|------|----------|
| `make docker-build` | 标准构建 (优化版) | 网络正常时 |
| `make docker-build-local` | 本地构建 | 网络问题时 |
| `make docker-build-fixed` | 网络优化构建 | 网络不稳定时 |
| `make docker-build-push` | 构建并推送 | 发布版本时 |

## 🔍 网络诊断

### 检查Docker Hub连接

```bash
# 测试Docker Hub连接
curl -I --connect-timeout 10 https://registry-1.docker.io/v2/

# 测试DNS解析
nslookup registry-1.docker.io

# 测试镜像拉取
docker pull hello-world
```

### 检查Docker配置

```bash
# 查看Docker信息
docker info

# 查看镜像源配置
docker info | grep -A 10 "Registry Mirrors"

# 查看buildx配置
docker buildx ls
```

## 🚀 推荐工作流

### 开发环境

```bash
# 1. 本地开发构建
make docker-build-local

# 2. 测试镜像
docker run --rm -e COMPONENT=server go-net-monitoring:latest --version

# 3. 启动服务
make docker-up-debug
```

### 生产环境

```bash
# 1. 网络优化构建
make docker-build-fixed

# 2. 推送到仓库 (如果需要)
make docker-build-push

# 3. 部署服务
make docker-up
```

## 🔧 故障排查

### 常见错误及解决方案

#### 1. 连接超时
```
dial tcp 31.13.95.33:443: i/o timeout
```
**解决**: 使用镜像源或本地构建

#### 2. DNS解析失败
```
failed to resolve source metadata
```
**解决**: 检查DNS配置，使用公共DNS

#### 3. 证书错误
```
x509: certificate signed by unknown authority
```
**解决**: 更新CA证书或使用insecure-registries

#### 4. 权限错误
```
permission denied while trying to connect to the Docker daemon
```
**解决**: 添加用户到docker组或使用sudo

### 调试技巧

```bash
# 启用详细日志
docker build --progress=plain .

# 检查网络连接
docker run --rm alpine:latest ping -c 3 registry-1.docker.io

# 测试镜像源
docker run --rm alpine:latest wget -O- https://docker.mirrors.ustc.edu.cn/v2/
```

## 📝 总结

**推荐策略:**

1. **开发阶段**: 使用`make docker-build-local`进行快速本地构建
2. **测试阶段**: 使用`make docker-build-fixed`进行网络优化构建
3. **生产阶段**: 配置好网络环境后使用`make docker-build-push`

**网络问题的根本解决方案:**
- 配置可靠的Docker镜像源
- 使用稳定的网络连接
- 预拉取常用基础镜像
- 设置合理的超时时间

通过这些方案，可以有效解决Docker构建过程中的网络问题。
