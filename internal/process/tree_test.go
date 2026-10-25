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

	"github.com/shruggietech/insonic/internal/qualificationtiming"
)

func TestTreeHelper(t *testing.T) {
	if os.Getenv("INSONIC_TREE_HELPER") != "1" {
		return
	}
	args := os.Args
	for i, arg := range args {
		if arg == "--worker" {
			_ = os.WriteFile(args[i+1]+".pid", []byte(strconv.Itoa(os.Getpid())), 0600)
			for n := 0; ; n++ {
				_ = os.WriteFile(args[i+1], []byte(strconv.Itoa(n)), 0600)
				time.Sleep(20 * time.Millisecond)
			}
		}
	}
	file := args[len(args)-1]
	exe, _ := os.Executable()
	worker := exec.Command(exe, "-test.run=^TestTreeHelper$", "--", "--worker", file)
	worker.Stdout, worker.Stderr = os.Stdout, os.Stderr
	Hide(worker, false)
	if err := worker.Start(); err != nil {
		os.Exit(2)
	}
	if !helperProcessFixture(file) {
		os.Exit(3)
	}
	if strings.Contains(strings.Join(args, " "), "--overflow") {
		fmt.Print(strings.Repeat("fixture-secret", 1000))
	}
	time.Sleep(qualificationtiming.Duration("readiness_seconds") + qualificationtiming.Duration("cleanup_seconds"))
	os.Exit(0)
}

func TestCancellationAndOutputLimitTerminateDescendants(t *testing.T) {
	for _, mode := range []string{"--wait", "--overflow"} {
		t.Run(mode, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "heartbeat")
			exe, _ := os.Executable()
			ctx, cancel := context.WithCancel(processTestContext(t))
			defer cancel()
			done, finished := make(chan error, 1), make(chan struct{})
			go func() {
				_, err := Run(ctx, Spec{Executable: exe, Args: []string{"-test.run=^TestTreeHelper$", "--", mode, file}, Env: []string{"INSONIC_TREE_HELPER=1"}, MaxOutput: 64})
				done <- err
				close(finished)
			}()
			t.Cleanup(func() {
				cancel()
				select {
				case <-finished:
				case <-time.After(qualificationtiming.Duration("cleanup_seconds")):
					t.Error("owned tree supervision did not finish")
				}
			})
			requireProcessFixture(t, ctx, file)
			pid := requireProcessFixture(t, ctx, file+".pid")
			if mode == "--wait" {
				cancel()
			}
			err := <-done
			if err == nil || strings.Contains(err.Error(), "fixture-secret") {
				t.Fatal("supervision/redaction", err)
			}
			before, err := os.ReadFile(file)
			if err != nil {
				t.Fatal("descendant did not start", err)
			}
			requireFixtureProcessStopped(t, processTestContext(t), pid)
			after, err := os.ReadFile(file)
			if err != nil || string(before) != string(after) {
				t.Fatal("descendant survived supervision")
			}
		})
	}
}
