// SPDX-License-Identifier: Apache-2.0
package speakerdiscovery

import (
	"bytes"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"io"
	"sort"
	"strings"
)

type citation struct {
	CueID string `json:"cue_id"`
	Quote string `json:"quote"`
}
type observation struct {
	Class        string     `json:"class"`
	Name         string     `json:"name"`
	Aliases      []string   `json:"aliases,omitempty"`
	Language     string     `json:"language,omitempty"`
	Role         string     `json:"role,omitempty"`
	Relationship string     `json:"relationship,omitempty"`
	VoiceID      string     `json:"voice_id,omitempty"`
	References   []citation `json:"references"`
	Uncertain    *bool      `json:"uncertain"`
}
type generated struct {
	Version      int           `json:"version"`
	Observations []observation `json:"observations"`
}

func Parse(raw []byte, c Context, max int) ([]Hypothesis, error) {
	out := []Hypothesis{}
	if len(raw) > MaxBytes || max < 1 || max > MaxObservations || contracts.ValidateJSON(raw) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	var g generated
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&g) != nil || d.Decode(new(any)) != io.EOF || g.Version != Version || g.Observations == nil || len(g.Observations) > max {
		return nil, contracts.Fail("invalid_request")
	}
	cues := map[string]Cue{}
	for _, cue := range c.Cues {
		if _, ok := cues[cue.ID]; ok {
			return nil, contracts.Fail("invalid_request")
		}
		cues[cue.ID] = cue
	}
	seen := map[string]bool{}
	for _, o := range g.Observations {
		if !classOK(o.Class) || !textOK(o.Name, true) || !textOK(o.Language, false) || !textOK(o.Role, false) || !textOK(o.Relationship, false) || len(o.Aliases) > 16 || len(o.References) < 1 || len(o.References) > MaxReferences || o.Uncertain == nil {
			return nil, contracts.Fail("invalid_request")
		}
		for _, a := range o.Aliases {
			if !textOK(a, true) {
				return nil, contracts.Fail("invalid_request")
			}
		}
		h := Hypothesis{Class: o.Class, Name: strings.TrimSpace(o.Name), Aliases: unique(o.Aliases), Language: o.Language, Role: o.Role, Relationship: o.Relationship, VoiceID: o.VoiceID, References: []Reference{}, Roots: []string{}, Uncertain: *o.Uncertain, Attributable: o.VoiceID != ""}
		refs := map[string]bool{}
		observedNames := map[string]bool{}
		for _, ref := range o.References {
			cue, ok := cues[ref.CueID]
			if !ok || ref.Quote == "" || len(ref.Quote) > 32768 || !strings.Contains(cue.Text, ref.Quote) || refs[ref.CueID] {
				return nil, contracts.Fail("invalid_request")
			}
			refs[ref.CueID] = true
			for _, name := range append([]string{o.Name}, o.Aliases...) {
				observedNames[Name(name)] = observedNames[Name(name)] || strings.Contains(Name(ref.Quote), Name(name))
			}
			found := o.VoiceID == ""
			for _, voice := range cue.Voices {
				found = found || voice == o.VoiceID
			}
			if !found {
				return nil, contracts.Fail("invalid_request")
			}
			// A partial quote or multi-voice cue cannot establish who made a direct
			// introduction. Citation checks prove occurrence, not model semantics.
			if len(unique(cue.Voices)) != 1 || strings.TrimSpace(ref.Quote) != strings.TrimSpace(cue.Text) || strings.ContainsAny(ref.Quote, "\"“”«»「」") {
				h.Attributable = false
			}
			h.References = append(h.References, Reference{CueID: ref.CueID, QuoteDigest: Hash([]byte(ref.Quote)), Start: strings.Index(cue.Text, ref.Quote), End: strings.Index(cue.Text, ref.Quote) + len(ref.Quote)})
			h.Roots = append(h.Roots, cue.Roots...)
		}
		for _, name := range append([]string{o.Name}, o.Aliases...) {
			if !observedNames[Name(name)] {
				return nil, contracts.Fail("invalid_request")
			}
		}
		h.Roots = unique(h.Roots)
		sort.Slice(h.References, func(i, j int) bool { return h.References[i].CueID < h.References[j].CueID })
		h.ID = Digest(h)
		if !ValidHypothesis(h) {
			return nil, contracts.Fail("invalid_request")
		}
		if !seen[h.ID] {
			out = append(out, h)
			seen[h.ID] = true
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
