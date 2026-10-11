// SPDX-License-Identifier: Apache-2.0
package process

import (
	"context"
	"fmt"
	"github.com/shruggietech/insonic/internal/contracts"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestStartupHelper(t *testing.T) {
	if os.Getenv("INSONIC_STARTUP_HELPER") != "1" {
		return
	}
	mode := os.Args[len(os.Args)-1]
	switch mode {
	case "lifetime-hang", "lifetime-invalid", "lifetime-ready":
		if mode == "lifetime-ready" {
			fmt.Println(`{"state":"ready"}`)
		}
		if mode == "lifetime-invalid" {
			fmt.Println(`{"state":"invalid"}`)
		}
		time.Sleep(350 * time.Millisecond)
		_ = os.WriteFile(os.Getenv("INSONIC_STARTUP_MARKER"), []byte("survived"), 0600)
	case "ready":
		fmt.Println("{\"state\":\"ready\"}")
	case "large":
		fmt.Println(strings.Repeat("secret-output-fixture", 500))
	case "hang":
		time.Sleep(500 * time.Millisecond)
	case "hold":
		time.Sleep(time.Second)
	case "exit-with-holder":
		exe, _ := os.Executable()
		child := exec.Command(exe, "-test.run=TestStartupHelper", "--", "hold")
		Hide(child, false)
		child.Stdout = os.Stdout
		if child.Start() != nil {
			os.Exit(2)
		}
	case "exit":
	}
	os.Exit(0)
}
func TestReportedStartupEarlyExitAndBounds(t *testing.T) {
	t.Setenv("INSONIC_STARTUP_HELPER", "1")
	exe, _ := os.Executable()
	for _, mode := range []string{"exit", "large", "hang", "ready"} {
		limit := time.Second
		if mode == "hang" {
			limit = 100 * time.Millisecond
		}
		ctx, cancel := context.WithTimeout(context.Background(), limit)
		start := time.Now()
		raw, e := startupReported(ctx, exe, []string{"-test.run=TestStartupHelper", "--", mode}, []byte(strings.Repeat("private-input", 8192)), false, validateFixtureReady)
		cancel()
		if time.Since(start) > 2*time.Second {
			t.Fatal("unbounded startup", mode)
		}
		if mode == "ready" {
			if e != nil || !strings.Contains(string(raw), "ready") {
				t.Fatal(mode, e, string(raw))
			}
		} else {
			if e == nil {
				t.Fatal("invalid startup admitted", mode)
			}
			safe := contracts.SafeError(e)
			if strings.Contains(safe.Error(), "secret-output-fixture") || strings.Contains(safe.Error(), "private-input") {
				t.Fatal("secret leaked")
			}
		}
	}
}

func TestChildExitIsObservedWhenDescendantRetainsReportPipe(t *testing.T) {
	t.Setenv("INSONIC_STARTUP_HELPER", "1")
	exe, _ := os.Executable()
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	begin := time.Now()
	_, err := startupReported(ctx, exe, []string{"-test.run=TestStartupHelper", "--", "exit-with-holder"}, nil, false, validateFixtureReady)
	typed, ok := err.(*contracts.Error)
	if !ok || typed.Reason != "startup_exit" || time.Since(begin) > 400*time.Millisecond {
		t.Fatal("child exit waited on inherited report handle", err)
	}
}

func TestUnreadyStartupChildDoesNotSurviveDeadline(t *testing.T) {
	t.Setenv("INSONIC_STARTUP_HELPER", "1")
	marker := t.TempDir() + "/marker"
	t.Setenv("INSONIC_STARTUP_MARKER", marker)
	exe, _ := os.Executable()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := startupReported(ctx, exe, []string{"-test.run=TestStartupHelper", "--", "lifetime-hang"}, nil, false, validateFixtureReady)
	if err == nil {
		t.Fatal("unready child admitted")
	}
	time.Sleep(450 * time.Millisecond)
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
	t.Setenv("INSONIC_STARTUP_HELPER", "1")
	exe, _ := os.Executable()
	for _, mode := range []string{"lifetime-invalid", "lifetime-ready", "lifetime-hang"} {
		t.Run(mode, func(t *testing.T) {
			marker := t.TempDir() + "/marker"
			t.Setenv("INSONIC_STARTUP_MARKER", marker)
			ctx, cancel := context.WithCancel(context.Background())
			if mode == "lifetime-hang" {
				go func() { time.Sleep(100 * time.Millisecond); cancel() }()
			}
			_, err := startupReported(ctx, exe, []string{"-test.run=TestStartupHelper", "--", mode}, nil, false, validateFixtureReady)
			cancel()
			if (err == nil) != (mode == "lifetime-ready") {
				t.Fatal(mode, err)
			}
			time.Sleep(450 * time.Millisecond)
			_, statErr := os.Stat(marker)
			if mode == "lifetime-ready" && statErr != nil {
				t.Fatal("ready owner did not survive caller cancellation", statErr)
			}
			if mode != "lifetime-ready" && !os.IsNotExist(statErr) {
				t.Fatal("unready child survived", mode)
			}
		})
	}
}
