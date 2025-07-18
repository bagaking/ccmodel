package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

	// 字段宽度
	indexWidth := 2
	indicatorWidth := 2
	nameWidth := 18
	sizeWidth := 6
	dateWidth := 10
	statusWidth := 7

	// 竖线数量：#、indicator、name、size、date、status 共 6 个分隔符
	// totalWidth := 1 + 1 + indexWidth + 3 + indicatorWidth + 1 + nameWidth + 3 + sizeWidth + 3 + dateWidth + 3 + statusWidth + 2

	// ┌────┬────┬──────────────────┬────────┬──────────┬─────────┐
	fmt.Printf("┌%s┬%s┬%s┬%s┬%s┬%s┐\n",
		strings.Repeat("─", indexWidth+2),
		strings.Repeat("─", indicatorWidth+2),
		strings.Repeat("─", nameWidth+2),
		strings.Repeat("─", sizeWidth+2),
		strings.Repeat("─", dateWidth+2),
		strings.Repeat("─", statusWidth+2),
	)
	fmt.Printf("│ %-*s │ %-*s │ %-*s │ %-*s │ %-*s │ %-*s │\n",
		indexWidth, "#",
		indicatorWidth, " ",
		nameWidth, "Name",
		sizeWidth, "Size",
		dateWidth, "Modified",
		statusWidth, "Status",
	)
	fmt.Printf("├%s┼%s┼%s┼%s┼%s┼%s┤\n",
		strings.Repeat("─", indexWidth+2),
		strings.Repeat("─", indicatorWidth+2),
		strings.Repeat("─", nameWidth+2),
		strings.Repeat("─", sizeWidth+2),
		strings.Repeat("─", dateWidth+2),
		strings.Repeat("─", statusWidth+2),
	)
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
	fmt.Printf("└%s┴%s┴%s┴%s┴%s┴%s┘\n",
		strings.Repeat("─", indexWidth+2),
		strings.Repeat("─", indicatorWidth+2),
		strings.Repeat("─", nameWidth+2),
		strings.Repeat("─", sizeWidth+2),
		strings.Repeat("─", dateWidth+2),
		strings.Repeat("─", statusWidth+2),
	)
	fmt.Println()

	// Enhanced footer info with better formatting
	ui.InfoBox("Registry Summary", []string{
		fmt.Sprintf("📁 Config Path: %s", configDir),
		fmt.Sprintf("📊 Total Models: %d", len(models)),
		fmt.Sprintf("⚡ Active Model: %s", func() string {
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
