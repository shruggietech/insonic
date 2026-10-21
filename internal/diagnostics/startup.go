// SPDX-License-Identifier: Apache-2.0
package diagnostics

import (
	"errors"
	"syscall"
)

func validStartupStage(stage string) bool {
	switch stage {
	case "workspace", "playback-transport", "desktop-window", "desktop-close":
		return true
	}
	return false
}

// StartupEvent retains generated failure metadata, never arbitrary error text.
func StartupEvent(stage string, cause error) Event {
	event := Event{Operation: "desktop.startup", Code: "unavailable", State: "failed"}
	if !validStartupStage(stage) {
		return event
	}
	event.FailureStage = stage
	var code syscall.Errno
	if errors.As(cause, &code) && code != 0 && uint64(code) <= 0xffffffff {
		value := uint32(code)
		event.SystemErrorCode = &value
	}
	return event
}
