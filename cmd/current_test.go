package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestModelStatusCollector_Basic(t *testing.T) {
	tempDir := t.TempDir()
	collector := NewModelStatusCollector(tempDir)
	
	if collector.configDir != tempDir {
		t.Errorf("Expected configDir %s, got %s", tempDir, collector.configDir)
	}
}

func TestModelStatusCollector_collectFileInfo(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "settings.json")
	
	// 创建测试配置文件
	testConfig := `{"test": "config"}`
	err := os.WriteFile(configFile, []byte(testConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	collector := NewModelStatusCollector(tempDir)
	status := &ModelStatus{
		Model:      "test-model",
		ConfigPath: configFile,
		IsActive:   true,
		IsCustom:   false,
	}
	
	err = collector.collectFileInfo(status)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	
	if status.FileSize == 0 {
		t.Error("Expected FileSize to be > 0")
	}
	
	if status.LastModified == "" {
		t.Error("Expected LastModified to be set")
	}
	
	// 验证 ISO 时间格式
	if len(status.LastModified) != len("2006-01-02T15:04:05Z") {
		t.Errorf("Expected ISO timestamp format, got: %s", status.LastModified)
	}
}

func TestModelStatusCollector_collectFileInfo_NoFile(t *testing.T) {
	collector := NewModelStatusCollector("/nonexistent")
	status := &ModelStatus{
		Model:      "test-model",
		ConfigPath: "/nonexistent/settings.json",
	}
	
	err := collector.collectFileInfo(status)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
	
	if status.FileSize != 0 {
		t.Error("Expected FileSize to remain 0")
	}
}

func TestModelStatusCollector_collectFileInfo_NoneModel(t *testing.T) {
	collector := NewModelStatusCollector("/tmp")
	status := &ModelStatus{
		Model: "none",
	}
	
	err := collector.collectFileInfo(status)
	if err != nil {
		t.Errorf("Expected no error for 'none' model, got: %v", err)
	}
}

func TestQuotaStatusJSON_Serialization(t *testing.T) {
	quota := &QuotaStatusJSON{
		Total:   1000.5,
		Used:    250.25,
		Percent: 25.0,
	}
	
	jsonData, err := json.Marshal(quota)
	if err != nil {
		t.Fatalf("Failed to marshal quota: %v", err)
	}
	
	var unmarshaled QuotaStatusJSON
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal quota: %v", err)
	}
	
	if unmarshaled.Total != quota.Total {
		t.Errorf("Expected total %f, got %f", quota.Total, unmarshaled.Total)
	}
	
	if unmarshaled.Used != quota.Used {
		t.Errorf("Expected used %f, got %f", quota.Used, unmarshaled.Used)
	}
	
	if unmarshaled.Percent != quota.Percent {
		t.Errorf("Expected percent %f, got %f", quota.Percent, unmarshaled.Percent)
	}
}

func TestModelStatus_JSONSerialization(t *testing.T) {
	status := &ModelStatus{
		Model:        "test-model",
		ConfigPath:   "/test/path/settings.json",
		FileSize:     1024,
		LastModified: "2025-09-08T00:42:27Z",
		IsActive:     true,
		IsCustom:     false,
		Quota: &QuotaStatusJSON{
			Total:   1000.0,
			Used:    250.0,
			Percent: 25.0,
		},
	}
	
	jsonData, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal status: %v", err)
	}
	
	var unmarshaled ModelStatus
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal status: %v", err)
	}
	
	if unmarshaled.Model != status.Model {
		t.Errorf("Expected model %s, got %s", status.Model, unmarshaled.Model)
	}
	
	if unmarshaled.Quota == nil {
		t.Error("Expected quota to be unmarshaled")
	} else {
		if unmarshaled.Quota.Total != status.Quota.Total {
			t.Errorf("Expected quota total %f, got %f", status.Quota.Total, unmarshaled.Quota.Total)
		}
	}
}

func TestModelStatus_JSONSerialization_WithError(t *testing.T) {
	status := &ModelStatus{
		Model:    "none",
		IsActive: false,
		Error:    "test error message",
	}
	
	jsonData, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal status: %v", err)
	}
	
	var unmarshaled ModelStatus
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal status: %v", err)
	}
	
	if unmarshaled.Error != status.Error {
		t.Errorf("Expected error %s, got %s", status.Error, unmarshaled.Error)
	}
}

func TestParseLastModified(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"2025-09-08T00:42:27Z", "2025-09-08 00:42:27"},
		{"", ""},
		{"invalid", "invalid"},
	}
	
	for _, tc := range testCases {
		result := parseLastModified(tc.input)
		if result != tc.expected {
			t.Errorf("For input %s, expected %s, got %s", tc.input, tc.expected, result)
		}
	}
}

// 测试 JSON 结构的完整性
func TestModelStatus_JSONCompleteStructure(t *testing.T) {
	status := &ModelStatus{
		Model:        "complete-test",
		ConfigPath:   "/complete/test/settings.json",
		FileSize:     2048,
		LastModified: "2025-09-08T00:42:27Z",
		IsActive:     true,
		IsCustom:     false,
		Quota: &QuotaStatusJSON{
			Total:   5000.0,
			Used:    1500.0,
			Percent: 30.0,
		},
	}
	
	jsonData, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal status: %v", err)
	}
	
	// 验证 JSON 包含期望的字段
	expectedFields := []string{
		`"model": "complete-test"`,
		`"config_path": "/complete/test/settings.json"`,
		`"file_size": 2048`,
		`"last_modified": "2025-09-08T00:42:27Z"`,
		`"is_active": true`,
		`"is_custom": false`,
		`"quota":`,
		`"total": 5000`,
		`"used": 1500`,
		`"percent": 30`,
	}
	
	jsonString := string(jsonData)
	for _, field := range expectedFields {
		if !containsString(jsonString, field) {
			t.Errorf("JSON missing expected field: %s\nActual JSON:\n%s", field, jsonString)
		}
	}
}

// 辅助函数：检查字符串包含关系
func containsString(text, substr string) bool {
	for i := 0; i <= len(text)-len(substr); i++ {
		if text[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}