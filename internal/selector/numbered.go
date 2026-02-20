package selector

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func (s *defaultSelector) selectWithNumberedFallback(prompt string, options []string) (int, error) {
	reader := bufio.NewReader(s.in)

	fmt.Fprintln(s.out, prompt)
	for i, option := range options {
		fmt.Fprintf(s.out, "%d) %s\n", i+1, option)
	}

	for attempt := 0; attempt < 3; attempt++ {
		fmt.Fprintf(s.out, "Enter number [1-%d]: ", len(options))

		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return -1, fmt.Errorf("failed to read selection input: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			if err == io.EOF {
				return -1, fmt.Errorf("no selection was provided. Please rerun and enter a number")
			}
			fmt.Fprintln(s.out, "Please enter a number.")
			continue
		}

		selected, parseErr := strconv.Atoi(line)
		if parseErr != nil {
			fmt.Fprintln(s.out, "Invalid input. Enter a numeric value.")
			continue
		}

		index := selected - 1
		if index < 0 || index >= len(options) {
			fmt.Fprintf(s.out, "Out of range. Choose a number between 1 and %d.\n", len(options))
			continue
		}

		return index, nil
	}

	return -1, fmt.Errorf("failed to select a worktree after multiple attempts")
}
