// SPDX-License-Identifier: Apache-2.0
package runtime

import (
	"context"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
)

func TestExpiredWorkerCallRetainsDeadlineWithoutDialing(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "deadline")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	_, err = Call(ctx, w, contracts.Request{Operation: "workers.inspect"})
	safe := contracts.SafeError(err)
	if safe.Reason != "worker_probe_deadline" {
		t.Fatalf("expired probe call became %s/%s", safe.Code, safe.Reason)
	}
}
