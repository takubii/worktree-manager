package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestVerboseFlag_ListWritesTraceToStderr(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	gitClient := &fakeGitClient{
		output: "worktree C:/repo\nHEAD abcdef\nbranch refs/heads/main\n\n",
	}

	cmd := NewRootCmd(Dependencies{
		Stdout: &stdout,
		Stderr: &stderr,
		Git:    gitClient,
	})
	cmd.SetArgs([]string{"--verbose", "list", "--format", "raw"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if !strings.Contains(stderr.String(), "[trace] list: running `git worktree list --porcelain`") {
		t.Fatalf("expected list trace output, got: %s", stderr.String())
	}
}

func TestVerboseFlag_OpenKeepsOutputPathContract(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	repoPath := toPosixPathForOpen(t.TempDir())
	gitClient := &fakeGitClient{
		output: "worktree " + repoPath + "\nHEAD abc\nbranch refs/heads/main\n\n",
	}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &stdout,
		Stderr:   &stderr,
		Git:      gitClient,
		Selector: &fakeSelector{index: 0},
		Opener:   &fakeOpener{},
	})
	cmd.SetArgs([]string{"--verbose", "open", "--output", "path"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if got := strings.TrimSpace(stdout.String()); got != repoPath {
		t.Fatalf("unexpected path output: %q", got)
	}
	if !strings.Contains(stderr.String(), "[trace] open: invoking opener") {
		t.Fatalf("expected open trace output, got: %s", stderr.String())
	}
}

func TestVerboseFlag_DefaultDoesNotWriteTrace(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	gitClient := &fakeGitClient{
		output: "worktree C:/repo\nHEAD abcdef\nbranch refs/heads/main\n\n",
	}

	cmd := NewRootCmd(Dependencies{
		Stdout: &stdout,
		Stderr: &stderr,
		Git:    gitClient,
	})
	cmd.SetArgs([]string{"list", "--format", "raw"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if strings.Contains(stderr.String(), "[trace]") {
		t.Fatalf("unexpected trace output when --verbose is not set: %s", stderr.String())
	}
}
