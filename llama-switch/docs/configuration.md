# 配置指南

本文档详细说明了llama-switch的配置选项和环境变量使用方法。

## 配置文件

服务支持通过`.env`文件进行配置。你可以复制`.env.example`文件并重命名为`.env`，然后根据需要修改配置项：

```bash
cp .env.example .env
```

## 环境变量

### 基本路径配置

```env
# llama.cpp 二进制文件路径
LLAMA_SERVER_PATH=E:/Downloads/llama-b5293-bin-win-cuda-cu12.4-x64/llama-server.exe
LLAMA_BENCH_PATH=E:/Downloads/llama-b5293-bin-win-cuda-cu12.4-x64/llama-bench.exe

# 模型目录
MODELS_DIR=E:/develop/Models
```

### API服务器配置

```env
# API服务器配置
SERVER_HOST=127.0.0.1    # 监听地址
SERVER_PORT=8080         # 服务端口
SERVER_TIMEOUT=600       # 超时时间（秒）
```

### 默认模型配置

```env
# 默认模型配置
DEFAULT_THREADS=8        # 线程数
DEFAULT_CTX_SIZE=4096   # 上下文大小
DEFAULT_BATCH_SIZE=512  # 批处理大小
DEFAULT_UBATCH_SIZE=512 # 微批处理大小
```

### GPU配置

```env
# GPU配置
DEFAULT_GPU_LAYERS=99   # GPU层数（99表示全部）
DEFAULT_SPLIT_MODE=layer # GPU分割模式（none/layer/row）
DEFAULT_MAIN_GPU=0      # 主GPU编号
ENABLE_FLASH_ATTN=true  # 启用Flash Attention
```

### 缓存配置

```env
# 缓存配置
DEFAULT_CACHE_TYPE_K=f16 # K缓存类型
DEFAULT_CACHE_TYPE_V=f16 # V缓存类型
```

支持的缓存类型：
- f32: 32位浮点
- f16: 16位浮点
- bf16: bfloat16
- q8_0: 8位量化
- q4_0/q4_1: 4位量化
- iq4_nl: 4位非线性量化
- q5_0/q5_1: 5位量化

### 内存管理

```env
# 内存管理
ENABLE_MLOCK=false     # 锁定内存
ENABLE_MMAP=true       # 启用内存映射
NUMA_STRATEGY=         # NUMA策略（distribute/isolate/numactl）
```

### 日志配置

```env
# 日志配置
LOG_LEVEL=info         # 日志级别（debug/info/warn/error）
LOG_FILE=             # 日志文件路径（留空表示不写入文件）
ENABLE_CONSOLE_LOG=true # 启用控制台日志
```

### 安全配置

```env
# 安全配置
API_KEY=              # API密钥
SSL_KEY_FILE=         # SSL私钥文件路径
SSL_CERT_FILE=        # SSL证书文件路径
```

## 配置优先级

配置项的加载优先级从高到低为：

1. 命令行参数（如果实现）
2. 环境变量
3. .env文件
4. .env.example文件
5. 程序默认值

## 配置验证

服务启动时会对配置进行验证，包括：

1. 文件路径验证
   - 检查llama-server和llama-bench是否存在
   - 检查模型目录是否存在

2. 参数范围验证
   - 端口号（1-65535）
   - 线程数（>= -1）
   - 上下文大小（>= 0）
   - 批处理大小（>= 0）

3. 枚举值验证
   - 分割模式（none/layer/row）
   - 缓存类型
   - NUMA策略
   - 日志级别

4. SSL配置验证
   - 如果指定了SSL密钥，必须同时指定证书
   - 如果指定了SSL证书，必须同时指定密钥

## 配置示例

### 基本CPU配置
```env
SERVER_HOST=127.0.0.1
SERVER_PORT=8080
DEFAULT_THREADS=8
DEFAULT_CTX_SIZE=4096
DEFAULT_BATCH_SIZE=512
ENABLE_MMAP=true
```

### GPU优化配置
```env
SERVER_HOST=127.0.0.1
SERVER_PORT=8080
DEFAULT_THREADS=8
DEFAULT_GPU_LAYERS=99
DEFAULT_SPLIT_MODE=layer
ENABLE_FLASH_ATTN=true
DEFAULT_BATCH_SIZE=2048
DEFAULT_CTX_SIZE=4096
```

### 生产环境配置
```env
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
SERVER_TIMEOUT=3600
DEFAULT_THREADS=32
DEFAULT_GPU_LAYERS=99
ENABLE_FLASH_ATTN=true
DEFAULT_BATCH_SIZE=2048
DEFAULT_UBATCH_SIZE=512
ENABLE_MLOCK=true
LOG_LEVEL=info
LOG_FILE=/var/log/llama-switch.log
API_KEY=your-api-key
SSL_KEY_FILE=/path/to/private.key
SSL_CERT_FILE=/path/to/certificate.crt
```

## 注意事项

1. 路径配置
   - Windows系统使用反斜杠（\）或正斜杠（/）
   - Linux/macOS系统使用正斜杠（/）
   - 建议使用绝对路径

2. 内存配置
   - ENABLE_MLOCK需要足够的系统权限
   - 大的上下文大小需要更多内存

3. GPU配置
   - GPU相关参数只在有GPU时生效
   - 多GPU配置需要正确设置分割模式

4. 安全配置
   - 生产环境建议启用API密钥
   - 公网访问建议配置SSL