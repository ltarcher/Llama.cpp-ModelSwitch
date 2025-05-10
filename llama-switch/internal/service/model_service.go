package service

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
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

// GetModelList 获取所有GGUF模型列表
func (s *ModelService) GetModelList() ([]model.ModelInfo, error) {
	var models []model.ModelInfo

	// 读取模型目录
	entries, err := os.ReadDir(s.config.ModelsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read models directory: %v", err)
	}

	// 遍历目录查找.gguf文件
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// 检查文件扩展名
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".gguf") {
			continue
		}

		// 获取文件信息
		info, err := entry.Info()
		if err != nil {
			log.Printf("Warning: failed to get info for %s: %v", entry.Name(), err)
			continue
		}

		// 构建模型信息
		modelPath := filepath.Join(s.config.ModelsDir, entry.Name())
		models = append(models, model.ModelInfo{
			Name: entry.Name(),
			Path: modelPath,
			Size: info.Size(),
		})
	}

	// 按名称排序
	sort.Slice(models, func(i, j int) bool {
		return models[i].Name < models[j].Name
	})

	return models, nil
}

// freeVRAM 释放足够显存(优先释放大显存模型)
func (s *ModelService) freeVRAM(required int) error {
	// 获取按显存使用排序的模型列表
	models := s.processManager.GetModelsByVRAMUsage()
	if len(models) == 0 {
		return fmt.Errorf("no running models to free VRAM from")
	}

	// 获取初始可用显存
	initialFree, err := s.getTotalAvailableVRAM()
	if err != nil {
		return fmt.Errorf("failed to get initial VRAM: %v", err)
	}

	stoppedModels := make([]string, 0)
	currentFree := initialFree

	for _, m := range models {
		// 获取停止前的可用显存
		beforeStop := currentFree

		// 尝试停止模型进程
		if err := s.processManager.stopProcessByPID(m.ProcessID); err != nil {
			log.Printf("Warning: failed to stop model %s (PID: %d): %v",
				m.ModelName, m.ProcessID, err)
			continue
		}

		// 等待显存释放（通常需要一点时间）
		time.Sleep(1 * time.Second)

		// 获取停止后的可用显存
		afterStop, err := s.getTotalAvailableVRAM()
		if err != nil {
			log.Printf("Warning: failed to get VRAM after stopping model %s: %v",
				m.ModelName, err)
			continue
		}

		// 计算实际释放的显存
		freedByThisModel := afterStop - beforeStop
		currentFree = afterStop

		s.processManager.RemoveModel(m.ProcessID)
		stoppedModels = append(stoppedModels, m.ModelName)

		log.Printf("Stopped model %s, freed %dMB VRAM", m.ModelName, freedByThisModel)

		// 检查是否已释放足够显存
		totalFreed := currentFree - initialFree
		if totalFreed >= required {
			log.Printf("Successfully freed %dMB VRAM by stopping models: %s",
				totalFreed, strings.Join(stoppedModels, ", "))
			return nil
		}
	}

	totalFreed := currentFree - initialFree
	return fmt.Errorf("could only free %dMB of %dMB required VRAM after stopping models: %s",
		totalFreed, required, strings.Join(stoppedModels, ", "))
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
		available, err := s.getTotalAvailableVRAM()
		if err != nil {
			return nil, fmt.Errorf("failed to check VRAM: %v", err)
		}
		log.Printf("Available VRAM: %dMB", available)

		totalAvailable := available
		if totalAvailable < requiredVRAM {
			// 如果强制使用显存，尝试释放
			if cfg.ForceVRAM {
				log.Printf("Insufficient VRAM (required: %dMB based on model size %dMB, available: %dMB), freeing VRAM", requiredVRAM, modelSizeMB, totalAvailable)
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

	// 新增参数 - 通用参数
	if c.Help {
		args = append(args, "--help")
	}
	if c.Version {
		args = append(args, "--version")
	}
	if c.CompletionBash {
		args = append(args, "--completion-bash")
	}
	if c.VerbosePrompt {
		args = append(args, "--verbose-prompt")
	}
	if c.Escape {
		args = append(args, "--escape")
	}
	if c.NoEscape {
		args = append(args, "--no-escape")
	}
	if c.DumpKVCache {
		args = append(args, "--dump-kv-cache")
	}
	if c.CheckTensors {
		args = append(args, "--check-tensors")
	}
	if c.RPC != "" {
		args = append(args, "--rpc", c.RPC)
	}
	if c.Parallel > 0 {
		args = append(args, "--parallel", strconv.Itoa(c.Parallel))
	}
	if c.OverrideTensor != "" {
		args = append(args, "--override-tensor", c.OverrideTensor)
	}
	if c.ListDevices {
		args = append(args, "--list-devices")
	}
	if c.Lora != "" {
		args = append(args, "--lora", c.Lora)
	}
	if c.LoraScaled != "" {
		args = append(args, "--lora-scaled", c.LoraScaled)
	}
	if c.ControlVector != "" {
		args = append(args, "--control-vector", c.ControlVector)
	}
	if c.ControlVectorScaled != "" {
		args = append(args, "--control-vector-scaled", c.ControlVectorScaled)
	}
	if c.ControlVectorLayerRange != "" {
		args = append(args, "--control-vector-layer-range", c.ControlVectorLayerRange)
	}
	if c.ModelUrl != "" {
		args = append(args, "--model-url", c.ModelUrl)
	}
	if c.HfRepo != "" {
		args = append(args, "--hf-repo", c.HfRepo)
	}
	if c.HfRepoDraft != "" {
		args = append(args, "--hf-repo-draft", c.HfRepoDraft)
	}
	if c.HfFile != "" {
		args = append(args, "--hf-file", c.HfFile)
	}
	if c.HfRepoV != "" {
		args = append(args, "--hf-repo-v", c.HfRepoV)
	}
	if c.HfFileV != "" {
		args = append(args, "--hf-file-v", c.HfFileV)
	}
	if c.HfToken != "" {
		args = append(args, "--hf-token", c.HfToken)
	}
	if c.LogDisable {
		args = append(args, "--log-disable")
	}
	if c.LogColors {
		args = append(args, "--log-colors")
	}
	if c.LogVerbose {
		args = append(args, "--log-verbose")
	}
	if c.LogVerbosity > 0 {
		args = append(args, "--log-verbosity", strconv.Itoa(c.LogVerbosity))
	}
	if c.LogPrefix {
		args = append(args, "--log-prefix")
	}
	if c.LogTimestamps {
		args = append(args, "--log-timestamps")
	}
	if c.Samplers != "" {
		args = append(args, "--samplers", c.Samplers)
	}
	if c.Seed > 0 {
		args = append(args, "--seed", strconv.Itoa(c.Seed))
	}
	if c.SamplerSeq != "" {
		args = append(args, "--sampler-seq", c.SamplerSeq)
	}
	if c.IgnoreEOS {
		args = append(args, "--ignore-eos")
	}
	if c.Temp > 0 {
		args = append(args, "--temp", fmt.Sprintf("%.2f", c.Temp))
	}
	if c.TopK > 0 {
		args = append(args, "--top-k", strconv.Itoa(c.TopK))
	}
	if c.TopP > 0 {
		args = append(args, "--top-p", fmt.Sprintf("%.2f", c.TopP))
	}
	if c.MinP > 0 {
		args = append(args, "--min-p", fmt.Sprintf("%.2f", c.MinP))
	}
	if c.XtcProbability > 0 {
		args = append(args, "--xtc-probability", fmt.Sprintf("%.2f", c.XtcProbability))
	}
	if c.XtcThreshold > 0 {
		args = append(args, "--xtc-threshold", fmt.Sprintf("%.2f", c.XtcThreshold))
	}
	if c.Typical > 0 {
		args = append(args, "--typical", fmt.Sprintf("%.2f", c.Typical))
	}
	if c.RepeatLastN > 0 {
		args = append(args, "--repeat-last-n", strconv.Itoa(c.RepeatLastN))
	}
	if c.RepeatPenalty > 0 {
		args = append(args, "--repeat-penalty", fmt.Sprintf("%.2f", c.RepeatPenalty))
	}
	if c.PresencePenalty > 0 {
		args = append(args, "--presence-penalty", fmt.Sprintf("%.2f", c.PresencePenalty))
	}
	if c.FrequencyPenalty > 0 {
		args = append(args, "--frequency-penalty", fmt.Sprintf("%.2f", c.FrequencyPenalty))
	}
	if c.DryMultiplier > 0 {
		args = append(args, "--dry-multiplier", fmt.Sprintf("%.2f", c.DryMultiplier))
	}
	if c.DryBase > 0 {
		args = append(args, "--dry-base", fmt.Sprintf("%.2f", c.DryBase))
	}
	if c.DryAllowedLength > 0 {
		args = append(args, "--dry-allowed-length", strconv.Itoa(c.DryAllowedLength))
	}
	if c.DryPenaltyLastN > 0 {
		args = append(args, "--dry-penalty-last-n", strconv.Itoa(c.DryPenaltyLastN))
	}
	if c.DrySequenceBreaker != "" {
		args = append(args, "--dry-sequence-breaker", c.DrySequenceBreaker)
	}
	if c.DynatempRange > 0 {
		args = append(args, "--dynatemp-range", fmt.Sprintf("%.2f", c.DynatempRange))
	}
	if c.DynatempExp > 0 {
		args = append(args, "--dynatemp-exp", fmt.Sprintf("%.2f", c.DynatempExp))
	}
	if c.Mirostat > 0 {
		args = append(args, "--mirostat", strconv.Itoa(c.Mirostat))
	}
	if c.MirostatLR > 0 {
		args = append(args, "--mirostat-lr", fmt.Sprintf("%.2f", c.MirostatLR))
	}
	if c.MirostatEnt > 0 {
		args = append(args, "--mirostat-ent", fmt.Sprintf("%.2f", c.MirostatEnt))
	}
	if c.LogitBias != "" {
		args = append(args, "--logit-bias", c.LogitBias)
	}
	if c.Grammar != "" {
		args = append(args, "--grammar", c.Grammar)
	}
	if c.GrammarFile != "" {
		args = append(args, "--grammar-file", c.GrammarFile)
	}
	if c.JsonSchema != "" {
		args = append(args, "--json-schema", c.JsonSchema)
	}
	if c.JsonSchemaFile != "" {
		args = append(args, "--json-schema-file", c.JsonSchemaFile)
	}
	if c.NoContextShift {
		args = append(args, "--no-context-shift")
	}
	if c.Special {
		args = append(args, "--special")
	}
	if c.NoWarmup {
		args = append(args, "--no-warmup")
	}
	if c.SpmInfill {
		args = append(args, "--spm-infill")
	}
	if c.Pooling != "" {
		args = append(args, "--pooling", c.Pooling)
	}
	if c.ContBatching {
		args = append(args, "--cont-batching")
	}
	if c.NoContBatching {
		args = append(args, "--no-cont-batching")
	}
	if c.Alias != "" {
		args = append(args, "--alias", c.Alias)
	}
	if c.NoWebui {
		args = append(args, "--no-webui")
	}
	if c.Embedding {
		args = append(args, "--embedding")
	}
	if c.Reranking {
		args = append(args, "--reranking")
	}
	if c.ApiKeyFile != "" {
		args = append(args, "--api-key-file", c.ApiKeyFile)
	}
	if c.ThreadsHttp > 0 {
		args = append(args, "--threads-http", strconv.Itoa(c.ThreadsHttp))
	}
	if c.CacheReuse > 0 {
		args = append(args, "--cache-reuse", strconv.Itoa(c.CacheReuse))
	}
	if c.Metrics {
		args = append(args, "--metrics")
	}
	if c.Slots {
		args = append(args, "--slots")
	}
	if c.Props {
		args = append(args, "--props")
	}
	if c.NoSlots {
		args = append(args, "--no-slots")
	}
	if c.SlotSavePath != "" {
		args = append(args, "--slot-save-path", c.SlotSavePath)
	}
	if c.Jinja {
		args = append(args, "--jinja")
	}
	if c.ReasoningFormat != "" {
		args = append(args, "--reasoning-format", c.ReasoningFormat)
	}
	if c.ChatTemplate != "" {
		args = append(args, "--chat-template", c.ChatTemplate)
	}
	if c.ChatTemplateFile != "" {
		args = append(args, "--chat-template-file", c.ChatTemplateFile)
	}
	if c.SlotPromptSimilarity > 0 {
		args = append(args, "--slot-prompt-similarity", fmt.Sprintf("%.2f", c.SlotPromptSimilarity))
	}
	if c.LoraInitWithoutApply {
		args = append(args, "--lora-init-without-apply")
	}
	if c.DraftMax > 0 {
		args = append(args, "--draft-max", strconv.Itoa(c.DraftMax))
	}
	if c.DraftMin > 0 {
		args = append(args, "--draft-min", strconv.Itoa(c.DraftMin))
	}
	if c.DraftPMin > 0 {
		args = append(args, "--draft-p-min", fmt.Sprintf("%.2f", c.DraftPMin))
	}
	if c.CtxSizeDraft > 0 {
		args = append(args, "--ctx-size-draft", strconv.Itoa(c.CtxSizeDraft))
	}
	if c.DeviceDraft != "" {
		args = append(args, "--device-draft", c.DeviceDraft)
	}
	if c.NGPULayersDraft > 0 {
		args = append(args, "--n-gpu-layers-draft", strconv.Itoa(c.NGPULayersDraft))
	}
	if c.ModelDraft != "" {
		args = append(args, "--model-draft", c.ModelDraft)
	}
	if c.ModelVocoder != "" {
		args = append(args, "--model-vocoder", c.ModelVocoder)
	}
	if c.TtsUseGuideTokens {
		args = append(args, "--tts-use-guide-tokens")
	}
	if c.EmbdBgeSmallEnDefault {
		args = append(args, "--embd-bge-small-en-default")
	}
	if c.EmbdE5SmallEnDefault {
		args = append(args, "--embd-e5-small-en-default")
	}
	if c.EmbdGteSmallDefault {
		args = append(args, "--embd-gte-small-default")
	}
	if c.FimQwen15bDefault {
		args = append(args, "--fim-qwen-1-5b-default")
	}
	if c.FimQwen3bDefault {
		args = append(args, "--fim-qwen-3b-default")
	}
	if c.FimQwen7bDefault {
		args = append(args, "--fim-qwen-7b-default")
	}
	if c.FimQwen7bSpec {
		args = append(args, "--fim-qwen-7b-spec")
	}
	if c.FimQwen14bSpec {
		args = append(args, "--fim-qwen-14b-spec")
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

// getAvailableVRAM 获取当前可用显存(MB)，返回每个GPU的可用显存
func (s *ModelService) getAvailableVRAM() ([]int, error) {
	cmd := exec.Command("nvidia-smi", "--query-gpu=memory.free", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to query GPU memory: %v", err)
	}

	// 解析输出，获取所有GPU的可用显存
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("no GPU memory information available")
	}

	var freeMemory []int
	for _, line := range lines {
		freeMB, err := strconv.Atoi(strings.TrimSpace(line))
		if err != nil {
			return nil, fmt.Errorf("failed to parse GPU memory: %v", err)
		}
		freeMemory = append(freeMemory, freeMB)
	}

	return freeMemory, nil
}

// getTotalAvailableVRAM 获取所有GPU的总可用显存(MB)
func (s *ModelService) getTotalAvailableVRAM() (int, error) {
	freeMemory, err := s.getAvailableVRAM()
	if err != nil {
		return 0, err
	}

	total := 0
	for _, mem := range freeMemory {
		total += mem
	}
	return total, nil
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

	// 验证新增参数
	if c.Parallel < 0 {
		return fmt.Errorf("invalid parallel value: %d", c.Parallel)
	}
	if c.Temp < 0 {
		return fmt.Errorf("invalid temperature value: %.2f", c.Temp)
	}
	if c.TopK < 0 {
		return fmt.Errorf("invalid top-k value: %d", c.TopK)
	}
	if c.TopP < 0 || c.TopP > 1 {
		return fmt.Errorf("invalid top-p value: %.2f (should be between 0 and 1)", c.TopP)
	}
	if c.MinP < 0 || c.MinP > 1 {
		return fmt.Errorf("invalid min-p value: %.2f (should be between 0 and 1)", c.MinP)
	}
	if c.XtcProbability < 0 || c.XtcProbability > 1 {
		return fmt.Errorf("invalid xtc probability: %.2f (should be between 0 and 1)", c.XtcProbability)
	}
	if c.XtcThreshold < 0 {
		return fmt.Errorf("invalid xtc threshold: %.2f", c.XtcThreshold)
	}
	if c.Typical < 0 || c.Typical > 1 {
		return fmt.Errorf("invalid typical value: %.2f (should be between 0 and 1)", c.Typical)
	}
	if c.RepeatLastN < 0 {
		return fmt.Errorf("invalid repeat last n value: %d", c.RepeatLastN)
	}
	if c.RepeatPenalty < 0 {
		return fmt.Errorf("invalid repeat penalty: %.2f", c.RepeatPenalty)
	}
	if c.PresencePenalty < 0 {
		return fmt.Errorf("invalid presence penalty: %.2f", c.PresencePenalty)
	}
	if c.FrequencyPenalty < 0 {
		return fmt.Errorf("invalid frequency penalty: %.2f", c.FrequencyPenalty)
	}
	if c.DryMultiplier < 0 {
		return fmt.Errorf("invalid dry multiplier: %.2f", c.DryMultiplier)
	}
	if c.DryBase < 0 {
		return fmt.Errorf("invalid dry base: %.2f", c.DryBase)
	}
	if c.DryAllowedLength < 0 {
		return fmt.Errorf("invalid dry allowed length: %d", c.DryAllowedLength)
	}
	if c.DryPenaltyLastN < 0 {
		return fmt.Errorf("invalid dry penalty last n: %d", c.DryPenaltyLastN)
	}
	if c.DynatempRange < 0 {
		return fmt.Errorf("invalid dynatemp range: %.2f", c.DynatempRange)
	}
	if c.DynatempExp < 0 {
		return fmt.Errorf("invalid dynatemp exp: %.2f", c.DynatempExp)
	}
	if c.Mirostat < 0 || c.Mirostat > 2 {
		return fmt.Errorf("invalid mirostat value: %d (should be 0, 1 or 2)", c.Mirostat)
	}
	if c.MirostatLR < 0 {
		return fmt.Errorf("invalid mirostat learning rate: %.2f", c.MirostatLR)
	}
	if c.MirostatEnt < 0 {
		return fmt.Errorf("invalid mirostat entropy: %.2f", c.MirostatEnt)
	}
	if c.ThreadsHttp < 0 {
		return fmt.Errorf("invalid http threads: %d", c.ThreadsHttp)
	}
	if c.CacheReuse < 0 {
		return fmt.Errorf("invalid cache reuse value: %d", c.CacheReuse)
	}
	if c.SlotPromptSimilarity < 0 || c.SlotPromptSimilarity > 1 {
		return fmt.Errorf("invalid slot prompt similarity: %.2f (should be between 0 and 1)", c.SlotPromptSimilarity)
	}
	if c.DraftMax < 0 {
		return fmt.Errorf("invalid draft max: %d", c.DraftMax)
	}
	if c.DraftMin < 0 {
		return fmt.Errorf("invalid draft min: %d", c.DraftMin)
	}
	if c.DraftPMin < 0 || c.DraftPMin > 1 {
		return fmt.Errorf("invalid draft p min: %.2f (should be between 0 and 1)", c.DraftPMin)
	}
	if c.CtxSizeDraft < 0 {
		return fmt.Errorf("invalid draft context size: %d", c.CtxSizeDraft)
	}
	if c.NGPULayersDraft < 0 {
		return fmt.Errorf("invalid draft GPU layers: %d", c.NGPULayersDraft)
	}

	// 验证文件路径参数
	if c.Lora != "" && !filepath.IsAbs(c.Lora) {
		return fmt.Errorf("lora adapter path must be absolute: %s", c.Lora)
	}
	if c.LoraScaled != "" && !filepath.IsAbs(c.LoraScaled) {
		return fmt.Errorf("scaled lora adapter path must be absolute: %s", c.LoraScaled)
	}
	if c.ControlVector != "" && !filepath.IsAbs(c.ControlVector) {
		return fmt.Errorf("control vector path must be absolute: %s", c.ControlVector)
	}
	if c.ControlVectorScaled != "" && !filepath.IsAbs(c.ControlVectorScaled) {
		return fmt.Errorf("scaled control vector path must be absolute: %s", c.ControlVectorScaled)
	}
	if c.GrammarFile != "" && !filepath.IsAbs(c.GrammarFile) {
		return fmt.Errorf("grammar file path must be absolute: %s", c.GrammarFile)
	}
	if c.JsonSchemaFile != "" && !filepath.IsAbs(c.JsonSchemaFile) {
		return fmt.Errorf("JSON schema file path must be absolute: %s", c.JsonSchemaFile)
	}
	if c.ApiKeyFile != "" && !filepath.IsAbs(c.ApiKeyFile) {
		return fmt.Errorf("API key file path must be absolute: %s", c.ApiKeyFile)
	}
	if c.SlotSavePath != "" && !filepath.IsAbs(c.SlotSavePath) {
		return fmt.Errorf("slot save path must be absolute: %s", c.SlotSavePath)
	}
	if c.ChatTemplateFile != "" && !filepath.IsAbs(c.ChatTemplateFile) {
		return fmt.Errorf("chat template file path must be absolute: %s", c.ChatTemplateFile)
	}
	if c.ModelDraft != "" && !filepath.IsAbs(c.ModelDraft) {
		return fmt.Errorf("draft model path must be absolute: %s", c.ModelDraft)
	}
	if c.ModelVocoder != "" && !filepath.IsAbs(c.ModelVocoder) {
		return fmt.Errorf("vocoder model path must be absolute: %s", c.ModelVocoder)
	}

	return nil
}
