package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
)

func TestHistoricalV4SnapshotIdentityMigrationAndDowngradeRejection(t *testing.T) {
	ctx := context.Background()
	s := localStore(t, contracts.ID())
	identity, e := s.PutSpeaker(ctx, contracts.ID(), 0, SpeakerIdentity{Speaker: Speaker{ID: contracts.ID(), Name: "Historical"}})
	if e != nil {
		t.Fatal(e)
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(snap.Records)
	var fields map[string]json.RawMessage
	json.Unmarshal(raw, &fields)
	fields["speakers"], _ = json.Marshal([]map[string]any{{"id": identity.Speaker.ID, "name": "Historical"}})
	raw, _ = json.Marshal(fields)
	envelope := snapshotEnvelope{Kind: snap.Kind, Version: snap.Version, CatalogSchema: 4, WorkspaceID: snap.WorkspaceID, Revision: snap.Revision, Records: raw, State: snap.State}
	envelope.Digest, _ = intent(envelope)
	data, _ := json.Marshal(envelope)
	if _, e = ReadSnapshot(data); e == nil {
		t.Fatal("schema5 authority downgraded to historical shape")
	}
	for i := range envelope.State {
		table := &envelope.State[i]
		if table.Name != "operation_receipt" {
			continue
		}
		for _, row := range table.Rows {
			var text string
			json.Unmarshal(row[3], &text)
			var result map[string]json.RawMessage
			json.Unmarshal([]byte(text), &result)
			delete(result, "speaker_proofs")
			raw, _ := json.Marshal(result)
			row[3], _ = json.Marshal(string(raw))
		}
	}
	envelope.Digest = ""
	envelope.Digest, _ = intent(envelope)
	data, _ = json.Marshal(envelope)
	upgraded, e := ReadSnapshotReader(bytes.NewReader(data))
	if e != nil {
		t.Fatal(e)
	}
	if e = localStore(t, s.workspace).Restore(ctx, upgraded); e != nil {
		t.Fatal("genuine historical identity", e)
	}
	// Conversion authority is private until an accepted restore emits its proof.
	serialized, _ := json.Marshal(upgraded)
	detached, e := ReadSnapshot(serialized)
	if e != nil {
		t.Fatal(e)
	}
	if e = localStore(t, s.workspace).Restore(ctx, detached); e == nil {
		t.Fatal("unaccepted conversion acquired journal authority")
	}
}
func TestPipelineAndSpeakerLatestProofRejectRollback(t *testing.T) {
	ctx := context.Background()
	s := localStore(t, contracts.ID())
	identity, e := s.PutSpeaker(ctx, contracts.ID(), 0, SpeakerIdentity{Speaker: Speaker{ID: contracts.ID(), Name: "Before"}})
	if e != nil {
		t.Fatal(e)
	}
	old := identity
	identity.Speaker.Name = "After"
	if _, e = s.PutSpeaker(ctx, contracts.ID(), identity.Speaker.Revision, identity); e != nil {
		t.Fatal(e)
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	snap.Records.Speakers[0] = old.Speaker
	snap.Digest, _ = snap.digest()
	if e = localStore(t, s.workspace).Restore(ctx, snap); e == nil {
		t.Fatal("speaker rollback accepted")
	}
	config, _ := json.Marshal(map[string]any{"recognition": map[string]any{"adapter": "faster-whisper", "contract_version": "1", "mode": "local", "model_id": contracts.ID()}, "diarization": map[string]any{"adapter": "pyannote", "contract_version": "1", "mode": "local", "model_id": contracts.ID()}})
	p, e := s.PutPipeline(ctx, contracts.ID(), 0, Pipeline{ID: contracts.ID(), Name: "Before", Preset: "local", Configuration: config})
	if e != nil {
		t.Fatal(e)
	}
	before := p
	p.Name = "After"
	if _, e = s.PutPipeline(ctx, contracts.ID(), p.Revision, p); e != nil {
		t.Fatal(e)
	}
	snap, e = s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	snap.Records.Pipelines[0] = before
	snap.Digest, _ = snap.digest()
	if e = localStore(t, s.workspace).Restore(ctx, snap); e == nil {
		t.Fatal("pipeline rollback accepted")
	}
	snap.Records.Pipelines = nil
	snap.Digest, _ = snap.digest()
	if e = localStore(t, s.workspace).Restore(ctx, snap); e == nil {
		t.Fatal("pipeline omission accepted")
	}
}
