// SPDX-License-Identifier: Apache-2.0
package speakerdiscovery

import (
	"encoding/json"
	"testing"
)

func TestParseRejectsWholeInvalidResponse(t *testing.T) {
	c := Context{Cues: []Cue{{ID: "c", Text: "I am Alex.", Voices: []string{"v"}, Roots: []string{"r"}}}}
	good := map[string]any{"version": 1, "observations": []any{map[string]any{"class": "self-identification", "name": "Alex", "voice_id": "v", "references": []any{map[string]any{"cue_id": "c", "quote": "I am Alex."}}, "uncertain": false}}}
	for _, mutate := range []func(map[string]any){
		func(o map[string]any) { o["voice_id"] = "foreign" },
		func(o map[string]any) { o["name"] = "Mallory" },
		func(o map[string]any) { o["name"] = "Mallory"; o["aliases"] = []string{"Alex"} },
		func(o map[string]any) { o["aliases"] = []string{"Bea"} }, func(o map[string]any) { o["speaker_id"] = "invented" },
		func(o map[string]any) {
			o["references"] = []any{map[string]any{"cue_id": "foreign", "quote": "I am Alex."}}
		},
		func(o map[string]any) { o["references"] = []any{map[string]any{"cue_id": "c", "quote": "not present"}} },
		func(o map[string]any) { o["start_ms"] = 0 },
	} {
		b, _ := json.Marshal(good)
		var x map[string]any
		json.Unmarshal(b, &x)
		mutate(x["observations"].([]any)[0].(map[string]any))
		b, _ = json.Marshal(x)
		if _, e := Parse(b, c, 128); e == nil {
			t.Fatalf("accepted %s", b)
		}
	}
	for _, raw := range []string{`{"version":1,"version":1,"observations":[]}`, `{"version":1,"observations":null}`, `{"version":2,"observations":[]}`} {
		if _, e := Parse([]byte(raw), c, 128); e == nil {
			t.Fatal(raw)
		}
	}
}
func TestQuotedAndMultiVoiceSelfIntroductionCannotIdentify(t *testing.T) {
	for _, cue := range []Cue{
		{ID: "c", Text: `She said "I am Alex."`, Voices: []string{"v"}, Roots: []string{"r"}},
		{ID: "c", Text: "I am Alex.", Voices: []string{"v", "other"}, Roots: []string{"r"}},
	} {
		b, _ := json.Marshal(map[string]any{"version": 1, "observations": []any{map[string]any{"class": "self-identification", "name": "Alex", "voice_id": "v", "references": []any{map[string]any{"cue_id": "c", "quote": cue.Text}}, "uncertain": false}}})
		h, e := Parse(b, Context{Cues: []Cue{cue}}, 128)
		if e != nil || h[0].Attributable {
			t.Fatal(h, e)
		}
	}
}
