package opener

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"reflect"
	"testing"
)

func TestParseWindowMode(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   string
		want    WindowMode
		wantErr bool
	}{
		{name: "default-empty", input: "", want: WindowNew},
		{name: "new", input: "new", want: WindowNew},
		{name: "reuse", input: "reuse", want: WindowReuse},
		{name: "uppercase", input: "NEW", want: WindowNew},
		{name: "invalid", input: "other", wantErr: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseWindowMode(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseWindowMode() returned error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("unexpected mode: want=%q got=%q", tc.want, got)
			}
		})
	}
}

func TestParseTmuxMode(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   string
		want    TmuxMode
		wantErr bool
	}{
		{name: "default-empty", input: "", want: TmuxModeAuto},
		{name: "auto", input: "auto", want: TmuxModeAuto},
		{name: "off", input: "off", want: TmuxModeOff},
		{name: "split", input: "split", want: TmuxModeSplit},
		{name: "window", input: "window", want: TmuxModeWindow},
		{name: "uppercase", input: "SPLIT", want: TmuxModeSplit},
		{name: "invalid", input: "other", wantErr: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseTmuxMode(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseTmuxMode() returned error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("unexpected mode: want=%q got=%q", tc.want, got)
			}
		})
	}
}

func TestOpenVSCode_UsesNewWindowFlag(t *testing.T) {
	t.Parallel()

	gotName, gotArgs, opener := newTestOpener("windows", map[string]bool{"code": true})
	if err := opener.openVSCode(context.Background(), "C:/repo", WindowNew); err != nil {
		t.Fatalf("openVSCode() returned error: %v", err)
	}

	if gotName() != "code" {
		t.Fatalf("unexpected command name: %q", gotName())
	}
	expected := []string{"--new-window", "C:/repo"}
	if !reflect.DeepEqual(gotArgs(), expected) {
		t.Fatalf("unexpected args: want=%v got=%v", expected, gotArgs())
	}
}

func TestOpenVSCode_UsesReuseWindowFlag(t *testing.T) {
	t.Parallel()

	gotName, gotArgs, opener := newTestOpener("windows", map[string]bool{"code": true})
	if err := opener.openVSCode(context.Background(), "C:/repo", WindowReuse); err != nil {
		t.Fatalf("openVSCode() returned error: %v", err)
	}

	if gotName() != "code" {
		t.Fatalf("unexpected command name: %q", gotName())
	}
	expected := []string{"--reuse-window", "C:/repo"}
	if !reflect.DeepEqual(gotArgs(), expected) {
		t.Fatalf("unexpected args: want=%v got=%v", expected, gotArgs())
	}
}

func TestOpenCursor_UsesNewWindowFlag(t *testing.T) {
	t.Parallel()

	gotName, gotArgs, opener := newTestOpener("windows", map[string]bool{"cursor": true})
	if err := opener.openCursor(context.Background(), "C:/repo", WindowNew); err != nil {
		t.Fatalf("openCursor() returned error: %v", err)
	}

	if gotName() != "cursor" {
		t.Fatalf("unexpected command name: %q", gotName())
	}
	expected := []string{"--new-window", "C:/repo"}
	if !reflect.DeepEqual(gotArgs(), expected) {
		t.Fatalf("unexpected args: want=%v got=%v", expected, gotArgs())
	}
}

func TestOpenCursor_UsesReuseWindowFlag(t *testing.T) {
	t.Parallel()

	gotName, gotArgs, opener := newTestOpener("windows", map[string]bool{"cursor": true})
	if err := opener.openCursor(context.Background(), "C:/repo", WindowReuse); err != nil {
		t.Fatalf("openCursor() returned error: %v", err)
	}

	if gotName() != "cursor" {
		t.Fatalf("unexpected command name: %q", gotName())
	}
	expected := []string{"--reuse-window", "C:/repo"}
	if !reflect.DeepEqual(gotArgs(), expected) {
		t.Fatalf("unexpected args: want=%v got=%v", expected, gotArgs())
	}
}

func newTestOpener(goos string, commandExists map[string]bool) (func() string, func() []string, *defaultOpener) {
	var name string
	var args []string

	o := &defaultOpener{
		goos: goos,
		lookPath: func(file string) (string, error) {
			if commandExists[file] {
				return file, nil
			}
			return "", errors.New("not found")
		},
		execCommand: func(ctx context.Context, gotName string, gotArgs ...string) *exec.Cmd {
			name = gotName
			args = append([]string(nil), gotArgs...)

			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestOpenerHelperProcess", "--")
			cmd.Env = append(os.Environ(), "GO_WANT_OPENER_HELPER_PROCESS=1")
			return cmd
		},
	}

	return func() string { return name }, func() []string { return args }, o
}

func TestOpenerHelperProcess(t *testing.T) {
	t.Helper()

	if os.Getenv("GO_WANT_OPENER_HELPER_PROCESS") != "1" {
		return
	}

	os.Exit(0)
}
