package cmd

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type SessionOverview struct {
	ClaudeCode []SessionInfo `json:"claude_code,omitempty"`
	Codex      []SessionInfo `json:"codex,omitempty"`
}

type SessionInfo struct {
	ID              string `json:"id"`
	Workspace       string `json:"workspace,omitempty"`
	Entry           string `json:"entry,omitempty"`
	StartTime       string `json:"start_time,omitempty"`
	LastActive      string `json:"last_active,omitempty"`
	DurationSeconds int64  `json:"duration_seconds,omitempty"`
	Version         string `json:"version,omitempty"`
	Branch          string `json:"branch,omitempty"`
	Origin          string `json:"origin,omitempty"`
	Model           string `json:"model,omitempty"`
}

const sessionSampleLimit = 3

func (msc *ModelStatusCollector) collectSessionInfo(status *ModelStatus) error {
	overview := &SessionOverview{}

	claudeSessions, claudeErr := collectClaudeCodeSessions(msc.configDir, sessionSampleLimit)
	if len(claudeSessions) > 0 {
		overview.ClaudeCode = claudeSessions
	}

	homeDir, homeErr := os.UserHomeDir()
	if homeErr != nil {
		homeDir = filepath.Dir(msc.configDir)
	}

	codexRoot := filepath.Join(homeDir, ".codex")
	codexSessions, codexErr := collectCodexSessions(codexRoot, sessionSampleLimit)
	if len(codexSessions) > 0 {
		overview.Codex = codexSessions
	}

	if len(overview.ClaudeCode) == 0 && len(overview.Codex) == 0 {
		var err error
		if claudeErr != nil {
			err = errors.Join(err, claudeErr)
		}
		if codexErr != nil {
			err = errors.Join(err, codexErr)
		}
		if err != nil {
			return err
		}
		return nil
	}

	status.Sessions = overview

	if claudeErr != nil || codexErr != nil {
		var err error
		if claudeErr != nil {
			err = errors.Join(err, claudeErr)
		}
		if codexErr != nil {
			err = errors.Join(err, codexErr)
		}
		return err
	}

	return nil
}

type sessionFile struct {
	path    string
	modTime time.Time
}

func collectClaudeCodeSessions(claudeDir string, limit int) ([]SessionInfo, error) {
	projectsDir := filepath.Join(claudeDir, "projects")
	if _, err := os.Stat(projectsDir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var files []sessionFile
	walkErr := filepath.WalkDir(projectsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, os.ErrPermission) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".jsonl" {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		files = append(files, sessionFile{path: path, modTime: info.ModTime()})
		return nil
	})

	if walkErr != nil {
		return nil, walkErr
	}

	if len(files) == 0 {
		return nil, nil
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})
	if limit > 0 && len(files) > limit {
		files = files[:limit]
	}

	sessions := make([]SessionInfo, 0, len(files))
	var errs error
	for _, file := range files {
		info, err := parseClaudeSessionFile(file.path)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("claude session %s: %w", file.path, err))
			continue
		}
		sessions = append(sessions, *info)
	}

	if len(sessions) == 0 {
		return nil, errs
	}

	return sessions, errs
}

func parseClaudeSessionFile(path string) (*SessionInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	var (
		sessionID string
		workspace string
		version   string
		branch    string
		entryText string
		startTime string
		lastTime  string
	)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var entry claudeProjectEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		if entry.SessionID != "" {
			sessionID = entry.SessionID
		}
		if workspace == "" && entry.CWD != "" {
			workspace = entry.CWD
		}
		if version == "" && entry.Version != "" {
			version = entry.Version
		}
		if branch == "" && entry.GitBranch != "" {
			branch = entry.GitBranch
		}
		if entry.Timestamp != "" {
			lastTime = entry.Timestamp
			if startTime == "" {
				startTime = entry.Timestamp
			}
		}
		if entryText == "" && entry.Message != nil && entry.Message.Role == "user" {
			entryText = extractClaudeEntry(entry.Message)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if sessionID == "" {
		sessionID = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	if startTime == "" {
		startTime = lastTime
	}
	if lastTime == "" {
		lastTime = startTime
	}

	info := &SessionInfo{
		ID:        sessionID,
		Workspace: workspace,
		Entry:     entryText,
		StartTime: startTime,
		LastActive: func() string {
			if lastTime == "" {
				return startTime
			}
			return lastTime
		}(),
		Version: version,
		Branch:  branch,
		Origin:  "claude_code",
	}

	if d := computeDuration(info.StartTime, info.LastActive); d > 0 {
		info.DurationSeconds = int64(d.Seconds())
	}

	return info, nil
}

type claudeProjectEntry struct {
	SessionID string         `json:"sessionId"`
	Timestamp string         `json:"timestamp"`
	CWD       string         `json:"cwd"`
	Version   string         `json:"version"`
	GitBranch string         `json:"gitBranch"`
	Message   *claudeMessage `json:"message"`
}

type claudeMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type claudeContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func extractClaudeEntry(msg *claudeMessage) string {
	if msg == nil || len(msg.Content) == 0 {
		return ""
	}

	var text string
	if err := json.Unmarshal(msg.Content, &text); err == nil {
		return strings.TrimSpace(text)
	}

	var items []claudeContentItem
	if err := json.Unmarshal(msg.Content, &items); err == nil {
		for _, item := range items {
			if item.Text != "" {
				return strings.TrimSpace(item.Text)
			}
		}
	}

	return ""
}

func collectCodexSessions(codexDir string, limit int) ([]SessionInfo, error) {
	sessionsRoot := filepath.Join(codexDir, "sessions")
	if _, err := os.Stat(sessionsRoot); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var files []sessionFile
	walkErr := filepath.WalkDir(sessionsRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, os.ErrPermission) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".jsonl" {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		files = append(files, sessionFile{path: path, modTime: info.ModTime()})
		return nil
	})

	if walkErr != nil {
		return nil, walkErr
	}

	if len(files) == 0 {
		return nil, nil
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})
	if limit > 0 && len(files) > limit {
		files = files[:limit]
	}

	sessions := make([]SessionInfo, 0, len(files))
	var errs error
	for _, file := range files {
		info, err := parseCodexSessionFile(file.path)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("codex session %s: %w", file.path, err))
			continue
		}
		sessions = append(sessions, *info)
	}

	if len(sessions) == 0 {
		return nil, errs
	}

	return sessions, errs
}

