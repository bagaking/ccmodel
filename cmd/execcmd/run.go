package execcmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	runModeTmux   = "tmux"
	runModeDirect = "direct"

	defaultTmuxSessionEnv = "CCMODEL_EXEC_TMUX_SESSION"
	defaultTmuxSession    = "ccmodel-exec"
)

type attachBehavior int

const (
	attachAuto attachBehavior = iota
	attachAlways
	attachNever
)

type runOptions struct {
	UseTmux             bool
	RequireTmux         bool
	AllowDirectFallback bool
	WindowName          string
	AttachBehavior      attachBehavior
	WorkingDir          string
	SessionName         string
	PostLaunch          func(sessionName, windowName, windowIndex string)
	ExtraEnv            map[string]string
}

func (r *runner) executeRun(targetName string, targetArgs []string, opts runOptions) (int, error) {
	resolved, err := resolveExecTarget(targetName, r.deps.LookPath)
	if err != nil {
		return -1, err
	}

	record, sessionPath, err := r.initializeExecSession(resolved, targetArgs, opts.WorkingDir)
	if err != nil {
		return -1, err
	}

	if opts.UseTmux {
		tmuxPath, tmuxErr := r.deps.LookPath("tmux")
		if tmuxErr != nil {
			if opts.RequireTmux || !opts.AllowDirectFallback {
				return -1, fmt.Errorf("tmux required but not found: %w", tmuxErr)
			}
			opts.UseTmux = false
		} else {
			code, err := r.runWithTmux(resolved, targetArgs, record, sessionPath, tmuxPath, opts)
			if err == nil {
				return code, nil
			}
			if opts.AllowDirectFallback {
				r.logExecWarning("tmux launch failed (%v), falling back to direct execution", err)
			} else {
				return -1, err
			}
		}
	}

	return r.runDirect(resolved, targetArgs, record, sessionPath, opts)
}

func (r *runner) runWithTmux(resolved *execTarget, targetArgs []string, record *execSessionRecord, sessionPath, tmuxPath string, opts runOptions) (int, error) {
	sessionName := strings.TrimSpace(opts.SessionName)
	if sessionName == "" {
		sessionName = strings.TrimSpace(os.Getenv(defaultTmuxSessionEnv))
		if sessionName == "" {
			sessionName = defaultTmuxSession
		}
	}

	windowName := strings.TrimSpace(opts.WindowName)
	if windowName == "" {
		windowName = r.defaultWindowName(resolved.Name, record.WorkingDir)
	}

	logDir := filepath.Join(strings.TrimSpace(r.deps.ConfigDir()), execSessionDirName, "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return -1, fmt.Errorf("failed to create log directory: %w", err)
	}
	logFile := filepath.Join(logDir, record.ID+".log")

	if err := ensureTmuxSession(tmuxPath, sessionName); err != nil {
		return -1, err
	}
	configureTmuxSession(tmuxPath, sessionName)

	if len(opts.ExtraEnv) > 0 {
		if err := syncTmuxEnvironment(tmuxPath, sessionName, opts.ExtraEnv); err != nil {
			r.logExecWarning("failed to sync tmux environment: %v", err)
		}
	}

	commandLine := buildCommandLine(resolved.Binary, targetArgs)

	windowIndex, err := tmuxNewWindow(tmuxPath, sessionName, windowName, record.WorkingDir, commandLine)
	if err != nil {
		return -1, err
	}

	if err := tmuxPipePane(tmuxPath, sessionName, windowIndex, logFile); err != nil {
		r.logExecWarning("failed to configure log pipe: %v", err)
	}

	now := r.deps.Now()
	record.Status = "running"
	record.StartedAt = now.Format(time.RFC3339Nano)
	record.RunMode = runModeTmux
	record.TmuxSession = sessionName
	record.TmuxWindow = windowName
	record.LogFile = logFile
	record.ExitCode = 0
	record.Error = ""

	if err := saveExecSessionRecord(sessionPath, record); err != nil {
		r.logExecWarning("failed to update session record: %v", err)
	}

	if opts.PostLaunch != nil {
		opts.PostLaunch(sessionName, windowName, windowIndex)
	}

	if err := r.handleAttachBehavior(tmuxPath, sessionName, windowIndex, opts.AttachBehavior); err != nil {
		r.logExecWarning("attach failed: %v", err)
	}

	return 0, nil
}

