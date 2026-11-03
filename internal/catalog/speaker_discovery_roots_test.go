// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"encoding/json"
	sd "github.com/shruggietech/insonic/internal/speakerdiscovery"
	"testing"
)

func TestDiscoverySourceRootsDeduplicateOverlap(t *testing.T) {
	r := Recording{SourceDigest: hash([]byte("source")), SourceMap: json.RawMessage(`{"stream_index":0,"channel":0,"start_numerator":"0","start_denominator":"1"}`), Document: json.RawMessage(`{"cues":[{"id":"a","speaker_attributions":[{"speaker_id":"v","start_milliseconds":0,"end_milliseconds":100}]},{"id":"b","speaker_attributions":[{"speaker_id":"v","start_milliseconds":50,"end_milliseconds":150}]},{"id":"c","speaker_attributions":[{"speaker_id":"v","start_milliseconds":150,"end_milliseconds":250}]}]}`)}
	c := sd.Context{Cues: []sd.Cue{{ID: "a"}, {ID: "b"}, {ID: "c"}}}
	if e := discoveryRoots(r, &c); e != nil {
		t.Fatal(e)
	}
	if len(c.Cues[0].Roots) != 1 || c.Cues[0].Roots[0] != c.Cues[1].Roots[0] || c.Cues[0].Roots[0] == c.Cues[2].Roots[0] {
		t.Fatal(c)
	}
	r.SourceMap = json.RawMessage("{}")
	c = sd.Context{Cues: []sd.Cue{{ID: "a"}, {ID: "b"}, {ID: "c"}}}
	if e := discoveryRoots(r, &c); e != nil {
		t.Fatal(e)
	}
	if c.Cues[0].Roots[0] != c.Cues[2].Roots[0] {
		t.Fatal("unknown provenance counted independently")
	}
}
