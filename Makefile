# Go Network Monitoring Makefile
# 构建Linux版本的eBPF Agent和Server

.PHONY: help build build-agent build-server build-linux build-agent-linux build-server-linux build-ebpf clean

# 默认目标
.DEFAULT_GOAL := help

# 颜色定义
GREEN := \033[32m
BLUE := \033[36m
YELLOW := \033[33m
RED := \033[31m
NC := \033[0m

# 项目信息
PROJECT_NAME := go-net-monitoring
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go构建参数
GO_VERSION := 1.21
GOOS_LINUX := linux
GOARCH := amd64
CGO_ENABLED := 1

# 构建标志
LDFLAGS := -X main.Version=$(VERSION) \
           -X main.BuildTime=$(BUILD_TIME) \
           -X main.GitCommit=$(GIT_COMMIT) \
           -w -s

# 输出目录
BUILD_DIR := bin
LINUX_BUILD_DIR := $(BUILD_DIR)/linux
BPF_DIR := $(BUILD_DIR)/bpf

# 确保release目录存在
RELEASE_DIR := release/linux-amd64/bpf
$(shell mkdir -p $(RELEASE_DIR))

.PHONY: all
all: build

# 检查操作系统
check-os:
	@if [ "$(shell uname -s)" != "Linux" ]; then \
		echo "$(RED)❌ 错误: 此程序只能在 Linux 环境下构建和运行$(NC)"; \
		echo "$(YELLOW)原因: eBPF 功能需要 Linux 内核支持$(NC)"; \
		echo "$(BLUE)建议: 请使用以下方式之一:$(NC)"; \
		echo "  1. 在 Linux 环境下构建"; \
		echo "  2. 使用 Docker 容器构建: make docker-build"; \
		echo "  3. 使用 Docker Compose: make docker-up"; \
		exit 1; \
	fi

# 修改 build 目标
.PHONY: build
build: check-os build-ebpf build-go ## 构建所有组件 (仅支持 Linux)

.PHONY: build-ebpf
build-ebpf:
	$(MAKE) -C bpf
	cp bpf/xdp_monitor.o $(RELEASE_DIR)/
	cp bpf/xdp_monitor_linux.o $(RELEASE_DIR)/

.PHONY: build-go
build-go:
	go build -o release/linux-amd64/agent-ebpf ./cmd/agent-ebpf

help: ## 显示帮助信息
	@echo "$(BLUE)Go Network Monitoring 构建工具$(NC)"
	@echo ""
	@echo "$(YELLOW)可用命令:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "$(YELLOW)构建信息:$(NC)"
	@echo "  版本: $(GREEN)$(VERSION)$(NC)"
	@echo "  提交: $(GREEN)$(GIT_COMMIT)$(NC)"
	@echo "  时间: $(GREEN)$(BUILD_TIME)$(NC)"

build: build-ebpf build-agent build-server ## 构建所有组件 (本地平台)

build-agent: ## 构建eBPF Agent (本地平台)
	@echo "$(BLUE)构建eBPF Agent (本地平台)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/agent-ebpf ./cmd/agent-ebpf/
	@echo "$(GREEN)✅ eBPF Agent构建完成: $(BUILD_DIR)/agent-ebpf$(NC)"

build-server: ## 构建Server (本地平台)
	@echo "$(BLUE)构建Server (本地平台)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/server ./cmd/server/
	@echo "$(GREEN)✅ Server构建完成: $(BUILD_DIR)/server$(NC)"

build-linux: build-ebpf build-agent-linux build-server-linux ## 构建所有组件 (Linux平台)

build-agent-linux: ## 构建eBPF Agent (Linux平台)
	@echo "$(BLUE)构建eBPF Agent (Linux x86_64)...$(NC)"
	@mkdir -p $(LINUX_BUILD_DIR)
ifeq ($(shell uname -s),Linux)
	@echo "$(GREEN)检测到Linux环境，构建完整eBPF支持版本$(NC)"
	GOOS=$(GOOS_LINUX) GOARCH=$(GOARCH) CGO_ENABLED=1 \
	go build -ldflags "$(LDFLAGS)" -o $(LINUX_BUILD_DIR)/agent-ebpf ./cmd/agent-ebpf/
else
	@echo "$(YELLOW)检测到非Linux环境，构建交叉编译版本$(NC)"
	GOOS=$(GOOS_LINUX) GOARCH=$(GOARCH) CGO_ENABLED=0 \
	go build -ldflags "$(LDFLAGS)" -tags netgo -o $(LINUX_BUILD_DIR)/agent-ebpf ./cmd/agent-ebpf/
	@echo "$(YELLOW)⚠️  此版本为交叉编译，eBPF功能将使用模拟模式$(NC)"
endif
	@echo "$(GREEN)✅ Linux eBPF Agent构建完成: $(LINUX_BUILD_DIR)/agent-ebpf$(NC)"
	@ls -lh $(LINUX_BUILD_DIR)/agent-ebpf

build-server-linux: ## 构建Server (Linux平台)
	@echo "$(BLUE)构建Server (Linux x86_64)...$(NC)"
	@mkdir -p $(LINUX_BUILD_DIR)
	GOOS=$(GOOS_LINUX) GOARCH=$(GOARCH) CGO_ENABLED=0 \
	go build -ldflags "$(LDFLAGS)" -o $(LINUX_BUILD_DIR)/server ./cmd/server/
	@echo "$(GREEN)✅ Linux Server构建完成: $(LINUX_BUILD_DIR)/server$(NC)"
	@ls -lh $(LINUX_BUILD_DIR)/server

build-ebpf: ## 构建eBPF字节码文件
	@echo "$(BLUE)构建eBPF字节码文件...$(NC)"
	@mkdir -p $(BPF_DIR)
ifeq ($(shell uname -s),Linux)
	@echo "$(GREEN)检测到Linux环境，使用本地clang构建$(NC)"
	@if command -v clang >/dev/null 2>&1; then \
		clang -O2 -target bpf -c bpf/programs/xdp_monitor.c -o $(BPF_DIR)/xdp_monitor.o; \
		clang -O2 -target bpf -c bpf/programs/xdp_monitor_linux.c -o $(BPF_DIR)/xdp_monitor_linux.o; \
		echo "$(GREEN)✅ eBPF字节码构建完成$(NC)"; \
	else \
		echo "$(RED)❌ 未找到clang，请安装: apt-get install clang llvm$(NC)"; \
		exit 1; \
	fi
else
	@echo "$(YELLOW)检测到非Linux环境，创建占位符文件$(NC)"
	@echo "// eBPF placeholder for non-Linux builds" > $(BPF_DIR)/xdp_monitor.o
	@echo "// eBPF placeholder for non-Linux builds" > $(BPF_DIR)/xdp_monitor_linux.o
	@echo "$(YELLOW)⚠️  已创建占位符文件，真实eBPF需要在Linux环境构建$(NC)"
endif
	@ls -la $(BPF_DIR)/

clean: ## 清理构建文件
	@echo "$(YELLOW)清理构建文件...$(NC)"
	rm -rf $(BUILD_DIR)
	@echo "$(GREEN)✅ 清理完成$(NC)"

# 显示构建信息
info: ## 显示构建信息
	@echo "$(BLUE)构建信息:$(NC)"
	@echo "  项目名称: $(GREEN)$(PROJECT_NAME)$(NC)"
	@echo "  版本: $(GREEN)$(VERSION)$(NC)"
	@echo "  Git提交: $(GREEN)$(GIT_COMMIT)$(NC)"
	@echo "  构建时间: $(GREEN)$(BUILD_TIME)$(NC)"
	@echo "  Go版本: $(GREEN)$(GO_VERSION)$(NC)"
	@echo "  目标平台: $(GREEN)$(GOOS_LINUX)/$(GOARCH)$(NC)"
	@echo "  输出目录: $(GREEN)$(LINUX_BUILD_DIR)$(NC)"

# 开发相关的目标
.PHONY: docker-dev docker-dev-build docker-dev-down docker-dev-logs

docker-dev-build: ## 构建开发环境镜像
	@echo "$(BLUE)构建开发环境镜像...$(NC)"
	docker-compose -f docker-compose.dev.yml build

docker-dev: ## 启动开发环境
	@echo "$(BLUE)启动开发环境...$(NC)"
	docker-compose -f docker-compose.dev.yml up -d
	@echo "$(GREEN)✅ 开发环境已启动$(NC)"
	@echo "$(YELLOW)提示:$(NC)"
	@echo "  - 查看日志: make docker-dev-logs"
	@echo "  - 停止环境: make docker-dev-down"

docker-dev-down: ## 停止开发环境
	@echo "$(BLUE)停止开发环境...$(NC)"
	docker-compose -f docker-compose.dev.yml down
	@echo "$(GREEN)✅ 开发环境已停止$(NC)"

docker-dev-logs: ## 查看开发环境日志
	docker-compose -f docker-compose.dev.yml logs -f

docker-dev-restart: ## 重启开发环境
	@echo "$(BLUE)重启开发环境...$(NC)"
	docker-compose -f docker-compose.dev.yml restart
	@echo "$(GREEN)✅ 开发环境已重启$(NC)"
	@echo "$(YELLOW)提示: 使用 make docker-dev-logs 查看日志$(NC)"
