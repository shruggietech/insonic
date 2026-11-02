// SPDX-License-Identifier: Apache-2.0
package speakerdiscovery

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestDiscoveryStrictEvidenceAndResolution(t *testing.T) {
	ctx := Context{Cues: []Cue{{ID: "c1", Text: "I am Alex.", Voices: []string{"v1"}, Roots: []string{"source-1"}}}, Voices: []string{"v1"}}
	raw := []byte("{\"version\":1,\"observations\":[{\"class\":\"self-identification\",\"name\":\"Alex\",\"voice_id\":\"v1\",\"references\":[{\"cue_id\":\"c1\",\"quote\":\"I am Alex.\"}],\"uncertain\":false}]}")
	h, e := Parse(raw, ctx, 128)
	if e != nil || len(h) != 1 {
		t.Fatal(h, e)
	}
	p, e := NormalizePolicy(Policy{})
	if e != nil {
		t.Fatal(e)
	}
	d, e := Resolve(ctx, h, p)
	if e != nil || len(d) != 1 || d[0].Outcome != "supported-contextual" || d[0].SpeakerID != nil || d[0].IdentityStatus != "unregistered" {
		t.Fatal(d, e)
	}
	again, e := Resolve(ctx, append(h, h...), p)
	if e != nil || !reflect.DeepEqual(d, again) {
		t.Fatal("duplicates changed evidence", d, again, e)
	}
	ctx.Identities = []Identity{{ID: "a", Name: "Alex"}, {ID: "b", Name: "Alex"}}
	d, e = Resolve(ctx, h, p)
	if e != nil || d[0].Outcome != "ambiguous" || len(d[0].Alternatives) != 2 {
		t.Fatal(d, e)
	}
	for _, class := range []string{"mention", "quotation", "context-association"} {
		h[0].Class = class
		d, e = Resolve(ctx, h, p)
		if e != nil || d[0].Outcome != "unknown" {
			t.Fatal(class, d, e)
		}
	}
	var obj map[string]any
	json.Unmarshal(raw, &obj)
	obj["timing"] = 1
	b, _ := json.Marshal(obj)
	if _, e = Parse(b, ctx, 128); e == nil {
		t.Fatal("invented fields accepted")
	}
}
