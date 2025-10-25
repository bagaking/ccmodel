package cmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestGetQuotaInfoForModel_PathLikeModelDoesNotReadOutsideConfigDir(t *testing.T) {
	claudeDir := setupQuotaTempHome(t)
	parentDir := filepath.Dir(claudeDir)

	if err := os.Mkdir(filepath.Join(claudeDir, "settings.x"), 0o755); err != nil {
		t.Fatalf("os.Mkdir(settings.x) error = %v, want nil", err)
	}
	serverCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverCalled = true
		_, _ = w.Write([]byte(`{"total":100,"used":1}`))
	}))
	t.Cleanup(server.Close)

	escapedConfig := []byte(`{
  "__cc": {
    "quota_test": {
      "get": {
        "url": "` + server.URL + `",
        "result": {
          "total": "total",
          "used": "used"
        }
      }
    }
  }
}`)
	escapedPath := filepath.Join(parentDir, "settings.evil.json")
	if err := os.WriteFile(escapedPath, escapedConfig, 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v, want nil", escapedPath, err)
	}

	model := "x/../../settings.evil"
	got, err := getQuotaInfoForModel(model, time.Second)
	if err == nil {
		t.Fatalf("getQuotaInfoForModel(%q) error = nil, got quota %+v, want error", model, got)
	}
	if serverCalled {
		t.Errorf("getQuotaInfoForModel(%q) called escaped quota endpoint, want no request", model)
	}
}

func TestGetQuotaInfoForModel_NormalCandidate(t *testing.T) {
	claudeDir := setupQuotaTempHome(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"total":100,"used":25}`))
	}))
	t.Cleanup(server.Close)

	config := []byte(`{
  "__cc": {
    "quota_test": {
      "get": {
        "url": "` + server.URL + `",
        "result": {
          "total": "total",
          "used": "used"
        }
      }
    }
  }
}`)
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.alpha.json"), config, 0o644); err != nil {
		t.Fatalf("os.WriteFile(settings.alpha.json) error = %v, want nil", err)
	}

	got, err := getQuotaInfoForModel("alpha", time.Second)
	if err != nil {
		t.Fatalf("getQuotaInfoForModel(%q) error = %v, want nil", "alpha", err)
	}
	if got == nil {
		t.Fatalf("getQuotaInfoForModel(%q) = nil, want quota info", "alpha")
	}
	if got.Total != 100 || got.Used != 25 {
		t.Errorf("getQuotaInfoForModel(%q) = total %v used %v, want total 100 used 25", "alpha", got.Total, got.Used)
	}
}

func setupQuotaTempHome(t *testing.T) string {
	t.Helper()

	tempHome := t.TempDir()
	claudeDir := filepath.Join(tempHome, ".claude")
	if err := os.Mkdir(claudeDir, 0o700); err != nil {
		t.Fatalf("os.Mkdir(%q) error = %v, want nil", claudeDir, err)
	}

	previousConfigDir := configDir
	previousQuotaHistoryManager := quotaHistoryManager
	previousQuotaCache := quotaCache
	t.Cleanup(func() {
		configDir = previousConfigDir
		quotaHistoryManager = previousQuotaHistoryManager
		quotaCache = previousQuotaCache
		resetQuotaHistoryOnceForTest()
	})

	t.Setenv("HOME", tempHome)
	configDir = claudeDir
	quotaHistoryManager = nil
	quotaCache = make(map[string]*QuotaCache)
	resetQuotaHistoryOnceForTest()

	return claudeDir
}

func resetQuotaHistoryOnceForTest() {
	quotaHistoryOnce = sync.Once{}
}
