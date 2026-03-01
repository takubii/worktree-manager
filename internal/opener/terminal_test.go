package opener

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

type terminalCall struct {
	name string
	args []string
	cmd  *exec.Cmd
}

func TestOpenWithResult_TerminalAutoWindowsPrefersWT(t *testing.T) {
	t.Parallel()

	call, opener := newTerminalTestOpener("windows", map[string]bool{
		"wt":         true,
		"cmd":        true,
		"powershell": true,
	})

	result, err := opener.OpenWithResult(context.Background(), OpenRequest{
		Kind:             KindTerminal,
		Path:             "C:/repo",
		Window:           WindowNew,
		TerminalProvider: TerminalProviderAuto,
	})
	if err != nil {
		t.Fatalf("OpenWithResult() returned error: %v", err)
	}
	if result.Provider != TerminalProviderWindowsTerminal {
		t.Fatalf("unexpected provider: %q", result.Provider)
	}
	if call.name != "wt" {
		t.Fatalf("unexpected command: %q", call.name)
	}
}

func TestOpenWithResult_TerminalAutoWindowsFallsBackToCMD(t *testing.T) {
	t.Parallel()

	call, opener := newTerminalTestOpener("windows", map[string]bool{
		"cmd":        true,
		"powershell": true,
	})

	result, err := opener.OpenWithResult(context.Background(), OpenRequest{
		Kind:             KindTerminal,
		Path:             "C:/repo",
		Window:           WindowNew,
		TerminalProvider: TerminalProviderAuto,
	})
	if err != nil {
		t.Fatalf("OpenWithResult() returned error: %v", err)
	}
	if result.Provider != TerminalProviderCMD {
		t.Fatalf("unexpected provider: %q", result.Provider)
	}
	if call.name != "cmd" {
		t.Fatalf("unexpected command: %q", call.name)
	}
}

func TestOpenWithResult_TerminalAutoLinuxFallsBackToXTerm(t *testing.T) {
	t.Parallel()

	call, opener := newTerminalTestOpener("linux", map[string]bool{
		"xterm": true,
	})

	result, err := opener.OpenWithResult(context.Background(), OpenRequest{
		Kind:             KindTerminal,
		Path:             "/repo",
		Window:           WindowNew,
		TerminalProvider: TerminalProviderAuto,
	})
	if err != nil {
		t.Fatalf("OpenWithResult() returned error: %v", err)
	}
	if result.Provider != terminalProviderXTerm {
		t.Fatalf("unexpected provider: %q", result.Provider)
	}
	if call.name != "xterm" {
		t.Fatalf("unexpected command: %q", call.name)
	}
}

func TestOpenWithResult_TerminalExplicitMissingReturnsError(t *testing.T) {
	t.Parallel()

	_, opener := newTerminalTestOpener("linux", map[string]bool{})
	_, err := opener.OpenWithResult(context.Background(), OpenRequest{
		Kind:             KindTerminal,
		Path:             "/repo",
		Window:           WindowNew,
		TerminalProvider: TerminalProviderWarp,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "terminal provider") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenWithResult_TerminalReuseReturnsWarningForWarp(t *testing.T) {
	t.Parallel()

	call, opener := newTerminalTestOpener("linux", map[string]bool{
		"warp": true,
	})
	workingDir := t.TempDir()

	result, err := opener.OpenWithResult(context.Background(), OpenRequest{
		Kind:             KindTerminal,
		Path:             workingDir,
		Window:           WindowReuse,
		TerminalProvider: TerminalProviderWarp,
	})
	if err != nil {
		t.Fatalf("OpenWithResult() returned error: %v", err)
	}
	if call.name != "warp" {
		t.Fatalf("unexpected command: %q", call.name)
	}
	if call.cmd == nil {
		t.Fatal("expected command to be captured")
	}
	if call.cmd.Dir != workingDir {
		t.Fatalf("unexpected dir: %q", call.cmd.Dir)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected warning for reuse mode")
	}
}

func TestOpenWithResult_TerminalUnknownProviderReturnsError(t *testing.T) {
	t.Parallel()

	_, opener := newTerminalTestOpener("linux", map[string]bool{})
	_, err := opener.OpenWithResult(context.Background(), OpenRequest{
		Kind:             KindTerminal,
		Path:             "/repo",
		Window:           WindowNew,
		TerminalProvider: "unknown-provider",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unknown terminal provider") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenWithResult_TerminalPowerShellStartsNewWindowViaCmd(t *testing.T) {
	t.Parallel()

	call, opener := newTerminalTestOpener("windows", map[string]bool{
		"cmd":        true,
		"powershell": true,
	})

	result, err := opener.OpenWithResult(context.Background(), OpenRequest{
		Kind:             KindTerminal,
		Path:             `C:/repo/path`,
		Window:           WindowNew,
		TerminalProvider: TerminalProviderPowerShell,
	})
	if err != nil {
		t.Fatalf("OpenWithResult() returned error: %v", err)
	}
	if result.Provider != TerminalProviderPowerShell {
		t.Fatalf("unexpected provider: %q", result.Provider)
	}
	if call.name != "cmd" {
		t.Fatalf("unexpected command: %q", call.name)
	}
	wantPrefix := []string{"/c", "start", "", "powershell", "-NoExit", "-Command"}
	if len(call.args) < len(wantPrefix) {
		t.Fatalf("unexpected args: %v", call.args)
	}
	for i, want := range wantPrefix {
		if call.args[i] != want {
			t.Fatalf("unexpected arg at %d: want=%q got=%q", i, want, call.args[i])
		}
	}
}

func newTerminalTestOpener(goos string, commandExists map[string]bool) (*terminalCall, *defaultOpener) {
	call := &terminalCall{}
	o := &defaultOpener{
		goos: goos,
		lookPath: func(file string) (string, error) {
			if commandExists[file] {
				return file, nil
			}
			return "", errors.New("not found")
		},
		execCommand: func(ctx context.Context, gotName string, gotArgs ...string) *exec.Cmd {
			call.name = gotName
			call.args = append([]string(nil), gotArgs...)

			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestTerminalOpenerHelperProcess", "--")
			cmd.Env = append(os.Environ(), "GO_WANT_TERMINAL_HELPER_PROCESS=1")
			call.cmd = cmd
			return cmd
		},
	}

	return call, o
}

func TestTerminalOpenerHelperProcess(t *testing.T) {
	t.Helper()

	if os.Getenv("GO_WANT_TERMINAL_HELPER_PROCESS") != "1" {
		return
	}

	os.Exit(0)
}
