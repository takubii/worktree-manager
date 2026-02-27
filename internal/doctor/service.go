package doctor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/takubii/git-worktree-opener/internal/config"
)

type repoRootFinder interface {
	RepoRoot(ctx context.Context) (string, error)
}

// Options customizes doctor service behavior.
type Options struct {
	LookPath      func(file string) (string, error)
	Git           repoRootFinder
	UserConfigDir func() (string, error)
	ReadFile      func(path string) ([]byte, error)
	Stat          func(name string) (os.FileInfo, error)
	GOOS          string
}

type service struct {
	lookPath      func(file string) (string, error)
	git           repoRootFinder
	userConfigDir func() (string, error)
	readFile      func(path string) ([]byte, error)
	stat          func(name string) (os.FileInfo, error)
	goos          string
}

// NewService returns the default doctor service.
func NewService(opts Options) Service {
	if opts.UserConfigDir == nil {
		opts.UserConfigDir = os.UserConfigDir
	}
	if opts.ReadFile == nil {
		opts.ReadFile = os.ReadFile
	}
	if opts.Stat == nil {
		opts.Stat = os.Stat
	}
	if strings.TrimSpace(opts.GOOS) == "" {
		opts.GOOS = runtime.GOOS
	}

	return &service{
		lookPath:      opts.LookPath,
		git:           opts.Git,
		userConfigDir: opts.UserConfigDir,
		readFile:      opts.ReadFile,
		stat:          opts.Stat,
		goos:          strings.TrimSpace(opts.GOOS),
	}
}

func (s *service) Run(ctx context.Context) Report {
	results := make([]Result, 0, 8)
	hasCritical := false

	gitOK := s.lookPath != nil
	if s.lookPath == nil {
		results = append(results, Result{
			Name:       "git",
			Level:      LevelCritical,
			Message:    "cannot check `git` command because lookPath is not configured",
			NextAction: "configure lookPath dependency, then run `wto doctor` again",
		})
		hasCritical = true
		gitOK = false
	} else if _, err := s.lookPath("git"); err != nil {
		results = append(results, Result{
			Name:       "git",
			Level:      LevelCritical,
			Message:    "`git` command was not found in PATH",
			NextAction: "install Git and ensure `git --version` works in this shell, then retry",
		})
		hasCritical = true
		gitOK = false
	} else {
		results = append(results, Result{
			Name:       "git",
			Level:      LevelOK,
			Message:    "`git` command is available",
			NextAction: "no action required",
		})
	}

	repoRoot, repoErr := s.repoRoot(ctx, gitOK)
	results = append(results, s.repoResult(repoRoot, repoErr)...)
	results = append(results, s.openerResults()...)
	results = append(results, s.configResults(repoRoot, repoErr)...)
	results = append(results, s.updatePrerequisiteResult()...)

	return Report{
		Results:     results,
		HasCritical: hasCritical,
	}
}

func (s *service) repoRoot(ctx context.Context, gitOK bool) (string, error) {
	if !gitOK {
		return "", errors.New("git command is unavailable")
	}
	if s.git == nil {
		return "", errors.New("git client is not configured")
	}
	return s.git.RepoRoot(ctx)
}

func (s *service) repoResult(repoRoot string, repoErr error) []Result {
	if repoErr == nil {
		return []Result{{
			Name:       "repository",
			Level:      LevelOK,
			Message:    fmt.Sprintf("Git repository is available at %s", repoRoot),
			NextAction: "no action required",
		}}
	}

	return []Result{{
		Name:       "repository",
		Level:      LevelWarn,
		Message:    fmt.Sprintf("Git repository check failed: %v", repoErr),
		NextAction: "run this command inside a Git repository when you want repo-specific checks",
	}}
}

func (s *service) openerResults() []Result {
	results := make([]Result, 0, 2)
	for _, cmd := range []string{"code", "cursor"} {
		if s.lookPath == nil {
			results = append(results, Result{
				Name:       "opener/" + cmd,
				Level:      LevelWarn,
				Message:    "cannot check command availability because lookPath is not configured",
				NextAction: "configure lookPath dependency, then run `wto doctor` again",
			})
			continue
		}
		if _, err := s.lookPath(cmd); err != nil {
			results = append(results, Result{
				Name:       "opener/" + cmd,
				Level:      LevelWarn,
				Message:    fmt.Sprintf("`%s` command is not available", cmd),
				NextAction: fmt.Sprintf("install `%s` CLI or use `--open system`", cmd),
			})
			continue
		}
		results = append(results, Result{
			Name:       "opener/" + cmd,
			Level:      LevelOK,
			Message:    fmt.Sprintf("`%s` command is available", cmd),
			NextAction: "no action required",
		})
	}
	return results
}

