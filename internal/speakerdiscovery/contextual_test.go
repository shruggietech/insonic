// SPDX-License-Identifier: Apache-2.0
package speakerdiscovery

import "testing"

func TestIndependentContextDoesNotBorrowAcousticDisambiguation(t *testing.T) {
	c := Context{Voices: []string{"v"}, Identities: []Identity{{ID: "a", Name: "Alex"}, {ID: "b", Name: "Alex"}}, Owners: map[string]string{}, Acoustic: AcousticEvidence{Availability: "current", Decisions: []AcousticDecision{{VoiceID: "v", SpeakerID: "a", State: "matched", Independent: true, Score: ptrScore(0.99)}}}}
	h := []Hypothesis{{Class: "self-identification", Name: "Alex", VoiceID: "v", Attributable: true, Roots: []string{"root"}}}
	d, e := Resolve(c, h, Policy{})
	if e != nil || d[0].SpeakerID == nil {
		t.Fatal(d, e)
	}
	independent, e := IndependentContext(c, h, Policy{})
	if e != nil || independent[0].Outcome != "ambiguous" {
		t.Fatal(independent, e)
	}
	c.Identities = c.Identities[:1]
	independent, e = IndependentContext(c, h, Policy{})
	if e != nil || independent[0].Outcome != "supported-contextual" {
		t.Fatal(independent, e)
	}
	c.Owners = map[string]string{"v": "a"}
	h = nil
	independent, e = IndependentContext(c, h, Policy{})
	if e != nil || independent[0].Outcome != "unknown" {
		t.Fatal(independent, e)
	}
}
func ptrScore(v float64) *float64 { return &v }
