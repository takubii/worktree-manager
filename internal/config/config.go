package config

import "context"

// Config stores effective runtime settings for wtm.
type Config struct {
	Remote              string `json:"remote"`
	BaseBranch          string `json:"baseBranch"`
	WorktreeDirTemplate string `json:"worktreeDirTemplate"`
	Create              Create `json:"create"`
	Remove              Remove `json:"remove"`
}

// Create stores defaults for `wtm create`.
type Create struct {
	Fetch     bool      `json:"fetch"`
	Prune     bool      `json:"prune"`
	Bootstrap Bootstrap `json:"bootstrap,omitempty"`
}

// Bootstrap stores optional post-create bootstrap actions.
type Bootstrap struct {
	CopyFiles  []CopyFileAction `json:"copyFiles,omitempty"`
	PostCreate []HookAction     `json:"postCreate,omitempty"`
}

// CopyFileAction copies one local file into a new worktree.
type CopyFileAction struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Overwrite bool   `json:"overwrite"`
	Required  bool   `json:"required"`
}

// HookAction runs one command after a worktree is created.
type HookAction struct {
	Name    string   `json:"name,omitempty"`
	Command []string `json:"command"`
	CWD     string   `json:"cwd,omitempty"`
}

// Remove stores removal defaults.
type Remove struct {
	DeleteBranch string `json:"deleteBranch"`
}

// Provider resolves effective config and handles config file initialization.
type Provider interface {
	Load(ctx context.Context) Config
	InitGlobal(force bool) (string, error)
}
