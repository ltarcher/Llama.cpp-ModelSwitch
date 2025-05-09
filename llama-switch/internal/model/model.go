package model

import (
	"context"
)

// ModelConfig 模型服务配置
type ModelConfig struct {
	ModelPath string `json:"model_path"` // 模型文件路径
	ModelName string `json:"model_name"` // 模型名称标识
	ForceVRAM bool   `json:"force_vram"` // 是否强制使用显存
	Config    struct {
		// 服务器配置
		Host    string `json:"host"`    // 监听地址
		Port    int    `json:"port"`    // 服务端口
		Timeout int    `json:"timeout"` // 超时时间（秒）

		// 系统资源配置
		Threads      int    `json:"threads"`       // 生成时的线程数
		ThreadsBatch int    `json:"threads_batch"` // 批处理时的线程数
		CPUMask      string `json:"cpu_mask"`      // CPU亲和性掩码
		CPURange     string `json:"cpu_range"`     // CPU范围
		CPUStrict    int    `json:"cpu_strict"`    // CPU严格模式
		Priority     int    `json:"priority"`      // 进程优先级
		Poll         int    `json:"poll"`          // 轮询级别

		// 模型参数
		CtxSize    int `json:"ctx_size"`    // 上下文大小
		BatchSize  int `json:"batch_size"`  // 批处理大小
		UBatchSize int `json:"ubatch_size"` // 微批处理大小
		NPredict   int `json:"n_predict"`   // 预测token数量
		Keep       int `json:"keep"`        // 保留初始提示的token数

		// GPU相关配置
		NGPULayers  int    `json:"n_gpu_layers"` // GPU层数
		SplitMode   string `json:"split_mode"`   // GPU分割模式
		TensorSplit string `json:"tensor_split"` // 张量分割比例
		MainGPU     int    `json:"main_gpu"`     // 主GPU
		Device      string `json:"device"`       // 设备列表

		// 内存管理
		Mlock       bool   `json:"mlock"`         // 锁定内存
		NoMMap      bool   `json:"no_mmap"`       // 禁用内存映射
		Numa        string `json:"numa"`          // NUMA策略
		NoKVOffload bool   `json:"no_kv_offload"` // 禁用KV卸载

		// 缓存配置
		CacheTypeK  string  `json:"cache_type_k"` // K缓存类型
		CacheTypeV  string  `json:"cache_type_v"` // V缓存类型
		DefragThold float64 `json:"defrag_thold"` // 碎片整理阈值

		// 性能优化
		FlashAttn   bool `json:"flash_attn"` // 启用Flash Attention
		NoPerfTimer bool `json:"no_perf"`    // 禁用性能计时器

		// RoPE配置
		RopeScaling   string  `json:"rope_scaling"`    // RoPE缩放方法
		RopeScale     float64 `json:"rope_scale"`      // RoPE上下文缩放因子
		RopeFreqBase  float64 `json:"rope_freq_base"`  // RoPE基础频率
		RopeFreqScale float64 `json:"rope_freq_scale"` // RoPE频率缩放因子

		// YaRN配置
		YarnOrigCtx    int     `json:"yarn_orig_ctx"`    // 原始上下文大小
		YarnExtFactor  float64 `json:"yarn_ext_factor"`  // 外推混合因子
		YarnAttnFactor float64 `json:"yarn_attn_factor"` // 注意力缩放因子
		YarnBetaSlow   float64 `json:"yarn_beta_slow"`   // 高校正维度
		YarnBetaFast   float64 `json:"yarn_beta_fast"`   // 低校正维度

		// 其他功能
		MMProj     bool   `json:"mmproj"`      // 启用MMProj
		Verbose    bool   `json:"verbose"`     // 详细日志
		LogFile    string `json:"log_file"`    // 日志文件
		StaticPath string `json:"static_path"` // 静态文件路径
		APIKey     string `json:"api_key"`     // API密钥
		SSLKey     string `json:"ssl_key"`     // SSL私钥文件
		SSLCert    string `json:"ssl_cert"`    // SSL证书文件
	} `json:"config"`
}

