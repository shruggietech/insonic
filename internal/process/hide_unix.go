//go:build !windows

// SPDX-License-Identifier: Apache-2.0
package process

import (
	"os/exec"
	"syscall"
)

func Hide(command *exec.Cmd, detached bool) {
	if detached {
		command.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	}
}
