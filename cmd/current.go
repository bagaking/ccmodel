package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bagaking/cmdux/ui"
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

	headerBox := ui.NewBox().
		Title("CURRENT MODEL STATUS").
		Content("Active AI configuration").
		Width(60).
		TitleStyle(app.Theme().Primary).
		ContentStyle(app.Theme().Secondary).
		BorderStyle(app.Theme().Primary)
	app.Render(headerBox)
	app.Println("")

	if current == "none" {
		errorBox := ui.NewBox().
			Title("❌ No Active Configuration").
			Content("No active configuration found").
			TitleStyle(app.Theme().Error).
			ContentStyle(app.Theme().Error).
			BorderStyle(app.Theme().Error)
		app.Render(errorBox)
		app.Println("")

		nextStepsContent := `Next Steps:

• Create model configurations in ~/.claude/
• Use 'ccmodel list' to see available models  
• Use 'ccmodel switch <model>' to activate one

Quick Commands:

ccmodel list         → List all available models
ccmodel current      → Show currently active model
ccmodel switch <name> → Switch to a different model
ccmodel --help       → Show detailed help`

		infoBox := ui.NewBox().
			Title("📚 Next Steps").
			Content(nextStepsContent).
			Width(55).
			TitleStyle(app.Theme().Accent2).
			ContentStyle(app.Theme().Primary).
			BorderStyle(app.Theme().Accent2)
		app.Render(infoBox)
		return nil
	}

	// Model status - use simple output instead of box for variety
	if current == "custom" {
		app.Println("⚙  " + app.Theme().Accent1.Sprint("Custom Configuration"))
		app.Println("   " + app.Theme().Accent1.Sprint("Not managed by ccmodel templates"))
	} else {
		app.Println("●  " + app.Theme().Success.Sprint(current))
	}
	app.Println("")

	// File details
	if info, err := os.Stat(configFile); err == nil {
		detailsContent := fmt.Sprintf(`Model: %s
Config File: %s
File Size: %s
Last Modified: %s`,
			current,
			configFile,
			formatFileSize(info.Size()),
			info.ModTime().Format("2006-01-02 15:04:05"))

		detailsBox := ui.NewBox().
			Title("📋 Configuration Details").
			Content(detailsContent).
			TitleStyle(app.Theme().Accent1).
			ContentStyle(app.Theme().Primary).
			BorderStyle(app.Theme().Accent1)
		app.Render(detailsBox)

		if current == "custom" {
			app.Println("")
			app.Println("⚠  " + app.Theme().Warning.Sprint("This configuration is not managed by ccmodel templates"))
		}
	} else {
		app.Println("❌  " + app.Theme().Error.Sprint("Configuration file not found"))
	}

	return nil
}
