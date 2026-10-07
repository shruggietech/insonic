// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/workspace"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func realRequest(a *App, op, id string, data any) contracts.Response {
	var raw json.RawMessage
	if data != nil {
		raw, _ = json.Marshal(data)
	}
	return a.Dispatch(contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: a.Workspace.Config.WorkspaceID, RequestID: contracts.ID(), Operation: op, ItemID: id, Data: raw})
}
func awaitWork(t *testing.T, a *App, id string) catalog.Work {
	t.Helper()
	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) {
		w, e := a.Catalog.Work(context.Background(), id)
		if e != nil {
			t.Fatal(e)
		}
		if w.State == "succeeded" || w.State == "failed" {
			return w
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("work did not finish")
	return catalog.Work{}
}
func TestRealImportExecutorRestartAndPartialCapture(t *testing.T) {
	w, e := workspace.Init(t.TempDir(), "real work")
	if e != nil {
		t.Fatal(e)
	}
	source := filepath.Join(t.TempDir(), "original.wav")
	if e = os.WriteFile(source, []byte("original fixture without extractor"), 0600); e != nil {
		t.Fatal(e)
	}
	a, e := New(w)
	if e != nil {
		t.Fatal(e)
	}
	r := realRequest(a, "media.import", "", library.ImportRequest{Items: []library.Item{{Source: source}}})
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	workID := r.Result.(map[string]any)["work_id"].(string)
	done := awaitWork(t, a, workID)
	if done.State != "succeeded" {
		t.Fatalf("executor %+v", done)
	}
	var result library.ImportResult
	if json.Unmarshal(done.Result, &result) != nil || !result.Partial || len(result.Items) != 1 || result.Items[0].MediaID == "" {
		t.Fatalf("partial admission %s", done.Result)
	}
	id := result.Items[0].MediaID
	a.Close()
	os.Remove(source)
	a, e = New(w)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	show := realRequest(a, "media.show", id, nil)
	if show.Error != nil || show.Result.(library.EntryView).Availability != "available" {
		t.Fatalf("restart show %+v", show)
	}
	old, e := a.Catalog.Library(context.Background(), id)
	if e != nil {
		t.Fatal(e)
	}
	corrected := realRequest(a, "media.set-origin", id, map[string]any{"revision": old.Revision, "options": library.Options{OriginatedAt: "2026-11-01T01:30:00", Timezone: "America/New_York", DSTFold: "later"}})
	if corrected.Error != nil {
		t.Fatal(corrected.Error)
	}
	refresh := realRequest(a, "media.refresh", id, library.Options{})
	if refresh.Error != nil {
		t.Fatal(refresh.Error)
	}
	updated := awaitWork(t, a, refresh.Result.(map[string]any)["work_id"].(string))
	if updated.State != "succeeded" {
		t.Fatalf("refresh %+v", updated)
	}
	if r = realRequest(a, "media.show", contracts.ID(), nil); r.Error == nil {
		t.Fatal("missing id admitted")
	}
}
func TestVaultAndSessionLiveCredentialBootstrap(t *testing.T) {
	w, e := workspace.Init(t.TempDir(), "bootstrap")
	if e != nil {
		t.Fatal(e)
	}
	a, e := New(w)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	id := contracts.ID()
	r := realRequest(a, "credentials.select", "", map[string]any{"arguments": []string{"select", "session"}})
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	r = realRequest(a, "credentials.add", "", map[string]any{"arguments": []string{"add", id}, "input": map[string]any{"value": "fixture-secret"}})
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	bytes, e := a.secrets.Resolve(context.Background(), id)
	if e != nil || string(bytes) != "fixture-secret" {
		t.Fatal("live credential unusable")
	}
	clear(bytes)
	raw, _ := json.Marshal(r)
	if string(raw) == "" || containsSecret(raw) {
		t.Fatal("saved value returned")
	}
}
func containsSecret(raw []byte) bool {
	for i := 0; i+14 <= len(raw); i++ {
		if string(raw[i:i+14]) == "fixture-secret" {
			return true
		}
	}
	return false
}
