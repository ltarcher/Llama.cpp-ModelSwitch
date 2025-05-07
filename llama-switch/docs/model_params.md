# 模型服务参数说明

本文档详细说明了llama-server支持的各项参数及其使用建议。

## 基本参数

### 服务器配置
- `host`: 监听地址（默认：127.0.0.1）
  - 可以设置为特定IP或"0.0.0.0"以允许远程访问
- `port`: 服务端口（默认：8080）
- `timeout`: 服务超时时间（秒）（默认：600）

### 系统资源配置

#### 线程管理
- `threads`: 生成时的线程数
  - -1表示使用所有可用线程
  - 建议设置为CPU核心数
- `threads_batch`: 批处理时的线程数
  - 默认与threads相同
  - 可以单独设置以优化批处理性能

#### CPU控制
- `cpu_mask`: CPU亲和性掩码（十六进制格式）
- `cpu_range`: CPU范围（格式：lo-hi）
- `cpu_strict`: CPU严格模式（0/1）
- `priority`: 进程优先级（0-3）
  - 0: 普通
  - 1: 中等
  - 2: 高
  - 3: 实时
- `poll`: 轮询级别（0-100）
  - 控制等待工作时的轮询强度

### 模型参数

#### 上下文和批处理
- `ctx_size`: 上下文大小
  - 影响模型能处理的最大文本长度
  - 较大的值需要更多内存
- `batch_size`: 批处理大小（默认：2048）
  - 影响处理效率和内存使用
- `ubatch_size`: 微批处理大小（默认：512）
  - 用于细粒度控制处理
- `n_predict`: 预测token数量
  - -1表示无限制
- `keep`: 保留初始提示的token数
  - 0表示不保留
  - -1表示保留所有

### GPU配置

#### 基本GPU设置
- `n_gpu_layers`: GPU层数
  - 控制有多少层运行在GPU上
  - 99表示所有层
  - 0表示仅CPU模式
- `split_mode`: GPU分割模式
  - none: 仅使用单GPU
  - layer: 按层分割（默认）
  - row: 按行分割
- `tensor_split`: 张量分割比例
  - 格式如"3,1"表示75%/25%分配
- `main_gpu`: 主GPU编号
  - 从0开始计数
- `device`: 设备列表
  - 逗号分隔的设备列表

### 内存管理

#### 内存选项
- `mlock`: 锁定内存（true/false）
  - 防止内存被交换到磁盘
- `no_mmap`: 禁用内存映射（true/false）
  - 可能降低加载速度但减少页面调出
- `numa`: NUMA策略
  - distribute: 分布式分配
  - isolate: 隔离模式
  - numactl: 使用numactl工具
- `no_kv_offload`: 禁用KV卸载（true/false）

### 缓存配置

#### 缓存类型
- `cache_type_k`: K缓存类型
- `cache_type_v`: V缓存类型
  支持的类型：
  - f32: 32位浮点
  - f16: 16位浮点
  - bf16: bfloat16
  - q8_0: 8位量化
  - q4_0/q4_1: 4位量化
  - iq4_nl: 4位非线性量化
  - q5_0/q5_1: 5位量化
- `defrag_thold`: 碎片整理阈值（0-1）

### 性能优化

#### 优化选项
- `flash_attn`: 启用Flash Attention
  - 需要硬件支持
- `no_perf`: 禁用性能计时器

### RoPE配置

#### RoPE参数
- `rope_scaling`: RoPE缩放方法
  - none: 不使用缩放
  - linear: 线性缩放
  - yarn: YaRN缩放
- `rope_scale`: RoPE上下文缩放因子
- `rope_freq_base`: RoPE基础频率
- `rope_freq_scale`: RoPE频率缩放因子

### YaRN配置

#### YaRN参数
- `yarn_orig_ctx`: 原始上下文大小
- `yarn_ext_factor`: 外推混合因子
  - -1.0表示禁用
  - 0.0表示完全插值
- `yarn_attn_factor`: 注意力缩放因子
- `yarn_beta_slow`: 高校正维度
- `yarn_beta_fast`: 低校正维度

### 其他功能

#### 日志和安全
- `verbose`: 详细日志（true/false）
- `log_file`: 日志文件路径
- `static_path`: 静态文件路径
- `api_key`: API密钥
- `ssl_key`: SSL私钥文件路径
- `ssl_cert`: SSL证书文件路径

## 配置示例

### 基本CPU配置
```json
{
    "model_path": "model.gguf",
    "config": {
        "host": "127.0.0.1",
        "port": 8080,
        "threads": 8,
        "ctx_size": 4096,
        "batch_size": 512
    }
}
```

### GPU优化配置
```json
{
    "model_path": "model.gguf",
    "config": {
        "host": "127.0.0.1",
        "port": 8080,
        "threads": 8,
        "n_gpu_layers": 99,
        "flash_attn": true,
        "batch_size": 2048,
        "ctx_size": 4096
    }
}
```

### 多GPU配置
```json
{
    "model_path": "model.gguf",
    "config": {
        "host": "127.0.0.1",
        "port": 8080,
        "threads": 8,
        "n_gpu_layers": 99,
        "split_mode": "layer",
        "tensor_split": "0.5,0.5",
        "main_gpu": 0,
        "flash_attn": true
    }
}
```

### 高性能配置
```json
{
    "model_path": "model.gguf",
    "config": {
        "host": "127.0.0.1",
        "port": 8080,
        "threads": 32,
        "threads_batch": 32,
        "n_gpu_layers": 99,
        "flash_attn": true,
        "batch_size": 2048,
        "ubatch_size": 512,
        "ctx_size": 4096,
        "mlock": true,
        "cache_type_k": "f16",
        "cache_type_v": "f16"
    }
}
```

## 注意事项

1. 参数配置需要根据具体硬件和模型大小调整
2. GPU相关参数只在有GPU可用时生效
3. 某些参数组合可能会相互影响，需要综合考虑
4. 内存相关参数要根据系统可用资源谨慎设置
5. 建议先使用基本配置测试，然后逐步优化参数