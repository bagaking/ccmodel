package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestDefaultFileLockManager_WithLock_Success 测试正常的锁获取和释放
func TestDefaultFileLockManager_WithLock_Success(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "lock_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建锁管理器
	mockLogger := &MockLogger{}
	lockManager := NewFileLockManager(mockLogger)
	lockFile := filepath.Join(tempDir, "test.lock")

	// 测试操作
	operationExecuted := false

	// 使用锁执行操作
	err = lockManager.WithLock(lockFile, func() error {
		operationExecuted = true

		// 验证锁文件存在
		if _, statErr := os.Stat(lockFile); os.IsNotExist(statErr) {
			t.Error("Lock file should exist during operation")
		}

		return nil
	})

	// 验证结果
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !operationExecuted {
		t.Error("Operation should have been executed")
	}

	// 验证锁文件被清理
	if _, statErr := os.Stat(lockFile); !os.IsNotExist(statErr) {
		t.Error("Lock file should be cleaned up after operation")
	}
}

// TestDefaultFileLockManager_WithLock_OperationError 测试操作失败时的处理
func TestDefaultFileLockManager_WithLock_OperationError(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lock_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mockLogger := &MockLogger{}
	lockManager := NewFileLockManager(mockLogger)
	lockFile := filepath.Join(tempDir, "test.lock")

	expectedError := fmt.Errorf("operation failed")

	// 使用锁执行失败的操作
	err = lockManager.WithLock(lockFile, func() error {
		return expectedError
	})

	// 验证错误传播
	if err != expectedError {
		t.Errorf("Expected %v, got %v", expectedError, err)
	}

	// 验证锁文件依然被清理
	if _, statErr := os.Stat(lockFile); !os.IsNotExist(statErr) {
		t.Error("Lock file should be cleaned up even after operation error")
	}
}

// TestDefaultFileLockManager_ConcurrentLocks 测试并发锁竞争
func TestDefaultFileLockManager_ConcurrentLocks(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lock_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mockLogger := &MockLogger{}
	lockManager := NewFileLockManager(mockLogger)
	lockFile := filepath.Join(tempDir, "concurrent.lock")

	// 并发执行计数器
	var executionOrder []int
	var mu sync.Mutex

	// 启动多个goroutine争抢同一个锁
	const numGoroutines = 5
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			err := lockManager.WithLock(lockFile, func() error {
				mu.Lock()
				executionOrder = append(executionOrder, id)
				mu.Unlock()

				// 模拟一些工作
				time.Sleep(10 * time.Millisecond)
				return nil
			})

			if err != nil {
				t.Errorf("Goroutine %d failed with error: %v", id, err)
			}
		}(i)
	}

	wg.Wait()

	// 验证所有goroutine都执行了
	if len(executionOrder) != numGoroutines {
		t.Errorf("Expected %d executions, got %d", numGoroutines, len(executionOrder))
	}

	// 验证锁文件最终被清理
	if _, statErr := os.Stat(lockFile); !os.IsNotExist(statErr) {
		t.Error("Lock file should be cleaned up after all operations")
	}
}

