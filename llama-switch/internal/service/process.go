package service

import (
	"context"
	"fmt"
	"llama-switch/internal/model"
	"log"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// ProcessManager 进程管理器
type ProcessManager struct {
	mu      sync.Mutex
	process *os.Process
	cmd     *exec.Cmd
	models  map[int]*model.ModelStatus // 跟踪运行中的模型及其显存使用
}

// init 初始化ProcessManager
func (pm *ProcessManager) init() {
	if pm.models == nil {
		pm.models = make(map[int]*model.ModelStatus)
	}
}

// NewProcessManager 创建新的进程管理器
func NewProcessManager() *ProcessManager {
	return &ProcessManager{}
}

// StartProcess 启动新进程
func (pm *ProcessManager) StartProcess(command string, args []string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 创建新的命令
	cmd := exec.Command(command, args...)

	// 设置进程组ID，这样可以一次性结束所有子进程
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}

	// 设置标准输出和错误输出
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 启动进程
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %v", err)
	}

	pm.process = cmd.Process
	pm.cmd = cmd

	// 在后台等待进程结束
	go func() {
		// 捕获进程退出状态
		err := cmd.Wait()

		pm.mu.Lock()
		// 清理进程状态
		if pm.process != nil && pm.process.Pid == cmd.Process.Pid {
			pm.process = nil
			pm.cmd = nil
		}

		// 清理模型状态
		if model, exists := pm.models[cmd.Process.Pid]; exists {
			delete(pm.models, cmd.Process.Pid)
			log.Printf("Model '%s' (PID: %d) exited: %v",
				model.ModelName, cmd.Process.Pid, err)
		} else {
			log.Printf("Process exited (PID: %d): %v",
				cmd.Process.Pid, err)
		}
		pm.mu.Unlock()
	}()

	return nil
}

// StopProcess 停止当前进程
func (pm *ProcessManager) StopProcess() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.process == nil {
		return nil
	}

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 使用通道接收停止结果
	done := make(chan error, 1)
	go func() {
		// 在Windows上，我们需要发送Ctrl+C信号来优雅地关闭进程
		if err := pm.process.Signal(os.Interrupt); err != nil {
			// 如果发送中断信号失败，则强制结束进程
			if err := pm.process.Kill(); err != nil {
				done <- fmt.Errorf("failed to kill process: %v", err)
				return
			}
		}

		// 等待进程退出
		_, err := pm.process.Wait()
		done <- err
	}()

	// 等待停止完成或超时
	select {
	case err := <-done:
		pm.process = nil
		pm.cmd = nil
		return err
	case <-ctx.Done():
		// 超时后强制终止进程
		if err := pm.process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process after timeout: %v", err)
		}
		return fmt.Errorf("process termination timed out")
	}
}

// IsRunning 检查进程是否在运行
func (pm *ProcessManager) IsRunning() bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.process != nil
}

// GetPID 获取当前运行进程的PID
func (pm *ProcessManager) GetPID() int {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if pm.process != nil {
		return pm.process.Pid
	}
	return 0
}

// IsProcessRunning 检查指定PID的进程是否在运行
func (pm *ProcessManager) IsProcessRunning(pid int) bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 检查当前管理的进程
	if pm.process != nil && pm.process.Pid == pid {
		return true
	}

	// 检查系统进程状态
	if runtime.GOOS == "windows" {
		// Windows平台使用tasklist检查进程
		out, err := exec.Command("tasklist", "/fi", fmt.Sprintf("PID eq %d", pid)).Output()
		if err != nil {
			return false
		}
		return strings.Contains(string(out), strconv.Itoa(pid))
	} else {
		// Unix平台使用Signal(0)检查进程
		process, err := os.FindProcess(pid)
		if err != nil {
			return false
		}
		err = process.Signal(syscall.Signal(0))
		return err == nil
	}
}

// AddModel 添加运行中的模型
func (pm *ProcessManager) AddModel(pid int, m *model.ModelStatus) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.init()
	pm.models[pid] = m
}

