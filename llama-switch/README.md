# Llama Switch API

这是一个用于管理llama.cpp模型服务和性能评估的RESTful API服务。

## 功能特性

- 模型服务管理（启动/停止/状态查询）
- 模型性能评估（基准测试）
- 异步任务处理
- 进程管理和监控

## 配置说明

默认配置：

- llama-server路径：`E:/Downloads/llama-b5293-bin-win-cuda-cu12.4-x64/llama-server.exe`
- llama-bench路径：`E:/Downloads/llama-b5293-bin-win-cuda-cu12.4-x64/llama-bench.exe`
- 模型目录：`E:/develop/Models`
- API服务器：`127.0.0.1:8080`

## API接口

### 模型服务管理

1. 获取模型列表

```http
GET /api/v1/models
```

响应示例：

```json
{
    "success": true,
    "message": "Found 3 GGUF models",
    "data": [
        {
            "name": "llama2-7b.gguf",
            "path": "E:/develop/Models/llama2-7b.gguf",
            "size": 4815463424
        },
        {
            "name": "mistral-7b.gguf",
            "path": "E:/develop/Models/mistral-7b.gguf",
            "size": 4523347968
        }
    ],
    "error": ""
}
```

2. 切换模型

```http
POST /api/v1/model/switch
```

请求示例：

```json
{
    "model_path": "model.gguf",
    "config": {
        "port": 8080,
        "threads": 4,
        "ctx_size": 2048,
        "batch_size": 512
    }
}
```

响应示例：

```json
{
    "success": true,
    "message": "Model switch initiated",
    "data": null,
    "error": ""
}
```

3. 停止模型服务

```http
POST /api/v1/model/stop
```

响应示例：

```json
{
    "success": true,
    "message": "Model stopped",
    "data": null,
    "error": ""
}
```

4. 获取模型状态

```http
GET /api/v1/model/status[?model_name=名称]
```

参数：

- `model_name` (可选): 指定要查询的模型名称

响应示例（单个模型）:

```json
{
    "success": true,
    "message": "Model status retrieved",
    "data": {
        "running": true,
        "model_name": "llama-7b",
        "model_path": "/path/to/model.gguf",
        "port": 8080,
        "start_time": "2023-01-01T00:00:00Z",
        "process_id": 12345,
        "vram_usage": 4096
    }
}
```

响应示例（多个模型）:

```json
{
    "success": true,
    "message": "Model status retrieved",
    "data": [
        {
            "running": true,
            "model_name": "llama-7b",
            "model_path": "/path/to/model.gguf",
            "port": 8080,
            "start_time": "2023-01-01T00:00:00Z",
            "process_id": 12345,
            "vram_usage": 4096
        },
        {
            "running": true,
            "model_name": "llama-13b",
            "model_path": "/path/to/model2.gguf",
            "port": 8081,
            "start_time": "2023-01-01T00:00:00Z",
            "process_id": 12346,
            "vram_usage": 8192
        }
    ]
}
```

### 基准测试

1. 启动基准测试

```http
POST /api/v1/benchmark
Content-Type: application/json

{
    "model_path": "model.gguf",
    "config": {
        "n_prompt": 512,      // 提示token数量
        "n_gen": 128,         // 生成token数量
        "n_depth": 0,         // 深度
        "batch_size": 2048,   // 批处理大小
        "ubatch_size": 512,   // 微批处理大小
        "cache_type_k": "f16", // K缓存类型 (f16/f32)
        "cache_type_v": "f16", // V缓存类型 (f16/f32)
        "threads": 32,        // 线程数
        "cpu_mask": "",       // CPU掩码 (hex,hex)
        "cpu_strict": 0,      // CPU严格模式 (0/1)
        "poll": 50,          // 轮询间隔 (0-100)
        "n_gpu_layers": 99,   // GPU层数
        "split_mode": "layer", // 分割模式 (none/layer/row)
        "main_gpu": 0,        // 主GPU
        "no_kv_offload": 0,   // 禁用KV卸载 (0/1)
        "flash_attn": 0,      // 闪现注意力 (0/1)
        "mmap": 1,           // 内存映射 (0/1)
        "numa": "",          // NUMA策略 (distribute/isolate/numactl)
        "embeddings": 0,      // 嵌入模式 (0/1)
        "tensor_split": "0",  // 张量分割
        "repetitions": 5,     // 重复次数
        "priority": 0,       // 优先级 (0-3)
        "delay": 0,          // 延迟（秒）
        "output": "md",      // 输出格式 (csv/json/jsonl/md/sql)
        "output_err": "none", // 错误输出格式
        "verbose": 0,        // 详细模式 (0/1)
        "progress": 0        // 显示进度 (0/1)
    }
}
```

2. 获取测试状态

```http
GET /api/v1/benchmark/status?task_id={task_id}
```

## 文档

- [配置指南](docs/configuration.md)
- [模型服务参数详细说明](docs/model_params.md)
- [基准测试参数详细说明](docs/benchmark_params.md)

## 快速开始

### 环境准备

1. 确保已安装Go 1.20+和CUDA(如需GPU加速)

2. 克隆仓库并进入项目目录：

```bash
git clone https://github.com/your-repo/llama-switch.git
cd llama-switch
```

### 配置文件设置

1. 复制环境变量文件：

```bash
cp llama-switch/.env.example llama-switch/.env
```

