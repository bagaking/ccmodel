package execcmd

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

func (r *runner) attachToManagedSession(cmd *cobra.Command, window string) error {
	tmuxPath, err := r.deps.LookPath("tmux")
	if err != nil {
		return fmt.Errorf("tmux not found on PATH: %w", err)
	}

	summaries, err := r.collectSessionSummaries()
	if err != nil {
		return err
	}
	if len(summaries) == 0 {
		return fmt.Errorf("no known sessions; start one with `ccmodel exec run`")
	}
	running := filterRunningSessions(summaries)
	if len(running) == 0 {
		return fmt.Errorf("no running tmux sessions. Restore one with `ccmodel exec resume --dir <path>`.")
	}

	arg := strings.TrimSpace(window)
	var sessionName, windowName string
	if arg != "" {
		sessionName, windowName = splitSessionWindow(arg)
	}

	if sessionName == "" && arg != "" {
		if guess := findSessionByWindow(arg, running); guess != "" {
			sessionName = guess
			windowName = arg
		}
	}

	if sessionName == "" {
		if !isInteractive() {
			sessionName = running[0].Name
		} else {
			cmd.Println("Running sessions:")
			for i, s := range running {
				cmd.Printf("  [%d] %s (%s) [running]\n", i+1, s.Name, s.Dir)
			}
			reader := bufio.NewReader(cmd.InOrStdin())
			input, err := promptSelect("Select session (number/name, default 1): ", reader)
			if err != nil {
				return err
			}
			selected := strings.TrimSpace(input)
			if selected == "" {
				sessionName = running[0].Name
			} else {
				sessionName = resolveSessionSelection(selected, running)
				if sessionName == "" {
					return fmt.Errorf("unknown session selection %q", selected)
				}
			}
		}
	}

	summary := findSummary(sessionName, summaries)
	if summary == nil {
		return fmt.Errorf("session %q not found", sessionName)
	}
	if !summary.Running {
		return fmt.Errorf("session %q is archived (dir %s). Restore it first with `ccmodel exec resume --dir %s`.", sessionName, summary.Dir, summary.Dir)
	}

	return r.attachWithinSession(cmd, tmuxPath, summary, windowName)
}

func tmuxSessionExists(tmuxPath, sessionName string) bool {
	cmd := exec.Command(tmuxPath, "has-session", "-t", sessionName)
	return cmd.Run() == nil
}

type tmuxWindowInfo struct {
	Index string
	Name  string
	Path  string
}

func listSessionWindows(tmuxPath, sessionName string) ([]tmuxWindowInfo, error) {
	cmd := exec.Command(tmuxPath, "list-windows", "-t", sessionName, "-F", "#{window_index}|#{window_name}|#{pane_current_path}")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	windows := make([]tmuxWindowInfo, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		info := tmuxWindowInfo{Index: parts[0]}
		if len(parts) > 1 {
			info.Name = parts[1]
		}
		if len(parts) > 2 {
			info.Path = parts[2]
		}
		windows = append(windows, info)
	}
	return windows, nil
}

func (r *runner) attachWithinSession(cmd *cobra.Command, tmuxPath string, summary *sessionSummary, hint string) error {
	if summary == nil {
		return fmt.Errorf("session information missing")
	}
	if !summary.Running {
		return fmt.Errorf("session %q is archived (dir %s). Restore it first with `ccmodel exec resume --dir %s`.", summary.Name, summary.Dir, summary.Dir)
	}
	if len(summary.Windows) == 0 {
		return fmt.Errorf("session %q has no windows", summary.Name)
	}

	choices := buildWindowChoices(summary.Windows)
	if len(choices) == 0 {
		return fmt.Errorf("session %q has no windows", summary.Name)
	}

	var target string
	if hint != "" {
		target = resolveWindowChoice(hint, choices)
		if target == "" {
			return fmt.Errorf("window %q not found in session %q", hint, summary.Name)
		}
	} else {
		if len(choices) == 1 || !isInteractive() {
			target = pickWindowName(choices[0].Window)
		} else {
			cmd.Println("Available windows:")
			for _, choice := range choices {
				w := choice.Window
				label := pickWindowName(w)
				cmd.Printf("  [%s] %s (index %s, dir %s)\n", choice.Selector, label, w.Index, w.Path)
			}
			reader := bufio.NewReader(cmd.InOrStdin())
			prompt := fmt.Sprintf("Select window (number/index/name, default %s): ", choices[0].Selector)
			input, err := promptSelect(prompt, reader)
			if err != nil {
				return err
			}
			selection := strings.TrimSpace(input)
			if selection == "" {
				target = pickWindowName(choices[0].Window)
			} else {
				target = resolveWindowChoice(selection, choices)
				if target == "" {
					return fmt.Errorf("unknown window selection %q", selection)
				}
			}
		}
	}

	return attachToWindow(tmuxPath, summary.Name, target, true)
}

