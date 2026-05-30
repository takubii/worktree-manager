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
	Fetch bool `json:"fetch"`
	Prune bool `json:"prune"`
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
