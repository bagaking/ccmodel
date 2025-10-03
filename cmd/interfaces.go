package cmd

import (
	"time"
)

// == 存储层抽象接口 ==

// ConfigStorage 配置文件存储接口，遵循单一职责原则
type ConfigStorage interface {
	// LoadModel 加载指定模型配置
	LoadModel(modelName string) ([]byte, error)

	// SaveActiveConfig 保存活动配置
	SaveActiveConfig(data []byte) error

	// BackupActiveConfig 备份当前活动配置
	BackupActiveConfig() error

	// GetActiveModel 获取当前活动模型名称
	GetActiveModel() (string, error)

	// ListAvailableModels 列出所有可用模型
	ListAvailableModels() ([]string, error)
}

// HistoryStorage 历史记录存储接口
type HistoryStorage interface {
	// RecordEvent 记录事件
	RecordEvent(entry HistoryEntry) error

	// GetRecentEvents 获取最近的事件
	GetRecentEvents(timeWindow time.Duration) ([]HistoryEntry, error)

	// CleanupOldEvents 清理旧事件
	CleanupOldEvents(retentionDays int) error
}

// QuotaStorage Quota专用存储接口
type QuotaStorage interface {
	HistoryStorage

	// GetRecentQuota 获取最近的quota数据
	GetRecentQuota(modelName string, timeWindow time.Duration) (*QuotaInfo, *QuotaHistoryEntry, error)

	// RecordQuota 记录quota数据
	RecordQuota(modelName string, quotaInfo *QuotaInfo) error
}

// SwitchStorage 模型切换专用存储接口
type SwitchStorage interface {
	HistoryStorage

	// RecordSwitch 记录模型切换
	RecordSwitch(from, to string, duration time.Duration, err error) error

	// GetRecentSwitches 获取最近的切换记录
	GetRecentSwitches(timeWindow time.Duration) ([]*SwitchHistoryEntry, error)
}

// == 基础设施抽象接口 ==

// FileLockManager 文件锁管理接口
type FileLockManager interface {
	// WithLock 使用写锁执行操作
	WithLock(lockFile string, operation func() error) error

	// WithReadLock 使用读锁执行操作
	WithReadLock(lockFile string, operation func() error) error
}

// Logger 统一日志接口
type Logger interface {
	// Info 信息日志
	Info(msg string, args ...any)

	// Warn 警告日志
	Warn(msg string, args ...any)

	// Error 错误日志
	Error(msg string, args ...any)

	// Debug 调试日志
	Debug(msg string, args ...any)
}

// == 业务层抽象接口 ==

// ModelSwitcher 模型切换器接口
type ModelSwitcher interface {
	// Switch 切换模型
	Switch(fromModel, toModel string) error

	// GetCurrentModel 获取当前模型
	GetCurrentModel() (string, error)
}

// QuotaMonitor Quota监控器接口
type QuotaMonitor interface {
	// GetQuota 获取quota信息（支持协商机制）
	GetQuota(modelName string, timeWindow time.Duration) (*QuotaInfo, error)

	// GetQuotaWithSource 获取quota和来源信息
	GetQuotaWithSource(modelName string, timeWindow time.Duration) (*QuotaInfo, int, bool, error)
}

// UIRenderer UI渲染接口
type UIRenderer interface {
	// ShowSuccess 显示成功消息
	ShowSuccess(title, message string)

	// ShowError 显示错误消息
	ShowError(title, message string)

	// ShowSpinner 显示进度指示器
	ShowSpinner(message string) SpinnerController

	// ShowInfo 显示信息
	ShowInfo(message string)
}

// SpinnerController 进度指示器控制接口
type SpinnerController interface {
	// Success 显示成功并停止
	Success(message string)

	// Error 显示错误并停止
	Error(message string)

	// Stop 停止指示器
	Stop()
}

// == 通用数据接口 ==

// HistoryEntry 通用历史记录条目接口
type HistoryEntry interface {
	// GetTimestamp 获取时间戳
	GetTimestamp() time.Time

	// GetEventType 获取事件类型
	GetEventType() string

	// GetProcessID 获取进程ID
	GetProcessID() int

	// ToJSON 转换为JSON
	ToJSON() ([]byte, error)
}

// CollaborationConfig 协商机制配置接口
type CollaborationConfig interface {
	// GetTimeWindow 获取协商时间窗口
	GetTimeWindow() time.Duration

	// IsWithinWindow 检查是否在时间窗口内
	IsWithinWindow(timestamp time.Time) bool
}
