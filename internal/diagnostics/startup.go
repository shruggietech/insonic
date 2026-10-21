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
	return validNativeStartupStage(stage)
}

func validNativeStartupStage(stage string) bool {
	switch stage {
	case "window-lifetime", "profile-directory", "browser-path", "environment-create", "runtime-version", "environment-completion", "controller-request", "controller-completion", "controller-configuration", "webview", "bootstrap-script", "message-loop", "window-destroyed", "browser-process":
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
	var native interface{ StartupStage() string }
	if stage == "desktop-window" && errors.As(cause, &native) && validNativeStartupStage(native.StartupStage()) {
		event.FailureStage = native.StartupStage()
	}
	var code syscall.Errno
	if errors.As(cause, &code) && code != 0 && uint64(code) <= 0xffffffff {
		value := uint32(code)
		event.SystemErrorCode = &value
	}
	return event
}
