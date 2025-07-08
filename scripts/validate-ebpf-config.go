package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"go-net-monitoring/internal/config"
	"github.com/spf13/viper"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: go run validate-ebpf-config.go <config-file>")
		os.Exit(1)
	}

	configFile := os.Args[1]
	
	// 直接使用 viper 读取配置进行调试
	fmt.Printf("=== 原始配置内容 ===\n")
	debugConfig(configFile)
	
	// 加载配置
	cfg, err := config.LoadAgentConfig(configFile)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	fmt.Printf("\n=== eBPF 配置验证 ===\n")
	fmt.Printf("配置文件: %s\n\n", configFile)

	// 验证 eBPF 配置
	validateEBPFConfig(cfg)
}

func debugConfig(configFile string) {
	v := viper.New()
	v.SetConfigFile(configFile)
	v.SetConfigType("yaml")
	
	if err := v.ReadInConfig(); err != nil {
		log.Printf("读取配置文件失败: %v", err)
		return
	}
	
	// 检查 eBPF 配置是否存在
	if v.IsSet("ebpf") {
		fmt.Printf("✓ 找到 eBPF 配置段\n")
		fmt.Printf("  program_path: %s\n", v.GetString("ebpf.program_path"))
		fmt.Printf("  enable_fallback: %t\n", v.GetBool("ebpf.enable_fallback"))
		fmt.Printf("  fallback_paths: %v\n", v.GetStringSlice("ebpf.fallback_paths"))
	} else {
		fmt.Printf("✗ 未找到 eBPF 配置段\n")
	}
}

func validateEBPFConfig(cfg *config.AgentConfig) {
	fmt.Printf("eBPF 配置:\n")
	fmt.Printf("  主要路径: %s\n", cfg.EBPF.ProgramPath)
	fmt.Printf("  启用回退: %t\n", cfg.EBPF.EnableFallback)
	fmt.Printf("  备用路径: %v\n\n", cfg.EBPF.FallbackPaths)

	// 检查主要路径
	fmt.Printf("路径验证:\n")
	checkPath("主要路径", cfg.EBPF.ProgramPath)

	// 检查备用路径
	for i, path := range cfg.EBPF.FallbackPaths {
		checkPath(fmt.Sprintf("备用路径 %d", i+1), path)
	}

	// 模拟路径解析
	fmt.Printf("\n路径解析测试:\n")
	resolvedPath := resolvePath(cfg.EBPF.ProgramPath)
	if resolvedPath != "" {
		fmt.Printf("✓ 主要路径解析成功: %s\n", resolvedPath)
		return
	}

	// 尝试备用路径
	for i, path := range cfg.EBPF.FallbackPaths {
		resolvedPath = resolvePath(path)
		if resolvedPath != "" {
			fmt.Printf("✓ 备用路径 %d 解析成功: %s\n", i+1, resolvedPath)
			return
		}
	}

	if cfg.EBPF.EnableFallback {
		fmt.Printf("⚠ 所有路径都失败，将使用模拟模式\n")
	} else {
		fmt.Printf("✗ 所有路径都失败，且未启用回退模式\n")
	}
}

func checkPath(name, path string) {
	if path == "" {
		fmt.Printf("  %s: (未配置)\n", name)
		return
	}

	if filepath.IsAbs(path) {
		if _, err := os.Stat(path); err == nil {
			fmt.Printf("  ✓ %s: %s (存在)\n", name, path)
		} else {
			fmt.Printf("  ✗ %s: %s (不存在)\n", name, path)
		}
	} else {
		fmt.Printf("  ~ %s: %s (相对路径)\n", name, path)
	}
}

func resolvePath(programPath string) string {
	if programPath == "" {
		return ""
	}

	// 绝对路径直接检查
	if filepath.IsAbs(programPath) {
		if _, err := os.Stat(programPath); err == nil {
			return programPath
		}
		return ""
	}

	// 相对路径搜索
	searchPaths := []string{
		programPath, // 当前工作目录
	}

	// 添加二进制文件目录
	if execPath, err := os.Executable(); err == nil {
		binDir := filepath.Dir(execPath)
		searchPaths = append(searchPaths, filepath.Join(binDir, programPath))
		
		// 项目根目录
		parentDir := filepath.Dir(binDir)
		searchPaths = append(searchPaths, filepath.Join(parentDir, programPath))
	}

	// 尝试每个路径
	for _, searchPath := range searchPaths {
		if _, err := os.Stat(searchPath); err == nil {
			return searchPath
		}
	}

	return ""
}
