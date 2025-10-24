package execcmd

import (
	"os"
	"path/filepath"
)

const (
	userOnlyDirMode  os.FileMode = 0o700
	userOnlyFileMode os.FileMode = 0o600
)

func ensureUserOnlyDir(path string) error {
	if err := os.MkdirAll(path, userOnlyDirMode); err != nil {
		return err
	}
	return os.Chmod(path, userOnlyDirMode)
}

func writeUserOnlyFile(path string, data []byte) error {
	file, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tempPath := file.Name()
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(tempPath)
		}
	}()

	if err := file.Chmod(userOnlyFileMode); err != nil {
		_ = file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}

	if err := os.Rename(tempPath, path); err != nil {
		return err
	}
	committed = true
	return os.Chmod(path, userOnlyFileMode)
}

func createUserOnlyFile(path string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, userOnlyFileMode)
	if err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Chmod(path, userOnlyFileMode)
}
