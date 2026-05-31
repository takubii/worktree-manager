package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/takubii/worktree-manager/internal/config"
	"github.com/takubii/worktree-manager/internal/git"
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
	onWorktreeAdd     func(git.WorktreeAddParams) error
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
	if f.worktreeAddErr != nil {
		return f.worktreeAddErr
	}
	if f.onWorktreeAdd != nil {
		if err := f.onWorktreeAdd(params); err != nil {
			return err
		}
	}
	return nil
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

func TestListCommand_DefaultFormatRendersTable(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() returned error: %v", err)
	}
	otherPath := filepath.Join(t.TempDir(), "worktree-a")
	if err := os.MkdirAll(otherPath, 0o755); err != nil {
		t.Fatalf("MkdirAll() returned error: %v", err)
	}

	raw := "worktree " + strings.ReplaceAll(otherPath, "\\", "/") + "\n" +
		"HEAD bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\n" +
		"branch refs/heads/aaa\n\n" +
		"worktree " + strings.ReplaceAll(cwd, "\\", "/") + "\n" +
		"HEAD aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n" +
		"branch refs/heads/main\n\n"

	gitClient := &fakeGitClient{
		output: raw,
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
	lines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected header + rows, got:\n%s", stdout.String())
	}
	if !strings.HasPrefix(lines[0], "   BRANCH") {
		t.Fatalf("unexpected table header: %s", lines[0])
	}
	if !strings.HasPrefix(lines[1], "*  ") || !strings.Contains(lines[1], "main") {
		t.Fatalf("expected current worktree to be first row, got: %s", lines[1])
	}
	if !strings.Contains(lines[2], "aaa") {
		t.Fatalf("expected secondary row branch `aaa`, got: %s", lines[2])
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr output: %s", stderr.String())
	}
}

func TestListCommand_TableFormatWarnsAboutUnavailableWorktrees(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	activePath := t.TempDir()
	missingPath := filepath.Join(t.TempDir(), "missing")
	stalePath := filepath.Join(t.TempDir(), "stale")
	gitClient := &fakeGitClient{
		output: "worktree " + strings.ReplaceAll(activePath, "\\", "/") + "\n" +
			"HEAD 1111111111111111111111111111111111111111\n" +
			"branch refs/heads/main\n\n" +
			"worktree " + strings.ReplaceAll(missingPath, "\\", "/") + "\n" +
			"HEAD 2222222222222222222222222222222222222222\n" +
			"branch refs/heads/feature/missing\n\n" +
			"worktree " + strings.ReplaceAll(stalePath, "\\", "/") + "\n" +
			"HEAD 3333333333333333333333333333333333333333\n" +
			"branch refs/heads/feature/stale\n" +
			"prunable gitdir file points to non-existent location\n\n",
	}

	cmd := NewRootCmd(Dependencies{
		Stdout: &stdout,
		Stderr: &stderr,
		Git:    gitClient,
	})
	cmd.SetArgs([]string{"list", "--format", "table"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), config.ListStatusMissing) {
		t.Fatalf("expected missing status in table output, got: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), config.ListStatusStale) {
		t.Fatalf("expected stale status in table output, got: %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "found 1 stale worktree") {
		t.Fatalf("expected stale warning, got: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "found 1 missing worktree") {
		t.Fatalf("expected missing warning, got: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "wtm remove <branch>") {
		t.Fatalf("expected actionable remove guidance, got: %s", stderr.String())
	}
}

func TestListCommand_RawFormatWritesGitOutput(t *testing.T) {
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
	cmd.SetArgs([]string{"list", "--format", "raw"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if stdout.String() != gitClient.output {
		t.Fatalf("unexpected stdout:\nwant:\n%s\ngot:\n%s", gitClient.output, stdout.String())
	}
}

func TestListCommand_RawFormatDoesNotWarnAboutUnavailableWorktrees(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	gitClient := &fakeGitClient{
		output: "worktree C:/worktrees/stale\nHEAD abcdef\nbranch refs/heads/feature/stale\nprunable gitdir file points to non-existent location\n\n",
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

	if stdout.String() != gitClient.output {
		t.Fatalf("unexpected stdout:\nwant:\n%s\ngot:\n%s", gitClient.output, stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("raw format should not write guidance to stderr, got: %s", stderr.String())
	}
}

func TestListCommand_JSONFormatWritesRows(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	activePath := t.TempDir()
	missingPath := filepath.Join(t.TempDir(), "missing")
	stalePath := filepath.Join(t.TempDir(), "stale")
	gitClient := &fakeGitClient{
		output: "worktree " + strings.ReplaceAll(activePath, "\\", "/") + "\n" +
			"HEAD 1111111111111111111111111111111111111111\n" +
			"branch refs/heads/main\n\n" +
			"worktree " + strings.ReplaceAll(missingPath, "\\", "/") + "\n" +
			"HEAD 2222222222222222222222222222222222222222\n" +
			"branch refs/heads/feature/missing\n\n" +
			"worktree " + strings.ReplaceAll(stalePath, "\\", "/") + "\n" +
			"HEAD 3333333333333333333333333333333333333333\n" +
			"branch refs/heads/feature/stale\n" +
			"prunable gitdir file points to non-existent location\n\n",
	}

	cmd := NewRootCmd(Dependencies{
		Stdout: &stdout,
		Stderr: &stderr,
		Git:    gitClient,
	})
	cmd.SetArgs([]string{"list", "--format", "json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	var rows []listRow
	if err := json.Unmarshal(stdout.Bytes(), &rows); err != nil {
		t.Fatalf("json.Unmarshal() returned error: %v\noutput:\n%s", err, stdout.String())
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}

	rowByPath := make(map[string]listRow, len(rows))
	for _, row := range rows {
		rowByPath[row.Path] = row
	}

	activeRow, ok := rowByPath[strings.ReplaceAll(activePath, "\\", "/")]
	if !ok {
		t.Fatalf("active row not found in output: %v", rowByPath)
	}
	if activeRow.Status != config.ListStatusActive {
		t.Fatalf("unexpected active status: %q", activeRow.Status)
	}
	if activeRow.Head != "1111111111111111111111111111111111111111" {
		t.Fatalf("unexpected active head: %q", activeRow.Head)
	}
	if activeRow.Branch != "main" {
		t.Fatalf("unexpected active branch: %q", activeRow.Branch)
	}

	missingRow, ok := rowByPath[strings.ReplaceAll(missingPath, "\\", "/")]
	if !ok {
		t.Fatalf("missing row not found in output: %v", rowByPath)
	}
	if missingRow.Status != config.ListStatusMissing {
		t.Fatalf("unexpected missing status: %q", missingRow.Status)
	}

	staleRow, ok := rowByPath[strings.ReplaceAll(stalePath, "\\", "/")]
	if !ok {
		t.Fatalf("stale row not found in output: %v", rowByPath)
	}
	if staleRow.Status != config.ListStatusStale {
		t.Fatalf("unexpected stale status: %q", staleRow.Status)
	}
	if !staleRow.Prunable {
		t.Fatalf("expected stale row prunable=true")
	}
	if !strings.Contains(stderr.String(), "found 1 stale worktree") {
		t.Fatalf("expected stale warning, got: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "found 1 missing worktree") {
		t.Fatalf("expected missing warning, got: %s", stderr.String())
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

func TestListCommand_ReturnsErrorForInvalidFormat(t *testing.T) {
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
	cmd.SetArgs([]string{"list", "--format", "invalid"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected Execute() to return error")
	}
	if !strings.Contains(err.Error(), "invalid --format value") {
		t.Fatalf("unexpected error message: %v", err)
	}
}
