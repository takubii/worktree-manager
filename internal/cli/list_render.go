package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/takubii/git-worktree-opener/internal/config"
	"github.com/takubii/git-worktree-opener/internal/git"
)

type listRow struct {
	Current  bool   `json:"current"`
	Branch   string `json:"branch"`
	Detached bool   `json:"detached"`
	Status   string `json:"status"`
	Head     string `json:"head"`
	Path     string `json:"path"`
	Prunable bool   `json:"prunable"`
}

const (
	listHeaderCurrent = ""
	listHeaderBranch  = "BRANCH"
	listHeaderStatus  = "STATUS"
	listHeaderHead    = "HEAD"
	listHeaderPath    = "PATH"
)

type listTableWidths struct {
	current int
	branch  int
	status  int
	head    int
	path    int
}

func buildListRows(worktrees []git.Worktree, cwd string) []listRow {
	rows := make([]listRow, 0, len(worktrees))
	for _, wt := range worktrees {
		rows = append(rows, listRow{
			Current:  isPathWithinWorktree(cwd, wt.Path),
			Branch:   normalizeBranch(wt.Branch),
			Detached: wt.Detached,
			Status:   resolveListStatus(wt),
			Head:     wt.Head,
			Path:     wt.Path,
			Prunable: wt.Prunable,
		})
	}

	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Current != rows[j].Current {
			return rows[i].Current
		}

		branchI := strings.ToLower(rows[i].Branch)
		branchJ := strings.ToLower(rows[j].Branch)
		if branchI != branchJ {
			return branchI < branchJ
		}

		return strings.ToLower(rows[i].Path) < strings.ToLower(rows[j].Path)
	})

	return rows
}

func renderListTable(w io.Writer, rows []listRow) error {
	widths := buildListTableWidths(rows)
	header := renderListTableLine(widths, listHeaderCurrent, listHeaderBranch, listHeaderStatus, listHeaderHead, listHeaderPath)
	if _, err := fmt.Fprintln(w, header); err != nil {
		return fmt.Errorf("failed to write list header: %w", err)
	}

	for _, row := range rows {
		current := ""
		if row.Current {
			current = "*"
		}

		branch := displayListBranch(row)

		if _, err := fmt.Fprintln(w, renderListTableLine(
			widths,
			current,
			branch,
			row.Status,
			shortHead(row.Head),
			row.Path,
		)); err != nil {
			return fmt.Errorf("failed to write list row: %w", err)
		}
	}

	return nil
}

func renderListTableLine(widths listTableWidths, current string, branch string, status string, head string, path string) string {
	return fmt.Sprintf(
		"%s  %s  %s  %s  %s",
		padOrTruncate(current, widths.current),
		padOrTruncate(branch, widths.branch),
		padOrTruncate(status, widths.status),
		padOrTruncate(head, widths.head),
		padOrTruncate(path, widths.path),
	)
}

func buildListTableWidths(rows []listRow) listTableWidths {
	widths := listTableWidths{
		current: max(config.ListColumnCurrentWidth, runeLen(listHeaderCurrent)),
		branch:  runeLen(listHeaderBranch),
		status:  max(config.ListColumnStatusWidth, runeLen(listHeaderStatus)),
		head:    max(config.ListColumnHeadWidth, runeLen(listHeaderHead)),
		path:    max(config.ListColumnPathWidth, runeLen(listHeaderPath)),
	}

	for _, row := range rows {
		widths.current = max(widths.current, runeLen("*"))
		widths.branch = max(widths.branch, runeLen(displayListBranch(row)))
	}

	if widths.branch > config.ListColumnBranchWidth {
		widths.branch = config.ListColumnBranchWidth
	}

	return widths
}

func padOrTruncate(value string, width int) string {
	if width <= 0 {
		return value
	}

	runes := []rune(value)
	if len(runes) > width {
		if width <= 3 {
			return string(runes[:width])
		}
		return string(runes[:width-3]) + "..."
	}

	if len(runes) < width {
		return value + strings.Repeat(" ", width-len(runes))
	}

	return value
}

func displayListBranch(row listRow) string {
	if row.Detached {
		return branchLabelDetached
	}
	if row.Branch == "" {
		return branchLabelUnknown
	}
	return row.Branch
}

func shortHead(head string) string {
	head = strings.TrimSpace(head)
	if head == "" {
		return "-"
	}

	runes := []rune(head)
	if len(runes) <= config.ListColumnHeadWidth {
		return head
	}
	return string(runes[:min(len(runes), config.ListColumnHeadWidth)])
}

func resolveListStatus(wt git.Worktree) string {
	if wt.Prunable {
		return config.ListStatusStale
	}

	_, err := os.Stat(wt.Path)
	switch {
	case err == nil:
		return config.ListStatusActive
	case os.IsNotExist(err):
		return config.ListStatusMissing
	default:
		return config.ListStatusActive
	}
}

func isPathWithinWorktree(cwd string, worktreePath string) bool {
	if strings.TrimSpace(cwd) == "" || strings.TrimSpace(worktreePath) == "" {
		return false
	}

	cwdAbs, err := filepath.Abs(cwd)
	if err != nil {
		return false
	}
	worktreeAbs, err := filepath.Abs(worktreePath)
	if err != nil {
		return false
	}

	cwdAbs = filepath.Clean(cwdAbs)
	worktreeAbs = filepath.Clean(worktreeAbs)

	rel, err := filepath.Rel(worktreeAbs, cwdAbs)
	if err != nil {
		return false
	}

	if rel == "." {
		return true
	}

	if rel == ".." {
		return false
	}

	return !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func runeLen(s string) int {
	return len([]rune(s))
}
