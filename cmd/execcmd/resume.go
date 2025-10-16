package execcmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

type resumeOptions struct {
	Dir    string
	Target string
	Detach bool
}

func (r *runner) resumeSessions(cmd *cobra.Command, opts resumeOptions) error {
	workingDir := opts.Dir
	if strings.TrimSpace(workingDir) == "" {
		workingDir = "."
	}
	absDir, err := filepath.Abs(workingDir)
	if err != nil {
		return fmt.Errorf("failed to resolve directory %q: %w", workingDir, err)
	}

	sessionName, err := deriveSessionName(absDir)
	if err != nil {
		return err
	}

	tmuxPath, err := r.deps.LookPath("tmux")
	if err != nil {
		return fmt.Errorf("tmux required for resume: %w", err)
	}

	records, err := r.recordsForDir(absDir)
	if err != nil {
		return err
	}

	if len(records) == 0 {
		cmd.Println("No saved windows for this directory.")
		return nil
	}

	targetFilter := strings.TrimSpace(strings.ToLower(opts.Target))
	if targetFilter != "" {
		filtered := make([]*execSessionRecord, 0, len(records))
		for _, rec := range records {
			if strings.ToLower(rec.Target) == targetFilter {
				filtered = append(filtered, rec)
			}
		}
		records = filtered
	}

	candidates := latestRecordsByWindow(records)
	if len(candidates) == 0 {
		cmd.Println("No tmux-backed windows to restore for this directory.")
		return nil
	}

	cmd.Printf("Restoring %d window(s) for %s (session %s)\n", len(candidates), absDir, sessionName)

	sessionExists := tmuxSessionExists(tmuxPath, sessionName)
	var restored, skipped, failed int
	var errs []error

	for _, rec := range candidates {
		windowLabel := rec.TmuxWindow
		if windowLabel == "" {
			windowLabel = rec.CommandLine
			if windowLabel == "" {
				windowLabel = rec.Target
			}
		}

		alreadyRunning := false
		if sessionExists {
			if ok, err := tmuxWindowExists(tmuxPath, sessionName, rec.TmuxWindow); err == nil && ok {
				cmd.Printf("[-] %s already running\n", windowLabel)
				skipped++
				alreadyRunning = true
			}
		}
		if alreadyRunning {
			continue
		}

		cmd.Printf("[.] starting %s\n", windowLabel)
		_, runErr := r.executeRun(rec.Target, rec.Args, runOptions{
			UseTmux:             true,
			RequireTmux:         true,
			AllowDirectFallback: false,
			WindowName:          rec.TmuxWindow,
			AttachBehavior:      attachNever,
			WorkingDir:          absDir,
			SessionName:         sessionName,
		})
		if runErr != nil {
			cmd.Printf("[!] %s failed: %v\n", windowLabel, runErr)
			errs = append(errs, runErr)
			failed++
		} else {
			cmd.Printf("[✓] %s\n", windowLabel)
			restored++
			sessionExists = true
		}
	}

	cmd.Printf("\nSummary: restored %d, skipped %d, failed %d\n", restored, skipped, failed)
	if opts.Detach {
		cmd.Println("(detach flag set; not attaching automatically)")
	}
	cmd.Printf("Attach with: ccmodel exec attach %s\n", sessionName)

	if len(errs) > 0 {
		return fmt.Errorf("%d window(s) failed to restore", failed)
	}
	return nil
}

func (r *runner) loadSessionRecords() ([]*execSessionRecord, error) {
	configDir := strings.TrimSpace(r.deps.ConfigDir())
	if configDir == "" {
		return nil, errors.New("config directory not configured")
	}
	sessionDir := filepath.Join(configDir, execSessionDirName)
	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var records []*execSessionRecord
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(sessionDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var record execSessionRecord
		if err := json.Unmarshal(data, &record); err != nil {
			continue
		}
		records = append(records, &record)
	}
	return records, nil
}

func tmuxWindowExists(tmuxPath, sessionName, windowName string) (bool, error) {
	target := strings.TrimSpace(windowName)
	if target == "" {
		return true, nil
	}
	cmd := exec.Command(tmuxPath, "list-windows", "-t", sessionName, "-F", "#{window_index}:#{window_name}")
	output, err := cmd.Output()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		}
		return false, err
	}
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		index := parts[0]
		name := ""
		if len(parts) > 1 {
			name = parts[1]
		}
		candidates := []string{
			index,
			name,
			fmt.Sprintf("%s:%s", sessionName, index),
			fmt.Sprintf("%s:%s", sessionName, name),
			line,
		}
		for _, cand := range candidates {
			if cand != "" && target == cand {
				return true, nil
			}
		}
	}
	return false, nil
}

func attachToSession(tmuxPath, sessionName string) error {
	if os.Getenv("TMUX") != "" {
		cmd := exec.Command(tmuxPath, "switch-client", "-t", sessionName)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		return cmd.Run()
	}
	cmd := exec.Command(tmuxPath, "attach", "-t", sessionName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
