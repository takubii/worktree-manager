package cli

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"

	"github.com/takubii/git-worktree-opener/internal/execerr"
	"github.com/takubii/git-worktree-opener/internal/opener"
)

type enterCommandContextFunc func(ctx context.Context, name string, args ...string) *exec.Cmd
type enterCommandFunc func(name string, args ...string) *exec.Cmd
type startCommandFunc func(cmd *exec.Cmd) error
type waitCommandFunc func(cmd *exec.Cmd) error
type signalNotifyFunc func(c chan<- os.Signal, sig ...os.Signal)
type signalStopFunc func(c chan<- os.Signal)
type prepareShellProcessFunc func(cmd *exec.Cmd) error
type sendShellInterruptFunc func(pid int) error
type killProcessFunc func(process *os.Process) error

// EnterRunner provides shell-specific behavior for `wto enter`.
type EnterRunner interface {
	FormatCDHints(path string) []string
	StartShell(ctx context.Context, path string, tmuxMode opener.TmuxMode) error
}

type defaultEnterRunner struct {
	goos           string
	getEnv         func(key string) string
	commandContext enterCommandContextFunc
	command        enterCommandFunc
	runCommand     func(cmd *exec.Cmd) error
	startCommand   startCommandFunc
	waitCommand    waitCommandFunc
	signalNotify   signalNotifyFunc
	signalStop     signalStopFunc
	prepareShell   prepareShellProcessFunc
	sendInterrupt  sendShellInterruptFunc
	killProcess    killProcessFunc
}

func newDefaultEnterRunner() EnterRunner {
	return &defaultEnterRunner{
		goos:           runtime.GOOS,
		getEnv:         os.Getenv,
		commandContext: exec.CommandContext,
		command:        exec.Command,
		runCommand:     (*exec.Cmd).Run,
		startCommand:   (*exec.Cmd).Start,
		waitCommand:    (*exec.Cmd).Wait,
		signalNotify:   signal.Notify,
		signalStop:     signal.Stop,
		prepareShell:   prepareShellProcessForInterruptForwarding,
		sendInterrupt:  sendShellInterruptToProcessGroup,
		killProcess:    (*os.Process).Kill,
	}
}

func (r *defaultEnterRunner) FormatCDHints(path string) []string {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}

	if r.goos == "windows" {
		return []string{
			`cmd.exe: cd /d "` + strings.ReplaceAll(path, `"`, `""`) + `"`,
			`PowerShell: Set-Location -LiteralPath '` + strings.ReplaceAll(path, `'`, `''`) + `'`,
		}
	}

	return []string{
		"cd " + shellSingleQuote(path),
	}
}

func (r *defaultEnterRunner) StartShell(ctx context.Context, path string, _ opener.TmuxMode) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return execerr.Build("interactive shell", "selected worktree path is empty", "select a valid worktree and retry")
	}

	var shell string
	switch r.goos {
	case "windows":
		shell = strings.TrimSpace(r.getEnv("ComSpec"))
		if shell == "" {
			shell = "cmd.exe"
		}
	default:
		shell = strings.TrimSpace(r.getEnv("SHELL"))
		if shell == "" {
			shell = "sh"
		}
	}

	var cmd *exec.Cmd
	if r.goos == "windows" {
		command := r.command
		if command == nil {
			command = exec.Command
		}
		cmd = command(shell)
	} else {
		commandContext := r.commandContext
		if commandContext == nil {
			commandContext = exec.CommandContext
		}
		cmd = commandContext(ctx, shell)
	}

	cmd.Dir = path
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if r.goos == "windows" {
		return r.startWindowsShell(ctx, cmd, shell)
	}

	runCommand := r.runCommand
	if runCommand == nil {
		runCommand = (*exec.Cmd).Run
	}
	if err := runCommand(cmd); err != nil {
		if isShellExitError(err) {
			return nil
		}

		return execerr.Build(
			shell,
			err.Error(),
			"use `wto enter --print-cd` to get a manual cd command, then retry",
		)
	}

	return nil
}

func (r *defaultEnterRunner) startWindowsShell(ctx context.Context, cmd *exec.Cmd, shell string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	prepareShell := r.prepareShell
	if prepareShell == nil {
		prepareShell = prepareShellProcessForInterruptForwarding
	}
	startCommand := r.startCommand
	if startCommand == nil {
		startCommand = (*exec.Cmd).Start
	}
	waitCommand := r.waitCommand
	if waitCommand == nil {
		waitCommand = (*exec.Cmd).Wait
	}
	signalNotify := r.signalNotify
	if signalNotify == nil {
		signalNotify = signal.Notify
	}
	signalStop := r.signalStop
	if signalStop == nil {
		signalStop = signal.Stop
	}
	sendInterrupt := r.sendInterrupt
	if sendInterrupt == nil {
		sendInterrupt = sendShellInterruptToProcessGroup
	}
	killProcess := r.killProcess
	if killProcess == nil {
		killProcess = (*os.Process).Kill
	}

	if err := prepareShell(cmd); err != nil {
		return execerr.Build(
			shell,
			err.Error(),
			"use `wto enter --print-cd` to get a manual cd command, then retry",
		)
	}

	sigCh := make(chan os.Signal, 1)
	signalNotify(sigCh, os.Interrupt)
	defer signalStop(sigCh)

	if err := startCommand(cmd); err != nil {
		return execerr.Build(
			shell,
			err.Error(),
			"use `wto enter --print-cd` to get a manual cd command, then retry",
		)
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- waitCommand(cmd)
	}()

	for {
		select {
		case waitErr := <-waitCh:
			if waitErr != nil {
				if isShellExitError(waitErr) {
					return nil
				}

				return execerr.Build(
					shell,
					waitErr.Error(),
					"use `wto enter --print-cd` to get a manual cd command, then retry",
				)
			}

			return nil
		case <-ctx.Done():
			if cmd.Process != nil {
				if err := killProcess(cmd.Process); err != nil && !errors.Is(err, os.ErrProcessDone) {
					tracef(ctx, "enter: failed to kill shell process on context cancellation: pid=%d err=%v", cmd.Process.Pid, err)
				}
			}
			waitErr := <-waitCh
			if waitErr != nil && !isShellExitError(waitErr) && !errors.Is(waitErr, os.ErrProcessDone) {
				tracef(ctx, "enter: shell exited after context cancellation with err=%v", waitErr)
			}
			return ctx.Err()
		case <-sigCh:
			if cmd.Process == nil {
				continue
			}
			if err := sendInterrupt(cmd.Process.Pid); err != nil {
				tracef(ctx, "enter: failed to forward interrupt to shell process group: pid=%d err=%v", cmd.Process.Pid, err)
			}
		}
	}
}

type exitCodeError interface {
	ExitCode() int
}

func isShellExitError(err error) bool {
	var codedErr exitCodeError
	return errors.As(err, &codedErr)
}

func shellSingleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
