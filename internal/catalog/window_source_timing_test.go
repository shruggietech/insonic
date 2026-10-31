// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/subtitles"
)

func TestWindowCanonicalSourceEndpointAcceptance(t *testing.T) {
	s := localStore(t, contracts.ID())
	ctx := context.Background()
	claim, entry, recording := recordingFixture(t, s)
	physicalDuration := int64(1_000_000)
	entry.DurationUS = &physicalDuration
	entry, err := s.UpdateLibrary(ctx, contracts.ID(), entry.Revision, entry)
	if err != nil {
		t.Fatal(err)
	}
	recording.SourceRevision = entry.Revision
	recording.SourceMap, _ = json.Marshal(map[string]any{"policy": "windowed-canonical-source-clock-v1", "source_digest": entry.Digest, "tracks": []any{map[string]any{"stream_index": 0, "source_sample_rate": 16000, "source_sample_count": 16000, "start_numerator": "10", "start_denominator": "1", "start_sample": 0, "end_sample": 16000}}})
	raw, err := os.ReadFile("../subtitles/testdata/source.cueson.json")
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]json.RawMessage
	json.Unmarshal(raw, &document)
	var cues []map[string]json.RawMessage
	json.Unmarshal(document["cues"], &cues)
	cues[0]["timing"] = json.RawMessage(`{"start_milliseconds":10000,"end_milliseconds":11000,"duration_milliseconds":1000}`)
	document["cues"], _ = json.Marshal(cues)
	var summary map[string]json.RawMessage
	json.Unmarshal(document["document"], &summary)
	summary["media_start_milliseconds"], summary["media_end_milliseconds"] = json.RawMessage(`10000`), json.RawMessage(`11000`)
	document["document"], _ = json.Marshal(summary)
	raw, _ = json.Marshal(document)
	canonicalEndNS := int64(11_000_000_000)
	assembled, err := subtitles.AssembleDocument(raw, &canonicalEndNS, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	recording.State, recording.Document, recording.DocumentDigest = "ready", assembled.Document, hash(assembled.Document)
	accepted, err := s.CommitRecording(ctx, claim, 0, recording)
	if err != nil {
		t.Fatal("positive source origin was clipped to elapsed physical duration", err)
	}
	snapshot, err := s.Export(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = localStore(t, s.workspace).Restore(ctx, snapshot); err != nil {
		t.Fatal("canonical endpoint did not survive restore", err)
	}
	json.Unmarshal(accepted.Document, &document)
	var timing map[string]json.RawMessage
	json.Unmarshal(document["media_timing"], &timing)
	timing["duration_milliseconds"] = json.RawMessage(`1000`)
	document["media_timing"], _ = json.Marshal(timing)
	accepted.Document, _ = json.Marshal(document)
	accepted.DocumentDigest = hash(accepted.Document)
	if _, err = s.CommitRecording(ctx, speakerWork(t, s, "recordings.process"), accepted.Revision, accepted); err == nil {
		t.Fatal("elapsed duration substituted for canonical source endpoint")
	}
}

func TestWindowSourceEndpointExactRounding(t *testing.T) {
	digest := hash([]byte("source"))
	for _, tc := range []struct {
		numerator, denominator string
		end                    int64
		want                   *int64
	}{{"1", "3", 16000, int64Pointer(1333)}, {"-2", "1", 16000, nil}, {"0", "1", 0, nil}, {"0", "1", 1, int64Pointer(0)}} {
		raw, _ := json.Marshal(map[string]any{"policy": "windowed-canonical-source-clock-v1", "source_digest": digest, "tracks": []any{map[string]any{"stream_index": 0, "source_sample_rate": 16000, "source_sample_count": 16000, "start_numerator": tc.numerator, "start_denominator": tc.denominator, "start_sample": 0, "end_sample": tc.end}}})
		got, err := windowSourceDurationMS(raw, digest)
		if err != nil || (got == nil) != (tc.want == nil) || got != nil && *got != *tc.want {
			t.Fatal("source endpoint rounding", got, tc.want, err)
		}
	}
}

func int64Pointer(value int64) *int64 { return &value }
