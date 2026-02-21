package selector

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
)

func (s *defaultSelector) selectWithPromptUI(prompt string, options []string) (int, error) {
	selector := promptui.Select{
		Label:    prompt,
		Items:    options,
		Size:     min(10, len(options)),
		Searcher: newOptionSearcher(options),
	}

	index, _, err := selector.Run()
	if err != nil {
		switch {
		case errors.Is(err, promptui.ErrInterrupt), errors.Is(err, promptui.ErrEOF):
			return -1, errSelectionCanceled
		default:
			return -1, fmt.Errorf("%w: %v", errPromptUIUnavailable, err)
		}
	}

	return index, nil
}

func (s *defaultSelector) selectOrCreateWithPromptUI(prompt string, options []string) (SelectOrCreateResult, error) {
	selector := promptui.SelectWithAdd{
		Label:    prompt,
		Items:    options,
		AddLabel: "Create a new branch",
	}

	index, value, err := selector.Run()
	if err != nil {
		switch {
		case errors.Is(err, promptui.ErrInterrupt), errors.Is(err, promptui.ErrEOF):
			return SelectOrCreateResult{}, errSelectionCanceled
		default:
			return SelectOrCreateResult{}, fmt.Errorf("%w: %v", errPromptUIUnavailable, err)
		}
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return SelectOrCreateResult{}, fmt.Errorf("branch name is empty. Enter a branch name and retry")
	}

	return SelectOrCreateResult{
		Value: value,
		IsNew: index == promptui.SelectedAdd,
	}, nil
}

func newOptionSearcher(options []string) func(string, int) bool {
	return func(input string, index int) bool {
		if index < 0 || index >= len(options) {
			return false
		}

		needle := strings.ToLower(strings.TrimSpace(input))
		if needle == "" {
			return true
		}

		haystack := strings.ToLower(options[index])
		return strings.Contains(haystack, needle)
	}
}

func (s *defaultSelector) defaultIsInteractive() bool {
	return isTerminalInput(s.in) && isTerminalOutput(s.out)
}

func isTerminalInput(input io.Reader) bool {
	file, ok := input.(*os.File)
	if !ok {
		return false
	}

	info, err := file.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}

func isTerminalOutput(output io.Writer) bool {
	file, ok := output.(*os.File)
	if !ok {
		return false
	}

	info, err := file.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}
