package config

import "github.com/takubii/git-worktree-opener/internal/opener"

const (
	// DefaultRemote is the built-in remote name.
	DefaultRemote = "origin"
	// DefaultBaseBranch is the built-in base branch.
	DefaultBaseBranch = "main"
	// DefaultWorktreeDirTemplate is the built-in worktree directory template.
	DefaultWorktreeDirTemplate = "{repoParent}/worktrees/{branch}"

	// DeleteBranchNone skips local branch deletion in rm.
	DeleteBranchNone = "none"
	// DeleteBranchSafe uses `git branch -d`.
	DeleteBranchSafe = "safe"
	// DeleteBranchForce uses `git branch -D`.
	DeleteBranchForce = "force"
)

// DefaultConfig returns the built-in defaults.
func DefaultConfig() Config {
	return Config{
		Remote:              DefaultRemote,
		BaseBranch:          DefaultBaseBranch,
		WorktreeDirTemplate: DefaultWorktreeDirTemplate,
		Open: Open{
			Default: opener.KindSystem,
			Window:  string(opener.WindowNew),
		},
		RM: RM{
			DeleteBranch: DeleteBranchSafe,
		},
	}
}
