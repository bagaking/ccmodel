package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// DefaultFileLockManager 文件锁管理器的具体实现
type DefaultFileLockManager struct {
	staleLockTimeout time.Duration
	logger           Logger
}

// NewFileLockManager 创建文件锁管理器
func NewFileLockManager(logger Logger) FileLockManager {
	return &DefaultFileLockManager{
		staleLockTimeout: 10 * time.Minute, // 10分钟后清理过期锁
		logger:           logger,
	}
}

// WithLock 使用写锁执行操作，确保并发安全
func (flm *DefaultFileLockManager) WithLock(lockFile string, operation func() error) error {
	if err := flm.acquireLock(lockFile); err != nil {
		return fmt.Errorf("failed to acquire lock: %v", err)
	}

	defer func() {
		if err := flm.releaseLock(lockFile); err != nil {
			flm.logger.Warn("Failed to release lock", "lockFile", lockFile, "error", err)
		}
	}()

	return operation()
}

// WithReadLock 使用读锁执行操作（当前实现与写锁相同，可根据需要优化）
func (flm *DefaultFileLockManager) WithReadLock(lockFile string, operation func() error) error {
	// 当前简化实现，与写锁相同
	// 未来可以实现读写锁分离优化并发性能
	return flm.WithLock(lockFile, operation)
}

// acquireLock 获取文件锁
func (flm *DefaultFileLockManager) acquireLock(lockFile string) error {
	maxRetries := 30
	retryDelay := 100 * time.Millisecond

	for i := 0; i < maxRetries; i++ {
		// 尝试创建锁文件
		file, err := os.OpenFile(lockFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err == nil {
			// 成功创建锁文件，写入进程信息
			processInfo := fmt.Sprintf("%d:%d", os.Getpid(), time.Now().Unix())
			_, writeErr := file.WriteString(processInfo)
			file.Close()

			if writeErr != nil {
				os.Remove(lockFile) // 清理无效锁文件
				return fmt.Errorf("failed to write lock info: %v", writeErr)
			}

			flm.logger.Debug("Lock acquired successfully", "lockFile", lockFile, "pid", os.Getpid())
			return nil
		}

		// 检查是否为过期锁
		if flm.isLockStale(lockFile) {
			flm.logger.Warn("Removing stale lock file", "lockFile", lockFile)
			if removeErr := os.Remove(lockFile); removeErr != nil {
				flm.logger.Error("Failed to remove stale lock", "lockFile", lockFile, "error", removeErr)
			}
			continue // 重试获取锁
		}

		// 等待后重试
		time.Sleep(retryDelay)
		if i%10 == 9 { // 每1秒记录一次等待日志
			flm.logger.Debug("Waiting for lock", "lockFile", lockFile, "attempt", i+1)
		}
	}

	return fmt.Errorf("failed to acquire lock after %d attempts", maxRetries)
}

// releaseLock 释放文件锁
func (flm *DefaultFileLockManager) releaseLock(lockFile string) error {
	// 验证锁文件是否属于当前进程
	if !flm.isLockOwnedByCurrentProcess(lockFile) {
		return fmt.Errorf("lock file not owned by current process: %s", lockFile)
	}

	err := os.Remove(lockFile)
	if err != nil {
		return fmt.Errorf("failed to remove lock file: %v", err)
	}

	flm.logger.Debug("Lock released successfully", "lockFile", lockFile, "pid", os.Getpid())
	return nil
}

// isLockStale 检查锁文件是否过期
func (flm *DefaultFileLockManager) isLockStale(lockFile string) bool {
	info, err := os.Stat(lockFile)
	if err != nil {
		return false // 文件不存在或无法读取
	}

	// 检查文件修改时间
	if time.Since(info.ModTime()) > flm.staleLockTimeout {
		return true
	}

	// 检查进程是否仍存在
	content, err := os.ReadFile(lockFile)
	if err != nil {
		return true // 无法读取锁文件内容，视为过期
	}

	parts := strings.Split(strings.TrimSpace(string(content)), ":")
	if len(parts) != 2 {
		return true // 锁文件格式无效
	}

	pid, err := strconv.Atoi(parts[0])
	if err != nil {
		return true // 无效的进程ID
	}

	return !flm.isProcessRunning(pid)
}

// isLockOwnedByCurrentProcess 检查锁文件是否属于当前进程
func (flm *DefaultFileLockManager) isLockOwnedByCurrentProcess(lockFile string) bool {
	content, err := os.ReadFile(lockFile)
	if err != nil {
		return false
	}

	parts := strings.Split(strings.TrimSpace(string(content)), ":")
	if len(parts) != 2 {
		return false
	}

	pid, err := strconv.Atoi(parts[0])
	if err != nil {
		return false
	}

	return pid == os.Getpid()
}

// isProcessRunning 检查指定PID的进程是否仍在运行
func (flm *DefaultFileLockManager) isProcessRunning(pid int) bool {
	running, err := isProcessRunning(pid)
	if err != nil {
		flm.logger.Warn("Unexpected error checking process", "pid", pid, "error", err)
	}

	return running
}

// DefaultLogger 默认日志实现
type DefaultLogger struct {
	verboseEnabled bool
}

// NewDefaultLogger 创建默认日志实例
func NewDefaultLogger(verbose bool) Logger {
	return &DefaultLogger{
		verboseEnabled: verbose,
	}
}

func (l *DefaultLogger) Info(msg string, args ...any) {
	if l.verboseEnabled {
		fmt.Printf("INFO: %s %v\n", msg, args)
	}
}

func (l *DefaultLogger) Warn(msg string, args ...any) {
	if l.verboseEnabled {
		fmt.Printf("WARN: %s %v\n", msg, args)
	}
}

func (l *DefaultLogger) Error(msg string, args ...any) {
	fmt.Printf("ERROR: %s %v\n", msg, args)
}

func (l *DefaultLogger) Debug(msg string, args ...any) {
	if l.verboseEnabled {
		fmt.Printf("DEBUG: %s %v\n", msg, args)
	}
}
