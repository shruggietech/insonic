// SPDX-License-Identifier: Apache-2.0
package answers

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/shruggietech/insonic/internal/contracts"
)

type ContextNode struct {
	ID        string   `json:"id"`
	Level     string   `json:"level"`
	Parent    string   `json:"parent,omitempty"`
	Text      string   `json:"text,omitempty"`
	Sources   []string `json:"sources"`
	Condensed bool     `json:"condensed,omitempty"`
}
type Context struct {
	Contract        string        `json:"contract"`
	Question        string        `json:"question"`
	TokenAccounting string        `json:"token_accounting"`
	Bytes           int           `json:"bytes"`
	Tokens          int           `json:"tokens"`
	Omitted         int           `json:"omitted"`
	Nodes           []ContextNode `json:"nodes"`
}

func CacheKey(r Request, workspace, sourceEpoch, graphEpoch string) string {
	return Digest(struct {
		Request                            Request
		Workspace, Source, Graph, Contract string
	}{r, workspace, sourceEpoch, graphEpoch, SummaryContract})
}

// Account the entire serialized envelope, including counters, hierarchy and
// source links. UTF-8 bytes are conservative token units, never a claim about
// a particular provider tokenizer. A caller's smaller limit always wins.
func contextSize(c *Context) int {
	for range 12 {
		raw, _ := json.Marshal(c)
		if c.Bytes == len(raw) && c.Tokens == len(raw) {
			return len(raw)
		}
		c.Bytes, c.Tokens = len(raw), len(raw)
	}
	return MaxPackageBytes
}
func contextNodes(items []Item, texts []string) []ContextNode {
	out := []ContextNode{}
	if len(items) == 0 {
		return out
	}
	out = append(out, ContextNode{ID: "library", Level: "cross-recording", Sources: []string{}})
	indices := map[string]int{"library": 0}
	for i, item := range items {
		recording := "recording:" + item.Reference.RecordingID
		ordinal, _ := Decimal(item.Reference.Ordinal, 0)
		section := fmt.Sprintf("%s:section:%d", recording, ordinal/16)
		for _, pair := range []struct{ id, level, parent string }{{recording, "recording", "library"}, {section, "section", recording}} {
			if _, ok := indices[pair.id]; !ok {
				indices[pair.id] = len(out)
				out = append(out, ContextNode{ID: pair.id, Level: pair.level, Parent: pair.parent, Sources: []string{}})
			}
		}
		for _, ancestor := range []string{"library", recording, section} {
			j := indices[ancestor]
			out[j].Sources = append(out[j].Sources, item.ID)
		}
		out = append(out, ContextNode{ID: item.ID, Level: "cue", Parent: section, Text: texts[i], Sources: []string{item.ID}, Condensed: texts[i] != item.Text})
	}
	return out
}
func BuildContext(r Request, items []Item) (Context, error) {
	c := Context{Contract: SummaryContract, Question: r.Question, TokenAccounting: "utf8-byte-upper-bound", Omitted: len(items), Nodes: []ContextNode{}}
	limit := min(r.Budget.ContextBytes, r.Budget.ContextTokens)
	if contextSize(&c) > limit {
		return c, contracts.Condition("output_limit", "context_envelope_budget")
	}
	selected, texts := []Item{}, []string{}
	for _, item := range items {
		runes := []rune(item.Text)
		// Extractive condensation keeps the first bounded, exact source span.
		lo, hi, best := 1, min(len(runes), 1024), 0
		var fitted Context
		for lo <= hi {
			mid := (lo + hi) / 2
			candidate := c
			candidate.Nodes = contextNodes(append(slices.Clone(selected), item), append(slices.Clone(texts), string(runes[:mid])))
			candidate.Omitted = len(items) - len(selected) - 1
			if contextSize(&candidate) <= limit {
				best, fitted, lo = mid, candidate, mid+1
			} else {
				hi = mid - 1
			}
		}
		if best > 0 {
			selected = append(selected, item)
			texts = append(texts, string(runes[:best]))
			c = fitted
		}
	}
	return c, nil
}
func ValidateContext(c Context, items []Item, budget Budget) error {
	raw, err := json.Marshal(c)
	if err != nil || c.Contract != SummaryContract || c.TokenAccounting != "utf8-byte-upper-bound" || len(raw) != c.Bytes || c.Tokens != c.Bytes || c.Bytes > budget.ContextBytes || c.Tokens > budget.ContextTokens || c.Omitted < 0 {
		return contracts.Condition("conflict", "context_integrity")
	}
	byID := map[string]Item{}
	for _, item := range items {
		byID[item.ID] = item
	}
	seen, used := map[string]string{}, map[string]bool{}
	for _, n := range c.Nodes {
		if _, ok := seen[n.ID]; ok || len(n.Sources) == 0 {
			return contracts.Fail("conflict")
		}
		switch n.Level {
		case "cross-recording":
			if n.Parent != "" || n.ID != "library" {
				return contracts.Fail("conflict")
			}
		case "recording":
			if seen[n.Parent] != "cross-recording" {
				return contracts.Fail("conflict")
			}
		case "section":
			if seen[n.Parent] != "recording" {
				return contracts.Fail("conflict")
			}
		case "cue":
			if seen[n.Parent] != "section" || len(n.Sources) != 1 || n.ID != n.Sources[0] {
				return contracts.Fail("conflict")
			}
			used[n.ID] = true
		default:
			return contracts.Fail("conflict")
		}
		seen[n.ID] = n.Level
		for _, id := range n.Sources {
			if _, ok := byID[id]; !ok {
				return contracts.Fail("conflict")
			}
		}
		if n.Level == "cue" {
			item := byID[n.Sources[0]]
			if n.Text == "" || !strings.HasPrefix(item.Text, n.Text) || n.Condensed != (n.Text != item.Text) {
				return contracts.Condition("conflict", "summary_source_mismatch")
			}
		} else if n.Text != "" {
			return contracts.Condition("conflict", "summary_source_mismatch")
		}
	}
	selected, texts := []Item{}, []string{}
	for _, n := range c.Nodes {
		if n.Level == "cue" {
			selected = append(selected, byID[n.ID])
			texts = append(texts, n.Text)
		}
	}
	if !reflect.DeepEqual(c.Nodes, contextNodes(selected, texts)) {
		return contracts.Condition("conflict", "context_hierarchy_integrity")
	}
	if c.Omitted != len(items)-len(used) {
		return contracts.Fail("conflict")
	}
	return nil
}
