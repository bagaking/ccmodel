package execcmd

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

func syncTmuxEnvironment(tmuxPath, sessionName string, envVars map[string]string) error {
	if len(envVars) == 0 {
		return nil
	}

	targetSession := strings.TrimSpace(sessionName)
	if targetSession == "" {
		targetSession = defaultTmuxSession
	}

	keys := make([]string, 0, len(envVars))
	for key := range envVars {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		keys = append(keys, trimmed)
	}
	sort.Strings(keys)

	if len(keys) == 0 {
		return nil
	}

	var failures []string
	for _, key := range keys {
		value := envVars[key]
		cmd := exec.Command(tmuxPath, "set-environment", "-t", targetSession, key, value)
		if err := cmd.Run(); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", key, err))
		}
	}

	if len(failures) > 0 {
		return fmt.Errorf("tmux set-environment errors: %s", strings.Join(failures, "; "))
	}

	return nil
}
