package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/takubii/worktree-manager/internal/updater"
)

type fakeUpdater struct {
	calls  int
	last   updater.Request
	result updater.Result
	err    error
}

func (f *fakeUpdater) Update(_ context.Context, req updater.Request) (updater.Result, error) {
	f.calls++
	f.last = req
	return f.result, f.err
}

func TestUpdateCommand_RunsUpdaterWithVersion(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	upd := &fakeUpdater{}

	cmd := NewRootCmd(Dependencies{
		Stdout:  &stdout,
		Stderr:  &stderr,
		Updater: upd,
	})
	cmd.SetArgs([]string{"update", "--version", "v0.1.0"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if upd.calls != 1 {
		t.Fatalf("expected updater to be called once, got %d", upd.calls)
	}
	if upd.last.Version != "v0.1.0" {
		t.Fatalf("expected version v0.1.0, got %q", upd.last.Version)
	}
	if upd.last.Stdout == nil {
		t.Fatal("expected stdout writer to be set")
	}
	if upd.last.Stderr == nil {
		t.Fatal("expected stderr writer to be set")
	}
}

func TestUpdateCommand_PrintsBackgroundNoticeForAsyncUpdate(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	upd := &fakeUpdater{
		result: updater.Result{Async: true},
	}

	cmd := NewRootCmd(Dependencies{
		Stdout:  &stdout,
		Stderr:  &stderr,
		Updater: upd,
	})
	cmd.SetArgs([]string{"update"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if !strings.Contains(stdout.String(), "update started in background") {
		t.Fatalf("expected async update notice, got: %q", stdout.String())
	}
}

func TestUpdateCommand_ReturnsUpdaterError(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	upd := &fakeUpdater{
		err: errors.New("network failure"),
	}

	cmd := NewRootCmd(Dependencies{
		Stdout:  &stdout,
		Stderr:  &stderr,
		Updater: upd,
	})
	cmd.SetArgs([]string{"update"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected Execute() to return error")
	}
	if !strings.Contains(err.Error(), "network failure") {
		t.Fatalf("unexpected error message: %v", err)
	}
}
