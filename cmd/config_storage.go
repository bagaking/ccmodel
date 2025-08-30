package cmd

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// FileConfigStorage 基于文件系统的配置存储实现
type FileConfigStorage struct {
	configDir  string
	backupDir  string
	lockManager FileLockManager
	logger     Logger
}

// NewFileConfigStorage 创建文件配置存储实例
func NewFileConfigStorage(configDir string, lockManager FileLockManager, logger Logger) ConfigStorage {
	backupDir := filepath.Join(configDir, "ccmodel", "backups")
	return &FileConfigStorage{
		configDir:   configDir,
		backupDir:   backupDir,
		lockManager: lockManager,
		logger:      logger,
	}
}

// LoadModel 加载指定模型的配置
func (fcs *FileConfigStorage) LoadModel(modelName string) ([]byte, error) {
	sourceFile := filepath.Join(fcs.configDir, fmt.Sprintf("settings.%s.json", modelName))
	
	if _, err := os.Stat(sourceFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("model '%s' configuration not found: %s", modelName, sourceFile)
	}
	
	data, err := os.ReadFile(sourceFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read model config: %v", err)
	}
	
	// 清理配置文件，移除quota相关字段
	cleanedData, err := fcs.cleanConfigData(data)
	if err != nil {
		return nil, fmt.Errorf("failed to clean config data: %v", err)
	}
	
	return cleanedData, nil
}

// SaveActiveConfig 保存活动配置
func (fcs *FileConfigStorage) SaveActiveConfig(data []byte) error {
	targetFile := filepath.Join(fcs.configDir, "settings.json")
	lockFile := targetFile + ".lock"
	
	return fcs.lockManager.WithLock(lockFile, func() error {
		return os.WriteFile(targetFile, data, 0644)
	})
}

// BackupActiveConfig 备份当前活动配置
func (fcs *FileConfigStorage) BackupActiveConfig() error {
	targetFile := filepath.Join(fcs.configDir, "settings.json")
	
	// 检查活动配置是否存在
	if _, err := os.Stat(targetFile); os.IsNotExist(err) {
		fcs.logger.Debug("No active config to backup", "targetFile", targetFile)
		return nil // 没有配置文件需要备份
	}
	
	// 确保备份目录存在
	if err := os.MkdirAll(fcs.backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %v", err)
	}
	
	// 生成备份文件名
	timestamp := time.Now().Format("20060102_150405")
	backupFile := filepath.Join(fcs.backupDir, fmt.Sprintf("settings.json.backup.%s", timestamp))
	
	// 执行备份
	return fcs.copyFile(targetFile, backupFile)
}

// GetActiveModel 获取当前活动模型名称
func (fcs *FileConfigStorage) GetActiveModel() (string, error) {
	targetFile := filepath.Join(fcs.configDir, "settings.json")
	
	if _, err := os.Stat(targetFile); os.IsNotExist(err) {
		return "none", nil
	}
	
	// 计算当前配置的校验和
	currentSum, err := fcs.calculateConfigChecksum(targetFile)
	if err != nil {
		return "custom", err
	}
	
	// 与各个模型配置比较
	models, err := fcs.ListAvailableModels()
	if err != nil {
		return "custom", err
	}
	
	for _, model := range models {
		modelFile := filepath.Join(fcs.configDir, fmt.Sprintf("settings.%s.json", model))
		modelSum, err := fcs.calculateConfigChecksum(modelFile)
		if err == nil && currentSum == modelSum {
			return model, nil
		}
	}
	
	return "custom", nil
}

// ListAvailableModels 列出所有可用模型
func (fcs *FileConfigStorage) ListAvailableModels() ([]string, error) {
	files, err := os.ReadDir(fcs.configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read config directory: %v", err)
	}
	
	var models []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		name := file.Name()
		// 匹配 settings.*.json 格式
		if len(name) > 13 && name[:9] == "settings." && name[len(name)-5:] == ".json" {
			modelName := name[9 : len(name)-5] // 提取中间部分
			if modelName != "" {
				models = append(models, modelName)
			}
		}
	}
	
	return models, nil
}

// cleanConfigData 清理配置数据，移除quota相关字段
func (fcs *FileConfigStorage) cleanConfigData(data []byte) ([]byte, error) {
	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		// 如果不是有效JSON，直接返回原数据
		return data, nil
	}
	
	// 移除quota配置字段
	delete(config, "__cc")
	delete(config, "__ccmodel")
	
	return json.MarshalIndent(config, "", "  ")
}

// calculateConfigChecksum 计算配置文件的校验和（排除quota字段）
func (fcs *FileConfigStorage) calculateConfigChecksum(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	
	// 清理配置数据后计算校验和
	cleanedData, err := fcs.cleanConfigData(data)
	if err != nil {
		return "", err
	}
	
	h := md5.New()
	h.Write(cleanedData)
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// copyFile 复制文件
func (fcs *FileConfigStorage) copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source file: %v", err)
	}
	
	return os.WriteFile(dst, data, 0644)
}