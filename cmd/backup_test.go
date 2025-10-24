package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestBackupCurrentPreservesRawCurrentConfig(t *testing.T) {
	tempDir := t.TempDir()
	previousConfigDir := configDir
	t.Cleanup(func() {
		configDir = previousConfigDir
	})
	configDir = tempDir

	currentConfig := []byte("{invalid json with __cc and raw spacing\n\t")
	currentFile := filepath.Join(tempDir, "settings.json")
	if err := os.WriteFile(currentFile, currentConfig, 0644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v, want nil", currentFile, err)
	}

	if err := backupCurrent(); err != nil {
		t.Fatalf("backupCurrent() error = %v, want nil", err)
	}

	backupDir := filepath.Join(tempDir, "backups")
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("os.ReadDir(%q) error = %v, want nil", backupDir, err)
	}
	if len(entries) != 1 {
		t.Fatalf("os.ReadDir(%q) entries = %d, want 1", backupDir, len(entries))
	}
	assertMode(t, backupDir, userOnlyDirMode)

	backupFile := filepath.Join(backupDir, entries[0].Name())
	backupData, err := os.ReadFile(backupFile)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v, want nil", backupFile, err)
	}
	if !bytes.Equal(backupData, currentConfig) {
		t.Errorf("backupCurrent() backup = %q, want %q", backupData, currentConfig)
	}
	assertMode(t, backupFile, userOnlyFileMode)
}
