// SPDX-License-Identifier: Apache-2.0
package process

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/sys/windows"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"testing/synctest"
	"time"

	"github.com/shruggietech/insonic/internal/qualificationtiming"
)

func TestCaptureDescendantHelper(t *testing.T) {
	mode := os.Getenv("INSONIC_CAPTURE_DESCENDANT")
	root := os.Getenv("INSONIC_CAPTURE_DIRECTORY")
	if mode == "" {
		return
	}
	if mode == "child" {
		os.WriteFile(filepath.Join(root, "ready"), []byte(strconv.Itoa(os.Getpid())), 0600)
		gate := "finish"
		if os.Getenv("INSONIC_CAPTURE_HOLD") == "1" {
			gate = "never"
		}
		if !helperProcessFixture(filepath.Join(root, gate)) {
			os.Exit(3)
		}
		os.WriteFile(filepath.Join(root, "released"), []byte("released naturally"), 0600)
		os.Exit(0)
	}
	if os.WriteFile(filepath.Join(root, "root"), []byte(strconv.Itoa(os.Getpid())), 0600) != nil {
		os.Exit(2)
	}
	executable, _ := os.Executable()
	command := exec.Command(executable, "-test.run=^TestCaptureDescendantHelper$")
	command.Env = append(os.Environ(), "INSONIC_CAPTURE_DESCENDANT=child")
	Hide(command, false)
	if err := command.Start(); err != nil {
		os.Exit(2)
	}
	if helperProcessFixture(filepath.Join(root, "ready")) {
		if delay, err := strconv.Atoi(os.Getenv("INSONIC_CAPTURE_ROOT_DELAY_MS")); err == nil && delay > 0 && delay <= 1000 {
			time.Sleep(time.Duration(delay) * time.Millisecond)
		}
		os.Exit(0)
	}
	os.Exit(3)
}

// Gates keep the descendant alive until the root has actually exited. Healthy
// child startup receives the shared readiness budget, independent of the short
// scheduling assumptions used by the original fixture.
func captureControlledFixture(t *testing.T, parent context.Context, spec Spec, root string, natural bool) (CaptureResult, error) {
	t.Helper()
	ctx, cancel := context.WithCancel(parent)
	type result struct {
		report CaptureResult
		err    error
	}
	done, finished := make(chan result, 1), make(chan struct{})
	go func() { report, err := Capture(ctx, spec); done <- result{report, err}; close(finished) }()
	defer func() {
		cancel()
		select {
		case <-finished:
		case <-time.After(qualificationtiming.Duration("cleanup_seconds")):
			t.Error("controlled Capture did not finish cleanup")
		}
	}()
	harness := processTestContext(t)
	childPID := requireProcessFixture(t, harness, filepath.Join(root, "ready"))
	rootPID := requireProcessFixture(t, harness, filepath.Join(root, "root"))
	waitCaptureFixtureProcess(t, rootPID)
	if natural {
		if err := os.WriteFile(filepath.Join(root, "finish"), []byte("release owned child"), 0600); err != nil {
			t.Fatal(err)
		}
		waitCaptureFixtureProcess(t, childPID)
	}
	got := <-done
	waitCaptureFixtureProcess(t, childPID)
	return got.report, got.err
}

// Native process waiting keeps synctest's virtual clock frozen while the real
// helper starts/exits. The wait is bounded by shared real readiness headroom.
func waitCaptureFixtureProcess(t *testing.T, raw []byte) {
	t.Helper()
	if err := waitCaptureFixtureProcessError(raw); err != nil {
		t.Fatal(err)
	}
}
func waitCaptureFixtureProcessError(raw []byte) error {
	pid, err := strconv.Atoi(string(raw))
	if err != nil || pid < 1 {
		return fmt.Errorf("invalid controlled process identity")
	}
	handle, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err == windows.ERROR_INVALID_PARAMETER {
		return nil
	}
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)
	state, err := windows.WaitForSingleObject(handle, uint32(qualificationtiming.Seconds("readiness_seconds")*1000))
	if err != nil || state != windows.WAIT_OBJECT_0 {
		return fmt.Errorf("controlled process did not exit: state=%d err=%v", state, err)
	}
	return nil
}

type captureFixtureInputDelay struct{ delay time.Duration }

func (r *captureFixtureInputDelay) Read([]byte) (int, error) { time.Sleep(r.delay); return 0, io.EOF }

