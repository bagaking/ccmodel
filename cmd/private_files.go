package cmd

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

func openUserOnlyAppendFile(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, userOnlyFileMode)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, userOnlyFileMode); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

func createUserOnlyExclusiveFile(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, userOnlyFileMode)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, userOnlyFileMode); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return nil, err
	}
	return file, nil
}
