package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/takubii/git-worktree-opener/internal/config"
)

func TestRmCommand_RemovesSelectedWorktreeAndDeletesBranchByDefault(t *testing.T) {
	t.Parallel()

	gitClient := &fakeGitClient{
		output: "worktree C:/repo\nHEAD abc\nbranch refs/heads/main\n\nworktree C:/worktrees/feature-x\nHEAD def\nbranch refs/heads/feature/x\n\n",
	}
	selector := &fakeSelector{index: 1}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: selector,
		Opener:   &fakeOpener{},
	})
	cmd.SetArgs([]string{"rm"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if selector.calls != 1 {
		t.Fatalf("expected selector to be called once, got %d", selector.calls)
	}
	if len(gitClient.worktreeRemove) != 1 {
		t.Fatalf("expected one WorktreeRemove call, got %d", len(gitClient.worktreeRemove))
	}
	if got := gitClient.worktreeRemove[0].path; got != "C:/worktrees/feature-x" {
		t.Fatalf("unexpected worktree remove path: %q", got)
	}
	if gitClient.worktreeRemove[0].force {
		t.Fatal("worktree remove should not be forced by default")
	}
	if len(gitClient.deleteBranchCalls) != 1 {
		t.Fatalf("expected one DeleteLocalBranch call, got %d", len(gitClient.deleteBranchCalls))
	}
	if got := gitClient.deleteBranchCalls[0].branch; got != "feature/x" {
		t.Fatalf("unexpected branch deleted: %q", got)
	}
	if gitClient.deleteBranchCalls[0].force {
		t.Fatal("branch deletion should be safe by default")
	}
}

func TestRmCommand_FindsWorktreeByBranchArgument(t *testing.T) {
	t.Parallel()

	gitClient := &fakeGitClient{
		output: "worktree C:/repo\nHEAD abc\nbranch refs/heads/main\n\nworktree C:/worktrees/feature-x\nHEAD def\nbranch refs/heads/feature/x\n\n",
	}
	selector := &fakeSelector{index: 0}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: selector,
		Opener:   &fakeOpener{},
	})
	cmd.SetArgs([]string{"rm", "feature/x"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if selector.calls != 0 {
		t.Fatalf("selector should not be called when branch arg is provided, got %d", selector.calls)
	}
	if len(gitClient.worktreeRemove) != 1 {
		t.Fatalf("expected one WorktreeRemove call, got %d", len(gitClient.worktreeRemove))
	}
	if got := gitClient.worktreeRemove[0].path; got != "C:/worktrees/feature-x" {
		t.Fatalf("unexpected worktree remove path: %q", got)
	}
}

func TestRmCommand_PreservesBranchArgumentWithRemotePrefix(t *testing.T) {
	t.Parallel()

	gitClient := &fakeGitClient{
		output: "worktree C:/worktrees/origin-feature-demo\nHEAD def\nbranch refs/heads/origin/feature-demo\n\n",
	}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: &fakeSelector{index: 0},
		Opener:   &fakeOpener{},
	})
	cmd.SetArgs([]string{"rm", "origin/feature-demo"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if len(gitClient.worktreeRemove) != 1 {
		t.Fatalf("expected one WorktreeRemove call, got %d", len(gitClient.worktreeRemove))
	}
	if got := gitClient.worktreeRemove[0].path; got != "C:/worktrees/origin-feature-demo" {
		t.Fatalf("unexpected worktree remove path: %q", got)
	}
	if len(gitClient.deleteBranchCalls) != 1 {
		t.Fatalf("expected one DeleteLocalBranch call, got %d", len(gitClient.deleteBranchCalls))
	}
	if got := gitClient.deleteBranchCalls[0].branch; got != "origin/feature-demo" {
		t.Fatalf("unexpected branch deleted: %q", got)
	}
}

func TestRmCommand_ForceAlsoForcesBranchDeletionWhenPolicyNotExplicit(t *testing.T) {
	t.Parallel()

	gitClient := &fakeGitClient{
		output: "worktree C:/worktrees/feature-x\nHEAD def\nbranch refs/heads/feature/x\n\n",
	}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: &fakeSelector{index: 0},
		Opener:   &fakeOpener{},
	})
	cmd.SetArgs([]string{"rm", "feature/x", "--force"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if len(gitClient.worktreeRemove) != 1 || !gitClient.worktreeRemove[0].force {
		t.Fatalf("expected forced worktree removal, got %+v", gitClient.worktreeRemove)
	}
	if len(gitClient.deleteBranchCalls) != 1 || !gitClient.deleteBranchCalls[0].force {
		t.Fatalf("expected forced branch deletion, got %+v", gitClient.deleteBranchCalls)
	}
}

func TestRmCommand_ForceDoesNotOverrideExplicitDeletePolicy(t *testing.T) {
	t.Parallel()

	gitClient := &fakeGitClient{
		output: "worktree C:/worktrees/feature-x\nHEAD def\nbranch refs/heads/feature/x\n\n",
	}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: &fakeSelector{index: 0},
		Opener:   &fakeOpener{},
	})
	cmd.SetArgs([]string{"rm", "feature/x", "--force", "--delete-branch", "safe"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if len(gitClient.worktreeRemove) != 1 || !gitClient.worktreeRemove[0].force {
		t.Fatalf("expected forced worktree removal, got %+v", gitClient.worktreeRemove)
	}
	if len(gitClient.deleteBranchCalls) != 1 || gitClient.deleteBranchCalls[0].force {
		t.Fatalf("expected safe branch deletion, got %+v", gitClient.deleteBranchCalls)
	}
}

func TestRmCommand_DoesNotDeleteBranchWhenPolicyIsNone(t *testing.T) {
	t.Parallel()

	gitClient := &fakeGitClient{
		output: "worktree C:/worktrees/feature-x\nHEAD def\nbranch refs/heads/feature/x\n\n",
	}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: &fakeSelector{index: 0},
		Opener:   &fakeOpener{},
	})
	cmd.SetArgs([]string{"rm", "feature/x", "--delete-branch", "none"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if len(gitClient.deleteBranchCalls) != 0 {
		t.Fatalf("DeleteLocalBranch should not be called, got %+v", gitClient.deleteBranchCalls)
	}
}

func TestRmCommand_DoesNotDeleteBranchForDetachedWorktree(t *testing.T) {
	t.Parallel()

	gitClient := &fakeGitClient{
		output: "worktree C:/repo\nHEAD abc\ndetached\n\n",
	}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: &fakeSelector{index: 0},
		Opener:   &fakeOpener{},
	})
	cmd.SetArgs([]string{"rm"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if len(gitClient.worktreeRemove) != 1 {
		t.Fatalf("expected one WorktreeRemove call, got %d", len(gitClient.worktreeRemove))
	}
	if len(gitClient.deleteBranchCalls) != 0 {
		t.Fatalf("DeleteLocalBranch should not be called for detached worktree, got %+v", gitClient.deleteBranchCalls)
	}
}

func TestRmCommand_ReturnsErrorWhenBranchDoesNotHaveWorktree(t *testing.T) {
	t.Parallel()

	gitClient := &fakeGitClient{
		output: "worktree C:/repo\nHEAD abc\nbranch refs/heads/main\n\n",
	}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: &fakeSelector{index: 0},
		Opener:   &fakeOpener{},
	})
	cmd.SetArgs([]string{"rm", "feature/missing"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected Execute() to return error")
	}
	if !strings.Contains(err.Error(), "does not have a linked worktree") {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gitClient.worktreeRemove) != 0 {
		t.Fatalf("WorktreeRemove should not be called, got %+v", gitClient.worktreeRemove)
	}
}

