.PHONY: help build build-optimized clean test docker-up docker-down docker-logs docker-clean

# 默认目标
help: ## 显示帮助信息
	@echo "可用的命令:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# 构建相关
build: ## 构建二进制文件
	@echo "构建二进制文件..."
	@mkdir -p bin
	CGO_ENABLED=1 go build -o bin/agent ./cmd/agent
	CGO_ENABLED=0 go build -o bin/server ./cmd/server
	@echo "构建完成: bin/agent, bin/server"

build-optimized: ## 优化构建Docker镜像
	@./scripts/build-optimized.sh

build-clean: ## 清理缓存后构建
	@./scripts/build-optimized.sh --clean-cache

build-test: ## 构建并测试
	@./scripts/build-optimized.sh --test

# 清理相关
clean: ## 清理构建文件
	@echo "清理构建文件..."
	@rm -rf bin/
	@docker system prune -f >/dev/null 2>&1 || true
	@echo "清理完成"

clean-all: ## 深度清理
	@echo "深度清理..."
	@rm -rf bin/ data/ logs/
	@docker system prune -af >/dev/null 2>&1 || true
	@docker volume prune -f >/dev/null 2>&1 || true
	@echo "深度清理完成"

# 测试相关
test: ## 运行测试
	@echo "运行测试..."
	@go test -v ./...

test-coverage: ## 运行测试并生成覆盖率报告
	@echo "生成测试覆盖率报告..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告: coverage.html"

# Docker相关
docker-up: ## 启动服务 (生产模式)
	@echo "启动服务..."
	@docker-compose up -d

docker-up-debug: ## 启动服务 (调试模式)
	@echo "启动服务 (调试模式)..."
	@DEBUG_MODE=true LOG_LEVEL=debug docker-compose up -d

docker-up-monitoring: ## 启动完整监控栈
	@echo "启动完整监控栈..."
	@docker-compose --profile monitoring up -d

docker-down: ## 停止服务
	@echo "停止服务..."
	@docker-compose down

docker-restart: ## 重启服务
	@echo "重启服务..."
	@docker-compose restart

docker-logs: ## 查看日志
	@docker-compose logs -f

docker-logs-agent: ## 查看Agent日志
	@docker-compose logs -f agent

docker-logs-server: ## 查看Server日志
	@docker-compose logs -f server

docker-clean: ## 清理Docker资源
	@echo "清理Docker资源..."
	@docker-compose down -v
	@docker system prune -f
	@echo "Docker清理完成"

# 开发相关
dev-setup: ## 设置开发环境
	@echo "设置开发环境..."
	@go mod download
	@mkdir -p bin data logs
	@echo "开发环境设置完成"

dev-run-server: ## 运行Server (开发模式)
	@echo "运行Server (开发模式)..."
	@go run ./cmd/server --config configs/server.yaml --debug

dev-run-agent: ## 运行Agent (开发模式，需要root权限)
	@echo "运行Agent (开发模式)..."
	@sudo go run ./cmd/agent --config configs/agent.yaml --debug

# 部署相关
deploy-build: build-optimized ## 构建部署镜像
	@echo "部署镜像构建完成"

deploy-push: ## 推送镜像到仓库
	@echo "推送镜像..."
	@docker tag go-net-monitoring:latest zhoushoujian/go-net-monitoring:latest
	@docker push zhoushoujian/go-net-monitoring:latest
	@echo "镜像推送完成"

# 监控相关
metrics: ## 查看指标
	@echo "当前指标:"
	@curl -s http://localhost:8080/metrics | grep -E "(network_domain|network_connections)" | head -10

health: ## 检查服务健康状态
	@echo "检查服务健康状态..."
	@curl -s http://localhost:8080/health || echo "Server不可用"
	@docker ps --filter "name=netmon" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

# 文档相关
docs: ## 生成文档
	@echo "生成文档..."
	@go doc -all ./... > docs/api.md
	@echo "文档生成完成: docs/api.md"

# 版本相关
version: ## 显示版本信息
	@echo "版本信息:"
	@git describe --tags --always --dirty 2>/dev/null || echo "dev"
	@echo "Git提交: $(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
	@echo "构建时间: $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")"
