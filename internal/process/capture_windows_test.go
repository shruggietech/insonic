// SPDX-License-Identifier: Apache-2.0
package process

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestCaptureDescendantHelper(t *testing.T) {
	mode := os.Getenv("INSONIC_CAPTURE_DESCENDANT")
	root := os.Getenv("INSONIC_CAPTURE_DIRECTORY")
	if mode == "" {
		return
	}
	if mode == "child" {
		os.WriteFile(filepath.Join(root, "ready"), []byte("ready"), 0600)
		delay := 500 * time.Millisecond
		if os.Getenv("INSONIC_CAPTURE_HOLD") == "1" {
			delay = 10 * time.Second
		}
		time.Sleep(delay)
		os.WriteFile(filepath.Join(root, "released"), []byte("released naturally"), 0600)
		os.Exit(0)
	}
	executable, _ := os.Executable()
	command := exec.Command(executable, "-test.run=^TestCaptureDescendantHelper$")
	command.Env = append(os.Environ(), "INSONIC_CAPTURE_DESCENDANT=child")
	Hide(command, false)
	if err := command.Start(); err != nil {
		os.Exit(2)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(filepath.Join(root, "ready")); err == nil {
			os.Exit(0)
		}
		time.Sleep(10 * time.Millisecond)
	}
	os.Exit(3)
}

func TestCaptureWaitsForNaturalOwnedDescendantExit(t *testing.T) {
	root := t.TempDir()
	executable, _ := os.Executable()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	report, err := Capture(ctx, Spec{Executable: executable, Args: []string{"-test.run=^TestCaptureDescendantHelper$"}, Env: append(os.Environ(), "INSONIC_CAPTURE_DESCENDANT=parent", "INSONIC_CAPTURE_DIRECTORY="+root), MaxOutput: 1024, WaitForDescendants: true})
	if err != nil || report.ExitCode != 0 {
		t.Fatalf("capture: %+v %v", report, err)
	}
	if report.Descendants == nil || report.Descendants.Remaining != 0 || !report.Descendants.Released {
		t.Fatalf("missing natural release observation: %+v", report.Descendants)
	}
	if _, err := os.Stat(filepath.Join(root, "released")); err != nil {
		t.Fatal("successful root return killed its owned descendant before natural release:", err)
	}
}

func TestCaptureDescendantDeadlineCannotQualifyCleanupKill(t *testing.T) {
	root := t.TempDir()
	executable, _ := os.Executable()
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	report, err := Capture(ctx, Spec{Executable: executable, Args: []string{"-test.run=^TestCaptureDescendantHelper$"}, Env: append(os.Environ(), "INSONIC_CAPTURE_DESCENDANT=parent", "INSONIC_CAPTURE_DIRECTORY="+root, "INSONIC_CAPTURE_HOLD=1"), MaxOutput: 1024, WaitForDescendants: true})
	if err == nil || ctx.Err() != context.DeadlineExceeded || report.ExitCode != 0 || report.Descendants == nil || report.Descendants.Released {
		t.Fatalf("cleanup termination became success: %+v %v", report, err)
	}
	if _, err := os.Stat(filepath.Join(root, "released")); err == nil {
		t.Fatal("deadline did not stop the owned descendant")
	}
}
