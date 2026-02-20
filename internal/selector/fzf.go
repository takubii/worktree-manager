package selector

import (
	"bytes"
	"context"
	"fmt"
	"strings"
)

func (s *defaultSelector) selectWithFZF(ctx context.Context, prompt string, options []string) (int, error) {
	input := strings.Join(options, "\n")

	cmd := s.execCommand(ctx, "fzf", "--prompt", prompt+" ", "--height", "40%", "--reverse")
	cmd.Stdin = strings.NewReader(input)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrText := strings.TrimSpace(stderr.String())
		if stderrText == "" {
			return -1, fmt.Errorf("worktree selection was canceled")
		}
		return -1, fmt.Errorf("failed to run fzf for selection: %s", stderrText)
	}

	selected := strings.TrimSpace(stdout.String())
	if selected == "" {
		return -1, fmt.Errorf("worktree selection was canceled")
	}

	for i, option := range options {
		if option == selected {
			return i, nil
		}
	}

	return -1, fmt.Errorf("selected option was not found in candidates")
}
