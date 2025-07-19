package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bagaking/cmdux/ui"
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

	// Beautiful header using cmdux
	headerBox := ui.NewBox().
		Title("AI MODEL REGISTRY").
		Content("Available configurations for Claude Code").
		Width(70).
		TitleStyle(app.Theme().Primary).
		ContentStyle(app.Theme().Secondary).
		BorderStyle(app.Theme().Primary)
	app.Render(headerBox)
	app.Println("")

	if len(models) == 0 {
		warningBox := ui.NewBox().
			Title("⚠ No Configurations Found").
			Content("No AI model configurations found in ~/.claude/").
			TitleStyle(app.Theme().Warning).
			ContentStyle(app.Theme().Warning).
			BorderStyle(app.Theme().Warning)
		app.Render(warningBox)
		app.Println("")

		infoContent := `Getting Started:

• Create configuration files in ~/.claude/
• Name them as settings.{model}.json  
• Example: settings.gpt4.json, settings.claude3.json

Quick Commands:

ccmodel list         → List all available models
ccmodel current      → Show currently active model
ccmodel switch <name> → Switch to a different model
ccmodel --help       → Show detailed help`

		infoBox := ui.NewBox().
			Title("📚 Quick Start Guide").
			Content(infoContent).
			Width(60).
			TitleStyle(app.Theme().Accent2).
			ContentStyle(app.Theme().Primary).
			BorderStyle(app.Theme().Accent2)
		app.Render(infoBox)
		return nil
	}

	// Current status - use simple output instead of box for variety
	if current == "none" {
		app.Println("⚠  " + app.Theme().Warning.Sprint("Status: No Active Configuration"))
	} else if current == "custom" {
		app.Println("⚙  " + app.Theme().Accent1.Sprint("Status: Custom Configuration (Not managed by ccmodel)"))
	} else {
		app.Println("●  " + app.Theme().Success.Sprint("Status: "+current))
	}
	app.Println("")

	// Model table using cmdux
	table := ui.NewTable().
		Headers("#", "Status", "Model Name", "Size", "Modified", "State").
		HeaderStyle(app.Theme().Header).
		RowStyle(app.Theme().Primary).
		AltRowStyle(app.Theme().Secondary).
		BorderStyle(app.Theme().Border)

	for i, model := range models {
		modelFile := filepath.Join(configDir, fmt.Sprintf("settings.%s.json", model))
		info, err := os.Stat(modelFile)
		if err != nil {
			continue
		}

		isActive := model == current
		status := "○"
		state := ""
		if isActive {
			status = "★"
			state = "ACTIVE"
		}

		size := formatFileSize(info.Size())
		modified := info.ModTime().Format("Jan 02 15:04")

		table.AddRow(
			fmt.Sprintf("%d", i+1),
			status,
			model,
			size,
			modified,
			state,
		)
	}

	app.Render(table)
	app.Println("")

	// Summary info - use simple output instead of box for variety
	app.Println("📁  " + app.Theme().Primary.Sprint("Config Path: ") + configDir)
	app.Println("📊  " + app.Theme().Primary.Sprint("Total Models: ") + fmt.Sprintf("%d", len(models)))

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
