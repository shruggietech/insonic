// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/workspace"
)

func TestFailedBatchItemsRemainRetryableWithoutDuplicatingAdmission(t *testing.T) {
	fixture := os.Getenv("INSONIC_LIBRARY_TOOLS_FILE")
	if fixture == "" {
		t.Skip("native canonical tools fixture not selected")
	}
	w, e := workspace.Init(t.TempDir(), "partial retry")
	if e != nil {
		t.Fatal(e)
	}
	a, e := New(w)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	config, e := os.ReadFile(fixture)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(w.Control, "media-tools.json"), config, 0600); e != nil {
		t.Fatal(e)
	}
	data, e := os.ReadFile("../../tests/fixtures/media/speech.flac")
	if e != nil {
		t.Fatal(e)
	}
	dir := t.TempDir()
	first, second := filepath.Join(dir, "one.wav"), filepath.Join(dir, "two.wav")
	if e = os.WriteFile(first, data, 0600); e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(library.ImportRequest{Items: []library.Item{{Source: first}, {Source: second, NewEntry: true}}})
	request := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: w.Config.WorkspaceID, RequestID: contracts.ID(), Operation: "media.import", Data: raw}
	r := a.Dispatch(request)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	id := r.Result.(map[string]any)["work_id"].(string)
	done := awaitWork(t, a, id)
	var result library.ImportResult
	if done.State != "failed" || json.Unmarshal(done.Result, &result) != nil || len(result.Items) != 2 || result.Items[0].State != "admitted" || result.Items[1].State != "failed" {
		t.Fatal("partial failure not retryable with results", done.State, string(done.Result))
	}
	firstID := result.Items[0].MediaID
	if e = os.Remove(first); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(second, data, 0600); e != nil {
		t.Fatal(e)
	}
	if r = realRequest(a, "work.retry", id, nil); r.Error != nil {
		t.Fatal(r.Error)
	}
	done = awaitWork(t, a, id)
	if done.State != "succeeded" || json.Unmarshal(done.Result, &result) != nil || result.Items[0].MediaID != firstID || result.Items[1].State != "admitted" {
		t.Fatal("retry failed or duplicated first item", done.State, string(done.Result))
	}
	replay := a.Dispatch(request)
	if replay.Error != nil || replay.Result.(map[string]any)["work_id"] != id {
		t.Fatal("accepted input scrubbing broke request replay", replay)
	}
	request.Data, _ = json.Marshal(library.ImportRequest{Items: []library.Item{{Source: first, NewEntry: true}}})
	if changed := a.Dispatch(request); changed.Error == nil || changed.Error.Code != "conflict" {
		t.Fatal("request identity collision accepted", changed)
	}
	entries, e := a.Catalog.Libraries(context.Background())
	if e != nil || len(entries) != 2 {
		t.Fatal("duplicate batch admissions", len(entries), e)
	}
}
