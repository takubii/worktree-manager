package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/takubii/git-worktree-opener/internal/config"
	openerpkg "github.com/takubii/git-worktree-opener/internal/opener"
)

type fakeSelector struct {
	index       int
	err         error
	calls       int
	lastPrompt  string
	lastOptions []string
}

func (s *fakeSelector) Select(_ context.Context, prompt string, options []string) (int, error) {
	s.calls++
	s.lastPrompt = prompt
	s.lastOptions = append([]string(nil), options...)
	return s.index, s.err
}

type fakeOpener struct {
	kind   string
	path   string
	window openerpkg.WindowMode
	err    error
	call   int
}

func (o *fakeOpener) Open(_ context.Context, kind string, path string, window openerpkg.WindowMode) error {
	o.call++
	o.kind = kind
	o.path = path
	o.window = window
	return o.err
}

type fakeAfterRunner struct {
	command string
	path    string
	call    int
	err     error
}

func (r *fakeAfterRunner) Run(_ context.Context, commandTemplate string, path string) error {
	r.call++
	r.command = commandTemplate
	r.path = path
	return r.err
}

func TestOpenCommand_OpensSelectedWorktree(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	mainPath := toPosixPathForOpen(t.TempDir())
	featurePath := toPosixPathForOpen(t.TempDir())
	gitClient := &fakeGitClient{
		output: "worktree " + mainPath + "\nHEAD abc\nbranch refs/heads/main\n\nworktree " + featurePath + "\nHEAD def\nbranch refs/heads/feature/x\n\n",
	}
	selector := &fakeSelector{index: 1}
	openExec := &fakeOpener{}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &stdout,
		Stderr:   &stderr,
		Git:      gitClient,
		LookPath: newTestLookPath(map[string]bool{"code": true}),
		Selector: selector,
		Opener:   openExec,
	})
	cmd.SetArgs([]string{"open", "--open", "vscode"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if selector.calls != 1 {
		t.Fatalf("expected selector to be called once, got %d", selector.calls)
	}
	if gitClient.worktreePruneCall != 1 {
		t.Fatalf("expected WorktreePrune to be called once, got %d", gitClient.worktreePruneCall)
	}
	if openExec.call != 1 {
		t.Fatalf("expected opener to be called once, got %d", openExec.call)
	}
	if openExec.kind != "vscode" {
		t.Fatalf("unexpected opener kind: %q", openExec.kind)
	}
	if openExec.path != featurePath {
		t.Fatalf("unexpected opener path: %q", openExec.path)
	}
	if openExec.window != openerpkg.WindowNew {
		t.Fatalf("unexpected window mode: %q", openExec.window)
	}
}

func TestOpenCommand_OpensWorktreeByBranch(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	mainPath := toPosixPathForOpen(t.TempDir())
	featurePath := toPosixPathForOpen(t.TempDir())
	gitClient := &fakeGitClient{
		output: "worktree " + mainPath + "\nHEAD abc\nbranch refs/heads/main\n\n" +
			"worktree " + featurePath + "\nHEAD def\nbranch refs/heads/feature/x\n\n",
	}
	selector := &fakeSelector{index: 0}
	openExec := &fakeOpener{}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &stdout,
		Stderr:   &stderr,
		Git:      gitClient,
		Selector: selector,
		Opener:   openExec,
	})
	cmd.SetArgs([]string{"open", "--branch", "feature/x"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if selector.calls != 0 {
		t.Fatalf("expected selector not to be called when --branch is provided, got %d", selector.calls)
	}
	if openExec.call != 1 {
		t.Fatalf("expected opener to be called once, got %d", openExec.call)
	}
	if openExec.path != featurePath {
		t.Fatalf("unexpected opener path: %q", openExec.path)
	}
}

func TestOpenCommand_ReturnsErrorWhenBranchHasNoLinkedWorktree(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	mainPath := toPosixPathForOpen(t.TempDir())
	gitClient := &fakeGitClient{
		output: "worktree " + mainPath + "\nHEAD abc\nbranch refs/heads/main\n\n",
	}
	selector := &fakeSelector{index: 0}
	openExec := &fakeOpener{}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &stdout,
		Stderr:   &stderr,
		Git:      gitClient,
		Selector: selector,
		Opener:   openExec,
	})
	cmd.SetArgs([]string{"open", "--branch", "feature/missing"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected Execute() to return error")
	}
	if !strings.Contains(err.Error(), "does not have a linked active worktree") {
		t.Fatalf("unexpected error message: %v", err)
	}
	if selector.calls != 0 {
		t.Fatalf("expected selector not to be called on --branch path, got %d", selector.calls)
	}
	if openExec.call != 0 {
		t.Fatalf("opener should not be called when branch has no worktree, got %d", openExec.call)
	}
}

