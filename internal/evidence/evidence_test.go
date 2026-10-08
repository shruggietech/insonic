package evidence

import (
	"context"
	"encoding/json"
	"testing"
)

func testCues() []Cue {
	return []Cue{{ID: "a", Text: "I did not approve it.", StartMS: 0, EndMS: 1000, Timed: true, Voices: []string{"11111111-1111-4111-8111-111111111111"}}, {ID: "b", Text: "If it changes, we may approve.", StartMS: 1000, EndMS: 2000, Timed: true}, {ID: "c", Text: "A quotation.", Timed: false}}
}
func TestCompleteCueWindowsAndDefaultReferences(t *testing.T) {
	cues := testCues()
	cfg := DefaultConfig()
	cfg.WindowCues = 2
	cfg.OverlapCues = 1
	chunks, e := Chunks(cues, cfg)
	if e != nil {
		t.Fatal(e)
	}
	seen := map[string]bool{}
	for _, c := range chunks {
		for _, cue := range c.Cues {
			seen[cue.ID] = true
		}
	}
	if len(seen) != 3 {
		t.Fatal("lost cues", chunks)
	}
	result, e := Extract(context.Background(), cues, cfg, nil)
	if e != nil || len(result.Assertions) != 3 {
		t.Fatal(result, e)
	}
	for _, a := range result.Assertions {
		if a.Polarity != "unspecified" || a.Modality != "verbatim" || a.Object == cues[0].Text {
			t.Fatal("invented semantics/copied text", a)
		}
	}
}
func TestRejectForeignCueVoiceAndInvalidProposition(t *testing.T) {
	cues := testCues()
	for _, a := range []Assertion{{Subject: "x", Relation: "r", Object: "o", Polarity: "positive", Modality: "asserted", CueIDs: []string{"foreign"}}, {Subject: "x", Relation: "r", Object: "o", Polarity: "positive", Modality: "asserted", CueIDs: []string{"a"}, LocalSpeakerID: "22222222-2222-4222-8222-222222222222"}, {Subject: "x", Relation: "r", Object: "o", Polarity: "invented", Modality: "asserted", CueIDs: []string{"a"}}} {
		if ValidateAssertion(a, cues) == nil {
			t.Fatal("invalid assertion accepted", a)
		}
	}
	cfg := DefaultConfig()
	cfg.MaxChunkBytes = 2
	if _, e := Chunks(cues, cfg); e == nil {
		t.Fatal("complete oversized cue split")
	}
}
func TestNegationAndConditionsDeduplicateOnlyExactMeaning(t *testing.T) {
	base := Assertion{Subject: "x", Relation: "r", Object: "o", Polarity: "positive", Modality: "asserted", CueIDs: []string{"a"}, Conditions: []string{"condition"}}
	second := base
	second.Polarity = "negative"
	third := base
	third.Conditions = []string{"different"}
	result := Deduplicate([]Assertion{base, base, second, third})
	if len(result) != 3 {
		raw, _ := json.Marshal(result)
		t.Fatal(string(raw))
	}
}
