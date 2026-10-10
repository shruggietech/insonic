// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shruggietech/insonic/internal/contracts"
	"math"
	"os"
	"testing"
	"time"
)

func TestSpeakerDatasetAdmitsCorpusAboveTenThousand(t *testing.T) {
	ctx := context.Background()
	s := localStore(t, contracts.ID())
	_, recording, local := evidenceRecording(t, s)
	const count = 10001
	var document map[string]json.RawMessage
	json.Unmarshal(recording.Document, &document)
	var original []map[string]json.RawMessage
	json.Unmarshal(document["cues"], &original)
	cues := make([]map[string]json.RawMessage, count)
	for i := range cues {
		cue := map[string]json.RawMessage{}
		for k, v := range original[0] {
			cue[k] = v
		}
		cue["id"], _ = json.Marshal(fmt.Sprintf("cue-%06d", i))
		cue["ordinal"], _ = json.Marshal(i)
		cue["source_order"], _ = json.Marshal(i)
		cues[i] = cue
	}
	document["cues"], _ = json.Marshal(cues)
	for _, key := range []string{"document", "stats"} {
		var metadata map[string]json.RawMessage
		json.Unmarshal(document[key], &metadata)
		metadata["cue_count"], _ = json.Marshal(count)
		document[key], _ = json.Marshal(metadata)
	}
	recording.Document, _ = json.Marshal(document)
	recording.DocumentDigest = hash(recording.Document)
	claim := speakerWork(t, s, "recordings.process")
	var e error
	recording, e = s.CommitRecording(ctx, claim, recording.Revision, recording)
	if e != nil {
		t.Fatal("large current document", e)
	}
	identity, e := s.PutSpeaker(ctx, contracts.ID(), 0, SpeakerIdentity{Speaker: Speaker{ID: contracts.ID(), Name: "Large corpus"}})
	if e != nil {
		t.Fatal(e)
	}
	mapping, e := s.SetSpeakerMapping(ctx, contracts.ID(), recording.Revision, SpeakerMapping{RecordingID: recording.ID, LocalSpeakerID: local, SpeakerID: identity.Speaker.ID, DocumentDigest: recording.DocumentDigest})
	if e != nil {
		t.Fatal(e)
	}
	page, e := s.CurrentSpeakerReferences(ctx, SpeakerSelection{SpeakerID: identity.Speaker.ID, ConfirmedOnly: true, Limit: 100})
	if e != nil || len(page.References) != 100 || page.Next == "" {
		t.Fatal("bounded first page", e)
	}
	next, e := s.CurrentSpeakerReferences(ctx, SpeakerSelection{SpeakerID: identity.Speaker.ID, ConfirmedOnly: true, Limit: 100, Cursor: page.Next})
	if e != nil || len(next.References) != 100 || next.References[0].CueID == page.References[0].CueID {
		t.Fatal("bounded continuation", e)
	}
	rawMap, _ := canonical(recording.SourceMap)
	refs := make([]CurrentReference, count)
	for i := range refs {
		refs[i] = CurrentReference{RecordingID: recording.ID, RecordingRevision: recording.Revision, DocumentDigest: recording.DocumentDigest, CueID: fmt.Sprintf("cue-%06d", i), LocalSpeakerID: local, SpeakerID: identity.Speaker.ID, MappingRevision: mapping.Revision, SourceDigest: recording.SourceDigest, SourceMapDigest: hash(rawMap)}
	}
	publication := availableArtifact(t, s)
	dataset, e := s.CreateSpeakerDataset(ctx, contracts.ID(), identity.Speaker.ID, refs, page.Epoch, json.RawMessage(`{}`), json.RawMessage(`{}`), publication.ID)
	if e != nil || len(dataset.References) != count {
		t.Fatal("supported corpus rejected", e, len(dataset.References))
	}
	shown, e := s.SpeakerDataset(ctx, dataset.Dataset.ID)
	if e != nil || len(shown.References) != count {
		t.Fatal("large accepted corpus unavailable", e)
	}
	tooMany := make([]CurrentReference, SpeakerDatasetReferenceLimit+1)
	revision, _ := s.Revision(ctx)
	if _, e = s.CreateSpeakerDataset(ctx, contracts.ID(), identity.Speaker.ID, tooMany, page.Epoch, json.RawMessage(`{}`), json.RawMessage(`{}`), publication.ID); e == nil {
		t.Fatal("out-of-bound corpus admitted")
	}
	if after, e := s.Revision(ctx); e != nil || after != revision {
		t.Fatal("out-of-bound corpus changed authority", e)
	}
}

