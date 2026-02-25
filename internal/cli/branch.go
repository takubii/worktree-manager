package cli

import "strings"

const localBranchRefPrefix = "refs/heads/"

const (
	branchLabelDetached = "(detached)"
	branchLabelUnknown  = "-"
)

func normalizeBranch(branch string) string {
	branch = strings.TrimSpace(branch)
	return strings.TrimPrefix(branch, localBranchRefPrefix)
}
