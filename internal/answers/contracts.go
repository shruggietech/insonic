// SPDX-License-Identifier: Apache-2.0
// Package answers defines bounded evidence and grounded synthesis contracts.
// Derived text never becomes an independent evidence source.
package answers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/evidence"
)

const SummaryContract = "source-extract-v1"
const MaxPackageBytes = 1 << 20

type Interval struct {
	StartUS string `json:"start_us"`
	EndUS   string `json:"end_us"`
}
type Filters struct {
	EntityIDs      []string                  `json:"entity_ids,omitempty"`
	MediaIDs       []string                  `json:"media_ids,omitempty"`
	SpeakerIDs     []string                  `json:"speaker_ids,omitempty"`
	ConceptIDs     []string                  `json:"concept_ids,omitempty"`
	Text           string                    `json:"text,omitempty"`
	RecordingDates *contracts.CalendarFilter `json:"recording_dates,omitempty"`
	SourceInterval *Interval                 `json:"source_interval,omitempty"`
}
type Budget struct {
	Rows          int  `json:"rows"`
	SourceBytes   int  `json:"source_bytes"`
	Items         int  `json:"items"`
	EvidenceBytes int  `json:"evidence_bytes"`
	Edges         int  `json:"edges"`
	Rounds        int  `json:"rounds"`
	DeadlineMS    int  `json:"deadline_ms"`
	NeighborCues  *int `json:"neighbor_cues,omitempty"`
	ContextBytes  int  `json:"context_bytes"`
	ContextTokens int  `json:"context_tokens"`
}

// Provider contains elected non-secret identity, not transport credentials or
// endpoints. Context preparation is deterministic and makes no provider call.
type Provider struct {
	ID                  string `json:"id,omitempty"`
	Model               string `json:"model,omitempty"`
	ConfigurationDigest string `json:"configuration_digest,omitempty"`
	PromptContract      string `json:"prompt_contract,omitempty"`
	SummaryContract     string `json:"summary_contract"`
}
type Request struct {
	Version   string                    `json:"version"`
	Question  string                    `json:"question"`
	Filters   Filters                   `json:"filters,omitempty"`
	Traversal *contracts.QueryTraversal `json:"traversal,omitempty"`
	Budget    Budget                    `json:"budget"`
	Provider  Provider                  `json:"provider"`
	Cursor    string                    `json:"cursor,omitempty"`
}
type Reference struct {
	WorkspaceID       string    `json:"workspace_id"`
	RecordingID       string    `json:"recording_id"`
	RecordingRevision string    `json:"recording_revision"`
	DocumentDigest    string    `json:"document_digest"`
	SourceDigest      string    `json:"source_digest"`
	SourceMapDigest   string    `json:"source_map_digest"`
	IdentityDigest    string    `json:"identity_digest"`
	CueID             string    `json:"cue_id"`
	Ordinal           string    `json:"ordinal"`
	Source            *Interval `json:"source,omitempty"`
	StreamIndex       *int      `json:"stream_index,omitempty"`
	Channel           *int      `json:"channel,omitempty"`
}
type Path struct {
	Kind  string   `json:"kind"`
	Nodes []string `json:"nodes,omitempty"`
	Edges []string `json:"edges,omitempty"`
}
type Score struct {
	Relevance     int `json:"relevance"`
	Independence  int `json:"independence"`
	Coverage      int `json:"coverage"`
	Contradiction int `json:"contradiction"`
	Total         int `json:"total"`
}
type Item struct {
	OriginStreamIndex    *int                 `json:"origin_stream_index,omitempty"`
	ID                   string               `json:"id"`
	Reference            Reference            `json:"reference"`
	Text                 string               `json:"text"`
	SpeakerIDs           []string             `json:"speaker_ids,omitempty"`
	LocalSpeakerIDs      []string             `json:"local_speaker_ids,omitempty"`
	RecordingDate        string               `json:"recording_date,omitempty"`
	RecordingDateThrough string               `json:"recording_date_through,omitempty"`
	DateKnown            bool                 `json:"date_known"`
	Family               string               `json:"family"`
	Lineage              string               `json:"lineage"`
	OverlapGroup         string               `json:"overlap_group,omitempty"`
	Paths                []Path               `json:"paths"`
	Statements           []evidence.Assertion `json:"statements,omitempty"`
	Score                Score                `json:"score"`
	Reasons              []string             `json:"reasons"`
}
type Conflict struct {
	EvidenceIDs []string `json:"evidence_ids"`
	Reason      string   `json:"reason"`
}
type Exclusion struct {
	EvidenceID string `json:"evidence_id,omitempty"`
	Reason     string `json:"reason"`
}
type Usage struct {
	Rows          int   `json:"rows"`
	SourceBytes   int   `json:"source_bytes"`
	EvidenceBytes int   `json:"evidence_bytes"`
	Edges         int   `json:"edges"`
	Rounds        int   `json:"rounds"`
	ElapsedMS     int64 `json:"elapsed_ms"`
}
type Observation struct {
	SourceEpoch     string `json:"source_epoch"`
	CatalogRevision string `json:"catalog_revision"`
	GraphGeneration string `json:"graph_generation,omitempty"`
	GraphSequence   string `json:"graph_sequence,omitempty"`
	GraphRevision   string `json:"graph_revision,omitempty"`
	GraphState      string `json:"graph_state"`
}
type Package struct {
	Kind                string      `json:"kind"`
	Version             string      `json:"schema_version"`
	ID                  string      `json:"id"`
	WorkspaceID         string      `json:"workspace_id"`
	Key                 string      `json:"key"`
	State               string      `json:"state"`
	Request             Request     `json:"request"`
	Observation         Observation `json:"observation"`
	Items               []Item      `json:"items"`
	IndependentFamilies int         `json:"independent_families"`
	Conflicts           []Conflict  `json:"conflicts"`
	Exclusions          []Exclusion `json:"exclusions"`
	Diagnostics         []string    `json:"diagnostics"`
	Usage               Usage       `json:"usage"`
	Next                string      `json:"next,omitempty"`
	Context             Context     `json:"context"`
}
type Citation struct {
	EvidenceID string    `json:"evidence_id"`
	Reference  Reference `json:"reference"`
	Quote      string    `json:"quote"`
}
type Claim struct {
	Kind        string     `json:"kind,omitempty"`
	ID          string     `json:"id"`
	Text        string     `json:"text"`
	Support     string     `json:"support"`
	Citations   []Citation `json:"citations,omitempty"`
	Uncertainty string     `json:"uncertainty,omitempty"`
}
type Envelope struct {
	Version     string   `json:"version"`
	PackageID   string   `json:"package_id"`
	State       string   `json:"state"`
	Claims      []Claim  `json:"claims"`
	Limitations []string `json:"limitations,omitempty"`
}

