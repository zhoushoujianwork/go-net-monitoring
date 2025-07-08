package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go-net-monitoring/internal/agent"
	"go-net-monitoring/internal/config"

	"github.com/sirupsen/logrus"
)

var (
	configFile = flag.String("config", "configs/agent.yaml", "配置文件路径")
	debug      = flag.Bool("debug", false, "启用调试模式")
	version    = flag.Bool("version", false, "显示版本信息")
)

const (
	AppName    = "go-net-monitoring-ebpf-agent"
	AppVersion = "2.0.0-ebpf"
)

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("%s version %s\n", AppName, AppVersion)
		fmt.Println("Built with eBPF support using cilium/ebpf")
		os.Exit(0)
	}

	// 加载配置
	cfg, err := config.LoadAgentConfig(*configFile)
	if err != nil {
		logrus.WithError(err).Fatal("加载配置失败")
	}

	// 调试模式覆盖配置
	if *debug {
		cfg.Log.Level = "debug"
	}

	// 创建eBPF Agent
	ebpfAgent, err := agent.NewEBPFAgent(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("创建eBPF Agent失败")
	}

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动Agent
	logrus.WithFields(logrus.Fields{
		"app":     AppName,
		"version": AppVersion,
		"config":  *configFile,
	}).Info("启动eBPF网络监控代理")

	if err := ebpfAgent.Start(); err != nil {
		logrus.WithError(err).Fatal("启动eBPF Agent失败")
	}

	// 等待信号
	sig := <-sigChan
	logrus.WithField("signal", sig).Info("收到停止信号")

	// 停止Agent
	if err := ebpfAgent.Stop(); err != nil {
		logrus.WithError(err).Error("停止eBPF Agent失败")
	}

	logrus.Info("eBPF网络监控代理已退出")
}
