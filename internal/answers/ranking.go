// SPDX-License-Identifier: Apache-2.0
package answers

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"
	"unicode"
)

func terms(s string) []string {
	words := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) })
	slices.Sort(words)
	return slices.Compact(words)
}
func reported(item Item) bool {
	if len(item.Statements) == 0 {
		return false
	}
	for _, a := range item.Statements {
		if a.QuotedAttribution == "" {
			return false
		}
	}
	return true
}
func originStream(item Item) *int {
	if item.OriginStreamIndex != nil {
		return item.OriginStreamIndex
	}
	if item.Lineage == "canonical-origin" {
		return nil
	}
	return item.Reference.StreamIndex
}
func overlapping(a, b Item) bool {
	if a.Reference.RecordingID != b.Reference.RecordingID && (originStream(a) == nil || originStream(b) == nil) {
		return false
	}
	if a.Family != b.Family || !reflect.DeepEqual(originStream(a), originStream(b)) || !reflect.DeepEqual(a.Reference.Channel, b.Reference.Channel) {
		return false
	}
	if a.Reference.Source == nil || b.Reference.Source == nil {
		return a.Reference.DocumentDigest == b.Reference.DocumentDigest && a.Reference.CueID == b.Reference.CueID
	}
	as, ae, ok := a.Reference.Source.Values()
	bs, be, yes := b.Reference.Source.Values()
	return ok && yes && as < be && bs < ae
}

// Rank retains duplicate-source members for inspection but awards family and
// overlap coverage once. Distinct source digests mean distinct observed source
// families, not acoustic proof that all unknown upstream origins differ.
func Rank(question string, input []Item) ([]Item, []Conflict, []Exclusion, int) {
	items := slices.Clone(input)
	sort.Slice(items, func(i, j int) bool {
		if items[i].ID != items[j].ID {
			return items[i].ID < items[j].ID
		}
		return Digest(items[i]) < Digest(items[j])
	})
	out, conflicts, excluded := []Item{}, []Conflict{}, []Exclusion{}
	for _, item := range items {
		if len(out) > 0 && out[len(out)-1].ID == item.ID {
			i := len(out) - 1
			out[i].Paths = append(out[i].Paths, item.Paths...)
			excluded = append(excluded, Exclusion{item.ID, "duplicate-evidence-identity"})
			continue
		}
		item.Paths = slices.Clone(item.Paths)
		item.Reasons = slices.DeleteFunc(slices.Clone(item.Reasons), func(reason string) bool {
			return strings.HasPrefix(reason, "relevance:") || strings.HasPrefix(reason, "lineage:") || slices.Contains([]string{"shared-source-family-or-reported-speech", "overlapping-source-span", "distinct-families-do-not-prove-independent-upstream-origin"}, reason)
		})
		item.Score = Score{}
		out = append(out, item)
	}
	parents := make([]int, len(out))
	for i := range parents {
		parents[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		for parents[i] != i {
			i = parents[i]
		}
		return i
	}
	for i := range out {
		for j := 0; j < i; j++ {
			if overlapping(out[i], out[j]) {
				parents[find(i)] = find(j)
			}
		}
	}
	q := terms(question)
	families, groups := map[string]bool{}, map[string]bool{}
	for i := range out {
		x := &out[i]
		x.OverlapGroup = out[find(i)].ID
		sort.Slice(x.Paths, func(i, j int) bool { return Digest(x.Paths[i]) < Digest(x.Paths[j]) })
		x.Paths = slices.CompactFunc(x.Paths, func(a, b Path) bool { return reflect.DeepEqual(a, b) })
		words := terms(x.Text)
		for _, w := range q {
			if slices.Contains(words, w) {
				x.Score.Relevance++
			}
		}
		if !reported(*x) && !families[x.Family] {
			x.Score.Independence = 1
			families[x.Family] = true
		} else {
			x.Reasons = append(x.Reasons, "shared-source-family-or-reported-speech")
		}
		if !groups[x.OverlapGroup] {
			x.Score.Coverage = 1
			groups[x.OverlapGroup] = true
		} else {
			x.Reasons = append(x.Reasons, "overlapping-source-span")
		}
		x.Reasons = append(x.Reasons, "relevance:question-term-overlap", "lineage:"+x.Lineage, "distinct-families-do-not-prove-independent-upstream-origin")
	}
	// Index propositions once. Comparing every assertion with every other
	// assertion multiplies repeated extraction cost without adding information.
	type polarities struct{ positive, negative map[int]bool }
	propositions := map[string]*polarities{}
	for i, item := range out {
		for _, a := range item.Statements {
			if a.QuotedAttribution != "" || a.Modality != "asserted" || (a.Polarity != "positive" && a.Polarity != "negative") {
				continue
			}
			conditions := slices.Clone(a.Conditions)
			slices.Sort(conditions)
			conditions = slices.Compact(conditions)
			key := Digest([]any{a.Subject, a.Relation, a.Object, conditions})
			if propositions[key] == nil {
				propositions[key] = &polarities{map[int]bool{}, map[int]bool{}}
			}
			if a.Polarity == "positive" {
				propositions[key].positive[i] = true
			} else {
				propositions[key].negative[i] = true
			}
		}
	}
	keys := []string{}
	for key := range propositions {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	seenPairs := map[string]bool{}
	opposingFamilies := make([]map[string]bool, len(out))
	for i := range out {
		opposingFamilies[i] = map[string]bool{}
	}
	conflictBytes := 0
	omitted := false
	for _, key := range keys {
		group := propositions[key]
		positive, negative := []int{}, []int{}
		for i := range group.positive {
			positive = append(positive, i)
		}
		for i := range group.negative {
			negative = append(negative, i)
		}
		slices.Sort(positive)
		slices.Sort(negative)
		for _, i := range positive {
			for _, j := range negative {
				opposingFamilies[i][out[j].Family] = true
				opposingFamilies[j][out[i].Family] = true
				a, b := min(i, j), max(i, j)
				pair := fmt.Sprintf("%d:%d", a, b)
				if seenPairs[pair] {
					continue
				}
				seenPairs[pair] = true
				ids := []string{out[a].ID}
				if a != b {
					ids = append(ids, out[b].ID)
				}
				conflict := Conflict{ids, "opposite-polarity-same-proposition-and-conditions"}
				raw, _ := json.Marshal(conflict)
				if len(conflicts) >= 500 || conflictBytes+len(raw) > 65536 {
					omitted = true
					continue
				}
				conflictBytes += len(raw)
				conflicts = append(conflicts, conflict)
			}
		}
	}
	if omitted {
		excluded = append(excluded, Exclusion{Reason: "contradiction_detail_budget"})
	}
	for i := range out {
		out[i].Score.Contradiction = len(opposingFamilies[i])
	}
	for i := range out {
		s := &out[i].Score
		s.Total = 4*s.Relevance + 2*s.Independence + s.Coverage + 2*s.Contradiction
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score.Total != out[j].Score.Total {
			return out[i].Score.Total > out[j].Score.Total
		}
		return out[i].ID < out[j].ID
	})
	return out, conflicts, excluded, len(families)
}
