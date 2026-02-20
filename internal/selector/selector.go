package selector

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
)

type execCommandFunc func(ctx context.Context, name string, args ...string) *exec.Cmd
type lookPathFunc func(file string) (string, error)

var errPromptUIUnavailable = errors.New("promptui is unavailable in current environment")
var errSelectionCanceled = errors.New("worktree selection was canceled")

// Selector chooses one option from a list.
type Selector interface {
	Select(ctx context.Context, prompt string, options []string) (int, error)
}

type defaultSelector struct {
	in            io.Reader
	out           io.Writer
	lookPath      lookPathFunc
	execCommand   execCommandFunc
	usePromptUI   func(prompt string, options []string) (int, error)
	isInteractive func() bool
}

// NewDefault returns the default selector implementation.
func NewDefault(in io.Reader, out io.Writer) Selector {
	s := &defaultSelector{
		in:          in,
		out:         out,
		lookPath:    exec.LookPath,
		execCommand: exec.CommandContext,
	}
	s.usePromptUI = s.selectWithPromptUI
	s.isInteractive = s.defaultIsInteractive
	return s
}

func (s *defaultSelector) Select(ctx context.Context, prompt string, options []string) (int, error) {
	switch len(options) {
	case 0:
		return -1, fmt.Errorf("no options are available to select")
	case 1:
		return 0, nil
	}

	if _, err := s.lookPath("fzf"); err == nil {
		return s.selectWithFZF(ctx, prompt, options)
	}

	if s.isInteractive != nil && s.isInteractive() {
		index, err := s.usePromptUI(prompt, options)
		if err == nil {
			return index, nil
		}
		if errors.Is(err, errSelectionCanceled) {
			return -1, err
		}
	}

	return s.selectWithNumberedFallback(prompt, options)
}
