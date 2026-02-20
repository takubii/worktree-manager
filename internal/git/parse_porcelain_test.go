package git

import "testing"

func TestParseWorktreeListPorcelain_BranchEntry(t *testing.T) {
	t.Parallel()

	raw := "worktree C:/repo\nHEAD abc\nbranch refs/heads/main\n\n"
	worktrees, err := ParseWorktreeListPorcelain(raw)
	if err != nil {
		t.Fatalf("ParseWorktreeListPorcelain() returned error: %v", err)
	}

	if len(worktrees) != 1 {
		t.Fatalf("expected 1 worktree, got %d", len(worktrees))
	}
	if worktrees[0].Path != "C:/repo" {
		t.Fatalf("unexpected path: %q", worktrees[0].Path)
	}
	if worktrees[0].Branch != "refs/heads/main" {
		t.Fatalf("unexpected branch: %q", worktrees[0].Branch)
	}
	if worktrees[0].Detached {
		t.Fatalf("unexpected detached state: %v", worktrees[0].Detached)
	}
}

func TestParseWorktreeListPorcelain_DetachedEntry(t *testing.T) {
	t.Parallel()

	raw := "worktree C:/repo2\nHEAD def\ndetached\n\n"
	worktrees, err := ParseWorktreeListPorcelain(raw)
	if err != nil {
		t.Fatalf("ParseWorktreeListPorcelain() returned error: %v", err)
	}

	if len(worktrees) != 1 {
		t.Fatalf("expected 1 worktree, got %d", len(worktrees))
	}
	if !worktrees[0].Detached {
		t.Fatalf("expected detached state true")
	}
}

func TestParseWorktreeListPorcelain_IgnoresUnknownLines(t *testing.T) {
	t.Parallel()

	raw := "worktree C:/repo3\nHEAD ghi\nprunable gitdir file\nbranch refs/heads/feature\n\n"
	worktrees, err := ParseWorktreeListPorcelain(raw)
	if err != nil {
		t.Fatalf("ParseWorktreeListPorcelain() returned error: %v", err)
	}

	if len(worktrees) != 1 {
		t.Fatalf("expected 1 worktree, got %d", len(worktrees))
	}
}

func TestParseWorktreeListPorcelain_ReturnsErrorWhenMissingWorktreeHeader(t *testing.T) {
	t.Parallel()

	raw := "HEAD abc\nbranch refs/heads/main\n\n"
	_, err := ParseWorktreeListPorcelain(raw)
	if err == nil {
		t.Fatal("expected ParseWorktreeListPorcelain() to return error")
	}
}
