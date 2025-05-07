package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// ProcessManager 进程管理器
type ProcessManager struct {
	mu      sync.Mutex
	process *os.Process
	cmd     *exec.Cmd
}

// NewProcessManager 创建新的进程管理器
func NewProcessManager() *ProcessManager {
	return &ProcessManager{}
}

// StartProcess 启动新进程
func (pm *ProcessManager) StartProcess(command string, args []string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 如果已有进程在运行，先停止它
	if pm.process != nil {
		if err := pm.StopProcess(); err != nil {
			return fmt.Errorf("failed to stop existing process: %v", err)
		}
	}

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
		cmd.Wait()
		pm.mu.Lock()
		pm.process = nil
		pm.cmd = nil
		pm.mu.Unlock()
		log.Printf("Process exited: %s", command)
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
