// SPDX-License-Identifier: Apache-2.0
package answers

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"
)

// ConceptID is the existing normalized graph concept identity. Kind separates
// equal subject/relation/object labels without guessing an entity resolution.
func ConceptID(kind, value string) string {
	h := sha256.Sum256([]byte(kind + "\x00" + value))
	h[6] = (h[6] & 15) | 80
	h[8] = (h[8] & 63) | 128
	s := hex.EncodeToString(h[:16])
	return s[:8] + "-" + s[8:12] + "-" + s[12:16] + "-" + s[16:20] + "-" + s[20:]
}
func Matches(item Item, r Request, expanded bool) bool {
	f := r.Filters
	if len(f.MediaIDs) > 0 && !slices.Contains(f.MediaIDs, item.Reference.RecordingID) {
		return false
	}
	if len(f.SpeakerIDs) > 0 {
		matched := false
		for _, id := range item.SpeakerIDs {
			matched = matched || slices.Contains(f.SpeakerIDs, id)
		}
		if !matched {
			return false
		}
	}
	if f.SourceInterval != nil {
		if item.Reference.Source == nil {
			return false
		}
		lo, hi, ok := item.Reference.Source.Values()
		a, b, yes := f.SourceInterval.Values()
		if !ok || !yes || hi <= a || lo >= b {
			return false
		}
	}
	if d := f.RecordingDates; d != nil {
		if !item.DateKnown {
			if !d.IncludeUndated {
				return false
			}
		} else if d.From != "" && item.RecordingDateThrough != "" && item.RecordingDateThrough < d.From || d.Through != "" && item.RecordingDate != "" && item.RecordingDate > d.Through {
			return false
		}
	}
	if expanded {
		return true
	}
	if f.Text != "" && !strings.Contains(strings.ToLower(item.Text), strings.ToLower(f.Text)) {
		return false
	}
	concepts, entities := []string{}, []string{item.ID, item.Reference.RecordingID, "media:" + item.Reference.RecordingID, "document:" + item.Reference.RecordingID + ":" + item.Reference.DocumentDigest}
	for _, id := range item.SpeakerIDs {
		entities = append(entities, "speaker:"+id)
	}
	for _, a := range item.Statements {
		entities = append(entities, "assertion:"+item.Reference.RecordingID+":"+item.Reference.DocumentDigest+":"+a.ID)
		for _, p := range []struct{ kind, value string }{{"subject", a.Subject}, {"relation", a.Relation}, {"object", a.Object}} {
			id := ConceptID(p.kind, p.value)
			concepts = append(concepts, id)
			entities = append(entities, "concept:"+id)
		}
	}
	for _, pair := range []struct{ wanted, available []string }{{f.ConceptIDs, concepts}, {f.EntityIDs, entities}} {
		if len(pair.wanted) > 0 {
			matched := false
			for _, id := range pair.wanted {
				matched = matched || slices.Contains(pair.available, id)
			}
			if !matched {
				return false
			}
		}
	}
	return true
}
