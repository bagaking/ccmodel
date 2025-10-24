package cmd

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bagaking/cmdux/ui"
	"github.com/bagaking/cmdux/ux"
	"github.com/spf13/cobra"
)

var switchCmd = &cobra.Command{
	Use:   "switch <model>",
	Short: "Switch to a different model configuration",
	Args:  cobra.ExactArgs(1),
	RunE:  runSwitch,
}

func runSwitch(cmd *cobra.Command, args []string) error {
	return switchModel(args[0])
}

func switchModel(model string) error {
	startTime := time.Now()

	// 获取当前模型用于历史记录
	fromModel, _ := getCurrentModel()
	if fromModel == "" {
		fromModel = "none"
	}

	// 获取切换历史管理器
	switchHistory := getSwitchHistoryManager()

	// 延迟记录切换结果（成功或失败）
	var switchError error
	defer func() {
		duration := time.Since(startTime)
		if recordErr := switchHistory.RecordSwitch(fromModel, model, duration, switchError); recordErr != nil && verbose {
			fmt.Printf("Warning: Failed to record switch history: %v\n", recordErr)
		}
	}()

	sourceFile := filepath.Join(configDir, fmt.Sprintf("settings.%s.json", model))
	targetFile := filepath.Join(configDir, "settings.json")
	// 将backup目录统一放在ccmodel目录下，便于集中管理
	backupDir := filepath.Join(configDir, "ccmodel", "backups")

	// Check if source file exists
	if _, err := os.Stat(sourceFile); os.IsNotExist(err) {
		switchError = fmt.Errorf("configuration for model '%s' not found: %s", model, sourceFile)
		errorBox := ui.NewBox().
			Title("❌ Model Not Found").
			Content(fmt.Sprintf("Configuration for model '%s' not found: %s", model, sourceFile)).
			TitleStyle(app.Theme().Error).
			ContentStyle(app.Theme().Error).
			BorderStyle(app.Theme().Error)
		app.Render(errorBox)
		return switchError
	}

	// Show loading with cmdux Spinner
	spinner := ux.NewSpinner(ux.SpinnerDots).Color(app.Theme().Primary)
	spinner.Start(fmt.Sprintf("Switching to %s...", model))

	if err := ensureUserOnlyDir(configDir); err != nil {
		spinner.Error("Failed to create config directory")
		switchError = fmt.Errorf("failed to create config directory: %v", err)
		return switchError
	}

	// Create backup directory if it doesn't exist
	if err := ensureUserOnlyDir(filepath.Join(configDir, "ccmodel")); err != nil {
		spinner.Error("Failed to create ccmodel directory")
		switchError = fmt.Errorf("failed to create ccmodel directory: %v", err)
		return switchError
	}
	if err := ensureUserOnlyDir(backupDir); err != nil {
		spinner.Error("Failed to create backup directory")
		switchError = fmt.Errorf("failed to create backup directory: %v", err)
		return switchError
	}

	// Backup current configuration if it exists
	if _, err := os.Stat(targetFile); err == nil {
		backupFile := filepath.Join(backupDir, fmt.Sprintf("settings.json.backup.%s", time.Now().Format("20060102_150405")))
		if err := copyFileRaw(targetFile, backupFile); err != nil {
			spinner.Error("Failed to backup current configuration")
			switchError = fmt.Errorf("failed to backup current configuration: %v", err)
			return switchError
		}
		if verbose {
			app.Println(fmt.Sprintf("📁 Backed up to: %s", backupFile))
		}
	}

	// Switch configuration
	if err := copyFile(sourceFile, targetFile); err != nil {
		spinner.Error("Failed to switch configuration")
		switchError = fmt.Errorf("failed to switch configuration: %v", err)
		return switchError
	}

	time.Sleep(1 * time.Second) // Show the loading animation
	spinner.Success(fmt.Sprintf("Successfully switched to %s!", model))
	app.Println("")

	successBox := ui.NewBox().
		Title("✅ Switch Complete").
		Content(fmt.Sprintf("Switched to %s configuration\nRestart Claude Code to apply changes", model)).
		TitleStyle(app.Theme().Success).
		ContentStyle(app.Theme().Success).
		BorderStyle(app.Theme().Success)
	app.Render(successBox)

	if verbose {
		app.Println("")
		app.Println("📁  " + app.Theme().Primary.Sprint("Source: ") + sourceFile)
		app.Println("📁  " + app.Theme().Primary.Sprint("Target: ") + targetFile)
	}

	return nil
}

func copyFile(src, dst string) error {
	// Read source file
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	// Parse JSON and remove __cc and __ccmodel fields
	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	// Remove quota config fields for clean copy
	delete(config, "__cc")
	delete(config, "__ccmodel")

	// Write cleaned config to destination
	cleanData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return writeUserOnlyFile(dst, cleanData)
}

func copyFileRaw(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return writeUserOnlyFile(dst, data)
}

func getCurrentModel() (string, error) {
	targetFile := filepath.Join(configDir, "settings.json")

	if _, err := os.Stat(targetFile); os.IsNotExist(err) {
		return "none", nil
	}

	// Calculate checksum of current settings
	currentSum, err := fileChecksum(targetFile)
	if err != nil {
		return "custom", err
	}

	// Compare with each model configuration
	models, err := getAvailableModels()
	if err != nil {
		return "custom", err
	}

	for _, model := range models {
		modelFile := filepath.Join(configDir, fmt.Sprintf("settings.%s.json", model))
		modelSum, err := fileChecksum(modelFile)
		if err == nil && currentSum == modelSum {
			return model, nil
		}
	}

	return "custom", nil
}

func fileChecksum(file string) (string, error) {
	// Read file content
	data, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}

	// Parse JSON and remove __cc and __ccmodel fields for comparison
	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		// If not valid JSON, fall back to raw content
		h := md5.New()
		h.Write(data)
		return fmt.Sprintf("%x", h.Sum(nil)), nil
	}

	// Remove quota config fields for consistent comparison
	delete(config, "__cc")
	delete(config, "__ccmodel")

	// Use cleaned config for checksum
	cleanData, err := json.Marshal(config)
	if err != nil {
		return "", err
	}

	h := md5.New()
	h.Write(cleanData)
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
