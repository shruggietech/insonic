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
		raw, e := startupReported(ctx, exe, []string{"-test.run=TestStartupHelper", "--", mode}, []byte(strings.Repeat("private-input", 8192)), false)
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
	_, err := startupReported(ctx, exe, []string{"-test.run=TestStartupHelper", "--", "exit-with-holder"}, nil, false)
	typed, ok := err.(*contracts.Error)
	if !ok || typed.Reason != "startup_exit" || time.Since(begin) > 400*time.Millisecond {
		t.Fatal("child exit waited on inherited report handle", err)
	}
}
