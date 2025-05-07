# 基准测试参数说明

本文档详细说明了llama-bench支持的各项参数及其使用建议。

## 基本参数

### 模型和输入控制
- `model_path`: 模型文件路径
- `n_prompt`: 提示token数量（默认：512）
  - 用于测试的提示文本长度
  - 较大的值可以测试模型处理长文本的能力
- `n_gen`: 生成token数量（默认：128）
  - 模型需要生成的token数量
  - 影响测试的总时长

### 性能相关参数

#### 批处理配置
- `batch_size`: 批处理大小（默认：2048）
  - 影响内存使用和处理速度
  - 较大的值可能提高吞吐量，但会增加内存使用
- `ubatch_size`: 微批处理大小（默认：512）
  - 控制内部处理的粒度
  - 需要根据硬件资源调整

#### 缓存配置
- `cache_type_k`: K缓存类型（默认：f16）
  - 可选值：f16/f32
  - f16通常能提供更好的性能/内存平衡
- `cache_type_v`: V缓存类型（默认：f16）
  - 可选值：f16/f32
  - 与K缓存类型相同的考虑

#### 计算资源配置
- `threads`: 线程数（默认：32）
  - 建议设置为CPU核心数
  - 过多的线程可能导致性能下降
- `cpu_mask`: CPU掩码
  - 用于精确控制CPU核心的使用
  - 格式为十六进制值，如"0x0"
- `cpu_strict`: CPU严格模式（0/1）
  - 启用后强制使用指定的CPU核心
- `poll`: 轮询间隔（0-100）
  - 控制内部轮询的频率
  - 较低的值可能提高响应性，但会增加CPU使用率

### GPU相关参数

- `n_gpu_layers`: GPU层数（默认：99）
  - 控制有多少层运行在GPU上
  - 设置为0可以强制CPU模式
- `split_mode`: 分割模式（none/layer/row）
  - 控制模型在多GPU间的分配方式
  - layer: 按层分割
  - row: 按行分割
  - none: 不分割
- `main_gpu`: 主GPU编号
  - 指定主要使用的GPU
  - 从0开始计数
- `no_kv_offload`: 禁用KV卸载（0/1）
  - 控制是否将KV缓存卸载到CPU
- `flash_attn`: 闪现注意力（0/1）
  - 启用可能提高性能，但需要硬件支持

### 内存管理

- `mmap`: 内存映射（0/1）
  - 启用可以减少内存使用
  - 可能影响加载速度
- `numa`: NUMA策略
  - distribute: 分布式分配
  - isolate: 隔离模式
  - numactl: 使用numactl工具
- `tensor_split`: 张量分割
  - 控制在多个GPU之间的权重分配
  - 格式如"0.3,0.7"表示30%/70%的分配

### 测试控制

- `repetitions`: 重复次数（默认：5）
  - 多次运行以获得稳定结果
  - 建议至少运行3次以上
- `priority`: 进程优先级（0-3）
  - 控制测试进程的系统优先级
- `delay`: 延迟秒数
  - 在测试开始前的等待时间
- `output`: 输出格式
  - 支持：csv/json/jsonl/md/sql
  - md格式最适合人类阅读
  - json/csv适合后续处理
- `output_err`: 错误输出格式
  - 与output相同的选项
  - 设置为none禁用错误输出
- `verbose`: 详细模式（0/1）
  - 启用可以看到更多调试信息
- `progress`: 显示进度（0/1）
  - 启用可以看到实时进度

## 配置建议

### 基本测试配置
```json
{
    "model_path": "model.gguf",
    "config": {
        "n_prompt": 512,
        "n_gen": 128,
        "threads": 32,
        "repetitions": 5,
        "output": "md"
    }
}
```

### GPU优化配置
```json
{
    "model_path": "model.gguf",
    "config": {
        "n_prompt": 512,
        "n_gen": 128,
        "n_gpu_layers": 99,
        "flash_attn": 1,
        "batch_size": 2048,
        "threads": 32,
        "repetitions": 5,
        "output": "md"
    }
}
```

### 多GPU配置
```json
{
    "model_path": "model.gguf",
    "config": {
        "n_prompt": 512,
        "n_gen": 128,
        "n_gpu_layers": 99,
        "split_mode": "layer",
        "tensor_split": "0.5,0.5",
        "main_gpu": 0,
        "threads": 32,
        "repetitions": 5,
        "output": "md"
    }
}
```

## 注意事项

1. 参数组合需要根据具体硬件和模型大小调整
2. 建议先使用基本配置测试，然后逐步调整参数
3. 对于大模型，注意监控内存使用情况
4. 测试时间可能较长，建议使用progress参数监控进度
5. 使用verbose模式可以帮助诊断问题