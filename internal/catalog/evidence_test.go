// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"os"
	"testing"
	"time"
)

func stringPointer(value string) *string { return &value }

func evidenceRecording(t *testing.T, s *Store) (Work, Recording, string) {
	t.Helper()
	claim, _, r := recordingFixture(t, s)
	raw, e := os.ReadFile("../subtitles/testdata/source.cueson.json")
	if e != nil {
		t.Fatal(e)
	}
	var doc map[string]json.RawMessage
	if e = json.Unmarshal(raw, &doc); e != nil {
		t.Fatal(e)
	}
	delete(doc, "media_timing")
	var cues []map[string]json.RawMessage
	if e = json.Unmarshal(doc["cues"], &cues); e != nil {
		t.Fatal(e)
	}
	local := contracts.ID()
	cues[0]["speaker_attributions"], _ = json.Marshal([]map[string]any{{"speaker_id": local, "start_milliseconds": 0, "end_milliseconds": 100}})
	doc["cues"], _ = json.Marshal(cues)
	r.Document, _ = json.Marshal(doc)
	r.DocumentDigest, r.State = hash(r.Document), "ready"
	r, e = s.CommitRecording(context.Background(), claim, 0, r)
	if e != nil {
		t.Fatal(e)
	}
	return claim, r, local
}

func TestEvidenceReferencesRejectCopiedAssignments(t *testing.T) {
	var segment Segment
	if strict([]byte(`{"id":"10000000-0000-4000-8000-000000000001","start_us":0,"end_us":100,"attribution":{}}`), &segment) == nil {
		t.Fatal("legacy copied interval/attribution accepted")
	}
	for _, raw := range []string{`{"nested":{"attribution":{"speaker_id":"known"}}}`, `{"start_us":0,"end_us":100}`, `{"transcript":"archived old text"}`} {
		if referenceOnlyOptions(json.RawMessage(raw)) {
			t.Fatal("copied evidence admitted via options", raw)
		}
	}
	s := localStore(t, contracts.ID())
	_, r, local := evidenceRecording(t, s)
	segment = Segment{ID: contracts.ID(), Revision: 1, RecordingID: r.ID, DocumentDigest: r.DocumentDigest, CueID: "cue-000000", LocalSpeakerID: local}
	rev, _ := s.Revision(context.Background())
	if _, e := s.Commit(context.Background(), Mutation{OperationID: contracts.ID(), Expected: rev, Records: Records{Segments: []Segment{segment}}}); e != nil {
		t.Fatal("current evidence reference rejected", e)
	}
	segment.ID, segment.CueID = contracts.ID(), "unknown-cue"
	rev, _ = s.Revision(context.Background())
	if _, e := s.Commit(context.Background(), Mutation{OperationID: contracts.ID(), Expected: rev, Records: Records{Segments: []Segment{segment}}}); e == nil {
		t.Fatal("reference to absent cue accepted")
	}
}

func TestEvidenceReplacementInvalidatesCorpusAndRetiresOnlyDerived(t *testing.T) {
	evidenceReplacementSuite(t, localStore(t, contracts.ID()))
}
func evidenceReplacementSuite(t *testing.T, s *Store) {
	ctx := context.Background()
	_, r, local := evidenceRecording(t, s)
	manifest, preparation, clip, weights := availableArtifact(t, s), availableArtifact(t, s), availableArtifact(t, s), availableArtifact(t, s)
	job, e := s.StartJob(ctx, contracts.ID(), contracts.ID(), 10, time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	speaker, segment, dataset, run, model, version := contracts.ID(), contracts.ID(), contracts.ID(), contracts.ID(), contracts.ID(), contracts.ID()
	rows := Records{Speakers: []Speaker{{ID: speaker, Name: "Known"}}, Segments: []Segment{{ID: segment, Revision: 1, RecordingID: r.ID, DocumentDigest: r.DocumentDigest, CueID: "cue-000000", LocalSpeakerID: local, ClipArtifactID: &clip.ArtifactID}}, Datasets: []Dataset{{ID: dataset, SpeakerID: speaker, ManifestArtifactID: &manifest.ArtifactID, Options: json.RawMessage(`{}`), State: "current"}}, Members: []DatasetMember{{contracts.ID(), dataset, 0, segment, 1}}, Runs: []TrainingRun{{ID: run, DatasetID: dataset, SpeakerID: speaker, JobID: job.JobID, PreparationArtifactID: &preparation.ArtifactID, Adapter: "fixture", Options: json.RawMessage(`{}`), State: "current"}}, Models: []Model{{model, speaker, "Family"}}, Versions: []ModelVersion{{ID: version, ModelID: model, RunID: run, DatasetID: dataset, ManifestArtifactID: &manifest.ArtifactID, Kind: "trained", State: "current"}}, ModelArtifacts: []ModelArtifact{{contracts.ID(), version, weights.ArtifactID, "weights", "fixture"}}}
	rev, _ := s.Revision(ctx)
	if _, e = s.Commit(ctx, Mutation{OperationID: contracts.ID(), Expected: rev, Records: rows}); e != nil {
		t.Fatal(e)
	}
	claim, _, _ := recordingFixture(t, s)
	next := r
	next.State = "no-speech"
	next.Document = json.RawMessage("null")
	next.DocumentDigest = ""
	if _, e = s.CommitRecording(ctx, claim, r.Revision, next); e != nil {
		t.Fatal(e)
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if len(snap.Records.Segments) != 0 || len(snap.Records.Members) != 0 {
		t.Fatal("stale evidence retained")
	}
	if len(snap.Records.Datasets) != 1 || snap.Records.Datasets[0].State != "invalidated" || snap.Records.Datasets[0].ManifestArtifactID != nil || snap.Records.Runs[0].PreparationArtifactID != nil || snap.Records.Versions[0].ManifestArtifactID != nil {
		t.Fatal("obsolete corpus bytes still referenced")
	}
	if len(snap.Records.ModelArtifacts) != 1 || snap.Records.ModelArtifacts[0].ArtifactID != weights.ArtifactID {
		t.Fatal("noncontent model lineage deleted")
	}
	forged := snap
	forged.Records.Datasets, forged.Records.Runs, forged.Records.Versions = rows.Datasets, rows.Runs, rows.Versions
	forged.Digest, _ = forged.digest()
	if e = localStore(t, s.workspace).Restore(ctx, forged); e == nil {
		t.Fatal("invalidated manifest/preparation state revived from snapshot")
	}
	for _, p := range []Publication{manifest, preparation, clip} {
		if _, e = s.ClaimRetirement(ctx, p.ID, contracts.ID(), 0, time.Minute); e != nil {
			t.Fatal("derived publication still referenced", e)
		}
	}
	if _, e = s.ClaimRetirement(ctx, weights.ID, contracts.ID(), 0, time.Minute); e == nil {
		t.Fatal("model weights retirement permitted")
	}
	if e = localStore(t, s.workspace).Restore(ctx, snap); e != nil {
		t.Fatal("valid invalidated lineage failed snapshot", e)
	}
}
