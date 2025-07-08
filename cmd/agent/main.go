package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go-net-monitoring/internal/agent"
	"go-net-monitoring/internal/config"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	configFile string
	debug      bool
)

var rootCmd = &cobra.Command{
	Use:   "agent",
	Short: "网络监控代理 (传统版本 - 已弃用)",
	Long: `网络监控代理 - 传统版本

注意: 此版本已被eBPF版本替代，建议使用 agent-ebpf 获得更好的性能。

此版本仅保留用于兼容性测试。`,
	RunE: runAgent,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "configs/agent.yaml", "配置文件路径")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "启用调试模式")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logrus.WithError(err).Fatal("程序执行失败")
	}
}

func runAgent(cmd *cobra.Command, args []string) error {
	// 显示弃用警告
	logrus.Warn("=== 传统Agent已弃用 ===")
	logrus.Warn("建议使用eBPF版本: ./bin/agent-ebpf")
	logrus.Warn("eBPF版本提供更好的性能和更低的资源消耗")
	logrus.Warn("========================")

	// 加载配置
	cfg, err := config.LoadAgentConfig(configFile)
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	// 调试模式覆盖配置
	if debug {
		cfg.Log.Level = "debug"
	}

	// 创建Agent
	agentInstance, err := agent.NewAgent(cfg)
	if err != nil {
		return fmt.Errorf("创建Agent失败: %w", err)
	}

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动Agent
	logrus.Info("启动传统网络监控代理 (已弃用)")
	if err := agentInstance.Start(); err != nil {
		return fmt.Errorf("启动Agent失败: %w", err)
	}

	// 等待信号
	sig := <-sigChan
	logrus.WithField("signal", sig).Info("收到停止信号")

	// 停止Agent
	if err := agentInstance.Stop(); err != nil {
		logrus.WithError(err).Error("停止Agent失败")
	}

	logrus.Info("传统网络监控代理已退出")
	return nil
}
