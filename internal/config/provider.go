package config

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type repoRootFinder interface {
	RepoRoot(ctx context.Context) (string, error)
}

// FileProviderOptions customizes the file-backed config provider.
type FileProviderOptions struct {
	Git           repoRootFinder
	Stderr        io.Writer
	UserConfigDir func() (string, error)
	ReadFile      func(path string) ([]byte, error)
	WriteFile     func(name string, data []byte, perm os.FileMode) error
	MkdirAll      func(path string, perm os.FileMode) error
	Stat          func(name string) (os.FileInfo, error)
}

type fileProvider struct {
	git           repoRootFinder
	stderr        io.Writer
	userConfigDir func() (string, error)
	readFile      func(path string) ([]byte, error)
	writeFile     func(name string, data []byte, perm os.FileMode) error
	mkdirAll      func(path string, perm os.FileMode) error
	stat          func(name string) (os.FileInfo, error)
}

type staticProvider struct {
	cfg Config
}

// NewFileProvider creates a provider that reads global/repo config files.
func NewFileProvider(opts FileProviderOptions) Provider {
	if opts.Stderr == nil {
		opts.Stderr = io.Discard
	}
	if opts.UserConfigDir == nil {
		opts.UserConfigDir = os.UserConfigDir
	}
	if opts.ReadFile == nil {
		opts.ReadFile = os.ReadFile
	}
	if opts.WriteFile == nil {
		opts.WriteFile = os.WriteFile
	}
	if opts.MkdirAll == nil {
		opts.MkdirAll = os.MkdirAll
	}
	if opts.Stat == nil {
		opts.Stat = os.Stat
	}

	return &fileProvider{
		git:           opts.Git,
		stderr:        opts.Stderr,
		userConfigDir: opts.UserConfigDir,
		readFile:      opts.ReadFile,
		writeFile:     opts.WriteFile,
		mkdirAll:      opts.MkdirAll,
		stat:          opts.Stat,
	}
}

// NewStaticProvider creates a provider with a fixed config (no file I/O).
func NewStaticProvider(cfg Config) Provider {
	return &staticProvider{cfg: cfg}
}

func (p *fileProvider) Load(ctx context.Context) Config {
	effective := DefaultConfig()

	globalPath, err := resolveGlobalConfigPath(p.userConfigDir)
	if err != nil {
		p.warnf(
			"failed to resolve global config path: %v. Continuing with built-in defaults",
			err,
		)
	} else {
		effective = p.mergeConfigSource(effective, globalPath, "global")
	}

	if p.git == nil {
		return effective
	}

	repoRoot, err := p.git.RepoRoot(ctx)
	if err != nil {
		// Outside a repository should not fail config loading.
		return effective
	}

	repoPath := resolveRepoConfigPath(repoRoot)
	if strings.TrimSpace(repoPath) == "" {
		return effective
	}

	return p.mergeConfigSource(effective, repoPath, "repo")
}

func (p *fileProvider) InitGlobal(force bool) (string, error) {
	path, err := resolveGlobalConfigPath(p.userConfigDir)
	if err != nil {
		return "", err
	}

	info, statErr := p.stat(path)
	switch {
	case statErr == nil:
		if info.IsDir() {
			return "", fmt.Errorf("config path points to a directory: %s. Remove it and retry", path)
		}
		if !force {
			return "", fmt.Errorf("config file already exists at %s. Use `wto config init --force` to overwrite", path)
		}
	case errors.Is(statErr, os.ErrNotExist):
		// continue
	default:
		return "", fmt.Errorf("failed to inspect config file at %s: %w", path, statErr)
	}

	if err := p.mkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("failed to create config directory for %s: %w", path, err)
	}

	body, err := marshalConfig(DefaultConfig())
	if err != nil {
		return "", err
	}

	if err := p.writeFile(path, body, 0o644); err != nil {
		return "", fmt.Errorf("failed to write config file at %s: %w", path, err)
	}

	return path, nil
}

func (p *fileProvider) mergeConfigSource(base Config, path string, source string) Config {
	override, err := loadOverrideFromFile(path, p.readFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return base
		}

		p.warnf(
			"ignoring invalid %s config at %s: %v. Fix the file format/keys and retry",
			source,
			path,
			err,
		)
		return base
	}

	return mergeConfig(base, override)
}

func (p *fileProvider) warnf(format string, args ...any) {
	if p.stderr == nil {
		return
	}
	_, _ = fmt.Fprintf(p.stderr, "warning: "+format+"\n", args...)
}

func (p *staticProvider) Load(context.Context) Config {
	return p.cfg
}

func (p *staticProvider) InitGlobal(bool) (string, error) {
	return "", fmt.Errorf("config init is not available for static provider")
}
