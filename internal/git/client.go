package git

import "context"

// Client defines Git operations used by CLI commands.
type Client interface {
	WorktreeListPorcelain(ctx context.Context) (string, error)
	WorktreePrune(ctx context.Context) error
	RepoRoot(ctx context.Context) (string, error)
	FetchPrune(ctx context.Context, remote string) error
	LocalBranches(ctx context.Context) ([]string, error)
	RemoteBranches(ctx context.Context, remote string) ([]string, error)
	CheckBranchName(ctx context.Context, branch string) error
	WorktreeAdd(ctx context.Context, params WorktreeAddParams) error
	WorktreeRemove(ctx context.Context, path string, force bool) error
	DeleteLocalBranch(ctx context.Context, branch string, force bool) error
}

// WorktreeAddParams defines inputs for `git worktree add`.
type WorktreeAddParams struct {
	Path       string
	Branch     string
	StartPoint string
}

// Worktree represents one git worktree entry.
type Worktree struct {
	Path     string
	Branch   string
	Head     string
	Detached bool
	Prunable bool
}

// NewClient returns a Client backed by the system git command.
func NewClient() Client {
	return newExecClient(nil)
}
