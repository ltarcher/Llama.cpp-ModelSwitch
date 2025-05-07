package service

import (
	"fmt"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"llama-switch/internal/config"
	"llama-switch/internal/model"
)

// ModelService 模型服务管理器
type ModelService struct {
	config         *config.Config
	processManager *ProcessManager
	currentStatus  *model.ModelStatus
	mu             sync.RWMutex
}

// NewModelService 创建新的模型服务管理器
func NewModelService(cfg *config.Config) *ModelService {
	return &ModelService{
		config:         cfg,
		processManager: NewProcessManager(),
		currentStatus: &model.ModelStatus{
			Running: false,
		},
	}
}

// StartModel 启动模型服务
func (s *ModelService) StartModel(cfg *model.ModelConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 验证模型文件路径
	modelPath := cfg.ModelPath
	if !filepath.IsAbs(modelPath) {
		modelPath = filepath.Join(s.config.ModelsDir, cfg.ModelPath)
	}
	if _, err := filepath.Abs(modelPath); err != nil {
		return fmt.Errorf("invalid model path: %v", err)
	}

	// 构建命令行参数
	args := []string{
		"--model", modelPath,
	}

	c := cfg.Config

	// 服务器配置
	if c.Host != "" {
		args = append(args, "--host", c.Host)
	}
	if c.Port > 0 {
		args = append(args, "--port", strconv.Itoa(c.Port))
	}
	if c.Timeout > 0 {
		args = append(args, "--timeout", strconv.Itoa(c.Timeout))
	}

	// 系统资源配置
	if c.Threads > 0 {
		args = append(args, "--threads", strconv.Itoa(c.Threads))
	}
	if c.ThreadsBatch > 0 {
		args = append(args, "--threads-batch", strconv.Itoa(c.ThreadsBatch))
	}
	if c.CPUMask != "" {
		args = append(args, "--cpu-mask", c.CPUMask)
	}
	if c.CPURange != "" {
		args = append(args, "--cpu-range", c.CPURange)
	}
	if c.CPUStrict > 0 {
		args = append(args, "--cpu-strict", strconv.Itoa(c.CPUStrict))
	}
	if c.Priority > 0 {
		args = append(args, "--prio", strconv.Itoa(c.Priority))
	}
	if c.Poll >= 0 {
		args = append(args, "--poll", strconv.Itoa(c.Poll))
	}

	// 模型参数
	if c.CtxSize > 0 {
		args = append(args, "--ctx-size", strconv.Itoa(c.CtxSize))
	}
	if c.BatchSize > 0 {
		args = append(args, "--batch-size", strconv.Itoa(c.BatchSize))
	}
	if c.UBatchSize > 0 {
		args = append(args, "--ubatch-size", strconv.Itoa(c.UBatchSize))
	}
	if c.NPredict != 0 {
		args = append(args, "--n-predict", strconv.Itoa(c.NPredict))
	}
	if c.Keep != 0 {
		args = append(args, "--keep", strconv.Itoa(c.Keep))
	}

	// GPU相关配置
	if c.NGPULayers > 0 {
		args = append(args, "--n-gpu-layers", strconv.Itoa(c.NGPULayers))
	}
	if c.SplitMode != "" {
		args = append(args, "--split-mode", c.SplitMode)
	}
	if c.TensorSplit != "" {
		args = append(args, "--tensor-split", c.TensorSplit)
	}
	if c.MainGPU >= 0 {
		args = append(args, "--main-gpu", strconv.Itoa(c.MainGPU))
	}
	if c.Device != "" {
		args = append(args, "--device", c.Device)
	}

	// 内存管理
	if c.Mlock {
		args = append(args, "--mlock")
	}
	if c.NoMMap {
		args = append(args, "--no-mmap")
	}
	if c.Numa != "" {
		args = append(args, "--numa", c.Numa)
	}
	if c.NoKVOffload {
		args = append(args, "--no-kv-offload")
	}

	// 缓存配置
	if c.CacheTypeK != "" {
		args = append(args, "--cache-type-k", c.CacheTypeK)
	}
	if c.CacheTypeV != "" {
		args = append(args, "--cache-type-v", c.CacheTypeV)
	}
	if c.DefragThold > 0 {
		args = append(args, "--defrag-thold", fmt.Sprintf("%.2f", c.DefragThold))
	}

	// 性能优化
	if c.FlashAttn {
		args = append(args, "--flash-attn")
	}
	if c.NoPerfTimer {
		args = append(args, "--no-perf")
	}

	// RoPE配置
	if c.RopeScaling != "" {
		args = append(args, "--rope-scaling", c.RopeScaling)
	}
	if c.RopeScale > 0 {
		args = append(args, "--rope-scale", fmt.Sprintf("%.2f", c.RopeScale))
	}
	if c.RopeFreqBase > 0 {
		args = append(args, "--rope-freq-base", fmt.Sprintf("%.2f", c.RopeFreqBase))
	}
	if c.RopeFreqScale > 0 {
		args = append(args, "--rope-freq-scale", fmt.Sprintf("%.2f", c.RopeFreqScale))
	}

	// YaRN配置
	if c.YarnOrigCtx > 0 {
		args = append(args, "--yarn-orig-ctx", strconv.Itoa(c.YarnOrigCtx))
	}
	if c.YarnExtFactor >= 0 {
		args = append(args, "--yarn-ext-factor", fmt.Sprintf("%.2f", c.YarnExtFactor))
	}
	if c.YarnAttnFactor > 0 {
		args = append(args, "--yarn-attn-factor", fmt.Sprintf("%.2f", c.YarnAttnFactor))
	}
	if c.YarnBetaSlow > 0 {
		args = append(args, "--yarn-beta-slow", fmt.Sprintf("%.2f", c.YarnBetaSlow))
	}
	if c.YarnBetaFast > 0 {
		args = append(args, "--yarn-beta-fast", fmt.Sprintf("%.2f", c.YarnBetaFast))
	}

	// 其他功能
	if c.Verbose {
		args = append(args, "--verbose")
	}
	if c.LogFile != "" {
		args = append(args, "--log-file", c.LogFile)
	}
	if c.StaticPath != "" {
		args = append(args, "--path", c.StaticPath)
	}
	if c.APIKey != "" {
		args = append(args, "--api-key", c.APIKey)
	}
	if c.SSLKey != "" {
		args = append(args, "--ssl-key-file", c.SSLKey)
	}
	if c.SSLCert != "" {
		args = append(args, "--ssl-cert-file", c.SSLCert)
	}

	// 启动服务进程
	if err := s.processManager.StartProcess(s.config.LLamaPath.Server, args); err != nil {
		return fmt.Errorf("failed to start model service: %v", err)
	}

	// 更新状态
	s.currentStatus = &model.ModelStatus{
		Running:   true,
		ModelPath: modelPath,
		Port:      cfg.Config.Port,
		StartTime: time.Now().Format(time.RFC3339),
		ProcessID: s.processManager.GetPID(),
	}

	return nil
}

