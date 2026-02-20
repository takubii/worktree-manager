package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	openerpkg "github.com/takubii/git-worktree-opener/internal/opener"
)

type fakeSelector struct {
	index int
	err   error
	calls int
}

func (s *fakeSelector) Select(_ context.Context, _ string, _ []string) (int, error) {
	s.calls++
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

func TestOpenCommand_OpensSelectedWorktree(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	gitClient := &fakeGitClient{
		output: "worktree C:/repo\nHEAD abc\nbranch refs/heads/main\n\nworktree C:/repo-feature\nHEAD def\nbranch refs/heads/feature/x\n\n",
	}
	selector := &fakeSelector{index: 1}
	openExec := &fakeOpener{}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &stdout,
		Stderr:   &stderr,
		Git:      gitClient,
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
	if openExec.call != 1 {
		t.Fatalf("expected opener to be called once, got %d", openExec.call)
	}
	if openExec.kind != "vscode" {
		t.Fatalf("unexpected opener kind: %q", openExec.kind)
	}
	if openExec.path != "C:/repo-feature" {
		t.Fatalf("unexpected opener path: %q", openExec.path)
	}
	if openExec.window != openerpkg.WindowNew {
		t.Fatalf("unexpected window mode: %q", openExec.window)
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

func TestOpenCommand_ReturnsSelectorError(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	gitClient := &fakeGitClient{
		output: "worktree C:/repo\nHEAD abc\nbranch refs/heads/main\n\nworktree C:/repo-2\nHEAD def\nbranch refs/heads/feature/x\n\n",
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
	cmd.SetArgs([]string{"open", "--window", "reuse"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if openExec.window != openerpkg.WindowReuse {
		t.Fatalf("unexpected window mode: %q", openExec.window)
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
