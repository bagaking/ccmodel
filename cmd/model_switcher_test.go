package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// == Mock 实现，用于测试 ==

// MockConfigStorage 配置存储的Mock实现
type MockConfigStorage struct {
	availableModels []string
	activeModel     string
	loadModelError  error
	saveConfigError error
	backupError     error
	getActiveError  error
}

func (m *MockConfigStorage) LoadModel(modelName string) ([]byte, error) {
	if m.loadModelError != nil {
		return nil, m.loadModelError
	}

	// 检查模型是否存在
	for _, model := range m.availableModels {
		if model == modelName {
			return []byte(fmt.Sprintf(`{"model":"%s"}`, modelName)), nil
		}
	}
	return nil, fmt.Errorf("model '%s' not found", modelName)
}

func (m *MockConfigStorage) SaveActiveConfig(data []byte) error {
	return m.saveConfigError
}

func (m *MockConfigStorage) BackupActiveConfig() error {
	return m.backupError
}

func (m *MockConfigStorage) GetActiveModel() (string, error) {
	if m.getActiveError != nil {
		return "", m.getActiveError
	}
	return m.activeModel, nil
}

func (m *MockConfigStorage) ListAvailableModels() ([]string, error) {
	return m.availableModels, nil
}

// MockSwitchStorage 切换存储的Mock实现
type MockSwitchStorage struct {
	recordSwitchError error
	switchRecords     []SwitchRecord
}

type SwitchRecord struct {
	From     string
	To       string
	Duration time.Duration
	Error    error
}

func (m *MockSwitchStorage) RecordEvent(entry HistoryEntry) error {
	return nil
}

func (m *MockSwitchStorage) GetRecentEvents(timeWindow time.Duration) ([]HistoryEntry, error) {
	return nil, nil
}

func (m *MockSwitchStorage) CleanupOldEvents(retentionDays int) error {
	return nil
}

func (m *MockSwitchStorage) RecordSwitch(from, to string, duration time.Duration, err error) error {
	if m.recordSwitchError != nil {
		return m.recordSwitchError
	}

	m.switchRecords = append(m.switchRecords, SwitchRecord{
		From:     from,
		To:       to,
		Duration: duration,
		Error:    err,
	})
	return nil
}

func (m *MockSwitchStorage) GetRecentSwitches(timeWindow time.Duration) ([]*SwitchHistoryEntry, error) {
	return nil, nil
}

// MockUIRenderer UI渲染器的Mock实现
type MockUIRenderer struct {
	successCalls []string
	errorCalls   []string
	infoCalls    []string
}

func (m *MockUIRenderer) ShowSuccess(title, message string) {
	m.successCalls = append(m.successCalls, fmt.Sprintf("%s: %s", title, message))
}

func (m *MockUIRenderer) ShowError(title, message string) {
	m.errorCalls = append(m.errorCalls, fmt.Sprintf("%s: %s", title, message))
}

func (m *MockUIRenderer) ShowSpinner(message string) SpinnerController {
	return &MockSpinnerController{}
}

func (m *MockUIRenderer) ShowInfo(message string) {
	m.infoCalls = append(m.infoCalls, message)
}

// MockSpinnerController 进度控制器的Mock实现
type MockSpinnerController struct {
	successMessage string
	errorMessage   string
}

func (m *MockSpinnerController) Success(message string) {
	m.successMessage = message
}

func (m *MockSpinnerController) Error(message string) {
	m.errorMessage = message
}

func (m *MockSpinnerController) Stop() {}

// MockLogger 日志器的Mock实现
type MockLogger struct {
	infoCalls  []string
	warnCalls  []string
	errorCalls []string
}

func (m *MockLogger) Info(msg string, args ...any) {
	m.infoCalls = append(m.infoCalls, fmt.Sprintf("%s %v", msg, args))
}

func (m *MockLogger) Warn(msg string, args ...any) {
	m.warnCalls = append(m.warnCalls, fmt.Sprintf("%s %v", msg, args))
}

func (m *MockLogger) Error(msg string, args ...any) {
	m.errorCalls = append(m.errorCalls, fmt.Sprintf("%s %v", msg, args))
}

func (m *MockLogger) Debug(msg string, args ...any) {}

// == 测试用例 ==

