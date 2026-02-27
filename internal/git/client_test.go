package git

import (
	"context"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestExecClientWorktreeListPorcelain_RunsGitWithExpectedArgs(t *testing.T) {
	t.Parallel()

	var gotName string
	var gotArgs []string
	client := newExecClient(func(ctx context.Context, name string, args ...string) *exec.Cmd {
		gotName = name
		gotArgs = append([]string(nil), args...)

		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestHelperProcess", "--")
		cmd.Env = append(
			os.Environ(),
			"GO_WANT_HELPER_PROCESS=1",
			"HELPER_STDOUT=worktree C:/repo\n",
		)
		return cmd
	})

	output, err := client.WorktreeListPorcelain(context.Background())
	if err != nil {
		t.Fatalf("WorktreeListPorcelain() returned error: %v", err)
	}

	if gotName != "git" {
		t.Fatalf("expected command name git, got %q", gotName)
	}
	expectedArgs := []string{"worktree", "list", "--porcelain"}
	if !reflect.DeepEqual(gotArgs, expectedArgs) {
		t.Fatalf("unexpected args: want=%v got=%v", expectedArgs, gotArgs)
	}
	if output != "worktree C:/repo\n" {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestExecClientWorktreeListPorcelain_ReturnsGuidanceForRepositoryError(t *testing.T) {
	t.Parallel()

	client := newExecClient(func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestHelperProcess", "--")
		cmd.Env = append(
			os.Environ(),
			"GO_WANT_HELPER_PROCESS=1",
			"HELPER_STDERR=fatal: not a git repository (or any of the parent directories): .git",
			"HELPER_EXIT_CODE=1",
		)
		return cmd
	})

	_, err := client.WorktreeListPorcelain(context.Background())
	if err == nil {
		t.Fatal("expected WorktreeListPorcelain() to return error")
	}

	errText := err.Error()
	if !strings.Contains(errText, "not a git repository") {
		t.Fatalf("error does not include git stderr output: %s", errText)
	}
	if !strings.Contains(errText, "Run this command inside a Git repository") {
		t.Fatalf("error does not include next action guidance: %s", errText)
	}
}

func TestExecClientWorktreeListPorcelain_ReturnsGuidanceWhenGitMissing(t *testing.T) {
	t.Parallel()

	client := newExecClient(func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "definitely-missing-git-binary-xyz")
	})

	_, err := client.WorktreeListPorcelain(context.Background())
	if err == nil {
		t.Fatal("expected WorktreeListPorcelain() to return error")
	}
	if !strings.Contains(err.Error(), "command was not found in PATH") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "Install Git") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHelperProcess(t *testing.T) {
	t.Helper()

	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	if stdout := os.Getenv("HELPER_STDOUT"); stdout != "" {
		_, _ = io.WriteString(os.Stdout, stdout)
	}
	if stderr := os.Getenv("HELPER_STDERR"); stderr != "" {
		_, _ = io.WriteString(os.Stderr, stderr)
	}

	exitCode := 0
	if raw := os.Getenv("HELPER_EXIT_CODE"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			exitCode = 2
		} else {
			exitCode = parsed
		}
	}
	os.Exit(exitCode)
}
