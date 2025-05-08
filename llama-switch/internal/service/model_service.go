package service

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"llama-switch/internal/config"
	"llama-switch/internal/model"
)

// ModelService 模型服务管理器
type ModelService struct {
	config         *config.Config
	processManager *ProcessManager
	mu             sync.RWMutex
}

// NewModelService 创建新的模型服务管理器
func NewModelService(cfg *config.Config) *ModelService {
	return &ModelService{
		config:         cfg,
		processManager: NewProcessManager(),
	}
}

// freeVRAM 释放足够显存(优先释放大显存模型)
func (s *ModelService) freeVRAM(required int) error {
	// 获取按显存使用排序的模型列表
	models := s.processManager.GetModelsByVRAMUsage()
	if len(models) == 0 {
		return fmt.Errorf("no running models to free VRAM from")
	}

	freed := 0
	stoppedModels := make([]string, 0)

	// 记录开始释放显存
	log.Printf("Attempting to free %dMB VRAM...", required)

	for _, m := range models {
		// 尝试停止模型进程
		if err := s.processManager.stopProcessByPID(m.ProcessID); err != nil {
			log.Printf("Warning: failed to stop model %s (PID: %d): %v",
				m.ModelName, m.ProcessID, err)
			continue
		}

		// 更新已释放显存
		freed += m.VRAMUsage
		s.processManager.RemoveModel(m.ProcessID)
		stoppedModels = append(stoppedModels,
			fmt.Sprintf("%s(%dMB)", m.ModelName, m.VRAMUsage))

		log.Printf("Stopped model %s, freed %dMB VRAM", m.ModelName, m.VRAMUsage)

		if freed >= required {
			log.Printf("Successfully freed %dMB VRAM by stopping models: %s",
				freed, strings.Join(stoppedModels, ", "))
			return nil
		}
	}

	return fmt.Errorf("could only free %dMB of %dMB required VRAM after stopping models: %s",
		freed, required, strings.Join(stoppedModels, ", "))
}

// StartModel 启动模型服务
func (s *ModelService) StartModel(cfg *model.ModelConfig) error {
	if cfg.ModelName == "" {
		return fmt.Errorf("model name is required")
	}

	// 检查是否存在同名模型
	existingModels := s.GetModelStatus(cfg.ModelName)
	if len(existingModels) > 0 {
		// 如果模型已存在且正在运行，直接返回
		if existingModels[0].Running {
			return fmt.Errorf("model with name '%s' is already running", cfg.ModelName)
		}
		// 如果模型存在但已停止，从管理器中移除
		s.processManager.RemoveModel(existingModels[0].ProcessID)
	}

	// 估算所需显存
	requiredVRAM := s.estimateVRAMUsage(cfg)

	// 检查显存
	if cfg.ForceVRAM || cfg.Config.NGPULayers > 0 {
		available, err := s.getAvailableVRAM()
		if err != nil {
			return fmt.Errorf("failed to check VRAM: %v", err)
		}

		if available < requiredVRAM {
			// 如果强制使用显存，尝试释放
			if cfg.ForceVRAM {
				if err := s.freeVRAM(requiredVRAM - available); err != nil {
					return fmt.Errorf("insufficient VRAM (required: %dMB, available: %dMB): %v",
						requiredVRAM, available, err)
				}
			} else {
				// 如果不强制使用显存，返回错误
				return fmt.Errorf("insufficient VRAM (required: %dMB, available: %dMB). Use force_vram=true to force start",
					requiredVRAM, available)
			}
		}
	}
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

	// 打印启动命令
	cmdStr := fmt.Sprintf("%s %s", s.config.LLamaPath.Server, strings.Join(args, " "))
	log.Printf("Starting model service with command:\n%s\n", cmdStr)

	// 启动服务进程
	if err := s.processManager.StartProcess(s.config.LLamaPath.Server, args); err != nil {
		return fmt.Errorf("failed to start model service: %v", err)
	}

	// 更新状态
	status := &model.ModelStatus{
		Running:   true,
		ModelName: cfg.ModelName,
		ModelPath: modelPath,
		Port:      cfg.Config.Port,
		StartTime: time.Now().Format(time.RFC3339),
		ProcessID: s.processManager.GetPID(),
		VRAMUsage: s.estimateVRAMUsage(cfg),
	}
	s.currentStatus = status

	// 添加到进程管理器
	s.processManager.AddModel(status.ProcessID, status)

	return nil
}