// StopModel 停止模型服务
func (s *ModelService) StopModel() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.processManager.StopProcess(); err != nil {
		return fmt.Errorf("failed to stop model service: %v", err)
	}

	s.currentStatus = &model.ModelStatus{
		Running: false,
	}

	return nil
}

// GetStatus 获取当前模型服务状态
func (s *ModelService) GetStatus() *model.ModelStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 确保进程状态与记录的状态一致
	if s.currentStatus.Running && !s.processManager.IsRunning() {
		s.currentStatus.Running = false
	}

	return s.currentStatus
}

// ValidateModelConfig 验证模型配置
func (s *ModelService) ValidateModelConfig(cfg *model.ModelConfig) error {
	if cfg.ModelPath == "" {
		return fmt.Errorf("model path is required")
	}

	c := cfg.Config

	// 验证服务器配置
	if c.Port < 0 || c.Port > 65535 {
		return fmt.Errorf("invalid port number: %d", c.Port)
	}
	if c.Timeout < 0 {
		return fmt.Errorf("invalid timeout value: %d", c.Timeout)
	}

	// 验证系统资源配置
	if c.Threads < -1 {
		return fmt.Errorf("invalid threads number: %d", c.Threads)
	}
	if c.ThreadsBatch < -1 {
		return fmt.Errorf("invalid threads batch number: %d", c.ThreadsBatch)
	}
	if c.Priority < 0 || c.Priority > 3 {
		return fmt.Errorf("invalid priority value: %d (should be between 0 and 3)", c.Priority)
	}
	if c.Poll < 0 || c.Poll > 100 {
		return fmt.Errorf("invalid poll value: %d (should be between 0 and 100)", c.Poll)
	}

	// 验证模型参数
	if c.CtxSize < 0 {
		return fmt.Errorf("invalid context size: %d", c.CtxSize)
	}
	if c.BatchSize < 0 {
		return fmt.Errorf("invalid batch size: %d", c.BatchSize)
	}
	if c.UBatchSize < 0 {
		return fmt.Errorf("invalid micro batch size: %d", c.UBatchSize)
	}

	// 验证GPU配置
	if c.NGPULayers < 0 {
		return fmt.Errorf("invalid number of GPU layers: %d", c.NGPULayers)
	}
	if c.SplitMode != "" && c.SplitMode != "none" && c.SplitMode != "layer" && c.SplitMode != "row" {
		return fmt.Errorf("invalid split mode: %s (should be none, layer, or row)", c.SplitMode)
	}
	if c.MainGPU < 0 {
		return fmt.Errorf("invalid main GPU index: %d", c.MainGPU)
	}

	// 验证NUMA配置
	if c.Numa != "" && c.Numa != "distribute" && c.Numa != "isolate" && c.Numa != "numactl" {
		return fmt.Errorf("invalid NUMA value: %s (should be distribute, isolate, or numactl)", c.Numa)
	}

	// 验证缓存配置
	validCacheTypes := map[string]bool{
		"f32": true, "f16": true, "bf16": true,
		"q8_0": true, "q4_0": true, "q4_1": true,
		"iq4_nl": true, "q5_0": true, "q5_1": true,
	}
	if c.CacheTypeK != "" && !validCacheTypes[c.CacheTypeK] {
		return fmt.Errorf("invalid cache type K: %s", c.CacheTypeK)
	}
	if c.CacheTypeV != "" && !validCacheTypes[c.CacheTypeV] {
		return fmt.Errorf("invalid cache type V: %s", c.CacheTypeV)
	}
	if c.DefragThold < 0 || c.DefragThold > 1 {
		return fmt.Errorf("invalid defrag threshold: %.2f (should be between 0 and 1)", c.DefragThold)
	}

	// 验证RoPE配置
	if c.RopeScaling != "" && c.RopeScaling != "none" && c.RopeScaling != "linear" && c.RopeScaling != "yarn" {
		return fmt.Errorf("invalid RoPE scaling: %s (should be none, linear, or yarn)", c.RopeScaling)
	}
	if c.RopeScale < 0 {
		return fmt.Errorf("invalid RoPE scale: %.2f", c.RopeScale)
	}
	if c.RopeFreqBase < 0 {
		return fmt.Errorf("invalid RoPE frequency base: %.2f", c.RopeFreqBase)
	}
	if c.RopeFreqScale < 0 {
		return fmt.Errorf("invalid RoPE frequency scale: %.2f", c.RopeFreqScale)
	}

	// 验证YaRN配置
	if c.YarnOrigCtx < 0 {
		return fmt.Errorf("invalid YaRN original context size: %d", c.YarnOrigCtx)
	}
	if c.YarnExtFactor < -1 {
		return fmt.Errorf("invalid YaRN extrapolation factor: %.2f", c.YarnExtFactor)
	}
	if c.YarnAttnFactor < 0 {
		return fmt.Errorf("invalid YaRN attention factor: %.2f", c.YarnAttnFactor)
	}
	if c.YarnBetaSlow < 0 {
		return fmt.Errorf("invalid YaRN beta slow: %.2f", c.YarnBetaSlow)
	}
	if c.YarnBetaFast < 0 {
		return fmt.Errorf("invalid YaRN beta fast: %.2f", c.YarnBetaFast)
	}

	return nil
}
