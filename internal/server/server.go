package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go-net-monitoring/internal/common"
	"go-net-monitoring/internal/config"
	"go-net-monitoring/pkg/metrics"
	"go-net-monitoring/pkg/network"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// Server 网络监控服务器
type Server struct {
	config     *config.ServerAppConfig
	logger     *logrus.Logger
	httpServer *http.Server
	ginEngine  *gin.Engine
	metrics    *metrics.Metrics
	storage    Storage
	agents     map[string]*common.AgentInfo
	agentsMu   sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	startTime  time.Time
}

// Storage 存储接口
type Storage interface {
	Store(key string, value interface{}) error
	Get(key string) (interface{}, error)
	Delete(key string) error
	List(prefix string) ([]interface{}, error)
	Close() error
}

// NewServer 创建新的服务器
func NewServer(cfg *config.ServerAppConfig) (*Server, error) {
	// 初始化日志
	logger := logrus.New()
	if err := setupLogger(logger, &cfg.Log); err != nil {
		return nil, fmt.Errorf("初始化日志失败: %w", err)
	}

	// 设置gin模式
	if cfg.HTTP.Debug {
		gin.SetMode(gin.DebugMode)
		logger.Info("Gin运行在Debug模式")
	} else {
		gin.SetMode(gin.ReleaseMode)
		logger.Info("Gin运行在Release模式")
	}

	// 创建gin引擎
	ginEngine := gin.New()

	// 添加中间件
	if cfg.HTTP.Debug {
		// Debug模式下使用gin的默认日志中间件，会打印路由信息
		ginEngine.Use(gin.Logger())
	}
	ginEngine.Use(gin.Recovery())

	// 初始化Prometheus指标
	metricsInstance := metrics.NewMetrics()

	// 初始化存储
	storage, err := NewStorage(&cfg.Storage)
	if err != nil {
		return nil, fmt.Errorf("初始化存储失败: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	s := &Server{
		config:    cfg,
		logger:    logger,
		ginEngine: ginEngine,
		metrics:   metricsInstance,
		storage:   storage,
		agents:    make(map[string]*common.AgentInfo),
		ctx:       ctx,
		cancel:    cancel,
		startTime: time.Now(),
	}

	// 设置路由
	s.setupRoutes()

	// 创建HTTP服务器
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.config.HTTP.Host, s.config.HTTP.Port),
		Handler:      s.ginEngine,
		ReadTimeout:  s.config.HTTP.ReadTimeout,
		WriteTimeout: s.config.HTTP.WriteTimeout,
	}

	logger.WithFields(logrus.Fields{
		"host":  cfg.HTTP.Host,
		"port":  cfg.HTTP.Port,
		"debug": cfg.HTTP.Debug,
	}).Info("Server初始化完成")

	return s, nil
}

// setupRoutes 设置路由
func (s *Server) setupRoutes() {
	// API路由组
	api := s.ginEngine.Group("/api/v1")
	{
		api.POST("/metrics", s.handleMetrics)
		api.POST("/heartbeat", s.handleHeartbeat)
		api.GET("/agents", s.handleGetAgents)
		api.GET("/agents/:id", s.handleGetAgent)
		api.DELETE("/agents/:id", s.handleDeleteAgent)
		api.GET("/stats", s.handleGetStats)
		api.GET("/status", s.handleStatus)
	}

	// Prometheus指标端点
	s.ginEngine.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// 健康检查端点
	s.ginEngine.GET("/health", s.handleHealth)
	s.ginEngine.GET("/ready", s.handleReady)

	// 根路径
	s.ginEngine.GET("/", s.handleRoot)

	// 如果是debug模式，打印所有注册的路由
	if s.config.HTTP.Debug {
		s.printRoutes()
	}
}

// printRoutes 打印所有注册的路由
func (s *Server) printRoutes() {
	routes := s.ginEngine.Routes()
	s.logger.Info("=== 注册的路由 ===")
	for _, route := range routes {
		s.logger.WithFields(logrus.Fields{
			"method": route.Method,
			"path":   route.Path,
		}).Info("路由已注册")
	}
	s.logger.Info("=== 路由注册完成 ===")
}

// Run 运行服务器
func (s *Server) Run() error {
	s.logger.Info("启动网络监控Server...")

	// 启动后台任务
	s.wg.Add(1)
	go s.backgroundTasks()

	// 启动HTTP服务器
	go func() {
		s.logger.WithFields(logrus.Fields{
			"addr":  s.httpServer.Addr,
			"debug": s.config.HTTP.Debug,
		}).Info("HTTP服务器启动")

		var err error
		if s.config.HTTP.EnableTLS {
			err = s.httpServer.ListenAndServeTLS(s.config.HTTP.TLSCertPath, s.config.HTTP.TLSKeyPath)
		} else {
			err = s.httpServer.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			s.logger.WithError(err).Error("HTTP服务器启动失败")
		}
	}()

	// 等待停止信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		s.logger.WithField("signal", sig).Info("收到停止信号")
	case <-s.ctx.Done():
		s.logger.Info("上下文取消")
	}

	return s.shutdown()
}

