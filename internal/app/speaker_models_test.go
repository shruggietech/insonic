// SPDX-License-Identifier: Apache-2.0
package app

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/voicemodels"
	"testing"
	"time"
)

func TestSpeakerDatasetResponseBoundsDoNotTruncateTrainingAuthority(t *testing.T) {
	dataset := catalog.SpeakerDataset{Dataset: catalog.Dataset{ID: contracts.ID(), SpeakerID: contracts.ID()}, References: make([]catalog.CurrentReference, 201), Recipe: json.RawMessage(`{}`), Summary: json.RawMessage(`{"references":201}`)}
	view := speakerDatasetView(dataset).(map[string]any)
	if view["reference_count"] != 201 || view["references_inline"] != false || view["references"] != nil || len(dataset.References) != 201 {
		t.Fatal("response pagination damaged corpus", view)
	}
	list := speakerDatasetSummary(dataset).(map[string]any)
	if list["references"] != nil || list["recipe"] != nil {
		t.Fatal("list leaked full corpus", list)
	}
}

func TestSpeakerModelListReportsFilteredIdentityProfile(t *testing.T) {
	a := configuredApp(t)
	identity, err := a.Catalog.PutSpeaker(a.ctx, contracts.ID(), 0, catalog.SpeakerIdentity{Speaker: catalog.Speaker{ID: contracts.ID(), Name: "Profile selection"}})
	if err != nil {
		t.Fatal(err)
	}
	view := configuredResult(t, realRequest(a, "models.speaker.list", "", map[string]any{"speaker_id": identity.Speaker.ID}))
	profile := view["profile"].(map[string]any)
	if profile["id"] != identity.Speaker.ID || profile["revision"] != float64(0) {
		t.Fatal("profile belongs to another identity", profile)
	}
}

func TestSpeakerWorkReplayRetainsOriginalItemAndElection(t *testing.T) {
	a := configuredApp(t)
	request := contracts.Request{Operation: "recordings.match", ItemID: contracts.ID(), RequestID: contracts.ID(), Data: json.RawMessage(`{"model_id":"` + contracts.ID() + `"}`)}
	digest := documentHash(configurationBytes(struct {
		Item  string          `json:"item_id"`
		Input json.RawMessage `json:"input"`
	}{request.ItemID, request.Data}))
	raw, _ := json.Marshal(voicemodels.MatchPayload{RequestDigest: digest})
	if _, err := a.Catalog.EnqueueWork(a.ctx, request.RequestID, request.Operation, raw); err != nil {
		t.Fatal(err)
	}
	if _, err := a.enqueueSpeakerWork(request); err != nil {
		t.Fatal("identical intent failed replay", err)
	}
	request.ItemID = contracts.ID()
	if _, err := a.enqueueSpeakerWork(request); err == nil {
		t.Fatal("same request ID replayed another recording")
	}
}

func TestSpeakerWorkEnvelopeAdmission(t *testing.T) {
	id := contracts.ID()
	for _, tc := range []struct {
		op, item string
		data     bool
	}{
		{"models.dataset.create", "", true}, {"models.dataset.list", "", false},
		{"models.dataset.show", id, false}, {"models.train", "", true},
		{"models.speaker.list", "", false}, {"models.speaker.show", id, false},
		{"models.speaker.fetch", id, true}, {"models.profile.set", id, true},
		{"recordings.match", id, true},
	} {
		request := contracts.Request{Operation: tc.op, ItemID: tc.item}
		if tc.data {
			request.Data = json.RawMessage(`{}`)
		}
		if !speakerWorkRequestValid(request) || !domainRequestValid(request) {
			t.Fatal("valid envelope rejected", tc.op)
		}
		request.JobID = id
		if speakerWorkRequestValid(request) {
			t.Fatal("job authority accepted", tc.op)
		}
		request.JobID = ""
		request.SourcePath = "arbitrary"
		if speakerWorkRequestValid(request) {
			t.Fatal("artifact authority accepted", tc.op)
		}
		request.SourcePath = ""
		if tc.item == "" {
			request.ItemID = id
		} else {
			request.ItemID = "name"
		}
		if speakerWorkRequestValid(request) {
			t.Fatal("wrong item accepted", tc.op)
		}
	}
}