// TestModelSwitcher_Switch_Success 测试成功切换场景
func TestModelSwitcher_Switch_Success(t *testing.T) {
	// 准备Mock对象
	mockConfig := &MockConfigStorage{
		availableModels: []string{"model1", "model2", "model3"},
		activeModel:     "model1",
	}
	mockSwitch := &MockSwitchStorage{}
	mockUI := &MockUIRenderer{}
	mockLogger := &MockLogger{}

	// 创建模型切换器
	switcher := NewModelSwitcher(mockConfig, mockSwitch, mockUI, mockLogger)

	// 执行切换
	err := switcher.Switch("model1", "model2")

	// 验证结果
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// 验证UI调用
	if len(mockUI.successCalls) != 1 {
		t.Errorf("Expected 1 success call, got %d", len(mockUI.successCalls))
	}

	// 验证历史记录
	if len(mockSwitch.switchRecords) != 1 {
		t.Errorf("Expected 1 switch record, got %d", len(mockSwitch.switchRecords))
	}

	record := mockSwitch.switchRecords[0]
	if record.From != "model1" || record.To != "model2" {
		t.Errorf("Expected switch from model1 to model2, got from %s to %s", record.From, record.To)
	}

	if record.Error != nil {
		t.Errorf("Expected no error in record, got %v", record.Error)
	}
}

// TestModelSwitcher_Switch_ModelNotFound 测试模型不存在的错误处理
func TestModelSwitcher_Switch_ModelNotFound(t *testing.T) {
	mockConfig := &MockConfigStorage{
		availableModels: []string{"model1", "model2"},
		activeModel:     "model1",
	}
	mockSwitch := &MockSwitchStorage{}
	mockUI := &MockUIRenderer{}
	mockLogger := &MockLogger{}

	switcher := NewModelSwitcher(mockConfig, mockSwitch, mockUI, mockLogger)

	// 尝试切换到不存在的模型
	err := switcher.Switch("model1", "nonexistent")

	// 验证返回错误
	if err == nil {
		t.Error("Expected error for nonexistent model, got nil")
	}

	// 验证UI显示错误
	if len(mockUI.errorCalls) != 1 {
		t.Errorf("Expected 1 error call, got %d", len(mockUI.errorCalls))
	}

	// 验证历史记录记录了失败
	if len(mockSwitch.switchRecords) != 1 {
		t.Errorf("Expected 1 switch record, got %d", len(mockSwitch.switchRecords))
	}

	record := mockSwitch.switchRecords[0]
	if record.Error == nil {
		t.Error("Expected error in switch record, got nil")
	}
}

// TestModelSwitcher_Switch_BackupFailure 测试备份失败的处理
func TestModelSwitcher_Switch_BackupFailure(t *testing.T) {
	mockConfig := &MockConfigStorage{
		availableModels: []string{"model1", "model2"},
		activeModel:     "model1",
		backupError:     fmt.Errorf("backup failed"),
	}
	mockSwitch := &MockSwitchStorage{}
	mockUI := &MockUIRenderer{}
	mockLogger := &MockLogger{}

	switcher := NewModelSwitcher(mockConfig, mockSwitch, mockUI, mockLogger)

	// 执行切换
	err := switcher.Switch("model1", "model2")

	// 验证返回错误
	if err == nil {
		t.Error("Expected error for backup failure, got nil")
	}

	// 验证日志记录了错误
	if len(mockLogger.errorCalls) == 0 {
		t.Error("Expected error log for backup failure, got none")
	}

	// 验证历史记录记录了失败
	record := mockSwitch.switchRecords[0]
	if record.Error == nil {
		t.Error("Expected error in switch record, got nil")
	}
}

// TestModelSwitcher_Switch_SaveConfigFailure 测试保存配置失败的处理
func TestModelSwitcher_Switch_SaveConfigFailure(t *testing.T) {
	mockConfig := &MockConfigStorage{
		availableModels: []string{"model1", "model2"},
		activeModel:     "model1",
		saveConfigError: fmt.Errorf("save failed"),
	}
	mockSwitch := &MockSwitchStorage{}
	mockUI := &MockUIRenderer{}
	mockLogger := &MockLogger{}

	switcher := NewModelSwitcher(mockConfig, mockSwitch, mockUI, mockLogger)

	// 执行切换
	err := switcher.Switch("model1", "model2")

	// 验证返回错误
	if err == nil {
		t.Error("Expected error for save failure, got nil")
	}

	// 验证历史记录记录了失败
	record := mockSwitch.switchRecords[0]
	if record.Error == nil {
		t.Error("Expected error in switch record, got nil")
	}
}

// TestModelSwitcher_GetCurrentModel 测试获取当前模型
func TestModelSwitcher_GetCurrentModel(t *testing.T) {
	mockConfig := &MockConfigStorage{
		activeModel: "current-model",
	}
	mockSwitch := &MockSwitchStorage{}
	mockUI := &MockUIRenderer{}
	mockLogger := &MockLogger{}

	switcher := NewModelSwitcher(mockConfig, mockSwitch, mockUI, mockLogger)

	// 获取当前模型
	currentModel, err := switcher.GetCurrentModel()

	// 验证结果
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if currentModel != "current-model" {
		t.Errorf("Expected 'current-model', got %s", currentModel)
	}
}

