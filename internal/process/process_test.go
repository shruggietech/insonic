package process

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestHelper(t *testing.T) {
	if os.Getenv("INSONIC_PROCESS_HELPER") != "1" {
		return
	}
	for _, arg := range os.Args {
		if arg == "--wait" {
			time.Sleep(time.Minute)
		}
	}
	for range 100 {
		os.Stdout.WriteString("test-secret")
	}
	os.Exit(0)
}

func TestLimitsCancellationAndNoOutputInErrors(t *testing.T) {
	exe, _ := os.Executable()
	_, err := Run(context.Background(), Spec{Executable: exe, Args: []string{"-test.run=TestHelper"}, Env: []string{"INSONIC_PROCESS_HELPER=1"}, MaxOutput: 32})
	if err == nil || strings.Contains(err.Error(), "test-secret") {
		t.Fatalf("limit/redaction: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	_, err = Run(ctx, Spec{Executable: exe, Args: []string{"-test.run=TestHelper", "--", "--wait"}, Env: []string{"INSONIC_PROCESS_HELPER=1"}, MaxOutput: 1024})
	if err == nil || ctx.Err() == nil {
		t.Fatal("cancellation failed")
	}
	_, err = Run(context.Background(), Spec{Executable: "relative-command", MaxOutput: 32})
	if err == nil {
		t.Fatal("relative executable accepted")
	}
}
