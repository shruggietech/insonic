// SPDX-License-Identifier: Apache-2.0
package process

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"syscall"

	"github.com/shruggietech/insonic/internal/contracts"
)

var startupDiagnosticSequence atomic.Uint32

// startupLaunchFailure preserves the public error contract. Elected CI evidence
// is written separately so protected input and CLI response streams stay intact.
func startupLaunchFailure(stage string, cause error) error {
	writeStartupDiagnostic(stage, cause)
	return contracts.Condition("unavailable", "startup_launch")
}

func writeStartupDiagnostic(stage string, cause error) {
	if os.Getenv("GITHUB_ACTIONS") != "true" || os.Getenv("INSONIC_PROCESS_STARTUP_DIAGNOSTICS") != "1" {
		return
	}
	if stage != "stdout_pipe" && stage != "stdin_pipe" && stage != "command_start" {
		return
	}
	root := os.Getenv("GITHUB_WORKSPACE")
	if !filepath.IsAbs(root) {
		return
	}
	sequence := startupDiagnosticSequence.Add(1)
	if sequence > 16 {
		return
	}
	var number syscall.Errno
	found := errors.As(cause, &number)
	// Do not serialize cause.Error(): wrappers contain executable paths, and
	// callers may include protected input or environment values in errors.
	data, err := json.Marshal(struct {
		Kind        string `json:"kind"`
		Stage       string `json:"stage"`
		Platform    string `json:"platform"`
		PID         int    `json:"pid"`
		HasErrno    bool   `json:"has_errno"`
		ErrorNumber uint64 `json:"error_number"`
	}{"process-startup-diagnostic", stage, runtime.GOOS, os.Getpid(), found, uint64(number)})
	if err != nil {
		return
	}
	directory := filepath.Join(root, "build", "native")
	if os.MkdirAll(directory, 0700) != nil {
		return
	}
	name := fmt.Sprintf("process-startup-%d-%d-receipt.json", os.Getpid(), sequence)
	file, err := os.OpenFile(filepath.Join(directory, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = file.Write(append(data, '\n'))
}
