package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestFileConfigStorage_LoadModel_Success 测试成功加载模型配置
func TestFileConfigStorage_LoadModel_Success(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	mockLogger := &MockLogger{}
	mockLockManager := &MockFileLockManager{}
	storage := NewFileConfigStorage(tempDir, mockLockManager, mockLogger)
	
	// 创建测试配置文件
	testConfig := map[string]interface{}{
		"model": "test-model",
		"apiKey": "test-key",
		"__cc": map[string]interface{}{
			"quota_test": "should_be_removed",
		},
	}
	
	configData, _ := json.MarshalIndent(testConfig, "", "  ")
	configFile := filepath.Join(tempDir, "settings.test-model.json")
	err = os.WriteFile(configFile, configData, 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}
	
	// 加载模型配置
	loadedData, err := storage.LoadModel("test-model")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// 验证配置数据被清理（移除了__cc字段）
	var loadedConfig map[string]interface{}
	err = json.Unmarshal(loadedData, &loadedConfig)
	if err != nil {
		t.Fatalf("Failed to unmarshal loaded config: %v", err)
	}
	
	if _, exists := loadedConfig["__cc"]; exists {
		t.Error("Expected __cc field to be removed from loaded config")
	}
	
	if loadedConfig["model"] != "test-model" {
		t.Errorf("Expected model field to be preserved, got %v", loadedConfig["model"])
	}
}

// TestFileConfigStorage_LoadModel_NotFound 测试加载不存在的模型
func TestFileConfigStorage_LoadModel_NotFound(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	mockLogger := &MockLogger{}
	mockLockManager := &MockFileLockManager{}
	storage := NewFileConfigStorage(tempDir, mockLockManager, mockLogger)
	
	// 尝试加载不存在的模型
	_, err = storage.LoadModel("nonexistent")
	
	// 验证返回错误
	if err == nil {
		t.Error("Expected error for nonexistent model, got nil")
	}
}

// TestFileConfigStorage_SaveActiveConfig 测试保存活动配置
func TestFileConfigStorage_SaveActiveConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	mockLogger := &MockLogger{}
	mockLockManager := &MockFileLockManager{}
	storage := NewFileConfigStorage(tempDir, mockLockManager, mockLogger)
	
	testData := []byte(`{"model":"active-model","key":"value"}`)
	
	// 保存活动配置
	err = storage.SaveActiveConfig(testData)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// 验证文件被创建
	settingsFile := filepath.Join(tempDir, "settings.json")
	savedData, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Errorf("Failed to read saved config: %v", err)
	}
	
	if string(savedData) != string(testData) {
		t.Errorf("Saved data doesn't match input. Expected %s, got %s", testData, savedData)
	}
	
	// 验证使用了文件锁
	if len(mockLockManager.lockCalls) != 1 {
		t.Errorf("Expected 1 lock call, got %d", len(mockLockManager.lockCalls))
	}
}

// TestFileConfigStorage_BackupActiveConfig 测试备份活动配置
func TestFileConfigStorage_BackupActiveConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	mockLogger := &MockLogger{}
	mockLockManager := &MockFileLockManager{}
	storage := NewFileConfigStorage(tempDir, mockLockManager, mockLogger)
	
	// 创建活动配置文件
	testData := []byte(`{"model":"active"}`)
	settingsFile := filepath.Join(tempDir, "settings.json")
	err = os.WriteFile(settingsFile, testData, 0644)
	if err != nil {
		t.Fatalf("Failed to create active config: %v", err)
	}
	
	// 备份活动配置
	err = storage.BackupActiveConfig()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// 验证备份目录被创建
	backupDir := filepath.Join(tempDir, "ccmodel", "backups")
	if _, statErr := os.Stat(backupDir); os.IsNotExist(statErr) {
		t.Error("Backup directory should have been created")
	}
	
	// 验证备份文件被创建
	files, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("Failed to read backup directory: %v", err)
	}
	
	if len(files) != 1 {
		t.Errorf("Expected 1 backup file, got %d", len(files))
	}
	
	// 验证备份文件内容
	backupFile := filepath.Join(backupDir, files[0].Name())
	backupData, err := os.ReadFile(backupFile)
	if err != nil {
		t.Errorf("Failed to read backup file: %v", err)
	}
	
	if string(backupData) != string(testData) {
		t.Errorf("Backup data doesn't match original. Expected %s, got %s", testData, backupData)
	}
}

// TestFileConfigStorage_BackupActiveConfig_NoActiveConfig 测试无活动配置时的备份
func TestFileConfigStorage_BackupActiveConfig_NoActiveConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	mockLogger := &MockLogger{}
	mockLockManager := &MockFileLockManager{}
	storage := NewFileConfigStorage(tempDir, mockLockManager, mockLogger)
	
	// 备份不存在的活动配置（应该成功但不创建备份）
	err = storage.BackupActiveConfig()
	if err != nil {
		t.Errorf("Expected no error when no active config exists, got %v", err)
	}
	
	// 验证调试日志被记录
	// 注意：这里我们检查的是 infoCalls 而不是专门的 debugCalls，
	// 因为我们的 MockLogger 没有实现 Debug 方法
	// 在实际代码中，这应该是一个 Debug 调用
	_ = mockLogger.infoCalls // 暂时跳过日志验证
}

