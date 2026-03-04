package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/takubii/git-worktree-opener/internal/config"
	"github.com/takubii/git-worktree-opener/internal/opener"
)

type fakeEnterRunner struct {
	hints      []string
	hintsPath  string
	startPath  string
	startTmux  opener.TmuxMode
	startCalls int
	startErr   error
}

func (r *fakeEnterRunner) FormatCDHints(path string) []string {
	r.hintsPath = path
	return append([]string(nil), r.hints...)
}

func (r *fakeEnterRunner) StartShell(_ context.Context, path string, tmuxMode opener.TmuxMode) error {
	r.startCalls++
	r.startPath = path
	r.startTmux = tmuxMode
	return r.startErr
}

func TestEnterCommand_PrintsSelectedPathByDefault(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	mainPath := toPosixPath(t.TempDir())
	featurePath := toPosixPath(t.TempDir())
	runner := &fakeEnterRunner{
		hints: []string{"cd /d C:\\repo-feature"},
	}
	gitClient := &fakeGitClient{
		output: "worktree " + mainPath + "\nHEAD abc\nbranch refs/heads/main\n\nworktree " + featurePath + "\nHEAD def\nbranch refs/heads/feature/x\n\n",
	}
	selector := &fakeSelector{index: 1}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &stdout,
		Stderr:   &stderr,
		Git:      gitClient,
		Selector: selector,
		Enter:    runner,
	})
	cmd.SetArgs([]string{"enter"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if got := strings.TrimSpace(stdout.String()); got != featurePath {
		t.Fatalf("unexpected selected path output: %q", got)
	}
	if runner.startCalls != 0 {
		t.Fatalf("expected StartShell() not to be called, got %d", runner.startCalls)
	}
	if runner.hintsPath != "" {
		t.Fatalf("expected FormatCDHints() not to be called, got %q", runner.hintsPath)
	}
	if gitClient.worktreePruneCall != 1 {
		t.Fatalf("expected WorktreePrune to be called once, got %d", gitClient.worktreePruneCall)
	}
}

func TestEnterCommand_PrintCDOutputsHints(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	worktreePath := toPosixPath(t.TempDir())
	runner := &fakeEnterRunner{
		hints: []string{
			`cmd.exe: cd /d "C:\repo"`,
			`PowerShell: Set-Location -LiteralPath 'C:\repo'`,
		},
	}
	gitClient := &fakeGitClient{
		output: "worktree " + worktreePath + "\nHEAD abc\nbranch refs/heads/main\n\n",
	}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &stdout,
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: &fakeSelector{index: 0},
		Enter:    runner,
	})
	cmd.SetArgs([]string{"enter", "--print-cd"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, `cmd.exe: cd /d "C:\repo"`) {
		t.Fatalf("expected cmd hint in output, got: %q", out)
	}
	if !strings.Contains(out, `PowerShell: Set-Location -LiteralPath 'C:\repo'`) {
		t.Fatalf("expected powershell hint in output, got: %q", out)
	}
	if runner.hintsPath != worktreePath {
		t.Fatalf("unexpected hint path: %q", runner.hintsPath)
	}
	if runner.startCalls != 0 {
		t.Fatalf("expected StartShell() not to be called, got %d", runner.startCalls)
	}
}

