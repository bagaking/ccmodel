package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPrivateFileHelpersUseUserOnlyPermissions(t *testing.T) {
	tempDir := t.TempDir()

	privateDir := filepath.Join(tempDir, "private")
	if err := ensureUserOnlyDir(privateDir); err != nil {
		t.Fatalf("ensureUserOnlyDir(%q) error = %v, want nil", privateDir, err)
	}
	assertMode(t, privateDir, userOnlyDirMode)

	privateFile := filepath.Join(privateDir, "settings.json")
	if err := writeUserOnlyFile(privateFile, []byte(`{"apiKey":"secret"}`)); err != nil {
		t.Fatalf("writeUserOnlyFile(%q) error = %v, want nil", privateFile, err)
	}
	assertMode(t, privateFile, userOnlyFileMode)

	appendFile := filepath.Join(privateDir, "history.jsonl")
	file, err := openUserOnlyAppendFile(appendFile)
	if err != nil {
		t.Fatalf("openUserOnlyAppendFile(%q) error = %v, want nil", appendFile, err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close(%q) error = %v, want nil", appendFile, err)
	}
	assertMode(t, appendFile, userOnlyFileMode)

	lockFile := filepath.Join(privateDir, "settings.json.lock")
	lock, err := createUserOnlyExclusiveFile(lockFile)
	if err != nil {
		t.Fatalf("createUserOnlyExclusiveFile(%q) error = %v, want nil", lockFile, err)
	}
	if err := lock.Close(); err != nil {
		t.Fatalf("Close(%q) error = %v, want nil", lockFile, err)
	}
	assertMode(t, lockFile, userOnlyFileMode)
}

func TestWriteUserOnlyFileReplacesExistingWideFileWithUserOnlyFile(t *testing.T) {
	tempDir := t.TempDir()
	privateFile := filepath.Join(tempDir, "settings.json")
	oldData := []byte(`{"model":"old"}`)
	newData := []byte(`{"model":"new"}`)

	if err := os.WriteFile(privateFile, oldData, 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q, oldData, 0644) error = %v, want nil", privateFile, err)
	}
	assertMode(t, privateFile, 0o644)

	if err := writeUserOnlyFile(privateFile, newData); err != nil {
		t.Fatalf("writeUserOnlyFile(%q, newData) error = %v, want nil", privateFile, err)
	}

	got, err := os.ReadFile(privateFile)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v, want nil", privateFile, err)
	}
	if string(got) != string(newData) {
		t.Errorf("writeUserOnlyFile(%q) content = %q, want %q", privateFile, got, newData)
	}
	assertMode(t, privateFile, userOnlyFileMode)
}

func TestFileConfigStorageWritesUserOnlyFiles(t *testing.T) {
	tempDir := t.TempDir()
	mockLogger := &MockLogger{}
	mockLockManager := &MockFileLockManager{}
	storage := NewFileConfigStorage(tempDir, mockLockManager, mockLogger)

	if err := storage.SaveActiveConfig([]byte(`{"apiKey":"secret"}`)); err != nil {
		t.Fatalf("SaveActiveConfig() error = %v, want nil", err)
	}

	settingsFile := filepath.Join(tempDir, "settings.json")
	assertMode(t, tempDir, userOnlyDirMode)
	assertMode(t, settingsFile, userOnlyFileMode)

	if err := storage.BackupActiveConfig(); err != nil {
		t.Fatalf("BackupActiveConfig() error = %v, want nil", err)
	}

	ccmodelDir := filepath.Join(tempDir, "ccmodel")
	backupDir := filepath.Join(ccmodelDir, "backups")
	assertMode(t, ccmodelDir, userOnlyDirMode)
	assertMode(t, backupDir, userOnlyDirMode)

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("os.ReadDir(%q) error = %v, want nil", backupDir, err)
	}
	if len(entries) != 1 {
		t.Fatalf("os.ReadDir(%q) entries = %d, want 1", backupDir, len(entries))
	}
	assertMode(t, filepath.Join(backupDir, entries[0].Name()), userOnlyFileMode)
}

func TestHistoryManagersWriteUserOnlyFiles(t *testing.T) {
	tempDir := t.TempDir()

	quotaHistory := &QuotaHistoryManager{
		historyDir:        filepath.Join(tempDir, ".claude", "ccmodel", "quota_history"),
		lastValues:        make(map[string]*QuotaInfo),
		lastHeartbeat:     make(map[string]time.Time),
		heartbeatInterval: time.Minute,
	}
	if err := quotaHistory.Initialize(); err != nil {
		t.Fatalf("QuotaHistoryManager.Initialize() error = %v, want nil", err)
	}
	if err := quotaHistory.RecordQuota("demo", &QuotaInfo{Total: 100, Used: 25}); err != nil {
		t.Fatalf("QuotaHistoryManager.RecordQuota() error = %v, want nil", err)
	}

	claudeDir := filepath.Join(tempDir, ".claude")
	ccmodelDir := filepath.Join(claudeDir, "ccmodel")
	assertMode(t, claudeDir, userOnlyDirMode)
	assertMode(t, ccmodelDir, userOnlyDirMode)
	assertMode(t, quotaHistory.historyDir, userOnlyDirMode)
	assertOnlyEntryMode(t, quotaHistory.historyDir, ".jsonl", userOnlyFileMode)

	switchHistory := &SwitchHistoryManager{
		historyDir: filepath.Join(tempDir, ".claude", "ccmodel", "switch_history"),
	}
	if err := switchHistory.Initialize(); err != nil {
		t.Fatalf("SwitchHistoryManager.Initialize() error = %v, want nil", err)
	}
	if err := switchHistory.RecordSwitch("old", "new", time.Second, nil); err != nil {
		t.Fatalf("SwitchHistoryManager.RecordSwitch() error = %v, want nil", err)
	}

	assertMode(t, switchHistory.historyDir, userOnlyDirMode)
	assertOnlyEntryMode(t, switchHistory.historyDir, ".jsonl", userOnlyFileMode)
}

func TestFileLockManagerCreatesUserOnlyLockFiles(t *testing.T) {
	tempDir := t.TempDir()
	mockLogger := &MockLogger{}
	lockManager := NewFileLockManager(mockLogger)
	lockFile := filepath.Join(tempDir, "settings.json.lock")

	err := lockManager.WithLock(lockFile, func() error {
		assertMode(t, lockFile, userOnlyFileMode)
		return nil
	})
	if err != nil {
		t.Fatalf("WithLock(%q) error = %v, want nil", lockFile, err)
	}
}

func assertOnlyEntryMode(t *testing.T, dir, suffix string, want os.FileMode) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("os.ReadDir(%q) error = %v, want nil", dir, err)
	}
	var matches int
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != suffix {
			continue
		}
		matches++
		assertMode(t, filepath.Join(dir, entry.Name()), want)
	}
	if matches != 1 {
		t.Fatalf("os.ReadDir(%q) entries with suffix %q = %d, want 1", dir, suffix, matches)
	}
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