func speakerMetadataFixture(t *testing.T, v SpeakerOutput) json.RawMessage {
	t.Helper()
	raw, err := os.ReadFile("../../schemas/v1.0.0/speaker-model.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Examples []map[string]json.RawMessage `json:"examples"`
	}
	if json.Unmarshal(raw, &schema) != nil || len(schema.Examples) < 1 {
		t.Fatal("speaker model example missing")
	}
	m := schema.Examples[0]
	for key, value := range map[string]any{"model_family_id": v.ModelID, "model_version_id": v.ID, "originating_speaker_id": v.SpeakerID, "display_name": v.Name} {
		m[key], _ = json.Marshal(value)
	}
	var training map[string]json.RawMessage
	json.Unmarshal(m["training"], &training)
	training["training_run_id"], _ = json.Marshal(operationID("speaker-training-run", v.WorkID))
	training["job_id"], _ = json.Marshal(v.WorkID)
	training["preparation"] = json.RawMessage(`null`)
	var dataset map[string]json.RawMessage
	json.Unmarshal(training["dataset"], &dataset)
	dataset["dataset_snapshot_id"], _ = json.Marshal(v.DatasetID)
	dataset["originating_speaker_id"], _ = json.Marshal(v.SpeakerID)
	training["dataset"], _ = json.Marshal(dataset)
	m["training"], _ = json.Marshal(training)
	var compatibility map[string]json.RawMessage
	json.Unmarshal(m["compatibility"], &compatibility)
	compatibility["model_kind"], _ = json.Marshal(v.Kind)
	m["compatibility"], _ = json.Marshal(compatibility)
	m["license_declarations"] = json.RawMessage(`[{"scope":"output","declared_license":null,"attribution":null,"source_basis":"configured-adapter"}]`)
	result, _ := json.Marshal(m)
	return result
}

func TestSpeakerOutputMetadataContext(t *testing.T) {
	v := SpeakerOutput{ID: contracts.ID(), ModelID: contracts.ID(), SpeakerID: contracts.ID(), DatasetID: contracts.ID(), WorkID: contracts.ID(), Kind: "voice-embedding", Name: "Elected"}
	v.PublicationIDs, _ = json.Marshal([]string{contracts.ID()})
	v.Metadata = speakerMetadataFixture(t, v)
	if !validSpeakerOutput(v) {
		t.Fatal("schema-valid license attribution rejected")
	}
	var m map[string]json.RawMessage
	json.Unmarshal(v.Metadata, &m)
	m["extensions"] = json.RawMessage(`{"fixture":{"attribution":"copied assignment"}}`)
	v.Metadata, _ = json.Marshal(m)
	if validSpeakerOutput(v) {
		t.Fatal("unrelated attribution path accepted")
	}
	v.Metadata = speakerMetadataFixture(t, v)
	v.ID = contracts.ID()
	if validSpeakerOutput(v) {
		t.Fatal("manifest output identity mismatch accepted")
	}
	v.Metadata = json.RawMessage(`{}`)
	if validSpeakerOutput(v) {
		t.Fatal("incomplete portable model accepted")
	}
}

