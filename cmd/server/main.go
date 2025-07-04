package main

import (
	"fmt"
	"os"

	"go-net-monitoring/internal/config"
	"go-net-monitoring/internal/server"

	"github.com/spf13/cobra"
)

var (
	configPath string
	debugMode  bool
	version    = "1.0.0"
	buildTime  = "unknown"
	gitCommit  = "unknown"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "server",
		Short: "网络流量监控Server",
		Long:  `网络流量监控Server，接收Agent上报的数据并通过Prometheus指标暴露。`,
		RunE:  runServer,
	}

	rootCmd.Flags().StringVarP(&configPath, "config", "c", "configs/server.yaml", "配置文件路径")
	rootCmd.Flags().BoolVarP(&debugMode, "debug", "d", false, "启用debug模式")

	// 添加--version标志支持
	var showVersion bool
	rootCmd.Flags().BoolVar(&showVersion, "version", false, "显示版本信息")

	// 在运行前检查版本标志
	rootCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if showVersion {
			fmt.Printf("网络监控Server\n")
			fmt.Printf("版本: %s\n", version)
			fmt.Printf("构建时间: %s\n", buildTime)
			fmt.Printf("Git提交: %s\n", gitCommit)
			os.Exit(0)
		}
		return nil
	}

	// 版本命令
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "显示版本信息",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("网络监控Server\n")
			fmt.Printf("版本: %s\n", version)
			fmt.Printf("构建时间: %s\n", buildTime)
			fmt.Printf("Git提交: %s\n", gitCommit)
		},
	}

	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}

func runServer(cmd *cobra.Command, args []string) error {
	// 加载配置
	cfg, err := config.SimpleLoadServerConfig(configPath)
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	// 如果命令行指定了debug模式，覆盖配置文件设置
	if debugMode {
		cfg.HTTP.Debug = true
		cfg.Log.Level = "debug"
		cfg.Log.Format = "text" // debug模式使用text格式更易读
		fmt.Println("Server Debug模式已启用")
	}

	// 创建Server
	server, err := server.NewServer(cfg)
	if err != nil {
		return fmt.Errorf("创建Server失败: %w", err)
	}

	// 运行Server
	return server.Run()
}
