package execcmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrivateFileHelpersUseUserOnlyPermissions(t *testing.T) {
	tempDir := t.TempDir()

	privateDir := filepath.Join(tempDir, "private")
	if err := ensureUserOnlyDir(privateDir); err != nil {
		t.Fatalf("ensureUserOnlyDir(%q) error = %v, want nil", privateDir, err)
	}
	assertMode(t, privateDir, userOnlyDirMode)

	privateFile := filepath.Join(privateDir, "session.json")
	if err := writeUserOnlyFile(privateFile, []byte(`{"working_dir":"workspace"}`)); err != nil {
		t.Fatalf("writeUserOnlyFile(%q) error = %v, want nil", privateFile, err)
	}
	assertMode(t, privateFile, userOnlyFileMode)

	logFile := filepath.Join(privateDir, "session.log")
	if err := createUserOnlyFile(logFile); err != nil {
		t.Fatalf("createUserOnlyFile(%q) error = %v, want nil", logFile, err)
	}
	assertMode(t, logFile, userOnlyFileMode)
}

func TestSaveExecSessionRecordUsesUserOnlyPermissions(t *testing.T) {
	tempDir := t.TempDir()
	sessionPath := filepath.Join(tempDir, "session.json")
	record := &execSessionRecord{
		Version:    execSessionVersion,
		ID:         "codex-20240101T000000-abcdef",
		Target:     "codex",
		Binary:     filepath.Join("bin", "codex"),
		WorkingDir: filepath.Join("private", "workspace"),
		CreatedAt:  "2024-01-01T00:00:00Z",
		Status:     "pending",
	}

	if err := saveExecSessionRecord(sessionPath, record); err != nil {
		t.Fatalf("saveExecSessionRecord(%q) error = %v, want nil", sessionPath, err)
	}

	assertMode(t, sessionPath, userOnlyFileMode)
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("os.Stat(%q) error = %v, want nil", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Errorf("mode(%q) = %04o, want %04o", path, got, want)
	}
}
