package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/takubii/git-worktree-opener/internal/config"
	"github.com/takubii/git-worktree-opener/internal/git"
)

type fakeGitClient struct {
	output       string
	outputByCall []string
	err          error
	calls        int

	fetchRemote       string
	fetchErr          error
	repoRoot          string
	repoRootErr       error
	localBranches     []string
	localBranchesErr  error
	remoteName        string
	remoteBranches    []string
	remoteBranchesErr error
	checkBranchName   []string
	checkBranchErr    error
	worktreeAddCalls  []git.WorktreeAddParams
	worktreeAddErr    error
	worktreePruneErr  error
	worktreePruneCall int
	worktreeRemove    []fakeWorktreeRemoveCall
	worktreeRemoveErr error
	deleteBranchCalls []fakeDeleteBranchCall
	deleteBranchErr   error
	callLog           []string
}

type fakeWorktreeRemoveCall struct {
	path  string
	force bool
}

type fakeDeleteBranchCall struct {
	branch string
	force  bool
}

type fakeConfigProvider struct {
	cfg       config.Config
	loadCalls int
	initForce []bool
	initPath  string
	initErr   error
}

func (f *fakeConfigProvider) Load(_ context.Context) config.Config {
	f.loadCalls++
	return f.cfg
}

func (f *fakeConfigProvider) InitGlobal(force bool) (string, error) {
	f.initForce = append(f.initForce, force)
	return f.initPath, f.initErr
}

func (f *fakeGitClient) WorktreeListPorcelain(_ context.Context) (string, error) {
	f.calls++
	f.callLog = append(f.callLog, "WorktreeListPorcelain")
	if len(f.outputByCall) > 0 {
		index := f.calls - 1
		if index >= 0 && index < len(f.outputByCall) {
			return f.outputByCall[index], f.err
		}
		return f.outputByCall[len(f.outputByCall)-1], f.err
	}
	return f.output, f.err
}

func (f *fakeGitClient) RepoRoot(_ context.Context) (string, error) {
	f.callLog = append(f.callLog, "RepoRoot")
	return f.repoRoot, f.repoRootErr
}

func (f *fakeGitClient) FetchPrune(_ context.Context, remote string) error {
	f.fetchRemote = remote
	f.callLog = append(f.callLog, "FetchPrune")
	return f.fetchErr
}

func (f *fakeGitClient) LocalBranches(_ context.Context) ([]string, error) {
	f.callLog = append(f.callLog, "LocalBranches")
	return append([]string(nil), f.localBranches...), f.localBranchesErr
}

func (f *fakeGitClient) RemoteBranches(_ context.Context, remote string) ([]string, error) {
	f.remoteName = remote
	f.callLog = append(f.callLog, "RemoteBranches")
	return append([]string(nil), f.remoteBranches...), f.remoteBranchesErr
}

func (f *fakeGitClient) WorktreeAdd(_ context.Context, params git.WorktreeAddParams) error {
	f.worktreeAddCalls = append(f.worktreeAddCalls, params)
	f.callLog = append(f.callLog, "WorktreeAdd")
	return f.worktreeAddErr
}

func (f *fakeGitClient) WorktreePrune(_ context.Context) error {
	f.worktreePruneCall++
	f.callLog = append(f.callLog, "WorktreePrune")
	return f.worktreePruneErr
}

func (f *fakeGitClient) CheckBranchName(_ context.Context, branch string) error {
	f.checkBranchName = append(f.checkBranchName, branch)
	f.callLog = append(f.callLog, "CheckBranchName")
	return f.checkBranchErr
}

func (f *fakeGitClient) WorktreeRemove(_ context.Context, path string, force bool) error {
	f.worktreeRemove = append(f.worktreeRemove, fakeWorktreeRemoveCall{
		path:  path,
		force: force,
	})
	f.callLog = append(f.callLog, "WorktreeRemove")
	return f.worktreeRemoveErr
}

func (f *fakeGitClient) DeleteLocalBranch(_ context.Context, branch string, force bool) error {
	f.deleteBranchCalls = append(f.deleteBranchCalls, fakeDeleteBranchCall{
		branch: branch,
		force:  force,
	})
	f.callLog = append(f.callLog, "DeleteLocalBranch")
	return f.deleteBranchErr
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
