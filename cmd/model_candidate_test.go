package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveModelCandidatePath(t *testing.T) {
	tempDir := t.TempDir()
	parentDir := filepath.Dir(tempDir)

	alphaPath := filepath.Join(tempDir, "settings.alpha.json")
	if err := os.WriteFile(alphaPath, []byte(`{"model":"alpha"}`), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v, want nil", alphaPath, err)
	}
	if err := os.Mkdir(filepath.Join(tempDir, "settings.x"), 0o755); err != nil {
		t.Fatalf("os.Mkdir(settings.x) error = %v, want nil", err)
	}
	escapedPath := filepath.Join(parentDir, "settings.evil.json")
	if err := os.WriteFile(escapedPath, []byte(`{"model":"escaped"}`), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v, want nil", escapedPath, err)
	}

	gotPath, err := resolveModelCandidatePath(tempDir, "alpha")
	if err != nil {
		t.Fatalf("resolveModelCandidatePath(%q, %q) error = %v, want nil", tempDir, "alpha", err)
	}
	if gotPath != alphaPath {
		t.Errorf("resolveModelCandidatePath(%q, %q) = %q, want %q", tempDir, "alpha", gotPath, alphaPath)
	}

	model := "x/../../settings.evil"
	gotPath, err = resolveModelCandidatePath(tempDir, model)
	if err == nil {
		t.Fatalf("resolveModelCandidatePath(%q, %q) error = nil, got path %q, want error", tempDir, model, gotPath)
	}
	if _, ok := err.(*modelCandidateNotFoundError); !ok {
		t.Errorf("resolveModelCandidatePath(%q, %q) error type = %T, want *modelCandidateNotFoundError", tempDir, model, err)
	}
}
