// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/shruggietech/insonic/internal/answers"
	"github.com/shruggietech/insonic/internal/contracts"
)

func answerEvidenceSuite(t *testing.T, s *Store) {
	ctx := context.Background()
	_, recording, local := evidenceRecording(t, s)
	_, second, _ := evidenceRecording(t, s)
	epoch, err := s.AnswerEpoch(ctx, nil, 100, 1<<20)
	if err != nil || !epoch.Complete || !answers.ValidDigest(epoch.Digest) {
		t.Fatal(epoch, err)
	}
	request := AnswerPageRequest{Limit: 1, MaxBytes: 1 << 20}
	seen := map[string]bool{}
	for range 20 {
		page, err := s.AnswerCandidates(ctx, request)
		if err != nil || page.Rows > 1 || page.Bytes > request.MaxBytes {
			t.Fatal(page, err)
		}
		for _, item := range page.Items {
			if seen[item.ID] || answers.ValidateReference(item.Reference) != nil {
				t.Fatal("duplicate/invalid reference", item)
			}
			seen[item.ID] = true
		}
		if page.Next == "" {
			break
		}
		if request.Cursor == page.Next {
			t.Fatal("nonadvancing continuation")
		}
		request.Cursor = page.Next
	}
	if len(seen) < 2 {
		t.Fatal("multiple recordings not returned", seen)
	}
	page, err := s.AnswerCandidates(ctx, AnswerPageRequest{RecordingID: recording.ID, Limit: 10, MaxBytes: 1 << 20})
	if err != nil || len(page.Items) == 0 {
		t.Fatal(page, err)
	}
	for _, item := range page.Items {
		if item.Reference.RecordingID != recording.ID {
			t.Fatal("scope leak")
		}
	}
	ref := page.Items[0].Reference
	resolved, err := s.ResolveAnswerEvidence(ctx, ref, 1<<20)
	if err != nil || resolved.ID != page.Items[0].ID {
		t.Fatal(resolved, err)
	}
	ref.WorkspaceID = contracts.ID()
	if _, err = s.ResolveAnswerEvidence(ctx, ref, 1<<20); err == nil {
		t.Fatal("cross workspace accepted")
	}
	ref = page.Items[0].Reference
	ref.DocumentDigest = strings.Repeat("e", 64)
	if _, err = s.ResolveAnswerEvidence(ctx, ref, 1<<20); err == nil {
		t.Fatal("stale digest accepted")
	}
	tiny, err := s.AnswerCandidates(ctx, AnswerPageRequest{Limit: 10, MaxBytes: 1})
	if err != nil || len(tiny.Items) != 0 || tiny.Bytes > 1 || tiny.LimitReason != "source_bytes" {
		t.Fatal("byte cap", tiny, err)
	}
	limited, err := s.AnswerEpoch(ctx, nil, 1, 1<<20)
	if err != nil || limited.Complete || limited.Rows > 1 {
		t.Fatal("epoch row cap", limited, err)
	}
	_, err = s.PutNamedSetting(ctx, contracts.ID(), "answer-context:test", 0, []byte(`{"key":"value"}`))
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.AnswerEpoch(ctx, nil, 100, 1<<20)
	if err != nil || again.Digest != epoch.Digest {
		t.Fatal("cache metadata invalidates source epoch", again, err)
	}
	graphBefore, err := s.GraphStatus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.PutNamedSetting(ctx, contracts.ID(), "answer-context:"+strings.Repeat("a", 64), 0, []byte(`{"key":"value"}`)); err != nil {
		t.Fatal(err)
	}
	graphAfter, err := s.GraphStatus(ctx)
	if err != nil || graphBefore["next_sequence"] != graphAfter["next_sequence"] {
		t.Fatal("derived cache index dirtied source graph", graphBefore, graphAfter, err)
	}
	person, err := s.PutSpeaker(ctx, contracts.ID(), 0, SpeakerIdentity{Speaker: Speaker{ID: contracts.ID(), Name: "Known"}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.SetSpeakerMapping(ctx, contracts.ID(), recording.Revision, SpeakerMapping{RecordingID: recording.ID, LocalSpeakerID: local, SpeakerID: person.Speaker.ID, DocumentDigest: recording.DocumentDigest})
	if err != nil {
		t.Fatal(err)
	}
	changed, err := s.AnswerEpoch(ctx, nil, 100, 1<<20)
	if err != nil || changed.Digest == epoch.Digest {
		t.Fatal("mapping identity not invalidated", changed, err)
	}
	if _, err = s.ResolveAnswerEvidence(ctx, page.Items[0].Reference, 1<<20); err == nil {
		t.Fatal("old mapping identity accepted")
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err = s.AnswerCandidates(cancelled, AnswerPageRequest{Limit: 1, MaxBytes: 1000}); err == nil {
		t.Fatal("cancelled read continued")
	}
	scoped, err := s.AnswerCandidates(ctx, AnswerPageRequest{MediaIDs: []string{second.ID}, Limit: 10, MaxBytes: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range scoped.Items {
		if item.Reference.RecordingID != second.ID {
			t.Fatal("media scope leak")
		}
	}
}
func TestSQLiteAnswerEvidence(t *testing.T) { answerEvidenceSuite(t, localStore(t, contracts.ID())) }

func answerNeighborBoundarySuite(t *testing.T, s *Store) {
	ctx := context.Background()
	_, r, _ := evidenceRecording(t, s)
	var doc map[string]json.RawMessage
	_ = json.Unmarshal(r.Document, &doc)
	var cues []map[string]json.RawMessage
	_ = json.Unmarshal(doc["cues"], &cues)
	for i := 1; i < 3; i++ {
		var cue map[string]json.RawMessage
		raw, _ := json.Marshal(cues[0])
		_ = json.Unmarshal(raw, &cue)
		cue["id"], _ = json.Marshal("neighbor-" + string(rune('0'+i)))
		cues = append(cues, cue)
	}
	doc["cues"], _ = json.Marshal(cues)
	raw, _ := json.Marshal(doc)
	if _, err := s.db.ExecContext(ctx, s.query("UPDATE current_recording SET document=?,document_digest=? WHERE workspace_id=? AND id=?"), string(raw), hash(raw), s.workspace, r.ID); err != nil {
		t.Fatal(err)
	}
	lo, hi := int64(0), int64(2)
	page, err := s.AnswerCandidates(ctx, AnswerPageRequest{RecordingID: r.ID, FirstOrdinal: &lo, LastOrdinal: &hi, Limit: 1, MaxBytes: 1 << 20})
	if err != nil || page.LimitReason != "neighbor_rows" {
		t.Fatal("truncated neighbor range unreported", page, err)
	}
	cues[1]["id"] = cues[0]["id"]
	doc["cues"], _ = json.Marshal(cues)
	raw, _ = json.Marshal(doc)
	if _, err = s.db.ExecContext(ctx, s.query("UPDATE current_recording SET document=?,document_digest=? WHERE workspace_id=? AND id=?"), string(raw), hash(raw), s.workspace, r.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.AnswerCandidates(ctx, AnswerPageRequest{RecordingID: r.ID, Limit: 1, MaxBytes: 1 << 20}); err == nil {
		t.Fatal("ambiguous current cue admitted")
	}
}
func TestSQLiteAnswerNeighborBoundaries(t *testing.T) {
	answerNeighborBoundarySuite(t, localStore(t, contracts.ID()))
}

func answerOriginSuite(t *testing.T, s *Store) {
	ctx := context.Background()
	_, one, _ := evidenceRecording(t, s)
	_, two, _ := evidenceRecording(t, s)
	items := []answers.Item{}
	for i, r := range []Recording{one, two} {
		stream := 2 + i*3
		facts, _ := json.Marshal(map[string]any{"canonical": map[string]any{"source_digest": strings.Repeat("a", 64), "tracks": []map[string]any{{"index": stream, "source_index": 7}}}})
		sourceMap, _ := json.Marshal(map[string]any{"stream_index": stream})
		if _, err := s.db.ExecContext(ctx, s.query("UPDATE library_entry SET facts=? WHERE workspace_id=? AND id=?"), string(facts), s.workspace, r.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := s.db.ExecContext(ctx, s.query("UPDATE current_recording SET source_map=? WHERE workspace_id=? AND id=?"), string(sourceMap), s.workspace, r.ID); err != nil {
			t.Fatal(err)
		}
		page, err := s.AnswerCandidates(ctx, AnswerPageRequest{RecordingID: r.ID, Limit: 10, MaxBytes: 1 << 20})
		if err != nil || len(page.Items) != 1 {
			t.Fatal(page, err)
		}
		item := page.Items[0]
		if item.Lineage != "canonical-origin" || item.OriginStreamIndex == nil || *item.OriginStreamIndex != 7 {
			t.Fatal("canonical lineage lost", item)
		}
		items = append(items, item)
	}
	ranked, _, _, families := answers.Rank("Research", items)
	if families != 1 || ranked[0].OverlapGroup != ranked[1].OverlapGroup {
		t.Fatal("alternate encodings inflated support", ranked, families)
	}
	facts := `{"canonical":{"source_digest":"` + strings.Repeat("a", 64) + `","tracks":[{"index":5}]}}`
	if _, err := s.db.ExecContext(ctx, s.query("UPDATE library_entry SET facts=? WHERE workspace_id=? AND id=?"), facts, s.workspace, two.ID); err != nil {
		t.Fatal(err)
	}
	page, err := s.AnswerCandidates(ctx, AnswerPageRequest{RecordingID: two.ID, Limit: 10, MaxBytes: 1 << 20})
	if err != nil || len(page.Items) != 1 || page.Items[0].OriginStreamIndex != nil {
		t.Fatal("missing original stream guessed", page, err)
	}
}
func TestSQLiteAnswerOrigins(t *testing.T) { answerOriginSuite(t, localStore(t, contracts.ID())) }
