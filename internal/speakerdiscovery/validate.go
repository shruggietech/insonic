// SPDX-License-Identifier: Apache-2.0
package speakerdiscovery

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
)

func ValidateReferences(c Context, values []Hypothesis) error {
	cues := map[string]Cue{}
	for _, cue := range c.Cues {
		cues[cue.ID] = cue
	}
	for _, h := range values {
		if !ValidHypothesis(h) {
			return contracts.Fail("invalid_request")
		}
		o := observation{Class: h.Class, Name: h.Name, Aliases: h.Aliases, Language: h.Language, Role: h.Role, Relationship: h.Relationship, VoiceID: h.VoiceID, References: []citation{}, Uncertain: &h.Uncertain}
		for _, ref := range h.References {
			cue, ok := cues[ref.CueID]
			if !ok || ref.Start < 0 || ref.End <= ref.Start || ref.End > len(cue.Text) {
				return contracts.Fail("invalid_request")
			}
			quote := cue.Text[ref.Start:ref.End]
			if Hash([]byte(quote)) != ref.QuoteDigest {
				return contracts.Fail("invalid_request")
			}
			o.References = append(o.References, citation{CueID: ref.CueID, Quote: quote})
		}
		raw, _ := json.Marshal(generated{Version: Version, Observations: []observation{o}})
		parsed, e := Parse(raw, c, 1)
		if e != nil || len(parsed) != 1 || Digest(parsed[0]) != Digest(h) {
			return contracts.Fail("invalid_request")
		}
	}
	return nil
}
