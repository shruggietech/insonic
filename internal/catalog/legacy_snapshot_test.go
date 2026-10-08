// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
	"testing"
)

func TestLegacySnapshotConvertsEvidenceWithoutArchivingCopies(t *testing.T) {
	ctx := context.Background()
	s := localStore(t, contracts.ID())
	source, manifest, clip := availableArtifact(t, s), availableArtifact(t, s), availableArtifact(t, s)
	asset, speaker, segment, dataset := contracts.ID(), contracts.ID(), contracts.ID(), contracts.ID()
	rev, _ := s.Revision(ctx)
	_, e := s.Commit(ctx, Mutation{OperationID: contracts.ID(), Expected: rev, Records: Records{Assets: []Asset{{asset, source.ArtifactID, "original", 1000000}}, Speakers: []Speaker{{ID: speaker, Name: "Known"}}}})
	if e != nil {
		t.Fatal(e)
	}
	base, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(base.Records)
	var fields map[string]json.RawMessage
	json.Unmarshal(raw, &fields)
	for i := range base.State {
		table := &base.State[i]
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
	fields["speakers"], _ = json.Marshal([]map[string]any{{"id": speaker, "name": "Known"}})
	fields["segments"], _ = json.Marshal([]legacySegment{{ID: segment, Revision: 1, AssetID: asset, SpeakerID: &speaker, StartUS: 0, EndUS: 1000000, Channel: 0, Attribution: json.RawMessage(`{"speaker_attributions":["copied-evidence-marker"]}`), ClipArtifactID: &clip.ArtifactID}})
	fields["datasets"], _ = json.Marshal([]map[string]any{{"id": dataset, "speaker_id": speaker, "manifest_artifact_id": manifest.ArtifactID, "options": map[string]any{"transcript": "copied-evidence-marker"}}})
	fields["members"], _ = json.Marshal([]DatasetMember{{contracts.ID(), dataset, 0, segment, 1}})
	raw, _ = json.Marshal(fields)
	envelope := snapshotEnvelope{Kind: base.Kind, Version: base.Version, CatalogSchema: 3, WorkspaceID: base.WorkspaceID, Revision: base.Revision, Records: raw, State: base.State}
	envelope.Digest, _ = intent(envelope)
	encoded, _ := json.Marshal(envelope)
	migrated, e := ReadSnapshotReader(bytes.NewReader(encoded))
	if e != nil {
		t.Fatal(e)
	}
	converted, _ := json.Marshal(migrated)
	serialized, e := ReadSnapshot(converted)
	if e != nil {
		t.Fatal(e)
	}
	if e = localStore(t, s.workspace).Restore(ctx, serialized); e == nil {
		t.Fatal("legacy conversion lost physical obligations through serialization")
	}
	if strings.Contains(string(converted), "copied-evidence-marker") || len(migrated.Records.Segments)+len(migrated.Records.Members) != 0 {
		t.Fatal("legacy assignments archived")
	}
	dst := localStore(t, s.workspace)
	if e = dst.Restore(ctx, migrated); e != nil {
		t.Fatal(e)
	}
	current, e := dst.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if len(current.Records.Cleanups) != 2 || len(current.Records.Assets) != 1 {
		t.Fatal("derived obligations or original identity lost")
	}
	clean, _ := json.Marshal(current)
	if strings.Contains(string(clean), "copied-evidence-marker") {
		t.Fatal("receipt archived obsolete assignments")
	}
	envelope.Digest = strings.Repeat("0", 64)
	bad, _ := json.Marshal(envelope)
	if _, e = ReadSnapshot(bad); e == nil {
		t.Fatal("legacy checksum bypassed during conversion")
	}
	envelope.Digest = ""
	fields["members"] = json.RawMessage(`[{"id":"10000000-0000-4000-8000-000000000001","start_us":1}]`)
	envelope.Records, _ = json.Marshal(fields)
	envelope.Digest, _ = intent(envelope)
	bad, _ = json.Marshal(envelope)
	if _, e = ReadSnapshot(bad); e == nil {
		t.Fatal("malformed legacy membership silently discarded")
	}
	current.CatalogSchema = SchemaVersion
	modern, _ := json.Marshal(current)
	var document map[string]json.RawMessage
	json.Unmarshal(modern, &document)
	json.Unmarshal(document["records"], &fields)
	fields["segments"] = json.RawMessage(`[{"start_us":0,"attribution":{}}]`)
	document["records"], _ = json.Marshal(fields)
	bad, _ = json.Marshal(document)
	if _, e = ReadSnapshot(bad); e == nil {
		t.Fatal("schema4 copied evidence accepted")
	}
}
