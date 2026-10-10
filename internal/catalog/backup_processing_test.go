// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/contracts"
)

func TestPortableRejectsUndeclaredAbsoluteArgumentsAndDeterministicallyBindsOverlaps(t *testing.T) {
	for _, argument := range []string{"--config=/source/private.json", `--config=C:\source\private.json`, `\\server\private\input.json`} {
		payload, _ := json.Marshal(map[string]any{"options": map[string]any{"adapter": map[string]any{"arguments": []string{argument}}}})
		if _, _, e := portableProcessingPayload(Work{Kind: "models.train", Payload: payload}); e == nil {
			t.Fatal("undeclared absolute dependency admitted", argument)
		}
	}
	short := `C:\tools\worker`
	long := short + ".py"
	first := hash([]byte("first"))
	second := hash([]byte("second"))
	payload, _ := json.Marshal(map[string]any{"options": map[string]any{"adapter": map[string]any{"executable": map[string]any{"path": short, "sha256": first}, "support_files": []any{map[string]any{"path": long, "sha256": second}}, "arguments": []string{long, "--threads=2", "/verbose"}}}})
	var expected string
	for i := 0; i < 20; i++ {
		output, changed, e := portableProcessingPayload(Work{Kind: "models.train", Payload: payload})
		if e != nil || !changed {
			t.Fatal(e)
		}
		if expected == "" {
			expected = string(output)
		}
		if string(output) != expected || strings.Contains(string(output), ".py") || !strings.Contains(string(output), "/verbose") {
			t.Fatal("overlapping pin substitution was nondeterministic")
		}
	}
}

func TestPortableAdapterOriginDigestNeedsNativeAcceptance(t *testing.T) {
	ctx := context.Background()
	source := localStore(t, contracts.ID())
	payload := json.RawMessage(`{"options":{"adapter":{"executable":{"path":"C:\\source\\trainer.exe","sha256":"` + hash([]byte("trainer")) + `"},"arguments":["--threads=2"]}}}`)
	if _, e := source.EnqueueWork(ctx, contracts.ID(), "models.train", payload); e != nil {
		t.Fatal(e)
	}
	snap, e := source.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	portable, e := PortableSnapshot(ctx, snap)
	if e != nil {
		t.Fatal(e)
	}
	changed := false
	for _, table := range portable.State {
		if table.Name != "operation_receipt" {
			continue
		}
		for _, row := range table.Rows {
			var encoded string
			if json.Unmarshal(row[3], &encoded) != nil {
				t.Fatal("invalid fixture receipt")
			}
			var result map[string]json.RawMessage
			if json.Unmarshal([]byte(encoded), &result) != nil {
				t.Fatal("invalid fixture result")
			}
			if _, exists := result["portable_adapter_digest"]; !exists {
				continue
			}
			result["portable_adapter_digest"], _ = json.Marshal(hash([]byte("altered adapter compatibility")))
			raw, _ := json.Marshal(result)
			row[3], _ = json.Marshal(string(raw))
			changed = true
		}
	}
	if !changed {
		t.Fatal("missing fixture transformation proof")
	}
	portable.Digest, _ = portable.digest()
	if ValidateSnapshot(ctx, portable) == nil {
		t.Fatal("origin adapter compatibility changed without native receipt acceptance")
	}
}

func TestPortableTrainingRunExcludesHostBindingsPreservesLineage(t *testing.T) {
	ctx := context.Background()
	source := localStore(t, contracts.ID())
	_, _, model, _ := portableAuthorityFixture(t, source)
	snap, e := source.Export(ctx)
	if e != nil || len(snap.Records.Runs) != 1 {
		t.Fatal(e)
	}
	run := snap.Records.Runs[0]
	run.Options = json.RawMessage(`{"executable":{"path":"C:\\source-private\\trainer.exe","sha256":"` + hash([]byte("trainer")) + `"},"epochs":9007199254740993}`)
	if e = source.write(ctx, func(tx *sql.Tx, rev int64) error {
		if e := source.putDomain(ctx, tx, "Runs", run); e != nil {
			return e
		}
		_, e := source.accept(ctx, tx, contracts.ID(), hash([]byte("accepted fixture training options")), rev, map[string]any{"run_id": run.ID})
		return e
	}); e != nil {
		t.Fatal(e)
	}
	snap, e = source.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	portable, e := PortableSnapshot(ctx, snap)
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(string(portable.Records.Runs[0].Options), "source-private") || !strings.Contains(string(portable.Records.Runs[0].Options), "9007199254740993") || portable.Records.Runs[0].ID != run.ID || portable.Records.SpeakerOutputs[0].ID != model.ID {
		t.Fatal("training lineage or exact parameter changed")
	}
	if e = ValidateSnapshot(ctx, portable); e != nil {
		t.Fatal("native transformed training lineage", e)
	}
	unchanged, e := source.Export(ctx)
	if e != nil || string(unchanged.Records.Runs[0].Options) != string(run.Options) {
		t.Fatal("source training lineage changed", e)
	}
}

