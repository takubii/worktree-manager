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
		LookPath: newTestLookPath(map[string]bool{"code": true}),
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

func TestNewCommand_SkipsPruneWhenNoPruneIsSet(t *testing.T) {
	t.Parallel()

	repoRoot := createTestRepoRoot(t)
	gitClient := &fakeGitClient{
		repoRoot:       repoRoot,
		localBranches:  []string{"main", "feature/local"},
		remoteBranches: []string{"origin/main"},
	}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: &fakeSelector{index: 0},
		Opener:   &fakeOpener{},
	})
	cmd.SetArgs([]string{"new", "feature/local", "--no-prune"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if gitClient.worktreePruneCall != 0 {
		t.Fatalf("expected WorktreePrune not to be called, got %d", gitClient.worktreePruneCall)
	}
	if gitClient.fetchRemote != "origin" {
		t.Fatalf("unexpected fetch remote: %q", gitClient.fetchRemote)
	}
}

func TestNewCommand_SkipsFetchWhenNoFetchIsSet(t *testing.T) {
	t.Parallel()

	repoRoot := createTestRepoRoot(t)
	gitClient := &fakeGitClient{
		repoRoot:       repoRoot,
		localBranches:  []string{"main", "feature/local"},
		remoteBranches: []string{"origin/main"},
	}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: &fakeSelector{index: 0},
		Opener:   &fakeOpener{},
	})
	cmd.SetArgs([]string{"new", "feature/local", "--no-fetch"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if gitClient.fetchRemote != "" {
		t.Fatalf("expected FetchPrune not to be called, got remote %q", gitClient.fetchRemote)
	}
	if gitClient.worktreePruneCall != 1 {
		t.Fatalf("expected WorktreePrune to be called once, got %d", gitClient.worktreePruneCall)
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
	if openExec.call != 0 {
		t.Fatalf("opener should not be called by default, got %d", openExec.call)
	}
}

func TestNewCommand_ReturnsErrorWhenExplicitVSCodeIsUnavailable(t *testing.T) {
	t.Parallel()

	repoRoot := createTestRepoRoot(t)
	gitClient := &fakeGitClient{
		repoRoot:       repoRoot,
		localBranches:  []string{"main", "feature/local"},
		remoteBranches: []string{"origin/main"},
	}
	openExec := &fakeOpener{}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		LookPath: newTestLookPath(map[string]bool{}),
		Selector: &fakeSelector{index: 0},
		Opener:   openExec,
	})
	cmd.SetArgs([]string{"new", "feature/local", "--open", "vscode"})

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

func TestNewCommand_PreservesLocalBranchWithRemotePrefix(t *testing.T) {
	t.Parallel()

	repoRoot := createTestRepoRoot(t)
	gitClient := &fakeGitClient{
		repoRoot:       repoRoot,
		localBranches:  []string{"main", "origin/feature-demo"},
		remoteBranches: []string{"origin/main"},
	}
	openExec := &fakeOpener{}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: &fakeSelector{index: 0},
		Opener:   openExec,
	})
	cmd.SetArgs([]string{"new", "origin/feature-demo"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if len(gitClient.worktreeAddCalls) != 1 {
		t.Fatalf("expected one WorktreeAdd call, got %d", len(gitClient.worktreeAddCalls))
	}
	if got := gitClient.worktreeAddCalls[0].Branch; got != "origin/feature-demo" {
		t.Fatalf("unexpected branch: %q", got)
	}
	if got := gitClient.worktreeAddCalls[0].StartPoint; got != "" {
		t.Fatalf("unexpected start point: %q", got)
	}
}

func TestNewCommand_ResolvesRemoteBranchWithRemotePrefixedName(t *testing.T) {
	t.Parallel()

	repoRoot := createTestRepoRoot(t)
	gitClient := &fakeGitClient{
		repoRoot:       repoRoot,
		localBranches:  []string{"main"},
		remoteBranches: []string{"origin/main", "origin/origin/feature-demo"},
	}
	openExec := &fakeOpener{}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: &fakeSelector{index: 0},
		Opener:   openExec,
	})
	cmd.SetArgs([]string{"new", "origin/feature-demo"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if len(gitClient.worktreeAddCalls) != 1 {
		t.Fatalf("expected one WorktreeAdd call, got %d", len(gitClient.worktreeAddCalls))
	}
	if got := gitClient.worktreeAddCalls[0].Branch; got != "origin/feature-demo" {
		t.Fatalf("unexpected branch: %q", got)
	}
	if got := gitClient.worktreeAddCalls[0].StartPoint; got != "origin/origin/feature-demo" {
		t.Fatalf("unexpected start point: %q", got)
	}
}

func TestNewCommand_SelectsBranchWhenArgumentIsMissing(t *testing.T) {
	t.Parallel()

	repoRoot := createTestRepoRoot(t)
	linkedPath := strings.ReplaceAll(filepath.Join(filepath.Dir(repoRoot), "worktrees", "feature", "remote"), "\\", "/")
	gitClient := &fakeGitClient{
		repoRoot:       repoRoot,
		localBranches:  []string{"main"},
		remoteBranches: []string{"origin/main", "origin/feature/remote"},
		output: "worktree " + linkedPath + "\n" +
			"HEAD abcdefabcdefabcdefabcdefabcdefabcdefabcd\n" +
			"branch refs/heads/feature/remote\n\n",
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
	if len(selector.lastOptions) != 2 {
		t.Fatalf("unexpected candidate count: %d", len(selector.lastOptions))
	}
	if selector.lastOptions[1] != "feature/remote [worktree]" {
		t.Fatalf("expected decorated linked branch candidate, got %q", selector.lastOptions[1])
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
	prompt string
	items  []string
}

func (s *fakeSelectOrCreateSelector) Select(_ context.Context, _ string, _ []string) (int, error) {
	return -1, errors.New("Select should not be called")
}

func (s *fakeSelectOrCreateSelector) SelectOrCreate(_ context.Context, prompt string, items []string) (selectorpkg.SelectOrCreateResult, error) {
	s.calls++
	s.prompt = prompt
	s.items = append([]string(nil), items...)
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

func TestNewCommand_ResolvesDecoratedExistingSelectionFromSelectOrCreate(t *testing.T) {
	t.Parallel()

	repoRoot := createTestRepoRoot(t)
	linkedPath := strings.ReplaceAll(filepath.Join(filepath.Dir(repoRoot), "worktrees", "feature", "existing"), "\\", "/")
	gitClient := &fakeGitClient{
		repoRoot:       repoRoot,
		localBranches:  []string{"main", "feature/existing"},
		remoteBranches: []string{"origin/main"},
		output: "worktree " + linkedPath + "\n" +
			"HEAD abcdefabcdefabcdefabcdefabcdefabcdefabcd\n" +
			"branch refs/heads/feature/existing\n\n",
	}
	selectorWithCreate := &fakeSelectOrCreateSelector{
		result: selectorpkg.SelectOrCreateResult{
			Value: "feature/existing [worktree]",
			IsNew: false,
		},
	}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: selectorWithCreate,
		Opener:   &fakeOpener{},
	})
	cmd.SetArgs([]string{"new"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if selectorWithCreate.calls != 1 {
		t.Fatalf("expected SelectOrCreate to be called once, got %d", selectorWithCreate.calls)
	}
	if len(selectorWithCreate.items) == 0 || !strings.Contains(strings.Join(selectorWithCreate.items, "\n"), "[worktree]") {
		t.Fatalf("expected decorated branch candidates, got %v", selectorWithCreate.items)
	}
	if len(gitClient.worktreeAddCalls) != 1 {
		t.Fatalf("expected one WorktreeAdd call, got %d", len(gitClient.worktreeAddCalls))
	}
	if got := gitClient.worktreeAddCalls[0].Branch; got != "feature/existing" {
		t.Fatalf("unexpected branch: %q", got)
	}
	if len(gitClient.checkBranchName) != 0 {
		t.Fatalf("CheckBranchName should not run for existing decorated selection, got: %v", gitClient.checkBranchName)
	}
}

func TestNewCommand_DoesNotOpenByDefaultWhenFlagsAreNotProvided(t *testing.T) {
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
			New: config.New{
				Fetch: true,
				Prune: true,
			},
			Open: config.Open{
				Default: "cursor",
				Window:  "reuse",
				Prune:   true,
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
	if openExec.call != 0 {
		t.Fatalf("opener should not be called by default, got %d", openExec.call)
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
			New: config.New{
				Fetch: true,
				Prune: true,
			},
			Open: config.Open{
				Default: "cursor",
				Window:  "reuse",
				Prune:   true,
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
		LookPath: newTestLookPath(map[string]bool{"code": true}),
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

func TestNewCommand_OutputPathPrintsCreatedPath(t *testing.T) {
	t.Parallel()

	repoRoot := createTestRepoRoot(t)
	expectedPath := filepath.Join(filepath.Dir(repoRoot), "worktrees", filepath.FromSlash("feature/output-path"))
	var stdout bytes.Buffer
	gitClient := &fakeGitClient{
		repoRoot:       repoRoot,
		localBranches:  []string{"main"},
		remoteBranches: []string{"origin/main"},
	}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &stdout,
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: &fakeSelector{index: 0},
		Opener:   &fakeOpener{},
	})
	cmd.SetArgs([]string{"new", "feature/output-path", "--output", "path"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if got := strings.TrimSpace(stdout.String()); filepath.Clean(got) != filepath.Clean(expectedPath) {
		t.Fatalf("unexpected path output: want=%q got=%q", expectedPath, got)
	}
}

func TestNewCommand_OutputJSONPrintsPayload(t *testing.T) {
	t.Parallel()

	repoRoot := createTestRepoRoot(t)
	expectedPath := filepath.Join(filepath.Dir(repoRoot), "worktrees", filepath.FromSlash("feature/output-json"))
	var stdout bytes.Buffer
	gitClient := &fakeGitClient{
		repoRoot:       repoRoot,
		localBranches:  []string{"main"},
		remoteBranches: []string{"origin/main"},
	}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &stdout,
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		LookPath: newTestLookPath(map[string]bool{"code": true}),
		Selector: &fakeSelector{index: 0},
		Opener:   &fakeOpener{},
	})
	cmd.SetArgs([]string{"new", "feature/output-json", "--open", "vscode", "--output", "json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	var payload commandOutput
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() returned error: %v\noutput=%s", err, stdout.String())
	}
	if payload.Command != "new" {
		t.Fatalf("unexpected command: %q", payload.Command)
	}
	if filepath.Clean(payload.Path) != filepath.Clean(expectedPath) {
		t.Fatalf("unexpected path: %q", payload.Path)
	}
	if payload.Branch != "feature/output-json" {
		t.Fatalf("unexpected branch: %q", payload.Branch)
	}
	if !payload.Created {
		t.Fatal("expected created=true")
	}
	if !payload.Opened {
		t.Fatal("expected opened=true")
	}
}

func TestNewCommand_UsesConfigToSkipFetchAndPruneByDefault(t *testing.T) {
	t.Parallel()

	repoRoot := createTestRepoRoot(t)
	gitClient := &fakeGitClient{
		repoRoot:       repoRoot,
		localBranches:  []string{"main"},
		remoteBranches: []string{"origin/main"},
	}
	cfgProvider := &fakeConfigProvider{
		cfg: config.Config{
			Remote:              config.DefaultRemote,
			BaseBranch:          config.DefaultBaseBranch,
			WorktreeDirTemplate: config.DefaultWorktreeDirTemplate,
			New: config.New{
				Fetch: false,
				Prune: false,
			},
			Open: config.Open{
				Default: config.DefaultOpenKind,
				Window:  config.DefaultOpenWindow,
				Prune:   true,
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
		Opener:   &fakeOpener{},
		Config:   cfgProvider,
	})
	cmd.SetArgs([]string{"new", "feature/no-network"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if gitClient.fetchRemote != "" {
		t.Fatalf("expected FetchPrune to be skipped, got remote %q", gitClient.fetchRemote)
	}
	if gitClient.worktreePruneCall != 0 {
		t.Fatalf("expected WorktreePrune to be skipped, got %d", gitClient.worktreePruneCall)
	}
}

func TestNewCommand_FlagsOverrideConfigSkipSettings(t *testing.T) {
	t.Parallel()

	repoRoot := createTestRepoRoot(t)
	gitClient := &fakeGitClient{
		repoRoot:       repoRoot,
		localBranches:  []string{"main"},
		remoteBranches: []string{"origin/main"},
	}
	cfgProvider := &fakeConfigProvider{
		cfg: config.Config{
			Remote:              config.DefaultRemote,
			BaseBranch:          config.DefaultBaseBranch,
			WorktreeDirTemplate: config.DefaultWorktreeDirTemplate,
			New: config.New{
				Fetch: false,
				Prune: false,
			},
			Open: config.Open{
				Default: config.DefaultOpenKind,
				Window:  config.DefaultOpenWindow,
				Prune:   true,
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
		Opener:   &fakeOpener{},
		Config:   cfgProvider,
	})
	cmd.SetArgs([]string{"new", "feature/force-network", "--no-fetch=false", "--no-prune=false"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if gitClient.fetchRemote != config.DefaultRemote {
		t.Fatalf("expected FetchPrune to run with %q, got %q", config.DefaultRemote, gitClient.fetchRemote)
	}
	if gitClient.worktreePruneCall != 1 {
		t.Fatalf("expected WorktreePrune to run once, got %d", gitClient.worktreePruneCall)
	}
}

func TestNewCommand_PassesTerminalProviderToOpener(t *testing.T) {
	t.Parallel()

	repoRoot := createTestRepoRoot(t)
	gitClient := &fakeGitClient{
		repoRoot:       repoRoot,
		localBranches:  []string{"main", "feature/local"},
		remoteBranches: []string{"origin/main"},
	}
	openExec := &fakeOpener{}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: &fakeSelector{index: 0},
		Opener:   openExec,
	})
	cmd.SetArgs([]string{"new", "feature/local", "--open", "terminal", "--terminal-provider", "warp"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if openExec.kind != "terminal" {
		t.Fatalf("unexpected opener kind: %q", openExec.kind)
	}
	if openExec.terminalProvider != "warp" {
		t.Fatalf("unexpected terminal provider: %q", openExec.terminalProvider)
	}
}

func TestNewCommand_ReturnsErrorWhenTerminalProviderUsedWithoutTerminalOpen(t *testing.T) {
	t.Parallel()

	repoRoot := createTestRepoRoot(t)
	gitClient := &fakeGitClient{
		repoRoot:       repoRoot,
		localBranches:  []string{"main", "feature/local"},
		remoteBranches: []string{"origin/main"},
	}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: &fakeSelector{index: 0},
		Opener:   &fakeOpener{},
	})
	cmd.SetArgs([]string{"new", "feature/local", "--open", "none", "--terminal-provider", "warp"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected Execute() to return error")
	}
	if !strings.Contains(err.Error(), "`--terminal-provider` can only be used with `--open terminal`") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewCommand_PrintsOpenerWarningsToStderr(t *testing.T) {
	t.Parallel()

	repoRoot := createTestRepoRoot(t)
	gitClient := &fakeGitClient{
		repoRoot:       repoRoot,
		localBranches:  []string{"main", "feature/local"},
		remoteBranches: []string{"origin/main"},
	}
	openExec := &fakeOpener{
		result: openerpkg.OpenResult{
			Provider: "terminal",
			Warnings: []string{"reuse mode is best-effort"},
		},
	}
	var stderr bytes.Buffer
	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &stderr,
		Git:      gitClient,
		Selector: &fakeSelector{index: 0},
		Opener:   openExec,
	})
	cmd.SetArgs([]string{"new", "feature/local", "--open", "terminal", "--terminal-provider", "warp"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if !strings.Contains(stderr.String(), "warning: reuse mode is best-effort") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
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