// TestDefaultFileLockManager_StaleLockDetection 测试过期锁检测
func TestDefaultFileLockManager_StaleLockDetection(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lock_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mockLogger := &MockLogger{}

	// 创建一个修改过期时间的锁管理器用于测试
	lockManager := &DefaultFileLockManager{
		staleLockTimeout: 50 * time.Millisecond, // 很短的超时用于测试
		logger:           mockLogger,
	}

	lockFile := filepath.Join(tempDir, "stale.lock")

	// 手动创建一个过期的锁文件
	lockContent := fmt.Sprintf("%d:%d", 99999, time.Now().Unix()) // 使用不存在的PID
	err = os.WriteFile(lockFile, []byte(lockContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create stale lock file: %v", err)
	}

	// 等待锁文件过期
	time.Sleep(100 * time.Millisecond)

	// 尝试获取锁，应该能够清理过期锁
	operationExecuted := false
	err = lockManager.WithLock(lockFile, func() error {
		operationExecuted = true
		return nil
	})

	// 验证成功获取锁并执行操作
	if err != nil {
		t.Errorf("Expected no error after cleaning stale lock, got %v", err)
	}

	if !operationExecuted {
		t.Error("Operation should have been executed after cleaning stale lock")
	}

	// 验证警告日志被记录
	hasStaleWarning := false
	for _, warn := range mockLogger.warnCalls {
		if contains(warn, "Removing stale lock file") {
			hasStaleWarning = true
			break
		}
	}
	if !hasStaleWarning {
		t.Error("Expected warning about removing stale lock file")
	}
}

// TestDefaultFileLockManager_WithReadLock 测试读锁功能
func TestDefaultFileLockManager_WithReadLock(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lock_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mockLogger := &MockLogger{}
	lockManager := NewFileLockManager(mockLogger)
	lockFile := filepath.Join(tempDir, "read.lock")

	operationExecuted := false

	// 使用读锁执行操作
	err = lockManager.WithReadLock(lockFile, func() error {
		operationExecuted = true
		return nil
	})

	// 验证结果（当前实现中读锁与写锁相同）
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !operationExecuted {
		t.Error("Operation should have been executed")
	}
}

// TestDefaultFileLockManager_LockContentFormat 测试锁文件内容格式
func TestDefaultFileLockManager_LockContentFormat(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lock_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mockLogger := &MockLogger{}
	lockManager := NewFileLockManager(mockLogger)
	lockFile := filepath.Join(tempDir, "format.lock")

	var lockContent []byte

	// 使用锁并读取锁文件内容
	err = lockManager.WithLock(lockFile, func() error {
		var readErr error
		lockContent, readErr = os.ReadFile(lockFile)
		return readErr
	})

	if err != nil {
		t.Fatalf("Failed to execute lock operation: %v", err)
	}

	// 验证锁文件内容格式：PID:timestamp
	lockStr := string(lockContent)
	if !contains(lockStr, ":") {
		t.Errorf("Lock file content should contain ':' separator, got: %s", lockStr)
	}

	// 进一步验证格式
	parts := splitString(lockStr, ":")
	if len(parts) != 2 {
		t.Errorf("Lock file content should have exactly 2 parts, got %d: %s", len(parts), lockStr)
	}
}

// TestDefaultFileLockManager_MaxRetriesExceeded 测试超过最大重试次数
func TestDefaultFileLockManager_MaxRetriesExceeded(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lock_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mockLogger := &MockLogger{}
	lockManager := NewFileLockManager(mockLogger)
	lockFile := filepath.Join(tempDir, "busy.lock")

	// 创建一个持久的锁文件（当前进程拥有）
	firstLockManager := NewFileLockManager(mockLogger)

	// 第一个锁管理器获取锁并保持
	lockAcquired := make(chan bool)
	lockHold := make(chan bool)

	go func() {
		err := firstLockManager.WithLock(lockFile, func() error {
			lockAcquired <- true
			<-lockHold // 等待信号释放锁
			return nil
		})
		if err != nil {
			t.Errorf("First lock failed: %v", err)
		}
	}()

	// 等待第一个锁获取成功
	<-lockAcquired

	// 第二个锁管理器尝试获取锁（应该失败）
	start := time.Now()
	err = lockManager.WithLock(lockFile, func() error {
		t.Error("Second lock should not succeed")
		return nil
	})
	duration := time.Since(start)

	// 释放第一个锁
	lockHold <- true

	// 验证第二个锁获取失败
	if err == nil {
		t.Error("Expected error when max retries exceeded, got nil")
	}

	// 验证至少等待了一定时间（说明进行了重试）
	if duration < 100*time.Millisecond {
		t.Errorf("Expected to wait at least 100ms for retries, waited %v", duration)
	}
}

// == 辅助函数 ==

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func splitString(s, sep string) []string {
	if s == "" {
		return nil
	}

	var result []string
	start := 0

	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}

	// 添加最后一部分
	if start < len(s) {
		result = append(result, s[start:])
	}

	return result
}
