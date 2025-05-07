package service

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"llama-switch/internal/config"
	"llama-switch/internal/model"

	"github.com/google/uuid"
)

// BenchmarkService 基准测试服务
type BenchmarkService struct {
	config         *config.Config
	tasks          map[string]*model.BenchmarkStatus
	processManager *ProcessManager
	mu             sync.RWMutex
}

// NewBenchmarkService 创建新的基准测试服务
func NewBenchmarkService(cfg *config.Config) *BenchmarkService {
	return &BenchmarkService{
		config:         cfg,
		tasks:          make(map[string]*model.BenchmarkStatus),
		processManager: NewProcessManager(),
	}
}

// StartBenchmark 启动基准测试
func (s *BenchmarkService) StartBenchmark(cfg *model.BenchmarkConfig) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 生成任务ID
	taskID := uuid.New().String()

	// 验证模型文件路径
	modelPath := cfg.ModelPath
	if !filepath.IsAbs(modelPath) {
		modelPath = filepath.Join(s.config.ModelsDir, cfg.ModelPath)
	}
	if _, err := filepath.Abs(modelPath); err != nil {
		return "", fmt.Errorf("invalid model path: %v", err)
	}

	// 构建命令行参数
	args := []string{
		"--model", modelPath,
	}

	// 添加配置参数
	if cfg.Config.NPrompt > 0 {
		args = append(args, "--n-prompt", strconv.Itoa(cfg.Config.NPrompt))
	}
	if cfg.Config.NGen > 0 {
		args = append(args, "--n-gen", strconv.Itoa(cfg.Config.NGen))
	}
	if cfg.Config.NDepth > 0 {
		args = append(args, "--n-depth", strconv.Itoa(cfg.Config.NDepth))
	}
	if cfg.Config.BatchSize > 0 {
		args = append(args, "--batch-size", strconv.Itoa(cfg.Config.BatchSize))
	}
	if cfg.Config.UBatchSize > 0 {
		args = append(args, "--ubatch-size", strconv.Itoa(cfg.Config.UBatchSize))
	}
	if cfg.Config.CacheTypeK != "" {
		args = append(args, "--cache-type-k", cfg.Config.CacheTypeK)
	}
	if cfg.Config.CacheTypeV != "" {
		args = append(args, "--cache-type-v", cfg.Config.CacheTypeV)
	}
	if cfg.Config.Threads > 0 {
		args = append(args, "--threads", strconv.Itoa(cfg.Config.Threads))
	}
	if cfg.Config.CPUMask != "" {
		args = append(args, "--cpu-mask", cfg.Config.CPUMask)
	}
	if cfg.Config.CPUStrict > 0 {
		args = append(args, "--cpu-strict", strconv.Itoa(cfg.Config.CPUStrict))
	}
	if cfg.Config.Poll > 0 {
		args = append(args, "--poll", strconv.Itoa(cfg.Config.Poll))
	}
	if cfg.Config.NGPULayers > 0 {
		args = append(args, "--n-gpu-layers", strconv.Itoa(cfg.Config.NGPULayers))
	}
	if cfg.Config.SplitMode != "" {
		args = append(args, "--split-mode", cfg.Config.SplitMode)
	}
	if cfg.Config.MainGPU >= 0 {
		args = append(args, "--main-gpu", strconv.Itoa(cfg.Config.MainGPU))
	}
	if cfg.Config.NoKVOffload > 0 {
		args = append(args, "--no-kv-offload", strconv.Itoa(cfg.Config.NoKVOffload))
	}
	if cfg.Config.FlashAttn > 0 {
		args = append(args, "--flash-attn", strconv.Itoa(cfg.Config.FlashAttn))
	}
	if cfg.Config.Mmap >= 0 {
		args = append(args, "--mmap", strconv.Itoa(cfg.Config.Mmap))
	}
	if cfg.Config.Numa != "" {
		args = append(args, "--numa", cfg.Config.Numa)
	}
	if cfg.Config.Embeddings > 0 {
		args = append(args, "--embeddings", strconv.Itoa(cfg.Config.Embeddings))
	}
	if cfg.Config.TensorSplit != "" {
		args = append(args, "--tensor-split", cfg.Config.TensorSplit)
	}
	if cfg.Config.Repetitions > 0 {
		args = append(args, "--repetitions", strconv.Itoa(cfg.Config.Repetitions))
	}
	if cfg.Config.Priority > 0 {
		args = append(args, "--prio", strconv.Itoa(cfg.Config.Priority))
	}
	if cfg.Config.Delay > 0 {
		args = append(args, "--delay", strconv.Itoa(cfg.Config.Delay))
	}
	if cfg.Config.Output != "" {
		args = append(args, "--output", cfg.Config.Output)
	}
	if cfg.Config.OutputErr != "" {
		args = append(args, "--output-err", cfg.Config.OutputErr)
	}
	if cfg.Config.Verbose > 0 {
		args = append(args, "--verbose")
	}
	if cfg.Config.Progress > 0 {
		args = append(args, "--progress")
	}

	// 打印启动命令
	cmdStr := fmt.Sprintf("%s %s", s.config.LLamaPath.Bench, strings.Join(args, " "))
	log.Printf("Starting benchmark with command:\n%s\n", cmdStr)

	// 创建命令
	cmd := exec.Command(s.config.LLamaPath.Bench, args...)

	// 创建管道用于捕获输出
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	// 创建任务状态
	status := &model.BenchmarkStatus{
		TaskID:    taskID,
		Status:    "running",
		Progress:  0,
		StartTime: time.Now().Format(time.RFC3339),
	}
	s.tasks[taskID] = status

	// 启动命令
	if err := cmd.Start(); err != nil {
		delete(s.tasks, taskID)
		return "", fmt.Errorf("failed to start benchmark: %v", err)
	}

	// 处理输出
	go s.handleBenchmarkOutput(taskID, stdout, stderr)

	// 等待命令完成
	go func() {
		err := cmd.Wait()
		s.mu.Lock()
		if err != nil {
			s.tasks[taskID].Status = "failed"
			s.tasks[taskID].EndTime = time.Now().Format(time.RFC3339)
		}
		s.mu.Unlock()
	}()

	return taskID, nil
}

