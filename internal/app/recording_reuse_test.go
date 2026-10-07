// SPDX-License-Identifier: Apache-2.0
package app

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/subtitles"
	"os"
	"testing"
)

func overlappingReuseDocument(t *testing.T) []byte {
	t.Helper()
	raw, e := os.ReadFile("../subtitles/testdata/source.cueson.json")
	if e != nil {
		t.Fatal(e)
	}
	var d map[string]json.RawMessage
	json.Unmarshal(raw, &d)
	var cues []map[string]json.RawMessage
	json.Unmarshal(d["cues"], &cues)
	clone := map[string]json.RawMessage{}
	for k, v := range cues[0] {
		clone[k] = v
	}
	clone["id"] = json.RawMessage(`"cue-000001"`)
	clone["ordinal"] = json.RawMessage(`1`)
	clone["source_order"] = json.RawMessage(`1`)
	clone["timing"] = json.RawMessage(`{"start_milliseconds":500,"end_milliseconds":1500,"duration_milliseconds":1000}`)
	cues = append(cues, clone)
	d["cues"], _ = json.Marshal(cues)
	d["document"] = json.RawMessage(`{"cue_count":2,"media_start_milliseconds":0,"media_end_milliseconds":1500,"media_span_milliseconds":1500,"has_word_level_timing":false}`)
	d["stats"] = json.RawMessage(`{"cue_count":2,"diagnostic_count":0,"warning_count":0,"error_count":0,"has_word_level_timing":false,"media_span_milliseconds":1500}`)
	raw, _ = json.Marshal(d)
	return raw
}
func TestDiarizationReusePreservesPerCueMultiplicityAcrossOverlappingCues(t *testing.T) {
	raw := overlappingReuseDocument(t)
	voice := contracts.ID()
	turn := subtitles.Turn{SpeakerID: voice, StartNS: 0, EndNS: 1500000000}
	original, e := subtitles.AssembleDocument(raw, nil, []subtitles.Turn{turn, turn}, nil)
	if e != nil {
		t.Fatal(e)
	}
	reused, participation, e := currentTurns(original.Document)
	if e != nil {
		t.Fatal(e)
	}
	if len(reused) != 2 || reused[0] != turn || reused[1] != turn {
		t.Fatalf("fragmented or collapsed reuse: %+v", reused)
	}
	restored, e := subtitles.AssembleDocument(raw, nil, reused, participation)
	if e != nil {
		t.Fatal(e)
	}
	var d struct {
		Cues []struct {
			Assignments []json.RawMessage `json:"speaker_attributions"`
		} `json:"cues"`
	}
	json.Unmarshal(restored.Document, &d)
	for _, cue := range d.Cues {
		if len(cue.Assignments) != 2 {
			t.Fatal("reuse fabricated or collapsed assignments")
		}
	}
}
func TestUntimedReuseRequiresUnchangedCueEvidence(t *testing.T) {
	raw := overlappingReuseDocument(t)
	voice := contracts.ID()
	p := []subtitles.Participation{{CueID: "cue-000000", SpeakerID: voice}, {CueID: "cue-000000", SpeakerID: voice}}
	unchanged, diagnostics, e := reuseParticipation(raw, raw, p)
	if e != nil || len(unchanged) != 2 || len(diagnostics) != 0 {
		t.Fatal("unchanged participation lost", e)
	}
	var d map[string]json.RawMessage
	json.Unmarshal(raw, &d)
	var cues []map[string]json.RawMessage
	json.Unmarshal(d["cues"], &cues)
	cues[0]["payload"] = json.RawMessage(`{"raw_text":"Changed evidence","plain_text":"Changed evidence","lines":["Changed evidence"]}`)
	d["cues"], _ = json.Marshal(cues)
	changed, _ := json.Marshal(d)
	carried, diagnostics, e := reuseParticipation(raw, changed, p)
	if e != nil || len(carried) != 0 || len(diagnostics) != 2 || diagnostics[0].Code != "untimed_reuse_omitted" {
		t.Fatal("participation attributed to changed evidence", e)
	}
}
