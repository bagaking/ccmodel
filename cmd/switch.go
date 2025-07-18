package cmd

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/bagaking/ccmodel/internal/ui"
	"github.com/bagaking/ccmodel/internal/ux"
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
		return fmt.Errorf("configuration for model '%s' not found: %s", model, sourceFile)
	}

	// Create backup directory if it doesn't exist
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %v", err)
	}

	// Backup current configuration if it exists
	if _, err := os.Stat(targetFile); err == nil {
		backupFile := filepath.Join(backupDir, fmt.Sprintf("settings.json.backup.%s", time.Now().Format("20060102_150405")))
		if err := copyFile(targetFile, backupFile); err != nil {
			return fmt.Errorf("failed to backup current configuration: %v", err)
		}
		if verbose {
			fmt.Printf("📁 Backed up to: %s\n", backupFile)
		}
	}

	// Use advanced loading animation
	spinner := ux.NewSpinner("dots")
	spinner.Start(fmt.Sprintf("Switching to %s configuration...", model))
	
	// Switch configuration
	if err := copyFile(sourceFile, targetFile); err != nil {
		spinner.Error(fmt.Sprintf("Failed to switch configuration: %v", err))
		return fmt.Errorf("failed to switch configuration: %v", err)
	}
	
	spinner.Success(fmt.Sprintf("Switched to %s configuration", model))
	if verbose {
		ui.InfoBox("Operation Details", []string{
			fmt.Sprintf("Source: %s", sourceFile),
			"Restart Claude Code to apply changes",
		})
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