func TestOpenCommand_SkipsPrunableWorktrees(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	livePath := toPosixPathForOpen(t.TempDir())
	gitClient := &fakeGitClient{
		output: "worktree C:/worktrees/stale\nHEAD abc\nbranch refs/heads/aaa\nprunable gitdir file points to non-existent location\n\n" +
			"worktree " + livePath + "\nHEAD def\nbranch refs/heads/main\n\n",
	}
	selector := &fakeSelector{index: 0}
	openExec := &fakeOpener{}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &stderr,
		Git:      gitClient,
		Selector: selector,
		Opener:   openExec,
	})
	cmd.SetArgs([]string{"open"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if openExec.path != livePath {
		t.Fatalf("unexpected opener path: %q", openExec.path)
	}
	if !strings.Contains(stderr.String(), "skipped 1 stale worktree entries") {
		t.Fatalf("expected stale warning, got: %s", stderr.String())
	}
}

func TestOpenCommand_SkipsMissingWorktrees(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	livePath := toPosixPathForOpen(t.TempDir())
	gitClient := &fakeGitClient{
		output: "worktree C:/worktrees/missing\nHEAD abc\nbranch refs/heads/aaa\n\n" +
			"worktree " + livePath + "\nHEAD def\nbranch refs/heads/main\n\n",
	}
	selector := &fakeSelector{index: 0}
	openExec := &fakeOpener{}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &stderr,
		Git:      gitClient,
		Selector: selector,
		Opener:   openExec,
	})
	cmd.SetArgs([]string{"open"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if openExec.path != livePath {
		t.Fatalf("unexpected opener path: %q", openExec.path)
	}
	if !strings.Contains(stderr.String(), "skipped 1 missing worktree entries") {
		t.Fatalf("expected missing warning, got: %s", stderr.String())
	}
}

func TestOpenCommand_ReturnsErrorWhenOnlyPrunableRemain(t *testing.T) {
	t.Parallel()

	gitClient := &fakeGitClient{
		output: "worktree C:/worktrees/stale\nHEAD abc\nbranch refs/heads/aaa\nprunable gitdir file points to non-existent location\n\n",
	}
	selector := &fakeSelector{index: 0}
	openExec := &fakeOpener{}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: selector,
		Opener:   openExec,
	})
	cmd.SetArgs([]string{"open"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected Execute() to return error")
	}
	if !strings.Contains(err.Error(), "no valid worktrees found after filtering stale/missing entries") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenCommand_ReturnsErrorWhenOnlyMissingRemain(t *testing.T) {
	t.Parallel()

	gitClient := &fakeGitClient{
		output: "worktree C:/worktrees/missing\nHEAD abc\nbranch refs/heads/aaa\n\n",
	}
	selector := &fakeSelector{index: 0}
	openExec := &fakeOpener{}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: selector,
		Opener:   openExec,
	})
	cmd.SetArgs([]string{"open"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected Execute() to return error")
	}
	if !strings.Contains(err.Error(), "no valid worktrees found after filtering stale/missing entries") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenCommand_ReturnsErrorWhenNoWorktreeExists(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	gitClient := &fakeGitClient{output: ""}
	selector := &fakeSelector{index: 0}
	openExec := &fakeOpener{}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &stdout,
		Stderr:   &stderr,
		Git:      gitClient,
		Selector: selector,
		Opener:   openExec,
	})
	cmd.SetArgs([]string{"open"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected Execute() to return error")
	}
	if !strings.Contains(err.Error(), "no worktrees found") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestOpenCommand_ReturnsErrorWhenExplicitVSCodeIsUnavailable(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	gitClient := &fakeGitClient{
		output: "worktree C:/repo\nHEAD abc\nbranch refs/heads/main\n\n",
	}
	selector := &fakeSelector{index: 0}
	openExec := &fakeOpener{}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &stdout,
		Stderr:   &stderr,
		Git:      gitClient,
		LookPath: newTestLookPath(map[string]bool{}),
		Selector: selector,
		Opener:   openExec,
	})
	cmd.SetArgs([]string{"open", "--open", "vscode"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected Execute() to return error")
	}
	if !strings.Contains(err.Error(), "`--open vscode` was requested but `code` command was not found") {
		t.Fatalf("unexpected error: %v", err)
	}
	if openExec.call != 0 {
		t.Fatalf("opener should not be called, got %d", openExec.call)
	}
}

