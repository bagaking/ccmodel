package cmd

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"
)

var (
	configDir string
	verbose   bool
)

var rootCmd = &cobra.Command{
	Use:   "ccmodel [model]",
	Short: "Claude Code model configuration switcher",
	Long: `ccmodel is a tool to switch between different AI service configurations
for Claude Code by swapping settings.json files atomically.

It provides a simple interface to manage multiple AI service providers
(OpenRouter, Moonshot, Anthropic, etc.) without modifying business code.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRoot,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
	
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(currentCmd)
	rootCmd.AddCommand(switchCmd)
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(completionCmd)
}

func initConfig() {
	if configDir == "" {
		configDir = filepath.Join(os.Getenv("HOME"), ".claude")
	}
}

func runRoot(cmd *cobra.Command, args []string) error {
	// Check if we're being called for completion - avoid UI output
	if cmd.CalledAs() == "completion" || (len(os.Args) > 1 && os.Args[1] == "completion") {
		return nil
	}
	
	if len(args) == 0 {
		return runList(cmd, args)
	}
	return switchModel(args[0])
}

func getAvailableModels() ([]string, error) {
	pattern := filepath.Join(configDir, "settings.*.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	
	models := []string{}
	for _, file := range files {
		model := extractModelName(file)
		if model != "" && model != "json" {
			models = append(models, model)
		}
	}
	
	sort.Strings(models)
	return models, nil
}

func extractModelName(filename string) string {
	base := filepath.Base(filename)
	matched, _ := filepath.Match("settings.*.json", base)
	if !matched {
		return ""
	}
	
	// Remove "settings." prefix and ".json" suffix
	model := base[len("settings.") : len(base)-len(".json")]
	return model
}