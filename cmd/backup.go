package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Create a backup of the current configuration",
	Long:  "Create a timestamped backup of the current settings.json file",
	RunE:  runBackup,
}

func init() {
	// Command is added in root.go
}

func runBackup(cmd *cobra.Command, args []string) error {
	return backupCurrent()
}

func backupCurrent() error {
	currentFile := filepath.Join(configDir, "settings.json")
	backupDir := filepath.Join(configDir, "backups")

	if _, err := os.Stat(currentFile); os.IsNotExist(err) {
		return fmt.Errorf("no current configuration to backup")
	}

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %v", err)
	}

	backupFile := filepath.Join(backupDir, fmt.Sprintf("settings.json.backup.%s", time.Now().Format("20060102_150405")))
	if err := copyFile(currentFile, backupFile); err != nil {
		return fmt.Errorf("failed to create backup: %v", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()

	fmt.Printf("%s Configuration backed up to: %s\n", green("✅"), blue(backupFile))
	return nil
}
