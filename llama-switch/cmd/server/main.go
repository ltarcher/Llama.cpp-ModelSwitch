package main

import (
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

	// 设置优雅关闭
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")
		if err := server.Close(); err != nil {
			log.Printf("Error during server shutdown: %v\n", err)
		}
	}()

	// 打印配置信息
	log.Print(cfg.String())

	// 启动服务器
	log.Printf("Server starting on %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server error: %v\n", err)
	}
}