// shutdown 优雅关闭服务器
func (s *Server) shutdown() error {
	s.logger.Info("开始关闭Server...")

	// 取消上下文
	s.cancel()

	// 关闭HTTP服务器
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.WithError(err).Error("HTTP服务器关闭失败")
	}

	// 等待后台任务完成
	s.wg.Wait()

	// 关闭存储
	if err := s.storage.Close(); err != nil {
		s.logger.WithError(err).Error("存储关闭失败")
	}

	s.logger.Info("Server已关闭")
	return nil
}

// backgroundTasks 后台任务
func (s *Server) backgroundTasks() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.Metrics.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.updateMetrics()
		case <-s.ctx.Done():
			return
		}
	}
}

// updateMetrics 更新指标
func (s *Server) updateMetrics() {
	s.agentsMu.RLock()
	agentCount := len(s.agents)
	s.agentsMu.RUnlock()

	// 更新Agent统计 - 使用现有的方法
	uptime := time.Since(s.startTime).Seconds()
	s.metrics.UpdateAgentStats("server", uptime, float64(time.Now().Unix()))

	// 清理过期的Agent
	s.cleanupExpiredAgents()

	if s.config.HTTP.Debug {
		s.logger.WithField("agent_count", agentCount).Debug("更新指标")
	}
}

// cleanupExpiredAgents 清理过期的Agent
func (s *Server) cleanupExpiredAgents() {
	s.agentsMu.Lock()
	defer s.agentsMu.Unlock()

	now := time.Now()
	for id, agent := range s.agents {
		if now.Sub(agent.LastSeen) > 5*time.Minute {
			delete(s.agents, id)
			s.logger.WithField("agent_id", id).Info("清理过期Agent")
		}
	}
}

// handleRoot 处理根路径请求
func (s *Server) handleRoot(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service": "网络流量监控Server",
		"version": "1.0.0",
		"status":  "running",
		"debug":   s.config.HTTP.Debug,
		"uptime":  time.Since(s.startTime).String(),
	})
}

// handleHealth 处理健康检查
func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"time":   time.Now().Unix(),
	})
}

// handleReady 处理就绪检查
func (s *Server) handleReady(c *gin.Context) {
	// 检查存储是否可用
	if err := s.storage.Store("health_check", time.Now()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"error":  "storage not available",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
		"time":   time.Now().Unix(),
	})
}

// handleMetrics 处理指标上报
func (s *Server) handleMetrics(c *gin.Context) {
	var request common.ReportRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		s.logger.WithError(err).Error("解析指标数据失败")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	// 更新Agent信息
	s.updateAgentInfo(request.AgentID, request.Hostname)

	// 处理指标数据
	s.processMetricsData(&request.Metrics)

	// 存储数据
	key := fmt.Sprintf("metrics:%s:%d", request.AgentID, request.Timestamp.Unix())
	if err := s.storage.Store(key, request.Metrics); err != nil {
		s.logger.WithError(err).Error("存储指标数据失败")
	}

	if s.config.HTTP.Debug {
		s.logger.WithFields(logrus.Fields{
			"agent_id": request.AgentID,
			"hostname": request.Hostname,
		}).Debug("收到指标数据")
	}

	// 返回响应
	response := common.ReportResponse{
		Success:   true,
		Message:   "Metrics received successfully",
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// handleHeartbeat 处理心跳
func (s *Server) handleHeartbeat(c *gin.Context) {
	var agentInfo common.AgentInfo
	if err := c.ShouldBindJSON(&agentInfo); err != nil {
		s.logger.WithError(err).Error("解析心跳请求失败")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	// 更新Agent信息
	s.agentsMu.Lock()
	agentInfo.LastSeen = time.Now()
	s.agents[agentInfo.ID] = &agentInfo
	s.agentsMu.Unlock()

	if s.config.HTTP.Debug {
		s.logger.WithFields(logrus.Fields{
			"agent_id": agentInfo.ID,
			"hostname": agentInfo.Hostname,
		}).Debug("收到Agent心跳")
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"timestamp": time.Now(),
	})
}

// handleGetAgents 获取所有Agent信息
func (s *Server) handleGetAgents(c *gin.Context) {
	s.agentsMu.RLock()
	agents := make([]*common.AgentInfo, 0, len(s.agents))
	for _, agent := range s.agents {
		agents = append(agents, agent)
	}
	s.agentsMu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"agents": agents,
		"count":  len(agents),
	})
}