// getAvailableVRAM 获取当前可用显存(MB)
func (s *ModelService) getAvailableVRAM() (int, error) {
	cmd := exec.Command("nvidia-smi", "--query-gpu=memory.free", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to query GPU memory: %v", err)
	}

	// 解析输出，取第一个GPU的可用显存
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		return 0, fmt.Errorf("no GPU memory information available")
	}

	freeMB, err := strconv.Atoi(strings.TrimSpace(lines[0]))
	if err != nil {
		return 0, fmt.Errorf("failed to parse GPU memory: %v", err)
	}

	return freeMB, nil
}

// estimateVRAMUsage 估算模型所需显存(MB)
func (s *ModelService) estimateVRAMUsage(cfg *model.ModelConfig) int {
	// 简单估算：每GPU层大约需要200MB显存
	baseVRAM := 500 // 基础显存需求
	perLayer := 200 // 每层显存需求
	return baseVRAM + cfg.Config.NGPULayers*perLayer
}

// StopModel 停止最后启动的模型服务
func (s *ModelService) StopModel() (*model.ModelStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 获取所有运行中的模型
	models := s.processManager.GetRunningModels()
	if len(models) == 0 {
		return nil, fmt.Errorf("no model is currently running")
	}

	// 获取最后启动的模型（假设列表中最后一个是最近启动的）
	lastModel := models[len(models)-1]

	// 停止进程
	if err := s.processManager.stopProcessByPID(lastModel.ProcessID); err != nil {
		return nil, fmt.Errorf("failed to stop model service: %v", err)
	}

	// 从进程管理器中移除
	s.processManager.RemoveModel(lastModel.ProcessID)

	log.Printf("Stopped model '%s' (PID: %d)", lastModel.ModelName, lastModel.ProcessID)
	return lastModel, nil
}

// StopModelByName 按名称停止模型
func (s *ModelService) StopModelByName(name string) (*model.ModelStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 查找指定名称的模型
	models := s.processManager.GetRunningModels()
	var targetModel *model.ModelStatus
	for _, m := range models {
		if m.ModelName == name {
			targetModel = m
			break
		}
	}

	if targetModel == nil {
		return nil, fmt.Errorf("model '%s' not found or not running", name)
	}

	// 停止进程
	if err := s.processManager.stopProcessByPID(targetModel.ProcessID); err != nil {
		return nil, fmt.Errorf("failed to stop model '%s': %v", name, err)
	}

	// 从进程管理器中移除
	s.processManager.RemoveModel(targetModel.ProcessID)

	log.Printf("Stopped model '%s' (PID: %d)", targetModel.ModelName, targetModel.ProcessID)
	return targetModel, nil
}

// GetModelStatus 获取模型状态
func (s *ModelService) GetModelStatus(name string) []*model.ModelStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 统一从进程管理器获取所有模型状态
	allModels := s.processManager.GetRunningModels()

	// 同步当前模型状态
	if s.currentStatus != nil {
		// 检查当前模型是否在管理器中
		found := false
		for _, m := range allModels {
			if m.ProcessID == s.currentStatus.ProcessID {
				found = true
				break
			}
		}

		// 如果当前模型正在运行但不在管理器中，添加到结果
		if !found && s.currentStatus.Running {
			allModels = append(allModels, s.currentStatus)
		}

		// 如果当前模型已停止，从管理器中移除
		if !s.currentStatus.Running {
			s.processManager.RemoveModel(s.currentStatus.ProcessID)
		}
	}

	// 如果有指定名称，返回匹配的模型
	if name != "" {
		var result []*model.ModelStatus
		for _, m := range allModels {
			if m.ModelName == name {
				result = append(result, m)
			}
		}
		return result
	}

	// 返回所有模型状态
	return allModels
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
