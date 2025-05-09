package service

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	persistentMgr  *config.PersistentManager
	mu             sync.RWMutex
	autoRestore    bool
}

// NewModelService 创建新的模型服务管理器
func NewModelService(cfg *config.Config, autoRestore bool) *ModelService {
	return &ModelService{
		config:         cfg,
		processManager: NewProcessManager(),
		persistentMgr:  config.NewPersistentManager(cfg),
		autoRestore:    autoRestore,
	}
}

// RestoreModels 从持久化配置恢复模型
func (s *ModelService) RestoreModels() error {
	if !s.autoRestore {
		log.Println("Auto restore is disabled, skipping model restoration")
		return nil
	}

	configs, err := s.persistentMgr.GetModelConfigs()
	if err != nil {
		return fmt.Errorf("failed to load persistent configs: %v", err)
	}

	if len(configs) == 0 {
		log.Println("No models to restore")
		return nil
	}

	var lastError error
	restoredCount := 0

	for modelName, item := range configs {
		// 跳过配置不完整的模型
		if item.ModelConfig == nil {
			continue
		}

		// 检查模型是否已在运行
		if item.LastStatus.Running && item.LastStatus.ProcessID > 0 {
			// 验证进程是否实际存在
			if s.processManager.IsProcessRunning(item.LastStatus.ProcessID) {
				log.Printf("Model %s is already running (PID: %d), skipping restore",
					modelName, item.LastStatus.ProcessID)
				continue
			} else {
				// 进程已终止但状态未更新，修正状态
				log.Printf("Model %s process (PID: %d) not found, updating status",
					modelName, item.LastStatus.ProcessID)
				item.LastStatus.Running = false
				item.LastStatus.StopTime = time.Now().Format(time.RFC3339)
				if err := s.persistentMgr.UpdateModelConfig(modelName, item.ModelConfig, &item.LastStatus); err != nil {
					log.Printf("Warning: Failed to update model status: %v", err)
				}
				continue
			}
		}

		// 验证模型配置
		if err := s.ValidateModelConfig(item.ModelConfig); err != nil {
			log.Printf("Invalid config for model %s: %v", modelName, err)
			lastError = err
			continue
		}

		log.Printf("Restoring model: %s", modelName)
		_, err := s.StartModel(item.ModelConfig)
		if err != nil {
			log.Printf("Failed to restore model %s: %v", modelName, err)
			lastError = err
			continue
		}
		restoredCount++
	}

	if restoredCount > 0 {
		log.Printf("Successfully restored %d models", restoredCount)
	}

	if lastError != nil {
		return fmt.Errorf("some models failed to restore, last error: %v", lastError)
	}

	return nil
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

	for _, m := range models {
		// 可以停止任何模型来释放VRAM

		// 尝试停止模型进程
		if err := s.processManager.stopProcessByPID(m.ProcessID); err != nil {
			log.Printf("Warning: failed to stop model %s (PID: %d): %v",
				m.ModelName, m.ProcessID, err)
			continue
		}

		// 更新已释放显存
		freed += m.VRAMUsage
		s.processManager.RemoveModel(m.ProcessID)
		stoppedModels = append(stoppedModels, m.ModelName)

		if freed >= required {
			log.Printf("Freed %dMB VRAM by stopping models: %s",
				freed, strings.Join(stoppedModels, ", "))
			return nil
		}
	}

	return fmt.Errorf("could only free %dMB of %dMB required VRAM after stopping models: %s",
		freed, required, strings.Join(stoppedModels, ", "))
}

