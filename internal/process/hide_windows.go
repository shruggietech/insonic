// SPDX-License-Identifier: Apache-2.0
package process

import (
	"os/exec"
	"syscall"
)

func Hide(command *exec.Cmd, detached bool) {
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
}
