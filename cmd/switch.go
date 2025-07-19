package cmd

import (
	"crypto/md5"
	"fmt"
	"io"
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
	sourceFile := filepath.Join(configDir, fmt.Sprintf("settings.%s.json", model))
	targetFile := filepath.Join(configDir, "settings.json")
	backupDir := filepath.Join(configDir, "backups")

	// Check if source file exists
	if _, err := os.Stat(sourceFile); os.IsNotExist(err) {
		errorBox := ui.NewBox().
			Title("❌ Model Not Found").
			Content(fmt.Sprintf("Configuration for model '%s' not found: %s", model, sourceFile)).
			TitleStyle(app.Theme().Error).
			ContentStyle(app.Theme().Error).
			BorderStyle(app.Theme().Error)
		app.Render(errorBox)
		return fmt.Errorf("configuration for model '%s' not found: %s", model, sourceFile)
	}

	// Show loading with cmdux Spinner
	spinner := ux.NewSpinner(ux.SpinnerDots).Color(app.Theme().Primary)
	spinner.Start(fmt.Sprintf("Switching to %s...", model))

	// Create backup directory if it doesn't exist
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		spinner.Error("Failed to create backup directory")
		return fmt.Errorf("failed to create backup directory: %v", err)
	}

	// Backup current configuration if it exists
	if _, err := os.Stat(targetFile); err == nil {
		backupFile := filepath.Join(backupDir, fmt.Sprintf("settings.json.backup.%s", time.Now().Format("20060102_150405")))
		if err := copyFile(targetFile, backupFile); err != nil {
			spinner.Error("Failed to backup current configuration")
			return fmt.Errorf("failed to backup current configuration: %v", err)
		}
		if verbose {
			app.Println(fmt.Sprintf("📁 Backed up to: %s", backupFile))
		}
	}

	// Switch configuration
	if err := copyFile(sourceFile, targetFile); err != nil {
		spinner.Error("Failed to switch configuration")
		return fmt.Errorf("failed to switch configuration: %v", err)
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
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
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
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
