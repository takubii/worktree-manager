package selector

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

func TestSelect_UsesFZFWhenAvailable(t *testing.T) {
	t.Parallel()

	s := &defaultSelector{
		lookPath: func(file string) (string, error) {
			if file == "fzf" {
				return "fzf", nil
			}
			return "", errors.New("not found")
		},
		execCommand: func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestSelectorHelperProcess", "--")
			cmd.Env = append(
				os.Environ(),
				"GO_WANT_SELECTOR_HELPER_PROCESS=1",
				"HELPER_STDOUT=second",
			)
			return cmd
		},
	}

	index, err := s.Select(context.Background(), "pick", []string{"first", "second"})
	if err != nil {
		t.Fatalf("Select() returned error: %v", err)
	}
	if index != 1 {
		t.Fatalf("unexpected index: want=1 got=%d", index)
	}
}

func TestSelectOrCreate_UsesFZFWhenAvailable(t *testing.T) {
	t.Parallel()

	var gotArgs []string
	s := &defaultSelector{
		lookPath: func(file string) (string, error) {
			if file == "fzf" {
				return "fzf", nil
			}
			return "", errors.New("not found")
		},
		execCommand: func(ctx context.Context, _ string, args ...string) *exec.Cmd {
			gotArgs = append([]string(nil), args...)

			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestSelectorHelperProcess", "--")
			cmd.Env = append(
				os.Environ(),
				"GO_WANT_SELECTOR_HELPER_PROCESS=1",
				"HELPER_STDOUT=\nsecond\n",
			)
			return cmd
		},
	}
	s.useFZFSelectOrCreate = s.selectOrCreateWithFZF

	result, err := s.SelectOrCreate(context.Background(), "pick", []string{"first", "second"})
	if err != nil {
		t.Fatalf("SelectOrCreate() returned error: %v", err)
	}
	if result.IsNew {
		t.Fatalf("expected existing selection, got new value: %+v", result)
	}
	if result.Value != "second" {
		t.Fatalf("unexpected selected value: %q", result.Value)
	}

	argsText := strings.Join(gotArgs, " ")
	if !strings.Contains(argsText, "--print-query") {
		t.Fatalf("fzf args must contain --print-query: %v", gotArgs)
	}
}

func TestSelectOrCreate_UsesFZFQueryForNewValue(t *testing.T) {
	t.Parallel()

	s := &defaultSelector{
		lookPath: func(file string) (string, error) {
			if file == "fzf" {
				return "fzf", nil
			}
			return "", errors.New("not found")
		},
		execCommand: func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestSelectorHelperProcess", "--")
			cmd.Env = append(
				os.Environ(),
				"GO_WANT_SELECTOR_HELPER_PROCESS=1",
				"HELPER_STDOUT=feature/new-branch\n",
			)
			return cmd
		},
	}
	s.useFZFSelectOrCreate = s.selectOrCreateWithFZF

	result, err := s.SelectOrCreate(context.Background(), "pick", []string{"first", "second"})
	if err != nil {
		t.Fatalf("SelectOrCreate() returned error: %v", err)
	}
	if !result.IsNew {
		t.Fatalf("expected new value result, got existing: %+v", result)
	}
	if result.Value != "feature/new-branch" {
		t.Fatalf("unexpected new value: %q", result.Value)
	}
}

func TestSelectOrCreate_UsesFZFQueryForNewValueWhenExitCodeIsOne(t *testing.T) {
	t.Parallel()

	s := &defaultSelector{
		lookPath: func(file string) (string, error) {
			if file == "fzf" {
				return "fzf", nil
			}
			return "", errors.New("not found")
		},
		execCommand: func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestSelectorHelperProcess", "--")
			cmd.Env = append(
				os.Environ(),
				"GO_WANT_SELECTOR_HELPER_PROCESS=1",
				"HELPER_STDOUT=feature/new-branch\n",
				"HELPER_EXIT_CODE=1",
			)
			return cmd
		},
	}
	s.useFZFSelectOrCreate = s.selectOrCreateWithFZF

	result, err := s.SelectOrCreate(context.Background(), "pick", []string{"first", "second"})
	if err != nil {
		t.Fatalf("SelectOrCreate() returned error: %v", err)
	}
	if !result.IsNew {
		t.Fatalf("expected new value result, got existing: %+v", result)
	}
	if result.Value != "feature/new-branch" {
		t.Fatalf("unexpected new value: %q", result.Value)
	}
}

