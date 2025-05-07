package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config 应用程序的主配置结构
type Config struct {
	// LLamaPath llama.cpp二进制文件路径
	LLamaPath struct {
		Server string `json:"server"` // llama-server路径
		Bench  string `json:"bench"`  // llama-bench路径
	} `json:"llama_path"`

	// ModelsDir 模型文件目录
	ModelsDir string `json:"models_dir"`

	// Server API服务器配置
	Server struct {
		Host    string `json:"host"`
		Port    int    `json:"port"`
		Timeout int    `json:"timeout"`
	} `json:"server"`

	// DefaultModel 默认模型配置
	DefaultModel struct {
		Threads    int `json:"threads"`
		CtxSize    int `json:"ctx_size"`
		BatchSize  int `json:"batch_size"`
		UBatchSize int `json:"ubatch_size"`
	} `json:"default_model"`

	// GPU 默认GPU配置
	GPU struct {
		Layers    int    `json:"layers"`
		SplitMode string `json:"split_mode"`
		MainGPU   int    `json:"main_gpu"`
		FlashAttn bool   `json:"flash_attn"`
	} `json:"gpu"`

	// Cache 缓存配置
	Cache struct {
		TypeK string `json:"type_k"`
		TypeV string `json:"type_v"`
	} `json:"cache"`

	// Memory 内存管理配置
	Memory struct {
		Mlock bool   `json:"mlock"`
		Mmap  bool   `json:"mmap"`
		Numa  string `json:"numa"`
	} `json:"memory"`

	// Log 日志配置
	Log struct {
		Level         string `json:"level"`
		File          string `json:"file"`
		EnableConsole bool   `json:"enable_console"`
	} `json:"log"`

	// Security 安全配置
	Security struct {
		APIKey  string `json:"api_key"`
		SSLKey  string `json:"ssl_key"`
		SSLCert string `json:"ssl_cert"`
	} `json:"security"`
}

// LoadConfig 加载配置
func LoadConfig() (*Config, error) {
	// 首先尝试加载.env文件
	if err := godotenv.Load(); err != nil {
		// 如果.env文件不存在，尝试加载.env.example
		if err := godotenv.Load(".env.example"); err != nil {
			fmt.Printf("Warning: No .env or .env.example file found\n")
		}
	}

	cfg := &Config{}

	// 加载二进制文件路径
	cfg.LLamaPath.Server = getEnv("LLAMA_SERVER_PATH", "E:/Downloads/llama-b5293-bin-win-cuda-cu12.4-x64/llama-server.exe")
	cfg.LLamaPath.Bench = getEnv("LLAMA_BENCH_PATH", "E:/Downloads/llama-b5293-bin-win-cuda-cu12.4-x64/llama-bench.exe")

	// 加载模型目录
	cfg.ModelsDir = getEnv("MODELS_DIR", "E:/develop/Models")

	// 加载服务器配置
	cfg.Server.Host = getEnv("SERVER_HOST", "127.0.0.1")
	cfg.Server.Port = getEnvInt("SERVER_PORT", 8080)
	cfg.Server.Timeout = getEnvInt("SERVER_TIMEOUT", 600)

	// 加载默认模型配置
	cfg.DefaultModel.Threads = getEnvInt("DEFAULT_THREADS", 8)
	cfg.DefaultModel.CtxSize = getEnvInt("DEFAULT_CTX_SIZE", 4096)
	cfg.DefaultModel.BatchSize = getEnvInt("DEFAULT_BATCH_SIZE", 512)
	cfg.DefaultModel.UBatchSize = getEnvInt("DEFAULT_UBATCH_SIZE", 512)

	// 加载GPU配置
	cfg.GPU.Layers = getEnvInt("DEFAULT_GPU_LAYERS", 99)
	cfg.GPU.SplitMode = getEnv("DEFAULT_SPLIT_MODE", "layer")
	cfg.GPU.MainGPU = getEnvInt("DEFAULT_MAIN_GPU", 0)
	cfg.GPU.FlashAttn = getEnvBool("ENABLE_FLASH_ATTN", true)

	// 加载缓存配置
	cfg.Cache.TypeK = getEnv("DEFAULT_CACHE_TYPE_K", "f16")
	cfg.Cache.TypeV = getEnv("DEFAULT_CACHE_TYPE_V", "f16")

	// 加载内存管理配置
	cfg.Memory.Mlock = getEnvBool("ENABLE_MLOCK", false)
	cfg.Memory.Mmap = getEnvBool("ENABLE_MMAP", true)
	cfg.Memory.Numa = getEnv("NUMA_STRATEGY", "")

	// 加载日志配置
	cfg.Log.Level = getEnv("LOG_LEVEL", "info")
	cfg.Log.File = getEnv("LOG_FILE", "")
	cfg.Log.EnableConsole = getEnvBool("ENABLE_CONSOLE_LOG", true)

	// 加载安全配置
	cfg.Security.APIKey = getEnv("API_KEY", "")
	cfg.Security.SSLKey = getEnv("SSL_KEY_FILE", "")
	cfg.Security.SSLCert = getEnv("SSL_CERT_FILE", "")

	return cfg, nil
}

