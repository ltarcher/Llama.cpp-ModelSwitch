package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"llama-switch/internal/config"
	"llama-switch/internal/handler"
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

	// 创建处理器
	h := handler.NewHandler(cfg)

	// 设置路由
	mux := http.NewServeMux()

	// 模型服务相关路由
	mux.HandleFunc("/api/v1/model/switch", h.SwitchModel)
	mux.HandleFunc("/api/v1/model/stop", h.StopModel)
	mux.HandleFunc("/api/v1/model/status", h.GetModelStatus)

	// 基准测试相关路由
	mux.HandleFunc("/api/v1/benchmark", h.StartBenchmark)
	mux.HandleFunc("/api/v1/benchmark/status", h.GetBenchmarkStatus)

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
		if err := h.ModelService.StopModel(); err != nil {
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

	// 启动服务器
	log.Printf("Server starting on %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Printf("Server error: %v\n", err)
		cancel() // 确保在服务器错误时也能触发清理
	}
}
