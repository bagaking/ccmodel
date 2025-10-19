//go:build windows

package cmd

import "syscall"

const windowsErrorInvalidParameter syscall.Errno = 87

func isProcessRunning(pid int) (bool, error) {
	if pid <= 0 {
		return false, nil
	}

	handle, err := syscall.OpenProcess(syscall.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		if errno, ok := err.(syscall.Errno); ok {
			switch errno {
			case syscall.ERROR_FILE_NOT_FOUND, syscall.ERROR_NOT_FOUND, windowsErrorInvalidParameter:
				return false, nil
			case syscall.ERROR_ACCESS_DENIED:
				return true, nil
			}
		}
		return true, err
	}
	defer syscall.CloseHandle(handle)

	event, err := syscall.WaitForSingleObject(handle, 0)
	if err != nil {
		return true, err
	}

	switch event {
	case syscall.WAIT_TIMEOUT:
		return true, nil
	case syscall.WAIT_OBJECT_0:
		return false, nil
	default:
		return true, syscall.Errno(event)
	}
}
