// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/secrets"
	"github.com/shruggietech/insonic/internal/workspace"
	"github.com/shruggietech/insonic/schemas"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPagedLargeWorkResultsAndTypedRequests(t *testing.T) {
	w, e := workspace.Init(t.TempDir(), "pages")
	if e != nil {
		t.Fatal(e)
	}
	a, e := New(w)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	ctx := context.Background()
	work, e := a.Catalog.EnqueueWork(ctx, contracts.ID(), "media.import", json.RawMessage(`{"items":[]}`))
	if e != nil {
		t.Fatal(e)
	}
	work, e = a.Catalog.ClaimWork(ctx, work.ID, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	out := library.ImportResult{CapturedTimezone: "UTC", Items: make([]library.ItemResult, 10000)}
	for i := range out.Items {
		out.Items[i] = library.ItemResult{Ordinal: i, MediaID: contracts.ID(), State: "admitted", CaptureState: "captured", DateState: "unknown", QueuedJobIDs: []string{}}
	}
	raw, _ := json.Marshal(out)
	if _, e = a.Catalog.CheckpointWork(ctx, work, "complete", "succeeded", raw, time.Minute); e != nil {
		t.Fatal(e)
	}
	r := realRequest(a, "work.show", work.ID, nil)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	encoded, _ := json.Marshal(r)
	if len(encoded) > 1<<20 || strings.Contains(string(encoded), "\"payload\"") {
		t.Fatal("unbounded or archived payload exposed")
	}
	for start := 0; start < 10000; start += 100 {
		r = realRequest(a, "work.results", work.ID, map[string]any{"after_ordinal": start, "limit": 100})
		if r.Error != nil {
			t.Fatal(r.Error)
		}
		page := r.Result.(map[string]any)
		items := page["items"].([]library.ItemResult)
		if len(items) != 100 || items[0].Ordinal != start || items[99].Ordinal != start+99 {
			t.Fatal("missing or duplicated page")
		}
	}
	for _, op := range []string{"media.list", "models.list", "work.list", "work.results"} {
		req := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: w.Config.WorkspaceID, RequestID: contracts.ID(), Operation: op, Data: json.RawMessage(`{"limit":10}`)}
		if op == "work.results" {
			req.ItemID = work.ID
		}
		bytes, _ := json.Marshal(req)
		if e = schemas.ValidateRequest(bytes); e != nil {
			t.Fatal(e)
		}
		req.Data = json.RawMessage(`{"limit":101}`)
		bytes, _ = json.Marshal(req)
		if schemas.ValidateRequest(bytes) == nil {
			t.Fatal("unbounded page accepted")
		}
	}
}
func TestCompleteToolSupportManifestAndSelectionRollback(t *testing.T) {
	w, e := workspace.Init(t.TempDir(), "large tools")
	if e != nil {
		t.Fatal(e)
	}
	tools := library.Tools{Kind: "media-tools", Version: contracts.Version}
	for i := 0; i < 500; i++ {
		tools.ExifTool.SupportFiles = append(tools.ExifTool.SupportFiles, library.PinnedFile{Path: filepath.Join(w.Root, contracts.ID()+".pm"), SHA256: strings.Repeat("0", 64)})
	}
	raw, _ := json.Marshal(tools)
	if len(raw) < 16384 {
		t.Fatal("fixture misses former ceiling")
	}
	if e = os.WriteFile(filepath.Join(w.Control, "media-tools.json"), raw, 0600); e != nil {
		t.Fatal(e)
	}
	a, e := New(w)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	service, e := a.libraryService()
	if e != nil || len(service.Tools.ExifTool.SupportFiles) != 500 {
		t.Fatalf("support tree refused %v", e)
	}
	t.Setenv("INSONIC_SESSION_CREDENTIALS", `{"invalid":"secret"}`)
	r := realRequest(a, "credentials.select", "", map[string]any{"arguments": []string{"select", "session"}})
	if r.Error == nil {
		t.Fatal("malformed target accepted")
	}
	mode, e := secrets.ReadSelection(w)
	if e != nil || mode != "native" {
		t.Fatal("failed selection mutated workspace")
	}
}
func TestStatusIgnoresPassphraseAndUnlockCannotLoadSession(t *testing.T) {
	w, e := workspace.Init(t.TempDir(), "status")
	if e != nil {
		t.Fatal(e)
	}
	a, e := New(w)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	id := contracts.ID()
	r := realRequest(a, "credentials.select", "", map[string]any{"arguments": []string{"select", "vault"}})
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	r = realRequest(a, "credentials.add", "", map[string]any{"arguments": []string{"add", id}, "input": map[string]any{"value": "secret", "passphrase": "fixture passphrase"}})
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	r = realRequest(a, "credentials.status", "", map[string]any{"arguments": []string{"status", id}, "input": map[string]any{"passphrase": "wrong passphrase"}})
	if r.Error != nil || r.Result.(secrets.CredentialStatus).State != "configured" {
		t.Fatal("status decrypted supplied passphrase")
	}
	r = realRequest(a, "credentials.select", "", map[string]any{"arguments": []string{"select", "session"}})
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	r = realRequest(a, "credentials.unlock", "", map[string]any{"arguments": []string{"unlock"}, "input": map[string]any{"session": map[string]any{id: "secret"}}})
	if r.Error == nil {
		t.Fatal("unlock reset session")
	}
	r = realRequest(a, "credentials.load", "", map[string]any{"arguments": []string{"load"}, "input": map[string]any{"session": map[string]any{}}})
	if r.Error != nil {
		t.Fatal("explicit empty session refused")
	}
}
