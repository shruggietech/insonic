// SPDX-License-Identifier: Apache-2.0
package explore

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/graph"
	"sort"
	"strconv"
	"strings"
)

type Result struct {
	Rows            []Row        `json:"items"`
	Edges           []graph.Edge `json:"edges"`
	Total           int          `json:"total"`
	Omitted         int          `json:"omitted"`
	NextCursor      string       `json:"next_cursor,omitempty"`
	CatalogRevision int64        `json:"catalog_revision"`
	Basis           string       `json:"basis"`
	TotalEdges      int          `json:"total_edges"`
	OmittedEdges    int          `json:"omitted_edges"`
	GraphError      string       `json:"graph_error,omitempty"`
}
type cursor struct {
	Digest string `json:"digest"`
	Offset int    `json:"offset"`
}

func Query(c Corpus, refs graph.References, q contracts.QueryInput) (Result, error) {
	out := Result{Rows: []Row{}, Edges: []graph.Edge{}, CatalogRevision: c.Revision, Basis: "current-catalog-references"}
	if e := q.Validate(); e != nil {
		return out, e
	}
	if q.Definition.Mode != "normalized" {
		return out, contracts.Fail("unsupported_capability")
	}
	// Copy hydrated rows: viewport filtering must never mutate a shared corpus.
	rows := map[string]Row{}
	for id, r := range c.Rows {
		rows[id] = r
	}
	c.Rows = rows
	if e := applyDates(&c, q.Definition.Filters.RecordingDates); e != nil {
		return out, e
	}
	current := map[string]bool{}
	for _, n := range c.Refs.Nodes {
		current[n.ID] = true
	}
	available := map[string]bool{}
	for _, n := range refs.Nodes {
		if current[n.ID] {
			available[n.ID] = true
		}
	}
	edges := []graph.Edge{}
	for _, e := range refs.Edges {
		if available[e.From] && available[e.To] {
			edges = append(edges, e)
		}
	}
	// Resolve assertion source ownership from current cited cues, and lineage filters
	// through relationships rather than storing another speaker-assignment array.
	for pass := 0; pass < 4; pass++ {
		for _, e := range edges {
			from, ok := c.Rows[e.From]
			to, tok := c.Rows[e.To]
			if !ok || !tok {
				continue
			}
			if e.Kind == "supported-by" || e.Kind == "has-version" || e.Kind == "contains-segment" {
				for _, id := range to.SpeakerIDs {
					if !contains(from.SpeakerIDs, id) {
						from.SpeakerIDs = append(from.SpeakerIDs, id)
					}
				}
				for _, id := range from.SpeakerIDs {
					if e.Kind == "has-version" && !contains(to.SpeakerIDs, id) {
						to.SpeakerIDs = append(to.SpeakerIDs, id)
					}
				}
				if e.Kind == "supported-by" && from.StartUS == "" {
					from.StartUS = to.StartUS
					from.EndUS = to.EndUS
					from.SeekSeconds = to.SeekSeconds
					from.CueID = to.CueID
				}
				if e.Kind == "has-version" {
					from.ModelKind = to.ModelKind
				}
				c.Rows[e.From] = from
				c.Rows[e.To] = to
			}
		}
	}
	f := q.Definition.Filters
	texts := []string{strings.ToLower(f.Text)}
	speakerText := map[string]bool{}
	if f.Text != "" {
		for _, t := range c.Records.Terms {
			if t.State != "active" {
				continue
			}
			terms := []string{t.Canonical}
			var variants []string
			_ = json.Unmarshal(t.Variants, &variants)
			terms = append(terms, variants...)
			match := false
			for _, v := range terms {
				if strings.EqualFold(v, f.Text) {
					match = true
				}
			}
			if match {
				for _, v := range terms {
					texts = append(texts, strings.ToLower(v))
				}
			}
		}
		for _, a := range c.Records.Aliases {
			if a.State == "active" && strings.Contains(strings.ToLower(a.Text), texts[0]) {
				speakerText[a.SpeakerID] = true
			}
		}
		for _, s := range c.Records.Speakers {
			if strings.Contains(strings.ToLower(s.Name), texts[0]) {
				speakerText[s.ID] = true
			}
		}
	}
	matches := func(r Row) bool {
		if len(f.EntityIDs) > 0 && !contains(f.EntityIDs, r.ID) {
			return false
		}
		if len(f.MediaIDs) > 0 && !contains(f.MediaIDs, r.MediaID) {
			return false
		}
		if len(f.SpeakerIDs) > 0 {
			found := false
			for _, id := range r.SpeakerIDs {
				if contains(f.SpeakerIDs, id) {
					found = true
				}
			}
			if !found {
				return false
			}
		}
		if f.ModelKind != "" && r.ModelKind != f.ModelKind {
			return false
		}
		if f.SourceInterval != nil {
			lo, hi, ok := sourceTime(r)
			matched := ok && hi > f.SourceInterval.StartUS && lo < f.SourceInterval.EndUS
			if len(r.SourceSpans) > 0 {
				matched = false
				for _, span := range r.SourceSpans {
					x, xe := strconv.ParseInt(span.StartUS, 10, 64)
					y, ye := strconv.ParseInt(span.EndUS, 10, 64)
					if xe == nil && ye == nil && y > f.SourceInterval.StartUS && x < f.SourceInterval.EndUS {
						matched = true
					}
				}
			}
			if !matched {
				return false
			}
		}
		if len(f.ConceptIDs) > 0 && !contains(f.ConceptIDs, strings.TrimPrefix(r.ID, "concept:")) {
			return false
		}
		if f.Text != "" {
			found := false
			hay := strings.ToLower(r.Label + " " + r.Text)
			for _, word := range texts {
				if strings.Contains(hay, word) {
					found = true
				}
			}
			for _, id := range r.SpeakerIDs {
				if speakerText[id] {
					found = true
				}
			}
			if !found {
				return false
			}
		}
		return true
	}
	op := q.Definition.Operation
	selected := map[string]bool{}
	for id, r := range c.Rows {
		if !available[id] || !matches(r) {
			continue
		}
		kindOK := op == "graph-view" || op == "evidence-traverse" || (op == "media-list" && r.Kind == "media") || (op == "speaker-search" && r.Kind == "speaker") || (op == "model-list" && (r.Kind == "model" || r.Kind == "model-version" || r.Kind == "base-model")) || ((op == "text-search" || op == "time-range") && (r.Kind == "cue" || r.Kind == "assertion" || r.Kind == "segment"))
		if kindOK {
			selected[id] = true
		}
	}
	if tr := q.Definition.Traversal; tr != nil && (op == "graph-view" || op == "evidence-traverse") {
		frontier := map[string]bool{}
		for id := range selected {
			frontier[id] = true
		}
		for depth := 0; depth < tr.MaxDepth; depth++ {
			next := map[string]bool{}
			for _, e := range edges {
				if len(tr.RelationshipTypes) > 0 && !contains(tr.RelationshipTypes, e.Kind) {
					continue
				}
				visit := func(from, to string) {
					if frontier[from] && !selected[to] {
						if _, ok := c.Rows[to]; ok {
							selected[to] = true
							next[to] = true
						}
					}
				}
				if tr.Direction != "in" {
					visit(e.From, e.To)
				}
				if tr.Direction != "out" {
					visit(e.To, e.From)
				}
			}
			frontier = next
			if len(frontier) == 0 {
				break
			}
		}
	}
	ordered := []Row{}
	for id := range selected {
		ordered = append(ordered, c.Rows[id])
	}
	value := func(r Row, field string) string {
		field = mapOrderField(field)
		switch field {
		case "title":
			return r.Label
		case "recording_date":
			return r.RecordingDate
		case "source_start_us":
			return r.StartUS
		case "speaker_name":
			for _, id := range r.SpeakerIDs {
				if s, ok := c.Rows["speaker:"+id]; ok {
					return s.Label
				}
			}
		case "relevance":
			return "1"
		}
		return r.ID
	}
	sort.Slice(ordered, func(i, j int) bool {
		for _, o := range q.Definition.OrderBy {
			a, b := value(ordered[i], o.Field), value(ordered[j], o.Field)
			cmp := strings.Compare(a, b)
			if mapOrderField(o.Field) == "source_start_us" {
				x, xe := strconv.ParseInt(a, 10, 64)
				y, ye := strconv.ParseInt(b, 10, 64)
				if xe == nil && ye == nil {
					cmp = 0
					if x < y {
						cmp = -1
					} else if x > y {
						cmp = 1
					}
				}
			}
			if cmp != 0 {
				if o.Direction == "desc" {
					return cmp > 0
				}
				return cmp < 0
			}
		}
		return ordered[i].ID < ordered[j].ID
	})
	digestInput := q
	digestInput.Definition.Pagination = nil
	raw, _ := json.Marshal([]any{digestInput, ordered, edges})
	sum := sha256.Sum256(raw)
	digest := hex.EncodeToString(sum[:])
	limit, offset := 100, 0
	if p := q.Definition.Pagination; p != nil {
		limit = p.Limit
		if p.Cursor != "" {
			b, e := base64.RawURLEncoding.DecodeString(p.Cursor)
			var cur cursor
			if e != nil || contracts.DecodeExplore(b, &cur) != nil || cur.Offset < 0 || cur.Offset > len(ordered) {
				return out, contracts.Fail("invalid_request")
			}
			if cur.Digest != digest {
				return out, contracts.Fail("conflict")
			}
			offset = cur.Offset
		}
	}
	out.Total = len(ordered)
	end := min(offset+limit, len(ordered))
	out.Rows = ordered[offset:end]
	out.Omitted = len(ordered) - len(out.Rows)
	if end < len(ordered) {
		b, _ := json.Marshal(cursor{digest, end})
		out.NextCursor = base64.RawURLEncoding.EncodeToString(b)
	}
	visible := map[string]bool{}
	for _, r := range out.Rows {
		visible[r.ID] = true
	}
	for _, e := range edges {
		if selected[e.From] && selected[e.To] {
			out.TotalEdges++
		}
		if visible[e.From] && visible[e.To] {
			out.Edges = append(out.Edges, e)
		}
	}
	out.OmittedEdges = out.TotalEdges - len(out.Edges)
	output, _ := json.Marshal(out)
	if len(output) > 512<<10 {
		return Result{}, contracts.Fail("output_limit")
	}
	return out, nil
}