// StartModel 启动模型服务并返回状态
func (s *ModelService) StartModel(cfg *model.ModelConfig) (status *model.ModelStatus, err error) {
	// 初始化状态对象
	status = &model.ModelStatus{
		ModelName: cfg.ModelName,
		Running:   false,
	}
	if cfg.ModelName == "" {
		return nil, fmt.Errorf("model name is required")
	}

	// 检查是否存在同名模型
	existingModels := s.GetModelStatus(cfg.ModelName)
	if len(existingModels) > 0 {
		// 如果模型已存在且正在运行，直接返回
		if existingModels[0].Running {
			return nil, fmt.Errorf("model with name '%s' is already running", cfg.ModelName)
		}
		// 如果模型存在但已停止，从管理器中移除
		s.processManager.RemoveModel(existingModels[0].ProcessID)
	}

	// 验证模型文件路径
	modelPath := cfg.ModelPath
	if !filepath.IsAbs(modelPath) {
		modelPath = filepath.Join(s.config.ModelsDir, cfg.ModelPath)
	}
	if _, err := filepath.Abs(modelPath); err != nil {
		return nil, fmt.Errorf("invalid model path: %v", err)
	}

	// 获取模型文件大小
	fileInfo, err := os.Stat(modelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get model file info: %v", err)
	}
	modelSizeMB := fileInfo.Size() / (1024 * 1024)

	// 估算所需显存
	requiredVRAM := min(s.estimateVRAMUsage(cfg), int(modelSizeMB))

	// 记录估算信息
	log.Printf("Model VRAM estimation - FileSize: %dMB, EstimatedVRAM: %dMB",
		modelSizeMB, requiredVRAM)

	// 检查显存
	if cfg.ForceVRAM || cfg.Config.NGPULayers > 0 {
		// 计算所有运行中模型使用的VRAM总和
		var usedVRAM int
		for _, m := range s.processManager.GetRunningModels() {
			usedVRAM += m.VRAMUsage
		}

		available, err := s.getAvailableVRAM()
		if err != nil {
			return nil, fmt.Errorf("failed to check VRAM: %v", err)
		}

		totalAvailable := available - usedVRAM
		if totalAvailable < requiredVRAM {
			// 如果强制使用显存，尝试释放
			if cfg.ForceVRAM {
				if err := s.freeVRAM(requiredVRAM - totalAvailable); err != nil {
					return nil, fmt.Errorf("insufficient VRAM (required: %dMB based on model size %dMB, available: %dMB): %v",
						requiredVRAM, modelSizeMB, totalAvailable, err)
				}
			} else {
				// 如果不强制使用显存，返回错误
				return nil, fmt.Errorf("insufficient VRAM (required: %dMB based on model size %dMB, available: %dMB). Use force_vram=true to force start",
					requiredVRAM, modelSizeMB, totalAvailable)
			}
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()

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
		return nil, fmt.Errorf("failed to start model service: %v", err)
	}
	pid := s.processManager.GetPID()

	// 创建并添加模型状态到进程管理器
	status = &model.ModelStatus{
		Running:   true,
		ModelName: cfg.ModelName,
		ModelPath: modelPath,
		Port:      cfg.Config.Port,
		StartTime: time.Now().Format(time.RFC3339),
		ProcessID: pid,
		VRAMUsage: requiredVRAM,
	}
	s.processManager.AddModel(pid, status)

	// 保存模型配置到持久化存储
	if status != nil {
		if err := s.persistentMgr.UpdateModelConfig(cfg.ModelName, cfg, status); err != nil {
			log.Printf("Warning: Failed to save model config: %v", err)
		}
	} else {
		log.Printf("Warning: Cannot save model config - status is nil")
	}

	// Windows平台需要特殊处理进程检测
	if runtime.GOOS == "windows" {
		go func() {
			time.Sleep(5 * time.Second) // 等待进程稳定
			if !s.processManager.IsProcessRunning(pid) {
				log.Printf("Warning: Process %d (model: %s) failed to start", pid, cfg.ModelName)
				s.processManager.RemoveModel(pid)
				// 从持久化存储中移除配置
				if err := s.persistentMgr.RemoveModelConfig(cfg.ModelName); err != nil {
					log.Printf("Warning: Failed to remove model config: %v", err)
				}
			}
		}()
	}

	return status, nil
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

// StopModel 停止指定模型
func (s *ModelService) StopModel(model_name string) (*model.ModelStatus, error) {
	if model_name == "" {
		return nil, fmt.Errorf("model_name parameter is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 直接从进程管理器停止指定名称的模型
	modelStatus, err := s.processManager.StopModel(model_name)
	if err != nil {
		return nil, fmt.Errorf("failed to stop model '%s': %v", model_name, err)
	}

	// 更新持久化配置中的状态
	configs, err := s.persistentMgr.GetModelConfigs()
	if err != nil {
		log.Printf("Warning: Failed to load model configs: %v", err)
	} else if item, exists := configs[model_name]; exists {
		// 创建状态副本并更新
		updatedStatus := item.LastStatus
		updatedStatus.Running = false
		updatedStatus.StopTime = time.Now().Format(time.RFC3339)
		if err := s.persistentMgr.UpdateModelConfig(model_name, item.ModelConfig, &updatedStatus); err != nil {
			log.Printf("Warning: Failed to update model config: %v", err)
		}
	}

	return modelStatus, nil
}

// StopAllModel 停止所有运行中的模型
func (s *ModelService) StopAllModel() ([]*model.ModelStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 获取所有运行中的模型
	runningModels := s.processManager.GetRunningModels()
	if len(runningModels) == 0 {
		return nil, nil
	}

	// 停止每个模型并收集结果
	stoppedModels := make([]*model.ModelStatus, 0, len(runningModels))
	var lastError error

	for _, m := range runningModels {
		_, err := s.processManager.StopModel(m.ModelName)
		if err != nil {
			log.Printf("Failed to stop model '%s': %v", m.ModelName, err)
			lastError = err
			continue
		}
		stoppedModels = append(stoppedModels, m)
	}

	if lastError != nil {
		return stoppedModels, fmt.Errorf("some models failed to stop, last error: %v", lastError)
	}

	return stoppedModels, nil
}

// GetModelStatus 获取模型状态
func (s *ModelService) GetModelStatus(name string) []*model.ModelStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 从进程管理器获取所有运行中的模型
	runningModels := s.processManager.GetRunningModels()

	// 获取持久化配置中的所有模型
	persistentConfigs, err := s.persistentMgr.GetModelConfigs()
	if err != nil {
		log.Printf("Warning: Failed to load persistent configs: %v", err)
		persistentConfigs = make(map[string]config.ModelConfigItem)
	}

	// 合并运行中和持久化的模型状态
	var allModels []*model.ModelStatus

	// 首先添加运行中的模型
	allModels = append(allModels, runningModels...)

	// 添加持久化配置中但未运行的模型
	for modelName, item := range persistentConfigs {
		// 检查是否已经在运行中模型中
		found := false
		for _, m := range runningModels {
			if m.ModelName == modelName {
				found = true
				break
			}
		}

		if !found {
			// 添加持久化状态，确保时间格式一致
			status := &model.ModelStatus{
				ModelName: modelName,
				Running:   item.LastStatus.Running,
				ModelPath: item.ModelConfig.ModelPath,
				Port:      item.ModelConfig.Config.Port,
				ProcessID: item.LastStatus.ProcessID,
				VRAMUsage: item.LastStatus.VRAMUsage,
			}

			// 处理时间字段
			if item.LastStatus.StartTime != "" {
				status.StartTime = item.LastStatus.StartTime
			}
			if item.LastStatus.StopTime != "" {
				status.StopTime = item.LastStatus.StopTime
			}
			allModels = append(allModels, status)
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
