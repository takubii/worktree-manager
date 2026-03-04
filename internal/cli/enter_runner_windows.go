//go:build windows

package cli

import (
	"fmt"
	"os/exec"
	"syscall"
)

var (
	kernel32DLL                  = syscall.NewLazyDLL("kernel32.dll")
	generateConsoleCtrlEventProc = kernel32DLL.NewProc("GenerateConsoleCtrlEvent")
)

func prepareShellProcessForInterruptForwarding(cmd *exec.Cmd) error {
	if cmd == nil {
		return fmt.Errorf("shell command is nil")
	}

	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= syscall.CREATE_NEW_PROCESS_GROUP
	return nil
}

func sendShellInterruptToProcessGroup(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("shell process id is invalid: %d", pid)
	}

	r1, _, e1 := generateConsoleCtrlEventProc.Call(
		uintptr(syscall.CTRL_BREAK_EVENT),
		uintptr(uint32(pid)),
	)
	if r1 == 0 {
		if e1 != syscall.Errno(0) {
			return e1
		}
		return fmt.Errorf("GenerateConsoleCtrlEvent failed")
	}

	return nil
}
