// SPDX-License-Identifier: Apache-2.0
package catalog

import "encoding/json"

type Pipeline struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Preset        string          `json:"preset"`
	Revision      int64           `json:"revision"`
	Configuration json.RawMessage `json:"configuration"`
}
type SpeakerAlias struct {
	ID         string          `json:"id"`
	SpeakerID  string          `json:"speaker_id"`
	Text       string          `json:"text"`
	Language   string          `json:"language"`
	Scope      string          `json:"scope"`
	State      string          `json:"state"`
	Provenance json.RawMessage `json:"provenance"`
}
type SpeakerIdentity struct {
	Speaker Speaker        `json:"speaker"`
	Aliases []SpeakerAlias `json:"aliases"`
}
type Term struct {
	ID         string          `json:"id"`
	Revision   int64           `json:"revision"`
	Canonical  string          `json:"canonical"`
	Variants   json.RawMessage `json:"variants"`
	Language   string          `json:"language"`
	Context    string          `json:"context"`
	State      string          `json:"state"`
	SpeakerID  *string         `json:"speaker_id"`
	AliasID    *string         `json:"alias_id"`
	Provenance json.RawMessage `json:"provenance"`
}
type ContextFilter struct {
	Language   string   `json:"language,omitempty"`
	Context    string   `json:"context,omitempty"`
	SpeakerIDs []string `json:"speaker_ids,omitempty"`
}
type ContextSnapshot struct {
	Revision   int64             `json:"revision"`
	Filter     ContextFilter     `json:"filter"`
	Speakers   []SpeakerIdentity `json:"speakers"`
	Terms      []Term            `json:"terms"`
	ExtraHints []string          `json:"extra_hints,omitempty"`
}
type SpeakerSelection struct {
	SpeakerID   string `json:"speaker_id"`
	RecordingID string `json:"recording_id,omitempty"`
	Cursor      string `json:"cursor,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}
type CurrentReference struct {
	RecordingID       string `json:"recording_id"`
	RecordingRevision int64  `json:"recording_revision"`
	DocumentDigest    string `json:"document_digest"`
	CueID             string `json:"cue_id"`
	LocalSpeakerID    string `json:"local_speaker_id"`
	SpeakerID         string `json:"speaker_id"`
	MappingRevision   int64  `json:"mapping_revision"`
	SourceDigest      string `json:"source_digest"`
	SourceMapDigest   string `json:"source_map_digest"`
}
type SpeakerSelectionPage struct {
	References []CurrentReference `json:"references"`
	Next       string             `json:"next_cursor,omitempty"`
	Revision   int64              `json:"revision"`
	Epoch      string             `json:"-"`
}
type PriorEvidenceComparison struct {
	AssignmentOrdinal int   `json:"assignment_ordinal"`
	Duplicate         bool  `json:"duplicate"`
	Overlap           bool  `json:"overlap"`
	LocalVoices       int64 `json:"local_voices"`
	FirstReference    bool  `json:"first_reference"`
}
type ResolvedEvidence struct {
	Reference CurrentReference `json:"reference"`
	Cue       json.RawMessage  `json:"cue"`
	SourceMap json.RawMessage  `json:"source_map"`
	Mapping   SpeakerMapping   `json:"mapping"`
}
