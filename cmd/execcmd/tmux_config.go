package execcmd

import "os/exec"

func configureTmuxSession(tmuxPath, sessionName string) {
	_ = exec.Command(tmuxPath, "set-option", "-t", sessionName, "mouse", "on").Run()
	_ = exec.Command(tmuxPath, "set-option", "-t", sessionName, "mode-keys", "vi").Run()
}
