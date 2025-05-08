package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"llama-switch/internal/config"
	"llama-switch/internal/model"
	"llama-switch/internal/service"
)

// Handler HTTP处理器
type Handler struct {
	ModelService     *service.ModelService
	BenchmarkService *service.BenchmarkService
}

// NewHandler 创建新的HTTP处理器
func NewHandler(cfg *config.Config) *Handler {
	return &Handler{
		ModelService:     service.NewModelService(cfg),
		BenchmarkService: service.NewBenchmarkService(cfg),
	}
}

// SwitchModel 切换模型处理器
func (h *Handler) SwitchModel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var cfg model.ModelConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// 验证必要参数
	if cfg.ModelName == "" {
		h.respondWithError(w, http.StatusBadRequest, "Model name is required")
		return
	}

	// 验证ForceVRAM参数
	if cfg.ForceVRAM && cfg.Config.NGPULayers <= 0 {
		h.respondWithError(w, http.StatusBadRequest,
			"ForceVRAM requires NGPULayers > 0")
		return
	}

	if err := h.ModelService.ValidateModelConfig(&cfg); err != nil {
		h.respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 记录请求日志和当前运行模型
	currentModels := h.ModelService.GetModelStatus("")
	log.Printf("Current running models (%d):", len(currentModels))
	for i, m := range currentModels {
		log.Printf("  [%d] %s (PID: %d, VRAM: %dMB)", i+1, m.ModelName, m.ProcessID, m.VRAMUsage)
	}
	log.Printf("Starting model switch: %s (%s)", cfg.ModelName, cfg.ModelPath)

	if _, err := h.ModelService.StartModel(&cfg); err != nil {
		log.Printf("Failed to start model %s: %v", cfg.ModelName, err)
		h.respondWithError(w, http.StatusInternalServerError,
			fmt.Sprintf("Failed to start model: %v", err))
		return
	}

	// 获取并验证模型状态
	statuses := h.ModelService.GetModelStatus(cfg.ModelName)
	if len(statuses) == 0 {
		errMsg := fmt.Sprintf("Model %s failed to start (no status available)", cfg.ModelName)
		log.Println(errMsg)
		h.respondWithError(w, http.StatusInternalServerError, errMsg)
		return
	}

	// 确保模型已正确加载
	if !statuses[0].Running {
		errMsg := fmt.Sprintf("Model %s is not running after start", cfg.ModelName)
		log.Println(errMsg)
		h.respondWithError(w, http.StatusInternalServerError, errMsg)
		return
	}

	log.Printf("Model %s started successfully (PID: %d)", cfg.ModelName, statuses[0].ProcessID)

	h.respondWithJSON(w, http.StatusOK, model.NewAPIResponse(
		true,
		fmt.Sprintf("Model '%s' switched successfully", cfg.ModelName),
		map[string]interface{}{
			"model":     statuses[0],
			"load_time": time.Since(time.Now()).String(),
		},
		"",
	))
}

// StopModel 停止模型处理器
func (h *Handler) StopModel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// 解析请求参数
	query := r.URL.Query()
	modelName := query.Get("model_name")

	if modelName == "" {
		h.respondWithError(w, http.StatusBadRequest, "Model name is required")
		return
	}

	// 获取当前模型状态
	var targetStatus *model.ModelStatus
	if modelName != "" {
		// 检查指定模型是否存在
		statuses := h.ModelService.GetModelStatus(modelName)
		if len(statuses) == 0 {
			h.respondWithError(w, http.StatusNotFound,
				fmt.Sprintf("Model '%s' not found or not running", modelName))
			return
		}
		targetStatus = statuses[0]
	}

	// 获取并记录当前所有运行模型
	currentModels := h.ModelService.GetModelStatus("")
	log.Printf("Current running models before stopping (%d):", len(currentModels))
	for i, m := range currentModels {
		log.Printf("  [%d] Model: %s", i+1, m.ModelName)
		log.Printf("     PID: %d", m.ProcessID)
		log.Printf("     VRAM: %dMB", m.VRAMUsage)
		log.Printf("     StartTime: %s", m.StartTime)
		log.Printf("     Port: %d", m.Port)
	}
	log.Printf("Stopping model: %s", modelName)

	var err error
	var status *model.ModelStatus
	// 按名称停止特定模型
	status, err = h.ModelService.StopModel(modelName)

	if err != nil {
		log.Printf("Failed to stop model %s: %v", modelName, err)
		h.respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 验证模型是否真的已停止
	time.Sleep(100 * time.Millisecond) // 给进程一点时间完全退出
	statuses := h.ModelService.GetModelStatus(modelName)
	if len(statuses) > 0 && statuses[0].Running {
		errMsg := fmt.Sprintf("Model '%s' is still running after stop request", modelName)
		log.Println(errMsg)
		h.respondWithError(w, http.StatusInternalServerError, errMsg)
		return
	}

	msg := "Model stopped successfully"
	if modelName != "" {
		msg = fmt.Sprintf("Model '%s' stopped successfully", modelName)
	}

	// 记录成功日志
	log.Printf("Successfully stopped model: %s", modelName)

	// 构建响应数据
	responseData := map[string]interface{}{
		"stopped_model": status,
		"stop_time":     time.Now().Format(time.RFC3339),
	}

	// 只有在有目标状态时才添加显存信息
	if targetStatus != nil {
		responseData["vram_freed"] = targetStatus.VRAMUsage
	}

	h.respondWithJSON(w, http.StatusOK, model.NewAPIResponse(
		true,
		msg,
		responseData,
		"",
	))
}