2. 编辑配置文件：

```bash
# Windows使用记事本
notepad llama-switch/.env

# Linux/macOS使用nano
nano llama-switch/.env
```

3. 配置示例：

```env
# 必需配置
# llama.cpp 二进制文件路径
LLAMA_SERVER_PATH=E:/Downloads/llama-b5293-bin-win-cuda-cu12.4-x64/llama-server.exe
LLAMA_BENCH_PATH=E:/Downloads/llama-b5293-bin-win-cuda-cu12.4-x64/llama-bench.exe

MODELS_DIR=/path/to/models

# 可选配置
SERVER_PORT=8080
GPU_LAYERS=20  # GPU加速层数
```

### 启动服务

1. 开发模式：

```bash
# 从项目根目录启动
cd llama-switch
go run cmd/server/main.go

# 指定配置文件路径
CONFIG_PATH=$(pwd)/llama-switch/.env go run cmd/server/main.go
```

2. 生产模式：

```bash
# 编译
go build -o llama-switch cmd/server/main.go

# 运行
./llama-switch
```

### 故障排查

- 如果提示"找不到.env文件"：

  ```bash
  # 检查文件是否存在
  ls -l llama-switch/.env
  
  # 使用绝对路径
  CONFIG_PATH=/absolute/path/to/.env go run cmd/server/main.go
  ```

- 如果提示"权限不足"：

  ```bash
  # Linux/macOS
  chmod 644 llama-switch/.env
  
  # Windows检查文件属性
  ```

## 配置示例

### 1. 基本CPU模式

适用于基本的CPU推理场景：

```json
{
    "model_path": "llama2-7b-chat.gguf",
    "config": {
        "host": "127.0.0.1",
        "port": 8080,
        "threads": 8,
        "ctx_size": 4096,
        "batch_size": 512
    }
}
```

### 2. GPU加速模式

适用于单GPU推理场景：

```json
{
    "model_path": "llama2-7b-chat.gguf",
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

### 3. 多GPU分布式模式

适用于多GPU推理场景：

```json
{
    "model_path": "llama2-7b-chat.gguf",
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

### 4. 高性能服务模式

适用于生产环境的高性能配置：

```json
{
    "model_path": "llama2-7b-chat.gguf",
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

## 使用示例

1. 启动服务器：

```bash
cd cmd/server
go run main.go
```

2. 获取可用模型列表：

```bash
curl http://localhost:8080/api/v1/models
```

3. 切换模型：
```bash
curl -X POST http://localhost:8080/api/v1/model/switch \
    -H "Content-Type: application/json" \
    -d '{
        "model_path": "llama2-7b-chat.gguf",
        "config": {
            "port": 8080,
            "threads": 4,
            "ctx_size": 2048,
            "batch_size": 512
        }
    }'
```

4. 运行基准测试（基本配置）：

```bash
curl -X POST http://localhost:8080/api/v1/benchmark \
    -H "Content-Type: application/json" \
    -d '{
        "model_path": "llama2-7b-chat.gguf",
        "config": {
            "n_prompt": 512,
            "n_gen": 128,
            "threads": 32,
            "repetitions": 5,
            "output": "md",
            "progress": 1
        }
    }'
```

5. GPU优化的基准测试：

```bash
curl -X POST http://localhost:8080/api/v1/benchmark \
    -H "Content-Type: application/json" \
    -d '{
        "model_path": "llama2-7b-chat.gguf",
        "config": {
            "n_prompt": 512,
            "n_gen": 128,
            "n_gpu_layers": 99,
            "flash_attn": 1,
            "batch_size": 2048,
            "threads": 32,
            "repetitions": 5,
            "output": "md",
            "progress": 1
        }
    }'
```

6. 获取基准测试状态：

```bash
curl http://localhost:8080/api/v1/benchmark/status?task_id=<task_id>
```

## 常见配置场景

### 1. CPU优化配置

适用于CPU推理场景：

```json
{
    "model_path": "llama2-7b-chat.gguf",
    "config": {
        "n_prompt": 512,
        "n_gen": 128,
        "threads": 32,
        "batch_size": 1024,
        "n_gpu_layers": 0,
        "mmap": 1,
        "repetitions": 5
    }
}
```

### 2. 单GPU配置

适用于单GPU推理场景：

```json
{
    "model_path": "llama2-7b-chat.gguf",
    "config": {
        "n_prompt": 512,
        "n_gen": 128,
        "n_gpu_layers": 99,
        "flash_attn": 1,
        "batch_size": 2048,
        "threads": 32,
        "repetitions": 5
    }
}
```

### 3. 多GPU配置

适用于多GPU推理场景：

```json
{
    "model_path": "llama2-7b-chat.gguf",
    "config": {
        "n_prompt": 512,
        "n_gen": 128,
        "n_gpu_layers": 99,
        "split_mode": "layer",
        "tensor_split": "0.5,0.5",
        "main_gpu": 0,
        "flash_attn": 1,
        "threads": 32,
        "repetitions": 5
    }
}
```

## 注意事项

1. 确保llama.cpp的二进制文件（llama-server和llama-bench）已经正确编译并放置在配置指定的位置
2. 确保模型文件(.gguf格式)已经放置在配置指定的模型目录中
3. 基准测试任务是异步执行的，需要通过task_id查询结果