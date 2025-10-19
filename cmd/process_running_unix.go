//go:build !windows

package cmd

import "syscall"

func isProcessRunning(pid int) (bool, error) {
	if pid <= 0 {
		return false, nil
	}

	// kill(pid, 0) checks process existence without sending a signal.
	err := syscall.Kill(pid, 0)
	if err == nil {
		return true, nil
	}

	if errno, ok := err.(syscall.Errno); ok {
		switch errno {
		case syscall.ESRCH:
			return false, nil
		case syscall.EPERM:
			return true, nil
		}
	}

	return true, err
}
