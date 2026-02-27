package updater

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/takubii/git-worktree-opener/internal/execerr"
)

func replaceBinaryWindows(
	ctx context.Context,
	commandContext commandContextFunc,
	startCommand startCommandFunc,
	stagedBinaryPath string,
	targetBinaryPath string,
) error {
	scriptPath, err := writeWindowsReplaceScript(stagedBinaryPath, targetBinaryPath)
	if err != nil {
		return err
	}

	cmd := commandContext(ctx, "cmd", "/c", scriptPath)
	if err := startCommand(cmd); err != nil {
		_ = os.Remove(scriptPath)
		return execerr.Build(
			"cmd /c "+scriptPath,
			err.Error(),
			"run `wto update` again or copy the downloaded wto.exe over your current binary manually",
		)
	}

	return nil
}

func writeWindowsReplaceScript(stagedBinaryPath string, targetBinaryPath string) (string, error) {
	path := filepath.Join(os.TempDir(), fmt.Sprintf("wto-replace-%d.cmd", os.Getpid()))
	content := buildWindowsReplaceScript(stagedBinaryPath, targetBinaryPath)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return "", fmt.Errorf("failed to write Windows update helper script: %w", err)
	}

	return path, nil
}

func buildWindowsReplaceScript(stagedBinaryPath string, targetBinaryPath string) string {
	staged := escapeBatchValue(stagedBinaryPath)
	target := escapeBatchValue(targetBinaryPath)

	return strings.Join([]string{
		"@echo off",
		"setlocal",
		"set \"SOURCE=" + staged + "\"",
		"set \"TARGET=" + target + "\"",
		"for /L %%I in (1,1,60) do (",
		"  copy /Y \"%SOURCE%\" \"%TARGET%\" >nul 2>nul && goto :done",
		"  timeout /T 1 /NOBREAK >nul",
		")",
		"echo failed to replace wto.exe automatically. Close running wto processes and retry update.",
		"goto :cleanup",
		":done",
		"echo wto update completed. Run `wto --version` in a new terminal.",
		":cleanup",
		"del /F /Q \"%SOURCE%\" >nul 2>nul",
		"del /F /Q \"%~f0\" >nul 2>nul",
	}, "\r\n")
}

func escapeBatchValue(value string) string {
	return strings.ReplaceAll(value, "\"", "\"\"")
}