// handleGetAgent 获取特定Agent信息
func (s *Server) handleGetAgent(c *gin.Context) {
	agentID := c.Param("id")

	s.agentsMu.RLock()
	agent, exists := s.agents[agentID]
	s.agentsMu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "agent not found",
		})
		return
	}

	c.JSON(http.StatusOK, agent)
}

// handleDeleteAgent 删除Agent
func (s *Server) handleDeleteAgent(c *gin.Context) {
	agentID := c.Param("id")

	s.agentsMu.Lock()
	_, exists := s.agents[agentID]
	if exists {
		delete(s.agents, agentID)
	}
	s.agentsMu.Unlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "agent not found",
		})
		return
	}

	s.logger.WithField("agent_id", agentID).Info("Agent已删除")
	c.JSON(http.StatusOK, gin.H{
		"status": "deleted",
	})
}

// handleGetStats 获取统计信息
func (s *Server) handleGetStats(c *gin.Context) {
	s.agentsMu.RLock()
	agentCount := len(s.agents)
	s.agentsMu.RUnlock()

	stats := gin.H{
		"agents": agentCount,
		"uptime": time.Since(s.startTime).String(),
		"debug":  s.config.HTTP.Debug,
	}

	c.JSON(http.StatusOK, stats)
}

// handleStatus 处理状态查询
func (s *Server) handleStatus(c *gin.Context) {
	s.agentsMu.RLock()
	agentCount := len(s.agents)
	s.agentsMu.RUnlock()

	status := gin.H{
		"status":      "running",
		"timestamp":   time.Now(),
		"agent_count": agentCount,
		"version":     "1.0.0",
		"debug":       s.config.HTTP.Debug,
		"uptime":      time.Since(s.startTime).String(),
	}

	c.JSON(http.StatusOK, status)
}

// updateAgentInfo 更新Agent信息
func (s *Server) updateAgentInfo(agentID, hostname string) {
	s.agentsMu.Lock()
	defer s.agentsMu.Unlock()

	if agent, exists := s.agents[agentID]; exists {
		agent.LastSeen = time.Now()
		agent.Status = "online"
	} else {
		s.agents[agentID] = &common.AgentInfo{
			ID:        agentID,
			Hostname:  hostname,
			Version:   "1.0.0",
			StartTime: time.Now(),
			LastSeen:  time.Now(),
			Status:    "online",
		}
		s.logger.WithField("agent_id", agentID).Info("新Agent注册")
	}
}

// processMetricsData 处理指标数据
func (s *Server) processMetricsData(metrics *common.NetworkMetrics) {
	// 更新Prometheus指标
	s.metrics.UpdateNetworkMetrics(*metrics)

	// 更新网卡信息指标
	s.updateInterfaceMetrics(metrics)
}

// updateInterfaceMetrics 更新网卡信息指标
func (s *Server) updateInterfaceMetrics(metrics *common.NetworkMetrics) {
	// 清除旧的网卡信息指标
	s.metrics.ClearInterfaceInfo()

	// 如果有接口信息，创建网络接口管理器来获取详细信息
	if metrics.Interface != "" && metrics.Interface != "unknown" {
		// 创建临时的网络接口管理器
		interfaceManager := network.NewInterfaceManager(s.logger)
		if err := interfaceManager.RefreshInterfaces(); err != nil {
			s.logger.WithError(err).Debug("刷新网络接口信息失败")
			return
		}

		// 更新所有网卡的信息指标
		interfaceManager.UpdateMetrics(func(interfaceName, ipAddress, macAddress, hostname string) {
			s.metrics.UpdateInterfaceInfo(interfaceName, ipAddress, macAddress, metrics.Hostname)
		})
	}
}

// setupLogger 设置日志
func setupLogger(logger *logrus.Logger, cfg *config.LogConfig) error {
	// 设置日志级别
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		return fmt.Errorf("无效的日志级别: %s", cfg.Level)
	}
	logger.SetLevel(level)

	// 设置日志格式
	switch cfg.Format {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	case "text":
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
		})
	default:
		return fmt.Errorf("不支持的日志格式: %s", cfg.Format)
	}

	// 设置日志输出
	switch cfg.Output {
	case "stdout":
		logger.SetOutput(os.Stdout)
	case "stderr":
		logger.SetOutput(os.Stderr)
	default:
		// 输出到文件
		file, err := os.OpenFile(cfg.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return fmt.Errorf("打开日志文件失败: %w", err)
		}
		logger.SetOutput(file)
	}

	return nil
}
