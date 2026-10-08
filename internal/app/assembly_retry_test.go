// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/contracts"
)

func TestAssemblyWorkRetryRequiresInputResubmissionWithoutChangingState(t *testing.T) {
	for _, state := range []string{"failed", "interrupted"} {
		t.Run(state, func(t *testing.T) {
			a := configuredApp(t)
			ctx := context.Background()
			id, owner := contracts.ID(), contracts.ID()
			if _, e := a.Catalog.EnqueueWork(ctx, id, "recordings.assemble", json.RawMessage(`{}`)); e != nil {
				t.Fatal(e)
			}
			claim, e := a.Catalog.ClaimWork(ctx, id, owner, time.Minute)
			if e != nil {
				t.Fatal(e)
			}
			if state == "failed" {
				_, e = a.Catalog.CheckpointWork(ctx, claim, "assembly-failed", state, json.RawMessage(`{"retry":"resubmit-ephemeral-input"}`), time.Minute)
			} else {
				e = a.Catalog.InterruptOwner(ctx, owner)
			}
			if e != nil {
				t.Fatal(e)
			}
			before, e := a.Catalog.Work(ctx, id)
			if e != nil || before.State != state {
				t.Fatal("fixture did not enter selected durable state", before.State, e)
			}
			out := realRequest(a, "work.retry", id, nil)
			if out.Error == nil || out.Error.Code != "invalid_request" || !strings.Contains(out.Error.Message, "recordings.assemble") {
				t.Fatal("generic retry did not explain assembly input resubmission", out)
			}
			after, e := a.Catalog.Work(ctx, id)
			if e != nil || !reflect.DeepEqual(before, after) {
				t.Fatal("rejected retry changed durable work", before, after, e)
			}
		})
	}
}
