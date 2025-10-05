package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	execSessionDirName   = "ccmodel/exec_sessions"
	execSessionEnvKey    = "CCMODEL_EXEC_SESSION"
	execSessionIDEnvKey  = "CCMODEL_EXEC_SESSION_ID"
	execTargetEnvKey     = "CCMODEL_EXEC_TARGET"
	execModelEnvKey      = "CCMODEL_ACTIVE_MODEL"
	execConfigEnvKey     = "CCMODEL_ACTIVE_CONFIG"
	execChecksumEnvKey   = "CCMODEL_ACTIVE_CHECKSUM"
	execSessionVersion   = 1
)

var (
	execTargetAlias = map[string]string{
		"claude": "claude",
		"codex":  "codex",
		"code":   "codex",
	}

	execTargetDefaults = map[string]execTargetConfig{
		"claude": {DefaultBinary: "claude", EnvVar: "CCMODEL_EXEC_CLAUDE"},
		"codex":  {DefaultBinary: "codex", EnvVar: "CCMODEL_EXEC_CODEX"},
	}

	execLookPath = exec.LookPath
	exitFunc     = os.Exit
	nowFunc      = time.Now
)

// execTargetConfig describes how to resolve a proxy target.
type execTargetConfig struct {
	DefaultBinary string
	EnvVar        string
}

// execTarget represents a resolved proxy target.
type execTarget struct {
	Name        string
	Binary      string
	DisplayName string
	EnvVar      string
}

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
}

// execSessionModel captures model info when the proxy starts.
type execSessionModel struct {
	Name       string `json:"name,omitempty"`
	ConfigPath string `json:"config_path,omitempty"`
	Checksum   string `json:"checksum,omitempty"`
}

var execCmd = &cobra.Command{
	Use:                "exec <claude|codex> [-- <args>]",
	Short:              "Run a Claude/Codex CLI through ccmodel with session tracking",
	Long:               "Proxy execution that launches the native Claude or Codex CLI, while recording session metadata for later restoration.",
	DisableFlagParsing: true,
	SilenceUsage:       true,
	RunE:               runExec,
	ValidArgsFunction:  execCompletion,
}

func showExecHelp(cmd *cobra.Command) {
	if err := cmd.Help(); err != nil {
		logExecWarning("failed to display help: %v", err)
	}
}

func runExec(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		showExecHelp(cmd)
		return errors.New("target required: use 'claude' or 'codex'")
	}

	if args[0] == "-h" || args[0] == "--help" {
		return cmd.Help()
	}

	targetName := args[0]
	if strings.HasPrefix(targetName, "-") {
		showExecHelp(cmd)
		return fmt.Errorf("unknown target %q (expected claude or codex)", targetName)
	}
	targetArgs := []string{}
	if len(args) > 1 {
		if args[1] == "--" {
			targetArgs = append(targetArgs, args[2:]...)
		} else {
			targetArgs = append(targetArgs, args[1:]...)
		}
	}

	exitCode, err := executeProxyCommand(targetName, targetArgs)
	if err != nil {
		return err
	}

	if exitCode != 0 {
		exitFunc(exitCode)
	}
	return nil
}