// RemoveModel 移除已停止的模型
func (pm *ProcessManager) RemoveModel(pid int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	delete(pm.models, pid)
}

// UpdateModel 更新指定PID的模型状态
func (pm *ProcessManager) UpdateModel(pid int, status *model.ModelStatus) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.models[pid] = status
}

// GetRunningModels 获取运行中的模型列表
func (pm *ProcessManager) GetRunningModels() []*model.ModelStatus {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.init()

	models := make([]*model.ModelStatus, 0, len(pm.models))
	toRemove := make([]int, 0)

	for pid, m := range pm.models {
		// 验证进程是否还在运行
		var isRunning bool
		if runtime.GOOS == "windows" {
			// Windows平台使用tasklist检查进程
			out, err := exec.Command("tasklist", "/fi", fmt.Sprintf("PID eq %d", pid)).Output()
			if err != nil {
				log.Printf("Warning: Failed to check process %d status: %v", pid, err)
				toRemove = append(toRemove, pid)
				continue
			}
			isRunning = strings.Contains(string(out), strconv.Itoa(pid))
		} else {
			// Unix平台使用Signal(0)检查进程
			process, err := os.FindProcess(pid)
			if err != nil {
				log.Printf("Warning: Failed to find process %d: %v", pid, err)
				toRemove = append(toRemove, pid)
				continue
			}
			err = process.Signal(syscall.Signal(0))
			isRunning = err == nil
		}

		if !isRunning {
			log.Printf("Warning: Process %d (model: %s) is not running", pid, m.ModelName)
			toRemove = append(toRemove, pid)
			continue
		}

		// 进程正在运行，添加到结果列表
		models = append(models, m)
	}

	// 清理已停止的进程状态
	for _, pid := range toRemove {
		log.Printf("Cleaning up stopped model (PID: %d, Name: %s)",
			pid, pm.models[pid].ModelName)
		delete(pm.models, pid)
	}

	return models
}

// GetModelsByVRAMUsage 按显存使用排序(降序)
func (pm *ProcessManager) GetModelsByVRAMUsage() []*model.ModelStatus {
	models := pm.GetRunningModels()
	sort.Slice(models, func(i, j int) bool {
		return models[i].VRAMUsage > models[j].VRAMUsage
	})
	return models
}

// StopModel 停止指定模型
func (pm *ProcessManager) StopModel(model_name string) (*model.ModelStatus, error) {
	if model_name == "" {
		return nil, fmt.Errorf("model_name parameter is required")
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 查找匹配的模型
	var targetModel *model.ModelStatus
	var targetPID int
	found := false

	for pid, m := range pm.models {
		if m.ModelName == model_name {
			targetModel = m
			targetPID = pid
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("model '%s' not found", model_name)
	}

	// 停止进程
	if err := pm.stopProcessByPID(targetPID); err != nil {
		return nil, fmt.Errorf("failed to stop model '%s': %v", model_name, err)
	}

	// 清理模型状态
	delete(pm.models, targetPID)
	log.Printf("Model '%s' (PID: %d) stopped successfully", model_name, targetPID)

	return targetModel, nil
}

// stopProcessByPID 停止指定PID的进程
func (pm *ProcessManager) stopProcessByPID(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process %d: %v", pid, err)
	}

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 使用通道接收停止结果
	done := make(chan error, 1)
	go func() {
		// 发送中断信号
		if err := process.Signal(os.Interrupt); err != nil {
			// 如果发送中断信号失败，则强制结束进程
			if err := process.Kill(); err != nil {
				done <- fmt.Errorf("failed to kill process %d: %v", pid, err)
				return
			}
		}

		// 等待进程退出
		_, err := process.Wait()
		done <- err
	}()

	// 等待停止完成或超时
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		// 超时后强制终止进程
		if err := process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process %d after timeout: %v", pid, err)
		}
		return fmt.Errorf("process %d termination timed out", pid)
	}
}