type codexLogEntry struct {
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

type codexSessionMeta struct {
	ID         string        `json:"id"`
	Timestamp  string        `json:"timestamp"`
	CWD        string        `json:"cwd"`
	Originator string        `json:"originator"`
	CLIVersion string        `json:"cli_version"`
	Git        *codexGitMeta `json:"git"`
}

type codexGitMeta struct {
	Branch     string `json:"branch"`
	CommitHash string `json:"commit_hash"`
}

type codexResponseItem struct {
	Type    string              `json:"type"`
	Role    string              `json:"role"`
	Content []codexContentBlock `json:"content"`
}

type codexContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type codexTurnContext struct {
	Model string `json:"model"`
}

func parseCodexSessionFile(path string) (*SessionInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	var (
		sessionID  string
		startTime  string
		lastActive string
		workspace  string
		originator string
		version    string
		branch     string
		entryText  string
		model      string
	)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var entry codexLogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		if entry.Timestamp != "" {
			lastActive = entry.Timestamp
			if startTime == "" {
				startTime = entry.Timestamp
			}
		}

		switch entry.Type {
		case "session_meta":
			var meta codexSessionMeta
			if err := json.Unmarshal(entry.Payload, &meta); err != nil {
				continue
			}
			if meta.ID != "" {
				sessionID = meta.ID
			}
			if meta.Timestamp != "" {
				startTime = meta.Timestamp
			}
			if meta.CWD != "" {
				workspace = meta.CWD
			}
			if meta.Originator != "" {
				originator = meta.Originator
			}
			if meta.CLIVersion != "" {
				version = meta.CLIVersion
			}
			if meta.Git != nil {
				if meta.Git.Branch != "" {
					branch = meta.Git.Branch
				}
			}
		case "response_item":
			if entry.Payload == nil {
				continue
			}
			var response codexResponseItem
			if err := json.Unmarshal(entry.Payload, &response); err != nil {
				continue
			}
			if entryText == "" && response.Role == "user" {
				for _, block := range response.Content {
					if strings.TrimSpace(block.Text) != "" {
						entryText = strings.TrimSpace(block.Text)
						break
					}
				}
			}
		case "turn_context":
			if model != "" {
				continue
			}
			var ctx codexTurnContext
			if err := json.Unmarshal(entry.Payload, &ctx); err != nil {
				continue
			}
			if ctx.Model != "" {
				model = ctx.Model
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if sessionID == "" {
		base := filepath.Base(path)
		trimmed := strings.TrimSuffix(base, filepath.Ext(base))
		parts := strings.Split(trimmed, "-")
		if len(parts) >= 6 {
			sessionID = strings.Join(parts[len(parts)-5:], "-")
		}
	}

	info := &SessionInfo{
		ID:        sessionID,
		Workspace: workspace,
		Entry:     entryText,
		StartTime: startTime,
		LastActive: func() string {
			if lastActive == "" {
				return startTime
			}
			return lastActive
		}(),
		Origin:  originator,
		Version: version,
		Branch:  branch,
		Model:   model,
	}

	if d := computeDuration(info.StartTime, info.LastActive); d > 0 {
		info.DurationSeconds = int64(d.Seconds())
	}

	return info, nil
}

func computeDuration(start, end string) time.Duration {
	if start == "" || end == "" {
		return 0
	}

	startTime, err := time.Parse(time.RFC3339Nano, start)
	if err != nil {
		return 0
	}
	endTime, err := time.Parse(time.RFC3339Nano, end)
	if err != nil {
		return 0
	}

	if endTime.Before(startTime) {
		return 0
	}

	return endTime.Sub(startTime)
}
