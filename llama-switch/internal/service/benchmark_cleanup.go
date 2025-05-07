package service

import (
	"fmt"
	"log"
	"time"
)

// StopTask 停止指定的基准测试任务
func (s *BenchmarkService) StopTask(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	status, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if status.Status != "running" {
		return fmt.Errorf("task is not running: %s (current status: %s)", taskID, status.Status)
	}

	// 调用取消函数
	if status.CancelFunc != nil {
		status.CancelFunc()
	}

	// 更新任务状态
	status.Status = "cancelled"
	status.EndTime = time.Now().Format(time.RFC3339)
	log.Printf("Benchmark task cancelled: %s", taskID)

	return nil
}

// StopAllTasks 停止所有正在运行的基准测试任务
func (s *BenchmarkService) StopAllTasks() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for taskID, status := range s.tasks {
		if status.Status == "running" {
			if status.CancelFunc != nil {
				status.CancelFunc()
			}
			status.Status = "cancelled"
			status.EndTime = time.Now().Format(time.RFC3339)
			log.Printf("Benchmark task cancelled: %s", taskID)
		}
	}
}

// Cleanup 清理所有任务资源
func (s *BenchmarkService) Cleanup() {
	log.Println("Cleaning up benchmark service...")
	s.StopAllTasks()
}
