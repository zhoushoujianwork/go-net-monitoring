# Debug 模式功能实现总结

## 实现的功能

### 1. 命令行参数支持

**Server:**
```bash
./bin/server -d                    # 启用debug模式
./bin/server --debug               # 启用debug模式（长格式）
./bin/server -d -c config.yaml     # debug模式 + 自定义配置
```

**Agent:**
```bash
sudo ./bin/agent -d                # 启用debug模式
sudo ./bin/agent --debug           # 启用debug模式（长格式）
sudo ./bin/agent -d -c config.yaml # debug模式 + 自定义配置
```

### 2. 配置文件支持

**Server 配置文件中的 debug 选项:**
```yaml
http:
  debug: true    # 启用debug模式
```

**优先级：** 命令行参数 > 配置文件设置

### 3. Gin 框架 Debug 模式集成

- **Debug 模式下：** `gin.SetMode(gin.DebugMode)`
  - 打印所有路由注册信息
  - 显示每个HTTP请求的详细日志
  - 包含响应时间、状态码、客户端IP等信息

- **Release 模式下：** `gin.SetMode(gin.ReleaseMode)`
  - 不打印路由信息
  - 更好的性能
  - 适合生产环境

### 4. 自定义路由打印功能

实现了 `printRoutes()` 函数，在 debug 模式下打印所有注册的路由：

```
=== 注册的路由 ===
路由已注册 method=POST path=/api/v1/metrics
路由已注册 method=POST path=/api/v1/heartbeat
路由已注册 method=GET path=/api/v1/agents
...
=== 路由注册完成 ===
```

### 5. 详细的调试日志

**Server Debug 模式特性：**
- 显示接收到的指标数据详情
- 记录 Agent 连接和心跳信息
- 输出配置加载和初始化信息

**Agent Debug 模式特性：**
- 显示使用的网络监控工具
- 输出详细的启动信息
- 记录配置文件路径

### 6. HTTP 请求日志

Debug 模式下，每个 HTTP 请求都会被记录：

```
[GIN] 2025/07/02 - 14:50:26 | 200 |      95.945µs |             ::1 | GET      "/"
[GIN] 2025/07/02 - 14:50:26 | 200 |     115.163µs |             ::1 | GET      "/health"
```

包含信息：
- 时间戳
- HTTP 状态码
- 响应时间
- 客户端 IP
- HTTP 方法和路径

## 技术实现细节

### 1. 配置结构修改

在 `HTTPConfig` 中添加了 `Debug` 字段：

```go
type HTTPConfig struct {
    Host         string        `yaml:"host"`
    Port         int           `yaml:"port"`
    // ... 其他字段
    Debug        bool          `yaml:"debug"`        // Debug模式
}
```

### 2. 命令行参数处理

使用 Cobra 框架添加 debug 标志：

```go
rootCmd.Flags().BoolVarP(&debugMode, "debug", "d", false, "启用debug模式")
```

### 3. Gin 模式设置

根据配置动态设置 Gin 模式：

```go
if cfg.HTTP.Debug {
    gin.SetMode(gin.DebugMode)
    logger.Info("Gin运行在Debug模式")
} else {
    gin.SetMode(gin.ReleaseMode)
    logger.Info("Gin运行在Release模式")
}
```

### 4. 中间件配置

Debug 模式下添加 Gin 的 Logger 中间件：

```go
if cfg.HTTP.Debug {
    ginEngine.Use(gin.Logger())
}
ginEngine.Use(gin.Recovery())
```

### 5. 路由打印实现

```go
func (s *Server) printRoutes() {
    routes := s.ginEngine.Routes()
    s.logger.Info("=== 注册的路由 ===")
    for _, route := range routes {
        s.logger.WithFields(logrus.Fields{
            "method": route.Method,
            "path":   route.Path,
        }).Info("路由已注册")
    }
    s.logger.Info("=== 路由注册完成 ===")
}
```

## 配置文件示例

### server-debug.yaml
```yaml
http:
  host: "0.0.0.0"
  port: 8080
  debug: true        # 启用debug模式

log:
  level: "debug"     # debug级别日志
  format: "text"     # 使用text格式，更易读
  output: "stdout"
```

### server.yaml (生产环境)
```yaml
http:
  host: "0.0.0.0"
  port: 8080
  debug: false       # 关闭debug模式

log:
  level: "info"      # info级别日志
  format: "json"     # 使用json格式
  output: "stdout"
```

## 使用场景

1. **开发调试**：查看路由注册和请求处理详情
2. **问题排查**：获取详细的错误信息和执行流程
3. **性能分析**：查看请求响应时间
4. **API 测试**：验证路由和端点是否正确注册

## 注意事项

1. **生产环境不建议使用**：会产生大量日志，影响性能
2. **命令行参数优先级高**：`-d` 参数会覆盖配置文件设置
3. **日志格式建议**：debug 模式建议使用 text 格式，更易读
4. **端口冲突**：测试时注意端口占用问题

## 测试验证

所有功能都已通过测试验证：

1. ✅ 命令行 `-d` 参数正常工作
2. ✅ 配置文件 `debug: true` 正常工作
3. ✅ Gin Debug 模式正确切换
4. ✅ 路由信息正确打印
5. ✅ HTTP 请求日志正常记录
6. ✅ Release 模式性能优化正常
7. ✅ Agent debug 模式正常工作

## 文档

- [Debug 模式使用指南](docs/debug-mode.md)
- [README.md](README.md) - 包含快速开始部分