func TestSpeakerOutputAdmissionRequiresCurrentPublication(t *testing.T) {
	ctx := context.Background()
	s := localStore(t, contracts.ID())
	dataset, _, _, _ := speakerDatasetFixture(t, s)
	claim := speakerWork(t, s, "speaker.train")
	v := SpeakerOutput{ID: contracts.ID(), ModelID: contracts.ID(), SpeakerID: dataset.Dataset.SpeakerID, DatasetID: dataset.Dataset.ID, WorkID: claim.ID, Kind: "voice-embedding", Name: "Elected", PublicationIDs: json.RawMessage(`[]`)}
	v.Metadata = speakerMetadataFixture(t, v)
	if _, e := s.CommitSpeakerOutput(ctx, claim, v); e == nil {
		t.Fatal("publication-free current output admitted")
	}
	var metadata map[string]json.RawMessage
	json.Unmarshal(v.Metadata, &metadata)
	metadata["state"] = json.RawMessage(`"invalidated"`)
	metadata["manifest_sha256"] = json.RawMessage(`null`)
	var training map[string]json.RawMessage
	json.Unmarshal(metadata["training"], &training)
	var lineage map[string]json.RawMessage
	json.Unmarshal(training["dataset"], &lineage)
	lineage["state"] = json.RawMessage(`"invalidated"`)
	lineage["manifest_sha256"] = json.RawMessage(`null`)
	training["dataset"], _ = json.Marshal(lineage)
	metadata["training"], _ = json.Marshal(training)
	v.Metadata, _ = json.Marshal(metadata)
	if !validSpeakerOutput(v) {
		t.Fatal("invalidated output representation invalid")
	}
	if _, e := s.CommitSpeakerOutput(ctx, claim, v); e == nil {
		t.Fatal("new invalidated output admitted")
	}
}

