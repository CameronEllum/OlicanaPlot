//go:build !windows

package ipc

import "os/exec"

func configureCommand(cmd *exec.Cmd, hide bool) {
	// No special configuration needed for other platforms
}
