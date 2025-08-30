package cmd

import (
	"fmt"
	"time"
)

// DefaultModelSwitcher 模型切换器的默认实现
type DefaultModelSwitcher struct {
	configStorage ConfigStorage
	switchStorage SwitchStorage
	uiRenderer    UIRenderer
	logger        Logger
}

// NewModelSwitcher 创建模型切换器实例
func NewModelSwitcher(
	configStorage ConfigStorage,
	switchStorage SwitchStorage,
	uiRenderer UIRenderer,
	logger Logger,
) ModelSwitcher {
	return &DefaultModelSwitcher{
		configStorage: configStorage,
		switchStorage: switchStorage,
		uiRenderer:    uiRenderer,
		logger:        logger,
	}
}

// Switch 切换模型，这是核心业务逻辑，遵循SRP原则
func (ms *DefaultModelSwitcher) Switch(fromModel, toModel string) error {
	startTime := time.Now()
	
	// 延迟记录切换结果（成功或失败）
	var switchError error
	defer func() {
		duration := time.Since(startTime)
		if recordErr := ms.switchStorage.RecordSwitch(fromModel, toModel, duration, switchError); recordErr != nil {
			ms.logger.Warn("Failed to record switch history", "error", recordErr)
		}
	}()
	
	// 显示进度指示器
	spinner := ms.uiRenderer.ShowSpinner(fmt.Sprintf("Switching to %s...", toModel))
	
	// 1. 验证目标模型是否存在
	if switchError = ms.validateTargetModel(toModel); switchError != nil {
		spinner.Error("Model validation failed")
		ms.uiRenderer.ShowError("Model Not Found", switchError.Error())
		return switchError
	}
	
	// 2. 备份当前配置
	if switchError = ms.configStorage.BackupActiveConfig(); switchError != nil {
		spinner.Error("Backup failed")
		ms.logger.Error("Failed to backup current configuration", "error", switchError)
		return fmt.Errorf("failed to backup current configuration: %v", switchError)
	}
	
	ms.logger.Info("Current configuration backed up successfully")
	
	// 3. 加载目标模型配置
	configData, err := ms.configStorage.LoadModel(toModel)
	if err != nil {
		switchError = err
		spinner.Error("Failed to load model config")
		return fmt.Errorf("failed to load model configuration: %v", err)
	}
	
	// 4. 保存新的活动配置
	if switchError = ms.configStorage.SaveActiveConfig(configData); switchError != nil {
		spinner.Error("Failed to save configuration")
		return fmt.Errorf("failed to save configuration: %v", switchError)
	}
	
	// 5. 显示成功
	spinner.Success(fmt.Sprintf("Successfully switched to %s!", toModel))
	
	ms.uiRenderer.ShowSuccess(
		"Switch Complete",
		fmt.Sprintf("Switched to %s configuration\nRestart Claude Code to apply changes", toModel),
	)
	
	// 6. 显示详细信息（如果启用详细模式）
	ms.showSwitchDetails(fromModel, toModel)
	
	return nil
}

// GetCurrentModel 获取当前模型
func (ms *DefaultModelSwitcher) GetCurrentModel() (string, error) {
	return ms.configStorage.GetActiveModel()
}

// validateTargetModel 验证目标模型是否存在
func (ms *DefaultModelSwitcher) validateTargetModel(modelName string) error {
	availableModels, err := ms.configStorage.ListAvailableModels()
	if err != nil {
		return fmt.Errorf("failed to list available models: %v", err)
	}
	
	// 检查目标模型是否在可用列表中
	for _, model := range availableModels {
		if model == modelName {
			return nil // 找到目标模型
		}
	}
	
	return fmt.Errorf("model '%s' not found in available models: %v", modelName, availableModels)
}

// showSwitchDetails 显示切换详细信息
func (ms *DefaultModelSwitcher) showSwitchDetails(fromModel, toModel string) {
	details := fmt.Sprintf("Switch completed: %s → %s", fromModel, toModel)
	ms.uiRenderer.ShowInfo(details)
	ms.logger.Info("Model switch completed successfully", 
		"from", fromModel, 
		"to", toModel,
	)
}

// SwitchCommand 切换命令的协调器，遵循组合优于继承原则
type SwitchCommand struct {
	switcher ModelSwitcher
}

// NewSwitchCommand 创建切换命令实例
func NewSwitchCommand(switcher ModelSwitcher) *SwitchCommand {
	return &SwitchCommand{
		switcher: switcher,
	}
}

// Execute 执行切换命令
func (sc *SwitchCommand) Execute(targetModel string) error {
	// 获取当前模型
	currentModel, err := sc.switcher.GetCurrentModel()
	if err != nil {
		return fmt.Errorf("failed to get current model: %v", err)
	}
	
	// 如果目标模型与当前模型相同，给出提示
	if currentModel == targetModel {
		return fmt.Errorf("already using model '%s'", targetModel)
	}
	
	// 执行切换
	return sc.switcher.Switch(currentModel, targetModel)
}