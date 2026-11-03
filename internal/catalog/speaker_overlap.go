// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"math/big"
	"sort"
	"strings"
)

// MixedSpeakerEvidence indexes all current local voices, including unmapped
// voices and later cues. Two directional sweeps avoid pairwise corpus scans.
func MixedSpeakerEvidence(r Recording) (map[string]bool, error) {
	spans, e := discoveryIntervals(r)
	if e != nil {
		return nil, e
	}
	out := map[string]bool{}
	groups := map[string][]discoveryInterval{}
	channels := map[string]map[string]bool{}
	unknown := []discoveryInterval{}
	for _, v := range spans {
		if !v.known {
			unknown = append(unknown, v)
			continue
		}
		p := strings.Split(v.track, ":")
		if channels[p[0]] == nil {
			channels[p[0]] = map[string]bool{"downmix": true}
		}
		channels[p[0]][p[1]] = true
	}
	for _, v := range spans {
		if !v.known {
			continue
		}
		p := strings.Split(v.track, ":")
		if p[1] == "downmix" {
			for ch := range channels[p[0]] {
				groups[p[0]+":"+ch] = append(groups[p[0]+":"+ch], v)
			}
		} else {
			groups[v.track] = append(groups[v.track], v)
		}
	}
	key := func(v discoveryInterval) string { return v.cue + "/" + v.voice }
	// Unknown routing/timing cannot establish isolation; diagnose rather than mix.
	if len(unknown) > 0 {
		voices := map[string]bool{}
		for _, v := range unknown {
			voices[v.voice] = true
		}
		for _, v := range spans {
			if len(voices) > 1 || len(voices) == 1 && !voices[v.voice] {
				out[key(v)] = true
			}
		}
		allVoices := map[string]bool{}
		for _, v := range spans {
			allVoices[v.voice] = true
		}
		if len(allVoices) > 1 {
			for _, v := range unknown {
				out[key(v)] = true
			}
		}
	}
	type edge struct {
		voice string
		at    *big.Rat
	}
	for _, g := range groups {
		for _, reverse := range []bool{false, true} {
			sort.Slice(g, func(i, j int) bool {
				if reverse {
					return g[i].end.Cmp(g[j].end) > 0
				}
				return g[i].start.Cmp(g[j].start) < 0
			})
			best := []edge{}
			for _, v := range g {
				for _, b := range best {
					if b.voice == v.voice {
						continue
					}
					overlap := b.at.Cmp(v.start) > 0
					if reverse {
						overlap = b.at.Cmp(v.end) < 0
					}
					if overlap {
						out[key(v)] = true
					}
					break
				}
				value := v.end
				if reverse {
					value = v.start
				}
				found := false
				for i := range best {
					if best[i].voice == v.voice {
						found = true
						if !reverse && value.Cmp(best[i].at) > 0 || reverse && value.Cmp(best[i].at) < 0 {
							best[i].at = value
						}
					}
				}
				if !found {
					best = append(best, edge{v.voice, value})
				}
				sort.Slice(best, func(i, j int) bool {
					if reverse {
						return best[i].at.Cmp(best[j].at) < 0
					}
					return best[i].at.Cmp(best[j].at) > 0
				})
				if len(best) > 2 {
					best = best[:2]
				}
			}
		}
	}
	return out, nil
}
