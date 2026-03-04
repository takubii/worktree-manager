//go:build !windows

package cli

import "os/exec"

func prepareShellProcessForInterruptForwarding(_ *exec.Cmd) error {
	return nil
}

func sendShellInterruptToProcessGroup(_ int) error {
	return nil
}
