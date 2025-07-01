package main

import (
	"fmt"
	"os"

	"go-net-monitoring/internal/agent"
	"go-net-monitoring/internal/config"

	"github.com/spf13/cobra"
)

var (
	configPath string
	version    = "1.0.0"
	buildTime  = "unknown"
	gitCommit  = "unknown"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "agent",
		Short: "网络流量监控Agent",
		Long:  `一个用于监控主机网络流量的Agent，支持域名和IP地址访问监控，并通过HTTP接口上报到配套的Server。`,
		RunE:  runAgent,
	}

	rootCmd.Flags().StringVarP(&configPath, "config", "c", "configs/agent.yaml", "配置文件路径")

	// 版本命令
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "显示版本信息",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("网络监控Agent\n")
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

func runAgent(cmd *cobra.Command, args []string) error {
	// 加载配置
	cfg, err := config.SimpleLoadAgentConfig(configPath)
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	// 创建Agent
	agent, err := agent.NewAgent(cfg)
	if err != nil {
		return fmt.Errorf("创建Agent失败: %w", err)
	}

	// 运行Agent
	return agent.Run()
}
