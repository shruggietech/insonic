// SPDX-License-Identifier: Apache-2.0
package answers

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/evidence"
)

func TestRankingUsesLineageAndPreservesDisagreement(t *testing.T) {
	a := testItem()
	a.ID = "a"
	a.Text = "solar power"
	a.Reference.Source = &Interval{"100", "300"}
	stream := 0
	a.Reference.StreamIndex = &stream
	b := a
	b.ID = "b"
	b.Reference.RecordingID = contracts.ID()
	b.Reference.Source = &Interval{"200", "400"}
	b.Lineage = "canonical-origin"
	b.OriginStreamIndex = &stream
	c := a
	c.ID = "c"
	c.Family = "source:independent"
	c.Reference.RecordingID = contracts.ID()
	a.Statements = []evidence.Assertion{{Subject: "solar", Relation: "powers", Object: "plant", Polarity: "positive", Modality: "asserted"}}
	c.Statements = []evidence.Assertion{{Subject: "solar", Relation: "powers", Object: "plant", Polarity: "negative", Modality: "asserted"}}
	d := c
	d.ID = "d"
	d.Family = "source:reported"
	d.Statements = []evidence.Assertion{{Subject: "solar", Relation: "powers", Object: "plant", Polarity: "positive", Modality: "asserted", QuotedAttribution: "Someone else"}}
	items, conflicts, exclusions, families := Rank("solar power", []Item{b, d, c, a, a})
	if len(items) != 4 || len(exclusions) != 1 || families != 2 || len(conflicts) != 1 {
		t.Fatal(items, conflicts, exclusions, families)
	}
	byID := map[string]Item{}
	for _, x := range items {
		byID[x.ID] = x
	}
	if byID["a"].OverlapGroup != byID["b"].OverlapGroup || byID["a"].OverlapGroup == byID["c"].OverlapGroup || byID["d"].Score.Independence != 0 {
		t.Fatal("lineage/overlap/quotation", byID)
	}
	again, cc, ee, ff := Rank("solar power", []Item{a, c, a, d, b})
	if !reflect.DeepEqual(items, again) || !reflect.DeepEqual(conflicts, cc) || !reflect.DeepEqual(exclusions, ee) || families != ff {
		t.Fatal("input order changed ranking")
	}
}

func TestContextRejectsCrossRecordingAncestorSources(t *testing.T) {
	r, _ := Normalize(Request{Version: contracts.Version, Question: "Context"})
	a := testItem()
	b := a
	b.ID = "second"
	b.Reference.RecordingID = contracts.ID()
	c, err := BuildContext(r, []Item{a, b})
	if err != nil {
		t.Fatal(err)
	}
	for i := range c.Nodes {
		if c.Nodes[i].Level == "recording" {
			c.Nodes[i].Sources = []string{b.ID}
			break
		}
	}
	contextSize(&c)
	if ValidateContext(c, []Item{a, b}, r.Budget) == nil {
		t.Fatal("ancestor claimed evidence from another recording")
	}
}

func TestContextFullWireBudgetAndSourcePreservation(t *testing.T) {
	r, _ := Normalize(Request{Version: contracts.Version, Question: "What happened?"})
	a := testItem()
	a.Text = strings.Repeat("世界🌍<quoted>&\n", 300)
	b := a
	b.ID = "second"
	b.Reference.RecordingID = contracts.ID()
	for _, cap := range []int{1, 256, 1024, 4096, 32768} {
		r.Budget.ContextBytes = cap
		r.Budget.ContextTokens = cap
		c, err := BuildContext(r, []Item{a, b})
		if err != nil {
			if cap > 256 {
				t.Fatal(cap, err)
			}
			continue
		}
		raw, _ := json.Marshal(c)
		if len(raw) > cap || c.Bytes != len(raw) || c.Tokens != len(raw) || !utf8.Valid(raw) {
			t.Fatal(cap, len(raw), c)
		}
		if err := ValidateContext(c, []Item{a, b}, r.Budget); err != nil {
			t.Fatal(err)
		}
		if len(c.Nodes) > 0 {
			c.Nodes[len(c.Nodes)-1].Text = "Unsupported summary"
			if ValidateContext(c, []Item{a, b}, r.Budget) == nil {
				t.Fatal("summary invented text")
			}
		}
	}
	r.Budget.ContextBytes = 32768
	r.Budget.ContextTokens = 32768
	c, err := BuildContext(r, []Item{a, b})
	if err != nil {
		t.Fatal(err)
	}
	levels := map[string]bool{}
	for _, n := range c.Nodes {
		levels[n.Level] = true
	}
	for _, level := range []string{"cue", "section", "recording", "cross-recording"} {
		if !levels[level] {
			t.Fatal("missing hierarchy", level)
		}
	}
	key := CacheKey(r, "workspace", "source-epoch", "graph-epoch")
	mutations := []func(*Request){
		func(r *Request) { r.Provider.Model = "different" }, func(r *Request) { r.Provider.PromptContract = "next" },
		func(r *Request) { r.Provider.ID = "different" }, func(r *Request) { r.Provider.ConfigurationDigest = strings.Repeat("a", 64) },
		func(r *Request) { r.Filters.Text = "new scope" },
	}
	for _, mutate := range mutations {
		other := r
		mutate(&other)
		if CacheKey(other, "workspace", "source-epoch", "graph-epoch") == key {
			t.Fatal("incompatible cache identity")
		}
	}
	if CacheKey(r, "workspace", "new-source-epoch", "graph-epoch") == key || CacheKey(r, "other-workspace", "source-epoch", "graph-epoch") == key {
		t.Fatal("source/workspace cache reuse")
	}
}

func TestContradictionDetailBudgetAndUnknownStream(t *testing.T) {
	items := []Item{}
	for i := 0; i < 50; i++ {
		item := testItem()
		item.ID = string(rune('A' + i))
		item.Family = item.ID
		polarity := "positive"
		if i >= 25 {
			polarity = "negative"
		}
		item.Statements = []evidence.Assertion{{Subject: "s", Relation: "r", Object: "o", Polarity: polarity, Modality: "asserted"}}
		items = append(items, item)
	}
	_, conflicts, excluded, _ := Rank("s", items)
	if len(conflicts) > 500 || len(excluded) == 0 {
		t.Fatal("contradiction detail was unbounded", len(conflicts), excluded)
	}
	a := testItem()
	b := a
	b.ID = "other"
	b.Reference.RecordingID = contracts.ID()
	a.Reference.Source = &Interval{"0", "100"}
	b.Reference.Source = &Interval{"0", "100"}
	if overlapping(a, b) {
		t.Fatal("unknown stream established cross-recording overlap")
	}
}
