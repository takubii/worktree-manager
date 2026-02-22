package cli

import (
	"strings"
	"testing"

	"github.com/takubii/git-worktree-opener/internal/config"
)

func TestBuildListTableWidths_AdjustsBranchWidthToContent(t *testing.T) {
	t.Parallel()

	rows := []listRow{
		{Current: true, Branch: "main"},
		{Branch: "feature/long-name"},
	}

	widths := buildListTableWidths(rows)

	if widths.current != config.ListColumnCurrentWidth {
		t.Fatalf("unexpected current width: want=%d got=%d", config.ListColumnCurrentWidth, widths.current)
	}

	wantBranchWidth := runeLen("feature/long-name")
	if widths.branch != wantBranchWidth {
		t.Fatalf("unexpected branch width: want=%d got=%d", wantBranchWidth, widths.branch)
	}
}

func TestBuildListTableWidths_CapsBranchWidthAtConfiguredMax(t *testing.T) {
	t.Parallel()

	rows := []listRow{
		{Branch: strings.Repeat("x", config.ListColumnBranchWidth+20)},
	}

	widths := buildListTableWidths(rows)
	if widths.branch != config.ListColumnBranchWidth {
		t.Fatalf("unexpected capped branch width: want=%d got=%d", config.ListColumnBranchWidth, widths.branch)
	}
}
