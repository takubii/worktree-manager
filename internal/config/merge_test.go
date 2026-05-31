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

func TestNormalizeOverride_ReadsBootstrap(t *testing.T) {
	t.Parallel()

	overwrite := true
	required := true
	name := "install dependencies"
	cwd := "frontend"
	override, err := normalizeOverride(rawConfig{
		Create: &rawCreate{
			Bootstrap: &rawBootstrap{
				CopyFiles: []rawCopyFileAction{{
					From:      stringPtr(".env"),
					To:        stringPtr(".env"),
					Overwrite: &overwrite,
					Required:  &required,
				}},
				PostCreate: []rawHookAction{{
					Name:    &name,
					Command: []string{"npm", "install"},
					CWD:     &cwd,
				}},
			},
		},
	})
	if err != nil {
		t.Fatalf("normalizeOverride() returned error: %v", err)
	}
	if override.CreateBootstrap == nil {
		t.Fatal("expected bootstrap override")
	}
	if len(override.CreateBootstrap.CopyFiles) != 1 {
		t.Fatalf("unexpected copyFiles: %+v", override.CreateBootstrap.CopyFiles)
	}
	copyAction := override.CreateBootstrap.CopyFiles[0]
	if copyAction.From != ".env" || copyAction.To != ".env" || !copyAction.Overwrite || !copyAction.Required {
		t.Fatalf("unexpected copy action: %+v", copyAction)
	}
	if len(override.CreateBootstrap.PostCreate) != 1 {
		t.Fatalf("unexpected postCreate: %+v", override.CreateBootstrap.PostCreate)
	}
	hook := override.CreateBootstrap.PostCreate[0]
	if hook.Name != "install dependencies" || strings.Join(hook.Command, " ") != "npm install" || hook.CWD != "frontend" {
		t.Fatalf("unexpected hook: %+v", hook)
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

func TestDefaultConfig_HasNoBootstrapActions(t *testing.T) {
	t.Parallel()

	got := DefaultConfig()
	if len(got.Create.Bootstrap.CopyFiles) != 0 {
		t.Fatalf("unexpected default copyFiles: %+v", got.Create.Bootstrap.CopyFiles)
	}
	if len(got.Create.Bootstrap.PostCreate) != 0 {
		t.Fatalf("unexpected default postCreate: %+v", got.Create.Bootstrap.PostCreate)
	}
}

func TestNormalizeOverride_ReturnsErrorForInvalidBootstrap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  rawConfig
		want string
	}{
		{
			name: "missing copy source",
			raw: rawConfig{Create: &rawCreate{Bootstrap: &rawBootstrap{
				CopyFiles: []rawCopyFileAction{{To: stringPtr(".env")}},
			}}},
			want: "copyFiles[0].from",
		},
		{
			name: "empty copy destination",
			raw: rawConfig{Create: &rawCreate{Bootstrap: &rawBootstrap{
				CopyFiles: []rawCopyFileAction{{From: stringPtr(".env"), To: stringPtr(" ")}},
			}}},
			want: "copyFiles[0].to",
		},
		{
			name: "empty command",
			raw: rawConfig{Create: &rawCreate{Bootstrap: &rawBootstrap{
				PostCreate: []rawHookAction{{Command: []string{}}},
			}}},
			want: "postCreate[0].command",
		},
		{
			name: "empty command argument",
			raw: rawConfig{Create: &rawCreate{Bootstrap: &rawBootstrap{
				PostCreate: []rawHookAction{{Command: []string{"npm", ""}}},
			}}},
			want: "command[1]",
		},
		{
			name: "unsupported placeholder",
			raw: rawConfig{Create: &rawCreate{Bootstrap: &rawBootstrap{
				PostCreate: []rawHookAction{{Command: []string{"echo", "{repoParent}"}}},
			}}},
			want: "placeholder",
		},
		{
			name: "empty cwd",
			raw: rawConfig{Create: &rawCreate{Bootstrap: &rawBootstrap{
				PostCreate: []rawHookAction{{Command: []string{"npm", "install"}, CWD: stringPtr(" ")}},
			}}},
			want: "cwd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := normalizeOverride(tt.raw)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
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

func stringPtr(value string) *string {
	return &value
}
