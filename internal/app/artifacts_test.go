// SPDX-License-Identifier: Apache-2.0
package app

import (
	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"os"
	"path/filepath"
	"testing"
)

func TestArtifactRuntimeDispatch(t *testing.T) {
	w, e := workspace.Init(t.TempDir(), "artifact cli")
	if e != nil {
		t.Fatal(e)
	}
	a, e := New(w)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	source := filepath.Join(t.TempDir(), "音声.txt")
	if e = os.WriteFile(source, []byte("source"), 0600); e != nil {
		t.Fatal(e)
	}
	request := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: w.Config.WorkspaceID, RequestID: contracts.ID(), Operation: "artifacts.publish", SourcePath: source, ArtifactKind: "other"}
	r := a.Dispatch(request)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	p, ok := r.Result.(artifact.Receipt)
	if !ok || p.State != "available" {
		t.Fatalf("publication %+v", r.Result)
	}
	bad := request
	bad.WorkspaceID = contracts.ID()
	bad.RequestID = contracts.ID()
	if a.Dispatch(bad).Error == nil {
		t.Fatal("cross-workspace publication")
	}
	bad = request
	bad.Operation = "artifacts.verify"
	bad.PublicationID = p.ID
	if a.Dispatch(bad).Error == nil {
		t.Fatal("unrelated source fields accepted")
	}
	request = contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: w.Config.WorkspaceID, RequestID: contracts.ID(), Operation: "artifacts.verify", PublicationID: p.ID}
	if r = a.Dispatch(request); r.Error != nil {
		t.Fatal(r.Error)
	}
}
