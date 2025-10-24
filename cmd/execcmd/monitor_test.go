package execcmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatusCommandOutputsSessionJSON(t *testing.T) {
	tempDir := t.TempDir()
	sessionDir := filepath.Join(tempDir, execSessionDirName)
	if err := ensureUserOnlyDir(sessionDir); err != nil {
		t.Fatalf("ensureUserOnlyDir(%q) error = %v, want nil", sessionDir, err)
	}

	workDir := filepath.Join(tempDir, "workspace")
	record := &execSessionRecord{
		Version:     execSessionVersion,
		ID:          "codex-session",
		Target:      "codex",
		WorkingDir:  workDir,
		Status:      "running",
		RunMode:     runModeTmux,
		TmuxSession: "ccmodel-workspace",
		TmuxWindow:  "codex",
	}
	if err := saveExecSessionRecord(filepath.Join(sessionDir, record.ID+".json"), record); err != nil {
		t.Fatalf("saveExecSessionRecord() error = %v, want nil", err)
	}

	r := newRunner(Dependencies{
		ConfigDir: func() string { return tempDir },
		GetCurrentModel: func() (string, error) {
			return "codex-test", nil
		},
		FileChecksum: func(string) (string, error) {
			return "", nil
		},
		LookPath: func(string) (string, error) {
			return "", os.ErrNotExist
		},
	})

	cmd := r.buildStatusCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("status --json error = %v, output:\n%s", err, out.String())
	}

	var payload execStatusJSON
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal(status output) error = %v, output:\n%s", err, out.String())
	}
	if len(payload.Sessions) != 1 {
		t.Fatalf("len(sessions) = %d, want 1; payload = %+v", len(payload.Sessions), payload)
	}
	session := payload.Sessions[0]
	if session.Name != "ccmodel-workspace" {
		t.Errorf("session.name = %q, want %q", session.Name, "ccmodel-workspace")
	}
	if session.Dir != workDir {
		t.Errorf("session.dir = %q, want %q", session.Dir, workDir)
	}
	if session.Running {
		t.Errorf("session.running = true, want false without live tmux")
	}
	if !session.HasHistory {
		t.Errorf("session.has_history = false, want true")
	}
	if len(session.Windows) != 1 {
		t.Fatalf("len(session.windows) = %d, want 1", len(session.Windows))
	}
	if session.Windows[0].Name != "codex" {
		t.Errorf("window.name = %q, want %q", session.Windows[0].Name, "codex")
	}
}

func TestStatusLogsJSONReturnsUnsupportedScopeError(t *testing.T) {
	r := newRunner(Dependencies{
		ConfigDir: func() string { return t.TempDir() },
		GetCurrentModel: func() (string, error) {
			return "codex-test", nil
		},
		FileChecksum: func(string) (string, error) {
			return "", nil
		},
		LookPath: func(string) (string, error) {
			return "", os.ErrNotExist
		},
	})

	cmd := r.buildStatusCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"logs", "--json"})
	err := cmd.Execute()
	if err == nil {
		t.Fatalf("status logs --json error = nil, want unsupported scope error")
	}
	if !strings.Contains(err.Error(), "status logs does not support --json") {
		t.Fatalf("status logs --json error = %v, want unsupported scope error", err)
	}
}
