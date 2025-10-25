package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfigDirUsesUserHomeDir(t *testing.T) {
	tempHome := t.TempDir()
	withUserHomeDir(t, func() (string, error) {
		return tempHome, nil
	})

	got, err := defaultConfigDir()
	if err != nil {
		t.Fatalf("defaultConfigDir() error = %v, want nil", err)
	}

	want := filepath.Join(tempHome, ".claude")
	if got != want {
		t.Fatalf("defaultConfigDir() = %q, want %q", got, want)
	}
}

func TestDefaultConfigDirRejectsEmptyHome(t *testing.T) {
	withUserHomeDir(t, func() (string, error) {
		return "", nil
	})

	got, err := defaultConfigDir()
	if err == nil {
		t.Fatalf("defaultConfigDir() error = nil, want error")
	}
	if got != "" {
		t.Fatalf("defaultConfigDir() = %q, want empty string", got)
	}
	if strings.Contains(err.Error(), ".claude") {
		t.Fatalf("defaultConfigDir() error = %q, want no fallback to .claude", err)
	}
	if !strings.Contains(err.Error(), "user home directory") {
		t.Fatalf("defaultConfigDir() error = %q, want clear home resolution error", err)
	}
}

func TestDefaultConfigDirReturnsHomeResolutionError(t *testing.T) {
	withUserHomeDir(t, func() (string, error) {
		return "", errors.New("home lookup failed")
	})

	got, err := defaultConfigDir()
	if err == nil {
		t.Fatalf("defaultConfigDir() error = nil, want error")
	}
	if got != "" {
		t.Fatalf("defaultConfigDir() = %q, want empty string", got)
	}
	if strings.Contains(err.Error(), ".claude") {
		t.Fatalf("defaultConfigDir() error = %q, want no fallback to .claude", err)
	}
	if !strings.Contains(err.Error(), "user home directory unavailable") {
		t.Fatalf("defaultConfigDir() error = %q, want clear home resolution error", err)
	}
}

func TestCompletionDoesNotRequireConfigInit(t *testing.T) {
	restore := preserveRootState(t)
	defer restore()

	withUserHomeDir(t, func() (string, error) {
		return "", nil
	})

	outputFile, err := os.CreateTemp(t.TempDir(), "completion-*.out")
	if err != nil {
		t.Fatalf("CreateTemp() error = %v", err)
	}
	defer outputFile.Close()

	previousStdout := os.Stdout
	os.Stdout = outputFile
	t.Cleanup(func() {
		os.Stdout = previousStdout
	})

	configDir = ""
	configInitErr = nil
	app = nil

	rootCmd.SetArgs([]string{"completion", "bash"})
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("rootCmd.Execute() error = %v, want nil", err)
	}
	if configDir != "" {
		t.Fatalf("configDir = %q, want empty string", configDir)
	}
	if configInitErr == nil {
		t.Fatalf("configInitErr = nil, want recorded home resolution error")
	}

	data, err := os.ReadFile(outputFile.Name())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	output := string(data)
	if !strings.Contains(output, "ccmodel") || !strings.Contains(output, "completion") {
		t.Fatalf("completion output missing expected content: %q", output)
	}
}

func TestExecuteReturnsConfigInitErrorForEmptyHome(t *testing.T) {
	restore := preserveRootState(t)
	defer restore()

	withUserHomeDir(t, func() (string, error) {
		return "", nil
	})

	configDir = ""
	configInitErr = nil
	app = nil

	rootCmd.SetArgs([]string{"list"})
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true

	err := rootCmd.Execute()
	if err == nil {
		t.Fatalf("rootCmd.Execute() error = nil, want config init error")
	}
	if configDir != "" {
		t.Fatalf("configDir = %q, want empty string", configDir)
	}
	if strings.Contains(err.Error(), ".claude") {
		t.Fatalf("rootCmd.Execute() error = %q, want no fallback to .claude", err)
	}
	if !strings.Contains(err.Error(), "user home directory") {
		t.Fatalf("rootCmd.Execute() error = %q, want clear home resolution error", err)
	}
}

func withUserHomeDir(t *testing.T, fn func() (string, error)) {
	t.Helper()

	previousUserHomeDir := userHomeDir
	t.Cleanup(func() {
		userHomeDir = previousUserHomeDir
	})
	userHomeDir = fn
}

func preserveRootState(t *testing.T) func() {
	t.Helper()

	previousConfigDir := configDir
	previousConfigInitErr := configInitErr
	previousApp := app
	previousArgs := rootCmd.Flags().Args()
	previousSilenceUsage := rootCmd.SilenceUsage
	previousSilenceErrors := rootCmd.SilenceErrors

	return func() {
		configDir = previousConfigDir
		configInitErr = previousConfigInitErr
		app = previousApp
		rootCmd.SetArgs(previousArgs)
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SilenceUsage = previousSilenceUsage
		rootCmd.SilenceErrors = previousSilenceErrors
	}
}
