package execcmd

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	execSessionDirName  = "ccmodel/exec_sessions"
	execSessionEnvKey   = "CCMODEL_EXEC_SESSION"
	execSessionIDEnvKey = "CCMODEL_EXEC_SESSION_ID"
	execTargetEnvKey    = "CCMODEL_EXEC_TARGET"
	execModelEnvKey     = "CCMODEL_ACTIVE_MODEL"
	execConfigEnvKey    = "CCMODEL_ACTIVE_CONFIG"
	execChecksumEnvKey  = "CCMODEL_ACTIVE_CHECKSUM"
	execSessionVersion  = 1
)

// execSessionRecord stores metadata about a proxied execution.
type execSessionRecord struct {
	Version         int              `json:"version"`
	ID              string           `json:"id"`
	Target          string           `json:"target"`
	Binary          string           `json:"binary"`
	CommandLine     string           `json:"command_line"`
	Args            []string         `json:"args,omitempty"`
	WorkingDir      string           `json:"working_dir,omitempty"`
	CreatedAt       string           `json:"created_at"`
	StartedAt       string           `json:"started_at,omitempty"`
	CompletedAt     string           `json:"completed_at,omitempty"`
	DurationSeconds int64            `json:"duration_seconds,omitempty"`
	ExitCode        int              `json:"exit_code,omitempty"`
	Status          string           `json:"status"`
	Error           string           `json:"error,omitempty"`
	PID             int              `json:"pid,omitempty"`
	Model           execSessionModel `json:"model"`
	RunMode         string           `json:"run_mode,omitempty"`
	TmuxSession     string           `json:"tmux_session,omitempty"`
	TmuxWindow      string           `json:"tmux_window,omitempty"`
	LogFile         string           `json:"log_file,omitempty"`
}

// execSessionModel captures model info when the proxy starts.
type execSessionModel struct {
	Name       string `json:"name,omitempty"`
	ConfigPath string `json:"config_path,omitempty"`
	Checksum   string `json:"checksum,omitempty"`
}

func (r *runner) initializeExecSession(target *execTarget, args []string, workingDir string) (*execSessionRecord, string, error) {
	configDir := strings.TrimSpace(r.deps.ConfigDir())
	if configDir == "" {
		return nil, "", fmt.Errorf("config directory is not configured")
	}

	if workingDir == "" {
		if wd, err := os.Getwd(); err == nil {
			workingDir = wd
		}
	}

	createdAt := r.deps.Now()
	record := &execSessionRecord{
		Version:     execSessionVersion,
		ID:          generateSessionID(target.Name, createdAt),
		Target:      target.Name,
		Binary:      target.Binary,
		CommandLine: buildCommandLine(target.Binary, args),
		Args:        append([]string(nil), args...),
		WorkingDir:  workingDir,
		CreatedAt:   createdAt.Format(time.RFC3339Nano),
		Status:      "pending",
	}

	if model, err := r.deps.GetCurrentModel(); err == nil {
		record.Model.Name = model
	} else if model != "" {
		record.Model.Name = model
	}

	configFile := filepath.Join(configDir, "settings.json")
	if info, err := os.Stat(configFile); err == nil && !info.IsDir() {
		record.Model.ConfigPath = configFile
		if checksum, err := r.deps.FileChecksum(configFile); err == nil {
			record.Model.Checksum = checksum
		}
	}

	sessionDir := filepath.Join(configDir, execSessionDirName)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return nil, "", fmt.Errorf("failed to create session directory: %w", err)
	}

	sessionPath := filepath.Join(sessionDir, record.ID+".json")
	if err := saveExecSessionRecord(sessionPath, record); err != nil {
		return nil, "", fmt.Errorf("failed to write session record: %w", err)
	}

	return record, sessionPath, nil
}

func saveExecSessionRecord(path string, record *execSessionRecord) error {
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}

	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, append(data, '\n'), 0o644); err != nil {
		return err
	}

	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	return nil
}

func buildCommandLine(binary string, args []string) string {
	parts := append([]string{binary}, args...)
	quoted := make([]string, len(parts))
	for i, part := range parts {
		quoted[i] = shellQuote(part)
	}
	return strings.Join(quoted, " ")
}

func shellQuote(part string) string {
	if part == "" {
		return "''"
	}
	if strings.ContainsAny(part, " \t\n\"'\\") {
		return "'" + strings.ReplaceAll(part, "'", "'\"'\"'") + "'"
	}
	return part
}

func generateSessionID(target string, at time.Time) string {
	cleaned := strings.ReplaceAll(strings.ToLower(target), string(os.PathSeparator), "-")
	cleaned = strings.ReplaceAll(cleaned, " ", "-")
	suffix := randomHex(3)
	return fmt.Sprintf("%s-%s-%s", cleaned, at.Format("20060102T150405"), suffix)
}

func randomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}
