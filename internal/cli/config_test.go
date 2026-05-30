package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/takubii/worktree-manager/internal/config"
)

func TestConfigInitCommand_InitializesGlobalConfig(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cfgProvider := &fakeConfigProvider{
		cfg:      config.DefaultConfig(),
		initPath: "C:/Users/main/AppData/Roaming/worktree-manager/config.json",
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
		initPath: "C:/Users/main/AppData/Roaming/worktree-manager/config.json",
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
			Create: config.Create{
				Fetch: false,
				Prune: true,
			},
			Remove: config.Remove{
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

func TestConfigPathCommand_PrintsGlobalAndRepoPaths(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	gitClient := &fakeGitClient{
		repoRoot: "C:/repo/project",
	}

	cmd := NewRootCmd(Dependencies{
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
		Git:    gitClient,
		Config: &fakeConfigProvider{cfg: config.DefaultConfig()},
	})
	cmd.SetArgs([]string{"config", "path"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "global: ") {
		t.Fatalf("expected global path output, got: %s", out)
	}
	if !strings.Contains(out, config.AppConfigDirName) || !strings.Contains(out, config.GlobalConfigFileName) {
		t.Fatalf("expected global config path components, got: %s", out)
	}

	expectedRepoPath := filepath.Join("C:/repo/project", config.RepoConfigFileName)
	if !strings.Contains(out, "repo: "+expectedRepoPath) {
		t.Fatalf("expected repo path output %q, got: %s", expectedRepoPath, out)
	}
}

func TestConfigPathCommand_PrintsRepoUnavailableOutsideRepository(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	gitClient := &fakeGitClient{
		repoRootErr: errors.New("not a git repository"),
	}

	cmd := NewRootCmd(Dependencies{
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
		Git:    gitClient,
		Config: &fakeConfigProvider{cfg: config.DefaultConfig()},
	})
	cmd.SetArgs([]string{"config", "path"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "repo: not in a git repository") {
		t.Fatalf("expected repo unavailable message, got: %s", out)
	}
}

func TestConfigPathCommand_JSONOutput(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	gitClient := &fakeGitClient{
		repoRoot: "C:/repo/project",
	}

	cmd := NewRootCmd(Dependencies{
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
		Git:    gitClient,
		Config: &fakeConfigProvider{cfg: config.DefaultConfig()},
	})
	cmd.SetArgs([]string{"config", "path", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() returned error: %v\noutput:\n%s", err, stdout.String())
	}

	globalVal, ok := got["global"].(map[string]any)
	if !ok {
		t.Fatalf("missing global object: %v", got)
	}
	if _, ok := globalVal["path"].(string); !ok {
		t.Fatalf("global.path must be string, got: %v", globalVal["path"])
	}

	repoVal, ok := got["repo"].(map[string]any)
	if !ok {
		t.Fatalf("missing repo object: %v", got)
	}
	if repoVal["path"] != filepath.Join("C:/repo/project", config.RepoConfigFileName) {
		t.Fatalf("unexpected repo.path: %v", repoVal["path"])
	}
	if repoVal["available"] != true {
		t.Fatalf("expected repo.available=true, got: %v", repoVal["available"])
	}
}