func TestEnterCommand_ShellModeStartsRunner(t *testing.T) {
	t.Parallel()

	worktreePath := toPosixPath(t.TempDir())
	runner := &fakeEnterRunner{
		hints: []string{"cd /repo"},
	}
	gitClient := &fakeGitClient{
		output: "worktree " + worktreePath + "\nHEAD abc\nbranch refs/heads/main\n\n",
	}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: &fakeSelector{index: 0},
		Enter:    runner,
	})
	cmd.SetArgs([]string{"enter", "--shell"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if runner.startCalls != 1 {
		t.Fatalf("expected StartShell() to be called once, got %d", runner.startCalls)
	}
	if runner.startPath != worktreePath {
		t.Fatalf("unexpected StartShell path: %q", runner.startPath)
	}
	if runner.startTmux != opener.TmuxModeAuto {
		t.Fatalf("unexpected StartShell tmux mode: %q", runner.startTmux)
	}
}

func TestEnterCommand_BranchModeSelectsWithoutPrompt(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	mainPath := toPosixPath(t.TempDir())
	featurePath := toPosixPath(t.TempDir())
	runner := &fakeEnterRunner{}
	selector := &fakeSelector{index: 0, err: errors.New("selector should not be called")}
	gitClient := &fakeGitClient{
		output: "worktree " + mainPath + "\nHEAD abc\nbranch refs/heads/main\n\n" +
			"worktree " + featurePath + "\nHEAD def\nbranch refs/heads/feature/x\n\n",
	}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &stdout,
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: selector,
		Enter:    runner,
	})
	cmd.SetArgs([]string{"enter", "--branch", "feature/x"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if selector.calls != 0 {
		t.Fatalf("expected selector not to be called, got %d", selector.calls)
	}
	if got := strings.TrimSpace(stdout.String()); got != featurePath {
		t.Fatalf("unexpected path output: %q", got)
	}
}

func TestEnterCommand_BranchModeReturnsErrorWhenNoMatch(t *testing.T) {
	t.Parallel()

	mainPath := toPosixPath(t.TempDir())
	cmd := NewRootCmd(Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Git: &fakeGitClient{
			output: "worktree " + mainPath + "\nHEAD abc\nbranch refs/heads/main\n\n",
		},
		Selector: &fakeSelector{index: 0},
		Enter:    &fakeEnterRunner{},
	})
	cmd.SetArgs([]string{"enter", "--branch", "feature/missing"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected Execute() to return error")
	}
	if !strings.Contains(err.Error(), "does not have a linked active worktree") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestEnterCommand_BranchModeReturnsErrorWhenMultipleMatches(t *testing.T) {
	t.Parallel()

	cmd := NewRootCmd(Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Git: &fakeGitClient{
			output: "worktree C:/worktrees/a\nHEAD abc\nbranch refs/heads/feature/x\n\n" +
				"worktree C:/worktrees/b\nHEAD def\nbranch refs/heads/feature/x\n\n",
		},
		Selector: &fakeSelector{index: 0},
		Enter:    &fakeEnterRunner{},
	})
	cmd.SetArgs([]string{"enter", "--branch", "feature/x"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected Execute() to return error")
	}
	if !strings.Contains(err.Error(), "multiple worktrees matched branch") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestEnterCommand_ReturnsErrorForConflictingModes(t *testing.T) {
	t.Parallel()

	cmd := NewRootCmd(Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Git:    &fakeGitClient{},
		Enter:  &fakeEnterRunner{},
	})
	cmd.SetArgs([]string{"enter", "--shell", "--print-cd"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected Execute() to return error")
	}
	if !strings.Contains(err.Error(), "cannot be used together") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestEnterCommand_ReturnsErrorWhenTmuxModeUsedWithoutShell(t *testing.T) {
	t.Parallel()

	cmd := NewRootCmd(Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Git: &fakeGitClient{
			output: "worktree C:/repo\nHEAD abc\nbranch refs/heads/main\n\n",
		},
		Selector: &fakeSelector{index: 0},
		Enter:    &fakeEnterRunner{},
	})
	cmd.SetArgs([]string{"enter", "--tmux-mode", "split"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected Execute() to return error")
	}
	if !strings.Contains(err.Error(), "`--tmux-mode` can only be used with `--shell`") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestEnterCommand_ShellModePassesExplicitTmuxMode(t *testing.T) {
	t.Parallel()

	worktreePath := toPosixPath(t.TempDir())
	runner := &fakeEnterRunner{}
	gitClient := &fakeGitClient{
		output: "worktree " + worktreePath + "\nHEAD abc\nbranch refs/heads/main\n\n",
	}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: &fakeSelector{index: 0},
		Enter:    runner,
	})
	cmd.SetArgs([]string{"enter", "--shell", "--tmux-mode", "window"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if runner.startTmux != opener.TmuxModeWindow {
		t.Fatalf("unexpected StartShell tmux mode: %q", runner.startTmux)
	}
}

func TestEnterCommand_ReturnsErrorForInvalidTmuxMode(t *testing.T) {
	t.Parallel()

	worktreePath := toPosixPath(t.TempDir())
	cmd := NewRootCmd(Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Git: &fakeGitClient{
			output: "worktree " + worktreePath + "\nHEAD abc\nbranch refs/heads/main\n\n",
		},
		Selector: &fakeSelector{index: 0},
		Enter:    &fakeEnterRunner{},
	})
	cmd.SetArgs([]string{"enter", "--shell", "--tmux-mode", "invalid"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected Execute() to return error")
	}
	if !strings.Contains(err.Error(), "invalid tmux mode") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestEnterCommand_ShellModeUsesConfigTmuxMode(t *testing.T) {
	t.Parallel()

	worktreePath := toPosixPath(t.TempDir())
	runner := &fakeEnterRunner{}
	gitClient := &fakeGitClient{
		output: "worktree " + worktreePath + "\nHEAD abc\nbranch refs/heads/main\n\n",
	}
	cfgProvider := &fakeConfigProvider{
		cfg: config.Config{
			Remote:              config.DefaultRemote,
			BaseBranch:          config.DefaultBaseBranch,
			WorktreeDirTemplate: config.DefaultWorktreeDirTemplate,
			New:                 config.DefaultConfig().New,
			Open:                config.DefaultConfig().Open,
			Tmux: config.Tmux{
				Mode: config.TmuxModeSplit,
			},
			RM: config.DefaultConfig().RM,
		},
	}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: &fakeSelector{index: 0},
		Enter:    runner,
		Config:   cfgProvider,
	})
	cmd.SetArgs([]string{"enter", "--shell"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if runner.startTmux != opener.TmuxModeSplit {
		t.Fatalf("unexpected StartShell tmux mode: %q", runner.startTmux)
	}
}

func TestEnterCommand_ReturnsErrorWhenNoWorktreesExist(t *testing.T) {
	t.Parallel()

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      &fakeGitClient{output: ""},
		Selector: &fakeSelector{index: 0},
		Enter:    &fakeEnterRunner{},
	})
	cmd.SetArgs([]string{"enter"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected Execute() to return error")
	}
	if !strings.Contains(err.Error(), "no worktrees found") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestEnterCommand_ReturnsErrorWhenOnlyPrunableRemain(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	cmd := NewRootCmd(Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
		Git: &fakeGitClient{
			output: "worktree C:/worktrees/stale\nHEAD abc\nbranch refs/heads/aaa\nprunable gitdir file points to non-existent location\n\n",
		},
		Selector: &fakeSelector{index: 0},
		Enter:    &fakeEnterRunner{},
	})
	cmd.SetArgs([]string{"enter"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected Execute() to return error")
	}
	if !strings.Contains(err.Error(), "no valid worktrees found after pruning stale entries") {
		t.Fatalf("unexpected error message: %v", err)
	}
	if !strings.Contains(stderr.String(), "skipped 1 stale worktree entries") {
		t.Fatalf("expected stale warning, got: %s", stderr.String())
	}
}

func TestEnterCommand_ReturnsSelectorError(t *testing.T) {
	t.Parallel()

	cmd := NewRootCmd(Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Git: &fakeGitClient{
			output: "worktree C:/repo\nHEAD abc\nbranch refs/heads/main\n\n",
		},
		Selector: &fakeSelector{err: errors.New("selection canceled")},
		Enter:    &fakeEnterRunner{},
	})
	cmd.SetArgs([]string{"enter"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected Execute() to return error")
	}
	if !strings.Contains(err.Error(), "selection canceled") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnterCommand_ReturnsErrorWhenSelectedPathIsMissing(t *testing.T) {
	t.Parallel()

	cmd := NewRootCmd(Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Git: &fakeGitClient{
			output: "worktree C:/repo-missing\nHEAD abc\nbranch refs/heads/main\n\n",
		},
		Selector: &fakeSelector{index: 0},
		Enter:    &fakeEnterRunner{},
	})
	cmd.SetArgs([]string{"enter"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected Execute() to return error")
	}
	if !strings.Contains(err.Error(), "does not exist locally") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnterCommand_BranchModeReturnsErrorWhenSelectedPathIsMissing(t *testing.T) {
	t.Parallel()

	cmd := NewRootCmd(Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Git: &fakeGitClient{
			output: "worktree C:/repo-missing\nHEAD abc\nbranch refs/heads/main\n\n",
		},
		Selector: &fakeSelector{index: 0},
		Enter:    &fakeEnterRunner{},
	})
	cmd.SetArgs([]string{"enter", "--branch", "main"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected Execute() to return error")
	}
	if !strings.Contains(err.Error(), "does not exist locally") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func toPosixPath(path string) string {
	return strings.ReplaceAll(path, "\\", "/")
}
