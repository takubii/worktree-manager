package config

import "context"

// Config stores effective runtime settings for wto.
type Config struct {
	Remote              string `json:"remote"`
	BaseBranch          string `json:"baseBranch"`
	WorktreeDirTemplate string `json:"worktreeDirTemplate"`
	Open                Open   `json:"open"`
	RM                  RM     `json:"rm"`
}

// Open stores opener defaults.
type Open struct {
	Default string `json:"default"`
	Window  string `json:"window"`
}

// RM stores removal defaults.
type RM struct {
	DeleteBranch string `json:"deleteBranch"`
}

// Provider resolves effective config and handles config file initialization.
type Provider interface {
	Load(ctx context.Context) Config
	InitGlobal(force bool) (string, error)
}
