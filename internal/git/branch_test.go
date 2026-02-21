package git

import (
	"context"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

func TestExecClientRepoRoot_RunsExpectedCommand(t *testing.T) {
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
			"HELPER_STDOUT=C:/repo/project\n",
		)
		return cmd
	})

	root, err := client.RepoRoot(context.Background())
	if err != nil {
		t.Fatalf("RepoRoot() returned error: %v", err)
	}
	if gotName != "git" {
		t.Fatalf("expected command name git, got %q", gotName)
	}
	expectedArgs := []string{"rev-parse", "--show-toplevel"}
	if !reflect.DeepEqual(gotArgs, expectedArgs) {
		t.Fatalf("unexpected args: want=%v got=%v", expectedArgs, gotArgs)
	}
	if root != "C:/repo/project" {
		t.Fatalf("unexpected repo root: %q", root)
	}
}

func TestExecClientFetchPrune_RunsExpectedCommand(t *testing.T) {
	t.Parallel()

	var gotArgs []string
	client := newExecClient(func(ctx context.Context, _ string, args ...string) *exec.Cmd {
		gotArgs = append([]string(nil), args...)

		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestHelperProcess", "--")
		cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
		return cmd
	})

	if err := client.FetchPrune(context.Background(), "origin"); err != nil {
		t.Fatalf("FetchPrune() returned error: %v", err)
	}
	expectedArgs := []string{"fetch", "origin", "--prune"}
	if !reflect.DeepEqual(gotArgs, expectedArgs) {
		t.Fatalf("unexpected args: want=%v got=%v", expectedArgs, gotArgs)
	}
}

func TestExecClientLocalBranches_ParsesOutput(t *testing.T) {
	t.Parallel()

	client := newExecClient(func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestHelperProcess", "--")
		cmd.Env = append(
			os.Environ(),
			"GO_WANT_HELPER_PROCESS=1",
			"HELPER_STDOUT=main\nfeature/a\n\nfeature/a\n",
		)
		return cmd
	})

	branches, err := client.LocalBranches(context.Background())
	if err != nil {
		t.Fatalf("LocalBranches() returned error: %v", err)
	}
	expected := []string{"main", "feature/a"}
	if !reflect.DeepEqual(branches, expected) {
		t.Fatalf("unexpected branches: want=%v got=%v", expected, branches)
	}
}

func TestExecClientRemoteBranches_FiltersByRemote(t *testing.T) {
	t.Parallel()

	client := newExecClient(func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestHelperProcess", "--")
		cmd.Env = append(
			os.Environ(),
			"GO_WANT_HELPER_PROCESS=1",
			"HELPER_STDOUT=origin/main\norigin/feature/x\norigin/HEAD\norigin/HEAD -> origin/main\nupstream/main\n",
		)
		return cmd
	})

	branches, err := client.RemoteBranches(context.Background(), "origin")
	if err != nil {
		t.Fatalf("RemoteBranches() returned error: %v", err)
	}
	expected := []string{"origin/main", "origin/feature/x"}
	if !reflect.DeepEqual(branches, expected) {
		t.Fatalf("unexpected branches: want=%v got=%v", expected, branches)
	}
}

func TestExecClientCheckBranchName_RunsExpectedCommand(t *testing.T) {
	t.Parallel()

	var gotArgs []string
	client := newExecClient(func(ctx context.Context, _ string, args ...string) *exec.Cmd {
		gotArgs = append([]string(nil), args...)

		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestHelperProcess", "--")
		cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
		return cmd
	})

	if err := client.CheckBranchName(context.Background(), "refs/heads/feature/x"); err != nil {
		t.Fatalf("CheckBranchName() returned error: %v", err)
	}

	expectedArgs := []string{"check-ref-format", "--branch", "feature/x"}
	if !reflect.DeepEqual(gotArgs, expectedArgs) {
		t.Fatalf("unexpected args: want=%v got=%v", expectedArgs, gotArgs)
	}
}

func TestExecClientCheckBranchName_ValidatesEmptyInput(t *testing.T) {
	t.Parallel()

	client := newExecClient(func(context.Context, string, ...string) *exec.Cmd {
		t.Fatal("exec command should not be called on invalid input")
		return nil
	})

	err := client.CheckBranchName(context.Background(), "   ")
	if err == nil {
		t.Fatal("expected CheckBranchName() to return error")
	}
	if !strings.Contains(err.Error(), "branch name is empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecClientWorktreeAdd_RunsExpectedArgsWithoutStartPoint(t *testing.T) {
	t.Parallel()

	var gotArgs []string
	client := newExecClient(func(ctx context.Context, _ string, args ...string) *exec.Cmd {
		gotArgs = append([]string(nil), args...)

		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestHelperProcess", "--")
		cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
		return cmd
	})

	err := client.WorktreeAdd(context.Background(), WorktreeAddParams{
		Path:   "/tmp/worktrees/feature/x",
		Branch: "refs/heads/feature/x",
	})
	if err != nil {
		t.Fatalf("WorktreeAdd() returned error: %v", err)
	}

	expectedArgs := []string{"worktree", "add", "/tmp/worktrees/feature/x", "feature/x"}
	if !reflect.DeepEqual(gotArgs, expectedArgs) {
		t.Fatalf("unexpected args: want=%v got=%v", expectedArgs, gotArgs)
	}
}

func TestExecClientWorktreeAdd_RunsExpectedArgsWithStartPoint(t *testing.T) {
	t.Parallel()

	var gotArgs []string
	client := newExecClient(func(ctx context.Context, _ string, args ...string) *exec.Cmd {
		gotArgs = append([]string(nil), args...)

		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestHelperProcess", "--")
		cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
		return cmd
	})

	err := client.WorktreeAdd(context.Background(), WorktreeAddParams{
		Path:       "/tmp/worktrees/feature/y",
		Branch:     "feature/y",
		StartPoint: "origin/feature/y",
	})
	if err != nil {
		t.Fatalf("WorktreeAdd() returned error: %v", err)
	}

	expectedArgs := []string{"worktree", "add", "-b", "feature/y", "/tmp/worktrees/feature/y", "origin/feature/y"}
	if !reflect.DeepEqual(gotArgs, expectedArgs) {
		t.Fatalf("unexpected args: want=%v got=%v", expectedArgs, gotArgs)
	}
}

func TestExecClientWorktreeAdd_ValidatesInputs(t *testing.T) {
	t.Parallel()

	client := newExecClient(func(context.Context, string, ...string) *exec.Cmd {
		t.Fatal("exec command should not be called on invalid input")
		return nil
	})

	err := client.WorktreeAdd(context.Background(), WorktreeAddParams{
		Path:   "",
		Branch: "feature/x",
	})
	if err == nil {
		t.Fatal("expected WorktreeAdd() to return error")
	}
	if !strings.Contains(err.Error(), "worktree path is empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}