// GetStatus 获取基准测试状态
func (s *BenchmarkService) GetStatus(taskID string) (*model.BenchmarkStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status, exists := s.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return status, nil
}

// handleBenchmarkOutput 处理基准测试输出
func (s *BenchmarkService) handleBenchmarkOutput(taskID string, stdout, stderr io.ReadCloser) {
	scanner := bufio.NewScanner(stdout)
	results := &model.BenchmarkResults{}

	for scanner.Scan() {
		line := scanner.Text()
		s.mu.Lock()

		// 解析输出并更新结果
		if strings.Contains(line, "tokens per second") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				if tps, err := strconv.ParseFloat(parts[3], 64); err == nil {
					results.TokensPerSecond = tps
				}
			}
		} else if strings.Contains(line, "memory used") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				if mem, err := strconv.ParseInt(parts[2], 10, 64); err == nil {
					results.MemoryUsed = mem
				}
			}
		}

		// 更新进度（这里是一个简化的进度计算，实际应该根据llama-bench的具体输出格式调整）
		s.tasks[taskID].Progress += 1.0
		if s.tasks[taskID].Progress > 100 {
			s.tasks[taskID].Progress = 100
		}

		s.mu.Unlock()
	}

	// 处理stderr
	errScanner := bufio.NewScanner(stderr)
	var errOutput strings.Builder
	for errScanner.Scan() {
		errOutput.WriteString(errScanner.Text() + "\n")
	}

	s.mu.Lock()
	if errOutput.Len() > 0 {
		s.tasks[taskID].Status = "failed"
	} else {
		s.tasks[taskID].Status = "completed"
		s.tasks[taskID].Results = results
	}
	s.tasks[taskID].EndTime = time.Now().Format(time.RFC3339)
	s.mu.Unlock()
}

