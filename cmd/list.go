package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bagaking/ccmodel/internal/ui"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List available model configurations",
	Long:    "Display all available Claude Code model configurations with detailed information",
	Aliases: []string{"ls"},
	RunE:    runList,
}

func init() {
	// Commands are added in root.go
}

func runList(cmd *cobra.Command, args []string) error {
	models, err := getAvailableModels()
	if err != nil {
		return err
	}

	current, _ := getCurrentModel()

	// Beautiful header
	ui.Header("AI MODEL REGISTRY", "Available configurations for Claude Code")

	if len(models) == 0 {
		ui.WarningBox("No AI model configurations found")
		fmt.Println()
		
		ui.InfoBox("Getting Started", []string{
			"Create configuration files in ~/.claude/",
			"Name them as settings.{model}.json",
			"Example: settings.gpt4.json, settings.claude3.json",
		})
		
		ui.QuickStartBox()
		return nil
	}

	// Current status
	if current == "" {
		ui.StatusLine("⚠", "No Active Configuration", "", ui.Warning)
	} else if current == "custom" {
		ui.StatusLine("⚙", "Custom Configuration", "Not managed by ccmodel", ui.Accent1)
	} else {
		ui.StatusLine("●", "Active Model", current, ui.Success)
	}
	
	fmt.Println()
	
	// Model listing header
	fmt.Printf("%s %s\n", ui.Primary.Sprint("┌─"), ui.Bold.Sprint("AVAILABLE MODELS"))
	fmt.Printf("%s\n", ui.Muted.Sprint("├─────────────────────────────────────────────────────"))

	for i, model := range models {
		modelFile := filepath.Join(configDir, fmt.Sprintf("settings.%s.json", model))
		info, err := os.Stat(modelFile)
		if err != nil {
			continue
		}

		isActive := model == current
		size := formatFileSize(info.Size())
		modified := info.ModTime().Format("2006-01-02 15:04")

		ui.ModelEntry(i+1, model, size, modified, isActive)
	}
	
	fmt.Printf("%s\n", ui.Muted.Sprint("└─────────────────────────────────────────────────────"))
	fmt.Println()
	
	// Footer info
	ui.InfoBox("Registry Information", []string{
		fmt.Sprintf("Config Directory: %s", configDir),
		fmt.Sprintf("Total Models: %d", len(models)),
		fmt.Sprintf("Active Model: %s", func() string {
			if current == "" {
				return "None"
			}
			return current
		}()),
	})

	return nil
}

func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%c", float64(bytes)/float64(div), "KMGTPE"[exp])
}
