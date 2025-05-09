package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"llama-switch/internal/config"
	"llama-switch/internal/handler"
	"llama-switch/internal/service"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v\n", err)
	}

	// 验证配置
	if err := config.ValidateConfig(cfg); err != nil {
		log.Fatalf("Invalid configuration: %v\n", err)
	}

	// 初始化模型服务 (启用自动恢复)
	modelService := service.NewModelService(cfg, true)

	// 输出持久化配置路径
	configDir := filepath.Join(filepath.Dir(cfg.ModelsDir), "config")
	configPath := filepath.Join(configDir, "model_persistent.json")
	log.Printf("Persistent config location: %s", configPath)

	// 启动时恢复之前运行的模型
	if err := modelService.RestoreModels(); err != nil {
		log.Printf("Warning: Failed to restore models: %v", err)
	}

	// 初始化基准测试服务
	benchmarkService := service.NewBenchmarkService(cfg)

	// 创建处理器
	h := handler.NewHandlerWithService(cfg, modelService, benchmarkService)

	// 设置带日志的路由
	mux := http.NewServeMux()

	// 请求日志中间件
	loggingMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Incoming request: %s %s", r.Method, r.URL.Path)
			next(w, r)
		}
	}

	// 模型服务相关路由
	mux.HandleFunc("/api/v1/model/switch", loggingMiddleware(h.SwitchModel))
	mux.HandleFunc("/api/v1/model/stop", loggingMiddleware(h.StopModel))
	mux.HandleFunc("/api/v1/model/status", loggingMiddleware(h.GetModelStatus))

	// 基准测试相关路由
	mux.HandleFunc("/api/v1/benchmark", loggingMiddleware(h.StartBenchmark))
	mux.HandleFunc("/api/v1/benchmark/status", loggingMiddleware(h.GetBenchmarkStatus))

	// 添加健康检查端点
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	log.Println("Registered API endpoints:")
	log.Println("POST   /api/v1/model/switch")
	log.Println("POST   /api/v1/model/stop")
	log.Println("GET    /api/v1/model/status")
	log.Println("POST   /api/v1/benchmark")
	log.Println("GET    /api/v1/benchmark/status")
	log.Println("GET    /health")

	// 创建服务器
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler: mux,
	}

	// 创建取消上下文，用于控制关闭流程
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 打印配置信息
	log.Print(cfg.String())

	// 设置优雅关闭
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Initiating graceful shutdown...")

		// 取消上下文
		cancel()

		// 停止模型服务
		if _, err := h.ModelService.StopAllModel(); err != nil {
			log.Printf("Error stopping model service: %v\n", err)
		}

		// 清理基准测试服务
		h.BenchmarkService.Cleanup()

		// 关闭HTTP服务器
		if err := server.Close(); err != nil {
			log.Printf("Error during server shutdown: %v\n", err)
		}

		log.Println("Server shutdown completed")
	}()

	// 打印注册的路由
	log.Println("Registered routes:")
	for _, route := range []struct {
		path    string
		handler string
	}{
		{"/api/v1/model/switch", "SwitchModel"},
		{"/api/v1/model/stop", "StopModel"},
		{"/api/v1/model/status", "GetModelStatus"},
		{"/api/v1/benchmark", "StartBenchmark"},
		{"/api/v1/benchmark/status", "GetBenchmarkStatus"},
	} {
		log.Printf("  %-25s -> %s\n", route.path, route.handler)
	}

	// 启动服务器
	log.Printf("Server starting on %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Printf("Server error: %v\n", err)
		cancel() // 确保在服务器错误时也能触发清理
	}
}
