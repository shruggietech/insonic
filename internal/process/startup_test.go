// SPDX-License-Identifier: Apache-2.0
package process

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/qualificationtiming"
)

func TestStartupHelper(t *testing.T) {
	if os.Getenv("INSONIC_STARTUP_HELPER") != "1" {
		return
	}
	mode := os.Args[len(os.Args)-1]
	control := os.Getenv("INSONIC_STARTUP_CONTROL_DIRECTORY")
	if control != "" {
		name := "started"
		if mode == "hold" {
			name = "holder-ready"
		}
		if os.WriteFile(filepath.Join(control, name), []byte(strconv.Itoa(os.Getpid())), 0600) != nil {
			os.Exit(2)
		}
	}
	switch mode {
	case "lifetime-hang", "lifetime-invalid", "lifetime-ready":
		if mode == "lifetime-ready" {
			fmt.Println("{\"state\":\"ready\"}")
		}
		if mode == "lifetime-invalid" {
			fmt.Println("{\"state\":\"invalid\"}")
		}
		if !helperProcessFixture(filepath.Join(control, "allow-lifetime")) {
			os.Exit(3)
		}
		_ = os.WriteFile(os.Getenv("INSONIC_STARTUP_MARKER"), []byte("survived"), 0600)
	case "ready":
		fmt.Println("{\"state\":\"ready\"}")
	case "large":
		fmt.Println(strings.Repeat("secret-output-fixture", 500))
	case "hang":
		if !helperProcessFixture(filepath.Join(control, "never")) {
			os.Exit(3)
		}
	case "hold":
		if !helperProcessFixture(filepath.Join(control, "release-holder")) {
			os.Exit(3)
		}
	case "exit-with-holder":
		exe, _ := os.Executable()
		child := exec.Command(exe, "-test.run=TestStartupHelper", "--", "hold")
		Hide(child, false)
		child.Stdout = os.Stdout
		if child.Start() != nil {
			os.Exit(2)
		}
		if !helperProcessFixture(filepath.Join(control, "holder-ready")) {
			os.Exit(3)
		}
	case "exit":
	}
	os.Exit(0)
}

func startupFixture(t *testing.T) (string, string) {
	t.Helper()
	t.Setenv("INSONIC_STARTUP_HELPER", "1")
	control := t.TempDir()
	t.Setenv("INSONIC_STARTUP_CONTROL_DIRECTORY", control)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return exe, control
}

func TestReportedStartupEarlyExitAndBounds(t *testing.T) {
	for _, mode := range []string{"exit", "large", "hang", "ready"} {
		t.Run(mode, func(t *testing.T) {
			exe, control := startupFixture(t)
			ctx, cancel := context.WithCancel(processTestContext(t))
			defer cancel()
			start := time.Now()
			type result struct {
				raw []byte
				err error
			}
			done := make(chan result, 1)
			go func() {
				raw, err := startupReported(ctx, exe, []string{"-test.run=TestStartupHelper", "--", mode}, []byte(strings.Repeat("private-input", 8192)), false, validateFixtureReady)
				done <- result{raw, err}
			}()
			pid := requireProcessFixture(t, ctx, filepath.Join(control, "started"))
			// Cancellation starts after a controlled child actually starts.
			if mode == "hang" {
				cancel()
			}
			got := <-done
			if time.Since(start) > qualificationtiming.Duration("readiness_seconds")+qualificationtiming.Duration("cleanup_seconds") {
				t.Fatal("unbounded startup", mode)
			}
			if mode == "ready" {
				if got.err != nil || !strings.Contains(string(got.raw), "ready") {
					t.Fatal(mode, got.err, string(got.raw))
				}
			} else {
				if got.err == nil {
					t.Fatal("invalid startup admitted", mode)
				}
				safe := contracts.SafeError(got.err)
				if strings.Contains(safe.Error(), "secret-output-fixture") || strings.Contains(safe.Error(), "private-input") {
					t.Fatal("secret leaked")
				}
			}
			requireFixtureProcessStopped(t, processTestContext(t), pid)
		})
	}
}