// GetModelStatus 获取模型状态处理器
func (h *Handler) GetModelStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// 解析查询参数
	modelName := r.URL.Query().Get("model_name")

	// 获取并记录当前所有运行模型
	currentModels := h.ModelService.GetModelStatus("")
	log.Printf("Current running models (%d):", len(currentModels))
	for i, m := range currentModels {
		log.Printf("  [%d] Model: %s", i+1, m.ModelName)
		log.Printf("     PID: %d", m.ProcessID)
		log.Printf("     VRAM: %dMB", m.VRAMUsage)
		log.Printf("     StartTime: %s", m.StartTime)
		log.Printf("     Port: %d", m.Port)
	}

	if modelName != "" {
		log.Printf("Requesting status for specific model: %s", modelName)
	} else {
		log.Printf("Requesting status for all models")
	}

	// 获取模型状态
	statuses := h.ModelService.GetModelStatus(modelName)
	if len(statuses) == 0 {
		if modelName != "" {
			msg := fmt.Sprintf("Model '%s' not found", modelName)
			log.Println(msg)
			h.respondWithError(w, http.StatusNotFound, msg)
			return
		}
		h.respondWithJSON(w, http.StatusOK, model.NewAPIResponse(
			true,
			"No models running",
			nil,
			"",
		))
		return
	}

	// 收集性能指标
	responseData := make([]map[string]interface{}, 0, len(statuses))
	for _, status := range statuses {
		// TODO: 实现真实的进程资源使用统计
		// 这里使用模拟数据作为示例
		cpuUsage := "15%"   // 模拟CPU使用率
		memUsage := "256MB" // 模拟内存使用

		modelInfo := map[string]interface{}{
			"model": status,
			"performance": map[string]string{
				"cpu_usage":    cpuUsage,
				"memory_usage": memUsage,
				"vram_usage":   fmt.Sprintf("%dMB", status.VRAMUsage),
				"uptime":       time.Since(time.Now()).String(),
			},
			"timestamps": map[string]string{
				"start_time":  status.StartTime,
				"last_update": time.Now().Format(time.RFC3339),
			},
		}
		responseData = append(responseData, modelInfo)
	}

	// 如果是单个模型查询，直接返回单个对象
	var data interface{} = responseData[0]
	if modelName == "" && len(responseData) > 1 {
		data = responseData
	}

	log.Printf("Returning status for %d models", len(statuses))
	h.respondWithJSON(w, http.StatusOK, model.NewAPIResponse(
		true,
		"Model status retrieved successfully",
		data,
		"",
	))
}

// StartBenchmark 启动基准测试处理器
func (h *Handler) StartBenchmark(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var cfg model.BenchmarkConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.BenchmarkService.ValidateBenchmarkConfig(&cfg); err != nil {
		h.respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	taskID, err := h.BenchmarkService.StartBenchmark(&cfg)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondWithJSON(w, http.StatusOK, model.NewAPIResponse(
		true,
		"Benchmark started successfully",
		map[string]string{"task_id": taskID},
		"",
	))
}

// GetBenchmarkStatus 获取基准测试状态处理器
func (h *Handler) GetBenchmarkStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	taskID := r.URL.Query().Get("task_id")
	if taskID == "" {
		h.respondWithError(w, http.StatusBadRequest, "Task ID is required")
		return
	}

	status, err := h.BenchmarkService.GetStatus(taskID)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, err.Error())
		return
	}

	h.respondWithJSON(w, http.StatusOK, model.NewAPIResponse(
		true,
		"Benchmark status retrieved successfully",
		status,
		"",
	))
}

// respondWithError 返回错误响应
func (h *Handler) respondWithError(w http.ResponseWriter, code int, message string) {
	h.respondWithJSON(w, code, model.NewAPIResponse(
		false,
		message,
		nil,
		message,
	))
}

// respondWithJSON 返回JSON响应
func (h *Handler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
