package execcmd

import (
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type sessionSummary struct {
	Name       string
	Dir        string
	Running    bool
	HasHistory bool
	Windows    []sessionWindowInfo
}

type sessionWindowInfo struct {
	Index string
	Name  string
	Path  string
	Live  bool
}

func (r *runner) collectSessionSummaries() ([]sessionSummary, error) {
	tmuxPath, _ := r.deps.LookPath("tmux")
	active := map[string][]sessionWindowInfo{}
	if tmuxPath != "" {
		sessions, err := listActiveSessions(tmuxPath)
		if err == nil {
			for name, windows := range sessions {
				active[name] = windows
			}
		}
	}

	records, err := r.loadSessionRecords()
	if err != nil {
		return nil, err
	}

	summaries := map[string]*sessionSummary{}

	for name, windows := range active {
		entry := &sessionSummary{Name: name, Running: true}
		entry.Windows = append(entry.Windows, windows...)
		for _, w := range windows {
			if entry.Dir == "" && w.Path != "" {
				if abs, err := filepath.Abs(w.Path); err == nil {
					entry.Dir = abs
				} else {
					entry.Dir = w.Path
				}
				break
			}
		}
		summaries[name] = entry
	}

	for _, rec := range records {
		if rec.TmuxSession == "" || rec.RunMode != runModeTmux {
			continue
		}
		entry, ok := summaries[rec.TmuxSession]
		if !ok {
			entry = &sessionSummary{Name: rec.TmuxSession}
			summaries[rec.TmuxSession] = entry
		}
		entry.HasHistory = true
		if entry.Dir == "" && rec.WorkingDir != "" {
			if abs, err := filepath.Abs(rec.WorkingDir); err == nil {
				entry.Dir = abs
			} else {
				entry.Dir = rec.WorkingDir
			}
		}
		if rec.TmuxWindow == "" {
			continue
		}
		if !windowPresent(entry.Windows, rec.TmuxWindow) {
			entry.Windows = append(entry.Windows, sessionWindowInfo{
				Index: rec.TmuxWindow,
				Name:  rec.TmuxWindow,
				Path:  rec.WorkingDir,
				Live:  entry.Running,
			})
		}
	}

	result := make([]sessionSummary, 0, len(summaries))
	for _, entry := range summaries {
		if entry.Dir == "" {
			entry.Dir = "(unknown)"
		}
		// sort windows by index/name
		sort.Slice(entry.Windows, func(i, j int) bool {
			wi, wj := entry.Windows[i], entry.Windows[j]
			if wi.Live != wj.Live {
				return wi.Live
			}
			if wi.Index != "" && wj.Index != "" {
				return wi.Index < wj.Index
			}
			return strings.Compare(wi.Name, wj.Name) < 0
		})
		result = append(result, *entry)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Running != result[j].Running {
			return result[i].Running
		}
		if result[i].HasHistory != result[j].HasHistory {
			return result[i].HasHistory
		}
		if result[i].Dir != result[j].Dir {
			return strings.Compare(result[i].Dir, result[j].Dir) < 0
		}
		return strings.Compare(result[i].Name, result[j].Name) < 0
	})

	return result, nil
}

func windowPresent(list []sessionWindowInfo, name string) bool {
	for _, w := range list {
		if w.Name == name || w.Index == name {
			return true
		}
	}
	return false
}

func listActiveSessions(tmuxPath string) (map[string][]sessionWindowInfo, error) {
	cmd := exec.Command(tmuxPath, "list-sessions", "-F", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	result := make(map[string][]sessionWindowInfo, len(lines))
	for _, name := range lines {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		rawWindows, err := listSessionWindows(tmuxPath, name)
		if err != nil {
			continue
		}
		windows := make([]sessionWindowInfo, 0, len(rawWindows))
		for _, w := range rawWindows {
			windows = append(windows, sessionWindowInfo{
				Index: w.Index,
				Name:  w.Name,
				Path:  w.Path,
				Live:  true,
			})
		}
		result[name] = windows
	}
	return result, nil
}
