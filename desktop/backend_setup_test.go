// SPDX-License-Identifier: Apache-2.0
package desktop

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/graph"
	"github.com/shruggietech/insonic/internal/workspace"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBackendBootstrapParityAndFailurePreservesSelection(t *testing.T) {
	old, e := workspace.Init(t.TempDir(), "Previous")
	if e != nil {
		t.Fatal(e)
	}
	b := &Bridge{Workspace: old, backendGraphFactory: func(context.Context, *workspace.Workspace, contracts.SecretProvider) (graph.Adapter, error) {
		return &bootstrapGraph{}, nil
	}}
	planned := b.BackendSetup("plan", "", "{\"kind\":\"backend-plan\",\"display_name\":\"Configured\"}", "{}", "native", contracts.ID())
	if planned.Error != nil {
		t.Fatal(planned.Error)
	}
	bytes, _ := json.Marshal(planned.Result)
	var out struct {
		Configuration workspace.Config `json:"configuration"`
	}
	if json.Unmarshal(bytes, &out) != nil {
		t.Fatal("invalid plan")
	}
	raw, _ := json.Marshal(out.Configuration)
	target := filepath.Join(t.TempDir(), "absent")
	probe := b.BackendSetup("preflight", target, string(raw), "{}", "native", contracts.ID())
	if probe.Error != nil {
		t.Fatal(probe.Error)
	}
	if _, e = os.Stat(target); !os.IsNotExist(e) {
		t.Fatal("probe created target")
	}
	if b.Workspace.Config.WorkspaceID != old.Config.WorkspaceID {
		t.Fatal("probe changed selection")
	}
	conflict := b.BackendSetup("provision", old.Root, string(raw), "{}", "native", contracts.ID())
	if conflict.Error == nil || b.Workspace.Config.WorkspaceID != old.Config.WorkspaceID {
		t.Fatal("failed provision changed selection")
	}
	success := b.BackendSetup("provision", target, string(raw), "{}", "native", contracts.ID())
	if success.Error != nil || b.Workspace.Config.WorkspaceID != out.Configuration.WorkspaceID {
		t.Fatal(success.Error)
	}
}
func TestOperationCancellationOwnsUnderlyingContext(t *testing.T) {
	b := &Bridge{}
	id := contracts.ID()
	ctx, finish, e := b.beginOperation(id, time.Second)
	if e != nil {
		t.Fatal(e)
	}
	defer finish()
	if !b.CancelOperation(id) {
		t.Fatal("missing operation")
	}
	if ctx.Err() != context.Canceled {
		t.Fatal("context not cancelled")
	}
	if _, _, e = b.beginOperation(id, time.Second); e == nil {
		t.Fatal("duplicate ownership")
	}
}

type bootstrapGraph struct{ graph.Adapter }

func (*bootstrapGraph) Capabilities() map[string]any {
	return map[string]any{"backend_version": "fixture"}
}
func (*bootstrapGraph) EnsureSchema(context.Context) error          { return nil }
func (*bootstrapGraph) BindWorkspace(context.Context, string) error { return nil }
func (*bootstrapGraph) Close()                                      {}
