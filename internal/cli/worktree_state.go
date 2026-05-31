package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/takubii/worktree-manager/internal/git"
)

type unavailableWorktrees struct {
	active  []git.Worktree
	stale   []git.Worktree
	missing []git.Worktree
}

func splitPrunableWorktrees(worktrees []git.Worktree) ([]git.Worktree, []git.Worktree) {
	active := make([]git.Worktree, 0, len(worktrees))
	prunable := make([]git.Worktree, 0)

	for _, wt := range worktrees {
		if wt.Prunable {
			prunable = append(prunable, wt)
			continue
		}
		active = append(active, wt)
	}

	return active, prunable
}

func splitUnavailableWorktreesForPath(worktrees []git.Worktree) ([]git.Worktree, []git.Worktree, []git.Worktree) {
	state := classifyUnavailableWorktrees(worktrees)
	return state.active, state.stale, state.missing
}

func classifyUnavailableWorktrees(worktrees []git.Worktree) unavailableWorktrees {
	state := unavailableWorktrees{
		active:  make([]git.Worktree, 0, len(worktrees)),
		stale:   make([]git.Worktree, 0),
		missing: make([]git.Worktree, 0),
	}

	for _, wt := range worktrees {
		if wt.Prunable {
			state.stale = append(state.stale, wt)
			continue
		}
		if _, err := os.Stat(wt.Path); err != nil {
			if os.IsNotExist(err) {
				state.missing = append(state.missing, wt)
				continue
			}
		}

		state.active = append(state.active, wt)
	}

	return state
}

func warnSkippedPrunableWorktrees(w io.Writer, commandName string, prunable []git.Worktree) {
	if len(prunable) == 0 || w == nil {
		return
	}

	paths := make([]string, 0, len(prunable))
	for _, wt := range prunable {
		paths = append(paths, wt.Path)
	}

	_, _ = fmt.Fprintf(
		w,
		"warning: skipped %d stale worktree entries in `%s` (marked `prunable`): %s\n",
		len(prunable),
		commandName,
		strings.Join(paths, ", "),
	)
	warnStaleWorktreeGuidance(w)
}

func warnSkippedMissingWorktrees(w io.Writer, commandName string, missing []git.Worktree) {
	if len(missing) == 0 || w == nil {
		return
	}

	paths := make([]string, 0, len(missing))
	for _, wt := range missing {
		paths = append(paths, wt.Path)
	}

	_, _ = fmt.Fprintf(
		w,
		"warning: skipped %d missing worktree entries in `%s` (path not found locally): %s\n",
		len(missing),
		commandName,
		strings.Join(paths, ", "),
	)
	warnMissingWorktreeGuidance(w)
}

func warnUnavailableWorktreesForList(w io.Writer, state unavailableWorktrees) {
	if w == nil {
		return
	}
	warnListedPrunableWorktrees(w, state.stale)
	warnListedMissingWorktrees(w, state.missing)
}

func noValidWorktreesForPathError(stale []git.Worktree, missing []git.Worktree) error {
	return fmt.Errorf(
		"no valid worktrees found after filtering %d stale and %d missing entries. Run `wtm list` to inspect current state, then run `wtm remove <branch>` or `wtm remove` to clean stale/missing entries before retrying `wtm path`",
		len(stale),
		len(missing),
	)
}

func warnListedPrunableWorktrees(w io.Writer, prunable []git.Worktree) {
	if len(prunable) == 0 || w == nil {
		return
	}

	paths := make([]string, 0, len(prunable))
	for _, wt := range prunable {
		paths = append(paths, wt.Path)
	}

	_, _ = fmt.Fprintf(
		w,
		"warning: found %d stale worktree entries in `wtm list` (marked `prunable`): %s\n",
		len(prunable),
		strings.Join(paths, ", "),
	)
	warnStaleWorktreeGuidance(w)
}

func warnListedMissingWorktrees(w io.Writer, missing []git.Worktree) {
	if len(missing) == 0 || w == nil {
		return
	}

	paths := make([]string, 0, len(missing))
	for _, wt := range missing {
		paths = append(paths, wt.Path)
	}

	_, _ = fmt.Fprintf(
		w,
		"warning: found %d missing worktree entries in `wtm list` (path not found locally): %s\n",
		len(missing),
		strings.Join(paths, ", "),
	)
	warnMissingWorktreeGuidance(w)
}

func warnStaleWorktreeGuidance(w io.Writer) {
	_, _ = fmt.Fprintln(
		w,
		"warning: stale entries are Git metadata only; run `wtm list` to inspect them, then run `wtm remove <branch>` or `wtm remove` to prune stale metadata explicitly.",
	)
}

func warnMissingWorktreeGuidance(w io.Writer) {
	_, _ = fmt.Fprintln(
		w,
		"warning: missing entries keep their Git metadata but the worktree path is gone; run `wtm list` to inspect them, then restore the path or run `wtm remove <branch>` / `wtm remove` to remove the registered worktree.",
	)
}
