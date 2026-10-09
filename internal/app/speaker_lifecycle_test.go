// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/models"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/voicemodels"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func speakerRuntimeRecording(t *testing.T, a *App) (catalog.Recording, string) {
	t.Helper()
	id := importRecordingFixture(t, a)
	entry, err := a.Catalog.Library(a.ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile("../subtitles/testdata/source.cueson.json")
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]json.RawMessage
	json.Unmarshal(raw, &document)
	var cues []map[string]json.RawMessage
	json.Unmarshal(document["cues"], &cues)
	local := contracts.ID()
	cues[0]["speaker_attributions"], _ = json.Marshal([]map[string]any{{"speaker_id": local, "start_milliseconds": 0, "end_milliseconds": 500}})
	document["cues"], _ = json.Marshal(cues)
	raw, _ = json.Marshal(document)
	queued, err := a.Catalog.EnqueueWork(a.ctx, contracts.ID(), "recordings.process", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	claim, err := a.Catalog.ClaimWork(a.ctx, queued.ID, contracts.ID(), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	record := catalog.Recording{ID: id, SourceDigest: entry.Digest, SourceRevision: entry.Revision, State: "ready", Document: raw, DocumentDigest: documentHash(raw), SourceMap: json.RawMessage(`{"start_numerator":"0","start_denominator":"1","sample_rate":16000,"sample_count":16000,"stream_index":0,"policy":"fixture"}`), Provenance: json.RawMessage(`{}`), Diagnostics: json.RawMessage(`[]`)}
	record, err = a.Catalog.CommitRecording(a.ctx, claim, 0, record)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.Catalog.CheckpointWork(a.ctx, claim, "complete", "succeeded", json.RawMessage(`{}`), time.Minute); err != nil {
		t.Fatal(err)
	}
	return record, local
}

func TestSpeakerSharedRuntimeTrainingMatchingAndExactFetch(t *testing.T) {
	a := configuredApp(t)
	// The real base-acquisition protocol runs against inert pinned fixture bytes.
	// Acoustic execution is replaced only by an explicit deterministic factory.
	manifest := dependencyManifest(t, "speaker-runtime", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("synthetic model bytes")) }))
	root := strings.TrimSuffix(manifest.Files[0].URL, "config.json")
	manifest.Capabilities = []string{"diarization", "voice-matching", "speaker-model-training"}
	manifest.Files = nil
	for _, role := range []string{"config.yaml", "embedding/pytorch_model.bin", "segmentation/pytorch_model.bin", "plda/plda.npz", "plda/xvec_transform.npz"} {
		manifest.Files = append(manifest.Files, models.File{Role: role, SHA256: documentHash([]byte("synthetic model bytes")), Size: 21, URL: root + role, LocalHTTP: true})
	}
	registerDependencyManifest(t, a, manifest)
	baseID := models.InstallationID(manifest)
	path := filepath.Join(t.TempDir(), "audio.wav")
	audio := make([]byte, 32044)
	copy(audio, "RIFF")
	binary.LittleEndian.PutUint32(audio[4:], 32036)
	copy(audio[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(audio[16:], 16)
	binary.LittleEndian.PutUint16(audio[20:], 1)
	binary.LittleEndian.PutUint16(audio[22:], 1)
	binary.LittleEndian.PutUint32(audio[24:], 16000)
	binary.LittleEndian.PutUint32(audio[28:], 32000)
	binary.LittleEndian.PutUint16(audio[32:], 2)
	binary.LittleEndian.PutUint16(audio[34:], 16)
	copy(audio[36:], "data")
	binary.LittleEndian.PutUint32(audio[40:], 32000)
	for i := 44; i < len(audio); i += 2 {
		binary.LittleEndian.PutUint16(audio[i:], 1000)
	}
	if err := os.WriteFile(path, audio, 0600); err != nil {
		t.Fatal(err)
	}
	a.recordingFactory = func() (*recordingExecution, error) {
		t.Fatal("training/matching invoked transcription or diarization")
		return nil, nil
	}
	a.speakerFactory = func(service *voicemodels.Service) *voicemodels.Service {
		service.Prepare = func(context.Context, catalog.LibraryEntry, processing.AudioOptions) (*voicemodels.Audio, error) {
			return &voicemodels.Audio{Path: path, SourceMap: processing.SourceMap{StartNumerator: "0", StartDenominator: "1", SampleRate: 16000, SampleCount: 16000, Policy: "fixture"}, Close: func() error { return nil }}, nil
		}
		service.Embed = func(context.Context, string, processing.SourceMap, string, string) ([]float64, error) {
			return []float64{1, 0}, nil
		}
		return service
	}
	training, voice := speakerRuntimeRecording(t, a)
	target, targetVoice := speakerRuntimeRecording(t, a)
	person, err := a.Catalog.PutSpeaker(a.ctx, contracts.ID(), 0, catalog.SpeakerIdentity{Speaker: catalog.Speaker{ID: contracts.ID(), Name: "Independent identity"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.Catalog.SetSpeakerMapping(a.ctx, contracts.ID(), training.Revision, catalog.SpeakerMapping{RecordingID: training.ID, LocalSpeakerID: voice, SpeakerID: person.Speaker.ID, DocumentDigest: training.DocumentDigest}); err != nil {
		t.Fatal(err)
	}
	dataset := configuredResult(t, realRequest(a, "models.dataset.create", "", map[string]any{"speaker_id": person.Speaker.ID}))
	datasetID := dataset["dataset"].(map[string]any)["id"].(string)
	adapter := voicemodels.Adapter{ID: "pyannote-profile", ContractVersion: "1", Mode: "local", Architecture: "pyannote", OutputKinds: []string{"voice-embedding"}, Consumers: []string{"voice-matching"}}
	queued := configuredResult(t, realRequest(a, "models.train", "", voicemodels.TrainOptions{DatasetID: datasetID, Name: "Grounded", Kind: "voice-embedding", Adapter: adapter, BaseModelID: baseID}))
	done := awaitWork(t, a, queued["work_id"].(string))
	if done.State != "succeeded" {
		t.Fatal("training", string(done.Result))
	}
	var result map[string]any
	json.Unmarshal(done.Result, &result)
	version := result["version_id"].(string)
	configuredResult(t, realRequest(a, "models.profile.set", person.Speaker.ID, map[string]any{"expected_revision": 0, "version_id": version}))
	configuredResult(t, realRequest(a, "recordings.roster.add", target.ID, map[string]any{"expected_revision": 0, "speakers": []string{person.Speaker.ID}}))
	match := configuredResult(t, realRequest(a, "recordings.match", target.ID, map[string]any{"model_id": baseID, "adapter": adapter, "threshold": 0.8, "ambiguity_margin": 0.1, "min_evidence_us": 1000}))
	matched := awaitWork(t, a, match["work_id"].(string))
	if matched.State != "succeeded" {
		t.Fatal("match", string(matched.Result))
	}
	mappings, err := a.Catalog.SpeakerMappings(a.ctx, target.ID)
	if err != nil || len(mappings) != 1 || mappings[0].LocalSpeakerID != targetVoice || mappings[0].SpeakerID != person.Speaker.ID || mappings[0].Origin != "automatic" {
		t.Fatal("automatic mapping", mappings, err)
	}
	unchanged, err := a.Catalog.Recording(a.ctx, target.ID)
	if err != nil || unchanged.DocumentDigest != target.DocumentDigest {
		t.Fatal("matching changed current document", err)
	}
	corrected, err := a.Catalog.PutSpeaker(a.ctx, contracts.ID(), 0, catalog.SpeakerIdentity{Speaker: catalog.Speaker{ID: contracts.ID(), Name: "Explicit correction"}})
	if err != nil {
		t.Fatal(err)
	}
	configuredResult(t, realRequest(a, "recordings.map-speaker", target.ID, map[string]any{"revision": unchanged.Revision, "mapping_revision": mappings[0].Revision, "local_speaker_id": targetVoice, "speaker_id": corrected.Speaker.ID, "document_digest": target.DocumentDigest}))
	rerun := configuredResult(t, realRequest(a, "recordings.match", target.ID, map[string]any{"model_id": baseID, "adapter": adapter, "threshold": 0.8, "ambiguity_margin": 0.1, "min_evidence_us": 1000}))
	rematched := awaitWork(t, a, rerun["work_id"].(string))
	if rematched.State != "succeeded" {
		t.Fatal("manual correction rerun", string(rematched.Result))
	}
	mappings, err = a.Catalog.SpeakerMappings(a.ctx, target.ID)
	if err != nil || len(mappings) != 1 || mappings[0].SpeakerID != corrected.Speaker.ID || mappings[0].Origin != "manual" {
		t.Fatal("rerun overwrote explicit correction", mappings, err)
	}
	list := configuredResult(t, realRequest(a, "models.speaker.list", "", map[string]any{"speaker_id": person.Speaker.ID}))
	if len(list["items"].([]any)) != 1 {
		t.Fatal("speaker discovery", list)
	}
	all := configuredResult(t, realRequest(a, "models.speaker.list", "", nil))
	if len(all["items"].([]any)) != 1 || all["profile"] != nil {
		t.Fatal("global speaker output discovery", all)
	}
	allDatasets := configuredResult(t, realRequest(a, "models.dataset.list", "", nil))
	if len(allDatasets["items"].([]any)) != 1 {
		t.Fatal("global dataset discovery", allDatasets)
	}
	destination := filepath.Join(t.TempDir(), "retrieved")
	fetched := realRequest(a, "models.speaker.fetch", version, map[string]any{"destination": destination})
	if fetched.Error != nil {
		t.Fatal("exact fetch", fetched.Error)
	}
	entries, err := os.ReadDir(destination)
	if err != nil || len(entries) < 2 {
		t.Fatal("missing portable fetch", entries, err)
	}
	// Replacing current evidence scrubs obsolete execution snapshots. A retry of
	// the accepted request must still reconcile its original terminal work.
	current, err := a.Catalog.Recording(a.ctx, target.ID)
	if err != nil {
		t.Fatal(err)
	}
	var edited map[string]json.RawMessage
	json.Unmarshal(current.Document, &edited)
	var editedCues []map[string]json.RawMessage
	json.Unmarshal(edited["cues"], &editedCues)
	var payload map[string]json.RawMessage
	json.Unmarshal(editedCues[0]["payload"], &payload)
	payload["plain_text"] = json.RawMessage(`"Replaced current evidence."`)
	payload["raw_text"] = payload["plain_text"]
	payload["lines"] = json.RawMessage(`["Replaced current evidence."]`)
	editedCues[0]["payload"], _ = json.Marshal(payload)
	edited["cues"], _ = json.Marshal(editedCues)
	current.Document, _ = json.Marshal(edited)
	current.DocumentDigest = documentHash(current.Document)
	replacement, err := a.Catalog.EnqueueWork(a.ctx, contracts.ID(), "recordings.process", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	claim, err := a.Catalog.ClaimWork(a.ctx, replacement.ID, contracts.ID(), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Catalog.CommitRecording(a.ctx, claim, current.Revision, current); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Catalog.CheckpointWork(a.ctx, claim, "complete", "succeeded", json.RawMessage(`{}`), time.Minute); err != nil {
		t.Fatal(err)
	}
	originalInput, _ := json.Marshal(map[string]any{"model_id": baseID, "adapter": adapter, "threshold": 0.8, "ambiguity_margin": 0.1, "min_evidence_us": 1000})
	request := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: a.Workspace.Config.WorkspaceID, RequestID: matched.ID, Operation: "recordings.match", ItemID: target.ID, Data: originalInput}
	replayed := configuredResult(t, a.Dispatch(request))
	if replayed["work_id"] != matched.ID || replayed["phase"] != "evidence-invalidated" {
		t.Fatal("accepted replay lost after snapshot scrubbing", replayed)
	}
	request.Data = json.RawMessage(`{}`)
	if response := a.Dispatch(request); response.Error == nil || response.Error.Code != "conflict" {
		t.Fatal("changed request replayed scrubbed intent", response)
	}
}