// ValidateBenchmarkConfig 验证基准测试配置
func (s *BenchmarkService) ValidateBenchmarkConfig(cfg *model.BenchmarkConfig) error {
	if cfg.ModelPath == "" {
		return fmt.Errorf("model path is required")
	}

	c := cfg.Config

	// 验证必需的参数
	if c.Threads < 0 {
		return fmt.Errorf("invalid threads number: %d", c.Threads)
	}

	// 验证可选参数的有效范围
	if c.NPrompt < 0 {
		return fmt.Errorf("invalid n-prompt value: %d", c.NPrompt)
	}
	if c.NGen < 0 {
		return fmt.Errorf("invalid n-gen value: %d", c.NGen)
	}
	if c.NDepth < 0 {
		return fmt.Errorf("invalid n-depth value: %d", c.NDepth)
	}
	if c.BatchSize < 0 {
		return fmt.Errorf("invalid batch-size value: %d", c.BatchSize)
	}
	if c.UBatchSize < 0 {
		return fmt.Errorf("invalid ubatch-size value: %d", c.UBatchSize)
	}

	// 验证枚举值
	if c.CacheTypeK != "" && c.CacheTypeK != "f16" && c.CacheTypeK != "f32" {
		return fmt.Errorf("invalid cache-type-k value: %s", c.CacheTypeK)
	}
	if c.CacheTypeV != "" && c.CacheTypeV != "f16" && c.CacheTypeV != "f32" {
		return fmt.Errorf("invalid cache-type-v value: %s", c.CacheTypeV)
	}

	// 验证范围值
	if c.Poll < 0 || c.Poll > 100 {
		return fmt.Errorf("invalid poll value: %d (should be between 0 and 100)", c.Poll)
	}

	// 验证模式值
	if c.SplitMode != "" && c.SplitMode != "none" && c.SplitMode != "layer" && c.SplitMode != "row" {
		return fmt.Errorf("invalid split-mode value: %s", c.SplitMode)
	}

	// 验证NUMA策略
	if c.Numa != "" && c.Numa != "distribute" && c.Numa != "isolate" && c.Numa != "numactl" {
		return fmt.Errorf("invalid numa value: %s", c.Numa)
	}

	// 验证二进制标志
	if c.CPUStrict != 0 && c.CPUStrict != 1 {
		return fmt.Errorf("invalid cpu-strict value: %d (should be 0 or 1)", c.CPUStrict)
	}
	if c.NoKVOffload != 0 && c.NoKVOffload != 1 {
		return fmt.Errorf("invalid no-kv-offload value: %d (should be 0 or 1)", c.NoKVOffload)
	}
	if c.FlashAttn != 0 && c.FlashAttn != 1 {
		return fmt.Errorf("invalid flash-attn value: %d (should be 0 or 1)", c.FlashAttn)
	}
	if c.Mmap != 0 && c.Mmap != 1 {
		return fmt.Errorf("invalid mmap value: %d (should be 0 or 1)", c.Mmap)
	}
	if c.Embeddings != 0 && c.Embeddings != 1 {
		return fmt.Errorf("invalid embeddings value: %d (should be 0 or 1)", c.Embeddings)
	}

	// 验证优先级
	if c.Priority < 0 || c.Priority > 3 {
		return fmt.Errorf("invalid priority value: %d (should be between 0 and 3)", c.Priority)
	}

	// 验证延迟
	if c.Delay < 0 {
		return fmt.Errorf("invalid delay value: %d (should be >= 0)", c.Delay)
	}

	// 验证输出格式
	validOutputFormats := map[string]bool{
		"csv": true, "json": true, "jsonl": true, "md": true, "sql": true,
	}
	if c.Output != "" && !validOutputFormats[c.Output] {
		return fmt.Errorf("invalid output format: %s", c.Output)
	}
	if c.OutputErr != "" && !validOutputFormats[c.OutputErr] && c.OutputErr != "none" {
		return fmt.Errorf("invalid output-err format: %s", c.OutputErr)
	}

	return nil
}
