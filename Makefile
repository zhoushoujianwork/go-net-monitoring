# Go Network Monitoring Makefile
# 只支持容器化部署

.PHONY: help docker-build docker-up docker-up-debug docker-up-monitoring docker-down docker-logs docker-logs-agent docker-logs-server docker-restart docker-clean health metrics test clean

# 默认目标
.DEFAULT_GOAL := help

# 颜色定义
BLUE := \033[36m
GREEN := \033[32m
YELLOW := \033[33m
RED := \033[31m
NC := \033[0m

# 帮助信息
help: ## 显示帮助信息
	@echo "$(BLUE)Go Network Monitoring - 容器化部署$(NC)"
	@echo ""
	@echo "$(GREEN)可用命令:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(YELLOW)%-20s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "$(GREEN)快速开始:$(NC)"
	@echo "  1. $(YELLOW)make docker-build$(NC)     - 构建Docker镜像"
	@echo "  2. $(YELLOW)make docker-up$(NC)        - 启动服务"
	@echo "  3. $(YELLOW)make health$(NC)           - 检查服务状态"
	@echo ""
	@echo "$(GREEN)构建选项:$(NC)"
	@echo "  $(YELLOW)make docker-build-local$(NC)  - 本地构建 (推荐，避免网络问题)"
	@echo "  $(YELLOW)make docker-build-fixed$(NC)  - 网络优化构建"
	@echo "  $(YELLOW)make docker-build-push$(NC)   - 构建并推送到Docker Hub"
	@echo ""
	@echo "$(GREEN)调试模式:$(NC)"
	@echo "  $(YELLOW)make docker-up-debug$(NC)    - 启动调试模式"
	@echo "  $(YELLOW)make docker-logs$(NC)        - 查看日志"

# Docker构建
docker-build: ## 构建Docker镜像
	@echo "$(BLUE)[INFO]$(NC) 构建Docker镜像..."
	@./scripts/build-optimized.sh

docker-build-local: ## 本地构建Docker镜像 (避免网络问题)
	@echo "$(BLUE)[INFO]$(NC) 本地构建Docker镜像..."
	@chmod +x scripts/build-local.sh
	@./scripts/build-local.sh

docker-build-fixed: ## 网络优化构建Docker镜像
	@echo "$(BLUE)[INFO]$(NC) 网络优化构建Docker镜像..."
	@chmod +x scripts/build-docker-fixed.sh
	@./scripts/build-docker-fixed.sh

docker-build-push: ## 构建并推送Docker镜像
	@echo "$(BLUE)[INFO]$(NC) 构建并推送Docker镜像..."
	@echo "$(YELLOW)[STEP 1/2]$(NC) 本地构建镜像..."
	@chmod +x scripts/build-local.sh
	@./scripts/build-local.sh
	@echo "$(YELLOW)[STEP 2/2]$(NC) 推送镜像..."
	@chmod +x scripts/push-image.sh
	@./scripts/push-image.sh

docker-push: ## 推送已构建的Docker镜像
	@echo "$(BLUE)[INFO]$(NC) 推送Docker镜像..."
	@chmod +x scripts/push-image.sh
	@./scripts/push-image.sh

# Docker服务管理
docker-up: ## 启动服务 (生产模式)
	@echo "$(GREEN)启动服务...$(NC)"
	@DEBUG_MODE=false docker-compose up -d

docker-up-debug: ## 启动服务 (调试模式)
	@echo "$(GREEN)启动服务 (调试模式)...$(NC)"
	@DEBUG_MODE=true docker-compose up -d

docker-up-monitoring: ## 启动完整监控栈
	@echo "$(GREEN)启动完整监控栈...$(NC)"
	@docker-compose --profile monitoring up -d

docker-down: ## 停止服务
	@echo "$(YELLOW)停止服务...$(NC)"
	@docker-compose down

docker-restart: ## 重启服务
	@echo "$(YELLOW)重启服务...$(NC)"
	@docker-compose restart

# 日志查看
docker-logs: ## 查看所有服务日志
	@docker-compose logs -f

docker-logs-agent: ## 查看Agent日志
	@docker-compose logs -f agent

docker-logs-server: ## 查看Server日志
	@docker-compose logs -f server

# 服务监控
health: ## 检查服务健康状态
	@echo "$(BLUE)[INFO]$(NC) 检查服务健康状态..."
	@echo "$(GREEN)Server状态:$(NC)"
	@curl -s http://localhost:8080/health 2>/dev/null && echo " ✅ Server运行正常" || echo " ❌ Server无响应"
	@echo "$(GREEN)容器状态:$(NC)"
	@docker-compose ps

metrics: ## 查看监控指标
	@echo "$(BLUE)[INFO]$(NC) 获取监控指标..."
	@curl -s http://localhost:8080/metrics | head -20
	@echo "..."
	@echo "$(GREEN)完整指标:$(NC) curl http://localhost:8080/metrics"

# 测试
test: ## 运行测试
	@echo "$(BLUE)[INFO]$(NC) 运行测试..."
	@docker-compose exec server go test ./... -v

# 清理
docker-clean: ## 清理Docker资源
	@echo "$(YELLOW)清理Docker资源...$(NC)"
	@docker-compose down -v
	@docker system prune -f
	@docker volume prune -f

clean: ## 清理所有资源
	@echo "$(YELLOW)清理所有资源...$(NC)"
	@docker-compose down -v --remove-orphans
	@docker system prune -af
	@docker volume prune -f
	@echo "$(GREEN)清理完成$(NC)"

# 开发辅助
dev-logs: ## 开发模式日志 (彩色输出)
	@docker-compose logs -f --no-log-prefix

dev-shell-server: ## 进入Server容器
	@docker-compose exec server /bin/sh

dev-shell-agent: ## 进入Agent容器
	@docker-compose exec agent /bin/sh
