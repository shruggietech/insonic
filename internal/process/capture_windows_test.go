// SPDX-License-Identifier: Apache-2.0
package process

import (
	"context"
	"encoding/json"
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
			if delay, err := strconv.Atoi(os.Getenv("INSONIC_CAPTURE_ROOT_DELAY_MS")); err == nil && delay > 0 && delay <= 1000 {
				time.Sleep(time.Duration(delay) * time.Millisecond)
			}
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

// These children spend different portions of one end-to-end budget in the root.
// Descendant observation must retain that consumed time instead of starting a
// new release deadline, and cleanup must not overwrite the failed observation.
func TestCaptureDescendantTimingRetainsOriginalCombinedBudget(t *testing.T) {
	type timing struct {
		LaunchElapsedNS      int64 `json:"launch_elapsed_ns"`
		RootWaitElapsedNS    int64 `json:"root_wait_elapsed_ns"`
		RootExitElapsedNS    int64 `json:"root_exit_elapsed_ns"`
		ReleaseDeadline      bool  `json:"release_deadline"`
		ReleaseBudgetNS      int64 `json:"release_budget_ns"`
		ReleaseWaitElapsedNS int64 `json:"release_wait_elapsed_ns"`
	}
	for _, tc := range []struct {
		name    string
		delayMS int
		hold    bool
	}{
		{"fast-root-natural", 0, false},
		{"slow-root-natural", 800, false},
		{"fast-root-deadline", 0, true},
		{"slow-root-deadline", 800, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			executable, _ := os.Executable()
			const budget = 2500 * time.Millisecond
			started := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), budget)
			defer cancel()
			deadline, _ := ctx.Deadline()
			env := append(os.Environ(), "INSONIC_CAPTURE_DESCENDANT=parent", "INSONIC_CAPTURE_DIRECTORY="+root, "INSONIC_CAPTURE_ROOT_DELAY_MS="+strconv.Itoa(tc.delayMS), "INSONIC_CAPTURE_HOLD=0")
			if tc.hold {
				env = append(env, "INSONIC_CAPTURE_HOLD=1")
			}
			report, err := Capture(ctx, Spec{Executable: executable, Args: []string{"-test.run=^TestCaptureDescendantHelper$"}, Env: env, MaxOutput: 1024, WaitForDescendants: true})
			elapsed := time.Since(started)
			if report.ExitCode != 0 || report.Descendants == nil {
				t.Fatalf("root did not finish before descendant observation: %+v %v", report, err)
			}
			var encoded struct {
				Timing *timing `json:"timing"`
			}
			raw, marshalErr := json.Marshal(report.Descendants)
			if marshalErr != nil || json.Unmarshal(raw, &encoded) != nil || encoded.Timing == nil {
				t.Fatalf("missing original-budget timing: %s %v", raw, marshalErr)
			}
			observed := encoded.Timing
			rootElapsed := time.Duration(observed.RootExitElapsedNS)
			releaseBudget := time.Duration(observed.ReleaseBudgetNS)
			releaseWait := time.Duration(observed.ReleaseWaitElapsedNS)
			if observed.LaunchElapsedNS <= 0 || observed.RootWaitElapsedNS <= 0 || observed.RootExitElapsedNS != observed.LaunchElapsedNS+observed.RootWaitElapsedNS || !observed.ReleaseDeadline || releaseBudget <= 0 || releaseWait < 0 {
				t.Fatalf("invalid monotonic stage timings: %+v", observed)
			}
			if rootElapsed < time.Duration(tc.delayMS)*time.Millisecond || rootElapsed+releaseBudget > budget || rootElapsed+releaseBudget < budget-250*time.Millisecond {
				t.Fatalf("release wait lost original consumed budget: root=%s remaining=%s budget=%s", rootElapsed, releaseBudget, budget)
			}
			if tc.hold {
				if err == nil || ctx.Err() != context.DeadlineExceeded || report.Descendants.Released || report.Descendants.Remaining == 0 || releaseWait < releaseBudget || releaseWait > releaseBudget+500*time.Millisecond || elapsed > budget+time.Second {
					t.Fatalf("deadline or failed release changed: report=%+v timing=%+v elapsed=%s err=%v", report, observed, elapsed, err)
				}
				live := false
				for _, process := range report.Descendants.Processes {
					live = live || process.Observed && process.Running && process.Image == "test"
				}
				if !live {
					t.Fatal("lost pre-cleanup live descendant")
				}
				if _, statErr := os.Stat(filepath.Join(root, "released")); !os.IsNotExist(statErr) {
					t.Fatal("cleanup became natural release", statErr)
				}
			} else {
				if err != nil || !report.Descendants.Released || report.Descendants.Remaining != 0 || releaseWait >= releaseBudget {
					t.Fatalf("natural release changed: report=%+v timing=%+v err=%v", report, observed, err)
				}
				if _, statErr := os.Stat(filepath.Join(root, "released")); statErr != nil {
					t.Fatal("descendant did not release naturally", statErr)
				}
			}
			if after, _ := ctx.Deadline(); !after.Equal(deadline) {
				t.Fatal("capture changed the caller deadline")
			}
			t.Logf("root=%s launch=%s wait=%s release-budget=%s release-wait=%s released=%t", rootElapsed, time.Duration(observed.LaunchElapsedNS), time.Duration(observed.RootWaitElapsedNS), releaseBudget, releaseWait, report.Descendants.Released)
		})
	}
}

func TestCaptureDescendantTimingWithoutDeadline(t *testing.T) {
	root := t.TempDir()
	executable, _ := os.Executable()
	report, err := Capture(context.Background(), Spec{Executable: executable, Args: []string{"-test.run=^TestCaptureDescendantHelper$"}, Env: append(os.Environ(), "INSONIC_CAPTURE_DESCENDANT=parent", "INSONIC_CAPTURE_DIRECTORY="+root), MaxOutput: 1024, WaitForDescendants: true})
	if err != nil || report.ExitCode != 0 || report.Descendants == nil || !report.Descendants.Released {
		t.Fatalf("deadline-free natural release changed: %+v %v", report, err)
	}
	timing := report.Descendants.Timing
	if timing == nil || timing.ReleaseDeadline || timing.ReleaseBudgetNS != 0 || timing.ReleaseWaitElapsedNS <= 0 {
		t.Fatalf("deadline-free observation invented a release budget: %+v", timing)
	}
}
