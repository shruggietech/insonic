// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	sd "github.com/shruggietech/insonic/internal/speakerdiscovery"
	"math/big"
	"sort"
	"strconv"
	"strings"
)

type discoveryInterval struct {
	cue, voice, source, track string
	start, end                *big.Rat
	known                     bool
}

func discoveryIntervals(r Recording) ([]discoveryInterval, error) {
	var doc struct {
		Cues []struct {
			ID           string `json:"id"`
			Attributions []struct {
				ID    string `json:"speaker_id"`
				Start *int64 `json:"start_milliseconds"`
				End   *int64 `json:"end_milliseconds"`
			} `json:"speaker_attributions"`
		} `json:"cues"`
	}
	if json.Unmarshal(r.Document, &doc) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	out := []discoveryInterval{}
	for _, cue := range doc.Cues {
		for _, a := range cue.Attributions {
			raw, e := ResolveEvidenceSourceMap(r.SourceMap, r.Provenance, cue.ID, a.ID)
			if e != nil {
				return nil, e
			}
			var m struct {
				Stream      *int   `json:"stream_index"`
				Channel     *int   `json:"channel"`
				Numerator   string `json:"start_numerator"`
				Denominator string `json:"start_denominator"`
			}
			if json.Unmarshal(raw, &m) != nil {
				return nil, contracts.Fail("invalid_request")
			}
			v := discoveryInterval{cue: cue.ID, voice: a.ID, source: r.SourceDigest}
			if m.Stream != nil && a.Start != nil && a.End != nil && *a.End > *a.Start {
				v.track = strconv.Itoa(*m.Stream) + ":downmix"
				if m.Channel != nil {
					v.track = strconv.Itoa(*m.Stream) + ":" + strconv.Itoa(*m.Channel)
				}
				offset := new(big.Rat)
				if m.Numerator != "" || m.Denominator != "" {
					var ok bool
					offset, ok = new(big.Rat).SetString(m.Numerator + "/" + m.Denominator)
					if !ok {
						return nil, contracts.Fail("invalid_request")
					}
				}
				v.start = new(big.Rat).Add(offset, new(big.Rat).SetFrac(big.NewInt(*a.Start), big.NewInt(1000)))
				v.end = new(big.Rat).Add(offset, new(big.Rat).SetFrac(big.NewInt(*a.End), big.NewInt(1000)))
				v.known = true
			}
			out = append(out, v)
		}
	}
	return out, nil
}
func discoveryOverlap(a, b discoveryInterval) bool {
	if a.source != b.source {
		return false
	}
	if !a.known || !b.known {
		return true
	}
	as, bs := strings.Split(a.track, ":"), strings.Split(b.track, ":")
	if as[0] != bs[0] {
		return false
	}
	if as[1] != bs[1] && as[1] != "downmix" && bs[1] != "downmix" {
		return false
	}
	return a.start.Cmp(b.end) < 0 && b.start.Cmp(a.end) < 0
}
func discoveryRoots(r Recording, c *sd.Context) error {
	all, e := discoveryIntervals(r)
	if e != nil {
		return e
	}
	selected := map[string]bool{}
	for _, cue := range c.Cues {
		selected[cue.ID] = true
	}
	values := []discoveryInterval{}
	unknown := map[string]bool{}
	downmix := map[string]bool{}
	for _, v := range all {
		if !selected[v.cue] {
			continue
		}
		values = append(values, v)
		if !v.known {
			unknown[v.source] = true
		} else {
			parts := strings.Split(v.track, ":")
			if parts[1] == "downmix" {
				downmix[v.source+":"+parts[0]] = true
			}
		}
	}
	if len(values) > 8000 {
		return contracts.Fail("input_limit")
	}
	groups := map[string][]discoveryInterval{}
	for _, v := range values {
		key := v.source + ":unknown"
		if !unknown[v.source] {
			parts := strings.Split(v.track, ":")
			key = v.source + ":" + v.track
			if downmix[v.source+":"+parts[0]] {
				key = v.source + ":" + parts[0] + ":mixed"
			}
		}
		groups[key] = append(groups[key], v)
	}
	byCue := map[string]map[string]bool{}
	publish := func(group string, parts []discoveryInterval) {
		keys := map[string]bool{}
		for _, v := range parts {
			key := v.source + ":unknown"
			if v.known {
				key = v.source + ":" + v.track + ":" + v.start.RatString() + ":" + v.end.RatString()
			}
			keys[key] = true
		}
		canonical := []string{group}
		for key := range keys {
			canonical = append(canonical, key)
		}
		sort.Strings(canonical)
		root := sd.Digest(canonical)
		for _, v := range parts {
			if byCue[v.cue] == nil {
				byCue[v.cue] = map[string]bool{}
			}
			byCue[v.cue][root] = true
		}
	}
	for key, parts := range groups {
		if unknown[parts[0].source] {
			publish(key, parts)
			continue
		}
		sort.Slice(parts, func(i, j int) bool {
			if n := parts[i].start.Cmp(parts[j].start); n != 0 {
				return n < 0
			}
			return parts[i].end.Cmp(parts[j].end) < 0
		})
		start := 0
		end := parts[0].end
		for i := 1; i < len(parts); i++ {
			if parts[i].start.Cmp(end) < 0 {
				if parts[i].end.Cmp(end) > 0 {
					end = parts[i].end
				}
			} else {
				publish(key, parts[start:i])
				start = i
				end = parts[i].end
			}
		}
		publish(key, parts[start:])
	}
	for i := range c.Cues {
		for root := range byCue[c.Cues[i].ID] {
			c.Cues[i].Roots = append(c.Cues[i].Roots, root)
		}
		sort.Strings(c.Cues[i].Roots)
	}
	return nil
}
