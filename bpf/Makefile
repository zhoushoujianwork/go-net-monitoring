# eBPF 程序编译 Makefile (跨平台版本)

# 编译器设置
CLANG ?= clang
LLVM_STRIP ?= llvm-strip

# 检测操作系统
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

# 编译标志
CFLAGS := -O2 -g -Wall -Werror
CFLAGS += -target bpf

# 平台特定设置
ifeq ($(UNAME_S),Darwin)
    # macOS 设置 - 使用兼容头文件
    CFLAGS += -D__TARGET_ARCH_$(shell echo $(UNAME_M) | sed 's/x86_64/x86/' | sed 's/arm64/arm64/')
    CFLAGS += -I./headers
    BPF_HEADERS := headers
else
    # Linux 设置 - 使用系统头文件
    CFLAGS += -D__TARGET_ARCH_$(shell echo $(UNAME_M) | sed 's/x86_64/x86/' | sed 's/aarch64/arm64/')
    CFLAGS += -I/usr/include/$(shell uname -m)-linux-gnu
    CFLAGS += -I/usr/src/linux-headers-$(shell uname -r)/include
    CFLAGS += -I/usr/src/linux-headers-$(shell uname -r)/arch/$(shell uname -m)/include
    BPF_HEADERS := 
endif

# 源文件和目标文件
PROGRAMS_DIR := programs
HEADERS_DIR := headers
OUTPUT_DIR := ../bin/bpf

# 查找所有 .c 文件
SOURCES := $(wildcard $(PROGRAMS_DIR)/*.c)
OBJECTS := $(SOURCES:$(PROGRAMS_DIR)/%.c=$(OUTPUT_DIR)/%.o)

# 默认目标
.PHONY: all clean install headers docker-build

all: $(BPF_HEADERS) $(OUTPUT_DIR) $(OBJECTS)

# 创建输出目录
$(OUTPUT_DIR):
	mkdir -p $(OUTPUT_DIR)

# 创建必要的头文件（macOS兼容）
headers:
	mkdir -p $(HEADERS_DIR)

# 编译 eBPF 程序
$(OUTPUT_DIR)/%.o: $(PROGRAMS_DIR)/%.c
	@echo "Compiling eBPF program: $< ($(UNAME_S))"
	$(CLANG) $(CFLAGS) -c $< -o $@
	@if command -v $(LLVM_STRIP) >/dev/null 2>&1; then \
		$(LLVM_STRIP) -g $@; \
		echo "Stripped debug symbols from $@"; \
	else \
		echo "Warning: llvm-strip not found, skipping strip"; \
	fi

# Docker 编译（推荐用于 macOS）
docker-build:
	@echo "Building eBPF programs in Docker container..."
	docker-compose -f ../docker-compose.ebpf-dev.yml run --rm ebpf-build

# 清理
clean:
	rm -rf $(OUTPUT_DIR)
	@echo "Cleaned build artifacts"

# 安装（复制到系统目录）
install: all
	@echo "eBPF programs compiled successfully:"
	@ls -la $(OUTPUT_DIR)/ 2>/dev/null || echo "No compiled programs found"

# 验证编译结果
verify: all
	@echo "Verifying eBPF programs..."
	@for obj in $(OBJECTS); do \
		if [ -f "$$obj" ]; then \
			echo "✅ Checking $$obj..."; \
			file $$obj; \
			echo ""; \
		fi; \
	done

# 显示环境信息
env:
	@echo "Build environment:"
	@echo "  OS: $(UNAME_S)"
	@echo "  Arch: $(UNAME_M)"
	@echo "  Clang: $(shell which $(CLANG) 2>/dev/null || echo 'not found')"
	@echo "  LLVM Strip: $(shell which $(LLVM_STRIP) 2>/dev/null || echo 'not found')"
	@echo "  CFLAGS: $(CFLAGS)"
	@echo "  BPF Headers: $(BPF_HEADERS)"

# 测试编译环境
test-env:
	@echo "Testing compilation environment..."
	@$(CLANG) --version | head -2
	@echo ""
	@echo "Testing BPF target support..."
	@echo 'int main() { return 0; }' | $(CLANG) -target bpf -c -x c - -o /tmp/test.o 2>/dev/null && \
		echo "✅ BPF target supported" || \
		echo "❌ BPF target not supported - use Docker"
	@rm -f /tmp/test.o

# 帮助信息
help:
	@echo "Available targets:"
	@echo "  all          - Compile all eBPF programs"
	@echo "  docker-build - Build using Docker (recommended for macOS)"
	@echo "  clean        - Remove compiled objects"
	@echo "  install      - Install compiled programs"
	@echo "  verify       - Verify compiled programs"
	@echo "  env          - Show build environment"
	@echo "  test-env     - Test compilation environment"
	@echo "  help         - Show this help message"
	@echo ""
	@echo "Quick start:"
	@echo "  make test-env    # Check if local compilation works"
	@echo "  make all         # Compile locally (Linux)"
	@echo "  make docker-build # Compile in Docker (macOS/any)"
