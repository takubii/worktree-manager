package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewRootCmd_HelpIncludesListCommand(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := NewRootCmd(Dependencies{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	helpText := stdout.String()
	if !strings.Contains(helpText, "list") {
		t.Fatalf("help text does not include list command:\n%s", helpText)
	}
	if !strings.Contains(helpText, "create") {
		t.Fatalf("help text does not include create command:\n%s", helpText)
	}
	if !strings.Contains(helpText, "path") {
		t.Fatalf("help text does not include path command:\n%s", helpText)
	}
	if !strings.Contains(helpText, "remove") {
		t.Fatalf("help text does not include remove command:\n%s", helpText)
	}
	if !strings.Contains(helpText, "config") {
		t.Fatalf("help text does not include config command:\n%s", helpText)
	}
	if !strings.Contains(helpText, "update") {
		t.Fatalf("help text does not include update command:\n%s", helpText)
	}
	if !strings.Contains(helpText, "version") {
		t.Fatalf("help text does not include version command:\n%s", helpText)
	}
	if !strings.Contains(helpText, "doctor") {
		t.Fatalf("help text does not include doctor command:\n%s", helpText)
	}
}
