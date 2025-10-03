package cmd

import (
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Container 依赖注入容器，管理所有服务的生命周期
type Container struct {
	// 基础设施层
	logger      Logger
	lockManager FileLockManager

	// 存储层
	configStorage ConfigStorage
	quotaStorage  QuotaStorage
	switchStorage SwitchStorage

	// 业务层
	modelSwitcher ModelSwitcher
	quotaMonitor  QuotaMonitor

	// UI层
	uiRenderer UIRenderer

	// 配置
	configDir string
	verbose   bool

	// 初始化控制
	once sync.Once
}

// NewContainer 创建新的依赖注入容器
func NewContainer(configDir string, verbose bool) *Container {
	return &Container{
		configDir: configDir,
		verbose:   verbose,
	}
}

// Initialize 初始化所有依赖，确保按正确顺序创建
func (c *Container) Initialize() error {
	var initError error
	c.once.Do(func() {
		initError = c.initializeDependencies()
	})
	return initError
}

// initializeDependencies 初始化所有依赖关系
func (c *Container) initializeDependencies() error {
	// 1. 基础设施层
	c.logger = NewDefaultLogger(c.verbose)
	c.lockManager = NewFileLockManager(c.logger)
	c.uiRenderer = NewCmduxUIRenderer(c.verbose)

	// 2. 存储层
	c.configStorage = NewFileConfigStorage(c.configDir, c.lockManager, c.logger)
	c.quotaStorage = c.createQuotaStorage()
	c.switchStorage = c.createSwitchStorage()

	// 3. 业务层
	c.modelSwitcher = NewModelSwitcher(
		c.configStorage,
		c.switchStorage,
		c.uiRenderer,
		c.logger,
	)
	c.quotaMonitor = c.createQuotaMonitor()

	return nil
}

// createQuotaStorage 创建quota存储实例
func (c *Container) createQuotaStorage() QuotaStorage {
	// 重用现有的QuotaHistoryManager，但通过接口适配器模式使其符合新接口
	manager := &QuotaHistoryManager{
		historyDir:        filepath.Join(c.configDir, "ccmodel", "quota_history"),
		heartbeatInterval: 3 * time.Minute,
	}

	// 初始化存储目录
	if err := manager.Initialize(); err != nil {
		c.logger.Warn("Failed to initialize quota storage", "error", err)
	}

	// 自动清理旧日志
	if err := manager.CleanupOldLogs(30); err != nil {
		c.logger.Warn("Failed to cleanup old quota logs", "error", err)
	}

	return &QuotaStorageAdapter{
		manager:     manager,
		lockManager: c.lockManager,
		logger:      c.logger,
	}
}

// createSwitchStorage 创建switch存储实例
func (c *Container) createSwitchStorage() SwitchStorage {
	manager := &SwitchHistoryManager{
		historyDir: filepath.Join(c.configDir, "ccmodel", "switch_history"),
	}

	// 初始化存储目录
	if err := manager.Initialize(); err != nil {
		c.logger.Warn("Failed to initialize switch storage", "error", err)
	}

	// 自动清理旧日志
	if err := manager.CleanupOldLogs(30); err != nil {
		c.logger.Warn("Failed to cleanup old switch logs", "error", err)
	}

	return &SwitchStorageAdapter{
		manager:     manager,
		lockManager: c.lockManager,
		logger:      c.logger,
	}
}

// createQuotaMonitor 创建quota监控器实例
func (c *Container) createQuotaMonitor() QuotaMonitor {
	return &DefaultQuotaMonitor{
		quotaStorage: c.quotaStorage,
		logger:       c.logger,
	}
}

// == Getter 方法，提供依赖访问 ==

// GetModelSwitcher 获取模型切换器
func (c *Container) GetModelSwitcher() ModelSwitcher {
	c.Initialize() // 确保已初始化
	return c.modelSwitcher
}

// GetQuotaMonitor 获取quota监控器
func (c *Container) GetQuotaMonitor() QuotaMonitor {
	c.Initialize()
	return c.quotaMonitor
}

// GetConfigStorage 获取配置存储
func (c *Container) GetConfigStorage() ConfigStorage {
	c.Initialize()
	return c.configStorage
}

// GetUIRenderer 获取UI渲染器
func (c *Container) GetUIRenderer() UIRenderer {
	c.Initialize()
	return c.uiRenderer
}

// GetLogger 获取日志器
func (c *Container) GetLogger() Logger {
	c.Initialize()
	return c.logger
}

// GetLockManager 获取文件锁管理器
func (c *Container) GetLockManager() FileLockManager {
	c.Initialize()
	return c.lockManager
}

// == 适配器，将现有实现适配到新接口 ==

// QuotaStorageAdapter 将QuotaHistoryManager适配到QuotaStorage接口
type QuotaStorageAdapter struct {
	manager     *QuotaHistoryManager
	lockManager FileLockManager
	logger      Logger
}

func (qsa *QuotaStorageAdapter) RecordEvent(entry HistoryEntry) error {
	// 暂时的实现，可根据需要完善
	return nil
}

func (qsa *QuotaStorageAdapter) GetRecentEvents(timeWindow time.Duration) ([]HistoryEntry, error) {
	// 暂时的实现，可根据需要完善
	return nil, nil
}

func (qsa *QuotaStorageAdapter) CleanupOldEvents(retentionDays int) error {
	return qsa.manager.CleanupOldLogs(retentionDays)
}

func (qsa *QuotaStorageAdapter) GetRecentQuota(modelName string, timeWindow time.Duration) (*QuotaInfo, *QuotaHistoryEntry, error) {
	return qsa.manager.GetRecentQuotaFromHistory(modelName, timeWindow)
}

func (qsa *QuotaStorageAdapter) RecordQuota(modelName string, quotaInfo *QuotaInfo) error {
	return qsa.manager.RecordQuota(modelName, quotaInfo)
}

// SwitchStorageAdapter 将SwitchHistoryManager适配到SwitchStorage接口
type SwitchStorageAdapter struct {
	manager     *SwitchHistoryManager
	lockManager FileLockManager
	logger      Logger
}

func (ssa *SwitchStorageAdapter) RecordEvent(entry HistoryEntry) error {
	// 暂时的实现，可根据需要完善
	return nil
}

func (ssa *SwitchStorageAdapter) GetRecentEvents(timeWindow time.Duration) ([]HistoryEntry, error) {
	// 暂时的实现，可根据需要完善
	return nil, nil
}

func (ssa *SwitchStorageAdapter) CleanupOldEvents(retentionDays int) error {
	return ssa.manager.CleanupOldLogs(retentionDays)
}

func (ssa *SwitchStorageAdapter) RecordSwitch(from, to string, duration time.Duration, err error) error {
	return ssa.manager.RecordSwitch(from, to, duration, err)
}

func (ssa *SwitchStorageAdapter) GetRecentSwitches(timeWindow time.Duration) ([]*SwitchHistoryEntry, error) {
	return ssa.manager.GetRecentSwitches(timeWindow)
}

// DefaultQuotaMonitor quota监控器的默认实现
type DefaultQuotaMonitor struct {
	quotaStorage QuotaStorage
	logger       Logger
}

func (dqm *DefaultQuotaMonitor) GetQuota(modelName string, timeWindow time.Duration) (*QuotaInfo, error) {
	quotaInfo, _, err := dqm.quotaStorage.GetRecentQuota(modelName, timeWindow)
	return quotaInfo, err
}

func (dqm *DefaultQuotaMonitor) GetQuotaWithSource(modelName string, timeWindow time.Duration) (*QuotaInfo, int, bool, error) {
	quotaInfo, sourceEntry, err := dqm.quotaStorage.GetRecentQuota(modelName, timeWindow)
	if err != nil || quotaInfo == nil {
		return nil, 0, false, err
	}

	return quotaInfo, sourceEntry.ProcessID, true, nil
}

// == 全局容器实例管理 ==

var (
	globalContainer     *Container
	globalContainerOnce sync.Once
)

// GetGlobalContainer 获取全局容器实例（单例模式）
func GetGlobalContainer() *Container {
	globalContainerOnce.Do(func() {
		// 使用默认配置目录
		homeDir, _ := os.UserHomeDir()
		configDir := filepath.Join(homeDir, ".claude")
		globalContainer = NewContainer(configDir, verbose)
	})
	return globalContainer
}

// SetGlobalContainer 设置全局容器（主要用于测试）
func SetGlobalContainer(container *Container) {
	globalContainer = container
}