func (s *service) configResults(repoRoot string, repoErr error) []Result {
	results := make([]Result, 0, 2)

	globalPath, err := config.GlobalConfigPath(s.userConfigDir)
	if err != nil {
		results = append(results, Result{
			Name:       "config/global",
			Level:      LevelWarn,
			Message:    fmt.Sprintf("failed to resolve global config path: %v", err),
			NextAction: "ensure the user config directory is accessible, then retry",
		})
	} else {
		results = append(results, s.validateConfigFile("config/global", globalPath))
	}

	if repoErr != nil || strings.TrimSpace(repoRoot) == "" {
		results = append(results, Result{
			Name:       "config/repo",
			Level:      LevelWarn,
			Message:    "repository config check was skipped (repository root unavailable)",
			NextAction: "run `wto doctor` inside a Git repository to validate .wtoconfig.json",
		})
		return results
	}

	repoPath := config.RepoConfigPath(repoRoot)
	results = append(results, s.validateConfigFile("config/repo", repoPath))
	return results
}

func (s *service) validateConfigFile(name string, path string) Result {
	info, err := s.stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Result{
				Name:       name,
				Level:      LevelOK,
				Message:    fmt.Sprintf("config file not found at %s (0-config mode)", path),
				NextAction: "no action required",
			}
		}
		return Result{
			Name:       name,
			Level:      LevelWarn,
			Message:    fmt.Sprintf("failed to inspect config file %s: %v", path, err),
			NextAction: "check file permissions and retry",
		}
	}

	if info.IsDir() {
		return Result{
			Name:       name,
			Level:      LevelWarn,
			Message:    fmt.Sprintf("config path points to a directory: %s", path),
			NextAction: "replace the directory with a JSON config file and retry",
		}
	}

	if err := config.ValidateFile(path, s.readFile); err != nil {
		return Result{
			Name:       name,
			Level:      LevelWarn,
			Message:    fmt.Sprintf("config file is invalid at %s: %v", path, err),
			NextAction: "fix JSON keys/values and retry",
		}
	}

	return Result{
		Name:       name,
		Level:      LevelOK,
		Message:    fmt.Sprintf("config file is valid at %s", path),
		NextAction: "no action required",
	}
}

func (s *service) updatePrerequisiteResult() []Result {
	if s.lookPath == nil {
		return []Result{{
			Name:       "update/prerequisites",
			Level:      LevelWarn,
			Message:    "cannot check update prerequisites because lookPath is not configured",
			NextAction: "configure lookPath dependency, then run `wto doctor` again",
		}}
	}

	if s.goos == "windows" {
		missing := make([]string, 0, 3)
		for _, cmd := range []string{"curl", "tar", "certutil"} {
			if _, err := s.lookPath(cmd); err != nil {
				missing = append(missing, cmd)
			}
		}
		if len(missing) > 0 {
			return []Result{{
				Name:       "update/prerequisites",
				Level:      LevelWarn,
				Message:    fmt.Sprintf("missing commands for Windows update flow: %s", strings.Join(missing, ", ")),
				NextAction: "install missing commands or use an environment where they are available",
			}}
		}
		return []Result{{
			Name:       "update/prerequisites",
			Level:      LevelOK,
			Message:    "required commands for Windows update flow are available (curl, tar, certutil)",
			NextAction: "no action required",
		}}
	}

	if _, err := s.lookPath("curl"); err == nil {
		return []Result{{
			Name:       "update/prerequisites",
			Level:      LevelOK,
			Message:    "update prerequisite is available (`curl`)",
			NextAction: "no action required",
		}}
	}
	if _, err := s.lookPath("wget"); err == nil {
		return []Result{{
			Name:       "update/prerequisites",
			Level:      LevelOK,
			Message:    "update prerequisite is available (`wget`)",
			NextAction: "no action required",
		}}
	}

	return []Result{{
		Name:       "update/prerequisites",
		Level:      LevelWarn,
		Message:    "missing update prerequisites (`curl` or `wget`)",
		NextAction: "install curl or wget and retry",
	}}
}