func TestSelect_UsesPromptUIWhenFZFIsUnavailable(t *testing.T) {
	t.Parallel()

	s := &defaultSelector{
		in:  bytes.NewBuffer(nil),
		out: &bytes.Buffer{},
		lookPath: func(string) (string, error) {
			return "", errors.New("not found")
		},
		isInteractive: func() bool {
			return true
		},
		usePromptUI: func(_ string, _ []string) (int, error) {
			return 1, nil
		},
	}

	index, err := s.Select(context.Background(), "pick", []string{"first", "second"})
	if err != nil {
		t.Fatalf("Select() returned error: %v", err)
	}
	if index != 1 {
		t.Fatalf("unexpected index: want=1 got=%d", index)
	}
}

func TestSelectOrCreate_UsesPromptUIWhenFZFIsUnavailable(t *testing.T) {
	t.Parallel()

	s := &defaultSelector{
		in:  bytes.NewBuffer(nil),
		out: &bytes.Buffer{},
		lookPath: func(string) (string, error) {
			return "", errors.New("not found")
		},
		isInteractive: func() bool {
			return true
		},
		usePromptUISelectOrCreate: func(_ string, _ []string) (SelectOrCreateResult, error) {
			return SelectOrCreateResult{Value: "feature/new", IsNew: true}, nil
		},
	}

	result, err := s.SelectOrCreate(context.Background(), "pick", []string{"first", "second"})
	if err != nil {
		t.Fatalf("SelectOrCreate() returned error: %v", err)
	}
	if !result.IsNew || result.Value != "feature/new" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestSelect_ReturnsCancelWhenPromptUICanceled(t *testing.T) {
	t.Parallel()

	s := &defaultSelector{
		in:  bytes.NewBufferString("1\n"),
		out: &bytes.Buffer{},
		lookPath: func(string) (string, error) {
			return "", errors.New("not found")
		},
		isInteractive: func() bool {
			return true
		},
		usePromptUI: func(_ string, _ []string) (int, error) {
			return -1, errSelectionCanceled
		},
	}

	_, err := s.Select(context.Background(), "pick", []string{"first", "second"})
	if err == nil {
		t.Fatal("expected Select() to return error")
	}
	if !errors.Is(err, errSelectionCanceled) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSelectOrCreate_ReturnsCancelWhenPromptUICanceled(t *testing.T) {
	t.Parallel()

	s := &defaultSelector{
		in:  bytes.NewBufferString("1\n"),
		out: &bytes.Buffer{},
		lookPath: func(string) (string, error) {
			return "", errors.New("not found")
		},
		isInteractive: func() bool {
			return true
		},
		usePromptUISelectOrCreate: func(_ string, _ []string) (SelectOrCreateResult, error) {
			return SelectOrCreateResult{}, errSelectionCanceled
		},
	}

	_, err := s.SelectOrCreate(context.Background(), "pick", []string{"first", "second"})
	if err == nil {
		t.Fatal("expected SelectOrCreate() to return error")
	}
	if !errors.Is(err, errSelectionCanceled) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSelect_FallsBackToNumberedWhenPromptUIUnavailable(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	s := &defaultSelector{
		in:  bytes.NewBufferString("2\n"),
		out: &out,
		lookPath: func(string) (string, error) {
			return "", errors.New("not found")
		},
		isInteractive: func() bool {
			return true
		},
		usePromptUI: func(_ string, _ []string) (int, error) {
			return -1, errPromptUIUnavailable
		},
	}

	index, err := s.Select(context.Background(), "pick", []string{"first", "second"})
	if err != nil {
		t.Fatalf("Select() returned error: %v", err)
	}
	if index != 1 {
		t.Fatalf("unexpected index: want=1 got=%d", index)
	}
}

func TestSelectOrCreate_FallsBackToNumberedWhenPromptUIUnavailable(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	s := &defaultSelector{
		in:  bytes.NewBufferString("2\n"),
		out: &out,
		lookPath: func(string) (string, error) {
			return "", errors.New("not found")
		},
		isInteractive: func() bool {
			return true
		},
		usePromptUISelectOrCreate: func(_ string, _ []string) (SelectOrCreateResult, error) {
			return SelectOrCreateResult{}, errPromptUIUnavailable
		},
	}

	result, err := s.SelectOrCreate(context.Background(), "pick", []string{"first", "second"})
	if err != nil {
		t.Fatalf("SelectOrCreate() returned error: %v", err)
	}
	if result.IsNew {
		t.Fatalf("expected existing selection, got new result: %+v", result)
	}
	if result.Value != "second" {
		t.Fatalf("unexpected selected value: %q", result.Value)
	}
}

func TestSelect_FallsBackToNumberedWhenNonInteractive(t *testing.T) {
	t.Parallel()

	var calledPromptUI bool
	s := &defaultSelector{
		in:  bytes.NewBufferString("1\n"),
		out: &bytes.Buffer{},
		lookPath: func(string) (string, error) {
			return "", errors.New("not found")
		},
		isInteractive: func() bool {
			return false
		},
		usePromptUI: func(_ string, _ []string) (int, error) {
			calledPromptUI = true
			return 0, nil
		},
	}

	index, err := s.Select(context.Background(), "pick", []string{"first", "second"})
	if err != nil {
		t.Fatalf("Select() returned error: %v", err)
	}
	if index != 0 {
		t.Fatalf("unexpected index: want=0 got=%d", index)
	}
	if calledPromptUI {
		t.Fatal("promptui should not be called in non-interactive mode")
	}
}

func TestSelectOrCreate_FallsBackToNumberedWhenNonInteractive(t *testing.T) {
	t.Parallel()

	var calledPromptUI bool
	s := &defaultSelector{
		in:  bytes.NewBufferString("1\n"),
		out: &bytes.Buffer{},
		lookPath: func(string) (string, error) {
			return "", errors.New("not found")
		},
		isInteractive: func() bool {
			return false
		},
		usePromptUISelectOrCreate: func(_ string, _ []string) (SelectOrCreateResult, error) {
			calledPromptUI = true
			return SelectOrCreateResult{Value: "x", IsNew: true}, nil
		},
	}

	result, err := s.SelectOrCreate(context.Background(), "pick", []string{"first", "second"})
	if err != nil {
		t.Fatalf("SelectOrCreate() returned error: %v", err)
	}
	if result.IsNew {
		t.Fatalf("expected existing selection, got new result: %+v", result)
	}
	if result.Value != "first" {
		t.Fatalf("unexpected selected value: %q", result.Value)
	}
	if calledPromptUI {
		t.Fatal("promptui should not be called in non-interactive mode")
	}
}

func TestSelect_NumberedFallbackRetryStillWorks(t *testing.T) {
	t.Parallel()

	s := &defaultSelector{
		in:  bytes.NewBufferString("x\n2\n"),
		out: &bytes.Buffer{},
		lookPath: func(string) (string, error) {
			return "", errors.New("not found")
		},
		isInteractive: func() bool {
			return false
		},
		usePromptUI: func(_ string, _ []string) (int, error) {
			return -1, errors.New("should not be called")
		},
	}

	index, err := s.Select(context.Background(), "pick", []string{"first", "second"})
	if err != nil {
		t.Fatalf("Select() returned error: %v", err)
	}
	if index != 1 {
		t.Fatalf("unexpected index: want=1 got=%d", index)
	}
}

func TestSelectorHelperProcess(t *testing.T) {
	t.Helper()

	if os.Getenv("GO_WANT_SELECTOR_HELPER_PROCESS") != "1" {
		return
	}

	if stdout := os.Getenv("HELPER_STDOUT"); stdout != "" {
		_, _ = io.WriteString(os.Stdout, stdout)
	}
	if stderr := os.Getenv("HELPER_STDERR"); stderr != "" {
		_, _ = io.WriteString(os.Stderr, stderr)
	}

	exitCode := 0
	if raw := os.Getenv("HELPER_EXIT_CODE"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			exitCode = parsed
		} else {
			exitCode = 2
		}
	}

	os.Exit(exitCode)
}
