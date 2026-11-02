// SPDX-License-Identifier: Apache-2.0
package speakerdiscovery

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"sort"
)

func Resolve(c Context, hypotheses []Hypothesis, p Policy) ([]Decision, error) {
	var err error
	p, err = NormalizePolicy(p)
	if err != nil {
		return nil, err
	}
	if len(hypotheses) > MaxObservations || len(c.Voices) > 1000 || len(c.Identities) > 1000 {
		return nil, contracts.Fail("input_limit")
	}
	ids := map[string]Identity{}
	lookup := map[string][]string{}
	for _, id := range c.Identities {
		if _, ok := ids[id.ID]; ok {
			return nil, contracts.Fail("invalid_request")
		}
		ids[id.ID] = id
		lookup[Name(id.Name)] = append(lookup[Name(id.Name)], id.ID)
		for _, a := range id.Aliases {
			lookup[Name(a.Text)] = append(lookup[Name(a.Text)], id.ID)
		}
	}
	// Inactive manual owners remain current authority, but never join the
	// provider candidate list or another voice's name lookup.
	for _, id := range c.OwnerIdentities {
		if _, ok := ids[id.ID]; !ok {
			ids[id.ID] = id
		}
	}
	out := []Decision{}
	for _, voice := range unique(c.Voices) {
		owner := c.Owners[voice]
		candidateIDs := func(name string) []string {
			v := append([]string{}, lookup[Name(name)]...)
			if owner != "" && Name(ids[owner].Name) == Name(name) {
				v = append(v, owner)
			}
			return unique(v)
		}
		d := Decision{VoiceID: voice, Outcome: "unknown", IdentityStatus: "unknown", Alternatives: []Candidate{}, Reasons: []string{}, EvidenceIDs: []string{}, Roots: []string{}}
		type supportGroup struct{ keys, roots []string }
		groups := []supportGroup{}
		candidates := map[string]Candidate{}
		support := map[string][]string{}
		strong := map[string]bool{}
		acoustic := map[string]bool{}
		add := func(key, name string) {
			if _, ok := candidates[key]; ok {
				return
			}
			candidate := Candidate{Name: name}
			if id, ok := ids[key]; ok {
				v := id.ID
				candidate.SpeakerID = &v
				candidate.Name = id.Name
			}
			candidates[key] = candidate
		}
		for _, h := range hypotheses {
			if h.VoiceID != voice {
				continue
			}
			d.EvidenceIDs = append(d.EvidenceIDs, h.ID)
			d.Roots = append(d.Roots, h.Roots...)
			names := append([]string{h.Name}, h.Aliases...)
			keys := []string{}
			for _, name := range names {
				keys = append(keys, candidateIDs(name)...)
			}
			keys = unique(keys)
			if len(keys) == 0 {
				keys = []string{"name:" + Name(h.Name)}
			}
			qualifying := h.Class == "self-identification" && h.Attributable && !h.Uncertain
			if qualifying {
				groups = append(groups, supportGroup{keys, h.Roots})
			}
			for _, key := range keys {
				add(key, h.Name)
				if qualifying {
					support[key] = append(support[key], h.Roots...)
				}
			}
			d.Reasons = append(d.Reasons, "evidence:"+h.Class)
			if !qualifying {
				d.Reasons = append(d.Reasons, "non_identifying_context")
			}
		}
		for key, roots := range support {
			if len(unique(roots)) >= p.MinimumRoots {
				strong[key] = true
			}
		}
		if owner != "" {
			if _, ok := ids[owner]; !ok {
				return nil, contracts.Fail("conflict")
			}
			add(owner, ids[owner].Name)
			strong[owner] = true
			d.Reasons = append(d.Reasons, "current_owner_mapping")
		}
		ambiguousAcoustic := false
		if c.Acoustic.Availability == "current" {
			for _, a := range c.Acoustic.Decisions {
				if a.VoiceID != voice {
					continue
				}
				if !a.Independent {
					d.Reasons = append(d.Reasons, "acoustic_not_independent")
					continue
				}
				if a.State == "ambiguous" {
					ambiguousAcoustic = true
					d.Reasons = append(d.Reasons, "acoustic_ambiguous_unranked")
					continue
				}
				if a.State != "matched" || a.Score == nil || !unit((*a.Score+1)/2) {
					continue
				}
				runner := 0.0
				if a.RunnerUpScore != nil {
					runner = *a.RunnerUpScore
					if !unit((runner + 1) / 2) {
						return nil, contracts.Fail("invalid_request")
					}
				}
				if _, ok := ids[a.SpeakerID]; !ok {
					return nil, contracts.Fail("conflict")
				}
				add(a.SpeakerID, ids[a.SpeakerID].Name)
				if *a.Score >= *p.MinimumScore && *a.Score-runner >= *p.MinimumMargin {
					strong[a.SpeakerID] = true
					acoustic[a.SpeakerID] = true
					d.Reasons = append(d.Reasons, "independent_acoustic_match")
				} else {
					ambiguousAcoustic = true
					d.Reasons = append(d.Reasons, "acoustic_below_policy")
				}
			}
		} else {
			d.Reasons = append(d.Reasons, "acoustic_"+c.Acoustic.Availability)
		}
		keys := []string{}
		for key := range candidates {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		strongKeys := []string{}
		for _, key := range keys {
			d.Alternatives = append(d.Alternatives, candidates[key])
			if strong[key] {
				strongKeys = append(strongKeys, key)
			}
		}
		chosen := ""
		if len(strongKeys) > 1 {
			// Shared spelling creates alternatives, not contradictory independent
			// assertions. Distinct names or independent acoustic/owner disagreement do.
			contradictory := false
			for id := range acoustic {
				if owner != "" && id != owner {
					contradictory = true
				}
			}
			for _, h := range hypotheses {
				if h.VoiceID == voice && h.Class == "self-identification" && h.Attributable && !h.Uncertain {
					hs := map[string]bool{}
					for _, n := range append([]string{h.Name}, h.Aliases...) {
						for _, id := range candidateIDs(n) {
							hs[id] = true
						}
					}
					if owner != "" && !hs[owner] {
						contradictory = true
					}
					for id := range acoustic {
						if !hs[id] {
							contradictory = true
						}
					}
				}
			}
			for i, left := range groups {
				for _, right := range groups[:i] {
					leftKeys, rightKeys := map[string]bool{}, map[string]bool{}
					for _, key := range left.keys {
						if strong[key] {
							leftKeys[key] = true
						}
					}
					for _, key := range right.keys {
						if strong[key] {
							rightKeys[key] = true
						}
					}
					sharedIdentity, sharedRoot := false, false
					for key := range leftKeys {
						sharedIdentity = sharedIdentity || rightKeys[key]
					}
					for _, a := range left.roots {
						for _, b := range right.roots {
							sharedRoot = sharedRoot || a == b
						}
					}
					if len(leftKeys) > 0 && len(rightKeys) > 0 && !sharedIdentity && !sharedRoot {
						contradictory = true
					}
				}
			}
			if contradictory {
				d.Outcome = "conflicting"
				d.Reasons = append(d.Reasons, "incompatible_support")
			} else if owner != "" {
				chosen = owner
				d.Outcome = "matched"
			} else if len(acoustic) == 1 {
				for id := range acoustic {
					chosen = id
				}
				d.Outcome = "matched"
			} else {
				d.Outcome = "ambiguous"
				d.Reasons = append(d.Reasons, "duplicate_identity_name")
			}
		} else if owner != "" {
			chosen = owner
			d.Outcome = "matched"
		} else if ambiguousAcoustic {
			d.Outcome = "ambiguous"
		} else if len(strongKeys) == 1 {
			chosen = strongKeys[0]
			if acoustic[chosen] {
				d.Outcome = "matched"
			} else {
				d.Outcome = "supported-contextual"
			}
		}
		if owner != "" {
			chosen = owner
		}
		if chosen != "" {
			candidate := candidates[chosen]
			d.SpeakerID = candidate.SpeakerID
			d.Name = candidate.Name
			if candidate.SpeakerID == nil {
				d.IdentityStatus = "unregistered"
			} else {
				d.IdentityStatus = "registered"
			}
		}
		if len(strongKeys) == 0 && !ambiguousAcoustic {
			d.Reasons = append(d.Reasons, "insufficient_independent_support")
		}
		d.Reasons = unique(d.Reasons)
		d.EvidenceIDs = unique(d.EvidenceIDs)
		d.Roots = unique(d.Roots)
		out = append(out, d)
	}
	return out, nil
}
