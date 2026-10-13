// SPDX-License-Identifier: Apache-2.0
package backendsetup

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/graph"
	"github.com/shruggietech/insonic/internal/workspace"
	"os"
	"path/filepath"
	"testing"
)

func TestPlanDefaultsAndSecretExclusion(t *testing.T) {
	c, e := Build([]byte("{\"kind\":\"backend-plan\",\"display_name\":\"Test\"}"))
	if e != nil {
		t.Fatal(e)
	}
	if !contracts.ValidID(c.WorkspaceID) || c.Profiles.Storage.Adapter != "filesystem" || c.Profiles.Catalog.Adapter != "sqlite" || c.Profiles.Graph.Adapter != "ladybugdb" {
		t.Fatal(c)
	}
	if _, e = Build([]byte("{\"display_name\":\"Test\",\"password\":\"secret\"}")); e == nil {
		t.Fatal("accepted secret field")
	}
	raw, _ := json.Marshal(c)
	again, e := Build(raw)
	if e != nil || again.WorkspaceID != c.WorkspaceID || again.Profiles.Catalog.ID != c.Profiles.Catalog.ID {
		t.Fatal("plan identity changed", e)
	}
}
func TestPreflightDoesNotCreateAndCancellation(t *testing.T) {
	c, _ := Build([]byte("{\"display_name\":\"Test\"}"))
	root := filepath.Join(t.TempDir(), "not-created")
	report, e := Preflight(context.Background(), root, c, nil)
	if e != nil || len(report.Roles) != 3 {
		t.Fatal(report, e)
	}
	if _, e = os.Stat(root); !os.IsNotExist(e) {
		t.Fatal("preflight created target")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e = Preflight(ctx, root, c, nil); e == nil {
		t.Fatal("ignored cancellation")
	}
}
func TestProvisionRetryAndPreservation(t *testing.T) {
	c, _ := Build([]byte("{\"display_name\":\"Test\"}"))
	root := t.TempDir()
	w, e := provisionFixture(context.Background(), root, c, nil)
	if e != nil {
		t.Fatal(e)
	}
	old, _ := os.ReadFile(filepath.Join(w.Control, "workspace.json"))
	again, e := provisionFixture(context.Background(), root, c, nil)
	if e != nil || again.Config.WorkspaceID != w.Config.WorkspaceID {
		t.Fatal("retry", e)
	}
	c.DisplayName = "Changed"
	if _, e = provisionFixture(context.Background(), root, c, nil); e == nil {
		t.Fatal("replaced config")
	}
	now, _ := os.ReadFile(filepath.Join(w.Control, "workspace.json"))
	if string(old) != string(now) {
		t.Fatal("failed provision modified config")
	}
}

func TestAllBackendPlanFieldsAndCorrections(t *testing.T) {
	cases := []string{
		"{\"display_name\":\"S3\",\"profiles\":{\"storage\":{\"adapter_id\":\"s3\",\"configuration\":{\"endpoint\":\"https://storage.example\",\"bucket\":\"fixture\",\"prefix\":\"media/\",\"region\":\"test\",\"addressing_style\":\"path\",\"authentication\":\"anonymous\"}}}}",
		"{\"display_name\":\"PG\",\"profiles\":{\"catalog\":{\"adapter_id\":\"postgresql\",\"configuration\":{\"host\":\"127.0.0.1\",\"port\":5432,\"database\":\"fixture\",\"schema\":\"fixture\",\"tls_mode\":\"local\"}}}}",
		"{\"display_name\":\"Arcade\",\"profiles\":{\"graph\":{\"adapter_id\":\"arcadedb\",\"configuration\":{\"endpoint\":\"https://graph.example\",\"database\":\"fixture\",\"preferred_dialect\":\"arcade-sql\"}}}}",
	}
	for _, raw := range cases {
		if _, e := Build([]byte(raw)); e != nil {
			t.Fatal(e)
		}
	}
	_, e := Build([]byte("{\"display_name\":\"Bad\",\"profiles\":{\"storage\":{\"adapter_id\":\"s3\",\"configuration\":{\"endpoint\":\"secret-value\"}}}}"))
	if e == nil {
		t.Fatal("invalid plan accepted")
	}
	out, _ := json.Marshal(Failure(e))
	if !bytes.Contains(out, []byte("field_errors")) || bytes.Contains(out, []byte("secret-value")) {
		t.Fatal(string(out))
	}
}

func TestPendingPlanConflictAfterBackendMutation(t *testing.T) {
	c, _ := Build([]byte("{\"display_name\":\"Pending\"}"))
	root := t.TempDir()
	svc := Provisioner{GraphFactory: func(context.Context, *workspace.Workspace, contracts.SecretProvider) (graph.Adapter, error) {
		return nil, contracts.Fail("unavailable")
	}}
	if _, e := svc.ProvisionMode(context.Background(), root, c, nil, "native"); e == nil {
		t.Fatal("graph failure accepted")
	}
	if _, e := os.Stat(filepath.Join(root, ".insonic", "catalog.sqlite")); e != nil {
		t.Fatal("did not exercise post-catalog failure", e)
	}
	different, _ := Build([]byte("{\"display_name\":\"Different\"}"))
	if _, e := svc.ProvisionMode(context.Background(), root, different, nil, "native"); e == nil || contracts.SafeError(e).Code != "conflict" {
		t.Fatal("changed pending identity admitted", e)
	}
	if _, e := os.Stat(filepath.Join(root, ".insonic", "workspace.json")); !os.IsNotExist(e) {
		t.Fatal("failed graph published config")
	}
}

func TestGraphSchemaOwnerAndVersionBeforePublication(t *testing.T) {
	for _, phase := range []string{"open", "schema", "bind", "version"} {
		t.Run(phase, func(t *testing.T) {
			c, _ := Build([]byte("{\"display_name\":\"Graph fixture\"}"))
			root := t.TempDir()
			g := &fixtureGraph{version: "actual"}
			svc := Provisioner{GraphFactory: func(context.Context, *workspace.Workspace, contracts.SecretProvider) (graph.Adapter, error) {
				if phase == "open" {
					return nil, contracts.Fail("unavailable")
				}
				return g, nil
			}}
			if phase == "schema" {
				g.schemaErr = contracts.Fail("operation_failed")
			}
			if phase == "bind" {
				g.bindErr = contracts.Fail("conflict")
			}
			if phase == "version" {
				c.Profiles.Graph.ExpectedBackendVersion = "expected"
			}
			if _, e := svc.ProvisionMode(context.Background(), root, c, nil, "native"); e == nil {
				t.Fatal("graph failure reported success")
			}
			if _, e := os.Stat(filepath.Join(root, ".insonic", "workspace.json")); !os.IsNotExist(e) {
				t.Fatal("published before graph readiness")
			}
			if phase != "open" && !g.closed {
				t.Fatal("graph handle leaked")
			}
		})
	}
	c, _ := Build([]byte("{\"display_name\":\"Graph success\"}"))
	g := &fixtureGraph{version: "actual"}
	svc := Provisioner{GraphFactory: func(context.Context, *workspace.Workspace, contracts.SecretProvider) (graph.Adapter, error) {
		return g, nil
	}}
	if _, e := svc.ProvisionMode(context.Background(), t.TempDir(), c, nil, "native"); e != nil || !g.initialized || g.bound != c.WorkspaceID || !g.closed {
		t.Fatal(g, e)
	}
}
