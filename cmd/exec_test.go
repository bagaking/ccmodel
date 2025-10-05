package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestShellQuote(t *testing.T) {
	if got := shellQuote("hello"); got != "hello" {
		t.Fatalf("expected no quoting, got %q", got)
	}

	if got := shellQuote("complex value"); got != "'complex value'" {
		t.Fatalf("unexpected quoted value: %q", got)
	}

	if got := shellQuote("quote'"); got != "'quote'\"'\"''" {
		t.Fatalf("unexpected single quote escaping: %q", got)
	}
}

func TestBuildCommandLine(t *testing.T) {
	line := buildCommandLine("/bin/echo", []string{"hello", "complex value", "quote'"})
	expected := "/bin/echo hello 'complex value' 'quote'\"'\"''"
	if line != expected {
		t.Fatalf("unexpected command line: %q", line)
	}
}

func TestInitializeExecSession(t *testing.T) {
	tempDir := t.TempDir()
	originalConfigDir := configDir
	configDir = tempDir
	t.Cleanup(func() { configDir = originalConfigDir })

	settingsPath := filepath.Join(tempDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"model":"test"}`), 0o644); err != nil {
		t.Fatalf("failed to create settings file: %v", err)
	}

	baseTime := time.Date(2024, 10, 12, 15, 4, 5, 0, time.UTC)
	originalNow := nowFunc
	counter := 0
	nowFunc = func() time.Time {
		defer func() { counter++ }()
		return baseTime.Add(time.Duration(counter) * time.Second)
	}
	t.Cleanup(func() { nowFunc = originalNow })

	target := &execTarget{
		Name:        "claude",
		Binary:      "/usr/bin/claude",
		DisplayName: "claude",
		EnvVar:      "CCMODEL_EXEC_CLAUDE",
	}

	record, sessionPath, err := initializeExecSession(target, []string{"--foo", "bar"})
	if err != nil {
		t.Fatalf("initializeExecSession failed: %v", err)
	}

	if record.ID == "" {
		t.Fatal("expected session ID to be set")
	}

	expectedDir := filepath.Join(tempDir, execSessionDirName)
	if !strings.HasPrefix(sessionPath, expectedDir) {
		t.Fatalf("expected session path under %s, got %s", expectedDir, sessionPath)
	}

	data, err := os.ReadFile(sessionPath)
	if err != nil {
		t.Fatalf("failed to read session file: %v", err)
	}

	var stored execSessionRecord
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatalf("failed to decode session record: %v", err)
	}

	if stored.Model.ConfigPath != settingsPath {
		t.Fatalf("expected config path %s, got %s", settingsPath, stored.Model.ConfigPath)
	}

	if stored.Status != "pending" {
		t.Fatalf("expected status pending, got %s", stored.Status)
	}
}

func TestExecuteProxyCommandExitCode(t *testing.T) {
	tempDir := t.TempDir()
	originalConfigDir := configDir
	configDir = tempDir
	t.Cleanup(func() { configDir = originalConfigDir })

	scriptPath := filepath.Join(tempDir, "claude-script.sh")
	script := "#!/bin/sh\nexit 3\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	t.Setenv("CCMODEL_EXEC_CLAUDE", scriptPath)

	originalNow := nowFunc
	nowFunc = func() time.Time { return time.Date(2024, 10, 12, 0, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { nowFunc = originalNow })

	exitCode, err := executeProxyCommand("claude", []string{})
	if err != nil {
		t.Fatalf("executeProxyCommand returned error: %v", err)
	}
	if exitCode != 3 {
		t.Fatalf("expected exit code 3, got %d", exitCode)
	}

	sessionDir := filepath.Join(tempDir, execSessionDirName)
	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		t.Fatalf("failed to read session dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 session record, got %d", len(entries))
	}

	sessionPath := filepath.Join(sessionDir, entries[0].Name())
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		t.Fatalf("failed to read session record: %v", err)
	}

	var stored execSessionRecord
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatalf("failed to parse session record: %v", err)
	}

	if stored.ExitCode != 3 {
		t.Fatalf("expected stored exit code 3, got %d", stored.ExitCode)
	}
	if stored.Status != "failed" {
		t.Fatalf("expected status failed for non-zero exit, got %s", stored.Status)
	}
}

func TestRunExecShowsHelpForFlagOnly(t *testing.T) {
	var out bytes.Buffer
	cmd := &cobra.Command{
		Use:               execCmd.Use,
		Short:             execCmd.Short,
		Long:              execCmd.Long,
		DisableFlagParsing: true,
		SilenceUsage:      true,
	}
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	err := runExec(cmd, []string{"-f"})
	if err == nil {
		t.Fatal("expected error for flag-only invocation")
	}
	if !strings.Contains(out.String(), "ccmodel exec") {
		t.Fatalf("expected help output, got %q", out.String())
	}
}
