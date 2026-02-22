package config

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/takubii/git-worktree-opener/internal/opener"
)

var placeholderPattern = regexp.MustCompile(`\{([^{}]+)\}`)

type configOverride struct {
	Remote              *string
	BaseBranch          *string
	WorktreeDirTemplate *string
	OpenDefault         *string
	OpenWindow          *string
	RMDeleteBranch      *string
}

func mergeConfig(base Config, override configOverride) Config {
	merged := base

	if override.Remote != nil {
		merged.Remote = *override.Remote
	}
	if override.BaseBranch != nil {
		merged.BaseBranch = *override.BaseBranch
	}
	if override.WorktreeDirTemplate != nil {
		merged.WorktreeDirTemplate = *override.WorktreeDirTemplate
	}
	if override.OpenDefault != nil {
		merged.Open.Default = *override.OpenDefault
	}
	if override.OpenWindow != nil {
		merged.Open.Window = *override.OpenWindow
	}
	if override.RMDeleteBranch != nil {
		merged.RM.DeleteBranch = *override.RMDeleteBranch
	}

	return merged
}

func normalizeOverride(raw rawConfig) (configOverride, error) {
	var out configOverride

	remote, err := normalizeNonEmptyString("remote", raw.Remote)
	if err != nil {
		return configOverride{}, err
	}
	out.Remote = remote

	baseBranch, err := normalizeNonEmptyString("baseBranch", raw.BaseBranch)
	if err != nil {
		return configOverride{}, err
	}
	out.BaseBranch = baseBranch

	template, err := normalizeWorktreeDirTemplate(raw.WorktreeDirTemplate)
	if err != nil {
		return configOverride{}, err
	}
	out.WorktreeDirTemplate = template

	if raw.Open != nil {
		openDefault, err := normalizeOpenKind(raw.Open.Default)
		if err != nil {
			return configOverride{}, err
		}
		out.OpenDefault = openDefault

		openWindow, err := normalizeWindow(raw.Open.Window)
		if err != nil {
			return configOverride{}, err
		}
		out.OpenWindow = openWindow
	}

	if raw.RM != nil {
		deleteBranch, err := normalizeDeleteBranch(raw.RM.DeleteBranch)
		if err != nil {
			return configOverride{}, err
		}
		out.RMDeleteBranch = deleteBranch
	}

	return out, nil
}

func normalizeNonEmptyString(field string, value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil, fmt.Errorf("%s is empty. Provide a non-empty value", field)
	}
	return &trimmed, nil
}

func normalizeOpenKind(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}

	trimmed := strings.ToLower(strings.TrimSpace(*value))
	if _, ok := supportedOpenKinds[trimmed]; ok {
		return &trimmed, nil
	}

	return nil, fmt.Errorf("open.default %q is invalid. Use one of: %s", *value, SupportedOpenKindsText)
}

func normalizeWindow(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}

	trimmed := strings.ToLower(strings.TrimSpace(*value))
	if _, err := opener.ParseWindowMode(trimmed); err != nil {
		return nil, fmt.Errorf("open.window is invalid: %w", err)
	}
	return &trimmed, nil
}

func normalizeDeleteBranch(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}

	trimmed := strings.ToLower(strings.TrimSpace(*value))
	switch trimmed {
	case DeleteBranchNone, DeleteBranchSafe, DeleteBranchForce:
		return &trimmed, nil
	default:
		return nil, fmt.Errorf("rm.deleteBranch %q is invalid. Use one of: %s", *value, SupportedDeleteBranchModesText)
	}
}

func normalizeWorktreeDirTemplate(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil, fmt.Errorf("worktreeDirTemplate is empty. Provide a non-empty template")
	}
	if err := validateTemplatePlaceholders(trimmed); err != nil {
		return nil, err
	}

	return &trimmed, nil
}

func validateTemplatePlaceholders(template string) error {
	matches := placeholderPattern.FindAllStringSubmatch(template, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		if _, ok := supportedTemplateTokens[match[1]]; !ok {
			return fmt.Errorf(
				"worktreeDirTemplate placeholder %q is not supported. Use only: %s",
				match[0],
				SupportedWorktreeTemplateTokensText,
			)
		}
	}

	return nil
}

// RenderWorktreeDir resolves a template into the concrete worktree path.
func RenderWorktreeDir(template string, repoRoot string, branch string) (string, error) {
	template = strings.TrimSpace(template)
	if template == "" {
		return "", fmt.Errorf("worktreeDirTemplate is empty. Set a valid template and retry")
	}
	if err := validateTemplatePlaceholders(template); err != nil {
		return "", err
	}

	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		return "", fmt.Errorf("repository root is empty. Run this command inside a Git repository, then retry")
	}
	repoRoot = filepath.Clean(repoRoot)

	branch = strings.TrimSpace(strings.TrimPrefix(branch, "refs/heads/"))
	if branch == "" {
		return "", fmt.Errorf("branch name is empty. Specify a branch and retry")
	}

	rendered := template
	rendered = strings.ReplaceAll(rendered, "{repoParent}", filepath.Dir(repoRoot))
	rendered = strings.ReplaceAll(rendered, "{repoRoot}", repoRoot)
	rendered = strings.ReplaceAll(rendered, "{branch}", filepath.FromSlash(branch))

	if placeholderPattern.MatchString(rendered) {
		return "", fmt.Errorf("worktreeDirTemplate produced unresolved placeholders in %q. Check placeholders and retry", rendered)
	}

	if !filepath.IsAbs(rendered) {
		rendered = filepath.Join(repoRoot, rendered)
	}

	return filepath.Clean(rendered), nil
}
