// SPDX-License-Identifier: Apache-2.0
package contracts

import (
	"bytes"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"time"
)

type QueryFilter struct {
	EntityIDs      []string        `json:"entity_ids,omitempty"`
	MediaIDs       []string        `json:"media_ids,omitempty"`
	SpeakerIDs     []string        `json:"speaker_ids,omitempty"`
	Text           string          `json:"text,omitempty"`
	ModelKind      string          `json:"model_kind,omitempty"`
	RecordingDates *CalendarFilter `json:"recording_dates,omitempty"`
	SourceInterval *SourceInterval `json:"source_interval,omitempty"`
	ConceptIDs     []string        `json:"concept_ids,omitempty"`
}
type CalendarFilter struct {
	From           string `json:"from,omitempty"`
	Through        string `json:"through,omitempty"`
	Timezone       string `json:"timezone,omitempty"`
	IncludeUndated bool   `json:"include_undated,omitempty"`
}
type SourceInterval struct {
	StartUS int64 `json:"start_us"`
	EndUS   int64 `json:"end_us"`
}
type QueryOrder struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}
type QueryTraversal struct {
	RelationshipTypes []string `json:"relationship_types,omitempty"`
	Direction         string   `json:"direction"`
	MaxDepth          int      `json:"max_depth"`
}
type QueryDefinition struct {
	Mode       string           `json:"mode"`
	Operation  string           `json:"operation,omitempty"`
	Filters    QueryFilter      `json:"filters,omitempty"`
	Traversal  *QueryTraversal  `json:"traversal,omitempty"`
	OrderBy    []QueryOrder     `json:"order_by,omitempty"`
	Pagination *QueryPagination `json:"pagination,omitempty"`
	Dialect    string           `json:"dialect,omitempty"`
	Text       string           `json:"text,omitempty"`
}
type QueryPagination struct {
	Limit  int    `json:"limit"`
	Cursor string `json:"cursor,omitempty"`
}
type QueryParameter struct {
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value"`
}
type QueryInput struct {
	Title      string                    `json:"title,omitempty"`
	Definition QueryDefinition           `json:"definition"`
	Parameters map[string]QueryParameter `json:"parameters,omitempty"`
}

