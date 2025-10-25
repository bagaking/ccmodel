package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	execcmd "github.com/bagaking/ccmodel/cmd/execcmd"
	"github.com/bagaking/cmdux"
	"github.com/spf13/cobra"
)

var (
	configDir     string
	configInitErr error
	verbose       bool
	app           *cmdux.App
	userHomeDir   = os.UserHomeDir
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
	rootCmd.AddCommand(withConfigInit(listCmd))
	rootCmd.AddCommand(withConfigInit(currentCmd))
	rootCmd.AddCommand(withConfigInit(switchCmd))
	rootCmd.AddCommand(withConfigInit(backupCmd))
	rootCmd.AddCommand(withConfigInit(completionCmd))
	rootCmd.AddCommand(withConfigInit(execcmd.NewCommand(execcmd.Dependencies{
		ConfigDir:       func() string { return configDir },
		Verbose:         func() bool { return verbose },
		GetCurrentModel: getCurrentModel,
		FileChecksum:    fileChecksum,
	})))
	// Note: demo command removed as requested
}

func withConfigInit(cmd *cobra.Command) *cobra.Command {
	if cmd.Run != nil {
		run := cmd.Run
		cmd.Run = nil
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			if configInitErr != nil {
				return configInitErr
			}
			run(cmd, args)
			return nil
		}
	}
	if cmd.RunE != nil {
		runE := cmd.RunE
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			if configInitErr != nil {
				return configInitErr
			}
			return runE(cmd, args)
		}
	}
	for _, child := range cmd.Commands() {
		withConfigInit(child)
	}
	return cmd
}

func initConfig() {
	if configDir == "" {
		var err error
		configDir, err = defaultConfigDir()
		if err != nil {
			configInitErr = err
		} else {
			configInitErr = nil
		}
	} else {
		configInitErr = nil
	}

	// Initialize cmdux app
	app = cmdux.New()
}

func defaultConfigDir() (string, error) {
	home, err := userHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve config directory: user home directory unavailable: %w", err)
	}
	if home == "" {
		return "", fmt.Errorf("resolve config directory: user home directory is empty")
	}
	return filepath.Join(home, ".claude"), nil
}

func runRoot(cmd *cobra.Command, args []string) error {
	if configInitErr != nil {
		return configInitErr
	}

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
	candidates, err := listModelCandidates(configDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	return modelCandidateNames(candidates), nil
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

type modelCandidate struct {
	Name string
	Path string
}

type modelCandidateNotFoundError struct {
	model           string
	availableModels []string
}

func (e *modelCandidateNotFoundError) Error() string {
	return "model candidate not found"
}

func (e *modelCandidateNotFoundError) AvailableModels() []string {
	return append([]string(nil), e.availableModels...)
}

func listModelCandidates(dir string) ([]modelCandidate, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	candidates := []modelCandidate{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			continue
		}

		name := entry.Name()
		model := extractModelName(name)
		if model == "" || model == "json" {
			continue
		}

		candidates = append(candidates, modelCandidate{
			Name: model,
			Path: filepath.Join(dir, name),
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Name < candidates[j].Name
	})
	return candidates, nil
}

func modelCandidateNames(candidates []modelCandidate) []string {
	models := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		models = append(models, candidate.Name)
	}
	return models
}

func resolveModelCandidatePath(dir, model string) (string, error) {
	candidates, err := listModelCandidates(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", &modelCandidateNotFoundError{model: model}
		}
		return "", err
	}

	for _, candidate := range candidates {
		if candidate.Name == model {
			return candidate.Path, nil
		}
	}

	return "", &modelCandidateNotFoundError{
		model:           model,
		availableModels: modelCandidateNames(candidates),
	}
}