func TestOpenCommand_ReturnsErrorWhenExplicitCursorIsUnavailable(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	gitClient := &fakeGitClient{
		output: "worktree C:/repo\nHEAD abc\nbranch refs/heads/main\n\n",
	}
	selector := &fakeSelector{index: 0}
	openExec := &fakeOpener{}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &stdout,
		Stderr:   &stderr,
		Git:      gitClient,
		LookPath: newTestLookPath(map[string]bool{}),
		Selector: selector,
		Opener:   openExec,
	})
	cmd.SetArgs([]string{"open", "--open", "cursor"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected Execute() to return error")
	}
	if !strings.Contains(err.Error(), "`--open cursor` was requested but `cursor` command was not found") {
		t.Fatalf("unexpected error: %v", err)
	}
	if openExec.call != 0 {
		t.Fatalf("opener should not be called, got %d", openExec.call)
	}
}

func TestOpenCommand_ReturnsSelectorError(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	mainPath := toPosixPathForOpen(t.TempDir())
	featurePath := toPosixPathForOpen(t.TempDir())
	gitClient := &fakeGitClient{
		output: "worktree " + mainPath + "\nHEAD abc\nbranch refs/heads/main\n\nworktree " + featurePath + "\nHEAD def\nbranch refs/heads/feature/x\n\n",
	}
	selector := &fakeSelector{err: errors.New("selection canceled")}
	openExec := &fakeOpener{}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &stdout,
		Stderr:   &stderr,
		Git:      gitClient,
		Selector: selector,
		Opener:   openExec,
	})
	cmd.SetArgs([]string{"open"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected Execute() to return error")
	}
	if !strings.Contains(err.Error(), "selection canceled") {
		t.Fatalf("unexpected error message: %v", err)
	}
	if openExec.call != 0 {
		t.Fatalf("opener should not be called on selector error, got %d", openExec.call)
	}
}

func TestOpenCommand_UsesReuseWindowWhenRequested(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	repoPath := toPosixPathForOpen(t.TempDir())
	gitClient := &fakeGitClient{
		output: "worktree " + repoPath + "\nHEAD abc\nbranch refs/heads/main\n\n",
	}
	selector := &fakeSelector{index: 0}
	openExec := &fakeOpener{}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &stdout,
		Stderr:   &stderr,
		Git:      gitClient,
		Selector: selector,
		Opener:   openExec,
	})
	cmd.SetArgs([]string{"open", "--window", "reuse"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if openExec.window != openerpkg.WindowReuse {
		t.Fatalf("unexpected window mode: %q", openExec.window)
	}
}

func TestOpenCommand_PrintCDOutputsHintsAfterOpen(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	worktreePath := toPosixPathForOpen(t.TempDir())
	hints := &fakeEnterRunner{
		hints: []string{
			"cmd.exe: cd /d \"C:\\repo\"",
			"PowerShell: Set-Location -LiteralPath 'C:\\repo'",
		},
	}
	gitClient := &fakeGitClient{
		output: "worktree " + worktreePath + "\nHEAD abc\nbranch refs/heads/main\n\n",
	}
	selector := &fakeSelector{index: 0}
	openExec := &fakeOpener{}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &stdout,
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: selector,
		Opener:   openExec,
		Enter:    hints,
	})
	cmd.SetArgs([]string{"open", "--print-cd"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if openExec.call != 1 {
		t.Fatalf("expected opener to be called once, got %d", openExec.call)
	}
	if !strings.Contains(stdout.String(), "cmd.exe: cd /d") {
		t.Fatalf("expected cd hint output, got: %s", stdout.String())
	}
	if hints.hintsPath != worktreePath {
		t.Fatalf("unexpected hint path: %q", hints.hintsPath)
	}
}

func TestOpenCommand_RunsAfterCommand(t *testing.T) {
	t.Parallel()

	worktreePath := toPosixPathForOpen(t.TempDir())
	gitClient := &fakeGitClient{
		output: "worktree " + worktreePath + "\nHEAD abc\nbranch refs/heads/main\n\n",
	}
	selector := &fakeSelector{index: 0}
	openExec := &fakeOpener{}
	after := &fakeAfterRunner{}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: selector,
		Opener:   openExec,
		After:    after,
	})
	cmd.SetArgs([]string{"open", "--after", "echo {path}"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if after.call != 1 {
		t.Fatalf("expected after command to run once, got %d", after.call)
	}
	if after.command != "echo {path}" {
		t.Fatalf("unexpected after command template: %q", after.command)
	}
	if after.path != worktreePath {
		t.Fatalf("unexpected after command path: %q", after.path)
	}
}

func TestOpenCommand_ReturnsErrorWhenAfterCommandFails(t *testing.T) {
	t.Parallel()

	worktreePath := toPosixPathForOpen(t.TempDir())
	gitClient := &fakeGitClient{
		output: "worktree " + worktreePath + "\nHEAD abc\nbranch refs/heads/main\n\n",
	}
	selector := &fakeSelector{index: 0}
	openExec := &fakeOpener{}
	after := &fakeAfterRunner{err: errors.New("after failed")}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: selector,
		Opener:   openExec,
		After:    after,
	})
	cmd.SetArgs([]string{"open", "--after", "echo {path}"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected Execute() to return error")
	}
	if !strings.Contains(err.Error(), "after failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenCommand_ReturnsErrorForInvalidWindowMode(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	gitClient := &fakeGitClient{
		output: "worktree C:/repo\nHEAD abc\nbranch refs/heads/main\n\n",
	}
	selector := &fakeSelector{index: 0}
	openExec := &fakeOpener{}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &stdout,
		Stderr:   &stderr,
		Git:      gitClient,
		Selector: selector,
		Opener:   openExec,
	})
	cmd.SetArgs([]string{"open", "--window", "invalid"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected Execute() to return error")
	}
	if !strings.Contains(err.Error(), "invalid window mode") {
		t.Fatalf("unexpected error message: %v", err)
	}
	if openExec.call != 0 {
		t.Fatalf("opener should not be called on invalid window mode, got %d", openExec.call)
	}
}

