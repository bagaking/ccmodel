package execcmd

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
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
	settingsPath := filepath.Join(tempDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"model":"test"}`), 0o644); err != nil {
		t.Fatalf("failed to create settings file: %v", err)
	}

	baseTime := time.Date(2024, 10, 12, 15, 4, 5, 0, time.UTC)
	counter := 0

	r := newRunner(Dependencies{
		ConfigDir: func() string { return tempDir },
		Verbose:   func() bool { return false },
		GetCurrentModel: func() (string, error) {
			return "claude-sonnet", nil
		},
		FileChecksum: func(path string) (string, error) {
			data, err := os.ReadFile(path)
			if err != nil {
				return "", err
			}
			sum := md5.Sum(data)
			return hex.EncodeToString(sum[:]), nil
		},
		Now: func() time.Time {
			defer func() { counter++ }()
			return baseTime.Add(time.Duration(counter) * time.Second)
		},
		LookPath: func(name string) (string, error) {
			return "/usr/bin/" + name, nil
		},
		Exit: func(int) {},
	})

	workDir := filepath.Join(tempDir, "project")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("failed to create work dir: %v", err)
	}

	target := &execTarget{
		Name:        "claude",
		Binary:      "/usr/bin/claude",
		DisplayName: "claude",
		EnvVar:      "CCMODEL_EXEC_CLAUDE",
	}

	record, sessionPath, err := r.initializeExecSession(target, []string{"--foo", "bar"}, workDir)
	if err != nil {
		t.Fatalf("initializeExecSession failed: %v", err)
	}

	if record.WorkingDir != workDir {
		t.Fatalf("expected working dir %s, got %s", workDir, record.WorkingDir)
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

func TestExecuteRunDirect(t *testing.T) {
	tempDir := t.TempDir()
	settingsPath := filepath.Join(tempDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"model":"test"}`), 0o644); err != nil {
		t.Fatalf("failed to create settings file: %v", err)
	}

	scriptPath := filepath.Join(tempDir, "codex-script.sh")
	script := "#!/bin/sh\nexit 3\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	r := newRunner(Dependencies{
		ConfigDir: func() string { return tempDir },
		Verbose:   func() bool { return false },
		GetCurrentModel: func() (string, error) {
			return "codex-test", nil
		},
		FileChecksum: func(path string) (string, error) {
			data, err := os.ReadFile(path)
			if err != nil {
				return "", err
			}
			sum := md5.Sum(data)
			return hex.EncodeToString(sum[:]), nil
		},
		Now: func() time.Time { return time.Date(2024, 10, 12, 0, 0, 0, 0, time.UTC) },
		LookPath: func(name string) (string, error) {
			switch name {
			case "codex":
				return scriptPath, nil
			case "tmux":
				return "", os.ErrNotExist
			default:
				return "/usr/bin/" + name, nil
			}
		},
		Exit: func(int) {},
	})

	exitCode, err := r.executeRun("codex", nil, runOptions{
		UseTmux:             false,
		AllowDirectFallback: true,
	})
	if err != nil {
		t.Fatalf("executeRun returned error: %v", err)
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
	if stored.RunMode != runModeDirect {
		t.Fatalf("expected run mode %s, got %s", runModeDirect, stored.RunMode)
	}
}
