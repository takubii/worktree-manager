package cli

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/takubii/git-worktree-opener/internal/opener"
)

type fakeExitCodeError struct {
	code int
	msg  string
}

func (e fakeExitCodeError) Error() string {
	if strings.TrimSpace(e.msg) == "" {
		return "exit error"
	}
	return e.msg
}

func (e fakeExitCodeError) ExitCode() int {
	return e.code
}

func TestDefaultEnterRunner_StartShell_IgnoresShellExitErrorOnNonWindows(t *testing.T) {
	t.Parallel()

	runner := &defaultEnterRunner{
		goos: "linux",
		getEnv: func(key string) string {
			if key == "SHELL" {
				return "sh"
			}
			return ""
		},
		commandContext: func(_ context.Context, _ string, _ ...string) *exec.Cmd {
			return &exec.Cmd{}
		},
		runCommand: func(_ *exec.Cmd) error {
			return fakeExitCodeError{
				code: 42,
				msg:  "exit status 42",
			}
		},
	}

	if err := runner.StartShell(context.Background(), t.TempDir(), opener.TmuxModeAuto); err != nil {
		t.Fatalf("StartShell() returned error: %v", err)
	}
}

func TestDefaultEnterRunner_StartShell_ReturnsErrorForRunFailureOnNonWindows(t *testing.T) {
	t.Parallel()

	runner := &defaultEnterRunner{
		goos: "linux",
		getEnv: func(key string) string {
			if key == "SHELL" {
				return "sh"
			}
			return ""
		},
		commandContext: func(_ context.Context, _ string, _ ...string) *exec.Cmd {
			return &exec.Cmd{}
		},
		runCommand: func(_ *exec.Cmd) error {
			return errors.New("spawn failed")
		},
	}

	err := runner.StartShell(context.Background(), t.TempDir(), opener.TmuxModeAuto)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "use `wto enter --print-cd` to get a manual cd command, then retry") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDefaultEnterRunner_StartShell_WindowsForwardsInterruptToProcessGroup(t *testing.T) {
	t.Parallel()

	started := make(chan struct{})
	forwarded := make(chan int, 1)
	interruptDelivered := make(chan struct{})
	stopCalled := false
	prepareCalled := false

	runner := &defaultEnterRunner{
		goos: "windows",
		getEnv: func(key string) string {
			if key == "ComSpec" {
				return "cmd.exe"
			}
			return ""
		},
		command: func(_ string, _ ...string) *exec.Cmd {
			return &exec.Cmd{}
		},
		prepareShell: func(_ *exec.Cmd) error {
			prepareCalled = true
			return nil
		},
		startCommand: func(cmd *exec.Cmd) error {
			cmd.Process = &os.Process{Pid: 4321}
			close(started)
			return nil
		},
		waitCommand: func(_ *exec.Cmd) error {
			select {
			case pid := <-forwarded:
				if pid != 4321 {
					t.Fatalf("unexpected forwarded pid: %d", pid)
				}
				close(interruptDelivered)
			case <-time.After(2 * time.Second):
				t.Fatal("timed out waiting for forwarded interrupt")
			}
			return fakeExitCodeError{code: 130, msg: "exit status 130"}
		},
		signalNotify: func(c chan<- os.Signal, _ ...os.Signal) {
			go func() {
				<-started
				c <- os.Interrupt
			}()
		},
		signalStop: func(chan<- os.Signal) {
			stopCalled = true
		},
		sendInterrupt: func(pid int) error {
			forwarded <- pid
			return nil
		},
	}

	if err := runner.StartShell(context.Background(), t.TempDir(), opener.TmuxModeAuto); err != nil {
		t.Fatalf("StartShell() returned error: %v", err)
	}

	select {
	case <-interruptDelivered:
	case <-time.After(2 * time.Second):
		t.Fatal("interrupt was not delivered")
	}
	if !prepareCalled {
		t.Fatal("expected prepareShell to be called")
	}
	if !stopCalled {
		t.Fatal("expected signalStop to be called")
	}
}

