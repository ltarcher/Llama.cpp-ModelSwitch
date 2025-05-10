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

		// 新增参数 - 通用参数
		Help           bool `json:"help"`            // 显示帮助信息并退出
		Version        bool `json:"version"`         // 显示版本和构建信息
		CompletionBash bool `json:"completion_bash"` // 打印可源化的bash完成脚本
		VerbosePrompt  bool `json:"verbose_prompt"`  // 在生成前打印详细提示
		Escape         bool `json:"escape"`          // 处理转义序列(\n, \r, \t, \', \", \\)
		NoEscape       bool `json:"no_escape"`       // 不处理转义序列
		DumpKVCache    bool `json:"dump_kv_cache"`   // 详细打印KV缓存
		CheckTensors   bool `json:"check_tensors"`   // 检查模型张量数据是否有无效值

		// RPC相关
		RPC string `json:"rpc"` // 逗号分隔的RPC服务器列表

		// 并行处理
		Parallel int `json:"parallel"` // 并行解码的序列数量

		// 模型加载和内存管理
		OverrideTensor string `json:"override_tensor"` // 覆盖张量缓冲区类型

		// 设备和GPU相关
		ListDevices bool `json:"list_devices"` // 打印可用设备列表并退出

		// 模型加载选项
		Lora                    string `json:"lora"`                       // LoRA适配器路径
		LoraScaled              string `json:"lora_scaled"`                // 带用户定义缩放的LoRA适配器路径
		ControlVector           string `json:"control_vector"`             // 添加控制向量
		ControlVectorScaled     string `json:"control_vector_scaled"`      // 添加带用户定义缩放的控制向量
		ControlVectorLayerRange string `json:"control_vector_layer_range"` // 应用控制向量的层范围

		// 模型源选项
		ModelUrl    string `json:"model_url"`     // 模型下载URL
		HfRepo      string `json:"hf_repo"`       // Hugging Face模型仓库
		HfRepoDraft string `json:"hf_repo_draft"` // 草稿模型的Hugging Face仓库
		HfFile      string `json:"hf_file"`       // Hugging Face模型文件
		HfRepoV     string `json:"hf_repo_v"`     // 声码器模型的Hugging Face仓库
		HfFileV     string `json:"hf_file_v"`     // 声码器模型的Hugging Face模型文件
		HfToken     string `json:"hf_token"`      // Hugging Face访问令牌

		// 日志选项
		LogDisable    bool `json:"log_disable"`    // 禁用日志
		LogColors     bool `json:"log_colors"`     // 启用彩色日志
		LogVerbose    bool `json:"log_verbose"`    // 设置详细级别为无限
		LogVerbosity  int  `json:"log_verbosity"`  // 设置详细阈值
		LogPrefix     bool `json:"log_prefix"`     // 在日志消息中启用前缀
		LogTimestamps bool `json:"log_timestamps"` // 在日志消息中启用时间戳

		// 采样参数
		Samplers           string  `json:"samplers"`             // 用于生成的采样器
		Seed               int     `json:"seed"`                 // RNG种子
		SamplerSeq         string  `json:"sampler_seq"`          // 采样器的简化序列
		IgnoreEOS          bool    `json:"ignore_eos"`           // 忽略流结束标记并继续生成
		Temp               float64 `json:"temp"`                 // 温度
		TopK               int     `json:"top_k"`                // top-k采样
		TopP               float64 `json:"top_p"`                // top-p采样
		MinP               float64 `json:"min_p"`                // min-p采样
		XtcProbability     float64 `json:"xtc_probability"`      // xtc概率
		XtcThreshold       float64 `json:"xtc_threshold"`        // xtc阈值
		Typical            float64 `json:"typical"`              // 局部典型采样
		RepeatLastN        int     `json:"repeat_last_n"`        // 考虑惩罚的最后n个标记
		RepeatPenalty      float64 `json:"repeat_penalty"`       // 惩罚重复标记序列
		PresencePenalty    float64 `json:"presence_penalty"`     // 重复alpha存在惩罚
		FrequencyPenalty   float64 `json:"frequency_penalty"`    // 重复alpha频率惩罚
		DryMultiplier      float64 `json:"dry_multiplier"`       // DRY采样乘数
		DryBase            float64 `json:"dry_base"`             // DRY采样基值
		DryAllowedLength   int     `json:"dry_allowed_length"`   // DRY采样允许长度
		DryPenaltyLastN    int     `json:"dry_penalty_last_n"`   // DRY对最后n个标记的惩罚
		DrySequenceBreaker string  `json:"dry_sequence_breaker"` // DRY采样的序列断路器
		DynatempRange      float64 `json:"dynatemp_range"`       // 动态温度范围
		DynatempExp        float64 `json:"dynatemp_exp"`         // 动态温度指数
		Mirostat           int     `json:"mirostat"`             // Mirostat采样
		MirostatLR         float64 `json:"mirostat_lr"`          // Mirostat学习率
		MirostatEnt        float64 `json:"mirostat_ent"`         // Mirostat目标熵
		LogitBias          string  `json:"logit_bias"`           // 修改标记出现在完成中的可能性
		Grammar            string  `json:"grammar"`              // BNF-like语法约束生成
		GrammarFile        string  `json:"grammar_file"`         // 从文件读取语法
		JsonSchema         string  `json:"json_schema"`          // JSON模式约束生成
		JsonSchemaFile     string  `json:"json_schema_file"`     // 包含JSON模式的文件

		// 示例特定参数
		NoContextShift        bool    `json:"no_context_shift"`          // 禁用无限文本生成的上下文移位
		Special               bool    `json:"special"`                   // 启用特殊标记输出
		NoWarmup              bool    `json:"no_warmup"`                 // 跳过用空运行预热模型
		SpmInfill             bool    `json:"spm_infill"`                // 使用后缀/前缀/中间模式进行填充
		Pooling               string  `json:"pooling"`                   // 嵌入的池化类型
		ContBatching          bool    `json:"cont_batching"`             // 启用连续批处理
		NoContBatching        bool    `json:"no_cont_batching"`          // 禁用连续批处理
		Alias                 string  `json:"alias"`                     // 设置模型名称的别名
		NoWebui               bool    `json:"no_webui"`                  // 禁用Web UI
		Embedding             bool    `json:"embedding"`                 // 仅支持嵌入用例
		Reranking             bool    `json:"reranking"`                 // 在服务器上启用重新排名端点
		ApiKeyFile            string  `json:"api_key_file"`              // 包含API密钥的文件路径
		ThreadsHttp           int     `json:"threads_http"`              // 用于处理HTTP请求的线程数
		CacheReuse            int     `json:"cache_reuse"`               // 尝试通过KV移位从缓存中重用的最小块大小
		Metrics               bool    `json:"metrics"`                   // 启用Prometheus兼容的指标端点
		Slots                 bool    `json:"slots"`                     // 启用插槽监控端点
		Props                 bool    `json:"props"`                     // 启用通过POST /props更改全局属性
		NoSlots               bool    `json:"no_slots"`                  // 禁用插槽监控端点
		SlotSavePath          string  `json:"slot_save_path"`            // 保存插槽kv缓存的路径
		Jinja                 bool    `json:"jinja"`                     // 使用jinja模板进行聊天
		ReasoningFormat       string  `json:"reasoning_format"`          // 推理格式
		ChatTemplate          string  `json:"chat_template"`             // 设置自定义jinja聊天模板
		ChatTemplateFile      string  `json:"chat_template_file"`        // 设置自定义jinja聊天模板文件
		SlotPromptSimilarity  float64 `json:"slot_prompt_similarity"`    // 请求提示与插槽提示匹配程度
		LoraInitWithoutApply  bool    `json:"lora_init_without_apply"`   // 加载LoRA适配器而不应用它们
		DraftMax              int     `json:"draft_max"`                 // 推测解码的草稿标记数
		DraftMin              int     `json:"draft_min"`                 // 推测解码使用的最小草稿标记数
		DraftPMin             float64 `json:"draft_p_min"`               // 最小推测解码概率
		CtxSizeDraft          int     `json:"ctx_size_draft"`            // 草稿模型的提示上下文大小
		DeviceDraft           string  `json:"device_draft"`              // 用于卸载草稿模型的设备列表
		NGPULayersDraft       int     `json:"n_gpu_layers_draft"`        // 草稿模型在VRAM中存储的层数
		ModelDraft            string  `json:"model_draft"`               // 推测解码的草稿模型
		ModelVocoder          string  `json:"model_vocoder"`             // 音频生成的声码器模型
		TtsUseGuideTokens     bool    `json:"tts_use_guide_tokens"`      // 使用引导标记改善TTS单词回忆
		EmbdBgeSmallEnDefault bool    `json:"embd_bge_small_en_default"` // 使用默认bge-small-en-v1.5模型
		EmbdE5SmallEnDefault  bool    `json:"embd_e5_small_en_default"`  // 使用默认e5-small-v2模型
		EmbdGteSmallDefault   bool    `json:"embd_gte_small_default"`    // 使用默认gte-small模型
		FimQwen15bDefault     bool    `json:"fim_qwen_1_5b_default"`     // 使用默认Qwen 2.5 Coder 1.5B
		FimQwen3bDefault      bool    `json:"fim_qwen_3b_default"`       // 使用默认Qwen 2.5 Coder 3B
		FimQwen7bDefault      bool    `json:"fim_qwen_7b_default"`       // 使用默认Qwen 2.5 Coder 7B
		FimQwen7bSpec         bool    `json:"fim_qwen_7b_spec"`          // 使用Qwen 2.5 Coder 7B + 0.5B草稿
		FimQwen14bSpec        bool    `json:"fim_qwen_14b_spec"`         // 使用Qwen 2.5 Coder 14B + 0.5B草稿
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

// ModelInfo 模型基本信息
type ModelInfo struct {
	Name string `json:"name"` // 模型文件名
	Path string `json:"path"` // 模型完整路径
	Size int64  `json:"size"` // 模型文件大小(字节)
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