func TestOpenCommand_ReturnsPruneError(t *testing.T) {
	t.Parallel()

	gitClient := &fakeGitClient{
		worktreePruneErr: errors.New("failed to run `git worktree prune --expire now`"),
	}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: &fakeSelector{index: 0},
		Opener:   &fakeOpener{},
	})
	cmd.SetArgs([]string{"open"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected Execute() to return error")
	}
	if !strings.Contains(err.Error(), "worktree prune") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestOpenCommand_UsesConfigDefaultsWhenFlagsAreNotProvided(t *testing.T) {
	t.Parallel()

	repoPath := toPosixPathForOpen(t.TempDir())
	gitClient := &fakeGitClient{
		output: "worktree " + repoPath + "\nHEAD abc\nbranch refs/heads/main\n\n",
	}
	selector := &fakeSelector{index: 0}
	openExec := &fakeOpener{}
	cfgProvider := &fakeConfigProvider{
		cfg: config.Config{
			Remote:              config.DefaultRemote,
			BaseBranch:          config.DefaultBaseBranch,
			WorktreeDirTemplate: config.DefaultWorktreeDirTemplate,
			Open: config.Open{
				Default: "vscode",
				Window:  "reuse",
			},
			RM: config.RM{
				DeleteBranch: config.DeleteBranchSafe,
			},
		},
	}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: selector,
		Opener:   openExec,
		Config:   cfgProvider,
	})
	cmd.SetArgs([]string{"open"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if openExec.kind != "vscode" {
		t.Fatalf("unexpected opener kind: %q", openExec.kind)
	}
	if openExec.window != openerpkg.WindowReuse {
		t.Fatalf("unexpected window mode: %q", openExec.window)
	}
	if cfgProvider.loadCalls != 1 {
		t.Fatalf("expected one config load, got %d", cfgProvider.loadCalls)
	}
}

func TestOpenCommand_FlagsOverrideConfigDefaults(t *testing.T) {
	t.Parallel()

	repoPath := toPosixPathForOpen(t.TempDir())
	gitClient := &fakeGitClient{
		output: "worktree " + repoPath + "\nHEAD abc\nbranch refs/heads/main\n\n",
	}
	selector := &fakeSelector{index: 0}
	openExec := &fakeOpener{}
	cfgProvider := &fakeConfigProvider{
		cfg: config.Config{
			Remote:              config.DefaultRemote,
			BaseBranch:          config.DefaultBaseBranch,
			WorktreeDirTemplate: config.DefaultWorktreeDirTemplate,
			Open: config.Open{
				Default: "cursor",
				Window:  "reuse",
			},
			RM: config.RM{
				DeleteBranch: config.DeleteBranchSafe,
			},
		},
	}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: selector,
		Opener:   openExec,
		Config:   cfgProvider,
	})
	cmd.SetArgs([]string{"open", "--open", "system", "--window", "new"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if openExec.kind != "system" {
		t.Fatalf("unexpected opener kind: %q", openExec.kind)
	}
	if openExec.window != openerpkg.WindowNew {
		t.Fatalf("unexpected window mode: %q", openExec.window)
	}
}

func newTestLookPath(available map[string]bool) func(file string) (string, error) {
	return func(file string) (string, error) {
		if available[file] {
			return file, nil
		}
		return "", fmt.Errorf("%s not found", file)
	}
}

func toPosixPathForOpen(path string) string {
	return strings.ReplaceAll(path, "\\", "/")
}
