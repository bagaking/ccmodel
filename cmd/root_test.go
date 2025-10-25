package cmd

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfigDirUsesUserHomeDir(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

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
	t.Setenv("HOME", "")

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
	previousUserHomeDir := userHomeDir
	t.Cleanup(func() {
		userHomeDir = previousUserHomeDir
	})
	userHomeDir = func() (string, error) {
		return "", errors.New("home lookup failed")
	}

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

func TestExecuteReturnsConfigInitErrorForEmptyHome(t *testing.T) {
	previousConfigDir := configDir
	previousConfigInitErr := configInitErr
	previousApp := app
	previousUserHomeDir := userHomeDir
	previousArgs := rootCmd.Flags().Args()
	previousSilenceUsage := rootCmd.SilenceUsage
	previousSilenceErrors := rootCmd.SilenceErrors
	t.Cleanup(func() {
		configDir = previousConfigDir
		configInitErr = previousConfigInitErr
		app = previousApp
		userHomeDir = previousUserHomeDir
		rootCmd.SetArgs(previousArgs)
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SilenceUsage = previousSilenceUsage
		rootCmd.SilenceErrors = previousSilenceErrors
	})

	t.Setenv("HOME", "")
	configDir = ""
	configInitErr = nil
	app = nil
	userHomeDir = func() (string, error) {
		return "", nil
	}

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
