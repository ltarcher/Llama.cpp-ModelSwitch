package service

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type BenchmarkResult struct {
	DeviceInfo struct {
		CUDADevices []struct {
			ID                int    `json:"id"`
			Name              string `json:"name"`
			ComputeCapability string `json:"compute_capability"`
			VMM               bool   `json:"vmm"`
		} `json:"cuda_devices"`
		BackendsLoaded []string `json:"backends_loaded"`
	} `json:"device_info"`

	// Deprecated: Use Models instead
	Tests []struct {
		Model           string  `json:"model"`
		Size            string  `json:"size"`
		Params          string  `json:"params"`
		Backend         string  `json:"backend"`
		GPULayers       int     `json:"gpu_layers"`
		MMap            bool    `json:"mmap"`
		TestType        string  `json:"test_type"`
		TokensPerSecond float64 `json:"tokens_per_second"`
		Variation       float64 `json:"variation"`
	} `json:"tests"`

	Models []ModelResult `json:"models"`

	BuildInfo struct {
		CommitHash  string `json:"commit_hash"`
		BuildNumber string `json:"build_number"`
	} `json:"build_info"`
}

type ModelResult struct {
	Model       string      `json:"model"`
	Size        string      `json:"size"`
	Params      string      `json:"params"`
	Backend     string      `json:"backend"`
	GPULayers   int         `json:"gpu_layers"`
	MMap        bool        `json:"mmap"`
	TestResults []TestEntry `json:"test_results"`
}

type TestEntry struct {
	TestType        string  `json:"test_type"`
	TokensPerSecond float64 `json:"tokens_per_second"`
	Variation       float64 `json:"variation"`
}

func ParseBenchmarkOutput(output string) (*BenchmarkResult, error) {
	if output == "" {
		return nil, fmt.Errorf("empty input")
	}

	// 检查是否包含必要的内容
	if !strings.Contains(output, "Device") && !strings.Contains(output, "model") {
		return nil, fmt.Errorf("invalid benchmark format: missing required content")
	}

	result := &BenchmarkResult{}
	modelMap := make(map[string]*ModelResult)

	// 解析CUDA设备信息
	cudaDeviceRe := regexp.MustCompile(`Device (\d+): ([^,]+), compute capability ([^,]+), VMM: (yes|no)`)
	matches := cudaDeviceRe.FindAllStringSubmatch(output, -1)
	for _, match := range matches {
		device := struct {
			ID                int    `json:"id"`
			Name              string `json:"name"`
			ComputeCapability string `json:"compute_capability"`
			VMM               bool   `json:"vmm"`
		}{
			ID:                mustAtoi(match[1]),
			Name:              match[2],
			ComputeCapability: match[3],
			VMM:               match[4] == "yes",
		}
		result.DeviceInfo.CUDADevices = append(result.DeviceInfo.CUDADevices, device)
	}

	// 解析加载的后端
	backendRe := regexp.MustCompile(`load_backend: loaded (\w+) backend`)
	backendMatches := backendRe.FindAllStringSubmatch(output, -1)
	for _, match := range backendMatches {
		result.DeviceInfo.BackendsLoaded = append(result.DeviceInfo.BackendsLoaded, match[1])
	}

	// 解析测试结果表格
	testRe := regexp.MustCompile(`\|\s*([^\|]+)\s*\|\s*([^\|]+)\s*\|\s*([^\|]+)\s*\|\s*([^\|]+)\s*\|\s*([^\|]+)\s*\|\s*([^\|]+)\s*\|\s*([^\|]+)\s*\|\s*([^\|]+)\s*\|`)
	testMatches := testRe.FindAllStringSubmatch(output, -1)

	// 跳过标题行和分隔线
	for _, match := range testMatches {
		if len(match) < 9 {
			continue
		}

		// 跳过标题行和分隔线
		modelName := strings.TrimSpace(match[1])
		if strings.HasPrefix(modelName, "----") || modelName == "model" {
			continue
		}

		tokensPerSecond, variation := parseTokensPerSecond(match[8])
		gpuLayers := mustAtoi(strings.TrimSpace(match[5]))
		mmap := strings.TrimSpace(match[6]) == "1"

		testType := strings.TrimSpace(match[7])

		// 确保是有效的测试数据行
		if modelName == "" || testType == "" {
			continue
		}

		// 填充旧Tests结构
		test := struct {
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
			Model:           modelName,
			Size:            strings.TrimSpace(match[2]),
			Params:          strings.TrimSpace(match[3]),
			Backend:         strings.TrimSpace(match[4]),
			GPULayers:       gpuLayers,
			MMap:            mmap,
			TestType:        testType,
			TokensPerSecond: tokensPerSecond,
			Variation:       variation,
		}
		result.Tests = append(result.Tests, test)

		// 填充新Models结构
		modelKey := fmt.Sprintf("%s|%s|%d|%v",
			modelName,
			strings.TrimSpace(match[4]),
			gpuLayers,
			mmap)

		if existing, ok := modelMap[modelKey]; ok {
			// 检查是否已存在相同的测试类型
			exists := false
			for _, tr := range existing.TestResults {
				if tr.TestType == testType {
					exists = true
					break
				}
			}
			if !exists {
				existing.TestResults = append(existing.TestResults, TestEntry{
					TestType:        testType,
					TokensPerSecond: tokensPerSecond,
					Variation:       variation,
				})
			}
		} else {
			modelMap[modelKey] = &ModelResult{
				Model:     modelName,
				Size:      strings.TrimSpace(match[2]),
				Params:    strings.TrimSpace(match[3]),
				Backend:   strings.TrimSpace(match[4]),
				GPULayers: gpuLayers,
				MMap:      mmap,
				TestResults: []TestEntry{{
					TestType:        testType,
					TokensPerSecond: tokensPerSecond,
					Variation:       variation,
				}},
			}
		}
	}

	// 将map转换为slice
	for _, model := range modelMap {
		result.Models = append(result.Models, *model)
	}

	// 解析构建信息
	buildRe := regexp.MustCompile(`build: (\w+) \((\d+)\)`)
	buildMatch := buildRe.FindStringSubmatch(output)
	if len(buildMatch) == 3 {
		result.BuildInfo.CommitHash = buildMatch[1]
		result.BuildInfo.BuildNumber = buildMatch[2]
	}

	// 验证解析结果
	if len(result.Models) == 0 {
		return nil, fmt.Errorf("no valid test results found")
	}

	// 验证是否至少有一个测试结果包含有效数据
	hasValidTest := false
	for _, model := range result.Models {
		if len(model.TestResults) > 0 {
			hasValidTest = true
			break
		}
	}
	if !hasValidTest {
		return nil, fmt.Errorf("no valid test results found in parsed data")
	}

	return result, nil
}

func parseTokensPerSecond(s string) (float64, float64) {
	parts := strings.Split(s, "±")
	if len(parts) != 2 {
		return 0, 0
	}

	tokens, _ := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	variation, _ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	return tokens, variation
}

func mustAtoi(s string) int {
	i, _ := strconv.Atoi(strings.TrimSpace(s))
	return i
}