// TestSwitchCommand_Execute_SameModel 测试切换到相同模型
func TestSwitchCommand_Execute_SameModel(t *testing.T) {
	mockConfig := &MockConfigStorage{
		activeModel: "model1",
	}
	mockSwitch := &MockSwitchStorage{}
	mockUI := &MockUIRenderer{}
	mockLogger := &MockLogger{}

	switcher := NewModelSwitcher(mockConfig, mockSwitch, mockUI, mockLogger)
	command := NewSwitchCommand(switcher)

	// 尝试切换到相同模型
	err := command.Execute("model1")

	// 验证返回错误
	if err == nil {
		t.Error("Expected error for switching to same model, got nil")
	}

	// 验证没有记录切换历史
	if len(mockSwitch.switchRecords) != 0 {
		t.Errorf("Expected no switch records, got %d", len(mockSwitch.switchRecords))
	}
}

// TestSwitchCommand_Execute_Success 测试命令成功执行
func TestSwitchCommand_Execute_Success(t *testing.T) {
	mockConfig := &MockConfigStorage{
		availableModels: []string{"model1", "model2"},
		activeModel:     "model1",
	}
	mockSwitch := &MockSwitchStorage{}
	mockUI := &MockUIRenderer{}
	mockLogger := &MockLogger{}

	switcher := NewModelSwitcher(mockConfig, mockSwitch, mockUI, mockLogger)
	command := NewSwitchCommand(switcher)

	// 执行命令
	err := command.Execute("model2")

	// 验证成功
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// 验证切换被记录
	if len(mockSwitch.switchRecords) != 1 {
		t.Errorf("Expected 1 switch record, got %d", len(mockSwitch.switchRecords))
	}
}

func TestSwitchModel_BackupPreservesRawCurrentConfig(t *testing.T) {
	tempDir := setupSwitchModelTempHome(t)

	currentConfig := []byte("{invalid json with __cc that must stay in backup")
	if err := os.WriteFile(filepath.Join(tempDir, "settings.json"), currentConfig, 0644); err != nil {
		t.Fatalf("os.WriteFile(settings.json) error = %v, want nil", err)
	}

	modelConfig := []byte(`{"model":"target","__cc":{"quota":"removed"}}`)
	if err := os.WriteFile(filepath.Join(tempDir, "settings.target.json"), modelConfig, 0644); err != nil {
		t.Fatalf("os.WriteFile(settings.target.json) error = %v, want nil", err)
	}

	if err := switchModel("target"); err != nil {
		t.Fatalf("switchModel(%q) error = %v, want nil", "target", err)
	}

	backupDir := filepath.Join(tempDir, "ccmodel", "backups")
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("os.ReadDir(%q) error = %v, want nil", backupDir, err)
	}
	if len(entries) != 1 {
		t.Fatalf("os.ReadDir(%q) entries = %d, want 1", backupDir, len(entries))
	}
	assertMode(t, backupDir, userOnlyDirMode)

	backupFile := filepath.Join(backupDir, entries[0].Name())
	backupData, err := os.ReadFile(backupFile)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v, want nil", entries[0].Name(), err)
	}
	if string(backupData) != string(currentConfig) {
		t.Errorf("switchModel(%q) backup = %q, want %q", "target", backupData, currentConfig)
	}
	assertMode(t, backupFile, userOnlyFileMode)

	activeData, err := os.ReadFile(filepath.Join(tempDir, "settings.json"))
	if err != nil {
		t.Fatalf("os.ReadFile(settings.json) error = %v, want nil", err)
	}
	if strings.Contains(string(activeData), "__cc") {
		t.Errorf("switchModel(%q) active config = %q, want quota fields removed", "target", activeData)
	}
	assertMode(t, filepath.Join(tempDir, "settings.json"), userOnlyFileMode)
}

