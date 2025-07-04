.PHONY: help build build-optimized clean test docker-up docker-down docker-logs docker-clean

# 默认目标
help: ## 显示帮助信息
	@echo "可用的命令:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# 构建相关
build: ## 构建二进制文件 (当前平台)
	@echo "构建二进制文件..."
	@mkdir -p bin
	CGO_ENABLED=1 go build -o bin/agent ./cmd/agent
	CGO_ENABLED=0 go build -o bin/server ./cmd/server
	@echo "构建完成: bin/agent, bin/server"

build-all: ## 构建所有平台的二进制文件
	@echo "构建所有平台的二进制文件..."
	@./scripts/build-release.sh

build-release: ## 构建发布包 (推荐用于分发)
	@./scripts/build-release.sh

build-cross: ## 跨平台构建 (使用脚本)
	@./scripts/build-cross-platform.sh $(ARGS)

build-cross-current: ## 构建当前平台 (使用脚本)
	@./scripts/build-cross-platform.sh --current

build-cross-linux: ## 构建Linux版本 (使用脚本)
	@./scripts/build-cross-platform.sh --linux

build-cross-darwin: ## 构建macOS版本 (使用脚本)
	@./scripts/build-cross-platform.sh --darwin

build-cross-windows: ## 构建Windows版本 (使用脚本)
	@./scripts/build-cross-platform.sh --windows

# Linux 构建
build-linux: build-linux-amd64 build-linux-arm64 ## 构建Linux版本 (amd64 + arm64)

build-linux-amd64: ## 构建Linux AMD64版本
	@echo "构建Linux AMD64版本..."
	@mkdir -p bin dist
	@if pkg-config --exists libpcap 2>/dev/null; then \
		echo "libpcap可用，构建完整版本"; \
		CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o bin/agent-linux-amd64 ./cmd/agent; \
	else \
		echo "警告: libpcap不可用，跳过Agent构建"; \
		echo "请安装libpcap: sudo apt-get install libpcap-dev (Ubuntu) 或 sudo yum install libpcap-devel (CentOS)"; \
	fi
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/server-linux-amd64 ./cmd/server
	@if [ -f bin/agent-linux-amd64 ] && [ -f bin/server-linux-amd64 ]; then \
		cd bin && tar -czf ../dist/go-net-monitoring-linux-amd64.tar.gz agent-linux-amd64 server-linux-amd64; \
	elif [ -f bin/server-linux-amd64 ]; then \
		cd bin && tar -czf ../dist/go-net-monitoring-linux-amd64.tar.gz server-linux-amd64; \
	fi
	@echo "Linux AMD64构建完成"

build-linux-arm64: ## 构建Linux ARM64版本
	@echo "构建Linux ARM64版本..."
	@mkdir -p bin dist
	@if pkg-config --exists libpcap 2>/dev/null; then \
		echo "libpcap可用，构建完整版本"; \
		CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build -o bin/agent-linux-arm64 ./cmd/agent; \
	else \
		echo "警告: libpcap不可用，跳过Agent构建"; \
	fi
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o bin/server-linux-arm64 ./cmd/server
	@if [ -f bin/agent-linux-arm64 ] && [ -f bin/server-linux-arm64 ]; then \
		cd bin && tar -czf ../dist/go-net-monitoring-linux-arm64.tar.gz agent-linux-arm64 server-linux-arm64; \
	elif [ -f bin/server-linux-arm64 ]; then \
		cd bin && tar -czf ../dist/go-net-monitoring-linux-arm64.tar.gz server-linux-arm64; \
	fi
	@echo "Linux ARM64构建完成"

# macOS 构建
build-darwin: build-darwin-amd64 build-darwin-arm64 ## 构建macOS版本 (Intel + Apple Silicon)

build-darwin-amd64: ## 构建macOS Intel版本
	@echo "构建macOS Intel版本..."
	@mkdir -p bin dist
	@echo "注意: macOS交叉编译需要在macOS系统上进行，或配置交叉编译工具链"
	@if [ "$$(uname)" = "Darwin" ]; then \
		CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -o bin/agent-darwin-amd64 ./cmd/agent; \
		CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o bin/server-darwin-amd64 ./cmd/server; \
		cd bin && tar -czf ../dist/go-net-monitoring-darwin-amd64.tar.gz agent-darwin-amd64 server-darwin-amd64; \
	else \
		echo "警告: 非macOS系统，跳过Agent构建 (需要CGO支持)"; \
		CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o bin/server-darwin-amd64 ./cmd/server; \
		cd bin && tar -czf ../dist/go-net-monitoring-darwin-amd64.tar.gz server-darwin-amd64; \
	fi
	@echo "macOS Intel构建完成"

