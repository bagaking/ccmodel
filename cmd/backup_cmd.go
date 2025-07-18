package cmd

import (
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