// BenchmarkConfig 基准测试配置
type BenchmarkConfig struct {
	ModelPath string `json:"model_path"` // 模型文件路径
	Config    struct {
		NPrompt     int    `json:"n_prompt"`      // 提示token数量
		NGen        int    `json:"n_gen"`         // 生成token数量
		NDepth      int    `json:"n_depth"`       // 深度
		BatchSize   int    `json:"batch_size"`    // 批处理大小
		UBatchSize  int    `json:"ubatch_size"`   // 微批处理大小
		CacheTypeK  string `json:"cache_type_k"`  // K缓存类型
		CacheTypeV  string `json:"cache_type_v"`  // V缓存类型
		Threads     int    `json:"threads"`       // 线程数
		CPUMask     string `json:"cpu_mask"`      // CPU掩码
		CPUStrict   int    `json:"cpu_strict"`    // CPU严格模式
		Poll        int    `json:"poll"`          // 轮询间隔
		NGPULayers  int    `json:"n_gpu_layers"`  // GPU层数
		SplitMode   string `json:"split_mode"`    // 分割模式
		MainGPU     int    `json:"main_gpu"`      // 主GPU
		NoKVOffload int    `json:"no_kv_offload"` // 禁用KV卸载
		FlashAttn   int    `json:"flash_attn"`    // 闪现注意力
		Mmap        int    `json:"mmap"`          // 内存映射
		Numa        string `json:"numa"`          // NUMA策略
		Embeddings  int    `json:"embeddings"`    // 嵌入模式
		TensorSplit string `json:"tensor_split"`  // 张量分割
		Repetitions int    `json:"repetitions"`   // 重复次数
		Priority    int    `json:"priority"`      // 优先级
		Delay       int    `json:"delay"`         // 延迟（秒）
		Output      string `json:"output"`        // 输出格式
		OutputErr   string `json:"output_err"`    // 错误输出格式
		Verbose     int    `json:"verbose"`       // 详细模式
		Progress    int    `json:"progress"`      // 显示进度
	} `json:"config"`
}

// ModelStatus 模型服务状态
type ModelStatus struct {
	Running   bool   `json:"running"`    // 是否正在运行
	ModelName string `json:"model_name"` // 模型名称标识
	ModelPath string `json:"model_path"` // 当前运行的模型路径
	Port      int    `json:"port"`       // 当前服务端口
	StartTime string `json:"start_time"` // 服务启动时间
	StopTime  string `json:"stop_time"`  // 服务停止时间
	ProcessID int    `json:"process_id"` // 进程ID
	VRAMUsage int    `json:"vram_usage"` // 显存使用量(MB)
}

// ModelStopRequest 停止模型请求
type ModelStopRequest struct {
	ModelName string `json:"model_name"` // 模型名称标识
}

// BenchmarkStatus 基准测试状态
type BenchmarkStatus struct {
	TaskID     string              `json:"task_id"`               // 任务ID
	Status     string              `json:"status"`                // 任务状态：pending/running/completed/failed/cancelled
	Progress   float64             `json:"progress"`              // 进度（0-100）
	StartTime  string              `json:"start_time"`            // 开始时间
	EndTime    string              `json:"end_time"`              // 结束时间（如果已完成）
	AllResults []*BenchmarkResults `json:"all_results,omitempty"` // 所有测试结果
	CancelFunc context.CancelFunc  `json:"-"`                     // 取消函数（不序列化）
}

// BenchmarkResults 基准测试结果
type BenchmarkResults struct {
	Model           string  `json:"model"`             // 模型名称
	Size            string  `json:"size"`              // 模型大小
	Params          string  `json:"params"`            // 模型参数
	Backend         string  `json:"backend"`           // 使用的后端
	GPULayers       int     `json:"gpu_layers"`        // GPU层数
	MMap            bool    `json:"mmap"`              // 是否使用内存映射
	TestType        string  `json:"test_type"`         // 测试类型
	TokensPerSecond float64 `json:"tokens_per_second"` // 每秒处理的token数
	Variation       float64 `json:"variation"`         // 性能波动
	TotalTokens     int     `json:"total_tokens"`      // 总处理token数
	TotalTime       float64 `json:"total_time"`        // 总耗时（秒）
	MemoryUsed      int64   `json:"memory_used"`       // 使用的内存（字节）
}

// APIResponse API通用响应结构
type APIResponse struct {
	Success bool        `json:"success"`         // 是否成功
	Data    interface{} `json:"data,omitempty"`  // 响应数据
	Error   string      `json:"error,omitempty"` // 错误信息
	Message string      `json:"message"`         // 响应消息
}

// NewAPIResponse 创建新的API响应
func NewAPIResponse(success bool, message string, data interface{}, err string) *APIResponse {
	return &APIResponse{
		Success: success,
		Message: message,
		Data:    data,
		Error:   err,
	}
}
