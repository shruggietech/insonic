// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"testing"
)

func TestRosterSharedRuntimeAudioOnly(t *testing.T) {
	w, e := workspace.Init(t.TempDir(), "roster runtime")
	if e != nil {
		t.Fatal(e)
	}
	a, e := New(w)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	id := importRecordingFixture(t, a)
	sp, e := a.Catalog.PutSpeaker(context.Background(), contracts.ID(), 0, catalog.SpeakerIdentity{Speaker: catalog.Speaker{ID: contracts.ID(), Name: "Known", State: "active"}})
	if e != nil {
		t.Fatal(e)
	}
	read := realRequest(a, "recordings.roster.show", id, nil)
	if read.Error != nil || read.Result.(catalog.Roster).Declared {
		t.Fatalf("read %+v", read)
	}
	for _, mode := range []string{"add", "remove"} {
		empty := realRequest(a, "recordings.roster."+mode, id, map[string]any{"expected_revision": 0, "speakers": []string{}})
		if empty.Error == nil || empty.Error.Code != "invalid_request" {
			t.Fatal("empty roster mutation accepted", mode)
		}
	}
	read = realRequest(a, "recordings.roster.show", id, nil)
	if read.Error != nil || read.Result.(catalog.Roster).Declared {
		t.Fatal("blank edit changed roster authority")
	}
	add := realRequest(a, "recordings.roster.add", id, map[string]any{"expected_revision": 0, "speakers": []string{sp.Speaker.ID}})
	if add.Error != nil {
		t.Fatal(add.Error)
	}
	r := add.Result.(catalog.Roster)
	if len(r.Members) != 1 {
		t.Fatal("member")
	}
	bad := realRequest(a, "recordings.roster.clear", id, map[string]any{"speakers": []string{}})
	if bad.Error == nil {
		t.Fatal("missing expected revision accepted")
	}
	stale := realRequest(a, "recordings.roster.clear", id, map[string]any{"expected_revision": 0, "speakers": []string{}})
	if stale.Error == nil || stale.Error.Code != "conflict" {
		t.Fatal("stale", stale.Error)
	}
	clear := realRequest(a, "recordings.roster.clear", id, map[string]any{"expected_revision": r.Revision, "speakers": []string{}})
	if clear.Error != nil || !clear.Result.(catalog.Roster).Declared || len(clear.Result.(catalog.Roster).Members) != 0 {
		t.Fatal("clear", clear.Error)
	}
}