build-darwin-arm64: ## 构建macOS Apple Silicon版本
	@echo "构建macOS Apple Silicon版本..."
	@mkdir -p bin dist
	@echo "注意: macOS交叉编译需要在macOS系统上进行，或配置交叉编译工具链"
	@if [ "$$(uname)" = "Darwin" ]; then \
		CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o bin/agent-darwin-arm64 ./cmd/agent; \
		CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o bin/server-darwin-arm64 ./cmd/server; \
		cd bin && tar -czf ../dist/go-net-monitoring-darwin-arm64.tar.gz agent-darwin-arm64 server-darwin-arm64; \
	else \
		echo "警告: 非macOS系统，跳过Agent构建 (需要CGO支持)"; \
		CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o bin/server-darwin-arm64 ./cmd/server; \
		cd bin && tar -czf ../dist/go-net-monitoring-darwin-arm64.tar.gz server-darwin-arm64; \
	fi
	@echo "macOS Apple Silicon构建完成"

# Windows 构建
build-windows: build-windows-amd64 ## 构建Windows版本

build-windows-amd64: ## 构建Windows AMD64版本
	@echo "构建Windows AMD64版本..."
	@mkdir -p bin dist
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -o bin/agent-windows-amd64.exe ./cmd/agent
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o bin/server-windows-amd64.exe ./cmd/server
	@cd bin && zip -q ../dist/go-net-monitoring-windows-amd64.zip agent-windows-amd64.exe server-windows-amd64.exe
	@echo "Windows AMD64构建完成: bin/agent-windows-amd64.exe, bin/server-windows-amd64.exe"

# 当前平台检测和构建
build-current: ## 构建当前平台版本
	@echo "检测当前平台并构建..."
	@GOOS=$$(go env GOOS); GOARCH=$$(go env GOARCH); \
	echo "当前平台: $$GOOS/$$GOARCH"; \
	mkdir -p bin; \
	if [ "$$GOOS" = "darwin" ]; then \
		if [ "$$GOARCH" = "arm64" ]; then \
			$(MAKE) build-darwin-arm64; \
			ln -sf agent-darwin-arm64 bin/agent; \
			ln -sf server-darwin-arm64 bin/server; \
		else \
			$(MAKE) build-darwin-amd64; \
			ln -sf agent-darwin-amd64 bin/agent; \
			ln -sf server-darwin-amd64 bin/server; \
		fi \
	elif [ "$$GOOS" = "linux" ]; then \
		if [ "$$GOARCH" = "arm64" ]; then \
			$(MAKE) build-linux-arm64; \
			ln -sf agent-linux-arm64 bin/agent; \
			ln -sf server-linux-arm64 bin/server; \
		else \
			$(MAKE) build-linux-amd64; \
			ln -sf agent-linux-amd64 bin/agent; \
			ln -sf server-linux-amd64 bin/server; \
		fi \
	elif [ "$$GOOS" = "windows" ]; then \
		$(MAKE) build-windows-amd64; \
		ln -sf agent-windows-amd64.exe bin/agent.exe; \
		ln -sf server-windows-amd64.exe bin/server.exe; \
	fi
	@echo "当前平台构建完成，可执行文件: bin/agent, bin/server"

build-optimized: ## 优化构建Docker镜像
	@./scripts/build-optimized.sh

build-clean: ## 清理缓存后构建
	@./scripts/build-optimized.sh --clean-cache

build-test: ## 构建并测试
	@./scripts/build-optimized.sh --test

# CI/CD相关
ci-lint: ## CI: 代码质量检查
	@echo "运行代码质量检查..."
	@go fmt ./...
	@echo "代码格式化完成"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "运行golangci-lint..."; \
		golangci-lint run; \
	else \
		echo "golangci-lint未安装，跳过linter检查"; \
	fi
	@echo "代码质量检查完成"

ci-test: ## CI: 运行测试
	@echo "运行单元测试..."
	@if pkg-config --exists libpcap; then \
		echo "libpcap可用，运行完整测试"; \
		go test -v -race -coverprofile=coverage.out ./...; \
	else \
		echo "libpcap不可用，跳过需要pcap的测试"; \
		go test -v -race -coverprofile=coverage.out -tags=nopcap ./pkg/metrics ./internal/server ./internal/common; \
	fi
	@echo "测试完成"

