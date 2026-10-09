// SPDX-License-Identifier: Apache-2.0
package app

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/models"
)

func encodedModelTarget(target catalog.ModelTarget) json.RawMessage {
	raw, _ := json.Marshal(target)
	return raw
}

func TestReferenceRuntimeCRUDInspectionDoesNotAcquire(t *testing.T) {
	a := configuredApp(t)
	m := dependencyManifest(t, "1", http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("inspection downloaded model bytes") }))
	registerDependencyManifest(t, a, m)
	before, err := a.Catalog.Works(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	alias := catalog.ModelAlias{ID: contracts.ID(), Name: "speech", State: "active", Target: encodedModelTarget(catalog.ModelTarget{Kind: "base", ID: models.InstallationID(m), Operation: "transcription"})}
	saved := configuredResult(t, realRequest(a, "models.alias.set", alias.ID, map[string]any{"expected_revision": 0, "alias": alias}))
	configuredResult(t, realRequest(a, "models.alias.show", alias.ID, nil))
	list := configuredResult(t, realRequest(a, "models.alias.list", "", nil))
	if len(list["items"].([]any)) != 1 {
		t.Fatal("empty model list was routed outside domain")
	}
	selected := configuredResult(t, realRequest(a, "models.resolve", "", map[string]any{"reference": "speech", "operation": "transcription"}))
	if selected["state"] != "registered" || selected["manifest_digest"] != m.Digest() || selected["compatible"] != true {
		t.Fatal("resolution mistook registration for availability", selected)
	}
	inventory := configuredResult(t, realRequest(a, "models.list", "", nil))
	choices := inventory["items"].([]any)
	if len(choices) != 1 {
		t.Fatal("model inventory lost registered choice", inventory)
	}
	choice := choices[0].(map[string]any)
	capabilities := choice["capabilities"].([]any)
	compatibility := choice["operation_compatibility"].(map[string]any)
	if len(capabilities) != 1 || capabilities[0] != "transcription" || compatibility["transcription"] != true || compatibility["diarization"] != false || compatibility["voice-matching"] != false || compatibility["speaker-model-training"] != false {
		t.Fatal("inventory failed to inspect default adapter compatibility", choice)
	}
	if _, exposed := choice["manifest"]; exposed {
		t.Fatal("bounded inventory exposed complete manifest")
	}
	if response := realRequest(a, "models.alias.set", alias.ID, map[string]any{"expected_revision": 0, "alias": alias}); response.Error == nil || response.Error.Code != "conflict" {
		t.Fatal("stale alias edit accepted", response.Error)
	}
	removed := configuredResult(t, realRequest(a, "models.alias.remove", alias.ID, map[string]any{"expected_revision": saved["revision"]}))
	if removed["state"] != "deleted" {
		t.Fatal("alias removal lost tombstone")
	}
	if response := realRequest(a, "models.resolve", "", map[string]any{"reference": "speech", "operation": "transcription"}); response.Error == nil || response.Error.Code != "not_found" {
		t.Fatal("deleted alias resolved", response.Error)
	}
	source := catalog.ModelSource{ID: contracts.ID(), Name: "examples", State: "active", URL: "https://example.org/catalog.json"}
	created := configuredResult(t, realRequest(a, "models.source.set", source.ID, map[string]any{"expected_revision": 0, "source": source}))
	configuredResult(t, realRequest(a, "models.source.show", source.ID, nil))
	sources := configuredResult(t, realRequest(a, "models.source.list", "", nil))
	if len(sources["items"].([]any)) != 1 {
		t.Fatal("empty source list was routed outside domain")
	}
	configuredResult(t, realRequest(a, "models.source.remove", source.ID, map[string]any{"expected_revision": created["revision"]}))
	after, err := a.Catalog.Works(a.ctx)
	if err != nil || len(after) != len(before) {
		t.Fatal("model/source inspection created acquisition work", err)
	}
}

func TestHostedReferenceInspectionDoesNotGrantLocalExecution(t *testing.T) {
	a := configuredApp(t)
	target := catalog.ModelTarget{Kind: "hosted", Operation: "transcription", Adapter: "insonic-http", ContractVersion: "1", Endpoint: "https://example.org/inference", RemoteModel: "pinned-model", UpstreamRevision: "provider-revision-1"}
	alias := catalog.ModelAlias{ID: contracts.ID(), Name: "remote", State: "active", Target: encodedModelTarget(target)}
	if _, err := a.Catalog.PutModelAlias(a.ctx, contracts.ID(), 0, alias); err != nil {
		t.Fatal(err)
	}
	resolved := configuredResult(t, realRequest(a, "models.resolve", "", map[string]any{"reference": "remote", "operation": "transcription"}))
	if resolved["state"] != "hosted-only" {
		t.Fatal("hosted reference advertised weights")
	}
	_, _, _, err := a.electProcessingModels(RecordingOptions{Transcription: "generate", RecognitionModelID: "remote", Diarization: "reuse"}, nil)
	if err == nil || !strings.Contains(err.Error(), "explicit configured hosted pipeline") {
		t.Fatal("hosted reference did not explain execution route", err)
	}
	works, err := a.Catalog.Works(a.ctx)
	if err != nil || len(works) != 0 {
		t.Fatal("inspection acquired nonexistent hosted weights", err)
	}
}
