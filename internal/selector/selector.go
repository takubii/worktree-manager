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

// SelectOrCreateResult is the outcome of selecting an existing option or entering a new value.
type SelectOrCreateResult struct {
	Value string
	IsNew bool
}

// SelectOrCreator chooses one option or returns a newly entered value.
type SelectOrCreator interface {
	SelectOrCreate(ctx context.Context, prompt string, options []string) (SelectOrCreateResult, error)
}

type defaultSelector struct {
	in                        io.Reader
	out                       io.Writer
	lookPath                  lookPathFunc
	execCommand               execCommandFunc
	usePromptUI               func(prompt string, options []string) (int, error)
	usePromptUISelectOrCreate func(prompt string, options []string) (SelectOrCreateResult, error)
	useFZFSelectOrCreate      func(ctx context.Context, prompt string, options []string) (SelectOrCreateResult, error)
	isInteractive             func() bool
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
	s.usePromptUISelectOrCreate = s.selectOrCreateWithPromptUI
	s.useFZFSelectOrCreate = s.selectOrCreateWithFZF
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

func (s *defaultSelector) SelectOrCreate(ctx context.Context, prompt string, options []string) (SelectOrCreateResult, error) {
	if len(options) == 0 {
		return SelectOrCreateResult{}, fmt.Errorf("no options are available to select or create")
	}

	if _, err := s.lookPath("fzf"); err == nil {
		if s.useFZFSelectOrCreate == nil {
			return SelectOrCreateResult{}, fmt.Errorf("fzf selector is not configured")
		}
		return s.useFZFSelectOrCreate(ctx, prompt, options)
	}

	if s.isInteractive != nil && s.isInteractive() {
		if s.usePromptUISelectOrCreate == nil {
			return SelectOrCreateResult{}, fmt.Errorf("promptui selector is not configured")
		}

		result, err := s.usePromptUISelectOrCreate(prompt, options)
		if err == nil {
			return result, nil
		}
		if errors.Is(err, errSelectionCanceled) {
			return SelectOrCreateResult{}, err
		}
	}

	if len(options) == 1 {
		return SelectOrCreateResult{
			Value: options[0],
			IsNew: false,
		}, nil
	}

	index, err := s.selectWithNumberedFallback(prompt, options)
	if err != nil {
		return SelectOrCreateResult{}, err
	}
	if index < 0 || index >= len(options) {
		return SelectOrCreateResult{}, fmt.Errorf("selected option index is out of range: %d", index)
	}

	return SelectOrCreateResult{
		Value: options[index],
		IsNew: false,
	}, nil
}
