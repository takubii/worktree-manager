package selector

import (
	"context"
	"fmt"
	"io"
	"os/exec"
)

type execCommandFunc func(ctx context.Context, name string, args ...string) *exec.Cmd
type lookPathFunc func(file string) (string, error)

// Selector chooses one option from a list.
type Selector interface {
	Select(ctx context.Context, prompt string, options []string) (int, error)
}

type defaultSelector struct {
	in          io.Reader
	out         io.Writer
	lookPath    lookPathFunc
	execCommand execCommandFunc
}

// NewDefault returns the default selector implementation.
func NewDefault(in io.Reader, out io.Writer) Selector {
	return &defaultSelector{
		in:          in,
		out:         out,
		lookPath:    exec.LookPath,
		execCommand: exec.CommandContext,
	}
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

	return s.selectWithNumberedFallback(prompt, options)
}
