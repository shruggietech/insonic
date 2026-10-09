// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
	"testing"
	"time"
)

func TestImportModelElectionProofSurvivesAcceptedInputScrub(t *testing.T) {
	s := localStore(t, contracts.ID())
	ctx := context.Background()
	_, entry := entryFixture(t, s)
	model, dependency := contracts.ID(), contracts.ID()
	envelope := map[string]any{"kind": "import-manifest", "schema_version": contracts.Version, "model_selections": []any{map[string]any{"target": map[string]any{"kind": "base", "id": model, "operation": "diarization"}, "manifest_digest": strings.Repeat("a", 64)}}, "model_dependencies": []string{dependency}, "items": []any{map[string]any{"source": "/fixture/audio.wav", "diarization_model_id": model, "diarization_model_digest": strings.Repeat("a", 64)}}}
	payload, _ := json.Marshal(envelope)
	before, _, e := importInputProof(payload)
	if e != nil {
		t.Fatal(e)
	}
	work, e := s.EnqueueWork(ctx, contracts.ID(), "library.import", payload)
	if e != nil {
		t.Fatal(e)
	}
	claim, e := s.ClaimWork(ctx, work.ID, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	if _, _, e = s.CommitAdmission(ctx, claim, 0, 0, entry, nil); e != nil {
		t.Fatal(e)
	}
	scrubbed, e := s.Work(ctx, claim.ID)
	if e != nil {
		t.Fatal(e)
	}
	after, _, e := importInputProof(scrubbed.Payload)
	if e != nil || before != after {
		t.Fatal("frozen envelope changed", e)
	}
	var got map[string]json.RawMessage
	if json.Unmarshal(scrubbed.Payload, &got) != nil {
		t.Fatal("payload")
	}
	if !strings.Contains(string(got["model_selections"]), model) || !strings.Contains(string(got["model_dependencies"]), dependency) {
		t.Fatal("model authority lost")
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if e = localStore(t, s.workspace).Restore(ctx, snap); e != nil {
		t.Fatal("portable frozen import", e)
	}
	for i, v := range snap.Records.Works {
		if v.ID == claim.ID {
			got["model_dependencies"], _ = json.Marshal([]string{contracts.ID()})
			snap.Records.Works[i].Payload, _ = json.Marshal(got)
		}
	}
	snap.Digest, _ = snap.digest()
	if e = localStore(t, s.workspace).Restore(ctx, snap); e == nil {
		t.Fatal("retargeted frozen dependency accepted")
	}
}