func DecodeExplore(raw []byte, v any) error {
	if len(raw) == 0 || len(raw) > 1<<20 {
		return Fail("invalid_request")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	d.UseNumber()
	if d.Decode(v) != nil || d.Decode(new(any)) != io.EOF {
		return Fail("invalid_request")
	}
	return nil
}
func (q QueryInput) Validate() error {
	d := q.Definition
	if len(q.Title) > 512 || len(d.Text) > 65536 || len(q.Parameters) > 128 {
		return Fail("input_limit")
	}
	if d.Mode == "native" {
		if hasNormalizedFilters(d.Filters) || d.Text == "" || !strings.Contains("|ladybug-cypher|arcade-opencypher|arcade-sql|", "|"+d.Dialect+"|") || d.Operation != "" || len(d.OrderBy) > 0 || d.Traversal != nil || d.Pagination != nil {
			return Fail("invalid_request")
		}
	} else if d.Mode == "normalized" {
		if !strings.Contains("|media-list|speaker-search|model-list|text-search|time-range|evidence-traverse|graph-view|", "|"+d.Operation+"|") || d.Text != "" || d.Dialect != "" || len(q.Parameters) > 0 {
			return Fail("invalid_request")
		}
		if d.Pagination != nil && (d.Pagination.Limit < 1 || d.Pagination.Limit > 500 || len(d.Pagination.Cursor) > 2048) {
			return Fail("invalid_request")
		}
		if len(d.Filters.EntityIDs) > 500 {
			return Fail("input_limit")
		}
		for _, id := range d.Filters.EntityIDs {
			if id == "" || len(id) > 1024 {
				return Fail("invalid_request")
			}
		}
		if len(d.OrderBy) > 8 || len(d.Filters.Text) > 4096 {
			return Fail("input_limit")
		}
		for _, o := range d.OrderBy {
			if !strings.Contains("|id|title|recording_date|source_start_us|speaker_name|relevance|media.originated_at|media.id|speaker.id|model.id|cue.start_us|score|", "|"+o.Field+"|") || (o.Direction != "asc" && o.Direction != "desc") {
				return Fail("invalid_request")
			}
		}
		if d.Traversal != nil && (d.Traversal.MaxDepth < 1 || d.Traversal.MaxDepth > 32 || !strings.Contains("|in|out|both|", "|"+d.Traversal.Direction+"|") || len(d.Traversal.RelationshipTypes) > 64) {
			return Fail("invalid_request")
		}
		if d.Filters.SourceInterval != nil && (d.Filters.SourceInterval.StartUS < 0 || d.Filters.SourceInterval.EndUS <= d.Filters.SourceInterval.StartUS) {
			return Fail("invalid_request")
		}
		for _, ids := range [][]string{d.Filters.MediaIDs, d.Filters.SpeakerIDs, d.Filters.ConceptIDs} {
			if len(ids) > 500 {
				return Fail("input_limit")
			}
			for _, id := range ids {
				if !ValidID(id) {
					return Fail("invalid_request")
				}
			}
		}
		if c := d.Filters.RecordingDates; c != nil {
			for _, s := range []string{c.From, c.Through} {
				if s != "" {
					if _, e := time.Parse("2006-01-02", s); e != nil {
						return Fail("invalid_request")
					}
				}
			}
			if c.From != "" && c.Through != "" && c.From > c.Through {
				return Fail("invalid_request")
			}
			if c.Timezone != "" {
				if _, e := time.LoadLocation(c.Timezone); e != nil {
					return Fail("invalid_request")
				}
			}
		}
	} else {
		return Fail("invalid_request")
	}
	for name, p := range q.Parameters {
		if name == "" || len(name) > 128 || strings.HasPrefix(name, "$") || strings.HasPrefix(name, "_") {
			return Fail("invalid_request")
		}
		var value any
		d := json.NewDecoder(bytes.NewReader(p.Value))
		d.UseNumber()
		if d.Decode(&value) != nil || d.Decode(new(any)) != io.EOF {
			return Fail("invalid_request")
		}
		ok := false
		switch p.Type {
		case "null":
			ok = value == nil
		case "boolean":
			_, ok = value.(bool)
		case "string", "uuid", "date", "date-time":
			s, yes := value.(string)
			ok = yes
			if yes {
				switch p.Type {
				case "uuid":
					ok = ValidID(s)
				case "date":
					_, e := time.Parse("2006-01-02", s)
					ok = e == nil
				case "date-time":
					_, e := time.Parse(time.RFC3339Nano, s)
					ok = e == nil
				}
			}
		case "integer":
			_, e := IntegerParameter(value)
			ok = e == nil
		case "number":
			n, yes := value.(json.Number)
			if yes {
				_, e := strconv.ParseFloat(string(n), 64)
				ok = e == nil
			}
		case "array":
			_, ok = value.([]any)
		case "object":
			_, ok = value.(map[string]any)
		}
		if !ok {
			return Fail("invalid_request")
		}
	}
	return nil
}
func ExploreOperation(op string) bool {
	return strings.Contains("|query.assist|query.assistance-show|query.assistance-set|graph.capabilities|graph.status|graph.publish|graph.rebuild|evidence.extract|evidence.show|query.run|query.explain|query.list|query.show|query.save|query.validate|timeline.calendar|timeline.recording|views.show|views.save|", "|"+op+"|")
}
func ExploreRequestValid(r Request) bool {
	if !ExploreOperation(r.Operation) || r.JobID != "" || r.DurationMS != 0 || r.AfterGeneration != 0 || r.PublicationID != "" || r.SourcePath != "" || r.ArtifactKind != "" || r.LeaseID != "" || r.ReferenceID != "" || r.MaxBytes != 0 || len(r.Data) > 1<<20 {
		return false
	}
	switch r.Operation {
	case "evidence.extract", "evidence.show", "timeline.recording", "query.show", "query.save", "views.show", "views.save":
		return ValidID(r.ItemID)
	default:
		return r.ItemID == ""
	}
}

// IntegerParameter accepts legacy JSON integers and canonical decimal strings.
// Strings preserve exact 64-bit values across desktop JSON number boundaries.
func IntegerParameter(value any) (int64, error) {
	var text string
	canonical := false
	switch v := value.(type) {
	case json.Number:
		text = string(v)
	case string:
		text = v
		canonical = true
	default:
		return 0, Fail("invalid_request")
	}
	n, e := strconv.ParseInt(text, 10, 64)
	if e != nil || (canonical && strconv.FormatInt(n, 10) != text) {
		return 0, Fail("invalid_request")
	}
	return n, nil
}