func TestChildExitIsObservedWhenDescendantRetainsReportPipe(t *testing.T) {
	exe, control := startupFixture(t)
	ctx := processTestContext(t)
	defer os.WriteFile(filepath.Join(control, "release-holder"), []byte("release owned fixture"), 0600)
	done := make(chan error, 1)
	go func() {
		_, err := startupReported(ctx, exe, []string{"-test.run=TestStartupHelper", "--", "exit-with-holder"}, nil, false, validateFixtureReady)
		done <- err
	}()
	pid := requireProcessFixture(t, ctx, filepath.Join(control, "holder-ready"))
	err := <-done
	typed, ok := err.(*contracts.Error)
	if !ok || typed.Reason != "startup_exit" {
		t.Fatal("root exit waited on inherited report handle", err)
	}
	// The still-live holder retains stdout, proving root-exit observation
	// without claiming that process startup always finishes within 400ms.
	number, _ := strconv.Atoi(string(pid))
	if stopped, probeErr := fixtureProcessStopped(number); probeErr != nil || stopped {
		t.Fatal("report pipe fixture did not remain live", stopped, probeErr)
	}
	if err := os.WriteFile(filepath.Join(control, "release-holder"), []byte("release owned fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	requireFixtureProcessStopped(t, processTestContext(t), pid)
}

func TestUnreadyStartupChildDoesNotSurviveDeadline(t *testing.T) {
	exe, control := startupFixture(t)
	marker := filepath.Join(control, "marker")
	t.Setenv("INSONIC_STARTUP_MARKER", marker)
	// Deliberate pre-readiness expiry tests cancellation authority, not healthy
	// child-startup performance. A child need not reach its started marker.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := startupReported(ctx, exe, []string{"-test.run=TestStartupHelper", "--", "lifetime-hang"}, nil, false, validateFixtureReady)
	if err == nil || ctx.Err() != context.DeadlineExceeded {
		t.Fatal("unready child admitted", err)
	}
	if pid, readErr := os.ReadFile(filepath.Join(control, "started")); readErr == nil {
		requireFixtureProcessStopped(t, processTestContext(t), pid)
	}
	if err := os.WriteFile(filepath.Join(control, "allow-lifetime"), []byte("release owned fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("unready child survived startup failure")
	}
}

func validateFixtureReady(raw []byte) error {
	if strings.TrimSpace(string(raw)) != "{\"state\":\"ready\"}" {
		return contracts.Condition("unavailable", "startup_report")
	}
	return nil
}

func TestStartupChildLifetimeFollowsValidatedReadiness(t *testing.T) {
	for _, mode := range []string{"lifetime-invalid", "lifetime-ready", "lifetime-hang"} {
		t.Run(mode, func(t *testing.T) {
			exe, control := startupFixture(t)
			marker := filepath.Join(control, "marker")
			t.Setenv("INSONIC_STARTUP_MARKER", marker)
			ctx, cancel := context.WithCancel(processTestContext(t))
			defer cancel()
			done := make(chan error, 1)
			go func() {
				_, err := startupReported(ctx, exe, []string{"-test.run=TestStartupHelper", "--", mode}, nil, false, validateFixtureReady)
				done <- err
			}()
			pid := requireProcessFixture(t, ctx, filepath.Join(control, "started"))
			if mode == "lifetime-hang" {
				cancel()
			}
			err := <-done
			cancel()
			if (err == nil) != (mode == "lifetime-ready") {
				t.Fatal(mode, err)
			}
			if mode != "lifetime-ready" {
				requireFixtureProcessStopped(t, processTestContext(t), pid)
			}
			if err := os.WriteFile(filepath.Join(control, "allow-lifetime"), []byte("release owned fixture"), 0600); err != nil {
				t.Fatal(err)
			}
			if mode == "lifetime-ready" {
				requireProcessFixture(t, processTestContext(t), marker)
				requireFixtureProcessStopped(t, processTestContext(t), pid)
			} else if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
				t.Fatal("unready child survived", mode)
			}
		})
	}
}
