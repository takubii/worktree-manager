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
	name    string
	args    []string
	cmd     *exec.Cmd
	history []terminalCall
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
		TmuxMode:         TmuxModeOff,
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
		TmuxMode:         TmuxModeOff,
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
		TmuxMode:         TmuxModeOff,
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
		TmuxMode:         TmuxModeOff,
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
		TmuxMode:         TmuxModeOff,
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
		TmuxMode:         TmuxModeOff,
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
		TmuxMode:         TmuxModeOff,
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

func TestOpenWithResult_TerminalTmuxAutoInsideSessionUsesTmuxWindow(t *testing.T) {
	t.Parallel()

	call, opener := newTerminalTestOpenerWithEnv("linux", map[string]bool{
		"tmux": true,
	}, map[string]string{
		"TMUX": ":session,1,0",
	})

	result, err := opener.OpenWithResult(context.Background(), OpenRequest{
		Kind:             KindTerminal,
		Path:             "/repo",
		Window:           WindowNew,
		TerminalProvider: TerminalProviderAuto,
		TmuxMode:         TmuxModeAuto,
	})
	if err != nil {
		t.Fatalf("OpenWithResult() returned error: %v", err)
	}
	if result.Provider != terminalProviderTmux {
		t.Fatalf("unexpected provider: %q", result.Provider)
	}
	if call.name != "tmux" {
		t.Fatalf("unexpected command: %q", call.name)
	}
	want := []string{"new-window", "-c", "/repo"}
	if len(call.args) != len(want) {
		t.Fatalf("unexpected args: %v", call.args)
	}
	for i, expected := range want {
		if call.args[i] != expected {
			t.Fatalf("unexpected arg at %d: want=%q got=%q", i, expected, call.args[i])
		}
	}
}

func TestOpenWithResult_TerminalTmuxSplitOutsideSessionFallsBackWithWarning(t *testing.T) {
	t.Parallel()

	call, opener := newTerminalTestOpenerWithEnv("linux", map[string]bool{
		"xterm": true,
	}, map[string]string{
		"DISPLAY": ":0",
	})

	result, err := opener.OpenWithResult(context.Background(), OpenRequest{
		Kind:             KindTerminal,
		Path:             "/repo",
		Window:           WindowNew,
		TerminalProvider: TerminalProviderAuto,
		TmuxMode:         TmuxModeSplit,
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
	if len(result.Warnings) == 0 || !strings.Contains(result.Warnings[0], "no active tmux session") {
		t.Fatalf("unexpected warnings: %v", result.Warnings)
	}
}

func TestOpenWithResult_TerminalTmuxOffIgnoresTmuxSession(t *testing.T) {
	t.Parallel()

	call, opener := newTerminalTestOpenerWithEnv("linux", map[string]bool{
		"xterm": true,
		"tmux":  true,
	}, map[string]string{
		"DISPLAY": ":0",
		"TMUX":    ":session,1,0",
	})

	result, err := opener.OpenWithResult(context.Background(), OpenRequest{
		Kind:             KindTerminal,
		Path:             "/repo",
		Window:           WindowNew,
		TerminalProvider: TerminalProviderAuto,
		TmuxMode:         TmuxModeOff,
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

func TestOpenWithResult_TerminalAutoWSL2UsesWindowsTerminalBridge(t *testing.T) {
	t.Parallel()

	call, opener := newTerminalTestOpenerWithEnv("linux", map[string]bool{
		"wt.exe":  true,
		"wsl.exe": true,
	}, map[string]string{
		"WSL_INTEROP":     "/run/WSL/123.sock",
		"WSL_DISTRO_NAME": "Ubuntu",
	})

	result, err := opener.OpenWithResult(context.Background(), OpenRequest{
		Kind:             KindTerminal,
		Path:             "/repo/path",
		Window:           WindowNew,
		TerminalProvider: TerminalProviderAuto,
		TmuxMode:         TmuxModeOff,
	})
	if err != nil {
		t.Fatalf("OpenWithResult() returned error: %v", err)
	}
	if result.Provider != TerminalProviderWindowsTerminal {
		t.Fatalf("unexpected provider: %q", result.Provider)
	}
	if call.name != "wt.exe" {
		t.Fatalf("unexpected command: %q", call.name)
	}
	wantPrefix := []string{"wsl.exe", "-d", "Ubuntu", "--cd", "/repo/path"}
	if len(call.args) < len(wantPrefix) {
		t.Fatalf("unexpected args: %v", call.args)
	}
	for i, want := range wantPrefix {
		if call.args[i] != want {
			t.Fatalf("unexpected arg at %d: want=%q got=%q", i, want, call.args[i])
		}
	}
}

func TestOpenWithResult_TerminalAutoLinuxHeadlessReturnsActionableError(t *testing.T) {
	t.Parallel()

	_, opener := newTerminalTestOpenerWithEnv("linux", map[string]bool{
		"xterm": true,
	}, map[string]string{})

	_, err := opener.OpenWithResult(context.Background(), OpenRequest{
		Kind:             KindTerminal,
		Path:             "/repo",
		Window:           WindowNew,
		TerminalProvider: TerminalProviderAuto,
		TmuxMode:         TmuxModeOff,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Linux GUI session") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenWithResult_TerminalAutoWSL2WithoutBridgeAndGUIReturnsActionableError(t *testing.T) {
	t.Parallel()

	_, opener := newTerminalTestOpenerWithEnv("linux", map[string]bool{}, map[string]string{
		"WSL_INTEROP": "/run/WSL/123.sock",
	})

	_, err := opener.OpenWithResult(context.Background(), OpenRequest{
		Kind:             KindTerminal,
		Path:             "/repo",
		Window:           WindowNew,
		TerminalProvider: TerminalProviderAuto,
		TmuxMode:         TmuxModeOff,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "WSL2") || !strings.Contains(err.Error(), "wt.exe") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func newTerminalTestOpener(goos string, commandExists map[string]bool) (*terminalCall, *defaultOpener) {
	env := map[string]string{}
	if goos == "linux" {
		env["DISPLAY"] = ":0"
	}
	return newTerminalTestOpenerWithEnv(goos, commandExists, env)
}

func newTerminalTestOpenerWithEnv(goos string, commandExists map[string]bool, env map[string]string) (*terminalCall, *defaultOpener) {
	call := &terminalCall{}
	o := &defaultOpener{
		goos: goos,
		getEnv: func(key string) string {
			if value, ok := env[key]; ok {
				return value
			}
			return ""
		},
		lookPath: func(file string) (string, error) {
			if commandExists[file] {
				return file, nil
			}
			return "", errors.New("not found")
		},
		execCommand: func(ctx context.Context, gotName string, gotArgs ...string) *exec.Cmd {
			call.name = gotName
			call.args = append([]string(nil), gotArgs...)
			call.history = append(call.history, terminalCall{
				name: gotName,
				args: append([]string(nil), gotArgs...),
			})

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
