package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeGitClient struct {
	output string
	err    error
	calls  int
}

func (f *fakeGitClient) WorktreeListPorcelain(_ context.Context) (string, error) {
	f.calls++
	return f.output, f.err
}

func TestListCommand_WritesGitOutput(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	gitClient := &fakeGitClient{
		output: "worktree C:/repo\nHEAD abcdef\nbranch refs/heads/main\n",
	}

	cmd := NewRootCmd(Dependencies{
		Stdout: &stdout,
		Stderr: &stderr,
		Git:    gitClient,
	})
	cmd.SetArgs([]string{"list"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if gitClient.calls != 1 {
		t.Fatalf("expected WorktreeListPorcelain to be called once, got %d", gitClient.calls)
	}
	if stdout.String() != gitClient.output {
		t.Fatalf("unexpected stdout:\nwant:\n%s\ngot:\n%s", gitClient.output, stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr output: %s", stderr.String())
	}
}

func TestListCommand_ReturnsGitError(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	expectedErr := errors.New("failed to run `git worktree list --porcelain`: fatal: not a git repository")
	gitClient := &fakeGitClient{err: expectedErr}

	cmd := NewRootCmd(Dependencies{
		Stdout: &stdout,
		Stderr: &stderr,
		Git:    gitClient,
	})
	cmd.SetArgs([]string{"list"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected Execute() to return error")
	}
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Fatalf("unexpected error message: %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout output, got: %s", stdout.String())
	}
}
