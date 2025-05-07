package config

import (
	"fmt"
	"strings"
)

// String 返回格式化的配置信息
func (c *Config) String() string {
	var sb strings.Builder

	// 添加标题
	sb.WriteString("\n=== Configuration ===\n\n")

	// LLama.cpp 路径
	sb.WriteString("LLama.cpp Paths:\n")
	sb.WriteString(fmt.Sprintf("  %-15s: %s\n", "Server Binary", c.LLamaPath.Server))
	sb.WriteString(fmt.Sprintf("  %-15s: %s\n", "Bench Binary", c.LLamaPath.Bench))
	sb.WriteString("\n")

	// 模型目录
	sb.WriteString(fmt.Sprintf("Models Directory: %s\n\n", c.ModelsDir))

	// 服务器配置
	sb.WriteString("Server Configuration:\n")
	sb.WriteString(fmt.Sprintf("  %-15s: %s\n", "Host", c.Server.Host))
	sb.WriteString(fmt.Sprintf("  %-15s: %d\n", "Port", c.Server.Port))
	sb.WriteString(fmt.Sprintf("  %-15s: %d seconds\n", "Timeout", c.Server.Timeout))
	sb.WriteString("\n")

	// 默认模型配置
	sb.WriteString("Default Model Settings:\n")
	sb.WriteString(fmt.Sprintf("  %-15s: %d\n", "Threads", c.DefaultModel.Threads))
	sb.WriteString(fmt.Sprintf("  %-15s: %d\n", "Context Size", c.DefaultModel.CtxSize))
	sb.WriteString(fmt.Sprintf("  %-15s: %d\n", "Batch Size", c.DefaultModel.BatchSize))
	sb.WriteString(fmt.Sprintf("  %-15s: %d\n", "µBatch Size", c.DefaultModel.UBatchSize))
	sb.WriteString("\n")

	// GPU配置
	sb.WriteString("GPU Configuration:\n")
	sb.WriteString(fmt.Sprintf("  %-15s: %d\n", "GPU Layers", c.GPU.Layers))
	sb.WriteString(fmt.Sprintf("  %-15s: %s\n", "Split Mode", c.GPU.SplitMode))
	sb.WriteString(fmt.Sprintf("  %-15s: %d\n", "Main GPU", c.GPU.MainGPU))
	sb.WriteString(fmt.Sprintf("  %-15s: %v\n", "Flash Attention", c.GPU.FlashAttn))
	sb.WriteString("\n")

	// 缓存配置
	sb.WriteString("Cache Configuration:\n")
	sb.WriteString(fmt.Sprintf("  %-15s: %s\n", "K Cache Type", c.Cache.TypeK))
	sb.WriteString(fmt.Sprintf("  %-15s: %s\n", "V Cache Type", c.Cache.TypeV))
	sb.WriteString("\n")

	// 内存配置
	sb.WriteString("Memory Management:\n")
	sb.WriteString(fmt.Sprintf("  %-15s: %v\n", "mlock", c.Memory.Mlock))
	sb.WriteString(fmt.Sprintf("  %-15s: %v\n", "mmap", c.Memory.Mmap))
	if c.Memory.Numa != "" {
		sb.WriteString(fmt.Sprintf("  %-15s: %s\n", "NUMA Strategy", c.Memory.Numa))
	}
	sb.WriteString("\n")

	// 日志配置
	sb.WriteString("Logging Configuration:\n")
	sb.WriteString(fmt.Sprintf("  %-15s: %s\n", "Log Level", c.Log.Level))
	if c.Log.File != "" {
		sb.WriteString(fmt.Sprintf("  %-15s: %s\n", "Log File", c.Log.File))
	}
	sb.WriteString(fmt.Sprintf("  %-15s: %v\n", "Console Log", c.Log.EnableConsole))
	sb.WriteString("\n")

	// 安全配置
	sb.WriteString("Security Configuration:\n")
	if c.Security.APIKey != "" {
		sb.WriteString("  API Key        : [Set]\n")
	} else {
		sb.WriteString("  API Key        : [Not Set]\n")
	}
	if c.Security.SSLKey != "" && c.Security.SSLCert != "" {
		sb.WriteString("  SSL            : Enabled\n")
		sb.WriteString(fmt.Sprintf("  %-15s: %s\n", "SSL Key", c.Security.SSLKey))
		sb.WriteString(fmt.Sprintf("  %-15s: %s\n", "SSL Cert", c.Security.SSLCert))
	} else {
		sb.WriteString("  SSL            : Disabled\n")
	}
	sb.WriteString("\n")

	sb.WriteString("===================\n")

	return sb.String()
}
