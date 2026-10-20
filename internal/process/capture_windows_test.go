// SPDX-License-Identifier: Apache-2.0
package process

import (
	"context"
	"golang.org/x/sys/windows"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
		os.WriteFile(filepath.Join(root, "ready"), []byte(strconv.Itoa(os.Getpid())), 0600)
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
	foundLiveTest := false
	for _, member := range report.Descendants.Processes {
		if member.Observed && member.Running && member.Image == "test" {
			foundLiveTest = true
		}
	}
	if !foundLiveTest {
		t.Fatalf("missing owned live-process identity: %+v", report.Descendants)
	}
	if _, err := os.Stat(filepath.Join(root, "released")); err == nil {
		t.Fatal("deadline did not stop the owned descendant")
	}
}

func TestCaptureNaturalExitRemainsObservedWithExternalProcessHandle(t *testing.T) {
	root := t.TempDir()
	executable, _ := os.Executable()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	type result struct {
		report CaptureResult
		err    error
	}
	done := make(chan result, 1)
	go func() {
		report, err := Capture(ctx, Spec{Executable: executable, Args: []string{"-test.run=^TestCaptureDescendantHelper$"}, Env: append(os.Environ(), "INSONIC_CAPTURE_DESCENDANT=parent", "INSONIC_CAPTURE_DIRECTORY="+root), MaxOutput: 1024, WaitForDescendants: true})
		done <- result{report, err}
	}()
	var pid int
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		raw, _ := os.ReadFile(filepath.Join(root, "ready"))
		pid, _ = strconv.Atoi(string(raw))
		if pid > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if pid == 0 {
		t.Fatal("owned descendant did not start")
	}
	handle, err := windows.OpenProcess(windows.SYNCHRONIZE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(handle)
	state, err := windows.WaitForSingleObject(handle, 3000)
	if err != nil || state != windows.WAIT_OBJECT_0 {
		t.Fatal("descendant did not exit naturally", state, err)
	}
	got := <-done
	if got.err != nil || got.report.Descendants == nil || !got.report.Descendants.Released {
		t.Fatalf("held handle hid actual process exit: %+v %v", got.report, got.err)
	}
}