ci-build: ## CI: 构建验证
	@echo "CI构建验证..."
	@mkdir -p bin
	@if pkg-config --exists libpcap; then \
		echo "libpcap可用，构建完整版本"; \
		CGO_ENABLED=1 GOOS=linux go build -o bin/agent ./cmd/agent; \
	else \
		echo "libpcap不可用，跳过agent构建"; \
	fi
	CGO_ENABLED=0 GOOS=linux go build -o bin/server ./cmd/server
	@echo "CI构建完成"

ci-docker: ## CI: Docker构建测试
	@echo "Docker构建测试..."
	@docker build --tag go-net-monitoring:test .
	@echo "Docker构建完成"

ci-integration: ## CI: 集成测试
	@echo "运行集成测试..."
	@docker-compose -f docker-compose.test.yml up -d
	@sleep 30
	@curl -f http://localhost:8081/health || (docker-compose -f docker-compose.test.yml logs && exit 1)
	@curl -f http://localhost:8081/metrics || (docker-compose -f docker-compose.test.yml logs && exit 1)
	@docker-compose -f docker-compose.test.yml down
	@echo "集成测试完成"

ci-all: ci-lint ci-test ci-build ci-docker ci-integration ## CI: 运行完整CI流程

# 清理相关
clean: ## 清理构建文件
	@echo "清理构建文件..."
	@rm -rf bin/ dist/
	@docker system prune -f >/dev/null 2>&1 || true
	@echo "清理完成"

clean-all: ## 深度清理
	@echo "深度清理..."
	@rm -rf bin/ dist/ data/ logs/ coverage.out coverage.html
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

test-integration: ## 运行集成测试
	@echo "运行集成测试..."
	@make ci-integration

# Docker相关
docker-up: ## 启动服务 (生产模式)
	@echo "启动服务..."
	@DEBUG_MODE=false docker-compose up -d

docker-up-debug: ## 启动服务 (调试模式)
	@echo "启动服务 (调试模式)..."
	@DEBUG_MODE=true docker-compose up -d

docker-up-monitoring: ## 启动完整监控栈
	@echo "启动完整监控栈..."
	@docker-compose --profile monitoring up -d

docker-up-test: ## 启动测试环境
	@echo "启动测试环境..."
	@docker-compose -f docker-compose.test.yml up -d

docker-debug-on: ## 切换到debug模式
	@echo "切换到debug模式..."
	@DEBUG_MODE=true docker-compose up -d
	@echo "Debug模式已启用，查看日志: make docker-logs"

docker-debug-off: ## 切换到生产模式
	@echo "切换到生产模式..."
	@DEBUG_MODE=false docker-compose up -d
	@echo "生产模式已启用"

docker-down: ## 停止服务
	@echo "停止服务..."
	@docker-compose down

docker-down-test: ## 停止测试环境
	@echo "停止测试环境..."
	@docker-compose -f docker-compose.test.yml down

docker-restart: ## 重启服务
	@echo "重启服务..."
	@docker-compose restart

docker-logs: ## 查看日志
	@docker-compose logs -f

docker-logs-agent: ## 查看Agent日志
	@docker-compose logs -f agent

docker-logs-server: ## 查看Server日志
	@docker-compose logs -f server

docker-logs-test: ## 查看测试环境日志
	@docker-compose -f docker-compose.test.yml logs

docker-clean: ## 清理Docker资源
	@echo "清理Docker资源..."
	@docker-compose down -v
	@docker-compose -f docker-compose.test.yml down -v >/dev/null 2>&1 || true
	@docker system prune -f
	@echo "Docker清理完成"

# 开发相关
dev-setup: ## 设置开发环境
	@echo "设置开发环境..."
	@go mod download
	@mkdir -p bin data logs dist
	@echo "检查平台特定依赖..."
	@if [ "$$(uname)" = "Darwin" ]; then \
		echo "检测到macOS系统"; \
		if ! command -v brew >/dev/null 2>&1; then \
			echo "警告: 未检测到Homebrew，请安装: /bin/bash -c \"\$$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""; \
		fi; \
		if ! brew list libpcap >/dev/null 2>&1; then \
			echo "安装libpcap依赖..."; \
			brew install libpcap || echo "请手动安装: brew install libpcap"; \
		fi; \
	elif [ "$$(uname)" = "Linux" ]; then \
		echo "检测到Linux系统"; \
		if command -v apt-get >/dev/null 2>&1; then \
			echo "Ubuntu/Debian系统，请确保已安装: sudo apt-get install libpcap-dev"; \
		elif command -v yum >/dev/null 2>&1; then \
			echo "CentOS/RHEL系统，请确保已安装: sudo yum install libpcap-devel"; \
		fi; \
	fi
	@echo "开发环境设置完成"

