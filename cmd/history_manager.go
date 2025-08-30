package cmd

import (
	"time"
)

// HistoryManager 定义历史管理器的统一接口，遵循依赖倒置原则
type HistoryManager interface {
	// Initialize 初始化历史记录系统
	Initialize() error
	
	// GetHistoryPath 返回历史记录目录路径
	GetHistoryPath() string
	
	// CleanupOldLogs 清理超过指定天数的历史记录
	CleanupOldLogs(retentionDays int) error
}

// CollaborationTimeWindow 协商机制时间窗口配置
type CollaborationTimeWindow struct {
	// DetectionInterval 检测周期，用于计算协商时间窗口
	DetectionInterval time.Duration
	
	// WindowMultiplier 窗口倍数，实际窗口 = DetectionInterval * WindowMultiplier
	WindowMultiplier float64
}

// CalculateWindow 计算实际的协商时间窗口
func (ctw *CollaborationTimeWindow) CalculateWindow() time.Duration {
	if ctw.WindowMultiplier <= 0 {
		ctw.WindowMultiplier = 1.0 // 默认倍数
	}
	return time.Duration(float64(ctw.DetectionInterval) * ctw.WindowMultiplier)
}

// NewCollaborationTimeWindow 创建协商时间窗口配置
func NewCollaborationTimeWindow(detectionInterval time.Duration) *CollaborationTimeWindow {
	return &CollaborationTimeWindow{
		DetectionInterval: detectionInterval,
		WindowMultiplier:  1.0, // 默认时间窗口等于检测周期
	}
}

// NewDefaultCollaborationTimeWindow 创建默认的协商时间窗口（30秒）
// 用于一般的quota查询，提供合理的协商窗口避免重复API调用
func NewDefaultCollaborationTimeWindow() *CollaborationTimeWindow {
	return &CollaborationTimeWindow{
		DetectionInterval: 30 * time.Second, // 30秒默认窗口
		WindowMultiplier:  1.0,
	}
}

// ValidateMinimumInterval 验证并调整检测间隔，确保不小于10秒
func ValidateMinimumInterval(interval time.Duration) time.Duration {
	const MinInterval = 10 * time.Second
	if interval < MinInterval {
		return MinInterval
	}
	return interval
}