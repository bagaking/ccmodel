package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollectClaudeCodeSessions(t *testing.T) {
	tempDir := t.TempDir()
	projectsDir := filepath.Join(tempDir, "projects", "demo-project")
	if err := os.MkdirAll(projectsDir, 0o755); err != nil {
		t.Fatalf("failed to create projects dir: %v", err)
	}

	sessionPath := filepath.Join(projectsDir, "session-1.jsonl")
	contents := "" +
		"{\"timestamp\":\"2025-10-03T00:00:00Z\",\"sessionId\":\"session-1\",\"cwd\":\"/tmp/project\",\"version\":\"1.0.0\",\"gitBranch\":\"main\",\"message\":{\"role\":\"user\",\"content\":\"hello world\"}}\n" +
		"{\"timestamp\":\"2025-10-03T00:01:00Z\",\"sessionId\":\"session-1\"}\n"
	if err := os.WriteFile(sessionPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("failed to write session file: %v", err)
	}

	sessions, err := collectClaudeCodeSessions(tempDir, 3)
	if err != nil {
		t.Fatalf("collectClaudeCodeSessions returned error: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}

	session := sessions[0]
	if session.ID != "session-1" {
		t.Errorf("expected session ID session-1, got %s", session.ID)
	}
	if session.Entry != "hello world" {
		t.Errorf("expected entry text 'hello world', got %q", session.Entry)
	}
	if session.DurationSeconds != 60 {
		t.Errorf("expected duration 60, got %d", session.DurationSeconds)
	}
}

func TestCollectCodexSessions(t *testing.T) {
	tempDir := t.TempDir()
	sessionDir := filepath.Join(tempDir, "sessions", "2025", "10", "03")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("failed to create session dir: %v", err)
	}

	payload := "" +
		"{\"timestamp\":\"2025-10-03T01:00:00Z\",\"type\":\"session_meta\",\"payload\":{\"id\":\"0199-session\",\"timestamp\":\"2025-10-03T01:00:00Z\",\"cwd\":\"/tmp/codex\",\"originator\":\"codex_cli_rs\",\"cli_version\":\"0.1.0\",\"git\":{\"branch\":\"main\"}}}\n" +
		"{\"timestamp\":\"2025-10-03T01:00:05Z\",\"type\":\"response_item\",\"payload\":{\"type\":\"message\",\"role\":\"user\",\"content\":[{\"type\":\"input_text\",\"text\":\"start coding\"}]}}\n" +
		"{\"timestamp\":\"2025-10-03T01:05:00Z\",\"type\":\"turn_context\",\"payload\":{\"model\":\"gpt-5-codex\"}}\n"
	sessionPath := filepath.Join(sessionDir, "rollout-2025-10-03T01-00-00-0199-session.jsonl")
	if err := os.WriteFile(sessionPath, []byte(payload), 0o644); err != nil {
		t.Fatalf("failed to write codex session: %v", err)
	}

	sessions, err := collectCodexSessions(tempDir, 1)
	if err != nil {
		t.Fatalf("collectCodexSessions returned error: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}

	session := sessions[0]
	if session.ID != "0199-session" {
		t.Errorf("expected session ID 0199-session, got %s", session.ID)
	}
	if session.Entry != "start coding" {
		t.Errorf("expected entry 'start coding', got %q", session.Entry)
	}
	if session.Model != "gpt-5-codex" {
		t.Errorf("expected model gpt-5-codex, got %s", session.Model)
	}
	if session.DurationSeconds != 300 {
		t.Errorf("expected duration 300, got %d", session.DurationSeconds)
	}
}
