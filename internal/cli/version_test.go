package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommand_PrintsVersion(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	cmd := NewRootCmd(Dependencies{
		Stdout:  &stdout,
		Stderr:  &bytes.Buffer{},
		Version: "v0.2.0-test",
	})
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if got := strings.TrimSpace(stdout.String()); got != "v0.2.0-test" {
		t.Fatalf("unexpected version output: %q", got)
	}
}

func TestRootVersionFlag_PrintsVersion(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	cmd := NewRootCmd(Dependencies{
		Stdout:  &stdout,
		Stderr:  &bytes.Buffer{},
		Version: "v0.2.0-test",
	})
	cmd.SetArgs([]string{"--version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if got := strings.TrimSpace(stdout.String()); got != "v0.2.0-test" {
		t.Fatalf("unexpected version output: %q", got)
	}
}