func TestSpeakerWorkMalformedAdmissionNeverStartsRecognition(t *testing.T) {
	a := configuredApp(t)
	a.recordingFactory = func() (*recordingExecution, error) {
		t.Fatal("speaker admission invoked recognition/diarization executor")
		return nil, nil
	}
	for _, op := range []string{"models.dataset.create", "models.train", "recordings.match"} {
		item := ""
		if op == "recordings.match" {
			item = contracts.ID()
		}
		response := realRequest(a, op, item, map[string]any{"unexpected": "input"})
		if response.Error == nil || response.Error.Code != "invalid_request" {
			t.Fatal(op, response)
		}
	}
	works, err := a.Catalog.Works(a.ctx)
	if err != nil || len(works) != 0 {
		t.Fatal("malformed request created work", works, err)
	}
}

func TestSpeakerWorkAdmissionRejectsCallerOwnedFrozenIdentities(t *testing.T) {
	a := configuredApp(t)
	adapter := map[string]any{"id": "pyannote-profile", "contract_version": "1", "mode": "local", "architecture": "pyannote", "output_kinds": []string{"voice-embedding"}, "consumers": []string{"voice-matching"}}
	for _, field := range []string{"recording_id", "model_digest"} {
		input := map[string]any{"model_id": contracts.ID(), "adapter": adapter, "threshold": .7, "ambiguity_margin": .1, "min_evidence_us": 1000, field: ""}
		response := realRequest(a, "recordings.match", contracts.ID(), input)
		if response.Error == nil || response.Error.Code != "invalid_request" {
			t.Fatal("caller supplied server-frozen field", field, response)
		}
	}
	response := realRequest(a, "models.train", "", map[string]any{"dataset_id": contracts.ID(), "name": "Election", "output_kind": "voice-embedding", "adapter": adapter, "base_digest": ""})
	if response.Error == nil || response.Error.Code != "invalid_request" {
		t.Fatal("caller supplied frozen training digest", response)
	}
	works, err := a.Catalog.Works(a.ctx)
	if err != nil || len(works) != 0 {
		t.Fatal("rejected frozen authority created work", works, err)
	}
}

func TestSpeakerTrainingRequiresExplicitPositiveSavedRevision(t *testing.T) {
	a := configuredApp(t)
	for _, revision := range []any{nil, 0, -1} {
		input := map[string]any{"dataset_id": contracts.ID(), "name": "Exact profile", "pipeline_id": contracts.ID()}
		if revision != nil {
			input["pipeline_revision"] = revision
		}
		response := realRequest(a, "models.train", "", input)
		if response.Error == nil || response.Error.Code != "invalid_request" {
			t.Fatal("saved profile accepted without exact positive revision", revision, response)
		}
	}
	works, err := a.Catalog.Works(a.ctx)
	if err != nil || len(works) != 0 {
		t.Fatal("invalid profile election queued work", works, err)
	}
}

func TestSpeakerMatchingResultsRemainAvailableBeyondFirstPage(t *testing.T) {
	a := configuredApp(t)
	queued, err := a.Catalog.EnqueueWork(a.ctx, contracts.ID(), "recordings.match", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	claim, err := a.Catalog.ClaimWork(a.ctx, queued.ID, contracts.ID(), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	decisions := make([]map[string]any, 201)
	for index := range decisions {
		decisions[index] = map[string]any{"local_speaker_id": contracts.ID(), "state": "unknown", "evidence_us": index}
	}
	raw, _ := json.Marshal(map[string]any{"recording_id": contracts.ID(), "decisions": decisions, "missing_profiles": []string{}, "mapping_count": 0})
	if _, err := a.Catalog.CheckpointWork(a.ctx, claim, "complete", "succeeded", raw, time.Minute); err != nil {
		t.Fatal(err)
	}
	first := configuredResult(t, realRequest(a, "work.results", claim.ID, map[string]any{"limit": 100}))
	if len(first["result"].(map[string]any)["decisions"].([]any)) != 100 || first["next_ordinal"] != float64(100) {
		t.Fatal("first outcomes page", first)
	}
	last := configuredResult(t, realRequest(a, "work.results", claim.ID, map[string]any{"after_ordinal": 200, "limit": 100}))
	page := last["result"].(map[string]any)["decisions"].([]any)
	if len(page) != 1 || page[0].(map[string]any)["local_speaker_id"] != decisions[200]["local_speaker_id"] || last["next_ordinal"] != nil {
		t.Fatal("final outcomes page", last)
	}
}
