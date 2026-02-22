package config

import "github.com/takubii/git-worktree-opener/internal/opener"

const (
	// DefaultOpenKind is the built-in opener kind.
	DefaultOpenKind = opener.KindSystem
	// DefaultOpenWindow is the built-in window mode.
	DefaultOpenWindow = string(opener.WindowNew)

	// SupportedOpenKindsText is used in help/error messages.
	SupportedOpenKindsText = "system|vscode|cursor|vim"
	// SupportedWindowModesText is used in help/error messages.
	SupportedWindowModesText = "new|reuse"
	// SupportedDeleteBranchModesText is used in help/error messages.
	SupportedDeleteBranchModesText = "none|safe|force"
	// SupportedWorktreeTemplateTokensText is used in help/error messages.
	SupportedWorktreeTemplateTokensText = "{repoParent}, {repoRoot}, {branch}"

	// WorktreeTemplateTokenRepoParent is the {repoParent} placeholder.
	WorktreeTemplateTokenRepoParent = "repoParent"
	// WorktreeTemplateTokenRepoRoot is the {repoRoot} placeholder.
	WorktreeTemplateTokenRepoRoot = "repoRoot"
	// WorktreeTemplateTokenBranch is the {branch} placeholder.
	WorktreeTemplateTokenBranch = "branch"
)

var (
	supportedOpenKinds = map[string]struct{}{
		opener.KindSystem: {},
		opener.KindVSCode: {},
		opener.KindCursor: {},
		opener.KindVim:    {},
	}
	supportedTemplateTokens = map[string]struct{}{
		WorktreeTemplateTokenRepoParent: {},
		WorktreeTemplateTokenRepoRoot:   {},
		WorktreeTemplateTokenBranch:     {},
	}
)