func TestPortableProcessingAllKindsPreserveElectionAndHistory(t *testing.T) {
	for _, kind := range []string{"recordings.process", "models.ensure", "models.train", "recordings.match"} {
		for _, state := range []string{"pending", "interrupted", "succeeded"} {
			t.Run(kind+"/"+state, func(t *testing.T) {
				ctx := context.Background()
				source := localStore(t, contracts.ID())
				path := `C:\private-tools\worker.exe`
				hint := "TransientRecognitionHint"
				digest := hash([]byte("pinned fixture executable"))
				selected := contracts.ID()
				dependency := contracts.ID()
				payload, _ := json.Marshal(map[string]any{"tools": map[string]any{"cueson": map[string]any{"executable": path, "executable_sha256": digest}}, "processing_tools": map[string]any{"processing": map[string]any{"worker": map[string]any{"path": path, "sha256": digest}}}, "model_dependencies": []string{dependency}, "model_selections": []any{map[string]any{"target": map[string]any{"id": selected}, "manifest_digest": digest}}, "options": map[string]any{"recognition": map[string]any{"hints": []string{hint}, "context_digest": digest}, "adapter": map[string]any{"executable": map[string]any{"path": path, "sha256": digest}, "arguments": []string{"--worker=" + path}}, "parameters": map[string]any{"steps": 20}}, "election": map[string]any{"context": map[string]any{"hints": []string{hint}, "joined": hint, "digest": digest}, "configuration": map[string]any{"recognition": map[string]any{"recognition": map[string]any{"hints": []string{hint}}}}}, "snapshot": map[string]any{"digest": digest, "recording": map[string]any{"document": hint}}})
				if kind != "recordings.match" {
					var value map[string]any
					json.Unmarshal(payload, &value)
					delete(value, "snapshot")
					payload, _ = json.Marshal(value)
				}
				w, e := source.EnqueueWork(ctx, contracts.ID(), kind, payload)
				if e != nil {
					t.Fatal(e)
				}
				if state != "pending" {
					claim, e := source.ClaimWork(ctx, w.ID, contracts.ID(), time.Second)
					if e != nil {
						t.Fatal(e)
					}
					checkpointState := state
					if state == "interrupted" {
						checkpointState = "running"
					}
					w, e = source.CheckpointWork(ctx, claim, "fixture", checkpointState, json.RawMessage(`{"state":"accepted","version_id":"`+selected+`"}`), time.Second)
					if e != nil {
						t.Fatal(e)
					}
					if state == "interrupted" {
						if e = source.InterruptOwner(ctx, claim.Owner); e != nil {
							t.Fatal(e)
						}
						w, e = source.Work(ctx, w.ID)
						if e != nil {
							t.Fatal(e)
						}
					}
				}
				snap, e := source.Export(ctx)
				if e != nil {
					t.Fatal(e)
				}
				portable, e := PortableSnapshot(ctx, snap)
				if e != nil {
					t.Fatal(e)
				}
				raw, _ := json.Marshal(portable)
				if strings.Contains(string(raw), "private-tools") || strings.Contains(string(raw), hint) {
					t.Fatal("private work input survived portable capture")
				}
				original, e := source.Work(ctx, w.ID)
				if e != nil || string(original.Payload) != string(payload) || original.State != state {
					t.Fatal("source work changed", e)
				}
				dest := localStore(t, source.workspace)
				if e = dest.Restore(ctx, portable); e != nil {
					t.Fatal("native portable proof", e)
				}
				got, e := dest.Work(ctx, w.ID)
				if e != nil || got.State != state || string(got.Result) != string(w.Result) {
					t.Fatal("work election/history changed", e)
				}
				if !strings.Contains(string(got.Payload), selected) || !strings.Contains(string(got.Payload), dependency) || !strings.Contains(string(got.Payload), `"steps":20`) {
					t.Fatal("frozen model identity or parameters changed")
				}
				proven, adapter, e := dest.PortableWorkInput(ctx, got)
				if e != nil || !proven || !digestPattern.MatchString(adapter) {
					t.Fatal("native portable input proof missing", e)
				}
				if state != "succeeded" {
					claim, e := dest.ClaimWork(ctx, w.ID, contracts.ID(), time.Second)
					if e != nil {
						t.Fatal(e)
					}
					if _, e = dest.CheckpointWork(ctx, claim, "recovered", "succeeded", json.RawMessage(`{"state":"accepted"}`), time.Second); e != nil {
						t.Fatal(e)
					}
					after, e := dest.Export(ctx)
					if e != nil || ValidateSnapshot(ctx, after) != nil {
						t.Fatal("claim journal lost transformation proof", e)
					}
				}
				// A second capture is idempotent; it retains the native transformed proof.
				again, e := PortableSnapshot(ctx, portable)
				if e != nil || again.Digest != portable.Digest {
					t.Fatal("portable recapture changed an already portable election", e)
				}
			})
		}
	}
}

func TestPortableProcessingPayloadChangeNeedsNativeProof(t *testing.T) {
	ctx := context.Background()
	source := localStore(t, contracts.ID())
	path := `A:\source\worker.exe`
	_, e := source.EnqueueWork(ctx, contracts.ID(), "recordings.process", json.RawMessage(`{"tools":{"processing":{"worker":{"path":"`+strings.ReplaceAll(path, `\`, `\\`)+`","sha256":"`+hash([]byte("worker"))+`"}}}}`))
	if e != nil {
		t.Fatal(e)
	}
	snap, _ := source.Export(ctx)
	portable, e := PortableSnapshot(ctx, snap)
	if e != nil {
		t.Fatal(e)
	}
	portable.Records.Works[0].Payload = json.RawMessage(`{}`)
	// Recomputing the outer snapshot cannot replace input acceptance evidence.
	portable.Digest, _ = portable.digest()
	if ValidateSnapshot(ctx, portable) == nil {
		t.Fatal("unproven input replacement accepted")
	}
}
