package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// SwitchHistoryEntry 表示一次模型切换记录
type SwitchHistoryEntry struct {
	Timestamp    time.Time `json:"timestamp"`
	FromModel    string    `json:"from_model"`
	ToModel      string    `json:"to_model"`
	EventType    string    `json:"event_type"` // "switch", "error"
	ErrorMessage string    `json:"error_message,omitempty"`
	ProcessID    int       `json:"process_id"`
	ProcessName  string    `json:"process_name,omitempty"`
	Duration     int64     `json:"duration_ms,omitempty"` // 切换耗时(毫秒)
}

// SwitchHistoryManager 管理模型切换历史记录，实现HistoryManager接口
type SwitchHistoryManager struct {
	historyDir string
	mutex      sync.RWMutex
}

// NewSwitchHistoryManager 创建新的切换历史管理器
func NewSwitchHistoryManager() *SwitchHistoryManager {
	claudeDir := filepath.Join(os.Getenv("HOME"), ".claude")
	historyDir := filepath.Join(claudeDir, "ccmodel", "switch_history")

	return &SwitchHistoryManager{
		historyDir: historyDir,
	}
}

// Initialize 实现HistoryManager接口，创建历史记录目录结构
func (sh *SwitchHistoryManager) Initialize() error {
	sh.mutex.Lock()
	defer sh.mutex.Unlock()

	if err := os.MkdirAll(sh.historyDir, 0755); err != nil {
		return fmt.Errorf("failed to create switch history directory: %v", err)
	}

	return nil
}

// GetHistoryPath 实现HistoryManager接口，返回历史记录目录路径
func (sh *SwitchHistoryManager) GetHistoryPath() string {
	return sh.historyDir
}

// CleanupOldLogs 实现HistoryManager接口，清理超过指定天数的历史记录
func (sh *SwitchHistoryManager) CleanupOldLogs(retentionDays int) error {
	sh.mutex.RLock()
	defer sh.mutex.RUnlock()

	if retentionDays <= 0 {
		return nil // No cleanup needed
	}

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)

	files, err := os.ReadDir(sh.historyDir)
	if err != nil {
		return fmt.Errorf("failed to read switch history directory: %v", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Parse date from filename (YYYY-MM-DD.jsonl)
		name := file.Name()
		if len(name) < 10 || filepath.Ext(name) != ".jsonl" {
			continue
		}

		dateStr := name[:10] // First 10 characters should be YYYY-MM-DD
		fileDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue // Skip files with invalid date format
		}

		if fileDate.Before(cutoffTime) {
			fullPath := filepath.Join(sh.historyDir, name)
			if err := os.Remove(fullPath); err != nil {
				// Log the error but don't fail the entire cleanup
				if verbose {
					fmt.Printf("Warning: Failed to remove old switch log file %s: %v\n", fullPath, err)
				}
			}
		}
	}

	return nil
}

// RecordSwitch 记录模型切换事件
func (sh *SwitchHistoryManager) RecordSwitch(fromModel, toModel string, duration time.Duration, err error) error {
	sh.mutex.Lock()
	defer sh.mutex.Unlock()

	now := time.Now()

	// 创建历史记录条目
	entry := &SwitchHistoryEntry{
		Timestamp:   now,
		FromModel:   fromModel,
		ToModel:     toModel,
		ProcessID:   os.Getpid(),
		ProcessName: "ccmodel",
		Duration:    duration.Milliseconds(),
	}

	if err != nil {
		entry.EventType = "error"
		entry.ErrorMessage = err.Error()
	} else {
		entry.EventType = "switch"
	}

	// 写入日志文件
	return sh.writeToLogFile(entry)
}

// writeToLogFile 将条目写入适当的日志文件
func (sh *SwitchHistoryManager) writeToLogFile(entry *SwitchHistoryEntry) error {
	// 基于日期生成文件名: YYYY-MM-DD.jsonl
	dateStr := entry.Timestamp.Format("2006-01-02")
	filename := filepath.Join(sh.historyDir, fmt.Sprintf("%s.jsonl", dateStr))
	lockFile := filename + ".lock"

	// 转换为JSON
	jsonData, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal switch entry: %v", err)
	}

	// 使用文件锁确保并发安全
	quotaHistoryManager := getQuotaHistoryManager()
	return quotaHistoryManager.withFileLock(lockFile, func() error {
		// 追加到文件（如果不存在则创建）
		file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open switch log file: %v", err)
		}
		defer file.Close()

		// 写入JSON行
		if _, err := file.WriteString(string(jsonData) + "\n"); err != nil {
			return fmt.Errorf("failed to write switch entry: %v", err)
		}

		return nil
	})
}

// GetRecentSwitches 获取最近的切换记录
func (sh *SwitchHistoryManager) GetRecentSwitches(timeWindow time.Duration) ([]*SwitchHistoryEntry, error) {
	sh.mutex.RLock()
	defer sh.mutex.RUnlock()

	cutoffTime := time.Now().Add(-timeWindow)

	// 读取今天的日志文件
	dateStr := time.Now().Format("2006-01-02")
	filename := filepath.Join(sh.historyDir, fmt.Sprintf("%s.jsonl", dateStr))
	lockFile := filename + ".lock"

	var entries []*SwitchHistoryEntry

	// 使用读锁进行读取
	quotaHistoryManager := getQuotaHistoryManager()
	err := quotaHistoryManager.withFileReadLock(lockFile, func() error {
		content, err := os.ReadFile(filename)
		if err != nil {
			if os.IsNotExist(err) {
				return nil // No log file exists yet
			}
			return err
		}

		lines := strings.Split(strings.TrimSpace(string(content)), "\n")

		// 从最后一行开始向前查找
		for i := len(lines) - 1; i >= 0; i-- {
			line := strings.TrimSpace(lines[i])
			if line == "" {
				continue
			}

			var entry SwitchHistoryEntry
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				continue // Skip malformed entries
			}

			// 检查是否在时间窗口内
			if entry.Timestamp.After(cutoffTime) {
				entries = append([]*SwitchHistoryEntry{&entry}, entries...) // Prepend for chronological order
			} else {
				break // Since we're going backwards, older entries won't match
			}
		}

		return nil
	})

	return entries, err
}

// Global switch history manager instance
var switchHistoryManager *SwitchHistoryManager
var switchHistoryOnce sync.Once

// getSwitchHistoryManager 返回全局切换历史管理器实例
func getSwitchHistoryManager() *SwitchHistoryManager {
	switchHistoryOnce.Do(func() {
		switchHistoryManager = NewSwitchHistoryManager()
		if err := switchHistoryManager.Initialize(); err != nil {
			if verbose {
				fmt.Printf("Warning: Failed to initialize switch history: %v\n", err)
			}
		}

		// 自动清理：删除超过30天的日志
		if err := switchHistoryManager.CleanupOldLogs(30); err != nil {
			if verbose {
				fmt.Printf("Warning: Failed to cleanup old switch logs: %v\n", err)
			}
		}
	})
	return switchHistoryManager
}
