package config

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
		Host string `json:"host"`
		Port int    `json:"port"`
	} `json:"server"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	cfg := &Config{}

	// 设置默认的二进制文件路径
	cfg.LLamaPath.Server = "E:/Downloads/llama-b5293-bin-win-cuda-cu12.4-x64/llama-server.exe"
	cfg.LLamaPath.Bench = "E:/Downloads/llama-b5293-bin-win-cuda-cu12.4-x64/llama-bench.exe"

	// 设置默认的模型目录
	cfg.ModelsDir = "E:/develop/Models"

	// 设置默认的服务器配置
	cfg.Server.Host = "127.0.0.1"
	cfg.Server.Port = 8080

	return cfg
}
