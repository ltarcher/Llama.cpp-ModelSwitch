package service

import (
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

	BuildInfo struct {
		CommitHash  string `json:"commit_hash"`
		BuildNumber string `json:"build_number"`
	} `json:"build_info"`
}

func ParseBenchmarkOutput(output string) (*BenchmarkResult, error) {
	result := &BenchmarkResult{}

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
	for _, match := range testMatches {
		if len(match) < 9 {
			continue
		}

		tokensPerSecond, variation := parseTokensPerSecond(match[8])
		gpuLayers := mustAtoi(strings.TrimSpace(match[5]))
		mmap := strings.TrimSpace(match[6]) == "1"

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
			Model:           strings.TrimSpace(match[1]),
			Size:            strings.TrimSpace(match[2]),
			Params:          strings.TrimSpace(match[3]),
			Backend:         strings.TrimSpace(match[4]),
			GPULayers:       gpuLayers,
			MMap:            mmap,
			TestType:        strings.TrimSpace(match[7]),
			TokensPerSecond: tokensPerSecond,
			Variation:       variation,
		}

		result.Tests = append(result.Tests, test)
	}

	// 解析构建信息
	buildRe := regexp.MustCompile(`build: (\w+) \((\d+)\)`)
	buildMatch := buildRe.FindStringSubmatch(output)
	if len(buildMatch) == 3 {
		result.BuildInfo.CommitHash = buildMatch[1]
		result.BuildInfo.BuildNumber = buildMatch[2]
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
