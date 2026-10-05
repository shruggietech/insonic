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

func TestDetachedOwnerHasExplicitIndependentLifetime(t *testing.T) {
	command := exec.Command("fixture")
	Hide(command, true)
	if command.SysProcAttr.CreationFlags&0x01000000 == 0 ||
		command.SysProcAttr.CreationFlags&0x08000000 == 0 {
		t.Fatal("detached owner lacks independent hidden startup")
	}
}
