// SPDX-License-Identifier: Apache-2.0
package speakerdiscovery

import (
	"encoding/json"
	"reflect"
	"testing"
)

func introduction(t *testing.T, c Context) []Hypothesis {
	t.Helper()
	b, _ := json.Marshal(map[string]any{"version": 1, "observations": []any{map[string]any{"class": "self-identification", "name": "Alex", "voice_id": "v", "references": []any{map[string]any{"cue_id": "c", "quote": "I am Alex."}}, "uncertain": false}}})
	h, e := Parse(b, c, 128)
	if e != nil {
		t.Fatal(e)
	}
	return h
}
func TestResolutionIndependentAgreementConflictAndPolicy(t *testing.T) {
	score, runner := 0.95, 0.2
	c := Context{Cues: []Cue{{ID: "c", Text: "I am Alex.", Voices: []string{"v"}, Roots: []string{"r"}}}, Voices: []string{"v"}, Identities: []Identity{{ID: "a", Name: "Alex"}, {ID: "b", Name: "Bea"}}, Acoustic: AcousticEvidence{Availability: "current", WorkID: "accepted", Decisions: []AcousticDecision{{VoiceID: "v", SpeakerID: "a", State: "matched", Score: &score, RunnerUpScore: &runner, Independent: true}}}}
	h := introduction(t, c)
	p, _ := NormalizePolicy(Policy{})
	d, e := Resolve(c, h, p)
	if e != nil || d[0].Outcome != "matched" {
		t.Fatal(d, e)
	}
	c.Acoustic.Decisions[0].SpeakerID = "b"
	d, e = Resolve(c, h, p)
	if e != nil || d[0].Outcome != "conflicting" {
		t.Fatal(d, e)
	}
	c.Acoustic.Decisions[0].Independent = false
	d, e = Resolve(c, h, p)
	if e != nil || d[0].Outcome != "supported-contextual" {
		t.Fatal(d, e)
	}
	c.Owners = map[string]string{"v": "b"}
	d, e = Resolve(c, h, p)
	if e != nil || d[0].Outcome != "conflicting" || *d[0].SpeakerID != "b" {
		t.Fatal(d, e)
	}
	c.Owners = nil
	p.MinimumRoots = 2
	d, e = Resolve(c, append(h, h...), p)
	if e != nil || d[0].Outcome != "unknown" {
		t.Fatal("duplicate evidence inflated", d, e)
	}
	c.Acoustic.Decisions[0].State = "ambiguous"
	c.Acoustic.Decisions[0].Independent = true
	d, e = Resolve(c, nil, p)
	if e != nil || d[0].Outcome != "ambiguous" {
		t.Fatal(d, e)
	}
	x, e := Resolve(c, nil, p)
	if e != nil || !reflect.DeepEqual(d, x) {
		t.Fatal("nondeterminism")
	}
}

func TestDuplicateAliasesPreserveAlternativesWithoutInventedConflict(t *testing.T) {
	c := Context{Cues: []Cue{{ID: "c", Text: "I am Alex.", Voices: []string{"v"}, Roots: []string{"r"}}}, Voices: []string{"v"}, Identities: []Identity{{ID: "a", Name: "Alexandra", Aliases: []Alias{{Text: "Alex"}}}, {ID: "b", Name: "Alexander", Aliases: []Alias{{Text: "Alex"}}}}}
	h := introduction(t, c)
	p, _ := NormalizePolicy(Policy{})
	d, e := Resolve(c, h, p)
	if e != nil || d[0].Outcome != "ambiguous" || len(d[0].Alternatives) != 2 {
		t.Fatal(d, e)
	}
}

func TestCurrentOwnerAndAcousticDisambiguation(t *testing.T) {
	score := 0.95
	c := Context{Cues: []Cue{{ID: "c", Text: "I am Alex.", Voices: []string{"v"}, Roots: []string{"r"}}}, Voices: []string{"v"}, Identities: []Identity{{ID: "a", Name: "Alexandra", Aliases: []Alias{{Text: "Alex"}}}, {ID: "b", Name: "Alexander", Aliases: []Alias{{Text: "Alex"}}}}, Owners: map[string]string{"v": "a"}}
	p, _ := NormalizePolicy(Policy{})
	d, e := Resolve(c, nil, p)
	if e != nil || d[0].Outcome != "matched" || *d[0].SpeakerID != "a" {
		t.Fatal(d, e)
	}
	c.Acoustic = AcousticEvidence{Availability: "current", Decisions: []AcousticDecision{{VoiceID: "v", SpeakerID: "b", State: "matched", Score: &score, Independent: true}}}
	d, e = Resolve(c, nil, p)
	if e != nil || d[0].Outcome != "conflicting" || *d[0].SpeakerID != "a" {
		t.Fatal(d, e)
	}
	c.Owners = nil
	d, e = Resolve(c, introduction(t, c), p)
	if e != nil || d[0].Outcome != "matched" || *d[0].SpeakerID != "b" || len(d[0].Alternatives) != 2 {
		t.Fatal(d, e)
	}
}
func TestMultilingualObservedAliasesAndRoleChanges(t *testing.T) {
	c := Context{Cues: []Cue{{ID: "c", Text: "私は明です。Alexと呼んでください。", Voices: []string{"v"}, Roots: []string{"r"}}}, Voices: []string{"v"}, Identities: []Identity{{ID: "a", Name: "明", Aliases: []Alias{{Text: "Alex", Language: "en"}}}}}
	raw := []byte(`{"version":1,"observations":[{"class":"self-identification","name":"明","aliases":["Alex"],"language":"ja","role":"presenter","relationship":"host","voice_id":"v","references":[{"cue_id":"c","quote":"私は明です。Alexと呼んでください。"}],"uncertain":false}]}`)
	h, e := Parse(raw, c, 128)
	if e != nil || h[0].Language != "ja" || h[0].Role != "presenter" || h[0].Relationship != "host" {
		t.Fatal(h, e)
	}
	d, e := Resolve(c, h, Policy{})
	if e != nil || d[0].Outcome != "supported-contextual" || *d[0].SpeakerID != "a" {
		t.Fatal(d, e)
	}
	if e := ValidateReferences(c, h); e != nil {
		t.Fatal("UTF-8 reference changed", e)
	}
}
