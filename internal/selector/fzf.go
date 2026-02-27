package selector

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/takubii/git-worktree-opener/internal/execerr"
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
		return -1, execerr.Build("fzf", stderrText, "confirm `fzf --version` works in this shell, then retry")
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

func (s *defaultSelector) selectOrCreateWithFZF(ctx context.Context, prompt string, options []string) (SelectOrCreateResult, error) {
	input := strings.Join(options, "\n")

	cmd := s.execCommand(
		ctx,
		"fzf",
		"--prompt", prompt+" ",
		"--height", "40%",
		"--reverse",
		"--print-query",
	)
	cmd.Stdin = strings.NewReader(input)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	query, selected := parseFZFQueryAndSelection(stdout.String())
	if runErr != nil && query == "" && selected == "" {
		stderrText := strings.TrimSpace(stderr.String())
		if stderrText == "" {
			return SelectOrCreateResult{}, errSelectionCanceled
		}
		return SelectOrCreateResult{}, execerr.Build("fzf", stderrText, "confirm `fzf --version` works in this shell, then retry")
	}

	if selected != "" {
		for _, option := range options {
			if option == selected {
				return SelectOrCreateResult{
					Value: selected,
					IsNew: false,
				}, nil
			}
		}
		return SelectOrCreateResult{}, fmt.Errorf("selected option was not found in candidates")
	}

	if query != "" {
		return SelectOrCreateResult{
			Value: query,
			IsNew: true,
		}, nil
	}

	return SelectOrCreateResult{}, errSelectionCanceled
}

func parseFZFQueryAndSelection(output string) (string, string) {
	output = strings.ReplaceAll(output, "\r\n", "\n")
	output = strings.TrimRight(output, "\n")
	if output == "" {
		return "", ""
	}

	lines := strings.Split(output, "\n")
	query := strings.TrimSpace(lines[0])
	if len(lines) == 1 {
		return query, ""
	}

	selected := strings.TrimSpace(lines[len(lines)-1])
	return query, selected
}
