package main

import (
	"flag"
	"fmt"
	"os"

	"go-net-monitoring/internal/config"
	"go-net-monitoring/internal/server"

	"github.com/sirupsen/logrus"
)

var (
	configFile = flag.String("config", "configs/server.yaml", "配置文件路径")
	debug      = flag.Bool("debug", false, "启用调试模式")
	version    = flag.Bool("version", false, "显示版本信息")
)

const (
	AppName    = "go-net-monitoring-server"
	AppVersion = "2.0.0"
)

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("%s version %s\n", AppName, AppVersion)
		os.Exit(0)
	}

	// 加载配置
	cfg, err := config.LoadServerConfig(*configFile)
	if err != nil {
		logrus.WithError(err).Fatal("加载配置失败")
	}

	// 调试模式覆盖配置
	if *debug {
		cfg.Log.Level = "debug"
	}

	// 创建服务器
	srv, err := server.NewServer(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("创建服务器失败")
	}

	// 启动服务器
	logrus.WithFields(logrus.Fields{
		"app":     AppName,
		"version": AppVersion,
		"config":  *configFile,
	}).Info("启动网络监控服务器")

	if err := srv.Run(); err != nil {
		logrus.WithError(err).Fatal("服务器运行失败")
	}
}