func TestCaptureWaitsForNaturalOwnedDescendantExit(t *testing.T) {
	root := t.TempDir()
	executable, _ := os.Executable()
	ctx, cancel := qualificationtiming.Context(context.Background(), "readiness_seconds")
	defer cancel()
	report, err := captureControlledFixture(t, ctx, Spec{Executable: executable, Args: []string{"-test.run=^TestCaptureDescendantHelper$"}, Env: append(os.Environ(), "INSONIC_CAPTURE_DESCENDANT=parent", "INSONIC_CAPTURE_DIRECTORY="+root), MaxOutput: 1024, WaitForDescendants: true}, root, true)
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
	synctest.Test(t, func(t *testing.T) {
		root := t.TempDir()
		executable, _ := os.Executable()
		ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
		defer cancel()
		report, err := captureControlledFixture(t, ctx, Spec{Executable: executable, Args: []string{"-test.run=^TestCaptureDescendantHelper$"}, Env: append(os.Environ(), "INSONIC_CAPTURE_DESCENDANT=parent", "INSONIC_CAPTURE_DIRECTORY="+root, "INSONIC_CAPTURE_HOLD=1"), MaxOutput: 1024, WaitForDescendants: true}, root, false)
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
	})
}

func TestCaptureNaturalExitRemainsObservedWithExternalProcessHandle(t *testing.T) {
	root := t.TempDir()
	executable, _ := os.Executable()
	ctx, cancel := qualificationtiming.Context(context.Background(), "readiness_seconds")
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
	deadline := time.Now().Add(qualificationtiming.Duration("readiness_seconds"))
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
	rootPID := requireProcessFixture(t, ctx, filepath.Join(root, "root"))
	requireFixtureProcessStopped(t, ctx, rootPID)
	if err := os.WriteFile(filepath.Join(root, "finish"), []byte("release owned child"), 0600); err != nil {
		t.Fatal(err)
	}
	state, err := windows.WaitForSingleObject(handle, uint32(qualificationtiming.Seconds("readiness_seconds")*1000))
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
			check := func(t *testing.T) {
				root := t.TempDir()
				executable, _ := os.Executable()
				budget := qualificationtiming.Duration("readiness_seconds")
				if tc.hold {
					budget = 2500 * time.Millisecond
				}

				started := time.Now()
				ctx, cancel := context.WithTimeout(context.Background(), budget)
				defer cancel()
				deadline, _ := ctx.Deadline()
				env := append(os.Environ(), "INSONIC_CAPTURE_DESCENDANT=parent", "INSONIC_CAPTURE_DIRECTORY="+root, "INSONIC_CAPTURE_ROOT_DELAY_MS=0", "INSONIC_CAPTURE_HOLD=0")
				if tc.hold {
					env = append(env, "INSONIC_CAPTURE_HOLD=1")
				}
				report, err := captureControlledFixture(t, ctx, Spec{Executable: executable, Args: []string{"-test.run=^TestCaptureDescendantHelper$"}, Env: env, Input: &captureFixtureInputDelay{delay: time.Duration(tc.delayMS) * time.Millisecond}, MaxOutput: 1024, WaitForDescendants: true}, root, !tc.hold)
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
				if observed.LaunchElapsedNS < 0 || observed.RootWaitElapsedNS < 0 || observed.RootExitElapsedNS != observed.LaunchElapsedNS+observed.RootWaitElapsedNS || !observed.ReleaseDeadline || releaseBudget <= 0 || releaseWait < 0 {
					t.Fatalf("invalid monotonic stage timings: %+v", observed)
				}
				if rootElapsed < time.Duration(tc.delayMS)*time.Millisecond || rootElapsed+releaseBudget > budget || rootElapsed+releaseBudget < budget-250*time.Millisecond {
					t.Fatalf("release wait lost original consumed budget: root=%s remaining=%s budget=%s", rootElapsed, releaseBudget, budget)
				}
				if tc.hold {
					if err == nil || ctx.Err() != context.DeadlineExceeded || report.Descendants.Released || report.Descendants.Remaining == 0 || releaseWait < releaseBudget || releaseWait > releaseBudget+qualificationtiming.Duration("cleanup_seconds") || elapsed > budget+qualificationtiming.Duration("cleanup_seconds") {
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
			}
			// Real natural release includes Windows job-accounting updates.
			// Only held-descendant expiry uses virtual time; its original
			// standard context deadline remains fixed throughout Capture.
			if tc.hold {
				synctest.Test(t, check)
			} else {
				check(t)
			}

		})
	}
}

func TestCaptureDescendantTimingWithoutDeadline(t *testing.T) {
	root := t.TempDir()
	executable, _ := os.Executable()
	report, err := captureControlledFixture(t, context.Background(), Spec{Executable: executable, Args: []string{"-test.run=^TestCaptureDescendantHelper$"}, Env: append(os.Environ(), "INSONIC_CAPTURE_DESCENDANT=parent", "INSONIC_CAPTURE_DIRECTORY="+root), MaxOutput: 1024, WaitForDescendants: true}, root, true)
	if err != nil || report.ExitCode != 0 || report.Descendants == nil || !report.Descendants.Released {
		t.Fatalf("deadline-free natural release changed: %+v %v", report, err)
	}
	timing := report.Descendants.Timing
	if timing == nil || timing.ReleaseDeadline || timing.ReleaseBudgetNS != 0 || timing.ReleaseWaitElapsedNS <= 0 {
		t.Fatalf("deadline-free observation invented a release budget: %+v", timing)
	}
}
