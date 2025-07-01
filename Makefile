# Makefile for go-net-monitoring

# 变量定义
BINARY_DIR := bin
AGENT_BINARY := $(BINARY_DIR)/agent
SERVER_BINARY := $(BINARY_DIR)/server
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")

# Go构建参数
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)"
GO_BUILD := go build $(LDFLAGS)

# 默认目标
.PHONY: all
all: build

# 创建bin目录
$(BINARY_DIR):
	mkdir -p $(BINARY_DIR)

# 构建Agent
.PHONY: build-agent
build-agent: $(BINARY_DIR)
	$(GO_BUILD) -o $(AGENT_BINARY) ./cmd/agent

# 构建Server
.PHONY: build-server
build-server: $(BINARY_DIR)
	$(GO_BUILD) -o $(SERVER_BINARY) ./cmd/server

# 构建全部
.PHONY: build
build: build-agent build-server

# 运行测试
.PHONY: test
test:
	go test -v ./...

# 运行测试并生成覆盖率报告
.PHONY: test-coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# 代码格式化
.PHONY: fmt
fmt:
	go fmt ./...

# 代码检查
.PHONY: lint
lint:
	golangci-lint run

# 清理构建文件
.PHONY: clean
clean:
	rm -rf $(BINARY_DIR)
	rm -f coverage.out coverage.html

# 安装依赖
.PHONY: deps
deps:
	go mod download
	go mod tidy

# 运行Agent（开发模式）
.PHONY: run-agent
run-agent: build-agent
	sudo $(AGENT_BINARY) --config configs/agent.yaml

# 运行Server（开发模式）
.PHONY: run-server
run-server: build-server
	$(SERVER_BINARY) --config configs/server.yaml

# Docker构建
.PHONY: docker-build
docker-build:
	docker build -t go-net-monitoring:$(VERSION) .

# 生成文档
.PHONY: docs
docs:
	@echo "生成API文档..."
	@mkdir -p docs/api
	@echo "文档生成完成"

# 部署脚本
.PHONY: deploy
deploy: build
	@echo "部署到生产环境..."
	@./scripts/deploy.sh

# 开发环境设置
.PHONY: dev-setup
dev-setup:
	@echo "设置开发环境..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "开发环境设置完成"

# 检查代码质量
.PHONY: check
check: fmt lint test

# 发布版本
.PHONY: release
release: check build
	@echo "发布版本 $(VERSION)"
	@./scripts/release.sh $(VERSION)

# 帮助信息
.PHONY: help
help:
	@echo "可用的make目标:"
	@echo "  all          - 构建全部组件"
	@echo "  build        - 构建全部组件"
	@echo "  build-agent  - 构建Agent"
	@echo "  build-server - 构建Server"
	@echo "  test         - 运行测试"
	@echo "  test-coverage- 运行测试并生成覆盖率报告"
	@echo "  fmt          - 格式化代码"
	@echo "  lint         - 代码检查"
	@echo "  clean        - 清理构建文件"
	@echo "  deps         - 安装依赖"
	@echo "  run-agent    - 运行Agent（需要sudo权限）"
	@echo "  run-server   - 运行Server"
	@echo "  docker-build - Docker构建"
	@echo "  docs         - 生成文档"
	@echo "  deploy       - 部署到生产环境"
	@echo "  dev-setup    - 设置开发环境"
	@echo "  check        - 检查代码质量"
	@echo "  release      - 发布版本"
	@echo "  help         - 显示帮助信息"