func executeProxyCommand(targetName string, targetArgs []string) (int, error) {
	resolved, err := resolveExecTarget(targetName)
	if err != nil {
		return -1, err
	}

	record, sessionPath, err := initializeExecSession(resolved, targetArgs)
	if err != nil {
		return -1, err
	}

	cmd := exec.Command(resolved.Binary, targetArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	env := append([]string{}, os.Environ()...)
	env = append(env,
		fmt.Sprintf("%s=%s", execTargetEnvKey, resolved.Name),
		fmt.Sprintf("%s=%s", execSessionIDEnvKey, record.ID),
		fmt.Sprintf("%s=%s", execSessionEnvKey, sessionPath),
	)
	if record.Model.Name != "" {
		env = append(env, fmt.Sprintf("%s=%s", execModelEnvKey, record.Model.Name))
	}
	if record.Model.ConfigPath != "" {
		env = append(env, fmt.Sprintf("%s=%s", execConfigEnvKey, record.Model.ConfigPath))
	}
	if record.Model.Checksum != "" {
		env = append(env, fmt.Sprintf("%s=%s", execChecksumEnvKey, record.Model.Checksum))
	}
	cmd.Env = env

	startErr := cmd.Start()
	if startErr != nil {
		record.Status = "error"
		record.Error = startErr.Error()
		record.ExitCode = -1
		record.CompletedAt = nowFunc().Format(time.RFC3339Nano)
		if err := saveExecSessionRecord(sessionPath, record); err != nil {
			logExecWarning("session update failed: %v", err)
		}
		return -1, fmt.Errorf("failed to start %s: %w", resolved.DisplayName, startErr)
	}

	record.PID = cmd.Process.Pid
	record.StartedAt = nowFunc().Format(time.RFC3339Nano)
	record.Status = "running"
	if err := saveExecSessionRecord(sessionPath, record); err != nil {
		logExecWarning("session update failed: %v", err)
	}

	waitErr := cmd.Wait()
	completedAt := nowFunc()
	record.CompletedAt = completedAt.Format(time.RFC3339Nano)
	if record.StartedAt != "" {
		if startedAt, err := time.Parse(time.RFC3339Nano, record.StartedAt); err == nil {
			if duration := completedAt.Sub(startedAt); duration > 0 {
				record.DurationSeconds = int64(duration.Seconds())
			}
		}
	}

	exitCode := 0
	if state := cmd.ProcessState; state != nil {
		exitCode = state.ExitCode()
	} else if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	record.ExitCode = exitCode

	if waitErr != nil {
		record.Status = "failed"
		record.Error = waitErr.Error()
	} else {
		record.Status = "success"
	}

	if err := saveExecSessionRecord(sessionPath, record); err != nil {
		logExecWarning("session update failed: %v", err)
	}

	if waitErr != nil {
		if _, ok := waitErr.(*exec.ExitError); ok {
			return exitCode, nil
		}
		return exitCode, fmt.Errorf("failed to execute %s: %w", resolved.DisplayName, waitErr)
	}

	return exitCode, nil
}

func resolveExecTarget(name string) (*execTarget, error) {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if normalized == "" {
		return nil, errors.New("empty target")
	}

	key, ok := execTargetAlias[normalized]
	if !ok {
		return nil, fmt.Errorf("unknown target %q (expected claude or codex)", name)
	}

	cfg := execTargetDefaults[key]
	commandName := cfg.DefaultBinary
	if envValue := strings.TrimSpace(os.Getenv(cfg.EnvVar)); envValue != "" {
		commandName = envValue
	}

	resolvedPath, err := execLookPath(commandName)
	if err != nil {
		return nil, fmt.Errorf("executable for %s not found (looked for %q). Set %s to override", key, commandName, cfg.EnvVar)
	}

	return &execTarget{
		Name:        key,
		Binary:      resolvedPath,
		DisplayName: commandName,
		EnvVar:      cfg.EnvVar,
	}, nil
}

func initializeExecSession(target *execTarget, args []string) (*execSessionRecord, string, error) {
	createdAt := nowFunc()
	record := &execSessionRecord{
		Version:     execSessionVersion,
		ID:          generateSessionID(target.Name, createdAt),
		Target:      target.Name,
		Binary:      target.Binary,
		CommandLine: buildCommandLine(target.Binary, args),
		Args:        append([]string(nil), args...),
		CreatedAt:   createdAt.Format(time.RFC3339Nano),
		Status:      "pending",
	}

	if wd, err := os.Getwd(); err == nil {
		record.WorkingDir = wd
	}

	if model, err := getCurrentModel(); err == nil {
		record.Model.Name = model
	} else if model != "" {
		record.Model.Name = model
	}

	configFile := filepath.Join(configDir, "settings.json")
	if info, err := os.Stat(configFile); err == nil && !info.IsDir() {
		record.Model.ConfigPath = configFile
		if checksum, err := fileChecksum(configFile); err == nil {
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

func execCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	candidates := []string{"claude", "codex"}
	matches := make([]string, 0, len(candidates))
	for _, c := range candidates {
		if strings.HasPrefix(c, strings.ToLower(toComplete)) {
			matches = append(matches, c)
		}
	}
	return matches, cobra.ShellCompDirectiveNoFileComp
}

func logExecWarning(format string, args ...any) {
	if !verbose {
		return
	}
	fmt.Fprintf(os.Stderr, "ccmodel exec: "+format+"\n", args...)
}
