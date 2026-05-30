package config

import (
	"strings"
	"testing"
)

func TestNormalizeOverride_ReadsCreateAndRemove(t *testing.T) {
	t.Parallel()

	fetch := false
	prune := false
	deleteBranch := DeleteBranchForce
	override, err := normalizeOverride(rawConfig{
		Create: &rawCreate{
			Fetch: &fetch,
			Prune: &prune,
		},
		Remove: &rawRemove{
			DeleteBranch: &deleteBranch,
		},
	})
	if err != nil {
		t.Fatalf("normalizeOverride() returned error: %v", err)
	}
	if override.CreateFetch == nil || *override.CreateFetch {
		t.Fatalf("unexpected CreateFetch override: %+v", override.CreateFetch)
	}
	if override.CreatePrune == nil || *override.CreatePrune {
		t.Fatalf("unexpected CreatePrune override: %+v", override.CreatePrune)
	}
	if override.RemoveDeleteBranch == nil || *override.RemoveDeleteBranch != DeleteBranchForce {
		t.Fatalf("unexpected RemoveDeleteBranch override: %+v", override.RemoveDeleteBranch)
	}
}

func TestMergeConfig_AppliesOverrides(t *testing.T) {
	t.Parallel()

	fetch := false
	deleteBranch := DeleteBranchNone
	got := mergeConfig(DefaultConfig(), configOverride{
		CreateFetch:        &fetch,
		RemoveDeleteBranch: &deleteBranch,
	})

	if got.Create.Fetch {
		t.Fatalf("unexpected create.fetch: %v", got.Create.Fetch)
	}
	if got.Remove.DeleteBranch != DeleteBranchNone {
		t.Fatalf("unexpected remove.deleteBranch: %q", got.Remove.DeleteBranch)
	}
}

func TestNormalizeOverride_ReturnsErrorForInvalidRemoveDeleteBranch(t *testing.T) {
	t.Parallel()

	value := "invalid"
	_, err := normalizeOverride(rawConfig{
		Remove: &rawRemove{DeleteBranch: &value},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "remove.deleteBranch") {
		t.Fatalf("unexpected error: %v", err)
	}
}
