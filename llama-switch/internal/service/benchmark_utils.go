package service

import (
	"strconv"
	"strings"
)

// calculateTotalTokens 根据测试类型计算总token数
func calculateTotalTokens(testType string) int {
	switch {
	case strings.HasPrefix(testType, "pp"):
		tokens, _ := strconv.Atoi(strings.TrimPrefix(testType, "pp"))
		return tokens
	case strings.HasPrefix(testType, "tg"):
		tokens, _ := strconv.Atoi(strings.TrimPrefix(testType, "tg"))
		return tokens
	default:
		return 0
	}
}

// calculateTotalTime 根据token数和速度计算总时间
func calculateTotalTime(testType string, tokensPerSecond float64) float64 {
	totalTokens := float64(calculateTotalTokens(testType))
	if tokensPerSecond <= 0 {
		return 0
	}
	return totalTokens / tokensPerSecond
}
