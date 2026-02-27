//go:build windows

package opener

import (
	"reflect"
	"testing"
)

func TestSystemOpenCommandWindows_NewWindow(t *testing.T) {
	t.Parallel()

	name, args, err := systemOpenCommand("C:/repo/path", WindowNew)
	if err != nil {
		t.Fatalf("systemOpenCommand() returned error: %v", err)
	}

	if name != "cmd" {
		t.Fatalf("unexpected command name: %q", name)
	}

	expected := []string{
		"/c",
		"start",
		"",
		"explorer.exe",
		"/n,C:/repo/path",
	}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("unexpected args: want=%v got=%v", expected, args)
	}
}

func TestSystemOpenCommandWindows_ReuseWindow(t *testing.T) {
	t.Parallel()

	name, args, err := systemOpenCommand("C:/repo/path", WindowReuse)
	if err != nil {
		t.Fatalf("systemOpenCommand() returned error: %v", err)
	}

	if name != "cmd" {
		t.Fatalf("unexpected command name: %q", name)
	}

	expected := []string{
		"/c",
		"start",
		"",
		"explorer.exe",
		"C:/repo/path",
	}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("unexpected args: want=%v got=%v", expected, args)
	}
}
