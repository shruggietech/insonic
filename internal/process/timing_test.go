// SPDX-License-Identifier: Apache-2.0
package process

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/qualificationtiming"
)

func processTestContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := qualificationtiming.Context(context.Background(), "readiness_seconds")
	t.Cleanup(cancel)
	return ctx
}

func waitProcessFixture(ctx context.Context, path string) ([]byte, error) {
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		raw, err := os.ReadFile(path)
		if err == nil && len(raw) > 0 {
			return raw, nil
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("fixture did not become ready: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func requireProcessFixture(t *testing.T, ctx context.Context, path string) []byte {
	t.Helper()
	raw, err := waitProcessFixture(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func helperProcessFixture(path string) bool {
	ctx, stop := context.WithTimeout(context.Background(), qualificationtiming.Duration("readiness_seconds")+qualificationtiming.Duration("cleanup_seconds"))
	defer stop()
	_, err := waitProcessFixture(ctx, path)
	return err == nil
}

func requireFixtureProcessStopped(t *testing.T, ctx context.Context, rawPID []byte) {
	t.Helper()
	pid, err := strconv.Atoi(string(rawPID))
	if err != nil || pid < 1 {
		t.Fatal("invalid controlled fixture process identity")
	}
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		stopped, err := fixtureProcessStopped(pid)
		if err != nil {
			t.Fatal("fixture process observation failed", err)
		}
		if stopped {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatal("controlled fixture process survived supervision", ctx.Err())
		case <-ticker.C:
		}
	}
}
