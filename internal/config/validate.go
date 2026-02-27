package config

// GlobalConfigPath resolves the global config file path.
func GlobalConfigPath(userConfigDir func() (string, error)) (string, error) {
	return resolveGlobalConfigPath(userConfigDir)
}

// RepoConfigPath resolves the repository override config file path.
func RepoConfigPath(repoRoot string) string {
	return resolveRepoConfigPath(repoRoot)
}

// ValidateFile validates a config file using the same parser/normalizer as runtime loading.
func ValidateFile(path string, readFile func(string) ([]byte, error)) error {
	_, err := loadOverrideFromFile(path, readFile)
	return err
}
