package config

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var placeholderPattern = regexp.MustCompile(`\{([^{}]+)\}`)

type configOverride struct {
	Remote              *string
	BaseBranch          *string
	WorktreeDirTemplate *string
	CreateFetch         *bool
	CreatePrune         *bool
	CreateBootstrap     *Bootstrap
	RemoveDeleteBranch  *string
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
	if override.CreateFetch != nil {
		merged.Create.Fetch = *override.CreateFetch
	}
	if override.CreatePrune != nil {
		merged.Create.Prune = *override.CreatePrune
	}
	if override.CreateBootstrap != nil {
		merged.Create.Bootstrap = *override.CreateBootstrap
	}
	if override.RemoveDeleteBranch != nil {
		merged.Remove.DeleteBranch = *override.RemoveDeleteBranch
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

	if raw.Create != nil {
		out.CreateFetch = normalizeOptionalBool(raw.Create.Fetch)
		out.CreatePrune = normalizeOptionalBool(raw.Create.Prune)
		bootstrap, err := normalizeBootstrap(raw.Create.Bootstrap)
		if err != nil {
			return configOverride{}, err
		}
		out.CreateBootstrap = bootstrap
	}

	if raw.Remove != nil {
		deleteBranch, err := normalizeDeleteBranch(raw.Remove.DeleteBranch)
		if err != nil {
			return configOverride{}, err
		}
		out.RemoveDeleteBranch = deleteBranch
	}

	return out, nil
}

func normalizeBootstrap(raw *rawBootstrap) (*Bootstrap, error) {
	if raw == nil {
		return nil, nil
	}

	copyFiles := make([]CopyFileAction, 0, len(raw.CopyFiles))
	for i, action := range raw.CopyFiles {
		normalized, err := normalizeCopyFileAction(i, action)
		if err != nil {
			return nil, err
		}
		copyFiles = append(copyFiles, normalized)
	}

	postCreate := make([]HookAction, 0, len(raw.PostCreate))
	for i, action := range raw.PostCreate {
		normalized, err := normalizeHookAction(i, action)
		if err != nil {
			return nil, err
		}
		postCreate = append(postCreate, normalized)
	}

	return &Bootstrap{
		CopyFiles:  copyFiles,
		PostCreate: postCreate,
	}, nil
}

func normalizeCopyFileAction(index int, raw rawCopyFileAction) (CopyFileAction, error) {
	from, err := normalizeNonEmptyString(fmt.Sprintf("create.bootstrap.copyFiles[%d].from", index), raw.From)
	if err != nil {
		return CopyFileAction{}, err
	}
	if from == nil {
		return CopyFileAction{}, fmt.Errorf("create.bootstrap.copyFiles[%d].from is required. Provide a source file path", index)
	}
	if err := validateBootstrapPlaceholders("create.bootstrap.copyFiles[].from", *from); err != nil {
		return CopyFileAction{}, err
	}

	to, err := normalizeNonEmptyString(fmt.Sprintf("create.bootstrap.copyFiles[%d].to", index), raw.To)
	if err != nil {
		return CopyFileAction{}, err
	}
	if to == nil {
		return CopyFileAction{}, fmt.Errorf("create.bootstrap.copyFiles[%d].to is required. Provide a destination file path", index)
	}
	if err := validateBootstrapPlaceholders("create.bootstrap.copyFiles[].to", *to); err != nil {
		return CopyFileAction{}, err
	}

	return CopyFileAction{
		From:      *from,
		To:        *to,
		Overwrite: boolValue(raw.Overwrite),
		Required:  boolValue(raw.Required),
	}, nil
}

func normalizeHookAction(index int, raw rawHookAction) (HookAction, error) {
	command := make([]string, 0, len(raw.Command))
	for argIndex, arg := range raw.Command {
		trimmed := strings.TrimSpace(arg)
		if trimmed == "" {
			return HookAction{}, fmt.Errorf("create.bootstrap.postCreate[%d].command[%d] is empty. Provide a non-empty command argument", index, argIndex)
		}
		if err := validateBootstrapPlaceholders("create.bootstrap.postCreate[].command", trimmed); err != nil {
			return HookAction{}, err
		}
		command = append(command, trimmed)
	}
	if len(command) == 0 {
		return HookAction{}, fmt.Errorf("create.bootstrap.postCreate[%d].command is required. Provide argv such as [\"npm\", \"install\"]", index)
	}

	name := ""
	if raw.Name != nil {
		trimmed := strings.TrimSpace(*raw.Name)
		if trimmed != "" {
			name = trimmed
		}
	}

	cwd := ""
	if raw.CWD != nil {
		trimmed := strings.TrimSpace(*raw.CWD)
		if trimmed == "" {
			return HookAction{}, fmt.Errorf("create.bootstrap.postCreate[%d].cwd is empty. Omit it or provide a worktree-relative path", index)
		}
		if err := validateBootstrapPlaceholders("create.bootstrap.postCreate[].cwd", trimmed); err != nil {
			return HookAction{}, err
		}
		cwd = trimmed
	}

	return HookAction{
		Name:    name,
		Command: command,
		CWD:     cwd,
	}, nil
}

func boolValue(value *bool) bool {
	if value == nil {
		return false
	}
	return *value
}

func validateBootstrapPlaceholders(field string, value string) error {
	matches := placeholderPattern.FindAllStringSubmatch(value, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		switch match[1] {
		case WorktreeTemplateTokenRepoRoot, WorktreeTemplateTokenBranch, "worktree":
		default:
			return fmt.Errorf(
				"%s placeholder %q is not supported. Use only: {repoRoot}, {worktree}, {branch}",
				field,
				match[0],
			)
		}
	}

	return nil
}

func normalizeOptionalBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	normalized := *value
	return &normalized
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

func normalizeDeleteBranch(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}

	trimmed := strings.ToLower(strings.TrimSpace(*value))
	switch trimmed {
	case DeleteBranchNone, DeleteBranchSafe, DeleteBranchForce:
		return &trimmed, nil
	default:
		return nil, fmt.Errorf("remove.deleteBranch %q is invalid. Use one of: %s", *value, SupportedDeleteBranchModesText)
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
