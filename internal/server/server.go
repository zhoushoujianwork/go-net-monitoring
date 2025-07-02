package server

import (
	"context"
	"encoding/json"
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

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// Server 网络监控服务器
type Server struct {
	config     *config.ServerAppConfig
	logger     *logrus.Logger
	httpServer *http.Server
	metrics    *metrics.Metrics
	storage    Storage
	agents     map[string]*common.AgentInfo
	agentsMu   sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
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

	ctx, cancel := context.WithCancel(context.Background())

	// 初始化存储
	storage, err := NewStorage(&cfg.Storage)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("初始化存储失败: %w", err)
	}

	// 初始化指标
	metricsCollector := metrics.NewMetrics()

	server := &Server{
		config:  cfg,
		logger:  logger,
		metrics: metricsCollector,
		storage: storage,
		agents:  make(map[string]*common.AgentInfo),
		ctx:     ctx,
		cancel:  cancel,
	}

	// 设置HTTP服务器
	server.setupHTTPServer()

	return server, nil
}

// setupHTTPServer 设置HTTP服务器
func (s *Server) setupHTTPServer() {
	mux := http.NewServeMux()

	// API路由
	mux.HandleFunc("/api/v1/metrics", s.handleMetrics)
	mux.HandleFunc("/api/v1/heartbeat", s.handleHeartbeat)
	mux.HandleFunc("/api/v1/agents", s.handleAgents)
	mux.HandleFunc("/api/v1/status", s.handleStatus)

	// Prometheus指标路由
	if s.config.Metrics.Enabled {
		mux.Handle(s.config.Metrics.Path, promhttp.Handler())
	}

	// 健康检查路由
	mux.HandleFunc("/health", s.handleHealth)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.config.HTTP.Host, s.config.HTTP.Port),
		Handler:      s.loggingMiddleware(mux),
		ReadTimeout:  s.config.HTTP.ReadTimeout,
		WriteTimeout: s.config.HTTP.WriteTimeout,
	}
}

// Start 启动服务器
func (s *Server) Start() error {
	s.logger.Info("启动网络监控服务器")

	// 启动清理协程
	s.wg.Add(1)
	go s.cleanupWorker()

	// 启动HTTP服务器
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		
		s.logger.Infof("HTTP服务器监听地址: %s", s.httpServer.Addr)
		
		if s.config.HTTP.EnableTLS {
			if err := s.httpServer.ListenAndServeTLS(s.config.HTTP.TLSCertPath, s.config.HTTP.TLSKeyPath); err != nil && err != http.ErrServerClosed {
				s.logger.WithError(err).Error("HTTPS服务器启动失败")
			}
		} else {
			if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				s.logger.WithError(err).Error("HTTP服务器启动失败")
			}
		}
	}()

	s.logger.Info("网络监控服务器启动成功")
	return nil
}

// Stop 停止服务器
func (s *Server) Stop() error {
	s.logger.Info("停止网络监控服务器")

	s.cancel()

	// 停止HTTP服务器
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.WithError(err).Error("停止HTTP服务器失败")
	}

	// 关闭存储
	if err := s.storage.Close(); err != nil {
		s.logger.WithError(err).Error("关闭存储失败")
	}

	s.wg.Wait()

	s.logger.Info("网络监控服务器已停止")
	return nil
}

// Run 运行服务器（阻塞直到收到停止信号）
func (s *Server) Run() error {
	// 启动服务器
	if err := s.Start(); err != nil {
		return err
	}

	// 等待停止信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		s.logger.Infof("收到信号 %v，开始停止服务器", sig)
	case <-s.ctx.Done():
		s.logger.Info("服务器上下文已取消")
	}

	// 停止服务器
	return s.Stop()
}

// handleMetrics 处理指标上报
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request common.ReportRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.logger.WithError(err).Error("解析指标请求失败")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 更新Agent信息
	s.updateAgentInfo(request.AgentID, request.Hostname)

	// 存储指标数据
	key := fmt.Sprintf("metrics:%s:%d", request.AgentID, request.Timestamp.Unix())
	if err := s.storage.Store(key, request.Metrics); err != nil {
		s.logger.WithError(err).Error("存储指标数据失败")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// 更新Prometheus指标
	s.metrics.UpdateNetworkMetrics(request.Metrics)

	// 返回响应
	response := common.ReportResponse{
		Success:   true,
		Message:   "Metrics received successfully",
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	s.logger.WithFields(logrus.Fields{
		"agent_id": request.AgentID,
		"hostname": request.Hostname,
	}).Debug("成功接收指标数据")
}

// handleHeartbeat 处理心跳
func (s *Server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var agentInfo common.AgentInfo
	if err := json.NewDecoder(r.Body).Decode(&agentInfo); err != nil {
		s.logger.WithError(err).Error("解析心跳请求失败")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 更新Agent信息
	s.agentsMu.Lock()
	agentInfo.LastSeen = time.Now()
	s.agents[agentInfo.ID] = &agentInfo
	s.agentsMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"timestamp": time.Now(),
	})

	s.logger.WithFields(logrus.Fields{
		"agent_id": agentInfo.ID,
		"hostname": agentInfo.Hostname,
	}).Debug("收到Agent心跳")
}

// handleAgents 处理Agent列表查询
func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.agentsMu.RLock()
	agents := make([]*common.AgentInfo, 0, len(s.agents))
	for _, agent := range s.agents {
		agents = append(agents, agent)
	}
	s.agentsMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"agents": agents,
		"count":  len(agents),
	})
}

// handleStatus 处理状态查询
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.agentsMu.RLock()
	agentCount := len(s.agents)
	s.agentsMu.RUnlock()

	status := map[string]interface{}{
		"status":      "running",
		"timestamp":   time.Now(),
		"agent_count": agentCount,
		"version":     "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleHealth 处理健康检查
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"timestamp": time.Now(),
	})
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
	}
}

// cleanupWorker 清理工作协程
func (s *Server) cleanupWorker() {
	defer s.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.cleanupOfflineAgents()
		}
	}
}

// cleanupOfflineAgents 清理离线Agent
func (s *Server) cleanupOfflineAgents() {
	s.agentsMu.Lock()
	defer s.agentsMu.Unlock()

	now := time.Now()
	offlineThreshold := 5 * time.Minute

	for agentID, agent := range s.agents {
		if now.Sub(agent.LastSeen) > offlineThreshold {
			agent.Status = "offline"
			s.logger.WithFields(logrus.Fields{
				"agent_id": agentID,
				"hostname": agent.Hostname,
				"last_seen": agent.LastSeen,
			}).Warn("Agent离线")
		}
	}
}

// loggingMiddleware 日志中间件
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// 包装ResponseWriter以捕获状态码
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		next.ServeHTTP(wrapped, r)
		
		duration := time.Since(start)
		
		s.logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"path":        r.URL.Path,
			"status":      wrapped.statusCode,
			"duration":    duration.String(),
			"remote_addr": r.RemoteAddr,
			"user_agent":  r.UserAgent(),
		}).Info("HTTP请求")
	})
}

// responseWriter 包装ResponseWriter以捕获状态码
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
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