func TestRmCommand_ReturnsErrorForInvalidDeleteBranchPolicy(t *testing.T) {
	t.Parallel()

	gitClient := &fakeGitClient{
		output: "worktree C:/worktrees/feature-x\nHEAD def\nbranch refs/heads/feature/x\n\n",
	}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: &fakeSelector{index: 0},
		Opener:   &fakeOpener{},
	})
	cmd.SetArgs([]string{"rm", "--delete-branch", "invalid"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected Execute() to return error")
	}
	if !strings.Contains(err.Error(), "invalid --delete-branch value") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRmCommand_UsesConfigDeleteBranchWhenFlagIsNotProvided(t *testing.T) {
	t.Parallel()

	gitClient := &fakeGitClient{
		output: "worktree C:/worktrees/feature-x\nHEAD def\nbranch refs/heads/feature/x\n\n",
	}
	cfgProvider := &fakeConfigProvider{
		cfg: config.Config{
			Remote:              config.DefaultRemote,
			BaseBranch:          config.DefaultBaseBranch,
			WorktreeDirTemplate: config.DefaultWorktreeDirTemplate,
			Open: config.Open{
				Default: "system",
				Window:  "new",
			},
			RM: config.RM{
				DeleteBranch: config.DeleteBranchNone,
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
	cmd.SetArgs([]string{"rm"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if len(gitClient.deleteBranchCalls) != 0 {
		t.Fatalf("DeleteLocalBranch should not be called, got %+v", gitClient.deleteBranchCalls)
	}
}

func TestRmCommand_FlagDeleteBranchOverridesConfig(t *testing.T) {
	t.Parallel()

	gitClient := &fakeGitClient{
		output: "worktree C:/worktrees/feature-x\nHEAD def\nbranch refs/heads/feature/x\n\n",
	}
	cfgProvider := &fakeConfigProvider{
		cfg: config.Config{
			Remote:              config.DefaultRemote,
			BaseBranch:          config.DefaultBaseBranch,
			WorktreeDirTemplate: config.DefaultWorktreeDirTemplate,
			Open: config.Open{
				Default: "system",
				Window:  "new",
			},
			RM: config.RM{
				DeleteBranch: config.DeleteBranchNone,
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
	cmd.SetArgs([]string{"rm", "--delete-branch", "safe"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if len(gitClient.deleteBranchCalls) != 1 {
		t.Fatalf("expected one DeleteLocalBranch call, got %d", len(gitClient.deleteBranchCalls))
	}
	if gitClient.deleteBranchCalls[0].force {
		t.Fatalf("expected safe delete, got force=true")
	}
}

func TestRmCommand_AllowsSelectingStaleWorktree(t *testing.T) {
	t.Parallel()

	gitClient := &fakeGitClient{
		output: "worktree C:/worktrees/stale\nHEAD abc\nbranch refs/heads/aaa\nprunable gitdir file points to non-existent location\n\n" +
			"worktree C:/worktrees/live\nHEAD def\nbranch refs/heads/feature/x\n\n",
	}
	selector := &fakeSelector{index: 0}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: selector,
		Opener:   &fakeOpener{},
	})
	cmd.SetArgs([]string{"rm"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if len(gitClient.worktreeRemove) != 0 {
		t.Fatalf("WorktreeRemove should not be called for stale selection, got %+v", gitClient.worktreeRemove)
	}
	if gitClient.worktreePruneCall != 1 {
		t.Fatalf("expected one WorktreePrune call for stale cleanup, got %d", gitClient.worktreePruneCall)
	}
	if len(selector.lastOptions) != 2 {
		t.Fatalf("expected 2 selector options, got %d", len(selector.lastOptions))
	}
	if !strings.HasSuffix(selector.lastOptions[0], "\t[stale]") {
		t.Fatalf("expected stale status suffix, got: %q", selector.lastOptions[0])
	}
	if !strings.HasSuffix(selector.lastOptions[1], "\t[active]") {
		t.Fatalf("expected active status suffix, got: %q", selector.lastOptions[1])
	}
}

func TestRmCommand_RemovesStaleWhenOnlyPrunableExists(t *testing.T) {
	t.Parallel()

	gitClient := &fakeGitClient{
		output: "worktree C:/worktrees/stale\nHEAD abc\nbranch refs/heads/aaa\nprunable gitdir file points to non-existent location\n\n",
	}
	selector := &fakeSelector{index: 0}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: selector,
		Opener:   &fakeOpener{},
	})
	cmd.SetArgs([]string{"rm"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if len(gitClient.worktreeRemove) != 0 {
		t.Fatalf("WorktreeRemove should not be called for stale entry, got %+v", gitClient.worktreeRemove)
	}
	if gitClient.worktreePruneCall != 1 {
		t.Fatalf("expected WorktreePrune to be called once, got %d", gitClient.worktreePruneCall)
	}
}

func TestRmCommand_RemovesActiveWhenSelected(t *testing.T) {
	t.Parallel()

	gitClient := &fakeGitClient{
		output: "worktree C:/worktrees/stale\nHEAD abc\nbranch refs/heads/aaa\nprunable gitdir file points to non-existent location\n\n" +
			"worktree C:/worktrees/live\nHEAD def\nbranch refs/heads/feature/x\n\n",
	}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: &fakeSelector{index: 1},
		Opener:   &fakeOpener{},
	})
	cmd.SetArgs([]string{"rm"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if len(gitClient.worktreeRemove) != 1 {
		t.Fatalf("expected one WorktreeRemove call, got %d", len(gitClient.worktreeRemove))
	}
	if gitClient.worktreeRemove[0].path != "C:/worktrees/live" {
		t.Fatalf("unexpected removed worktree path: %q", gitClient.worktreeRemove[0].path)
	}
	if gitClient.worktreePruneCall != 0 {
		t.Fatalf("did not expect prune for active removal, got %d", gitClient.worktreePruneCall)
	}
}

func TestRmCommand_RemovesStaleByBranchArgument(t *testing.T) {
	t.Parallel()

	gitClient := &fakeGitClient{
		output: "worktree C:/worktrees/stale\nHEAD abc\nbranch refs/heads/feature/stale\nprunable gitdir file points to non-existent location\n\n" +
			"worktree C:/repo\nHEAD def\nbranch refs/heads/main\n\n",
	}

	cmd := NewRootCmd(Dependencies{
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Git:      gitClient,
		Selector: &fakeSelector{index: 0},
		Opener:   &fakeOpener{},
	})
	cmd.SetArgs([]string{"rm", "feature/stale"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if gitClient.worktreePruneCall != 1 {
		t.Fatalf("expected WorktreePrune to be called once, got %d", gitClient.worktreePruneCall)
	}
	if len(gitClient.worktreeRemove) != 0 {
		t.Fatalf("WorktreeRemove should not be called for stale branch cleanup, got %+v", gitClient.worktreeRemove)
	}
}