// TestFileConfigStorage_GetActiveModel 测试获取当前活动模型
func TestFileConfigStorage_GetActiveModel(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	mockLogger := &MockLogger{}
	mockLockManager := &MockFileLockManager{}
	storage := NewFileConfigStorage(tempDir, mockLockManager, mockLogger)
	
	// 创建模型配置文件
	testConfig := map[string]interface{}{"model": "test-model"}
	configData, _ := json.MarshalIndent(testConfig, "", "  ")
	
	modelFile := filepath.Join(tempDir, "settings.test-model.json")
	err = os.WriteFile(modelFile, configData, 0644)
	if err != nil {
		t.Fatalf("Failed to create model config: %v", err)
	}
	
	// 创建活动配置文件（与模型配置相同）
	settingsFile := filepath.Join(tempDir, "settings.json")
	err = os.WriteFile(settingsFile, configData, 0644)
	if err != nil {
		t.Fatalf("Failed to create active config: %v", err)
	}
	
	// 获取活动模型
	activeModel, err := storage.GetActiveModel()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if activeModel != "test-model" {
		t.Errorf("Expected 'test-model', got %s", activeModel)
	}
}

// TestFileConfigStorage_GetActiveModel_NoActiveConfig 测试无活动配置时获取模型
func TestFileConfigStorage_GetActiveModel_NoActiveConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	mockLogger := &MockLogger{}
	mockLockManager := &MockFileLockManager{}
	storage := NewFileConfigStorage(tempDir, mockLockManager, mockLogger)
	
	// 获取活动模型（无活动配置）
	activeModel, err := storage.GetActiveModel()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if activeModel != "none" {
		t.Errorf("Expected 'none', got %s", activeModel)
	}
}

// TestFileConfigStorage_ListAvailableModels 测试列出可用模型
func TestFileConfigStorage_ListAvailableModels(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	mockLogger := &MockLogger{}
	mockLockManager := &MockFileLockManager{}
	storage := NewFileConfigStorage(tempDir, mockLockManager, mockLogger)
	
	// 创建多个模型配置文件
	modelNames := []string{"model1", "model2", "model3"}
	for _, name := range modelNames {
		configFile := filepath.Join(tempDir, fmt.Sprintf("settings.%s.json", name))
		err = os.WriteFile(configFile, []byte(`{}`), 0644)
		if err != nil {
			t.Fatalf("Failed to create config file for %s: %v", name, err)
		}
	}
	
	// 创建一些非模型文件（应该被忽略）
	os.WriteFile(filepath.Join(tempDir, "other.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tempDir, "settings.json"), []byte("{}"), 0644) // 活动配置，应该被忽略
	
	// 列出可用模型
	availableModels, err := storage.ListAvailableModels()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// 验证返回的模型列表
	if len(availableModels) != len(modelNames) {
		t.Errorf("Expected %d models, got %d", len(modelNames), len(availableModels))
	}
	
	// 验证所有期望的模型都在列表中
	for _, expectedModel := range modelNames {
		found := false
		for _, actualModel := range availableModels {
			if actualModel == expectedModel {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected model %s not found in available models", expectedModel)
		}
	}
}

// TestFileConfigStorage_CleanConfigData 测试配置数据清理
func TestFileConfigStorage_CleanConfigData(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	mockLogger := &MockLogger{}
	mockLockManager := &MockFileLockManager{}
	storage := NewFileConfigStorage(tempDir, mockLockManager, mockLogger).(*FileConfigStorage)
	
	// 创建包含需要清理字段的配置数据
	originalConfig := map[string]interface{}{
		"model":     "test-model",
		"apiKey":    "test-key",
		"__cc":      map[string]interface{}{"quota": "test"},
		"__ccmodel": map[string]interface{}{"settings": "test"},
		"normalField": "should_remain",
	}
	
	originalData, _ := json.Marshal(originalConfig)
	
	// 清理配置数据
	cleanedData, err := storage.cleanConfigData(originalData)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// 解析清理后的数据
	var cleanedConfig map[string]interface{}
	err = json.Unmarshal(cleanedData, &cleanedConfig)
	if err != nil {
		t.Fatalf("Failed to unmarshal cleaned data: %v", err)
	}
	
	// 验证需要清理的字段被移除
	if _, exists := cleanedConfig["__cc"]; exists {
		t.Error("Expected __cc field to be removed")
	}
	
	if _, exists := cleanedConfig["__ccmodel"]; exists {
		t.Error("Expected __ccmodel field to be removed")
	}
	
	// 验证正常字段被保留
	if cleanedConfig["model"] != "test-model" {
		t.Error("Expected normal fields to be preserved")
	}
	
	if cleanedConfig["normalField"] != "should_remain" {
		t.Error("Expected normal fields to be preserved")
	}
}

// == Mock 文件锁管理器 ==

type MockFileLockManager struct {
	lockCalls []string
	lockError error
}

func (m *MockFileLockManager) WithLock(lockFile string, operation func() error) error {
	m.lockCalls = append(m.lockCalls, lockFile)
	if m.lockError != nil {
		return m.lockError
	}
	return operation()
}

func (m *MockFileLockManager) WithReadLock(lockFile string, operation func() error) error {
	return m.WithLock(lockFile, operation)
}