dev-run-server: ## 运行Server (开发模式)
	@echo "运行Server (开发模式)..."
	@if [ ! -f bin/server ]; then \
		echo "二进制文件不存在，正在构建..."; \
		$(MAKE) build-current; \
	fi
	@./bin/server --config configs/server.yaml --debug

dev-run-agent: ## 运行Agent (开发模式，需要root权限)
	@echo "运行Agent (开发模式)..."
	@if [ ! -f bin/agent ]; then \
		echo "二进制文件不存在，正在构建..."; \
		$(MAKE) build-current; \
	fi
	@if [ "$$(uname)" = "Darwin" ]; then \
		echo "macOS系统，使用sudo运行Agent..."; \
		sudo ./bin/agent --config configs/agent.yaml --debug; \
	else \
		echo "Linux系统，使用sudo运行Agent..."; \
		sudo ./bin/agent --config configs/agent.yaml --debug; \
	fi

# macOS特定命令
macos-setup: ## macOS环境设置
	@echo "设置macOS开发环境..."
	@if ! command -v brew >/dev/null 2>&1; then \
		echo "安装Homebrew..."; \
		/bin/bash -c "$$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"; \
	fi
	@echo "安装依赖..."
	@brew install libpcap go
	@$(MAKE) dev-setup
	@echo "macOS环境设置完成"

macos-build: ## macOS构建 (自动检测架构)
	@echo "构建macOS版本..."
	@ARCH=$$(uname -m); \
	if [ "$$ARCH" = "arm64" ]; then \
		echo "检测到Apple Silicon (M1/M2)"; \
		$(MAKE) build-darwin-arm64; \
		ln -sf agent-darwin-arm64 bin/agent; \
		ln -sf server-darwin-arm64 bin/server; \
	else \
		echo "检测到Intel处理器"; \
		$(MAKE) build-darwin-amd64; \
		ln -sf agent-darwin-amd64 bin/agent; \
		ln -sf server-darwin-amd64 bin/server; \
	fi
	@echo "macOS构建完成"

macos-run-agent: ## macOS运行Agent
	@echo "在macOS上运行Agent..."
	@if [ ! -f bin/agent ]; then \
		echo "Agent不存在，正在构建..."; \
		$(MAKE) macos-build; \
	fi
	@echo "注意: Agent需要root权限进行网络监控"
	@echo "如果遇到权限问题，请运行: sudo $(MAKE) macos-run-agent-sudo"
	@sudo ./bin/agent --config configs/agent.yaml --debug

macos-run-agent-sudo: ## macOS以sudo权限运行Agent
	@./bin/agent --config configs/agent.yaml --debug

macos-run-server: ## macOS运行Server
	@echo "在macOS上运行Server..."
	@if [ ! -f bin/server ]; then \
		echo "Server不存在，正在构建..."; \
		$(MAKE) macos-build; \
	fi
	@./bin/server --config configs/server.yaml --debug

# 部署相关
deploy-build: build-optimized ## 构建部署镜像
	@echo "部署镜像构建完成"

deploy-push: ## 推送镜像到仓库
	@echo "推送镜像..."
	@docker tag go-net-monitoring:latest zhoushoujian/go-net-monitoring:latest
	@docker push zhoushoujian/go-net-monitoring:latest
	@echo "镜像推送完成"

deploy-k8s: ## 部署到Kubernetes
	@echo "部署到Kubernetes..."
	@kubectl apply -f k8s/namespace.yaml
	@kubectl apply -f k8s/server-deployment.yaml
	@kubectl apply -f k8s/agent-daemonset.yaml
	@echo "Kubernetes部署完成"

# 监控相关
metrics: ## 查看指标
	@echo "当前指标:"
	@curl -s http://localhost:8080/metrics | grep -E "(network_domain|network_connections)" | head -10

health: ## 检查服务健康状态
	@echo "检查服务健康状态..."
	@curl -s http://localhost:8080/health || echo "Server不可用"
	@docker ps --filter "name=netmon" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

# 质量检查
fmt: ## 格式化代码
	@echo "格式化代码..."
	@go fmt ./...

vet: ## 代码静态检查
	@echo "运行go vet..."
	@go vet ./...

lint: ## 运行linter
	@echo "运行golangci-lint..."
	@golangci-lint run

quality: fmt vet lint test ## 运行所有质量检查

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

# 发布相关
release-prepare: ## 准备发布
	@echo "准备发布..."
	@make quality
	@make ci-all
	@echo "发布准备完成"

release-tag: ## 创建发布标签
	@echo "创建发布标签..."
	@read -p "输入版本号 (例如: v1.0.0): " version; \
	git tag -a $$version -m "Release $$version"; \
	git push origin $$version
	@echo "发布标签创建完成"