func Digest(v any) string {
	raw, _ := json.Marshal(v)
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:])
}
func ValidDigest(s string) bool {
	b, err := hex.DecodeString(s)
	return err == nil && len(b) == 32 && strings.ToLower(s) == s
}
func Decimal(s string, minimum int64) (int64, bool) {
	n, err := strconv.ParseInt(s, 10, 64)
	return n, err == nil && n >= minimum && strconv.FormatInt(n, 10) == s
}
func (i Interval) Values() (int64, int64, bool) {
	a, ok := Decimal(i.StartUS, 0)
	b, yes := Decimal(i.EndUS, 0)
	return a, b, ok && yes && b > a
}
func Normalize(r Request) (Request, error) {
	if r.Version != contracts.Version || !contracts.ValidTextLength(r.Question, 1, 4096) || strings.TrimSpace(r.Question) == "" || len(r.Cursor) > 8192 {
		return r, contracts.Fail("invalid_request")
	}
	for _, v := range []struct {
		p        *int
		def, cap int
	}{
		{&r.Budget.Rows, 4096, 100000}, {&r.Budget.SourceBytes, 4 << 20, 8 << 20},
		{&r.Budget.Items, 128, 500}, {&r.Budget.EvidenceBytes, 256 << 10, 512 << 10},
		{&r.Budget.Edges, 128, 1000}, {&r.Budget.Rounds, 32, 256},
		{&r.Budget.DeadlineMS, 20000, 60000}, {&r.Budget.ContextBytes, 32768, 256 << 10},
		{&r.Budget.ContextTokens, 8192, 262144},
	} {
		if *v.p == 0 {
			*v.p = v.def
		}
		if *v.p < 1 || *v.p > v.cap {
			return r, contracts.Fail("invalid_request")
		}
	}
	if r.Budget.NeighborCues == nil {
		n := 1
		r.Budget.NeighborCues = &n
	}
	if *r.Budget.NeighborCues < 0 || *r.Budget.NeighborCues > 8 {
		return r, contracts.Fail("invalid_request")
	}
	if r.Traversal == nil {
		r.Traversal = &contracts.QueryTraversal{Direction: "both", MaxDepth: 3}
	}
	if r.Traversal.MaxDepth > 8 {
		return r, contracts.Fail("invalid_request")
	}
	f := r.Filters
	q := contracts.QueryInput{Definition: contracts.QueryDefinition{Mode: "normalized", Operation: "evidence-traverse", Traversal: r.Traversal, Filters: contracts.QueryFilter{EntityIDs: f.EntityIDs, MediaIDs: f.MediaIDs, SpeakerIDs: f.SpeakerIDs, ConceptIDs: f.ConceptIDs, Text: f.Text, RecordingDates: f.RecordingDates}}}
	if err := q.Validate(); err != nil {
		return r, err
	}
	if f.SourceInterval != nil {
		if _, _, ok := f.SourceInterval.Values(); !ok {
			return r, contracts.Fail("invalid_request")
		}
	}
	for _, kind := range r.Traversal.RelationshipTypes {
		if !contracts.ValidTextLength(kind, 1, 128) {
			return r, contracts.Fail("invalid_request")
		}
	}
	for _, s := range []string{r.Provider.ID, r.Provider.PromptContract} {
		if !contracts.ValidTextLength(s, 0, 256) {
			return r, contracts.Fail("invalid_request")
		}
	}
	if !contracts.ValidTextLength(r.Provider.Model, 0, 512) {
		return r, contracts.Fail("invalid_request")
	}
	if r.Provider.ConfigurationDigest != "" && !ValidDigest(r.Provider.ConfigurationDigest) {
		return r, contracts.Fail("invalid_request")
	}
	if r.Provider.SummaryContract == "" {
		r.Provider.SummaryContract = SummaryContract
	}
	if r.Provider.SummaryContract != SummaryContract {
		return r, contracts.Fail("invalid_request")
	}
	// Set-valued scope cannot produce distinct cache entries merely by ordering.
	for _, list := range []*[]string{&r.Filters.EntityIDs, &r.Filters.MediaIDs, &r.Filters.SpeakerIDs, &r.Filters.ConceptIDs, &r.Traversal.RelationshipTypes} {
		*list = slices.Clone(*list)
		slices.Sort(*list)
		*list = slices.Compact(*list)
	}
	return r, nil
}
func ValidateReference(r Reference) error {
	if !contracts.ValidID(r.WorkspaceID) || !contracts.ValidID(r.RecordingID) || !contracts.ValidTextLength(r.CueID, 1, 1024) {
		return contracts.Fail("invalid_request")
	}
	if _, ok := Decimal(r.RecordingRevision, 1); !ok {
		return contracts.Fail("invalid_request")
	}
	if _, ok := Decimal(r.Ordinal, 0); !ok {
		return contracts.Fail("invalid_request")
	}
	for _, s := range []string{r.DocumentDigest, r.SourceDigest, r.SourceMapDigest, r.IdentityDigest} {
		if !ValidDigest(s) {
			return contracts.Fail("invalid_request")
		}
	}
	if r.Source != nil {
		if _, _, ok := r.Source.Values(); !ok {
			return contracts.Fail("invalid_request")
		}
	}
	if r.StreamIndex != nil && *r.StreamIndex < 0 || r.Channel != nil && (*r.Channel < 0 || *r.Channel > 63) {
		return contracts.Fail("invalid_request")
	}
	return nil
}

