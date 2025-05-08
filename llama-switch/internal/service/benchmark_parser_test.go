package service

import (
	"testing"
)

func TestParseBenchmarkOutput_MultipleTests(t *testing.T) {
	input := `ggml_cuda_init: GGML_CUDA_FORCE_MMQ:    no
ggml_cuda_init: GGML_CUDA_FORCE_CUBLAS: no
ggml_cuda_init: found 1 CUDA devices:
  Device 0: Tesla P40, compute capability 6.1, VMM: no
load_backend: loaded CUDA backend from E:\Downloads\llama-b5293-bin-win-cuda-cu12.4-x64\ggml-cuda.dll
load_backend: loaded RPC backend from E:\Downloads\llama-b5293-bin-win-cuda-cu12.4-x64\ggml-rpc.dll
load_backend: loaded CPU backend from E:\Downloads\llama-b5293-bin-win-cuda-cu12.4-x64\ggml-cpu-skylakex.dll
| model                          |       size |     params | backend    | ngl | mmap |            test |                  t/s |
| ------------------------------ | ---------: | ---------: | ---------- | --: | ---: | --------------: | -------------------: |
| qwen2 32B Q4_K - Medium        |  18.48 GiB |    32.76 B | CUDA,RPC   |  99 |    0 |           pp512 |        212.25 ± 0.47 |
| qwen2 32B Q4_K - Medium        |  18.48 GiB |    32.76 B | CUDA,RPC   |  99 |    0 |           tg128 |          9.48 ± 0.00 |

build: 1e333d5b (5293)`

	result, err := ParseBenchmarkOutput(input)
	if err != nil {
		t.Fatalf("ParseBenchmarkOutput failed: %v", err)
	}

	// 验证设备信息
	if len(result.DeviceInfo.CUDADevices) != 1 {
		t.Errorf("Expected 1 CUDA device, got %d", len(result.DeviceInfo.CUDADevices))
	}

	// 验证后端信息
	if len(result.DeviceInfo.BackendsLoaded) < 2 {
		t.Errorf("Expected at least 2 backends, got %d", len(result.DeviceInfo.BackendsLoaded))
	}

	// 验证Models分组
	if len(result.Models) != 1 {
		t.Errorf("Expected 1 model, got %d", len(result.Models))
	}

	model := result.Models[0]
	if model.Model != "qwen2 32B Q4_K - Medium" {
		t.Errorf("Unexpected model name: %s", model.Model)
	}

	// 验证测试结果分组
	if len(model.TestResults) != 2 {
		t.Errorf("Expected 2 test results, got %d", len(model.TestResults))
	}

	// 验证旧Tests结构
	if len(result.Tests) != 2 {
		t.Errorf("Expected 2 tests in legacy structure, got %d", len(result.Tests))
	}
}

func TestParseBenchmarkOutput_SingleTest(t *testing.T) {
	input := `| model                          |       size |     params | backend    | ngl | mmap |            test |                  t/s |
| ------------------------------ | ---------: | ---------: | ---------- | --: | ---: | --------------: | -------------------: |
| qwen2 32B Q4_K - Medium        |  18.48 GiB |    32.76 B | CUDA,RPC   |  99 |    0 |           pp512 |        212.25 ± 0.47 |

build: 1e333d5b (5293)`

	result, err := ParseBenchmarkOutput(input)
	if err != nil {
		t.Fatalf("ParseBenchmarkOutput failed: %v", err)
	}

	if len(result.Models) != 1 {
		t.Errorf("Expected 1 model, got %d", len(result.Models))
	}

	if len(result.Models[0].TestResults) != 1 {
		t.Errorf("Expected 1 test result, got %d", len(result.Models[0].TestResults))
	}
}

func TestParseBenchmarkOutput_InvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"Empty", ""},
		{"MissingTable", "build: 1e333d5b (5293)"},
		{"MalformedTable", "| header1 | header2 |\n| ------- | ------- |\n| value1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseBenchmarkOutput(tt.input)
			if err == nil {
				t.Error("Expected error for invalid input, got nil")
			}
		})
	}
}
