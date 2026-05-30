package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// AppConfigDirName is the global config directory under os.UserConfigDir().
	AppConfigDirName = "worktree-manager"
	// GlobalConfigFileName is the global config filename.
	GlobalConfigFileName = "config.json"
	// RepoConfigFileName is the repository-local override config filename.
	RepoConfigFileName = ".wtmconfig.json"
)

func resolveGlobalConfigPath(userConfigDir func() (string, error)) (string, error) {
	if userConfigDir == nil {
		userConfigDir = os.UserConfigDir
	}

	root, err := userConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve user config directory: %w", err)
	}

	root = strings.TrimSpace(root)
	if root == "" {
		return "", fmt.Errorf("user config directory is empty")
	}

	return filepath.Join(root, AppConfigDirName, GlobalConfigFileName), nil
}

func resolveRepoConfigPath(repoRoot string) string {
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		return ""
	}
	return filepath.Join(repoRoot, RepoConfigFileName)
}
