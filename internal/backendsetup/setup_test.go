// SPDX-License-Identifier: Apache-2.0
package backendsetup

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
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
	w, e := Provision(context.Background(), root, c, nil)
	if e != nil {
		t.Fatal(e)
	}
	old, _ := os.ReadFile(filepath.Join(w.Control, "workspace.json"))
	again, e := Provision(context.Background(), root, c, nil)
	if e != nil || again.Config.WorkspaceID != w.Config.WorkspaceID {
		t.Fatal("retry", e)
	}
	c.DisplayName = "Changed"
	if _, e = Provision(context.Background(), root, c, nil); e == nil {
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
