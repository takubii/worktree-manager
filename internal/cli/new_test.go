package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/takubii/git-worktree-opener/internal/config"
	openerpkg "github.com/takubii/git-worktree-opener/internal/opener"
	selectorpkg "github.com/takubii/git-worktree-opener/internal/selector"
)

func TestNewCommand_CreatesWorktreeFromLocalBranch(t *testing.T) {
	t.Parallel()

	repoRoot := createTestRepoRoot(t)
	expectedPath := filepath.Join(filepath.Dir(repoRoot), "worktrees", filepath.FromSlash("feature/local"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	gitClient := &fakeGitClient{
		repoRoot:       repoRoot,
		localBranches:  []string{"main", "feature/local"},
		remoteBranches: []string{"origin/main", "origin/feature/remote"},
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
	cmd.SetArgs([]string{"new", "feature/local", "--open", "vscode"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if gitClient.fetchRemote != "origin" {
		t.Fatalf("unexpected fetch remote: %q", gitClient.fetchRemote)
	}
	if gitClient.worktreePruneCall != 1 {
		t.Fatalf("expected WorktreePrune to be called once, got %d", gitClient.worktreePruneCall)
	}
	if gitClient.remoteName != "origin" {
		t.Fatalf("unexpected remote query name: %q", gitClient.remoteName)
	}
	if selector.calls != 0 {
		t.Fatalf("selector should not be called when branch arg is provided, got %d", selector.calls)
	}
	if len(gitClient.worktreeAddCalls) != 1 {
		t.Fatalf("expected one WorktreeAdd call, got %d", len(gitClient.worktreeAddCalls))
	}

	add := gitClient.worktreeAddCalls[0]
	if add.Path != expectedPath {
		t.Fatalf("unexpected worktree path: want=%q got=%q", expectedPath, add.Path)
	}
	if add.Branch != "feature/local" {
		t.Fatalf("unexpected branch: %q", add.Branch)
	}
	if add.StartPoint != "" {
		t.Fatalf("unexpected start point: %q", add.StartPoint)
	}
	if openExec.call != 1 {
		t.Fatalf("expected opener to be called once, got %d", openExec.call)
	}
	if openExec.kind != "vscode" {
		t.Fatalf("unexpected opener kind: %q", openExec.kind)
	}
	if openExec.path != expectedPath {
		t.Fatalf("unexpected opener path: want=%q got=%q", expectedPath, openExec.path)
	}
	if openExec.window != openerpkg.WindowNew {
		t.Fatalf("unexpected window mode: %q", openExec.window)
	}
}

func TestNewCommand_UsesRemoteBranchAsStartPoint(t *testing.T) {
	t.Parallel()

	repoRoot := createTestRepoRoot(t)
	gitClient := &fakeGitClient{
		repoRoot:       repoRoot,
		localBranches:  []string{"main"},
		remoteBranches: []string{"origin/main", "origin/feature/remote"},
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
	cmd.SetArgs([]string{"new", "feature/remote"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if len(gitClient.worktreeAddCalls) != 1 {
		t.Fatalf("expected one WorktreeAdd call, got %d", len(gitClient.worktreeAddCalls))
	}
	if got := gitClient.worktreeAddCalls[0].StartPoint; got != "origin/feature/remote" {
		t.Fatalf("unexpected start point: %q", got)
	}
}

func TestNewCommand_SelectsBranchWhenArgumentIsMissing(t *testing.T) {
	t.Parallel()

	repoRoot := createTestRepoRoot(t)
	gitClient := &fakeGitClient{
		repoRoot:       repoRoot,
		localBranches:  []string{"main"},
		remoteBranches: []string{"origin/main", "origin/feature/remote"},
	}
	selector := &fakeSelector{index: 1}
	openExec := &fakeOpener{}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: selector,
		Opener:   openExec,
	})
	cmd.SetArgs([]string{"new"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if selector.calls != 1 {
		t.Fatalf("expected selector to be called once, got %d", selector.calls)
	}
	if got := gitClient.worktreeAddCalls[0].Branch; got != "feature/remote" {
		t.Fatalf("unexpected selected branch: %q", got)
	}
	if got := gitClient.worktreeAddCalls[0].StartPoint; got != "origin/feature/remote" {
		t.Fatalf("unexpected selected start point: %q", got)
	}
}

func TestNewCommand_UsesRemoteBaseWhenBaseIsNotLocal(t *testing.T) {
	t.Parallel()

	repoRoot := createTestRepoRoot(t)
	gitClient := &fakeGitClient{
		repoRoot:       repoRoot,
		localBranches:  []string{"feature/seed"},
		remoteBranches: []string{"origin/main", "origin/feature/seed"},
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
	cmd.SetArgs([]string{"new", "feature/new-one", "--base", "main"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if len(gitClient.worktreeAddCalls) != 1 {
		t.Fatalf("expected one WorktreeAdd call, got %d", len(gitClient.worktreeAddCalls))
	}
	if got := gitClient.worktreeAddCalls[0].StartPoint; got != "origin/main" {
		t.Fatalf("unexpected start point: %q", got)
	}
}

func TestNewCommand_ReturnsErrorWhenNoBranchesAreAvailable(t *testing.T) {
	t.Parallel()

	repoRoot := createTestRepoRoot(t)
	gitClient := &fakeGitClient{
		repoRoot:       repoRoot,
		localBranches:  nil,
		remoteBranches: nil,
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
	cmd.SetArgs([]string{"new"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected Execute() to return error")
	}
	if !strings.Contains(err.Error(), "no branches available") {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gitClient.worktreeAddCalls) != 0 {
		t.Fatalf("WorktreeAdd should not be called, got %d", len(gitClient.worktreeAddCalls))
	}
	if openExec.call != 0 {
		t.Fatalf("opener should not be called, got %d", openExec.call)
	}
}

type fakeSelectOrCreateSelector struct {
	result selectorpkg.SelectOrCreateResult
	err    error
	calls  int
}

func (s *fakeSelectOrCreateSelector) Select(_ context.Context, _ string, _ []string) (int, error) {
	return -1, errors.New("Select should not be called")
}

func (s *fakeSelectOrCreateSelector) SelectOrCreate(_ context.Context, _ string, _ []string) (selectorpkg.SelectOrCreateResult, error) {
	s.calls++
	return s.result, s.err
}

func TestNewCommand_ValidatesNewBranchFromInteractiveInput(t *testing.T) {
	t.Parallel()

	repoRoot := createTestRepoRoot(t)
	gitClient := &fakeGitClient{
		repoRoot:       repoRoot,
		localBranches:  []string{"main"},
		remoteBranches: []string{"origin/main"},
	}
	selectorWithCreate := &fakeSelectOrCreateSelector{
		result: selectorpkg.SelectOrCreateResult{
			Value: "feature/typed",
			IsNew: true,
		},
	}
	openExec := &fakeOpener{}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: selectorWithCreate,
		Opener:   openExec,
	})
	cmd.SetArgs([]string{"new"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if selectorWithCreate.calls != 1 {
		t.Fatalf("expected SelectOrCreate to be called once, got %d", selectorWithCreate.calls)
	}
	if len(gitClient.checkBranchName) != 1 || gitClient.checkBranchName[0] != "feature/typed" {
		t.Fatalf("unexpected branch validation calls: %v", gitClient.checkBranchName)
	}
	if len(gitClient.worktreeAddCalls) != 1 {
		t.Fatalf("expected one WorktreeAdd call, got %d", len(gitClient.worktreeAddCalls))
	}
	if got := gitClient.worktreeAddCalls[0].StartPoint; got != "main" {
		t.Fatalf("unexpected start point: %q", got)
	}
}

func TestNewCommand_ReturnsErrorWhenInteractiveBranchValidationFails(t *testing.T) {
	t.Parallel()

	repoRoot := createTestRepoRoot(t)
	gitClient := &fakeGitClient{
		repoRoot:       repoRoot,
		localBranches:  []string{"main"},
		remoteBranches: []string{"origin/main"},
		checkBranchErr: errors.New("invalid branch name"),
		worktreeAddErr: nil,
	}
	selectorWithCreate := &fakeSelectOrCreateSelector{
		result: selectorpkg.SelectOrCreateResult{
			Value: "bad branch",
			IsNew: true,
		},
	}
	openExec := &fakeOpener{}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: selectorWithCreate,
		Opener:   openExec,
	})
	cmd.SetArgs([]string{"new"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected Execute() to return error")
	}
	if !strings.Contains(err.Error(), "invalid branch name") {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gitClient.worktreeAddCalls) != 0 {
		t.Fatalf("WorktreeAdd should not be called, got %d", len(gitClient.worktreeAddCalls))
	}
	if openExec.call != 0 {
		t.Fatalf("opener should not be called, got %d", openExec.call)
	}
}

func TestNewCommand_DoesNotValidateExistingBranchSelection(t *testing.T) {
	t.Parallel()

	repoRoot := createTestRepoRoot(t)
	gitClient := &fakeGitClient{
		repoRoot:       repoRoot,
		localBranches:  []string{"main", "feature/existing"},
		remoteBranches: []string{"origin/main"},
		checkBranchErr: errors.New("should not be called"),
	}
	selectorWithCreate := &fakeSelectOrCreateSelector{
		result: selectorpkg.SelectOrCreateResult{
			Value: "feature/existing",
			IsNew: false,
		},
	}
	openExec := &fakeOpener{}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: selectorWithCreate,
		Opener:   openExec,
	})
	cmd.SetArgs([]string{"new"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if len(gitClient.checkBranchName) != 0 {
		t.Fatalf("CheckBranchName should not be called for existing selection, got: %v", gitClient.checkBranchName)
	}
}

func TestNewCommand_UsesConfigDefaultsWhenFlagsAreNotProvided(t *testing.T) {
	t.Parallel()

	repoRoot := createTestRepoRoot(t)
	expectedPath := filepath.Clean(filepath.Join(repoRoot, "..", "custom-worktrees", "feature", "new-one"))

	gitClient := &fakeGitClient{
		repoRoot:       repoRoot,
		localBranches:  []string{"develop"},
		remoteBranches: []string{"upstream/develop"},
	}
	openExec := &fakeOpener{}
	cfgProvider := &fakeConfigProvider{
		cfg: config.Config{
			Remote:              "upstream",
			BaseBranch:          "develop",
			WorktreeDirTemplate: "{repoRoot}/../custom-worktrees/{branch}",
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
		Selector: &fakeSelector{index: 0},
		Opener:   openExec,
		Config:   cfgProvider,
	})
	cmd.SetArgs([]string{"new", "feature/new-one"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if gitClient.fetchRemote != "upstream" {
		t.Fatalf("unexpected fetch remote: %q", gitClient.fetchRemote)
	}
	if gitClient.remoteName != "upstream" {
		t.Fatalf("unexpected remote query name: %q", gitClient.remoteName)
	}
	if len(gitClient.worktreeAddCalls) != 1 {
		t.Fatalf("expected one WorktreeAdd call, got %d", len(gitClient.worktreeAddCalls))
	}
	if got := gitClient.worktreeAddCalls[0].StartPoint; got != "develop" {
		t.Fatalf("unexpected start point: %q", got)
	}
	if got := filepath.Clean(gitClient.worktreeAddCalls[0].Path); got != expectedPath {
		t.Fatalf("unexpected worktree path: want=%q got=%q", expectedPath, got)
	}
	if openExec.kind != "cursor" {
		t.Fatalf("unexpected opener kind: %q", openExec.kind)
	}
	if openExec.window != openerpkg.WindowReuse {
		t.Fatalf("unexpected window mode: %q", openExec.window)
	}
}

func TestNewCommand_FlagsOverrideConfigDefaults(t *testing.T) {
	t.Parallel()

	repoRoot := createTestRepoRoot(t)
	gitClient := &fakeGitClient{
		repoRoot:       repoRoot,
		localBranches:  []string{"main", "develop"},
		remoteBranches: []string{"upstream/main", "upstream/develop"},
	}
	openExec := &fakeOpener{}
	cfgProvider := &fakeConfigProvider{
		cfg: config.Config{
			Remote:              "upstream",
			BaseBranch:          "develop",
			WorktreeDirTemplate: "{repoParent}/worktrees/{branch}",
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
		Selector: &fakeSelector{index: 0},
		Opener:   openExec,
		Config:   cfgProvider,
	})
	cmd.SetArgs([]string{"new", "feature/new-two", "--base", "main", "--open", "vscode"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if len(gitClient.worktreeAddCalls) != 1 {
		t.Fatalf("expected one WorktreeAdd call, got %d", len(gitClient.worktreeAddCalls))
	}
	if got := gitClient.worktreeAddCalls[0].StartPoint; got != "main" {
		t.Fatalf("unexpected start point: %q", got)
	}
	if openExec.kind != "vscode" {
		t.Fatalf("unexpected opener kind: %q", openExec.kind)
	}
	if openExec.window != openerpkg.WindowReuse {
		t.Fatalf("unexpected window mode: %q", openExec.window)
	}
}

func createTestRepoRoot(t *testing.T) string {
	t.Helper()

	parent := t.TempDir()
	repoRoot := filepath.Join(parent, "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("failed to create repo root: %v", err)
	}
	return repoRoot
}
