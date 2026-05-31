package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/takubii/worktree-manager/internal/config"
	"github.com/takubii/worktree-manager/internal/git"
)

type fakeCommandRunner struct {
	calls []fakeCommandCall
	err   error
}

type fakeCommandCall struct {
	name string
	args []string
	cwd  string
}

func (f *fakeCommandRunner) Run(_ context.Context, name string, args []string, cwd string, _ io.Writer, _ io.Writer) error {
	f.calls = append(f.calls, fakeCommandCall{
		name: name,
		args: append([]string(nil), args...),
		cwd:  cwd,
	})
	return f.err
}

func TestCreateCommand_CreatesWorktreeFromLocalBranch(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	expectedPath := filepath.Clean(filepath.Join(repoRoot, "..", "worktrees", "feature", "local"))
	gitClient := &fakeGitClient{
		repoRoot:      repoRoot,
		localBranches: []string{"main", "feature/local"},
		output:        "worktree " + strings.ReplaceAll(repoRoot, "\\", "/") + "\nHEAD abc\nbranch refs/heads/main\n\n",
	}

	var stdout bytes.Buffer
	cmd := NewRootCmd(Dependencies{
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
		Git:    gitClient,
		Config: config.NewStaticProvider(config.DefaultConfig()),
	})
	cmd.SetArgs([]string{"create", "feature/local", "--output", "path"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != expectedPath {
		t.Fatalf("unexpected path output: want=%q got=%q", expectedPath, got)
	}
	if len(gitClient.worktreeAddCalls) != 1 {
		t.Fatalf("expected one WorktreeAdd call, got %d", len(gitClient.worktreeAddCalls))
	}
	if got := gitClient.worktreeAddCalls[0].Path; got != expectedPath {
		t.Fatalf("unexpected worktree path: want=%q got=%q", expectedPath, got)
	}
}

func TestCreateCommand_UsesCreateConfigDefaults(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.Create.Fetch = false
	cfg.Create.Prune = false

	gitClient := &fakeGitClient{
		repoRoot:      t.TempDir(),
		localBranches: []string{"main", "feature/local"},
		output:        "worktree C:/repo\nHEAD abc\nbranch refs/heads/main\n\n",
	}

	cmd := NewRootCmd(Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Git:    gitClient,
		Config: config.NewStaticProvider(cfg),
	})
	cmd.SetArgs([]string{"create", "feature/local"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if gitClient.fetchRemote != "" {
		t.Fatalf("FetchPrune should not be called, got remote=%q", gitClient.fetchRemote)
	}
	if gitClient.worktreePruneCall != 0 {
		t.Fatalf("WorktreePrune should not be called, got %d", gitClient.worktreePruneCall)
	}
}

func TestCreateCommand_CopiesBootstrapFileAfterWorktreeAdd(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	sourcePath := filepath.Join(repoRoot, ".env")
	if err := os.WriteFile(sourcePath, []byte("TOKEN=value\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() returned error: %v", err)
	}
	expectedPath := filepath.Clean(filepath.Join(repoRoot, "..", "worktrees", "feature", "local"))
	cfg := config.DefaultConfig()
	cfg.Create.Fetch = false
	cfg.Create.Prune = false
	cfg.Create.Bootstrap.CopyFiles = []config.CopyFileAction{{
		From: ".env",
		To:   ".env",
	}}

	gitClient := &fakeGitClient{
		repoRoot:      repoRoot,
		localBranches: []string{"main", "feature/local"},
		output:        "worktree " + strings.ReplaceAll(repoRoot, "\\", "/") + "\nHEAD abc\nbranch refs/heads/main\n\n",
		onWorktreeAdd: func(params git.WorktreeAddParams) error {
			return os.MkdirAll(params.Path, 0o755)
		},
	}

	var stderr bytes.Buffer
	cmd := NewRootCmd(Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
		Git:    gitClient,
		Config: config.NewStaticProvider(cfg),
	})
	cmd.SetArgs([]string{"create", "feature/local"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(expectedPath, ".env"))
	if err != nil {
		t.Fatalf("ReadFile() returned error: %v", err)
	}
	if string(got) != "TOKEN=value\n" {
		t.Fatalf("unexpected copied file contents: %q", got)
	}
	if len(gitClient.callLog) < 1 || gitClient.callLog[len(gitClient.callLog)-1] != "WorktreeAdd" {
		t.Fatalf("expected copy after WorktreeAdd, call log: %v", gitClient.callLog)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestCreateCommand_NoBootstrapSkipsConfiguredActions(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Create.Fetch = false
	cfg.Create.Prune = false
	cfg.Create.Bootstrap.CopyFiles = []config.CopyFileAction{{
		From: ".env",
		To:   ".env",
	}}
	cfg.Create.Bootstrap.PostCreate = []config.HookAction{{
		Command: []string{"npm", "install"},
	}}
	runner := &fakeCommandRunner{}
	gitClient := &fakeGitClient{
		repoRoot:      repoRoot,
		localBranches: []string{"main", "feature/local"},
		output:        "worktree " + strings.ReplaceAll(repoRoot, "\\", "/") + "\nHEAD abc\nbranch refs/heads/main\n\n",
		onWorktreeAdd: func(params git.WorktreeAddParams) error {
			return os.MkdirAll(params.Path, 0o755)
		},
	}

	cmd := NewRootCmd(Dependencies{
		Stdout:        &bytes.Buffer{},
		Stderr:        &bytes.Buffer{},
		Git:           gitClient,
		Config:        config.NewStaticProvider(cfg),
		CommandRunner: runner,
	})
	cmd.SetArgs([]string{"create", "feature/local", "--no-bootstrap"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	expectedPath := filepath.Clean(filepath.Join(repoRoot, "..", "worktrees", "feature", "local"))
	if _, err := os.Stat(filepath.Join(expectedPath, ".env")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("bootstrap copy should be skipped, stat err=%v", err)
	}
	if len(runner.calls) != 0 {
		t.Fatalf("post-create hooks should be skipped, got %+v", runner.calls)
	}
}

func TestCreateCommand_DryRunDoesNotMutate(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Create.Bootstrap.CopyFiles = []config.CopyFileAction{{
		From: ".env",
		To:   ".env",
	}}
	cfg.Create.Bootstrap.PostCreate = []config.HookAction{{
		Name:    "install dependencies",
		Command: []string{"npm", "install"},
	}}
	runner := &fakeCommandRunner{}
	gitClient := &fakeGitClient{
		repoRoot:      repoRoot,
		localBranches: []string{"main", "feature/local"},
		output:        "worktree " + strings.ReplaceAll(repoRoot, "\\", "/") + "\nHEAD abc\nbranch refs/heads/main\n\n",
	}
	var stdout bytes.Buffer
	cmd := NewRootCmd(Dependencies{
		Stdout:        &stdout,
		Stderr:        &bytes.Buffer{},
		Git:           gitClient,
		Config:        config.NewStaticProvider(cfg),
		CommandRunner: runner,
	})
	cmd.SetArgs([]string{"create", "feature/local", "--dry-run"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if gitClient.fetchRemote != "" {
		t.Fatalf("FetchPrune should not be called in dry-run, got %q", gitClient.fetchRemote)
	}
	if gitClient.worktreePruneCall != 0 {
		t.Fatalf("WorktreePrune should not be called in dry-run, got %d", gitClient.worktreePruneCall)
	}
	if len(gitClient.worktreeAddCalls) != 0 {
		t.Fatalf("WorktreeAdd should not be called in dry-run, got %+v", gitClient.worktreeAddCalls)
	}
	if len(runner.calls) != 0 {
		t.Fatalf("hooks should not run in dry-run, got %+v", runner.calls)
	}
	if !strings.Contains(stdout.String(), "dry-run: planned create actions") {
		t.Fatalf("expected dry-run output, got: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "run hook \"install dependencies\"") {
		t.Fatalf("expected hook plan, got: %s", stdout.String())
	}
}

func TestCreateCommand_BootstrapMissingOptionalFileWarns(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Create.Fetch = false
	cfg.Create.Prune = false
	cfg.Create.Bootstrap.CopyFiles = []config.CopyFileAction{{
		From: ".env",
		To:   ".env",
	}}
	gitClient := &fakeGitClient{
		repoRoot:      repoRoot,
		localBranches: []string{"main", "feature/local"},
		output:        "worktree " + strings.ReplaceAll(repoRoot, "\\", "/") + "\nHEAD abc\nbranch refs/heads/main\n\n",
		onWorktreeAdd: func(params git.WorktreeAddParams) error {
			return os.MkdirAll(params.Path, 0o755)
		},
	}
	var stderr bytes.Buffer
	cmd := NewRootCmd(Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
		Git:    gitClient,
		Config: config.NewStaticProvider(cfg),
	})
	cmd.SetArgs([]string{"create", "feature/local"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if !strings.Contains(stderr.String(), "optional bootstrap file is missing") {
		t.Fatalf("expected missing optional warning, got: %s", stderr.String())
	}
}

func TestCreateCommand_BootstrapExistingDestinationWarnsWhenOverwriteFalse(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, ".env"), []byte("SOURCE=value\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() returned error: %v", err)
	}
	cfg := config.DefaultConfig()
	cfg.Create.Fetch = false
	cfg.Create.Prune = false
	cfg.Create.Bootstrap.CopyFiles = []config.CopyFileAction{{
		From: ".env",
		To:   ".env",
	}}
	gitClient := &fakeGitClient{
		repoRoot:      repoRoot,
		localBranches: []string{"main", "feature/local"},
		output:        "worktree " + strings.ReplaceAll(repoRoot, "\\", "/") + "\nHEAD abc\nbranch refs/heads/main\n\n",
		onWorktreeAdd: func(params git.WorktreeAddParams) error {
			if err := os.MkdirAll(params.Path, 0o755); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(params.Path, ".env"), []byte("EXISTING=value\n"), 0o600)
		},
	}
	var stderr bytes.Buffer
	cmd := NewRootCmd(Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
		Git:    gitClient,
		Config: config.NewStaticProvider(cfg),
	})
	cmd.SetArgs([]string{"create", "feature/local"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	expectedPath := filepath.Clean(filepath.Join(repoRoot, "..", "worktrees", "feature", "local"))
	got, err := os.ReadFile(filepath.Join(expectedPath, ".env"))
	if err != nil {
		t.Fatalf("ReadFile() returned error: %v", err)
	}
	if string(got) != "EXISTING=value\n" {
		t.Fatalf("existing destination should not be overwritten, got: %q", got)
	}
	if !strings.Contains(stderr.String(), "destination already exists") {
		t.Fatalf("expected existing destination warning, got: %s", stderr.String())
	}
}

func TestCreateCommand_BootstrapMissingRequiredFileFails(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Create.Fetch = false
	cfg.Create.Prune = false
	cfg.Create.Bootstrap.CopyFiles = []config.CopyFileAction{{
		From:     ".env",
		To:       ".env",
		Required: true,
	}}
	gitClient := &fakeGitClient{
		repoRoot:      repoRoot,
		localBranches: []string{"main", "feature/local"},
		output:        "worktree " + strings.ReplaceAll(repoRoot, "\\", "/") + "\nHEAD abc\nbranch refs/heads/main\n\n",
		onWorktreeAdd: func(params git.WorktreeAddParams) error {
			return os.MkdirAll(params.Path, 0o755)
		},
	}
	cmd := NewRootCmd(Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Git:    gitClient,
		Config: config.NewStaticProvider(cfg),
	})
	cmd.SetArgs([]string{"create", "feature/local"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "required bootstrap file is missing") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateCommand_BootstrapRejectsOutsideDestination(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	sourcePath := filepath.Join(repoRoot, ".env")
	if err := os.WriteFile(sourcePath, []byte("TOKEN=value\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() returned error: %v", err)
	}
	cfg := config.DefaultConfig()
	cfg.Create.Fetch = false
	cfg.Create.Prune = false
	cfg.Create.Bootstrap.CopyFiles = []config.CopyFileAction{{
		From: ".env",
		To:   "../outside.env",
	}}
	gitClient := &fakeGitClient{
		repoRoot:      repoRoot,
		localBranches: []string{"main", "feature/local"},
		output:        "worktree " + strings.ReplaceAll(repoRoot, "\\", "/") + "\nHEAD abc\nbranch refs/heads/main\n\n",
		onWorktreeAdd: func(params git.WorktreeAddParams) error {
			return os.MkdirAll(params.Path, 0o755)
		},
	}
	cmd := NewRootCmd(Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Git:    gitClient,
		Config: config.NewStaticProvider(cfg),
	})
	cmd.SetArgs([]string{"create", "feature/local"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "outside the worktree") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateCommand_PostCreateRunsInOrderWithCWD(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Create.Fetch = false
	cfg.Create.Prune = false
	cfg.Create.Bootstrap.PostCreate = []config.HookAction{
		{Command: []string{"npm", "install"}},
		{Command: []string{"npm", "run", "build"}, CWD: "frontend"},
	}
	runner := &fakeCommandRunner{}
	gitClient := &fakeGitClient{
		repoRoot:      repoRoot,
		localBranches: []string{"main", "feature/local"},
		output:        "worktree " + strings.ReplaceAll(repoRoot, "\\", "/") + "\nHEAD abc\nbranch refs/heads/main\n\n",
		onWorktreeAdd: func(params git.WorktreeAddParams) error {
			if err := os.MkdirAll(filepath.Join(params.Path, "frontend"), 0o755); err != nil {
				return err
			}
			return nil
		},
	}
	cmd := NewRootCmd(Dependencies{
		Stdout:        &bytes.Buffer{},
		Stderr:        &bytes.Buffer{},
		Git:           gitClient,
		Config:        config.NewStaticProvider(cfg),
		CommandRunner: runner,
	})
	cmd.SetArgs([]string{"create", "feature/local"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	expectedPath := filepath.Clean(filepath.Join(repoRoot, "..", "worktrees", "feature", "local"))
	if len(runner.calls) != 2 {
		t.Fatalf("expected 2 hook calls, got %+v", runner.calls)
	}
	if runner.calls[0].name != "npm" || strings.Join(runner.calls[0].args, " ") != "install" || runner.calls[0].cwd != expectedPath {
		t.Fatalf("unexpected first hook call: %+v", runner.calls[0])
	}
	if runner.calls[1].cwd != filepath.Join(expectedPath, "frontend") {
		t.Fatalf("unexpected second hook cwd: %+v", runner.calls[1])
	}
}

func TestCreateCommand_PostCreateRejectsOutsideCWD(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Create.Fetch = false
	cfg.Create.Prune = false
	cfg.Create.Bootstrap.PostCreate = []config.HookAction{{
		Command: []string{"npm", "install"},
		CWD:     "../outside",
	}}
	runner := &fakeCommandRunner{}
	gitClient := &fakeGitClient{
		repoRoot:      repoRoot,
		localBranches: []string{"main", "feature/local"},
		output:        "worktree " + strings.ReplaceAll(repoRoot, "\\", "/") + "\nHEAD abc\nbranch refs/heads/main\n\n",
		onWorktreeAdd: func(params git.WorktreeAddParams) error {
			return os.MkdirAll(params.Path, 0o755)
		},
	}
	cmd := NewRootCmd(Dependencies{
		Stdout:        &bytes.Buffer{},
		Stderr:        &bytes.Buffer{},
		Git:           gitClient,
		Config:        config.NewStaticProvider(cfg),
		CommandRunner: runner,
	})
	cmd.SetArgs([]string{"create", "feature/local"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "outside the worktree") {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runner.calls) != 0 {
		t.Fatalf("hook should not run with invalid cwd, got %+v", runner.calls)
	}
}

func TestCreateCommand_PostCreateFailureStopsHooks(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Create.Fetch = false
	cfg.Create.Prune = false
	cfg.Create.Bootstrap.PostCreate = []config.HookAction{
		{Name: "install", Command: []string{"npm", "install"}},
		{Name: "build", Command: []string{"npm", "run", "build"}},
	}
	runner := &fakeCommandRunner{err: errors.New("exit status 1")}
	gitClient := &fakeGitClient{
		repoRoot:      repoRoot,
		localBranches: []string{"main", "feature/local"},
		output:        "worktree " + strings.ReplaceAll(repoRoot, "\\", "/") + "\nHEAD abc\nbranch refs/heads/main\n\n",
		onWorktreeAdd: func(params git.WorktreeAddParams) error {
			return os.MkdirAll(params.Path, 0o755)
		},
	}
	cmd := NewRootCmd(Dependencies{
		Stdout:        &bytes.Buffer{},
		Stderr:        &bytes.Buffer{},
		Git:           gitClient,
		Config:        config.NewStaticProvider(cfg),
		CommandRunner: runner,
	})
	cmd.SetArgs([]string{"create", "feature/local"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if len(runner.calls) != 1 {
		t.Fatalf("expected hook execution to stop after first failure, got %+v", runner.calls)
	}
	if !strings.Contains(err.Error(), "post-create hook \"install\" failed") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "npm install") {
		t.Fatalf("expected failed command in error, got: %v", err)
	}
}
