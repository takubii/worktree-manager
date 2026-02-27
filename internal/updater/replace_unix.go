package updater

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func replaceBinaryUnix(sourcePath string, targetPath string) error {
	tmpPath := filepath.Join(filepath.Dir(targetPath), "."+filepath.Base(targetPath)+".tmp")
	if err := copyFile(sourcePath, tmpPath, 0o755); err != nil {
		return fmt.Errorf("failed to stage updated binary: %w", err)
	}

	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to replace current wto binary. Close running processes and retry: %w", err)
	}

	return nil
}

func copyFile(sourcePath string, targetPath string, mode os.FileMode) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", sourcePath, err)
	}
	defer source.Close()

	target, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("failed to create target file %s: %w", targetPath, err)
	}

	if _, err := io.Copy(target, source); err != nil {
		target.Close()
		return fmt.Errorf("failed to copy file %s to %s: %w", sourcePath, targetPath, err)
	}

	if err := target.Close(); err != nil {
		return fmt.Errorf("failed to finalize file %s: %w", targetPath, err)
	}

	return nil
}