// ValidateEnvelope checks the historical package binding. The runtime also
// resolves that package against current catalog authority before accepting it.
// Faithful quotes establish provenance; they do not prove semantic entailment.
func ValidateEnvelope(e Envelope, p Package) error {
	if e.Version != contracts.Version || !slices.Contains([]string{"complete", "partial", "insufficient", "cancelled"}, e.State) || len(e.Claims) > 500 || len(e.Limitations) > 128 || e.PackageID != p.ID {
		return contracts.Fail("invalid_request")
	}
	if e.State == "complete" && (len(e.Claims) == 0 || p.State != "complete") {
		return contracts.Fail("invalid_request")
	}
	items := map[string]Item{}
	for _, item := range p.Items {
		items[item.ID] = item
	}
	seen := map[string]bool{}
	for _, c := range e.Claims {
		if !contracts.ValidTextLength(c.ID, 1, 128) || seen[c.ID] || !contracts.ValidTextLength(c.Text, 1, 16384) || !contracts.ValidTextLength(c.Uncertainty, 0, 4096) || len(c.Citations) > 128 {
			return contracts.Fail("invalid_request")
		}
		seen[c.ID] = true
		if c.Kind != "" && c.Kind != "quote" && c.Kind != "paraphrase" {
			return contracts.Fail("invalid_request")
		}
		switch c.Support {
		case "supported":
			if len(c.Citations) == 0 {
				return contracts.Fail("invalid_request")
			}
		case "unsupported", "uncertain", "conflicting":
			if c.Uncertainty == "" || e.State == "complete" {
				return contracts.Fail("invalid_request")
			}
		default:
			return contracts.Fail("invalid_request")
		}
		for _, cite := range c.Citations {
			item, ok := items[cite.EvidenceID]
			if !ok || ValidateReference(cite.Reference) != nil || cite.Reference.WorkspaceID != p.WorkspaceID || !reflect.DeepEqual(cite.Reference, item.Reference) || cite.Quote == "" || !strings.Contains(item.Text, cite.Quote) {
				return contracts.Condition("conflict", "citation_authority")
			}
		}
	}
	for _, s := range e.Limitations {
		if !contracts.ValidTextLength(s, 1, 4096) {
			return contracts.Fail("invalid_request")
		}
	}
	return nil
}
