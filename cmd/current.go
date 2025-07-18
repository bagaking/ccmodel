package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bagaking/ccmodel/internal/ui"
	"github.com/spf13/cobra"
)

var currentCmd = &cobra.Command{
	Use:     "current",
	Short:   "Show current model configuration",
	Long:    "Display detailed information about the currently active AI model configuration",
	Aliases: []string{"status", "whoami"},
	RunE:    runCurrent,
}

func init() {
	// Command is added in root.go
}

func runCurrent(cmd *cobra.Command, args []string) error {
	current, err := getCurrentModel()
	if err != nil {
		return err
	}

	configFile := filepath.Join(configDir, "settings.json")
	
	ui.Header("CURRENT MODEL STATUS", "Active AI configuration")
	
	if current == "none" {
		ui.ErrorBox("No active configuration found")
		fmt.Println()
		
		ui.InfoBox("Next Steps", []string{
			"Create model configurations in ~/.claude/",
			"Use 'ccmodel list' to see available models",
			"Use 'ccmodel switch <model>' to activate one",
		})
		
		ui.QuickStartBox()
		return nil
	}

	// Model status
	if current == "custom" {
		ui.StatusLine("⚙", "Custom Configuration", "Not managed by ccmodel templates", ui.Accent1)
	} else {
		ui.StatusLine("●", "Active Model", current, ui.Success)
	}
	
	fmt.Println()
	
	// File details
	if info, err := os.Stat(configFile); err == nil {
		ui.InfoBox("Configuration Details", []string{
			fmt.Sprintf("Model: %s", current),
			fmt.Sprintf("Config File: %s", configFile),
			fmt.Sprintf("File Size: %s", formatFileSize(info.Size())),
			fmt.Sprintf("Last Modified: %s", info.ModTime().Format("2006-01-02 15:04:05")),
		})
		
		if current == "custom" {
			ui.WarningBox("This configuration is not managed by ccmodel templates")
		}
	} else {
		ui.ErrorBox("Configuration file not found")
	}

	return nil
}
