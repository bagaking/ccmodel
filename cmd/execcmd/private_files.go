package execcmd

import "os"

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
	if err := os.WriteFile(path, data, userOnlyFileMode); err != nil {
		return err
	}
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
