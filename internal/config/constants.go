package config

import "github.com/takubii/git-worktree-opener/internal/opener"

const (
	// DefaultOpenKind is the built-in opener kind.
	DefaultOpenKind = opener.KindSystem
	// DefaultOpenWindow is the built-in window mode.
	DefaultOpenWindow = string(opener.WindowNew)
	// OpenKindTerminal is the terminal opener kind.
	OpenKindTerminal = "terminal"
	// DefaultOpenTerminalProvider is the built-in terminal provider selection.
	DefaultOpenTerminalProvider = TerminalProviderAuto

	// ListFormatTable is the table format for `wto list`.
	ListFormatTable = "table"
	// ListFormatRaw is the raw porcelain format for `wto list`.
	ListFormatRaw = "raw"
	// ListFormatJSON is the JSON format for `wto list`.
	ListFormatJSON = "json"
	// ListDefaultFormat is the default output format for `wto list`.
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

	// SupportedOpenKindsText is used in help/error messages.
	SupportedOpenKindsText = "system|vscode|cursor|vim|terminal"
	// SupportedWindowModesText is used in help/error messages.
	SupportedWindowModesText = "new|reuse"
	// SupportedTerminalProvidersText is used in help/error messages.
	SupportedTerminalProvidersText = "auto|windows-terminal|cmd|powershell|terminal|gnome-terminal|wezterm|iterm2|ghostty|warp|tabby"
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

	// TerminalProviderAuto lets runtime choose provider by OS policy.
	TerminalProviderAuto = "auto"
	// TerminalProviderWindowsTerminal is the Windows Terminal provider.
	TerminalProviderWindowsTerminal = "windows-terminal"
	// TerminalProviderCMD is the Windows cmd provider.
	TerminalProviderCMD = "cmd"
	// TerminalProviderPowerShell is the Windows PowerShell provider.
	TerminalProviderPowerShell = "powershell"
	// TerminalProviderMacTerminal is the macOS Terminal.app provider.
	TerminalProviderMacTerminal = "terminal"
	// TerminalProviderGNOMETerminal is the GNOME Terminal provider.
	TerminalProviderGNOMETerminal = "gnome-terminal"
	// TerminalProviderWezTerm is the WezTerm provider.
	TerminalProviderWezTerm = "wezterm"
	// TerminalProviderITerm2 is the iTerm2 provider.
	TerminalProviderITerm2 = "iterm2"
	// TerminalProviderGhostty is the Ghostty provider.
	TerminalProviderGhostty = "ghostty"
	// TerminalProviderWarp is the Warp provider.
	TerminalProviderWarp = "warp"
	// TerminalProviderTabby is the Tabby provider.
	TerminalProviderTabby = "tabby"
)

var (
	supportedOpenKinds = map[string]struct{}{
		opener.KindSystem: {},
		opener.KindVSCode: {},
		opener.KindCursor: {},
		opener.KindVim:    {},
		OpenKindTerminal:  {},
	}
	supportedTerminalProviders = map[string]struct{}{
		TerminalProviderAuto:            {},
		TerminalProviderWindowsTerminal: {},
		TerminalProviderCMD:             {},
		TerminalProviderPowerShell:      {},
		TerminalProviderMacTerminal:     {},
		TerminalProviderGNOMETerminal:   {},
		TerminalProviderWezTerm:         {},
		TerminalProviderITerm2:          {},
		TerminalProviderGhostty:         {},
		TerminalProviderWarp:            {},
		TerminalProviderTabby:           {},
	}
	supportedTemplateTokens = map[string]struct{}{
		WorktreeTemplateTokenRepoParent: {},
		WorktreeTemplateTokenRepoRoot:   {},
		WorktreeTemplateTokenBranch:     {},
	}
)
