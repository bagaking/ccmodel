package execcmd

import (
	"bufio"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func (r *runner) maybeRestoreSession(cmd *cobra.Command, sessionName, workingDir string) error {
	tmuxPath, err := r.deps.LookPath("tmux")
	if err != nil {
		return nil
	}
	if tmuxSessionExists(tmuxPath, sessionName) {
		return nil
	}

	records, err := r.recordsForDir(workingDir)
	if err != nil {
		return err
	}
	candidates := latestRecordsByWindow(records)
	if len(candidates) == 0 {
		return nil
	}

	absoluteDir, _ := filepath.Abs(workingDir)
	cmd.Printf("Found %d previous window(s) for %s:\n", len(candidates), absoluteDir)
	for i, rec := range candidates {
		description := rec.CommandLine
		if description == "" {
			description = rec.Target
		}
		cmd.Printf("  [%d] %s  %s\n", i+1, rec.TmuxWindow, description)
	}

	if !isInteractive() {
		cmd.Printf("(non-interactive) Skip auto-restore. Use `ccmodel exec resume --dir %s` to restore later.\n", absoluteDir)
		return nil
	}

	reader := bufio.NewReader(cmd.InOrStdin())
	input, err := promptSelect("Restore which windows? (comma-separated numbers, 'a' for all, Enter to skip): ", reader)
	if err != nil {
		return err
	}
	if input == "" {
		cmd.Println("Skipping restoration.")
		return nil
	}

	selected := pickRecords(input, candidates)
	if len(selected) == 0 {
		cmd.Println("No valid selections; continuing without restore.")
		return nil
	}

	cmd.Println("Restoring selected windows...")
	for _, rec := range selected {
		_, runErr := r.executeRun(rec.Target, rec.Args, runOptions{
			UseTmux:             true,
			RequireTmux:         true,
			AllowDirectFallback: false,
			WindowName:          rec.TmuxWindow,
			AttachBehavior:      attachNever,
			WorkingDir:          workingDir,
			SessionName:         sessionName,
		})
		if runErr != nil {
			cmd.Printf("  [!] %s failed: %v\n", rec.TmuxWindow, runErr)
		} else {
			cmd.Printf("  [✓] %s\n", rec.TmuxWindow)
		}
	}
	cmd.Println("Restoration complete.")
	return nil
}

func (r *runner) recordsForDir(dir string) ([]*execSessionRecord, error) {
	records, err := r.loadSessionRecords()
	if err != nil {
		return nil, err
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	results := make([]*execSessionRecord, 0)
	for _, rec := range records {
		if rec.RunMode != runModeTmux {
			continue
		}
		if rec.WorkingDir == "" || rec.TmuxWindow == "" {
			continue
		}
		recDir, err := filepath.Abs(rec.WorkingDir)
		if err != nil {
			continue
		}
		if recDir == absDir {
			results = append(results, rec)
		}
	}
	return results, nil
}

func latestRecordsByWindow(records []*execSessionRecord) []*execSessionRecord {
	byWindow := make(map[string]*execSessionRecord)
	for _, rec := range records {
		window := rec.TmuxWindow
		if window == "" {
			continue
		}
		if existing, ok := byWindow[window]; !ok || parseTimeSafe(existing.CreatedAt).Before(parseTimeSafe(rec.CreatedAt)) {
			byWindow[window] = rec
		}
	}
	result := make([]*execSessionRecord, 0, len(byWindow))
	for _, rec := range byWindow {
		result = append(result, rec)
	}
	sort.Slice(result, func(i, j int) bool {
		return parseTimeSafe(result[i].CreatedAt).After(parseTimeSafe(result[j].CreatedAt))
	})
	return result
}

func pickRecords(input string, candidates []*execSessionRecord) []*execSessionRecord {
	token := strings.TrimSpace(strings.ToLower(input))
	if token == "a" || token == "all" {
		return candidates
	}
	indices := strings.Split(input, ",")
	seen := make(map[int]struct{})
	selected := make([]*execSessionRecord, 0, len(indices))
	for _, idx := range indices {
		n := strings.TrimSpace(idx)
		if n == "" {
			continue
		}
		value, err := strconv.Atoi(n)
		if err != nil {
			continue
		}
		value--
		if value < 0 || value >= len(candidates) {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		selected = append(selected, candidates[value])
	}
	return selected
}
