package updater

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

func TestInstallerUpdateUnixRunsInstallScript(t *testing.T) {
	t.Parallel()

	var gotName string
	var gotArgs []string
	runCalls := 0
	startCalls := 0

	installer := &Installer{
		goos: "linux",
		commandContext: func(_ context.Context, name string, args ...string) *exec.Cmd {
			gotName = name
			gotArgs = append([]string(nil), args...)
			return &exec.Cmd{}
		},
		runCommand: func(_ *exec.Cmd) error {
			runCalls++
			return nil
		},
		startCommand: func(_ *exec.Cmd) error {
			startCalls++
			return nil
		},
	}

	result, err := installer.Update(context.Background(), Request{})
	if err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}

	if result.Async {
		t.Fatalf("expected Async=false, got true")
	}
	if runCalls != 1 {
		t.Fatalf("expected runCommand to be called once, got %d", runCalls)
	}
	if startCalls != 0 {
		t.Fatalf("expected startCommand not to be called, got %d", startCalls)
	}
	if gotName != "sh" {
		t.Fatalf("expected command `sh`, got %q", gotName)
	}
	if len(gotArgs) != 2 || gotArgs[0] != "-c" {
		t.Fatalf("unexpected args: %v", gotArgs)
	}
	if !strings.Contains(gotArgs[1], installScriptURLUnix) {
		t.Fatalf("script does not include install URL: %q", gotArgs[1])
	}
	if strings.Contains(gotArgs[1], "WTO_VERSION=") {
		t.Fatalf("latest update should not include explicit WTO_VERSION: %q", gotArgs[1])
	}
}

func TestInstallerUpdateUnixRunsInstallScriptWithVersion(t *testing.T) {
	t.Parallel()

	var gotScript string
	installer := &Installer{
		goos: "darwin",
		commandContext: func(_ context.Context, _ string, args ...string) *exec.Cmd {
			if len(args) == 2 {
				gotScript = args[1]
			}
			return &exec.Cmd{}
		},
		runCommand: func(_ *exec.Cmd) error {
			return nil
		},
		startCommand: func(_ *exec.Cmd) error {
			return nil
		},
	}

	_, err := installer.Update(context.Background(), Request{
		Version: "v0.1.0",
	})
	if err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}

	if !strings.Contains(gotScript, "WTO_VERSION='v0.1.0'") {
		t.Fatalf("expected version to be included in shell script, got %q", gotScript)
	}
}

func TestInstallerUpdateWindowsStartsBackgroundInstaller(t *testing.T) {
	t.Parallel()

	var gotName string
	var gotArgs []string
	runCalls := 0
	startCalls := 0

	installer := &Installer{
		goos: "windows",
		commandContext: func(_ context.Context, name string, args ...string) *exec.Cmd {
			gotName = name
			gotArgs = append([]string(nil), args...)
			return &exec.Cmd{}
		},
		runCommand: func(_ *exec.Cmd) error {
			runCalls++
			return nil
		},
		startCommand: func(_ *exec.Cmd) error {
			startCalls++
			return nil
		},
	}

	result, err := installer.Update(context.Background(), Request{
		Version: "v0.1.0",
	})
	if err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}

	if !result.Async {
		t.Fatalf("expected Async=true, got false")
	}
	if startCalls != 1 {
		t.Fatalf("expected startCommand to be called once, got %d", startCalls)
	}
	if runCalls != 0 {
		t.Fatalf("expected runCommand not to be called, got %d", runCalls)
	}
	if gotName != "powershell" {
		t.Fatalf("expected command `powershell`, got %q", gotName)
	}
	if len(gotArgs) < 5 {
		t.Fatalf("unexpected args: %v", gotArgs)
	}
	commandScript := gotArgs[len(gotArgs)-1]
	if !strings.Contains(commandScript, "Start-Sleep -Milliseconds 300") {
		t.Fatalf("expected delay in script, got %q", commandScript)
	}
	if !strings.Contains(commandScript, "$env:WTO_VERSION = 'v0.1.0'") {
		t.Fatalf("expected version env in script, got %q", commandScript)
	}
	if !strings.Contains(commandScript, installScriptURLWindows) {
		t.Fatalf("expected install URL in script, got %q", commandScript)
	}
}