func (r *runner) handleAttachBehavior(tmuxPath, sessionName, windowIndex string, behavior attachBehavior) error {
	switch behavior {
	case attachNever:
		return nil
	case attachAlways:
		return attachToWindow(tmuxPath, sessionName, windowIndex, true)
	case attachAuto:
		return attachToWindow(tmuxPath, sessionName, windowIndex, true)
	default:
		return nil
	}
}

func (r *runner) defaultWindowName(target, workingDir string) string {
	base := filepath.Base(workingDir)
	if base == "." || base == string(filepath.Separator) || base == "" {
		base = "session"
	}
	base = strings.ReplaceAll(base, " ", "-")
	return fmt.Sprintf("%s-%s-%s", target, base, randomHex(2))
}

func ensureTmuxSession(tmuxPath, sessionName string) error {
	check := exec.Command(tmuxPath, "has-session", "-t", sessionName)
	if err := check.Run(); err == nil {
		return nil
	}

	cmd := exec.Command(tmuxPath, "new-session", "-d", "-s", sessionName, "-n", "home")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create tmux session %q: %w", sessionName, err)
	}
	return nil
}

func tmuxNewWindow(tmuxPath, sessionName, windowName, workDir, commandLine string) (string, error) {
	args := []string{"new-window", "-t", sessionName, "-P", "-F", "#{window_index}", "-n", windowName}
	if workDir != "" {
		args = append(args, "-c", workDir)
	}
	args = append(args, commandLine)

	cmd := exec.Command(tmuxPath, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tmux new-window failed: %s", strings.TrimSpace(out.String()))
	}

	index := strings.TrimSpace(out.String())
	if index == "" {
		index = "last"
	}
	return index, nil
}

func tmuxPipePane(tmuxPath, sessionName, windowIndex, logFile string) error {
	target := fmt.Sprintf("%s:%s", sessionName, windowIndex)
	command := fmt.Sprintf("cat >> %s", shellQuote(logFile))
	cmd := exec.Command(tmuxPath, "pipe-pane", "-o", "-t", target, command)
	return cmd.Run()
}

func attachToWindow(tmuxPath, sessionName, targetWindow string, attachOutside bool) error {
	target := fmt.Sprintf("%s:%s", sessionName, targetWindow)
	if os.Getenv("TMUX") != "" {
		cmd := exec.Command(tmuxPath, "select-window", "-t", target)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	if !attachOutside {
		return nil
	}
	if strings.TrimSpace(targetWindow) != "" {
		prep := exec.Command(tmuxPath, "select-window", "-t", target)
		prep.Stdout = os.Stdout
		prep.Stderr = os.Stderr
		prep.Stdin = os.Stdin
		_ = prep.Run()
	}
	cmd := exec.Command(tmuxPath, "attach", "-t", sessionName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func (r *runner) runDirect(resolved *execTarget, targetArgs []string, record *execSessionRecord, sessionPath string, opts runOptions) (int, error) {
	cmd := exec.Command(resolved.Binary, targetArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if record.WorkingDir != "" {
		cmd.Dir = record.WorkingDir
	}

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
	for key, value := range opts.ExtraEnv {
		if key == "" {
			continue
		}
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	cmd.Env = env

	startErr := cmd.Start()
	now := r.deps.Now()
	if startErr != nil {
		record.Status = "error"
		record.Error = startErr.Error()
		record.ExitCode = -1
		record.CompletedAt = now.Format(time.RFC3339Nano)
		record.RunMode = runModeDirect
		if err := saveExecSessionRecord(sessionPath, record); err != nil {
			r.logExecWarning("session update failed: %v", err)
		}
		return -1, fmt.Errorf("failed to start %s: %w", resolved.DisplayName, startErr)
	}

	record.PID = cmd.Process.Pid
	record.StartedAt = now.Format(time.RFC3339Nano)
	record.Status = "running"
	record.RunMode = runModeDirect
	if err := saveExecSessionRecord(sessionPath, record); err != nil {
		r.logExecWarning("session update failed: %v", err)
	}

	waitErr := cmd.Wait()
	completedAt := r.deps.Now()
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
		r.logExecWarning("session update failed: %v", err)
	}

	if waitErr != nil {
		if _, ok := waitErr.(*exec.ExitError); ok {
			return exitCode, nil
		}
		return exitCode, fmt.Errorf("failed to execute %s: %w", resolved.DisplayName, waitErr)
	}

	return exitCode, nil
}
