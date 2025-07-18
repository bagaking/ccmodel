package cmd

import (
	"fmt"
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
	if len(args) == 0 {
		return listModels()
	}
	return switchModel(args[0])
}

func listModels() error {
	models, err := getAvailableModels()
	if err != nil {
		return err
	}
	
	current, _ := getCurrentModel()
	
	fmt.Println("Claude Code Model Switcher")
	fmt.Println("==========================")
	fmt.Println()
	
	if current == "" {
		fmt.Println("❌ No configuration found")
	} else if current == "custom" {
		fmt.Println("⚙️  Current: Custom configuration")
	} else {
		fmt.Printf("✅ Current: %s\n", current)
	}
	fmt.Println()
	
	if len(models) == 0 {
		fmt.Println("No configuration templates found")
		fmt.Println()
		fmt.Printf("Add new templates as %s/settings.{model}.json\n", configDir)
		return nil
	}
	
	fmt.Println("Available models:")
	for _, model := range models {
		if model == current {
			fmt.Printf("  * %s (active)\n", model)
		} else {
			fmt.Printf("  - %s\n", model)
		}
	}
	
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ccmodel k2           # Switch to k2 configuration")
	fmt.Println("  ccmodel openrouter   # Switch to OpenRouter configuration")
	fmt.Println("  ccmodel list         # List all available models")
	fmt.Println("  ccmodel current      # Show current model")
	
	return nil
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