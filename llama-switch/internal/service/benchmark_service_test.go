package service

import (
	"encoding/json"
	"testing"

	"llama-switch/internal/model"
)

func TestBenchmarkOutputJSON(t *testing.T) {
	// 模拟解析结果
	result := &BenchmarkResult{
		Tests: []struct {
			Model           string  `json:"model"`
			Size            string  `json:"size"`
			Params          string  `json:"params"`
			Backend         string  `json:"backend"`
			GPULayers       int     `json:"gpu_layers"`
			MMap            bool    `json:"mmap"`
			TestType        string  `json:"test_type"`
			TokensPerSecond float64 `json:"tokens_per_second"`
			Variation       float64 `json:"variation"`
		}{
			{
				Model:           "qwen2 32B Q4_K - Medium",
				Size:            "18.48 GiB",
				Params:          "32.76 B",
				Backend:         "CUDA,RPC",
				GPULayers:       99,
				MMap:            false,
				TestType:        "pp512",
				TokensPerSecond: 212.13,
				Variation:       0.29,
			},
			{
				Model:           "qwen2 32B Q4_K - Medium",
				Size:            "18.48 GiB",
				Params:          "32.76 B",
				Backend:         "CUDA,RPC",
				GPULayers:       99,
				MMap:            false,
				TestType:        "tg128",
				TokensPerSecond: 9.49,
				Variation:       0.00,
			},
		},
	}

	// 处理结果
	status := &model.BenchmarkStatus{}
	var allResults []*model.BenchmarkResults
	for _, testResult := range result.Tests {
		allResults = append(allResults, &model.BenchmarkResults{
			Model:           testResult.Model,
			Size:            testResult.Size,
			Params:          testResult.Params,
			Backend:         testResult.Backend,
			GPULayers:       testResult.GPULayers,
			MMap:            testResult.MMap,
			TestType:        testResult.TestType,
			TokensPerSecond: testResult.TokensPerSecond,
			Variation:       testResult.Variation,
		})
	}
	status.AllResults = allResults

	// 转换为JSON
	jsonData, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	// 验证JSON结构
	var jsonMap map[string]interface{}
	if err := json.Unmarshal(jsonData, &jsonMap); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	// 检查AllResults字段
	if allResults, ok := jsonMap["all_results"].([]interface{}); !ok {
		t.Error("Missing all_results field in JSON")
	} else if len(allResults) != 2 {
		t.Errorf("Expected 2 results in all_results, got %d", len(allResults))
	}

	// 检查AllResults字段
	if allResults, ok := jsonMap["all_results"].([]interface{}); !ok {
		t.Error("Missing all_results field in JSON")
	} else if len(allResults) != 2 {
		t.Errorf("Expected 2 results in all_results, got %d", len(allResults))
	}
}