// 辅助函数：获取环境变量，如果不存在则返回默认值
func getEnv(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// 辅助函数：获取整数类型的环境变量
func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// 辅助函数：获取布尔类型的环境变量
func getEnvBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		return strings.ToLower(value) == "true" || value == "1"
	}
	return defaultValue
}

// ValidateConfig 验证配置
func ValidateConfig(cfg *Config) error {
	// 验证文件路径
	if !fileExists(cfg.LLamaPath.Server) {
		return fmt.Errorf("llama-server not found at: %s", cfg.LLamaPath.Server)
	}
	if !fileExists(cfg.LLamaPath.Bench) {
		return fmt.Errorf("llama-bench not found at: %s", cfg.LLamaPath.Bench)
	}

	// 验证模型目录
	if !directoryExists(cfg.ModelsDir) {
		return fmt.Errorf("models directory not found at: %s", cfg.ModelsDir)
	}

	// 验证端口范围
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return fmt.Errorf("invalid port number: %d", cfg.Server.Port)
	}

	// 验证超时设置
	if cfg.Server.Timeout < 0 {
		return fmt.Errorf("invalid timeout value: %d", cfg.Server.Timeout)
	}

	// 验证模型参数
	if cfg.DefaultModel.Threads < -1 {
		return fmt.Errorf("invalid threads number: %d", cfg.DefaultModel.Threads)
	}
	if cfg.DefaultModel.CtxSize < 0 {
		return fmt.Errorf("invalid context size: %d", cfg.DefaultModel.CtxSize)
	}
	if cfg.DefaultModel.BatchSize < 0 {
		return fmt.Errorf("invalid batch size: %d", cfg.DefaultModel.BatchSize)
	}
	if cfg.DefaultModel.UBatchSize < 0 {
		return fmt.Errorf("invalid micro batch size: %d", cfg.DefaultModel.UBatchSize)
	}

	// 验证GPU配置
	if cfg.GPU.Layers < 0 {
		return fmt.Errorf("invalid GPU layers: %d", cfg.GPU.Layers)
	}
	validSplitModes := map[string]bool{"none": true, "layer": true, "row": true}
	if !validSplitModes[cfg.GPU.SplitMode] {
		return fmt.Errorf("invalid split mode: %s", cfg.GPU.SplitMode)
	}

	// 验证缓存类型
	validCacheTypes := map[string]bool{
		"f32": true, "f16": true, "bf16": true,
		"q8_0": true, "q4_0": true, "q4_1": true,
		"iq4_nl": true, "q5_0": true, "q5_1": true,
	}
	if !validCacheTypes[cfg.Cache.TypeK] {
		return fmt.Errorf("invalid cache type K: %s", cfg.Cache.TypeK)
	}
	if !validCacheTypes[cfg.Cache.TypeV] {
		return fmt.Errorf("invalid cache type V: %s", cfg.Cache.TypeV)
	}

	// 验证NUMA策略
	if cfg.Memory.Numa != "" {
		validNumaStrategies := map[string]bool{
			"distribute": true,
			"isolate":    true,
			"numactl":    true,
		}
		if !validNumaStrategies[cfg.Memory.Numa] {
			return fmt.Errorf("invalid NUMA strategy: %s", cfg.Memory.Numa)
		}
	}

	// 验证日志级别
	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
	}
	if !validLogLevels[strings.ToLower(cfg.Log.Level)] {
		return fmt.Errorf("invalid log level: %s", cfg.Log.Level)
	}

	// 验证SSL配置
	if cfg.Security.SSLKey != "" && cfg.Security.SSLCert == "" {
		return fmt.Errorf("SSL key file specified but certificate file is missing")
	}
	if cfg.Security.SSLCert != "" && cfg.Security.SSLKey == "" {
		return fmt.Errorf("SSL certificate file specified but key file is missing")
	}

	return nil
}

// 辅助函数：检查文件是否存在
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// 辅助函数：检查目录是否存在
func directoryExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