func TestSpeakerEmptyDatasetAndNoBaseCheckpointPortability(t *testing.T) {
	ctx := context.Background()
	s := localStore(t, contracts.ID())
	sp, e := s.PutSpeaker(ctx, contracts.ID(), 0, SpeakerIdentity{Speaker: Speaker{ID: contracts.ID(), Name: "Empty"}})
	if e != nil {
		t.Fatal(e)
	}
	page, e := s.CurrentSpeakerReferences(ctx, SpeakerSelection{SpeakerID: sp.Speaker.ID, ConfirmedOnly: true})
	if e != nil {
		t.Fatal(e)
	}
	pub := availableArtifact(t, s)
	dataset, e := s.CreateSpeakerDataset(ctx, contracts.ID(), sp.Speaker.ID, page.References, page.Epoch, json.RawMessage(`{}`), json.RawMessage(`{"included":0}`), pub.ID)
	if e != nil {
		t.Fatal(e)
	}
	if len(dataset.References) != 0 {
		t.Fatal("empty snapshot has members")
	}
	w, e := s.EnqueueWork(ctx, contracts.ID(), "models.train", json.RawMessage(`{"options":{"dataset_id":"`+dataset.Dataset.ID+`"}}`))
	if e != nil {
		t.Fatal(e)
	}
	w, e = s.ClaimWork(ctx, w.ID, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	checkpoint := SpeakerCheckpoint{ID: contracts.ID(), WorkID: w.ID, Attempt: w.Generation, Step: 0, AdapterDigest: hash([]byte("adapter")), Compatibility: "fixture", PublicationID: pub.ID}
	if _, e = s.CommitSpeakerCheckpoint(ctx, w, checkpoint); e != nil {
		t.Fatal("no-base checkpoint rejected", e)
	}
	checkpoint.ID = contracts.ID()
	checkpoint.BaseDigest = hash([]byte("substituted base"))
	if _, e = s.CommitSpeakerCheckpoint(ctx, w, checkpoint); e == nil {
		t.Fatal("checkpoint invented base")
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if e = localStore(t, s.workspace).Restore(ctx, snap); e != nil {
		t.Fatal("portable empty dataset/checkpoint", e)
	}
}

func TestSpeakerDatasetRejectsStaleAndAutomaticEvidence(t *testing.T) {
	s := localStore(t, contracts.ID())
	ctx := context.Background()
	_, rec, local := evidenceRecording(t, s)
	sp, e := s.PutSpeaker(ctx, contracts.ID(), 0, SpeakerIdentity{Speaker: Speaker{ID: contracts.ID(), Name: "Independent"}})
	if e != nil {
		t.Fatal(e)
	}
	m, e := s.SetSpeakerMapping(ctx, contracts.ID(), rec.Revision, SpeakerMapping{RecordingID: rec.ID, LocalSpeakerID: local, SpeakerID: sp.Speaker.ID, DocumentDigest: rec.DocumentDigest})
	if e != nil {
		t.Fatal(e)
	}
	page, e := s.CurrentSpeakerReferences(ctx, SpeakerSelection{SpeakerID: sp.Speaker.ID, ConfirmedOnly: true})
	if e != nil || len(page.References) != 1 {
		t.Fatal(e, page)
	}
	pub := availableArtifact(t, s)
	m, e = s.SetSpeakerMapping(ctx, contracts.ID(), rec.Revision, m)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.CreateSpeakerDataset(ctx, contracts.ID(), sp.Speaker.ID, page.References, page.Epoch, json.RawMessage(`{}`), json.RawMessage(`{}`), pub.ID); e == nil {
		t.Fatal("stale mapping became frozen corpus")
	}
}

func TestSpeakerDatasetIntentReplaySurvivesInvalidation(t *testing.T) {
	ctx := context.Background()
	s := localStore(t, contracts.ID())
	_, speaker, recording, _ := speakerDatasetFixture(t, s)
	page, e := s.CurrentSpeakerReferences(ctx, SpeakerSelection{SpeakerID: speaker.ID, ConfirmedOnly: true})
	if e != nil {
		t.Fatal(e)
	}
	op := contracts.ID()
	recipe := json.RawMessage(`{"min_duration_us":250,"languages":["en"]}`)
	publication := availableArtifact(t, s)
	dataset, e := s.CreateSpeakerDataset(ctx, op, speaker.ID, page.References, page.Epoch, recipe, json.RawMessage(`{}`), publication.ID)
	if e != nil {
		t.Fatal(e)
	}
	if replay, found, e := s.ReplaySpeakerDataset(ctx, op, speaker.ID, json.RawMessage(`{"languages":["en"],"min_duration_us":250}`)); e != nil || !found || replay.Dataset.ID != dataset.Dataset.ID {
		t.Fatal("canonical original request replay", e)
	}
	mappings, e := s.SpeakerMappings(ctx, recording.ID)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.SetSpeakerMapping(ctx, contracts.ID(), recording.Revision, mappings[0]); e != nil {
		t.Fatal(e)
	}
	revision, e := s.Revision(ctx)
	if e != nil {
		t.Fatal(e)
	}
	replay, found, e := s.ReplaySpeakerDataset(ctx, op, speaker.ID, recipe)
	if e != nil || !found || replay.Dataset.ID != dataset.Dataset.ID || replay.Dataset.State != "invalidated" || len(replay.References) != 0 || string(replay.Recipe) != "{}" {
		t.Fatal("invalidated original election replay", e)
	}
	if _, _, e = s.ReplaySpeakerDataset(ctx, op, speaker.ID, json.RawMessage(`{"min_duration_us":251,"languages":["en"]}`)); e == nil {
		t.Fatal("changed recipe accepted for original operation")
	}
	if _, _, e = s.ReplaySpeakerDataset(ctx, op, contracts.ID(), recipe); e == nil {
		t.Fatal("changed speaker accepted for original operation")
	}
	if _, found, e = s.ReplaySpeakerDataset(ctx, contracts.ID(), speaker.ID, recipe); e != nil || found {
		t.Fatal("missing election treated as replay", e)
	}
	if after, e := s.Revision(ctx); e != nil || after != revision {
		t.Fatal("replay modified catalog authority", e)
	}
	snapshot, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	restored := localStore(t, s.workspace)
	if e = restored.Restore(ctx, snapshot); e != nil {
		t.Fatal("portable request digest", e)
	}
	if replay, found, e = restored.ReplaySpeakerDataset(ctx, op, speaker.ID, recipe); e != nil || !found || replay.Dataset.ID != dataset.Dataset.ID {
		t.Fatal("restored original request replay", e)
	}
}

func speakerDatasetFixture(t *testing.T, s *Store) (SpeakerDataset, Speaker, Recording, string) {
	t.Helper()
	ctx := context.Background()
	_, r, voice := evidenceRecording(t, s)
	sp, e := s.PutSpeaker(ctx, contracts.ID(), 0, SpeakerIdentity{Speaker: Speaker{ID: contracts.ID(), Name: "Grounded"}})
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.SetSpeakerMapping(ctx, contracts.ID(), r.Revision, SpeakerMapping{RecordingID: r.ID, LocalSpeakerID: voice, SpeakerID: sp.Speaker.ID, DocumentDigest: r.DocumentDigest})
	if e != nil {
		t.Fatal(e)
	}
	page, e := s.CurrentSpeakerReferences(ctx, SpeakerSelection{SpeakerID: sp.Speaker.ID, ConfirmedOnly: true})
	if e != nil {
		t.Fatal(e)
	}
	p := availableArtifact(t, s)
	ds, e := s.CreateSpeakerDataset(ctx, contracts.ID(), sp.Speaker.ID, page.References, page.Epoch, json.RawMessage(`{}`), json.RawMessage(`{"included":1}`), p.ID)
	if e != nil {
		t.Fatal(e)
	}
	return ds, sp.Speaker, r, voice
}
func speakerWork(t *testing.T, s *Store, kind string) Work {
	t.Helper()
	ctx := context.Background()
	w, e := s.EnqueueWork(ctx, contracts.ID(), kind, json.RawMessage(`{}`))
	if e != nil {
		t.Fatal(e)
	}
	w, e = s.ClaimWork(ctx, w.ID, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	return w
}
func speakerOutputFixture(t *testing.T, s *Store, ds SpeakerDataset) SpeakerOutput {
	t.Helper()
	p := availableArtifact(t, s)
	weights := availableArtifact(t, s)
	ids, _ := json.Marshal([]string{p.ID, weights.ID})
	claim := speakerWork(t, s, "speaker.train")
	v := SpeakerOutput{ID: contracts.ID(), ModelID: contracts.ID(), SpeakerID: ds.Dataset.SpeakerID, DatasetID: ds.Dataset.ID, WorkID: claim.ID, Kind: "voice-embedding", Name: "Elected", PublicationIDs: ids}
	v.Metadata = speakerMetadataFixture(t, v)
	out, e := s.CommitSpeakerOutput(context.Background(), claim, v)
	if e != nil {
		t.Fatal(e)
	}
	return out
}

func speakerModelSuite(t *testing.T, s *Store) {
	ctx := context.Background()
	ds, sp, source, _ := speakerDatasetFixture(t, s)
	out := speakerOutputFixture(t, s, ds)
	allDatasets, listErr := s.SpeakerDatasets(ctx, "")
	if listErr != nil || len(allDatasets) != 1 {
		t.Fatal("all-dataset list", listErr)
	}
	allOutputs, listErr := s.SpeakerOutputs(ctx, "")
	if listErr != nil || len(allOutputs) != 1 {
		t.Fatal("all-output list", listErr)
	}
	head, e := s.SetSpeakerProfile(ctx, contracts.ID(), sp.ID, 0, out.ID)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.SetSpeakerProfile(ctx, contracts.ID(), sp.ID, 0, out.ID); e == nil {
		t.Fatal("stale profile CAS accepted")
	}
	_, target, targetVoice := evidenceRecording(t, s)
	if _, e = s.MutateRoster(ctx, contracts.ID(), target.ID, 0, "replace", []string{sp.ID}); e != nil {
		t.Fatal(e)
	}
	frozen, e := s.FreezeSpeakerMatching(ctx, target.ID)
	if e != nil || len(frozen.Outputs) != 1 || string(frozen.Recording.Document) != "null" {
		t.Fatal("thin roster snapshot", e)
	}
	claim := speakerWork(t, s, "speaker.match")
	score := 0.9
	decisions := []SpeakerMatchDecision{{LocalSpeakerID: targetVoice, SpeakerID: sp.ID, State: "accepted", Score: &score, CueIDs: []string{"cue-000000"}, Diagnostics: json.RawMessage(`{}`)}}
	maps, e := s.CommitSpeakerMatching(ctx, claim, frozen, decisions, json.RawMessage(`{"threshold":0.8}`))
	if e != nil || len(maps) != 1 || maps[0].Origin != "automatic" {
		t.Fatal("automatic matching", e, maps)
	}
	current, e := s.CurrentSpeakerReferences(ctx, SpeakerSelection{SpeakerID: sp.ID, ConfirmedOnly: true})
	if e != nil || len(current.References) != 1 {
		t.Fatal("automatic guess became independent training evidence", e, current)
	}
	frozen, e = s.FreezeSpeakerMatching(ctx, target.ID)
	if e != nil {
		t.Fatal(e)
	}
	claim = speakerWork(t, s, "speaker.match")
	decisions[0].SpeakerID = ""
	decisions[0].State = "unknown"
	maps, e = s.CommitSpeakerMatching(ctx, claim, frozen, decisions, json.RawMessage(`{}`))
	if e != nil || len(maps) != 0 {
		t.Fatal("unknown retained prior automatic mapping", e)
	}
	frozen, e = s.FreezeSpeakerMatching(ctx, target.ID)
	if e != nil {
		t.Fatal(e)
	}
	manual, e := s.SetSpeakerMapping(ctx, contracts.ID(), target.Revision, SpeakerMapping{RecordingID: target.ID, LocalSpeakerID: targetVoice, SpeakerID: sp.ID, DocumentDigest: target.DocumentDigest})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.CommitSpeakerMatching(ctx, speakerWork(t, s, "speaker.match"), frozen, decisions, json.RawMessage(`{}`)); e == nil {
		t.Fatal("concurrent manual correction overwritten")
	}
	frozen, e = s.FreezeSpeakerMatching(ctx, target.ID)
	if e != nil {
		t.Fatal(e)
	}
	maps, e = s.CommitSpeakerMatching(ctx, speakerWork(t, s, "speaker.match"), frozen, decisions, json.RawMessage(`{}`))
	if e != nil || len(maps) != 1 || maps[0].Revision != manual.Revision || maps[0].Origin != "manual" {
		t.Fatal("manual mapping changed", e, maps)
	}
	frozen, e = s.FreezeSpeakerMatching(ctx, target.ID)
	if e != nil {
		t.Fatal(e)
	}
	decisions[0].Score = new(float64)
	*decisions[0].Score = math.NaN()
	if _, e = s.CommitSpeakerMatching(ctx, speakerWork(t, s, "speaker.match"), frozen, decisions, json.RawMessage(`{}`)); e == nil {
		t.Fatal("nonfinite score accepted")
	}
	decisions[0].Score = &score
	if _, e = s.SetSpeakerProfile(ctx, contracts.ID(), sp.ID, head.Revision, ""); e != nil {
		t.Fatal(e)
	}
	if _, e = s.CommitSpeakerMatching(ctx, speakerWork(t, s, "speaker.match"), frozen, decisions, json.RawMessage(`{}`)); e == nil {
		t.Fatal("changed profile accepted")
	}
	sourceMap, e := s.SpeakerMappings(ctx, source.ID)
	if e != nil {
		t.Fatal(e)
	}
	oldMatchPayload, _ := json.Marshal(map[string]any{"snapshot": frozen})
	oldMatch, e := s.EnqueueWork(ctx, contracts.ID(), "recordings.match", oldMatchPayload)
	if e != nil {
		t.Fatal(e)
	}
	oldMatch, e = s.ClaimWork(ctx, oldMatch.ID, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.SetSpeakerMapping(ctx, contracts.ID(), source.Revision, sourceMap[0])
	if e != nil {
		t.Fatal(e)
	}
	staleMatch, e := s.Work(ctx, oldMatch.ID)
	if e != nil || staleMatch.State != "failed" || staleMatch.Phase != "evidence-invalidated" {
		t.Fatal("different recording retained obsolete model lineage work", e)
	}
	var scrubbed map[string]json.RawMessage
	json.Unmarshal(staleMatch.Payload, &scrubbed)
	if _, ok := scrubbed["snapshot"]; ok {
		t.Fatal("old output metadata remains in work payload")
	}
	invalid, e := s.SpeakerDataset(ctx, ds.Dataset.ID)
	if e != nil || invalid.Dataset.State != "invalidated" || len(invalid.References) != 0 {
		t.Fatal("stale corpus content retained", e)
	}
	retained, e := s.SpeakerOutput(ctx, out.ID)
	if e != nil || retained.ID != out.ID {
		t.Fatal("immutable output removed", e)
	}
	var retainedIDs, originalIDs []string
	json.Unmarshal(retained.PublicationIDs, &retainedIDs)
	json.Unmarshal(out.PublicationIDs, &originalIDs)
	if len(retainedIDs) != 1 || retainedIDs[0] != originalIDs[1] {
		t.Fatal("obsolete manifest retained or weights discarded")
	}
	var metadata struct {
		State    string `json:"state"`
		Manifest any    `json:"manifest_sha256"`
		Training struct {
			Preparation any `json:"preparation"`
			Dataset     struct {
				State    string `json:"state"`
				Manifest any    `json:"manifest_sha256"`
			} `json:"dataset"`
		} `json:"training"`
	}
	json.Unmarshal(retained.Metadata, &metadata)
	if metadata.State != "invalidated" || metadata.Manifest != nil || metadata.Training.Preparation != nil || metadata.Training.Dataset.State != "invalidated" || metadata.Training.Dataset.Manifest != nil {
		t.Fatal("obsolete model input lineage retained")
	}
	stale := out
	stale.ID = contracts.ID()
	badClaim := speakerWork(t, s, "speaker.train")
	stale.WorkID = badClaim.ID
	if _, e = s.CommitSpeakerOutput(ctx, badClaim, stale); e == nil {
		t.Fatal("invalidated dataset published fresh output")
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	dest := localStore(t, s.workspace)
	if e = dest.Restore(ctx, snap); e != nil {
		t.Fatal("portable immutable output/profiles", e)
	}
	snap.Records.SpeakerOutputs[0].Name = "Tampered"
	snap.Digest, _ = snap.digest()
	if e = localStore(t, s.workspace).Restore(ctx, snap); e == nil {
		t.Fatal("forged immutable output accepted")
	}
}
func TestSQLiteSpeakerModelsAndMatching(t *testing.T) {
	speakerModelSuite(t, localStore(t, contracts.ID()))
}

func TestSpeakerExplicitAssociationPreservesOrigin(t *testing.T) {
	ctx := context.Background()
	s := localStore(t, contracts.ID())
	ds, original, _, _ := speakerDatasetFixture(t, s)
	output := speakerOutputFixture(t, s, ds)
	other, e := s.PutSpeaker(ctx, contracts.ID(), 0, SpeakerIdentity{Speaker: Speaker{ID: contracts.ID(), Name: "Current identity"}})
	if e != nil {
		t.Fatal(e)
	}
	head, e := s.SetSpeakerProfile(ctx, contracts.ID(), other.Speaker.ID, 0, output.ID)
	if e != nil {
		t.Fatal("explicit current association", e)
	}
	versions, e := s.SpeakerOutputs(ctx, other.Speaker.ID)
	if e != nil || len(versions) != 1 || versions[0].SpeakerID != original.ID {
		t.Fatal("origin rewritten or current discovery absent", e)
	}
	_, target, _ := evidenceRecording(t, s)
	if _, e = s.MutateRoster(ctx, contracts.ID(), target.ID, 0, "replace", []string{other.Speaker.ID}); e != nil {
		t.Fatal(e)
	}
	frozen, e := s.FreezeSpeakerMatching(ctx, target.ID)
	if e != nil || len(frozen.Profiles) != 1 || frozen.Profiles[0].ID != other.Speaker.ID || frozen.Outputs[0].SpeakerID != original.ID {
		t.Fatal("current matching association lost origin", e)
	}
	if _, e = s.SetSpeakerProfile(ctx, contracts.ID(), other.Speaker.ID, head.Revision, ""); e != nil {
		t.Fatal(e)
	}
	versions, e = s.SpeakerOutputs(ctx, other.Speaker.ID)
	if e != nil || len(versions) != 0 {
		t.Fatal("cleared current association remained active", e)
	}
	if _, e = s.SpeakerOutput(ctx, output.ID); e != nil {
		t.Fatal("clear destroyed immutable output", e)
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if e = localStore(t, s.workspace).Restore(ctx, snap); e != nil {
		t.Fatal("portable current association", e)
	}
}
func TestSpeakerReplacementScrubsWorkAndFencesLease(t *testing.T) {
	ctx := context.Background()
	s := localStore(t, contracts.ID())
	_, rec, _ := evidenceRecording(t, s)
	claim := speakerWork(t, s, "recordings.process")
	frozen, e := s.FreezeSpeakerMatching(ctx, rec.ID)
	if e != nil {
		t.Fatal(e)
	}
	requestDigest := hash([]byte("original public matching request"))
	payload, _ := json.Marshal(map[string]any{"snapshot": frozen, "request_digest": requestDigest, "options": map[string]string{"recording_id": rec.ID}})
	w, e := s.EnqueueWork(ctx, contracts.ID(), "recordings.match", payload)
	if e != nil {
		t.Fatal(e)
	}
	w, e = s.ClaimWork(ctx, w.ID, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	replacement := rec
	replacement.State = "no-timed-subtitles"
	replacement.Document = json.RawMessage(`null`)
	replacement.DocumentDigest = ""
	if _, e = s.CommitRecording(ctx, claim, rec.Revision, replacement); e != nil {
		t.Fatal(e)
	}
	current, e := s.Work(ctx, w.ID)
	if e != nil || current.State != "failed" || current.Phase != "evidence-invalidated" {
		t.Fatal("stale work retained live authority", e, current)
	}
	var marker map[string]string
	if json.Unmarshal(current.Payload, &marker) != nil || marker["invalidated_recording_id"] != rec.ID {
		t.Fatal("obsolete snapshot persisted")
	}
	if marker["request_digest"] != requestDigest {
		t.Fatal("accepted public request identity lost during scrub")
	}
	if _, e = s.CheckpointWork(ctx, w, "published", "succeeded", json.RawMessage(`{}`), time.Minute); e == nil {
		t.Fatal("old attempt lease accepted")
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if e = localStore(t, s.workspace).Restore(ctx, snap); e != nil {
		t.Fatal("portable scrub proof", e)
	}
}

func TestSpeakerProfileRejectsAbsentOutput(t *testing.T) {
	s := localStore(t, contracts.ID())
	ctx := context.Background()
	a, e := s.PutSpeaker(ctx, contracts.ID(), 0, SpeakerIdentity{Speaker: Speaker{ID: contracts.ID(), Name: "A"}})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.SetSpeakerProfile(ctx, contracts.ID(), a.Speaker.ID, 0, contracts.ID()); e == nil {
		t.Fatal("absent version accepted")
	}
}
