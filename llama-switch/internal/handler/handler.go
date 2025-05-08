package handler

import (
	"encoding/json"
	"net/http"

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

	if err := h.ModelService.StartModel(&cfg); err != nil {
		h.respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondWithJSON(w, http.StatusOK, model.NewAPIResponse(
		true,
		"Model switched successfully",
		h.ModelService.GetStatus(),
		"",
	))
}

// StopModel 停止模型处理器
func (h *Handler) StopModel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if err := h.ModelService.StopModel(); err != nil {
		h.respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondWithJSON(w, http.StatusOK, model.NewAPIResponse(
		true,
		"Model stopped successfully",
		nil,
		"",
	))
}

// GetModelStatus 获取模型状态处理器
func (h *Handler) GetModelStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	status := h.ModelService.GetStatus()
	h.respondWithJSON(w, http.StatusOK, model.NewAPIResponse(
		true,
		"Model status retrieved successfully",
		status,
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
