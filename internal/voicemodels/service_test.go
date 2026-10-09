// SPDX-License-Identifier: Apache-2.0
package voicemodels

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/workspace"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fixture(t *testing.T) (*Service, *catalog.Store) {
	t.Helper()
	ctx := context.Background()
	w, err := workspace.Init(t.TempDir(), "speaker")
	if err != nil {
		t.Fatal("fixture init", err)
	}
	db, err := catalog.OpenWorkspace(ctx, w, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if err = db.RegisterWorkspace(ctx, w); err != nil {
		t.Fatal(err)
	}
	a, err := artifact.NewService(ctx, w, db, nil, contracts.ID())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close(); db.Close() })
	s := NewService(a, db, nil)
	s.Prepare = func(_ context.Context, _ catalog.LibraryEntry, options processing.AudioOptions) (*Audio, error) {
		dir := t.TempDir()
		path := filepath.Join(dir, "mapped.wav")
		pcm := make([]byte, 32000)
		for i := 0; i < len(pcm); i += 2 {
			binary.LittleEndian.PutUint16(pcm[i:], 1000)
		}
		if err := writeWAV(path, pcm); err != nil {
			t.Fatal(err)
		}
		return &Audio{Path: path, SourceMap: processing.SourceMap{StartNumerator: "0", StartDenominator: "1", SampleCount: 16000, SampleRate: 16000, StreamIndex: 0, Channel: options.Channel, Policy: "fixture"}, Close: func() error { return nil }}, nil
	}
	s.Embed = func(context.Context, string, processing.SourceMap, string, string) ([]float64, error) {
		return []float64{1, 0}, nil
	}
	return s, db
}
func work(t *testing.T, db catalog.Catalog, kind string, payload any) catalog.Work {
	t.Helper()
	raw, _ := json.Marshal(payload)
	ctx := context.Background()
	w, err := db.EnqueueWork(ctx, contracts.ID(), kind, raw)
	if err != nil {
		t.Fatal("enqueue", kind, err)
	}
	w, err = db.ClaimWork(ctx, w.ID, contracts.ID(), time.Minute)
	if err != nil {
		t.Fatal("claim", kind, err)
	}
	return w
}
func recording(t *testing.T, db *catalog.Store, count int) (catalog.Recording, string) {
	t.Helper()
	ctx := context.Background()
	entry := catalog.LibraryEntry{ID: contracts.ID(), AssetID: contracts.ID(), Title: "Fixture", Class: "audio", Mode: "reference", SourceLocator: "fixture.wav", Digest: strings.Repeat("a", 64), Size: 44, Facts: json.RawMessage(`{}`), Metadata: json.RawMessage(`{"capture_state":"failed"}`), Dates: json.RawMessage(`[]`), ReportPublicationIDs: json.RawMessage(`[]`)}
	var err error
	entry, err = db.CommitLibrary(ctx, work(t, db, "media.import", map[string]any{}), entry)
	if err != nil {
		t.Fatal("commit library", err)
	}
	raw, err := os.ReadFile("../subtitles/testdata/source.cueson.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]json.RawMessage
	json.Unmarshal(raw, &doc)
	delete(doc, "media_timing")
	var cues []map[string]json.RawMessage
	json.Unmarshal(doc["cues"], &cues)
	local := contracts.ID()
	original := cues[0]
	cues = nil
	for i := 0; i < count; i++ {
		cue := map[string]json.RawMessage{}
		for key, value := range original {
			cue[key] = value
		}
		cue["id"], _ = json.Marshal(fmt.Sprintf("cue-%06d", i))
		cue["ordinal"], _ = json.Marshal(i)
		cue["source_order"], _ = json.Marshal(i)
		cue["speaker_attributions"], _ = json.Marshal([]map[string]any{{"speaker_id": local, "start_milliseconds": i * 5, "end_milliseconds": i*5 + 5}})
		cues = append(cues, cue)
	}
	doc["cues"], _ = json.Marshal(cues)
	var description map[string]any
	json.Unmarshal(doc["document"], &description)
	description["cue_count"] = count
	doc["document"], _ = json.Marshal(description)
	var stats map[string]any
	json.Unmarshal(doc["stats"], &stats)
	stats["cue_count"] = count
	doc["stats"], _ = json.Marshal(stats)
	document, _ := json.Marshal(doc)
	r := catalog.Recording{ID: entry.ID, SourceDigest: entry.Digest, SourceRevision: entry.Revision, State: "ready", Document: document, DocumentDigest: hash(document), SourceMap: json.RawMessage(`{"start_numerator":"0","start_denominator":"1","sample_rate":16000,"sample_count":16000,"stream_index":0,"policy":"fixture"}`), Provenance: json.RawMessage(`{}`), Diagnostics: json.RawMessage(`[]`)}
	r, err = db.CommitRecording(ctx, work(t, db, "recordings.process", map[string]any{}), 0, r)
	if err != nil {
		t.Fatal("commit recording", err)
	}
	return r, local
}
func datasetFixture(t *testing.T, s *Service, db *catalog.Store, count int) (catalog.SpeakerDataset, catalog.Speaker) {
	t.Helper()
	ctx := context.Background()
	r, local := recording(t, db, count)
	speaker, err := db.PutSpeaker(ctx, contracts.ID(), 0, catalog.SpeakerIdentity{Speaker: catalog.Speaker{ID: contracts.ID(), Name: "Grounded"}})
	if err != nil {
		t.Fatal("put speaker", err)
	}
	_, err = db.SetSpeakerMapping(ctx, contracts.ID(), r.Revision, catalog.SpeakerMapping{RecordingID: r.ID, LocalSpeakerID: local, SpeakerID: speaker.Speaker.ID, DocumentDigest: r.DocumentDigest})
	if err != nil {
		t.Fatal("map speaker", err)
	}
	dataset, err := s.CreateDataset(ctx, contracts.ID(), DatasetOptions{SpeakerID: speaker.Speaker.ID})
	if err != nil {
		t.Fatal("create dataset", err)
	}
	return dataset, speaker.Speaker
}
func builtin() Adapter {
	return Adapter{ID: "pyannote-profile", ContractVersion: "1", Mode: "local", Architecture: "pyannote", OutputKinds: []string{"voice-embedding"}, Consumers: []string{"voice-matching"}}
}
func TestCorpusEnrollmentFetchAndRosterMatching(t *testing.T) {
	s, db := fixture(t)
	ctx := context.Background()
	dataset, speaker := datasetFixture(t, s, db, 105)
	if len(dataset.References) != 105 {
		t.Fatal("paginated corpus truncated", len(dataset.References))
	}
	base := contracts.ID()
	digest := strings.Repeat("b", 64)
	options := TrainOptions{DatasetID: dataset.Dataset.ID, Name: "Acoustic", Kind: "voice-embedding", Adapter: builtin(), BaseModelID: base, BaseDigest: digest}
	claim := work(t, db, "models.train", TrainPayload{Options: options})
	result, err := s.ExecuteTrain(ctx, claim)
	if err != nil {
		t.Fatal("enrollment", err)
	}
	version := result.(map[string]any)["version_id"].(string)
	out, err := s.Show(ctx, version)
	if err != nil || out.Kind != "voice-embedding" {
		t.Fatal(err)
	}
	var originalPublications []string
	if json.Unmarshal(out.PublicationIDs, &originalPublications) != nil {
		t.Fatal("publication lineage missing")
	}
	destination := filepath.Join(t.TempDir(), "fetched")
	if _, err = s.Fetch(ctx, version, destination); err != nil {
		t.Fatal("fetch", err)
	}
	if _, err = os.Stat(filepath.Join(destination, "profile.json")); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Fetch(ctx, version, destination); err == nil {
		t.Fatal("overwrite accepted")
	}
	if _, err = db.SetSpeakerProfile(ctx, contracts.ID(), speaker.ID, 0, version); err != nil {
		t.Fatal(err)
	}
	target, local := recording(t, db, 1)
	if _, err = db.MutateRoster(ctx, contracts.ID(), target.ID, 0, "replace", []string{speaker.ID}); err != nil {
		t.Fatal(err)
	}
	match := MatchOptions{RecordingID: target.ID, ModelID: base, ModelDigest: digest, Adapter: builtin(), Threshold: 0.8, Margin: 0.05, MinEvidenceUS: 1000}
	short := match
	short.MinEvidenceUS = 10000
	shortPayload, err := s.ValidateMatch(ctx, short)
	if err != nil {
		t.Fatal(err)
	}
	priorEmbed := s.Embed
	s.Embed = func(context.Context, string, processing.SourceMap, string, string) ([]float64, error) {
		t.Fatal("insufficient evidence entered engine")
		return nil, nil
	}
	shortResult, err := s.ExecuteMatch(ctx, work(t, db, "recordings.match", shortPayload))
	if err != nil {
		t.Fatal(err)
	}
	shortDecisions := shortResult.(map[string]any)["decisions"].([]Decision)
	if len(shortDecisions) != 1 || shortDecisions[0].State != "unknown" || shortDecisions[0].EvidenceUS != 5000 {
		t.Fatal("observed evidence incorrectly reported", shortDecisions)
	}
	s.Embed = priorEmbed
	payload, err := s.ValidateMatch(ctx, match)
	if err != nil {
		t.Fatal(err)
	}
	matched, err := s.ExecuteMatch(ctx, work(t, db, "recordings.match", payload))
	if err != nil {
		t.Fatal("match", err)
	}
	reports := matched.(map[string]any)["decisions"].([]Decision)
	if len(reports) != 1 || reports[0].State != "accepted" || reports[0].LocalSpeakerID != local || reports[0].SpeakerID != speaker.ID {
		t.Fatal(reports)
	}
	page, err := db.CurrentSpeakerReferences(ctx, catalog.SpeakerSelection{SpeakerID: speaker.ID, ConfirmedOnly: true, Limit: 100})
	if err != nil || len(page.References) != 100 || page.Next == "" {
		t.Fatal("automatic inference became enrollment", err, page)
	}
	// The same single roster member must not capture an unrelated voice.
	s.Embed = func(context.Context, string, processing.SourceMap, string, string) ([]float64, error) {
		return []float64{-1, 0}, nil
	}
	payload, err = s.ValidateMatch(ctx, match)
	if err != nil {
		t.Fatal(err)
	}
	matched, err = s.ExecuteMatch(ctx, work(t, db, "recordings.match", payload))
	if err != nil {
		t.Fatal(err)
	}
	reports = matched.(map[string]any)["decisions"].([]Decision)
	if reports[0].State != "unknown" {
		t.Fatal(reports)
	}
	maps, err := db.SpeakerMappings(ctx, target.ID)
	if err != nil || len(maps) != 0 {
		t.Fatal("unknown retained prior prediction", maps, err)
	}
	// Saved output remains retrievable after correction invalidates preparation.
	source := dataset.References[0]
	mapping, err := db.SpeakerMappings(ctx, source.RecordingID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.SetSpeakerMapping(ctx, contracts.ID(), source.RecordingRevision, mapping[0]); err != nil {
		t.Fatal(err)
	}
	if err = library.NewService(s.Artifacts, db, nil, library.Tools{}).Cleanup(ctx, source.RecordingID); err != nil {
		t.Fatal("lineage cleanup", err)
	}
	for _, id := range originalPublications {
		publication, err := db.Publication(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if publication.Kind == "speaker-preparation" || publication.ID == originalPublications[0] {
			if publication.State == "available" {
				t.Fatal("retired corpus lineage still available", publication.Kind, publication.ID)
			}
		}
	}
	if _, err = s.Fetch(ctx, version, filepath.Join(t.TempDir(), "retained")); err != nil {
		t.Fatal("immutable weights tied to stale corpus", err)
	}
	snapshot, err := db.Export(ctx)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := catalog.OpenSQLite(ctx, filepath.Join(t.TempDir(), "restore.sqlite"), s.Artifacts.Workspace.Config.WorkspaceID)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	if err = restored.Restore(ctx, snapshot); err != nil {
		t.Fatal("portable speaker output", err)
	}
}
func TestHostedTrainingUploadsOnlyElectedPreparedData(t *testing.T) {
	s, db := fixture(t)
	dataset, _ := datasetFixture(t, s, db, 1)
	weights := []byte("actual adapter output fixture")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Operation string `json:"operation"`
			Uploads   []struct {
				Path string `json:"path"`
				Data []byte `json:"data"`
				Text string `json:"text"`
			} `json:"uploads"`
		}
		if json.NewDecoder(r.Body).Decode(&req) != nil || req.Operation != "train" || len(req.Uploads) != 1 || req.Uploads[0].Path != "" || len(req.Uploads[0].Data) < 44 || req.Uploads[0].Text != "" {
			t.Error("bad upload", req)
		}
		json.NewEncoder(w).Encode(Output{ContractVersion: "1", Kind: "weights", Architecture: "custom", Consumers: []string{"custom"}, SupportedOperations: []string{"custom-inference"}, Artifacts: []Artifact{{Role: "weights.bin", Format: "binary", SHA256: hash(weights), Size: int64(len(weights)), Data: weights}}})
	}))
	defer server.Close()
	s.Client = server.Client()
	options := TrainOptions{DatasetID: dataset.Dataset.ID, Name: "Hosted", Kind: "weights", Adapter: Adapter{ID: "insonic-speaker", ContractVersion: "1", Mode: "hosted", Architecture: "custom", OutputKinds: []string{"weights"}, Consumers: []string{"custom"}, SupportedOperations: []string{"custom-inference"}, Endpoint: server.URL, RemoteModel: "trainer", UpstreamRevision: "immutable"}}
	claim := work(t, db, "models.train", TrainPayload{Options: options})
	if _, err := s.ExecuteTrain(context.Background(), claim); err != nil {
		current, _ := db.Work(context.Background(), claim.ID)
		t.Fatal("hosted execute", current.Phase, string(current.Result), err)
	}
}
func TestProductionAdaptersAndEmbeddingRefuseCI(t *testing.T) {
	t.Setenv("CI", "1")
	s, _ := fixture(t)
	a := Adapter{ID: "insonic-speaker", ContractVersion: "1", Mode: "local", Architecture: "custom", OutputKinds: []string{"weights"}, Consumers: []string{"custom"}, Executable: libraryPin()}
	if _, err := s.invoke(context.Background(), a, AdapterRequest{}); err == nil {
		t.Fatal("local trainer entered CI")
	}
	engine := &processing.Service{}
	if _, err := engine.Embed(context.Background(), "missing", processing.SourceMap{}, contracts.ID(), strings.Repeat("a", 64)); err == nil {
		t.Fatal("embedding entered CI")
	}
}
func libraryPin() library.PinnedFile {
	return library.PinnedFile{Path: filepath.Join(os.TempDir(), "not-executed"), SHA256: strings.Repeat("a", 64)}
}
