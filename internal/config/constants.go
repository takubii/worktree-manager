package config

const (
	// ListFormatTable is the table format for `wtm list`.
	ListFormatTable = "table"
	// ListFormatRaw is the raw porcelain format for `wtm list`.
	ListFormatRaw = "raw"
	// ListFormatJSON is the JSON format for `wtm list`.
	ListFormatJSON = "json"
	// ListDefaultFormat is the default output format for `wtm list`.
	ListDefaultFormat = ListFormatTable

	// ListStatusActive is shown for healthy worktree entries.
	ListStatusActive = "active"
	// ListStatusStale is shown for prunable worktree entries.
	ListStatusStale = "stale"
	// ListStatusMissing is shown when a worktree path does not exist locally.
	ListStatusMissing = "missing"

	// ListColumnCurrentWidth is the table width for current marker.
	ListColumnCurrentWidth = 1
	// ListColumnBranchWidth is the max table width for BRANCH.
	ListColumnBranchWidth = 24
	// ListColumnStatusWidth is the table width for STATUS.
	ListColumnStatusWidth = 7
	// ListColumnHeadWidth is the table width for HEAD.
	ListColumnHeadWidth = 7
	// ListColumnPathWidth is the table width for PATH.
	ListColumnPathWidth = 64

	// SupportedDeleteBranchModesText is used in help/error messages.
	SupportedDeleteBranchModesText = "none|safe|force"
	// ListSupportedFormatsText is used in help/error messages.
	ListSupportedFormatsText = "table|raw|json"
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
	supportedTemplateTokens = map[string]struct{}{
		WorktreeTemplateTokenRepoParent: {},
		WorktreeTemplateTokenRepoRoot:   {},
		WorktreeTemplateTokenBranch:     {},
	}
)
