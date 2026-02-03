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
		// CREATE_NO_WINDOW (0x08000000) hides the console window without affecting GUI windows.
		cmd.SysProcAttr.CreationFlags = 0x08000000
	}
}
