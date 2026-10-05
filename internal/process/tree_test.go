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
)

func TestTreeHelper(t *testing.T) {
	if os.Getenv("INSONIC_TREE_HELPER") != "1" {
		return
	}
	args := os.Args
	for i, arg := range args {
		if arg == "--worker" {
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
	for {
		if _, err := os.Stat(file); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if strings.Contains(strings.Join(args, " "), "--overflow") {
		fmt.Print(strings.Repeat("fixture-secret", 1000))
	}
	time.Sleep(time.Minute)
	os.Exit(0)
}

func TestCancellationAndOutputLimitTerminateDescendants(t *testing.T) {
	for _, mode := range []string{"--wait", "--overflow"} {
		t.Run(mode, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "heartbeat")
			exe, _ := os.Executable()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_, err := Run(ctx, Spec{Executable: exe, Args: []string{"-test.run=^TestTreeHelper$", "--", mode, file}, Env: []string{"INSONIC_TREE_HELPER=1"}, MaxOutput: 64})
			if err == nil || strings.Contains(err.Error(), "fixture-secret") {
				t.Fatal("supervision/redaction", err)
			}
			before, err := os.ReadFile(file)
			if err != nil {
				t.Fatal("descendant did not start", err)
			}
			time.Sleep(120 * time.Millisecond)
			after, err := os.ReadFile(file)
			if err != nil || string(before) != string(after) {
				t.Fatal("descendant survived supervision")
			}
		})
	}
}
