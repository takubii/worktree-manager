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

	if name != "powershell" {
		t.Fatalf("unexpected command name: %q", name)
	}

	expected := []string{
		"-NoProfile",
		"-Command",
		"Start-Process -FilePath explorer.exe -ArgumentList '/n,C:/repo/path'",
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

	if name != "powershell" {
		t.Fatalf("unexpected command name: %q", name)
	}

	expected := []string{
		"-NoProfile",
		"-Command",
		"Start-Process -FilePath explorer.exe -ArgumentList 'C:/repo/path'",
	}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("unexpected args: want=%v got=%v", expected, args)
	}
}

func TestQuoteForPowerShellSingle(t *testing.T) {
	t.Parallel()

	got := quoteForPowerShellSingle("C:/repo/o'neil")
	want := "C:/repo/o''neil"
	if got != want {
		t.Fatalf("unexpected quoted value: want=%q got=%q", want, got)
	}
}
