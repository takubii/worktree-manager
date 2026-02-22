package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/takubii/git-worktree-opener/internal/config"
)

func TestConfigInitCommand_InitializesGlobalConfig(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cfgProvider := &fakeConfigProvider{
		cfg:      config.DefaultConfig(),
		initPath: "C:/Users/main/AppData/Roaming/git-worktree-opener/config.json",
	}

	cmd := NewRootCmd(Dependencies{
		Stdout: &stdout,
		Stderr: &stderr,
		Config: cfgProvider,
	})
	cmd.SetArgs([]string{"config", "init"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if len(cfgProvider.initForce) != 1 {
		t.Fatalf("expected one init call, got %d", len(cfgProvider.initForce))
	}
	if cfgProvider.initForce[0] {
		t.Fatal("expected force=false by default")
	}
	if !strings.Contains(stdout.String(), "initialized config file") {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestConfigInitCommand_PassesForceFlag(t *testing.T) {
	t.Parallel()

	cfgProvider := &fakeConfigProvider{
		cfg:      config.DefaultConfig(),
		initPath: "C:/Users/main/AppData/Roaming/git-worktree-opener/config.json",
	}

	cmd := NewRootCmd(Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Config: cfgProvider,
	})
	cmd.SetArgs([]string{"config", "init", "--force"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if len(cfgProvider.initForce) != 1 || !cfgProvider.initForce[0] {
		t.Fatalf("expected force=true init call, got %+v", cfgProvider.initForce)
	}
}

func TestConfigShowCommand_PrintsEffectiveConfigJSON(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	cfgProvider := &fakeConfigProvider{
		cfg: config.Config{
			Remote:              "upstream",
			BaseBranch:          "develop",
			WorktreeDirTemplate: "{repoRoot}/../worktrees/{branch}",
			Open: config.Open{
				Default: "vscode",
				Window:  "reuse",
			},
			RM: config.RM{
				DeleteBranch: "force",
			},
		},
	}

	cmd := NewRootCmd(Dependencies{
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
		Config: cfgProvider,
	})
	cmd.SetArgs([]string{"config", "show"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, `"remote": "upstream"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if !strings.Contains(out, `"deleteBranch": "force"`) {
		t.Fatalf("unexpected output: %s", out)
	}
	if cfgProvider.loadCalls != 1 {
		t.Fatalf("expected one config load, got %d", cfgProvider.loadCalls)
	}
}
