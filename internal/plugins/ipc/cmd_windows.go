//go:build windows

package ipc

import (
	"os/exec"
	"syscall"
)

func configureCommand(cmd *exec.Cmd, hide bool) {
	if hide {
		if cmd.SysProcAttr == nil {
			cmd.SysProcAttr = &syscall.SysProcAttr{}
		}
		cmd.SysProcAttr.HideWindow = true
	}
}