func pickWindowName(win sessionWindowInfo) string {
	if win.Name != "" {
		return win.Name
	}
	return win.Index
}

func splitSessionWindow(input string) (string, string) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", ""
	}
	if strings.Contains(trimmed, ":") {
		parts := strings.SplitN(trimmed, ":", 2)
		session := strings.TrimSpace(parts[0])
		window := ""
		if len(parts) > 1 {
			window = strings.TrimSpace(parts[1])
		}
		return session, window
	}
	return trimmed, ""
}

func findSessionByWindow(target string, sessions []sessionSummary) string {
	lower := strings.ToLower(strings.TrimSpace(target))
	for _, s := range sessions {
		if !s.Running {
			continue
		}
		for _, w := range s.Windows {
			if strings.ToLower(w.Name) == lower || strings.ToLower(w.Index) == lower {
				return s.Name
			}
		}
	}
	return ""
}

func resolveSessionSelection(input string, sessions []sessionSummary) string {
	trimmed := strings.TrimSpace(strings.ToLower(input))
	for i, s := range sessions {
		if trimmed == strings.ToLower(s.Name) || trimmed == fmt.Sprintf("%d", i+1) {
			return s.Name
		}
	}
	return ""
}

func findSummary(name string, sessions []sessionSummary) *sessionSummary {
	for i := range sessions {
		if sessions[i].Name == name {
			return &sessions[i]
		}
	}
	return nil
}

type windowChoice struct {
	Selector string
	Window   sessionWindowInfo
}

func buildWindowChoices(windows []sessionWindowInfo) []windowChoice {
	nonHome := make([]sessionWindowInfo, 0, len(windows))
	home := make([]sessionWindowInfo, 0, 1)
	for _, w := range windows {
		if isHomeWindow(w) {
			home = append(home, w)
		} else {
			nonHome = append(nonHome, w)
		}
	}
	choices := make([]windowChoice, 0, len(windows))
	seq := 1
	for _, w := range nonHome {
		choices = append(choices, windowChoice{
			Selector: fmt.Sprintf("%d", seq),
			Window:   w,
		})
		seq++
	}
	for _, w := range home {
		choices = append(choices, windowChoice{
			Selector: "-1",
			Window:   w,
		})
	}
	return choices
}

func resolveWindowChoice(input string, choices []windowChoice) string {
	trimmed := strings.TrimSpace(strings.ToLower(input))
	if trimmed == "" {
		return pickWindowName(choices[0].Window)
	}
	for _, choice := range choices {
		if trimmed == strings.ToLower(choice.Selector) {
			return pickWindowName(choice.Window)
		}
	}
	for _, choice := range choices {
		w := choice.Window
		if strings.ToLower(w.Name) == trimmed || strings.ToLower(w.Index) == trimmed {
			return pickWindowName(w)
		}
	}
	return ""
}

func isHomeWindow(w sessionWindowInfo) bool {
	if strings.EqualFold(w.Name, "home") {
		return true
	}
	if w.Name == "" && w.Index == "0" {
		return true
	}
	return false
}

func filterRunningSessions(sessions []sessionSummary) []sessionSummary {
	running := make([]sessionSummary, 0)
	for _, s := range sessions {
		if s.Running {
			running = append(running, s)
		}
	}
	return running
}
