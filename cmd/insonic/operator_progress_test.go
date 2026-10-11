// SPDX-License-Identifier: Apache-2.0
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"strings"
	"testing"
	"time"
)

func TestChangedWorkObservationsAndGeneratedFailureCause(t *testing.T) {
	id := contracts.ID()
	w := &workspace.Workspace{}
	w.Config.WorkspaceID = contracts.ID()
	states := []string{"pending", "running", "running", "failed"}
	calls := 0
	observations := []string{}
	raw, _ := json.Marshal(map[string]any{"cause": map[string]string{"code": "conflict", "reason": "source_map_incompatible", "message": "untrusted-body"}})
	call := func(context.Context, *workspace.Workspace, contracts.Request) (contracts.Response, error) {
		state := states[calls]
		calls++
		return contracts.Response{Result: catalog.Work{ID: id, State: state, Phase: "import", Error: "conflict", Result: raw}}, nil
	}
	_, err := waitForWork(context.Background(), w, id, contracts.ID(), time.Millisecond, call, func(item catalog.Work) error { observations = append(observations, item.State); return nil })
	typed, ok := err.(*contracts.Error)
	if !ok || typed.Reason != "source_map_incompatible" || strings.Contains(typed.Message, "untrusted-body") || len(observations) != 3 {
		t.Fatal(observations, err)
	}
}
func TestProgressDoesNotPolluteRedirectedOutput(t *testing.T) {
	var out bytes.Buffer
	observer := workProgressObserver(&out, false, false, false)
	if observer != nil {
		t.Fatal("redirected output enables progress")
	}
	observer = workProgressObserver(&out, true, false, false)
	if observer == nil {
		t.Fatal("terminal progress disabled")
	}
	raw, _ := json.Marshal(map[string]any{"items": []any{map[string]string{"state": "admitted"}, map[string]string{"state": "failed"}}})
	if err := observer(catalog.Work{ID: contracts.ID(), State: "running", Phase: "hostile\x1b[2Jphase", Result: raw}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "\x1b") || !strings.Contains(out.String(), "running") || !strings.Contains(out.String(), "2 observed") {
		t.Fatal(out.String())
	}
	if workProgressObserver(&out, true, true, false) != nil || workProgressObserver(&out, true, false, true) != nil {
		t.Fatal("machine/quiet progress enabled")
	}
}
