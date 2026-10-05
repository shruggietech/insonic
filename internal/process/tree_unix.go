//go:build !windows

package process

import (
	"os/exec"
	"syscall"
)

type processTree struct{ pid int }

func startTree(command *exec.Cmd) (*processTree, error) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := command.Start(); err != nil {
		return nil, err
	}
	return &processTree{command.Process.Pid}, nil
}

func (tree *processTree) Kill()  { _ = syscall.Kill(-tree.pid, syscall.SIGKILL) }
func (tree *processTree) Close() { tree.Kill() }
