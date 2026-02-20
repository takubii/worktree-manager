package git

import "context"

// Client defines Git operations used by CLI commands.
type Client interface {
	WorktreeListPorcelain(ctx context.Context) (string, error)
}

// Worktree represents one git worktree entry.
type Worktree struct {
	Path     string
	Branch   string
	Detached bool
}

// NewClient returns a Client backed by the system git command.
func NewClient() Client {
	return newExecClient(nil)
}
