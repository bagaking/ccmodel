package execcmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func (r *runner) watchLogs(cmd *cobra.Command, mode string) error {
	configDir := strings.TrimSpace(r.deps.ConfigDir())
	if configDir == "" {
		return fmt.Errorf("config directory is not configured")
	}
	logDir := filepath.Join(configDir, execSessionDirName, "logs")
	files, err := filepath.Glob(filepath.Join(logDir, "*.log"))
	if err != nil {
		return err
	}
	if len(files) == 0 {
		cmd.Printf("No logs yet in %s\n", logDir)
		cmd.Println("Launch sessions with `ccmodel exec run <target>` first.")
		return nil
	}

	mode = strings.ToLower(mode)
	if mode == "" {
		mode = "auto"
	}

	switch mode {
	case "tail":
		return invokeStreaming(cmd, "tail", append([]string{"-n", "200", "-F"}, files...)...)
	case "multitail":
		if _, err := r.deps.LookPath("multitail"); err != nil {
			cmd.Println("multitail not found; falling back to tail.")
			return invokeStreaming(cmd, "tail", append([]string{"-n", "200", "-F"}, files...)...)
		}
		return invokeStreaming(cmd, "multitail", append([]string{"-n", "200"}, files...)...)
	default:
		if _, err := r.deps.LookPath("multitail"); err == nil {
			return invokeStreaming(cmd, "multitail", append([]string{"-n", "200"}, files...)...)
		}
		return invokeStreaming(cmd, "tail", append([]string{"-n", "200", "-F"}, files...)...)
	}
}

func invokeStreaming(cmd *cobra.Command, name string, args ...string) error {
	proc := exec.Command(name, args...)
	proc.Stdout = os.Stdout
	proc.Stderr = os.Stderr
	proc.Stdin = os.Stdin
	return proc.Run()
}

func (r *runner) printStatus(cmd *cobra.Command, scope string) error {
	scope = strings.TrimSpace(strings.ToLower(scope))
	switch scope {
	case "", "sessions":
		return r.printSessionsAndWindows(cmd)
	case "logs":
		return r.printLogStatus(cmd)
	default:
		return fmt.Errorf("unknown status scope %q", scope)
	}
}

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorGray   = "\033[90m"
)

func (r *runner) printSessionsAndWindows(cmd *cobra.Command) error {
	summaries, err := r.collectSessionSummaries()
	if err != nil {
		return err
	}
	if len(summaries) == 0 {
		cmd.Println("No tmux sessions recorded yet. Launch with `ccmodel exec run`.")
		return nil
	}

	cmd.Println("Known sessions:")
	for _, summary := range summaries {
		icon := "○"
		color := colorGray
		status := "archived"
		if summary.Running {
			icon = "●"
			color = colorGreen
			status = "running"
		} else if !summary.HasHistory {
			status = "stopped"
		}
		header := fmt.Sprintf("%s %s (%s)", icon, summary.Name, summary.Dir)
		header = applyColor(header, color)
		cmd.Printf("%s %s\n", header, applyColor("["+status+"]", color))

		if len(summary.Windows) == 0 {
			cmd.Println("  └─ (no recorded windows)")
			continue
		}

		for i, window := range summary.Windows {
			branch := "├─"
			if i == len(summary.Windows)-1 {
				branch = "└─"
			}
			label := window.Name
			if label == "" {
				label = window.Index
			}
			if label == "" {
				label = "(unnamed)"
			}
			statusTag := "saved"
			statusColor := colorGray
			if window.Live && summary.Running {
				statusTag = "live"
				statusColor = colorGreen
			}
			info := fmt.Sprintf("%s %s", branch, label)
			if window.Path != "" {
				info += fmt.Sprintf(" (%s)", window.Path)
			}
			info += " " + applyColor("["+statusTag+"]", statusColor)
			cmd.Println("  " + info)
		}
	}

	cmd.Println()
	cmd.Println("Use `ccmodel exec resume --dir <path>` to restore archived sessions.")
	return nil
}

func (r *runner) printLogStatus(cmd *cobra.Command) error {
	configDir := strings.TrimSpace(r.deps.ConfigDir())
	if configDir == "" {
		return fmt.Errorf("config directory is not configured")
	}
	logDir := filepath.Join(configDir, execSessionDirName, "logs")

	entries, err := os.ReadDir(logDir)
	if err != nil {
		if os.IsNotExist(err) {
			cmd.Printf("Log directory does not exist yet: %s\n", logDir)
			return nil
		}
		return err
	}

	type logInfo struct {
		Path string
		Mod  time.Time
		Size int64
	}

	var logs []logInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		logs = append(logs, logInfo{
			Path: filepath.Join(logDir, entry.Name()),
			Mod:  info.ModTime(),
			Size: info.Size(),
		})
	}

	if len(logs) == 0 {
		cmd.Printf("No log files in %s\n", logDir)
		return nil
	}

	sort.Slice(logs, func(i, j int) bool {
		return logs[i].Mod.After(logs[j].Mod)
	})

	cmd.Printf("Log files in %s:\n", logDir)
	for _, log := range logs {
		cmd.Printf("- %s (%s, %d bytes)\n", filepath.Base(log.Path), log.Mod.Format(time.RFC3339), log.Size)
	}
	return nil
}

func applyColor(text, color string) string {
	if !shouldUseColor() {
		return text
	}
	return color + text + colorReset
}

func shouldUseColor() bool {
	return os.Getenv("NO_COLOR") == ""
}
