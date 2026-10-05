package process

import (
	"os/exec"
	"testing"
)

func TestHiddenCreationFlags(t *testing.T) {
	command := exec.Command("unused")
	Hide(command, false)
	if command.SysProcAttr == nil || !command.SysProcAttr.HideWindow || command.SysProcAttr.CreationFlags&0x08000000 == 0 {
		t.Fatal("console hiding not guaranteed")
	}
}