func TestDefaultEnterRunner_StartShell_WindowsReturnsErrorForPrepareFailure(t *testing.T) {
	t.Parallel()

	runner := &defaultEnterRunner{
		goos: "windows",
		getEnv: func(key string) string {
			if key == "ComSpec" {
				return "cmd.exe"
			}
			return ""
		},
		command: func(_ string, _ ...string) *exec.Cmd {
			return &exec.Cmd{}
		},
		prepareShell: func(_ *exec.Cmd) error {
			return errors.New("prepare failed")
		},
	}

	err := runner.StartShell(context.Background(), t.TempDir(), opener.TmuxModeAuto)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "prepare failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDefaultEnterRunner_StartShell_WindowsReturnsErrorForStartFailure(t *testing.T) {
	t.Parallel()

	runner := &defaultEnterRunner{
		goos: "windows",
		getEnv: func(key string) string {
			if key == "ComSpec" {
				return "cmd.exe"
			}
			return ""
		},
		command: func(_ string, _ ...string) *exec.Cmd {
			return &exec.Cmd{}
		},
		prepareShell: func(_ *exec.Cmd) error {
			return nil
		},
		startCommand: func(_ *exec.Cmd) error {
			return errors.New("start failed")
		},
		signalNotify: func(chan<- os.Signal, ...os.Signal) {},
		signalStop:   func(chan<- os.Signal) {},
	}

	err := runner.StartShell(context.Background(), t.TempDir(), opener.TmuxModeAuto)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "start failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDefaultEnterRunner_StartShell_WindowsReturnsErrorForWaitFailure(t *testing.T) {
	t.Parallel()

	runner := &defaultEnterRunner{
		goos: "windows",
		getEnv: func(key string) string {
			if key == "ComSpec" {
				return "cmd.exe"
			}
			return ""
		},
		command: func(_ string, _ ...string) *exec.Cmd {
			return &exec.Cmd{}
		},
		prepareShell: func(_ *exec.Cmd) error {
			return nil
		},
		startCommand: func(cmd *exec.Cmd) error {
			cmd.Process = &os.Process{Pid: 1234}
			return nil
		},
		waitCommand: func(_ *exec.Cmd) error {
			return errors.New("wait failed")
		},
		signalNotify: func(chan<- os.Signal, ...os.Signal) {},
		signalStop:   func(chan<- os.Signal) {},
		sendInterrupt: func(int) error {
			return nil
		},
	}

	err := runner.StartShell(context.Background(), t.TempDir(), opener.TmuxModeAuto)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "wait failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDefaultEnterRunner_StartShell_WindowsKillsShellOnContextCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	started := make(chan struct{})
	killCalled := false

	runner := &defaultEnterRunner{
		goos: "windows",
		getEnv: func(key string) string {
			if key == "ComSpec" {
				return "cmd.exe"
			}
			return ""
		},
		command: func(_ string, _ ...string) *exec.Cmd {
			return &exec.Cmd{}
		},
		prepareShell: func(_ *exec.Cmd) error {
			return nil
		},
		startCommand: func(cmd *exec.Cmd) error {
			cmd.Process = &os.Process{Pid: 5678}
			close(started)
			return nil
		},
		waitCommand: func(_ *exec.Cmd) error {
			<-ctx.Done()
			return errors.New("killed")
		},
		killProcess: func(process *os.Process) error {
			if process == nil {
				t.Fatal("expected process")
			}
			killCalled = true
			return nil
		},
		signalNotify: func(chan<- os.Signal, ...os.Signal) {},
		signalStop:   func(chan<- os.Signal) {},
		sendInterrupt: func(int) error {
			return nil
		},
	}

	go func() {
		<-started
		cancel()
	}()

	err := runner.StartShell(ctx, t.TempDir(), opener.TmuxModeAuto)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled error, got: %v", err)
	}
	if !killCalled {
		t.Fatal("expected killProcess to be called")
	}
}