func TestSwitchModel_UsesTemporaryHomeAndStripsMacroFields(t *testing.T) {
	claudeDir := setupSwitchModelTempHome(t)
	homeDir := filepath.Dir(claudeDir)

	currentConfig := []byte(`{"model":"baseline","env":{"TOKEN":"old"}}`)
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), currentConfig, 0o644); err != nil {
		t.Fatalf("os.WriteFile(settings.json) error = %v, want nil", err)
	}
	candidateConfig := []byte(`{
  "model": "target",
  "env": {"ANTHROPIC_MODEL": "claude-sonnet"},
  "__cc": {"quota": "removed"},
  "__ccmodel": {"quota": "also removed"}
}`)
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.target.json"), candidateConfig, 0o644); err != nil {
		t.Fatalf("os.WriteFile(settings.target.json) error = %v, want nil", err)
	}

	if err := switchModel("target"); err != nil {
		t.Fatalf("switchModel(%q) error = %v, want nil", "target", err)
	}

	activePath := filepath.Join(claudeDir, "settings.json")
	activeData, err := os.ReadFile(activePath)
	if err != nil {
		t.Fatalf("os.ReadFile(settings.json) error = %v, want nil", err)
	}
	var active map[string]any
	if err := json.Unmarshal(activeData, &active); err != nil {
		t.Fatalf("json.Unmarshal(settings.json) error = %v, want nil", err)
	}
	if _, exists := active["__cc"]; exists {
		t.Errorf("settings.json contains __cc after switch: %s", activeData)
	}
	if _, exists := active["__ccmodel"]; exists {
		t.Errorf("settings.json contains __ccmodel after switch: %s", activeData)
	}
	if active["model"] != "target" {
		t.Errorf("settings.json model = %v, want target", active["model"])
	}
	assertMode(t, activePath, userOnlyFileMode)
	assertMode(t, claudeDir, userOnlyDirMode)

	backupDir := filepath.Join(claudeDir, "ccmodel", "backups")
	backupEntries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("os.ReadDir(%q) error = %v, want nil", backupDir, err)
	}
	if len(backupEntries) != 1 {
		t.Fatalf("os.ReadDir(%q) entries = %d, want 1", backupDir, len(backupEntries))
	}
	backupPath := filepath.Join(backupDir, backupEntries[0].Name())
	backupData, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v, want nil", backupPath, err)
	}
	if string(backupData) != string(currentConfig) {
		t.Errorf("backup data = %q, want original current config %q", backupData, currentConfig)
	}
	assertMode(t, backupDir, userOnlyDirMode)
	assertMode(t, backupPath, userOnlyFileMode)

	switchHistoryDir := filepath.Join(homeDir, ".claude", "ccmodel", "switch_history")
	assertMode(t, switchHistoryDir, userOnlyDirMode)
	assertOnlyEntryMode(t, switchHistoryDir, ".jsonl", userOnlyFileMode)
}

func TestSwitchModel_MissingModelDoesNotOverwriteCurrentConfig(t *testing.T) {
	claudeDir := setupSwitchModelTempHome(t)

	currentConfig := []byte(`{"model":"baseline","env":{"TOKEN":"old"}}`)
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), currentConfig, 0o644); err != nil {
		t.Fatalf("os.WriteFile(settings.json) error = %v, want nil", err)
	}

	if err := switchModel("missing"); err == nil {
		t.Fatalf("switchModel(%q) error = nil, want error", "missing")
	}

	activeData, err := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	if err != nil {
		t.Fatalf("os.ReadFile(settings.json) error = %v, want nil", err)
	}
	if string(activeData) != string(currentConfig) {
		t.Errorf("settings.json after failed switch = %q, want original %q", activeData, currentConfig)
	}

	backupDir := filepath.Join(claudeDir, "ccmodel", "backups")
	if _, err := os.Stat(backupDir); !os.IsNotExist(err) {
		t.Fatalf("os.Stat(%q) error = %v, want not exist", backupDir, err)
	}
	switchHistoryDir := filepath.Join(claudeDir, "ccmodel", "switch_history")
	assertMode(t, switchHistoryDir, userOnlyDirMode)
	assertOnlyEntryMode(t, switchHistoryDir, ".jsonl", userOnlyFileMode)
}

func setupSwitchModelTempHome(t *testing.T) string {
	t.Helper()

	tempHome := t.TempDir()
	claudeDir := filepath.Join(tempHome, ".claude")
	if err := os.Mkdir(claudeDir, 0o700); err != nil {
		t.Fatalf("os.Mkdir(%q) error = %v, want nil", claudeDir, err)
	}

	previousConfigDir := configDir
	previousApp := app
	previousVerbose := verbose
	previousSwitchHistoryManager := switchHistoryManager
	t.Cleanup(func() {
		configDir = previousConfigDir
		app = previousApp
		verbose = previousVerbose
		switchHistoryManager = previousSwitchHistoryManager
		resetSwitchHistoryOnceForTest()
	})

	t.Setenv("HOME", tempHome)
	configDir = ""
	app = nil
	verbose = false
	switchHistoryManager = nil
	resetSwitchHistoryOnceForTest()
	initConfig()

	if configDir != claudeDir {
		t.Fatalf("configDir = %q, want temp HOME config dir %q", configDir, claudeDir)
	}
	return claudeDir
}

func resetSwitchHistoryOnceForTest() {
	switchHistoryOnce = sync.Once{}
}
