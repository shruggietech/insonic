//go:build !windows

// SPDX-License-Identifier: Apache-2.0
package launcher

import (
	"os"
	"os/exec"
	"time"
)

func openDocumentation(directory, entry string) error {
	if entry == "" {
		var err error
		entry, err = DocumentationEntry(directory)
		if err != nil {
			return err
		}
	}
	null, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return documentationCorrection
	}
	defer null.Close()
	cmd := exec.Command("xdg-open", entry)
	cmd.Stdin = null
	cmd.Stdout = null
	cmd.Stderr = null
	if cmd.Start() != nil {
		return documentationCorrection
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case err := <-done:
		if err != nil {
			return documentationCorrection
		}
	case <-timer.C:
	}
	// Some desktop handlers remain alive with their browser. Do not terminate
	// elected user applications merely because bounded dispatch observation ends.
	return nil
}

func ReportDocumentationError() {}
