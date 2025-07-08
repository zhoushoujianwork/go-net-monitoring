package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-net-monitoring/pkg/ebpf/loader"

	"github.com/sirupsen/logrus"
)

var (
	iface      = flag.String("interface", "lo0", "Network interface to monitor")
	interval   = flag.Duration("interval", 5*time.Second, "Stats collection interval")
	debug      = flag.Bool("debug", false, "Enable debug logging")
	programPath = flag.String("program", "bin/bpf/xdp_monitor.o", "Path to eBPF program")
)

func main() {
	flag.Parse()

	// 设置日志
	logger := logrus.New()
	if *debug {
		logger.SetLevel(logrus.DebugLevel)
	}

	logger.WithFields(logrus.Fields{
		"interface": *iface,
		"interval":  *interval,
		"program":   *programPath,
	}).Info("Starting eBPF network monitor")

	// 创建XDP加载器
	xdpLoader := loader.NewXDPLoader(*iface, logger)

	// 检查eBPF程序文件是否存在
	if _, err := os.Stat(*programPath); os.IsNotExist(err) {
		logger.WithField("path", *programPath).Warn("eBPF program file not found, running in simulation mode")
		runSimulationMode(logger)
		return
	}

	// 加载eBPF程序
	if err := xdpLoader.Load(*programPath); err != nil {
		logger.WithError(err).Fatal("Failed to load eBPF program")
	}
	defer xdpLoader.Close()

	// 附加到网络接口
	if err := xdpLoader.Attach(); err != nil {
		logger.WithError(err).Fatal("Failed to attach XDP program")
	}

	// 设置信号处理
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动统计收集
	xdpLoader.StartStatsCollection(*interval, func(stats *loader.PacketStats) {
		logger.WithFields(logrus.Fields{
			"total_packets": stats.TotalPackets,
			"total_bytes":   stats.TotalBytes,
			"tcp_packets":   stats.TCPPackets,
			"udp_packets":   stats.UDPPackets,
			"other_packets": stats.OtherPackets,
		}).Info("Network statistics")
	})

	logger.Info("eBPF monitor started, press Ctrl+C to stop")

	// 等待信号
	select {
	case sig := <-sigChan:
		logger.WithField("signal", sig).Info("Received signal, shutting down")
	case <-ctx.Done():
		logger.Info("Context cancelled, shutting down")
	}

	logger.Info("eBPF monitor stopped")
}

// 模拟模式 - 用于测试框架
func runSimulationMode(logger *logrus.Logger) {
	logger.Info("Running in simulation mode - generating mock network statistics")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	var totalPackets, totalBytes uint64

	for {
		select {
		case <-ticker.C:
			// 模拟网络统计数据
			totalPackets += 100 + uint64(time.Now().Unix()%50)
			totalBytes += 64000 + uint64(time.Now().Unix()%32000)

			logger.WithFields(logrus.Fields{
				"total_packets": totalPackets,
				"total_bytes":   totalBytes,
				"tcp_packets":   totalPackets * 70 / 100,
				"udp_packets":   totalPackets * 20 / 100,
				"other_packets": totalPackets * 10 / 100,
				"mode":          "simulation",
			}).Info("Mock network statistics")

		case sig := <-sigChan:
			logger.WithField("signal", sig).Info("Received signal, shutting down simulation")
			return
		}
	}
}
