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
	if !strings.Contains(helpText, "open") {
		t.Fatalf("help text does not include open command:\n%s", helpText)
	}
	if !strings.Contains(helpText, "new") {
		t.Fatalf("help text does not include new command:\n%s", helpText)
	}
	if !strings.Contains(helpText, "rm") {
		t.Fatalf("help text does not include rm command:\n%s", helpText)
	}
}
